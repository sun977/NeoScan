# Web 扫描 JS 接口提取实施文档

> 文档版本：v1.3
> 修改时间：2026-08-04
> 依据方案：[`Web-JS接口提取方案.md`](./Web-JS接口提取方案.md)（v1.2，本文档只负责把方案拆成可以直接照做的步骤，不重复方案的背景论证，结论有分歧以方案文档为准；方案文档第六节"集成点"已与本文档 v1.3 同步修正为挂载在 `buildWebResult`）、[`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md)（规则设计的调研依据）
> 文档性质：实施清单，后续 JS 接口提取功能的代码开发按本文档的步骤顺序进行；每步给出改动文件、改动内容、验收标准
> v1.2 变更：新增第八节，CLI 表格输出补充 `APIs` 列，并对单页 JS 文件数超过上限导致的结果截断做显式提示（`WebResult` 新增 `APIsTruncated` 字段，全链路透传 + CLI 汇总提示），对应第三、五、六节同步补充截断标记的传递设计
> v1.3 变更（重要架构修正）：**`--js-api` 是独立于 `--crawl` 的全局开关，语义是"本次扫描产出的每一个页面结果都做提取"，与是否触发深度爬取无关。** v1.2 版本把 `ExtractJSAPI` 挂在 `crawler.Options` 上是架构错误——`crawler.Options` 只在触发深度爬取时才会被构造（`web_scanner.go` 第 386~387 行 `depth > 0 && len(seedLinks) > 0 && !isCDN`），这导致两个后果：① 首页结果走的是 `buildWebResult` 直接组装，根本不经过 `Crawler`，永远拿不到 JS API 提取；② 若本次任务不满足深度爬取条件（未传 `--crawl`、首页无链接、目标是 CDN），`Crawler` 根本不会被创建，即使用户显式传了 `--js-api` 也会整体静默失效、且没有任何提示。v1.3 把挂载点从 `crawler.Options`/`fetchAndExtract` 改为 `WebScanner.buildWebResult`——这是首页结果与所有深度爬取子页面结果唯一共同流经的收口函数，判断和调用只需要写一次，不需要在每个产出页面结果的调用点分别决定要不要做，从根本上消除"漏掉某条路径"的可能性。对应第二、三、四、五、六节均有改动，`crawler` 包不再感知 `js_api` 开关的存在，职责回归"只管爬页面拿 Body"。

---

## 零、实施前必读的三条硬约束

1. **这不是默认动作，必须用户显式开启才会执行**。方案文档里"不新建模块、不主动验证"的边界成立的前提，是这个功能本身也不会在用户没有明确表达意愿的情况下悄悄多做事情——下载外链 JS 文件、逐个发起额外的 `net/http` 请求，这本身就是比"分析已经到手的首页 HTML"更进一步的行为（会显著增加对目标发起的请求数量），必须比照现有 `crawl` 三态开关的先例，做成同一套"显式参数控制"的模式，不能默认跟随 `web_scan` 自动执行。详细设计见第二节。
2. **复用已导出的基础设施，不写第二套实现**：并发限流复用 `qos.AdaptiveLimiter`（`internal/core/lib/network/qos/limiter.go`），Crawler 已经持有一个实例（`web_scanner.go` 里 `crawler.New(crawler.Options{...}, s.limiter)` 传进去的那个），新增的 JS 下载逻辑必须复用这个已注入的 `limiter` 字段，不允许新建第二个限流器实例。
3. **JS 文件内容全程只在内存里流转，不落盘**，理由和具体做法见第四节 4.7 小节，这是本次实施在"设计方案没有明确到字节级细节"的地方需要遵循的具体决定，不是可以自由发挥的空间。

---

## 一、实施步骤总览

严格按顺序执行，后一步依赖前一步的产出：

| 步骤 | 改动文件 | 类型 | 状态 |
|---|---|---|---|
| 1 | `internal/core/model/result_types.go` | 修改（新增 `APIEndpoint` 类型 + `WebResult.APIs`/`APIsTruncated` 字段） | ⬜ 待开始 |
| 2 | `internal/core/scanner/web/crawler/jsapi.go` | 新建（提取规则 + 提取/过滤两段式实现 + 对外唯一入口 `ExtractPageAPIs`，独立函数，不挂在 `*Crawler` 上） | ⬜ 待开始 |
| 3 | `internal/core/scanner/web/crawler/jsapi_test.go` | 新建（表驱动单元测试） | ⬜ 待开始 |
| 4 | `internal/core/scanner/web/web_scanner.go` | 修改（新增二态开关解析函数 `resolveJSAPIExtract`；`pageData` 新增 `Body` 透传字段；`buildWebResult` 签名加 `ctx`，内部统一调用 `crawler.ExtractPageAPIs`） | ⬜ 待开始 |
| 5 | `internal/core/options/scan_web.go` | 修改（`WebScanOptions` 新增 `JSAPI bool` 字段，`ToTask()` 写入 `task.Params["js_api"]`） | ⬜ 待开始 |
| 6 | `cmd/agent/scan/web.go` | 修改（新增 `--js-api` CLI Flag，帮助文本内内置风险提示） | ⬜ 待开始 |
| 7 | `docs/Agent指令集规范.md` | 修改（补充 `js_api` 参数行，带风险提示） | ⬜ 待开始 |
| 8 | `internal/core/scanner/web/web_scanner_jsapi_test.go` | 新建（端到端集成测试，参照 `web_scanner_cdn_e2e_test.go` 风格，**必须覆盖首页单独命中的场景**） | ⬜ 待开始 |
| 9 | `internal/core/model/result_types.go` | 修改（`WebResult.Headers()`/`Rows()` 新增 `APIs` 列，截断时追加星号） | ⬜ 待开始 |
| 10 | `cmd/agent/scan/web.go` | 修改（`PrintResults` 之后追加截断汇总提示行） | ⬜ 待开始 |

不需要新增 `Scanner`、不需要新增 `TaskType`，改动范围完全在 `web` 扫描器内部闭环，和方案文档第七节的边界一致。**`crawler` 包不再感知 `js_api` 开关**（这是 v1.3 相对 v1.2 的关键差异，详见第二节）——`crawler.Options`/`crawler.Page`/`fetchAndExtract` 都不需要为这个功能新增任何字段或调用，`ExtractPageAPIs` 是一个只依赖 `(ctx, pageURL, body, limiter, maxFiles)` 的独立函数，`WebScanner` 在唯一的结果收口点 `buildWebResult` 里直接调用它，首页和子页面天然共用同一次判断、同一次调用。步骤 5~7 是完整参数链路——`resolveJSAPIExtract` 读的 `task.Params["js_api"]` 必须有上游写入才能生效，参照 `crawl`/`crawl_depth` 现有的完整链路（CLI Flag → `WebScanOptions` → `ToTask()` → `Task.Params`）一样打通，不能只写 `web_scanner.go` 一头而略过 CLI 入口。步骤 9~10 是 CLI 展示环节新增的内容——`APIs` 字段已经进了 `WebResult`（步骤 1），但 `WebResult.Rows()` 是固定列表，不会自动把新字段带出来，必须显式加列，否则用户在表格模式下完全看不到这个功能生效与否，只有翻 JSON 才知道；同理 `APIsTruncated` 也不会自动出现在任何地方，必须显式处理。

---

## 二、显式开关设计：比照 `crawl` 三态开关，新增独立的 `js_api` 开关

### 2.1 为什么不能默认执行，也不能和 `crawl` 共用一个开关

方案文档 v1.0 讨论阶段的隐含假设是"只要 `web_scan` 跑起来，顺手多做一步 JS 分析"，但重新审视后发现这个假设站不住脚，原因分两层：

**第一层：行为已经不是纯粹"被动"**。方案文档反复强调"被动分析、不发起额外验证请求"，这个表述本身是准确的——本功能确实不会去验证提取出的 API 是否存活。但**下载外链 JS 文件这个动作本身就是额外的网络请求**：一个页面可能引用几个到几十个 JS 文件（大型 SPA 打包出的 chunk 文件尤其多），这些文件现状是完全不下载的（`context.go` 的 `ExtractRichContext` 只拿 URL 列表，见方案文档 5.1 节引用的代码）。一旦默认开启，相当于给每次 `web_scan` 任务无条件叠加"N 次到目标域名/第三方 CDN 域名的额外 HTTP 请求"，这个量级的变化必须让用户知情并选择，不能藏在"顺手做的事"这个说法后面被默认执行掉。

**第二层：和 `crawl` 语义不同，合并会本身把复杂度做混**。`crawl` 控制的是"要不要对 BFS 队列做深度爬取"，`js_api` 控制的是"要不要下载并分析 JS 文件"——首页本身的内联 `<script>` 分析（方案文档 5.3 节）其实不需要额外请求，但外链 JS 下载需要。如果把这两件事强行塞进 `crawl` 这一个开关，会导致"用户关闭 `crawl`（不想爬子页面）却意外多出了一堆 JS 请求"或者反过来的语义混乱。两个开关必须独立，各自管好自己的一件事——这也是"好品味"的具体体现：消除"一个开关做两件不相关的事"这种特殊情况，而不是在代码里加 `if crawl && js_api` 这种耦合判断。

### 2.2 参数设计（照抄 `resolveCrawlDepth` 的三态模式，不发明新模式）

```944:962:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/web_scanner.go
// resolveCrawlDepth 综合三态参数（task.Params["crawl"] 显式开启/显式关闭/未指定）
// 与首页自动判断，得出最终爬取深度。三态优先级：显式参数 > 自动判断，这是
// "用户明确表达的意图永远盖过系统的猜测"这条原则在爬虫开关上的落地。
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

`js_api` 开关**不做三态，只做二态**，这一点和 `crawl` 刻意不同：`crawl` 有"自动判断"这个默认档位是因为"要不要深度爬"可以靠免费信号（状态码/Content-Type/链接数）低成本猜出合理默认值；而"要不要下载 JS 文件发起额外请求"没有类似的免费判断依据——无论首页看起来像什么，下载 JS 文件都是一个需要用户知情同意的主动决定，不存在"系统帮用户猜一个默认值"的中间地带，猜错了默认打开会让用户在不知情的情况下多发请求（违反第 2.1 节的结论），默认关闭又不能叫"三态"（因为压根不会有"自动判断为需要"这一支）。所以只需要 `task.Params["js_api"].(bool)`，且**只有显式传 `true` 才会执行**，未传或传其他类型一律视为关闭：

```go
// resolveJSAPIExtract 判断本次任务是否需要下载外链 JS 文件并提取接口地址。
// 这不是三态判断（不像 resolveCrawlDepth 那样有"未指定时自动判断"的默认档），
// 是纯粹的二态开关：只有用户显式传 task.Params["js_api"] = true 才会执行，
// 任何其他情况（未传、传了非 bool 类型、显式传 false）都视为不执行。
//
// 为什么没有"自动判断"档位：下载 JS 文件是一个会实际增加对目标网络请求数量的
// 主动行为，不存在能够零成本判断"这次该不该下载"的免费信号，让系统去猜measure
// 只会导致用户在不知情的情况下被动承担这个决定的后果，所以干脆不提供自动挡，
// 用户必须自己明确决定要不要开。
func (s *WebScanner) resolveJSAPIExtract(task *model.Task) bool {
	enable, _ := task.Params["js_api"].(bool)
	return enable
}
```

**`crawler.Options` 不新增任何字段，`crawler` 包对 `js_api` 这个开关一无所知。** 这是 v1.3 相对 v1.2 最核心的修正：v1.2 把 `ExtractJSAPI` 塞进 `crawler.Options`，隐含的假设是"JS API 提取是爬虫遍历行为的一部分"，但这个假设是错的——`crawler.Options` 只有在 `web_scanner.go` 第 386~387 行 `depth > 0 && len(seedLinks) > 0 && !isCDN` 成立、真正需要深度爬取时才会被构造出来传给 `crawler.New`。首页处理完全不经过这条路径（`Run()` 主干第 375 行直接调 `s.buildWebResult`，不创建 `Crawler` 实例），把开关挂在 `crawler.Options` 上等于说"只有触发深度爬取时才能用这个功能"，这既不是用户的意图（用户要的是"整个任务级别的开关"），也会在爬虫不触发时（未传 `--crawl`、首页没有可爬链接、目标是 CDN）导致 `--js-api` 整体静默失效。

正确的做法是**把"要不要提取"这个判断，从"要不要构造 `crawler.Options`"这件事上彻底解耦**。`resolveJSAPIExtract(task) bool` 只在一个地方被调用：`WebScanner.buildWebResult` 内部（详见第五节）。`buildWebResult` 是首页结果和每一个深度爬取子页面结果唯一共同流经的收口函数（`Run()` 里第 375 行首页、第 393 行每个子页面都调用它），在这里做判断和调用，天然覆盖所有页面，不需要 `crawler` 包知道 `js_api` 是什么。

`MaxJSFilesPerPage` 同理不进 `crawler.Options`，直接作为 `crawler.ExtractPageAPIs` 函数的入参（见第四节），或包级常量 `crawler.JSAPIDefaultMaxFiles` 兜底——不需要额外的任务参数控制，这是刻意的取舍——一次性暴露太多可调参数会增加用户理解成本，`crawl_depth` 已经证明"确实需要用户自定义的参数"值得暴露，但"单页最多下载几个 JS"目前没有类似的真实需求信号，先用一个经验值把功能跑起来，需要时再照 `crawl_depth` 的先例加一个 `js_api_max_files` 参数，不预先设计一个可能永远用不上的旋钮。

### 2.3 风险提示：在哪里提示、提示什么

方案要求"必须明确提示用户这会带来的风险"，落地方式分两处，覆盖"执行前"和"执行中"两个时间点：

**执行前（文档层面的提示，面向下达任务的人）**：`resolveJSAPIExtract` 函数的注释（已在 2.2 节写出）以及第四节 `ExtractPageAPIs` 函数的注释本身就是提示的一部分——任何阅读代码或者后续给 `js_api` 参数写外部接口文档的人，都会先看到"这会增加对目标的请求数量"这句话。这不是走过场：`docs/Agent指令集规范.md` 里 Master 下发任务的参数说明，如果未来给 `web_scan` 补充 `js_api` 参数的说明文档，必须原样带上这句风险提示，不能只写"提取 JS 接口"这种不含风险信息的一句话描述。

**执行中（日志层面的提示，面向运行时观测扫描行为的人）**：`crawler.ExtractPageAPIs`（第四节）在真正触发 JS 下载前打一条 `logger.Warnf` 级别日志（不是 `Infof`——用 Warn 级别是为了让这类"会增加请求量"的行为在日志里比普通信息更显眼，符合日志分级"Warn 用于需要引起注意但不是错误"的惯例），内容包含即将下载的 JS 文件数量和目标域名，让运维/审计人员能在日志流水里直接看到"这次扫描对 xxx 目标额外发起了 N 次 JS 下载请求"，不需要翻源码才能知道发生了什么：

```go
if len(jsURLs) > 0 {
	logger.Warnf("[crawler] JS API 提取已启用，将对 %s 额外下载 %d 个 JS 文件（此为用户显式开启的主动行为，会增加对目标的请求数量）", host, len(jsURLs))
}
```

注意：这里不再需要 `opts.ExtractJSAPI` 这个前置判断了——`ExtractPageAPIs` 函数本身就只会在 `WebScanner.buildWebResult` 确认 `resolveJSAPIExtract(task)` 为 `true` 时才被调用（见第五节），函数内部不需要再重复判断一次总开关。

### 2.4 打通完整参数链路：CLI Flag → `WebScanOptions` → `Task.Params`

`web_scanner.go` 里 `resolveJSAPIExtract` 读取的是 `task.Params["js_api"]`，这个值不会凭空出现，必须有上游写入。参照 `crawl`/`crawl_depth` 现有的完整链路（`internal/core/options/scan_web.go` 的 `WebScanOptions.ToTask()` → `cmd/agent/scan/web.go` 的 CLI Flag 绑定），新增同样的一套，**不能只改 `web_scanner.go` 就当作功能完成**——这是实施过程中最容易漏掉的一环，因为 `web_scanner.go` 单独看确实能编译通过、单元测试也能通过（测试里可以直接构造 `task.Params["js_api"] = true`），但通过真实 CLI 命令完全没有办法触发，等于功能没有真正对用户可用。

`internal/core/options/scan_web.go` 新增字段与写入逻辑：

```10:18:c:/mytools/code/go/NeoScan/neoAgent/internal/core/options/scan_web.go
type WebScanOptions struct {
	Target     string
	Ports      string
	Path       string
	Method     string
	Crawl      string // "auto"(默认) / "true" / "false"
	CrawlDepth int
	JSAPI      bool // 新增：是否下载外链 JS 文件并提取接口地址，默认 false，
	               // 二态而非三态（没有 "auto"，理由见方案文档/实施文档第二节）
	Output     OutputOptions
}
```

`ToTask()` 追加：

```go
if o.JSAPI {
	task.Params["js_api"] = true
}
// false 时不写 key，与 resolveJSAPIExtract 的"未传视为关闭"语义一致，
// 不需要显式写 task.Params["js_api"] = false 这种冗余分支。
```

`cmd/agent/scan/web.go` 新增 Flag，**帮助文本必须包含风险提示**，这是 CLI 层用户能看到的第一道提示，不能只写功能描述：

```go
flags.BoolVar(&opts.JSAPI, "js-api", opts.JSAPI, "是否下载外链 JS 文件并提取接口调用地址（默认 false）。"+
	"警告：开启后会对目标额外发起 JS 文件下载请求，请确认已获得授权后再开启")
```

`docs/Agent指令集规范.md` 第 3.3 节表格追加一行（紧邻 `crawl_depth` 之后），风险提示原文照抄，不允许精简掉：

```
| `js_api` | `--js-api` | bool | No | `false` | 是否下载外链 JS 文件并提取接口调用地址。**警告：开启后会对目标额外发起 JS 文件下载请求，属于主动网络行为，请确认已获得授权后再开启** |
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

`WebResult` 追加字段（紧邻 `EdgeComponents` 之后）：

```go
	// --- 以下为 JS 接口提取功能新增字段，omitempty，不影响现有序列化 ---
	APIs          []APIEndpoint `json:"apis,omitempty"`           // 从 JS 代码中静态提取到的接口调用地址清单。
	                                                                // 只有 task.Params["js_api"]=true 时才会非空，
	                                                                // 未开启该功能时这个字段和现在一样保持缺省（omitempty）。
	APIsTruncated bool          `json:"apis_truncated,omitempty"` // true 表示本页引用的外链 JS 文件数超过了
	                                                                // MaxJSFilesPerPage 上限，被截断丢弃了一部分文件
	                                                                // 未下载/未分析——也就是说 APIs 不是本页可能存在的
	                                                                // 完整清单，只是"上限范围内"的产出。刻意不放进
	                                                                // APIEndpoint 结构体里（否则每一条记录都要重复同一个
	                                                                // bool 值，是没有必要的冗余），这是页面级别的事实，
	                                                                // 只需要在 WebResult 顶层出现一次。
```

### 3.2 `crawler.Page` **不新增任何字段**

v1.2 版本曾计划给 `crawler.Page` 加 `APIs`/`APIsTruncated` 字段，这一步在 v1.3 里被去掉了——原因是 `crawler.Page` 只是 BFS 爬取子页面的原始数据载体（`URL`/`Body`/`Headers` 等），JS API 提取的判断和调用点已经全部收拢到 `WebScanner.buildWebResult`（见第五节），`Page` 只需要把已经有的 `Body` 字段原样交给 `buildWebResult`，提取结果直接进 `model.WebResult`，不需要在 `Page` 这一层再中转一次。这样 `crawler` 包的数据结构完全不为这个功能新增负担，`Page` 的字段列表和实施前一模一样。

### 3.3 验收标准

- `go build ./...` 通过；
- `go vet ./...` 无警告；
- 新增字段全部带 `omitempty`（除了 `Source`/`Confidence`/`URL` 这几个"只要有这条记录就必然有值"的必填字段），不影响任何现有 JSON 序列化输出的既有字段；
- `APIsTruncated` 只有在真实发生截断时才为 `true`：单页 JS 文件数 ≤ `MaxJSFilesPerPage` 时必须是 `false`（哪怕 `APIs` 是空的，也不能因为"没提取到东西"就误标为截断，两者是完全独立的判断维度）；
- `crawler.Page` 结构体字段数量与实施前完全一致（用 `git diff` 确认 `crawler.go` 里 `type Page struct` 这一块没有任何改动）。

---

## 四、`jsapi.go` 新建：提取规则、下载与两段式实现，对外只暴露一个函数

### 4.0 本节相对 v1.2 的结构调整

v1.2 把"提取规则"（原第四节）和"下载"（原第五节）分别放在 `jsapi.go` 新建、`crawler.go` 修改两处，因为下载函数（`downloadJS`/`downloadJSFiles`）当时设计成 `*Crawler` 的方法，需要用到 `c.client`/`c.limiter`/`c.opts`。v1.3 里下载函数不再属于 `*Crawler`——`ExtractPageAPIs` 是一个完全独立的包级函数，不需要、也不应该依赖一个 `*Crawler` 实例（首页调用它时压根没有 `Crawler`）。因此**下载相关的函数全部并入本节、同一个 `jsapi.go` 文件**，`crawler.go` 不再有任何改动，第五节原有的"外链 JS 文件下载"内容合并到本节 4.5~4.6 小节。

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
// 为什么这是一个需要用户显式开启的功能，而不是像 leak.go 一样默认跟随扫描执行：
//   leak.go 检测的是"已经到手的页面正文"里有没有敏感信息，不需要额外的网络
//   请求。本文件的输入除了内联 script（零成本，见 4.7 节），还包括外链 js 文件
//   的下载内容——下载动作本身就是新增的网络请求，必须由调用方在用户显式传入
//   task.Params["js_api"]=true 时才触发下载并调用本文件对外暴露的唯一入口
//   函数 ExtractPageAPIs，见 Web-JS接口提取实施文档.md 第二节、第六节。
//
// 本文件不依赖 *Crawler：ExtractPageAPIs 是纯粹的包级函数，只依赖调用方显式
// 传入的 (ctx, pageURL, body, limiter, maxFiles) 这几个参数，不读取任何
// *Crawler 内部状态。这是刻意的设计——首页结果（WebScanner.Run 主干）和
// 深度爬取子页面结果都要调用同一份提取逻辑，而首页处理时根本不存在一个
// *Crawler 实例，如果提取函数绑在 *Crawler 方法上，首页就永远调不到它
// （这正是 v1.2 版本的架构错误，见实施文档 v1.3 变更说明）。
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
// 这是一个纯粹的包级函数，不依赖任何 *Crawler 实例——调用方无论是
// WebScanner.Run 主干（处理首页）还是深度爬取子页面，都通过
// WebScanner.buildWebResult 这一个统一入口调用它（见 Web-JS接口提取
// 实施文档.md 第五节），不存在"首页调不到"的问题。
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
- `go test ./internal/core/scanner/web/crawler/... -run TestExtractPageAPIs -v` 全部通过；
- `go vet` 确认 `crawler.go` 文件本身没有任何改动（`git diff internal/core/scanner/web/crawler/crawler.go` 应为空）。

---

## 五、集成点：`WebScanner.buildWebResult` 接入提取，首页与子页面共用同一次调用

### 5.1 为什么挂在 `buildWebResult`，不是 `fetchAndExtract`

`buildWebResult`（`web_scanner.go` 第 862 行）是首页结果与所有深度爬取子页面结果唯一共同流经的收口函数：

```375:381:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/web_scanner.go
	homeResult := s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
		URL: targetURL, Depth: 0, StatusCode: homeStatusCode, Title: homeTitle,
		Body: homeBody, Headers: homeHeaders, ContentLength: homeContentLen, RichContext: homeRichCtx,
		Forms: homeForms, Params: homeParams,
		Screenshot: screenshotB64, Favicon: faviconB64,
		EdgeComponents: edgeComponents,
	})
	results = append(results, homeResult)
```

```392:397:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/web_scanner.go
		for _, p := range pages {
			results = append(results, s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
				URL: p.URL, Depth: p.Depth, StatusCode: p.StatusCode, Title: p.Title,
				Body: p.Body, Headers: p.Headers, Forms: p.Forms, Params: p.Params, Leaks: p.Leaks,
			}))
		}
```

两处调用传入的 `pageData` 字段不同，但都汇入同一个 `buildWebResult`。把 `resolveJSAPIExtract(task)` 的判断和 `crawler.ExtractPageAPIs` 的调用放在这一个函数内部，首页和每一个子页面自动、无差别地获得提取能力，不需要在两个调用点分别写判断——这正好对应用户提出的要求：**`--js-api` 是任务级别的全局开关，与是否触发深度爬取无关，深度为 N 的爬虫子页面同样要覆盖，不存在需要特殊处理的路径。**

### 5.2 `pageData` 新增 `Body` 透传字段说明

`pageData`（`web_scanner.go` 第 8xx 行附近）已经有 `Body string` 字段（首页和子页面调用处都已经在传，见上面 5.1 节引用的代码），`buildWebResult` 内部本来就能拿到 `pd.Body` 和 `pd.URL`，**不需要为了这个功能新增任何 `pageData` 字段**，只需要在函数体内部多几行调用逻辑。这是 v1.3 相对 v1.2 的另一处简化：v1.2 需要在 `crawler.Page`/`pageData` 之间新增 `APIs`/`APIsTruncated` 两个透传字段（原 3.2/6.3/6.4 节），v1.3 里 `ExtractPageAPIs` 直接在 `buildWebResult` 内部对 `pd.Body` 调用，产出直接组装进返回的 `model.WebResult`，不需要任何中间结构体透传。

### 5.3 `buildWebResult` 签名改动：新增 `ctx context.Context` 参数

JS 下载是网络请求，必须能被上层的超时/取消控制约束，`buildWebResult` 原签名 `(task *model.Task, startTime time.Time, ip string, port int, pd pageData)` 里没有 `ctx`，需要新增：

```go
func (s *WebScanner) buildWebResult(ctx context.Context, task *model.Task, startTime time.Time, ip string, port int, pd pageData) *model.TaskResult {
	...
}
```

两个调用点（5.1 节引用的第 375 行、第 393 行）相应改为 `s.buildWebResult(ctx, task, ...)`——`Run()` 方法本身已经持有 `ctx` 参数（方法签名 `func (s *WebScanner) Run(ctx context.Context, task *model.Task)`），直接传下去即可，不需要新开 context。

### 5.4 `buildWebResult` 内部接入：组装 `model.WebResult` 之前调用一次

在现有 `contentLength` 计算之后、`return &model.TaskResult{...}` 之前插入：

```go
apis, apisTruncated := s.resolveJSAPIResult(ctx, task, pd.URL, pd.Body)
```

```go
// resolveJSAPIResult 是 buildWebResult 接入 JS 接口提取的唯一调用点。
// 只有 resolveJSAPIExtract(task) 为 true（用户显式传 --js-api）才会真正
// 触发下载，否则直接返回 (nil, false)，不产生任何额外请求——这个判断
// 只写在这一个地方，首页和爬虫子页面共用同一次判断，不会出现只覆盖
// 其中一条路径的情况。
func (s *WebScanner) resolveJSAPIResult(ctx context.Context, task *model.Task, pageURL, body string) ([]model.APIEndpoint, bool) {
	if !s.resolveJSAPIExtract(task) {
		return nil, false
	}
	return crawler.ExtractPageAPIs(ctx, pageURL, body, nil, s.limiter, crawler.JSAPIDefaultMaxFiles)
}
```

`model.WebResult` 组装处追加两个字段：

```go
	Result: &model.WebResult{
		...
		EdgeComponents:  pd.EdgeComponents,
		APIs:            apis,
		APIsTruncated:   apisTruncated,
	},
```

`limiter` 直接传 `s.limiter`——`WebScanner` 结构体本来就持有这个字段（`web_scanner.go` 第 48 行 `limiter *qos.AdaptiveLimiter`），无论首页还是爬虫子页面调用，用的都是同一个全局限流实例，与第四节 4.7 小节"复用调用方传入的 limiter，不新建限流器"的原则一致。`http.Client` 传 `nil`，`ExtractPageAPIs` 内部会用默认 client 兜底（见 4.8 节），不需要 `WebScanner` 专门再持有一个下载用的 client。

### 5.5 若干易漏点核对

- **子页面 `pageData` 目前没有传 `EdgeComponents`**（见 5.1 节第二段引用代码），这是既有行为（子页面不重复做 CDN 判断），与本次改动无关，不需要顺带补上；
- **`crawler.Page` 不需要任何改动**（详见第 3.2 节），`p.Body` 已经在 `Page` 结构体里，`pageData{Body: p.Body, ...}` 这行赋值本来就存在，不是新增的；
- `resolveJSAPIResult` 内部只做一次 `resolveJSAPIExtract` 判断，`crawler.ExtractPageAPIs` 本身不再重复判断（详见第四节 4.8 节的函数注释），避免同一个开关判断出现两处。

### 5.6 验收标准

- `go build ./...`、`go vet ./...` 通过；
- `go test ./internal/core/scanner/web/... -race` 通过，重点关注 `downloadJSFiles` 的并发写 map 场景（已用 `sync.Mutex` 保护，`-race` 应该跑不出竞争）；
- 手工验证：`task.Params["js_api"]` 不传或传 `false` 时，`buildWebResult` 完全不会调用 `crawler.ExtractPageAPIs`，日志里不应出现任何 JS 下载相关的 Warn 日志；
- **首页单独命中场景**（新增，v1.3 核心验收项）：构造一个不触发深度爬取的目标（比如首页没有可爬链接，或显式传 `--crawl=false`），只传 `--js-api`，验证首页对应的 `WebResult.APIs` 依然非空——这条断言专门验证 v1.2 版本"首页拿不到提取"的 bug 已经被修复；
- **深度爬取子页面命中场景**：同时传 `--crawl=true --crawl-depth=2 --js-api`，验证深度为 1、2 的子页面 `WebResult.APIs` 同样非空，不是只有首页（Depth=0）才有数据；
- 构造一个引用 JS 文件数超过 `MaxJSFilesPerPage` 的测试页面，验证：`WebResult.APIsTruncated == true`，且日志里能看到明确的"已丢弃 N 个未下载"提示（不是只在返回值里体现，日志必须同步出现）。

---

## 六、端到端集成测试

参照 `web_scanner_cdn_e2e_test.go` 的风格，新建 `web_scanner_jsapi_test.go`，用 `httptest.Server` 搭建一个包含内联 `<script>` 和外链 `<script src>` 的测试页面，覆盖：

1. `task.Params["js_api"]` 未设置：`WebResult.APIs` 应为空，`WebResult.APIsTruncated` 应为 `false`，且测试服务器记录的 JS 文件请求数应为 0（验证"不传参数就完全不触发下载"）；
2. `task.Params["js_api"] = false`：同上，行为等价于未设置；
3. `task.Params["js_api"] = true`：`WebResult.APIs` 非空，且能验证到高/中/低三种置信度各自至少命中一条；测试页面引用的 JS 文件数不超过上限，`WebResult.APIsTruncated` 必须是 `false`；
4. `task.Params["js_api"] = true` 且测试页面引用超过 `JSAPIDefaultMaxFiles` 个 JS 文件：验证实际下载请求数被截断在上限以内，且 `WebResult.APIsTruncated == true`——这一条是本次新增的核心断言，专门验证"结果不完整"这个事实能从调用方一路透传到最终 `WebResult`，中间任何一环（`downloadJSFiles` → `ExtractPageAPIs` → `buildWebResult`）漏传都会导致这条测试失败；
5. **首页单独命中场景**（不依赖深度爬取）：测试目标首页本身不含任何可爬链接（或显式传 `task.Params["crawl"] = false`），只传 `task.Params["js_api"] = true`，验证 `results` 中唯一的 `WebResult`（首页，`Depth == 0`）的 `APIs` 字段非空——这一条直接对应第五节 5.1 节描述的架构修正，是 v1.3 相对 v1.2 新增的核心断言，v1.2 的实现在这条测试下会失败（首页拿不到提取结果）；
6. **深度爬取多页命中场景**：`task.Params["crawl"] = true`、`task.Params["crawl_depth"] = 2`、`task.Params["js_api"] = true`，测试站点构造深度 1、2 的子页面各自包含不同的内联/外链脚本，验证 `results` 里每一层深度对应的 `WebResult.APIs` 都各自非空且内容不同（不是所有页面共享同一份结果），证明提取是逐页面独立执行的，不是只对首页做了一次然后复制给所有子页面。

以上 5、6 两条测试合并覆盖了第五节 5.6 节列出的"首页单独命中"和"深度爬取子页面命中"验收项，实施时两处引用同一份测试用例即可，不需要重复编写。

---

## 七、CLI 表格输出：新增 `APIs` 列，截断时必须让用户知道结果不全

### 7.1 为什么要做这一步

`APIs`/`APIsTruncated` 字段进了 `WebResult`（第三节），但 `WebResult.Rows()` 是手写的固定列清单（`internal/core/model/result_types.go` 第 153~189 行现有的 `Headers()`/`Rows()`），新加的字段不会自动出现在表格里——不加这一步，用户用默认的表格模式跑 `--js-api`，看到的表格和没开这个功能时一模一样，唯一能确认功能生效的办法是去翻 `-json` 输出，这违背了"CLI 是用户第一手感知入口"的基本预期。

`APIsTruncated` 更进一步：这是一个"当前结果不完整"的事实，如果只在 JSON 里加一个字段，不在 CLI 表格里做任何提示，用户拿着一份看起来正常的表格去做后续判断（比如统计接口数量、评估攻击面），实际上是在一份被静默阉割的数据上做决策而不自知。这是不可接受的——**宁可啰嗦地告诉用户"这份数据不全"，也不能让不完整的数据看起来和完整数据一模一样**。

### 7.2 表格列设计：`APIs` 列显示数量，截断时追加显眼标记

不新增 `Truncated` 独立列。理由：截断是"APIs 这个数字的可信程度"的从属信息，不是一个独立维度，为它单开一列会让表格在绝大多数未开启 `--js-api` 或未发生截断的场景下始终多一列空白（`Headers()` 是固定表头，列在不在数据无关都会显示），对现有 95% 场景的用户是纯粹的视觉噪音。把截断标记直接编码进 `APIs` 列的字符串内容里，只有真正发生截断时才会让这个单元格看起来"有点不一样"，符合"没有特殊情况时表现和以前完全一致"的原则。

```153:189:c:/mytools/code/go/NeoScan/neoAgent/internal/core/model/result_types.go
// Headers 实现 TabularData 接口
func (r WebResult) Headers() []string {
	return []string{"URL", "IP", "Port", "Status", "Len", "Title", "Server", "TechStack"}
}

// Rows 实现 TabularData 接口
func (r WebResult) Rows() [][]string {
	...
	return [][]string{{
		r.URL,
		r.IP,
		fmt.Sprintf("%d", r.Port),
		fmt.Sprintf("%d", r.StatusCode),
		fmt.Sprintf("%d", r.ContentLength),
		r.Title,
		server,
		stack,
	}}
}
```

改为：

```go
// Headers 实现 TabularData 接口
func (r WebResult) Headers() []string {
	return []string{"URL", "IP", "Port", "Status", "Len", "Title", "Server", "TechStack", "APIs"}
}

// Rows 实现 TabularData 接口
func (r WebResult) Rows() [][]string {
	...
	return [][]string{{
		r.URL,
		r.IP,
		fmt.Sprintf("%d", r.Port),
		fmt.Sprintf("%d", r.StatusCode),
		fmt.Sprintf("%d", r.ContentLength),
		r.Title,
		server,
		stack,
		r.apisCell(),
	}}
}

// apisCell 渲染 APIs 列的单元格内容：
//   - 未开启 --js-api（APIs 为 nil 且未截断）：空字符串，和现有其他 omitempty
//     字段的表格展示习惯一致（如当前 TechStack 为空时也是空字符串，不是显式的 "0"）；
//   - 开启且完整：纯数字，如 "12"；
//   - 开启且发生截断：数字后追加 "*"，如 "12*"——星号是终端表格里最省空间、
//     最不会破坏列宽对齐的标记方式，不使用需要变色的方案（pterm 的表格渲染
//     不为单个单元格单独上色，颜色方案的收益和实现复杂度不成正比）。
func (r WebResult) apisCell() string {
	if len(r.APIs) == 0 && !r.APIsTruncated {
		return ""
	}
	cell := fmt.Sprintf("%d", len(r.APIs))
	if r.APIsTruncated {
		cell += "*"
	}
	return cell
}
```

### 7.3 截断汇总提示：`cmd/agent/scan/web.go` 里 `PrintResults` 之后追加

星号本身足够让细心的用户注意到异常，但不能只靠一个容易被忽略的符号——必须有一行清晰的文字提示告诉用户"星号是什么意思、为什么会出现"。这段提示逻辑**不应该放进 `console.go`**：`ConsoleReporter.PrintResults` 是给 `Port`/`OS`/`Brute`/`Web` 等所有扫描器共用的通用表格打印器，只认 `TabularData` 接口，完全不知道"APIs 截断"这种 Web 扫描器专属的业务语义；如果把这种业务判断塞进通用组件，以后每种扫描器想加自己的专属提示，`console.go` 就会堆满和具体业务耦合的 `if` 分支，这是需要避免的架构混乱。正确的位置是 `web.go` 这个 Web 扫描命令自己的 `RunE` 里，在拿到 `results` 之后自行判断。

```336:345:c:/mytools/code/go/NeoScan/neoAgent/cmd/agent/scan/web.go
			// 输出结果
			console := reporter.NewConsoleReporter()
			console.PrintResults(results)
```

改为：

```go
			// 输出结果
			console := reporter.NewConsoleReporter()
			console.PrintResults(results)

			// --js-api 场景下，如果任意一条结果发生了 JS 文件截断，必须显式提示，
			// 不能让用户以为表格里的 APIs 数字就是完整清单。只在真正发生截断时
			// 才打印，未开启 --js-api 或未截断时这里是没有任何输出的空操作，
			// 不会给不相关的用户增加噪音。
			if opts.JSAPI {
				printAPIsTruncatedHint(results)
			}
```

```go
// printAPIsTruncatedHint 遍历本次扫描结果，如果存在任意一条 WebResult.APIsTruncated == true，
// 汇总打印一条提示。只在这里做一次性汇总提示，不在每一行都重复啰嗦——表格里
// 每一行截断的记录已经用 "*" 标记出来了，这里只需要告诉用户 "*" 代表什么。
func printAPIsTruncatedHint(results []*model.TaskResult) {
	truncatedCount := 0
	for _, res := range results {
		if wr, ok := res.Result.(model.WebResult); ok && wr.APIsTruncated {
			truncatedCount++
		}
	}
	if truncatedCount == 0 {
		return
	}
	fmt.Printf("[!] 注意：%d 个目标的 JS 文件数超过单页上限（%d 个），APIs 列标记 \"*\" 的结果不完整，"+
		"仅反映上限内已下载文件的提取结果，完整数据需要提高上限或针对该目标单独分析\n",
		truncatedCount, crawler.JSAPIDefaultMaxFiles)
}
```

需要 `cmd/agent/scan/web.go` 新增 `"neoagent/internal/core/model"` 和 `"neoagent/internal/core/scanner/web/crawler"` 两个 import（`crawler.JSAPIDefaultMaxFiles` 只用于提示文案里报数字，如果嫌跨包引用不划算，也可以直接把默认值 `20` 以外部已知常量的形式硬编码在提示文案里，两种做法都可以接受，实施时二选一即可，不必纠结）。

### 7.4 验收标准

- 未传 `--js-api`：表格里 `APIs` 列全部为空字符串，末尾不出现任何截断提示，和实施前的表格输出完全一致（除了多一列空表头，这是唯一可见的变化）；
- 传 `--js-api` 且没有发生截断：`APIs` 列显示纯数字，不带星号，不出现截断提示；
- 传 `--js-api` 且至少一个目标发生截断：对应行 `APIs` 列显示形如 `12*` 的内容，且表格下方打印出汇总提示，提示文字包含具体的截断目标数量和上限数值；
- `go vet ./...` 通过（重点检查新增 import 没有循环依赖——`cmd/agent/scan` 引用 `internal/core/scanner/web/crawler` 是单向依赖，`crawler` 包不会反向依赖 `cmd`，不存在循环问题）。

---

## 八、实施完成后的收尾检查

- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] `go test ./... -race`（至少覆盖 `internal/core/scanner/web/...`）
- [ ] 确认改动中没有任何文件写入调用（`grep -rn "os.Create\|WriteFile\|TempFile"` 改动的文件应无命中）
- [ ] 确认 `js_api` 关闭状态（不传参数）下行为与实施前完全一致（回归验证，防止意外影响到没开启这个功能的现有用户）
- [ ] 验证完整链路真实走通：用 `neoagent scan web --target ... --js-api` 实际执行一次，确认 `--js-api` 能真正触发下载（不能只在单元测试里直接构造 `task.Params["js_api"]=true` 就算完成，这个验证针对的是第 2.4 节提到的"CLI 入口没打通"风险）
- [ ] **（v1.3 新增，核心检查项）** 用户举例的两种命令行都要手工跑一遍，确认行为符合预期：
  - `neoScan-Agent-web.exe scan web -t <target> --crawl=true --crawl-depth=2 --js-api --oj result.json`：确认 JSON 结果里深度为 0（首页）、1、2 的每一条 `WebResult` 都各自带有非空的 `APIs` 字段（前提是对应页面确实引用了可提取的接口地址），不是只有首页有数据；
  - `neoScan-Agent-web.exe scan web -t <target> --js-api`（不传 `--crawl`）：确认首页的 `WebResult.APIs` 有数据，即使本次没有触发深度爬取（比如目标首页链接很少、自动判断不需要爬取）——这条专门验证 v1.2 版本"未触发爬虫时 `--js-api` 整体失效"的 bug 已修复；
- [ ] `--js-api` 的 `-h` 帮助输出中能看到完整的风险提示文字（不是被截断或省略）；
- [ ] `docs/Agent指令集规范.md` 第 3.3 节表格已补充 `js_api` 行，风险提示与代码里的提示文字一致（不允许文档和代码里的提示语句不一致导致用户看到两个版本）；
- [ ] CLI 表格模式下人工验证一次真实的截断场景（构造一个引用 JS 文件数超过 20 的测试页面），确认 `APIs` 列星号和汇总提示行都能正确出现，且未开启/未截断场景下表格观感与实施前一致；
- [ ] `git diff internal/core/scanner/web/crawler/crawler.go` 应为空——`crawler.go` 本身不应该有任何改动（详见第四节 4.0 节的说明），如果这个文件有改动，说明实现偏离了 v1.3 的架构设计，需要重新检查。
