# SubdomainScanner 模块设计文档

## 1. 概述

### 1.1 定位
SubdomainScanner 是基于字典的 DNS 子域名枚举模块。给定一个主域名（如 `example.com`），使用词表穷举可能的子域名前缀，对每个候选 `prefix.example.com` 发起 DNS A/AAAA/CNAME 查询，返回所有成功解析的子域名及其 IP 和 CNAME 记录。

### 1.2 核心特点
- **纯 DNS 查询**：不发起 HTTP 请求，速度快、噪音低。
- **字典驱动**：内置词表 + 用户自定义词表。
- **原子扫描器隔离**：不依赖也不被其他扫描器的内部包引用。

### 1.3 与 DirScanner 的关系
两者架构高度相似，可以看作"同源异流"：

| 维度 | DirScanner | SubdomainScanner |
|------|-----------|-----------------|
| 输入 | URL + 路径字典 | 域名 + 子域名前缀字典 |
| 探测 | HTTP GET | DNS A/AAAA/CNAME |
| 命中条件 | 状态码过滤 + 通配符检测 | DNS 成功解析（有 A/AAAA/CNAME） |
| 输出 | 路径、状态码、大小、标题 | IP、IP 列表、CNAME |
| QoS | AdaptiveLimiter | AdaptiveLimiter |

**结论**：实现流程与 DirScanner 完全一致（初始化 → Worker 池 → Channel 汇总），不需要为 SubdomainScanner 发明新的架构模式。

---

## 2. 技术选型

### 2.1 DNS 查询方式
**方案**：Go 标准库 `net.Resolver.LookupIP`。

理由：
- 子域名扫描是纯 DNS 查询场景，不需要自定义 DNS 协议、不需要 Zone Transfer。
- `net.LookupIP` 内部已经处理了递归查询、CNAME 链、A/AAAA 并发解析。
- 零额外依赖（不用引入 `miekg/dns`）。
- 性能足够：单次 DNS 查询 RTT 通常在几 ms 到几十 ms，并发控制由 `AdaptiveLimiter` 完成。

### 2.2 字典管理
**方案**：与 DirScanner 一致的 `go:embed` 内置字典 + 运行时外部字典加载。

内置字典文件：`rules/subdomain/default.txt`（嵌入 `dict/` 包）。
外部字典：通过 `--dict` 或 `--wordlists` 参数指定。

### 2.3 并发模型
**方案**：Worker 池 + Dictionary 队列（与 DirScanner 完全一致）。

```
Dictionary.Next() → Worker 池 → hits channel → Result 汇总
```

---

## 3. 目录结构

```
internal/core/scanner/subdomain/
├── subdomain_scanner.go   # 主扫描器，实现 runner.Runner 接口
├── dict/
│   ├── wordlists/
│   │   └── default.txt    # 内置子域名前缀字典（go:embed）
│   ├── dictionary.go      # 字典管理器（主队列 + 自定义加载）
│   └── wordlists.go       # 内置字典加载（LoadBuiltinWordlist）
├── engine/
│   ├── resolver.go        # DNS 查询引擎（封装 net.Resolver.LookupIP）
│   └── filter.go          # 结果过滤（可选：去重、IP 清洗）
└── result/
    └── result.go          # 输出数据结构（SubdomainResult / SubdomainEntry）
```

---

## 4. 数据结构设计

### 4.1 SubdomainEntry（单条命中记录）

```go
// result/result.go
type SubdomainEntry struct {
    Subdomain string   `json:"subdomain"`  // 完整子域名（如 api.example.com）
    IP        string   `json:"ip"`         // 首选 IP（A 记录第一个）
    IPs       []string `json:"ips,omitempty"` // 所有解析出的 IP
    CNAME     string   `json:"cname,omitempty"` // CNAME 记录（如有）
    RTT       int64    `json:"rtt,omitempty"`   // 查询往返时间（纳秒）
}
```

**设计说明**：
- `IP` 是首选 IP（A 记录的第一个），保证单字符串字段方便展示。
- `IPs` 是全部解析出的 IP 列表（A + AAAA），`omitempty` 保证无值时不序列化。
- `CNAME` 可选，有值时记录 CNAME 链终点的域名。
- `RTT` 用 `int64` 纳秒存储，JSON 序列化时转为整数（纳秒级精度）。

### 4.2 ScanStats（扫描统计）

```go
type ScanStats struct {
    TotalRequests  int `json:"total_requests"`
    SuccessfulReqs int `json:"successful_reqs"` // 成功解析的子域名数
    FailedReqs     int `json:"failed_reqs"`     // DNS 查询失败数
    Found          int `json:"found"`           // 命中数量（等于 SuccessfulReqs）
}
```

### 4.3 SubdomainResult（完整结果）

```go
type SubdomainResult struct {
    Domain    string             `json:"domain"`
    StartTime time.Time          `json:"start_time"`
    EndTime   time.Time          `json:"end_time,omitempty"`
    DictSize  int                `json:"dict_size"`
    Found     int                `json:"found"`
    Entries   []*SubdomainEntry  `json:"entries"`
    Stats     ScanStats          `json:"stats"`
}
```

**与 DirResult 对齐**：`Domain` 对应 `Target`，`Entries` 对应 `Hits`，语义一致。

---

## 5. 核心组件设计

### 5.1 SubdomainScanner（主扫描器）

```go
// subdomain_scanner.go
type SubdomainScanner struct {
    limiter *qos.AdaptiveLimiter
}

func NewSubdomainScanner() *SubdomainScanner {
    return &SubdomainScanner{
        limiter: qos.NewAdaptiveLimiter(100, 10, 2000),
    }
}

func (s *SubdomainScanner) Name() model.TaskType {
    return model.TaskTypeSubdomain
}

func (s *SubdomainScanner) Run(ctx context.Context, task *model.Task) (results []*model.TaskResult, err error) {
    defer recover()
    // 1. 解析参数
    opts := parseSubdomainOptions(task.Params)
    domain := normalizeDomain(task.Target)
    
    // 2. 初始化字典
    dictionary := dict.New(opts)
    
    // 3. 初始化 DNS 解析器
    resolver := engine.NewResolver(engine.ResolverConfig{
        Timeout: opts.Timeout,
    })
    
    // 4. 启动 Worker 池
    hits := make(chan *SubdomainEntry, 256)
    var wg sync.WaitGroup
    for i := 0; i < opts.Threads; i++ {
        wg.Add(1)
        go worker(ctx, s.limiter, resolver, dictionary, hits)
    }
    go func() { wg.Wait(); close(hits) }()
    
    // 5. 汇总结果
    result := NewSubdomainResult(domain, dictionary.Total())
    for entry := range hits {
        result.Add(entry)
    }
    result.Finish()
    
    return []*model.TaskResult{{
        TaskID: task.ID, Status: model.TaskStatusSuccess,
        Result: result,
    }}, nil
}
```

**Worker 函数**：

```go
func worker(ctx context.Context, limiter *qos.AdaptiveLimiter, resolver *engine.Resolver, dictionary *dict.Dictionary, hits chan<- *SubdomainEntry) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
        }
        
        prefix, ok := dictionary.Next()
        if !ok {
            return
        }
        
        if err := limiter.Acquire(ctx); err != nil {
            return
        }
        
        subdomain := prefix + "." + dictionary.Domain()
        entry, err := resolver.Query(ctx, subdomain)
        
        limiter.Release()
        
        if err != nil {
            continue // DNS 解析失败，跳过
        }
        
        select {
        case hits <- entry:
        case <-ctx.Done():
            return
        }
    }
}
```

### 5.2 DNS Resolver（引擎层）

```go
// engine/resolver.go
type Resolver struct {
    timeout time.Duration
    resolver *net.Resolver
}

func (r *Resolver) Query(ctx context.Context, domain string) (*SubdomainEntry, error) {
    start := time.Now()
    
    // A + AAAA 查询
    ips, err := r.resolver.LookupIP(ctx, "ip", domain)
    if err != nil {
        return nil, err
    }
    
    // CNAME 查询（独立查询）
    cname, _ := r.resolver.LookupCNAME(ctx, domain)
    
    return &SubdomainEntry{
        Subdomain: domain,
        IP:        ips[0].String(),
        IPs:       ipListToString(ips),
        CNAME:     cname,
        RTT:       int64(time.Since(start)),
    }, nil
}
```

**设计说明**：
- 使用 `net.Resolver` 而不是 `net.LookupIP`，方便在测试中注入 mock resolver。
- A/AAAA 查询使用 `LookupIP`（自动处理）。
- CNAME 查询使用 `LookupCNAME`（err 被忽略，因为 CNAME 不是必有的）。
- 超时通过 `ctx` 控制，与标准上下文一致。

### 5.3 Dictionary（字典管理）

与 DirScanner 的 `dict/dictionary.go` 高度一致：

```go
// dict/dictionary.go
type Dictionary struct {
    domain  string
    main    []string  // 主队列条目
    extra   []string  // 递归追加的条目（暂不实现，预留）
    cur     int
    total   int
}

func New(opts *SubdomainOptions) *Dictionary {
    // 1. 加载内置字典（默认）
    // 2. 如果用户指定了 --dict，加载用户字典替换内置字典
    // 3. 返回 Dictionary 实例
}

func (d *Dictionary) Next() (string, bool) {
    if d.cur >= len(d.main) {
        return "", false
    }
    entry := d.main[d.cur]
    d.cur++
    
    // 如果 extra 有数据，合并进来（预留扩展）
    if d.cur == len(d.main) && len(d.extra) > 0 {
        d.main = append(d.main, d.extra...)
        d.extra = nil
    }
    
    return d.main[d.cur-1], true
}
```

---

## 6. CLI 命令设计

CLI 命令已在 `cmd/agent/scan/subdomain.go` 和 `options/scan_subdomain.go` 中搭建骨架，只需补全 `RunE` 中的实现：

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if err := opts.Validate(); err != nil {
        return err
    }
    
    // 1. 创建 Scanner
    scanner := factory.NewSubdomainScanner()
    
    // 2. 转换 Task
    task := opts.ToTask()
    
    // 3. 执行扫描
    results, err := scanner.Run(cmd.Context(), task)
    if err != nil {
        return err
    }
    
    // 4. 输出结果
    return outputResult(results, opts.Output)
}
```

### 6.1 参数设计

基于现有的 `SubdomainScanOptions`：

| 参数 | Flag | 默认值 | 说明 |
|------|------|--------|------|
| 域名 | `-d, --domain` | （必填） | 目标主域名 |
| 字典路径 | `--dict` | 内置 default.txt | 自定义字典文件 |
| 并发线程 | `--threads` | 10 | 并发 Worker 数 |
| 超时 | `--timeout` | 30s | 单次 DNS 查询超时 |

**可后续扩展的参数**（不必在一期实现）：
- `--resolve`：是否解析PTR反向域名
- `--output, -o`：输出文件路径
- `--silent`：静默模式（仅输出 JSON）
- `--recursive`：递归扫描（对解析出的子域名继续扫描，`www.api.example.com`）

---

## 7. 与 RunnerManager 集成

在 `internal/core/runner/manager.go` 中注册：

```go
// 在 NewRunnerManager() 中
m.Register(factory.NewSubdomainScanner())
```

**不需要 Adapter 包装**，因为 `SubdomainScanner` 直接实现 `Runner` 接口（与 `DirScanner`/`WebScanner`/`ApiScanner` 一致）。

---

## 8. 与 Pipeline/Dispatcher 集成

当前 `ServiceDispatcher` 中已有以下调度逻辑：

```
Phase 1 (Sequential): Alive → Port → Service → OS
Phase 2 (Parallel):   Web → Vuln → Brute
```

子域名扫描的定位：

**选项 A**：在 `scan run` 全流程中不自动触发子域名扫描（仅 CLI 手动调用 `scan subdomain`）。
- 优点：简单，不增加 Pipeline 复杂度。
- 缺点：自动化程度低。

**选项 B**：在 Phase 1 开头加入子域名扫描（Alive 之后、Port 扫描之前）。
- 优点：先枚举子域名，再对每个子域名做后续扫描，覆盖更完整。
- 缺点：需要改造 PipelineContext 数据结构（新增 `[]*SubdomainEntry`），改动面较大。

**建议**：**先实现选项 A**，确保模块本身完整可用。后续当需要自动化集成时再升级到选项 B。子域名扫描的独立价值已经足够（很多场景下它就是唯一需要的扫描）。

---

## 9. 内置字典设计

### 9.1 文件位置
`internal/core/scanner/subdomain/dict/wordlists/default.txt`

### 9.2 字典内容
全量子域名前缀词表（1000+ 条），参考业界通用词表（Sublist3r、Subfinder、oneforall 等），覆盖常见服务、云/基础设施、开发/协作、测试环境、移动端、IoT、数据库等场景。

示例（部分）：
```
# 常见服务
www
mail
ftp
api
dev
stg
test
prod
admin
login

# 云/基础设施
cdn
static
assets
images
media
vpc
vpn
ns1
ns2

# 开发/协作
git
github
gitlab
jenkins
ci
cd
docs
wiki
jira
confluence

# 测试/预发布
qa
staging
sandbox
beta
alpha
preprod
pre
stg1
stg2

# 邮件/安全
smtp
pop
imap
exchange
owa
autodiscover
security
safe

# 监控/运维
monitor
nagios
zabbix
grafana
prometheus
kibana
elk

# ... 更多场景化前缀
```

**设计原则**：
- 数量：1000+ 条，覆盖面广，类似业界主流工具的词表规模。
- 分类组织：按场景分组（带 `#` 注释），方便用户理解和维护。
- 与 DirScanner 的内置字典文件风格一致（纯文本、支持 `#` 注释、空行跳过）。

---

## 10. 质量控制

### 10.1 单元测试

| 测试对象 | 测试内容 | 用例数 |
|----------|---------|--------|
| `dict/dictionary_test.go` | 内置字典加载、用户字典加载、`Next()` 遍历 | 5+ |
| `engine/resolver_test.go` | `Query()` 正确性（mock DNS） | 3+ |
| `result/result_test.go` | `NewSubdomainResult`、`Add()`、序列化 | 3+ |
| `subdomain_scanner_test.go` | 端到端（mock resolver，验证结果完整性） | 3+ |

### 10.2 验收标准
- `go build ./...` 通过
- `go vet ./internal/core/scanner/subdomain/...` 无告警
- `go test ./internal/core/scanner/subdomain/... -race -v` 全部通过
- CLI `go run ./cmd/agent scan subdomain -d example.com` 可执行且输出正确

### 10.3 性能目标
- 1000 条字典 + 10 线程：扫描一个域名 < 30 秒（局域网/本地 DNS）
- 1000 条字典 + 10 线程：扫描一个域名 < 120 秒（公网 DNS，有网络延迟）
- 默认 10 线程时不应出现明显线程饥饿（`AdaptiveLimiter` 的 AIMD 算法自动调整）

---

## 11. 与现有代码的关联

| 文件 | 改动 | 说明 |
|------|------|------|
| `options/scan_subdomain.go` | 微调 | 补充 `Timeout` 字段，调整 `ToTask()` |
| `cmd/agent/scan/subdomain.go` | 重写 | 实现 `RunE`，调用 `SubdomainScanner` |
| `runner/manager.go` | 新增一行 | `m.Register(factory.NewSubdomainScanner())` |
| `factory/` | 新增 | `NewSubdomainScanner()` 工厂函数 |
| 新增 `internal/core/scanner/subdomain/` | 新包 | 核心实现 |

**零破坏性**：所有改动都是新增文件或微调现有 stub，不影响任何其他扫描器。

---

## 12. 后续演进路线

### Phase 1（本期）
- [x] 基础字典枚举（A/AAAA/CNAME）
- [x] CLI 命令实现
- [x] RunnerManager 注册
- [x] 端到端测试

### Phase 2（可选）
- [ ] 递归扫描：对解析出的子域名继续枚举（`sub.www.example.com`）
- [ ] PTR 反向解析：补充主机名信息
- [ ] 自定义 DNS 服务器：支持 `--dns-server` 指定解析器

### Phase 3（可选）
- [ ] CT（Certificate Transparency）日志源：接入 Crtsh 等公开 API
- [ ] 增量扫描：记录上次结果，仅展示新增子域名
- [ ] Master 集群下发：完善 `task_to_core.go` 映射
