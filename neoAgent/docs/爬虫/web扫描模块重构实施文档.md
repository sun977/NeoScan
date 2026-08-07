# Web 扫描模块重构实施文档

> 文档版本：v1.0
> 修改时间：2026-08-07
> 依据方案：[`web扫描模块重构文档.md`](./web扫描模块重构文档.md)（本文档只负责把方案第三节"方案设计"拆成可以直接照做的步骤，不重复方案的背景论证与破坏性分析，结论有分歧以方案文档为准）
> 文档性质：实施清单，后续 `crawler` 包迁移、`WebScanner` 瘦身、`ApiScanner` 骨架搭建的代码开发按本文档的步骤顺序进行；每步给出改动文件、改动内容、验收标准
> 范围边界：本文档只搭 `ApiScanner` 的骨架（六件套打通，`Run` 内部调用 `FetchAndCrawl` + 组装空的 `ApiResult`），**不包含** JS API 提取本身的逻辑（`jsapi.go`/`ExtractPageAPIs`/三层提取规则等）——那部分是 [`../API扫描-js提取/Web-JS接口提取实施文档.md`](../API扫描-js提取/Web-JS接口提取实施文档.md) 的范围，且明确依赖本文档先落地完成（该文档"零、实施前必读"与"一、实施步骤总览"的前置前提）。

---

## 零、实施前必读的三条硬约束

来自方案文档第三、四、六节，写代码前必须遵守，不是建议：

1. **`WebScanner` 对外行为零回归**：`crawler.FetchAndCrawl` 只是把 `runOnePort` 里原有的抓取代码原样搬迁到新包、包一层新函数签名，**不允许在搬迁过程中顺手"优化"任何一行现有逻辑**（比如调整超时时间、调整 fallback 触发条件）。搬迁与优化是两件事，混在一起会让"这次改动到底动了什么"无法审查，且一旦出现回归，无法定位是搬迁引入的还是优化引入的。现有测试（`web_scanner_cdn_e2e_test.go`、`web_scanner_multiport_test.go`、`web_scanner_escalate_test.go` 等）必须在改动前后全部保持通过，这是验证"零回归"的唯一客观标准。
2. **图片/二进制展示衍生品永远只通过 `OnPageReady` 回调向外流动**：`crawler.HomePage`/`crawler.Page` 任何时候都不允许新增截图/favicon 相关字段（方案文档第六节明确列出的边界）。截图/favicon 采集逻辑原样保留在 `WebScanner` 侧，只是从"直接操作本地 `page` 变量"改成"通过回调参数操作传入的 `*rod.Page`"，这个改动不影响截图/favicon 本身的实现代码一行。
3. **依赖方向必须是彻底的单向链**：`core/scanner/web` 和 `core/scanner/api` 都只能是 `core/lib/crawler` 的调用方，`core/lib/crawler` 不能反向 import 任何一个 Scanner 包。每完成一步迁移，用 `go build ./...` 让编译器暴露所有需要改的 import 路径，不允许出现循环依赖（Go 编译器会直接报错，这是最省心的自检手段）。

---

## 一、实施步骤总览

严格按顺序执行，后一步依赖前一步的产出：

| 步骤 | 改动文件 | 类型 | 状态 |
|---|---|---|---|
| 1 | `internal/core/lib/crawler/`（`crawler.go`/`extract.go`/`leak.go`/对应 `_test.go`） | 迁移（从 `internal/core/scanner/web/crawler` 原样搬迁，含包内所有 import 路径更新） | ✅ 已完成 |
| 2 | `internal/core/lib/crawler/fetch.go` | 新建（`FetchAndCrawl`/`FetchOptions`/`HomePage`，从 `web_scanner.go` 抽取协议探测 + go-rod 渲染 + fallback 逻辑） | ✅ 已完成 |
| 3 | `internal/core/lib/crawler/fetch_test.go` | 新建（`FetchAndCrawl` 的单元/集成测试，覆盖搬迁后行为与搬迁前一致） | ⬜ 待开始 |
| 4 | `internal/core/scanner/web/web_scanner.go` | 修改（`runOnePort` 改为调用 `crawler.FetchAndCrawl`，移除已下沉的抓取代码；import 路径从 `scanner/web/crawler` 改为 `lib/crawler`） | ⬜ 待开始 |
| 5 | `internal/core/model/task.go` | 修改（新增 `TaskTypeApiScan` 常量） | ⬜ 待开始 |
| 6 | `internal/core/scanner/api/api_scanner.go` | 新建（`ApiScanner` 骨架：`Name()`/`Run()`，`Run` 内部调用 `crawler.FetchAndCrawl`，暂时组装空的 `model.ApiResult`，JS 提取逻辑留给后续实施文档填充） | ⬜ 待开始 |
| 7 | `internal/core/model/result_types.go` | 修改（新增空壳 `ApiResult` 类型，只含 `URL`/`Depth` 两个定位字段，`APIs`/`APIsTruncated` 留给 JS 提取实施文档补充） | ⬜ 待开始 |
| 8 | `internal/core/factory/api_factory.go` | 新建（`NewApiScanner`，模式对齐 `web_factory.go`） | ⬜ 待开始 |
| 9 | `internal/core/runner/manager.go` | 修改（注册 `factory.NewApiScanner()`） | ⬜ 待开始 |
| 10 | `internal/core/options/scan_api.go` | 新建（`ApiScanOptions`：`Target`/`Ports`/`Crawl`/`CrawlDepth`，暂不含 `MaxFiles`，留给 JS 提取实施文档补充） | ⬜ 待开始 |
| 11 | `cmd/agent/scan/api.go` | 新建（`api_scan` CLI 子命令骨架，帮助文本先占位，风险提示文案留给 JS 提取实施文档补充） | ⬜ 待开始 |
| 12 | `cmd/agent/scan/root.go` | 修改（挂载 `NewApiScanCmd()`） | ⬜ 待开始 |
| 13 | `internal/service/adapter/task_to_core.go` | 修改（`api_scan`/`apiScan` case 改为映射到 `model.TaskTypeApiScan`，`mode:"api"` 参数废弃） | ⬜ 待开始 |
| 14 | `docs/Master-Agent扫描类型映射说明.md` | 修改（`apiScan` 映射行从 `web_scan` 改为 `api_scan`） | ⬜ 待开始 |
| 15 | `docs/Agent指令集规范.md` | 修改（`api_scan` 独立成新章节，不再挂在 `web_scan` 参数表里；移除"Master 下发的 `api_scan` 也会被翻译成同一个 TaskType"这句过时描述） | ⬜ 待开始 |
| 16 | 端到端验收 | 新增/运行回归测试，人工核对破坏性分析清单 | ⬜ 待开始 |

**不在本文档范围内**（留给 [`Web-JS接口提取实施文档.md`](../API扫描-js提取/Web-JS接口提取实施文档.md)）：`jsapi.go`、`ExtractPageAPIs`、`ApiResult.APIs`/`APIsTruncated` 字段、`ApiScanOptions.MaxFiles`、`api_scan` 命令的风险提示文案、`ApiResult` 的 `TabularData` 实现。本文档步骤 6/7/10/11 只搭空壳，字段和函数签名会在后续实施文档里被**追加**（不是修改），两份文档的改动不冲突。

---

## 二、步骤 1：`crawler` 包整体迁移

### 2.1 目标路径

```
internal/core/scanner/web/crawler/   →   internal/core/lib/crawler/
    crawler.go / crawler_test.go
    extract.go / extract_test.go
    leak.go / leak_test.go
    escalation_test.go
    README.md
```

包名 `crawler` 不变（方案文档 7.1 节已确认，与 `core/lib/browser`、`core/lib/network` 平级），只搬目录、不改包内代码逻辑。

### 2.2 操作步骤

1. 用文件系统操作（`git mv` 或直接移动）把上述文件整体从 `internal/core/scanner/web/crawler/` 移到 `internal/core/lib/crawler/`，保持文件名、包内代码原封不动。
2. 全仓库搜索 `neoagent/internal/core/scanner/web/crawler` 这个 import 路径字符串，逐个替换为 `neoagent/internal/core/lib/crawler`。已知需要改的文件：
   - `internal/core/scanner/web/web_scanner.go`（第 22 行 import）
   - 该包自身测试文件内部如果有互相引用绝对路径的地方（一般没有，包内文件用相对同包引用，不需要改 import）
3. `go build ./...`，编译器会把所有遗漏的 import 路径暴露出来（这是方案文档 2.3 节"低风险搬迁"结论的验证方式——`crawler` 包对外依赖只有 `qos` 和 `model`，无反向依赖，编译器足以兜底）。

### 2.3 验收标准

- `go build ./...` 通过，无残留的旧路径引用。
- `go vet ./...` 无新增警告。
- `go test ./internal/core/lib/crawler/... -v` 全部通过（迁移后原有测试用例应该原样通过，因为代码逻辑未改动）。
- 用 `git diff` 确认迁移后的 `crawler.go`/`extract.go`/`leak.go` 文件内容与迁移前逐字节一致（除了文件路径本身），这是本步骤"只搬不改"的核心检查项。
- `internal/core/scanner/web/crawler/` 目录已删除，不留旧目录和旧文件（避免出现两份同名包造成混淆）。

---

## 三、步骤 2：新建 `fetch.go`——`FetchAndCrawl` 统一抓取入口

### 3.1 为什么单独新建文件，不放进迁移过来的 `crawler.go`

`crawler.go` 现有职责是"BFS 深度爬取"（`Crawler`/`Crawl`/`worker`/`fetchAndExtract`），这是一个自成一体、已经过充分测试的模块。`FetchAndCrawl` 要做的是一件不同的事——"拿首页"（协议探测 + go-rod 渲染 + fallback 双发选优），这段逻辑目前散落在 `web_scanner.go` 的 `runOnePort` 里，与 `WebScanner` 的截图/CDN 关注点混在一起。单独建 `fetch.go` 承接这部分下沉逻辑，与 `crawler.go` 的 BFS 逻辑保持文件级别的关注点分离，`FetchAndCrawl` 内部在需要深度爬取时才调用 `crawler.go` 已有的 `New`/`Crawl`，两个文件是"上层编排调用下层能力"的关系，不是平级重复。

### 3.2 从 `web_scanner.go` 迁移的函数清单

以下函数/逻辑段从 `web_scanner.go` 原样搬到 `fetch.go`，**搬迁时只做"抽取成独立函数 + 调整签名以接受显式参数"这两件事，不改动内部任何一行业务逻辑**：

| 原位置（`web_scanner.go`） | 原函数/逻辑段 | 搬迁后 |
|---|---|---|
| 第 490~520 行 | `normalizeURL` | 原样迁移为 `crawler.normalizeURL`（包内私有，`FetchAndCrawl` 内部调用） |
| 第 531~536 行 | `isProtocolGuessed` | 原样迁移为 `crawler.isProtocolGuessed` |
| 第 226~296 行 | go-rod 路径：`Launch → OpenPage → EachEvent 监听 → Navigate → WaitLoad → ExtractRichContext` | 迁移进 `FetchAndCrawl` 内部，`ExtractRichContext`/`ExtractLinks` 两个函数原本定义在 `web` 包的 `context.go`（需要确认实际文件名与位置，若在 `web` 包内则需要一并迁移或改为 `crawler` 包导出复用，避免 `crawler` 反向依赖 `web`——对照零、第 3 条硬约束） |
| 第 279~286 行（截图/favicon 采集） | 截图与 `extractFaviconFromPage` 调用 | **不迁移**，改为通过 `opts.OnPageReady(page)` 回调交还给 `WebScanner`（方案文档 3.2 节的核心设计） |
| 第 298~359 行 | 统一降级 + 协议误判纠正判断（`needsVerification`/`fallbackFetchBestProtocol`/`pickBestFetchOutcome`） | 原样迁移进 `FetchAndCrawl` |
| 第 638~789 行 | `fallbackFetch`/`fallbackFetchBestProtocol`/`fetchOutcome`/`pickBestFetchOutcome`/`flipProtocol` | 原样迁移为 `crawler` 包内私有函数 |
| 第 799~813 行 | `extractTitleFromHTML` | 原样迁移（`crawler.go` 内已有同职责的 `extractTitle`，需要核对两者是否本质重复——若重复，`fetch.go` 直接复用 `crawler.go` 已有的 `extractTitle`，不新增第二个同职责函数，这是实施阶段发现的一个"顺手消除重复"的机会，不算范围蔓延） |
| 第 384~398 行 | 深度爬取触发判断（`depth := s.resolveCrawlDepth(...)`；`cr := crawler.New(...); pages := cr.Crawl(...)`） | `resolveCrawlDepth` 依赖 `task.Params`，属于业务参数解析，保留在 `WebScanner`；`FetchAndCrawl` 只接受解析好的 `CrawlDepth int` 参数，内部调用 `crawler.New`/`Crawl`（同包内直接调用，不需要跨包） |

**保留在 `WebScanner` 侧、不迁移的逻辑**（对照方案文档 2.4 节的职责区分）：

- CDN 判断（`checkCDN`）——这是"要不要跳过某个动作"的业务决策，不是"怎么抓页面"的机制，`isCDN` 结果通过 `WebScanner` 自己的判断来决定要不要在 `OnPageReady` 回调里执行截图；
- 指纹匹配（`fpEngine.Match`）、`buildWebResult` 结果组装；
- `escalateIfNeeded`/`renderWithBrowser`（按需升级渲染）——这段逻辑目前依赖 `crawler.Page`/`crawler.Crawler` 已有的导出方法（`EnqueueExtra`），迁移到 `lib/crawler` 后这些类型和方法本身随第二节整体迁移过去，但升级触发的编排逻辑（"哪些页面需要升级、升级后回填到哪里"）是 `WebScanner` 的业务判断，保留在 `web_scanner.go`。

### 3.3 `fetch.go` 完整接口设计

```go
// internal/core/lib/crawler/fetch.go
package crawler

import (
	"context"

	"neoagent/internal/core/lib/network/qos"

	"github.com/go-rod/rod"
)

// FetchOptions 控制 FetchAndCrawl 的抓取行为。
type FetchOptions struct {
	ProtocolHint string // 复用现有 protocolHint 语义：""(自动猜)/"http"/"https"
	CrawlDepth   int    // <=0 表示不做 BFS 深度爬取，仅抓首页

	// OnPageReady 可选回调：仅当 go-rod 渲染成功、导航完成、WaitLoad 之后、
	// page.Close() 之前调用一次，把 *rod.Page 原样交给调用方。
	//
	// crawler 包完全不关心回调里做了什么——WebScanner 传入截图/favicon 采集逻辑，
	// ApiScanner 现在传 nil，未来若要做"动态渲染后捕获真实 XHR/Fetch"，
	// 直接在回调里挂 CDP page.EachEvent 网络监听即可，不需要改这里的函数签名。
	//
	// 边界约束（零、第 2 条硬约束）：图片/二进制类的展示衍生品永远只通过这个
	// 回调向外流动，不进入 HomePage 返回值结构，这条边界不能松动。
	OnPageReady func(page *rod.Page)
}

// HomePage 是 FetchAndCrawl 的首页产出，只包含"数据"，不包含任何图片字段。
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
// WebScanner 和 ApiScanner 都只需要认识这一个函数，具体实现是
// web_scanner.go 原 runOnePort 里协议探测/go-rod 渲染/fallback 逻辑的
// 原样搬迁，见本文档第三节的迁移清单，逻辑不变。
func FetchAndCrawl(ctx context.Context, target, port, protocolHint string,
	limiter *qos.AdaptiveLimiter, opts FetchOptions) (home *HomePage, subPages []*Page, err error) {
	// 实现步骤（对照迁移清单第 3.2 节逐段填入）：
	// 1. protocolGuessed := isProtocolGuessed(target, protocolHint)
	//    targetURL := normalizeURL(target, port, protocolHint)
	// 2. go-rod 路径：Launch → OpenPage → EachEvent 监听 → Navigate → WaitLoad
	//    → ExtractRichContext，成功后调用 opts.OnPageReady(page)（若非 nil），
	//    再 page.Close()（注意：这里需要 browser.BrowserLauncher，FetchAndCrawl
	//    需要接受一个 launcher 参数或在内部持有默认实现，见 3.4 节的讨论）
	// 3. needsVerification 判断 + fallbackFetchBestProtocol 双发选优
	// 4. opts.CrawlDepth > 0 时调用本包已有的 Crawler.Crawl 做 BFS
	panic("TODO: 按第 3.2 节迁移清单填充，禁止提前 return 一个假的空实现")
}
```

### 3.4 待实施阶段确认的技术细节：`browser.BrowserLauncher` 归属

`web_scanner.go` 现有的 go-rod 渲染依赖 `s.browserLauncher`（`*browser.BrowserLauncher` 实例，由 `WebScanner` 持有）。`FetchAndCrawl` 下沉后必须能访问同样的浏览器启动能力，两种可行方案二选一（实施时确定，不影响本文档其余步骤）：

- **方案 A（推荐）**：`FetchAndCrawl` 新增一个 `launcher *browser.BrowserLauncher` 参数，由调用方（`WebScanner`/`ApiScanner`）各自持有并传入。理由：`browser.BrowserLauncher` 本身是无状态的启动器封装，`WebScanner`/`ApiScanner` 各自持有一份实例的成本很低（不像 `qos.AdaptiveLimiter` 那样需要维护跨调用的限流状态），显式传参符合"`crawler` 包不偷偷读取全局单例"的一贯风格。
- **方案 B**：`lib/crawler` 包内部自己 `import browser` 并持有一个包级单例。缺点是引入了 `lib/crawler → lib/browser` 的隐式依赖，且如果未来 `WebScanner`/`ApiScanner` 需要各自定制 `BrowserLauncher` 的参数（比如不同的浏览器启动超时），包级单例做不到差异化配置。

方案设计倾向 A，实施时按 A 编码；若实施过程中发现 A 有未预见的阻碍（如 `browser` 包的初始化强依赖 `web` 包内部状态），退回方案 B 并在本文档此处补充记录变更原因。

### 3.5 `Page` 类型不需要任何改动

`Page`（`crawler.go` 第 35~47 行）已经是 BFS 子页面的完整数据结构（`URL`/`Depth`/`StatusCode`/`Body`/`Headers`/`Forms`/`Params`/`Leaks`/`NeedsEscalation`），`FetchAndCrawl` 的 `subPages []*Page` 直接复用这个已有类型，不新增字段。

### 3.6.1 实施落地记录（与 3.3 节接口设计的实际出入）

以下两点是编码时确认的技术细节，记录在此以便步骤 3/4 对齐，不影响 3.1/3.2 节的结论：

1. **`FetchAndCrawl` 最终签名比 3.3 节草稿多一个 `launcher *browser.BrowserLauncher` 参数**：这是 3.4 节"待实施阶段确认"的技术细节，实施时按方案 A 落地——`launcher` 由调用方显式传入，`crawler` 包不反向 import 并持有单例。3.4 节的讨论到此可视为已确认，不需要在实施阶段之外单独决策。
2. **`context.go`（`ExtractRichContext`/`ExtractLinks`）随 `fetch.go` 一并从 `web` 包迁移到 `lib/crawler` 包**：3.2 节迁移清单里已经预见到这个问题（"若在 `web` 包内则需要一并迁移……避免 `crawler` 反向依赖 `web`"），实施时确认了这两个函数确实在 `web` 包内，因此用 `git mv` 把 `context.go` 整体迁移过去，包名从 `web` 改为 `crawler`，函数签名不变。`web_scanner.go` 里仍保留在原地、尚未下沉的 `renderWithBrowser`（升级渲染逻辑，5.4 节明确保留在 `WebScanner` 侧）相应地改为调用 `crawler.ExtractRichContext`/`crawler.ExtractLinks`。
3. **`HomePage` 未包含 Forms/Params 字段**：与 3.3 节接口设计一致——Forms/Params 提取（`ExtractLinksAndForms`）保留由调用方在拿到 `HomePage.Body` 后自行统一调用，`FetchAndCrawl` 不重复做这件事。
4. **`extractTitleFromHTML` 与 `crawler.go` 已有的 `extractTitle` 重复**：3.2 节已经预判到这个重复，`fetch.go` 内的新 `fallbackFetch` 直接复用 `extractTitle`。`web_scanner.go` 里旧的 `extractTitleFromHTML` 函数本身连同它所属的旧 `fallbackFetch`/`fallbackFetchBestProtocol` 等整段代码，会在步骤 4（`runOnePort` 改为调用 `FetchAndCrawl`）一并删除，本步骤不提前动它——这部分代码目前仍是 `web_scanner.go` 现有测试路径依赖的活代码，提前删除会破坏零、第 1 条硬约束"搬迁与优化是两件事"。

### 3.6 验收标准

- `go build ./internal/core/lib/crawler/...` 通过。
- `go vet ./...` 无新增警告。
- `fetch_test.go`（步骤 3）验证 `FetchAndCrawl` 在以下场景的行为与迁移前 `web_scanner.go` 的 `runOnePort` 完全一致：
  - go-rod 渲染成功，`OnPageReady` 回调被调用且传入的 `*rod.Page` 可用（可以在回调里调用 `page.Screenshot` 验证不 panic）；
  - go-rod 渲染失败（模拟 `Launch`/`Navigate` 失败），触发 fallback 双发选优，`home.Body` 非空；
  - `protocolGuessed=true` 且拿到 400 响应，触发双发验证；
  - `CrawlDepth<=0` 时 `subPages` 为空切片/`nil`，不发起任何 BFS 请求；
  - `CrawlDepth>0` 且首页有 `SeedLinks` 时，`subPages` 非空。

---

## 四、步骤 3：`fetch_test.go`

参照 `web_scanner_cdn_e2e_test.go`/`web_scanner_multiport_test.go` 现有的 `httptest.Server` 测试风格，覆盖第 3.6 节列出的场景。测试重点不是"重新验证 go-rod/fallback 本身的正确性"（这部分逻辑零改动，已经被 `web_scanner.go` 现有测试覆盖过），而是验证"迁移后的函数签名和调用方式是否正确传导了各项参数"，即：

- `OnPageReady` 回调确实在正确的时机（`WaitLoad` 之后、`page.Close()` 之前）被调用了恰好一次；
- `protocolHint`/`port` 参数确实影响了 `normalizeURL` 的行为（可以复用 `normalizeURL` 现有单元测试用例，只是调用方式从直接调用私有函数改为通过 `FetchAndCrawl` 间接验证）；
- `limiter` 参数确实被 `Acquire`/`Release`（用一个自定义的 mock limiter 或复用 `qos.AdaptiveLimiter` 观察其内部计数变化）。

### 验收标准

- `go test ./internal/core/lib/crawler/... -run TestFetchAndCrawl -v` 全部通过。
- `go test ./internal/core/lib/crawler/... -race` 通过。

---

## 五、步骤 4：`web_scanner.go` 改造为调用 `FetchAndCrawl`

### 5.1 改造后的 `runOnePort` 骨架

`runOnePort`（现状第 184~402 行）改造后应该显著变短，因为协议探测/go-rod 渲染/fallback 选优这三大段（约 150 行）整体委托给 `FetchAndCrawl`：

```go
func (s *WebScanner) runOnePort(ctx context.Context, task *model.Task, startTime time.Time, port string, protocolHint string) (results []*model.TaskResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[WebScanner] PANIC RECOVERED: %v", r)
			err = fmt.Errorf("panic during web scan: %v", r)
			results = nil
		}
	}()

	// CDN 判断：发起抓取之前，逻辑不变（checkCDN 保留在 WebScanner 侧，
	// 见方案文档第六节"明确不做的事情"之外的既有设计，不受本次重构影响）
	isCDN, cdnProvider := s.checkCDN(task.Target)

	var screenshotB64, faviconB64 string
	depth := 0 // 先用一个占位值，真正的 depth 在拿到 homeStatusCode/homeHeaders/seedLinks 之后才能计算，见下方说明

	home, subPages, errFetch := crawler.FetchAndCrawl(ctx, task.Target, port, protocolHint, s.limiter,
		crawler.FetchOptions{
			CrawlDepth: -1, // 占位：depth 依赖 home 的返回值才能算出来，无法在调用前算好，见 5.2 节的解决方案
			OnPageReady: func(page *rod.Page) {
				if capture, ok := task.Params["screenshot"].(bool); ok && capture && !isCDN {
					if buf, errShot := page.Screenshot(true, nil); errShot == nil {
						screenshotB64 = base64.StdEncoding.EncodeToString(buf)
					} else {
						logger.Warnf("[WebScanner] Screenshot failed: %v", errShot)
					}
				}
				faviconB64 = extractFaviconFromPage(page, /* richCtx 从哪来见 5.2 节 */ nil)
			},
		})
	if errFetch != nil {
		s.limiter.OnFailure()
		return nil, errFetch
	}
	s.limiter.OnSuccess()

	// ...后续：resolveIPPortForResult、ExtractLinksAndForms、组装 edgeComponents、
	// buildWebResult、escalateIfNeeded，这部分逻辑完全不受本次重构影响，原样保留...
}
```

### 5.2 关键设计难点：`CrawlDepth` 依赖首页响应结果，无法在调用 `FetchAndCrawl` 之前算出来

这是实施阶段必须解决、方案文档未展开到这个细节的真实矛盾：`resolveCrawlDepth`（现状第 947~962 行）的自动判断分支需要 `homeStatusCode`/`headers`/`seedLinks` 三个只有拿到首页响应之后才知道的值，但 `FetchAndCrawl` 的参数 `FetchOptions.CrawlDepth` 必须在调用前传入，形成"先有鸡还是先有蛋"的依赖顺序问题。两种可行方案，实施时二选一：

- **方案 A（推荐）**：`FetchAndCrawl` 内部把"抓首页"和"BFS 深度爬取"两个阶段解耦成可以分两次调用的形态——一次只抓首页（`CrawlDepth` 传 0 或负数），调用方拿到 `home` 之后自己算出真正的 `depth`，如果 `depth>0` 再调用一次（或者调用 `crawler` 包内独立导出的 BFS 触发函数，不必重新走一遍首页抓取）来触发 BFS。这样 `FetchAndCrawl` 保持"一次调用只做一件确定的事"的单一职责，`WebScanner` 的业务判断（自动探测深度）自然地发生在两次调用之间。
- **方案 B**：`FetchAndCrawl` 的 `FetchOptions` 增加一个可选的 `ResolveCrawlDepth func(home *HomePage) int` 回调字段，内部拿到首页数据后先调用这个回调算出真正的深度，再决定要不要触发 BFS。缺点是 `crawler` 包因此需要认识"深度要怎么算"这件事的存在（哪怕只是通过回调），比方案 A 更早地把业务判断的钩子函数签名嵌入进通用工具包的接口设计里，增加了 `FetchOptions` 的复杂度。

方案设计倾向 A（更符合"`crawler` 包只做机制、不掺和策略"的既定分层原则），实施时先按 A 编码，遇到明显阻碍再考虑 B。**这是本文档相对方案文档新增的一处技术细节澄清，需要在真正写代码前与方案文档的框架保持一致性检查**——如果方案 A 的"分两次调用"设计与 `FetchAndCrawl` 现有签名冲突，应该先回到方案文档（而不是本实施文档）更新第 3.2 节的接口设计，再回来同步修改本节。

### 5.3 `RichContext`/截图/favicon 的数据流调整

`OnPageReady` 回调运行在 `FetchAndCrawl` 内部，此时 `richCtx`（`ExtractRichContext` 的产出）也是在 `FetchAndCrawl` 内部产生的局部变量，`extractFaviconFromPage(page, richCtx)` 需要 `richCtx` 才能工作。两种处理方式：

- 把 `favicon_url` 的提取责任也下沉进 `FetchAndCrawl`（作为 `HomePage.RichContext` 的一部分返回，`WebScanner` 从 `home.RichContext["favicon_url"]` 里取，而不是在回调闭包里直接引用 `richCtx` 局部变量）；
- 或者把 `richCtx` 作为 `OnPageReady` 回调的第二个参数一并传出：`OnPageReady func(page *rod.Page, richCtx map[string]interface{})`。

前者更干净（`HomePage.RichContext` 本来就该包含这类信息，`OnPageReady` 回调的职责保持"只给 `page` 对象，不额外传业务数据"的最小化原则），实施时优先采用前者。`favicon_url` 本身已经是 `ExtractRichContext` 现有产出的一部分（`richCtx["favicon_url"]`，见 `web_scanner.go` 第 463 行 `extractFaviconFromPage` 内部读取方式），下沉后只是读取时机从"回调闭包内直接访问局部变量"改为"从 `home.RichContext` 里取"，不涉及新的提取逻辑。

### 5.4 `WebScanner` 结构体与初始化：无需改动

`browserLauncher`/`fpEngine`/`edgeDetector`/`limiter` 四个字段全部保留（`FetchAndCrawl` 需要的 `limiter` 和可能需要的 `launcher` 由 `WebScanner` 继续持有并传入，参照 3.4 节方案 A）。`ensureInit`/`checkCDN`/`resolveIPPortForResult`/`resolveCrawlDepth`/`escalateIfNeeded`/`renderWithBrowser`/`buildWebResult`/`pageData` 全部原样保留在 `web_scanner.go`，一行不改（这些都是"业务判断"和"结果组装"，不属于本次下沉范围）。

### 5.5 验收标准

- `go build ./...` 通过。
- `go vet ./...` 无新增警告。
- **核心回归验收项**：`go test ./internal/core/scanner/web/... -v` 全量跑一遍，改造前后测试结果（通过的用例集合）必须完全一致——尤其是 `web_scanner_cdn_e2e_test.go` 的 4 个 CDN 用例、`web_scanner_multiport_test.go` 的多端口用例、`web_scanner_escalate_test.go` 的升级渲染用例，一个都不能变红。
- `web_scanner.go` 文件总行数应显著减少（预期减少 150~200 行，对应第 3.2 节迁移清单里搬走的代码量），这是"瘦身"这个重构目标是否真正达成的直观指标。
- 用真实目标（如内网测试站点 `10.201.28.126`）手工跑一次 `neoAgent scan web -t <target> --screenshot`，确认截图/favicon/指纹/CDN 判断行为与重构前完全一致。

---

## 六、步骤 5：`model.TaskTypeApiScan` 新增

### 6.1 改动位置

`internal/core/model/task.go`，紧邻 `TaskTypeWebScan`（第 23 行）之后新增：

```go
TaskTypeApiScan TaskType = "api_scan" // API 扫描（JS 接口提取等），独立于 web_scan，见 docs/爬虫/web扫描模块重构文档.md
```

### 6.2 验收标准

- `go build ./...` 通过。
- 只新增一行常量，不改动、不删除任何已有 `TaskType` 常量（对照 `00.原子扫描器开发指南[定稿].md` 第 2 节"新增前先确认类型是否已存在"的检查步骤，`api_scan` 确实是全新能力，新增合理）。

---

## 七、步骤 6：新建 `ApiScanner` 骨架

### 7.1 骨架范围说明

本步骤只搭"能跑通空转"的骨架：`ApiScanner.Run` 调用 `crawler.FetchAndCrawl` 拿到页面，组装一个**只有 `URL`/`Depth` 两个字段、`APIs` 永远为空**的 `model.ApiResult` 返回。真正的 JS 提取逻辑（`crawler.ExtractPageAPIs` 调用）由 [`Web-JS接口提取实施文档.md`](../API扫描-js提取/Web-JS接口提取实施文档.md) 第五节负责填充，那份文档的前置依赖就是本步骤先落地。

### 7.2 文件：`internal/core/scanner/api/api_scanner.go`

```go
// Package api 提供 API 扫描能力（当前阶段：从抓取到的页面中提取 JS 接口
// 调用地址，未来可扩展外部输入源，见 web扫描模块重构文档.md 5.3 节）。
package api

import (
	"context"
	"time"

	"neoagent/internal/core/lib/crawler"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
)

// ApiScanner 是独立于 WebScanner 的原子扫描器，与 WebScanner 平级，
// 共享同一个 crawler.FetchAndCrawl 抓取入口，自己只关心"拿到页面之后
// 做 API 相关的分析"，不重复实现任何抓取逻辑（web扫描模块重构文档.md 第一节）。
type ApiScanner struct {
	// limiter 是 ApiScanner 独立持有的限流器实例，不与 WebScanner 共享
	// （方案文档 7.2 节：两者资源消耗特征相近，但流量不应互相挤占对方的
	// 并发配额）。
	limiter *qos.AdaptiveLimiter
}

// NewApiScanner 创建一个 ApiScanner 实例。
func NewApiScanner() *ApiScanner {
	return &ApiScanner{
		limiter: qos.NewAdaptiveLimiter(5, 1, 10), // 与 WebScanner 相同的默认参数，具体数值后续可按需独立调整
	}
}

// Name 扫描器名称。
func (s *ApiScanner) Name() model.TaskType {
	return model.TaskTypeApiScan
}

// Run 执行 API 扫描任务：调用 crawler.FetchAndCrawl 拿首页 +（可选）深度
// 爬取子页面，对每个页面组装一条 model.ApiResult。
//
// 当前阶段（本文件对应的实施文档只搭骨架）：APIs 字段固定为空，真正的
// JS 提取逻辑由 Web-JS接口提取实施文档.md 补充实现，届时本函数体会被
// 修改（追加对 crawler.ExtractPageAPIs 的调用），函数签名保持不变。
func (s *ApiScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	crawlDepth, port, protocolHint := parseApiScanParams(task)

	home, subPages, err := crawler.FetchAndCrawl(ctx, task.Target, port, protocolHint, s.limiter,
		crawler.FetchOptions{
			CrawlDepth:  crawlDepth,
			OnPageReady: nil, // ApiScanner 不需要截图/favicon，回调传 nil（重构文档 3.2 节约束）
		})
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	var results []*model.TaskResult
	results = append(results, s.buildApiResult(task, startTime, home.URL, 0))
	for _, p := range subPages {
		results = append(results, s.buildApiResult(task, startTime, p.URL, p.Depth))
	}
	return results, nil
}

// buildApiResult 组装单个页面的空壳结果。首页（depth=0）和每一个深度
// 爬取子页面都调用这一个函数，不存在"首页走一条路径、子页面走另一条
// 路径"的分叉——这是为后续 JS 提取逻辑接入预留的统一收口点。
func (s *ApiScanner) buildApiResult(task *model.Task, startTime time.Time, pageURL string, depth int) *model.TaskResult {
	return &model.TaskResult{
		TaskID:      task.ID,
		Status:      model.TaskStatusSuccess,
		ExecutedAt:  startTime,
		CompletedAt: time.Now(),
		Result: &model.ApiResult{
			URL:   pageURL,
			Depth: depth,
		},
	}
}

// parseApiScanParams 从 task.Params/task.PortRange 解析 ApiScanner 需要的
// 最小参数集：crawlDepth/port/protocolHint。crawl/crawl_depth 的三态语义
// 直接复用与 WebScanner.resolveCrawlDepth 相同的判断规则（不重新发明），
// 具体实现在 JS 提取实施文档第五节补全为完整版本，本步骤先给出最小可用实现，
// 保证 ApiScanner 骨架能够独立编译运行。
func parseApiScanParams(task *model.Task) (crawlDepth int, port string, protocolHint string) {
	if p, ok := task.Params["protocol"].(string); ok {
		protocolHint = p
	}
	port = task.PortRange

	enableCrawl, explicit := task.Params["crawl"].(bool)
	switch {
	case explicit && !enableCrawl:
		return 0, port, protocolHint
	case explicit && enableCrawl:
		depth := 2
		if d, ok := task.Params["crawl_depth"].(int); ok && d > 0 {
			depth = d
		}
		return depth, port, protocolHint
	default:
		return 0, port, protocolHint // 骨架阶段默认不自动开启深度爬取，比 WebScanner 更保守（方案文档 3.4 节）
	}
}
```

### 7.3 验收标准

- `go build ./internal/core/scanner/api/...` 通过。
- `go vet ./...` 无新增警告。
- 手动构造一个简单 `httptest.Server` 页面，调用 `ApiScanner{}.Run`，验证：`results` 至少包含首页一条记录，`Depth==0`，`APIs` 为 `nil`（骨架阶段的预期行为，不是 bug）。

---

## 八、步骤 7：`model.ApiResult` 空壳类型

### 8.1 改动位置

`internal/core/model/result_types.go`，紧邻 `LeakInfo`（第 138 行附近）之后新增：

```go
// ApiResult 是 ApiScanner 的任务产出。与 WebResult 完全独立，不复用它的
// 任何字段——ApiScanner 不做指纹识别、不截图、不判断 CDN，这些 WebResult
// 字段对 ApiResult 没有意义。
//
// 当前阶段（本类型对应的实施文档只搭骨架）：只有 URL/Depth 两个定位字段。
// APIs/APIsTruncated 字段由 Web-JS接口提取实施文档.md 第三节追加，
// 届时这里会新增两个字段，不会修改/删除现有字段。
type ApiResult struct {
	URL   string `json:"url"`   // 本页面 URL（与 WebResult.URL 同义）
	Depth int    `json:"depth"` // BFS 深度，0 表示首页
}
```

不向 `WebResult` 追加任何字段（这是验证"两个结果类型彻底独立"这一架构目标是否落地的核心检查项，见方案文档 7.3 节结论）。

### 8.2 验收标准

- `go build ./internal/core/model/...` 通过。
- `git diff` 确认 `result_types.go` 里 `type WebResult struct` 部分无任何改动。
- 序列化一个 `ApiResult{URL: "http://x.com", Depth: 0}` 实例，确认输出 `{"url":"http://x.com","depth":0}`。

---

## 九、步骤 8：`internal/core/factory/api_factory.go`

对齐 `web_factory.go`（第 1~15 行）的模式：

```go
package factory

import (
	"neoagent/internal/core/scanner/api"
)

// NewApiScanner 创建一个标准的 API 扫描器。
// 如果未来有全局配置（如最大并发数、代理等），应在这里传入。
func NewApiScanner() *api.ApiScanner {
	return api.NewApiScanner()
}
```

### 验收标准

- `go build ./internal/core/factory/...` 通过。

---

## 十、步骤 9：`RunnerManager` 注册

`internal/core/runner/manager.go`，`NewRunnerManager()` 里紧邻 `WebScanner` 注册（第 46~51 行）之后新增：

```go
// 注册 ApiScanner
// 补齐 Master 集群下发 api_scan 任务的执行能力：task_to_core.go 中
// "api_scan"/"apiScan" 改为翻译为独立的 TaskTypeApiScan（不再借道
// TaskTypeWebScan），若不在此注册，RunnerManager.Get 会报
// "no runner found"，Master 侧任务必然失败。
as := factory.NewApiScanner()
m.Register(as)
```

### 验收标准

- `go build ./...` 通过。
- 按照 `00.原子扫描器开发指南[定稿].md` 第 3 节"检查清单"逐项自查：`task_to_core.go` 的 `api_scan` case 是否已经 `case` 到 `model.TaskTypeApiScan`（本步骤先注册 Runner，第十三节才改 `task_to_core.go`，两者顺序不影响最终正确性，但完整验收要等两步都做完）；`NewRunnerManager()` 是否已注册；两者 `TaskType` 字符串是否完全一致（`"api_scan"`，大小写、下划线一个字符都不能错）。

---

## 十一、步骤 10：`internal/core/options/scan_api.go`

### 11.1 骨架版本（不含 `MaxFiles`）

```go
package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// ApiScanOptions 骨架版本：只包含抓取相关的参数（Target/Ports/Crawl/CrawlDepth），
// 与 WebScanOptions 的对应字段语义完全一致，直接照搬，不重新设计。
// MaxFiles（单页最多下载的外链 JS 文件数）由 Web-JS接口提取实施文档.md
// 第二节追加，本步骤不包含。
type ApiScanOptions struct {
	Target     string
	Ports      string
	Crawl      string // "auto"(默认) / "true" / "false"，语义与 WebScanOptions.Crawl 一致
	CrawlDepth int
	Output     OutputOptions
}

func NewApiScanOptions() *ApiScanOptions {
	return &ApiScanOptions{
		Ports:      "80,443",
		Crawl:      "auto",
		CrawlDepth: 2,
	}
}

func (o *ApiScanOptions) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

func (o *ApiScanOptions) ToTask() *model.Task {
	task := model.NewTask(model.TaskTypeApiScan, o.Target)
	task.PortRange = o.Ports
	task.Timeout = 30 * time.Minute

	switch o.Crawl {
	case "true":
		task.Params["crawl"] = true
	case "false":
		task.Params["crawl"] = false
	}
	if o.CrawlDepth > 0 {
		task.Params["crawl_depth"] = o.CrawlDepth
	}

	o.Output.ApplyToParams(task.Params)
	return task
}
```

### 验收标准

- `go build ./internal/core/options/...` 通过。
- `ApiScanOptions{}.ToTask()` 产出的 `Task.Type` 必须是 `model.TaskTypeApiScan`。

---

## 十二、步骤 11：`cmd/agent/scan/api.go` CLI 命令骨架

```go
package scan

import (
	"context"
	"fmt"
	"time"

	"neoagent/internal/core/options"
	"neoagent/internal/core/reporter"
	"neoagent/internal/core/runner"

	"github.com/spf13/cobra"
)

// NewApiScanCmd 创建 api_scan 命令骨架。
//
// 风险提示文案（下载外链 JS 文件是主动网络行为）由 Web-JS接口提取实施
// 文档.md 第二节补充完整，本步骤先给出可编译运行的最小骨架，Short/Long
// 占位文本会在那份文档实施时被替换，不是最终文案。
func NewApiScanCmd() *cobra.Command {
	opts := options.NewApiScanOptions()

	cmd := &cobra.Command{
		Use:   "api",
		Short: "API 接口扫描（骨架阶段，JS 提取逻辑待实现）",
		Long:  "对目标页面（及可选的深度爬取子页面）提取接口调用地址。当前为骨架版本，尚未接入真实的 JS 提取逻辑。",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			task := opts.ToTask()
			manager := runner.NewRunnerManager()

			fmt.Printf("[*] Starting API Scan against %s (Ports: %s)...\n", opts.Target, opts.Ports)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			results, err := manager.Execute(ctx, task)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			console := reporter.NewConsoleReporter()
			console.PrintResults(results)

			if globalOutputOptions.OutputJson != "" {
				saveJsonResult(globalOutputOptions.OutputJson, results)
			}
			if globalOutputOptions.OutputCsv != "" {
				if err := reporter.SaveCsvResult(globalOutputOptions.OutputCsv, results); err != nil {
					fmt.Printf("[-] Failed to save csv: %v\n", err)
				}
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Target, "target", "t", opts.Target, "目标 URL/IP")
	flags.StringVarP(&opts.Ports, "ports", "p", opts.Ports, "端口范围")
	flags.StringVar(&opts.Crawl, "crawl", opts.Crawl, "是否深度爬取子页面（auto/true/false）")
	flags.IntVar(&opts.CrawlDepth, "crawl-depth", opts.CrawlDepth, "深度爬取的层数（仅 crawl=true 时生效）")

	cmd.MarkFlagRequired("target")
	return cmd
}
```

**注意**：`reporter.ConsoleReporter.PrintResults` 依赖结果类型实现 `TabularData` 接口，`ApiResult` 骨架版本尚未实现这个接口（留给 JS 提取实施文档第七节）。骨架阶段运行这个命令，表格输出大概率是空白或退化显示——**这是已知且可接受的骨架期过渡状态**，不需要在本步骤额外补一个临时的 `Headers()`/`Rows()` 实现再在下一份文档里替换掉（避免两次改动同一处代码，一次到位更省事）。骨架阶段验收以 `-oj` JSON 输出为准，不依赖表格显示。

### 验收标准

- `go build ./cmd/agent/scan/...` 通过。
- `neoScan-Agent-web.exe scan api -t <target> --oj result.json` 能正常跑完，`result.json` 里能看到 `"url"`/`"depth"` 字段，`api_scan` 命令本身通过 `RunnerManager` 正确调度到了 `ApiScanner`（而不是报 `no runner found`）。

---

## 十三、步骤 12：挂载到 `cmd/agent/scan/root.go`

```go
cmd.AddCommand(NewApiScanCmd()) // API 扫描（JS 接口提取等）
```

插入位置：`NewWebScanCmd()` 之后（第 37 行之后），保持"新命令紧邻相关命令"的既有排列习惯。

### 验收标准

- `go build ./cmd/...` 通过。
- `neoScan-Agent-web.exe scan -h` 的子命令列表里能看到 `api`。

---

## 十四、步骤 13：`task_to_core.go` 双轨支持

### 14.1 改动内容

`internal/service/adapter/task_to_core.go` 第 96~98 行：

```go
case "api_scan", "apiScan":
    coreTask.Type = model.TaskTypeWebScan
    coreTask.Params["mode"] = "api"
```

改为：

```go
case "api_scan", "apiScan":
    coreTask.Type = model.TaskTypeApiScan // 不再借道 web_scan（web扫描模块重构文档.md 第三节）
    // mode:"api" 参数随之废弃：ApiScanner 是独立 TaskType，被调度到即代表
    // 用户已显式表达意图，不需要 mode 参数区分。Params 只需要传
    // crawl/crawl_depth 等真正影响执行方式的参数（与上面 "web_scan" 分支
    // 的处理方式对称，不重复写一遍相同的三态解析逻辑——如果两个 case 未来
    // 的参数解析代码开始出现大量重复，再考虑抽取公共函数，当前只有两行
    // 不构成重复到需要抽取的程度）。
```

### 14.2 关于"双轨支持"的落地方式说明

方案文档第五节"Master 侧待办"提到"Agent 侧先完成双轨支持（新旧两种 `apiScan` 映射方式都能跑通）"。实际落地方式是：**`case "api_scan", "apiScan":` 这一个分支只能映射到一个 `TaskType`，不能同时映射两个**，所谓"双轨"指的是 `RunnerManager` 层面——`TaskTypeWebScan` 和 `TaskTypeApiScan` 两个 Runner 都已注册（步骤 9 + 已有的 `WebScanner` 注册），如果后续发现 Master 侧还没跟着切换、仍然可能下发依赖旧行为的任务，回退方式是把这一个 `case` 分支临时改回旧映射，而不是同时支持两种映射并存（同一个字符串不可能有两个不同的翻译结果）。**真正的"双轨"发生在时间线上，不是代码分支上**：本次改动直接切换 `case` 分支的目标 `TaskType`，如果切换后发现 Master 尚未同步（约定的发布顺序见方案文档 5.2 节），影响范围仅限于 Master 主动下发 `apiScan` 业务场景的场景，`web_scan` 本身的下发和执行完全不受影响。

### 验收标准

- `go build ./internal/service/adapter/...` 通过。
- 单元测试（如果 `task_to_core.go` 已有测试文件）新增一条断言：输入 `"api_scan"`/`"apiScan"` 字符串，输出 `coreTask.Type == model.TaskTypeApiScan`，且 `coreTask.Params` 不包含 `"mode"` 键。
- 手工验证：模拟 Master 下发一条 `task_type: "api_scan"` 的任务（如果有本地 Master 联调环境），确认 Agent 侧正确调度到 `ApiScanner` 而不是报错或误跑 `WebScanner`。

---

## 十五、步骤 14：`docs/Master-Agent扫描类型映射说明.md` 更新

### 15.1 改动位置

第 23 行：

```
| **`apiScan`**<br>(API扫描) | `web_scan` | `mode: "api"`<br>`path: "/api/v1"` | 针对 API 接口的特定扫描 |
```

改为：

```
| **`apiScan`**<br>(API扫描) | `api_scan` | `crawl`<br>`crawl_depth` | 独立的 API 接口扫描（JS 接口提取等），不再借道 web_scan，见 docs/爬虫/web扫描模块重构文档.md |
```

### 验收标准

- 文档表格渲染正常（Markdown 表格语法不被破坏）。
- 全文搜索该文档，确认没有其它地方仍然残留"`apiScan` 映射到 `web_scan`"的过时描述。

---

## 十六、步骤 15：`docs/Agent指令集规范.md` 更新

### 16.1 改动位置

第 81 行：

```
**TaskType**: `web_scan`（`model.TaskTypeWebScan`）。Master 下发的 `api_scan` 也会被翻译成同一个 TaskType，仅多写入 `mode="api"` 作为区分。
```

改为：

```
**TaskType**: `web_scan`（`model.TaskTypeWebScan`）。
```

（移除对 `api_scan` 的引用，因为 `api_scan` 已经独立成自己的 `TaskType`，不再与 `web_scan` 共用同一段描述）

### 16.2 新增独立章节

在文档合适位置（`web_scan` 章节之后）新增 `api_scan` 的独立章节骨架，本步骤只描述已经存在的 `crawl`/`crawl_depth` 两个参数（`ports` 作为目标端口范围也应列出）；风险提示文案与最终完整参数表由 [`Web-JS接口提取实施文档.md`](../API扫描-js提取/Web-JS接口提取实施文档.md) 第二节补充（该文档已包含具体的表格和风险提示措辞，本步骤不重复编写，避免两处文案不一致）：

```
### api_scan（API 扫描）

> 本章节参数表为骨架阶段的最小版本，`max_files` 参数与风险提示文案见
> Web-JS接口提取实施文档.md 第二节，实施完成后需要合并补充到本节。

**TaskType**: `api_scan`（`model.TaskTypeApiScan`），与 `web_scan` 是完全独立的 TaskType，不再共享同一段处理逻辑。

| 参数 | CLI Flag | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|---|
| `ports` | `--ports`/`-p` | string | No | `80,443` | 目标端口范围 |
| `crawl` | `--crawl` | string | No | `auto` | 是否深度爬取子页面 |
| `crawl_depth` | `--crawl-depth` | int | No | `2` | 深度爬取层数 |
```

### 验收标准

- 文档结构清晰，`web_scan`/`api_scan` 两节不再互相引用对方的 TaskType。

---

## 十七、步骤 16：端到端验收

### 17.1 破坏性回归检查清单（对照方案文档第四节逐条验证）

- [ ] `git diff` 确认 `crawler.go`/`extract.go`/`leak.go` 迁移前后内容逐字节一致（除路径外）；
- [ ] `go test ./internal/core/scanner/web/... -v` 全量通过，用例集合与重构前一致，无新增失败；
- [ ] `go test ./internal/core/lib/crawler/... -v` 全量通过；
- [ ] `go test ./internal/core/scanner/api/... -v` 通过（骨架阶段的最小验证：能跑出非空首页结果）；
- [ ] `go build ./...`、`go vet ./...` 全仓库无错误无新增警告；
- [ ] 手工执行 `neoAgent scan web -t <已知可正常工作的目标> --screenshot` 与重构前的历史结果比对（截图/指纹/CDN 判断/深度爬取子页面数量应完全一致）；
- [ ] 手工执行 `neoAgent scan api -t <target>`，确认命令能跑通、不报 `no runner found`、`-oj` 输出的 JSON 里能看到首页和（如果触发了深度爬取）子页面各一条 `ApiResult` 记录；
- [ ] 确认 `RunnerManager` 同时能正确调度 `TaskTypeWebScan` 和 `TaskTypeApiScan` 两个独立 Runner（方案文档 4 节"双轨支持"的验收方式：分别构造两个任务并发跑，互不干扰）；
- [ ] `docs/Master-Agent扫描类型映射说明.md`、`docs/Agent指令集规范.md` 已同步更新，且相互之间、与代码行为之间不存在矛盾描述。

### 17.2 性能基线检查

- `go test ./internal/core/scanner/web/... -v` 全量回归总耗时应与重构前处于同一量级（参照 `Web扫描CDN识别实施文档.md` 8.3 节"11.686s"的历史基线做法），本次重构是纯内部结构调整，不引入任何新的网络请求或计算开销，不应该出现可感知的性能回退。

---

## 十八、实施完成后需要同步更新的文档

代码全部落地并通过步骤 16 验收后，回填以下文档，不要让文档和代码状态脱节：

1. [`web扫描模块重构文档.md`](./web扫描模块重构文档.md) 第五节"5.1 本次范围"：把各项从待办状态标注为已完成。
2. 通知 [`Web-JS接口提取实施文档.md`](../API扫描-js提取/Web-JS接口提取实施文档.md) 的前置依赖已就绪，可以开始其"一、实施步骤总览"里的步骤 4~10（`ApiScanner.Run` 填充真正的 JS 提取逻辑、`ApiScanOptions` 补充 `MaxFiles`、CLI 补充风险提示与表格输出）。
3. `Agent重构开发总纲.md`（如果存在对应的 Sprint 记录机制）：补充一条 Sprint 记录，一句话说清"做了什么、解决了什么真实问题"（参照 `Web扫描CDN识别实施文档.md` 第九节的记录格式）。

本实施文档本身不需要在完成后删除，作为这次改动的历史记录保留。
