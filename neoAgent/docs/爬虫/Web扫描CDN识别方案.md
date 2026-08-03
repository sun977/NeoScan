# Web 扫描 CDN 识别方案

> 文档版本：v1.4
> 修改时间：2026-08-03（新增行号漂移提示：第二节的行号是撰写时的快照，实际编码请以实施文档为准；补充与实施文档的互链）
> 分析对象：`internal/core/scanner/web/web_scanner.go`、`internal/core/model/result_types.go`、`internal/pkg/utils/ip.go`、`rules/fingerprint/web/`（现有规则文件加载模式参照）
> 文档性质：方案设计文档，负责论证"要不要做、怎么做的取舍"；具体到每一步怎么改代码，见配套的 [`Web扫描CDN识别实施文档.md`](./Web扫描CDN识别实施文档.md)
> 关联文档：[`httpx与xray参考价值评估.md`](./httpx与xray参考价值评估.md)（问题最早在此文档中被指出）、[`Web扫描CDN识别实施文档.md`](./Web扫描CDN识别实施文档.md)（本方案的实施步骤拆解，代码开发以该文档为准）

---

## 一、结论先行

【核心判断】
✅ 值得做。这是一个真实存在的问题，不是臆想出来的场景：对 CDN 边缘节点做全套 Web 扫描（截图、BFS 深度爬取、指纹匹配）纯属浪费资源，而且**产出的结果是错的**——扫到的是 CDN 缓存节点返回的内容，不是目标真实源站，这类"看似成功、实则误报"的结果比"没扫到"更危险，会污染整份扫描报告的可信度。

一句话说清方案：**在发起任何昂贵操作（浏览器启动、HTTP 请求）之前，先对目标 IP 查一次本地维护的 CDN 网段表；命中则跳过截图和深度爬取，只做基础的首页探测，并在结果里标注"这是 CDN 节点"。**

| 维度 | 现状 | 方案后 |
|---|---|---|
| CDN 识别能力 | 无，`web_scanner.go` 全文没有任何 CDN 相关判断代码 | 有，基于 IP 段查表 |
| 命中 CDN 后的行为 | 不适用（无法识别） | 保留首页基础探测，跳过截图 + BFS 深度爬取 |
| 数据来源 | 无 | 静态 CIDR 网段表，参照 `rules/fingerprint/web/` 现有的外置 JSON 规则文件模式 |
| IP 格式判断 | ~~`web_scanner.go` 自带一个简陋的 `isIP`（点号计数）~~ | 🟢 **已完成**：`web_scanner.go` 已改为调用 `utils.IsIP`，简陋实现已删除（见第三节） |
| CIDR 包含判断 | ~~无 import 关系，`utils` 包内私有函数不能跨包调用~~ | 🟢 **已完成**：`isIPInCIDR` 已导出为 `utils.IsIPInCIDR`，`web` 包可直接调用（见第三节） |
| 新增第三方依赖 | 无 | 无（不引入 `projectdiscovery/cdncheck` 等库） |
| 判断时机 | 不适用 | `normalizeURL` 之后、浏览器/HTTP 请求发起之前 |

---

## 二、问题从哪来：现状代码里 IP 信息拿到手的时机太晚

`runOnePort` 是当前 Web 扫描单端口探测的核心函数（`web_scanner.go` 第177行起）。梳理它的执行顺序（**下方行号是撰写本节时的快照，此后 `isIP` 清理等改动已让实际行号漂移几行；真正动手写代码时请以 [`Web扫描CDN识别实施文档.md`](./Web扫描CDN识别实施文档.md) 第七节引用的当前行号为准，本节保留原行号只是为了如实记录分析过程**）：

```
196: protocolGuessed := isProtocolGuessed(task.Target, protocolHint)
197: targetURL := normalizeURL(task.Target, port, protocolHint)
     ↓
214: 启动浏览器（go-rod）
219: 注册网络事件监听
252: page.Navigate(targetURL)                ← 真正发起请求
224: remoteIP = e.Response.RemoteIPAddress   ← IP 信息此时才从网络事件里拿到
     ↓
266: 截图（如果 task.Params["screenshot"] 为 true）
     ↓
（Run() 外层，crawler.Crawl）触发 BFS 深度爬取
```

关键问题：**`remoteIP` 要等到网络事件回调触发才拿得到**（第224行），这时候浏览器已经启动、页面已经渲染完成。哪怕这时候才判断出"这是 CDN"，前面最贵的开销（浏览器启动、页面渲染）已经花出去了，判断毫无意义，属于"马后炮"式检测。

真正该做判断的位置在第196~197行**之后**、发起浏览器/HTTP 请求**之前**——此时已经拿到 `task.Target`（可能是域名，也可能已经是 IP），需要的只是提前做一次 DNS 解析（如果 `task.Target` 已经是 IP 则跳过这一步），再查一次 CDN 网段表。

---

## 三、IP 判断基础设施：两处技术债已提前清理完成

写这份方案时排查发现项目里存在重复/受限的 IP 判断实现，属于阻塞 CDN 判断逻辑落地的前置技术债。这两处已经在方案定稿后单独修复，**不再是待办**，记录如下供后续实施 CDN 功能时直接使用。

### 3.1 〔已修复〕`web_scanner.go` 曾经自带的简陋 `isIP`

修复前的实现（已删除）：

```go
func isIP(target string) bool {
	// 简单判断，实际可以使用 net.ParseIP
	return strings.Count(target, ".") == 3 && !strings.ContainsAny(target, "abcdefghijklmnopqrstuvwxyz")
}
```

这个函数连注释自己都写了"实际可以使用 `net.ParseIP`"，说明作者当时是知道更好的写法但图省事没做。它至少有两个真实缺陷：不支持 IPv6；对 `"1.2.3.a"` 这种非法格式会因为字母判断而误判为非 IP，但对 `"999.999.999.999"` 这种数字越界的非法 IP 会误判为"是 IP"（只判断了点号数量和是否含字母，没有真正校验数值范围）。

**现状**：`isIP` 已从 `web_scanner.go` 删除，唯一调用点 `resolveIPPortForResult` 已改为调用下面 3.2 节的 `utils.IsIP`，`web_scanner.go` 已新增 `"neoagent/internal/pkg/utils"` import。

### 3.2 `internal/pkg/utils.IsIP`——现在 `web` 包已在直接复用的正确实现

```go
// internal/pkg/utils/ip.go
func IsIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
```

导出函数，用标准库 `net.ParseIP` 做校验，同时支持 IPv4/IPv6，没有 3.1 节旧实现的两个缺陷。**CDN 判断逻辑正式实施时应继续复用这个函数**，不要在 `web` 包内再写第三个 IP 判断实现。

### 3.3 〔已导出〕`internal/pkg/utils.IsIPInCIDR`——CDN 网段判断可直接调用的核心原语

修复后的实现（已从私有 `isIPInCIDR` 导出为公开 `IsIPInCIDR`）：

```112:120:neoAgent/internal/pkg/utils/ip.go
// IsIPInCIDR 检查IP是否在CIDR范围内。导出为公开函数，供包外需要做
// IP 网段匹配的场景直接复用（如 Web 扫描 CDN 网段识别），避免各处重复实现
// net.ParseCIDR + Contains 的判断逻辑。
func IsIPInCIDR(clientIP net.IP, cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false // CIDR格式无效
	}
	return ipNet.Contains(clientIP)
}
```

这正是 CDN 网段判断所需要的核心原语——"给一个 IP，判断是否落在某个 CIDR 网段内"，实现只有 6 行：`net.ParseCIDR` 解析网段，`ipNet.Contains` 做包含判断。原先是未导出的私有函数，`internal/core/scanner/web` 包无法跨包调用；现在已导出，**CDN 判断逻辑正式实施时直接调用 `utils.IsIPInCIDR` 即可**，不需要在 `web` 包或新的 `cdn` 包里重新实现一遍。

包内唯一调用方 `IsIPInWhitelist` 已同步改用新的导出名，行为未变。改名时顺带把同一个函数分组里的 `isIPEqual`（单 IP 精确匹配）也一并导出为 `IsIPEqual`，保持这一组 IP 判断辅助函数的可见性风格一致——这个函数本次 CDN 方案用不上，但既然要导出就不留一个风格不一致的漏网之鱼。

`web` 包在此次修复后已经 import `internal/pkg/utils`（此前是全新依赖引入点，现已引入），后续 CDN 功能直接复用即可，不需要再单独处理这次引入。

---

## 四、落地方案

### 4.1 数据从哪来：CDN 网段表

不引入 `projectdiscovery/cdncheck` 等第三方库，原因：
- 该库大部分能力（Cloud 服务商细分识别、WAF 类型细分）当前用不上，只需要"是不是 CDN"这一个布尔值。
- 《Agent重构开发总纲》明确"Native First"原则，能自己实现的轻量能力优先自己写，不为了省事引入整库的维护负担（版本升级、License 合规、体积膨胀）。
- 真正有技术含量的部分是网段数据本身（各 CDN 厂商公开维护的 IP 段列表），不是查表这几行代码，数据可以借鉴/抓取，第三方库的代码没必要照搬依赖。

规则文件的组织方式参照项目里已有的先例——`rules/fingerprint/web/web_fingerprints.json`（外置 JSON + 内置兜底的加载模式），新增一个独立目录：

```
rules/edge/cdn.json
rules/edge/waf.json   （预留，本方案不实现，见下方说明）
```

目录取名 `edge`（边缘节点）而不是 `cdn`：CDN 和 WAF 都是部署在目标源站前面的"边缘网络组件"，是同一个能力域下的两类判断（本方案只落地 CDN 这一类，`waf.json` 先占位，留给后续识别 WAF 拦截页/响应头特征时使用，避免临时再拆目录）。两类数据判断方式不同、更新节奏也可能不同，拆成两个文件而不是塞进一个文件里用字段区分，职责更清晰。

`cdn.json` 内容格式示例（数据来源为各大 CDN 厂商公开发布的 IP 段页面，属于静态数据，不需要实时查询任何第三方 API）：

```json
[
  { "provider": "Cloudflare", "cidrs": ["173.245.48.0/20", "103.21.244.0/22"] },
  { "provider": "Akamai", "cidrs": ["23.32.0.0/11", "104.64.0.0/10"] },
  { "provider": "Fastly", "cidrs": ["151.101.0.0/16"] }
]
```

### 4.2 判断逻辑放在哪、什么时候触发

在 `runOnePort` 第196~197行之后新增一步（伪代码，仅示意插入位置，不代表最终实现）：

```go
protocolGuessed := isProtocolGuessed(task.Target, protocolHint)
targetURL := normalizeURL(task.Target, port, protocolHint)

// 新增：CDN 判断，发起任何请求之前
targetIP := task.Target
if !utils.IsIP(task.Target) {
    targetIP = resolveFirstIP(task.Target) // 新增的小函数：域名解析，解析失败则跳过 CDN 判断，按老路径继续
}
isCDN, cdnProvider := cdnDetector.Check(targetIP) // cdnDetector 内部按 rules/edge/cdn.json 逐条调用 utils.IsIPInCIDR（已导出，可直接跨包调用）
```

命中 CDN 之后**不是直接放弃扫描**，而是记录一个信号，照旧探测首页拿到基础的状态码/标题/Server 头，但跳过两项开销最大、且在 CDN 场景下产出价值最低的动作：
1. 跳过截图（`task.Params["screenshot"]` 即使为 `true` 也不执行）
2. 跳过 BFS 深度爬取（不触发 `crawler.Crawl`）

这样用户依然能看到"这个端口有响应、疑似是 CDN"，而不是整条结果凭空消失——比直接跳过更贴合安全扫描"宁可多报、不可漏报"的诉求，也不会让用户误以为这个端口探测失败了。

### 4.3 数据结构改动：只加字段，不改类型

参照架构方案里"决策1"一贯的原则（爬虫新增字段时的做法），`WebResult` 新增两个 `omitempty` 字段，不影响现有序列化和下游消费方：

```go
// internal/core/model/result_types.go，WebResult 新增
IsCDN       bool   `json:"is_cdn,omitempty"`
CDNProvider string `json:"cdn_provider,omitempty"` // 命中的 CDN 厂商名，如 "Cloudflare"
```

---

## 五、破坏性分析

逐条确认零破坏：

1. **`WebResult` 只做字段新增**，不删不改已有字段，JSON 序列化向后兼容，CSV/表格 Reporter 不需要改（新字段按需展示，属于锦上添花，不在本方案范围内）。
2. **`runOnePort` 签名不变**，CDN 判断是函数体内部新增的一段前置逻辑，不影响调用方 `Run()`。
3. **默认行为**：CDN 网段表如果加载失败或为空，判断函数应直接返回"不是 CDN"，退化为现状行为（不影响任何现有扫描逻辑），不能因为规则文件缺失导致扫描任务失败。
4. **`utils` 包导出函数**：`isIPInCIDR` 改名为 `IsIPInCIDR` 属于纯提升可见性，函数体和包内现有调用方 `IsIPInWhitelist` 的行为均未改变，已实测 `go build ./...` 通过。

---

## 六、现阶段不做，但已规划后续路径的部分

### 6.1 云服务商与 CDN/WAF 的细分识别（真不建议做，不纳入后续规划）

当前诉求只是"要不要跳过深度扫描"这一个二元判断，云服务商 IP 段的识别是另一个更复杂的话题（云服务器本身也可能是真实源站，不能一刀切跳过），不在本方案范围内。

### 6.2 CDN/WAF 规则的自动更新机制（现阶段不做，但不是"永远不做"）

这一项与 6.1 性质不同：6.1 是需求不存在，这一项是需求存在、但排期依据明确——**Agent-Master 通信本身还没开发**，当前阶段是先把 Agent 侧独立功能做完，再统一联通 Master。规则自动更新依赖的正是这条通信链路，链路不存在，同步机制无从谈起，这不是"锦上添花可以往后拖"，而是"物理上做不了"。

现状核实（非推测）：Master 侧在 `neoMaster/internal/service/agent/update.go` 已经实现了一套**类型无关的通用规则同步基础设施**——`RuleType` 枚举 + `rulePathMap` 路径映射 + 通用快照打包/加密/签名逻辑，当前已接入 `fingerprint`/`poc`/`virus`/`webshell` 四种类型，路由也已注册（`GET /api/v1/agent/rules/{type}/version`、`.../download`，见 `neoMaster/internal/app/master/router/agent_routers.go`）。但这套协议目前是 Master 单方面的"纸面协议"，Agent 侧尚未开发对应的通信客户端——这与"Agent-Master 通信还没开发"的现状一致，不冲突。

**结论：等 Agent-Master 通信打通时，规则同步应该做成一次性的通用能力（版本比对 → 下载 → 解密验签 → 落盘 → 触发对应规则包 Reload），类型无关，指纹/POC/CDN/WAF 共用同一套客户端逻辑，新增一种规则类型只是多注册一行 `RuleType`，不需要每种类型单独写一遍同步代码。CDN/WAF 到时候是顺带接入的第五个类型，不需要现在为它单独排期，也不需要现在动手写任何同步相关的代码。**

现阶段唯一需要注意的一点（不是新增工作量，是对第四节实现方式的一个约束）：`rules/edge/cdn.json` 的加载逻辑要设计成可以重复调用的 `Load()`/`Reload()`，而不是只能在进程启动时 `sync.Once` 执行一次（参照 `web_scanner.go` 现有 `ensureInit` 的模式即可，两者半斤八两，不需要额外设计）。这样将来通信打通后，同步客户端只需要"下载文件覆盖本地路径 + 调用一次 `Reload()`"，不需要回头改这次新写的加载逻辑本身。

---

## 七、实施范围小结

改动量与此前 Sprint 6（协议自适应双发选优）是同一量级：

1. ~~`internal/pkg/utils/ip.go`：新增一个导出的 CIDR 匹配函数~~ 🟢 **已完成**（`IsIPInCIDR`，见第三节）。
2. 新增 `rules/edge/cdn.json` 规则文件 + 对应的加载/查询逻辑（可放在 `internal/pkg/utils` 或新建一个轻量的 `internal/pkg/edge` 包，视规则文件加载逻辑复杂度决定，倾向于新建 `internal/pkg/edge` 包，与 `internal/pkg/fingerprint` 的组织方式保持一致，为后续 `waf.json` 接入预留同一个包）；加载入口需设计成可重复调用的 `Load()`/`Reload()`，不要只能进程启动时 `sync.Once` 执行一次（原因见第六节 6.2，为后续 Master 同步预留接口，不增加本次工作量）。**待实施**。
3. `internal/core/model/result_types.go`：`WebResult` 新增 `IsCDN`/`CDNProvider` 两个字段。**待实施**。
4. `internal/core/scanner/web/web_scanner.go`：`runOnePort` 内插入判断点。**待实施**；~~顺带把现有 `isIP` 替换为调用 `utils.IsIP`~~ 🟢 **已完成**（见第三节）。

不需要新增 Scanner、不需要新增 `TaskType`，可以作为 Phase 5.1 之后一个独立的小任务排期，不需要和 Phase 5.2（Vuln Scanner）混在一起做。剩余待实施部分是第 2/3/4 项（网段规则文件与加载逻辑、`WebResult` 字段、`runOnePort` 判断点接入）。
