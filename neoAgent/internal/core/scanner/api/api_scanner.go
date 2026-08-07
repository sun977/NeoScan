// Package api 提供 API 扫描能力（当前阶段：从抓取到的页面中提取 JS 接口
// 调用地址，未来可扩展外部输入源，见 web扫描模块重构文档.md 5.3 节）。
package api

import (
	"context"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/crawler"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
)

// ApiScanner 是独立于 WebScanner 的原子扫描器，与 WebScanner 平级，
// 共享同一个 crawler.FetchAndCrawl 抓取入口，自己只关心"拿到页面之后
// 做 API 相关的分析"，不重复实现任何抓取逻辑（web扫描模块重构文档.md 第一节）。
type ApiScanner struct {
	// browserLauncher 是 FetchAndCrawl 内部按需使用 go-rod 渲染时需要的实例。
	// ApiScanner 自己不关心浏览器细节（不像 WebScanner 需要截图/favicon），
	// 但 FetchAndCrawl 的签名要求调用方显式传入 launcher（方案文档 3.4 节：
	// launcher 无状态，各 Scanner 各自持有一份成本很低，crawler 包不维护
	// 包级单例）。
	browserLauncher *browser.BrowserLauncher

	// limiter 是 ApiScanner 独立持有的限流器实例，不与 WebScanner 共享
	// （方案文档 7.2 节：两者资源消耗特征相近，但流量不应互相挤占对方的
	// 并发配额）。
	limiter *qos.AdaptiveLimiter
}

// NewApiScanner 创建一个 ApiScanner 实例。
func NewApiScanner() *ApiScanner {
	bm := browser.NewBrowserManager()
	return &ApiScanner{
		browserLauncher: browser.NewLauncher(bm),
		limiter:         qos.NewAdaptiveLimiter(5, 1, 10), // 与 WebScanner 相同的默认参数，具体数值后续可按需独立调整
	}
}

// Name 扫描器名称。
func (s *ApiScanner) Name() model.TaskType {
	return model.TaskTypeApiScan
}

// Run 执行 API 扫描任务：调用 crawler.FetchAndCrawl 拿首页 +（可选）深度
// 爬取子页面，对每个页面组装一条 model.ApiResult。
//
// 当前阶段（本文件对应的实施文档只搭骨架）：APIs 字段固定为空，真正的
// JS 提取逻辑由 Web-JS接口提取实施文档.md 补充实现，届时本函数体会被
// 修改（追加对 crawler.ExtractPageAPIs 的调用），函数签名保持不变。
func (s *ApiScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	crawlDepth, port, protocolHint := parseApiScanParams(task)

	home, subPages, err := crawler.FetchAndCrawl(ctx, task.Target, port, protocolHint,
		s.limiter, s.browserLauncher, crawler.FetchOptions{
			CrawlDepth:  crawlDepth,
			OnPageReady: nil, // ApiScanner 不需要截图/favicon，回调传 nil（重构文档 3.2 节约束）
		})
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	var results []*model.TaskResult
	results = append(results, s.buildApiResult(task, startTime, home.URL, 0))
	for _, p := range subPages {
		results = append(results, s.buildApiResult(task, startTime, p.URL, p.Depth))
	}
	return results, nil
}

// buildApiResult 组装单个页面的空壳结果。首页（depth=0）和每一个深度
// 爬取子页面都调用这一个函数，不存在"首页走一条路径、子页面走另一条
// 路径"的分叉——这是为后续 JS 提取逻辑接入预留的统一收口点。
func (s *ApiScanner) buildApiResult(task *model.Task, startTime time.Time, pageURL string, depth int) *model.TaskResult {
	return &model.TaskResult{
		TaskID:      task.ID,
		Status:      model.TaskStatusSuccess,
		ExecutedAt:  startTime,
		CompletedAt: time.Now(),
		Result: &model.ApiResult{
			URL:   pageURL,
			Depth: depth,
		},
	}
}

// parseApiScanParams 从 task.Params/task.PortRange 解析 ApiScanner 需要的
// 最小参数集：crawlDepth/port/protocolHint。crawl/crawl_depth 的三态语义
// 直接复用与 WebScanner.resolveCrawlDepth 相同的判断规则外壳（显式开启/
// 显式关闭/未指定），但骨架阶段的"未指定"档不做 WebScanner 那种依赖首页
// 响应结果的自动判断（ApiScanner 骨架阶段没有首页数据可用，这个判断本身
// 也留给后续实施文档按 ApiScanner 的真实需求决定），保守地默认不爬。
func parseApiScanParams(task *model.Task) (crawlDepth int, port string, protocolHint string) {
	if p, ok := task.Params["protocol"].(string); ok {
		protocolHint = p
	}
	port = task.PortRange

	enableCrawl, explicit := task.Params["crawl"].(bool)
	switch {
	case explicit && !enableCrawl:
		return 0, port, protocolHint
	case explicit && enableCrawl:
		depth := 2
		if d, ok := task.Params["crawl_depth"].(int); ok && d > 0 {
			depth = d
		}
		return depth, port, protocolHint
	default:
		return 0, port, protocolHint // 骨架阶段默认不自动开启深度爬取，比 WebScanner 更保守（方案文档 3.4 节）
	}
}
