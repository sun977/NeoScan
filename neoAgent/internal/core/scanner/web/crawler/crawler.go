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

// fetchAndExtract 是单个页面的完整处理流程："抓取原始数据" + "提取攻击面信息"，
// 这是 worker 对每个队列元素真正执行的核心动作，process 只是围绕它做限流和记账。
//
// 分两大步：
//  1. 用 net/http 发起真实的 GET 请求，拿到状态码、响应头、响应体（Sprint 1 就已完成，本次不变）。
//  2. 把响应体交给 extract.go 的 ExtractLinksAndForms 做 DOM 解析，一次性拿到
//     "继续爬取需要的链接" + "攻击面需要的表单/参数"，再交给 leak.go 的 DetectLeaks
//     做敏感信息扫描（leak.go 是 Sprint 3 的产出，Sprint 2 阶段这一步还不存在）。
//
// 返回值里的 bool 表示"这次抓取是否成功"：
//   - 网络层面的失败（连不上、超时、读 body 出错）返回 false，调用方 process 会据此
//     告知限流器 OnFailure()，让自适应限流收紧并发；
//   - 只要拿到了 HTTP 响应（哪怕是 404/500 这类业务错误状态码），也算"成功"返回 true，
//     因为这本身就是一次有效的网络探测结果，不是网络故障。
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

	// 限制最大读取 2MB，防止极端情况下服务端返回一个超大响应体把内存打爆
	// （比如误爬到一个视频文件的直链）。这个上限和 web_scanner.go 里 fallbackScan
	// 使用的上限保持一致，全项目统一这一个"安全边界"数值，不额外发明新的常量。
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, nil, false
	}
	body := string(bodyBytes)

	headers := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		// resp.Header 的值类型是 []string（同一个 Header 名可以出现多次，
		// 比如 Set-Cookie），这里用逗号拼接成单个字符串存进 map，
		// 和项目里其它地方处理 http.Header 的方式保持一致。
		headers[k] = strings.Join(v, ", ")
	}

	// 用 goquery 一次性提取链接、表单、URL 参数。links 是继续 BFS 需要的"下一层种子"，
	// forms/params 是攻击面信息，直接挂到 Page 上，最终会原样透传进 WebResult。
	links, forms, params := ExtractLinksAndForms(it.URL, body)

	page := &Page{
		URL:        it.URL,
		Depth:      it.Depth,
		StatusCode: resp.StatusCode,
		Title:      extractTitle(body),
		Body:       body,
		Headers:    headers,
		Forms:      forms,
		Params:     params,
	}
	return page, links, true
}

// extractTitle 从 HTML 文本里取出 <title> 标签内的文字，用作 Page.Title。
//
// 为什么不用 goquery（extract.go 已经引入了这个依赖，看起来顺手就能用 doc.Find("title")）？
//   两个原因：
//     1. 职责边界——extract.go 的注释里已经明确它只负责"攻击面提取"（链接/表单/参数），
//        Title 是页面展示信息，不是攻击面，混进 ExtractLinksAndForms 会让那个函数的
//        职责变得模糊（"提取攻击面"和"顺便查个标题"是两件不同的事）。
//     2. 性能——extractTitle 只需要找一个标签，字符串查找的开销远小于把整个 HTML
//        解析成一棵 DOM 树；而 ExtractLinksAndForms 已经要解析一次 DOM 树了，
//        如果 Title 也放进去，等于是"因为顺手"而把两个函数耦合在一起，之后任何一个
//        需求变化都可能牵连另一个，不划算。
//
// 实现上用 strings.ToLower 统一大小写后再查找 <title>，是因为 HTML 标签大小写
// 不敏感（<TITLE> 和 <title> 都合法），但真正截取内容时用的是原始 body 而不是
// 转小写后的字符串，这样才不会把标题文字本身的大小写也弄丢。
func extractTitle(body string) string {
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
