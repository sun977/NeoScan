# ApiScanner — API 接口提取扫描模块

## 模块定位

`ApiScanner` 是独立于 `WebScanner` 的**原子扫描器**，与其平级。遵循项目的"原子隔离原则"：

- 持有独立的 `browser.BrowserLauncher` 实例，不与其他扫描器共享浏览器运行时
- 持有独立的 `qos.AdaptiveLimiter` 实例，不共享限流状态
- 不 import 任何其他扫描器的内部包（`web/crawler` 等），同类逻辑独立实现

**职责一句话**：通过 BFS 深度爬取目标站点，从 HTML / 内联 JS / 外链 JS 中静态提取 HTTP 接口路径，输出带置信度分级的 `ApiResult`。

---

## 模块结构

```
api/
├── api_scanner.go        # 入口：Run()，组装 TaskResult，对外唯一暴露点
├── bfs.go                # BFS 爬取调度：队列、去重、Scope 判断、并发 worker
├── fetch.go              # 单页抓取：go-rod 渲染 + 内联/外链 JS 获取
├── extract.go            # 三层正则提取：high / medium / low 置信度
├── filter.go             # 过滤清洗：静态资源排除、黑名单、来源过滤、去重
└── *_test.go             # 单元测试 + E2E 验收测试
```

---

## 数据流

```
Task.Target/Ports
      │
      ▼
normalizeTarget()          → 换算起始 URL（fetch.go）
      │
      ▼
apiCrawler.crawl()         → BFS 深度爬取（bfs.go）
      │  每页并发抓取
      ▼
fetchPage()                → go-rod 渲染页面，产出：
  ├── HTML body（整页）
  ├── 内联 <script> 文本
  └── 外链 .js 文件文本（net/http 下载）
      │
      ▼
extractAPICandidates()     → 三层正则，产出原始 candidate 列表（extract.go）
      │
      ▼
filterAPICandidates()      → 多级过滤，产出干净的 candidate 列表（filter.go）
      │
      ▼
[]model.ApiResult          → 每页一条，携带 []APIInfo（api_scanner.go）
```

---

## 提取层：三级置信度（extract.go）

| 级别 | 规则 | 示例命中 | Method |
|------|------|----------|--------|
| **high** | `fetch(url)`、`axios.method(url)`、`.ajax({url:...})` 结构化调用 | `fetch('/api/user/info')` | axios 分支有值，其余为空 |
| **medium** | 含目标域名的完整 URL（`https://host/...`） | `https://api.mysite.com/v1/orders` | 无 |
| **low** | 通用路径字面量 `/seg1/seg2/...` | `'/internal/config/list'` | 无 |

`low` 置信度误报率最高，依赖过滤层重点清洗。

---

## 过滤层：多级降噪策略（filter.go）

过滤按**开销从低到高**短路执行，顺序如下：

### 1. 转义清理（全部）
去首尾空白、还原 `\/` → `/`、URL 解码 `%3A` / `%2F`。

### 2. 静态资源后缀过滤（全部）
命中以下后缀直接丢弃（不是接口）：

```
.js .css .png .jpeg .jpg .gif .svg
.woff .woff2 .eot .ttf .ico .map .json
```

### 3. 黑名单关键词过滤（全部）
排除 JS 属性访问模式（`.src`、`.href`、`location.href` 等）及无意义域名（`www.w3.org`、`example.com`）。完整规则参见 `filter.go` 中的 `blacklistPattern`。

### 4. 页面后缀过滤（仅 medium）
`medium` 置信度容易把完整页面 URL 误识别为接口，额外过滤：

```
.html .htm .php .asp .aspx .jsp .cfm .do
```

### 5. Source 来源过滤（仅 low）⭐ 核心降噪
**只保留来自 JS 代码的 `low` 候选项**，来自页面 HTML 本体的全部丢弃。

| Source 类型 | 是否保留 | 原因 |
|-------------|----------|------|
| `"inline"`（内联 `<script>`） | ✅ 保留 | 属于 JS 代码块 |
| 以 `.js` 结尾的外链 URL | ✅ 保留 | JS bundle 中路径命中率高 |
| 页面 HTML 的 URL（非 `.js`） | ❌ 丢弃 | 路由表、CSS 类名、i18n key 等噪声大量存在 |
| 空字符串 | ❌ 丢弃 | 来源不明 |

**背景**：`low` 置信度的通用路径正则在页面 HTML 中会命中大量非接口内容（如 `href`、`src`、i18n 键名、Tailwind CSS 类路径等）。这些噪声在纯正则层面几乎无法与真实 API 路径区分，但它们几乎不出现在 JS bundle 里。这是**成本最低、效果最显著**的降噪手段。

### 6. 路径段数过滤（仅 low）
要求至少 **2 个非空路径段**，过滤 `/en/`、`/v1/`、`/zh-CN/` 等语言/版本前缀噪声。

```
/en/          → 1 段 → 丢弃
/api/users    → 2 段 → 保留
/api/v1/list  → 3 段 → 保留
```

### 7. 去重
以 `URL + Method` 为组合键全局去重，跨页面结果也不重复输出。

---

## BFS 爬取参数

| 参数 | 默认值 | Task.Params key |
|------|--------|-----------------|
| 爬取深度 | 2 | `crawl_depth` |
| 每页最多下载外链 JS 数 | 20 | `max_js_files` |
| 最大爬取页数 | 200（硬上限） | — |
| BFS 并发 worker 数 | 5 | — |

---

## 输出

每页输出一条 `model.ApiResult`：

```go
type ApiResult struct {
    URL           string      // 页面 URL
    Depth         int         // BFS 深度（0 = 起始页）
    APIs          []APIInfo   // 提取到的接口列表
    APIsTruncated bool        // 外链 JS 数量超出上限时为 true
}

type APIInfo struct {
    URL        string  // 接口路径或完整 URL
    Method     string  // HTTP Method（仅 high/axios 有值）
    Source     string  // 来源（"inline" 或外链 JS 文件 URL）
    Confidence string  // "high" / "medium" / "low"
}
```

CLI 输出通过 `TabularData` 接口以表格形式呈现，列为：`Page | Depth | API URL | Method | Confidence | Source`。
