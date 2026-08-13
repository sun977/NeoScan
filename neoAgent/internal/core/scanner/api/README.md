# ApiScanner — API 接口提取扫描模块

## 模块定位

`ApiScanner` 是独立于 `WebScanner` 的**原子扫描器**，与其平级。遵循项目的"原子隔离原则"：

- 持有独立的 `browser.BrowserLauncher` 实例，不与其他扫描器共享浏览器运行时
- 持有独立的 `qos.AdaptiveLimiter` 实例，不共享限流状态
- 不 import 任何其他扫描器的内部包（`web/crawler` 等），同类逻辑独立实现

**职责一句话**：通过 BFS 深度爬取目标站点，从 HTML / 内联 JS / 外链 JS 中静态提取 HTTP 接口路径和 WebSocket/SSE 端点，输出带置信度分级的 `ApiResult`。

---

## ApiScanner 做了什么

用一个具体例子说明完整执行过程，目标：`https://example.com/`

**第一步：渲染首页**

用 go-rod 驱动 Chromium 打开首页，等待 JS 执行完成后读取最终 DOM。这样可以拿到 SPA（React/Vue 等）动态渲染后的完整 HTML，而不是服务端返回的空 `<div id="root">`。

**第二步：从首页提取三类文本来源**

每个页面包含三种文本，分别送入提取管线：

| 来源类型 | 说明 | `Source` 字段值 |
|----------|------|-----------------|
| **HTML body** | 整个渲染后的 DOM 文本 | 页面 URL（如 `https://example.com/`） |
| **内联脚本** | `<script>` 标签内直接写在 HTML 里的 JS 代码 | `"inline"` |
| **外链脚本** | `<script src="...">` 引用的独立 `.js` 文件 | JS 文件的完整 URL |

> **内联 vs 外链**：
> - **内联脚本（inline script）**：JS 代码直接嵌入 HTML 文件里，形如 `<script>fetch('/api/user')</script>`，没有 `src` 属性。页面加载时浏览器直接执行，不需要额外下载。
> - **外链脚本（external script）**：JS 代码存放在独立文件里，HTML 通过 `<script src="/static/app.abc123.js"></script>` 引用。浏览器需要额外发起 HTTP 请求下载这个文件。现代 SPA 通常把所有业务逻辑打包成一个或多个 `.js` bundle，所有页面共享同一套外链脚本。

**第三步：从文本中提取 API 候选项（三级置信度）**

对每段文本跑正则匹配，产出"API 候选项"，按置信度分三档（详见下方"提取层"说明）。

**第四步：过滤降噪**

候选项经过多级过滤（静态资源后缀、黑名单、来源过滤等），去掉明显的误报（详见下方"过滤层"说明）。

**第五步：BFS 继续爬取子页面**

从首页的 `<a href>` 链接里筛选出同域 URL，按深度（默认 2 层）继续爬取，每页重复上述流程。

**JS 文件全局缓存**：同一个外链 `.js` 文件被多个页面引用时（SPA 的常见情况），只下载和提取**一次**，后续页面直接从缓存复用结果，避免重复 HTTP 请求和重复提取。

---

## 模块结构

```
api/
├── api_scanner.go        # 入口：NewApiScanner() + Run()，组装 TaskResult，对外唯一暴露点
├── bfs.go                # BFS 爬取调度：队列、去重、Scope 判断、并发 worker、JS 缓存
├── fetch.go              # 单页抓取：go-rod 渲染 + 内联脚本提取 + 外链 URL 列表收集
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
  ├── HTML body（整页渲染后的 DOM 文本）
  ├── 内联 <script> 文本（el.Property("textContent") 读取）
  └── 外链 JS URL 列表（不直接下载，交给 process() 处理）
      │
      ▼
getOrFetchJS()             → 查 JS 缓存（bfs.go）
  ├── 命中：直接复用已提取的 candidates
  └── 未命中：net/http 下载 .js 文件 → 提取 → 写缓存
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
| **high** | `fetch(url)`、`axios.method(url)`、`.ajax({url:...})`、`new EventSource(url)`、`new WebSocket(url)` 结构化调用 | `fetch('/api/user/info')` | axios 分支有值，其余为空 |
| **medium** | 含目标域名的完整 URL（`https://host/...`） | `https://api.mysite.com/v1/orders` | 无 |
| **low** | 通用路径字面量 `/seg1/seg2/...`（支持 `{}` 动态路由） | `'/api/users/{id}'` | 无 |

`low` 置信度误报率最高，依赖过滤层重点清洗。

**能力边界**：GraphQL 客户端通常只发向 `/graphql` 一个端点，query body 里的 operation name（`query GetUser { ... }`）当前无法提取。目标使用 GraphQL 时，只能看到 `/graphql` 这个单一端点。

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
| `"inline"`（内联脚本） | ✅ 保留 | 属于 JS 代码块，命中真实接口的概率高 |
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
以 `URL + "|" + Method` 为组合键单页去重。外链 JS 的全局缓存机制（见"ApiScanner 做了什么"第五步）从根源上消除了跨页面的 JS 重复提取问题。

---

## BFS 爬取参数

| 参数 | 默认值 | Task.Params key | 取值范围 |
|------|--------|-----------------|----------|
| 爬取深度 | 2 | `crawl_depth` | > 0 |
| 每页最多处理外链 JS 数 | 20 | `max_js_files` | > 0 |
| BFS 并发 worker 数 | 5 | `concurrency` | 1–20 |
| 最大爬取页数 | 200（硬上限） | — | — |

---

## 并发安全设计

### enqueue() 的 visited 写入时机
`visited` 的写入发生在 **channel send 成功之后**（而非之前）：
1. 预检：去重 + Scope + MaxPages，**不写 visited**
2. `atomic.AddInt32(&c.pending, 1)`
3. `select`：channel 满走 `default` 分支 → `visited 不写`，`pending` 回退
4. 入队成功 → 才写 `visited`

这样避免"URL 进了 visited 但未入队"的静默丢页 bug。代价是在 channel 满载的极小窗口内，同一 URL 可能被两个 goroutine 同时通过预检并各自入队一次（重复处理一次），但这是幂等的（去重在 `filterAPICandidates` 和最终 `seen` map 层面保证）。

### JS 缓存锁外下载
`getOrFetchJS()`：先加锁读缓存 → 未命中后**释放锁再下载**（下载是耗时 IO，不能持锁） → 下载完成后再次加锁写入。极端情况下同一 URL 可能被两个 goroutine 同时下载，但写入时用 `alreadySet` 检查避免覆盖。

---

## 使用方式

### CLI

```bash
# 基本用法（必填 --target）
neoAgent scan api -t http://example.com

# 指定端口
neoAgent scan api -t http://example.com -p "8080,8443"

# 自定义爬取参数
neoAgent scan api -t http://example.com --crawl-depth 3 --concurrency 10 --max-js-files 50

# 输出到文件
neoAgent scan api -t http://example.com --oc results.csv --oj results.json
```

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--target` | `-t` | (必填) | 目标 URL/IP |
| `--ports` | `-p` | `80,443` | 端口范围 |
| `--crawl-depth` | `-d` | `2` | 深度爬取层数 |
| `--max-js-files` | `-m` | `20` | 单页最多下载外链 JS 数 |
| `--concurrency` | `-c` | `5` | BFS 并发 worker 数（1–20） |

### 编程调用

```go
import (
    "context"
    "time"

    "neoagent/internal/core/model"
    "neoagent/internal/core/scanner/api"
)

scanner := api.NewApiScanner()

task := &model.Task{
    ID:        "unique-task-id",
    Type:      model.TaskTypeApiScan,
    Target:    "http://example.com",
    PortRange: "8080,443",
    Timeout:   30 * time.Minute,
    Params: map[string]interface{}{
        "crawl_depth":    3,
        "max_js_files":   50,
        "concurrency":    10,
    },
}

ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
defer cancel()

results, err := scanner.Run(ctx, task)
// results 为 []*model.TaskResult，每页一条
```

### 通过 RunnerManager 注册调用

`ApiScanner` 已注册到 `RunnerManager`，通过 Task 类型自动路由：

```go
manager := runner.NewRunnerManager()
results, err := manager.Execute(ctx, task) // task.Type == "api_scan" 时路由到 ApiScanner
```

### Master 集群下发

Master 下发的 `api_scan` 类型任务会自动映射到 `ApiScanner`，meta 参数合并到 `task.Params`：

```json
{
    "type": "api_scan",
    "target": "http://example.com",
    "port_range": "80,443",
    "meta": {
        "crawl_depth": 3,
        "concurrency": 10
    }
}
```

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

CLI 以表格逐行输出，每行完整携带 Page/Depth 信息（自描述，支持流式追加），列为：

```
Page | Depth | API URL | Method | Confidence | Source
```
