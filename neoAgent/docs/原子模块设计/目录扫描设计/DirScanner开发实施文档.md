# NeoAgent DirScanner 开发实施文档

**日期**: 2026-08-17
**状态**: 待开发
**依据**:
- [目录扫描模块设计文档.md](./目录扫描模块设计文档.md)
- [NeoAgent目录扫描模块功能参数设计.md](./NeoAgent目录扫描模块功能参数设计.md)

---

## 开发原则

1. **逐任务交付**：每个任务完成后独立可测试，不做大爆炸式提交
2. **不动现有代码**：除非任务明确要求修改，否则不改动已有文件
3. **测试先行**：每个核心组件必须有单元测试覆盖才算完成
4. **每完成一条勾选**：用 `[x]` 标记完成状态，方便追踪进度

---

## 阶段划分

```
Phase 1: 基础设施（字典 + 请求器 + 过滤器）  ← 当前阶段
Phase 2: 核心引擎（通配符检测 + 并发调度）
Phase 3: CLI 接入（参数 + 命令 + 输出）
Phase 4: 集成（RunnerManager + 验收测试）
```

---

## Phase 1：基础设施

### Task 1.1 — 复制 dirsearch 字典文件

**目标**：将 dirsearch 的字典文件复制到 `internal/core/scanner/dir/dict/wordlists/` 下，作为 `go:embed` 嵌入源

**来源目录**：`docs/references/dirsearch/db/`

**操作清单**：

- [ ] 创建目录 `internal/core/scanner/dir/dict/wordlists/`
- [ ] 复制 `docs/references/dirsearch/db/dicc.txt` → `dict/wordlists/dicc.txt`
- [ ] 复制 `docs/references/dirsearch/db/400_blacklist.txt` → `dict/wordlists/blacklist/400_blacklist.txt`
- [ ] 复制 `docs/references/dirsearch/db/403_blacklist.txt` → `dict/wordlists/blacklist/403_blacklist.txt`
- [ ] 复制 `docs/references/dirsearch/db/500_blacklist.txt` → `dict/wordlists/blacklist/500_blacklist.txt`
- [ ] 复制 `docs/references/dirsearch/db/categories/` 全部内容 → `dict/wordlists/categories/`（保留子目录结构）
- [ ] 复制 `docs/references/dirsearch/db/templates/` 全部内容 → `dict/wordlists/templates/`
- [ ] 复制 `docs/references/dirsearch/db/user-agents.txt` → `dict/wordlists/user-agents.txt`
- [ ] 确认文件数量与来源一致（执行 `ls -la` 对比）

**验收**：`dict/wordlists/` 目录结构与设计文档 2.1 节一致

---

### Task 1.2 — 实现 `dict/wordlists.go`（go:embed 嵌入声明）

**目标**：通过 `go:embed` 将 `wordlists/` 目录嵌入二进制，并提供加载接口

**文件**：`internal/core/scanner/dir/dict/wordlists.go`

**实现要求**：

- [ ] 使用 `//go:embed wordlists` 嵌入整个目录（`embed.FS` 类型）
- [ ] 实现 `LoadBuiltinWordlist() ([]string, error)`：加载 `wordlists/dicc.txt`，返回路径行列表，自动去空行/去注释（`#` 开头）
- [ ] 实现 `LoadCategoryWordlist(category string) ([]string, error)`：按技术栈名称（如 `wordpress`、`spring`）查找对应文件并加载，文件不存在返回 `ErrCategoryNotFound`
- [ ] 实现 `LoadTemplateWordlist(template string) ([]string, error)`：按场景名称（如 `admin`、`api`）加载模板字典
- [ ] 实现 `LoadBlacklist(statusCode int) (map[string]bool, error)`：加载对应状态码的黑名单，支持 400/403/500
- [ ] 实现 `LoadUserAgents() ([]string, error)`：加载 UA 列表
- [ ] 实现 `ListCategories() []string`：列出所有可用技术栈名称（遍历 `categories/` 子目录）

**单元测试** `dict/wordlists_test.go`：

- [ ] `TestLoadBuiltinWordlist`：验证加载条目数 > 0，不含空行，不含 `#` 开头的行
- [ ] `TestLoadCategoryWordlist`：验证 `wordpress`/`spring`/`django` 等可正常加载
- [ ] `TestLoadCategoryWordlist_NotFound`：验证不存在的 category 返回 `ErrCategoryNotFound`
- [ ] `TestLoadBlacklist`：验证 403 黑名单包含 `/.git/config`
- [ ] `TestLoadUserAgents`：验证列表非空

---

### Task 1.3 — 实现 `dict/template.go`（模板令牌系统）

**目标**：实现 `%EXT%` 令牌展开逻辑（参考设计文档 3.4 节）

**文件**：`internal/core/scanner/dir/dict/template.go`

**实现要求**：

- [ ] 实现 `ExpandLine(line string, extensions []string, mode ExtensionMode) []string`
  - `ExtensionModeClassic`（默认）：仅替换 `%EXT%`；无令牌的行原样返回
  - `ExtensionModeForce`：无扩展的路径，追加 `{line}.{ext}` 形式的变体
- [ ] `ExtensionMode` 类型为 `int` const，不用 string
- [ ] 路径中若含 `%EXT%` 但 `extensions` 为空，返回原行（不展开）
- [ ] 不实现 `overwrite_extensions`（已决策不做，见参数设计文档 3.2 节）

**单元测试** `dict/template_test.go`：

- [ ] `TestExpandLine_Classic_WithEXT`：`/backup.%EXT%` + `[php,bak]` → `[/backup.php, /backup.bak]`
- [ ] `TestExpandLine_Classic_NoEXT`：`/admin` + `[php]` → `[/admin]`（原样返回）
- [ ] `TestExpandLine_Force`：`/admin` + `[php,html]` → `[/admin, /admin.php, /admin.html]`
- [ ] `TestExpandLine_EmptyExtensions`：含 `%EXT%` 但 extensions 为空 → 原样返回

---

### Task 1.4 — 实现 `dict/dictionary.go`（双队列字典管理器）

**目标**：实现线程安全的双队列字典迭代器（参考设计文档 3.3 节）

**文件**：`internal/core/scanner/dir/dict/dictionary.go`

**数据结构**：

```go
type Dictionary struct {
    mu       sync.Mutex
    main     []string  // 主字典（预加载，只读）
    extra    []string  // 动态扩展队列（追加）
    mainIdx  int
    extraIdx int
    maxSize  int       // 字典条目上限（默认 500_000）
    total    int       // 已生成条目计数（含展开）
}
```

**实现要求**：

- [ ] `New(opts *DirOptions) (*Dictionary, error)`：
  - 加载内置字典（`LoadBuiltinWordlist()`）
  - 若 `opts.Wordlists` 非空，额外加载用户指定文件（追加到 main）
  - 扫描 `rules/dir/custom/` 下所有 `.txt` 文件，追加到 main（运行时 `os.ReadFile` 加载，多路径 fallback 依次尝试：`rules/dir/custom/`、`../rules/dir/custom/`、`../../rules/dir/custom/` 等；**不引用 `WebScanner` 的任何代码**，加载模式相同但各自独立实现，符合原子扫描器隔离原则）
  - 对 main 按 `opts` 中的扩展配置展开（调用 `ExpandLine`）
  - 应用 `opts.Uppercase`/`Lowercase`/`Capital` 大小写变换
  - 应用 `opts.Prefixes`/`Suffixes` 前后缀变换
  - 超过 `maxSize` 时截断并记录警告日志
- [ ] `Next() (string, bool)`：先取 extra（优先），再取 main；两者耗尽返回 `("", false)`；线程安全
- [ ] `AddExtra(paths ...string)`：追加到 extra 队列头部；检查 total 是否超限；线程安全
- [ ] `Len() int`：返回剩余条目数（估算）
- [ ] `Total() int`：返回已处理总条目数

**单元测试** `dict/dictionary_test.go`：

- [ ] `TestDictionary_Next`：顺序取出所有条目，最终返回 false
- [ ] `TestDictionary_AddExtra_Priority`：AddExtra 后 Next 优先返回 extra 条目
- [ ] `TestDictionary_MaxSize`：超过 maxSize 时截断
- [ ] `TestDictionary_Concurrent`：100 goroutine 并发 Next，无 data race（`go test -race`）
- [ ] `TestDictionary_UserCustomWordlist`：`rules/dir/custom/` 下放一个测试文件，验证被加载

---

### Task 1.5 — 实现 `result/result.go`（数据模型）

**目标**：定义 DirScanner 输出的数据结构（参考设计文档 8.1 节）

**文件**：`internal/core/scanner/dir/result/result.go`

**实现要求**：

- [ ] 定义 `DirHit` 结构体（Path/Status/Size/Words/Lines/Title/Location/ContentType）
- [ ] 定义 `ScanStats` 结构体（TotalRequests/SuccessfulReqs/FilteredReqs/WildcardReqs/ErrorReqs/AvgRTT/MaxRTT/MinRTT）
- [ ] 定义 `DirResult` 结构体（Target/StartTime/EndTime/DictSize/Found/Hits/Stats）
- [ ] 实现 `DirResult.Add(hit *DirHit)`：追加命中，更新 Found 计数
- [ ] 实现 `DirResult.Headers() []string` 和 `DirResult.Rows() [][]string`：实现 `TabularData` 接口，供 ConsoleReporter 使用
- [ ] `DirHit.String()` 格式：`[STATUS] /path (SIZE bytes)`

---

### Task 1.6 — 实现 `engine/filter.go`（多层过滤管道）

**目标**：实现 3 层过滤管道（参考设计文档 5.1 节）

**文件**：`internal/core/scanner/dir/engine/filter.go`

**数据结构**：

```go
type FilterConfig struct {
    IncludeStatus    []int
    ExcludeStatus    []int
    ExcludeSize      []int64
    ExcludeKeywords  []string
    ExcludeRegex     []*regexp.Regexp
    ErrorPaths       map[int]map[string]bool  // statusCode → path set
}

type Response struct {
    StatusCode  int
    Body        string
    Size        int64
    ContentType string
    Location    string  // 重定向目标
}
```

**实现要求**：

- [ ] `NewFilter(cfg FilterConfig) *Filter`
- [ ] `Filter.Match(resp *Response, path string) bool`：
  - Layer 1a：状态码白名单过滤（IncludeStatus 非空时，不在白名单则排除）
  - Layer 1b：状态码黑名单过滤（ExcludeStatus，默认 `[404, 500, 502, 503]`）
  - Layer 1c：响应大小过滤（ExcludeSize）
  - Layer 2：错误路径黑名单（errorPaths[statusCode][path]）
  - Layer 3a：关键词排除（ExcludeKeywords，Body 中包含则排除）
  - Layer 3b：正则排除（ExcludeRegex）
  - 全部通过返回 `true`
- [ ] `DefaultExcludeStatus() []int`：返回 `[404, 500, 502, 503]`

**单元测试** `engine/filter_test.go`：

- [ ] `TestFilter_ExcludeStatus`：404 被排除
- [ ] `TestFilter_IncludeStatus`：仅保留白名单状态码
- [ ] `TestFilter_ErrorPath`：`/.git/config + 403` 命中黑名单被排除
- [ ] `TestFilter_ExcludeKeyword`：Body 含关键词被排除
- [ ] `TestFilter_ExcludeRegex`：Body 匹配正则被排除
- [ ] `TestFilter_AllPass`：正常 200 响应通过所有层

---

### Task 1.7 — 实现 `engine/requester.go`（HTTP 请求器）

**目标**：实现保留原始路径的 HTTP 请求器（参考设计文档 6.1~6.3 节）

**文件**：`internal/core/scanner/dir/engine/requester.go`

**实现要求**：

- [ ] 定义 `RequesterConfig` 结构体（Timeout/MaxRetries/Proxy/Method/Headers/UserAgents/RateLimit/Delay/FollowRedirects/IP/NetworkInterface）
- [ ] 实现 `PathPreservingRoundTripper`：自定义 `RoundTripper`，通过直接操作 TCP 连接或 `Opaque` 字段绕过 Go 标准库的 URL 规范化，保留 `%2F`、`..` 等原始字符
- [ ] `NewRequester(cfg RequesterConfig) *Requester`：
  - 构建携带 `PathPreservingRoundTripper` 的 `http.Client`
  - 配置代理（`cfg.Proxy`）
  - 若 `cfg.FollowRedirects == false`，设置 `CheckRedirect` 返回 `http.ErrUseLastResponse`
- [ ] `Requester.Do(ctx context.Context, baseURL, path string) (*Response, error)`：
  - 拼接请求 URL
  - 注入请求头（含 User-Agent 随机选择）
  - 重试循环（最多 `MaxRetries` 次，指数退避）
  - 错误分类：`ErrConnectionTimeout`、`ErrConnectionRefused`、`ErrDNSFailed`、`ErrSSLError`、`ErrOther`
  - 连接失败不重试，超时重试
  - 读取响应体上限 10MB（防内存爆炸）
  - 返回 `*Response`（含 Body/StatusCode/Size/ContentType/Location）

**单元测试** `engine/requester_test.go`（使用 `httptest.Server`）：

- [ ] `TestRequester_BasicGet`：正常 200 响应
- [ ] `TestRequester_PathPreserving`：路径中含 `%2F` 不被转义
- [ ] `TestRequester_Timeout`：服务器超时，验证错误类型
- [ ] `TestRequester_Retry`：第 1 次超时、第 2 次成功，验证重试逻辑
- [ ] `TestRequester_NoFollowRedirect`：301 不跟随重定向，返回 301

---

## Phase 2：核心引擎

### Task 2.1 — 实现 `engine/scanner.go`（通配符检测引擎）

**目标**：移植 dirsearch `Scanner + DynamicContentParser`，Go 语言实现（参考设计文档 4.1~4.4 节）

**文件**：`internal/core/scanner/dir/engine/scanner.go`

**数据结构**：

```go
type WildcardScanner struct {
    requester          *Requester
    baseURL            string
    sampleThreshold    int               // 采样阈值，默认 5
    samples            map[string][]string // path-prefix → []body
    staticPatterns     map[string][]string // path-prefix → 静态词列表
    isStatic           map[string]bool
    baseWordCount      map[string]int
    fingerprints       map[string]int    // fp → 出现次数
    wildCardFPs        map[string]bool   // 已确认通配符指纹
    fingerprintThresh  int               // 自校准阈值，默认 10
    mu                 sync.Mutex
}
```

**实现要求**：

- [ ] `NewWildcardScanner(req *Requester, baseURL string) *WildcardScanner`
- [ ] `normalizeDynamic(content string) string`：移除 UUID/长hex/时间戳/Base64（4 个正则，见设计文档 4.3 节）；正则**编译一次，包级变量**，不在函数内 `MustCompile`
- [ ] `extractStaticPatterns(body1, body2 string) ([]string, int)`：返回共同词列表和基准词数
- [ ] `fingerprint(resp *Response) string`：按设计文档 4.4 节格式生成指纹
- [ ] `WildcardScanner.Check(ctx context.Context, pathPrefix string, resp *Response) (isUnique bool, reason string)`：
  - 检查指纹是否已知通配符（`wildCardFPs`）→ 直接返回 false
  - 未达采样阈值 → 发送探测请求（随机不存在路径 `pathPrefix + randStealthWord()`），更新样本
  - 采样未完成 → 返回 `(false, "sampling")`（采样阶段压制输出）
  - 采样完成后提取 `staticPatterns`
  - 比对当前响应与静态模式：相似度 ≥ 0.9 且长度差异 ≤ 35% → 通配符，返回 false
  - 自校准：调用 `autoCalibrate`，更新 `fingerprints`，超阈值加入 `wildCardFPs`
  - 通过所有检测 → 返回 `(true, "unique")`
- [ ] `randStealthWord() string`：生成随机 8 字符字符串（避免与真实路径冲突）

**单元测试** `engine/scanner_test.go`（使用 `httptest.Server`）：

- [ ] `TestNormalizeDynamic`：UUID/hex/时间戳/Base64 被正确替换
- [ ] `TestExtractStaticPatterns`：两个相同结构但动态内容不同的响应，提取到共同词
- [ ] `TestWildcardScanner_CDN`：mock 服务器对任意路径返回同一个 404 页面 → Check 返回 false
- [ ] `TestWildcardScanner_Unique`：mock 服务器对特定路径返回独特内容 → Check 返回 true
- [ ] `TestWildcardScanner_SamplingPhase`：前 5 次 Check 均返回 `(false, "sampling")`
- [ ] `TestWildcardScanner_Concurrent`：并发 Check，无 data race

---

### Task 2.2 — 实现 `dir_scanner.go`（主扫描器）

**目标**：实现 `DirScanner` 核心逻辑，接入 Worker 池 + 递归调度（参考设计文档 7.1 节）

**文件**：`internal/core/scanner/dir/dir_scanner.go`

**实现要求**：

- [ ] 定义 `DirScanner` 结构体，持有 `limiter *qos.AdaptiveLimiter` 字段（**不使用 Go struct embedding**，而是普通字段持有，与 `ApiScanner`/`IpAliveScanner` 的做法一致，保持封装边界清晰）
- [ ] 实现 `Name() model.TaskType`：返回 `model.TaskTypeDirScan`
- [ ] 实现 `Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error)`：
  - 调用 `parseDirOptions(task.Params)` 解析参数（见 Task 3.1）
  - 初始化 `Dictionary`、`Requester`、`Filter`、`WildcardScanner`
  - 处理 `MaxTime`/`TargetMaxTime`：若非 0，用 `context.WithTimeout` 包装 ctx
  - 启动 Worker 池（goroutine 数 = `opts.Threads`）
  - 等待所有 Worker 完成，收集结果
  - 返回 `[]*model.TaskResult`（包含 `DirResult`）
- [ ] 实现 `worker(ctx, dict, req, filter, scanner, results chan, opts)`：
  - 从 `dict.Next()` 取路径
  - `limiter.Acquire(ctx)` 速率控制
  - 若 `opts.Delay > 0`，`time.Sleep(opts.Delay)`
  - `req.Do()` 发送请求
  - `filter.Match()` 过滤
  - `scanner.Check()` 通配符检测
  - 命中则发送到 results channel，并判断是否递归
- [ ] 实现 `shouldRecursion(resp, path, opts)`（三种递归模式，见设计文档 7.1 节）
- [ ] 实现 `generateSubpaths(path string) []string`（深度递归子路径生成）
- [ ] 递归调用 `dict.AddExtra(subpaths...)` 时检查递归深度（`MaxRecursionDepth`），超限则跳过
- [ ] 实现 `parseDirOptions(params map[string]interface{}) *DirOptions`：从 task.Params 解析所有字段，含默认值（`Threads=25`、`Timeout=10s`、`MaxRetries=2`、`ExcludeStatus=[404,500,502,503]`、`MaxRecursionDepth=3`）

**单元测试** `dir_scanner_test.go`（使用 `httptest.Server`）：

- [ ] `TestDirScanner_BasicScan`：mock 服务器有 `/admin`（200）和 `/secret`（200），其他 404 → 均被发现
- [ ] `TestDirScanner_WildcardCDN`：mock 服务器所有路径返回同一个 404 页面 → 零结果（无误报）
- [ ] `TestDirScanner_Recursive`：`/api/`（301）触发递归，子路径被扫描
- [ ] `TestDirScanner_MaxRecursionDepth`：递归深度限制为 2 时，第 3 层不扫描
- [ ] `TestDirScanner_ContextCancel`：context 取消后扫描立即停止，无 goroutine 泄漏（检查 goroutine 数量）
- [ ] `TestDirScanner_MaxEntries`：字典超过 500K 条时被截断，扫描不崩溃

---

## Phase 3：CLI 接入

### Task 3.1 — 重写 `internal/core/options/scan_dir.go`

**目标**：将现有骨架扩展为完整的 `DirScanOptions`，覆盖所有已决策参数（参考参数设计文档第 4 章）

**文件**：`internal/core/options/scan_dir.go`

**当前状态**：仅有 `Target/Dict/Extensions/Threads/Output` 4 个字段，需全部重写

**实现要求**：

- [ ] 定义 `DirScanOptions` 结构体，字段与设计文档 8.2 节 `DirOptions` 完全对应（但不含 `ExcludeRegex`，由 `Validate` 编译）
- [ ] `NewDirScanOptions() *DirScanOptions`：设置所有默认值：
  - `Threads = 25`
  - `Timeout = 10`（秒）
  - `MaxRetries = 2`
  - `ExcludeStatus = "404,500,502,503"`
  - `Recursive = true`
  - `MaxRecursionDepth = 3`
  - `Extensions = "php,asp,html,js,json,bak,git,env"`（默认扩展列表）
  - `HTTPMethod = "GET"`
- [ ] `Validate() error`：
  - Target 必填
  - 解析 ExcludeStatus 字符串为 `[]int`（逗号分隔），非法值报错
  - 解析 ExcludeRegex 字符串为 `[]*regexp.Regexp`，编译失败报错
  - `MaxRecursionDepth` 最大值限制为 10（防误操作）
  - `Threads` 范围 1~500
  - `Timeout` 范围 1~300
- [ ] `ToTask() *model.Task`：将所有字段写入 `task.Params`（key 名统一使用下划线命名，如 `max_recursion_depth`）；复用 `Output.ApplyToParams()`

---

### Task 3.2 — 重写 `cmd/agent/scan/dir.go`（CLI 命令）

**目标**：将现有占位命令扩展为完整的 `scan dir` 子命令（参考参数设计文档 4.1~4.3 节）

**文件**：`cmd/agent/scan/dir.go`

**当前状态**：仅注册了 `target/dict/extensions/threads`，`RunE` 直接返回 `not implemented`

**P0 核心参数**（必须实现）：

- [ ] `<target>`（位置参数，args[0]）或 `-t/--target` flag，两者互斥，位置参数优先
- [ ] `--extensions/-e string`（默认 `php,asp,html,js,json,bak,git,env`）
- [ ] `--wordlists/-w string`（字典文件路径，默认空=使用内置字典）
- [ ] `--threads/-t int`（默认 25）
- [ ] `--timeout int`（默认 10）
- [ ] `--recursive/-r`（默认 true）
- [ ] 全局继承 `--oj`/`--oc`（已在 `scan` 父命令注册，无需重复）

**P1 重要参数**（必须实现）：

- [ ] `--force-extensions/-f bool`
- [ ] `--exclude-status/-x string`（默认 `"404,500,502,503"`）
- [ ] `--deep-recursive bool`
- [ ] `--max-recursion-depth/-R int`（默认 3）
- [ ] `--verbose/-v bool`
- [ ] `--quiet/-q bool`
- [ ] `--header/-H string`（可多次指定，cobra `StringArrayVar`）
- [ ] `--proxy/-p string`
- [ ] `--follow-redirects/-F bool`
- [ ] `--max-retries int`（默认 2）
- [ ] `--method/-m string`（默认 `GET`）

**P2 可选参数**（实现，但不标注 required）：

- [ ] `--exclude-extensions string`
- [ ] `--prefixes string`（逗号分隔）
- [ ] `--suffixes string`（逗号分隔）
- [ ] `--uppercase/-U bool`
- [ ] `--lowercase/-L bool`
- [ ] `--capital/-C bool`
- [ ] `--include-status/-i string`
- [ ] `--exclude-text string`（可多次，`StringArrayVar`）
- [ ] `--exclude-regex string`
- [ ] `--exclude-size string`（逗号分隔的字节数）
- [ ] `--rate-limit int`
- [ ] `--max-time int`
- [ ] `--target-max-time int`
- [ ] `--headers-file string`
- [ ] `--random-agent bool`
- [ ] `--auth string`
- [ ] `--user-agent string`
- [ ] `--ip string`
- [ ] `--network-interface string`
- [ ] `--targets-file/-l string`（多目标文件）
- [ ] `--delay int`

**RunE 逻辑**：

- [ ] 调用 `opts.Validate()`，失败则返回错误
- [ ] 调用 `opts.ToTask()` 生成 task
- [ ] 通过 `RunnerManager` 执行（复用其他 `scan` 子命令的 `runTask()` 封装）
- [ ] 结果通过 `ConsoleReporter`（verbose/quiet 控制详细程度）和 `--oj`/`--oc` 输出
- [ ] 移除 `not implemented yet` 占位错误

---

## Phase 4：集成

### Task 4.1 — 注册到 RunnerManager

**目标**：将 `DirScanner` 注册到 `RunnerManager`，使 Master 下发的 `dir_scan` 任务也能执行

**文件**：`internal/core/factory/runner_factory.go`（或现有注册文件，需确认实际路径）

**操作清单**：

- [ ] 确认 RunnerManager 的工厂函数位置（参考 brute 模块的注册方式）
- [ ] 在工厂函数中添加 `dir.NewDirScanner()` 的注册
- [ ] 确认 `model.TaskTypeDirScan` 常量已定义（检查 `internal/core/model/`）
- [ ] 如 `TaskTypeDirScan` 未定义，补充常量定义

---

### Task 4.2 — E2E 验收测试

**目标**：验证完整扫描流程，对应设计文档 11.1 功能验收清单

**文件**：`internal/core/scanner/dir/dir_scanner_e2e_test.go`

**测试用例**（每条对应设计文档 11.1 的一行验收标准）：

- [ ] `TestE2E_KnownSensitivePaths`：mock 服务器开放 `/.git/config`（200）、`/.env`（200）、`/admin`（200）→ 三者均被发现，状态码正确
- [ ] `TestE2E_CDNWildcard`：mock 服务器对所有路径返回相同 404 页面（体积相同、内容相同）→ 扫描结果为空，无误报
- [ ] `TestE2E_BlacklistPath`：mock 服务器对 `/.git/config` 返回 403 → 命中黑名单，不出现在结果中
- [ ] `TestE2E_ExtensionTemplate`：内置字典含 `/backup.%EXT%`，传 `extensions=php,bak` → 扫描队列含 `/backup.php` 和 `/backup.bak`
- [ ] `TestE2E_HighConcurrency`：`--threads 100`，mock 服务器正常响应 → 扫描完成，无 goroutine 泄漏，无 panic
- [ ] `TestE2E_RetryOnTimeout`：mock 服务器前 2 次不响应、第 3 次成功 → 自动重试，不阻塞
- [ ] `TestE2E_BasicRecursion`：`/api/`（301→/api/v1/）触发递归，`/api/v1/users`（200）被发现
- [ ] `TestE2E_DeepRecursion`：`--deep-recursive`，`/a/b/c/` 命中 → `/a/`、`/a/b/` 被追加扫描
- [ ] `TestE2E_OutputJSON`：指定 `--oj /tmp/result.json`，扫描完成后文件存在，JSON 格式合法，字段完整

---

### Task 4.3 — 性能基准测试

**目标**：对应设计文档 11.2 性能验收标准

**文件**：`internal/core/scanner/dir/dir_scanner_bench_test.go`

- [ ] `BenchmarkDirScanner_SingleThread`：单线程扫 10K 条字典，验证吞吐（目标：内网 ≤ 60s）
- [ ] `BenchmarkDirScanner_25Threads`：25 线程扫 10K 条字典（目标：内网 ≤ 5s）
- [ ] `BenchmarkDirScanner_MemoryUsage`：加载全部字典后测量内存（目标：≤ 150MB）
- [ ] `BenchmarkWildcardScanner_Sampling`：通配符检测采样开销（目标：≤ 5 次额外请求）

---

## 关键约束与注意事项

### 不可违反的约束

| 约束 | 说明 |
|------|------|
| `go:embed` 只能在包目录内 | 字典文件必须在 `dict/wordlists/` 下，不能引用 `rules/` |
| 不实现 `overwrite_extensions` | 已决策，见参数设计文档 3.2 节 |
| 文件输出只用 `--oj`/`--oc` | 不引入 `--output-format`/`--output-file`，见设计文档 3.7 节 |
| 递归深度必须有上限 | 即使用户只传 `-r`，也自动限制 3 层，见参数设计文档 7.5 节 |
| 通配符检测默认开启 | 无需用户参数，采样阈值 5，见参数设计文档 7.3 节 |
| 正则变量包级编译 | `normalizeDynamic` 的 4 个正则不在函数内 `MustCompile`，避免每次调用重新编译 |
| **原子扫描器隔离原则**（铁律） | `scanner/dir/` 包内的所有子模块（`dict/`、`engine/`、`result/`）只能被 `DirScanner` 自身 import，不能被其他扫描器包引用；`DirScanner` 也不能 import `scanner/web/`、`scanner/api/` 等其他扫描器的内部包；见 `docs/00.原子扫描器开发指南[定稿].md` 第 0.1 节 |
| **不使用 struct embedding** | `DirScanner` 持有 `limiter`/`config` 等字段，一律用普通字段赋值，不用 Go embedding 提升方法，保持封装边界清晰 |

### 与现有代码的接口约定

| 接口 | 说明 |
|------|------|
| `model.TaskTypeDirScan` | 确认常量存在，Task.Params key 统一下划线命名 |
| `OutputOptions.ApplyToParams()` | `ToTask()` 中复用，`--oj`/`--oc` 自动写入 Params |
| `qos.AdaptiveLimiter` | 复用 `internal/core/lib/network/qos/`（公共基础设施层，非任何扫描器私有），`DirScanner` 独立实例化自己的 limiter，不与其他扫描器共享实例 |
| `Runner` 接口 | 实现 `Name() TaskType` 和 `Run(ctx, task) ([]*TaskResult, error)` |
| `TabularData` 接口 | `DirResult` 实现 `Headers()/Rows()`，供 ConsoleReporter 渲染 |

### 依赖检查（开发前确认）

- [ ] `internal/core/lib/network/qos/` 路径确认（Task 2.2 开始前）
- [ ] `model.TaskTypeDirScan` 常量确认（Task 4.1 前）
- [ ] `ConsoleReporter` 的 `TabularData` 接口签名确认（Task 1.5 前）
- [ ] `RunnerManager` 工厂函数位置确认（Task 4.1 前）
- [ ] `rules/dir/custom/` 目录已存在（Task 1.4 前，已由用户创建）

---

## 任务完成状态跟踪

| 任务 | 状态 | 备注 |
|------|:---:|------|
| 1.1 复制字典文件 | ⬜ 待开始 | |
| 1.2 wordlists.go | ⬜ 待开始 | |
| 1.3 template.go | ⬜ 待开始 | |
| 1.4 dictionary.go | ⬜ 待开始 | 依赖 1.2、1.3 |
| 1.5 result.go | ⬜ 待开始 | |
| 1.6 filter.go | ⬜ 待开始 | |
| 1.7 requester.go | ⬜ 待开始 | |
| 2.1 scanner.go（通配符） | ⬜ 待开始 | 依赖 1.7 |
| 2.2 dir_scanner.go | ⬜ 待开始 | 依赖 1.2~1.7、2.1 |
| 3.1 scan_dir.go（Options） | ⬜ 待开始 | 依赖 2.2 |
| 3.2 dir.go（CLI） | ⬜ 待开始 | 依赖 3.1 |
| 4.1 RunnerManager 注册 | ⬜ 待开始 | 依赖 2.2 |
| 4.2 E2E 验收测试 | ⬜ 待开始 | 依赖 2.2、3.1 |
| 4.3 性能基准测试 | ⬜ 待开始 | 依赖 2.2 |
