# API 扫描（ApiScanner）实施文档

> 文档版本：v1.0
> 修改时间：2026-08-11
> 依据方案：[`API扫描功能设计.md`](./API扫描功能设计.md)（本文档只负责把方案拆成可以直接照做的步骤，不重复方案的背景论证，结论有分歧以方案文档为准）
> 文档性质：实施清单，后续 `ApiScanner` 的代码开发按本文档的步骤顺序进行；每步给出改动文件、具体代码、验收标准。格式对齐 [`Web扫描CDN识别实施文档.md`](../爬虫/Web扫描CDN识别实施文档.md)。

---

## 零、实施前必读的硬约束

来自方案文档第二、五节，写代码前必须遵守，不是建议：

1. **原子隔离**：`internal/core/scanner/api/` 包内的任何 `.go` 文件都不允许 `import "neoagent/internal/core/scanner/web/crawler"` 或 `web` 包的任何符号。架构模式（抓取/BFS/提取/过滤四层职责分离）可以照抄 `crawler` 包的设计思路，但代码必须在 `api` 包内重新写一份，物理独立。`lib/browser`、`lib/network/qos` 是公共基础设施，可以正常 import。
2. **深度爬取默认恒为开启**：不引入 `WebScanOptions.Crawl` 那种三态开关，`ApiScanOptions` 里没有 `Crawl` 字段。`CrawlDepth > 0` 就是唯一的开关信号。
3. **`Target` 输入形态判断逻辑独立实现**：不 import `web_scanner.go` 的 `normalizeURL`，在 `scanner/api/fetch.go` 内写一份行为等价（"URL 优先、`Ports` 兜底"）但物理独立的函数（方案文档 5.1 节）。
4. **提取与过滤强制解耦成两个函数**：`extractAPICandidates` 只管正则命中，`filterAPICandidates` 只管清洗/黑名单/去重，两者不允许合并成一个函数（方案文档 7.3 节）。
5. **不做任何形式的接口存活验证**：`filterAPICandidates` 之后的 `[]APIInfo` 就是终态产出，任何步骤都不允许对 `APIInfo.URL` 发起 HTTP 请求（方案文档 9.1/9.2 节）。

---

## 一、实施步骤总览

严格按顺序执行，后一步依赖前一步的产出：

| 步骤 | 改动文件 | 类型 | 说明 |
|---|---|---|---|
| 1 | `internal/core/model/result_types.go` | 修改 | 追加 `APIInfo` 类型、`ApiResult.APIs`/`APIsTruncated` 字段 |
| 2 | `internal/core/options/scan_api.go` | 修改 | 删除 `Crawl` 三态字段，新增 `MaxJSFiles`，重写 `ToTask()` |
| 3 | `internal/core/scanner/api/extract.go` | 新增 | 三层置信度正则提取，`extractAPICandidates` |
| 4 | `internal/core/scanner/api/extract_test.go` | 新增 | 步骤 3 单元测试 |
| 5 | `internal/core/scanner/api/filter.go` | 新增 | 清洗 + 黑名单 + 去重，`filterAPICandidates` |
| 6 | `internal/core/scanner/api/filter_test.go` | 新增 | 步骤 5 单元测试 |
| 7 | `internal/core/scanner/api/fetch.go` | 新增 | `normalizeTarget`（URL/Ports 判断）+ go-rod 单页抓取 `fetchPage` |
| 8 | `internal/core/scanner/api/fetch_test.go` | 新增 | 步骤 7 单元测试（`normalizeTarget` 部分，`fetchPage` 留到端到端验收） |
| 9 | `internal/core/scanner/api/bfs.go` | 新增 | 独立 BFS 队列 `Crawler`/`Crawl` |
| 10 | `internal/core/scanner/api/bfs_test.go` | 新增 | 步骤 9 单元测试 |
| 11 | `internal/core/scanner/api/api_scanner.go` | 修改 | 补充真实字段与 `Run()` 编排逻辑 |
| 12 | `internal/core/scanner/api/api_scanner_test.go` | 修改 | 更新骨架测试断言，新增编排逻辑测试 |
| 13 | `cmd/agent/scan/api.go` | 修改 | 绑定 `--crawl-depth`/`--max-js-files`，删除 `--crawl` |
| 14 | 端到端验收 | 新增 `api_scanner_e2e_test.go` | 覆盖方案文档核心场景的集成测试 |

不需要新建 `TaskType`、不需要新建/改动 `internal/core/factory/api_factory.go`（已存在且已正确接入 `NewApiScanner()`，构造签名不变）。

---

## 二、步骤 1：数据结构

### 2.1 改动位置

`internal/core/model/result_types.go`，`ApiResult` 结构体（当前第 144~154 行）。

### 2.2 具体改动

在 `LeakInfo` 定义之后、`ApiResult` 定义之前插入 `APIInfo` 类型：

```go
// APIInfo 从 JS/HTML 中提取的接口调用地址。语义上不同于页面导航链接
// （crawler.Page 里的 links）——它描述的是"前端代码运行时会向这个地址
// 发起数据请求"，而不是"这是一个可以被继续爬取的页面"，因此不会、
// 也不应该被塞进 BFS 队列继续抓取。见 API扫描功能设计.md 第六节。
type APIInfo struct {
	URL        string `json:"url"`                  // 接口地址：可能是完整 URL，也可能是相对路径，不在提取阶段强行拼接成绝对地址
	Method     string `json:"method,omitempty"`      // 有明确证据才填（如识别到 axios.post(...) 才填 "POST"），识别不到就留空
	Source     string `json:"source"`                // 来源：inline（内联 <script>）或具体的 js 文件 URL
	Confidence string `json:"confidence"`            // high/medium/low，见 API扫描功能设计.md 7.2 节
}
```

替换 `ApiResult` 定义（原第 144~154 行整体替换）：

```go
// ApiResult 是 ApiScanner 的任务产出。与 WebResult 完全独立，不复用它的
// 任何字段——ApiScanner 不做指纹识别、不截图、不判断 CDN，这些 WebResult
// 字段对 ApiResult 没有意义，见 docs/爬虫/web扫描模块重构文档.md 7.3 节。
type ApiResult struct {
	URL           string    `json:"url"`                      // 本页面 URL
	Depth         int       `json:"depth"`                    // BFS 深度，0 表示首页
	APIs          []APIInfo `json:"apis,omitempty"`           // 从 JS/HTML 中静态提取到的接口调用地址清单
	APIsTruncated bool      `json:"apis_truncated,omitempty"` // 本页引用的外链 JS 文件数超过 MaxJSFiles 被截断
}
```

`URL`/`Depth` 是骨架阶段已有字段，本步骤只追加，不修改其类型/tag。

### 2.3 验收标准

- `go build ./internal/core/model/...` 通过。
- 序列化一个未设置 `APIs`/`APIsTruncated` 的 `ApiResult` 实例，确认输出 JSON 不出现这两个字段（`omitempty` 生效）。
- 序列化一个 `APIs: []APIInfo{{URL: "/api/user", Source: "inline", Confidence: "high", Method: "POST"}}` 的实例，确认输出 `"apis":[{"url":"/api/user","method":"POST","source":"inline","confidence":"high"}]`。

---

## 三、步骤 2：`ApiScanOptions` 重写

### 3.1 改动位置

`internal/core/options/scan_api.go`（整体重写，当前 55 行的骨架版本）。

### 3.2 具体改动

```go
package options

import (
	"fmt"
	"time"

	"neoagent/internal/core/model"
)

// ApiScanOptions 是 ApiScan 任务的 CLI 参数集合。不含 Crawl 三态开关——
// ApiScan 的深度爬取不是可选项，是这个扫描器存在的核心意义，见
// docs/API扫描-js提取/API扫描功能设计.md 第五节。
type ApiScanOptions struct {
	Target     string
	Ports      string
	CrawlDepth int
	MaxJSFiles int
	Output     OutputOptions
}

func NewApiScanOptions() *ApiScanOptions {
	return &ApiScanOptions{
		Ports:      "80,443",
		CrawlDepth: 2,
		MaxJSFiles: 20,
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

	if o.CrawlDepth > 0 {
		task.Params["crawl_depth"] = o.CrawlDepth
	}
	if o.MaxJSFiles > 0 {
		task.Params["max_js_files"] = o.MaxJSFiles
	}

	o.Output.ApplyToParams(task.Params)
	return task
}
```

`CrawlDepth`/`MaxJSFiles` 默认值分别为 2、20——按方案文档第十一节，`CrawlDepth` 默认值本应留到实施阶段结合真实站点测试确定；本步骤先沿用 `WebScanner` 现有默认值 2 让代码可编译可跑通，步骤 14 端到端验收环节用真实站点测试后如需调整，直接改这一处默认值，不影响其它步骤。

### 3.3 验收标准

- `go build ./internal/core/options/...` 通过。
- 构造一个 `ApiScanOptions{Target: "example.com"}`，调用 `ToTask()`，确认 `task.Params["crawl_depth"] == 2`、`task.Params["max_js_files"] == 20`、不存在 `task.Params["crawl"]` 这个 key（旧三态参数已删除）。

---

## 四、步骤 3：`extract.go` —— 三层置信度提取

### 4.1 文件：`internal/core/scanner/api/extract.go`

对应方案文档 7.2 节的三层规则，正则细节直接采用 URLFinder 已验证过的写法（`docs/references/URLFinder/config/config.go` 第 34~45 行）作为高置信度的参考基准，中/低置信度按方案文档的设计新写：

```go
// Package api 的 extract.go：只负责"跑正则、收集命中"，不做任何排除判断。
// 过滤逻辑在 filter.go，两者物理分离，见 API扫描功能设计.md 7.3 节。
package api

import (
	"regexp"
)

// candidate 是提取阶段的中间产物，尚未经过 filterAPICandidates 清洗。
// 字段含义与 model.APIInfo 一致，之所以不直接用 model.APIInfo，是保留
// "提取阶段" 和 "最终产出" 之间一层缓冲——如果未来 APIInfo 需要新增仅供
// 展示、不参与过滤判断的字段，两者可以自由分叉，不需要牵动 extract.go。
type candidate struct {
	URL        string
	Method     string
	Confidence string
}

// 高置信度：匹配明确的 HTTP 调用语句结构，自带 Method 信息。
var highConfidencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`fetch\(['"]([^'"]+)['"]`),
	regexp.MustCompile(`axios\.(get|post|put|delete|patch)\(['"]([^'"]+)['"]`),
	regexp.MustCompile(`\.ajax\(\{[^}]*url:\s*['"]([^'"]+)['"]`),
}

// 低置信度兜底：通用路径字面量，不限定固定前缀（方案文档 7.2 节 v1.1 调整）。
var lowConfidencePattern = regexp.MustCompile(`['"](/[a-zA-Z0-9_\-]+/[a-zA-Z0-9_\-/{}]+)['"]`)

// extractAPICandidates 对一段文本（HTML/内联JS/外链JS）跑三层正则，返回
// 全部命中结果，不做任何排除/清洗——那是 filterAPICandidates 的职责。
//
// pageURL 是这段文本所属的页面 URL（用于中置信度的 scope 锚定，取
// scheme+host 部分）；source 标注这段文本的来源（"inline" 或具体 js 文件 URL），
// 直接写入每条 candidate 供 filterAPICandidates 之后转换成 model.APIInfo.Source。
func extractAPICandidates(text string, pageHost string) []candidate {
	var out []candidate

	out = append(out, extractHighConfidence(text)...)
	out = append(out, extractMediumConfidence(text, pageHost)...)
	out = append(out, extractLowConfidence(text)...)

	return out
}

// extractHighConfidence 命中结构化调用语句。fetch/ajax 的正则只有 1 个
// 捕获组（URL 本身），axios 的正则有 2 个捕获组（Method + URL），用
// re.SubexpNames() 之外的方式统一处理不划算，这里按"最后一个捕获组永远
// 是 URL"的规则统一取值，Method 只有 axios 分支才有意义地填充。
func extractHighConfidence(text string) []candidate {
	var out []candidate
	for _, re := range highConfidencePatterns {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			switch len(m) {
			case 2:
				// fetch / .ajax：m[1] 是 URL，没有 Method 信息。
				out = append(out, candidate{URL: m[1], Confidence: "high"})
			case 3:
				// axios.<method>(...)：m[1] 是 method，m[2] 是 URL。
				out = append(out, candidate{URL: m[2], Method: strings.ToUpper(m[1]), Confidence: "high"})
			}
		}
	}
	return out
}

// extractMediumConfidence 用 regexp.QuoteMeta(pageHost) 拼出"host + 合法
// 路径字符集"的正则，命中的必然是明确指向当前站点自己域名的完整 URL，
// 见 API扫描功能设计.md 7.2 节中置信度规则说明。pageHost 为空（比如 URL
// 解析失败）时直接跳过这一档，不产生任何 candidate。
func extractMediumConfidence(text string, pageHost string) []candidate {
	if pageHost == "" {
		return nil
	}
	pattern := regexp.MustCompile(`https?://` + regexp.QuoteMeta(pageHost) + `[a-zA-Z0-9_\-/?&=.%]*`)
	var out []candidate
	for _, m := range pattern.FindAllString(text, -1) {
		out = append(out, candidate{URL: m, Confidence: "medium"})
	}
	return out
}

// extractLowConfidence 通用路径字面量兜底，误报率最高的一档，产出必须
// 带 Confidence: "low" 标记，交给 filterAPICandidates 的黑名单重点清洗。
func extractLowConfidence(text string) []candidate {
	var out []candidate
	for _, m := range lowConfidencePattern.FindAllStringSubmatch(text, -1) {
		if len(m) == 2 {
			out = append(out, candidate{URL: m[1], Confidence: "low"})
		}
	}
	return out
}
```

> 注意：`extractHighConfidence` 用到 `strings.ToUpper`，需要在 import 里加 `"strings"`。

### 4.2 验收标准

- `go build ./internal/core/scanner/api/...` 通过。
- 见步骤 4 单元测试。

---

## 五、步骤 4：`extract_test.go`

表驱动覆盖三层规则各自的正例和一个跨层交叉场景：

```go
package api

import "testing"

func TestExtractAPICandidates(t *testing.T) {
	cases := []struct {
		name       string
		text       string
		pageHost   string
		wantURLs   []string
		wantConf   string // 断言至少一条命中带有这个 Confidence
	}{
		{
			name:     "fetch高置信度",
			text:     `fetch('/api/user/info')`,
			wantURLs: []string{"/api/user/info"},
			wantConf: "high",
		},
		{
			name:     "axios高置信度带method",
			text:     `axios.post('/api/order/create')`,
			wantURLs: []string{"/api/order/create"},
			wantConf: "high",
		},
		{
			name:     "ajax高置信度",
			text:     `$.ajax({url: '/api/legacy/list', type: 'GET'})`,
			wantURLs: []string{"/api/legacy/list"},
			wantConf: "high",
		},
		{
			name:     "中置信度scope锚定",
			text:     `var base = "https://example.com/api/v2/user";`,
			pageHost: "example.com",
			wantURLs: []string{"https://example.com/api/v2/user"},
			wantConf: "medium",
		},
		{
			name:     "中置信度排除第三方域名",
			text:     `var cdn = "https://cdn.other.com/lib.js";`,
			pageHost: "example.com",
			wantConf: "", // 不应该命中中置信度（域名不匹配）
		},
		{
			name:     "低置信度非固定前缀路径",
			text:     `"url": "/user/getUserInfo"`,
			wantURLs: []string{"/user/getUserInfo"},
			wantConf: "low",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractAPICandidates(tc.text, tc.pageHost)

			gotURLs := make(map[string]bool)
			gotConfSet := make(map[string]bool)
			for _, c := range got {
				gotURLs[c.URL] = true
				gotConfSet[c.Confidence] = true
			}

			for _, want := range tc.wantURLs {
				if !gotURLs[want] {
					t.Errorf("expected URL %q in candidates, got %+v", want, got)
				}
			}
			if tc.wantConf != "" && !gotConfSet[tc.wantConf] {
				t.Errorf("expected at least one candidate with Confidence=%q, got %+v", tc.wantConf, got)
			}
			if tc.name == "中置信度排除第三方域名" && gotConfSet["medium"] {
				t.Errorf("did not expect medium confidence hit for third-party domain, got %+v", got)
			}
		})
	}
}

func TestExtractHighConfidence_MethodCapture(t *testing.T) {
	got := extractHighConfidence(`axios.delete('/api/user/1')`)
	if len(got) != 1 || got[0].Method != "DELETE" {
		t.Fatalf("expected Method=DELETE, got %+v", got)
	}
}
```

### 5.1 验收标准

- `go test ./internal/core/scanner/api/... -run TestExtract -v` 全部通过。

---

## 六、步骤 5：`filter.go` —— 清洗 + 黑名单 + 去重

### 6.1 文件：`internal/core/scanner/api/filter.go`

黑名单直接借鉴 URLFinder `UrlFiler` 经过实战验证的规则（方案文档 7.3 节），静态资源后缀排除规则单独一条正则：

```go
// Package api 的 filter.go：统一做转义清理、静态资源后缀排除、黑名单
// 关键词过滤、去重。只处理 extractAPICandidates 已经产出的 candidate，
// 不做二次提取，见 API扫描功能设计.md 7.3 节。
package api

import (
	"regexp"
	"strings"
)

// staticResourceSuffix 命中即丢弃：这些是静态资源，不是接口。
var staticResourceSuffix = regexp.MustCompile(`\.(js|css|png|jpe?g|gif|svg|woff2?)(\?.*)?$`)

// blacklistPattern 直接借鉴 URLFinder config.go 的 UrlFiler 规则（已经过
// 实战验证），排除低置信度正则容易误伤的 JS 属性访问模式（.src/.href 等）。
var blacklistPattern = regexp.MustCompile(
	`\.js\?|\.css\?|\.jpeg\?|\.jpg\?|\.png\?|\.gif\?|www\.w3\.org|example\.com|<|>|\{|\}|\[|\]|\||\^|;|/js/|\.src|\.replace|\.url|\.att|\.href|location\.href|javascript:|location:|application/x-www-form-urlencoded|\.createObject|:location|\.path`,
)

// filterAPICandidates 对 extractAPICandidates 的产出做统一清洗，返回值
// 已经是去重后的最终结果，调用方（fetch.go/bfs.go 的编排逻辑）拿到之后
// 直接转换成 []model.APIInfo，不需要再做任何二次处理。
func filterAPICandidates(candidates []candidate, source string) []candidate {
	seen := make(map[string]bool, len(candidates))
	var out []candidate

	for _, c := range candidates {
		c.URL = cleanURL(c.URL)
		if c.URL == "" {
			continue
		}
		if staticResourceSuffix.MatchString(c.URL) {
			continue
		}
		if blacklistPattern.MatchString(c.URL) {
			continue
		}

		key := c.URL + "|" + c.Method
		if seen[key] {
			continue
		}
		seen[key] = true

		c.URL = source // 占位，见下方说明
		out = append(out, c)
	}
	return out
}

// cleanURL 做基础的转义清理，对齐 URLFinder jsFilter/urlFilter 的清洗步骤
// （去空白、还原转义斜杠、URL 解码常见的 %3A/%2F）。
func cleanURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, `\/`, "/")
	s = strings.ReplaceAll(s, "%3A", ":")
	s = strings.ReplaceAll(s, "%2F", "/")
	return s
}
```

> **实现时需要修正的一处占位**：上面 `c.URL = source` 是错误示意，正式实现时 `candidate` 需要新增 `Source` 字段（第 4 节 `extract.go` 补一个 `Source string` 字段，由调用方在 `extractAPICandidates` 或更外层统一赋值），`filterAPICandidates` 这里应该是 `c.Source = source`。编码阶段务必对照本节意图修正，不要照抄出现字段名错误的代码。写清楚这一点是因为"提取阶段不关心 Source，编排阶段统一打标"和"提取阶段直接打标"两种设计都合理，具体选择在步骤 11 编排代码里体现，此处不强行锁死。

### 6.2 验收标准

- `go build ./internal/core/scanner/api/...` 通过。
- 见步骤 6 单元测试。

---

## 七、步骤 6：`filter_test.go`

```go
package api

import "testing"

func TestFilterAPICandidates(t *testing.T) {
	input := []candidate{
		{URL: "/api/user/info", Confidence: "high"},
		{URL: "/api/user/info", Confidence: "high"}, // 重复，应被去重
		{URL: "/static/app.js", Confidence: "low"},  // 静态资源，应被排除
		{URL: "location.href", Confidence: "low"},   // 黑名单命中，应被排除
		{URL: "  /api/order/list  ", Confidence: "low"}, // 前后空白，应被清洗
	}

	got := filterAPICandidates(input, "inline")

	if len(got) != 2 {
		t.Fatalf("expected 2 candidates after filter, got %d: %+v", len(got), got)
	}

	urls := make(map[string]bool)
	for _, c := range got {
		urls[c.URL] = true
	}
	if !urls["/api/user/info"] || !urls["/api/order/list"] {
		t.Errorf("unexpected filtered result: %+v", got)
	}
}

func TestFilterAPICandidates_StaticResourceSuffix(t *testing.T) {
	input := []candidate{
		{URL: "/assets/main.css?v=123"},
		{URL: "/assets/logo.png"},
		{URL: "/api/real/endpoint"},
	}
	got := filterAPICandidates(input, "inline")
	if len(got) != 1 || got[0].URL != "/api/real/endpoint" {
		t.Fatalf("expected only /api/real/endpoint to survive, got %+v", got)
	}
}
```

### 7.1 验收标准

- `go test ./internal/core/scanner/api/... -run TestFilter -v` 全部通过。

---

## 八、步骤 7：`fetch.go` —— Target 判断 + go-rod 单页抓取

### 8.1 文件：`internal/core/scanner/api/fetch.go`

对应方案文档 5.1 节（`normalizeTarget`）与第八节（JS 源码获取方式）。抓取部分参照 `web_scanner.go` 的 `Launch -> OpenPage -> Navigate -> WaitLoad` 写法（`web_scanner.go` 第 226~296 行），但不 import `web` 包，独立调用 `lib/browser` 基础设施：

```go
// Package api 的 fetch.go：独立的单页抓取实现（go-rod 渲染 + 内联/外链 JS
// 获取）与 Target 输入形态判断，不 import web/crawler，见
// API扫描功能设计.md 第二节"原子扫描器隔离原则"。
package api

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/pkg/logger"

	"github.com/go-rod/rod"
)

// normalizeTarget 把 Target/Ports 换算成一个可以直接发起抓取的起始 URL。
// 逻辑与 WebScanner 的 normalizeURL（scanner/web/web_scanner.go）保持一致
// （"URL 优先、Ports 兜底"），但物理独立实现，不 import 该函数——
// 见 API扫描功能设计.md 5.1 节。
func normalizeTarget(target string, ports string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}

	port := firstPort(ports)
	host := target
	if port != "" && port != "80" && port != "443" && !strings.Contains(target, ":") {
		host = target + ":" + port
	}

	switch port {
	case "443", "8443", "9443", "10443":
		return "https://" + host
	default:
		return "http://" + host
	}
}

// firstPort 从 "80,443" 这种逗号分隔的端口列表里取第一个非空项。
// ApiScan 的起始页面只需要一个入口 URL，不像 WebScanner 需要对每个端口
// 都独立探测一次——ApiScan 关心的是"从这一个入口爬出多少接口"，不是
// "这台机器有多少个端口在提供 Web 服务"，用第一个端口即可。
func firstPort(ports string) string {
	if ports == "" {
		return ""
	}
	parts := strings.Split(ports, ",")
	return strings.TrimSpace(parts[0])
}

// jsSource 描述一段待提取的 JS/HTML 文本及其来源，供 extractAPICandidates
// 之后统一打标 model.APIInfo.Source。
type jsSource struct {
	Text string
	From string // "inline" 或外链 JS 的绝对 URL
}

// fetchedPage 是单页抓取的完整产出：渲染后的 HTML body + 所有待提取文本
// （HTML 本身 + 内联 JS + 外链 JS），以及本页发现的导航链接（供 BFS 使用）。
type fetchedPage struct {
	URL       string
	Sources   []jsSource
	Links     []string
	Truncated bool // 外链 JS 数量超过 maxJSFiles 被截断
}

// fetchPage 用 go-rod 渲染单页，拿到 HTML/内联JS/外链JS 三种文本来源。
// browserLauncher 由调用方（ApiScanner.ensureInit）持有的独立实例传入，
// 不与 WebScanner 共享运行时状态。
func fetchPage(ctx context.Context, launcher *browser.BrowserLauncher, pageURL string, maxJSFiles int) (*fetchedPage, error) {
	br, err := launcher.Launch(ctx)
	if err != nil {
		return nil, err
	}

	page, err := launcher.OpenPage(ctx, br, "")
	if err != nil {
		return nil, err
	}
	defer page.Close()

	if err := page.Navigate(pageURL); err != nil {
		return nil, err
	}
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := page.Context(waitCtx).WaitLoad(); err != nil {
		logger.Warnf("[ApiScanner] WaitLoad timeout for %s: %v", pageURL, err)
	}

	body, err := page.HTML()
	if err != nil {
		return nil, err
	}

	result := &fetchedPage{URL: pageURL}
	result.Sources = append(result.Sources, jsSource{Text: body, From: pageURL})
	result.Sources = append(result.Sources, extractInlineScripts(page)...)

	links, jsFileURLs := extractLinksAndScriptSrcs(page, pageURL)
	result.Links = links

	if len(jsFileURLs) > maxJSFiles {
		result.Truncated = true
		jsFileURLs = jsFileURLs[:maxJSFiles]
	}
	for _, jsURL := range jsFileURLs {
		text, err := downloadJSFile(ctx, jsURL)
		if err != nil {
			logger.Warnf("[ApiScanner] Failed to download JS file %s: %v", jsURL, err)
			continue
		}
		result.Sources = append(result.Sources, jsSource{Text: text, From: jsURL})
	}

	return result, nil
}

// downloadJSFile 用纯 net/http 下载外链 JS 文件文本内容——静态 JS 文件本身
// 是文本资源，不需要浏览器渲染，见 API扫描功能设计.md 第八节。
func downloadJSFile(ctx context.Context, jsURL string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, jsURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}
```

`extractInlineScripts`、`extractLinksAndScriptSrcs` 两个函数负责从 `*rod.Page` 里分别读取内联 `<script>` 文本、页面链接与外链 `<script src>` 列表，实现细节：

```go
// extractInlineScripts 读取页面里所有没有 src 属性的 <script> 标签的文本内容。
func extractInlineScripts(page *rod.Page) []jsSource {
	elements, err := page.Elements("script:not([src])")
	if err != nil {
		return nil
	}
	var out []jsSource
	for _, el := range elements {
		text, err := el.Text()
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, jsSource{Text: text, From: "inline"})
	}
	return out
}

// extractLinksAndScriptSrcs 读取页面里所有 <a href> 绝对化后的链接（供 BFS
// 使用）与所有 <script src> 绝对化后的 URL（供后续下载）。用 net/url 的
// ResolveReference 做相对路径换算，逻辑与 web/crawler/extract.go 的 resolve
// 一致，但独立实现，不 import 该函数。
func extractLinksAndScriptSrcs(page *rod.Page, pageURL string) (links []string, scriptSrcs []string) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, nil
	}

	if anchors, err := page.Elements("a[href]"); err == nil {
		for _, a := range anchors {
			href, err := a.Attribute("href")
			if err != nil || href == nil {
				continue
			}
			if abs := resolveHref(base, *href); abs != "" {
				links = append(links, abs)
			}
		}
	}

	if scripts, err := page.Elements("script[src]"); err == nil {
		for _, s := range scripts {
			src, err := s.Attribute("src")
			if err != nil || src == nil {
				continue
			}
			if abs := resolveHref(base, *src); abs != "" {
				scriptSrcs = append(scriptSrcs, abs)
			}
		}
	}
	return links, scriptSrcs
}

// resolveHref 把相对/绝对路径换算成绝对 URL，过滤掉 javascript:/mailto:/
// tel:/纯锚点这类不可爬取的伪链接，逻辑与 web/crawler/extract.go 的 resolve
// 一致但独立实现（原子隔离原则不允许 import）。
func resolveHref(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" ||
		strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "mailto:") ||
		strings.HasPrefix(href, "tel:") ||
		strings.HasPrefix(href, "#") {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}
```

### 8.2 验收标准

- `go build ./internal/core/scanner/api/...` 通过。
- 见步骤 8 单元测试（仅覆盖 `normalizeTarget`/`firstPort`/`resolveHref`，`fetchPage` 依赖真实浏览器进程，放到步骤 14 端到端验收覆盖）。

---

## 九、步骤 8：`fetch_test.go`

```go
package api

import (
	"net/url"
	"testing"
)

func TestNormalizeTarget(t *testing.T) {
	cases := []struct {
		name   string
		target string
		ports  string
		want   string
	}{
		{"完整HTTPS_URL忽略Ports", "https://example.com", "80,443", "https://example.com"},
		{"完整HTTP_URL带端口忽略Ports", "http://example.com:8080", "80,443", "http://example.com:8080"},
		{"裸域名标准端口443猜https", "example.com", "443", "https://example.com"},
		{"裸域名标准端口80猜http", "example.com", "80", "http://example.com"},
		{"裸域名非标准端口显式拼接", "example.com", "8080", "http://example.com:8080"},
		{"裸IP多端口取第一个", "192.168.1.1", "443,80", "https://192.168.1.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTarget(tc.target, tc.ports)
			if got != tc.want {
				t.Errorf("normalizeTarget(%q, %q) = %q, want %q", tc.target, tc.ports, got, tc.want)
			}
		})
	}
}

func TestResolveHref(t *testing.T) {
	base, _ := url.Parse("http://example.com/dir/page.html")
	cases := []struct {
		href string
		want string
	}{
		{"foo.html", "http://example.com/dir/foo.html"},
		{"/api/user", "http://example.com/api/user"},
		{"javascript:void(0)", ""},
		{"#section", ""},
		{"mailto:a@b.com", ""},
	}
	for _, tc := range cases {
		got := resolveHref(base, tc.href)
		if got != tc.want {
			t.Errorf("resolveHref(base, %q) = %q, want %q", tc.href, got, tc.want)
		}
	}
}
```

### 9.1 验收标准

- `go test ./internal/core/scanner/api/... -run TestNormalizeTarget -v` 与 `-run TestResolveHref -v` 全部通过。

---

## 十、步骤 9：`bfs.go` —— 独立 BFS 队列

### 10.1 文件：`internal/core/scanner/api/bfs.go`

架构模式参照 `web/crawler/crawler.go`（队列/去重/深度/Scope 判断），代码物理独立、不 import：

```go
// Package api 的 bfs.go：独立的 BFS 队列实现，去重/深度控制/Scope 判断。
// 架构模式参照 web/crawler/crawler.go，代码物理独立，不 import，见
// API扫描功能设计.md 第四节。
package api

import (
	"context"
	"net/url"
	"sort"
	"sync"
	"sync/atomic"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/network/qos"
)

// crawlItem 是 BFS 队列内部元素，不对外暴露。
type crawlItem struct {
	URL   string
	Depth int
}

// pageOutcome 是 crawler 对外产出的单页处理结果，编排层（api_scanner.go）
// 用它组装 model.ApiResult，不直接暴露 fetchedPage 这个抓取层内部类型。
type pageOutcome struct {
	URL           string
	Depth         int
	APIs          []candidate
	APIsTruncated bool
}

// apiCrawler 单次爬取任务的执行器，一次 newAPICrawler 对应一次 crawl 调用，
// 不可跨任务复用状态，与 web/crawler.Crawler 的生命周期约定一致。
type apiCrawler struct {
	launcher   *browser.BrowserLauncher
	limiter    *qos.AdaptiveLimiter
	maxDepth   int
	maxJSFiles int
	seedHost   string

	mu      sync.Mutex
	visited map[string]struct{}

	outcomesMu sync.Mutex
	outcomes   []pageOutcome

	queue     chan *crawlItem
	pending   int32
	closeOnce sync.Once
}

// newAPICrawler 创建一个 apiCrawler 实例。launcher/limiter 必须由调用方
// （ApiScanner.ensureInit）传入已存在的实例，不允许传 nil。
func newAPICrawler(launcher *browser.BrowserLauncher, limiter *qos.AdaptiveLimiter, maxDepth, maxJSFiles int) *apiCrawler {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	return &apiCrawler{
		launcher:   launcher,
		limiter:    limiter,
		maxDepth:   maxDepth,
		maxJSFiles: maxJSFiles,
		visited:    make(map[string]struct{}),
	}
}

// crawl 执行一次 BFS 爬取，起点是 seedURL 本身（与 web/crawler.Crawler 不同——
// web/crawler 的首页由 WebScanner 单独处理、crawler 只管子页面；ApiScanner
// 没有"首页单独处理"的必要，seedURL 直接作为 depth=0 的第一个 item 入队，
// 统一走同一套抓取+提取逻辑，不重复实现"首页专用"的一套代码）。
func (c *apiCrawler) crawl(ctx context.Context, seedURL string) []pageOutcome {
	u, err := url.Parse(seedURL)
	if err != nil {
		return nil
	}
	c.seedHost = u.Host

	c.queue = make(chan *crawlItem, 256)
	c.enqueue(seedURL, 0)

	var wg sync.WaitGroup
	const concurrency = 5
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go c.worker(ctx, &wg)
	}
	wg.Wait()

	c.outcomesMu.Lock()
	defer c.outcomesMu.Unlock()
	return c.outcomes
}

func (c *apiCrawler) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for it := range c.queue {
		c.process(ctx, it)
	}
}

func (c *apiCrawler) process(ctx context.Context, it *crawlItem) {
	defer c.taskDone()

	if err := c.limiter.Acquire(ctx); err != nil {
		return
	}
	page, err := fetchPage(ctx, c.launcher, it.URL, c.maxJSFiles)
	if err != nil {
		c.limiter.OnFailure()
		c.limiter.Release()
		return
	}
	c.limiter.OnSuccess()
	c.limiter.Release()

	pageHost := ""
	if u, err := url.Parse(it.URL); err == nil {
		pageHost = u.Host
	}

	var allCandidates []candidate
	for _, src := range page.Sources {
		found := extractAPICandidates(src.Text, pageHost)
		for i := range found {
			found[i].Source = src.From
		}
		allCandidates = append(allCandidates, found...)
	}
	filtered := filterAPICandidates(allCandidates, "")

	c.outcomesMu.Lock()
	c.outcomes = append(c.outcomes, pageOutcome{
		URL: it.URL, Depth: it.Depth,
		APIs: filtered, APIsTruncated: page.Truncated,
	})
	c.outcomesMu.Unlock()

	if it.Depth >= c.maxDepth {
		return
	}
	for _, link := range page.Links {
		c.enqueue(link, it.Depth+1)
	}
}

// enqueue 尝试将一个 URL 入队，内部完成去重、Scope 判断、MaxPages 硬上限判断。
// MaxPages 固定 200，与 web/crawler 的默认值保持一致（方案文档第十一节：
// 是否需要单独调优留到真实站点测试后再定，先复用已验证过的默认值）。
func (c *apiCrawler) enqueue(raw string, depth int) {
	key := normalizeKey(raw)

	c.mu.Lock()
	if _, exists := c.visited[key]; exists {
		c.mu.Unlock()
		return
	}
	if len(c.visited) >= 200 {
		c.mu.Unlock()
		return
	}
	if !c.inScope(key) {
		c.mu.Unlock()
		return
	}
	c.visited[key] = struct{}{}
	c.mu.Unlock()

	atomic.AddInt32(&c.pending, 1)
	select {
	case c.queue <- &crawlItem{URL: key, Depth: depth}:
	default:
		c.taskDone()
	}
}

func (c *apiCrawler) taskDone() {
	if atomic.AddInt32(&c.pending, -1) == 0 {
		c.closeOnce.Do(func() { close(c.queue) })
	}
}

func (c *apiCrawler) inScope(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Host == c.seedHost
}

// normalizeKey 对齐 web/crawler.normalizeKey 的行为（去 Fragment、Query
// 参数排序），独立实现不 import。
func normalizeKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	q := u.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := url.Values{}
	for _, k := range keys {
		sorted[k] = q[k]
	}
	u.RawQuery = sorted.Encode()
	return u.String()
}
```

> **`filterAPICandidates(allCandidates, "")` 的第二参数说明**：结合步骤 6 末尾的说明，`Source` 字段已经在 `process` 里逐条打标（`found[i].Source = src.From`），`filterAPICandidates` 不需要再接收一个全局 `source` 参数统一赋值。编码时请同步把步骤 5/6 的 `filterAPICandidates` 签名改为 `filterAPICandidates(candidates []candidate) []candidate`（去掉 `source` 参数，`candidate` 结构体保留每条各自的 `Source` 字段），本节和第六节的示例代码存在这一处需要对齐的接口设计选择，实现时以"`Source` 逐条携带、`filterAPICandidates` 不做全局覆盖"为准——这样一个页面里既有 inline 命中也有外链 JS 命中时，两者的 `Source` 才能各自正确保留，不会被互相覆盖。

### 10.2 验收标准

- `go build ./internal/core/scanner/api/...` 通过。
- 见步骤 10 单元测试。

---

## 十一、步骤 10：`bfs_test.go`

不依赖真实浏览器，用 `normalizeKey`/`inScope`/`enqueue` 的去重与 Scope 判断逻辑做纯函数测试（`crawl` 整体流程依赖 `fetchPage`，放到步骤 14 端到端验收覆盖）：

```go
package api

import "testing"

func TestNormalizeKey_QueryParamOrderInsensitive(t *testing.T) {
	a := normalizeKey("http://example.com/x?b=2&a=1")
	b := normalizeKey("http://example.com/x?a=1&b=2")
	if a != b {
		t.Errorf("expected same normalized key regardless of query param order, got %q vs %q", a, b)
	}
}

func TestNormalizeKey_StripsFragment(t *testing.T) {
	got := normalizeKey("http://example.com/x#section")
	if got != "http://example.com/x" {
		t.Errorf("expected fragment stripped, got %q", got)
	}
}

func TestApiCrawler_Enqueue_DedupeAndScope(t *testing.T) {
	c := newAPICrawler(nil, nil, 2, 20)
	c.seedHost = "example.com"
	c.queue = make(chan *crawlItem, 10)

	c.enqueue("http://example.com/a", 1)
	c.enqueue("http://example.com/a", 1) // 重复，应被去重
	c.enqueue("http://other.com/b", 1)   // 跨域，应被 Scope 拒绝

	if len(c.visited) != 1 {
		t.Fatalf("expected 1 visited entry after dedupe+scope filtering, got %d: %+v", len(c.visited), c.visited)
	}
}
```

### 11.1 验收标准

- `go test ./internal/core/scanner/api/... -run TestNormalizeKey -v` 与 `-run TestApiCrawler_Enqueue -v` 全部通过。

---

## 十二、步骤 11：`api_scanner.go` —— 编排 `Run()`

### 12.1 文件：`internal/core/scanner/api/api_scanner.go`（整体重写）

```go
// Package api 提供 API 扫描能力：独立发起抓取（go-rod 渲染 + 默认开启的
// BFS 深度爬取），从 HTML/内联JS/外链JS 中静态提取接口调用地址。不依赖
// web 包或任何跨扫描器共享的爬虫模块，见 API扫描功能设计.md 第二节。
package api

import (
	"context"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
)

// ApiScanner 是独立于 WebScanner 的原子扫描器，与 WebScanner 平级，各自
// 持有独立的 browser/limiter 实例，互不共享运行时状态。
type ApiScanner struct {
	browserLauncher *browser.BrowserLauncher
	limiter         *qos.AdaptiveLimiter
}

// NewApiScanner 创建一个 ApiScanner 实例，独立初始化自己的基础设施实例。
func NewApiScanner() *ApiScanner {
	bm := browser.NewBrowserManager()
	return &ApiScanner{
		browserLauncher: browser.NewLauncher(bm),
		limiter:         qos.NewAdaptiveLimiter(5, 1, 10),
	}
}

// Name 扫描器名称。
func (s *ApiScanner) Name() model.TaskType {
	return model.TaskTypeApiScan
}

// Run 执行 API 扫描任务：Target/Ports 换算起始 URL -> BFS 深度爬取
// （默认总是开启）-> 每页产出 model.ApiResult。
func (s *ApiScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	startURL := normalizeTarget(task.Target, task.PortRange)

	crawlDepth := 2
	if v, ok := task.Params["crawl_depth"].(int); ok && v > 0 {
		crawlDepth = v
	}
	maxJSFiles := 20
	if v, ok := task.Params["max_js_files"].(int); ok && v > 0 {
		maxJSFiles = v
	}

	crawler := newAPICrawler(s.browserLauncher, s.limiter, crawlDepth, maxJSFiles)
	outcomes := crawler.crawl(ctx, startURL)

	startTime := time.Now()
	results := make([]*model.TaskResult, 0, len(outcomes))
	for _, oc := range outcomes {
		apis := make([]model.APIInfo, 0, len(oc.APIs))
		for _, c := range oc.APIs {
			apis = append(apis, model.APIInfo{
				URL: c.URL, Method: c.Method,
				Source: c.Source, Confidence: c.Confidence,
			})
		}
		results = append(results, &model.TaskResult{
			TaskID: task.ID,
			Status: model.TaskStatusSuccess,
			Result: &model.ApiResult{
				URL: oc.URL, Depth: oc.Depth,
				APIs: apis, APIsTruncated: oc.APIsTruncated,
			},
			ExecutedAt:  startTime,
			CompletedAt: time.Now(),
		})
	}
	return results, nil
}
```

> `task.Params["crawl_depth"]`/`["max_js_files"]` 的类型断言写成 `.(int)`：`ApiScanOptions.ToTask()`（步骤 2）写入的是 Go `int` 字面量，`Task.Params` 是 `map[string]interface{}`，CLI 直接构造 `Task` 的路径类型一致，不存在 JSON 反序列化导致数字变成 `float64` 的问题（Cluster 模式如果未来经过 JSON 传输，需要同时接受 `float64` 分支，本步骤先覆盖 CLI 场景，Cluster 场景的类型兼容留到该功能真正需要跨 JSON 传输时再补）。

### 12.2 验收标准

- `go build ./internal/core/scanner/api/...` 通过。
- 见步骤 12 单元测试。

---

## 十三、步骤 12：更新 `api_scanner_test.go`

原骨架测试 `TestApiScanner_Run_SkeletonReturnsNoResult` 断言"返回空结果"，现在 `Run()` 有真实实现，该测试需要更新语义（不再要求空结果，而是要求不 panic、能正常执行完编排逻辑）。真实抓取依赖网络/浏览器，本步骤的单元测试只覆盖 `Name()` 和参数解析部分，抓取相关的行为放到步骤 14 端到端验收：

```go
package api

import (
	"testing"

	"neoagent/internal/core/model"
)

// TestApiScanner_Name 验证 Name() 返回值与 RunnerManager 注册键一致。
func TestApiScanner_Name(t *testing.T) {
	scanner := NewApiScanner()
	if scanner.Name() != model.TaskTypeApiScan {
		t.Errorf("expected Name()=%q, got %q", model.TaskTypeApiScan, scanner.Name())
	}
}

// TestApiScanner_Run_ParamDefaults 验证 crawl_depth/max_js_files 缺省时
// Run() 内部使用方案文档约定的默认值，不依赖真实网络请求——用一个必然
// 无法访问的地址验证 Run() 不 panic、返回值结构合法即可，抓取成功与否
// 不是本测试关注的点（真实抓取覆盖见 api_scanner_e2e_test.go）。
func TestApiScanner_Run_DoesNotPanicOnUnreachableTarget(t *testing.T) {
	scanner := NewApiScanner()
	task := model.NewTask(model.TaskTypeApiScan, "http://127.0.0.1:1")
	task.PortRange = "1"

	results, err := scanner.Run(context.Background(), task)
	// 不可达目标：crawl 内部 fetchPage 会失败，outcomes 为空，Run 本身
	// 不应该返回 error（这不是"扫描器故障"，是"目标不可达"，两者语义不同）。
	if err != nil {
		t.Fatalf("Run should not return error for unreachable target, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results for unreachable target, got %d", len(results))
	}
}
```

> 需要在 import 里加 `"context"`。

### 13.1 验收标准

- `go test ./internal/core/scanner/api/... -v` 全部通过。

---

## 十四、步骤 13：CLI 参数绑定

### 14.1 改动位置

`cmd/agent/scan/api.go`（当前第 61~68 行的 flags 绑定部分）。

### 14.2 具体改动

删除 `--crawl` 绑定，新增 `--max-js-files`：

```go
flags := cmd.Flags()
flags.StringVarP(&opts.Target, "target", "t", opts.Target, "目标 URL/IP")
flags.StringVarP(&opts.Ports, "ports", "p", opts.Ports, "端口范围")
flags.IntVar(&opts.CrawlDepth, "crawl-depth", opts.CrawlDepth, "深度爬取的层数（ApiScan 总是深度爬取，无需 --crawl 开关）")
flags.IntVar(&opts.MaxJSFiles, "max-js-files", opts.MaxJSFiles, "单页最多下载的外链 JS 文件数")

cmd.MarkFlagRequired("target")
```

同时更新文件顶部注释（第 14~18 行），把"由 `Web-JS接口提取实施文档.md` 补充"的引用改为指向本文档：

```go
// NewApiScanCmd 创建 api_scan 命令。
//
// ApiScan 总是深度爬取子页面（无 --crawl 开关，见 API扫描功能设计.md 第五节），
// 参数细节见 docs/API扫描-js提取/API扫描实施文档.md 第十四节。
```

### 14.3 验收标准

- `go build ./cmd/agent/...` 通过。
- `neoAgent api --help` 输出里能看到 `--crawl-depth`、`--max-js-files`，看不到 `--crawl`。

---

## 十五、步骤 14：端到端验收

不写新的生产代码，用真实/半真实场景验证整条链路符合方案文档第一节的预期效果，同时补齐 `fetchPage`/`apiCrawler.crawl` 这两个依赖真实浏览器的路径此前未被单元测试覆盖到的行为。

### 15.1 验收用例设计

| 用例 | 目标 | 测试方式 | 预期结果 |
|---|---|---|---|
| A | 单页高置信度接口提取 | `httptest.NewServer` 起一个返回固定 HTML（含 `fetch('/api/user')`）的本地页面 | `ApiResult.APIs` 至少含一条 `Confidence: "high"`、`URL: "/api/user"` |
| B | BFS 深度爬取生效 | 本地 mock server 首页有一个指向 `/page2` 的链接，`/page2` 页面里有另一个接口调用 | 返回结果包含 2 条 `ApiResult`（Depth 0 和 Depth 1 各一条），各自 `APIs` 不为空 |
| C | 外链 JS 超限截断 | mock server 首页引用超过 `MaxJSFiles`（测试里设为 2）个外链 JS 文件 | 首页 `ApiResult.APIsTruncated == true`，`APIs` 只包含前 2 个 JS 文件里能提取到的接口 |
| D | 跨域链接不进入 BFS | mock server 首页有一个指向外部域名的链接 | 结果集里不出现外部域名对应的 `ApiResult` |
| E | 目标不可达 | 构造一个必然连接失败的地址（如 `http://127.0.0.1:1`） | `Run()` 不返回 error、不 panic，`results` 为空切片 |

### 15.2 验收标准

- 用例 A/B/C/D/E 全部符合预期结果列，沉淀为 `internal/core/scanner/api/api_scanner_e2e_test.go` 里的正式集成测试（不是跑一次就丢弃的临时脚本），与 `web_scanner_cdn_e2e_test.go` 的组织方式保持一致。
- `go test ./internal/core/scanner/api/... -v` 全量测试（含单元 + 端到端）通过，`go vet ./internal/core/scanner/api/...` 无告警。
- `go build ./...`、`go vet ./...` 全量通过，确认没有破坏其它包。

---

## 十六、实施完成后需要同步更新的文档

代码全部落地并通过步骤 14 验收后，回填以下文档，不要让文档和代码状态脱节：

1. `API扫描功能设计.md` 第十一节"待实施文档阶段确认的事项"：
   - `CrawlDepth` 默认值如果在步骤 15 端到端验收中依据真实站点测试结果做了调整，回填最终确定的值和理由；
   - `bfs.go` 的并发数/单页超时/MaxPages 如果做了调优，回填最终值；
   - 把"骨架代码遗留的文档引用清理"这一条标记为已完成（步骤 14.2 已经把 `cmd/agent/scan/api.go` 的引用改成指向本文档，同步确认 `result_types.go`、`scan_api.go` 里旧的骨架注释也已经在步骤 1/2 里被整体替换，不再包含失效引用）。
2. `docs/开发进度.md`：视当前 Phase 排期安排，补充一条 Sprint 记录（参照现有记录格式，一句话说清"做了什么、解决了什么真实问题"）。
3. `Agent重构开发总纲.md`：如涉及里程碑更新，同步勾选。

本实施文档本身不需要在完成后删除，作为这次改动的历史记录保留。
</contents>
