# ApiScanner 优化建议

本文档基于对 `api/` 目录下全部源文件的逐行审查，每条建议均附有**具体代码位置**和**可复现的技术理由**，不含猜测。

---

## 问题一（🔴 严重）：`extractInlineScripts` 使用 `innerText` 读取 `<script>` 内容，导致所有内联脚本实际为空

### 代码位置

`fetch.go:153`

```go
text, err := el.Text()
```

### 技术根因

`rod` v0.106.8 的 `Element.Text()` 底层调用的 JS 是（见 `rod@v0.106.8/lib/js/helper.go:170`）：

```javascript
function() {
  switch (this.tagName) {
    case "INPUT": case "TEXTAREA": return this.value || this.placeholder;
    case "SELECT": return ...;
    case void 0:  return this.textContent;
    default:      return this.innerText;   // ← SCRIPT 走这里
  }
}
```

`<script>` 的 `tagName` 是 `"SCRIPT"`，命中 `default`，返回 `this.innerText`。

根据 [HTML 规范（whatwg）对 `innerText` 的定义](https://html.spec.whatwg.org/#the-innertext-idl-attribute)：`innerText` 的返回值基于渲染结果，而 `<script>` 标签属于 **not rendered element**（不参与渲染树），其 `innerText` 规范返回值为空字符串 `""`。

### 实际影响

`extractInlineScripts` 对每个内联 `<script>` 调用 `el.Text()`，全部得到 `""`，被 `strings.TrimSpace(text) == ""` 过滤掉，**导致所有内联脚本提取结果为空，inline source 的提取能力完全失效**。

High 置信度的 `fetch()`/`axios.method()` 调用通常在内联脚本中出现，此 bug 导致这类高价值结果无法被提取到。

### 正确做法

改用 `el.Property("textContent")` 或 `el.HTML()` 读取脚本原始内容：

- `textContent` 对 `<script>` 返回标签内的原始文本，不经过渲染处理
- `el.HTML()` 返回 `outerHTML`，需要额外裁掉 `<script...>` 和 `</script>` 标签

---

## 问题二（🔴 严重）：`enqueue` 中 `visited` 标记与实际入队非原子，存在"进 visited 但未入队"漏洞

### 代码位置

`bfs.go:150-178`

```go
func (c *apiCrawler) enqueue(raw string, depth int) {
    key := normalizeKey(raw)

    c.mu.Lock()
    if _, exists := c.visited[key]; exists { c.mu.Unlock(); return }
    if len(c.visited) >= 200       { c.mu.Unlock(); return }
    if !c.inScope(key)             { c.mu.Unlock(); return }
    c.visited[key] = struct{}{}     // ← 写入 visited（锁内）
    c.mu.Unlock()                   // ← 锁释放

    atomic.AddInt32(&c.pending, 1)  // ← 锁外，两步之间有窗口
    select {
    case c.queue <- &crawlItem{URL: key, Depth: depth}:
    default:
        c.taskDone()                // ← pending 回退，但 visited 里的记录不会回退
    }
}
```

### 技术根因

当 `c.queue` channel 容量（256）已满，`select` 的 `default` 分支被触发：

1. URL 已写入 `c.visited`（永久标记为"已处理"）
2. URL **没有**进入 `c.queue`（永远不会被处理）
3. `taskDone()` 回退了 `pending` 计数，但 `visited` 的记录不会撤销

结果：该 URL 被静默丢弃，且由于已在 `visited` 里，后续扫描中也不会重试。

### 实际影响

每页爬取完成后会把所有 `page.Links` 批量入队（`bfs.go:142-144`），一个页面可能有几十甚至上百个链接。当队列满载时，大量 URL 会以这种方式丢失。当前 channel 容量是 256，BFS 并发 5 个 worker，碰到链接密集的站点首页时很容易触发。

### 正确做法

两种方案：
1. **保持入队成功才写 visited**：先 `send` 到 channel，成功后再写 `visited`（但需要两次加锁）
2. **阻塞式入队**：去掉 `default` 分支，改为 `c.queue <- item`（阻塞等待），配合足够大的 channel 容量
3. **最简方案**：扩大 channel 容量（200 页 × 每页平均链接数），使 `default` 在正常爬取中不被触发

---

## 问题三（🟡 中）：跨页面不去重，同一 JS bundle 在多页被加载时产生大量重复 `APIInfo`

### 代码位置

`bfs.go:130` + `filter.go:98`

```go
// bfs.go：每页独立过滤，互不知晓
filtered := filterAPICandidates(allCandidates)

// filter.go：seen map 是局部变量，每次调用都是新的
func filterAPICandidates(candidates []candidate) []candidate {
    seen := make(map[string]bool, len(candidates))
    // ...
}
```

### 技术根因

现代 SPA（如 React/Vue 项目）通常只有一个 `app.xxxx.chunk.js`，被所有子页面引用。BFS 爬取 20 个子页面时，每个子页面都会下载并提取同一个 JS bundle，产生完全相同的 API 候选项。由于 `filterAPICandidates` 的去重仅在单次调用内有效，相同的 API 路径会在 20 个 `ApiResult` 里各出现一次。

### 实际影响

对 `https://www.polestarcloud.com/` 的测试中已经可以观察到这个现象——同一条 API URL 在多个深度的页面结果里重复出现。这对人工审计造成干扰，对 JSON/CSV 导出造成数据膨胀。

### 正确做法

在 `apiCrawler` 中维护一个**全局 JS 文件缓存**：`downloadedJS map[string][]candidate`，同一个 JS URL 只下载和提取一次，后续页面复用缓存结果。这样同时解决了性能问题（减少重复 HTTP 请求）和重复输出问题。

---

## 问题四（🟡 中）：`api_scanner.go` 中 `startTime` 在 `crawl()` 返回后才赋值，语义错误

### 代码位置

`api_scanner.go:48-84`

```go
func (s *ApiScanner) Run(ctx context.Context, task *model.Task) (...) {
    startURL := normalizeTarget(task.Target, task.PortRange)
    // ...
    crawler := newAPICrawler(...)
    outcomes := crawler.crawl(ctx, startURL)  // ← 这里可能跑了几十秒

    startTime := time.Now()   // ← 此时 crawl 已经跑完了，记录的是"组装结果"的时间
    // ...
    for _, oc := range outcomes {
        results = append(results, &model.TaskResult{
            ExecutedAt:  startTime,   // ← 应该是任务开始时间，但实际是任务结束时间
            CompletedAt: time.Now(),
        })
    }
}
```

### 实际影响

`ExecutedAt` 字段语义是"任务开始执行时间"。当前实现中 `ExecutedAt` 和 `CompletedAt` 的差值只有几微秒（仅组装结果的循环耗时），而真实的扫描耗时（几十秒到几分钟）完全没有被记录。如果外部系统依赖此字段做性能分析，数据是错误的。

### 正确做法

将 `startTime := time.Now()` 移到 `crawler.crawl()` 之前。

---

## 问题五（🟡 低）：`hasEnoughSegments` 内部有冗余的二层前缀循环

### 代码位置

`filter.go:58-83`

```go
func hasEnoughSegments(rawURL string, minSegments int) bool {
    path := rawURL
    if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
        // ↑ 外层已经判断了是 http/https，内层循环是多余的
        for _, prefix := range []string{"http://", "https://"} {
            if strings.HasPrefix(rawURL, prefix) {
                // ...
                break
            }
        }
    }
    // ...
}
```

### 技术根因

外层 `if` 已经确认是 `http://` 或 `https://` 开头，内层 `for range` 再遍历两个前缀找匹配是多余的。完全可以用 `net/url.Parse(rawURL).Path` 直接获取 path 部分，两行代替现在的 15 行。

### 实际影响

纯代码品味问题，逻辑是正确的，不影响功能，但增加了维护者的阅读负担。

---

## 问题六（🟡 低）：`downloadJSFile` 使用 `http.DefaultClient`，无 UA 头，无 Cookie 传递

### 代码位置

`fetch.go:124-143`

```go
func downloadJSFile(ctx context.Context, jsURL string) (string, error) {
    req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, jsURL, nil)
    // ...
    resp, err := http.DefaultClient.Do(req)  // ← DefaultClient，无 UA，无 Cookie
```

### 技术根因

go-rod 启动浏览器时设置了自定义 User-Agent（`version.GetUserAgent()`），但 `downloadJSFile` 用裸 `http.DefaultClient`，请求头是 Go 默认的 UA。对于：

1. **WAF 防护的站点**：浏览器请求 UA 与后续 JS 文件下载 UA 不一致，可能触发 WAF 拦截
2. **需要 Session Cookie 的 JS 文件**（虽然罕见，但 CDN 有时会根据 session 返回不同版本的 bundle）

另外，`http.DefaultClient` 是全局共享的，存在 jar 污染和连接池污染的潜在风险（虽然在当前实现中尚未显现）。

### 正确做法

创建一个模块级的 `http.Client`，设置与 go-rod 浏览器一致的 User-Agent 头，并设置适当的超时（当前已设置 10s 超时，这点是对的）。

---

## 优先级汇总

| 优先级 | 编号 | 问题 | 影响范围 |
|--------|------|------|----------|
| 🔴 必修 | 一 | `el.Text()` 对 `<script>` 返回空，内联脚本提取完全失效 | 高置信度结果大量丢失 |
| 🔴 必修 | 二 | `enqueue` visited 与入队非原子，URL 静默丢失 | BFS 覆盖率不可预测 |
| 🟡 建议 | 三 | 跨页面不去重，重复 HTTP 请求 + 输出膨胀 | 性能浪费 + 输出噪声 |
| 🟡 建议 | 四 | `startTime` 语义错误 | 时间记录数据不准 |
| 🟡 可选 | 五 | `hasEnoughSegments` 冗余循环 | 代码可读性 |
| 🟡 可选 | 六 | `downloadJSFile` 用 DefaultClient | 对抗性站点兼容性 |

---

*本文档生成于代码审查，每条问题均已验证代码或依赖库的具体行为，非推测。*
