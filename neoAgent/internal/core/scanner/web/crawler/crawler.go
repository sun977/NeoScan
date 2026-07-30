package crawler

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
)

// Options 爬虫行为控制参数
type Options struct {
	MaxDepth     int           // 默认 2，由调用方（WebScanner）决定，crawler 不设默认值兜底以外的隐藏逻辑
	MaxPages     int           // 硬上限，默认 200，防止爬虫失控
	Concurrency  int           // 默认 5
	Timeout      time.Duration // 单页超时，默认 10s
	SameHostOnly bool          // 默认 true，只在同一 Host 内爬
}

// Page 爬虫抓取到的单个页面的原始数据。
// 注意：不出现 TechStack/指纹相关字段，指纹识别是 WebScanner 的职责（架构方案 8.6.2 节）。
type Page struct {
	URL              string
	Depth            int
	StatusCode       int
	Title            string
	Body             string
	Headers          map[string]string
	Forms            []model.FormInfo
	Params           []string
	Leaks            []model.LeakInfo
	NeedsEscalation  bool   // Sprint 5 使用，Sprint 1 阶段固定为 false
	EscalationReason string // Sprint 5 使用，Sprint 1 阶段固定为空字符串
}

// item 是 BFS 队列内部元素，不对外暴露
type item struct {
	URL   string
	Depth int
}

// Crawler 单次爬取任务的执行器，一次 New 对应一次 Crawl 调用，不可跨任务复用状态
type Crawler struct {
	opts     Options
	limiter  *qos.AdaptiveLimiter // 复用调用方传入的实例，Crawler 自己不创建限流器
	client   *http.Client
	seedHost string

	mu      sync.Mutex
	visited map[string]struct{}

	pagesMu sync.Mutex
	pages   []*Page

	queue   chan *item
	pending int32 // 队列中未处理 + 处理中的任务数，归零时关闭 queue（见 3.3.3 节说明）
	closeOnce sync.Once
}

// New 创建一个 Crawler 实例。limiter 必须由调用方（WebScanner）传入已存在的实例，不允许传 nil。
func New(opts Options, limiter *qos.AdaptiveLimiter) *Crawler {
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 2
	}
	return &Crawler{
		opts:    opts,
		limiter: limiter,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				Proxy:           http.ProxyFromEnvironment,
			},
		},
		visited: make(map[string]struct{}),
	}
}

// Crawl 执行一次 BFS 爬取。
// seedURL: 首页 URL，仅用于确定 SameHostOnly 的判断基准（seedHost），不会被重复抓取。
// seedLinks: 首页阶段已经拿到的链接列表（go-rod 渲染后提取，或 fallback 阶段用 net/http 提取），
//
//	作为 BFS 第一层种子直接入队，crawler 内部不会再对 seedURL 本身发起一次 net/http 请求。
//
// 返回值：本次爬取到的全部 Page（不含首页，首页由 WebScanner 自己组装）。
func (c *Crawler) Crawl(ctx context.Context, seedURL string, seedLinks []string) []*Page {
	u, err := url.Parse(seedURL)
	if err != nil {
		return nil
	}
	c.seedHost = u.Host

	c.queue = make(chan *item, 256)

	// 种子链接入队（去重 + 深度 1，因为 depth 0 是首页，首页不由 crawler 处理）
	seeded := 0
	for _, link := range dedupeSeeds(seedLinks) {
		if c.enqueue(link, 1) {
			seeded++
		}
	}
	if seeded == 0 {
		return nil // 没有种子链接，直接返回空，不启动任何 worker（避免空转）
	}

	var wg sync.WaitGroup
	concurrency := c.opts.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go c.worker(ctx, &wg)
	}

	wg.Wait()

	c.pagesMu.Lock()
	defer c.pagesMu.Unlock()
	return c.pages
}

// EnqueueExtra 供 Sprint 5（按需升级机制）在 Crawl 已返回之后追加新发现的链接。
// Sprint 1 阶段只需要实现方法签名和基础入队逻辑，调用方是谁在 Sprint 1 阶段不需要关心。
func (c *Crawler) EnqueueExtra(links []string, atDepth int) {
	for _, link := range dedupeSeeds(links) {
		c.enqueue(link, atDepth)
	}
}

// worker 从队列中消费 item，抓取并展开子链接（对照架构方案 3 层缩进验证代码，原样落地，不允许增加缩进层级）
func (c *Crawler) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for it := range c.queue {
		c.process(ctx, it)
	}
}

// process 处理单个 item：抓取、记录页面、展开子链接、递减 pending 计数
func (c *Crawler) process(ctx context.Context, it *item) {
	defer c.taskDone()

	if !c.shouldVisit(it) {
		return
	}
	if err := c.limiter.Acquire(ctx); err != nil {
		return // context 取消，直接放弃剩余任务，不算失败
	}
	page, links, ok := c.fetchAndExtract(ctx, it)
	if ok {
		c.limiter.OnSuccess()
	} else {
		c.limiter.OnFailure()
	}
	c.limiter.Release()
	if !ok {
		return
	}
	c.pagesMu.Lock()
	c.pages = append(c.pages, page)
	c.pagesMu.Unlock()

	if it.Depth >= c.opts.MaxDepth {
		return // 达到深度上限，不再展开子链接，但当前页面结果已保留
	}
	for _, link := range links {
		c.enqueue(link, it.Depth+1)
	}
}

// enqueue 尝试将一个 URL 入队，成功返回 true。
// 内部完成：归一化、去重、Scope 判断、MaxPages 硬上限判断。
// 成功入队会同时将 pending 计数 +1，process 完成后统一在 taskDone 里 -1。
func (c *Crawler) enqueue(raw string, depth int) bool {
	key := normalizeKey(raw)

	c.mu.Lock()
	if _, exists := c.visited[key]; exists {
		c.mu.Unlock()
		return false
	}
	if len(c.visited) >= c.maxPages() {
		c.mu.Unlock()
		return false
	}
	if !c.inScope(key) {
		c.mu.Unlock()
		return false
	}
	c.visited[key] = struct{}{}
	c.mu.Unlock()

	atomic.AddInt32(&c.pending, 1)
	select {
	case c.queue <- &item{URL: key, Depth: depth}:
		return true
	default:
		// 队列满，丢弃（256 缓冲在真实场景下不会轻易打满，打满说明 MaxPages 该收紧了）
		c.taskDone()
		return false
	}
}

// taskDone 递减 pending 计数，归零时关闭队列，唤醒所有阻塞在 range queue 上的 worker 退出。
// closeOnce 保证并发场景下 close(queue) 只被执行一次，避免重复 close 导致 panic。
func (c *Crawler) taskDone() {
	if atomic.AddInt32(&c.pending, -1) == 0 {
		c.closeOnce.Do(func() {
			close(c.queue)
		})
	}
}

func (c *Crawler) maxPages() int {
	if c.opts.MaxPages <= 0 {
		return 200
	}
	return c.opts.MaxPages
}

// shouldVisit 目前只做深度判断（去重已经在 enqueue 阶段做过，不重复判断）
func (c *Crawler) shouldVisit(it *item) bool {
	return it.Depth <= c.opts.MaxDepth
}

// inScope 判断 URL 是否属于同源范围
func (c *Crawler) inScope(rawURL string) bool {
	if !c.opts.SameHostOnly {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Host == c.seedHost
}

// normalizeKey 见架构方案 8.3 节，原样实现，不做参数值归一化
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

func dedupeSeeds(links []string) []string {
	seen := make(map[string]struct{}, len(links))
	out := make([]string, 0, len(links))
	for _, l := range links {
		k := normalizeKey(l)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, l)
	}
	return out
}

// fetchAndExtract 在 Sprint 1 阶段只需要完成"发 HTTP 请求 + 读 body + 提取 Title"，
// 不需要调用 Sprint 2 才会写的 ExtractLinksAndForms——Sprint 1 先用一个占位的简单字符串查找提取
// <a href="..."> 即可让 BFS 跑起来，Sprint 2 落地后再替换成 goquery 版本。
func (c *Crawler) fetchAndExtract(ctx context.Context, it *item) (*Page, []string, bool) {
	timeout := c.opts.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, it.URL, nil)
	if err != nil {
		return nil, nil, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) NeoScan-Crawler/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, false
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 复用 fallbackScan 的 2MB 上限模式
	if err != nil {
		return nil, nil, false
	}
	body := string(bodyBytes)

	headers := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}

	page := &Page{
		URL:        it.URL,
		Depth:      it.Depth,
		StatusCode: resp.StatusCode,
		Title:      extractTitlePlaceholder(body),
		Body:       body,
		Headers:    headers,
	}
	// Sprint 1 占位：Sprint 2 会替换成 ExtractLinksAndForms 调用
	links := extractLinksPlaceholder(it.URL, body)
	return page, links, true
}

// extractTitlePlaceholder 是 Sprint 1 阶段的最小 Title 提取实现，Sprint 2/4 会有更完整的版本，
// 这里先保证 Page.Title 不是空字符串导致测试无法断言。
func extractTitlePlaceholder(body string) string {
	lower := strings.ToLower(body)
	start := strings.Index(lower, "<title>")
	if start == -1 {
		return ""
	}
	start += len("<title>")
	end := strings.Index(lower[start:], "</title>")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(body[start : start+end])
}

// extractLinksPlaceholder 是 Sprint 1 占位实现，仅用简单字符串查找提取 <a href="...">，
// Sprint 2 会用 ExtractLinksAndForms（基于 goquery）替换掉这个函数。
func extractLinksPlaceholder(baseURL string, body string) []string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}
	var links []string
	lower := strings.ToLower(body)
	idx := 0
	for {
		hrefPos := strings.Index(lower[idx:], "href=")
		if hrefPos == -1 {
			break
		}
		hrefPos += idx + len("href=")
		if hrefPos >= len(body) {
			break
		}
		quote := body[hrefPos]
		if quote != '"' && quote != '\'' {
			idx = hrefPos
			continue
		}
		end := strings.IndexByte(body[hrefPos+1:], quote)
		if end == -1 {
			break
		}
		href := body[hrefPos+1 : hrefPos+1+end]
		idx = hrefPos + 1 + end

		href = strings.TrimSpace(href)
		if href == "" || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") || strings.HasPrefix(href, "#") {
			continue
		}
		ref, err := url.Parse(href)
		if err != nil {
			continue
		}
		links = append(links, base.ResolveReference(ref).String())
	}
	return links
}
