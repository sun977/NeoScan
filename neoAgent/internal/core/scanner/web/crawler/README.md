# crawler —— Web 攻击面爬虫

`crawler` 是 `WebScanner`（`internal/core/scanner/web`）内部按需触发的一个深度补充阶段，不是独立的 `TaskType`，也不是独立的 Scanner。

一句话说清它做什么：**从一个种子 URL 出发，用 BFS 在同源范围内走 N 层，走过的每一页都过一遍"提取链接 + 提取表单 + 正则找泄露"，把原始抓取结果交还给 `WebScanner`。**

设计依据：[`docs/爬虫/Web爬虫与被动分析器架构方案-sonnet5-v3.0.md`](../../../../../docs/爬虫/Web爬虫与被动分析器架构方案-sonnet5-v3.0.md)（后续简称"架构方案"）。任何行为差异都以该文档 + 本包源码注释为准。

## 为什么需要它

`WebScanner` 原本只探测首页，一个 IP:Port 下的真实技术栈从来不是单一的：`/` 可能是 Nginx 静态站，`/admin` 是一个 Vue 后台，`/api/swagger-ui` 是 Swagger 面板——这些只有真正访问到对应路径才能被发现。`crawler` 存在的意义就是把"单点快照"变成"顺藤摸瓜"，为后续漏洞扫描（Vuln Scanner）提供更多攻击面输入点（表单字段、URL 参数）。

## 它不做什么（职责边界）

- **不做指纹识别**。`Page` 结构体里没有、也不应该出现 `TechStack` 字段——本包不 import `fingerprint` 包。指纹识别的调用点在 `WebScanner`，它对每个 `Page` 各调一次 `fpEngine.Match`，把 `crawler` 的原始数据和指纹引擎的匹配结果粘合成最终的 `WebResult`（详见架构方案 8.6 节）。
- **不调用浏览器**。`Crawler` 全程只用 `net/http`。首页的 JS 渲染由 `WebScanner` 通过 go-rod 提前完成，渲染后拿到的链接作为 BFS 第一层种子（`Crawl` 的 `seedLinks` 参数）传进来；爬虫本身不重复请求首页、不内嵌浏览器实例。
- **不做分布式/断点续爬**。一次 Agent 任务运行几秒到几十秒，中断了重新发一次 Task 即可，不引入 Redis 或任何跨进程状态存储。
- **不做多进程/多子 Agent**。爬取是纯 IO 密集型任务，并发通过 Goroutine Worker Pool 解决，与 `AutoRunner.Run()` 已有的 Semaphore 模型是同一套范式（详见架构方案第五节）。

## 目录结构

三个文件，每个文件对应一个单一职责，没有为了凑对称性而拆分：

| 文件 | 职责 |
|---|---|
| `crawler.go` | BFS 主循环 + 去重 + Scope 判断 + 分层探测（JS 跳转 / SPA 空壳识别） |
| `extract.go` | 用 `goquery` 从 HTML 中提取攻击面信息：链接、表单、URL 参数 |
| `leak.go` | 对页面正文做被动敏感信息扫描（AK/SK、JWT、内网 IP） |

## 核心类型

```go
// Options 爬虫行为控制参数，零值必须是"安全"的默认配置
type Options struct {
    MaxDepth       int           // 默认 2
    MaxPages       int           // 硬上限，默认 200，防止爬虫失控
    Concurrency    int           // 默认 5
    Timeout        time.Duration // 单页超时，默认 10s
    AllowCrossHost bool          // 默认 false，只在种子 Host 内爬
}

// Page 爬虫抓取到的单个页面的原始数据，不含任何指纹相关字段
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
    NeedsEscalation  bool   // 是否建议用真实浏览器重新渲染
    EscalationReason string // 建议升级的原因：js_redirect / spa_shell
}

func New(opts Options, limiter *qos.AdaptiveLimiter) *Crawler
func (c *Crawler) Crawl(ctx context.Context, seedURL string, seedLinks []string) []*Page
func (c *Crawler) EnqueueExtra(links []string, atDepth int) // 供按需升级机制追加新发现的链接
```

`limiter` 必须由调用方（`WebScanner`）传入已经存在的 `qos.AdaptiveLimiter` 实例，不允许传 `nil`，也不会在内部新建——爬虫和首页扫描共享同一份限流状态，因为它们打的是同一个目标。

## 关键设计点

### 1. `AllowCrossHost` 而不是 `SameHostOnly`

字段语义故意设计成"是否允许跨域"而不是"是否限制同域"。原因：早期版本用 `SameHostOnly bool` 表达同源限制，但 `bool` 零值是 `false`，调用方一旦忘记显式传 `true`，同源限制就会悄悄失效——生产环境唯一调用点 `web_scanner.go` 就踩过这个坑，实测对 `baidu.com` 爬出了 `weibo.com`/`qq.com` 等一堆外部域名。反转字段语义后，零值 `Options{}` 本身就是安全默认值，不再依赖调用方记得传参。

### 2. 去重与队列：`map` + `channel`，不用 Redis/Heap

单次爬取的 URL 量级通常是几十到几百个，最多上千。这个规模下：
- 去重用一个受 `sync.Mutex` 保护的 `visited map[string]struct{}` 完全够用，不需要 Redis。
- 排序/调度用一个带缓冲的 `chan *item` 做广度优先遍历，不需要 `container/heap` 优先级队列。

数据结构的复杂度要匹配数据的真实规模，引入分布式组件在这个量级下收益约等于零，但会强迫 Agent 依赖外部依赖，违反《Agent重构开发总纲》里"独立、自包含"的原则。

### 3. 并发模型：Goroutine Worker Pool

`Crawl` 内部按 `Options.Concurrency`（默认 5）启动固定数量的 worker goroutine，共享同一个 `chan *item` 队列；`pending` 计数器归零时关闭队列，唤醒所有 worker 自然退出（`taskDone`，用 `sync.Once` 保证 `close(queue)` 只执行一次）。选择 Goroutine 而不是多进程/子 Agent 的原因见架构方案第五节：爬取是 IO 密集型任务，Go runtime 的调度器天然适合这类场景，多开进程唯一的效果是多消耗内存，不会带来任何吞吐提升。

### 4. 分层探测：三层信号判断是否需要浏览器升级渲染

`net/http` 抓到的 `body` 有两种"看起来抓到了、实际没内容"的情况：
- **JS 跳转中间页**（`isJSRedirect`）：页面很小（< 1KB）且包含 `location.href`/`window.location` 赋值。
- **SPA 空壳**（`isSPAShell`）：命中 `id="root"`/`id="app"` 挂载点，且去掉 `script`/`style` 标签后可见文本少于 200 字符。

命中任意一种，`Page.NeedsEscalation` 置为 `true`，`EscalationReason` 记录原因，但 `crawler` **只负责产出这个信号，不擅自决策**——是否真的调用浏览器重新渲染，由 `WebScanner` 决定（详见架构方案 8.8 节）。HTTP 3xx 跳转由 `http.Client` 默认重定向行为处理，不在这三层判断范围内。

### 5. `fetchAndExtract` 的"成功"定义

抓取返回的 `bool` 表示"这次抓取是否成功"：
- 网络层面失败（连不上、超时、读 body 出错）才算失败，会告知限流器 `OnFailure()`。
- 只要拿到了 HTTP 响应，哪怕是 404/500，也算成功——这本身是一次有效的网络探测结果，不是故障。

响应体读取上限为 2MB（`io.LimitReader`），与 `web_scanner.go` 的 `fallbackScan` 保持一致，避免误爬到大文件（如视频直链）把内存打爆。

## 与 `WebScanner` 的接入方式

```
WebScanner.Run()
    │  1. go-rod 渲染首页，拿到 richCtx 和首页链接（seedLinks）
    │  2. 首页 WebResult 照常产出（不变）
    │  3. 按 task.Params["crawl"] 决定是否触发深度爬取
    ▼
crawler.New(opts, limiter).Crawl(ctx, seedURL, seedLinks)
    │  BFS 遍历，每访问一个页面产出一个 *crawler.Page
    ▼
WebScanner 对每个 Page 循环调用 s.fpEngine.Match（复用首页那一套指纹识别代码）
    │  Page + 指纹匹配结果 → 组装成一条独立的 *model.WebResult（Depth > 0）
    ▼
pCtx.AddWebResult(...)（逐条累加，Pipeline/Dispatcher 零改动）
```

`crawler` 产出多少个页面，`WebScanner` 就向 Pipeline 报多少条独立的 `WebResult`——不做聚合，不新增 `TaskType`，下游消费方式和首页扫描完全一致。

## 测试

- `crawler_test.go`：BFS 遍历、深度限制、去重、Scope 判断
- `extract_test.go`：链接/表单/参数提取的各类边界场景
- `leak_test.go`：四类内置泄露规则的命中与脱敏
- `escalation_test.go`：JS 跳转 / SPA 空壳识别的正负样本（含"正常大型 React 应用不应被误判"的用例）
