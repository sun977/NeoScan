# NeoScan Web 爬虫与被动分析器实施文档 v1.1

> 本文档是 `Web爬虫与被动分析器架构方案-sonnet5-v3.0.md`（以下简称"架构方案"）的**唯一施工图纸**。架构方案负责回答"为什么这么设计"，本文档只负责回答"具体改哪个文件、哪一行、写成什么样、怎么验收"。
>
> **约定**：本文档中所有"现状代码"均已于撰写时逐行核实（文件路径、行号、函数签名），不是从架构方案的伪代码直接照抄。如果实施过程中发现现状与本文档描述不一致（比如行号因为其他改动漂移），以现状代码为准，但改动的**结构和顺序**必须遵守本文档，不允许自由发挥架构。
>
> **文档定位**：Sprint 0-5 共 6 个阶段，每个阶段给出「改动文件清单 → 新增/修改的函数签名 → 详细步骤 → 验收标准（含具体命令）→ 本阶段结束时代码必须能编译通过」。**开发严禁跳跃阶段**——Sprint N 的代码必须先跑通验收标准，才能开始 Sprint N+1，因为后面每个阶段都依赖前面阶段产出的真实类型（不是接口占位）。
>
> **v1.1 变更说明（2026-07-31）**：Sprint 0-5（Phase 5.1）已全部验收完毕并投入真实站点测试。测试过程中发现两个 `Run()` 主流程的原生缺陷（不是新需求，是现有代码一直存在但没暴露的 bug），追加 **Sprint 6：多端口探测与协议自适应** 修复，详见第七节之后新增的章节。

---

## 〇、实施前必读：3 个容易踩坑的现状事实

这三点是审查现有代码时发现的、**架构方案里没有强调或者可能引起误解**的事实，写在最前面，避免实施中途才发现返工。

### 0.1 现有单元测试会因为 Sprint 4 的改动直接编译失败，必须同步修

`internal/core/scanner/web/web_scanner_test.go` 现状：

```39:44:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/web_scanner_test.go
	ctx := context.Background()
	results, err := scanner.fallbackScan(ctx, task, ts.URL, time.Now())

	if err != nil {
		t.Fatalf("fallbackScan failed: %v", err)
	}
```

它直接调用私有方法 `fallbackScan(ctx, task, targetURL, startTime) ([]*model.TaskResult, error)`，断言返回值是 `[]*model.TaskResult`。Sprint 4 要把这个方法改造成 `fallbackFetch(ctx, targetURL) (body string, headers map[string]string, statusCode int, title string, links []string, err error)`，**签名完全变了，这个测试文件不修改整个包会编译不过**。Sprint 4 的任务清单里必须包含"同步重写这个测试用例"，不能只改生产代码。

### 0.2 `crawl` 参数目前有两条独立的下发路径，缺一不可

现状是**两条完全独立、互不调用的路径**都能触发 `WebScanner.Run()`：

| 路径 | 文件 | 现状 |
|---|---|---|
| CLI 直接扫描 | `cmd/agent/scan/web.go` + `internal/core/options/scan_web.go` | `WebScanOptions.ToTask()` 里根本没有写 `task.Params["crawl"]`，也没有 `--crawl` flag |
| `scan run` 全流程编排 | `internal/core/pipeline/dispatcher.go` 第 274-286 行 `runWebScan` | 目前只写了 `task.Params["protocol"]` 和 `task.Params["screenshot"]`，同样没有 `crawl` |
| Master 下发（集群模式）| `internal/service/adapter/task_to_core.go` 第 86-89 行 | **唯一目前会写 `crawl` 的地方**：`coreTask.Params["crawl"] = true`，硬编码为 true，这是 8.9 节要清理的对象 |

三条路径都要在 Sprint 5 里逐一确认接入，漏掉任何一条，那条路径就永远走不到自动判断或用户开关，**这在架构方案第九节里只写了 `scan_web.go` 和 `dispatcher.go` 两处，遗漏了 `task_to_core.go` 这条 Master 下发路径的显式修改步骤**（虽然 10.94 行提了一句"硬编码已删除"，但没有单列接入步骤），本文档 Sprint 5 会把它列为独立子项。

### 0.3 `WebScanOptions.ToTask()` 目前不接收 Output/其他扫描共享的 `--crawl`，需要新增字段

`internal/core/options/scan_web.go` 现状字段只有 `Target/Ports/Path/Method/Output`，新增 `Crawl string`（三态字符串）和 `CrawlDepth int` 字段是 Sprint 5 的一部分，不是"顺便加一下"，要按 9.2 节的三态语义实现，写进本文档 Sprint 5.3。

---

## 一、总体依赖顺序（不可颠倒）

```mermaid
flowchart LR
    S0["Sprint 0<br/>依赖与目录骨架"] --> S1["Sprint 1<br/>crawler 核心 BFS"]
    S1 --> S2["Sprint 2<br/>攻击面提取 extract.go"]
    S2 --> S3["Sprint 3<br/>被动泄露检测 leak.go"]
    S1 --> S3
    S3 --> S4["Sprint 4<br/>web_scanner.go 收口重构<br/>(fallbackFetch/buildWebResult)"]
    S4 --> S5["Sprint 5<br/>三处接入点 + 自动决策 + 按需升级联调"]
    S5 --> S6["Sprint 6<br/>多端口探测 + 协议自适应<br/>(真实站点测试后追加)"]
```

- Sprint 2、3 都依赖 Sprint 1 产出的 `crawler.Page`/`crawler.Crawler` 类型，但 2、3 之间互相独立，可以并行开发（都只是往 `Page` 上追加字段的纯函数）。
- Sprint 4 **不依赖** Sprint 1-3 的任何代码，只重构 `web_scanner.go` 内部现有逻辑，理论上可以和 Sprint 1-3 并行，但由于 Sprint 5 要把 Sprint 1-4 的产出粘合在一起，建议按顺序做，避免合并冲突。
- Sprint 5 是唯一的集成点，前面所有 Sprint 都不改 `web_scanner.go` 的对外行为（Sprint 4 只做内部收口，不改 `Run()` 的输入输出契约）。
- Sprint 6 依赖 Sprint 5 已经稳定的 `Run()` 主干（首页收口 → 决策 → BFS → 按需升级），是在这条主干**外层**包一层"多端口循环"、在 `fallbackFetch` 失败路径上补一次"协议翻转重试"，不改动 Sprint 1-5 内部任何一行已验收代码，属于纯增量。

---

## 二、Sprint 0：依赖引入与目录骨架（0.5 天）

### 2.1 改动文件清单

| 文件 | 操作 |
|---|---|
| `neoAgent/go.mod` / `go.sum` | 新增依赖 |
| `internal/core/scanner/web/crawler/` | 新建目录，放 3 个空文件占位 |
| `internal/core/model/result_types.go` | 新增 `FormInfo`/`LeakInfo` 类型 + `WebResult` 新增字段 |

### 2.2 详细步骤

**Step 1**：引入 `goquery`。

```powershell
cd c:\mytools\code\go\NeoScan\neoAgent
go get github.com/PuerkitoBio/goquery@latest
go mod tidy
```

**Step 2**：创建骨架文件（内容先留空/占位，Sprint 1-3 会填充）：

```
internal/core/scanner/web/crawler/crawler.go
internal/core/scanner/web/crawler/extract.go
internal/core/scanner/web/crawler/leak.go
```

**Step 3**：修改 `internal/core/model/result_types.go`，在现有 `WebResult` 定义（第 100-112 行）基础上**只做字段追加**，不删不改任何现有字段：

```go
// WebResult Web扫描结果
type WebResult struct {
	URL             string            `json:"url"`
	IP              string            `json:"ip"`
	Port            int               `json:"port"`
	Title           string            `json:"title"`
	StatusCode      int               `json:"status_code"`
	ContentLength   int64             `json:"content_length"`
	ResponseHeaders map[string]string `json:"headers,omitempty"`
	TechStack       []string          `json:"tech_stack,omitempty"`
	Screenshot      string            `json:"screenshot,omitempty"`
	Favicon         string            `json:"favicon,omitempty"`

	// --- 以下为爬虫功能新增字段，均为 omitempty，不影响现有序列化 ---
	Depth  int        `json:"depth,omitempty"`  // 爬取深度，0 = 首页
	Forms  []FormInfo `json:"forms,omitempty"`  // 表单/输入点
	Params []string   `json:"params,omitempty"` // URL Query 参数名集合
	Leaks  []LeakInfo `json:"leaks,omitempty"`  // 被动泄露检测结果
}

// FormInfo 表单信息（攻击面输入点）
type FormInfo struct {
	Action string   `json:"action"`
	Method string   `json:"method"`
	Fields []string `json:"fields"`
}

// LeakInfo 被动泄露检测命中信息
type LeakInfo struct {
	Type    string `json:"type"`              // aws_ak / aliyun_ak / jwt / internal_ip / ...
	Match   string `json:"match"`              // 脱敏后的命中内容，禁止存储明文密钥
	Context string `json:"context,omitempty"` // 命中上下文片段（可选）
}
```

**不要改** `Headers()`/`Rows()` 两个 `TabularData` 接口方法（第 115-131 行）——CLI 表格展示范围本次不扩展，这是刻意的最小改动。

### 2.3 验收标准

```powershell
cd c:\mytools\code\go\NeoScan\neoAgent
go build ./...
go vet ./...
```

两条命令都必须 0 报错。此时 `crawler` 包是空的，只要 3 个文件里各写一行 `package crawler` 即可通过编译。

---

## 三、Sprint 1：`crawler` 核心 BFS 骨架（1-2 天）

### 3.1 改动文件

`internal/core/scanner/web/crawler/crawler.go`（唯一改动文件，Sprint 0 建的空文件在此填充）

### 3.2 类型与函数签名（照此实现，不要自创字段名）

> 下面这个 import 块只列出 3.2 节类型定义直接用到的包；3.3 节 `fetchAndExtract`/`New` 的完整实现还会用到 `crypto/tls`、`io`、`net/http`、`strings`，实际编写 `crawler.go` 时导入的是这个完整列表：`context`、`crypto/tls`、`io`、`net/http`、`net/url`、`sort`、`strings`、`sync`、`time`、`neoagent/internal/core/lib/network/qos`、`neoagent/internal/core/model`。不要照抄下面代码块的 import 就去编译，会缺包报错。

```go
package crawler

import (
	"context"
	"net/url"
	"sort"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
)

// Options 爬虫行为控制参数
type Options struct {
	MaxDepth            int           // 默认 2，由调用方（WebScanner）决定，crawler 不设默认值兜底以外的隐藏逻辑
	MaxPages            int           // 硬上限，默认 200，防止爬虫失控
	Concurrency         int           // 默认 5
	Timeout             time.Duration // 单页超时，默认 10s
	SameHostOnly        bool          // 默认 true，只在同一 Host 内爬
}

// Page 爬虫抓取到的单个页面的原始数据。
// 注意：不出现 TechStack/指纹相关字段，指纹识别是 WebScanner 的职责（架构方案 8.6.2 节）。
type Page struct {
	URL               string
	Depth             int
	StatusCode        int
	Title             string
	Body              string
	Headers           map[string]string
	Forms             []model.FormInfo
	Params            []string
	Leaks             []model.LeakInfo
	NeedsEscalation   bool   // Sprint 5 使用，Sprint 1 阶段固定为 false
	EscalationReason  string // Sprint 5 使用，Sprint 1 阶段固定为空字符串
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
}

// New 创建一个 Crawler 实例。limiter 必须由调用方（WebScanner）传入已存在的实例，不允许传 nil。
func New(opts Options, limiter *qos.AdaptiveLimiter) *Crawler

// Crawl 执行一次 BFS 爬取。
// seedURL: 首页 URL，仅用于确定 SameHostOnly 的判断基准（seedHost），不会被重复抓取。
// seedLinks: 首页阶段已经拿到的链接列表（go-rod 渲染后提取，或 fallback 阶段用 net/http 提取），
//            作为 BFS 第一层种子直接入队，crawler 内部不会再对 seedURL 本身发起一次 net/http 请求。
// 返回值：本次爬取到的全部 Page（不含首页，首页由 WebScanner 自己组装）。
func (c *Crawler) Crawl(ctx context.Context, seedURL string, seedLinks []string) []*Page

// EnqueueExtra 供 Sprint 5（按需升级机制）在 Crawl 已返回之后追加新发现的链接。
// Sprint 1 阶段只需要实现方法签名和基础入队逻辑，调用方是谁在 Sprint 1 阶段不需要关心。
func (c *Crawler) EnqueueExtra(links []string, atDepth int)
```

> `EscalationReason`、`NeedsEscalation`、`EnqueueExtra` 三个是 Sprint 5 才会真正用上的字段/方法，但类型定义必须在 Sprint 1 一次到位——不要留到 Sprint 5 再给 `Page` 加字段，否则 Sprint 2/3 写好的 `extract.go`/`leak.go` 返回值组装代码到时候要跟着改一遍构造函数调用。**一次把结构定稳，后面 Sprint 只填内容不改骨架**，这是本文档拆分 Sprint 的核心原则。

### 3.3 核心实现要点（对照架构方案 4.1 节的伪代码，逐条转换成真实实现）

**3.3.1 BFS 主循环**（架构方案已给出伪代码框架，第 158-182 行，本文档补全遗漏的细节）：

```go
func (c *Crawler) Crawl(ctx context.Context, seedURL string, seedLinks []string) []*Page {
	u, err := url.Parse(seedURL)
	if err != nil {
		return nil
	}
	c.seedHost = u.Host

	queue := make(chan *item, 256)
	var inFlight sync.WaitGroup

	// 种子链接入队（去重 + 深度 1，因为 depth 0 是首页，首页不由 crawler 处理）
	seeded := 0
	for _, link := range dedupeSeeds(seedLinks) {
		if c.enqueue(queue, link, 1) {
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
		go c.worker(ctx, queue, &wg)
	}

	// 队列关闭时机：所有已入队的任务都被消费完，且没有新任务入队
	// 用 sync.WaitGroup 追踪"未完成的 item 数量"而不是猜测 close 时机
	go func() {
		inFlight.Wait()
		close(queue)
	}()
	_ = inFlight // 具体 inFlight 计数逻辑在 enqueue/worker 内部配合实现，见 3.3.3

	wg.Wait()

	c.pagesMu.Lock()
	defer c.pagesMu.Unlock()
	return c.pages
}
```

**3.3.2 worker（对照架构方案 3 层缩进验证代码，原样落地，不允许增加缩进层级）**：

```go
func (c *Crawler) worker(ctx context.Context, queue chan *item, wg *sync.WaitGroup) {
	defer wg.Done()
	for it := range queue {
		if !c.shouldVisit(it) {
			continue
		}
		if err := c.limiter.Acquire(ctx); err != nil {
			continue // context 取消，直接放弃剩余任务，不算失败
		}
		page, links, ok := c.fetchAndExtract(ctx, it)
		if ok {
			c.limiter.OnSuccess()
		} else {
			c.limiter.OnFailure()
		}
		c.limiter.Release()
		if !ok {
			continue
		}
		c.pagesMu.Lock()
		c.pages = append(c.pages, page)
		c.pagesMu.Unlock()

		if it.Depth >= c.opts.MaxDepth {
			continue // 达到深度上限，不再展开子链接，但当前页面结果已保留
		}
		for _, link := range links {
			c.enqueue(queue, link, it.Depth+1)
		}
	}
}
```

**3.3.3 关于"队列何时关闭"的实现说明**（这是架构方案伪代码里被简化掉、必须在实施时补全的部分）：

架构方案的伪代码用 `wg.Wait()` 等待所有 worker 退出，但没有说清楚 `close(queue)` 的时机——如果队列一直不关闭，`for it := range queue` 永远不会退出，`wg.Wait()` 会死锁。正确做法：**用一个原子计数器追踪"队列中未处理 + 处理中"的任务数，归零时关闭队列**，具体用 `sync.WaitGroup` 实现：

```go
// enqueue 尝试将一个 URL 入队，成功返回 true。
// 内部完成：归一化、去重、Scope 判断、MaxPages 硬上限判断。
// 调用 enqueue 前必须 Add(1)，enqueue 失败（未真正入队）时必须自己 Done() 抵消。
func (c *Crawler) enqueue(queue chan *item, raw string, depth int) bool {
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

	select {
	case queue <- &item{URL: key, Depth: depth}:
		return true
	default:
		// 队列满，丢弃（256 缓冲在真实场景下不会轻易打满，打满说明 MaxPages 该收紧了）
		return false
	}
}

func (c *Crawler) maxPages() int {
	if c.opts.MaxPages <= 0 {
		return 200
	}
	return c.opts.MaxPages
}
```

> **实现者注意**：3.3.1 里 `inFlight.Wait()` 的写法在真实实现时需要替换为更严谨的"未完成任务计数"模型，推荐做法：`queue` 改造为搭配一个 `pending int32`（`atomic` 操作），每次 `enqueue` 成功 `atomic.AddInt32(&pending, 1)`，每次 worker 处理完一个 item（不管是否展开出新链接）执行 `atomic.AddInt32(&pending, -1)`，用一个独立的监控 goroutine 轮询 `pending == 0` 时 `close(queue)`。这是本 Sprint 验收时**必须跑通的并发正确性细节**，不是可以偷懒跳过的部分——BFS 遍历型爬虫最容易出 bug 的地方就是"队列关闭时机"，单元测试里必须包含"大于 concurrency 数量的页面"的用例来暴露这类问题（见 3.5 节）。

**3.3.4 `shouldVisit`/`fetchAndExtract`/`inScope`/`normalizeKey`**：

```go
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
```

`fetchAndExtract` 在 Sprint 1 阶段只需要完成"发 HTTP 请求 + 读 body + 提取 Title"，**不需要**调用 Sprint 2 才会写的 `ExtractLinksAndForms`——Sprint 1 先用一个占位的正则或者简单字符串查找提取 `<a href="...">` 即可让 BFS 跑起来，Sprint 2 落地后再替换成 `goquery` 版本。这样 Sprint 1 的验收不依赖 Sprint 2：

```go
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
		Body:       body,
		Headers:    headers,
	}
	// Sprint 1 占位：Sprint 2 会替换成 ExtractLinksAndForms 调用
	links := extractLinksPlaceholder(it.URL, body)
	return page, links, true
}
```

### 3.4 `New` 函数实现

```go
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
```

### 3.5 单元测试要求（`internal/core/scanner/web/crawler/crawler_test.go`，新建）

必须覆盖以下用例，缺一不可：

1. **`TestCrawl_BasicBFS`**：起一个 `httptest.Server`，构造 3 层链接站点（首页种子 → 2 个二级页 → 每个二级页各 2 个三级页），`MaxDepth=2`，断言最终 `len(pages)` 精确等于二级+三级页面总数，且没有重复。
2. **`TestCrawl_MaxDepthRespected`**：同一个 3 层站点，`MaxDepth=1`，断言只抓到二级页面，三级页面不出现在结果里。
3. **`TestCrawl_Deduplication`**：构造两个互相链接指向同一个 URL（不同 `?a=1&b=2` 和 `?b=2&a=1` 顺序）的页面，断言只被访问一次。
4. **`TestCrawl_SameHostOnly`**：种子链接里混入一个外部域名的链接，断言外部链接不出现在结果里。
5. **`TestCrawl_MaxPagesLimit`**：构造一个链接数超过 `MaxPages`（测试里设小一点，比如 5）的站点，断言最终页面数不超过 `MaxPages`。
6. **`TestCrawl_ConcurrencyNoDeadlock`**：种子链接数量大于 `Concurrency`（比如种子 20 个、并发 5），跑 3 次，用 `go test -race` 断言不死锁、不 data race、每次运行结果数量一致。**这一条对应 3.3.3 节强调的并发正确性，必须跑 `-race` 通过**。

### 3.6 验收标准

```powershell
cd c:\mytools\code\go\NeoScan\neoAgent
go build ./...
go test ./internal/core/scanner/web/crawler/... -v -race
```

全部用例通过，`-race` 无告警。此时 `crawler` 包可以独立工作，但还没有接入 `WebScanner`。

---

## 四、Sprint 2：攻击面提取 `extract.go`（1 天）

### 4.1 改动文件

- `internal/core/scanner/web/crawler/extract.go`（新增实现）
- `internal/core/scanner/web/crawler/crawler.go`（把 Sprint 1 的 `extractLinksPlaceholder` 替换为真实调用）
- `internal/core/scanner/web/context.go`（新增 `ExtractLinks` 函数）

### 4.2 函数签名

```go
// extract.go
package crawler

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"neoagent/internal/core/model"
)

// ExtractLinksAndForms 从 HTML body 中提取链接、表单、URL 参数
func ExtractLinksAndForms(baseURL string, body string) (links []string, forms []model.FormInfo, params []string)

// resolve 把相对链接解析为绝对 URL，解析失败返回空字符串
func resolve(base *url.URL, href string) string
```

实现按架构方案 8.4 节给出的代码原样落地（已在架构方案里给出完整实现，不再重复抄写，实现者直接照抄架构方案 8.4 节代码块即可）。

**必须补充架构方案没写的部分**——`resolve` 函数的完整实现（架构方案只提了函数名，没给实现体）：

```go
func resolve(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") || strings.HasPrefix(href, "#") {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}
```

`javascript:`/`mailto:`/`tel:`/纯 Fragment 链接必须过滤掉，否则会污染 BFS 队列（这些不是可爬取的页面 URL），这是架构方案 8.4 节遗漏的边界处理，实施时必须补上。

### 4.3 `context.go` 新增 `ExtractLinks`

在现有 `ExtractRichContext` 函数（第 11-178 行）**之后**追加一个独立函数，不修改 `ExtractRichContext` 本体：

```go
// ExtractLinks 从 go-rod 页面提取 <a href> 链接列表，供爬虫 BFS 使用第一层种子
func ExtractLinks(page *rod.Page) []string {
	res, err := page.Eval(`(() => {
		const anchors = document.getElementsByTagName('a');
		const result = [];
		for (let i = 0; i < anchors.length; i++) {
			if (anchors[i].href) {
				result.push(anchors[i].href);
			}
		}
		return result;
	})()`)
	if err != nil {
		return nil
	}
	var links []string
	if valBytes, e := json.Marshal(res.Value); e == nil {
		_ = json.Unmarshal(valBytes, &links)
	}
	return links
}
```

用 `anchors[i].href`（而不是 `getAttribute('href')`）是刻意的——`.href` 属性会被浏览器自动解析成绝对 URL，不需要在 Go 侧再做一次 `resolve`，这与 `ExtractLinksAndForms`（处理的是 `net/http` 拿到的原始 HTML 字符串，必须手动 `resolve`）职责不同但结论一致：**最终吐出去的链接都是绝对 URL**。

### 4.4 替换 Sprint 1 占位函数

`crawler.go` 里的 `fetchAndExtract` 方法，把：

```go
links := extractLinksPlaceholder(it.URL, body)
```

替换为：

```go
links, forms, params := ExtractLinksAndForms(it.URL, body)
page.Forms = forms
page.Params = params
```

并删除 `extractLinksPlaceholder` 函数。

### 4.5 单元测试要求（`extract_test.go`，新建）

1. **`TestExtractLinksAndForms_BasicLinks`**：HTML 含 5 个 `<a href>`（含 1 个相对路径、1 个绝对路径、1 个 `#fragment`、1 个 `javascript:void(0)`、1 个 `mailto:`），断言只提取出 2 个有效链接（相对+绝对），且相对路径被正确 resolve。
2. **`TestExtractLinksAndForms_Forms`**：HTML 含 1 个 GET 表单和 1 个无 `method` 属性的表单，断言后者默认识别为 `GET`，字段列表正确提取 `input/select/textarea` 的 `name`。
3. **`TestExtractLinksAndForms_Params`**：`baseURL` 带 `?id=1&type=admin`，断言 `params` 返回 `["id", "type"]`（顺序不做强制断言，用 `sort` 后比较）。

### 4.6 验收标准

```powershell
go build ./...
go test ./internal/core/scanner/web/... -v -race
```

此时用真实 `httptest.Server` 跑 Sprint 1 的全部用例依然要通过（回归验证），额外新增的 `extract_test.go` 用例全部通过。

---

## 五、Sprint 3：被动泄露检测 `leak.go`（0.5 天）

### 5.1 改动文件

- `internal/core/scanner/web/crawler/leak.go`（新增）
- `internal/core/scanner/web/crawler/crawler.go`（`fetchAndExtract` 里追加一次 `DetectLeaks` 调用）

### 5.2 函数签名

```go
package crawler

import (
	"regexp"

	"neoagent/internal/core/model"
)

type leakRule struct {
	Name    string
	Pattern *regexp.Regexp
}

var defaultLeakRules = []leakRule{
	{"aws_ak", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"aliyun_ak", regexp.MustCompile(`LTAI[0-9A-Za-z]{12,20}`)},
	{"jwt", regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)},
	{"internal_ip", regexp.MustCompile(`\b(?:10|172\.(?:1[6-9]|2\d|3[0-1])|192\.168)\.\d{1,3}\.\d{1,3}\b`)},
}

// DetectLeaks 对页面 body 做正则扫描，命中项自动脱敏后返回
func DetectLeaks(body string) []model.LeakInfo

// mask 脱敏函数：保留前 4 位和后 4 位，中间用 **** 替代；短于 8 位的直接整体替换为 ****
func mask(s string) string
```

架构方案 8.5 节给出了 `DetectLeaks` 主体实现，**但没有给出 `mask` 函数的实现**，这是被动泄露检测里唯一有"安全红线"的部分（脱敏不到位等于白检测），本文档给出明确实现：

```go
func mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

func DetectLeaks(body string) []model.LeakInfo {
	var out []model.LeakInfo
	for _, r := range defaultLeakRules {
		for _, m := range r.Pattern.FindAllString(body, -1) {
			out = append(out, model.LeakInfo{Type: r.Name, Match: mask(m)})
		}
	}
	return out
}
```

**安全要求（不可省略）**：`out` 中的 `Match` 字段**任何时候都不允许存储正则命中的原始明文**，`mask` 函数是强制调用点，不允许在 `DetectLeaks` 之外的任何地方（比如日志打印）直接输出 `FindAllString` 的原始结果。Code Review 时这是一票否决项。

### 5.3 接入 `crawler.go`

`fetchAndExtract` 方法内，在设置完 `page.Forms/page.Params` 之后追加一行：

```go
page.Leaks = DetectLeaks(body)
```

### 5.4 单元测试要求（`leak_test.go`，新建）

1. **`TestDetectLeaks_AWSKey`**：body 中嵌入一个符合 `AKIA[0-9A-Z]{16}` 格式的测试字符串（**必须用明显是假的测试值，比如 `AKIAFAKEFAKEFAKEFAKE`，禁止使用任何真实或疑似真实的密钥**），断言命中且 `Match` 字段已脱敏（不等于原始字符串，且不包含中间 8 位以上的原文）。
2. **`TestDetectLeaks_InternalIP`**：分别测试 `10.x`、`172.16-31.x`、`192.168.x` 命中，以及 `172.32.x.x`（超出私有网段）不命中（回归防止正则范围写错）。
3. **`TestDetectLeaks_NoFalsePositiveOnPublicIP`**：body 含 `8.8.8.8` 等公网 IP，断言不命中 `internal_ip` 规则。
4. **`TestMask_ShortString`**：长度 ≤8 的字符串，断言返回固定的 `"****"`，不泄露任何字符。

### 5.5 验收标准

```powershell
go build ./...
go test ./internal/core/scanner/web/crawler/... -v -race
```

---

## 六、Sprint 4：`web_scanner.go` 收口重构（0.5-1 天）

> 这个 Sprint **不依赖** `crawler` 包，是对现有代码的纯重构，先做 4.1（拆分 `fallbackFetch`）再做 4.2（抽出 `buildWebResult`），两步顺序不能反——第二步要用到第一步暴露出来的统一数据形状。

### 6.1 步骤一：`fallbackScan` → `fallbackFetch`

**现状**（`web_scanner.go` 第 456-573 行）：`fallbackScan(ctx, task, targetURL, startTime) ([]*model.TaskResult, error)`，内部自己组装 `WebResult` 并返回，指纹匹配逻辑（第 523-540 行）也在里面。

**改造为**：

```go
// fallbackFetch 使用 net/http 抓取首页原始数据，不再自行组装 WebResult。
// 返回值供 Run() 主干统一走 buildWebResult 收口处理。
func (s *WebScanner) fallbackFetch(ctx context.Context, targetURL string) (body string, headers map[string]string, statusCode int, title string, links []string, err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", nil, 0, "", nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, 0, "", nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", nil, 0, "", nil, err
	}
	bodyStr := string(bodyBytes)

	headers = make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}

	title = extractTitleFromHTML(bodyStr) // 从原 fallbackScan 里第 497-521 行的 title 提取逻辑抽出为独立函数

	links, _, _ = crawler.ExtractLinksAndForms(targetURL, bodyStr) // 复用 Sprint 2 产出，首页表单/参数由后续 buildWebResult 统一处理

	logger.Infof("[WebScanner] Fallback fetch success for %s", targetURL)
	return bodyStr, headers, resp.StatusCode, title, links, nil
}

// extractTitleFromHTML 从原 fallbackScan 第 497-521 行的重复/带修正注释的实现中整理出的干净版本
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
	return bodyStr[start+7 : start+end]
}
```

> **顺带修复一个现状代码里的真实小问题**：原 `fallbackScan` 第 497-521 行有一段重复计算 `title` 的代码（先算一次，写了一堆注释说明位置算错了，然后又重新算一次），是历史遗留的调试痕迹。`extractTitleFromHTML` 顺手把这段清理成一次到位的正确实现，这不算范围蔓延，是重构必然路过的代码顺手清理。

**必须删除**：原 `fallbackScan` 方法整体（第 456-573 行），改名后不要同时保留两个方法。

### 6.2 步骤二：抽出 `buildWebResult` + `pageData`

在 `web_scanner.go` 内新增（放在 `fallbackFetch` 之后即可）：

```go
// pageData 是 buildWebResult 的统一输入形状，三条数据来源（go-rod 首页/fallback 首页/爬虫子页面）共用
type pageData struct {
	URL        string
	Depth      int
	StatusCode int
	Title      string
	Body       string
	Headers    map[string]string
	Forms      []model.FormInfo
	Params     []string
	Leaks      []model.LeakInfo
	Screenshot string
	Favicon    string
}

// buildWebResult 是首页与爬虫子页面共用的收口函数，替代原本三处重复的 "Input -> Match -> WebResult" 代码
func (s *WebScanner) buildWebResult(task *model.Task, startTime time.Time, ip string, port int, pd pageData) *model.TaskResult {
	input := &fingerprint.Input{
		Target:      task.Target,
		Body:        pd.Body,
		Headers:     pd.Headers,
		StatusCode:  pd.StatusCode,
		RichContext: map[string]interface{}{"body": pd.Body, "headers": pd.Headers, "title": pd.Title},
	}
	var techStack []string
	if s.fpEngine != nil {
		if matches, err := s.fpEngine.Match(input); err == nil {
			techStack = convertMatchesToTechStack(matches)
		}
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
			ContentLength:   int64(len(pd.Body)),
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
```

> **注意与架构方案 8.7.4 节伪代码的一处差异并说明原因**：架构方案伪代码里 `fingerprint.Input` 没有填 `RichContext`，但现状 `Run()` 主干（第 216-220 行）go-rod 路径是把完整 `richCtx`（含 DOM/JS/Meta/Cookies）传给 `Input.RichContext` 的。如果 `buildWebResult` 只传 `Body/Headers/StatusCode`，会导致**依赖 RichContext 的指纹规则在收口重构后完全失效**，这是绝对不能接受的功能回归。正确做法：`Run()` 主干在调用 `buildWebResult` 时，如果是 go-rod 路径拿到了完整 `richCtx`，要把 `richCtx` 通过 `pageData` 传进来（见 6.3 节新增 `RichContext` 字段），`buildWebResult` 内部优先使用调用方传入的 `RichContext`，没有传才退化成用 `Body/Headers/Title` 拼一个最小 `RichContext`。修正后的 `pageData` 和 `buildWebResult`：

```go
type pageData struct {
	URL         string
	Depth       int
	StatusCode  int
	Title       string
	Body        string
	Headers     map[string]string
	RichContext map[string]interface{} // 新增：go-rod 路径传完整 richCtx，fallback/爬虫路径传 nil
	Forms       []model.FormInfo
	Params      []string
	Leaks       []model.LeakInfo
	Screenshot  string
	Favicon     string
}

func (s *WebScanner) buildWebResult(task *model.Task, startTime time.Time, ip string, port int, pd pageData) *model.TaskResult {
	richCtx := pd.RichContext
	if richCtx == nil {
		richCtx = map[string]interface{}{"body": pd.Body, "headers": pd.Headers, "title": pd.Title}
	} else {
		richCtx["headers"] = pd.Headers // 保持与现状 Run() 第 233 行一致的行为：headers 覆盖进 richCtx
	}
	input := &fingerprint.Input{
		Target:      task.Target,
		Body:        pd.Body,
		Headers:     pd.Headers,
		StatusCode:  pd.StatusCode,
		RichContext: richCtx,
	}
	// ... 其余不变 ...
}
```

**这是本文档相对架构方案做出的一处必要修正，实施时必须按本节（而不是架构方案 8.7.4 节原文）执行，否则会造成指纹识别精度的静默回归。**

### 6.3 单元测试要求（修改现有 `web_scanner_test.go`，不是新建）

按照 0.1 节指出的问题，`TestWebScanner_Fingerprint` 必须同步重写：

```go
func TestWebScanner_Fingerprint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "nginx/1.18.0")
		w.Header().Set("X-Powered-By", "PHP/7.4.3")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, "<html><head><title>Test Page</title></head><body>Hello World</body></html>")
	}))
	defer ts.Close()

	scanner := NewWebScanner()
	scanner.ensureInit()

	ctx := context.Background()
	body, headers, statusCode, title, _, err := scanner.fallbackFetch(ctx, ts.URL)
	if err != nil {
		t.Fatalf("fallbackFetch failed: %v", err)
	}

	task := &model.Task{ID: "test-task", Target: "127.0.0.1"}
	result := scanner.buildWebResult(task, time.Now(), "127.0.0.1", 0, pageData{
		URL: ts.URL, StatusCode: statusCode, Title: title, Body: body, Headers: headers,
	})

	res, ok := result.Result.(*model.WebResult)
	if !ok {
		t.Fatal("Result is not *model.WebResult")
	}
	// ... 原有的 TechStack 断言逻辑不变 ...
}
```

新增用例：

- **`TestFallbackFetch_ReturnsSeedLinks`**：mock server 返回含 3 个 `<a href>` 的 HTML，断言 `fallbackFetch` 返回的 `links` 长度为 3。
- **`TestBuildWebResult_ConsistentAcrossSources`**：分别用"go-rod 模拟数据（带 RichContext）"和"fallback 数据（RichContext=nil）"调用两次 `buildWebResult`，对同一份 Body/Headers，断言两次得到的 `TechStack` 完全一致（对应架构方案 8.7.4 节"验证 go-rod 路径和 fallback 路径产出的首页 WebResult 字段一致"的验收要求）。

### 6.4 验收标准

```powershell
go build ./...
go test ./internal/core/scanner/web/... -v -race
```

此时 `Run()` 方法主干还没有改（还是老的一次性 `return []*model.TaskResult{result}, nil` 写法），只是 `fallbackScan`/重复代码被替换成了 `fallbackFetch`/`buildWebResult`，**功能上必须做到零回归**——用现有的 `scan web` CLI 手动跑一次基线站点，输出应该和重构前完全一致：

```powershell
go run ./cmd/agent scan web -t testphp.vulnweb.com --oj before_refactor_check.json
```

（此步骤只是回归验证手段，不是必须提交的产物，跑完确认无误后可删除生成的 json 文件。）

---

## 七、Sprint 5：三处接入点 + 自动决策 + 按需升级联调（1.5-2 天）

这是唯一改动 `Run()` 方法主干的 Sprint，也是唯一一个必须把 Sprint 1-4 的产出全部粘合在一起联调的阶段。拆成 5 个子任务，按顺序做。

### 7.1 子任务 5.1：`Run()` 主干改造——统一收口 + 挂上 crawler

改动文件：`internal/core/scanner/web/web_scanner.go`

把现状 `Run()` 方法（第 66-348 行）按以下结构重写。**不要逐行小修小补，直接按这个骨架重新组织**，因为控制流从"提前 return 两次"变成了"统一收口一次 return"：

```go
func (s *WebScanner) Run(ctx context.Context, task *model.Task) (results []*model.TaskResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[WebScanner] PANIC RECOVERED: %v", r)
			err = fmt.Errorf("panic during web scan: %v", r)
			results = nil
		}
	}()

	s.ensureInit()

	if err1 := s.limiter.Acquire(ctx); err1 != nil {
		return nil, err1
	}
	defer s.limiter.Release()

	startTime := time.Now()
	var protocolHint string
	if p, ok := task.Params["protocol"].(string); ok {
		protocolHint = p
	}
	targetURL := normalizeURL(task.Target, task.PortRange, protocolHint)

	var (
		homeBody       string
		homeHeaders    map[string]string
		homeStatusCode int
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

					if capture, ok := task.Params["screenshot"].(bool); ok && capture {
						if buf, errShot := page.Screenshot(true, nil); errShot == nil {
							screenshotB64 = base64.StdEncoding.EncodeToString(buf)
						}
					}
					faviconB64 = extractFaviconFromPage(page, richCtx)
				}
			} else {
				logger.Warnf("[WebScanner] Navigation failed for %s: %v. Will fallback.", targetURL, errNav)
			}
		}
	} else {
		logger.Warnf("[WebScanner] Failed to launch browser: %v. Will fallback.", errLaunch)
	}

	// --- 统一降级：只要 go-rod 路径没有拿到 body，一律走 fallbackFetch ---
	if homeBody == "" {
		body, headers, statusCode, title, links, errFetch := s.fallbackFetch(ctx, targetURL)
		if errFetch != nil {
			s.limiter.OnFailure()
			return nil, fmt.Errorf("both browser and fallback fetch failed: %w", errFetch)
		}
		homeBody, homeHeaders, homeStatusCode, homeTitle, seedLinks = body, headers, statusCode, title, links
		homeRichCtx = nil // fallback 路径没有富上下文
	}

	finalIP, finalPort := resolveIPPortForResult(task, targetURL, remoteIP, remotePort)

	homeResult := s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
		URL: targetURL, Depth: 0, StatusCode: homeStatusCode, Title: homeTitle,
		Body: homeBody, Headers: homeHeaders, RichContext: homeRichCtx,
		Screenshot: screenshotB64, Favicon: faviconB64,
	})
	results = append(results, homeResult)

	// --- 是否触发深度爬取：三态判断，见 7.2 节 decideCrawlDepth ---
	depth := s.resolveCrawlDepth(task, homeStatusCode, homeHeaders, seedLinks)
	if depth > 0 && len(seedLinks) > 0 {
		cr := crawler.New(crawler.Options{MaxDepth: depth}, s.limiter)
		pages := cr.Crawl(ctx, targetURL, seedLinks)

		s.escalateIfNeeded(ctx, cr, pages) // 见 7.4 节

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
```

**改动点清单（对照现状代码逐条核对，不能遗漏）**：

1. `resolveIPPortForResult` 是把现状第 294-325 行"兜底 IP/Port"那段逻辑抽出来的独立函数（原来是内联在 `Run()` 里的，抽出来是因为 `finalIP/finalPort` 现在要在 `homeResult` 和爬虫子页面结果之间共用）。
2. `extractFaviconFromPage` 是把现状第 253-279 行的 favicon 提取逻辑抽出来的独立函数，签名 `func extractFaviconFromPage(page *rod.Page, richCtx map[string]interface{}) string`。
3. 原来两处 `fallbackScan` 调用点（Launch 失败、Navigate 失败）合并成一处统一判断 `if homeBody == ""`，这是修复 0.2 节和架构方案 8.7 节共同指出的"降级路径爬虫失效"缺陷的核心改动。
4. `results` 从单元素 `return` 改成 `append` 模式。

### 7.2 子任务 5.2：`decideCrawlDepth` 自动决策函数

新增独立函数（放 `web_scanner.go` 内）：

```go
// decideCrawlDepth 基于首页免费信号自动判断是否需要深度爬取，返回 0 表示不爬。
// 判断依据只有三个：状态码、Content-Type、种子链接数量，不引入任何需要额外网络请求的信号。
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

// resolveCrawlDepth 综合三态参数与自动判断，得出最终爬取深度
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
```

**单元测试要求**（新增 `web_scanner_decide_test.go` 或追加进现有测试文件）：

| 用例 | statusCode | contentType | seedLinksCount | 期望 depth |
|---|---|---|---|---|
| 正常站点 | 200 | text/html | 10 | 2 |
| 404 | 404 | text/html | 0 | 0 |
| 401 但有链接 | 401 | text/html | 3 | 2 |
| 纯 JSON API | 200 | application/json | 0 | 0 |
| 200 但无链接 | 200 | text/html | 0 | 0 |
| 500 | 500 | text/html | 5 | 0 |

`resolveCrawlDepth` 额外测试三态分流：

- `task.Params["crawl"] = true`：不管首页信号如何，返回非 0（验证显式开启优先级最高）。
- `task.Params["crawl"] = false`：不管首页信号如何，返回 0。
- `task.Params` 不含 `crawl` key：走 `decideCrawlDepth` 分支（用上表任一组合验证）。
- `task.Params["crawl"] = true` + `task.Params["crawl_depth"] = 3`：返回 3。

### 7.3 子任务 5.3：三处 CLI/Dispatcher/Adapter 接入点改造

严格按 0.2 节列出的三条路径逐一改，**任何一条漏改都会导致那条路径永远走不到自动判断**：

**(a) `internal/core/options/scan_web.go`** 新增字段与三态解析：

```go
type WebScanOptions struct {
	Target     string
	Ports      string
	Path       string
	Method     string
	Crawl      string // "auto"(默认) / "true" / "false"
	CrawlDepth int
	Output     OutputOptions
}

func NewWebScanOptions() *WebScanOptions {
	return &WebScanOptions{
		Ports:      "80,443",
		Path:       "/",
		Method:     "GET",
		Crawl:      "auto",
		CrawlDepth: 2,
	}
}

func (o *WebScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeWebScan, o.Target)
	task.PortRange = o.Ports
	task.Timeout = 30 * time.Minute

	task.Params["path"] = o.Path
	task.Params["method"] = o.Method

	switch o.Crawl {
	case "true":
		task.Params["crawl"] = true
	case "false":
		task.Params["crawl"] = false
	// "auto" 或任何其他值：不写 key，交给 WebScanner 自动判断
	}
	if o.CrawlDepth > 0 {
		task.Params["crawl_depth"] = o.CrawlDepth
	}

	o.Output.ApplyToParams(task.Params)
	return task
}
```

**(b) `cmd/agent/scan/web.go`** 新增 CLI flag：

```go
flags.StringVar(&opts.Crawl, "crawl", opts.Crawl, "是否启用深度爬取: auto(默认，自动判断)/true/false")
flags.IntVar(&opts.CrawlDepth, "crawl-depth", opts.CrawlDepth, "爬取深度（仅 --crawl=true 时生效）")
```

**(c) `internal/core/options/scan_run.go`**（`ScanRunOptions`）新增字段：

```go
type ScanRunOptions struct {
	// ... 现有字段不变 ...
	NoWeb         bool
	WebScreenshot bool
	WebCrawl      *bool // nil = 未指定，交给自动判断；非 nil = 用户显式指定
	WebCrawlDepth int
}
```

`cmd/agent/scan/run.go` 现状（第 65-78 行）里所有 flag 都是 `flags.XxxVar(&opts.Field, ...)` 直接绑定到 `opts` 字段的写法，但 `WebCrawl` 是 `*bool`，`pflag` 没有 `PtrBoolVar` 这种东西，不能照搬这个模式，需要一个局部字符串变量 + `RunE` 内手动解析，**解析必须放在 `RunE` 函数体最前面、`opts.Validate()` 调用之前**（因为 `Validate` 之后的逻辑，包括 `pipeline.NewAutoRunner(opts)`，都依赖 `opts.WebCrawl` 已经被正确赋值）：

```go
func NewRunScanCmd() *cobra.Command {
	opts := options.NewScanRunOptions()
	var webCrawl string // 局部变量，三态字符串，不能直接绑定到 opts.WebCrawl（*bool）

	cmd := &cobra.Command{
		Use:   "run",
		// ... Short/Long/Example 不变 ...
		RunE: func(cmd *cobra.Command, args []string) error {
			// 三态字符串解析为 *bool，必须在 opts.Validate() 之前完成
			switch webCrawl {
			case "true":
				v := true
				opts.WebCrawl = &v
			case "false":
				v := false
				opts.WebCrawl = &v
			// "auto" 或其他值：opts.WebCrawl 保持 nil，交给自动判断
			}

			if err := opts.Validate(); err != nil {
				return err
			}
			// ... 其余 RunE 逻辑不变 ...
		},
	}

	flags := cmd.Flags()
	// ... 现有 flags.XxxVar 不变 ...

	// Web 参数新增
	flags.StringVar(&webCrawl, "crawl", "auto", "是否启用深度爬取: auto(默认，自动判断)/true/false")
	flags.IntVar(&opts.WebCrawlDepth, "crawl-depth", 2, "爬取深度（仅 --crawl=true 时生效）")

	return cmd
}
```

`WebCrawlDepth` 是普通 `int`，可以照现有风格直接 `flags.IntVar(&opts.WebCrawlDepth, ...)` 绑定，不需要额外处理；只有 `WebCrawl` 这一个字段因为要表达三态需要绕一下，这是本文档唯一一处 flag 注册方式和现有代码风格不同的地方，原因已经写清楚，不是随意发挥。

**(d) `internal/core/pipeline/dispatcher.go`** 第 274-286 行 `runWebScan` 内追加：

```go
task := model.NewTask(model.TaskTypeWebScan, pCtx.IP)
task.PortRange = fmt.Sprintf("%d", p)
if proto != "" {
	task.Params["protocol"] = proto
}

if d.opts != nil {
	task.Params["screenshot"] = d.opts.WebScreenshot
	if d.opts.WebCrawl != nil {
		task.Params["crawl"] = *d.opts.WebCrawl
	}
	if d.opts.WebCrawlDepth > 0 {
		task.Params["crawl_depth"] = d.opts.WebCrawlDepth
	}
} else {
	task.Params["screenshot"] = false
}
```

**(e) `internal/service/adapter/task_to_core.go`** 第 86-89 行，删除硬编码：

```go
case "web_scan", "webScan":
	coreTask.Type = model.TaskTypeWebScan
	coreTask.Params["method"] = "GET"
	// 不再写死 coreTask.Params["crawl"] = true。
	// 若 Master 下发的任务里携带了爬虫相关的显式参数，在此处透传（具体字段名以 Master 协议定义为准，
	// 若当前 Master 协议尚未定义对应字段，本行为空即可，效果是这条路径自动走 WebScanner 的自动判断）。
```

> 实施时需要检查 `task_to_core.go` 的输入结构体（Master 下发的原始任务参数）里是否已经预留了类似 `crawl` 的字段可以透传，如果没有，就是本文档描述的这样——直接删除硬编码这一行，让这条路径自然落入 `resolveCrawlDepth` 的 `default` 分支（自动判断）。这符合架构方案 8.9 节"统一 Master 下发任务与 CLI 任务的决策路径"的目标。

### 7.4 子任务 5.4：按需升级机制 `escalateIfNeeded`

改动文件：`internal/core/scanner/web/crawler/crawler.go`（三层检测）+ `web_scanner.go`（按需升级编排）

**(a) `crawler.go` 新增三层检测**（架构方案 8.8.2/8.8.3 节已给判断表，本节给出可编译的正则实现）：

```go
var (
	jsRedirectPattern = regexp.MustCompile(`(?i)(location\.(href|replace|assign)\s*=|window\.location\s*=)`)
	spaRootPattern    = regexp.MustCompile(`(?i)<div[^>]+id=["'](root|app)["'][^>]*>\s*</div>`)
	tagStripPattern   = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlTagPattern    = regexp.MustCompile(`(?is)<[^>]+>`)
)

func detectEscalation(body string) (needs bool, reason string) {
	if isJSRedirect(body) {
		return true, "js_redirect"
	}
	if isSPAShell(body) {
		return true, "spa_shell"
	}
	return false, ""
}

func isJSRedirect(body string) bool {
	return len(body) < 1024 && jsRedirectPattern.MatchString(body)
}

func isSPAShell(body string) bool {
	if !spaRootPattern.MatchString(body) {
		return false
	}
	stripped := tagStripPattern.ReplaceAllString(body, "")
	visibleText := htmlTagPattern.ReplaceAllString(stripped, "")
	return len(strings.TrimSpace(visibleText)) < 200
}
```

在 `fetchAndExtract` 里追加（HTTP 3xx 跳转交给 `http.Client` 默认行为处理，不需要手动实现第一层）：

```go
page.Leaks = DetectLeaks(body)
page.NeedsEscalation, page.EscalationReason = detectEscalation(body)
```

**(b) `web_scanner.go` 新增 `escalateIfNeeded` 与 `renderWithBrowser`**：

```go
const defaultMaxEscalationPages = 10

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
```

**注意**：`escalateIfNeeded` 内的 for 循环是**串行**渲染每个待升级页面（不是并发开多个浏览器 Tab），这是刻意的——架构方案 8.8.4 节的成本账已经说明升级是"极少数页面"的兜底路径，不值得为它单独设计并发控制，串行、简单、够用。硬上限 `defaultMaxEscalationPages=10` 本身就保证了最坏情况也只是 10 次串行渲染，量级可接受。

**单元测试要求**（`crawler/escalation_test.go` + `web_scanner_test.go` 追加）：

1. **`TestDetectEscalation_JSRedirect`**：body 是 `<html><script>location.href='/real'</script></html>`（小于 1KB），断言 `needs=true, reason="js_redirect"`。
2. **`TestDetectEscalation_SPAShell`**：body 是 `<div id="root"></div><script>...(占位超过 1KB 的脚本)...</script>`，断言 `needs=true, reason="spa_shell"`。
3. **`TestDetectEscalation_NormalPage`**：正常业务页面（有实际可见文本 > 200 字符），断言 `needs=false`。
4. **`TestDetectEscalation_NoFalsePositiveOnNormalReactApp`**：`<div id="root">` 但内部有大量文本内容（模拟服务端渲染的 React 应用，即 SSR），断言 `needs=false`（防止误报，这是 SPA 空壳检测最容易踩的坑）。
5. **`TestEscalateIfNeeded_ExceedsMaxPages`**：构造 11 个 `NeedsEscalation=true` 的 `Page`，断言 `escalateIfNeeded` 直接跳过（不调用 `Launch`），可以通过 mock 或者判断日志/计数器验证。

### 7.5 子任务 5.5：端到端联调验证

**验证矩阵**（严格按架构方案 10.94 节列出的验收点执行，缺一不可）：

| 场景 | 命令 | 期望结果 |
|---|---|---|
| 正常多页站点，不传 `--crawl` | `scan web -t testphp.vulnweb.com` | 自动判断触发爬虫，结果里出现 `Depth>0` 的条目 |
| 404 目标，不传 `--crawl` | `scan web -t <一个必定 404 的路径/端口>` | 只有首页 1 条结果，不触发爬虫 |
| 纯 JSON API，不传 `--crawl` | 起一个本地 `httptest.Server` 返回 `Content-Type: application/json` | 只有首页 1 条结果 |
| 正常站点，显式 `--crawl=false` | `scan web -t testphp.vulnweb.com --crawl=false` | 只有首页 1 条结果，覆盖自动判断 |
| 浏览器不可用 + `--crawl=true` | 临时改坏 Chromium 路径配置，`--crawl=true` | 依然能拿到子页面结果（验证 0.2 节 + 6.1 节修复的核心缺陷） |
| SPA 站点 | 本地起一个 React demo 或用公开的 SPA 测试站点，`--crawl=true` | 空壳页面被正确标记并升级渲染，渲染后新链接汇入 BFS，最终结果里能看到浏览器只启动了个位数次（通过日志观察 `escalateIfNeeded` 调用次数） |
| CSV/JSON/CLI 三种输出 | 对同一次扫描分别加 `--oj`/`--oc` | 三种格式的 `Depth/Forms/Params/Leaks` 字段一致 |
| `scan run` 全流程 | `scan run -t <目标网段>` | Web 阶段自动判断逻辑同样生效（验证 dispatcher.go 接入点） |

### 7.6 Sprint 5 验收标准

```powershell
cd c:\mytools\code\go\NeoScan\neoAgent
go build ./...
go vet ./...
go test ./... -race
```

全部通过后，按 7.5 节验证矩阵手动跑一遍，逐项打勾。全部打勾后，Sprint 5 才算完成。

---

## 七之二、Sprint 6：多端口探测与协议自适应（1-1.5 天）

### 7a.0 背景：两个真实缺陷，均由 Sprint 5 之后的真实站点测试发现

这两个问题**不是新需求**，是 `Run()` 现有代码里一直存在、Sprint 0-5 的单元测试没有覆盖到的真实 bug，测试用的都是 `httptest.NewServer`（单端口、单协议），没有覆盖"一个目标同时有 HTTP/HTTPS 两个独立服务"或"协议猜错"这两种场景，所以之前没暴露。

**缺陷一：`--ports` 传范围字符串时，`WebScanner` 从未真正探测多个端口**

`internal/core/options/scan_web.go` 第 22 行，`WebScanOptions` 默认 `Ports: "80,443"`——也就是说**不加任何参数的最基础用法**，`task.PortRange` 拿到的就是字符串 `"80,443"`。但 `web_scanner.go` 第 99 行 `targetURL := normalizeURL(task.Target, task.PortRange, protocolHint)` 把这个字符串原样传给 `normalizeURL`，而 `normalizeURL`（第 307-337 行）内部全部是对 `port` 参数做**精确字符串比较**（`port != "80"`、`switch port { case "443": ... }`），整串 `"80,443"` 既不等于 `"80"` 也不等于 `"443"`，也不会命中 `switch` 里任何一个 `case`，于是：
- 第 314 行 `host = target + ":" + port` 会把 host 拼成非法的 `target:80,443`（如果 `target` 本身没带冒号）
- 第 327 行 `switch port` 永远落进 `default` 分支，返回 `http://`

净效果：**`Run()` 从头到尾只探测一个（大概率还是错误猜测出来的）URL，产出一份结果**，`--ports` 参数形同虚设。真实验证过程：`https://www.baidu.com`（默认走 443）能扫到，是因为凑巧走的是 `strings.HasPrefix(target, "https://")` 那个提前 return 分支（用户在 `-t` 里直接带了完整协议），不代表 `"80,443"` 这个范围字符串被正确解析过。

**缺陷二：非标准端口猜错协议后，没有任何纠错机会**

`normalizeURL` 第 332-335 行自己的注释已经承认：「其他端口默认为 http，如果失败，Scanner 内部其实很难再自动切 https」。真实测试命中过这个缺陷：对 `10.201.28.126:9000`（实际是 HTTPS 服务）不显式指定协议时，`normalizeURL` 猜成 `http://`，请求发过去后拿到的是 nginx 返回的 `400 The plain HTTP request was sent to HTTPS port`——**这是一次拿到了响应但协议不对**的场景；而如果目标是"HTTPS 打到一个只监听 HTTP 明文的端口"或者"目标端口防火墙丢包"，则完全拿不到任何响应（超时/`EOF`/`connection reset`），错误文本因平台、Go 版本、对端实现而完全不一致，**不能靠 `strings.Contains` 匹配错误文案来判断"是不是猜错协议了"**，这种方案在目标沉默失败时必然漏判。

### 7a.1 改动文件清单

| 文件 | 操作 |
|---|---|
| `internal/core/scanner/web/web_scanner.go` | `Run()` 拆分出 `runOnePort` 私有方法；新增端口范围解析、多端口并发编排逻辑；`fallbackFetch` 失败路径新增协议翻转重试 |
| `internal/core/scanner/web/web_scanner_multiport_test.go` | 新增：多端口场景（同一目标 80 + 443 各自独立服务）覆盖测试 |
| `internal/core/scanner/web/web_scanner_protocol_test.go` | 新增：协议翻转重试的触发/不触发边界测试 |

**不改动的文件（明确排除，避免范围蔓延）**：`crawler/` 包下所有文件（Sprint 1-3 产出）、`buildWebResult`/`resolveCrawlDepth`/`escalateIfNeeded`（Sprint 4-5 产出）。Sprint 6 只改 `Run()` 这一层编排逻辑，不碰下游已验收的能力。

### 7a.2 设计一：多端口探测——`Run()` 拆分为"编排层 + 单端口执行层"

**核心判断**：不重新发明端口范围解析，直接复用 `internal/core/scanner/port_service/nmap_service` 包已有的 `ParsePortList(s string) []int` 函数（支持 `"80,443"`、`"1-100"`、`"top100"/"top1000"` 别名，逐行核实过，见该包 `parser.go` 第 181 行）。这是"好品味"的具体体现：现成的、已经过测试的解析逻辑就在旁边的包里，没有理由为 `WebScanner` 重新写一份端口范围解析正则。

**新签名**：

```go
// runOnePort 是 Sprint 0-5 里 Run() 函数体的原样内容（去掉最外层的 QoS Acquire/Release，
// 那部分现在只需要在 Run() 里做一次，不需要每个端口重复获取令牌），
// 输入变成"已经确定好协议+端口的完整 URL"，输出是这个端口对应的完整结果集
// （首页结果 + 该端口触发的 BFS 子页面结果，如果有）。
func (s *WebScanner) runOnePort(ctx context.Context, task *model.Task, targetURL string, startTime time.Time) ([]*model.TaskResult, error)

// Run 变成纯编排：解析端口列表 -> 对每个端口算出 targetURL -> 并发调用 runOnePort -> 汇总
func (s *WebScanner) Run(ctx context.Context, task *model.Task) (results []*model.TaskResult, err error)
```

**`Run()` 编排逻辑步骤**：

1. QoS `Acquire`/`ensureInit` 保持在 `Run()` 顶层只做一次，不要每个端口重复获取限流令牌（否则并发数会失控——`s.limiter` 本来就是"整个扫描器的资源保护阀"，不是"每个端口各自的"）。
2. 用 `nmap_service.ParsePortList(task.PortRange)` 解析出端口列表；如果解析结果为空（比如 `task.PortRange` 本身是空字符串），按 Sprint 0-5 现状行为兜底：只探测一次、端口交给 `normalizeURL` 自己判断（保持向后兼容，不让老的调用方式报错）。
3. 对端口列表去重（`ParsePortList` 不保证输入不重复，比如用户手滑传 `"80,80,443"`）。
4. 每个端口调用一次 `normalizeURL(task.Target, port, protocolHint)` 算出各自的 `targetURL`。
5. 多个端口之间**并发**执行 `runOnePort`（用 `sync.WaitGroup` + 收集结果的 `mutex` 保护的 slice，或者 `errgroup`，具体选型看项目现有并发风格是否已有偏好，`crawler.go` 用的是原生 `sync.WaitGroup`，本 Sprint 保持一致不引入新依赖），每个端口内部仍然受 `s.limiter` 统一限流。
6. 单个端口的 `runOnePort` 失败（返回 `error`）**不能让其他端口的结果也丢失**——这是"一个目标多个独立 Web 服务"场景下的核心正确性要求：80 端口探测失败不该连累 443 端口的正确结果消失。因此 `Run()` 汇总时，对每个端口的 `(results, err)` 分别处理：`err != nil` 只记录日志（`logger.Warnf`），不中断其他端口，也不让整个 `Run()` 返回 error；只有**所有端口全部失败**时，`Run()` 才返回 error（把最后一个错误或者用 `errors.Join` 拼接的错误返回给调用方，具体哪种方式实现时再定，原则是不能吞掉所有错误信息）。

**必须覆盖的测试场景**（`web_scanner_multiport_test.go`）：

- `TestRun_MultiPort_BothServicesReachable`：起两个 `httptest.NewServer`（模拟同一目标的 80 和 443 两个独立服务，内容不同），断言 `Run()` 返回的结果里两个端口的内容都存在，互不覆盖。
- `TestRun_MultiPort_OneUnreachable`：两个端口中一个正常、一个直接拒绝连接，断言正常端口的结果仍然完整返回，不因为另一个端口失败而丢失。
- `TestRun_MultiPort_EmptyPortRange`：`task.PortRange` 为空字符串时，行为与 Sprint 5 现状完全一致（回归测试，防止本次改动破坏老的调用方式，比如 `task_to_core.go` 如果有不传端口范围的调用路径）。
- `TestRun_MultiPort_DuplicatePorts`：`task.PortRange = "80,80,443"`，断言 80 端口只被探测一次（结果里不会出现两条重复的 80 端口结果）。

### 7a.3 设计二：协议自适应——仅在"确认无响应"时翻转协议重试一次

**核心判断（已与用户对齐两轮，收窄到最终版本）**：判断依据不能是"错误文本里有没有某个特征字符串"（脆弱，依赖对端实现，目标沉默失败时完全失效），必须是"传输层连接有没有建立成功"这个 Go 标准库自身就能提供的、平台无关的结构化信号。

**触发条件（必须同时满足，缺一不可）**：

1. 这个端口的协议是"猜"出来的，不是用户显式指定的（即调用 `normalizeURL` 时 `protocolHint == ""`，走的是内部 `switch port` 默认猜测分支）。
2. go-rod 路径和 `fallbackFetch` 路径都失败，且失败原因发生在**应用层握手/协议不匹配阶段**，而不是**TCP 连接阶段**——用 `errors.As` 判断：
   - 如果错误是 `*net.OpError` 且 `Op == "dial"`：说明 TCP 三次握手都没完成（端口没监听/网络不通/防火墙拒绝），这种情况**不重试**，换协议也连不上同一个端口，重试是浪费一次完整超时时间。
   - 如果 TCP 连接已建立，但后续在 TLS 握手阶段失败（错误类型是 `tls.RecordHeaderError`，或者 `*net.OpError` 里 `Op` 是 `"read"`/`"remote error"` 这种发生在读数据阶段的错误），说明"端口是通的，但对端不认识我们发送的协议"——这才是**值得翻转协议重试**的场景。
3. 满足以上两条，才把 `targetURL` 的 scheme 翻转（`http://` ↔ `https://`），只调用一次 `fallbackFetch` 重试（不需要重新走一遍 go-rod，翻转重试的目的只是"验证协议对不对"，用成本更低的 `net/http` 验证即可；如果 `fallbackFetch` 翻转后成功，直接采用这次的数据组装结果）。

**必须遵守的边界（明确排除，避免过度设计）**：

- 只翻转重试**一次**，翻转后依然失败就正常走现有报错路径，不允许二次翻转（会形成 `http→https→http→...` 的无意义循环）。
- 不对"TCP 层都没连上"的情况做翻转重试——目标机器根本不可达时，换协议也无济于事，徒增一倍超时等待。
- 不引入 httpx 那种"HTTP/HTTPS 无条件都探测一遍"的模式（见 `docs/爬虫/httpx与xray参考价值评估.md`），那是批量扫描器为了消除输入歧义而接受的双倍开销，与 NeoScan 单目标高成本扫描（可能启动浏览器、触发 BFS）的定位不匹配。
- 翻转重试成功后，`homeResult.URL` 必须使用翻转后真正成功的协议，不能让最终结果里协议字段和实际抓取来源的协议对不上；同时用 `logger.Warnf` 记录一条"自动纠正协议：原猜测 X，实际为 Y"，保持对用户透明，不做静默黑魔法。
- BFS 子页面爬取阶段（`crawler.Crawl`）**不需要**同样的协议翻转逻辑——子页面链接是从已成功抓取的页面里提取出的绝对/相对 URL，协议信息是明确的，不存在"猜"的环节，此处硬加纯属过度设计。

**必须覆盖的测试场景**（`web_scanner_protocol_test.go`）：

- `TestProtocolFallback_HTTPGuessedButHTTPSOnly_Retries`：起一个只监听 TLS 的 `httptest.NewTLSServer`，构造一个会被 `normalizeURL` 猜成 `http://` 的端口场景（比如非常规端口号），断言最终结果是通过翻转到 `https://` 后成功拿到的。
- `TestProtocolFallback_ExplicitProtocolHint_NeverRetries`：`task.Params["protocol"]` 显式传了值，即使猜测的协议连接失败，也不应该触发翻转重试（因为触发条件第 1 条不满足）。
- `TestProtocolFallback_TCPUnreachable_NeverRetries`：目标端口直接拒绝连接（`net.Dial` 阶段失败），断言不会触发翻转重试，且总耗时不应该出现"两倍超时"的现象（可以用 `time.Since` 断言总耗时在一个超时窗口以内，而不是两个）。
- `TestProtocolFallback_RetryFailsToo_ReturnsOriginalError`：翻转重试后依然失败，断言最终返回的 error 不为 nil，且不会有第二次翻转（可以用一个计数器包装 `http.RoundTripper` 断言总请求次数恰好是 2 次：原始 1 次 + 翻转重试 1 次）。

### 7a.4 Sprint 6 验收标准

```powershell
cd c:\mytools\code\go\NeoScan\neoAgent
go build ./...
go vet ./...
go test ./internal/core/scanner/web/... -race
```

全部通过后，额外做一次真实站点回归（不进自动化测试，人工执行）：

```powershell
# 验证多端口：找一个已知同时开 80 和 443 的目标，断言两个端口都出现在结果里
go run ./cmd/agent scan web -t <目标> -p 80,443

# 验证协议翻转：找一个非标准端口但实际是 HTTPS 的目标（比如之前测过的 10.201.28.126:9000），
# 不显式传 protocol，断言最终能正常拿到结果而不是报错
go run ./cmd/agent scan web -t <目标> -p 9000
```

两项都通过后，Sprint 6 才算完成。

---

## 八、收尾：文档与开发进度更新

以下操作在 Sprint 5 验收全部通过之后进行，**不要提前做**：

1. 更新 `neoAgent/docs/开发进度.md`：把 "P0: Phase 5.1" 状态从 `🏃 进行中` 改为 `🟢 已完成`，在 Changelog 里追加一条本次交付记录（爬虫 BFS、攻击面提取、被动泄露检测、自动决策机制、按需升级机制）。
2. 更新 `internal/core/scanner/web/README.md`：在"核心能力"里补充"深度爬取"一节，更新架构 Mermaid 图（加上 crawler 节点），更新"输出字段说明"表格加入 `depth/forms/params/leaks`。

这两处文档更新**只在代码全部验收通过后**进行，避免文档和代码状态不一致。

**Sprint 6 完成后，额外补充**：

3. 更新 `neoAgent/docs/开发进度.md`：在 Changelog 追加 Sprint 6 交付记录（多端口探测、协议自适应翻转重试），并注明这是真实站点测试驱动发现并修复的原生缺陷，不是新增能力需求。
4. 更新 `internal/core/scanner/web/README.md`：第 1.4 节「智能调度」补充"多端口并发探测"与"协议自适应重试"两条能力说明；第 5 节「开发指南」补充 `runOnePort` 的位置说明。

---

## 九、风险与回滚预案

| 风险 | 触发条件 | 缓解措施 |
|---|---|---|
| BFS 队列关闭时机实现错误导致死锁 | Sprint 1 并发测试覆盖不足 | 3.5 节 `TestCrawl_ConcurrencyNoDeadlock` 必须用 `-race` 跑通，且需要在 CI 里跑至少 3 次（并发 bug 有概率性，一次通过不代表没问题） |
| `buildWebResult` 收口后指纹识别精度回归 | 6.2 节 `RichContext` 传递遗漏 | 6.3 节 `TestBuildWebResult_ConsistentAcrossSources` 是硬性门槛，必须通过才能进入 Sprint 5 |
| 三处接入点漏改一处，自动判断名存实亡 | 0.2 节列出的路径遗漏 | 7.3 节按 (a)-(e) 五个子项逐条勾选，Code Review 时逐条对照检查 |
| SPA 空壳检测误报，把正常 SSR 页面也升级 | 阈值（200 字符）设置不合理 | 7.4 节 `TestDetectEscalation_NoFalsePositiveOnNormalReactApp` 是专门为这个风险设计的用例，实施时如发现真实站点误报率高，可调整阈值，但必须补充对应的回归测试用例 |
| 现有单测因签名变更编译失败但被忽略 | Sprint 4 只改生产代码没改测试 | Sprint 4 验收标准明确写了 `go test ./internal/core/scanner/web/...`，不通过不能进入 Sprint 5 |
| 多端口并发探测导致资源占用/QPS 失控 | Sprint 6 每个端口各自调用 `runOnePort`，如果误用独立的限流器而不是共享 `s.limiter` | 7a.2 节明确要求"QoS Acquire/Release 只在 Run() 顶层做一次"，Code Review 时检查 `runOnePort` 内部不能出现第二次 `s.limiter.Acquire` |
| 协议翻转重试对"目标彻底不可达"场景造成双倍超时 | 触发条件误判，把 TCP dial 失败也当成"协议猜错"去重试 | 7a.3 节触发条件第 2 条用 `errors.As` 严格区分 `dial` 阶段和握手/读取阶段失败，`TestProtocolFallback_TCPUnreachable_NeverRetries` 是专门覆盖这个风险的用例，必须断言总耗时不超过一个超时窗口 |
| 协议翻转重试形成无限循环 | 翻转后再次失败，代码误判又翻转回去 | 7a.3 节明确"只翻转重试一次"，`TestProtocolFallback_RetryFailsToo_ReturnsOriginalError` 用请求计数断言总请求数恰好为 2 |
| 多端口结果互相覆盖或一个端口失败拖累其他端口 | `Run()` 汇总逻辑没有对每个端口的 `(results, err)` 分别处理 | `TestRun_MultiPort_OneUnreachable` 是硬性门槛，必须断言正常端口结果完整存在 |

---

## 十、每个 Sprint 结束时必须能回答的问题（自查清单）

实施者在每个 Sprint 声称"完成"之前，必须能对本 Sprint 的产出逐条回答"是"：

- [ ] `go build ./...` 和 `go vet ./...` 全绿？
- [ ] 本 Sprint 新增/修改的代码都有对应单元测试，且 `-race` 跑通？
- [ ] 是否有任何现有测试因为这次改动而编译失败或行为改变？如果有，是否已同步修复？
- [ ] 是否有任何函数签名的改动，会影响到本文档后续 Sprint 里已经写好的调用代码？如果有，本文档是否需要同步修正（发现问题应先反馈，不要自行改变后续 Sprint 的既定接口）？
- [ ] 本 Sprint 的改动是否越界修改了不属于本 Sprint 范围的文件？（例如 Sprint 1-3 不应该动 `web_scanner.go`）
