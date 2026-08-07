package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/crawler"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/port_service/nmap_service"
	"neoagent/internal/pkg/edge"
	"neoagent/internal/pkg/fingerprint"
	fpHttp "neoagent/internal/pkg/fingerprint/engines/http"
	fpModel "neoagent/internal/pkg/fingerprint/model"
	"neoagent/internal/pkg/logger"
	"neoagent/internal/pkg/utils"
	"net"

	"github.com/go-rod/rod"
)

// WebScanner 实现 Web 指纹扫描与截图
type WebScanner struct {
	// 基础设施
	browserManager  *browser.BrowserManager
	browserLauncher *browser.BrowserLauncher

	// 指纹引擎 (复用 internal/pkg/fingerprint)
	fpEngine *fpHttp.HTTPEngine

	// CDN 边缘节点检测 (复用 internal/pkg/edge)
	edgeDetector *edge.Detector

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
		edgeDetector:    edge.NewDetector(),
		// Web 扫描非常耗资源，默认并发限制为 5
		limiter: qos.NewAdaptiveLimiter(5, 1, 10),
	}
}

// Name 扫描器名称
func (s *WebScanner) Name() model.TaskType {
	return model.TaskTypeWebScan
}

// Run 执行扫描任务，是纯编排层：把 task.PortRange（"80,443"/"1-100"/"top100"
// 这类范围字符串）解析成具体端口列表，每个端口独立探测（各自猜协议、各自抓取、
// 各自可能触发 BFS），互不影响、并发执行，最后把所有端口的结果汇总返回。
//
// Sprint 0-5 遗留的 Run() 只会探测一个 URL——normalizeURL 拿到 "80,443" 这种
// 整串范围字符串时，既不等于 "80" 也不等于 "443"，会直接落进 default 分支猜成
// http，"多端口探测"从未真正发生过。真实站点测试（10.201.28.126、内网多端口
// 服务）暴露了这个问题，现在改成"编排层 Run() + 单端口执行层 runOnePort()"，
// runOnePort() 就是原来 Run() 的函数体，行为不变，只是不再自己算 targetURL、
// 不再自己拿 QoS 令牌（改成每次实际抓取各自获取，见 runOnePort 内部）。
func (s *WebScanner) Run(ctx context.Context, task *model.Task) (results []*model.TaskResult, err error) {
	// 确保指纹规则已加载
	s.ensureInit()

	startTime := time.Now()
	var protocolHint string
	if p, ok := task.Params["protocol"].(string); ok {
		protocolHint = p
	}

	ports := parsePortsForScan(task.PortRange)
	if len(ports) == 0 {
		// PortRange 为空或解析不出任何端口：保持 Sprint 0-5 现状行为，只探测一次，
		// 端口交给 normalizeURL 自己按原始字符串处理（原样兼容旧调用方式，
		// 例如 target 本身已经是 "http://xxx" 完整 URL 的场景）。
		return s.runOnePort(ctx, task, startTime, task.PortRange, protocolHint)
	}

	type portResult struct {
		port    int
		results []*model.TaskResult
		err     error
	}

	resultsCh := make(chan portResult, len(ports))
	var wg sync.WaitGroup
	for _, port := range ports {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("[WebScanner] PANIC RECOVERED (port %d): %v", port, r)
					resultsCh <- portResult{port: port, err: fmt.Errorf("panic during web scan on port %d: %v", port, r)}
				}
			}()
			portStr := fmt.Sprintf("%d", port)
			res, errPort := s.runOnePort(ctx, task, startTime, portStr, protocolHint)
			resultsCh <- portResult{port: port, results: res, err: errPort}
		}(port)
	}
	wg.Wait()
	close(resultsCh)

	var errs []error
	for pr := range resultsCh {
		if pr.err != nil {
			logger.Warnf("[WebScanner] port %d scan failed: %v", pr.port, pr.err)
			errs = append(errs, pr.err)
			continue
		}
		results = append(results, pr.results...)
	}

	// 只有全部端口都失败时才把错误往上抛；只要有一个端口拿到结果，就不能让
	// 全局的失败掩盖掉已经成功的部分——这是"一个目标多个独立 Web 服务"场景下
	// 的核心正确性要求，80 端口探测失败不该连累 443 端口的正确结果消失。
	if len(results) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all ports failed: %w", errors.Join(errs...))
	}
	return results, nil
}

// parsePortsForScan 把 task.PortRange 解析成具体端口号列表，并去重。
//
// 直接复用 port_service/nmap_service 包已有的 ParsePortList，不重新发明端口
// 范围解析——它已经支持 "80,443"、"1-100"、"top100"/"top1000" 别名，是经过
// 测试的现成实现。PortRange 为空字符串时返回空列表，调用方 Run() 会据此走
// "只探测一次"的兼容路径。
func parsePortsForScan(portRange string) []int {
	if portRange == "" {
		return nil
	}
	raw := nmap_service.ParsePortList(portRange)
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(raw))
	ports := make([]int, 0, len(raw))
	for _, p := range raw {
		if !seen[p] {
			seen[p] = true
			ports = append(ports, p)
		}
	}
	return ports
}

// runOnePort 对单个端口执行完整的探测流程：调用 crawler.FetchAndCrawl 拿首页
// -> 收口组装结果 -> 按需再次调用 FetchAndCrawl 触发 BFS 深度爬取。
//
// 协议探测/go-rod 渲染/fallback 双发选优这三大段逻辑已经下沉到
// crawler.FetchAndCrawl（见 web扫描模块重构实施文档.md 步骤 2），本函数只
// 保留 WebScanner 自己的业务判断：CDN 是否跳过截图/爬取、截图与 favicon
// 采集（通过 OnPageReady 回调）、指纹匹配结果组装、深度爬取的自动判断。
//
// CrawlDepth 依赖首页响应结果（状态码/Headers/种子链接）才能算出来，而
// FetchAndCrawl 的 CrawlDepth 参数必须在调用前给出，这里按重构实施文档
// 5.2 节的方案 A 分两次调用解决：第一次 CrawlDepth=0 只拿首页，用首页结果
// 算出真实 depth，depth>0 时再调一次触发 BFS（此时不需要重新执行截图/
// favicon 采集，OnPageReady 只在第一次调用时传入）。
func (s *WebScanner) runOnePort(ctx context.Context, task *model.Task, startTime time.Time, port string, protocolHint string) (results []*model.TaskResult, err error) {
	// 0. Panic Recovery (Linus Style: Don't let a single crash take down the whole agent)
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[WebScanner] PANIC RECOVERED: %v", r)
			err = fmt.Errorf("panic during web scan: %v", r)
			results = nil
		}
	}()

	// 1. 获取 QoS 令牌。每次实际抓取各自获取/释放，与 crawler 内部 BFS 每个
	// 子页面各自 Acquire 的模式保持一致——s.limiter 保护的是"全局同时有多少个
	// 昂贵操作（启动浏览器/抓取）在跑"，多端口并发场景下，每个端口的首页抓取
	// 本质上也是"一次抓取"，理应各自受限流保护，不能只在最外层拿一次。
	if err1 := s.limiter.Acquire(ctx); err1 != nil {
		return nil, err1
	}
	defer s.limiter.Release()

	// --- CDN 判断：发起浏览器/HTTP 请求之前，见 Web扫描CDN识别方案.md 第二节 ---
	// checkCDN 内部只做网段查表，isCDN/cdnProvider 这两个局部变量只在本函数
	// 内部用于控制流程（是否跳过截图/深度爬取），组装成 model.EdgeComponent
	// 放进结果的时机在下面 buildWebResult 调用处。
	isCDN, cdnProvider := s.checkCDN(task.Target)

	var screenshotB64, faviconB64 string
	home, _, errFetch := crawler.FetchAndCrawl(ctx, task.Target, port, protocolHint,
		s.limiter, s.browserLauncher, crawler.FetchOptions{
			CrawlDepth: 0, // 首次调用只拿首页，真正的 depth 要等首页响应出来才能算，见下方第二次调用
			OnPageReady: func(page *rod.Page) {
				if capture, ok := task.Params["screenshot"].(bool); ok && capture && !isCDN {
					if buf, errShot := page.Screenshot(true, nil); errShot == nil {
						screenshotB64 = base64.StdEncoding.EncodeToString(buf)
					} else {
						logger.Warnf("[WebScanner] Screenshot failed: %v", errShot)
					}
				}
				faviconB64 = extractFaviconFromPage(page)
			},
		})
	if errFetch != nil {
		s.limiter.OnFailure()
		return nil, errFetch
	}

	finalIP, finalPort := resolveIPPortForResult(task, home.URL, home.RemoteIP, home.RemotePort)

	// Forms/Params 是首页攻击面信息，用已经拿到的 home.Body 统一提取一次，
	// 保证首页结果无论走哪条数据源（go-rod/fallback），Forms/Params 都不缺失。
	_, homeForms, homeParams := crawler.ExtractLinksAndForms(home.URL, home.Body)

	// isCDN/cdnProvider 在这里才第一次组装成 model.EdgeComponent（checkCDN 本身
	// 不依赖 model 包，保持 edge/web 包不反向依赖 model 之外的耦合最小化）。
	var edgeComponents []model.EdgeComponent
	if isCDN {
		edgeComponents = append(edgeComponents, model.EdgeComponent{Type: "cdn", Provider: cdnProvider})
	}

	homeResult := s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
		URL: home.URL, Depth: 0, StatusCode: home.StatusCode, Title: home.Title,
		Body: home.Body, Headers: home.Headers, ContentLength: home.ContentLength, RichContext: home.RichContext,
		Forms: homeForms, Params: homeParams,
		Screenshot: screenshotB64, Favicon: faviconB64,
		EdgeComponents: edgeComponents,
	})
	results = append(results, homeResult)

	// --- 是否触发深度爬取：三态判断，见 resolveCrawlDepth ---
	//
	// 这里不通过 FetchAndCrawl 的 CrawlDepth 参数触发 BFS（那会导致
	// WebScanner 拿不到 FetchAndCrawl 内部创建的 *crawler.Crawler 实例，
	// 而 escalateIfNeeded 需要用它调用 EnqueueExtra 把升级渲染后新发现的
	// 链接塞回同一个 Crawler 继续追踪，见该函数注释），而是像重构前一样
	// 自己持有 Crawler 实例——直接调用本包已经导出的 New/Crawl，两者是
	// 平级的公开能力，不是只能通过 FetchAndCrawl 才能访问。
	depth := s.resolveCrawlDepth(task, home.StatusCode, home.Headers, home.SeedLinks)
	if depth > 0 && len(home.SeedLinks) > 0 && !isCDN {
		cr := crawler.New(crawler.Options{MaxDepth: depth}, s.limiter)
		subPages := cr.Crawl(ctx, home.URL, home.SeedLinks)

		s.escalateIfNeeded(ctx, cr, subPages)

		for _, p := range subPages {
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
	if ip == "" && utils.IsIP(task.Target) {
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

// checkCDN 判断 target（可能是域名，也可能已经是 IP）是否落在已知 CDN
// 厂商的网段内。target 是域名时会做一次 DNS 解析取第一个 IP；解析失败
// 时直接返回"不是 CDN"，不阻塞后续流程——CDN 判断是锦上添花的优化，
// 不能因为 DNS 解析失败就影响正常扫描（见方案文档第二节）。
func (s *WebScanner) checkCDN(target string) (bool, string) {
	ip := target
	if !utils.IsIP(target) {
		addrs, err := net.LookupHost(target)
		if err != nil || len(addrs) == 0 {
			return false, ""
		}
		ip = addrs[0]
	}
	return s.edgeDetector.Check(ip)
}

// extractFaviconFromPage 从已经导航完成的页面里取 favicon 并转成 base64。
//
// 这段逻辑在重构前是内联在 Run() 里的（现状第 234-260 行），抽成独立函数纯粹是
// 为了让 Run() 主干的可读性不被这段和主流程无关的细节拖累。行为上有一处必要
// 调整：重构前依赖调用方已经跑过一次 ExtractRichContext、把 favicon_url 放进
// richCtx 再传进来；重构后（web扫描模块重构实施文档.md 步骤 4）favicon 采集
// 是在 crawler.FetchAndCrawl 的 OnPageReady 回调里触发的，此时 richCtx 是
// crawler 包内部的私有局部变量，WebScanner 拿不到，因此这里改为在 page.Eval
// 里现查一次 favicon_url（和 ExtractRichContext 内部查询用的是同一段 JS），
// 查询结果不落盘、不进入任何返回结构体，只是这一个函数内部的一步中间计算，
// 找到 URL 后立刻在同一个 Eval 环境里 fetch 并转 base64。任何一步失败都返回
// 空字符串，不影响调用方（favicon 是锦上添花的信息，不是关键路径）。
func extractFaviconFromPage(page *rod.Page) string {
	res, err := page.Eval(`() => {
		let link = document.querySelector("link[rel*='icon']");
		const url = link ? link.href : "";
		if (!url) {
			return "";
		}
		return fetch(url)
			.then(response => response.blob())
			.then(blob => new Promise((resolve, reject) => {
				const reader = new FileReader();
				reader.onloadend = () => resolve(reader.result); // data:image/png;base64,...
				reader.onerror = reject;
				reader.readAsDataURL(blob);
			}))
			.catch(() => "");
	}`)
	if err != nil {
		return ""
	}

	dataURL := res.Value.String()
	if idx := strings.Index(dataURL, ","); idx != -1 {
		return dataURL[idx+1:]
	}
	return ""
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

		// 加载 CDN 网段规则，路径试探列表与指纹规则同一套相对路径模式。
		edgePaths := []string{
			"rules/edge/cdn.json",
			"../rules/edge/cdn.json",
			"../../rules/edge/cdn.json",
			"../../../rules/edge/cdn.json",
			"../../../../rules/edge/cdn.json",
			"../../../../../rules/edge/cdn.json",
			"neoAgent/rules/edge/cdn.json",
		}
		for _, path := range edgePaths {
			if err := s.edgeDetector.Load(path); err == nil {
				logger.Infof("[WebScanner] Loaded CDN rules from %s", path)
				break
			}
		}
		// 全部路径都加载失败：edgeDetector 内部规则集为空，Check 恒定返回
		// false，退化为"没有 CDN 识别能力"，不影响扫描主流程，不需要额外处理。
	})
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

	// EdgeComponents 只有首页调用 buildWebResult 时才会显式传入非空值，
	// 爬虫子页面复用同一个端口的判断结果没有意义（同一端口的边缘节点归属
	// 不会因为爬到第几层页面而改变），子页面调用处保持 nil 即可。
	EdgeComponents []model.EdgeComponent
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
			EdgeComponents:  pd.EdgeComponents,
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

	richCtx, err := crawler.ExtractRichContext(page)
	if err != nil {
		return "", nil, false
	}
	body, _ = richCtx["body"].(string)
	if body == "" {
		return "", nil, false
	}
	links = crawler.ExtractLinks(page)
	return body, links, true
}
