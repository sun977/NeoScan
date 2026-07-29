# NeoScan Web 爬虫与被动分析器架构方案 v2.0

> 本文档独立于 `Web爬虫与被动分析器设计方案-v1.0.md` 给出，基于对 `BrowsertrixCrawler`、`Firecrawl` 源码的实际研读，
> 结合 NeoScan 现有代码（`internal/core/scanner/web`、`internal/core/pipeline`、`internal/core/factory`、`internal/pkg/fingerprint`）
> 给出的重新思考版本。核心目标只有一个：**用最少的新概念，把 Phase 5.1 的活干完，并且不破坏现有任何东西。**

---

## 一、先给结论

【核心判断】
✅ 值得做，但**不是**照搬 Browsertrix/Firecrawl 的架构。

这两个项目的抓取内核（Redis ZSET 分布式队列 + Lua 原子脚本、Engine quality/feature-flag 十几种引擎瀑布降级）都是为了解决它们自己的问题：
- Browsertrix 要解决的是**多容器分布式、可暂停可恢复的网页存档**（要保证 crawl 状态可以在 Pod 重启后 replay）。
- Firecrawl 要解决的是 **SaaS 化的"给我一个 URL，尽最大可能返回可用内容"**（要应对反爬、PDF、Twitter、Wikipedia 等各种奇葩站点）。

NeoScan 是**单进程 Agent，跑一次扫描，目的是发现攻击面**，不需要断点续爬、不需要跨引擎瀑布重试、不需要分布式状态机。如果照搬这套东西，就是"用航母的锅炉去烧开水"——这是典型的 **过度设计**，是要不得的。

真正的问题是：**现有 `WebScanner` 只探测了首页，攻击面收集停留在"单点快照"，没有"顺藤摸瓜"的能力**。要解决的就是这一个问题，别的都不要碰。

---

## 二、第一层：数据结构分析

> "Bad programmers worry about the code. Good programmers worry about data structures."

先看现状。NeoScan 当前的核心数据流是：

```
model.Task ──(Scanner.Run)──> []model.TaskResult{ Result: *model.WebResult }
```

`WebResult` 目前长这样（`internal/core/model/result_types.go`）：

```12:20:c:/mytools/code/go/NeoScan/neoAgent/internal/core/model/result_types.go
// WebResult Web扫描结果
type WebResult struct {
	URL             string            `json:"url"`
	IP              string            `json:"ip"`
	...
}
```

它是一个**单 URL 的快照结构**。而爬虫要产出的东西，本质上是"以 IP:Port 为根的一棵 URL 树，附带每个节点的攻击面标注"。这就带来第一个关键决策：

### 决策 1：爬虫的产出不是新的 TaskType，而是 WebResult 的“复数化”

v1.0 方案里，隐含的假设是"爬虫是 WebScanner 内部的一个子模块，爬完之后塞回 WebResult"。这个方向是对的，但 v1.0 没有回答一个根本问题：**爬到的 100 个页面，每个页面的指纹、状态码、表单信息，要不要都单独构成一条 `WebResult`？**

答案必须是 **Yes**。理由：
1. Pipeline 的下游（Vuln Scanner、Dir Scanner）消费的是 `[]*model.WebResult`，它们要拿到的是"每一个具体 URL 的攻击面"，而不是一个塞满子 URL 的大对象。
2. 现有 `pCtx.AddWebResult(res *model.WebResult)` 已经是"多次调用，逐条累加"的模式（参见 `pipeline/pipeline.go` 第 75-79 行）。爬虫只需要复用这个入口，对 Pipeline 和 Dispatcher **零改动**。
3. 如果搞一个 `WebResult.Children []WebResult` 或者单独一个 `CrawlResult` 大对象，就是在制造特殊情况——下游代码要么要多写一个分支处理"聚合结果"，要么要多写一次展开逻辑。**消除特殊情况的方法就是：爬虫内部产出多少个 URL，就向 Pipeline 报多少条 `WebResult`。**

所以数据结构上只需要做一件事：给 `WebResult` 增加"攻击面"相关的可选字段（不影响现有字段和现有调用方）：

```go
// WebResult 新增字段（其余字段不变）
type WebResult struct {
    // ... 现有字段不变 ...

    Depth       int               `json:"depth,omitempty"`        // 爬取深度，0=首页
    Forms       []FormInfo        `json:"forms,omitempty"`        // 表单/输入点
    Params      []string          `json:"params,omitempty"`       // URL Query 参数名集合
    Leaks       []LeakInfo        `json:"leaks,omitempty"`        // 被动泄露检测结果
}

type FormInfo struct {
    Action string   `json:"action"`
    Method string   `json:"method"`
    Fields []string `json:"fields"`
}

type LeakInfo struct {
    Type    string `json:"type"`    // aksk / jwt / internal_ip / ...
    Match   string `json:"match"`   // 脱敏后的命中内容
    Context string `json:"context,omitempty"`
}
```

这是**加字段，不是加类型**。老代码（CLI 表格输出、CSV/JSON Reporter）完全不受影响，`Headers()/Rows()` 不需要改，因为这些新字段只在需要时才展示（比如新增一列 `Leaks` 计数，属于锦上添花，不是本次范围）。

### 决策 2：去重表用什么结构？—— 一个 `map[string]struct{}` 就够了，别整 Redis/Heap

v1.0 提出用 `container/heap` 实现优先级队列、用哈希存储归一化 URL。Browsertrix 用 Redis ZSET + 5 个 Lua 脚本实现同样的事情。

问一个实用主义的问题：**NeoScan 单次爬取的 URL 量级是多少？** 一个安全扫描场景下的中小型 Web 应用，深度 2-3 层，一般是几十到几百个 URL，最多上千。这种量级：

- 排序用不着堆，`slice` 或者一个 `chan` 都能顶。
- 去重用不着 Redis，一个加锁的 `map[string]struct{}` 完全够用，Agent 是单机单进程。

**好品味的做法是：数据结构的复杂度要匹配数据的真实规模。** 引入 heap 不是错，但如果一个 `sync.Map` + 简单 FIFO 就能解决问题，堆就是过度设计——它带来的收益（严格的优先级排序）在这个规模下约等于 0，但带来的心智负担和 bug 面是实打实的。

**结论**：用最笨但最清晰的方式——一个受 `sync.Mutex` 保护的 `visited map[string]struct{}` 做去重，一个 **带缓冲的 channel 当队列**（`chan *crawlItem`）做广度优先遍历。深度限制通过 `crawlItem.Depth` 字段判断，超过 `MaxDepth` 就不入队，没有分支，没有堆，没有打分公式。

---

## 三、第二层：特殊情况识别

把 v1.0 和两个参考项目里的"特殊情况"摘出来看看，哪些是真需求，哪些是可以设计掉的：

| 特殊情况 | v1.0 / 参考项目的处理方式 | Linus 式处理 |
|---|---|---|
| URL 带 Hash / 排序参数不一致导致误判为不同 URL | 归一化引擎 + 相似度计算 + 参数模式聚合（Pattern Aggregation） | **只做最基本的归一化**（去 Fragment、Query 排序），不做"相似度聚合"——这是在猜测用户意图，真遇到 `/article/1..1000` 的翻页，靠 `MaxURLsPerPattern`（对同一路径模板计数，超过 N 次直接跳过）这种一行代码就能解决的方式，不需要引入相似度算法 |
| 大文件/非 HTML 资源拖爆内存 | Fast Abort + Content-Type 探测 | 保留，这是真实问题（HTTP 层面 `io.LimitReader` 已经在 `fallbackScan` 里用过，直接复用这个模式） |
</br>
| 403/429 限流 | 自适应限流 | **不新建限流器**，直接复用现有 `qos.AdaptiveLimiter`（`WebScanner` 已经持有一个实例），爬虫和首页扫描共享同一个限流器实例，因为它们打的是同一个目标 |
| 爬虫和 Headless 浏览器混合调度 | "智能降级策略"：首页用 go-rod，深层用 net/http | 保留这个大方向（对，这一点 v1.0 判断准确），但不需要"降级"这个词——本来就该是两种不同职责：**go-rod 只负责首页探测拿到 JS 渲染后的真实 DOM**，**爬虫全程只用 `net/http`**。这不是失败后的降级，是从一开始就该如此的正常路径 |
| 分布式状态持久化/断点续爬 | Redis ZSET + Lua 脚本 | **完全不需要**。Agent 单次任务运行几秒到几十秒，中断了重新发一次 Task 即可，Master-Agent 协议本身就有重试语义。引入 Redis 会强迫 Agent 依赖外部组件，直接违反《Agent重构开发总纲》里"独立、自包含"的第一条原则 |
| 多引擎（PDF/Wikipedia/Twitter/TLS-Client 等）瀑布降级 | Engine quality 矩阵 + feature flag 打分 | **完全不需要**。安全扫描不关心把 PDF 转成完美的 Markdown，只需要识别出"这是个文件资源"就记为资产，不解析 | 

**一句话说清爬虫的本质**：*从一个种子 URL 出发，用 BFS 在同源范围内走 N 层，走过的每一页都过一遍"提取链接 + 提取表单 + 正则找泄露"，输出一批 `WebResult`。*

这句话不需要"打分公式""相似度聚合""Engine 矩阵"任何一个高级词汇就能完整描述。**如果一句话说不清楚，说明方案设计者在解决自己臆想出来的问题。**

---

## 四、第三层：复杂度审查——目录结构怎么摆

v1.0 的目录设计：

```
internal/core/scanner/web/
├── crawler/
│   ├── crawler.go
│   ├── queue.go       # heap + 归一化
│   ├── extractor.go   # goquery 提取
│   ├── analyzer.go    # 正则泄露检测
│   └── filter.go      # Scope + DenialReason
├── web_scanner.go
└── context.go
```

这个划分本身没有大错，问题在于**颗粒度过细**（5 个文件对应一个几百行就能写完的功能），以及 `filter.go` 里的 `DenialReason` 这种"日志美化"需求被拔高成了一个独立模块。审计需求是真实的（"为什么没扫到"确实是排查刚需），但不需要为此设计一个类型系统，一行 `logger.Debugf` 加个原因字符串参数就够了。

**Linus 式精简方案**：3 个文件，每个文件职责单一，没有一个文件是"因为要凑对称性"而存在的。

```
internal/core/scanner/web/
├── web_scanner.go         # 现有文件，改动点：Run() 末尾加一段"是否需要深度爬取"的调用
├── context.go             # 现有文件，不变
└── crawler/                              # 新增
    ├── crawler.go         # BFS 主循环 + 去重 + Scope 判断（这三者是一个功能，不拆）
    ├── extract.go         # 用 goquery 从 HTML 提取 <a>/<form>/参数
    └── leak.go            # 正则泄露检测规则与匹配
```

为什么把"BFS + 去重 + Scope"揉进一个文件？因为它们是**同一个循环体里的顺序步骤**，人为拆到 3 个文件只会增加跳转成本，不会增加内聚性。这才是"消除不必要的抽象层"。

### 3 层缩进检验

`crawler.go` 的主循环伪代码验证是否超过 3 层缩进：

```go
func (c *Crawler) run(ctx context.Context, seedURL string) {
    queue := make(chan *item, 256)
    queue <- &item{URL: seedURL, Depth: 0}

    var wg sync.WaitGroup
    for i := 0; i < c.concurrency; i++ {
        wg.Add(1)
        go c.worker(ctx, queue, &wg)   // 1层：worker 内部再展开
    }
    wg.Wait()
}

func (c *Crawler) worker(ctx context.Context, queue chan *item, wg *sync.WaitGroup) {
    defer wg.Done()
    for it := range queue {                      // 1层
        if !c.shouldVisit(it) {                   // 2层
            continue
        }
        body, links := c.fetchAndExtract(ctx, it) // 2层
        c.reportLeak(it.URL, body)                // 2层
        for _, link := range links {              // 2层
            c.enqueue(queue, link, it.Depth+1)     // 3层
        }
    }
}
```

3 层封顶，`shouldVisit`（去重+Scope+深度）、`fetchAndExtract`（HTTP+goquery）、`reportLeak`（正则）全部下沉成独立函数，主循环只做编排。这是可以正常读下来的代码，不需要注释解释"这里为什么要这样"。

---

## 五、第四层：破坏性分析——会破坏什么？

逐条过一遍现有链路，确认零破坏：

1. **`WebScanner.Run()` 签名不变**，仍然是 `(ctx, task) -> ([]*model.TaskResult, error)`。爬虫是 `Run()` 内部在拿到首页 `richCtx` 之后追加的一段逻辑，通过 `task.Params["crawl"]`（bool）和 `task.Params["crawl_depth"]`（int）控制是否触发、触发多深，**默认关闭**，不传这个参数的老调用方（`ServiceDispatcher.runWebScan`、CLI `scan web`）行为完全不变。
2. **`model.WebResult` 只做字段新增，不删不改**已有字段，JSON 序列化向后兼容，CSV/表格 Reporter 不需要改。
3. **`PipelineContext.AddWebResult` 接口不变**，爬虫产出的每个子页面结果，就是再调用 N 次这个方法，Dispatcher 完全无感知。
4. **不引入新的外部依赖组件**（不需要 Redis，不需要 Rust 扩展），只需要引入 `github.com/PuerkitoBio/goquery`（纯 Go，MIT 协议，`go.mod` 加一行）。
5. **`qos.AdaptiveLimiter` 复用现有实例**，不新建令牌桶，爬虫产生的并发请求和首页扫描共享同一份限流状态，语义更准确（同一个目标站点，不管是首页扫描还是深度爬取，都应该被同一个限流策略约束）。

**唯一需要新增的公共契约**是 `Task.Params` 里两个新 key（`crawl`, `crawl_depth`），这是纯新增，不影响任何现有 Key 的读取逻辑。

---

## 六、第五层：实用性验证

- 这个问题在生产环境真实存在吗？**是**——"进度文档"里 Phase 5.1 明确写着当前是"单点首页探测"，下游 Vuln Scanner（Phase 5.2）需要更多攻击面输入点才有意义，这不是臆想出来的需求。
- 方案的复杂度是否匹配问题的严重性？**匹配**——三个新文件，一个新依赖库，一次 `WebResult` 字段扩展，一个 Task 参数扩展。这是能在 1-2 个迭代内交付、能被单元测试完整覆盖的规模。
- 有没有必要把 v1.0 里的"打分公式""相似度聚合""DenialReason 类型系统"都做出来？**没有**——这些是"可能未来用得上"的功能，属于典型的"解决假想问题"。等真的遇到"深分页地狱"再加"同路径模板计数上限"这一行代码也来得及，不需要提前设计。

---

## 七、最终架构设计

### 7.1 模块与数据流

```
                        ┌───────────────────────────────┐
                        │        WebScanner.Run()         │
                        │  1. go-rod 首页探测 (不变)       │
                        │  2. 提取 richCtx / matches       │
                        │  3. 首页 WebResult (不变)        │
                        │  4. if task.Params["crawl"]:     │
                        │       crawler.Crawl(seedURL)     │────┐
                        └───────────────────────────────┘    │
                                                                │
                        ┌───────────────────────────────┐    │
                        │      crawler.Crawler (新增)      │◄───┘
                        │  - BFS Worker Pool (net/http)    │
                        │  - visited map 去重 + Scope 判断  │
                        │  - 复用 WebScanner 的 AdaptiveLimiter │
                        │  - extract.go: goquery 提取 <a>/<form>│
                        │  - leak.go: 正则敏感信息检测       │
                        └───────────────────────────────┘
                                       │
                                       ▼
                     每访问一个页面 -> 产出一个 *model.WebResult（Depth>0）
                                       │
                                       ▼
                    WebScanner.Run() 汇总为 []*model.TaskResult 一次性返回
                                       │
                                       ▼
                  ServiceDispatcher.runWebScan -> pCtx.AddWebResult (循环调用，逻辑不变)
```

关键设计点：**爬虫不是一个独立的 Scanner/TaskType，而是 `WebScanner` 内部按需触发的一个"深度补充阶段"**。这是与 v1.0 的第二个重大分歧——v1.0 倾向于把 crawler 做成 web 包下平行的子系统，本方案认为它应该是 WebScanner 生命周期里的一环，因为：

1. 只有 WebScanner 拿到首页之后，才知道该不该继续深挖（比如首页 404，直接不用爬）。
2. 复用同一个限流器实例，语义更自洽，不用在 Dispatcher 层协调两个独立 Scanner 的并发关系。
3. 保持 `model.TaskTypeWebScan` 单一职责："给我一个 IP:Port，把这个 Web 服务的攻击面摸清楚"——不管摸清楚的手段是首页快照还是深挖 3 层，对外都是同一个任务类型，Dispatcher 不需要为"web_scan" 和 "web_crawl" 两种任务类型分别写分发规则。

### 7.2 核心类型（放在 `internal/core/scanner/web/crawler/crawler.go`）

```go
package crawler

type Options struct {
    MaxDepth    int           // 默认 2
    MaxPages    int           // 硬上限，默认 200，防止爬虫失控
    Concurrency int           // 默认 5，与 WebScanner 的 QoS 挂钩
    Timeout     time.Duration // 单页超时，默认 10s
    SameHostOnly bool         // 默认 true，只在同一 Host 内爬
}

type Page struct {
    URL        string
    Depth      int
    StatusCode int
    Title      string
    Body       string
    Headers    map[string]string
    Forms      []model.FormInfo
    Params     []string
    Leaks      []model.LeakInfo
}

type Crawler struct {
    opts    Options
    limiter *qos.AdaptiveLimiter // 复用 WebScanner 传入的实例，不新建
    client  *http.Client
    seedHost string

    mu      sync.Mutex
    visited map[string]struct{}
}

func New(opts Options, limiter *qos.AdaptiveLimiter) *Crawler
func (c *Crawler) Crawl(ctx context.Context, seedURL string, seedLinks []string) []*Page
```

`Crawl` 的第三个参数 `seedLinks`——这是刻意设计：首页在 go-rod 阶段已经拿到了 JS 渲染后的真实链接（`ExtractRichContext` 目前没提取 `<a>`，需要补一个小函数 `ExtractLinks(page)`，10 行代码），把这批链接作为 BFS 的第一层种子，避免爬虫重新用 `net/http` 打一次首页（首页可能是 SPA，`net/http` 拿到的是空壳）。这正是 v1.0 中"智能降级策略"的合理内核，本方案把它保留并做得更精确：**不是"首页用浏览器、其余用 HTTP"这种简单的深度切分，而是"浏览器只用于拿到 JS 渲染后的种子链接，一旦拿到链接，后续遍历全部走 HTTP"**。

### 7.3 URL 归一化与去重（`crawler.go` 内部函数，不单独成文件）

```go
func normalizeKey(raw string) string {
    u, err := url.Parse(raw)
    if err != nil {
        return raw
    }
    u.Fragment = ""              // 去 Hash
    q := u.Query()
    // 排序 query key，保证 ?a=1&b=2 与 ?b=2&a=1 视为同一 URL
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
```

不做参数值归一化（比如把 `?id=1` 和 `?id=2` 视为相同），因为那是"猜测"，不同的 `id` 值完全可能对应不同的业务对象（IDOR 漏洞检测还就指望这个差异）。**唯一需要防的是"同一个资源被不同顺序/大小写的 URL 重复访问"，不是"业务上相似的 URL"。** 如果真遇到 1000 个自增 ID 的爬取地狱，用 `MaxPages` 硬上限兜底即可，这是实用主义，不是理论洁癖。

### 7.4 攻击面提取（`extract.go`）

```go
func ExtractLinksAndForms(baseURL string, body string) (links []string, forms []model.FormInfo, params []string) {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
    if err != nil {
        return nil, nil, nil
    }
    base, _ := url.Parse(baseURL)

    doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
        href, _ := s.Attr("href")
        if abs := resolve(base, href); abs != "" {
            links = append(links, abs)
        }
    })

    doc.Find("form").Each(func(_ int, s *goquery.Selection) {
        action, _ := s.Attr("action")
        method, _ := s.Attr("method")
        if method == "" {
            method = "GET"
        }
        var fields []string
        s.Find("input[name],select[name],textarea[name]").Each(func(_ int, f *goquery.Selection) {
            if name, ok := f.Attr("name"); ok {
                fields = append(fields, name)
            }
        })
        forms = append(forms, model.FormInfo{Action: resolve(base, action), Method: strings.ToUpper(method), Fields: fields})
    })

    if base.RawQuery != "" {
        for k := range base.Query() {
            params = append(params, k)
        }
    }
    return
}
```

引入 `goquery` 是本方案唯一的新依赖，理由：它是 Go 生态里最成熟的 HTML 解析库（对标 jQuery API），比手写正则解析 HTML 靠谱得多——**这不是过度设计，是用对的工具做对的事**。v1.0 也建议了同样的库，这一点结论一致。

### 7.5 被动泄露检测（`leak.go`）

```go
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

规则先内置 4-5 条最高频的（AK/SK、JWT、内网 IP），**不要一上来就搞规则文件加载/热更新体系**——`rules/fingerprint/web/` 目录现有的 JSON 规则加载机制是给指纹用的，泄露检测的正则如果以后条数多了，参照同样的模式加一个 `rules/leak/leak_rules.json` 即可，那是"确实需要时再加"的复杂度，现在不需要为一个 4 条规则的功能设计配置文件加载器。

---

## 八、与现有代码的具体接入点

### 8.1 `internal/core/scanner/web/web_scanner.go`

在第 6 步"提取 Rich Context"之后（约第 211 行）追加：

```go
// 6.5 深度爬取 (可选，由 Task.Params["crawl"] 控制)
if enable, ok := task.Params["crawl"].(bool); ok && enable {
    depth := 2
    if d, ok := task.Params["crawl_depth"].(int); ok && d > 0 {
        depth = d
    }
    seedLinks := ExtractLinks(page) // context.go 新增的小函数，同 ExtractRichContext 风格
    cr := crawler.New(crawler.Options{MaxDepth: depth}, s.limiter)
    pages := cr.Crawl(ctx, targetURL, seedLinks)
    for _, p := range pages {
        results = append(results, &model.TaskResult{
            TaskID: task.ID, Status: model.TaskStatusSuccess,
            ExecutedAt: startTime, CompletedAt: time.Now(),
            Result: &model.WebResult{
                URL: p.URL, Depth: p.Depth, StatusCode: p.StatusCode,
                Title: p.Title, ResponseHeaders: p.Headers,
                Forms: p.Forms, Params: p.Params, Leaks: p.Leaks,
                IP: finalIP, Port: finalPort,
            },
        })
    }
}
```

（伪代码，实际实现需要注意 `results` 变量在函数末尾已经是 `return []*model.TaskResult{result}, nil` 的单元素写法，需要改成 `append` 模式——这是本次改动里对现有函数**唯一**的结构性调整，且是纯增量、无副作用的调整。）

### 8.2 `internal/core/options/scan_web.go` 和 `cmd/agent/scan/web.go`

新增两个 CLI flag：`--crawl`（bool，默认 false）、`--crawl-depth`（int，默认 2），走 `task.Params["crawl"] / ["crawl_depth"]` 透传，与现有 `--screenshot` 参数模式完全一致，零学习成本。

### 8.3 `internal/core/pipeline/dispatcher.go`

`runWebScan` 中构造 Task 的地方（约 274 行）追加两行：

```go
task.Params["crawl"] = d.opts.WebCrawl        // 新增 ScanRunOptions 字段
task.Params["crawl_depth"] = d.opts.WebCrawlDepth
```

`ServiceDispatcher` 内部逻辑完全不动，因为它本来就是"构造 Task -> 调用 Run -> 收集结果 -> AddWebResult"，爬虫产出的多条 `WebResult` 走的是一模一样的路径。

### 8.4 `go.mod`

新增一行依赖：

```
github.com/PuerkitoBio/goquery v1.9.x
```

---

## 九、实施顺序（Sprint 拆分）

和 v1.0 一样按 Sprint 走，但每个 Sprint 的验收标准更具体、更小：

**Sprint 1（骨架，约 1-2 天）**
- `crawler.go`：BFS 主循环 + `visited` 去重 + `SameHostOnly` Scope 判断 + `MaxPages` 硬上限。
- 单元测试：起一个 `httptest.Server` 模拟 3 层链接的站点，断言爬取到的 URL 集合与去重正确性。
- 里程碑：给一个种子 URL + 种子链接，能正确 BFS 出全部页面 URL，深度和数量符合预期。

**Sprint 2（攻击面提取，约 1 天）**
- `extract.go`：goquery 提取链接/表单/参数。
- `context.go` 新增 `ExtractLinks(page)`，从 go-rod 页面提取初始种子链接。
- 里程碑：`WebResult` 能正确带出 `Forms`/`Params` 字段。

**Sprint 3（被动泄露检测，约 0.5 天）**
- `leak.go`：4-5 条内置正则规则 + 命中脱敏。
- 里程碑：对含测试用 AK/SK 字符串的页面能正确识别并脱敏输出。

**Sprint 4（接入与联调，约 1 天）**
- `web_scanner.go` / `scan_web.go` / `dispatcher.go` 三处接入点改造。
- 端到端跑 `scan web --crawl --crawl-depth=2` 和 `scan run` 全流程验证输出一致性。
- 里程碑：CLI、CSV、JSON 三种输出下 `Depth/Forms/Params/Leaks` 字段一致、无回归。

全部工作量预计 **4-5 天**，不需要引入任何外部中间件，不需要新的 TaskType，不需要新的 Scanner 接口实现。这是与 v1.0（隐含一个更大的"Frontier 系统"）相比更小、更可控、且完全兼容现有架构的路径。

---

## 十、与 v1.0 方案的关键分歧总结

| 维度 | v1.0 方案 | 本方案 (v2.0) |
|---|---|---|
| 队列结构 | `container/heap` 优先级队列 + 打分公式 | `chan` + BFS，深度即优先级，无需打分 |
| 去重归一化 | 归一化 + 参数模式聚合（相似度计算） | 仅做 Fragment 剥离 + Query 排序，不做语义聚合 |
| 组织形式 | Crawler 作为 web 包下独立子系统（4 个文件） | Crawler 是 WebScanner.Run() 内部的可选阶段（3 个文件） |
| 任务类型 | 隐含仍是 `TaskTypeWebScan`，但架构上是平行系统 | 明确复用 `TaskTypeWebScan`，Dispatcher 零改动 |
| DenialReason | 独立类型系统 + 常量枚举 | 一条 `logger.Debugf` 日志，无需类型 |
| 限流 | "接入现有 AdaptiveLimiter"（提及但未强调复用同一实例） | 强制共享 WebScanner 已持有的同一实例，语义统一 |
| 结果承载 | 提及"塞入 WebResult"但未定义具体字段 | 明确定义 `Depth/Forms/Params/Leaks` 四个新增字段，向后兼容 |

两版本在"方向"上是一致的（HTTP 优先、浏览器仅探路、attack surface 提取、被动泄露检测），这部分判断都是对的，说明这确实是正确的技术方向。分歧全部集中在"复杂度控制"上——v1.0 明显受两个参考项目的架构影响过深，把它们为分布式/多租户场景设计的复杂机制（优先级队列、相似度聚合、独立类型系统）也一并搬了过来。本方案的核心贡献是：**把"正确的方向"和"过度的复杂度"剥离开，只保留前者。**
