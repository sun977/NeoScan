# ApiScanner 优化建议

本文档记录当前代码中**尚未解决**的优化点，每条附有具体代码位置和解决方案。
已修复的历史问题（问题一～七）已从本文档移除。

---

## 建议一（🟢 可选）：三类 API 调用形态当前无法提取

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
|  可选 | 一 | 跨域/GraphQL/WebSocket 提取能力扩展 | 按需单独实现，互相独立 |

---

*本文档持续维护，每次修复后同步删除对应条目。*
