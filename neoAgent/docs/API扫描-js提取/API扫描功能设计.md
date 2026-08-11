# API 扫描（ApiScanner）功能设计方案

> 文档版本：v1.0
> 修改时间：2026-08-11
> 文档性质：方案设计文档，负责论证"做成什么形态、边界在哪里、和现有架构怎么对齐"，不含逐行代码改动步骤。配套实施文档见后续独立产出（对齐 [`Web扫描CDN识别实施文档.md`](../爬虫/Web扫描CDN识别实施文档.md) 的编写格式，方案确认后再补）。
> 关联文档：
> - [`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md)（同类开源工具调研，本方案的规则设计与架构参考均来自该文档结论）
> - [`00.原子扫描器开发指南[定稿].md`](../00.原子扫描器开发指南[定稿].md) 第 0.1 节"原子扫描器隔离原则"（本方案必须遵守的架构铁律，是本方案与已归档方案最核心的区别所在）
> - `neoAgent/drop/Web-JS接口提取方案.md`（已归档的历史方案 v2.0，本方案评估后判定其中约 60% 内容依然成立并直接继承，详见第七节）
> - 当前代码骨架：`internal/core/scanner/api/api_scanner.go`（`Name()/Run()` 骨架）、`internal/core/model/result_types.go` 的 `ApiResult`（骨架字段）、`internal/core/options/scan_api.go`（`ApiScanOptions` 骨架）

---

## 一、结论先行

【核心判断】
✅ 值得做，且不是从零开始——`model.TaskTypeApiScan`、`ApiScanner` 骨架、`ApiResult` 骨架、CLI `scan api` 六件套已经全部就位（见 `docs/开发进度.md` 5.1.2 节），唯一缺的是 `ApiScanner.Run()` 里的真实抓取与提取逻辑。

一句话说清方案：**`ApiScanner` 独立发起抓取（go-rod 渲染 + 默认开启的深度 BFS），对拿到的 HTML/内联JS/外链JS 三种文本统一跑三层置信度正则，提取与过滤解耦为两段式处理，产出 `[]APIInfo` 挂到 `model.ApiResult`，不主动验证、不引入 AST、不做 CDP 拦截。**

| 维度 | 现状 | 方案后 |
|---|---|---|
| 抓取能力 | 无，`Run()` 直接返回空结果 | 有，`ApiScanner` 在自己包内独立实现 go-rod 渲染抓取 + BFS 深度爬取 |
| 归属模块 | `internal/core/scanner/api/`（仅骨架） | 同目录下新增 `fetch.go`/`bfs.go`/`extract.go`/`filter.go`，物理独立，不 import `scanner/web/crawler` |
| 是否复用 `WebScanner` 的 `crawler` 包 | 不适用 | **否**，违反"原子扫描器隔离原则"（`docs/00.原子扫描器开发指南[定稿].md` 第 0.1 节），架构模式可参照，代码不能共用 |
| 深度爬取是否默认开启 | 不适用（骨架未实现） | **是**，与 `WebScanner` 的"三态自动判断是否要爬"语义不同——ApiScan 存在的意义就是挖接口，不存在"要不要爬"的决策点 |
| 提取规则 | 无 | 三层置信度（高：结构化调用语句；中：scope 锚定当前域名；低：固定前缀兜底），提取与过滤解耦为两个独立函数 |
| 是否主动验证接口存活 | 不适用 | 否，仅静态提取，不发起任何额外验证请求 |
| 是否引入 AST/JS Parser | 不适用 | 否，正则的成本收益比已匹配当前诉求 |
| 新增第三方依赖 | 无 | 无 |

---

## 二、架构前提：两条不可违反的铁律

本方案的设计过程始终围绕两条已经在本仓库确立的铁律展开，任何方案细节与它们冲突时，以铁律为准。

### 2.1 原子扫描器隔离原则（`00.原子扫描器开发指南[定稿].md` 第 0.1 节）

> **各原子扫描器的内部实现（抓取、解析、分析等）必须自成一体，扫描器之间不共享内部模块，各自的能力在各自的包内独立实现。**

这条原则不是理论推导，而是一次真实的架构反转踩坑后确立的：曾经尝试把 `WebScanner` 的爬虫子模块下沉为 `core/lib/crawler` 通用模块，供 `WebScanner`/`ApiScanner` 共享调用，已完整实现过又完整回退（详见 `drop/web扫描模块重构文档.md`）。核心教训：两个扫描器对"抓取"这件事的真实需求天然不同，为了让一个通用接口同时满足两者，被迫引入回调等间接层，接口复杂度不降反升；且收益不成立——"复用"是对着规划中的需求做的预先设计，属于过度设计。

**对本方案的直接约束**：`internal/core/scanner/web/crawler/` 是 `WebScanner` 的私有实现，`ApiScanner` 不能 `import` 它，也不能通过"抽公共函数"的方式绕过去与它耦合。

**但这不等于孤立造轮子**：`internal/core/lib/` 下的公共基础设施（`lib/browser.BrowserLauncher`/`BrowserManager`、`lib/network/qos.AdaptiveLimiter`）是 Factory 模式下面向全体扫描器的基础设施层，`WebScanner` 本身也只是这层设施的一个使用方，不是私有拥有者。这一层可以正常复用，`ApiScanner` 只需要在 `NewApiScanner()` 里构造自己独立的实例（不与 `WebScanner` 共享运行时状态）即可。

| 组件 | 归属 | `ApiScanner` 能否使用 |
|---|---|---|
| `lib/browser.BrowserLauncher/BrowserManager` | 公共基础设施 | ✅ 能，独立实例 |
| `lib/network/qos.AdaptiveLimiter` | 公共基础设施 | ✅ 能，独立实例 |
| `scanner/web/crawler`（BFS/extract/leak） | `WebScanner` 私有 | ❌ 不能 import，需在 `scanner/api` 包内独立实现 |
| `net/http`、`goquery` 等三方库 | 语言/生态基础设施 | ✅ 能 |

### 2.2 TaskType 是唯一契约（`00.原子扫描器开发指南[定稿].md` 第 2 节）

`model.TaskTypeApiScan`（`"api_scan"`）已经独立于 `TaskTypeWebScan` 存在，六件套（`model`/`factory`/`runner`/CLI/`options`/`task_to_core`）已接入完成（见 `docs/开发进度.md`）。本方案只补充 `ApiScanner.Run()` 内部的真实逻辑，不改变这条已经走通的契约链路。

---

## 三、整体流程

一句话概括：**"独立发起抓取（go-rod 渲染首页 + 默认开启的 BFS 深度爬取）→ 对每个页面的 HTML/内联JS/外链JS 三种文本统一跑分层正则 → 提取与过滤解耦两段式处理 → 按页面/深度组装 `ApiResult` 返回"**。

```
ApiScanner.Run(ctx, task)
    │
    ├─ 1. ensureInit()：懒加载自己独立的 browser/limiter 实例
    │       （复用 lib/browser、lib/network/qos 公共基础设施，
    │        实例与 WebScanner 完全独立，互不共享运行时状态）
    │
    ├─ 2. fetchPage(ctx, url)：go-rod 渲染单页
    │       ├─ 渲染后的 body（能看到 JS 动态生成的 DOM 内容，比纯 net/http 更全面）
    │       ├─ 内联 <script> 文本（page.Eval 直接读，零额外请求）
    │       └─ 外链 <script src> 的 URL 列表 → 逐个下载文本内容
    │              （受 MaxJSFiles 上限约束，超限则本页 APIsTruncated=true）
    │
    ├─ 3. extractAPICandidates(body + 内联JS + 每个外链JS文本)
    │       只管正则命中，不做任何过滤判断
    │
    ├─ 4. filterAPICandidates(candidates)
    │       统一做转义清理、静态资源后缀排除、黑名单关键词过滤、去重
    │
    ├─ 5. 组装本页 ApiResult{URL, Depth, APIs, APIsTruncated}
    │
    ├─ 6. BFS 深度爬取（默认总是开启，深度由 CrawlDepth 控制，
    │       与 WebScanner 的"三态自动判断要不要爬"语义不同——
    │       ApiScan 存在的意义就是挖接口，没有"要不要爬"这个决策点）
    │       对每个子页面重复步骤 2-5，Depth 递增
    │
    └─ 7. 汇总所有页面的 ApiResult，返回 []*model.TaskResult
```

---

## 四、需要新增/改动的文件

沿用"原子扫描器隔离原则"下的独立实现要求，`scanner/api` 包内的文件结构在职责划分上参照 `scanner/web/crawler` 已经验证过的模式（抓取与队列调度分离、提取与过滤分离），但代码物理独立，不共享、不 import。

```
internal/core/scanner/api/
├── api_scanner.go       # 已有骨架，补充 Run() 编排逻辑（参照 WebScanner.Run 的写法 A：Scanner 直接实现 Runner）
├── fetch.go             # 新增：独立的单页抓取实现（go-rod 渲染 + 内联/外链 JS 获取），不 import web/crawler
├── bfs.go               # 新增：独立的 BFS 队列实现（去重/深度控制/Scope 判断），结构参照 crawler.go 但代码独立
├── extract.go           # 新增：extractAPICandidates，三层正则命中，只管提取不做过滤
├── filter.go            # 新增：filterAPICandidates，清洗 + 静态资源排除 + 黑名单过滤（对应 URLFinder 提取/过滤解耦的架构）
└── api_scanner_test.go  # 已有骨架测试，后续补充真实用例
```

| 文件 | 改动性质 | 内容概要 |
|---|---|---|
| `internal/core/model/result_types.go` | 修改 | `ApiResult` 追加 `APIs []APIInfo`、`APIsTruncated bool`；新增 `APIInfo` 类型 |
| `internal/core/options/scan_api.go` | 修改 | 删除 `Crawl` 三态字段（骨架阶段照搬 `WebScanOptions` 遗留的死语义）；`CrawlDepth` 语义收窄为"爬多深"的独立参数；新增 `MaxJSFiles`（默认 20，CLI 可配置） |
| `cmd/agent/scan/api.go` | 修改 | 绑定 `--crawl-depth`、`--max-js-files`；去掉 `--crawl` 三态参数（如已存在） |
| `internal/core/scanner/api/api_scanner.go` | 修改 | 补充真实 `Run()` 编排逻辑 |
| `internal/core/scanner/api/fetch.go` | 新增 | 独立抓取实现 |
| `internal/core/scanner/api/bfs.go` | 新增 | 独立 BFS 队列 |
| `internal/core/scanner/api/extract.go` | 新增 | 三层置信度正则提取 |
| `internal/core/scanner/api/filter.go` | 新增 | 清洗 + 黑名单 + 去重 |
| `internal/core/factory/api_factory.go` | 修改 | 已存在且已正确接入 `NewApiScanner()`，本方案不改变其构造签名，实施阶段仅在内部实现变化时同步跟进 |

**为什么要建独立的 `bfs.go` 而不是把 BFS 判断直接塞进 `fetch.go`**：`WebScanner` 的经验已经证明"抓取"和"BFS 队列调度"是两个不同粒度的职责（`crawler.go` 负责队列/去重/深度，`extract.go` 负责单页解析），`ApiScanner` 沿用这个职责划分不是重复造轮子，是复用一个已经被验证过的正确架构模式——**模式可以参照，代码不能复制**，这是本方案与"直接把 `web/crawler` 复制一份"这种做法的本质区别。

---

## 五、参数设计（`ApiScanOptions`，替换骨架照搬版本）

当前骨架版本的注释里已经写明："只包含抓取相关的参数（Target/Ports/Crawl/CrawlDepth），与 WebScanOptions 的对应字段语义完全一致，直接照搬，不重新设计"——这是骨架阶段的占位写法，不是针对 `ApiScan` 自身语义设计过的结果，本方案对其重新设计：

| 字段 | 现状（骨架照搬） | 方案调整 | 理由 |
|---|---|---|---|
| `Target` / `Ports` | 有 | 保留，明确输入形态与优先级（见 5.1 节） | 通用定位参数，语义正确，与 `ApiScan` 自身需求无冲突 |
| `Crawl string`（"auto"/"true"/"false" 三态） | 有，默认 "auto" | **删除** | 这个三态开关是为 `WebScanner` 设计的——`WebScanner` 的核心目的是"看首页"，深度爬取是锦上添花的可选项，需要"自动判断要不要爬"。但 `ApiScan` 的深度爬取不是可选项，是这个扫描器存在的核心意义，继续保留一个事实上恒为 "true" 的开关是死代码 |
| `CrawlDepth int` | 有，默认 2 | 保留，语义从"三态开关的附属参数"收窄为"爬多深"这一个独立参数 | 深度控制本身是有效需求；**默认值留到实施文档阶段结合真实站点测试确定**，不在方案阶段拍脑袋定死 |
| `MaxJSFiles int` | 无 | **新增**，默认值 20，CLI 可配置（`--max-js-files`） | 防止大型 SPA 打包出几十个 chunk 文件拖慢单页扫描耗时；默认值参考归档方案的设计思路，平衡覆盖率与耗时 |

对应地，`ToTask()` 不再解析 `"crawl"` 三态参数，只解析 `"crawl_depth"` 和 `"max_js_files"` 两个数值参数——`RunnerManager` 调度到 `TaskTypeApiScan` 这件事本身就是"要爬"的完整意图表达，不需要再包一层判断（这与 `drop/Web-JS接口提取实施文档.md` 中"`ApiScanner.Run` 不需要判断'要不要提取'"的既有结论一致，本方案把同样的思路延伸到"要不要爬"这一层）。

### 5.1 `Target` 的输入形态与 `Ports` 的关系（澄清"网址可以没有端口"）

`model.Task.Target` 的字段注释是"扫描目标 (IP/Domain/CIDR)"——即架构设计上 `Target` 默认是**裸 IP/域名，不含 scheme**。但实际入口（CLI 直接传参、`GenerateTargets` 批量分发）都不会强行剥离用户输入的 `http(s)://` 前缀，所以 `ApiScanner` 拿到的 `Target` 实际存在两类输入：

| 输入形态 | 示例 | `Ports` 是否生效 | 处理方式 |
|---|---|---|---|
| 完整 URL（含 scheme，可带可不带端口） | `https://example.com`、`http://example.com:8080` | 否，忽略 | 直接使用，不做任何拼接——URL 本身没端口不代表"缺参数"，缺省端口是 HTTP 协议的正常语义（`https`→443，`http`→80），不需要 `Ports` 补全 |
| 裸 IP/域名（无 scheme） | `example.com`、`192.168.1.1` | 是 | 遍历 `Ports` 逐个拼出候选 URL 探测（标准端口 80/443 不显式拼进 URL，非标准端口才拼，如 `example.com:8080`），协议按端口猜测（443 类→https，其余默认 http） |

判断逻辑与 `WebScanner` 的 `normalizeURL`（`scanner/web/web_scanner.go`）保持一致（"URL 优先、`Ports` 兜底"），但**不 import 该函数**，遵循第二节的原子隔离原则，在 `scanner/api/fetch.go` 内独立实现一份等价逻辑：

```go
if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
    // Target 已是完整网址，Ports 参数不生效，直接使用（无端口也合法）
} else {
    // Target 是裸 IP/域名，按 Ports 逐个拼出候选 URL 尝试连接
}
```

`Ports` 因此不是"强制给网址加端口"的参数，而是服务于两个场景：一是用户只给裸 host、需要 Scanner 自己猜端口和协议；二是 Pipeline 批量模式下 `GenerateTargets`（`pipeline/target.go`）产出的永远是裸 IP（域名会被提前解析成 IP），此时完全依赖 `Ports` 才知道要拼哪个端口。这一实现细节在实施文档阶段落到具体函数签名。

**与第九节"不主动验证接口存活"的边界区分（避免用词混淆）**：这里的"拼 URL 尝试连接"针对的是 `Target` 本身——目的是确定"从哪个 scheme+port 开始抓取"，本质是抓取流程的前置步骤，与 `WebScanner` 判断 `http`/`https` 的性质相同；第九节 9.2 节的"不做存活验证"针对的是**提取结果** `APIInfo.URL`——即挖出来的接口地址不会被二次请求去验证是否可访问。两者是完全不同层级的动作，不要混为一谈。

此外，`cmd/agent/scan/api.go` 现有 `--target` flag 的帮助文案已经是"目标 URL/IP"（与 `WebScanOptions` 一致），佐证了 CLI 层面从一开始就是按"两种输入形态都要支持"设计的，本节结论与既有 CLI 文案不冲突，只是首次把内部处理逻辑显式化。

---

## 六、数据结构设计

```go
// APIInfo 从 JS/HTML 中提取的接口调用地址。语义上不同于页面导航链接——
// 它描述的是"前端代码运行时会向这个地址发起数据请求"，而不是"这是一个可以
// 被继续爬取的页面"，因此不会、也不应该被塞进 BFS 队列继续抓取。
type APIInfo struct {
    URL        string // 接口地址：可能是完整 URL，也可能是相对路径。不在提取阶段
                       // 强行拼接成绝对地址——很多前端代码里的 base 地址本身是运行时
                       // 动态拼的，静态阶段拼出的"绝对地址"可能是错的，如实保留原文。
    Method     string // 有明确证据才填（如识别到 axios.post(...) 才填 "POST"）；
                       // 识别不到就留空，不猜默认值。
    Source     string // 来源：inline（内联 <script>）或具体的 js 文件 URL，便于定位复核。
    Confidence string // high：命中结构化调用（fetch/axios/ajax）；
                       // medium：命中 scope 锚定规则（同域名完整 URL）；
                       // low：命中泛化路径正则兜底。
}

// ApiResult 是 ApiScanner 的任务产出（在现有骨架基础上追加两个字段）。
// 与 WebResult 完全独立，不复用它的任何字段——ApiScanner 不做指纹识别、
// 不截图、不判断 CDN，这些 WebResult 字段对 ApiResult 没有意义。
type ApiResult struct {
    URL           string    `json:"url"`                      // 本页面 URL
    Depth         int       `json:"depth"`                    // BFS 深度，0 表示首页
    APIs          []APIInfo `json:"apis,omitempty"`           // 从 JS/HTML 中静态提取到的接口调用地址清单
    APIsTruncated bool      `json:"apis_truncated,omitempty"` // 本页引用的外链 JS 文件数超过 MaxJSFiles 被截断
}
```

`APIInfo` 与 `links`（页面导航链接）是两个本质不同的概念，必须在类型设计阶段就分清楚：

| 维度 | `links`（页面链接，`WebScanner` 概念） | `APIInfo`（接口地址，本方案概念） |
|---|---|---|
| 语义 | "这是另一个可浏览的页面" | "这是一个数据接口地址" |
| 用途 | 喂给 BFS 继续爬取 | 攻击面清单，供用户/后续能力查看 |
| 是否应被爬取（GET 打开看内容） | 是，这是它存在的意义 | 否，GET 打开通常拿不到有价值内容，甚至可能有副作用 |
| 是否进入 BFS 队列 | 是 | **否**，是终态产出 |

---

## 七、提取规则设计：三层置信度，提取与过滤解耦，不引入 AST

### 7.1 为什么不做真正的"JS 逆向"

真正的逆向（反混淆、还原压缩变量名、AST 还原业务逻辑）是另一个量级的工程，业界同类工具（JSFinder、URLFinder）也没有做到这一步，走的都是"正则扫描已有源码文本"路线（详见 [`JSFinder-URLFinder-dirsearch项目分析.md`](./JSFinder-URLFinder-dirsearch项目分析.md)）。这是一个**信息提取**问题，不是**逆向工程**问题，正则的成本和收益在当前场景下是匹配的。

### 7.2 三层规则

**高置信度**——匹配明确的 HTTP 调用语句结构，自带 Method 信息，误报率低（思路与 URLFinder 的 `jsFind`/`urlFind` 一致）：

- `fetch\(['"]([^'"]+)['"]`
- `axios\.(get|post|put|delete|patch)\(['"]([^'"]+)['"]`
- `\.ajax\(\{[^}]*url:\s*['"]([^'"]+)['"]`

**中置信度**（借鉴 dirsearch 的 scope 锚定思路）：`ApiScanner` 自己发起抓取，处理每一段 JS 文本时永远知道这段 JS 来自哪个页面 URL，具备"精确锚定 scope"的前提条件：

用 `regexp.QuoteMeta(host)`（`host` 取自当前页面 URL 的 scheme+host 部分）拼出 `host + 合法路径字符集` 的正则，命中的必然是"明确指向当前站点自己域名"的完整 URL，天然排除第三方 CDN/统计脚本域名的噪音，比"泛化猜测前缀"更精确，且不需要额外的黑名单去排除外部域名。

**低置信度兜底规则**（v1.1 调整：从"固定前缀白名单"放宽为"通用路径字面量"，提高召回率）：

- `['"](/[a-zA-Z0-9_\-]+/[a-zA-Z0-9_\-/{}]+)['"]`
- 不再限定必须是 `/api/`、`/v[0-9]+/` 开头——国内大量后台管理系统的接口命名并不遵循这个约定（如 `/user/getUserInfo`、`/order/list`），只认固定前缀会系统性漏掉这批真实接口。放宽后由过滤层（见 7.3 节黑名单）兜底降噪，而不是在提取阶段就收窄。
- 排除项：命中内容以 `.js`/`.css`/`.png`/`.jpg`/`.jpeg`/`.gif`/`.svg`/`.woff` 等静态资源后缀结尾的一律丢弃

### 7.3 提取与过滤解耦为两个独立函数（借鉴 URLFinder 架构）

参考 URLFinder `jsFind`/`jsFilter` 与提取正则相互独立的结构：

- `extractAPICandidates`：只负责跑三层正则、收集命中结果，不做任何排除判断；
- `filterAPICandidates`：统一做转义清理、静态资源后缀排除、黑名单关键词过滤、去重。

这样规则演进时，"新增一条提取正则"和"新增一条排除规则"是两件互不影响的事，不需要每次都去改一条大正则本身的否定环。

**v1.1 新增：低置信度规则放宽后，`filterAPICandidates` 需要同步加固黑名单**，直接借鉴 URLFinder 经过实战验证的排除规则（`config.go` 的 `UrlFiler`），排除提取正则容易误伤的 JS 属性访问模式：

```
\.js\?|\.css\?|\.jpeg\?|\.jpg\?|\.png\?|\.gif\?|www\.w3\.org|example\.com|<|>|\{|\}|\[|\]|\||\^|;|/js/|\.src|\.replace|\.url|\.att|\.href|location\.href|javascript:|location:|application/x-www-form-urlencoded|\.createObject|:location|\.path
```

这条规则的价值在于它是"具体实战踩坑经验的沉淀"——`.src`/`.href`/`location.href` 这类 JS 对象属性访问，字符串形态上和一个真实路径长得很像，容易被宽松后的低置信度正则误伤，靠黑名单在过滤阶段统一挡掉，不需要在提取正则里对每种情况写否定环。

### 7.4 实用主义取舍原则（v1.1 调整）

`ApiScan` 的产品定位是"尽可能挖掘页面上的 API 接口"，与 `WebScanner`/`web/crawler/leak.go` 的"被动检测、宁可漏报不要噪音"定位不同——`ApiScan` 存在的意义就是主动、尽量全面地覆盖接口面，因此**提取阶段允许一定误差换取更高召回率**是合理取舍，不是"规则堆砌"。低置信度这一档的产出天然带着 `Confidence: "low"` 标记，用户能清楚知道这一档是"广撒网"的结果，需要自行甄别，这是"允许误差"和"结果不可信"之间的边界——误差要显性标注，不能悄悄混进高置信度结果里。

---

## 八、JS 源码获取方式

采用 go-rod 无头浏览器渲染（而非纯 `net/http` 静态抓取），能看到 JS 动态生成的 DOM 内容和运行时变量，覆盖面更全，代价是更重（需要启动浏览器进程）。这一点是与 `WebScanner` 的抓取方式保持一致的技术选型，但**抓取的实现代码物理独立**，不共享 `web` 包内的任何抓取代码。

三种文本来源，分别处理：

1. **首页/子页面 HTML**：go-rod 渲染后拿到的完整 body（包含 JS 执行后动态生成的内容）。
2. **内联 `<script>` 文本**：`page.Eval` 直接从已渲染页面里读取，零额外网络请求。
3. **外链 JS 文件**：先拿到 `<script src>` 列表，再逐个下载文本内容（纯 `net/http`，不需要浏览器，静态 JS 文件本身是文本资源），受 `MaxJSFiles` 上限约束、复用现有 QoS 限流。

---

## 九、明确不做的事

### 9.1 不反哺 BFS 队列

`APIs` 是终态产出，不会被塞回抓取队列继续爬取。理由与 `web/crawler/leak.go` 的"被动检测"边界一致：对提取出的接口地址发起请求验证是否存活/是否需要鉴权，属于主动探测范畴，混进被动分析阶段会让"这次扫描到底对目标发起了多少次请求"变得不可预测。

**面向未来的心智模型记录**（不是本次方案需要实现的代码）：如果未来要做"接口存活验证"独立能力，必须前置引入 dirsearch 式的 Wildcard/软 404 检测机制（先探测一个几乎不可能真实存在的随机路径作为"标准反应模板"），并内置危险路由关键词黑名单（参考 URLFinder 的 `Risks` 机制，如 `delete/remove/drop/truncate`），避免验证请求无意间触发接口副作用。这两点现在不需要写代码，只是提前记录，避免以后重新踩坑。

### 9.2 不主动验证接口存活

纯静态提取，不对识别出的接口发起任何额外的验证请求。

### 9.3 不引入 AST/JS Parser 依赖

见 7.1 节。当前诉求下正则的命中率与维护成本已经匹配，AST 方案的开发和长期维护代价与当前收益不成比例。

### 9.4 不做基于 CDP 网络拦截的运行时接口捕获

`page.HijackRequests`/CDP `Network.responseReceived` 能捕获运行时真实发出的 XHR/Fetch 请求，比正则扫源码更准（能捕获运行时动态拼接、正则扫不出的 URL），但需要托管一次完整浏览器交互周期、处理请求去重、处理请求体，是量级更大的能力。先落地"扫 JS 源码"这个低成本高收益的版本，若生产反馈"正则扫描遗漏了大量运行时动态拼接的接口"，再考虑升级路径。

### 9.5 规则暂不外置为配置文件

区别于 URLFinder 把提取正则、过滤黑名单做成 YAML 外置配置的设计。本方案现阶段不采用，原因：

- 与 `web/crawler/leak.go` 现有的敏感信息规则（编译期常量）保持一致的处理方式，不在同一类问题上搞两套不同的规则管理机制；
- 规则外置本身是有代价的（配置文件解析、格式校验、错误处理、文档同步），现阶段规则数量少、改动频率低，这个代价没有对应的收益；
- 如果未来规则数量膨胀到需要频繁调整、且不方便每次都走发布流程，才是把规则拆成外置配置的合适时机，现在做属于提前优化一个还不存在的问题。

---

## 十、与已归档方案（`drop/Web-JS接口提取方案.md` v2.0）的关系

已归档方案是在"`crawler` 包下沉为 `core/lib/crawler` 通用模块供多个扫描器共享"这一架构前提下设计的，该前提已被验证不可行并完整回退（见 `00.原子扫描器开发指南[定稿].md` 第 0.1 节）。评估结论：**约 60% 内容依然成立，直接继承；约 40%（主要是抓取实现的挂载方式）因架构前提被推翻而作废，本方案重新设计**。

### 10.1 直接继承（与"谁来抓取"这个架构问题无关，纯粹是方案设计判断）

1. `links` 与 `APIInfo` 的类型区分（本方案第六节）；
2. `APIInfo`/`ApiResult` 的字段设计（本方案第六节，`ApiResult` 骨架字段已在代码中，本方案是追加而非推翻）；
3. 三层置信度规则 + 提取/过滤解耦的架构（本方案第七节）；
4. "不反哺 BFS、不做主动验证、不引入 AST、不做 CDP 拦截"等边界判断（本方案第九节）。

### 10.2 重新设计（因抓取实现的架构前提被推翻）

1. **JS 源码获取方式**：原方案假设"复用 `web/crawler` 已经抓到的 body"，本方案改为 `ApiScanner` 自己独立发起抓取（本方案第八节）；
2. **集成点**：原方案是"`ApiScanner.Run` 调用共享的 `crawler.FetchAndCrawl`"，本方案改为 `ApiScanner` 在自己包内独立实现 `fetch.go`/`bfs.go`（本方案第三、四节）；
3. **深度爬取的默认行为**：原方案未明确讨论"默认开不开"，本方案结合 `ApiScan` 自身语义明确为"默认总是开启"，并同步删除了骨架阶段照搬来的 `Crawl` 三态开关（本方案第五节）。

---

## 十一、待实施文档阶段确认的事项

以下事项本方案有意不做定论，留给实施文档阶段结合真实站点测试再决定，避免方案阶段拍脑袋定死：

- `CrawlDepth` 的默认值（`WebScanner` 默认 2，但 `ApiScan` 的目标是尽可能挖到接口，可能需要更深，需实测验证耗时与收益的平衡点）；
- `bfs.go` 的具体队列参数（并发数、单页超时、MaxPages 硬上限）是否需要与 `web/crawler` 的默认值保持一致，还是需要针对 `ApiScan` 的场景单独调优；
- 三层置信度规则的具体正则细节是否需要针对实测站点做补充调整；
- **骨架代码遗留的文档引用清理**：当前 `result_types.go`、`scan_api.go`、`cmd/agent/scan/api.go` 三处骨架注释都写着"由 `docs/API扫描-js提取/Web-JS接口提取实施文档.md` 补充"——这份文件当时是预告的文件名，实际方案已产出为本文档（《API扫描功能设计.md》），配套的实施文档尚未产出、最终文件名以实际产出为准。实施文档落地时需要顺带把这三处骨架注释里的引用路径更新为真实的实施文档路径，避免注释长期指向一个不存在的文件。

---

## 十二、后续步骤

本文档只完成方案论证，不含逐行代码改动步骤。下一步产出配套实施文档，明确：改动文件的具体函数签名、正则规则的最终版本、`ApiScanOptions`/CLI 参数绑定细节、单元测试用例清单（表驱动覆盖高/中/低置信度规则、静态资源排除、去重、BFS 深度控制）、验收标准（`go build`/`go vet`/`go test -race`）。**实施文档的产出时机以用户指令为准，本方案不预设时间线。**
