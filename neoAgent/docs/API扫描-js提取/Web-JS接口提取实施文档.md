# Web 扫描 JS 接口提取实施文档

> 文档版本：v2.0（架构调整：挂载点从 `WebScanner.buildWebResult` 改为独立 `ApiScanner.Run`）
> 修改时间：2026-08-07
> 依据方案：[`Web-JS接口提取方案.md`](./Web-JS接口提取方案.md)（v2.0，本文档只负责把方案拆成可以直接照做的步骤，不重复方案的背景论证，结论有分歧以方案文档为准）、[`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md)（规则设计的调研依据）、[`../爬虫/web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md)（**前置依赖**：`crawler` 包迁移到 `internal/core/lib/crawler`、`crawler.FetchAndCrawl` 统一抓取入口、`ApiScanner` 骨架必须先落地，本文档才能按步骤执行）
> 文档性质：实施清单，后续 JS 接口提取功能的代码开发按本文档的步骤顺序进行；每步给出改动文件、改动内容、验收标准
> v1.2 变更：新增第八节，CLI 表格输出补充 `APIs` 列，并对单页 JS 文件数超过上限导致的结果截断做显式提示
> v1.3 变更：`--js-api` 改为独立于 `--crawl` 的全局开关，挂载点从 `crawler.Options`/`fetchAndExtract` 改为 `WebScanner.buildWebResult`。以上两版的挂载点结论已被 v2.0 取代，仅作历史记录。
>
> **v2.0 变更（架构重大调整，取代 v1.3 的挂载点结论）**：`ApiScan` 不再是 `web_scan` 的一部分，而是独立的 `model.TaskTypeApiScan`/`ApiScanner`，完整架构论证见 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md)。对本文档的核心影响：
> 1. **挂载点从 `WebScanner.buildWebResult` 改为 `ApiScanner.Run`**（新增的独立原子扫描器）——v1.3 解决的是"在 `web_scan` 内部挂在哪个函数"，v2.0 进一步发现连"要不要借住 `web_scan`"这个前提本身都不成立了；
> 2. `crawler` 包整体从 `internal/core/scanner/web/crawler` 迁移到 `internal/core/lib/crawler`，新增 `crawler.FetchAndCrawl` 作为 `WebScanner`/`ApiScanner` 共享的统一抓取入口（详见重构文档第三节），本文档第五节“集成点”全面重写；
> 3. `WebScanner` 不再需要任何为 JS API 提取而做的改动（不需要 `resolveJSAPIExtract`/`buildWebResult` 新增 `ctx` 参数/`pageData` 新增字段），这些改动全部转移到新增的 `ApiScanner`；
> 4. `WebResult.APIs`/`APIsTruncated` 字段设计作废，改为独立的 `model.ApiResult` 类型（第三节重写）；
> 5. CLI 层面新增独立的 `cmd/agent/scan/api.go` 子命令（而不是在 `web.go` 上加 `--js-api` flag），第二、七节全面重写。
> 三层提取规则（第四节）、不落盘（第四节 4.7）、提取与过滤解耦（第四节 4.3）等**提取能力本身的实现**不受影响，v2.0 只改变"谁来调用这些实现、这些实现位于哪个包路径下"。

---

## 零、实施前必读的三条硬约束

1. **下载外链 JS 文件是主动网络行为，必须有风险提示，但不再需要"要不要做"的开关**。v1.3 及之前版本要求"必须用户显式开启才会执行"，是因为当时 JS 提取挂在 `web_scan` 下，需要一个开关区分用户的意图；v2.0 里 `api_scan` 本身就是独立的 `TaskType`，用户下发这个任务本身就是显式意图，不再需要二态开关（详见第二节）。但风险提示不能省略——下载外链 JS 文件、逐个发起额外的 `net/http` 请求，这本身就是比"分析已经到手的首页 HTML"更进一步的行为（会显著增加对目标发起的请求数量），必须在 CLI 帮助文本和运行时日志里都有清晰提示。
2. **复用已导出的基础设施，不写第二套实现**：并发限流复用 `qos.AdaptiveLimiter`（`internal/core/lib/network/qos/limiter.go`），`ApiScanner` 持有自己独立的限流器实例（详见重构文档 7.2 节，不与 `WebScanner` 共享），新增的 JS 下载逻辑必须复用这个已注入的 `limiter` 字段，不允许再新建第二个限流器实例。
3. **JS 文件内容全程只在内存里流转，不落盘**，理由和具体做法见第四节 4.7 小节，这是本次实施在"设计方案没有明确到字节级细节"的地方需要遵循的具体决定，不是可以自由发挥的空间。

---

## 一、实施步骤总览

> **前置前提**：下表假设 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md) 描述的 `crawler` 包迁移与 `ApiScanner` 骨架（空实现，仅 Factory/Runner 注册/CLI/options 六件套打通）已经落地。本文档只负责在这个骨架之上填充 JS API 提取的具体逻辑，不重复重构文档已经覆盖的部分。

严格按顺序执行，后一步依赖前一步的产出：

| 步骤 | 改动文件 | 类型 | 状态 |
|---|---|---|---|
| 1 | `internal/core/model/result_types.go` | 修改（新增 `APIEndpoint` 类型 + 独立的 `ApiResult` 类型，**不再改 `WebResult`**） | ⬜ 待开始 |
| 2 | `internal/core/lib/crawler/jsapi.go` | 新建（提取规则 + 提取/过滤两段式实现 + 对外唯一入口 `ExtractPageAPIs`，独立包级函数，不挂在 `*Crawler` 上；注意路径已随重构文档迁移至 `internal/core/lib/crawler`） | ⬜ 待开始 |
| 3 | `internal/core/lib/crawler/jsapi_test.go` | 新建（表驱动单元测试） | ⬜ 待开始 |
| 4 | `internal/core/scanner/api/api_scanner.go` | 修改（在重构文档已搭好的空壳上，实现 `Run` 内部：调用 `crawler.FetchAndCrawl` 拿页面，对每个页面调用 `crawler.ExtractPageAPIs`，组装 `model.ApiResult` 返回） | ⬜ 待开始 |
| 5 | `internal/core/options/scan_api.go` | 修改（重构文档已新建空壳，本步补充具体字段：`ApiScanOptions` 不需要 `JSAPI bool` 开关——`ApiScanner` 本身就是这个能力，不需要再用一个开关控制要不要执行） | ⬜ 待开始 |
| 6 | `cmd/agent/scan/api.go` | 修改（重构文档已新建空壳，本步补充帮助文本风险提示） | ⬜ 待开始 |
| 7 | `docs/Agent指令集规范.md` | 修改（补充 `api_scan` 任务类型说明，带风险提示） | ⬜ 待开始 |
| 8 | `internal/core/scanner/api/api_scanner_test.go` | 新建（端到端集成测试，**必须覆盖首页单独命中的场景**） | ⬜ 待开始 |
| 9 | `internal/core/model/result_types.go` | 修改（`ApiResult.Headers()`/`Rows()` 实现 `TabularData` 接口，截断时追加星号） | ⬜ 待开始 |
| 10 | `cmd/agent/scan/api.go` | 修改（`PrintResults` 之后追加截断汇总提示行） | ⬜ 待开始 |

**不再需要开关参数链路**（这是 v2.0 相对 v1.3 最大的简化）：v1.3 需要 `resolveJSAPIExtract` 二态开关判断"要不要做"，是因为当时 JS API 提取挂在 `web_scan` 下，必须用一个开关区分"用户要做 Web 扫描还是要做 JS 提取"。现在 `api_scan` 自己就是一个独立的 `TaskType`，用户下发 `api_scan` 任务本身就是"要做 JS 提取"的显式意图表达，不需要再多一层开关判断。但风险提示仍然保留（下载外链 JS 文件依旧是会增加请求量的主动行为，只是提示位置从"参数帮助文本"改为"命令本身的帮助文本"）。步骤 5~7 不再是打通开关链路，而是打通 `ApiScanner` 自己的 CLI/options/指令集链路。步骤 9~10 是 CLI 展示环节——`ApiResult` 是全新类型，需要从零实现 `TabularData` 接口，不是在 `WebResult` 现有列上追加。

---

## 二、参数设计：`ApiScanner` 自己的 `crawl`/`crawl_depth`/风险提示，不再需要"要不要做"的开关

> **v2.0 整节重写**：本节原内容（v1.2/v1.3）讨论的是"`js_api` 该不该做成独立开关、该不该和 `crawl` 合并"，这个问题的前提是 JS 提取挂在 `web_scan` 下，需要在同一个任务里用参数区分意图。`api_scan` 独立成 `TaskType` 后，"要不要做 JS 提取"这个问题已经由用户下发 `api_scan` 任务本身回答了，不存在的开关自然不需要再设计。本节改为讨论 `ApiScanner` 自己真正需要哪些参数。

### 2.1 `ApiScanner` 仍然需要 `crawl`/`crawl_depth`，语义与 `WebScanner` 一致

`ApiScanner` 复用 `crawler.FetchAndCrawl`（详见重构文档第三节），同样面临"只提取首页，还是连带深度爬取的子页面一起提取"的问题，这与 `WebScanner` 的 `crawl`/`crawl_depth` 是完全相同的语义，直接照搬同一套参数设计（三态：显式开启/显式关闭/未指定自动判断），不重新发明：

```go
// resolveCrawlDepth 是 WebScanner 已有实现（web_scanner.go）的同名逻辑，
// ApiScanner 直接复用同一套三态参数语义，不重新设计一套新的判断规则。
// 唯一区别：ApiScanner 的“自动判断”默认值可以更保守（如默认不开深度爬取），
// 具体默认值在实施阶段结合 ApiScanner 的资源消耗特征确定。
```

### 2.2 不再需要 `js_api` 这样的"要不要提取"二态开关

v1.3 版本 `resolveJSAPIExtract` 存在的意义，是在 `web_scan` 这一个任务里区分"用户是只想做 Web 扫描，还是也想顺带做 JS 提取"。现在 `api_scan` 本身就是一个独立的 `TaskType`，`RunnerManager` 只有在收到 `TaskTypeApiScan` 时才会调度到 `ApiScanner.Run`，用户下发这个任务这件事本身就是"要做 JS 提取"的完整意图表达，`ApiScanner.Run` 内部不需要、也不应该再判断一次"要不要做"——它的存在本身就是答案。这是"消除特殊情况"的具体体现：v1.3 需要一个 `if resolveJSAPIExtract(task) { ... }` 分支是因为提取能力寄居在别的 Scanner 里，必须用 if 判断隔离出"这次要不要做"这个特殊情况；现在 `ApiScanner.Run` 整个函数体本身就是"要做"这条分支，不需要再包一层判断。

### 2.3 风险提示：从"参数帮助文本"改为"命令帮助文本"，日志提示不变

下载外链 JS 文件依然是会增加请求量的主动网络行为，这一点不因为挂载点改变而改变，风险提示仍然是硬性要求，只是提示的位置变了：

**执行前（CLI 帮助文本）**：v1.3 是在 `--js-api` 这个 flag 的帮助文本里写风险提示，因为 `--js-api` 只是 `web_scan` 命令的一个可选参数。v2.0 里 `api_scan` 是一个独立命令（`cmd/agent/scan/api.go`），风险提示应该写在**命令本身**的 `Short`/`Long` 描述里，因为用户运行这个命令的默认行为（不是某个可选 flag 打开后）就会下载 JS 文件：

```go
var apiScanCmd = &cobra.Command{
	Use:   "api",
	Short: "API 接口扫描（从 JS 代码中提取接口调用地址）",
	Long: "对目标页面（及可选的深度爬取子页面）提取内联/外链 JS 中硬编码的接口调用地址。\n" +
		"警告：本命令会下载页面引用的外链 JS 文件，属于主动网络行为，会增加对目标的请求数量，请确认已获得授权后再执行。",
	...
}
```

**执行中（日志层面的提示，面向运行时观测扫描行为的人）**：`crawler.ExtractPageAPIs`（第四节）在真正触发 JS 下载前打一条 `logger.Warnf` 级别日志，这部分逻辑与挂载点无关，原样保留：

```go
if len(jsURLs) > 0 {
	logger.Warnf("[crawler] JS API 提取已启用，将对 %s 额外下载 %d 个 JS 文件（此为用户显式开启的主动行为，会增加对目标的请求数量）", host, len(jsURLs))
}
```

`ExtractPageAPIs` 函数本身不需要任何"总开关"判断——`ApiScanner.Run` 对每个页面调用它就是无条件调用，函数被调用到这件事本身就等价于"用户要做这件事"。

### 2.4 参数链路：`api_scan` 命令自己的 CLI Flag → `ApiScanOptions` → `Task.Params`

`internal/core/options/scan_api.go`（重构文档已建空壳）补充具体字段——注意**不包含** `JSAPI bool`：

```go
type ApiScanOptions struct {
	Target     string
	Ports      string
	Crawl      string // "auto"(默认) / "true" / "false"，语义与 WebScanOptions.Crawl 一致
	CrawlDepth int
	MaxFiles   int // 单页最多下载的外链 JS 文件数，<=0 使用 crawler.JSAPIDefaultMaxFiles 兜底
	Output     OutputOptions
}
```

`cmd/agent/scan/api.go` 新增 Flag（不需要 `--js-api`，因为这个命令本身就是做这件事的）：

```go
flags.StringVar(&opts.Crawl, "crawl", "auto", "是否深度爬取子页面（auto/true/false）")
flags.IntVar(&opts.CrawlDepth, "crawl-depth", 2, "深度爬取的层数（仅 crawl=true 时生效）")
flags.IntVar(&opts.MaxFiles, "max-files", 0, "单页最多下载的外链 JS 文件数（<=0 使用默认值）")
```

`docs/Agent指令集规范.md` 补充 `api_scan` 任务类型的独立章节（而不是在 `web_scan` 参数表里追加一行），风险提示放在任务类型描述的开头，不允许精简掉：

```
### api_scan（API 扫描）

**警告：本任务会下载目标页面引用的外链 JS 文件，属于主动网络行为，会增加对目标的请求数量，请确认已获得授权后再下发。**

| 参数 | CLI Flag | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `crawl` | `--crawl` | string | No | `auto` | 是否深度爬取子页面 |
| `crawl_depth` | `--crawl-depth` | int | No | `2` | 深度爬取层数 |
| `max_files` | `--max-files` | int | No | `20` | 单页最多下载的外链 JS 文件数 |
```

---

## 三、数据结构改动

### 3.1 `internal/core/model/result_types.go`

紧邻 `LeakInfo`（第 137~142 行）之后新增，风格与现有类型完全一致：

```go
// APIEndpoint 从 JS 代码中提取的接口调用地址。语义上不同于页面导航链接（links）——
// 它描述的是"前端代码运行时会向这个地址发起数据请求"，而不是"这是一个可以被
// BFS 继续爬取的页面"，因此不会、也不应该被塞进 crawler 的 BFS 队列
// （见 Web-JS接口提取方案.md 第二节）。
type APIEndpoint struct {
	URL        string `json:"url"`                // 接口地址：可能是完整 URL，也可能是相对路径（如 /api/v1/user）。
	                                               // 不在提取阶段强制拼接成绝对地址，理由见方案文档 3 节字段注释。
	Method     string `json:"method,omitempty"`    // 有明确证据才填（如识别到 axios.post(...) 才填 "POST"）；
	                                               // 识别不到就留空，不猜默认值。
	Source     string `json:"source"`              // 来源：inline（内联 <script>）或具体的 js 文件 URL，便于定位复核。
	Confidence string `json:"confidence"`          // high：命中结构化调用（fetch/axios/ajax）；
	                                               // medium：命中 scope 锚定规则（同域名完整 URL）；
	                                               // low：命中泛化路径正则兜底。
}
```

不再向 `WebResult` 追加任何字段。**新增一个独立的 `ApiResult` 类型**，作为 `ApiScanner` 的 `model.TaskResult.Result`（与方案文档 v2.0 第三节保持一致）：

```go
// ApiResult 是 ApiScanner 的任务产出。与 WebResult 完全独立，不复用它的任何字段。
type ApiResult struct {
	URL           string        `json:"url"`                      // 本页面 URL
	Depth         int           `json:"depth"`                    // BFS 深度，0 表示首页
	APIs          []APIEndpoint `json:"apis,omitempty"`           // 从 JS 代码中静态提取到的接口调用地址清单，
	                                                                // ApiScanner 对每个页面都无条件调用提取，
	                                                                // 不存在"未开启时保持空"这种开关语义（参见第二节）。
	APIsTruncated bool          `json:"apis_truncated,omitempty"` // true 表示本页引用的外链 JS 文件数超过了
	                                                                // MaxJSFilesPerPage 上限，被截断丢弃了一部分文件
	                                                                // 未下载/未分析——也就是说 APIs 不是本页可能存在的
	                                                                // 完整清单，只是"上限范围内"的产出。
}
```

### 3.2 `crawler.Page` **不新增任何字段**

`crawler.Page` 只是 BFS 爬取子页面的原始数据载体（`URL`/`Body`/`Headers` 等），JS API 提取的判断和调用点已经全部收拢到 `ApiScanner.Run`（见第五节），`Page` 只需要把已经有的 `Body` 字段原样交给 `ApiScanner`，提取结果直接组装进 `model.ApiResult`，不需要在 `Page` 这一层再中转一次。这样 `crawler` 包的数据结构完全不为这个功能新增负担，`WebScanner`/`ApiScanner` 共用同一个 `Page` 定义。

### 3.3 验收标准

- `go build ./...` 通过；
- `go vet ./...` 无警告；
- 新增字段全部带 `omitempty`（除了 `Source`/`Confidence`/`URL` 这几个"只要有这条记录就必然有值"的必填字段）；
- `WebResult` 结构体本次完全不改动（用 `git diff` 确认 `result_types.go` 里 `type WebResult struct` 这一块没有任何改动，这是 v2.0 相对 v1.x 新增的验收项，直接验证"两个结果类型彻底独立"这个核心设计目标）；
- `APIsTruncated` 只有在真实发生截断时才为 `true`：单页 JS 文件数 ≤ `MaxJSFilesPerPage` 时必须是 `false`（哪怕 `APIs` 是空的，也不能因为"没提取到东西"就误标为截断，两者是完全独立的判断维度）；
- `crawler.Page` 结构体字段数量与实施前完全一致（用 `git diff` 确认 `crawler.go` 里 `type Page struct` 这一块没有任何改动）。

---

## 四、`jsapi.go` 新建：提取规则、下载与两段式实现，对外只暴露一个函数

### 4.0 本节的结构说明（v2.0 路径已随重构文档迁移）

提取规则、过滤、下载都在同一个 `jsapi.go` 文件里——这个结构自 v1.3 确定后未变：`downloadJS`/`downloadJSFiles` 是包级函数，不是 `*Crawler` 的方法，不需要、也不应该依赖一个 `*Crawler` 实例（`ApiScanner` 调用它时根本没有 `*Crawler`）。**唯一变化是文件路径**：随着 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md) 描述的整体迁移，`jsapi.go` 从 `internal/core/scanner/web/crawler/` 移到 `internal/core/lib/crawler/`，包名 `crawler` 不变，代码内容不受影响。

### 4.1 文件头部说明注释（风格对齐 `leak.go`）

```go
package crawler

// ==============================================================================
// jsapi.go：JS 接口地址提取器（被动静态分析，非主动验证）
// ------------------------------------------------------------------------------
// 职责（只做这一件事）：
//   对页面已经拿到手的 JS 文本（内联 <script> + 已下载的外链 js 文件内容）做
//   正则扫描，抠出前端代码里硬编码/拼接的接口调用地址，产出 model.APIEndpoint
//   列表。不做 AST 解析、不做反混淆、不发起任何验证请求（详见方案文档
//   Web-JS接口提取方案.md 第四节的选型结论）。
//
// 为什么这里没有"要不要做"的开关，本文件的下载动作为什么依然需要风险提示：
//   leak.go 检测的是"已经到手的页面正文"里有没有敏感信息，不需要额外的网络
//   请求。本文件的输入除了内联 script（零成本，见 4.7 节），还包括外链 js 文件
//   的下载内容——下载动作本身就是新增的网络请求。v2.0 里 ApiScanner.Run
//   对抓到的每一个页面都无条件调用本文件对外暴露的唯一入口函数
//   ExtractPageAPIs，不需要再判断"要不要做"：用户下发 api_scan 任务这件事
//   本身就是显式意图（详见 Web-JS接口提取实施文档.md 第二节）。风险提示的
//   位置从"参数帮助文本"改为 api_scan 命令自身的帮助文本和运行时日志，见
//   实施文档第二节、第五节。
//
// 本文件不依赖 *Crawler：ExtractPageAPIs 是纯粹的包级函数，只依赖调用方显式
// 传入的 (ctx, pageURL, body, limiter, maxFiles) 这几个参数，不读取任何
// *Crawler 内部状态。这是刻意的设计——ApiScanner.Run 处理首页和深度爬取
// 子页面时都调用同一份提取逻辑，两条路径共用同一个函数，不存在"首页走一条
// 路径、子页面走另一条路径"导致遗漏的问题（历史上 v1.2 版本把提取逻辑绑在
// *Crawler 方法上导致首页调不到，是这个坑的来源，v2.0 里 ApiScanner 从设计
// 上就不存在这个问题，因为它本来就没有寄居在别的 Scanner 里）。
//
// 提取与过滤两段式解耦（借鉴 URLFinder 架构，见项目分析文档第二节）：
//   extractAPICandidates 只管正则命中，不做任何排除判断；
//   filterAPICandidates 统一做清洗、静态资源后缀排除、黑名单关键词过滤。
//   两者职责单一、互不影响，新增一条提取规则和新增一条排除规则是互不相关的
//   两件事，不需要在正则里堆否定环。
// ==============================================================================

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/pkg/logger"
)
```

### 4.2 三层规则定义（对应方案文档 4.2 节）

```go
// highConfidenceRules 命中明确的 HTTP 调用语句结构，自带 Method 信息。
// 不额外定义 apiRule{Pattern, Confidence} 这种包装结构体——三层规则的置信度
// 是固定的（high/medium/low 分别对应三个规则集合），置信度信息在
// extractAPICandidates 里直接以字面量写在对应的循环体内即可，没有必要为了
// "看起来更规范"而引入一个只有一种取值组合的结构体，这是需要消除的多余抽象层。
// 三条规则分别对应 fetch/axios/jQuery.ajax 三种最常见的前端请求写法，
// 选取依据见方案文档 4.2 节、及 JSFinder-URLFinder-dirsearch项目分析.md
// 第二节对 URLFinder jsFind/urlFind 的分析。
var highConfidenceRules = []*regexp.Regexp{
	regexp.MustCompile(`fetch\(['"]([^'"]+)['"]`),
	regexp.MustCompile(`axios\.(get|post|put|delete|patch)\(['"]([^'"]+)['"]`),
	regexp.MustCompile(`\.ajax\(\{[^}]*url:\s*['"]([^'"]+)['"]`),
}

// lowConfidenceRules 单纯匹配"看起来是接口路径"的字符串字面量，只认固定前缀，
// 不追求覆盖全部——过度泛化的正则见方案文档 5.4 节明确拒绝的理由。
var lowConfidenceRules = []*regexp.Regexp{
	regexp.MustCompile(`['"](/api/[a-zA-Z0-9_\-/{}]+)['"]`),
	regexp.MustCompile(`['"](/v[0-9]+/[a-zA-Z0-9_\-/{}]+)['"]`),
}

// staticResourceSuffixes 是低置信度层的排除清单：命中内容以这些后缀结尾的
// 一律视为静态资源，不是接口地址。清单来自方案文档 4.2 节的既定范围，
// 只覆盖最常见的几类，不追求穷举（穷举的维护成本和收益不成正比）。
var staticResourceSuffixes = []string{
	".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".woff", ".woff2", ".ttf", ".ico",
}
```

### 4.3 中置信度规则：scope 锚定（对应方案文档 4.2 节新增部分）

```go
// buildScopeAnchoredRule 按当前页面所在域名，动态构造一条"域名 + 合法路径
// 字符集"的正则，用于中置信度层的提取。这条规则不能是包级常量，因为它依赖
// 运行时才知道的 host（不同页面 host 不同），所以做成一个按需构造的函数，
// 由调用方（ExtractAPIsFromJS）每次调用时传入当前页面的 host 现场构造。
//
// 思路来自 dirsearch 的 text_crawl（见 JSFinder-URLFinder-dirsearch项目分析.md
// 第三节）：先用 regexp.QuoteMeta 精确转义当前域名，再匹配后面跟着的合法
// URL 路径字符。因为明确锚定了"必须是当前站点自己的域名"，天然排除第三方
// CDN/统计脚本域名的噪音，不需要额外维护一份"外部域名黑名单"。
func buildScopeAnchoredRule(host string) *regexp.Regexp {
	// 合法 URL 路径字符集参考 dirsearch text_crawl 的字符类定义，
	// 直接复用同一份字符集不重新设计，理由是这份字符集已经过实战检验。
	pattern := regexp.QuoteMeta(host) + `[a-zA-Z0-9\-._~!$&*+,;=:@?%/]*`
	return regexp.MustCompile(pattern)
}
```

### 4.4 提取函数（第一段：只管命中，不做过滤）

```go
// extractAPICandidates 对一段 JS/HTML 文本跑三层正则，返回命中的候选列表。
// 只负责"正则命中"这一件事，不做任何排除判断（排除逻辑在 filterAPICandidates
// 里，两者刻意分离，见文件头部说明）。
//
// host 用于构造中置信度的 scope 锚定规则；source 标注这段文本的来源
// （"inline" 或具体的 js 文件 URL），原样透传进每一条产出的 APIEndpoint。
func extractAPICandidates(text, host, source string) []model.APIEndpoint {
	var out []model.APIEndpoint

	// 高置信度：结构化调用，部分规则能拿到 Method
	for _, re := range highConfidenceRules {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			ep := model.APIEndpoint{Source: source, Confidence: "high"}
			switch len(m) {
			case 2:
				ep.URL = m[1]
			case 3:
				ep.Method = strings.ToUpper(m[1])
				ep.URL = m[2]
			default:
				continue
			}
			out = append(out, ep)
		}
	}

	// 中置信度：scope 锚定
	if host != "" {
		re := buildScopeAnchoredRule(host)
		for _, m := range re.FindAllString(text, -1) {
			out = append(out, model.APIEndpoint{URL: m, Source: source, Confidence: "medium"})
		}
	}

	// 低置信度：泛化路径兜底
	for _, re := range lowConfidenceRules {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			if len(m) < 2 {
				continue
			}
			out = append(out, model.APIEndpoint{URL: m[1], Source: source, Confidence: "low"})
		}
	}

	return out
}
```

### 4.5 过滤函数（第二段：清洗 + 排除）

```go
// filterAPICandidates 对 extractAPICandidates 的产出做清洗和降噪：
//  1. 去除首尾空白、反斜杠转义；
//  2. 排除命中静态资源后缀的项（图片/样式/脚本本身不是接口）；
//  3. 去重（同一 URL 出现多次只保留一条，去重逻辑与 crawler.go 现有的
//     visited 去重同一套思路：用 map 记录已见过的 URL）。
func filterAPICandidates(candidates []model.APIEndpoint) []model.APIEndpoint {
	seen := make(map[string]struct{}, len(candidates))
	out := make([]model.APIEndpoint, 0, len(candidates))

	for _, ep := range candidates {
		ep.URL = strings.TrimSpace(ep.URL)
		ep.URL = strings.ReplaceAll(ep.URL, `\/`, "/")
		if ep.URL == "" {
			continue
		}
		if hasStaticResourceSuffix(ep.URL) {
			continue
		}
		if _, dup := seen[ep.URL]; dup {
			continue
		}
		seen[ep.URL] = struct{}{}
		out = append(out, ep)
	}
	return out
}

func hasStaticResourceSuffix(u string) bool {
	// 忽略查询字符串再判断后缀，"/app.js?v=123" 也应该被排除
	if idx := strings.IndexAny(u, "?#"); idx >= 0 {
		u = u[:idx]
	}
	lower := strings.ToLower(u)
	for _, suf := range staticResourceSuffixes {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	return false
}
```

### 4.6 内部函数：整合提取结果（不导出）

```go
// extractAPIsFromJS 给一组 JS 文本片段（每段带各自的来源标注），返回提取
// 并过滤后的 APIEndpoint 列表。这是内部函数（小写开头），不对外导出——
// 本文件对外只暴露一个入口 ExtractPageAPIs（见 4.8 节），texts 的组装
// （内联 script + 外链 js 下载内容）也收在 ExtractPageAPIs 内部完成，
// 调用方不需要、也不应该知道 texts 这个中间数据结构的存在。
//
// texts 的 key 是来源标注（"inline" 或 js 文件 URL），value 是对应的文本内容。
// host 是当前页面所在域名（scheme://host，不含 path），用于构造中置信度的
// scope 锚定规则。
func extractAPIsFromJS(texts map[string]string, host string) []model.APIEndpoint {
	var all []model.APIEndpoint
	for source, text := range texts {
		all = append(all, extractAPICandidates(text, host, source)...)
	}
	return filterAPICandidates(all)
}
```

### 4.7 外链 JS 文件下载：内存流式处理，不落盘，包级函数不挂 `*Crawler`

**下载下来的 JS 文件内容全程只作为 Go 内存里的 `string` 变量存在，用完即弃，绝不写入磁盘临时文件。** 这不是"偷懒少做一步"，是经过对比后更优的方案，理由如下。

**理由一：现有架构本来就是这么做的，没有理由破例。** `fetchAndExtract` 现在处理首页 HTML 的方式就是纯内存字符串（`body := string(bodyBytes)`，见 `crawler.go` 第 322~326 行），`leak.go`/`extract.go` 都是直接对这个内存字符串跑正则/DOM 解析，从来没有落盘这一步。JS 文件和 HTML 页面本质上是同一类"抓来就为了跑一遍分析、分析完就不再需要"的临时文本数据，没有任何理由对 JS 文件单独引入一套"写文件"的处理路径。

**理由二：落盘会引入一整类新问题，而这些问题对应的收益是零。** 一旦落盘，就必须解决：临时文件命名与路径拼接（要防止恶意 JS 文件名或者 Content-Disposition 头造成路径穿越）、并发写入同名文件的竞争、扫描进程异常退出/被杀导致临时文件残留成为磁盘垃圾、多端口/多任务并发扫描时的目录隔离、临时文件的读写权限。这些问题不是"写起来麻烦"，而是每一个都可能变成新的安全隐患（NeoScan 本身是安全工具，如果自己因为写临时文件的路径处理不当被人反过来攻击，是不可接受的）。而这一切复杂度对应的收益是**零**——我们唯一需要对 JS 内容做的操作就是"跑几条正则"，正则可以直接对内存字符串操作，压根不需要文件系统参与。

**理由三：符合方案文档"实用主义"的既定原则。** 方案文档第 4.4 节引用 `leak.go` 的注释"宁可漏报，不做大而全但充满噪音的正则堆砌"，同样的实用主义原则用在这里：能用最简单的方式解决问题（内存字符串 + 正则），就不要为了假想的"以后可能需要保留原始文件"去引入文件系统依赖。如果未来真的出现"需要保留 JS 原始文件供人工复核"这种真实需求，那是一个新的、独立的功能（类似截图存证），应该有自己的存储路径设计和生命周期管理，不应该反过来污染这个纯粹做"提取"的模块。

**`downloadJS`/`downloadJSFiles` 是本文件内的包级函数，不是 `*Crawler` 的方法**——这是 v1.3 相对 v1.2 的关键差异：v1.2 设计成 `func (c *Crawler) downloadJS(...)`，依赖 `c.client`/`c.limiter`/`c.opts`，导致这套下载能力被绑死在"必须先有一个 `*Crawler` 实例"这个前提上，首页调用点没有 `Crawler`、自然用不了。v1.3 把 `http.Client`、`*qos.AdaptiveLimiter`、超时时间都改成显式参数传入，函数不再关心调用方是首页逻辑还是爬虫子页面逻辑。

#### 4.7.1 具体实现：下载即处理，不做中间态

```go
// downloadJS 下载单个外链 JS 文件的内容，返回内容字符串。这是 fetchAndExtract
// 现有 HTTP 下载逻辑的直接复用模式（不新写一套下载客户端），2MB 上限和现有
// c.client/io.LimitReader 的用法完全一致，全项目统一这一个"安全边界"数值。
//
// client/timeout 由调用方（ExtractPageAPIs）显式传入，本函数不持有、也不
// 创建任何 http.Client——这样首页和爬虫子页面调用时都可以传各自已有的
// client 实例（或干脆共用同一个默认 client），函数本身不关心调用方身份。
//
// 内容只作为函数返回值的 string 存在，函数调用结束、返回值被
// extractAPIsFromJS 消费完之后，这段内存会被 GC 正常回收，不存在任何
// 需要手动清理的中间状态（这正是"不落盘"带来的直接好处：没有临时文件，
// 也就没有"扫描中途 panic 导致垃圾文件没清理"这种问题需要考虑）。
func downloadJS(ctx context.Context, client *http.Client, jsURL string, timeout time.Duration) (string, bool) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, jsURL, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) NeoScan-Crawler/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	// 和 fetchAndExtract 现有的 2MB 上限保持一致，不为 JS 文件单独发明新的数值。
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", false
	}
	return string(bodyBytes), true
}
```

#### 4.7.2 并发下载：复用调用方传入的 `qos.AdaptiveLimiter`，不新建限流器

```go
// downloadJSFiles 并发下载一批外链 JS 文件，用调用方传入的 limiter 控制
// 并发数，不新建第二套限流机制（零、实施前必读约束 2）——limiter 由
// WebScanner.buildWebResult 传入，全局唯一同一个实例，无论首页还是爬虫
// 子页面调用 ExtractPageAPIs，最终都受同一个限流器约束，天然避免"首页
// 和子页面各自持有一个限流器、并发数失控"的问题。
//
// 单页最多下载 maxFiles 个文件，超出的直接丢弃不下载——这是方案文档
// "风险与验证要点"明确要求的上限，防止大型 SPA 打包出几十个 chunk 文件时
// 拖垮单页扫描耗时。maxFiles <= 0 时退化为包级默认值 JSAPIDefaultMaxFiles。
//
// 返回值新增 truncated：是否发生了截断。这个判断只应该在"决定丢弃哪些 URL"
// 的这一处代码里做一次，不能让调用方拿到 jsURLs 原始长度后自己再算一遍
// "是不是超过上限"——那样等于同一个判断逻辑写了两遍，上限常量以后一旦
// 调整，容易改了一处漏改另一处。谁做截断决策，谁负责把这个事实如实报告
// 出去，这是唯一数据源原则。
func downloadJSFiles(ctx context.Context, client *http.Client, limiter *qos.AdaptiveLimiter, jsURLs []string, maxFiles int, timeout time.Duration) (result map[string]string, truncated bool) {
	if maxFiles <= 0 {
		maxFiles = JSAPIDefaultMaxFiles
	}
	if len(jsURLs) > maxFiles {
		truncated = true
		jsURLs = jsURLs[:maxFiles]
	}

	results := make(map[string]string, len(jsURLs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, u := range jsURLs {
		u := u
		if err := limiter.Acquire(ctx); err != nil {
			break // 上下文取消/限流器关闭，停止继续下载
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer limiter.Release()

			content, ok := downloadJS(ctx, client, u, timeout)
			if !ok {
				limiter.OnFailure()
				return
			}
			limiter.OnSuccess()

			mu.Lock()
			results[u] = content
			mu.Unlock()
		}()
	}
	wg.Wait()
	return results, truncated
}
```

注意：`truncated` 只反映"引用的 JS 文件数是否超过上限"，与下载过程中个别文件请求失败（超时/404/连接错误）无关——下载失败是网络层面的正常损耗，`downloadJS` 已经用 `ok bool` 处理，不计入 `truncated` 语义，两者不要混在一起判断。

#### 4.7.3 包级常量与默认超时

```go
// JSAPIDefaultMaxFiles 单页最多下载的外链 JS 文件数量上限，无用户自定义参数时
// 的兜底默认值。取值理由见 Web-JS接口提取方案.md 第八节风险表：既要覆盖
// 大多数正常站点的 JS 文件数量，又要给大型 SPA 打包场景设一个硬上限。
const JSAPIDefaultMaxFiles = 20

// jsAPIDefaultTimeout 是 ExtractPageAPIs 调用方未显式指定超时时间的兜底值，
// 与页面抓取现有的 10s 默认超时保持一致，不为 JS 下载单独设计一套超时参数。
const jsAPIDefaultTimeout = 10 * time.Second
```

#### 4.7.4 小节验收标准（并入 4.9 节整体验收，此处不重复列出）

- 全程搜索改动涉及的文件，不出现 `os.Create`/`os.WriteFile`/`ioutil.TempFile` 等任何文件写入调用；
- `downloadJSFiles` 的单元测试用 `httptest.Server` 模拟多个 JS 文件端点，验证：并发下载数量不超过 `limiter` 当前额度、超过 `maxFiles` 的 URL 不会被请求、单个文件下载失败不影响其他文件的下载结果；
- 大文件（构造一个 >2MB 的响应体）验证被截断到 2MB 而不是内存无限增长；
- 新增 `truncated` 返回值的专项断言：传入 URL 数 ≤ `maxFiles` 时 `truncated` 必须是 `false`；传入 URL 数 > `maxFiles` 时 `truncated` 必须是 `true`，且实际发起的 HTTP 请求数等于 `maxFiles`（而不是全部传入的 URL 数）；单个文件下载失败（不影响截断判断）不应把 `truncated` 误置为 `true`；
- `downloadJS`/`downloadJSFiles` 均为包级函数（无接收者），`go vet` 确认它们不是任何类型的方法。

### 4.8 对外唯一入口：`ExtractPageAPIs`

本文件真正对外暴露、供 `web_scanner.go` 调用的函数只有这一个，整合了 4.6 节的规则提取和 5.3~5.4 节的下载能力：

```go
// ExtractPageAPIs 是本包对外暴露的唯一入口：给定一个页面的 URL 和已经拿到手
// 的 HTML body，提取内联 <script> 中的接口地址，并发下载 body 中引用的外链
// js 文件后一并提取，返回合并去重后的 APIEndpoint 列表，以及是否发生截断。
//
// 这是一个纯粹的包级函数，不依赖任何 *Crawler 实例——调用方是 ApiScanner.Run
// （见 Web-JS接口提取实施文档.md 第五节），对 crawler.FetchAndCrawl 拿到的
// 首页和每一个深度爬取子页面都调用同一个函数，不存在"首页调不到"的问题。
//
// 参数：
//   ctx：用于下载请求的超时/取消控制，由调用方传入（通常是 Run() 的顶层 ctx）；
//   pageURL：当前页面 URL，用于解析相对路径的 <script src> 和构造 scope 锚定规则；
//   body：页面 HTML 正文，已经在手，不产生额外请求；
//   client：下载外链 js 用的 http.Client，调用方传入（可传 nil，函数内部
//     退化为一个使用默认 Transport 的新实例，避免调用方必须专门维护一个
//     client 只为了调这一个函数）；
//   limiter：并发限流器，必须是调用方已持有的 *qos.AdaptiveLimiter 实例
//     （WebScanner.limiter），不允许传 nil，函数不会新建限流器；
//   maxFiles：单页最多下载的 js 文件数，<=0 时使用 JSAPIDefaultMaxFiles。
//
// 返回值：(提取到的 APIEndpoint 列表, 是否因文件数超过 maxFiles 发生截断)。
func ExtractPageAPIs(ctx context.Context, pageURL, body string, client *http.Client, limiter *qos.AdaptiveLimiter, maxFiles int) ([]model.APIEndpoint, bool) {
	host := schemeHost(pageURL) // scheme://host，不含 path，供 scope 锚定规则使用

	texts := map[string]string{}
	if inline := extractInlineScripts(body); inline != "" {
		texts["inline"] = inline
	}

	if client == nil {
		client = &http.Client{}
	}

	var truncated bool
	jsURLs := extractExternalScriptURLs(pageURL, body)
	if len(jsURLs) > 0 {
		logger.Warnf("[crawler] JS API 提取已启用，将对 %s 额外下载 %d 个 JS 文件（此为用户显式开启的主动行为，会增加对目标的请求数量）", host, len(jsURLs))
		var downloaded map[string]string
		downloaded, truncated = downloadJSFiles(ctx, client, limiter, jsURLs, maxFiles, jsAPIDefaultTimeout)
		if truncated {
			limit := maxFiles
			if limit <= 0 {
				limit = JSAPIDefaultMaxFiles
			}
			// 截断这件事必须在日志里单独喊出来，不能只是让调用方拿到一个
			// bool 值默默处理——运维/使用者排查"为什么这次扫描 API 数量
			// 比预期少"时，第一时间应该在日志里就能看到原因。
			logger.Warnf("[crawler] %s 引用的 JS 文件数超过上限，已丢弃 %d 个未下载，本页 apis 结果不完整", host, len(jsURLs)-limit)
		}
		for u, content := range downloaded {
			texts[u] = content
		}
	}

	if len(texts) == 0 {
		return nil, truncated
	}
	return extractAPIsFromJS(texts, host), truncated
}
```

`extractInlineScripts`/`extractExternalScriptURLs`/`schemeHost` 三个辅助函数复用 `extract.go` 现有的 `goquery.Document` 解析套路（`Find("script:not([src])").Text()` 和 `Find("script[src]")`，与方案文档 5.2/5.3 节描述一致），不重复贴代码，实现时直接照抄 `extract.go` 里 `ExtractLinksAndForms` 解析 `<script>` 标签的既有写法，放在 `jsapi.go` 内部即可（同样是包级函数，不需要 `*Crawler`）。

### 4.9 验收标准（含 4.7 节合并前 `jsapi_test.go` 原有的规则测试）

- `jsapi_test.go` 表驱动覆盖：
  - 高置信度三条规则各自的正例（`fetch("/api/x")`、`axios.post("/api/y")`、`.ajax({url: "/api/z"})`）；
  - 中置信度 scope 锚定规则命中同域名 URL、不命中第三方域名 URL；
  - 低置信度规则命中 `/api/...`、`/v1/...`，且被静态资源后缀排除（如 `"/api/a.js"` 不应出现在结果里，除非它本身也命中高/中置信度规则）；
  - 去重：同一 URL 在多段文本里重复出现，只保留第一条；
  - 空输入、纯噪音文本（不含任何 URL 特征）返回空切片而非 nil 导致 panic；
  - `ExtractPageAPIs` 端到端：用 `httptest.Server` 搭建内联 + 外链混合的测试页面，验证不传 `*Crawler` 也能正常工作（因为这个函数本来就不需要 `*Crawler`）。
- `go test ./internal/core/lib/crawler/... -run TestExtractPageAPIs -v` 全部通过；
- `go vet` 确认 `crawler.go` 文件本身没有任何改动（`git diff internal/core/lib/crawler/crawler.go` 应为空）。

---

## 五、集成点：`ApiScanner.Run` 接入提取，首页与子页面共用同一次调用

> **v2.0 整节重写**：本节原内容（v1.3）讨论的是"`WebScanner.buildWebResult` 内部怎么接入提取"，随着挂载点整体迁移到独立的 `ApiScanner`，这个函数、这个改法都不复存在。`WebScanner` 本次**完全不需要任何改动**——不需要新增 `resolveJSAPIExtract`、不需要 `buildWebResult` 新增 `ctx` 参数、不需要 `pageData` 新增字段，这些内容全部作废。下面重新论述 `ApiScanner.Run` 如何接入 `ExtractPageAPIs`。

### 5.1 前提：`ApiScanner` 骨架已经打通六件套

本节假设 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md) 描述的 `ApiScanner` 骨架（`internal/core/scanner/api/api_scanner.go`、`core/factory/api_factory.go`、`RunnerManager` 注册、`cmd/agent/scan/api.go`、`internal/core/options/scan_api.go`）已经落地，`ApiScanner.Run` 方法签名已经存在（重构文档 3.4 节）：

```go
type ApiScanner struct {
	limiter *qos.AdaptiveLimiter // ApiScanner 独立持有的限流器实例，不与 WebScanner 共享（重构文档 7.2 节）
}

func (s *ApiScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error)
```

本节要做的，是把这个空壳的 `Run` 方法体填充完整：调用 `crawler.FetchAndCrawl` 拿页面，对每个页面调用 `crawler.ExtractPageAPIs`，组装 `model.ApiResult` 返回。

### 5.2 `Run` 内部流程：一次抓取，逐页提取，无条件调用

```go
func (s *ApiScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	opts := parseApiScanParams(task) // 解析 crawl/crawl_depth/max_files，见 5.3 节

	home, subPages, err := crawler.FetchAndCrawl(ctx, task.Target, opts.port, opts.protocolHint, s.limiter,
		crawler.FetchOptions{
			CrawlDepth: opts.crawlDepth, // <=0 表示只抓首页，不做 BFS
			OnPageReady: nil,            // ApiScanner 不需要截图/favicon，回调传 nil（重构文档 3.2 节约束）
		})
	if err != nil {
		return nil, err
	}

	var results []*model.TaskResult
	results = append(results, s.buildApiResult(ctx, task, home.URL, 0, home.Body, opts.maxFiles))
	for _, p := range subPages {
		results = append(results, s.buildApiResult(ctx, task, p.URL, p.Depth, p.Body, opts.maxFiles))
	}
	return results, nil
}

// buildApiResult 是 ApiScanner 组装单个页面结果的唯一入口，首页（depth=0）和
// 每一个深度爬取子页面都调用这一个函数——不存在"首页走一条路径、子页面走
// 另一条路径"的分叉，这是 ApiScanner 相对 v1.3 时代"挂在别的 Scanner 里"最大
// 的简化：现在它自己就是任务入口，对自己抓到的每个页面一视同仁。
//
// 无条件调用 ExtractPageAPIs，不做任何"要不要提取"的判断——ApiScanner.Run
// 被调用到这件事本身就是用户的显式意图（详见第二节 2.2）。
func (s *ApiScanner) buildApiResult(ctx context.Context, task *model.Task, pageURL string, depth int, body string, maxFiles int) *model.TaskResult {
	apis, truncated := crawler.ExtractPageAPIs(ctx, pageURL, body, nil, s.limiter, maxFiles)
	return &model.TaskResult{
		TaskID: task.ID,
		Type:   model.TaskTypeApiScan,
		Result: &model.ApiResult{
			URL:           pageURL,
			Depth:         depth,
			APIs:          apis,
			APIsTruncated: truncated,
		},
	}
}
```

`limiter` 直接传 `s.limiter`——`ApiScanner` 结构体本来就持有这个独立字段（重构文档 7.2 节），无论首页还是子页面调用，用的都是同一个限流实例，与第四节 4.7 小节"复用调用方传入的 limiter，不新建限流器"的原则一致。`http.Client` 传 `nil`，`ExtractPageAPIs` 内部会用默认 client 兜底（见 4.8 节），不需要 `ApiScanner` 专门再持有一个下载用的 client。

### 5.3 参数解析：`crawl`/`crawl_depth`/`max_files` 从 `task.Params` 读取

```go
type apiScanParsedOpts struct {
	port         int
	protocolHint string
	crawlDepth   int
	maxFiles     int
}

// parseApiScanParams 从 task.Params 解析 ApiScanner 需要的参数。crawl/crawl_depth
// 的三态语义直接复用 WebScanner.resolveCrawlDepth 同一套判断规则（第二节 2.1），
// 不重新发明；max_files 对应 ApiScanOptions.MaxFiles，<=0 时在 ExtractPageAPIs
// 内部退化为 crawler.JSAPIDefaultMaxFiles（第四节 4.7.3 节已有兜底，这里不用重复判断）。
func parseApiScanParams(task *model.Task) apiScanParsedOpts {
	...
}
```

具体实现直接复用 `WebScanner` 现有 `resolveCrawlDepth` 的判断逻辑（这段逻辑本身不涉及"抓不抓页面"以外的业务耦合，属于可以安全复制到 `ApiScanner` 的通用参数解析代码），不在此重复贴代码。

### 5.4 若干易漏点核对

- **`ApiScanner.Run` 不需要判断"要不要提取"**：`RunnerManager` 只有收到 `TaskTypeApiScan` 才会调度到这个方法，调度到这件事本身就是判断结果，`buildApiResult` 内部对 `ExtractPageAPIs` 的调用是无条件的（详见第二节 2.2）；
- **`crawler.Page` 不需要任何改动**（详见第 3.2 节），子页面的 `p.Body`/`p.URL`/`p.Depth` 都是 `crawler.Page` 已有字段，`buildApiResult` 直接读取即可，不存在需要新增透传字段的情况；
- `ExtractPageAPIs` 本身不重复判断"总开关"（详见第四节 4.8 节的函数注释），避免同一件事在 `ApiScanner.Run` 和 `ExtractPageAPIs` 两处各判断一次；
- `WebScanner` 一行代码都不需要改——这是本节相对 v1.3 最重要的验收前提，用 `git diff` 确认 `web_scanner.go` 没有任何改动。

### 5.5 验收标准

- `go build ./...`、`go vet ./...` 通过；
- `go test ./internal/core/scanner/api/... -race` 通过，重点关注 `downloadJSFiles` 的并发写 map 场景（已用 `sync.Mutex` 保护，`-race` 应该跑不出竞争）；
- **首页单独命中场景**（核心验收项）：构造一个不触发深度爬取的目标（比如首页没有可爬链接，或显式传 `--crawl=false`），运行 `api_scan`，验证首页对应的 `ApiResult.APIs` 依然非空——这条断言专门验证"首页拿不到提取"这类历史 bug（v1.2 时代的教训）在新架构下不会重现，因为 `ApiScanner.Run` 对首页和子页面天然走同一个 `buildApiResult`；
- **深度爬取子页面命中场景**：传 `--crawl=true --crawl-depth=2` 运行 `api_scan`，验证深度为 1、2 的子页面 `ApiResult.APIs` 同样非空，不是只有首页（Depth=0）才有数据；
- 构造一个引用 JS 文件数超过 `MaxJSFilesPerPage` 的测试页面，验证：`ApiResult.APIsTruncated == true`，且日志里能看到明确的"已丢弃 N 个未下载"提示（不是只在返回值里体现，日志必须同步出现）；
- `git diff internal/core/scanner/web/web_scanner.go` 应为空——`WebScanner` 本次不应该有任何改动，如果这个文件有改动，说明实现偏离了 v2.0 的架构设计，需要重新检查。

---

## 六、端到端集成测试

> **v2.0 整节重写**：本节原内容以"`task.Params["js_api"]` 二态开关"为核心组织测试用例（未设置/false/true 三种取值分别断言），这套用例设计的前提在 v2.0 里不再成立——`ApiScanner` 没有这个开关，被调度到就是"要做"。本节改为围绕 `ApiScanner.Run` 本身的行为设计测试用例，不再需要为"关闭态"单独写断言。

新建 `internal/core/scanner/api/api_scanner_test.go`，参照 `web_scanner_cdn_e2e_test.go` 的风格，用 `httptest.Server` 搭建一个包含内联 `<script>` 和外链 `<script src>` 的测试页面，覆盖：

1. **基本提取场景**：运行 `ApiScanner.Run`，验证 `ApiResult.APIs` 非空，且能验证到高/中/低三种置信度各自至少命中一条；测试页面引用的 JS 文件数不超过上限，`ApiResult.APIsTruncated` 必须是 `false`；
2. **截断场景**：测试页面引用超过 `JSAPIDefaultMaxFiles` 个 JS 文件，验证实际下载请求数被截断在上限以内，且 `ApiResult.APIsTruncated == true`——这一条专门验证"结果不完整"这个事实能从调用方一路透传到最终 `ApiResult`，中间任何一环（`downloadJSFiles` → `ExtractPageAPIs` → `buildApiResult`）漏传都会导致这条测试失败；
3. **首页单独命中场景**（不依赖深度爬取，核心断言）：测试目标首页本身不含任何可爬链接（或显式传 `task.Params["crawl"] = false`），验证 `results` 中唯一的 `ApiResult`（首页，`Depth == 0`）的 `APIs` 字段非空——这一条直接对应第五节描述的架构设计，验证"首页和子页面共用同一个 `buildApiResult`"这个设计确实生效，不会重现 v1.2 时代"首页拿不到提取结果"的历史问题；
4. **深度爬取多页命中场景**：`task.Params["crawl"] = true`、`task.Params["crawl_depth"] = 2`，测试站点构造深度 1、2 的子页面各自包含不同的内联/外链脚本，验证 `results` 里每一层深度对应的 `ApiResult.APIs` 都各自非空且内容不同（不是所有页面共享同一份结果），证明提取是逐页面独立执行的，不是只对首页做了一次然后复制给所有子页面；
5. **与 `WebScanner` 零耦合场景**：同一个测试目标分别跑 `web_scan` 和 `api_scan` 两个任务，验证互不影响——`web_scan` 的 `WebResult` 不应该出现任何 `APIs`/`APIsTruncated` 字段（因为这些字段已经从 `WebResult` 里移除，编译期即可保证），`api_scan` 的 `ApiResult` 也不应该有截图/指纹等 `WebResult` 专属字段，这条测试用编译期类型约束替代了运行时断言，主要用来在代码评审时提醒"这两个结果类型必须保持独立"。

以上 3、4 两条测试合并覆盖了第五节 5.5 节列出的"首页单独命中"和"深度爬取子页面命中"验收项，实施时两处引用同一份测试用例即可，不需要重复编写。

---

## 七、CLI 表格输出：`ApiResult` 从零实现 `TabularData`，截断时必须让用户知道结果不全

> **v2.0 整节重写**：本节原内容讨论的是"在 `WebResult` 现有列上追加 `APIs` 列"，前提是 `APIs`/`APIsTruncated` 字段挂在 `WebResult` 上。v2.0 里这两个字段挂在全新的 `ApiResult` 类型上，`ApiResult` 本身还没有任何 `Headers()`/`Rows()` 实现，需要从零编写，不存在"追加列"这回事——`api_scan` 命令的表格输出本来就应该是它自己的独立列集合，不需要和 `web_scan` 的表格长得一样。

### 7.1 为什么要做这一步

`ApiResult`（第三节新增）目前还没有实现 `TabularData` 接口，如果不做这一步，`api_scan` 命令的表格模式输出会退化成什么都显示不出来（或者报错/panic，取决于 `ConsoleReporter` 对未实现接口类型的兜底行为），用户唯一能看到结果的办法是 `-json` 输出，这违背了"CLI 是用户第一手感知入口"的基本预期。

`APIsTruncated` 更进一步：这是一个"当前结果不完整"的事实，如果只在 JSON 里加一个字段，不在 CLI 表格里做任何提示，用户拿着一份看起来正常的表格去做后续判断（比如统计接口数量、评估攻击面），实际上是在一份被静默阉割的数据上做决策而不自知。这是不可接受的——**宁可啰嗦地告诉用户"这份数据不全"，也不能让不完整的数据看起来和完整数据一模一样**。

### 7.2 `ApiResult` 的 `Headers()`/`Rows()`：`APIs` 列显示数量，截断时追加显眼标记

`ApiResult` 是全新类型，不存在"现有列表"，直接按它自己的字段设计表格：

```go
// Headers 实现 TabularData 接口
func (r ApiResult) Headers() []string {
	return []string{"URL", "Depth", "APIs"}
}

// Rows 实现 TabularData 接口
func (r ApiResult) Rows() [][]string {
	return [][]string{{
		r.URL,
		fmt.Sprintf("%d", r.Depth),
		r.apisCell(),
	}}
}

// apisCell 渲染 APIs 列的单元格内容：
//   - 完整：纯数字，如 "12"；
//   - 发生截断：数字后追加 "*"，如 "12*"——星号是终端表格里最省空间、
//     最不会破坏列宽对齐的标记方式，不使用需要变色的方案（pterm 的表格渲染
//     不为单个单元格单独上色，颜色方案的收益和实现复杂度不成正比）。
//   - APIs 为空且未截断：纯数字 "0"——与 WebResult 表格里"未开启功能时留空"
//     的语义不同，ApiResult 存在这件事本身就代表"已经做了提取"，"0" 是一个
//     真实的提取结果（没提取到），不是"这个功能没跑"，不应该用空字符串掩盖。
func (r ApiResult) apisCell() string {
	cell := fmt.Sprintf("%d", len(r.APIs))
	if r.APIsTruncated {
		cell += "*"
	}
	return cell
}
```

不新增 `Truncated` 独立列，理由与 v1.3 一致：截断是"APIs 这个数字的可信程度"的从属信息，不是一个独立维度，把截断标记编码进 `APIs` 列的字符串内容里更省空间，也更符合"信息应该贴在它所修饰的对象旁边"的直觉。

### 7.3 截断汇总提示：`cmd/agent/scan/api.go` 里 `PrintResults` 之后追加

星号本身足够让细心的用户注意到异常，但不能只靠一个容易被忽略的符号——必须有一行清晰的文字提示告诉用户"星号是什么意思、为什么会出现"。这段提示逻辑**不应该放进 `console.go`**：`ConsoleReporter.PrintResults` 是给 `Port`/`OS`/`Brute`/`Web`/`Api` 等所有扫描器共用的通用表格打印器，只认 `TabularData` 接口，完全不知道"APIs 截断"这种 `api_scan` 专属的业务语义；如果把这种业务判断塞进通用组件，以后每种扫描器想加自己的专属提示，`console.go` 就会堆满和具体业务耦合的 `if` 分支，这是需要避免的架构混乱。正确的位置是 `cmd/agent/scan/api.go` 这个 `api_scan` 命令自己的 `RunE` 里，在拿到 `results` 之后自行判断：

```go
			// 输出结果
			console := reporter.NewConsoleReporter()
			console.PrintResults(results)

			// api_scan 只要发生了 JS 文件截断，必须显式提示，不能让用户以为
			// 表格里的 APIs 数字就是完整清单。未截断时这里是没有任何输出的
			// 空操作，不会给不相关的用户增加噪音；不需要像 v1.3 那样先判断
			// "opts.JSAPI 是否开启"——api_scan 命令本身跑起来就意味着这个
			// 功能在执行，不存在"命令跑了但功能没开"的中间状态。
			printAPIsTruncatedHint(results)
```

```go
// printAPIsTruncatedHint 遍历本次扫描结果，如果存在任意一条 ApiResult.APIsTruncated == true，
// 汇总打印一条提示。只在这里做一次性汇总提示，不在每一行都重复啰嗦——表格里
// 每一行截断的记录已经用 "*" 标记出来了，这里只需要告诉用户 "*" 代表什么。
func printAPIsTruncatedHint(results []*model.TaskResult) {
	truncatedCount := 0
	for _, res := range results {
		if ar, ok := res.Result.(*model.ApiResult); ok && ar.APIsTruncated {
			truncatedCount++
		}
	}
	if truncatedCount == 0 {
		return
	}
	fmt.Printf("[!] 注意：%d 个页面的 JS 文件数超过单页上限（%d 个），APIs 列标记 \"*\" 的结果不完整，"+
		"仅反映上限内已下载文件的提取结果，完整数据需要提高上限或针对该目标单独分析\n",
		truncatedCount, crawler.JSAPIDefaultMaxFiles)
}
```

需要 `cmd/agent/scan/api.go` 引入 `"neoagent/internal/core/model"` 和 `"neoagent/internal/core/lib/crawler"` 两个 import（`crawler.JSAPIDefaultMaxFiles` 只用于提示文案里报数字，如果嫌跨包引用不划算，也可以直接把默认值 `20` 以外部已知常量的形式硬编码在提示文案里，两种做法都可以接受，实施时二选一即可，不必纠结）。

### 7.4 验收标准

- 运行 `api_scan` 且没有发生截断：`APIs` 列显示纯数字，不带星号，不出现截断提示；
- 运行 `api_scan` 且至少一个页面发生截断：对应行 `APIs` 列显示形如 `12*` 的内容，且表格下方打印出汇总提示，提示文字包含具体的截断页面数量和上限数值；
- `web_scan` 的表格输出完全不受影响（`WebResult.Headers()`/`Rows()` 本次没有任何改动，用 `git diff` 确认）；
- `go vet ./...` 通过（重点检查新增 import 没有循环依赖——`cmd/agent/scan` 引用 `internal/core/lib/crawler` 是单向依赖，`crawler` 包不会反向依赖 `cmd`，不存在循环问题）。

---

## 八、实施完成后的收尾检查

> **v2.0 整节重写**：本节原内容以"`--js-api` 开关关闭状态回归"为核心检查项，随着 `api_scan` 独立成命令，不再存在"关闭状态"，相应检查项作废；新增"`WebScanner` 零改动""`ApiScanner` 独立限流器"等 v2.0 架构特有的检查项。

- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] `go test ./... -race`（至少覆盖 `internal/core/lib/crawler/...`、`internal/core/scanner/api/...`）
- [ ] 确认改动中没有任何文件写入调用（`grep -rn "os.Create\|WriteFile\|TempFile"` 改动的文件应无命中）
- [ ] **`WebScanner` 零改动核心检查**：`git diff internal/core/scanner/web/web_scanner.go` 应为空——本次是新增独立的 `ApiScanner`，不是在 `WebScanner` 里加开关，如果 `web_scanner.go` 有任何改动，说明实现偏离了 v2.0 的架构设计，需要重新检查（这是 v2.0 相对 v1.3 最重要的回归验证，替代了 v1.3 里"确认 `js_api` 关闭状态下行为与实施前一致"这条检查，因为 v2.0 里已经不存在挂在 `WebScanner` 上的开关需要验证"关闭时无影响"）；
- [ ] 验证完整链路真实走通：用 `neoagent scan api --target ...` 实际执行一次，确认命令能正常跑通并真实触发 JS 下载（不能只在单元测试里直接调用 `ApiScanner.Run` 就算完成，这个验证针对的是第 2.4 节提到的"CLI 入口没打通"风险）；
- [ ] **核心检查项**：以下命令行都要手工跑一遍，确认行为符合预期：
  - `neoScan-Agent-web.exe scan api -t <target> --crawl=true --crawl-depth=2 --oj result.json`：确认 JSON 结果里深度为 0（首页）、1、2 的每一条 `ApiResult` 都各自带有非空的 `APIs` 字段（前提是对应页面确实引用了可提取的接口地址），不是只有首页有数据；
  - `neoScan-Agent-web.exe scan api -t <target>`（不传 `--crawl`）：确认首页的 `ApiResult.APIs` 有数据，即使本次没有触发深度爬取（比如目标首页链接很少、自动判断不需要爬取）——这条专门验证"未触发爬虫时提取整体失效"这类历史问题（v1.2 时代的教训）在新架构下不会重现；
- [ ] `api_scan` 的 `-h` 帮助输出中能看到完整的风险提示文字（不是被截断或省略）；
- [ ] `docs/Agent指令集规范.md` 已补充 `api_scan` 任务类型的独立章节，风险提示与代码里的提示文字一致（不允许文档和代码里的提示语句不一致导致用户看到两个版本）；
- [ ] CLI 表格模式下人工验证一次真实的截断场景（构造一个引用 JS 文件数超过 20 的测试页面），确认 `APIs` 列星号和汇总提示行都能正确出现；
- [ ] `git diff internal/core/lib/crawler/crawler.go` 应为空——`crawler.go` 本身不应该有任何改动（详见第四节 4.0 节的说明）；
- [ ] `git diff internal/core/model/result_types.go` 里 `type WebResult struct` 部分应为空——`WebResult` 结构体本次完全不改动（详见第三节 3.3 节验收标准）；
- [ ] `task_to_core.go` 的 `api_scan`/`apiScan` case 已改为映射到 `model.TaskTypeApiScan`（不再是 `model.TaskTypeWebScan` + `Params["mode"]="api"`），且 `RunnerManager` 同时能正确调度 `TaskTypeWebScan` 和 `TaskTypeApiScan` 两个独立 Runner（重构文档 5.1 节范围，本文档实施前提）；
- [ ] `ApiScanner` 持有自己独立的 `*qos.AdaptiveLimiter` 实例，不与 `WebScanner` 共享（重构文档 7.2 节），可以通过并发跑 `web_scan` 和 `api_scan` 各一个任务、观察两者的并发/限流互不干扰来验证。
