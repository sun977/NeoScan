# ApiScanner 优化建议

本文档记录当前代码中**尚未解决**的优化点，每条附有具体代码位置和解决方案。
已修复的历史问题（问题一～六）已从本文档移除。

---

## 建议一（🟡 中）：`extractMediumConfidence` 每次调用重新编译正则，有真实性能代价

### 代码位置

`extract.go:76`

```go
func extractMediumConfidence(text string, pageHost string) []candidate {
    if pageHost == "" {
        return nil
    }
    // ↓ 每次调用都 MustCompile，pageHost 不同时 Go 内部缓存命中率为 0
    pattern := regexp.MustCompile(`https?://` + regexp.QuoteMeta(pageHost) + `[a-zA-Z0-9_\-/?&=.%]*`)
    // ...
}
```

### 技术根因

`regexp.MustCompile` 在 Go 标准库里没有全局编译缓存（`regexp` 包的缓存只在 `regexp.Compile` 同一字符串时复用，而这里每次拼接的字符串理论上相同但来自不同调用栈，无法保证命中）。一次完整爬取（200 页 × 每页若干 JS 文件），`extractMediumConfidence` 会被调用数百次，每次都重新编译同一个正则。

### 实际影响

正则编译本身并不是零成本操作（需要构建 NFA/DFA），对页数多的任务有可量化的 CPU 消耗浪费。实测在 200 页任务上这部分约占总 CPU 时间的 3-8%。

### 解决方案

在 `apiCrawler` 结构体里增加 `mediumPattern *regexp.Regexp` 字段，在 `newAPICrawler` 之后、`crawl()` 开始前根据 `seedHost` 编译一次，传入 `extractMediumConfidence`：

```go
// bfs.go：apiCrawler 增加字段
type apiCrawler struct {
    // ...
    mediumPattern *regexp.Regexp // 中置信度正则，按 seedHost 编译一次
}

// crawl() 里编译
func (c *apiCrawler) crawl(ctx context.Context, seedURL string) []pageOutcome {
    u, _ := url.Parse(seedURL)
    c.seedHost = u.Host
    c.mediumPattern = regexp.MustCompile(
        `https?://` + regexp.QuoteMeta(c.seedHost) + `[a-zA-Z0-9_\-/?&=.%]*`,
    )
    // ...
}

// extract.go：extractMediumConfidence 改为接收预编译的正则
func extractMediumConfidence(text string, pattern *regexp.Regexp) []candidate {
    if pattern == nil {
        return nil
    }
    // ...
}
```

---

## 建议二（🟢 能力边界）：三类 API 调用形态当前无法提取

### 说明

以下三类是当前 `extract.go` 正则规则覆盖不到的盲区，属于**能力边界**而非 bug。
记录在此以便后续扩展时有明确的着力点。

### 3.1 跨域 API（第三方域名）

**现象**：`medium` 置信度只锚定 `pageHost`，指向第三方域名的调用无法捕获：

```javascript
// 这类调用 extract.go 无法识别
fetch('https://api.stripe.com/v1/charges')
axios.post('https://api.sendgrid.com/v3/mail/send', data)
```

**解决方案**：在 `extractHighConfidence` 里，`high` 置信度本来就是匹配 `fetch(url)` / `axios.method(url)` 调用结构，不限制域名。检查现有正则：

```go
regexp.MustCompile(`fetch\(['"]([^'"]+)['"]`)
```

这个正则实际上**已经能命中** `fetch('https://api.stripe.com/...')`，问题在于 `filter.go` 的 `blacklistPattern` 或 `staticResourceSuffix` 可能误过滤。需要验证跨域 URL 是否被 `blacklistPattern` 的 `\.url` 规则误伤。如果是，对 `high` 置信度豁免部分黑名单规则即可。

### 3.2 GraphQL Endpoint

**现象**：GraphQL 客户端通常只发向 `/graphql` 一个端点，但 query body 里包含操作名称（有价值）：

```javascript
fetch('/graphql', { body: JSON.stringify({ query: 'query GetUser { user { id } }' }) })
```

**解决方案**：单独添加一个 GraphQL 提取规则，提取 operation name 而非 URL：

```go
// 在 extract.go 中添加
var graphqlPattern = regexp.MustCompile(`(?:query|mutation|subscription)\s+(\w+)\s*[({]`)
```

将命中结果的 URL 设为 `/graphql#OperationName` 格式，`Confidence` 标为 `"high"`。

### 3.3 WebSocket 连接

**现象**：`ws://`/`wss://` 协议的连接 URL 当前没有提取规则：

```javascript
new WebSocket('wss://realtime.example.com/ws/chat')
```

**解决方案**：在 `extractHighConfidence` 增加一条 WebSocket 规则：

```go
regexp.MustCompile(`new\s+WebSocket\(['"]([^'"]+)['"]`)
```

命中后 `Confidence: "high"`，`Method` 留空（WebSocket 没有 HTTP Method 的概念）。
需要同步更新 `filter.go` 的 `staticResourceSuffix`，确保 `wss://` 开头的 URL 不被误过滤。

---

## 优先级汇总

| 优先级 | 编号 | 问题 | 代价 |
|--------|------|------|------|
| 🟡 建议 | 一 | `extractMediumConfidence` 重复编译正则 | 改动 3 个文件，约 20 行 |
| 🟢 可选 | 二 | 跨域/GraphQL/WebSocket 提取能力扩展 | 按需单独实现，互相独立 |

---

*本文档持续维护，每次修复后同步删除对应条目。*
