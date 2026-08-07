package crawler

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/pkg/logger"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// FetchOptions 控制 FetchAndCrawl 的抓取行为。
type FetchOptions struct {
	ProtocolHint string // 复用现有 protocolHint 语义：""(自动猜)/"http"/"https"
	CrawlDepth   int    // <=0 表示不做 BFS 深度爬取，仅抓首页

	// OnPageReady 可选回调：仅当 go-rod 渲染成功、导航完成、WaitLoad 之后、
	// page.Close() 之前调用一次，把 *rod.Page 原样交给调用方。
	//
	// crawler 包完全不关心回调里做了什么——WebScanner 传入截图/favicon 采集逻辑，
	// ApiScanner 现在传 nil，未来若要做"动态渲染后捕获真实 XHR/Fetch"，
	// 直接在回调里挂 CDP page.EachEvent 网络监听即可，不需要改这里的函数签名。
	//
	// 边界约束（web扫描模块重构实施文档.md 零、第 2 条硬约束）：图片/二进制类的
	// 展示衍生品永远只通过这个回调向外流动，不进入 HomePage 返回值结构，
	// 这条边界不能松动。
	OnPageReady func(page *rod.Page)
}

// HomePage 是 FetchAndCrawl 的首页产出，只包含"数据"，不包含任何图片字段。
type HomePage struct {
	URL           string
	StatusCode    int
	Title         string
	Body          string
	Headers       map[string]string
	ContentLength int64
	RichContext   map[string]interface{}
	SeedLinks     []string
	RemoteIP      string
	RemotePort    int
}

// FetchAndCrawl 统一入口：拿首页 + （可选）BFS 深度爬取子页面。
// WebScanner 和 ApiScanner 都只需要认识这一个函数，具体实现是
// web_scanner.go 原 runOnePort 里协议探测/go-rod 渲染/fallback 逻辑的
// 原样搬迁，见 web扫描模块重构实施文档.md 第三节的迁移清单，逻辑不变。
//
// launcher 由调用方（WebScanner/ApiScanner）各自持有并显式传入（方案文档
// 3.4 节方案 A）：BrowserLauncher 本身无状态，调用方各自持有一份实例成本
// 很低，crawler 包不反向 import browser 并维护包级单例。
//
// CrawlDepth<=0 时只抓首页，subPages 返回 nil；CrawlDepth>0 时首页抓取
// 完成后，若拿到了种子链接，会额外触发一次 BFS（本包已有的 New/Crawl）。
// 调用方如果需要"先看首页结果再决定要不要爬"（比如 WebScanner 的自动深度
// 判断），按 CrawlDepth=0 与 CrawlDepth>0 分两次调用即可，见重构实施文档
// 5.2 节。
func FetchAndCrawl(ctx context.Context, target, port, protocolHint string,
	limiter *qos.AdaptiveLimiter, launcher *browser.BrowserLauncher, opts FetchOptions) (home *HomePage, subPages []*Page, err error) {

	protocolGuessed := isProtocolGuessed(target, protocolHint)
	targetURL := normalizeURL(target, port, protocolHint)

	var (
		homeBody       string
		homeHeaders    map[string]string
		homeStatusCode int
		homeContentLen int64
		homeTitle      string
		homeRichCtx    map[string]interface{}
		seedLinks      []string
		remoteIP       string
		remotePort     int
	)

	// --- go-rod 路径：Launch -> OpenPage -> 监听网络 -> Navigate -> WaitLoad -> ExtractRichContext ---
	if br, errLaunch := launcher.Launch(ctx); errLaunch == nil {
		if page, errOpen := launcher.OpenPage(ctx, br, ""); errOpen == nil {
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
					logger.Warnf("[crawler] WaitLoad timeout for %s: %v", targetURL, errWait)
				}
				cancel()

				richCtx, errCtx := ExtractRichContext(page)
				if errCtx == nil {
					homeRichCtx = richCtx
					homeBody, _ = richCtx["body"].(string)
					homeTitle = extractTitleFromCtx(richCtx)
					seedLinks = ExtractLinks(page)

					if opts.OnPageReady != nil {
						opts.OnPageReady(page)
					}
				} else {
					logger.Warnf("[crawler] Failed to extract rich context: %v", errCtx)
				}
			} else {
				logger.Warnf("[crawler] Navigation failed for %s: %v. Will fallback.", targetURL, errNav)
			}
		}
	} else {
		logger.Warnf("[crawler] Failed to launch browser: %v. Will fallback.", errLaunch)
	}

	// --- 统一降级 + 协议误判纠正：见 web_scanner.go 原 runOnePort 的详细注释
	// （web扫描模块重构实施文档.md 3.2 节第 298~359 行迁移条目），逻辑原样保留。
	needsVerification := homeBody == "" ||
		(protocolGuessed && homeStatusCode == http.StatusBadRequest)

	if needsVerification {
		rodOutcome := fetchOutcome{
			url: targetURL, body: homeBody, headers: homeHeaders,
			statusCode: homeStatusCode, title: homeTitle, links: seedLinks,
		}
		if homeBody == "" {
			rodOutcome.err = errors.New("go-rod path yielded no body")
		}

		altBody, altHeaders, altStatusCode, altTitle, altLinks, altURL, errFetch := fallbackFetchBestProtocol(ctx, targetURL, protocolGuessed)
		altOutcome := fetchOutcome{url: altURL, body: altBody, headers: altHeaders, statusCode: altStatusCode, title: altTitle, links: altLinks, err: errFetch}

		best := pickBestFetchOutcome(rodOutcome, altOutcome)
		if best.err != nil {
			return nil, nil, fmt.Errorf("both browser and fallback fetch failed: %w", best.err)
		}
		if best.url == altOutcome.url {
			// fallback 侧胜出：go-rod 那份结果（如果有的话）是基于错误协议
			// 渲染的，连同 RichContext 一并丢弃，避免"错误协议的渲染结果 +
			// 正确协议的正文"这种不自洽的组合。
			targetURL = altURL
			homeBody, homeHeaders, homeStatusCode, homeTitle, seedLinks = altBody, altHeaders, altStatusCode, altTitle, altLinks
			homeRichCtx = nil
			homeContentLen = 0
		}
	}

	home = &HomePage{
		URL:           targetURL,
		StatusCode:    homeStatusCode,
		Title:         homeTitle,
		Body:          homeBody,
		Headers:       homeHeaders,
		ContentLength: homeContentLen,
		RichContext:   homeRichCtx,
		SeedLinks:     seedLinks,
		RemoteIP:      remoteIP,
		RemotePort:    remotePort,
	}

	if opts.CrawlDepth > 0 && len(seedLinks) > 0 {
		cr := New(Options{MaxDepth: opts.CrawlDepth}, limiter)
		subPages = cr.Crawl(ctx, targetURL, seedLinks)
	}

	return home, subPages, nil
}

// normalizeURL 简单的 URL 规范化。原样从 web_scanner.go 迁移，逻辑不变。
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
// 默认猜测分支，才是"猜"。
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

func extractTitleFromCtx(ctx map[string]interface{}) string {
	if t, ok := ctx["title"].(string); ok {
		return t
	}
	return ""
}

// fallbackFetchBestProtocol 是 fallbackFetch 的协议自适应外壳，参考 httpx 的
// 做法：协议不确定时，不去猜错了再重试，而是 http/https 两个协议直接并发各发
// 一次，用响应质量选出更靠谱的那个。原样从 web_scanner.go 迁移，逻辑不变，
// 详细背景说明见该文件历史版本的同名函数注释。
//
// protocolGuessed 为 false（用户通过 target 前缀或 protocol 参数显式指定协议）
// 时只发 targetURL 这一个请求，尊重用户的明确意图，不做双发。
func fallbackFetchBestProtocol(ctx context.Context, targetURL string, protocolGuessed bool) (body string, headers map[string]string, statusCode int, title string, links []string, finalURL string, err error) {
	if !protocolGuessed {
		body, headers, statusCode, title, links, err = fallbackFetch(ctx, targetURL)
		return body, headers, statusCode, title, links, targetURL, err
	}

	altURL := flipProtocol(targetURL)
	if altURL == targetURL {
		// 不是标准的 http(s):// URL（理论上不会发生，normalizeURL 保证了 scheme），
		// 双发无意义，退化成单发。
		body, headers, statusCode, title, links, err = fallbackFetch(ctx, targetURL)
		return body, headers, statusCode, title, links, targetURL, err
	}

	var wg sync.WaitGroup
	outcomes := make([]fetchOutcome, 2)
	urls := [2]string{targetURL, altURL}
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			b, h, sc, t, l, e := fallbackFetch(ctx, u)
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
// httpx 的做法一致，原样从 web_scanner.go 迁移，逻辑不变。
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

// fallbackFetch 当 Headless Browser 失败时，降级使用标准库 net/http 抓取首页
// 原始数据。原样从 web_scanner.go 迁移，逻辑不变。
func fallbackFetch(ctx context.Context, targetURL string) (body string, headers map[string]string, statusCode int, title string, links []string, err error) {
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

	title = extractTitle(bodyStr)

	// 复用本包已有的 ExtractLinksAndForms 提取首页种子链接，forms/params 这里
	// 不需要（首页的 Forms/Params 由调用方在拿到 HomePage.Body 后自行统一
	// 提取，见 web扫描模块重构实施文档.md 3.2 节）。
	links, _, _ = ExtractLinksAndForms(targetURL, bodyStr)

	logger.Infof("[crawler] Fallback fetch success for %s", targetURL)
	return bodyStr, headers, resp.StatusCode, title, links, nil
}
