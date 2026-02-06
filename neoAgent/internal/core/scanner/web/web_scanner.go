package web

import (
	"context"
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
	return &WebScanner{
		browserManager:  bm,
		browserLauncher: browser.NewLauncher(bm),
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

	// 3. 打开页面
	page, err := s.browserLauncher.OpenPage(ctx, br, targetURL)
	if err != nil {
		s.limiter.OnFailure()
		return nil, fmt.Errorf("failed to open page: %w", err)
	}
	defer page.Close()

	// 4. 等待加载完成
	// 使用 MustWaitLoad 等待页面加载完成 (network idle)
	// 设置超时，防止挂死
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	// 注意: go-rod 的 API 设计大量使用 panic (MustXxx)，
	// 这里的 page 操作建议封装在 recover 中，或者使用非 Must 版本
	// 为了简单起见，我们假设 OpenPage 返回的 page 是可用的，
	// 并在后续操作中使用带有 Context 的方法
	
	// 简单等待加载
	// TODO: 更智能的等待策略 (WaitStable, WaitRequestIdle)
	if err := page.Timeout(15 * time.Second).WaitLoad(); err != nil {
		logger.Warnf("[WebScanner] WaitLoad timeout for %s: %v", targetURL, err)
		// 超时也继续尝试提取，因为可能部分加载了
	}

	// 5. 提取 Rich Context (DOM/JS/Meta)
	// 这是核心步骤，调用 context.go 中的逻辑
	richCtx, err := ExtractRichContext(page)
	if err != nil {
		logger.Warnf("[WebScanner] Failed to extract rich context: %v", err)
		// 即使提取失败，也尝试继续
	}

	// 6. 构造 Input 并匹配指纹
	// 这里需要将 Rich Context 转换为 fingerprint.Input
	// 注意: 目前 fingerprint.Input 主要设计给 HTTP 响应，
	// 我们需要将浏览器抓取的数据填充进去
	input := &fingerprint.Input{
		Target:      task.Target,
		RichContext: richCtx,
		// Header/Body 等基础字段也应该从 richCtx 中提取并填充到 Input 中
		// 以便 http_engine.go 中的 convertInputToMap 能正确处理
	}
	
	// 填充基础字段 (兼容旧逻辑)
	if body, ok := richCtx["body"].(string); ok {
		input.Body = body
	}
	// TODO: Headers 提取

	// 调用指纹引擎匹配
	// TODO: s.fpEngine 需要初始化 (加载规则)
	var matches []fingerprint.Match
	if s.fpEngine != nil {
		matches, _ = s.fpEngine.Match(input)
	}

	// 7. 截图 (如果启用)
	var screenshot []byte
	if capture, ok := task.Params["screenshot"].(bool); ok && capture {
		// screenshot, err = page.Screenshot(true, nil)
		// 暂时略过截图实现细节
	}

	// 8. 构造结果
	result := &model.TaskResult{
		TaskID:      task.ID,
		Status:      model.TaskStatusSuccess,
		ExecutedAt:  startTime,
		CompletedAt: time.Now(),
		Result: &model.WebResult{ // 假设 model 中有 WebResult
			URL:         targetURL,
			Fingerprints: convertMatches(matches),
			Screenshot:   screenshot,
			Title:        extractTitleFromCtx(richCtx),
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

// convertMatches 转换指纹格式
func convertMatches(matches []fingerprint.Match) []string {
	var res []string
	for _, m := range matches {
		res = append(res, fmt.Sprintf("%s (%s)", m.Product, m.Version))
	}
	return res
}

func extractTitleFromCtx(ctx map[string]interface{}) string {
	if t, ok := ctx["title"].(string); ok {
		return t
	}
	return ""
}
