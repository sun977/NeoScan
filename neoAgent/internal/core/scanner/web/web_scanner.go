package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/pkg/fingerprint"
	"neoagent/internal/pkg/fingerprint/engines/http"
	"neoagent/internal/pkg/logger"

	"github.com/go-rod/rod/lib/proto"
)

// WebScanner 实现 Web 指纹扫描与截图
type WebScanner struct {
	// 基础设施
	browserManager  *browser.BrowserManager
	browserLauncher *browser.BrowserLauncher

	// 指纹引擎 (复用 internal/pkg/fingerprint)
	fpEngine *http.HTTPEngine

	// 资源限制 (QoS)
	limiter *qos.AdaptiveLimiter

	mu sync.Mutex
}

// NewWebScanner 创建 Web 扫描器
func NewWebScanner() *WebScanner {
	bm := browser.NewBrowserManager()
	// 初始化空的指纹引擎
	// TODO: 从配置文件或 embedded FS 加载指纹规则
	fpEngine := http.NewHTTPEngine(nil)

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
func (s *WebScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	// 1. 获取 QoS 令牌
	if err := s.limiter.Acquire(ctx); err != nil {
		return nil, err
	}
	defer s.limiter.Release()

	startTime := time.Now()
	targetURL := normalizeURL(task.Target, task.PortRange)

	// 2. 启动浏览器 (Lazy Load)
	// 这里我们每次 Scan 都尝试 Launch，Launch 内部会复用已启动的 Browser
	// TODO: 支持从 Task 参数中读取 Proxy
	br, err := s.browserLauncher.Launch(ctx)
	if err != nil {
		s.limiter.OnFailure()
		return nil, fmt.Errorf("failed to launch browser: %w", err)
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
	stop := proto.NetworkResponseReceived{}.On(page, func(e *proto.NetworkResponseReceived) {
		// 我们主要关注 Document 类型的响应，且通常是最后一个（考虑重定向）
		// 或者匹配 targetURL 的那个
		// 简单起见，我们记录每一个 Document 的响应，最后留下的就是最终页面的
		if e.Type == proto.NetworkResourceTypeDocument {
			respMutex.Lock()
			defer respMutex.Unlock()

			statusCode = e.Response.Status
			remoteIP = e.Response.RemoteIPAddress
			remotePort = e.Response.RemotePort

			// Headers
			for k, v := range e.Response.Headers {
				var val string
				if err := json.Unmarshal(v, &val); err == nil {
					respHeaders[k] = val
				} else {
					respHeaders[k] = string(v)
				}
			}

			// Content-Length
			// 尝试从 Header 获取，或者从 EncodedDataLength
			if cl, ok := e.Response.Headers["Content-Length"]; ok {
				fmt.Sscanf(cl.String(), "%d", &contentLength)
			} else {
				// 如果 Header 里没有，尝试使用 EncodedDataLength
				// 注意: 这可能不准确，因为它是传输长度
				if e.Response.EncodedDataLength > 0 {
					contentLength = e.Response.EncodedDataLength
				}
			}
		}
	})
	defer stop()

	// 4. 导航到目标 URL
	if err := page.Navigate(targetURL); err != nil {
		s.limiter.OnFailure()
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// 5. 等待加载完成
	// 使用 MustWaitLoad 等待页面加载完成 (network idle)
	// 设置超时，防止挂死
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 注意: page.Timeout 会返回一个新的 page 对象，需要链式调用
	if err := page.Context(waitCtx).WaitLoad(); err != nil {
		logger.Warnf("[WebScanner] WaitLoad timeout for %s: %v", targetURL, err)
		// 超时也继续尝试提取，因为可能部分加载了
	}

	// 6. 提取 Rich Context (DOM/JS/Meta)
	richCtx, err := ExtractRichContext(page)
	if err != nil {
		logger.Warnf("[WebScanner] Failed to extract rich context: %v", err)
	}

	// 7. 构造 Input 并匹配指纹
	input := &fingerprint.Input{
		Target:      task.Target,
		RichContext: richCtx,
	}

	// 填充基础字段
	if body, ok := richCtx["body"].(string); ok {
		input.Body = body
	}
	// 将捕获到的 Headers 放入 Input (需要转换为 map[string][]string 或保持 map[string]string)
	// 目前 fingerprint.Input 主要看 Body 和 Headers
	// 这里我们需要适配一下类型，因为 rod 的 headers 是 map[string]json.RawMessage (v.String() 后是 string)
	// 而 http.Header 是 map[string][]string
	// 暂时只提取 Server, X-Powered-By 等关键头放入 RichContext 供 matcher 使用
	// 或者修改 Input 结构体?
	// 简单起见，我们将 respHeaders 放入 RichContext 的 "headers" 字段
	richCtx["headers"] = respHeaders

	// 调用指纹引擎匹配
	// TODO: 确保 s.fpEngine 已初始化
	var matches []fingerprint.Match
	if s.fpEngine != nil {
		matches, _ = s.fpEngine.Match(input)
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
	// 兜底 IP/Port
	if remoteIP == "" {
		// 尝试从 task 解析? 或者直接用 Target (如果是 IP)
		// 这里留空，让 Master 端去 resolve 或者后续处理
		// 为了满足契约，如果是 IP 形式的 Target，可以直接填
		if isIP(task.Target) {
			remoteIP = task.Target
		}
	}
	if remotePort == 0 {
		// 尝试从 task.PortRange 解析
		// ...
		if task.PortRange != "" {
			fmt.Sscanf(task.PortRange, "%d", &remotePort)
		}
	}

	result := &model.TaskResult{
		TaskID:      task.ID,
		Status:      model.TaskStatusSuccess,
		ExecutedAt:  startTime,
		CompletedAt: time.Now(),
		Result: &model.WebResult{
			URL:             targetURL,
			IP:              remoteIP,
			Port:            remotePort,
			Title:           extractTitleFromCtx(richCtx),
			StatusCode:      statusCode,
			ContentLength:   contentLength,
			ResponseHeaders: respHeaders,
			TechStack:       convertMatchesToTechStack(matches),
			Screenshot:      screenshotBase64,
			Favicon:         faviconBase64,
		},
	}

	s.limiter.OnSuccess()
	return []*model.TaskResult{result}, nil
}

// normalizeURL 简单的 URL 规范化
func normalizeURL(target string, port string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}
	// 默认 HTTP，如果端口是 443 则 HTTPS
	if port == "443" {
		return "https://" + target
	}
	return "http://" + target
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
