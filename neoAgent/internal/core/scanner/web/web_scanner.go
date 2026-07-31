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

// Run 执行扫描任务
func (s *WebScanner) Run(ctx context.Context, task *model.Task) (results []*model.TaskResult, err error) {
	// 0. Panic Recovery (Linus Style: Don't let a single crash take down the whole agent)
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[WebScanner] PANIC RECOVERED: %v", r)
			// 打印堆栈信息，方便定位 Segfault/Panic 位置
			// debug.Stack() 需要 import runtime/debug
			// 简单起见，我们至少返回一个错误结果，而不是让进程崩溃
			err = fmt.Errorf("panic during web scan: %v", r)
			// 尝试返回一个失败的任务结果
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
	// 尝试从 Params 中获取协议提示 (http/https)
	var protocolHint string
	if p, ok := task.Params["protocol"].(string); ok {
		protocolHint = p
	}
	targetURL := normalizeURL(task.Target, task.PortRange, protocolHint)

	// 2. 启动浏览器 (Lazy Load)
	// 这里我们每次 Scan 都尝试 Launch，Launch 内部会复用已启动的 Browser
	// TODO: 支持从 Task 参数中读取 Proxy
	br, err := s.browserLauncher.Launch(ctx)
	if err != nil {
		logger.Warnf("[WebScanner] Failed to launch browser: %v. Falling back to HTTP client.", err)
		res, errFallback := s.runFallback(ctx, task, targetURL, startTime)
		if errFallback != nil {
			s.limiter.OnFailure()
			return nil, fmt.Errorf("browser launch failed (%v) and fallback failed: %w", err, errFallback)
		}
		s.limiter.OnSuccess()
		return res, nil
	}

	// 3. 打开空白页面并设置监听
	// 我们先打开空白页，设置好事件监听，然后再 Navigate，这样能捕获到完整的网络请求
	page, err := s.browserLauncher.OpenPage(ctx, br, "")
	if err != nil {
		s.limiter.OnFailure()
		return nil, fmt.Errorf("failed to open page: %w", err)
	}
	defer page.Close()

	// 监听网络响应，提取 IP, Port, Status, Headers
	var (
		remoteIP      string
		remotePort    int
		statusCode    int
		contentLength int64
		respHeaders   = make(map[string]string)
		respMutex     sync.Mutex
	)

	// 启用 Network 域
	// page.MustWaitOpen() // 确保页面已打开? OpenPage 已经返回了 page
	// 开启网络事件监听
	// 使用 page.EachEvent 监听事件 (非阻塞模式运行)
	// 注意: 需要确保 page.Close() 会停止监听，或者我们应该手动 stop
	waitEvents := page.EachEvent(func(e *proto.NetworkResponseReceived) bool {
		// 我们主要关注 Document 类型的响应
		// 对于 SPA 页面，初始 HTML 可能只是一个骨架，但也算 Document
		// 某些情况下，重定向后的最终页面才是我们想要的

		// 调试日志: 打印所有响应类型和URL
		// logger.Debugf("[WebScanner] Response: %s %s %s", e.Type, e.Response.URL, e.Response.Status)

		if e.Type == proto.NetworkResourceTypeDocument {
			logger.Infof("[WebScanner] Document Response: Status=%d URL=%s", e.Response.Status, e.Response.URL)
			respMutex.Lock()
			defer respMutex.Unlock()

			// 只有当响应状态码有效时才更新 (排除 0 或 weird status)
			// 或者，我们记录最后一次 Document 响应
			statusCode = e.Response.Status
			remoteIP = e.Response.RemoteIPAddress
			if e.Response.RemotePort != nil {
				remotePort = *e.Response.RemotePort
			}

			// Headers
			for k, v := range e.Response.Headers {
				var val string
				if err1 := json.Unmarshal([]byte(v.String()), &val); err1 == nil {
					respHeaders[k] = val
				} else {
					respHeaders[k] = v.String()
				}
			}

			// Content-Length
			if cl, ok := e.Response.Headers["Content-Length"]; ok {
				var clVal string
				if err2 := json.Unmarshal([]byte(cl.String()), &clVal); err2 == nil {
					fmt.Sscanf(clVal, "%d", &contentLength)
				}
			} else {
				// 如果 Header 里没有，尝试使用 EncodedDataLength
				// 注意: 这可能不准确，因为它是传输长度
				if e.Response.EncodedDataLength > 0 {
					contentLength = int64(e.Response.EncodedDataLength)
				}
			}
		}
		return false
	})
	go waitEvents()
	// defer stop() // EachEvent 会在 page 关闭时自动停止，或者我们可以手动控制，这里简单起见让它随 page 生命周期

	// 4. 导航到目标 URL
	if err3 := page.Navigate(targetURL); err3 != nil {
		logger.Warnf("[WebScanner] Navigation failed for %s: %v. Falling back to HTTP client.", targetURL, err3)
		res, err1 := s.runFallback(ctx, task, targetURL, startTime)
		if err1 != nil {
			s.limiter.OnFailure()
			return nil, fmt.Errorf("navigation and fallback failed: %w", err1)
		}
		s.limiter.OnSuccess()
		return res, nil
	}

	// 5. 等待加载完成
	// 使用 MustWaitLoad 等待页面加载完成 (network idle)
	// 设置超时，防止挂死
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 注意: page.Timeout 会返回一个新的 page 对象，需要链式调用
	if err4 := page.Context(waitCtx).WaitLoad(); err4 != nil {
		logger.Warnf("[WebScanner] WaitLoad timeout for %s: %v", targetURL, err4)
		// 超时也继续尝试提取，因为可能部分加载了
	}

	// 6. 提取 Rich Context (DOM/JS/Meta)
	richCtx, err := ExtractRichContext(page)
	if err != nil {
		logger.Warnf("[WebScanner] Failed to extract rich context: %v", err)
	}

	// 7. 提取正文（供 buildWebResult 组装 fingerprint.Input.Body 使用）
	var pageBody string
	if body, ok := richCtx["body"].(string); ok {
		pageBody = body
	}

	// 8. 截图 (如果启用)
	var screenshotBase64 string
	if capture, ok := task.Params["screenshot"].(bool); ok && capture {
		if buf, err := page.Screenshot(true, nil); err == nil {
			screenshotBase64 = base64.StdEncoding.EncodeToString(buf)
		} else {
			logger.Warnf("[WebScanner] Screenshot failed: %v", err)
		}
	}

	// 9. 获取 Favicon
	var faviconBase64 string
	if favURL, ok := richCtx["favicon_url"].(string); ok && favURL != "" {
		// 尝试获取资源
		// 注意: GetResource 可能需要资源已经被加载过
		// 如果是外部链接，可能需要 page.Eval fetch
		// 简单尝试: 使用 page.GetResourceContent (如果缓存中有)
		// 或者直接 Eval fetch
		// 这里使用一个通用的 JS fetch 转 base64 方法
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

		if err == nil {
			// 结果是 data URL，需要去掉前缀
			dataURL := res.Value.String()
			if idx := strings.Index(dataURL, ","); idx != -1 {
				faviconBase64 = dataURL[idx+1:]
			}
		}
	}

	// 10. 构造结果
	// 获取网络信息快照 (Thread Safety)
	respMutex.Lock()
	finalStatusCode := statusCode
	finalContentLength := contentLength
	finalIP := remoteIP
	finalPort := remotePort
	finalHeaders := make(map[string]string)
	for k, v := range respHeaders {
		finalHeaders[k] = v
	}
	respMutex.Unlock()

	// 兜底 IP/Port
	if finalIP == "" {
		// 尝试从 task 解析? 或者直接用 Target (如果是 IP)
		// 这里留空，让 Master 端去 resolve 或者后续处理
		// 为了满足契约，如果是 IP 形式的 Target，可以直接填
		if isIP(task.Target) {
			finalIP = task.Target
		}
	}
	if finalPort == 0 {
		// 优先尝试从 URL 中解析端口
		if u, err := url.Parse(targetURL); err == nil {
			if port := u.Port(); port != "" {
				fmt.Sscanf(port, "%d", &finalPort)
			} else {
				// 如果 URL 没写端口，根据协议推断
				if u.Scheme == "https" {
					finalPort = 443
				} else if u.Scheme == "http" {
					finalPort = 80
				}
			}
		}

		// 如果还是 0，尝试从 task.PortRange 解析 (作为最后的兜底)
		if finalPort == 0 && task.PortRange != "" {
			// 只有当 PortRange 是单个端口时才采纳，避免 "80,443" 这种列表被解析为 80
			if !strings.Contains(task.PortRange, ",") && !strings.Contains(task.PortRange, "-") {
				fmt.Sscanf(task.PortRange, "%d", &finalPort)
			}
		}
	}

	// go-rod 路径的首页也统一走 buildWebResult 收口，不再自己重复一遍
	// "构造 fingerprint.Input -> 匹配指纹 -> 组装 WebResult"的逻辑。
	// RichContext 显式传入完整的 richCtx（含 DOM/JS 变量/Meta/Cookies），
	// 这是 go-rod 路径相比 fallback/爬虫路径的核心优势，必须保留，否则
	// 依赖这些字段的指纹规则会在这次收口重构后失效。
	result := s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
		URL:           targetURL,
		StatusCode:    finalStatusCode,
		Title:         extractTitleFromCtx(richCtx),
		Body:          pageBody,
		Headers:       finalHeaders,
		ContentLength: finalContentLength,
		RichContext:   richCtx,
		Screenshot:    screenshotBase64,
		Favicon:       faviconBase64,
	})

	s.limiter.OnSuccess()
	return []*model.TaskResult{result}, nil
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

// runFallback 是 Run() 主干里两处降级路径（浏览器启动失败 / 导航失败）共用的
// 桥接函数：调用 fallbackFetch 完成抓取，再调用 buildWebResult 完成组装，
// 拼成 Run() 现阶段仍然需要的 ([]*model.TaskResult, error) 返回形状。
//
// 为什么需要这一层桥接，而不是让 Run() 直接分别调用 fallbackFetch 和
// buildWebResult 两个函数：Sprint 4 明确不改 Run() 的对外行为和主干结构
// （那是 Sprint 5 的任务），Run() 里这两处调用点在重构前后都应该是
// "一行代码换取一个完整结果"的形态，不应该在 Sprint 4 阶段就把 Run() 主干
// 展开成多行组装代码——那样等于提前做了本该由 Sprint 5 做的主干重排，
// 反而增加了这次改动的影响面。runFallback 只是把 "fetch + build" 这两步
// 钉在一起，交换到 Run() 眼里和原来调用 fallbackScan 没有区别。
func (s *WebScanner) runFallback(ctx context.Context, task *model.Task, targetURL string, startTime time.Time) ([]*model.TaskResult, error) {
	body, headers, statusCode, title, _, err := s.fallbackFetch(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	// Forms/Params 是首页攻击面信息，fallback 路径下顺手用已经抓到的 body 提取一遍，
	// 和 crawler 包对子页面的处理方式保持一致，不让 fallback 路径的首页数据比
	// 爬虫子页面的数据更"贫瘠"。这里没有直接用 fallbackFetch 已经返回的 links——
	// 那是同一次 ExtractLinksAndForms 调用的另外两个返回值，本该一次拿全，
	// 但 fallbackFetch 的签名（对齐实施文档，也是 Sprint 5 要复用的形状）只对外
	// 暴露 links，不暴露 forms/params，所以这里只能就着 body 再解析一遍。
	// 多付出的这次 DOM 解析成本，只发生在"浏览器不可用，降级到 fallback"这个
	// 本就是异常路径的场景，不是每次扫描都要付出的代价，可以接受。
	_, forms, params := crawler.ExtractLinksAndForms(targetURL, body)

	finalIP := ""
	finalPort := 0
	if isIP(task.Target) {
		finalIP = task.Target
	}
	if task.PortRange != "" {
		fmt.Sscanf(task.PortRange, "%d", &finalPort)
	}

	result := s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
		URL:        targetURL,
		StatusCode: statusCode,
		Title:      title,
		Body:       body,
		Headers:    headers,
		Forms:      forms,
		Params:     params,
	})

	logger.Infof("[WebScanner] Fallback scan success for %s", targetURL)
	return []*model.TaskResult{result}, nil
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
