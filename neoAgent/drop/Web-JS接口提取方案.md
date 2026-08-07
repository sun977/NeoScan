# Web 扫描 JS 接口提取方案

> 文档版本：v2.0（架构调整：JS API 提取归属 `ApiScanner` 独立扫描器）
> 修改时间：2026-08-07
> 分析对象：`internal/core/lib/crawler/`（迁移后位置，含 `crawler.go`/`extract.go`/`leak.go`/`jsapi.go`）、`internal/core/scanner/api/api_scanner.go`、`internal/core/model/result_types.go`
> 文档性质：方案设计文档，负责论证"要不要做、做成什么形态、边界在哪里"；不含逐行代码改动步骤，动工前应参照本文档补一份实施文档（可参考 [`Web扫描CDN识别实施文档.md`](./Web扫描CDN识别实施文档.md) 的编写格式）。
> 关联文档：[`Web爬虫与被动分析器实施文档-v1.0.md`](./Web爬虫与被动分析器实施文档-v1.0.md)（`crawler` 包现有的 BFS/攻击面提取/被动泄露检测三件套，本方案是它的第四件套）、[`爬虫种类.md`](./爬虫种类.md)（本方案立项的原始动机记录）、[`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md)（同类开源工具调研，v1.1 第四节的规则设计据此调整）、[`Web-JS接口提取实施文档.md`](./Web-JS接口提取实施文档.md)（v2.0，逐行落地步骤，与本文档第六节的挂载点结论保持同步）、[`../爬虫/web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md)（**权威依据**：`ApiScan` 独立成 `TaskType`/`ApiScanner` 的架构决策与论证过程，本文档 v2.0 的全部改动均以该文档的结论为准）
>
> **v1.1 变更说明**：调研 JSFinder/URLFinder/dirsearch 三个同类工具后，对第四节"提取规则设计"做了两处调整（新增 scope 锚定规则档位、提取与过滤解耦为两个独立函数），并在第七节补充一条面向未来的心智模型记录（若做"接口存活验证"独立能力，需前置引入 Wildcard 检测）。核心结论（不新建模块、不主动验证、不引入 AST）未变。
>
> **v1.2 变更说明**：第六节"集成点"原描述把提取逻辑挂在 `fetchAndExtract`（只在触发深度爬取时才执行），实施阶段发现这是架构错误——会导致首页永远拿不到提取结果、且未触发深度爬取的任务整体失效。已改为挂载在 `WebScanner.buildWebResult`（首页与所有子页面结果唯一共同的收口函数）。这版结论已被 v2.0 取代，仅作历史记录保留。
>
> **v2.0 变更说明（架构重大调整，取代 v1.2 的挂载点结论）**：`ApiScan` 从"`web_scan` 的一个 `mode:"api"` 参数"升级为 Master 侧独立的业务场景与 Agent 侧独立的 `TaskTypeApiScan`/`ApiScanner`，详细论证见 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md)。核心影响：
> 1. JS API 提取不再挂载于 `WebScanner.buildWebResult`，改为挂载于新增的 `ApiScanner.Run`（第六节全面重写）；
> 2. `crawler` 包整体从 `internal/core/scanner/web/crawler` 迁移到 `internal/core/lib/crawler`，`WebScanner` 与 `ApiScanner` 平级复用同一个 `crawler.FetchAndCrawl` 抓取入口；
> 3. 第七节"不新建独立模块 / 不新增 `TaskType`"的结论被推翻，改为"已独立为 `TaskTypeApiScan`"，原有论证过程作为历史记录保留、并标注推翻原因；
> 4. `WebResult.APIs`/`APIsTruncated` 字段设计作废，改为新增独立的 `model.ApiResult` 类型（第三节重写）。
> 三层提取规则、不落盘、不反哺 BFS、不引入 AST、不主动验证等**提取能力本身**的结论不受影响，v2.0 只改变"这个能力挂在哪个 Scanner 下"。

---

## 一、结论先行

【核心判断】
✅ 值得做，且成本很低——不是"要不要做"的问题，是"做成什么形态"的问题。

一句话说清方案：**对已经抓到手的 JS 文本（首页内联 `<script>` + 外链 js 文件）跑一遍分层正则，抠出前端代码里硬编码/拼接的接口调用地址，作为独立 `ApiScanner` 的核心产出，挂到 `model.ApiResult.APIs` 字段，不引入 AST、不主动发起任何验证请求。**

| 维度 | 现状 | 方案后 |
|---|---|---|
| JS 接口提取能力 | 无，`crawler` 包只提取页面链接（`links`）、表单（`Forms`）、URL 参数名（`Params`）、敏感信息泄露（`Leaks`） | 有，新增第四种被动分析能力 `jsapi.go`，产出 `APIEndpoint` 列表 |
| 归属模块 | 不适用 | `internal/core/lib/crawler/`（迁移后位置），新增 `jsapi.go`，与 `extract.go`/`leak.go` 并列；调用方是新增的 `internal/core/scanner/api/api_scanner.go` |
| 是否新增 `TaskType` | 不适用 | **是**，`model.TaskTypeApiScan`，与 `WebScanner` 平级的独立原子扫描器，详见 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md) |
| 外链 JS 文件是否下载内容 | 否，`context.go` 的 `ExtractRichContext` 只提取了 `dom.scripts`（URL 列表），未下载内容 | 是，新增下载逻辑，复用现有 QoS 限流与大小上限 |
| 是否反哺 BFS 队列继续爬取 | 不适用 | 否，`APIs` 是终态产出，不进入 `crawler.EnqueueExtra` |
| 是否主动验证接口存活 | 不适用 | 否，仅静态提取，不发起任何额外验证请求（阶段二可选能力，见重构文档 5.3 节） |
| 新增第三方依赖 | 无 | 无（不引入 JS Parser/AST 库，正则即可覆盖当前诉求） |

---

## 二、两个容易混淆、必须先分清楚的概念

方案定稿前经过一次返工：最初把提取产物命名为 `APIEndpoint.Path`，容易让人误以为它和 `crawler` 包现有的 `links`（页面导航链接）是同一类东西，实际上二者语义、用途、后续处理方式完全不同，必须在类型设计阶段就分开，否则会在"这个 URL 要不要塞进 BFS 继续爬"这类问题上反复橡皮擦。

### 2.1 `links`（页面导航链接）—— `extract.go` 现状能力

```29:44:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/crawler/extract.go
// ExtractLinksAndForms 是本文件对外暴露的唯一入口函数。
//
// 输入：
//   - baseURL: 当前页面自己的 URL（用于把页面里的相对路径链接换算成绝对路径，
//     比如页面是 http://a.com/dir/page.html，页面里写的 <a href="foo.html">
//     必须换算成 http://a.com/dir/foo.html，否则爬虫拿到的链接根本没法访问）。
//   - body: 页面的原始 HTML 文本（从 net/http 响应体或 go-rod 渲染结果里拿到的）。
```

`links` 指向的是"另一个可以被浏览器直接打开、期望返回 HTML 页面"的地址，存在的唯一目的是喂给 BFS 队列继续爬取（`crawler.go` 的 `enqueue`）。

### 2.2 `APIEndpoint`（JS 接口调用地址）—— 本方案新增能力

指前端代码运行时用来发起 `fetch`/`axios`/`XMLHttpRequest` 等数据请求的地址。它通常**不能**直接在浏览器打开看到有意义的页面内容（可能是 404/405，或者返回一段裸 JSON/需要鉴权），本质是"攻击面清单里的一项"，和 `FormInfo`（表单攻击面）是同一层次的产物，不是链接。

```130:135:c:/mytools/code/go/NeoScan/neoAgent/internal/core/model/result_types.go
// FormInfo 表单信息（攻击面输入点）
type FormInfo struct {
	Action string   `json:"action"`
	Method string   `json:"method"`
	Fields []string `json:"fields"`
}
```

### 2.3 对比表

| 维度 | `links`（页面链接） | `APIEndpoint`（接口地址） |
|---|---|---|
| 语义 | "这是另一个可浏览的页面" | "这是一个数据接口地址" |
| 用途 | 喂给 BFS 继续爬取 | 攻击面清单，供用户/后续能力查看 |
| 附带信息 | 无，就是个 URL | 可能有 HTTP Method、来源文件、置信度 |
| 是否应被爬取（GET 打开看内容） | 是，这是它存在的意义 | 否，GET 打开通常拿不到有价值内容，甚至可能有副作用 |
| 归属数据结构 | `Page.Links`（BFS 内部队列用，不直接进最终结果） | 直接产出为 `model.ApiResult.APIs`（`ApiScanner` 结果的一部分，不再是 `WebResult` 的字段），不经过 `crawler.Page` 中转（挂载点见第六节） |

**结论：`APIEndpoint` 不会、也不应该被塞进 `crawler.EnqueueExtra`，两者在类型层面就应该是互不相通的两条管道。**

---

## 三、数据结构设计

> **v2.0 调整**：本节原方案把 `APIEndpoint` 和它的容器字段都设计在 `WebResult` 上（因为当时假设 JS 提取挂在 `web_scan` 下）。`ApiScan` 独立成 `TaskType` 后，`APIEndpoint` 本身的字段设计不变（仍然是下面这个结构），但**容器从 `WebResult.APIs` 改为新增的独立类型 `model.ApiResult`**，`WebResult` 不再持有任何与 API 提取相关的字段。

在 `internal/core/model/result_types.go` 中，与 `FormInfo`/`LeakInfo` 同级新增：

```go
// APIEndpoint 从 JS 代码中提取的接口调用地址。语义上不同于页面导航链接（links）——
// 它描述的是"前端代码运行时会向这个地址发起数据请求"，而不是"这是一个可以被
// BFS 继续爬取的页面"，因此不会、也不应该被塞进 crawler 的 BFS 队列。
type APIEndpoint struct {
	URL        string `json:"url"`                // 接口地址：可能是完整 URL，也可能是相对路径（如 /api/v1/user）。
	                                                // 不在提取阶段强制拼接成绝对地址——很多前端代码里的 base 地址本身是
	                                                // 运行时动态拼的（如 window.API_BASE + path），静态阶段拼出的
	                                                // "绝对地址"可能是错的，不如如实保留原文，交给使用方按需处理。
	Method     string `json:"method,omitempty"`   // 有明确证据才填（如识别到 axios.post(...) 才填 "POST"）；
	                                                // 识别不到就留空，不猜默认值——猜错的 Method 比没有更误导人。
	Source     string `json:"source"`             // 来源：inline（内联 <script>）或具体的 js 文件 URL，便于定位复核。
	Confidence string `json:"confidence"`         // high：命中结构化调用（fetch/axios/ajax）；
	                                                // medium：命中 scope 锚定规则（同域名完整 URL，见 4.2 节）；
	                                                // low：命中泛化路径正则兜底。
}
```

不再在 `WebResult` 上追加任何字段，而是新增一个独立的 `ApiResult` 类型，作为 `ApiScanner` 的 `model.TaskResult.Result`：

```go
// ApiResult 是 ApiScanner 的任务产出。与 WebResult 完全独立，不复用它的任何字段——
// ApiScanner 不做指纹识别、不截图、不判断 CDN，这些 WebResult 字段对 ApiResult 没有意义，
// 强行复用会让 ApiResult 背上一堆永远为空的字段。
type ApiResult struct {
	URL           string        `json:"url"`                      // 本页面 URL（与 WebResult.URL 同义）
	Depth         int           `json:"depth"`                    // BFS 深度，0 表示首页
	APIs          []APIEndpoint `json:"apis,omitempty"`           // 从 JS 代码中静态提取到的接口调用地址清单
	APIsTruncated bool          `json:"apis_truncated,omitempty"` // 本页引用的外链 JS 文件数超过上限被截断
}
```

`URL`/`Depth` 是 `ApiScanner` 自己组装结果时需要的最小定位信息（哪个页面、哪一层），不是从 `WebResult` 搭便车搭过来的，是 `ApiResult` 自身作为一个完整结果本应该具备的字段。具体字段设计在实施阶段可根据需要小幅调整（如是否需要 IP/Port），不影响本方案的核心结论。

`crawler.Page` **不追加任何字段**：`Page` 只是 BFS 爬取子页面的原始数据载体，JS 接口提取的判断和调用点收拢在 `ApiScanner.Run`（见第六节），`Page` 把已有的 `Body`/`URL` 字段原样交给 `ApiScanner` 即可，不需要在 `Page` 这一层再中转一次提取结果，`Forms`/`Params`/`Leaks` 那一套"爬到即挂在 Page 上"的模式不适用于 `APIs`。

---

## 四、提取规则设计：三层置信度，提取与过滤解耦，不引入 AST

> 本节 4.2/4.3 在 v1.1 中结合 [`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md) 的调研结果做了调整，详细对比见该文档第五节。

### 4.1 为什么不做真正的"JS 逆向"

立项讨论中出现过"JS 逆向"这个提法，需要先纠正认知边界：真正的逆向（反混淆、还原压缩变量名、AST 还原业务逻辑）是另一个量级的工程，业界做同类事情的工具（JSFinder、URLFinder、packer-fuzzer、FindSomething）也没有做到这一步，走的都是"正则扫描已有源码文本"路线（调研细节见 [`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md)）。这是一个**信息提取**问题，不是**逆向工程**问题，正则的成本和收益在当前场景下是匹配的，上 AST Parser 属于用"理论完美"的方案解决一个正则已经够用的问题。

### 4.2 三层规则（新增 scope 锚定档，参考 dirsearch）

**高置信度规则**：匹配明确的 HTTP 调用语句结构，自带 Method 信息，误报率低（思路与 URLFinder 的 `jsFind`/`urlFind` 一致，只是我们不把规则外置为配置文件，理由见 6.4 节）：

- `fetch\(['"]([^'"]+)['"]`
- `axios\.(get|post|put|delete|patch)\(['"]([^'"]+)['"]`
- `\.ajax\(\{[^}]*url:\s*['"]([^'"]+)['"]`

**中置信度规则（新增，借鉴 dirsearch 的 scope 锚定思路）**：`crawler` 包处理每一段 JS 文本时，永远知道这段 JS 来自哪个 `baseURL`（`fetchAndExtract` 里明确传入 `it.URL`），具备 dirsearch `text_crawl` 那种"精确锚定 scope"的前提条件：

```37:58:c:/mytools/code/go/NeoScan/neoAgent/docs/references/dirsearch/lib/utils/crawl.py
    @staticmethod
    @lru_cache(maxsize=None)
    def text_crawl(url, scope, content):
        results = []
        regex = re.escape(scope) + "[a-zA-Z0-9-._~!$&*+,;=:@?%]+"

        for match in re.findall(regex, content):
            results.append(match[len(scope):])

        return _filter(results)
```

对应到我们的实现：用 `regexp.QuoteMeta(host)`（`host` 取自 `baseURL` 的 scheme+host 部分）拼出 `host + 合法路径字符集` 的正则，命中的必然是"明确指向当前站点自己域名"的完整 URL，天然排除第三方 CDN/统计脚本域名的噪音，比"泛化猜测前缀"更精确，且不需要额外的黑名单去排除外部域名。

**低置信度兜底规则**：单纯匹配"看起来是接口路径"的字符串字面量，只认固定前缀，并显式排除静态资源后缀，参考 `leak.go` 现有的"认高置信度特征前缀、不追求覆盖全部"的取舍思路：

- `['"](/api/[a-zA-Z0-9_\-/{}]+)['"]`
- `['"](/v[0-9]+/[a-zA-Z0-9_\-/{}]+)['"]`
- 排除项：命中内容以 `.js`/`.css`/`.png`/`.jpg`/`.jpeg`/`.gif`/`.svg`/`.woff` 等静态资源后缀结尾的一律丢弃

去重逻辑与 `crawler.go` 现有的 `visited` 去重同一套路，不发明新机制：同一 `URL` 出现多次只记一条。

### 4.3 提取与过滤解耦为两个独立函数（借鉴 URLFinder 架构）

参考 URLFinder `jsFilter`/`urlFilter` 与提取正则相互独立的结构（`crawler/find.go` 只管正则命中，`crawler/filter.go` 单独做清洗和黑名单过滤），`jsapi.go` 内部按同样的两段式组织：

- `extractAPICandidates`：只负责跑三层正则、收集命中结果，不做任何排除判断；
- `filterAPICandidates`：统一做转义清理、静态资源后缀排除、黑名单关键词过滤。

这样规则演进时，"新增一条提取正则"和"新增一条排除规则"是两件互不影响的事，不需要每次都去改一条大正则本身的否定环，可维护性优于把过滤逻辑写进提取正则内部。

### 4.4 参考 `leak.go` 现有的实用主义取舍原则

```61:64:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/crawler/leak.go
// 这四条不是"能想到的所有敏感信息类型"，而是"误报率最低、正则最容易写对"
// 的一批，符合实用主义原则——与其追求大而全但充满误报的规则库，不如先把
// 最确定的几类做扎实，后续如果生产环境反馈需要更多类型，再按需增加规则，
```

接口提取规则遵循同样的原则：宁可漏报，不做大而全但充满噪音的正则堆砌。后续如生产反馈某类框架（如 GraphQL 单一端点、gRPC-Web）的接口提取效果差，再针对性补规则，而不是一次性设计一套试图覆盖所有框架的复杂规则体系。

---

## 五、JS 源码获取：复用现有两条数据源，不新增抓取通道

### 5.1 go-rod 路径：`context.go` 已经拿到 URL，缺的是下载内容这一步

```54:74:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/context.go
	// 4. 提取 Script 标签 (src)
	var scripts []string
	scriptRes, err := page.Eval(`() => {
		const scripts = document.getElementsByTagName('script');
		const result = [];
		for (let i = 0; i < scripts.length; i++) {
			if (scripts[i].src) {
				result.push(scripts[i].src);
			}
		}
		return result;
	}`)
```

现状只提取了外链 JS 的 URL 列表（供指纹匹配引擎使用），并未下载文件内容。本方案新增：对这批 URL 用现有 `qos.AdaptiveLimiter` 限流后逐个下载（纯 `net/http`，不需要浏览器，静态 JS 文件本身是文本资源），复用 `crawler.go` 里已有的 2MB 单文件读取上限，并新增"每页最多下载 N 个 JS 文件"的上限（防止大型 SPA 打包出几十个 chunk 文件时拖垮任务耗时和内存）。

### 5.2 fallback 路径：`ExtractLinksAndForms` 顺手多找一种标签

同一个 `goquery.Document` 里追加 `Find("script[src]")` 拿到外链地址，与现有 `extractLinks`/`extractForms` 的实现模式一致。

### 5.3 内联 `<script>` 不需要额外请求

`goquery.Find("script:not([src])").Text()` 直接从已有的 `body` 里取，零成本。这部分价值不小——很多站点的接口 base 地址就是直接内联在首页 `<script>` 里（如 `window.API_BASE = '/api/v2'`）。

---

## 六、集成点：挂载在独立的 `ApiScanner.Run`，与 `WebScanner` 共享同一个抓取入口

> **v2.0 整节重写**：本节原内容（v1.1~v1.2）讨论的是"提取能力该挂在 `web_scan` 内部哪个函数"（`fetchAndExtract` vs `buildWebResult`），这个问题本身已经被更上层的架构决策超越——`ApiScan` 不再是 `web_scan` 的一部分，而是一个独立的 `TaskType`/`Scanner`（完整论证见 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md)）。下面重新论述挂载点，原 6.1/6.2 节的历史论证（为什么不能挂在 `fetchAndExtract`）仍然成立且有参考价值，保留在 6.1a 节作为历史记录。

### 6.1 现在的正确挂载点：`ApiScanner.Run`

JS 接口提取能力挂载在新增的 `internal/core/scanner/api/api_scanner.go`——`ApiScanner` 是一个独立的原子扫描器，与 `WebScanner` 平级，都调用同一个 `crawler.FetchAndCrawl`（迁移到 `internal/core/lib/crawler` 后的统一抓取入口，详见重构文档第三节）拿到首页 + （可选）深度爬取的子页面，对每一个页面都调用 `crawler.ExtractPageAPIs(ctx, pageURL, body, client, limiter, maxFiles)` 提取，组装成 `model.ApiResult` 返回。

与 v1.2 时代相比，最大的区别是：`ApiScanner.Run` **自己就是任务的入口**，不需要再去借住 `WebScanner` 某个收口函数才能被调用到——因为它现在本身就是 `RunnerManager` 注册表里一个独立的 `TaskTypeApiScan` 入口，不存在"首页路径没有经过某个函数导致拿不到提取结果"这类问题——`ApiScanner.Run` 内部对抓到的每一个页面（无论首页还是子页面）都调用同一次 `ExtractPageAPIs`，这个函数本身就是它为自己服务的，不存在被别的 Scanner "途中挂载"的问题。

### 6.1a 历史记录：当 JS 提取还挂在 `web_scan` 下时的挂载点论证（已被上方取代，仅供参考）

**当时的错误挂载点：`fetchAndExtract`**——`crawler.Crawl` 只有在 `web_scanner.go` 判定需要深度爬取（用户传了 `--crawl=true`，或首页自动判断需要爬）时才会被调用；首页本身的处理完全不经过 `crawler` 包，是 `WebScanner.Run` 主干直接组装。如果把提取逻辑挂在 `fetchAndExtract` 里，会导致两个问题：① 首页永远拿不到提取结果；② 未触发深度爬取的任务整体拿不到任何提取结果。后来修正为挂在 `WebScanner.buildWebResult`（首页与所有子页面结果唯一共同流经的收口函数）。这个修正本身在当时的前提下（JS 提取挂在 `web_scan` 下）是完全正确的，但随着 `ApiScan` 独立成自己的 `TaskType`，"要不要借住 `WebScanner` 的某个函数"这个问题本身不再存在。

### 6.2 对现有编排逻辑的影响

不改 `WebScanner.Run`/`runOnePort` 的核心编排逻辑（BFS 顺序、深度控制、并发调度都不变）。`ApiScanner.Run` 是新增的平行入口，与 `WebScanner` 互不影响，属于完全增量改动。

### 6.3 规则暂不外置为配置文件（区别于 URLFinder 的设计）

URLFinder 把提取正则、过滤黑名单都做成了 YAML 外置配置（详见 [`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md) 第二节），可以不重新编译就调整规则，这是一个真实的工程优点。本方案现阶段不采用这个设计，原因：

- 现有 `crawler` 包的 `leak.go` 敏感信息规则（`defaultLeakRules`）也是编译期常量，`jsapi.go` 的规则和它保持一致的处理方式，不在同一个包里搞两套不同的规则管理机制；
- 规则外置为配置文件本身是有代价的（配置文件解析、格式校验、错误处理、文档同步），这个代价现阶段没有对应的收益——目前规则数量少（三层共几条），改动频率低，编译发布本身也不是这个项目的痛点；
- 如果未来规则数量膨胀到需要频繁调整、且不方便每次都走发布流程，才是把规则拆成外置配置（甚至独立规则目录，类比 `rules/fingerprint/`）的合适时机，现在做属于提前优化一个还不存在的问题。

---

## 七、边界与明确不做的事

### 7.1 不反哺 BFS（对应第二节的类型区分）

提取出的 `APIs` 是终态产物，不会调用 `crawler.EnqueueExtra` 把它们塞回队列。理由参照 `leak.go` 已经明确过的边界：

```10:17:c:/mytools/code/go/NeoScan/neoAgent/internal/core/scanner/web/crawler/leak.go
// 为什么叫"被动"检测：
//   这里的检测不发起任何额外的网络请求、不做任何主动探测（比如不会去尝试用
//   找到的 AK/SK 调用云厂商 API 验证真假），纯粹是"顺手看一眼已经抓到的页面
//   内容里有没有可疑字符串"。
```

对提取出的接口地址发起请求验证是否存活/是否需要鉴权，属于主动探测范畴，混进被动分析阶段会让"这次扫描到底对目标发起了多少次请求"变得不可预测，这对安全扫描工具的行为边界是需要谨慎对待的问题。是否需要"接口存活验证"应该是一个**用户显式开启的独立能力**（类比现有 `crawl` 三态开关的设计），不与被动提取混在一起。

**面向未来的心智模型记录（v1.1 新增，不是本次方案需要实现的代码）**：调研 dirsearch 后发现，它的 Wildcard/软 404 检测机制（`lib/core/scanner.py` 的 `classify`/`is_wildcard`/`is_probable_wildcard`，详见 [`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md) 第三节）是同类工具里降噪做得最严谨的实现。如果未来真的要做"接口存活验证"这个独立能力，**必须前置引入类似的 Wildcard 检测**：先对一个几乎不可能真实存在的随机路径发一次请求，拿到的响应作为"这个站点对不存在路径的标准反应模板"，后续每一次真正验证的结果都要先和这个模板比对，只有明显不同于模板的响应才算真实存在，否则遇到"任意路径都返回 200 + 统一错误 JSON"的站点时，验证结果会产出大量假阳性，污染整份结果的可信度。同时应参考 URLFinder 的 `Risks` 黑名单机制，内置危险路由关键词（`delete/remove/drop/truncate` 等），避免验证请求无意间触发接口的副作用。这两点现在不需要写任何代码，只是提前记录心智模型，避免以后设计验证能力时重新踩一遍坑。

### 7.2 已独立为 `TaskTypeApiScan`（推翻了 v1.2 的"不新建模块"结论）

> **v2.0 结论变更**：本节原结论是"不新建独立模块、不新增 `TaskType`"，现已被推翻。下面先说明现状结论，再保留原论证作为历史记录并说明为什么被推翻。

**现状结论**：`ApiScan` 已独立为 `model.TaskTypeApiScan`，与 `TaskTypeWebScan` 平级，对应独立的 `internal/core/scanner/api/api_scanner.go`。`docs/Master-Agent扫描类型映射说明.md` 的 `apiScan → web_scan` 映射行同步改为 `apiScan → api_scan`。完整论证、破坏性分析、Master/Agent 两侧分工见 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md)。

**为什么推翻了原结论**：原结论建立在两个隐含前提上：(1) Master 侧协议是固定不可改的；(2) 判断标准只看"当前这一个 JS 提取能力本身的数据依赖"。重新讨论后发现这两个前提都不成立：Master 协议本身在持续进化（参见重构文档中对《原子扫描器批量扫描能力扩充方案.md》第六节已经在讨论修复 `task_to_core.go` 多目标丢弃问题的先例，说明协议层面的调整并非禁区）；而且真正的判断依据不应该只看"当前这一个具体能力的数据依赖"，还应该考虑"API 扫描这个业务域未来要不要支持不依赖页面抓取的输入源"——接口存活验证、直接导入 OpenAPI 文档这类需求一旦成立，数据依赖就不再是"100% 来自 `web_scan`"了。现在先把 `TaskType` 独立出来，比以后能力进一步扩展时再回头拆分成本更低。

**历史记录（原 7.2 节内容，仅作参考）**：当时的论证是"`docs/Master-Agent扫描类型映射说明.md` 已经把 `apiScan` 映射定好到 `web_scan`，判断标准是数据依赖是否独立，若强行拆分要么重复实现页面抓取，要么新模块反过来依赖 `web` 包取数据"。这个担忧在重构方案中已经有了解决方案：把抓取能力下沉到 `internal/core/lib/crawler`，`WebScanner` 和 `ApiScanner` 平级复用同一个入口，不存在谁依赖谁的问题，也不需要重复实现。原来担忧的"要么...要么..."两难局面并不成立，只是当时没有想到第三条路（抽取公共层）。

### 7.3 不引入 AST/JS Parser 依赖

见第四节。当前诉求下正则的命中率与维护成本已经匹配，AST 方案的开发和长期维护代价与当前收益不成比例。

### 7.4 不做基于 CDP 网络拦截的运行时接口捕获

`page.HijackRequests`/CDP `Network.responseReceived` 能捕获"运行时真实发出的 XHR/Fetch 请求"，比正则扫源码更准（能捕获运行时动态拼接、正则扫不出的 URL），但需要托管一次完整浏览器交互周期、处理请求去重、处理请求体，是量级更大的能力。先落地"扫 JS 源码"这个低成本高收益的版本，若生产反馈"正则扫描遗漏了大量运行时动态拼接的接口"，再考虑升级路径，这是先解决真问题、再考虑理论上更完美方案的顺序。

---

## 八、风险与验证要点（供后续编写实施文档时展开）

| 风险 | 缓解措施 |
|---|---|
| 外链 JS 文件下载数量失控，拖慢单页扫描耗时 | 复用 `qos.AdaptiveLimiter`；新增"每页最多下载 N 个 JS 文件"上限；沿用 2MB 单文件大小上限 |
| 正则误报率高，结果被噪音污染 | 分层给置信度（high/medium/low），低置信度层排除静态资源后缀；不追求"抓全部像路径的字符串" |
| `APIEndpoint.URL` 命名/语义与 `links` 混淆，被误用于 BFS | 类型设计阶段已明确 `APIEndpoint` 产出（`model.ApiResult.APIs`）与 `Page.Links`（BFS 队列）互不相通（见第二节），代码评审时对照检查 |
| 大型 SPA 打包出大量 chunk 文件导致 JS 内容重复扫描开销大 | 沿用 `crawler.go` 现有的 URL 归一化 + 去重模式，同一 JS 文件 URL 只下载/扫描一次 |

---

## 九、后续步骤

本文档只完成方案论证，不含逐行代码改动步骤。配套实施文档见 [`Web-JS接口提取实施文档.md`](./Web-JS接口提取实施文档.md)（v2.0，已同步本文档的架构调整），明确：改动文件清单、`jsapi.go` 的具体函数签名、`ApiScanner` 的完整六件套接入步骤、单元测试用例清单（表驱动覆盖高/低置信度规则、静态资源排除、去重）、验收标准（`go build`/`go vet`/`go test -race`）。

在 `ApiScanner` 真正落地之前，`internal/core/lib/crawler` 的迁移（原 `internal/core/scanner/web/crawler`）与 `crawler.FetchAndCrawl` 统一抓取入口必须先完成——这是本方案第六节挂载点、以及 `WebScanner` 零改动这一前提成立的先决条件，实施顺序上属于 [`web扫描模块重构文档.md`](../爬虫/web扫描模块重构文档.md) 的范围，需要先于本方案的 `jsapi.go` 落地。


## 十 后续独立立项

结合本方案讨论过程中确认的"深入优化"意图，以下能力基于本次搭好的 `ApiScanner` 骨架，后续按需独立展开，不在本次讨论：

1. **提取质量升级**：`OnPageReady` 回调里挂 CDP 网络监听，捕获运行时真实发出的 XHR/Fetch 请求（而非当前的静态正则提取），弥补正则方案对动态拼接 URL 的盲区
2. **接口存活验证**：对提取出的 API 端点发起主动探测，确认真实存在、判断是否需要鉴权
3. **外部输入支持**：`Params["source"]="urls"`（直接给 URL 列表）或 `"openapi"`（直接喂 Swagger/OpenAPI 文档），不需要先爬网页
4. **API 资产管理**：端点归类、去重、与已知接口文档比对，生成独立的 API 资产清单
5. **批量/持续监控**：对一批目标定期扫描，跟踪 API 端点变化