// Package api 的 bfs.go：独立的 BFS 队列实现，去重/深度控制/Scope 判断。
// 架构模式参照 web/crawler/crawler.go，代码物理独立，不 import，见
// API扫描功能设计.md 第四节。
package api

import (
	"context"
	"net/url"
	"sort"
	"sync"
	"sync/atomic"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/network/qos"
)

// crawlItem 是 BFS 队列内部元素，不对外暴露。
type crawlItem struct {
	URL   string
	Depth int
}

// pageOutcome 是 apiCrawler 对外产出的单页处理结果，编排层（api_scanner.go）
// 用它组装 model.ApiResult，不直接暴露 fetchedPage 这个抓取层内部类型。
type pageOutcome struct {
	URL           string
	Depth         int
	APIs          []candidate
	APIsTruncated bool
}

// apiCrawler 单次爬取任务的执行器，一次 newAPICrawler 对应一次 crawl 调用，
// 不可跨任务复用状态，与 web/crawler.Crawler 的生命周期约定一致。
type apiCrawler struct {
	launcher   *browser.BrowserLauncher
	limiter    *qos.AdaptiveLimiter
	maxDepth   int
	maxJSFiles int
	seedHost   string

	mu      sync.Mutex
	visited map[string]struct{}

	outcomesMu sync.Mutex
	outcomes   []pageOutcome

	queue     chan *crawlItem
	pending   int32
	closeOnce sync.Once
}

// newAPICrawler 创建一个 apiCrawler 实例。launcher/limiter 必须由调用方
// （ApiScanner，在 NewApiScanner 时已初始化）传入已存在的实例。单元测试
// 里只测 enqueue/normalizeKey 等纯逻辑、不调用 fetchPage，允许传 nil。
func newAPICrawler(launcher *browser.BrowserLauncher, limiter *qos.AdaptiveLimiter, maxDepth, maxJSFiles int) *apiCrawler {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	return &apiCrawler{
		launcher:   launcher,
		limiter:    limiter,
		maxDepth:   maxDepth,
		maxJSFiles: maxJSFiles,
		visited:    make(map[string]struct{}),
	}
}

// crawl 执行一次 BFS 爬取，起点是 seedURL 本身（与 web/crawler.Crawler 不同——
// web/crawler 的首页由 WebScanner 单独处理、crawler 只管子页面；ApiScanner
// 没有"首页单独处理"的必要，seedURL 直接作为 depth=0 的第一个 item 入队，
// 统一走同一套抓取+提取逻辑，不重复实现"首页专用"的一套代码）。
func (c *apiCrawler) crawl(ctx context.Context, seedURL string) []pageOutcome {
	u, err := url.Parse(seedURL)
	if err != nil {
		return nil
	}
	c.seedHost = u.Host

	c.queue = make(chan *crawlItem, 256)
	c.enqueue(seedURL, 0)

	var wg sync.WaitGroup
	const concurrency = 5
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go c.worker(ctx, &wg)
	}
	wg.Wait()

	c.outcomesMu.Lock()
	defer c.outcomesMu.Unlock()
	return c.outcomes
}

func (c *apiCrawler) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for it := range c.queue {
		c.process(ctx, it)
	}
}

func (c *apiCrawler) process(ctx context.Context, it *crawlItem) {
	defer c.taskDone()

	if err := c.limiter.Acquire(ctx); err != nil {
		return
	}
	page, err := fetchPage(ctx, c.launcher, it.URL, c.maxJSFiles)
	if err != nil {
		c.limiter.OnFailure()
		c.limiter.Release()
		return
	}
	c.limiter.OnSuccess()
	c.limiter.Release()

	pageHost := ""
	if u, err := url.Parse(it.URL); err == nil {
		pageHost = u.Host
	}

	var allCandidates []candidate
	for _, src := range page.Sources {
		found := extractAPICandidates(src.Text, pageHost)
		for i := range found {
			found[i].Source = src.From
		}
		allCandidates = append(allCandidates, found...)
	}
	filtered := filterAPICandidates(allCandidates)

	c.outcomesMu.Lock()
	c.outcomes = append(c.outcomes, pageOutcome{
		URL: it.URL, Depth: it.Depth,
		APIs: filtered, APIsTruncated: page.Truncated,
	})
	c.outcomesMu.Unlock()

	if it.Depth >= c.maxDepth {
		return
	}
	for _, link := range page.Links {
		c.enqueue(link, it.Depth+1)
	}
}

// enqueue 尝试将一个 URL 入队，内部完成去重、Scope 判断、MaxPages 硬上限判断。
// MaxPages 固定 200，与 web/crawler 的默认值保持一致（方案文档第十一节：
// 是否需要单独调优留到真实站点测试后再定，先复用已验证过的默认值）。
//
// 并发安全说明：
// visited 的写入发生在 channel send 成功之后，而非之前。
// 这样可以避免"URL 进了 visited 但未入队"的静默丢页 bug：
// 若 channel 满导致 default 分支触发，visited 不写入，
// 下次遇到同一 URL 仍可重试入队。
// 代价是在 channel 满载的极小窗口内，同一 URL 可能被两个 goroutine
// 同时通过预检并各自入队一次（重复处理一次），但这是幂等的（去重在
// filterAPICandidates 和最终 seen map 层面保证），远优于静默丢页。
func (c *apiCrawler) enqueue(raw string, depth int) {
	key := normalizeKey(raw)

	// 预检：去重 + Scope + MaxPages，不写 visited。
	c.mu.Lock()
	if _, exists := c.visited[key]; exists {
		c.mu.Unlock()
		return
	}
	if len(c.visited) >= 200 {
		c.mu.Unlock()
		return
	}
	if !c.inScope(key) {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	atomic.AddInt32(&c.pending, 1)
	select {
	case c.queue <- &crawlItem{URL: key, Depth: depth}:
		// 入队成功，现在才写 visited——确保"已占位"与"实际入队"同步。
		c.mu.Lock()
		c.visited[key] = struct{}{}
		c.mu.Unlock()
	default:
		// channel 满载，入队失败，visited 不写，pending 回退。
		// 下次遇到同一 URL 时预检仍可通过，有机会重试入队。
		c.taskDone()
	}
}

func (c *apiCrawler) taskDone() {
	if atomic.AddInt32(&c.pending, -1) == 0 {
		c.closeOnce.Do(func() { close(c.queue) })
	}
}

func (c *apiCrawler) inScope(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Host == c.seedHost
}

// normalizeKey 对齐 web/crawler.normalizeKey 的行为（去 Fragment、Query
// 参数排序），独立实现不 import。
func normalizeKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	q := u.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := url.Values{}
	for _, k := range keys {
		sorted[k] = q[k]
	}
	u.RawQuery = sorted.Encode()
	return u.String()
}
