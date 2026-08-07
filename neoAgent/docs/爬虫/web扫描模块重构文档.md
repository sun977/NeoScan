# Web 扫描模块重构方案

- **版本**: v1.0（方案对齐阶段，未落地代码）
- **状态**: 讨论定稿，等待转入实施文档
- **关联文档**:
  - [`00.原子扫描器开发指南[定稿].md`](../00.原子扫描器开发指南[定稿].md)（Runner/RunnerManager 双路径架构铁律，本方案新增的 `ApiScanner` 必须遵守）
  - [`Master-Agent扫描类型映射说明.md`](../Master-Agent扫描类型映射说明.md)（`apiScan` 当前映射方式，本方案要修改的目标）
  - [`API扫描-js提取/Web-JS接口提取方案.md`](../API扫描-js提取/Web-JS接口提取方案.md) / [`API扫描-js提取/Web-JS接口提取实施文档.md`](../API扫描-js提取/Web-JS接口提取实施文档.md)（JS API 提取能力本身的详细设计，本方案不重复描述，只确定它挂载在哪个 Scanner 下）
  - [`原子能力批量扫描能力扩充/原子扫描器批量扫描能力扩充方案.md`](../原子能力批量扫描能力扩充/原子扫描器批量扫描能力扩充方案.md)（同类"先方案对齐、逐项确认、再写实施文档"的写作范式参考）

---

## 一、结论先行

【核心判断】
✅ 值得做。`ApiScan` 从 `web_scan` 的一个 `mode:"api"` 参数，升级为 Master 侧真正独立的业务场景（新 `TaskType`），Agent 侧同步把"抓取"能力从 `WebScanner` 中拆出，形成三层可复用架构。

一句话说清方案：**把"怎么拿到一个页面"（协议探测 + 渲染 + 爬取）与"拿到页面之后做什么"（指纹识别/截图 vs. JS API 提取）彻底解耦，前者下沉为 `crawler` 包的通用能力，后者分别归属 `WebScanner` 和新增的 `ApiScanner` 两个平级的业务 Scanner。**

| 维度 | 现状 | 方案后 |
|---|---|---|
| `api_scan` 的 Master 映射 | 借道 `web_scan` + `Params["mode"]="api"` | 独立 `TaskTypeApiScan`，Agent 侧优先支持，Master 侧先记录待办 |
| JS API 提取挂载点 | 计划挂在 `WebScanner.buildWebResult` 里 | 挂在新增 `ApiScanner` 里 |
| 抓取逻辑（协议探测/渲染/BFS） | 私有在 `web_scanner.go` + `scanner/web/crawler` 包内部，`WebScanner` 独占 | 下沉到 `core/lib/crawler` 包，`WebScanner`/`ApiScanner` 共同调用 |
| `WebScanner` 职责 | 抓取 + 指纹识别 + 截图 + CDN 判断 + 组装结果，一个类型背 4+ 种关注点 | 只保留指纹识别 + 截图 + CDN 判断 + 组装结果，抓取委托给 `crawler` 包 |
| `ApiScanner` 是否重复实现抓取 | 不适用（尚不存在） | 否，与 `WebScanner` 共享同一个 `crawler.FetchAndCrawl` 入口 |
| 是否破坏现有 `web_scan` CLI/Master 用法 | 不适用 | 否，`WebScanner` 对外行为不变，属于内部重构 |

---

## 二、现状分析

### 2.1 `WebScanner` 当前背负的职责

```35:52:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/web_scanner.go
type WebScanner struct {
	// 基础设施
	browserManager  *browser.BrowserManager
	browserLauncher *browser.BrowserLauncher

	// 指纹引擎 (复用 internal/pkg/fingerprint)
	fpEngine *fpHttp.HTTPEngine

	// CDN 边缘节点检测 (复用 internal/pkg/edge)
	edgeDetector *edge.Detector

	// 资源限制 (QoS)
	limiter *qos.AdaptiveLimiter

	mu       sync.Mutex
	initOnce sync.Once
}
```

`runOnePort` 一个函数内完整串联了：协议猜测 → CDN 查表 → go-rod 渲染（含截图/favicon）→ 协议误判纠正的 fallback 双发选优 → 攻击面提取 → 组装结果 → 触发 BFS 深度爬取（`web_scanner.go` 174-402 行）。这是一个典型的"上帝函数 + 上帝对象"组合：一次修改指纹识别逻辑，理论上不该影响截图逻辑，但两者共享同一个函数栈、同一组局部变量，耦合度高、认知负担重。

### 2.2 `apiScan` 当前的 Master 映射方式

```96:98:c:/mytools/code/go/NeoScan/neoAgent/internal/service/adapter/task_to_core.go
case "api_scan", "apiScan":
    coreTask.Type = model.TaskTypeWebScan
    coreTask.Params["mode"] = "api"
```

`docs/Master-Agent扫描类型映射说明.md` 第 23 行同样把 `apiScan` 记录为"映射到 `web_scan`，参数 `mode:"api"`"。这是早期"Agent 只提供最基础原子能力，业务场景靠参数组合"设计哲学下的产物，但 `mode:"api"` 参数目前在 `WebScanner` 内部并未有任何分支真正处理（搜索 `Params["mode"]` 在 `web_scanner.go` 中无引用），是一个只存在于文档和 Master 映射表里、Agent 侧从未真正实现的"名义映射"。

### 2.3 `crawler` 包现状（迁移前）

```go
// neoAgent/internal/core/scanner/web/crawler/
crawler.go   // Crawler 结构体 + Crawl() BFS 主循环，纯 net/http，无 go-rod 依赖
extract.go   // ExtractLinksAndForms 攻击面提取
leak.go      // DetectLeaks 被动泄露检测
```

对外依赖仅有 `core/lib/network/qos`（限流）和 `core/model`（`FormInfo`/`LeakInfo` 等纯数据结构），无任何反向依赖——外部只有 `web_scanner.go` 单向调用它。这是一次风险很低的搬迁：`go build` 能完整暴露所有需要改的 import 路径，不涉及逻辑改动。

### 2.4 截图/favicon 与 go-rod 渲染的关系（本方案的关键技术难点）

截图/favicon 采集依赖 go-rod 的 `*rod.Page` 对象在 `page.Close()` 之前的这段存活窗口期（`web_scanner.go` 279-286 行），但这段窗口期同时也是"渲染后 body/richCtx/links 到手"的产出时机——两者共享同一个时间窗口，却是两个不同性质的产出：

- **渲染数据**（body/headers/links/richCtx）：`WebScanner` 和 `ApiScanner` 都需要，是通用需求（`ApiScanner` 需要它来做 JS API 提取；未来若要做"动态渲染后捕获真实发出的 XHR/Fetch 请求"，同样需要渲染后的运行时环境）
- **展示衍生品**（截图/favicon 图片）：只有 `WebScanner` 需要，`ApiScanner` 现在和可预见的未来都不需要"看这个页面长什么样"

这两者不能用同一种方式下沉：如果把 go-rod 渲染整体下沉但不管截图，`WebScanner` 就拿不到 `page` 对象；如果把截图逻辑也一起下沉，`crawler` 包就要背上一个只有一个调用方用得到的图片语义，且以后 `WebScanner` 新增别的展示需求（如页面可访问性快照）还要继续污染 `crawler` 包的返回结构。

---

## 三、方案设计

### 3.1 目录结构改动

```
core/lib/crawler/                     ← 整体从 core/scanner/web/crawler 迁移
    crawler.go / extract.go / leak.go / jsapi.go   原样迁移（含各自的 _test.go）
                                        jsapi.go 是 JS API 提取新增文件，
                                        内容见《Web-JS接口提取实施文档.md》，
                                        不受本次迁移影响，随包一起迁移

    fetch.go                          ← 新增：协议探测 + go-rod 渲染 + net/http
                                        fallback 双发选优，从 web_scanner.go 的
                                        runOnePort/fallbackFetchBestProtocol/
                                        pickBestFetchOutcome 等函数原样搬迁

core/scanner/web/web_scanner.go       ← 瘦身：调用 crawler.FetchAndCrawl 拿页面，
                                        自己只做指纹识别 + 截图 + CDN 展示 + 
                                        组装 WebResult

core/scanner/api/api_scanner.go       ← 新增 ApiScanner：调用同一个
                                        crawler.FetchAndCrawl 拿页面，
                                        自己只做 JS API 提取 + 组装 ApiResult

core/factory/api_factory.go           ← 新增，参照 web_factory.go 模式
cmd/agent/scan/api.go                 ← 新增 CLI 子命令
internal/core/options/scan_api.go     ← 新增 ApiScanOptions
```

依赖方向调整为彻底的单向链：

```
core/scanner/web  ─┐
                    ├─→ core/lib/crawler ─→ core/lib/browser / core/lib/network/qos / core/model
core/scanner/api  ─┘
```

不存在任何"下层反过来 import 上层"的情况。

### 3.2 `crawler.FetchAndCrawl` 接口设计（核心技术方案）

```go
// core/lib/crawler/fetch.go

// FetchOptions 控制 FetchAndCrawl 的抓取行为
type FetchOptions struct {
	ProtocolHint string // 复用现有 protocolHint 语义
	CrawlDepth   int    // <=0 表示不做 BFS 深度爬取，仅抓首页

	// OnPageReady 可选回调：仅当 go-rod 渲染成功、导航完成、WaitLoad 之后、
	// page.Close() 之前调用一次，把 *rod.Page 原样交给调用方。
	//
	// crawler 包完全不关心回调里做了什么——WebScanner 传入截图/favicon 采集逻辑，
	// ApiScanner 现在传 nil，未来若要做"动态渲染后捕获真实 XHR/Fetch"，
	// 直接在回调里挂 CDP page.EachEvent 网络监听即可，不需要改这里的函数签名。
	//
	// 边界约束：图片/二进制类的展示衍生品永远只通过这个回调向外流动，
	// 不进入 HomePage 返回值结构，这条边界不能松动。
	OnPageReady func(page *rod.Page)
}

// HomePage 是 FetchAndCrawl 的首页产出，只包含"数据"，不包含任何图片字段
type HomePage struct {
	URL           string
	StatusCode    int
	Title         string
	Body          string
	Headers       map[string]string
	ContentLength int64
	RichContext   map[string]interface{}
	SeedLinks     []string
	RemoteIP      string
	RemotePort    int
}

// FetchAndCrawl 统一入口：拿首页 + （可选）BFS 深度爬取子页面。
// WebScanner 和 ApiScanner 都只需要认识这一个函数。
func FetchAndCrawl(ctx context.Context, target, port, protocolHint string,
	limiter *qos.AdaptiveLimiter, opts FetchOptions) (home *HomePage, subPages []*Page, err error)
```

**内部实现要点（从 `web_scanner.go` 迁移过来，逻辑不变）**：
1. `normalizeURL`/`isProtocolGuessed` 协议判断
2. go-rod 路径：`Launch → OpenPage → EachEvent 监听响应 → Navigate → WaitLoad → ExtractRichContext`，成功后调用 `opts.OnPageReady(page)`（若非 nil），再 `page.Close()`
3. 触发条件不变：`homeBody == "" || (protocolGuessed && statusCode == 400)` 时，发起 `fallbackFetchBestProtocol` 双发对比，`pickBestFetchOutcome` 选优（`web_scanner.go` 322-359 行原样迁移，逻辑零改动）
4. `opts.CrawlDepth > 0` 时调用包内已有的 `Crawler.Crawl` 做 BFS

### 3.3 `WebScanner` 改造后的调用方式

```go
home, subPages, err := crawler.FetchAndCrawl(ctx, task.Target, port, protocolHint, s.limiter,
	crawler.FetchOptions{
		CrawlDepth: depth,
		OnPageReady: func(page *rod.Page) {
			if capture && !isCDN {
				if buf, errShot := page.Screenshot(true, nil); errShot == nil {
					screenshotB64 = base64.StdEncoding.EncodeToString(buf)
				}
			}
			faviconB64 = extractFaviconFromPage(page, richCtx)
		},
	})
```

`WebScanner` 拿到 `home`/`subPages` 之后，只做指纹识别（`fpEngine.Match`）、CDN 边缘组件组装、`buildWebResult` 收口，不再关心页面是怎么抓到的。

### 3.4 `ApiScanner` 设计

```go
// core/scanner/api/api_scanner.go
type ApiScanner struct {
	limiter *qos.AdaptiveLimiter
}

func (s *ApiScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	home, subPages, err := crawler.FetchAndCrawl(ctx, task.Target, port, protocolHint, s.limiter,
		crawler.FetchOptions{CrawlDepth: depth, OnPageReady: nil})
	...
	apis, truncated := crawler.ExtractPageAPIs(ctx, home.URL, home.Body, httpClient, s.limiter, maxFiles)
	// 对 subPages 同样调用 ExtractPageAPIs，合并去重
	...
}
```

`ApiScanner` 天然比 `WebScanner` 轻：不持有 `browserLauncher`（`FetchAndCrawl` 内部按需使用 go-rod，`ApiScanner` 不需要关心）、不持有指纹引擎、不做 CDN 展示判断（CDN 跳过深度爬取的判断收敛在 `FetchAndCrawl` 内部统一处理一次，两个 Scanner 都受益，不用各自实现一遍）。

`Params["source"]` 预留输入路径扩展位（本次只实现 `"crawl"`，即从抓取网页产出提取；`"urls"`/`"openapi"` 等直接吃 URL 列表或文档的路径留待后续阶段，Task 结构不需要为此现在改动，`Params` 本身就是 `map[string]interface{}`，新增 key 不破坏现有协议）。

### 3.5 Master 映射与 TaskType 改动

```go
// model/task.go 新增
TaskTypeApiScan TaskType = "api_scan" // API 扫描（JS 接口提取等），独立于 web_scan

// task_to_core.go 改动
case "api_scan", "apiScan":
    coreTask.Type = model.TaskTypeApiScan   // 不再借道 web_scan
    // mode:"api" 参数随之废弃。注意：不需要新增 js_api 这样的"要不要提取"
    // 开关——api_scan 本身就是独立 TaskType，被调度到即代表用户已显式表达
    // 意图，Params 只需要传 crawl/crawl_depth/max_files 等真正影响执行方式
    // 的参数，具体见 Web-JS接口提取实施文档.md 第二节
```

`docs/Master-Agent扫描类型映射说明.md` 表格第 23 行同步更新：`apiScan → api_scan`（而非 `web_scan`）。

---

## 四、破坏性分析

| 改动点 | 影响面 | 破坏性 |
|---|---|---|
| `crawler` 包整体迁移路径 | 所有 import 该包的文件（`web_scanner.go`、`web_scanner_escalate_test.go`） | 无，纯路径重写，编译器保证不遗漏 |
| `WebScanner` 内部重构（抓取委托给 `crawler.FetchAndCrawl`） | `WebScanner` 自身 | 无，对外行为（CLI/Master 的 `web_scan` 输入输出）不变，属于内部实现重构，需要现有测试（`web_scanner_escalate_test.go` 等）全部通过来验证零回归 |
| 新增 `TaskTypeApiScan` | `model` 包新增常量 | 无，纯增量 |
| `task_to_core.go` 的 `api_scan` case 改变目标 TaskType | Master 若已经在下发 `apiScan` 任务 | **需要评估**：若线上 Master 已有依赖"`apiScan` 最终落到 `web_scan` 执行"这一行为的逻辑（如结果解析按 `WebResult` 结构处理），改为独立 `ApiScanner` 后结果结构变化，Master 侧需要同步适配。当前结论是 Master 侧暂不改（见五、待办），Agent 侧先双轨支持：`RunnerManager` 同时注册 `TaskTypeWebScan` 和 `TaskTypeApiScan` 两个 Runner，互不影响 |
| `docs/Master-Agent扫描类型映射说明.md` 表格更新 | 文档 | 无，文档修正 |

**结论**：Agent 侧改动风险可控，核心是"内部重构 + 新增能力"，不删除、不改变任何现有 `web_scan` 对外行为。真正的风险点在 Master-Agent 协议层面的新旧 `apiScan` 语义切换时机，需要与 Master 侧协调发布顺序（见第五节）。

---

## 五、明确的分工与待办

### 5.1 本次范围（Agent 侧）

1. `crawler` 包迁移到 `core/lib/crawler`，新增 `fetch.go`
2. `WebScanner` 重构为调用 `crawler.FetchAndCrawl`，现有测试零回归
3. 新增 `ApiScanner`（`source:"crawl"` 路径），完成 Factory/Runner 注册/CLI/`options` 六件套
4. `model.TaskTypeApiScan` 新增
5. `task_to_core.go` 新增 `TaskTypeApiScan` 独立 case
6. `docs/Master-Agent扫描类型映射说明.md` 同步更新

### 5.2 Master 侧待办（本次不动代码，先记录）

- `neoMaster` 仓库需要新增 `AgentScanTypeApiScan` 对应的独立下发协议（目前 `apiScan` 业务场景在 Master 侧编译逻辑里最终生成的 `TaskType` 字符串需要从 `"web_scan"`/`mode:"api"` 切换为 `"api_scan"`）
- 发布顺序建议：Agent 侧先完成双轨支持（新旧两种 `apiScan` 映射方式都能跑通）并上线，确认稳定后，Master 侧再切换下发的 `TaskType` 字符串，避免"Agent 还没升级、Master 已经改了下发协议"导致任务失败
- Master 侧结果解析逻辑如果对 `apiScan` 任务的返回结构有隐含假设（按 `WebResult` 解析），需要同步新增对 `ApiResult`（`ApiScanner` 产出的新结果类型）的解析支持

### 5.3 阶段二（不在本次范围内，后续独立立项）

结合本方案讨论过程中确认的"深入优化"意图，以下能力基于本次搭好的 `ApiScanner` 骨架，后续按需独立展开，不在本次讨论：

1. **提取质量升级**：`OnPageReady` 回调里挂 CDP 网络监听，捕获运行时真实发出的 XHR/Fetch 请求（而非当前的静态正则提取），弥补正则方案对动态拼接 URL 的盲区
2. **接口存活验证**：对提取出的 API 端点发起主动探测，确认真实存在、判断是否需要鉴权
3. **外部输入支持**：`Params["source"]="urls"`（直接给 URL 列表）或 `"openapi"`（直接喂 Swagger/OpenAPI 文档），不需要先爬网页
4. **API 资产管理**：端点归类、去重、与已知接口文档比对，生成独立的 API 资产清单
5. **批量/持续监控**：对一批目标定期扫描，跟踪 API 端点变化

---

## 六、明确不做的事情（边界）

- ❌ 不在 `crawler.FetchAndCrawl` 的返回结构里加任何图片/二进制字段——展示类衍生品永远只通过 `OnPageReady` 回调向外流动，避免"通用数据结构该不该加个 XXX 字段"的争论无限蔓延
- ❌ 不现在就实现阶段二列出的任何一项能力——本次只搭骨架，不预先实现未确认需求，避免过度设计
- ❌ 不在本次改动 `neoMaster` 仓库代码——协议升级需要跨仓库协调发布顺序，本次只在文档里记录待办
- ❌ 不给 `ApiScanner` 现在就接入 go-rod 动态渲染逻辑——`OnPageReady` 接口已预留扩展点，但具体的 CDP 网络监听实现留到阶段二真正需要时再做

---

## 七、待对齐的细节问题

> 以下问题在转入实施文档前需逐项确认。

### 7.1 `crawler` 包迁移后是否需要重命名

`core/lib/crawler` 与 `core/lib/browser`、`core/lib/network` 平级，包名 `crawler` 本身语义清晰（现有 `browser`/`network` 也是描述性命名），暂定不改包名，仅改目录路径。

### 7.2 `ApiScanner` 是否需要独立的限流器实例，还是与 `WebScanner` 共享

`WebScanner` 现有限流器 `qos.NewAdaptiveLimiter(5, 1, 10)` 语义是"Web 扫描非常耗资源"（含浏览器启动），`ApiScanner` 若 `source:"crawl"` 路径同样会用到 go-rod（复用 `FetchAndCrawl`），资源消耗特征相近，初步倾向 `ApiScanner` 拥有自己独立的限流器实例（而非共享同一个 `*qos.AdaptiveLimiter`），避免两个 Scanner 的流量互相挤占对方的并发配额，具体默认参数在实施阶段确定。

### 7.3 `ApiResult` 结果结构与 `WebResult.APIs` 字段的关系（已确定）

《Web-JS接口提取实施文档.md》此前的设计是在 `WebResult` 上新增 `APIs`/`APIsTruncated` 字段（因为当时假设 JS API 提取挂在 `web_scan` 下）。本方案确定独立为 `ApiScanner` 后，**已在《Web-JS接口提取方案.md》v2.0 第三节与《Web-JS接口提取实施文档.md》v2.0 第三节完成落地**，结论如下，本节不再是待对齐问题，仅做结论摘要与交叉引用：

- 新增独立的 `model.ApiResult` 类型，**不复用** `WebResult` 的任何字段，两者是平级独立的结果结构：

```go
type ApiResult struct {
	URL           string        `json:"url"`
	Depth         int           `json:"depth"`
	APIs          []APIEndpoint `json:"apis,omitempty"`
	APIsTruncated bool          `json:"apis_truncated,omitempty"`
}
```

- `WebResult` 结构体本次**零改动**（不新增 `APIs`/`APIsTruncated` 字段），这是验证"两个结果类型彻底独立"这一架构目标是否落地的核心检查项（`git diff` 确认 `result_types.go` 里 `type WebResult struct` 部分无改动）；
- `APIEndpoint`（`URL`/`Method`/`Source`/`Confidence`）作为 `ApiResult.APIs` 的元素类型，与 `crawler` 包已有的 `links`/`Forms` 等页面导航语义完全分离，不进入 BFS 队列；
- `ApiResult` 从零实现 `TabularData` 接口（`Headers()`/`Rows()`），不是在 `WebResult` 现有表格列上追加，CLI 展示细节见《Web-JS接口提取实施文档.md》第七节。

（后续问题待补充）
