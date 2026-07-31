# httpx / xray 对 NeoScan 的参考价值评估

> 分析对象：`neoAgent/docs/references/httpx/httpx`、`neoAgent/docs/references/xray/xray`
> 分析目的：判断这两个仓库对 NeoScan 当前的 Web 扫描 / 爬虫 / PoC 漏洞验证三个方向，各自有哪些可落地的参考价值
> 结论性质：纯评估文档，不涉及代码改动

---

## 一、先给结论：谁对应哪个模块

| 参考项目 | 真实定位 | 对 NeoScan 哪个模块有参考价值 |
|---|---|---|
| **httpx** | 单点 HTTP 探测器（存活探测 + 指纹 + TLS + CDN 识别），**不是爬虫，不是漏扫** | 主要对应 `internal/core/scanner/web/web_scanner.go`（Web 指纹扫描），少量对应 `crawler` |
| **xray**（本仓库） | 本仓库**只有 PoC 规则和生态配件，核心引擎（爬虫 + 扫描调度）是闭源的** | 只对 `rules/poc`（PoC 规则设计）有参考价值，对爬虫内核**没有**参考价值 |

这是必须先澄清的认知误区：xray 仓库里根本看不到"爬虫怎么写"，`docs/references/xray/shb/` 下已有的三份分析文档（项目概览 / 项目实现原理 / 项目亮点）已经把这点讲得很清楚，结论可信，本文不重复分析，只做延伸判断。httpx 也从来不是爬虫，它是"输入一批 URL/IP，输出一批探测结果"的批处理工具，不会自己发现新链接。

所以结论很明确：**这两个项目对"爬虫部分"几乎没用，真正有价值的是 web 指纹扫描（参考 httpx）和 PoC 漏洞验证（参考 xray 的规则设计），而 NeoScan 现在恰好在这两块都有明显缺口**。

---

## 二、NeoScan 现状盘点（先看数据结构，再看问题）

```
internal/core/scanner/web/
├── web_scanner.go   # 单页面：go-rod 渲染 + HTTP 降级 + 指纹匹配 + 截图
├── context.go       # 从渲染后的页面提取 DOM/JS/Meta/Cookie
└── crawler/
    ├── crawler.go   # BFS 爬虫，限流、去重、深度控制
    ├── extract.go   # 从 HTML 里抠链接/表单/参数
    └── leak.go      # 空文件（敏感信息泄露检测，设计了但没实现）

rules/
├── fingerprint/web/web_fingerprints.json   # 指纹规则，有数据
└── poc/poc文件.md                           # 空文档，无任何 PoC 规则、无执行引擎
```

`internal/pkg/matcher/matcher.go` 里的 `MatchRule` 目前是一棵 `And/Or` 布尔树，只能做**静态字段的被动比对**（contains/regex/equals…），不能发起新请求、不能保存中间变量、不能等待反连。这是"指纹识别 DSL"，而不是"漏洞验证 DSL"——这两者容易被误认为是同一件事，但能力边界完全不同。

**核心结论一句话**：NeoScan 目前只走到了"爬虫抓攻击面 → 存进 `WebResult.Forms/Params`"这一步，**没有任何代码消费这些攻击面数据去做漏洞验证**。`rules/poc` 是空的，说明这条链路是断的。

---

## 三、httpx 对 NeoScan 的参考价值：Web 指纹扫描模块

### 🟢 强烈建议参考的点

**1. `skipCDNPort`（`runner.go:2982`）—— CDN 识别与扫描降级策略**

```go
// 命中 CDN IP 时，只允许扫 80/443，非标准端口直接跳过
if isCdnIP && port != "80" && port != "443" {
    return true // skip
}
```

NeoScan 现在完全没有 CDN 识别能力。如果目标 IP 是 CDN 节点，对它做全端口 Web 扫描（截图、爬虫、指纹）纯属浪费资源，而且扫到的是 CDN 边缘节点的内容，不是真实源站，**产出的结果是噪音甚至误报**。这是一个真问题（不是臆想），httpx 用 `projectdiscovery/cdncheck` 库 + IP 段判断一次性解决，值得原样借鉴思路（不一定要引入同一个库，逻辑可以自己实现：解析 IP → 查 CDN IP 段 → 非 CDN 才继续深度扫描）。

**2. `wappalyzergo` 的指纹引擎选型 —— 你可能在"重新发明轮子"**

httpx 自己不写指纹匹配逻辑，直接依赖独立库 `github.com/projectdiscovery/wappalyzergo`（Wappalyzer 规则的 Go 移植版，规则库持续维护、覆盖上千种技术栈）。而 NeoScan 的 `internal/pkg/matcher` + `fingerprint/engines/http` 是**自己从零写的一套等价系统**（`And/Or` 树 + `contains/regex/equals` 操作符），规则数据只有一个 `web_fingerprints.json`。

这里要犀利地指出问题：`web/README.md` 里写"兼容 Wappalyzer 识别逻辑"，但事实上是自造了一套语法去"模拟" Wappalyzer，而不是直接用 Wappalyzer 的规则集（社区维护的 3000+ 条规则，现在肯定远不到这个数量）。

【核心判断】✅ 值得做：直接引入 `wappalyzergo` 库作为指纹识别的备选/补充引擎，而不是继续手工扩充自己的 JSON 规则文件。

【关键洞察】
- 数据结构：`wappalyzergo` 输入是 `(headers, body []byte)`，输出 `map[string]AppInfo`，接口比现在的 `matcher.MatchRule` 树更简单
- 复杂度：可以消除自己维护指纹规则库的长期成本（社区规则每天在更新，手动补全跟不上）
- 风险点：`matcher.go` 的 And/Or 树目前也被别的地方复用（比如端口服务识别），不能整体删除，只能在 web 指纹这条线上叠加/替换

**3. `title.go` 的 DOM 解析优先 + 正则兜底模式**

httpx 提取 Title 用 `golang.org/x/net/html` 做真实 DOM 解析，解析失败才退化到正则 `reTitle.FindAllString`。`crawler/crawler.go` 的 `extractTitle` 和 `web_scanner.go` 的 `fallbackScan` 里的 title 提取全部是手写字符串查找（`strings.Index("<title>")`），**这是能跑但不严谨的实现**，遇到 `<TITLE class="x">` 这种带属性的标签、或者 title 跨行、或者有 HTML 实体转义（`&amp;`）时会出错。而 `extract.go` 已经引入了 `goquery`（基于 `golang.org/x/net/html`），完全可以复用同一个依赖把 title 提取也换成 `doc.Find("title").Text()`，成本几乎为零。

**4. `MaxResponseBodySizeToRead` 的设计** —— 和现在的做法一致（2MB 硬限制），不需要改，只是确认这个思路是对的。

### 🟡 酌情参考

**TLS 证书抓取**（`tls.go`）：提取证书是否过期、自签名、通配符证书、指纹哈希。如果 NeoScan 未来要做资产测绘/证书类风险识别，这块可以整体照抄字段设计，但现阶段如果没有明确需求，不建议现在就加——这是"解决还没出现的问题"。

### 🔴 不建议参考

`runner.go` 整体（3000+ 行）高度耦合 CLI 参数、断点续传、多种输出格式（JSON/CSV/MD/HTML）、resume 机制。这是一个独立命令行工具的"胖 Runner"模式，和 NeoScan Agent 内部作为一个 Scanner 组件被 Pipeline 调度的架构完全不匹配，**没有参考价值，不要照搬这种大 Runner 结构**。

---

## 四、xray 对 NeoScan 的参考价值：PoC 漏洞验证引擎（目前是空白）

`docs/references/xray/shb/` 下的三份分析文档（项目概览/实现原理/亮点分析）质量很高，结论可信，这里不重复，直接给出**对 NeoScan 当前状态的针对性判断**。

### 真问题诊断

NeoScan 现在的链路是：

```
爬虫抓到 Forms/Params（攻击面）→ 存进 WebResult → 然后呢？没有然后了
```

`rules/poc/poc文件.md` 是空文件，`internal/pkg/matcher` 的 `MatchRule` 只能做静态字段匹配，**没有"发包 → 判断 → 提取变量 → 二次发包"这种多步骤攻击链的执行能力**，也没有反连（OOB）机制。这意味着：即使爬虫抓到了 100 个表单、500 个带参数的 URL，NeoScan 目前**没有任何模块能真正验证这些输入点是否存在漏洞**。这是一个货真价实的能力断层，不是臆想的问题。

### xray PoC DSL 的核心可抄结构（三段式）

```yaml
set:        # 变量池：反连句柄、随机数
rules:      # 若干条"发包 -> CEL表达式判断 -> 提取output变量"
expression: # 用 && || 组合多条 rule 的布尔结果
```

对应到 Go 实现，最小可行方案：

1. **表达式引擎选型**：`github.com/expr-lang/expr`（轻量，比 `cel-go` 上手快，个人/小团队项目更合适）
2. **规则 Schema** 直接搬 xray 三段式，字段名可以保持中文团队习惯，但语义结构不要变
3. **反连机制**：不需要一开始就搭 DNSLog 平台，最简实现是自建一个 HTTP 回调接收端点 + `map[string]bool` 记录是否被访问，`Wait(timeout)` 轮询这个 map 即可，成本很低
4. **`manual: true` 的风险分级思路值得直接抄**：高危 PoC（RCE/写文件类）默认不在被动扫描/爬虫模式下自动触发，必须显式指定，这是安全边界，不是可选项

### 落地判断

【核心判断】✅ 值得做，但顺序要对：**不要先搭 PoC 引擎，先把爬虫抓到的表单/参数利用起来做最基础的验证（比如先做被动的敏感信息泄露检测——也就是先把已经空着的 `leak.go` 填上），再考虑主动 PoC 验证引擎**。

理由：`leak.go` 已经在架构文档（`Web爬虫与被动分析器架构方案-sonnet5-v3.0.md`）里设计好了 `LeakInfo` 数据结构、`WebResult.Leaks` 字段也留好了，但代码是空的——这是"已经画好图纸但没盖的房子"，投入产出比最高，应该先补这里。PoC 主动验证引擎（会真正发起攻击性请求）复杂度和风险都更高，属于下一阶段的事，现在就去抄 xray 的 CEL+反连全套，是在"给还没使用的地基之上先造第二层"，本末倒置。

### 不建议照搬的部分（xray 借鉴指南里也提到了，本文认同）

- 不要一开始就上 CEL/expr 表达式引擎，如果短期内不会有"非开发人员贡献规则"这个真实场景，直接写 Go 代码判断分支更直接
- 不要照搬 evilpot 这种高成本对抗测试靶场，除非指纹/PoC 规则库已经膨胀到需要自动化回归的规模

---

## 五、优先级建议

按"先解决当下最痛的问题"排序：

1. **先补 `leak.go`**（敏感信息泄露被动检测）——数据结构已就位，纯粹是没写代码，投入最小
2. **Web 指纹引擎评估是否切换/补充 `wappalyzergo`**——现在自己维护的规则库大概率覆盖不全，长期看是技术债
3. **加 CDN 识别 + 端口降级策略**（参考 httpx `skipCDNPort`）——避免对 CDN 节点做无意义的深度扫描，性价比高、改动小
4. **表单/参数抓到之后要不要做主动 PoC 验证，是一个产品定位问题**，不是技术问题——如果 NeoScan 定位是"资产测绘 + 被动风险发现"，可以不做主动 PoC；如果定位要覆盖"漏洞扫描"，才需要认真评估引入类 xray 的 PoC DSL + 反连引擎，这是一个不小的工程量，需要专门立项，不应该顺手加

【风险点提醒】：如果决定做主动 PoC 验证，一定要有 `manual`/风险分级这层安全阀（xray 全部 352 条规则都标 `manual: true` 不是偶然），否则爬虫顺手爬到的每个参数都被自动拿去跑 SQLi/RCE payload，对目标是有破坏性风险的，这是产品层面的红线，不是代码质量问题。

---

## 六、延伸阅读

- `docs/references/xray/shb/项目概览.md`：xray 仓库结构与开源/闭源边界说明
- `docs/references/xray/shb/项目实现原理.md`：PoC YAML DSL 语法、webhook 插件系统、evilpot 靶场等逐模块拆解
- `docs/references/xray/shb/项目亮点.md`：按"好品味"标准对 xray 各模块打分
- `docs/references/xray/shb/爬虫开发借鉴指南.md`：面向自研爬虫的可落地决策清单（含不建议照搬的部分）
