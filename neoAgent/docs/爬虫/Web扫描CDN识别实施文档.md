# Web 扫描 CDN 识别实施文档

> 文档版本：v1.0
> 修改时间：2026-08-03
> 依据方案：[`Web扫描CDN识别方案.md`](./Web扫描CDN识别方案.md)（本文档只负责把方案第四节"落地方案"拆成可以直接照做的步骤，不重复方案的背景论证，结论有分歧以方案文档为准）
> 文档性质：实施清单，后续 CDN 识别功能的代码开发按本文档的步骤顺序进行；每步给出改动文件、改动内容、验收标准

---

## 零、实施前必读的两条硬约束

来自方案文档第三、六节，写代码前必须遵守，不是建议：

1. **复用已导出的 IP 判断函数，不写第三个实现**：`utils.IsIP`（`internal/pkg/utils/ip.go` 第 380 行）判断字符串是否为合法 IP；`utils.IsIPInCIDR`（同文件第 114 行）判断 IP 是否落在某个 CIDR 网段内。两个函数都已导出，`web` 包也已经 import `internal/pkg/utils`（`web_scanner.go` 第 27 行）。**新代码里不允许出现第三个 `isIP`/`isIPInCIDR` 风格的判断实现**。
2. **规则加载入口必须可重复调用**：新写的 `edge` 包规则加载函数要设计成 `Load()`/`Reload()`，不能只能在进程启动时通过 `sync.Once` 执行一次。原因见方案文档第 6.2 节——Agent-Master 通信链路还没打通，但一旦打通，规则同步客户端要能够"下载新文件后重新触发一次加载"，现在如果写成一次性初始化，到时候还要返工。参照 `web_scanner.go` 第 541~581 行 `ensureInit` 的现有写法（内部调用 `s.fpEngine.Reload(rules)`），`edge` 包的加载函数也要保留同样的"读文件 → 反序列化 → 灌入内存态"结构，唯一区别是不要用 `sync.Once` 包一层。

---

## 一、实施步骤总览

严格按顺序执行，后一步依赖前一步的产出：

| 步骤 | 改动文件 | 类型 | 状态 |
|---|---|---|---|
| 1 | `rules/edge/cdn.json` | 新增数据文件 | 🟢 已完成 |
| 2 | `internal/pkg/edge/model.go` | 新建 | 🟢 已完成 |
| 3 | `internal/pkg/edge/detector.go` | 新建 | 🟢 已完成 |
| 4 | `internal/pkg/edge/detector_test.go` | 新建（单元测试） | 🟢 已完成 |
| 5 | `internal/core/model/result_types.go` | 修改（加 `EdgeComponent` 类型 + `EdgeComponents` 字段） | 🟢 已完成 |
| 6 | `internal/core/scanner/web/web_scanner.go` | 修改（接入判断点） | 待实施 |
| 7 | 端到端验收 | 无代码改动，跑真实场景验证 | 待实施 |

不需要新增 `Scanner`、不需要新增 `TaskType`，改动范围完全在 `web` 扫描器内部闭环。

---

## 二、步骤 1：新增 CDN 网段规则文件

### 2.1 目标路径

```
neoAgent/rules/edge/cdn.json
```

`rules/edge/waf.json` 本次不创建（方案第四节已明确 WAF 是预留占位，本方案不实现），只建 `cdn.json`。

### 2.2 文件内容

初版收录主流 CDN 厂商公开发布的 IP 段，格式与方案文档 4.1 节一致：

```json
[
  {
    "provider": "Cloudflare",
    "cidrs": [
      "173.245.48.0/20",
      "103.21.244.0/22",
      "103.22.200.0/22",
      "103.31.4.0/22",
      "141.101.64.0/18",
      "108.162.192.0/18",
      "190.93.240.0/20",
      "188.114.96.0/20",
      "197.234.240.0/22",
      "198.41.128.0/17",
      "162.158.0.0/15",
      "104.16.0.0/13",
      "104.24.0.0/14",
      "172.64.0.0/13",
      "131.0.72.0/22"
    ]
  },
  {
    "provider": "Akamai",
    "cidrs": [
      "23.32.0.0/11",
      "23.192.0.0/11",
      "104.64.0.0/10",
      "184.24.0.0/13"
    ]
  },
  {
    "provider": "Fastly",
    "cidrs": [
      "151.101.0.0/16",
      "199.27.72.0/21",
      "146.75.0.0/16"
    ]
  },
  {
    "provider": "阿里云CDN",
    "cidrs": [
      "42.120.0.0/16",
      "59.82.0.0/16",
      "60.221.0.0/16"
    ]
  },
  {
    "provider": "腾讯云CDN",
    "cidrs": [
      "42.202.0.0/16",
      "119.28.0.0/16"
    ]
  }
]
```

> 数据来源说明：以上网段是各厂商长期公开且相对稳定的头部段，用于打通链路和验证判断逻辑。**不追求收录完整**——方案文档第 6.2 节已明确网段表现阶段手动维护，后续接入 Master 同步时会替换成真实的全量数据源，本次实施只要格式正确、逻辑跑通即可，不要在这一步过度投入时间去抓取穷举全网段。

### 2.3 验收标准

- 文件是合法 JSON，能被 `encoding/json` 正确 `Unmarshal` 到步骤 2 定义的结构体。
- 每个 `cidrs` 数组里的每一项都是合法 CIDR（可以用 `utils.IsCIDR` 逐条自查一遍，不合法的条目会在步骤 3 加载时被跳过并记日志，但不应该在初版数据里带着已知错误提交）。

---

## 三、步骤 2：新建 `internal/pkg/edge` 包 —— 数据结构

### 3.1 为什么新建包，不放进 `utils`

参照方案文档第七节的判断：`internal/pkg/fingerprint` 的组织方式是"规则模型（`model/rule.go`）+ 匹配引擎（`engines/`）"分离，`edge` 包要对齐同样的组织习惯，而不是把 CDN 判断逻辑塞进已经很大的 `utils` 包。`utils` 包只提供最底层、无业务语义的 IP 原语（`IsIP`/`IsIPInCIDR`），"给定一个 IP 判断是不是 CDN"已经是有业务含义的领域逻辑，不属于 `utils` 的职责范围。

### 3.2 文件：`internal/pkg/edge/model.go`

```go
// Package edge 提供边缘网络节点（CDN/WAF）识别能力。
//
// 命名与目录结构对齐 rules/edge/ 规则目录：CDN 和 WAF 都是部署在源站
// 前面的边缘网络组件，属于同一个能力域下的两类判断。本次只实现 CDN，
// WAF 的判断方式大概率不只是 IP 段（可能还要看拦截页/响应头特征），
// 届时在本包内新增文件即可，不需要再拆一个新包。
package edge

// ProviderRanges 是单个厂商的 CDN 网段定义，一一对应 rules/edge/cdn.json
// 里的一个数组元素。
type ProviderRanges struct {
	Provider string   `json:"provider"`
	CIDRs    []string `json:"cidrs"`
}
```

### 3.3 验收标准

- `go vet ./internal/pkg/edge/...` 通过。
- 结构体字段的 JSON tag 与步骤 1 的 `cdn.json` 字段名完全对应（`provider`/`cidrs`）。

---

## 四、步骤 3：新建 `internal/pkg/edge` 包 —— 检测器

### 4.1 文件：`internal/pkg/edge/detector.go`

核心设计要点（对应零、第 2 条硬约束）：

- 内部状态用 `sync.RWMutex` 保护，不用 `sync.Once`——`Load` 可以被多次调用（首次启动加载一次，未来 Master 同步后再调一次），每次调用都完整替换内部规则集，不是追加。
- 查找逻辑直接复用 `utils.IsIPInCIDR`，不重新实现网段包含判断。
- 规则文件缺失或解析失败时，`Load` 返回 error 但不 panic，调用方（`WebScanner.ensureInit`）按现状对指纹规则加载失败的处理方式一样，记警告日志、`Check` 退化为永远返回"不是 CDN"——对应方案文档第五节破坏性分析第 3 条："默认行为...不能因为规则文件缺失导致扫描任务失败"。

```go
package edge

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"neoagent/internal/pkg/utils"
)

// Detector 是 CDN 网段检测器，持有当前生效的规则集，支持并发读取与
// 运行期重新加载（Reload），为后续 Master 规则同步预留接口——见
// docs/爬虫/Web扫描CDN识别方案.md 第 6.2 节。
type Detector struct {
	mu    sync.RWMutex
	rules []ProviderRanges
}

// NewDetector 创建一个空的检测器，需要显式调用 Load 加载规则后才具备
// 判断能力；加载前 Check 恒定返回 false，不影响调用方的正常流程。
func NewDetector() *Detector {
	return &Detector{}
}

// Load 从指定路径读取 CDN 网段规则文件并替换当前生效的规则集。
//
// 可以被重复调用：每次调用都是一次完整替换（不是增量合并），语义上
// 对应"用一份新规则覆盖旧规则"，与 fpEngine.Reload 的语义保持一致。
func (d *Detector) Load(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read cdn rules file %s: %w", path, err)
	}

	var rules []ProviderRanges
	if err := json.Unmarshal(content, &rules); err != nil {
		return fmt.Errorf("unmarshal cdn rules file %s: %w", path, err)
	}

	d.mu.Lock()
	d.rules = rules
	d.mu.Unlock()
	return nil
}

// Check 判断给定 IP 是否命中某个 CDN 厂商的网段，命中则返回
// (true, 厂商名)，未命中或规则集为空返回 (false, "")。
//
// 传入非法 IP 字符串时同样返回 (false, "")，不返回 error——调用方
// （runOnePort）在无法判断时应该按"不是 CDN"处理，继续走原有流程，
// 而不是因为 CDN 判断本身失败就中断整个扫描任务。
func (d *Detector) Check(ip string) (bool, string) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false, ""
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, pr := range d.rules {
		for _, cidr := range pr.CIDRs {
			if utils.IsIPInCIDR(parsed, cidr) {
				return true, pr.Provider
			}
		}
	}
	return false, ""
}
```

### 4.2 与 `utils.IsIP` 的边界

`Detector.Check` 入参约定为"已经确定是 IP 字符串"，内部用 `net.ParseIP` 只是为了拿到 `net.IP` 类型供 `utils.IsIPInCIDR` 使用，**不是重复造一个 `IsIP` 判断**——调用方（步骤 6）在调用 `Check` 之前，应该已经用 `utils.IsIP` 判断过/或完成了域名解析，传进来的就是一个 IP 字符串。

### 4.3 验收标准

- `go build ./internal/pkg/edge/...` 通过。
- 手动构造一个不存在的路径调用 `Load`，确认返回 error 而不是 panic。
- 用步骤 1 文件里的一个真实网段内的 IP（如 `173.245.48.1`，属于 Cloudflare 段）和一个明显不在任何段内的 IP（如 `8.8.8.8`）分别调用 `Check`，确认返回结果符合预期。

---

## 五、步骤 4：单元测试

### 5.1 文件：`internal/pkg/edge/detector_test.go`

覆盖场景（不追求覆盖率数字，追求覆盖方案文档里明确提到的边界情况）：

1. `Load` 成功后，`Check` 命中网段内 IP，返回 `(true, "对应厂商名")`。
2. `Check` 传入不在任何网段的 IP，返回 `(false, "")`。
3. `Check` 传入非法 IP 字符串（如 `"not-an-ip"`），返回 `(false, "")`，不 panic。
4. `Load` 传入不存在的文件路径，返回 error；此时 `Check` 仍然安全返回 `(false, "")`（对应方案第五节"规则文件缺失退化为不影响现状"）。
5. `Load` 被调用两次（模拟未来的 Reload 场景），第二次传入的规则完全替换第一次的结果——用来验证零、第 2 条硬约束确实被遵守（不是只能加载一次）。

测试用例里的 CIDR 数据不依赖步骤 1 的真实文件内容，用测试专属的临时 JSON（`t.TempDir()` + 手写测试固件），避免测试和生产规则数据的变更相互影响。

### 5.2 验收标准

- `go test ./internal/pkg/edge/... -v` 全部通过。

---

## 六、步骤 5：`WebResult` 新增字段

### 6.1 改动位置

`internal/core/model/result_types.go`，`WebResult` 结构体（当前第 101~118 行）。

### 6.2 具体改动

> **设计调整说明**（v1.1）：原方案用 `IsCDN bool` + `CDNProvider string` 两个标量字段。开发过程中发现这个设计有结构性缺陷——CDN 和后续必然要做的 WAF 识别本质是同一类问题（"这个目标背后站着哪个边缘网络组件"），用标量字段对应类型，等 WAF 落地就要再加 `IsWAF`/`WAFProvider`，往后每加一种边缘组件类型就加一对字段，且无法表达"同一目标同时命中 CDN 和 WAF"（现实中 Cloudflare 这类厂商很常见）的情况。改为统一的 `EdgeComponent` 列表，`WebResult` 结构一次定型，未来加 WAF/反 DDoS 检测时这个文件不需要再改。

```go
// --- 以下为爬虫功能新增字段，均为 omitempty，不影响现有序列化 ---
Depth  int        `json:"depth,omitempty"`  // 爬取深度，0 = 首页
Forms  []FormInfo `json:"forms,omitempty"`  // 表单/输入点
Params []string   `json:"params,omitempty"` // URL Query 参数名集合
Leaks  []LeakInfo `json:"leaks,omitempty"`  // 被动泄露检测结果

// --- 以下为边缘网络组件识别功能新增字段，均为 omitempty，不影响现有序列化 ---
EdgeComponents []EdgeComponent `json:"edge_components,omitempty"` // 命中的边缘网络组件列表（CDN/WAF/...），一个目标可能同时命中多个
```

新增一个独立类型（与 `FormInfo`/`LeakInfo` 平级，紧跟在 `LeakInfo` 定义之后）：

```go
// EdgeComponent 描述命中的一个边缘网络组件（CDN/WAF/反 DDoS 等）。
// 一个 WebResult 可以同时命中多个（如 Cloudflare 常见 CDN+WAF 二合一），
// 因此 WebResult 里用切片承载，不用一个组件类型对应一对标量字段。
type EdgeComponent struct {
	Type     string `json:"type"`     // "cdn" / "waf"，字符串而非枚举类型，避免跨包引入类型依赖
	Provider string `json:"provider"` // 厂商名，如 "Cloudflare"
}
```

再加一个便利方法，供调用方（`runOnePort`）只关心"要不要跳过深度扫描"这一个问题，不需要每次都手写遍历：

```go
// IsEdgeNode 判断是否命中任意边缘网络组件。调用方（如 Web 扫描器判断
// 要不要跳过截图/深度爬取）通常不关心具体命中的是 CDN 还是 WAF，只关心
// "这是不是一个边缘节点"，用这个方法即可，不需要遍历 EdgeComponents。
func (r WebResult) IsEdgeNode() bool {
	return len(r.EdgeComponents) > 0
}
```

只追加字段/类型/方法，不改动、不删除任何已有字段。

### 6.3 验收标准

- `go build ./internal/core/model/...` 通过。
- 序列化一个未设置 `EdgeComponents` 的 `WebResult` 实例，确认输出 JSON 里不出现这个字段（`omitempty` 生效，向后兼容旧的下游消费方）。
- 序列化一个 `EdgeComponents: []EdgeComponent{{Type: "cdn", Provider: "Cloudflare"}}` 的实例，确认输出 `"edge_components":[{"type":"cdn","provider":"Cloudflare"}]`。

---

## 七、步骤 6：接入 `web_scanner.go`

这是唯一涉及现有核心流程改动的一步，务必对照方案文档第二节确认插入位置。

### 7.1 `WebScanner` 结构体新增字段

在 `WebScanner` 结构体（`web_scanner.go` 第 34~47 行）新增一个 `edgeDetector`，与现有的 `fpEngine` 并列：

```go
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

`NewWebScanner` 里同步初始化：

```go
func NewWebScanner() *WebScanner {
	bm := browser.NewBrowserManager()
	fpEngine := fpHttp.NewHTTPEngine(nil)

	return &WebScanner{
		browserManager:  bm,
		browserLauncher: browser.NewLauncher(bm),
		fpEngine:        fpEngine,
		edgeDetector:    edge.NewDetector(),
		limiter:         qos.NewAdaptiveLimiter(5, 1, 10),
	}
}
```

新增 import：`"neoagent/internal/pkg/edge"`。

### 7.2 `ensureInit` 里追加 CDN 规则加载

`ensureInit`（第 541~581 行）现在只加载指纹规则，追加一段加载 CDN 规则的逻辑，路径尝试列表复用同一套相对路径试探模式（只是把 `fingerprint/web` 换成 `edge`）：

```go
func (s *WebScanner) ensureInit() {
	s.initOnce.Do(func() {
		// ...现有指纹规则加载逻辑保持不变...

		// 加载 CDN 网段规则
		edgePaths := []string{
			"rules/edge/cdn.json",
			"../rules/edge/cdn.json",
			"../../rules/edge/cdn.json",
			"../../../rules/edge/cdn.json",
			"../../../../rules/edge/cdn.json",
			"../../../../../rules/edge/cdn.json",
			"neoAgent/rules/edge/cdn.json",
		}
		for _, path := range edgePaths {
			if err := s.edgeDetector.Load(path); err == nil {
				logger.Infof("[WebScanner] Loaded CDN rules from %s", path)
				break
			}
		}
		// 全部路径都加载失败：edgeDetector 内部规则集为空，Check 恒定返回
		// false，退化为"没有 CDN 识别能力"，不影响扫描主流程，不需要额外处理。
	})
}
```

> 注意：这里复用同一个 `s.initOnce`，CDN 规则和指纹规则一起在首次 `Run()` 时加载，不需要为 `edgeDetector` 单独搞一个 `sync.Once`——两者都是"扫描器启动时加载一次性静态数据"的同一类操作。`edgeDetector.Load` 本身可重复调用（零、第 2 条硬约束）指的是"这个方法不内置一次性保护"，`initOnce` 是调用方 `ensureInit` 这一层的保护，两者不矛盾：未来接入 Master 同步时，触发重新加载的代码不会经过 `ensureInit`（会直接调 `s.edgeDetector.Load(newPath)`），`initOnce` 不会挡路。

### 7.3 在 `runOnePort` 里插入判断点

对照方案文档第二节确认的位置：`normalizeURL` 之后（第 198 行）、浏览器启动之前（第 215 行）。

在 `targetURL := normalizeURL(task.Target, port, protocolHint)` 之后插入：

```go
protocolGuessed := isProtocolGuessed(task.Target, protocolHint)
targetURL := normalizeURL(task.Target, port, protocolHint)

// --- CDN 判断：发起浏览器/HTTP 请求之前，见 Web扫描CDN识别方案.md 第二节 ---
isCDN, cdnProvider := s.checkCDN(task.Target)
// checkCDN 内部只做网段查表，不需要关心 model.EdgeComponent 的组装细节，
// isCDN/cdnProvider 这两个局部变量只在本函数内部用于控制流程（是否跳过
// 截图/深度爬取），组装成 model.EdgeComponent 放进结果的时机在 7.5 节。
```

新增一个独立的小函数 `checkCDN`（不直接写进 `runOnePort` 主干，保持主干可读性，参照现有 `resolveIPPortForResult` 这类抽取风格）：

```go
// checkCDN 判断 target（可能是域名，也可能已经是 IP）是否落在已知 CDN
// 厂商的网段内。target 是域名时会做一次 DNS 解析取第一个 IP；解析失败
// 时直接返回"不是 CDN"，不阻塞后续流程——CDN 判断是锦上添花的优化，
// 不能因为 DNS 解析失败就影响正常扫描（见方案文档第二节）。
func (s *WebScanner) checkCDN(target string) (bool, string) {
	ip := target
	if !utils.IsIP(target) {
		addrs, err := net.LookupHost(target)
		if err != nil || len(addrs) == 0 {
			return false, ""
		}
		ip = addrs[0]
	}
	return s.edgeDetector.Check(ip)
}
```

新增 import：`"net"`（如果文件里尚未直接引入 `net` 包，需要确认；`net/http`、`net/url` 已经在 import 列表里，`net` 本身未必已引入，需要在实际编码时核实并按需添加）。

### 7.4 命中 CDN 后跳过截图与深度爬取

对照方案文档 4.2 节，命中 CDN 不是提前 `return`，而是让后续逻辑"跳过两个具体动作"：

**跳过截图**：截图逻辑在第 267~273 行：

```go
if capture, ok := task.Params["screenshot"].(bool); ok && capture {
	if buf, errShot := page.Screenshot(true, nil); errShot == nil {
		screenshotB64 = base64.StdEncoding.EncodeToString(buf)
	} else {
		logger.Warnf("[WebScanner] Screenshot failed: %v", errShot)
	}
}
```

改为在条件里叠加 `!isCDN`：

```go
if capture, ok := task.Params["screenshot"].(bool); ok && capture && !isCDN {
	if buf, errShot := page.Screenshot(true, nil); errShot == nil {
		screenshotB64 = base64.StdEncoding.EncodeToString(buf)
	} else {
		logger.Warnf("[WebScanner] Screenshot failed: %v", errShot)
	}
}
```

**跳过深度爬取**：深度爬取的触发逻辑在第 365~366 行：

```go
depth := s.resolveCrawlDepth(task, homeStatusCode, homeHeaders, seedLinks)
if depth > 0 && len(seedLinks) > 0 {
```

改为：

```go
depth := s.resolveCrawlDepth(task, homeStatusCode, homeHeaders, seedLinks)
if depth > 0 && len(seedLinks) > 0 && !isCDN {
```

这里选择在 `if` 条件上叠加判断，而不是让 `checkCDN` 直接改写 `depth`/`task.Params["screenshot"]`，理由：`isCDN` 只是"要不要执行这个动作"的一个独立布尔条件，不应该污染 `resolveCrawlDepth` 的返回语义（`resolveCrawlDepth` 的职责是"计算业务上应该爬多深"，CDN 是另一个维度的"值不值得爬"，两个决策来源不同，混在一个函数里会让 `resolveCrawlDepth` 承担它不该管的职责，这是方案文档第三层复杂度审查要避免的"特殊情况污染正常逻辑"）。

### 7.5 结果里标注 CDN 信息

`buildWebResult` 组装 `model.WebResult` 的位置（第 848~863 行），首页结果需要带上 `isCDN`/`cdnProvider`，但爬虫子页面结果（第 372~377 行的循环）不需要重复标注（同一个端口下所有子页面的 CDN 归属和首页一致，没必要每条子页面结果都重复计算/存储一遍）。

这要求 `buildWebResult` 能接收到命中的边缘组件信息，最小改动方式是加进 `pageData` 结构体（第 764~791 行）：

```go
type pageData struct {
	URL        string
	Depth      int
	StatusCode int
	Title      string
	Body       string
	Headers    map[string]string
	ContentLength int64
	RichContext map[string]interface{}
	Forms      []model.FormInfo
	Params     []string
	Leaks      []model.LeakInfo
	Screenshot string
	Favicon    string

	// EdgeComponents 只有首页调用 buildWebResult 时才会显式传入非空值，
	// 爬虫子页面复用同一个端口的判断结果没有意义（同一端口的边缘节点归属
	// 不会因为爬到第几层页面而改变），子页面调用处保持 nil 即可。
	EdgeComponents []model.EdgeComponent
}
```

`buildWebResult` 内部组装 `model.WebResult` 时（第 848~863 行附近）追加一行：

```go
Result: &model.WebResult{
	// ...现有字段不变...
	EdgeComponents: pd.EdgeComponents,
},
```

首页调用处（第 356~361 行）追加传参，`isCDN`/`cdnProvider` 在这里才第一次组装成 `model.EdgeComponent`（`checkCDN` 本身不依赖 `model` 包，保持 `edge`/`web` 包不反向依赖 `model` 之外的耦合最小化）：

```go
var edgeComponents []model.EdgeComponent
if isCDN {
	edgeComponents = append(edgeComponents, model.EdgeComponent{Type: "cdn", Provider: cdnProvider})
}

homeResult := s.buildWebResult(task, startTime, finalIP, finalPort, pageData{
	URL: targetURL, Depth: 0, StatusCode: homeStatusCode, Title: homeTitle,
	Body: homeBody, Headers: homeHeaders, ContentLength: homeContentLen, RichContext: homeRichCtx,
	Forms: homeForms, Params: homeParams,
	Screenshot: screenshotB64, Favicon: faviconB64,
	EdgeComponents: edgeComponents,
})
```

子页面调用处（第 372~377 行）不改动，`pageData` 里 `EdgeComponents` 保持零值 `nil`，符合 Go 切片零值语义，序列化时 `omitempty` 生效不会输出该字段。

### 7.6 验收标准

- `go build ./internal/core/scanner/web/...` 通过。
- `go vet ./...` 无新增警告。
- 现有 `web_scanner.go` 相关的单元测试（如果有）全部通过，确认没有破坏现状行为。

---

## 八、步骤 7：端到端验收

不写新代码，用真实场景验证整条链路符合方案文档第一节的预期效果。

### 8.1 验收用例设计

| 用例 | 目标 | 目标 IP/域名类型 | 预期结果 |
|---|---|---|---|
| A | 命中 CDN | 找一个已知使用 Cloudflare 的公开域名，或直接构造一个 `rules/edge/cdn.json` 网段内的测试 IP | 扫描结果 `edge_components: [{"type":"cdn","provider":"Cloudflare"}]`；`screenshot` 字段为空；结果集里只有一条首页记录，没有深度爬取产生的子页面记录 |
| B | 不命中 CDN | 之前测试过的 `10.201.28.126`（内网 IP，不在任何公开 CDN 段） | 扫描结果不出现 `edge_components` 字段（`omitempty` 生效，切片为空）；截图/深度爬取行为与 CDN 功能上线前完全一致 |
| C | 规则文件缺失 | 临时改名/移走 `rules/edge/cdn.json` 后跑一次扫描 | 扫描任务正常完成，不因规则文件缺失而报错或崩溃，行为等同于用例 B（退化为"不是 CDN"） |
| D | 域名解析失败 | 构造一个不存在的域名作为 target | CDN 判断跳过（`checkCDN` 内部 `LookupHost` 失败直接返回 `false, ""`），扫描任务按现状行为继续（大概率后续步骤本身也会因为域名不可达而失败，但不应该是被 CDN 判断这一步卡住） |

### 8.2 验收标准

- 用例 A/B/C/D 全部符合预期结果列。
- 用例 B 的执行耗时和功能上线前的历史记录（或重新跑一次未接入 CDN 判断的旧代码作为对照）相比，没有明显变慢——CDN 判断只多一次网段查表（内存操作）+ 可能的一次 DNS 解析，不应该造成可感知的性能回退。

---

## 九、实施完成后需要同步更新的文档

代码全部落地并通过步骤 7 验收后，回填以下文档，不要让文档和代码状态脱节：

1. `Web扫描CDN识别方案.md` 第七节"实施范围小结"：把第 2/3/4 项的"待实施"标记改为"已完成"。
2. `Agent重构开发总纲.md`：视当前 Phase 5.1 之后的排期安排，补充一条 Sprint 记录（参照现有 Sprint 6 的记录格式，一句话说清"做了什么、解决了什么真实问题"）。

本实施文档本身不需要在完成后删除，作为这次改动的历史记录保留。
