// Package api 提供 API 扫描能力：独立发起抓取（go-rod 渲染 + 默认开启的
// BFS 深度爬取），从 HTML/内联JS/外链JS 中静态提取接口调用地址。不依赖
// web 包或任何跨扫描器共享的爬虫模块，见 API扫描功能设计.md 第二节。
package api

import (
	"context"
	"fmt"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/pkg/logger"
)

// ApiScanner 是独立于 WebScanner 的原子扫描器，与 WebScanner 平级，各自
// 持有独立的 browser/limiter 实例，互不共享运行时状态。
type ApiScanner struct {
	browserLauncher *browser.BrowserLauncher
	limiter         *qos.AdaptiveLimiter
}

// NewApiScanner 创建一个 ApiScanner 实例，独立初始化自己的基础设施实例。
func NewApiScanner() *ApiScanner {
	bm := browser.NewBrowserManager()
	return &ApiScanner{
		browserLauncher: browser.NewLauncher(bm),
		limiter:         qos.NewAdaptiveLimiter(5, 1, 10),
	}
}

// Name 扫描器名称。
func (s *ApiScanner) Name() model.TaskType {
	return model.TaskTypeApiScan
}

// Run 执行 API 扫描任务：Target/Ports 换算起始 URL -> BFS 深度爬取
// （默认总是开启）-> 每页产出 model.ApiResult。
func (s *ApiScanner) Run(ctx context.Context, task *model.Task) (results []*model.TaskResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ApiScanner panic recovered: %v", r)
			logger.Errorf("[ApiScanner] panic recovered: %v", r)
		}
	}()

	startURL := normalizeTarget(task.Target, task.PortRange)

	crawlDepth := 2
	if v, ok := task.Params["crawl_depth"].(int); ok && v > 0 {
		crawlDepth = v
	}
	maxJSFiles := 20
	if v, ok := task.Params["max_js_files"].(int); ok && v > 0 {
		maxJSFiles = v
	}

	crawler := newAPICrawler(s.browserLauncher, s.limiter, crawlDepth, maxJSFiles)
	outcomes := crawler.crawl(ctx, startURL)

	startTime := time.Now()
	results = make([]*model.TaskResult, 0, len(outcomes))
	for _, oc := range outcomes {
		apis := make([]model.APIInfo, 0, len(oc.APIs))
		for _, c := range oc.APIs {
			apis = append(apis, model.APIInfo{
				URL:        c.URL,
				Method:     c.Method,
				Source:     c.Source,
				Confidence: c.Confidence,
			})
		}
		results = append(results, &model.TaskResult{
			TaskID: task.ID,
			Status: model.TaskStatusSuccess,
			Result: &model.ApiResult{
				URL:           oc.URL,
				Depth:         oc.Depth,
				APIs:          apis,
				APIsTruncated: oc.APIsTruncated,
			},
			ExecutedAt:  startTime,
			CompletedAt: time.Now(),
		})
	}
	return results, nil
}
