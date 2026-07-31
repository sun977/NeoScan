package web

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/web/crawler"
	"neoagent/internal/pkg/fingerprint"
	fpHttp "neoagent/internal/pkg/fingerprint/engines/http"
	fpModel "neoagent/internal/pkg/fingerprint/model"
	"neoagent/internal/pkg/logger"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// WebScanner 实现 Web 指纹扫描与截图
type WebScanner struct {
	// 基础设施
	browserManager  *browser.BrowserManager
	browserLauncher *browser.BrowserLauncher

	// 指纹引擎 (复用 internal/pkg/fingerprint)
	fpEngine *fpHttp.HTTPEngine

	// 资源限制 (QoS)
	limiter *qos.AdaptiveLimiter

	mu       sync.Mutex
	initOnce sync.Once
}

// NewWebScanner 创建 Web 扫描器
func NewWebScanner() *WebScanner {
	bm := browser.NewBrowserManager()
	// 初始化空的指纹引擎
	// 指纹规则将在首次运行 Run 时通过 ensureInit 自动加载（TODO: 从配置文件或 embedded FS 加载指纹规则）
	fpEngine := fpHttp.NewHTTPEngine(nil)

	return &WebScanner{
		browserManager:  bm,
		browserLauncher: browser.NewLauncher(bm),
		fpEngine:        fpEngine,
		// Web 扫描非常耗资源，默认并发限制为 5
		limiter: qos.NewAdaptiveLimiter(5, 1, 10),
	}
}

// Name 扫描器名称
func (s *WebScanner) Name() model.TaskType {
	return model.TaskTypeWebScan
}

// Run 执行扫描任务。
//
// 整体结构是"统一收口一次 return"：不管首页数据是 go-rod 拿到的还是降级
// fallback 拿到的，最终都在函数末尾组装一次结果、判断一次是否需要深度爬取、
// return 一次——不再像重构前那样在浏览器启动失败/导航失败两处提前 return，
// 那两处提前 return 各自绕过了后面的 BFS 触发逻辑，是"降级路径爬虫失效"
// 缺陷的根源（架构方案 8.7 节）。现在不管走到哪条路径，body 一旦拿到手，
// 后面的收口、决策、BFS 都是同一段代码，不存在"某条路径漏掉某个步骤"的可能。
func (s *WebScanner) Run(ctx context.Context, task *model.Task) (results []*model.TaskResult, err error) {
	// 0. Panic Recovery (Linus Style: Don't let a single crash take down the whole agent)
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[WebScanner] PANIC RECOVERED: %v", r)
			err = fmt.Errorf("panic during web scan: %v", r)
			results = nil
		}
	}()

	// 确保指纹规则已加载
	s.ensureInit()

	// 1. 获取 QoS 令牌
	if err1 := s.limiter.Acquire(ctx); err1 != nil {
		return nil, err1
	}
	defer s.limiter.Release()

	startTime := time.Now()
	var protocolHint string
	if p, ok := task.Params["protocol"].(string); ok {
		protocolHint = p
	}
	targetURL := normalizeURL(task.Target, task.PortRange, protocolHint)

	var (
		homeBody       string
		homeHeaders    map[string]string
		homeStatusCode int
		homeContentLen int64
		homeTitle      string
		homeRichCtx    map[string]interface{}
		seedLinks      []string
		screenshotB64  string
		faviconB64     string
		remoteIP       string
		remotePort     int
	)

	// --- go-rod 路径：Launch -> OpenPage -> 监听网络 -> Navigate -> WaitLoad -> ExtractRichContext ---
	if br, errLaunch := s.browserLauncher.Launch(ctx); errLaunch == nil {
		if page, errOpen := s.browserLauncher.OpenPage(ctx, br, ""); errOpen == nil {
			defer page.Close()

			var respMutex sync.Mutex
			waitEvents := page.EachEvent(func(e *proto.NetworkResponseReceived) bool {
				if e.Type == proto.NetworkResourceTypeDocument {
					respMutex.Lock()
					defer respMutex.Unlock()
					homeStatusCode = e.Response.Status
					remoteIP = e.Response.RemoteIPAddress
					if e.Response.RemotePort != nil {
						remotePort = *e.Response.RemotePort
					}
					if homeHeaders == nil {
						homeHeaders = make(map[string]string)
					}
					for k, v := range e.Response.Headers {
						var val string
						if err1 := json.Unmarshal([]byte(v.String()), &val); err1 == nil {
							homeHeaders[k] = val
						} else {
							homeHeaders[k] = v.String()
						}
					}
					if cl, ok := e.Response.Headers["Content-Length"]; ok {
						var clVal string
						if err2 := json.Unmarshal([]byte(cl.String()), &clVal); err2 == nil {
							fmt.Sscanf(clVal, "%d", &homeContentLen)
						}
					} else if e.Response.EncodedDataLength > 0 {
						homeContentLen = int64(e.Response.EncodedDataLength)
					}
				}
				return false
			})
			go waitEvents()

			if errNav := page.Navigate(targetURL); errNav == nil {
				waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				if errWait := page.Context(waitCtx).WaitLoad(); errWait != nil {
					logger.Warnf("[WebScanner] WaitLoad timeout for %s: %v", targetURL, errWait)
				}
				cancel()

				richCtx, errCtx := ExtractRichContext(page)
				if errCtx == nil {
					homeRichCtx = richCtx
					homeBody, _ = richCtx["body"].(string)
					homeTitle = extractTitleFromCtx(richCtx)
					seedLinks = ExtractLinks(page)

					if capture, ok := task.Params["screenshot"].(bool); ok && capture {
						if buf, errShot := page.Screenshot(true, nil); errShot == nil {
							screenshotB64 = base64.StdEncoding.EncodeToString(buf)
						} else {
							logger.Warnf("[WebScanner] Screenshot failed: %v", errShot)
						}
					}
					faviconB64 = extractFaviconFromPage(page, richCtx)
				} else {
					logger.Warnf("[WebScanner] Failed to extract rich context: %v", errCtx)
				}
			} else {
				logger.Warnf("[WebScanner] Navigation failed for %s: %v. Will fallback.", targetURL, errNav)
			}
		}
	} else {
		logger.Warnf("[WebScanner] Failed to launch browser: %v. Will fallback.", errLaunch)
	}

	// --- 统一降级：只要 go-rod 路径没有拿到 body，一律走 fallbackFetch。
	// 这一处判断取代了重构前分散在 Launch 失败、Navigate 失败两处的降级调用，
	// 是修复"降级路径爬虫失效"缺陷的核心改动——不管什么原因导致 go-rod 拿不到
	// body，后面统一走同一条路径继续组装结果、判断是否需要深度爬取。
	if homeBody == "" {
		body, headers, statusCode, title, links, errFetch := s.fallbackFetch(ctx, targetURL)
		if errFetch != nil {
			s.limiter.OnFailure()
			return nil, fmt.Errorf("both browser and fallback fetch failed: %w", errFetch)
		}
		homeBody, homeHeaders, homeStatusCode, homeTitle, seedLinks = body, headers, statusCode, title, links
		homeRichCtx = nil // fallback 路径没有富上下文
		homeContentLen = 0
	}

	finalIP, finalPort := resolveIPPortForResult(task, targetURL, remoteIP, remotePort)

	// Forms/Params 是首页攻击面信息。go-rod 路径没有顺手提取过（ExtractLinks 只管链接），
	// fallback 路径也一样（fallbackFetch 同理只返回 links）。这里用已经拿到的 homeBody
	// 统一提取一次，保证首页结果无论走哪条数据源，Forms/Params 都不缺失。
	_, homeForms, homeParams := crawler.ExtractLinksAndForms(targetURL, homeBody)

	homeResult := s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
		URL: targetURL, Depth: 0, StatusCode: homeStatusCode, Title: homeTitle,
		Body: homeBody, Headers: homeHeaders, ContentLength: homeContentLen, RichContext: homeRichCtx,
		Forms: homeForms, Params: homeParams,
		Screenshot: screenshotB64, Favicon: faviconB64,
	})
	results = append(results, homeResult)

	// --- 是否触发深度爬取：三态判断，见 resolveCrawlDepth ---
	depth := s.resolveCrawlDepth(task, homeStatusCode, homeHeaders, seedLinks)
	if depth > 0 && len(seedLinks) > 0 {
		cr := crawler.New(crawler.Options{MaxDepth: depth}, s.limiter)
		pages := cr.Crawl(ctx, targetURL, seedLinks)

		s.escalateIfNeeded(ctx, cr, pages)

		for _, p := range pages {
			results = append(results, s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
				URL: p.URL, Depth: p.Depth, StatusCode: p.StatusCode, Title: p.Title,
				Body: p.Body, Headers: p.Headers, Forms: p.Forms, Params: p.Params, Leaks: p.Leaks,
			}))
		}
	}

	s.limiter.OnSuccess()
	return results, nil
}

// resolveIPPortForResult 把首页结果需要展示的 IP/Port 归拢成一个函数：优先用
// go-rod 网络事件里拿到的真实远端 IP/Port（remoteIP/remotePort），拿不到
// （比如走的是 fallback 路径，net/http 不会暴露底层连接的远端地址）就依次
// 退化：Target 本身是 IP 就直接用；Port 从 targetURL 解析，解析不出来再从
// task.PortRange 兜底（且只在 PortRange 是单个端口时采纳，避免 "80,443"
// 这种列表被误解析成 80）。
//
// 这段逻辑在重构前是内联在 Run() 里的一段近 30 行的代码（现状第 275-306 行），
// 抽成独立函数是因为 Sprint 5 里 finalIP/finalPort 现在要在 homeResult 和
// 爬虫子页面结果之间共用，内联写法没法复用。
func resolveIPPortForResult(task *model.Task, targetURL, remoteIP string, remotePort int) (ip string, port int) {
	ip = remoteIP
	if ip == "" && isIP(task.Target) {
		ip = task.Target
	}

	port = remotePort
	if port == 0 {
		if u, errParse := url.Parse(targetURL); errParse == nil {
			if p := u.Port(); p != "" {
				fmt.Sscanf(p, "%d", &port)
			} else if u.Scheme == "https" {
				port = 443
			} else if u.Scheme == "http" {
				port = 80
			}
		}
		if port == 0 && task.PortRange != "" &&
			!strings.Contains(task.PortRange, ",") && !strings.Contains(task.PortRange, "-") {
			fmt.Sscanf(task.PortRange, "%d", &port)
		}
	}
	return ip, port
}

// extractFaviconFromPage 从已经导航完成的页面里取 favicon 并转成 base64。
//
// 这段逻辑在重构前是内联在 Run() 里的（现状第 234-260 行），抽成独立函数纯粹是
// 为了让 Run() 主干的可读性不被这段和主流程无关的细节拖累，行为完全不变：
// 用页面里探测到的 favicon_url，通过 page.Eval 执行一段 JS fetch，把结果
// data URL 转成不带前缀的纯 base64 字符串。任何一步失败都返回空字符串，
// 不影响调用方（favicon 是锦上添花的信息，不是关键路径）。
func extractFaviconFromPage(page *rod.Page, richCtx map[string]interface{}) string {
	favURL, ok := richCtx["favicon_url"].(string)
	if !ok || favURL == "" {
		return ""
	}

	res, err := page.Eval(`(url) => {
		return fetch(url)
			.then(response => response.blob())
			.then(blob => new Promise((resolve, reject) => {
				const reader = new FileReader();
				reader.onloadend = () => resolve(reader.result); // data:image/png;base64,...
				reader.onerror = reject;
				reader.readAsDataURL(blob);
			}));
	}`, favURL)
	if err != nil {
		return ""
	}

	dataURL := res.Value.String()
	if idx := strings.Index(dataURL, ","); idx != -1 {
		return dataURL[idx+1:]
	}
	return ""
}

// normalizeURL 简单的 URL 规范化
func normalizeURL(target string, port string, protocol string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}

	host := target
	// 如果端口存在且不是标准端口，且 target 中没有端口，则追加
	if port != "" && port != "80" && port != "443" && !strings.Contains(target, ":") {
		host = target + ":" + port
	}

	// 如果有明确的协议提示，直接使用
	if protocol == "https" {
		return "https://" + host
	}
	if protocol == "http" {
		return "http://" + host
	}

	// 默认猜测
	switch port {
	case "443", "8443", "9443", "10443":
		return "https://" + host
	case "80", "8080", "8000", "8008", "8888":
		return "http://" + host
	default:
		// 其他端口默认为 http，如果失败，Scanner 内部其实很难再自动切 https
		// 所以前面的 Service Detection 准确性很重要
		return "http://" + host
	}
}

// convertMatchesToTechStack 转换指纹格式为 TechStack 列表
func convertMatchesToTechStack(matches []fingerprint.Match) []string {
	var res []string
	seen := make(map[string]bool)
	for _, m := range matches {
		// 格式: Product/Version
		val := m.Product
		if m.Version != "" {
			val += "/" + m.Version
		}
		if !seen[val] {
			res = append(res, val)
			seen[val] = true
		}
	}
	return res
}

func extractTitleFromCtx(ctx map[string]interface{}) string {
	if t, ok := ctx["title"].(string); ok {
		return t
	}
	return ""
}

func isIP(target string) bool {
	// 简单判断，实际可以使用 net.ParseIP
	return strings.Count(target, ".") == 3 && !strings.ContainsAny(target, "abcdefghijklmnopqrstuvwxyz")
}

// ensureInit 确保指纹规则已加载
func (s *WebScanner) ensureInit() {
	s.initOnce.Do(func() {
		// 尝试加载指纹规则
		// 优先顺序:
		// 1. 环境变量/配置指定的路径 (暂未实现)
		// 2. 默认路径 rules/fingerprint/web/web_fingerprints.json
		// 3. 开发环境路径 ../rules/fingerprint/web/web_fingerprints.json

		paths := []string{
			"rules/fingerprint/web/web_fingerprints.json",
			"../rules/fingerprint/web/web_fingerprints.json",
			"../../rules/fingerprint/web/web_fingerprints.json",
			"../../../rules/fingerprint/web/web_fingerprints.json",
			"../../../../rules/fingerprint/web/web_fingerprints.json",
			"../../../../../rules/fingerprint/web/web_fingerprints.json", // For unit tests in internal/core/scanner/web
			"neoAgent/rules/fingerprint/web/web_fingerprints.json",
		}

		var rules []fpModel.FingerRule
		var loadedPath string

		for _, path := range paths {
			content, err := os.ReadFile(path)
			if err == nil {
				if err := json.Unmarshal(content, &rules); err == nil {
					loadedPath = path
					break
				} else {
					logger.Warnf("[WebScanner] Failed to unmarshal rules from %s: %v", path, err)
				}
			}
		}

		if len(rules) > 0 {
			s.fpEngine.Reload(rules)
			logger.Infof("[WebScanner] Loaded %d fingerprint rules from %s", len(rules), loadedPath)
		} else {
			logger.Warnf("[WebScanner] No fingerprint rules loaded")
		}
	})
}

// fallbackFetch 当 Headless Browser 失败时，降级使用标准库 net/http 抓取首页原始数据。
// 主要用于处理一些特殊情况，如 HTTPS 证书错误、网络问题等(chromium 意外情况兜底)。
//
// 与重构前的 fallbackScan 的关键区别：这个函数只负责"抓取"，不负责"组装 WebResult"、
// 不负责"跑指纹匹配"。原来 fallbackScan 和 go-rod 路径（Run 方法主干）各自独立组装了
// 一遍 WebResult + 指纹匹配的代码，两份逻辑分别维护，任何一处指纹匹配的细节改动
// （比如后续要给 RichContext 多塞一个字段）都得同时改两个地方，改漏一处就是一次
// 静默的行为不一致——这是"重复造成的隐性耦合"，比显式的函数调用耦合更危险，因为
// 编译器不会提醒你漏改了。fallbackFetch 把"抓取"这个动作单独拎出来，"组装 WebResult"
// 统一交给下面的 buildWebResult，两条数据来源（go-rod/fallback）从此共用同一份
// 组装逻辑，不可能再出现"改了一处忘了改另一处"的问题。
//
// 返回的 links 是顺手用 Sprint 2 产出的 ExtractLinksAndForms 提取的首页种子链接，
// 为 Sprint 5 挂上 crawler 后的 BFS 爬取做准备；Sprint 4 阶段 Run() 主干还不会用到
// 这个返回值，但函数签名一次性按最终形态定义，避免 Sprint 5 时再改一次签名。
func (s *WebScanner) fallbackFetch(ctx context.Context, targetURL string) (body string, headers map[string]string, statusCode int, title string, links []string, err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment, // 支持系统代理
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", nil, 0, "", nil, err
	}
	// 模拟浏览器 UA，防止被拦截
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, 0, "", nil, err
	}
	defer resp.Body.Close()

	// 限制读取 2MB，防止大文件把内存打爆
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", nil, 0, "", nil, err
	}
	bodyStr := string(bodyBytes)

	headers = make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}

	title = extractTitleFromHTML(bodyStr)

	// 复用 crawler 包 Sprint 2 的产出提取首页种子链接，forms/params 这里不需要
	// （首页的 Forms/Params 由 buildWebResult 的调用方按需通过 pageData 传入，
	// fallbackFetch 只关心"抓取"这一件事）。
	links, _, _ = crawler.ExtractLinksAndForms(targetURL, bodyStr)

	logger.Infof("[WebScanner] Fallback fetch success for %s", targetURL)
	return bodyStr, headers, resp.StatusCode, title, links, nil
}

// extractTitleFromHTML 从 HTML 文本里提取 <title> 标签内的文字。
//
// 这是从原 fallbackScan 里整理出来的干净版本：原实现里同一段"找 <title> 位置、
// 截取内容"的逻辑被写了两遍——第一遍算错了偏移量（+5 而不是 +7），中间插了一堆
// 注释自我纠正"为什么应该是 +7"，然后紧接着重新写了一遍正确的版本。这是历史遗留
// 的调试痕迹（明显是原作者当场调试时留下的，忘了删掉第一次的错误尝试），本次重构
// 顺手清理成一次到位的正确实现，不属于范围蔓延——重构一个函数时，路过一段自相
// 矛盾的重复代码却假装没看见，才是不负责任的做法。
func extractTitleFromHTML(bodyStr string) string {
	lowerBody := strings.ToLower(bodyStr)
	start := strings.Index(lowerBody, "<title>")
	if start == -1 {
		return ""
	}
	end := strings.Index(lowerBody[start:], "</title>")
	if end == -1 {
		return ""
	}
	// start 指向 '<title>' 的 '<'，这个标签本身占 7 个字符，所以内容从 start+7 开始；
	// end 是在 lowerBody[start:] 这个子串里找到的相对偏移量，换算成绝对位置就是
	// start+end，也就是 "</title>" 里 '<' 的位置，即内容的结束边界。
	return bodyStr[start+7 : start+end]
}

// pageData 是 buildWebResult 的统一输入形状。三条数据来源——go-rod 首页探测、
// fallback（net/http）首页抓取、爬虫子页面（Sprint 1-3 的 crawler.Page）——
// 字段含义完全一致，都是"一个页面的原始数据 + 已提取的攻击面信息"，用同一个
// 结构体表达可以让 buildWebResult 不用关心数据到底是从哪条路径来的。
type pageData struct {
	URL        string
	Depth      int
	StatusCode int
	Title      string
	Body       string
	Headers    map[string]string

	// ContentLength 默认取 0（表示"未显式指定"），此时 buildWebResult 会退化成用
	// len(Body) 兜底。go-rod 路径应该显式传入从网络事件里拿到的真实传输长度
	// （可能来自 Content-Length 响应头，也可能是 EncodedDataLength），因为这个值
	// 反映的是"实际在网络上传输了多少字节"，和 len(Body)（解码后的正文长度，
	// 可能因为 gzip 压缩等原因和传输字节数不同）是两个不同的语义，不能互相替代。
	ContentLength int64

	// RichContext 只有 go-rod 路径能提供（包含渲染后的 DOM/JS 变量/Meta/Cookies 等
	// 指纹引擎可能用到的完整上下文）。fallback 和爬虫子页面走的是 net/http，没有
	// 浏览器渲染结果，这里传 nil，buildWebResult 内部会退化成用 Body/Headers/Title
	// 拼一个最小可用的 RichContext，保证指纹匹配在两条路径下都能正常工作，只是
	// go-rod 路径的匹配精度更高（因为上下文信息更完整）。
	RichContext map[string]interface{}

	Forms      []model.FormInfo
	Params     []string
	Leaks      []model.LeakInfo
	Screenshot string
	Favicon    string
}

// buildWebResult 是首页与爬虫子页面共用的收口函数：接收 pageData 这一份统一的
// 原始数据，跑一遍指纹匹配，组装成最终的 *model.TaskResult。
//
// 这个函数存在之前，"构造 fingerprint.Input -> 调用 fpEngine.Match -> 组装
// WebResult" 这一整套逻辑在 Run() 主干（go-rod 路径）和 fallbackScan 里
// 分别重复了一遍，Sprint 5 接入爬虫子页面后会变成第三份重复。三份几乎一样
// 但又不完全一样的代码，是最容易滋生 bug 的地方——修一个地方的字段映射，
// 另外两个地方不会自动跟着变。buildWebResult 把这套逻辑收口成一个函数，
// 后续无论指纹匹配逻辑怎么演进，只需要改这一处。
func (s *WebScanner) buildWebResult(task *model.Task, startTime time.Time, ip string, port int, pd pageData) *model.TaskResult {
	// RichContext 优先使用调用方传入的（go-rod 路径的完整渲染上下文），
	// 只有在调用方没有提供时（fallback/爬虫路径，天然没有浏览器渲染结果）
	// 才现场拼一个最小版本。这里不能反过来无条件用 Body/Headers/Title 重新
	// 拼一份，否则会把 go-rod 路径原本就有的 DOM/JS 变量/Meta 信息扔掉，
	// 导致依赖这些字段的指纹规则在收口重构后失效——这是必须避免的功能回归。
	richCtx := pd.RichContext
	if richCtx == nil {
		richCtx = map[string]interface{}{
			"body":    pd.Body,
			"headers": pd.Headers,
			"title":   pd.Title,
		}
	} else {
		// 与重构前 Run() 主干第 233 行的行为保持一致：headers 是在导航完成后
		// 才从网络事件里收集到的，需要覆盖进已经提取好的 richCtx。
		richCtx["headers"] = pd.Headers
	}

	input := &fingerprint.Input{
		Target:      task.Target,
		Body:        pd.Body,
		Headers:     pd.Headers,
		RichContext: richCtx,
	}

	var techStack []string
	if s.fpEngine != nil {
		if matches, err := s.fpEngine.Match(input); err == nil {
			techStack = convertMatchesToTechStack(matches)
		}
	}

	// ContentLength 优先用调用方显式传入的真实传输长度；调用方没传（fallback/
	// 爬虫路径没有额外的网络层字节数统计）就退化成用解码后的正文长度兜底，
	// 两条路径下这个字段都不会是 0（除非页面正文真的是空的）。
	contentLength := pd.ContentLength
	if contentLength == 0 {
		contentLength = int64(len(pd.Body))
	}

	return &model.TaskResult{
		TaskID:      task.ID,
		Status:      model.TaskStatusSuccess,
		ExecutedAt:  startTime,
		CompletedAt: time.Now(),
		Result: &model.WebResult{
			URL:             pd.URL,
			Depth:           pd.Depth,
			IP:              ip,
			Port:            port,
			Title:           pd.Title,
			StatusCode:      pd.StatusCode,
			ContentLength:   contentLength,
			ResponseHeaders: pd.Headers,
			TechStack:       techStack,
			Screenshot:      pd.Screenshot,
			Favicon:         pd.Favicon,
			Forms:           pd.Forms,
			Params:          pd.Params,
			Leaks:           pd.Leaks,
		},
	}
}

// decideCrawlDepth 基于首页免费信号自动判断是否需要深度爬取，返回 0 表示不爬。
// 判断依据只有三个：状态码、Content-Type、种子链接数量，不引入任何需要额外网络
// 请求的信号——自动决策的前提是"零额外成本"，否则决策本身就变成了新的负担。
func decideCrawlDepth(statusCode int, contentType string, seedLinksCount int) int {
	if statusCode >= 400 && statusCode != 401 && statusCode != 403 {
		return 0 // 4xx/5xx 明确失败页面不爬，但 401/403 可能是"存在但需要认证"，仍值得看一眼
	}
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return 0 // 非 HTML（纯 JSON API、文件下载等）没有链接可爬
	}
	if seedLinksCount == 0 {
		return 0 // 首页没有任何链接，没有 BFS 起点
	}
	return 2 // 默认深度 2
}

// resolveCrawlDepth 综合三态参数（task.Params["crawl"] 显式开启/显式关闭/未指定）
// 与首页自动判断，得出最终爬取深度。三态优先级：显式参数 > 自动判断，这是
// "用户明确表达的意图永远盖过系统的猜测"这条原则在爬虫开关上的落地。
func (s *WebScanner) resolveCrawlDepth(task *model.Task, statusCode int, headers map[string]string, seedLinks []string) int {
	enableCrawl, explicit := task.Params["crawl"].(bool)
	switch {
	case explicit && !enableCrawl:
		return 0
	case explicit && enableCrawl:
		depth := 2
		if d, ok := task.Params["crawl_depth"].(int); ok && d > 0 {
			depth = d
		}
		return depth
	default:
		contentType := headers["Content-Type"]
		return decideCrawlDepth(statusCode, contentType, len(seedLinks))
	}
}

// defaultMaxEscalationPages 是单次扫描里允许被升级渲染的页面数上限。升级渲染要
// 启动真实的 Headless Browser 打开页面，成本远高于 net/http 请求，架构方案 8.8.4
// 节的成本账已经说明升级只应该发生在"极少数页面"身上；一旦某次扫描里需要升级的
// 页面超过这个上限，更可能是识别逻辑对这个站点整体误判（比如整站都是同一种
// 会触发误报的框架特征），继续升级只会成倍放大浏览器开销，不如干脆放弃，保留
// net/http 抓到的原始内容。
const defaultMaxEscalationPages = 10

// escalateIfNeeded 对 BFS 爬到的页面里被 crawler 标记为"需要升级"的页面
// （NeedsEscalation，见 crawler.go 的三层检测），用真实浏览器重新渲染一遍，
// 把渲染后的正文和新发现的链接回填进爬虫，让 JS 渲染出来的内容也能被后续的
// 指纹匹配、被动泄露检测覆盖到，新链接也能继续汇入 BFS。
//
// 这里的 for 循环是刻意串行渲染每个待升级页面（不是并发开多个浏览器 Tab）：
// 升级是极少数页面的兜底路径，不值得为它单独设计并发控制，串行、简单、够用，
// defaultMaxEscalationPages 已经把最坏情况锁定在个位数次串行渲染，量级可接受。
func (s *WebScanner) escalateIfNeeded(ctx context.Context, cr *crawler.Crawler, pages []*crawler.Page) {
	var toEscalate []*crawler.Page
	for _, p := range pages {
		if p.NeedsEscalation {
			toEscalate = append(toEscalate, p)
		}
	}
	if len(toEscalate) == 0 || len(toEscalate) > defaultMaxEscalationPages {
		return
	}

	br, err := s.browserLauncher.Launch(ctx)
	if err != nil {
		logger.Warnf("[WebScanner] escalation skipped, browser launch failed: %v", err)
		return
	}

	for _, p := range toEscalate {
		renderedBody, renderedLinks, ok := s.renderWithBrowser(ctx, br, p.URL)
		if !ok {
			continue // 失败降级：保留原始 net/http 抓到的内容，不中断
		}
		p.Body = renderedBody
		cr.EnqueueExtra(renderedLinks, p.Depth)
	}
}

// renderWithBrowser 用给定的浏览器实例打开一个页面、等待加载、提取正文与链接。
// 任何一步失败都返回 ok=false，调用方 escalateIfNeeded 会据此保留原始内容，
// 不会因为升级失败导致这个页面的数据整个丢失。
func (s *WebScanner) renderWithBrowser(ctx context.Context, br *rod.Browser, targetURL string) (body string, links []string, ok bool) {
	page, err := s.browserLauncher.OpenPage(ctx, br, "")
	if err != nil {
		return "", nil, false
	}
	defer page.Close()

	if err := page.Navigate(targetURL); err != nil {
		return "", nil, false
	}
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = page.Context(waitCtx).WaitLoad() // 超时也继续尝试提取

	richCtx, err := ExtractRichContext(page)
	if err != nil {
		return "", nil, false
	}
	body, _ = richCtx["body"].(string)
	if body == "" {
		return "", nil, false
	}
	links = ExtractLinks(page)
	return body, links, true
}
