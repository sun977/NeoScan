# NeoScan Web 爬虫与被动分析器 (Phase 5.1) 架构设计方案

基于对 `BrowsertrixCrawler` 和 `Firecrawl` 这两个业界顶级爬虫项目的源码分析与架构反思，结合 NeoScan 作为**安全扫描 Agent** 的"高内聚、重侦察、轻量级"特性，为您输出 Phase 5.1（Web Crawler & Passive Analyzer）的最佳实现路径。

---

## 一、 核心设计理念 (The Philosophy)

在安全扫描场景下，我们的 Crawler 不是为了做"搜索引擎收录"（如 Firecrawl 的全站转 Markdown），也不是为了做"完美网页存档"（如 Browsertrix）。我们的唯一目标是：**尽可能多地发现攻击面（输入点、接口、路径）和敏感信息，同时保持极低的资源消耗。**

因此，NeoScan Web Crawler 的核心原则是：
1. **优先便宜策略 (Direct Fetch First)**：绝不用 Headless Browser（`go-rod`）去遍历全站！浏览器仅用于首页指纹/SPA探路，深度遍历一律使用原生的 `net/http` 并发请求。
2. **拒绝 if/else 调度 (Score-based Queue)**：使用统一公式对 URL 打分，用优先级队列管理并发，天然实现广度优先和重试退避。
3. **精准的防冗余 (Strict Canonicalization)**：安全扫描最怕在伪静态页面（如 `?id=1` 到 `?id=1000`）中陷入死循环，去重和归一化是重中之重。
4. **所见即所得的拒绝记录 (Denial Reasons)**：被过滤的 URL 必须有明确的原因日志，方便排查"为什么没扫出那个接口"。

---

## 二、 架构模块划分与最佳落地路径

建议在 `internal/core/scanner/web/` 下新建 `crawler` 子包，按以下四大模块进行开发：

### 1. 调度与去重中心 (The Frontier)
**借鉴来源**：Browsertrix 的 ZSET 评分机制 & Firecrawl 的严谨去重。
*   **归一化引擎 (Canonicalizer)**：
    *   **实现**：去除 URL 的 Hash（`#xxx`，除非是 Vue 路由）、统一结尾 `/`、剥离无意义的追踪参数（如 `utm_source`）。
    *   **伪静态折叠**：引入相似度计算或参数模式聚合（Pattern Aggregation），将 `/article/1` 和 `/article/2` 识别为同一类页面，超过阈值即停止抓取该模式。
*   **打分公式 (Score Function)**：
    *   放弃复杂的 if-else 层级，用公式确定队列优先级：`Score = Depth * 10 + RetryCount * 100 - PriorityBoost`。值越小越先抓取。
*   **内存优先级队列 (Priority Queue)**：
    *   **实现**：Go 标准库 `container/heap` + `sync.RWMutex` 实现的并发安全队列。入队前原子校验 `visited` Map（存储归一化后的哈希）。

### 2. 多策略抓取仲裁器 (Fetcher Engine)
**借鉴来源**：Firecrawl 的 Waterfall Race 与 Browsertrix 的 Direct Fetch。
*   **智能降级策略**：
    *   **Phase 1 (首页/探路)**：复用现有的 `go-rod` 逻辑，获取 JS 动态渲染生成的接口（API Endpoint）和页面内跳转链接。
    *   **Phase 2 (深层爬取)**：Crawler 收到链接后，**只使用 `net/http` 发起请求**。
*   **资源类型拦截 (Fast Abort)**：
    *   使用 `HEAD` 请求或在读取 `GET` 响应的 `Content-Type` 时，若发现是 `.pdf`, `.mp4`, `.zip`，立刻截断 Body 读取，将该 URL 记录为"敏感文件资产"（Passive Asset），但不做 HTML 解析。
*   **自适应限流 (Adaptive Rate Limiting)**：
    *   复用 NeoScan 已有的 `qos.AdaptiveLimiter`。一旦检测到 HTTP 403 / 429 状态码，自动增加该域名的请求延迟。

### 3. DOM 提取与被动分析管道 (Extraction & Analyzer Pipeline)
**借鉴来源**：Firecrawl 的 Transformer 机制（纯函数式数据流）。
将每个下载好的 HTML 页面送入一条线性的处理管道（推荐使用 `PuerkitoBio/goquery`，性能极佳）：
*   **Link Extractor**：提取 `<a href>`、`<iframe src>`，并转换成绝对路径送回 Frontier。
*   **Form/Param Extractor (安全专属)**：
    *   提取 `<form>` 标签及其 `<input>` 属性（Action URL, Method, Params）。
    *   提取 URL 中的 Query 参数。
    *   产出：结构化的 `AttackSurface`（攻击面模型），为下一阶段的 Vuln Scanner (如 XSS/SQLi Fuzzing) 提供现成靶标。
*   **Passive Leak Analyzer (被动泄露检测)**：
    *   正则引擎扫描 Response Body & Headers。
    *   匹配规则：AWS/Aliyun AK/SK, JWT Tokens, 内部 IP 泄露, 身份证号等。
    *   这部分完全被动，不发任何包，是对全站源码的"白嫖式"审计。

### 4. 边界判定与拒绝追踪 (Scope & Denial Reason)
**借鉴来源**：Firecrawl 的 Rust 边界规则与 Browsertrix 的 Scope 隔离。
*   定义 `IsInScope(url string)` 函数，限定只能爬取同域名（或指定的子域名）。
*   **拒绝溯源**：
    ```go
    type DenialReason string
    const (
        ReasonDepthLimit   DenialReason = "超出最大深度(Depth>2)"
        ReasonOutOfScope   DenialReason = "跨域链接"
        ReasonStaticAsset  DenialReason = "静态资源后缀(.jpg/.css)"
    )
    ```
    丢弃的 URL 统一记录日志 `[Skip] url: xxx, reason: 超出最大深度`，终结安全测试中"为啥这页面没扫到"的玄学问题。

---

## 三、 代码集成架构图 (Go 伪结构)

建议在 `internal/core/scanner/web` 目录下进行如下扩充：

```text
internal/core/scanner/web/
├── crawler/
│   ├── crawler.go         # 爬虫主循环调度 (Worker Pool + WaitGroup)
│   ├── queue.go           # 基于 Heap 的优先级队列 & URL归一化去重
│   ├── extractor.go       # 基于 goquery 的链接/表单提取
│   ├── analyzer.go        # 正则表达式被动敏感信息匹配
│   └── filter.go          # Scope边界判断与 DenialReason 记录
├── web_scanner.go         # 现有的 WebScanner，作为总入口
└── context.go             # 现有的浏览器富上下文提取
```

**工作流接入点**：
修改现有的 `web_scanner.go` 中的 `Run` 方法：
1. 先跑完现有的 `go-rod` 获取首页信息。
2. 从 `richCtx` 获取首页的初始 Links，喂给 `crawler.Queue`。
3. 启动 `crawler.Start(ctx, maxDepth=2, threads=5)`。
4. Crawler 跑完后，汇总收集到的所有 URLs、Forms、SensitiveLeaks。
5. 将这些信息塞入 `model.WebResult` 中返回给 `ServiceDispatcher`。

---

## 四、 实施优先级 (Action Items)

建议按以下三个迭代进行开发（Stop designing. Start coding）：

1. **Sprint 1: 骨架与基建 (The Skeleton)**
   - 实现 URL 归一化逻辑（极其重要）。
   - 实现基于 `net/http` 的轻量级网页下载器 + `goquery` 解析提取所有的 `<a>` 标签。
   - 实现带 `Depth` 限制的内存并发队列（`WaitGroup` + `Channel`）。
   - *里程碑：给一个入口 URL，能迅速把整站的 URL 拓扑爬出来。*

2. **Sprint 2: 攻击面与敏感提取 (Security Focus)**
   - 在解析阶段加入 `Form` 表单解析逻辑。
   - 引入正则匹配库，对下载的 Body 进行 AK/SK 和 Token 的匹配。
   - 完善 `DenialReason` 审计日志。
   - *里程碑：能够输出网站的结构化攻击面和潜在泄露信息。*

3. **Sprint 3: 系统防雷与集成 (Hardening)**
   - 接入现有的 `qos.AdaptiveLimiter`。
   - 实现对下载文件大小的限制（如 `io.LimitReader` 最大 2MB，防止被大文件打挂内存）。
   - 接入 `PipelineDispatcher`，将产出对接给下一阶段的漏洞扫描器。

通过这种“主路浏览器 + 支路原生HTTP”的混合架构，NeoScan 可以在保持**扫描速度极快**的同时，获取到**极具深度的攻击面信息**。