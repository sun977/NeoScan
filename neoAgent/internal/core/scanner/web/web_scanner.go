package web

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"neoagent/internal/core/scanner/port_service/nmap_service"
	"neoagent/internal/core/scanner/web/crawler"
	"neoagent/internal/pkg/edge"
	"neoagent/internal/pkg/fingerprint"
	fpHttp "neoagent/internal/pkg/fingerprint/engines/http"
	fpModel "neoagent/internal/pkg/fingerprint/model"
	"neoagent/internal/pkg/logger"
	"neoagent/internal/pkg/utils"
	"net"

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

// runOnePort 对单个端口执行完整的探测流程：猜/确定协议 -> go-rod 抓取 ->
// 降级 fallback（含 http/https 双发选优）-> 收口组装结果 -> 按需触发 BFS 深度爬取。
//
// 这是 Sprint 0-5 里 Run() 函数体的原样内容，整体结构"统一收口一次 return"
// 保持不变：不管首页数据是 go-rod 拿到的还是降级 fallback 拿到的，最终都在
// 函数末尾组装一次结果、判断一次是否需要深度爬取、return 一次——不再像
// 重构前那样在浏览器启动失败/导航失败两处提前 return，那两处提前 return
// 各自绕过了后面的 BFS 触发逻辑，是"降级路径爬虫失效"缺陷的根源（架构方案
// 8.7 节）。现在不管走到哪条路径，body 一旦拿到手，后面的收口、决策、BFS
// 都是同一段代码，不存在"某条路径漏掉某个步骤"的可能。
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

	protocolGuessed := isProtocolGuessed(task.Target, protocolHint)
	targetURL := normalizeURL(task.Target, port, protocolHint)

	// --- CDN 判断：发起浏览器/HTTP 请求之前，见 Web扫描CDN识别方案.md 第二节 ---
	// checkCDN 内部只做网段查表，isCDN/cdnProvider 这两个局部变量只在本函数
	// 内部用于控制流程（是否跳过截图/深度爬取），组装成 model.EdgeComponent
	// 放进结果的时机在下面 buildWebResult 调用处。
	isCDN, cdnProvider := s.checkCDN(task.Target)

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

					if capture, ok := task.Params["screenshot"].(bool); ok && capture && !isCDN {
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

	// --- 统一降级 + 协议误判纠正：go-rod 路径拿到的结果，在两种情况下不能
	// 直接采信，必须用 fallbackFetchBestProtocol 的双发结果来比较、选优：
	//
	//  1. go-rod 完全没拿到 body（Launch/Navigate/ExtractRichContext 任一环节
	//     失败）——这是重构前就有的"降级路径爬虫失效"场景，取代了原来分散在
	//     Launch 失败、Navigate 失败两处的降级调用。
	//
	//  2. go-rod 拿到了 400 响应，且协议是"猜"出来的——真实 Chrome 环境验证
	//     过，协议猜错时（比如把 HTTPS 服务当成 HTTP 请求）Chromium 不会让
	//     导航失败：Go 的 TLS 服务端在握手阶段识别出明文请求后，会在同一条
	//     TCP 连接上直接写回一段合法的 HTTP/1.0 400 文本响应（"Client sent
	//     an HTTP request to an HTTPS server."），这是服务端真实吐出的、
	//     状态码明确的响应，不是 Chromium 编造的错误页，go-rod 路径因此
	//     "成功"拿到一个看似合法、实际是协议不匹配提示的 body。400 本身
	//     不能无条件当成失败（有些服务器对正常请求就是回 400），但协议是
	//     猜出来的场景下，400 已经是"猜错了协议"的强烈信号，值得让另一个
	//     协议也发一次、用响应质量客观比较，而不是不假思索采信。
	//
	// 两种情况统一走同一条路径：都发起双发对比，用 pickBestFetchOutcome 在
	// go-rod 已有结果（包装成 fetchOutcome，body 为空时天然在排序中垫底）
	// 和 fallback 双发结果之间选出更可信的一个。不给"go-rod 完全失败"和
	// "go-rod 拿到疑似协议错误的 400"写两套不同的分支逻辑——它们本质上是
	// 同一个问题："当前拿到的 go-rod 结果值不值得信任，需不需要用 fallback
	// 双发来验证/替换"，用一个排序函数统一回答，不需要特殊情况。
	needsVerification := homeBody == "" ||
		(protocolGuessed && homeStatusCode == http.StatusBadRequest)

	if needsVerification {
		rodOutcome := fetchOutcome{
			url: targetURL, body: homeBody, headers: homeHeaders,
			statusCode: homeStatusCode, title: homeTitle, links: seedLinks,
		}
		if homeBody == "" {
			// go-rod 侧没有任何可用数据，用一个带 err 的哨兵值代表它，
			// pickBestFetchOutcome 的排序规则下必然输给任何成功的 fallback
			// 结果，不需要为"完全没数据"单独判断。
			rodOutcome.err = errors.New("go-rod path yielded no body")
		}

		altBody, altHeaders, altStatusCode, altTitle, altLinks, altURL, errFetch := s.fallbackFetchBestProtocol(ctx, targetURL, protocolGuessed)
		altOutcome := fetchOutcome{url: altURL, body: altBody, headers: altHeaders, statusCode: altStatusCode, title: altTitle, links: altLinks, err: errFetch}

		best := pickBestFetchOutcome(rodOutcome, altOutcome)
		if best.err != nil {
			// 两侧都失败：go-rod 没数据，fallback 双发也失败，彻底没有可用结果。
			s.limiter.OnFailure()
			return nil, fmt.Errorf("both browser and fallback fetch failed: %w", best.err)
		}
		if best.url == altOutcome.url {
			// fallback 侧胜出：go-rod 那份结果（如果有的话）是基于错误协议
			// 渲染的，连同截图/favicon 这些衍生物一并丢弃，避免"错误协议的
			// 截图 + 正确协议的正文"这种不自洽的组合。
			targetURL = altURL
			homeBody, homeHeaders, homeStatusCode, homeTitle, seedLinks = altBody, altHeaders, altStatusCode, altTitle, altLinks
			homeRichCtx = nil
			homeContentLen = 0
			screenshotB64, faviconB64 = "", ""
		}
		// best 是 go-rod 侧：说明 go-rod 原有结果已经足够可信（比如两个协议
		// 下都是 400，或 fallback 双发反而更差），维持不变，保留 RichContext/
		// 截图这些只有 go-rod 路径才有的更完整数据。
	}

	finalIP, finalPort := resolveIPPortForResult(task, targetURL, remoteIP, remotePort)

	// Forms/Params 是首页攻击面信息。go-rod 路径没有顺手提取过（ExtractLinks 只管链接），
	// fallback 路径也一样（fallbackFetch 同理只返回 links）。这里用已经拿到的 homeBody
	// 统一提取一次，保证首页结果无论走哪条数据源，Forms/Params 都不缺失。
	_, homeForms, homeParams := crawler.ExtractLinksAndForms(targetURL, homeBody)

	// isCDN/cdnProvider 在这里才第一次组装成 model.EdgeComponent（checkCDN 本身
	// 不依赖 model 包，保持 edge/web 包不反向依赖 model 之外的耦合最小化）。
	var edgeComponents []model.EdgeComponent
	if isCDN {
		edgeComponents = append(edgeComponents, model.EdgeComponent{Type: "cdn", Provider: cdnProvider})
	}

	homeResult := s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
		URL: targetURL, Depth: 0, StatusCode: homeStatusCode, Title: homeTitle,
		Body: homeBody, Headers: homeHeaders, ContentLength: homeContentLen, RichContext: homeRichCtx,
		Forms: homeForms, Params: homeParams,
		Screenshot: screenshotB64, Favicon: faviconB64,
		EdgeComponents: edgeComponents,
	})
	results = append(results, homeResult)

	// --- 是否触发深度爬取：三态判断，见 resolveCrawlDepth ---
	depth := s.resolveCrawlDepth(task, homeStatusCode, homeHeaders, seedLinks)
	if depth > 0 && len(seedLinks) > 0 && !isCDN {
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

// isProtocolGuessed 判断 normalizeURL 对这次调用是不是"猜"出协议的，而不是
// 用户显式指定的——只有"猜"出来的协议才值得在 fallback 阶段做双发验证，用户
// 显式指定的协议就该只发一次，尊重用户明确的意图，不做自作主张的纠正。
//
// 判断逻辑必须和 normalizeURL 内部的分支条件完全对应：target 自带协议前缀、
// 或者 protocol 参数非空，都是"显式"；只有落进 normalizeURL 最后 switch 的
// 默认猜测分支，才是"猜"。这里不修改 normalizeURL 本身的签名（避免影响
// 唯一现有调用方之外可能存在的隐性契约），而是新增一个纯判断函数，用同样的
// 输入复现同样的分支路径。
func isProtocolGuessed(target string, protocol string) bool {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return false
	}
	return protocol != "https" && protocol != "http"
}

// flipProtocol 把 URL 的 scheme 在 http/https 之间翻转，其余部分（host、port、
// path）原样保留。
func flipProtocol(targetURL string) string {
	if strings.HasPrefix(targetURL, "https://") {
		return "http://" + strings.TrimPrefix(targetURL, "https://")
	}
	if strings.HasPrefix(targetURL, "http://") {
		return "https://" + strings.TrimPrefix(targetURL, "http://")
	}
	return targetURL
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
// fallbackFetchBestProtocol 是 fallbackFetch 的协议自适应外壳，参考 httpx 的
// 做法：协议不确定时，不去猜错了再重试，而是 http/https 两个协议直接并发各发
// 一次，用响应质量选出更靠谱的那个。
//
// 这个方案取代了最初设计的"先猜一个协议，抓取失败后再按错误类型判断要不要
// 换协议重试"——那个方案在真实 Chrome/Chromium 环境下失效了：协议猜错时
// （比如把 HTTPS 服务当成 HTTP 请求），Go 的 net/http 客户端在明文链路上收到
// TLS 服务端返回的裸文本响应，会被 http.Client 解析成一个"看似成功"的
// http.Response（因为 HTTP 响应本身就是纯文本协议，TLS 握手失败时对端吐出的
// 错误文本凑巧也能被解析成合法的状态行），err 是 nil，走不到任何"基于错误
// 类型判断"的重试分支。想要可靠识别这种情况，要么去解析 body 内容做启发式
// 判断（本质上是给"猜错协议"这个明确可以避免的场景打补丁，增加了不必要的
// 特殊情况），要么就是干脆不猜——两个协议都发，永远能拿到「真正尝试过」的
// 结果，用响应质量客观排序，不需要判断"是不是猜错了"这种本身就模糊的问题。
//
// protocolGuessed 为 false（用户通过 target 前缀或 protocol 参数显式指定协议）
// 时只发 targetURL 这一个请求，尊重用户的明确意图，不做双发。
func (s *WebScanner) fallbackFetchBestProtocol(ctx context.Context, targetURL string, protocolGuessed bool) (body string, headers map[string]string, statusCode int, title string, links []string, finalURL string, err error) {
	if !protocolGuessed {
		body, headers, statusCode, title, links, err = s.fallbackFetch(ctx, targetURL)
		return body, headers, statusCode, title, links, targetURL, err
	}

	altURL := flipProtocol(targetURL)
	if altURL == targetURL {
		// 不是标准的 http(s):// URL（理论上不会发生，normalizeURL 保证了 scheme），
		// 双发无意义，退化成单发。
		body, headers, statusCode, title, links, err = s.fallbackFetch(ctx, targetURL)
		return body, headers, statusCode, title, links, targetURL, err
	}

	var wg sync.WaitGroup
	outcomes := make([]fetchOutcome, 2)
	urls := [2]string{targetURL, altURL}
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			b, h, sc, t, l, e := s.fallbackFetch(ctx, u)
			outcomes[i] = fetchOutcome{url: u, body: b, headers: h, statusCode: sc, title: t, links: l, err: e}
		}(i, u)
	}
	wg.Wait()

	best := pickBestFetchOutcome(outcomes[0], outcomes[1])
	if best.err != nil {
		return "", nil, 0, "", nil, targetURL, best.err
	}
	return best.body, best.headers, best.statusCode, best.title, best.links, best.url, nil
}

// fetchOutcome 是一次 fallbackFetch 调用的完整结果，用于双发场景下在两个
// 协议之间做比较、选优。
type fetchOutcome struct {
	url        string
	body       string
	headers    map[string]string
	statusCode int
	title      string
	links      []string
	err        error
}

// pickBestFetchOutcome 从两个协议的抓取结果里选出更可信的一个，排序规则与
// httpx 的做法一致——优先级从高到低：
//  1. 请求成功（err == nil）且状态码不是"协议不匹配的典型特征"（4xx 里的 400，
//     纯文本协议服务器收到 TLS ClientHello 或反过来最常见的表现）。
//  2. 请求成功但状态码是 400：仍然是"拿到了响应"，比彻底失败强，但排在
//     "干净成功"后面。
//  3. 请求失败（err != nil）：排最后，两个都失败时返回先发起的那个的错误
//     （即 targetURL 对应的 outcome），错误信息里包含的 host:port 对用户
//     排查更有意义。
//
// 这里不做"看 body 内容像不像 HTML"这类启发式判断——状态码已经是服务端
// 主动给出的明确信号，没有必要绕过明确信号去猜内容，那是本末倒置。
func pickBestFetchOutcome(a, b fetchOutcome) fetchOutcome {
	rank := func(o fetchOutcome) int {
		switch {
		case o.err == nil && o.statusCode != http.StatusBadRequest:
			return 0
		case o.err == nil:
			return 1
		default:
			return 2
		}
	}
	if rank(a) <= rank(b) {
		return a
	}
	return b
}

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
