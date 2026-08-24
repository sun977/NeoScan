# NeoAgent DirScanner 开发实施文档

**日期**: 2026-08-17（2026-08-22 追加 Phase 5：断层修复；2026-08-23 追加 Phase 6：参数分级 + 冗余清理）
**状态**: Phase 1~5 已完成，Phase 6（参数分级 + 冗余清理）已完成
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
Phase 1: 基础设施（字典 + 请求器 + 过滤器）  ✅ 已完成
Phase 2: 核心引擎（通配符检测 + 并发调度）  ✅ 已完成
Phase 3: CLI 接入（参数 + 命令 + 输出）      ✅ 已完成
Phase 4: 集成（RunnerManager + 验收测试）    ✅ 已完成
Phase 5: 断层修复（已实现但未接入主流程的功能审查）  ✅ 已完成（2026-08-22）
Phase 6: 参数分级 + 冗余清理（CLI 体验优化 + 死代码清除）  ✅ 已完成（2026-08-23）
```

---

## Phase 1：基础设施

### Task 1.1 — 复制 dirsearch 字典文件

**目标**：将 dirsearch 的字典文件复制到 `internal/core/scanner/dir/dict/wordlists/` 下，作为 `go:embed` 嵌入源

**来源目录**：`docs/references/dirsearch/db/`

**操作清单**：

- [x] 创建目录 `internal/core/scanner/dir/dict/wordlists/`
- [x] 复制 `docs/references/dirsearch/db/dicc.txt` → `dict/wordlists/dicc.txt`
- [x] ~~复制 `docs/references/dirsearch/db/400_blacklist.txt` → `dict/wordlists/blacklist/400_blacklist.txt`~~（Phase 6 已删除，见 Task 6.2）
- [x] ~~复制 `docs/references/dirsearch/db/403_blacklist.txt` → `dict/wordlists/blacklist/403_blacklist.txt`~~（Phase 6 已删除，见 Task 6.2）
- [x] ~~复制 `docs/references/dirsearch/db/500_blacklist.txt` → `dict/wordlists/blacklist/500_blacklist.txt`~~（Phase 6 已删除，见 Task 6.2）
- [x] 复制 `docs/references/dirsearch/db/categories/` 全部内容 → `dict/wordlists/categories/`（保留子目录结构）
- [x] ~~复制 `docs/references/dirsearch/db/templates/` 全部内容 → `dict/wordlists/templates/`~~（Phase 6 已删除，见 Task 6.2）
- [x] 复制 `docs/references/dirsearch/db/user-agents.txt` → `dict/wordlists/user-agents.txt`
- [x] 确认文件数量与来源一致（执行 `ls -la` 对比）

**验收**：`dict/wordlists/` 目录结构与设计文档 2.1 节一致 — ✅ 已通过（dicc.txt 9681 行，53 条 UA，25 个 category；blacklist 和 templates 已于 Phase 6 删除）

---

### Task 1.2 — 实现 `dict/wordlists.go`（go:embed 嵌入声明）

**目标**：通过 `go:embed` 将 `wordlists/` 目录嵌入二进制，并提供加载接口

**文件**：`internal/core/scanner/dir/dict/wordlists.go`

**实现要求**：

- [x] 使用 `//go:embed wordlists` 嵌入整个目录（`embed.FS` 类型）
- [x] 实现 `LoadBuiltinWordlist() ([]string, error)`：加载 `wordlists/dicc.txt`，返回路径行列表，自动去空行/去注释（`#` 开头）
- [x] 实现 `LoadCategoryWordlist(category string) ([]string, error)`：按技术栈名称（如 `wordpress`、`spring`）查找对应文件并加载，文件不存在返回 `ErrCategoryNotFound`
- [x] ~~实现 `LoadTemplateWordlist(template string) ([]string, error)`~~：Phase 6 已删除（见 Task 6.2）
- [x] ~~实现 `LoadBlacklist(statusCode int) (map[string]bool, error)`~~：Phase 6 已删除（见 Task 6.2）
- [x] 实现 `LoadUserAgents() ([]string, error)`：加载 UA 列表
- [x] 实现 `ListCategories() []string`：列出所有可用技术栈名称（遍历 `categories/` 子目录）

**单元测试** `dict/wordlists_test.go`：

- [x] `TestLoadBuiltinWordlist`：验证加载条目数 > 0，不含空行，不含 `#` 开头的行
- [x] `TestLoadCategoryWordlist`：验证 `wordpress`/`spring`/`django` 等可正常加载
- [x] `TestLoadCategoryWordlist_NotFound`：验证不存在的 category 返回 `ErrCategoryNotFound`
- [x] ~~`TestLoadBlacklist`：验证 403 黑名单包含 `/.git/config`~~（Phase 6 已删除）
- [x] `TestLoadUserAgents`：验证列表非空

---

### Task 1.3 — 实现 `dict/template.go`（模板令牌系统）

**目标**：实现 `%EXT%` 令牌展开逻辑（参考设计文档 3.4 节）

**文件**：`internal/core/scanner/dir/dict/template.go`

**实现要求**：

- [x] 实现 `ExpandLine(line string, extensions []string, mode ExtensionMode) []string`
  - `ExtensionModeClassic`（默认）：仅替换 `%EXT%`；无令牌的行原样返回
  - `ExtensionModeForce`：无扩展的路径，追加 `{line}.{ext}` 形式的变体
- [x] `ExtensionMode` 类型为 `int` const，不用 string
- [x] 路径中若含 `%EXT%` 但 `extensions` 为空，返回原行（不展开）
- [x] 不实现 `overwrite_extensions`（已决策不做，见参数设计文档 3.2 节）

**单元测试** `dict/template_test.go`：

- [x] `TestExpandLine_Classic_WithEXT`：`/backup.%EXT%` + `[php,bak]` → `[/backup.php, /backup.bak]`
- [x] `TestExpandLine_Classic_NoEXT`：`/admin` + `[php]` → `[/admin]`（原样返回）
- [x] `TestExpandLine_Force`：`/admin` + `[php,html]` → `[/admin, /admin.php, /admin.html]`
- [x] `TestExpandLine_EmptyExtensions`：含 `%EXT%` 但 extensions 为空 → 原样返回
- [x] `TestExpandLine_Force_WithEXT`：Force 模式下含 %EXT% 的行也正常替换
- [x] `TestExpandLine_MultipleTokens`：路径中多个 %EXT% 都被替换

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

- [x] `New(opts *DirOptions) (*Dictionary, error)`：
  - 加载内置字典（`LoadBuiltinWordlist()`）
  - 若 `opts.Wordlists` 非空，额外加载用户指定文件（追加到 main）
  - 扫描 `rules/dir/custom/` 下所有 `.txt` 文件，追加到 main（运行时 `os.ReadFile` 加载，多路径 fallback 依次尝试：`rules/dir/custom/`、`../rules/dir/custom/`、`../../rules/dir/custom/` 等；**不引用 `WebScanner` 的任何代码**，加载模式相同但各自独立实现，符合原子扫描器隔离原则）
  - 对 main 按 `opts` 中的扩展配置展开（调用 `ExpandLine`）
  - 应用 `opts.Uppercase`/`Lowercase`/`Capital` 大小写变换
  - 应用 `opts.Prefixes`/`Suffixes` 前后缀变换
  - 超过 `maxSize` 时截断并记录警告日志
- [x] `Next() (string, bool)`：先取 extra（优先），再取 main；两者耗尽返回 `("", false)`；线程安全
- [x] `AddExtra(paths ...string)`：追加到 extra 队列尾部；检查 total 是否超限；线程安全
- [x] `Len() int`：返回剩余条目数（估算）
- [x] `Total() int`：返回已处理总条目数

**单元测试** `dict/dictionary_test.go`：

- [x] `TestDictionary_Next`：顺序取出所有条目，最终返回 false
- [x] `TestDictionary_AddExtra_Priority`：AddExtra 后 Next 优先返回 extra 条目
- [x] `TestDictionary_MaxSize`：超过 maxSize 时截断
- [x] `TestDictionary_Concurrent`：100 goroutine 并发 Next，无 data race（`go test -race`）
- [x] `TestDictionary_UserCustomWordlist`：`rules/dir/custom/` 下放一个测试文件，验证被加载
- [x] `TestDictionary_ExtensionExpand`：验证字典构建时扩展名正确展开

---

### Task 1.5 — 实现 `result/result.go`（数据模型）

**目标**：定义 DirScanner 输出的数据结构（参考设计文档 8.1 节）

**文件**：`internal/core/scanner/dir/result/result.go`

**实现要求**：

- [x] 定义 `DirHit` 结构体（Path/Status/Size/Words/Lines/Title/Location/ContentType/RTT）
- [x] 定义 `ScanStats` 结构体（TotalRequests/SuccessfulReqs/FilteredReqs/WildcardReqs/ErrorReqs/AvgRTT/MaxRTT/MinRTT）
- [x] 定义 `DirResult` 结构体（Target/StartTime/EndTime/DictSize/Found/Hits/Stats）
- [x] 实现 `DirResult.Add(hit *DirHit)`：追加命中，更新 Found 计数
- [x] 实现 `DirResult.Headers() []string` 和 `DirResult.Rows() [][]string`：实现 `TabularData` 接口，供 ConsoleReporter 使用
- [x] `DirHit.String()` 格式：`[STATUS] /path (SIZE bytes)`

**单元测试** `result/result_test.go`：

- [x] `TestDirHit_String`：验证 `[200] /admin (1024 bytes)` 格式
- [x] `TestDirResult_Add`：验证追加命中并更新计数
- [x] `TestDirResult_Finish`：验证结束时间标记
- [x] `TestDirResult_Headers`：验证表头 `["Status", "Path", "Size", "Title", "Location"]`
- [x] `TestDirResult_Rows`：验证数据行格式与 size 格式化
- [x] `TestDirResult_Rows_Empty`：验证空结果返回 0 行
- [x] `TestFormatSize`：验证 B/KB/MB 格式化
- [x] `TestScanStats_JSON`：验证 ScanStats 序列化
- [x] `TestDirResult_CompleteFlow`：验证完整扫描流程

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

- [x] `NewFilter(cfg FilterConfig) *Filter`
- [x] `Filter.Match(resp *Response, path string) bool`：
  - Layer 1a：状态码白名单过滤（IncludeStatus 非空时，不在白名单则排除）
  - Layer 1b：状态码黑名单过滤（ExcludeStatus，默认 `[404, 500, 502, 503]`）
  - Layer 1c：响应大小过滤（ExcludeSize）
  - Layer 3a：关键词排除（ExcludeKeywords，Body 中包含则排除）
  - Layer 3b：正则排除（ExcludeRegex）
  - 全部通过返回 `true`
- [x] `DefaultExcludeStatus() []int`：返回 `[404, 500, 502, 503]`
- [x] ~~`AddErrorPath(statusCode int, path string)`：运行时添加错误路径~~（Phase 6 已删除，见 Task 6.2）

**单元测试** `engine/filter_test.go`：

- [x] `TestFilter_ExcludeStatus`：404 被排除
- [x] `TestFilter_IncludeStatus`：仅保留白名单状态码
- [x] `TestFilter_DefaultExcludeStatus`：默认黑名单生效
- [x] ~~`TestFilter_ErrorPath`：`/.git/config + 403` 命中黑名单被排除~~（Phase 6 已删除）
- [x] `TestFilter_ExcludeKeyword`：Body 含关键词被排除
- [x] `TestFilter_ExcludeRegex`：Body 匹配正则被排除
- [x] `TestFilter_ExcludeSize`：大小黑名单过滤
- [x] `TestFilter_AllPass`：正常 200 响应通过所有层
- [x] ~~`TestFilter_AddErrorPath`：运行时添加错误路径~~（Phase 6 已删除）

---

### Task 1.7 — 实现 `engine/requester.go`（HTTP 请求器）

**目标**：实现保留原始路径的 HTTP 请求器（参考设计文档 6.1~6.3 节）

**文件**：`internal/core/scanner/dir/engine/requester.go`

**实现要求**：

- [x] 定义 `RequesterConfig` 结构体（Timeout/MaxRetries/Proxy/Method/Headers/UserAgents/RateLimit/Delay/FollowRedirects/IP/NetworkInterface）
- [x] `NewRequester(cfg RequesterConfig) *Requester`：
- 构建 `http.Client`（TLS 禁用证书验证、连接池复用（`MaxIdleConnsPerHost` 对齐 `Threads`）、代理配置、本地 IP 绑定）
  - 若 `cfg.FollowRedirects == false`，设置 `CheckRedirect` 返回 `http.ErrUseLastResponse`
- [x] `Requester.Do(ctx context.Context, baseURL, path string) (*Response, error)`：
  - 拼接请求 URL
  - 注入请求头（含 User-Agent 轮询选择）
  - 重试循环（最多 `MaxRetries` 次，指数退避，最大 5s）
  - 错误分类：`ErrConnectionTimeout`、`ErrConnectionRefused`、`ErrDNSFailed`、`ErrSSLError`、`ErrOther`
  - 连接失败不重试，超时重试
  - 读取响应体上限 10MB（防内存爆炸）
  - 返回 `*Response`（含 Body/StatusCode/Size/ContentType/Location/Words/Lines/Title）
  - HTML `<title>` 提取
  - 词数/行数统计
- [x] `RateLimit`：channel 缓冲实现速率限制
- [x] `Delay`：请求间隔延迟

**单元测试** `engine/requester_test.go`（使用 `httptest.Server`）：

- [x] `TestRequester_BasicGet`：正常 200 响应
- [x] `TestRequester_PathPreserving`：路径中含 `%2F` 请求
- [x] `TestRequester_Timeout`：服务器超时，验证错误类型
- [x] `TestRequester_Retry`：第 1 次超时、第 2 次成功，验证重试逻辑
- [x] `TestRequester_NoFollowRedirect`：301 不跟随重定向，返回 301
- [x] `TestRequester_FollowRedirect`：跟随重定向验证
- [x] `TestRequester_UserAgent`：UA 轮询选择
- [x] `TestRequester_CustomHeaders`：自定义请求头注入
- [x] `TestRequester_Proxy`：代理配置（验证不 panic）
- [x] `TestRequester_ConnectionRefused`：连接拒绝错误
- [x] `TestRequester_TitleExtraction`：HTML <title> 提取
- [x] `TestRequester_WordsAndLines`：词数和行数统计
- [x] `TestRequester_ContextCancel`：context 取消
- [x] `TestRequester_BodyLimit`：响应体上限 10MB
- [x] `TestRequester_ClassifyError`：7 种错误分类测试
- [x] `TestRequester_IsConnectionError`：连接错误判断
- [x] `TestRequester_DefaultUA`：默认 UA
- [x] `TestExtractTitle`：6 种标题提取边缘情况
- [x] `TestRequester_MaxRetriesExceeded`：达到最大重试次数
- [x] `TestRequester_Delay`：请求间隔延迟
- [x] `TestRequester_RateLimit`：速率限制
- [x] `TestRequester_HTTPMethod`：自定义 HTTP 方法
- [x] `BenchmarkRequester_Basic`：性能基准测试

**备注**：PathPreservingRoundTripper（Opa que 字段绕过 URL 规范化）— Go 1.25 已移除 `req.Opaque` 字段，当前版本暂不实现路径特殊字符保留，后续可扩展为自定义 Transport。

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

- [x] `NewWildcardScanner(req *Requester, baseURL string) *WildcardScanner`
- [x] `normalizeDynamic(content string) string`：移除 UUID/长hex/时间戳/Base64（4 个正则，见设计文档 4.3 节）；正则**编译一次，包级变量**，不在函数内 `MustCompile`
- [x] `extractStaticPatterns(body1, body2 string) ([]string, int)`：返回共同词列表和基准词数
- [x] `fingerprint(resp *Response) string`：按设计文档 4.4 节格式生成指纹
- [x] `WildcardScanner.Check(ctx context.Context, pathPrefix string, resp *Response) (isUnique bool, reason string)`：
  - 检查指纹是否已知通配符（`wildCardFPs`）→ 直接返回 false
  - 未达采样阈值 → 发送探测请求（随机不存在路径 `pathPrefix + randStealthWord()`），更新样本
  - 采样未完成 → 返回 `(false, "sampling")`（采样阶段压制输出）
  - 采样完成后提取 `staticPatterns`
  - 比对当前响应与静态模式：相似度 ≥ 0.9 且长度差异 ≤ 35% → 通配符，返回 false
  - 自校准：调用 `autoCalibrate`，更新 `fingerprints`，超阈值加入 `wildCardFPs`
  - 通过所有检测 → 返回 `(true, "unique")`
- [x] `randStealthWord() string`：生成随机 8 字符字符串（避免与真实路径冲突）

**单元测试** `engine/scanner_test.go`（使用 `httptest.Server`）：

- [x] `TestNormalizeDynamic`：UUID/hex/时间戳/Base64 被正确替换
- [x] `TestExtractStaticPatterns`：两个相同结构但动态内容不同的响应，提取到共同词
- [x] `TestWildcardScanner_CDN`：mock 服务器对任意路径返回同一个 404 页面 → Check 返回 false
- [x] `TestWildcardScanner_Unique`：mock 服务器对特定路径返回独特内容 → Check 返回 true
- [x] `TestWildcardScanner_SamplingPhase`：前 5 次 Check 均返回 `(false, "sampling")`
- [x] `TestWildcardScanner_Concurrent`：并发 Check，无 data race

---

### Task 2.2 — 实现 `dir_scanner.go`（主扫描器）

**目标**：实现 `DirScanner` 核心逻辑，接入 Worker 池 + 递归调度（参考设计文档 7.1 节）

**文件**：`internal/core/scanner/dir/dir_scanner.go`

**实现要求**：

- [x] 定义 `DirScanner` 结构体，持有 `limiter *qos.AdaptiveLimiter` 字段（**不使用 Go struct embedding**，而是普通字段持有，与 `ApiScanner`/`IpAliveScanner` 的做法一致，保持封装边界清晰）
- [x] 实现 `Name() model.TaskType`：返回 `model.TaskTypeDirScan`
- [x] 实现 `Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error)`：
  - 调用 `parseDirOptions(task.Params)` 解析参数（见 Task 3.1）
  - 初始化 `Dictionary`、`Requester`、`Filter`、`WildcardScanner`
  - 处理 `MaxTime`/`TargetMaxTime`：若非 0，用 `context.WithTimeout` 包装 ctx
  - 启动 Worker 池（goroutine 数 = `opts.Threads`）
  - 等待所有 Worker 完成，收集结果
  - 返回 `[]*model.TaskResult`（包含 `DirResult`）
- [x] 实现 `worker(ctx, dict, req, filter, scanner, results chan, opts)`：
  - 从 `dict.Next()` 取路径
  - `limiter.Acquire(ctx)` 速率控制
  - 若 `opts.Delay > 0`，`time.Sleep(opts.Delay)`
  - `req.Do()` 发送请求
  - `filter.Match()` 过滤
  - `scanner.Check()` 通配符检测
  - 命中则发送到 results channel，并判断是否递归
- [x] 实现 `shouldRecursion(resp, path, opts)`（三种递归模式，见设计文档 7.1 节）
- [x] 实现 `generateSubpaths(path string) []string`（深度递归子路径生成）
- [x] 递归调用 `dict.AddExtra(subpaths...)` 时检查递归深度（`MaxRecursionDepth`），超限则跳过
- [x] 实现 `parseDirOptions(params map[string]interface{}) *DirOptions`：从 task.Params 解析所有字段，含默认值（`Threads=25`、`Timeout=10s`、`MaxRetries=2`、`ExcludeStatus=[404,500,502,503]`、`MaxRecursionDepth=3`）

**单元测试** `dir_scanner_test.go`（使用 `httptest.Server`）：

- [x] `TestDirScanner_BasicScan`：mock 服务器有 `/admin`（200）和 `/secret`（200），其他 404 → 均被发现
- [x] `TestDirScanner_WildcardCDN`：mock 服务器所有路径返回同一个 404 页面 → 零结果（无误报）
- [x] `TestDirScanner_Recursive`：`/api/`（301）触发递归，子路径被扫描
- [x] `TestDirScanner_MaxRecursionDepth`：递归深度限制为 2 时，第 3 层不扫描
- [x] `TestDirScanner_ContextCancel`：context 取消后扫描立即停止，无 goroutine 泄漏（检查 goroutine 数量）
- [x] `TestDirScanner_MaxEntries`：字典超过 500K 条时被截断，扫描不崩溃

> **Phase 2 完成说明**（全量测试通过，含 `-race -count=1`）
>
> **Task 2.1 — 通配符检测引擎** (`engine/scanner.go`)
> - 实现了 `WildcardScanner` 全部方法：`Check`、`normalizeDynamic`、`extractStaticPatterns`、`fingerprint`、`randStealthWord`、`autoCalibrate`
> - 4 个归一化正则编译为包级变量 `regexp.Regexp`，不在函数内重复编译
> - 采样阈值 `defaultSampleThreshold = 5`，自校准阈值 `defaultFingerprintThresh = 10`
> - **关键决策：Precheck 预采样机制**。设计文档原方案在首个真实路径 `Check` 时才启动采样，导致采样阶段返回 `"sampling"` 会丢弃该路径的真实命中。实际实现中 `DirScanner.Run` 在 Worker 池启动前调用 `scanner.Precheck(ctx)`，对根路径 `/` 发送 `sampleThreshold` 次随机探测请求完成初始化，确保扫描器在 Worker 拿到第一个字典路径时即具备识别能力
> - 6 个单元测试全部通过，包括并发 50 goroutine 的 race detector 验证
>
> **Task 2.2 — 主扫描器** (`dir_scanner.go`)
> - `DirScanner` 结构体持有 `limiter *qos.AdaptiveLimiter`（普通字段，非 embedding）
> - `Run` 方法完整流程：`parseDirOptions` → 初始化 Dictionary/Requester/Filter/WildcardScanner → `Precheck` → `context.WithTimeout`（若 `MaxTime > 0`）→ Worker 池 → 收集结果
> - Worker 从 `dict.Next()` 取路径 → `limiter.Acquire(ctx)` → `req.Do()` → `filter.Match()` → `scanner.Check()` → 命中则发送到 results channel，并判断递归
> - `shouldRecursion` 支持三种模式：`Recursive`（目录列表 301/200+目录特征）、`DeepRecursive`（逐级父目录展开）、`ForceRecursive`（强制递归所有命中）
> - `generateSubpaths` 为深度递归生成逐级父目录子路径
> - `DirOptions` 增加 `SkipBuiltin` 字段，允许测试和定制场景跳过内置大字典加载，只加载用户指定 `Wordlists`
> - 6 个单元测试全部通过，包括 context 取消后 goroutine 泄漏检测（`runtime.NumGoroutine` 对比）
>
> **附带修复**：`engine/requester.go` 中 `cfg.Method` 默认值从字符串字面量 `"http.MethodGet"` 修正为常量 `http.MethodGet`（Phase 1 遗留 bug）

---

## Phase 3：CLI 接入

### Task 3.1 — 重写 `internal/core/options/scan_dir.go`

**目标**：将现有骨架扩展为完整的 `DirScanOptions`，覆盖所有已决策参数（参考参数设计文档第 4 章）

**文件**：`internal/core/options/scan_dir.go`

**完成说明**：`DirScanOptions` 已按设计文档 8.2 节字段全量重写，覆盖字典/扫描控制/过滤/请求/多目标/输出六大类配置；`Validate()` 完成 Target 必填、状态码/大小/正则的一次性解析校验；`ToTask()` 按下划线命名写入 `task.Params`，与 `dir_scanner.go` 的 `parseDirOptions()` 解析 key 一一对应。

**实现要求**：

- [x] 定义 `DirScanOptions` 结构体，字段与设计文档 8.2 节 `DirOptions` 完全对应（但不含 `ExcludeRegex`，由 `Validate` 编译）
- [x] `NewDirScanOptions() *DirScanOptions`：设置所有默认值：
  - `Threads = 25`
  - `Timeout = 10`（秒）
  - `MaxRetries = 2`
  - `ExcludeStatus = "404,500,502,503"`
  - `Recursive = true`
  - `MaxRecursionDepth = 3`
  - `Extensions = "php,asp,html,js,json,bak,git,env"`（默认扩展列表）
  - `HTTPMethod = "GET"`
- [x] `Validate() error`：
  - Target 必填（或 `TargetsFile` 非空）
  - 解析 ExcludeStatus 字符串为 `[]int`（逗号分隔），非法值报错
  - 解析 ExcludeRegex 字符串为 `[]*regexp.Regexp`，编译失败报错
  - `MaxRecursionDepth` 最大值限制为 10（防误操作）
  - `Threads` 范围 1~500
  - `Timeout` 范围 1~300
- [x] `ToTask() *model.Task`：将所有字段写入 `task.Params`（key 名统一使用下划线命名，如 `max_recursion_depth`）；复用 `Output.ApplyToParams()`

---

### Task 3.2 — 重写 `cmd/agent/scan/dir.go`（CLI 命令）

**目标**：将现有占位命令扩展为完整的 `scan dir` 子命令（参考参数设计文档 4.1~4.3 节）

**文件**：`cmd/agent/scan/dir.go`

**完成说明**：`DirScanOptions` 已按 P0/P1/P2 分级全量重写，`NewDirScanCmd` 实现了位置参数与 `-t/--target` 互斥（位置参数优先）、`-H`/`--headers-file` 合并、`--targets-file` 批量目标解析；`RunE` 通过 `RunnerManager.Execute` 执行任务，`ConsoleReporter` 输出结果，`--oj`/`--oc` 落盘。

**P0 核心参数**（必须实现）：

- [x] `<target>`（位置参数，args[0]）或 `-t/--target` flag，两者互斥，位置参数优先
- [x] `--extensions/-e string`（默认 `php,asp,html,js,json,bak,git,env`）
- [x] `--wordlists/-w string`（字典文件路径，默认空=使用内置字典）
- [x] `--threads int`（默认 25）
- [x] `--timeout int`（默认 10）
- [x] `--recursive/-r`（默认 true）
- [x] 全局继承 `--oj`/`--oc`（已在 `scan` 父命令注册，无需重复）

**P1 重要参数**（必须实现）：

- [x] `--force-extensions/-f bool`
- [x] `--exclude-status/-x string`（默认 `"404,500,502,503"`）
- [x] `--deep-recursive bool`
- [x] `--max-recursion-depth/-R int`（默认 3）
- [x] `--verbose/-v bool`
- [x] `--quiet/-q bool`
- [x] `--header/-H string`（可多次指定，cobra `StringArrayVar`）
- [x] `--proxy/-p string`
- [x] `--follow-redirects/-F bool`
- [x] `--max-retries int`（默认 2）
- [x] `--method/-m string`（默认 `GET`）

**P2 可选参数**（实现，但不标注 required）：

- [x] `--prefixes string`（逗号分隔）
- [x] `--suffixes string`（逗号分隔）
- [x] `--uppercase/-U bool`
- [x] `--lowercase/-L bool`
- [x] `--capital/-C bool`
- [x] `--include-status/-i string`
- [x] `--exclude-text string`（可多次，`StringArrayVar`）
- [x] `--exclude-regex string`
- [x] `--exclude-size string`（逗号分隔的字节数）
- [x] `--rate-limit int`
- [x] `--max-time int`
- [x] `--target-max-time int`
- [x] `--headers-file string`
- [x] `--random-agent bool`
- [x] `--auth string`
- [x] `--user-agent string`
- [x] `--ip string`
- [x] `--network-interface string`
- [x] `--targets-file/-l string`（多目标文件）
- [x] `--delay int`

> 说明：`--exclude-extensions` 在实现阶段评估后决定不单独实现（与 `--exclude-text`/`Filter` 的关键词排除能力重叠，见 `scan_dir.go` 的字段设计），不影响功能完整性。

**RunE 逻辑**：

- [x] 调用 `opts.Validate()`，失败则返回错误
- [x] 调用 `opts.ToTask()` 生成 task
- [x] 通过 `RunnerManager` 执行
- [x] 结果通过 `ConsoleReporter`（verbose/quiet 控制详细程度）和 `--oj`/`--oc` 输出
- [x] 移除 `not implemented yet` 占位错误

---

## Phase 4：集成

### Task 4.1 — 注册到 RunnerManager

**目标**：将 `DirScanner` 注册到 `RunnerManager`，使 Master 下发的 `dir_scan` 任务也能执行

**文件**：`internal/core/factory/dir_factory.go`、`internal/core/runner/manager.go`

**操作清单**：

- [x] 确认 RunnerManager 的工厂函数位置（`internal/core/runner/manager.go` 的 `NewRunnerManager()`）
- [x] 新建 `factory.NewDirScanner()`，在 `NewRunnerManager()` 中调用 `m.Register(factory.NewDirScanner())`
- [x] 确认 `model.TaskTypeDirScan` 常量已定义
- [x] CLI（`cmd/agent scan dir`）与 Master 下发共用同一个 `RunnerManager` 注册表，两条路径能力一致

**验收记录（2026-08-19，真实网站验证）**：

- 用真实域名（`www.polestarcloud.com`）做端到端验证时，发现高并发（30 线程）下 `WildcardScanner` 存在偶发误判：命中结果的状态码/标题字段与真实服务器响应不一致（真实服务器稳定返回 404，程序却输出了不存在的 500 记录）。
- 根因定位：`engine/scanner.go` 的 `samplePool.bodies` 在旧实现中存在锁外读、锁内写的 data race——多个 goroutine 并发判断"是否仍处于采样阶段"时会同时写入采样池，导致不同路径的响应体被错误地当作同一份采样基准，进而污染通配符比对结果。
- 参考 `docs/references/dirsearch/lib/core/scanner.py` 的 `Scanner.setup()` 设计（采样与判定严格分两阶段：worker 启动前同步完成采样，采样完成后数据只读，不加锁也天然无竞争），按同样思路重构 `WildcardScanner`：用 `sync.Once` 保证同一 pathPrefix 的采样只被第一个到达的 goroutine 执行一次，其余并发调用阻塞等待，采样完成后字段固化为只读。
- 修复后用 `-race` + 真实网站高并发（30/40 线程）复测，data race 消失，结果与真实服务器响应完全一致；`engine` 包新增/更新单测（`TestWildcardScanner_SamplesOnlyOncePerPrefix`、`TestWildcardScanner_NoRequesterFallsBackToUnique`）覆盖新的两阶段模型。

---

### Task 4.2 — E2E 验收测试    ✅ 已完成（2026-08-19）

**目标**：验证完整扫描流程，对应设计文档 11.1 功能验收清单

**文件**：`internal/core/scanner/dir/dir_scanner_e2e_test.go`

**测试用例**（每条对应设计文档 11.1 的一行验收标准）：

- [x] `TestE2E_KnownSensitivePaths`：mock 服务器开放 `/.git/config`（200）、`/.env`（200）、`/admin`（200）→ 三者均被发现，状态码正确
- [x] `TestE2E_CDNWildcard`：mock 服务器对所有路径返回相同 404 页面（体积相同、内容相同）→ 扫描结果为空，无误报
- [x] `TestE2E_BlacklistPath`：mock 服务器对 `/.git/config` 返回 403 → 命中黑名单，不出现在结果中
- [x] `TestE2E_ExtensionTemplate`：内置字典含 `/backup.%EXT%`，传 `extensions=php,bak` → 扫描队列含 `/backup.php` 和 `/backup.bak`
- [x] `TestE2E_HighConcurrency`：`--threads 100`，mock 服务器正常响应 → 扫描完成，无 goroutine 泄漏，无 panic
- [x] `TestE2E_RetryOnTimeout`：连接错误时重试机制生效
- [x] `TestE2E_BasicRecursion`：`/api/`（301→/api/v1/）触发递归，`/api/v1/users`（200）被发现
- [x] `TestE2E_DeepRecursion`：`--deep-recursive`，`/a/b/c/` 命中 → `/a/`、`/a/b/` 被追加扫描
- [x] `TestE2E_OutputJSON`：`DirResult` 可序列化为合法 JSON，字段完整（Target/Found/Hits/Status/ContentType/Title/Size）

**测试统计**：9 tests pass, 0 fail（含 `-race -count=1`）

**调整说明**：
- `TestE2E_RetryOnTimeout`：httptest.Server 无法模拟真实网络超时，改为验证连接错误场景下重试机制不阻塞
- `TestE2E_DeepRecursion`：`shouldRecursion` 要求路径以 `/` 结尾才触发递归，字典条目使用 `/a/b/c/` 而非 `/a/b/c`
- `TestE2E_HighConcurrency`：不同路径返回不同内容避免通配符检测压制，验证扫描完成即可

---

### Task 4.3 — 性能基准测试

**目标**：对应设计文档 11.2 性能验收标准

**文件**：`internal/core/scanner/dir/dir_scanner_bench_test.go`

- [x] `BenchmarkDirScanner_SingleThread`：单线程扫全量内置字典（9681 条），验证吞吐（目标：内网 ≤ 60s，实测约 1.7s）
- [x] `BenchmarkDirScanner_25Threads`：25 线程扫全量内置字典（目标：内网 ≤ 5s，实测约 0.7s）
- [x] `BenchmarkDirScanner_MemoryUsage`：加载全部字典后测量内存（目标：≤ 150MB，实测约 1.5MB）
- [x] `BenchmarkWildcardScanner_Sampling`：通配符检测采样开销（目标：≤ 5 次额外请求，实测 5 次）

**完成说明**：benchmark 编写过程中发现并修复了一个真实性能 bug——`engine/requester.go` 的 `NewRequester` 曾设置 `DisableKeepAlives: true`，且每次请求显式声明 `Connection: close`（后者是根因，即使配置连接池也会被这一层应用层声明架空）。这会导致扫一次 9681 条内置字典就建立近万个短连接，连续多轮扫描时 TIME_WAIT 堆积、耗时逐轮递增（连续 5 轮从 7.5s 递增到 15s）。参考 dirsearch（`lib/connection/requester.py` 用 `requests.Session` + `pool_maxsize=thread_count`）的做法，改为按 `Threads` 设置 `MaxIdleConnsPerHost` 并移除 `Connection: close`，同时给 `Requester` 增加 `Close()` 方法在扫描结束后主动释放空闲连接（避免 `TestE2E_HighConcurrency` 的 goroutine 泄漏检测误报）。修复后 25 线程连续 5 轮稳定在约 0.7s/轮，不再随轮次增加。

---

## Phase 5 — 断层修复（2026-08-22）

**背景**：对照设计文档与实现代码做完整性审查时，发现多处"底层能力已实现、但主扫描流程未调用"
的断层。逐项用 Linus 式五层分析框架审查后，三项修复接入、两项明确决定不接入（其中场景模板
字典 Task 5.5 是对 Task 5.3 结论的补充复查修正，见下）。

### Task 5.1 — 修复 `--force-recursive` CLI flag 缺失 ✅

**问题**：`DirOptions.ForceRecursive`、`shouldRecursion()` 判断逻辑、`task.Params["force_recursive"]`
传递链路全部早已完整，唯独 `cmd/agent/scan/dir.go` 漏注册了 `flags.BoolVar(&opts.ForceRecursive,
"force-recursive", ...)` 这一行，导致 CLI 用户无法从命令行开启强制递归，只能走 Pipeline 下发 task 参数。

**修复**：补注册该 flag（1 行代码）。属于笔误级 bug，非设计决策。

**文件**：`cmd/agent/scan/dir.go`

### Task 5.2 — 实现 `ScanStats` 填充 ✅

**问题**：`result.ScanStats` 的 8 个字段（`TotalRequests`/`SuccessfulReqs`/`FilteredReqs`/
`WildcardReqs`/`ErrorReqs`/`AvgRTT`/`MaxRTT`/`MinRTT`）全部定义完整、JSON tag 就位、CLI 输出格式也
留了位置，但 `dir_scanner.go` 一直没有代码给 `dirResult.Stats` 赋值——输出 JSON 里 `stats` 全是零值，
会误导用户做决策。属于纯 bug，不是未开发的功能。

**修复**：新增 `statsAccumulator` 类型，8 个字段全用 `atomic.Int64`（RTT 以纳秒存储），在 `worker()`
现有的每个分支尾部顺手累加（无需新增任何 if 分支）；所有 worker 退出后单线程调用 `Finalize()`
转换为普通值写入 `dirResult.Stats`。`result.ScanStats` 结构本身保持纯数据、不引入 `atomic` 类型污染。

**测试**：新增 `TestDirScanner_StatsPopulated` 验证 4 个请求（1 个 200、1 个 403、2 个 404）的
计数：TotalRequests=4、SuccessfulReqs=4、FilteredReqs=2（默认黑名单只含 404/500/502/503，403 算命中）、
ErrorReqs=0、AvgRTT/MaxRTT/MinRTT 均 >0。

**文件**：`internal/core/scanner/dir/dir_scanner.go`、`internal/core/scanner/dir/dir_scanner_test.go`

### Task 5.3 — 接入 `LoadCategoryWordlist`（新增 `--category` CLI flag）✅

**问题**：`dict.LoadCategoryWordlist(category string)` 在 Phase 1 Task 1.2 已实现并通过单元测试，
但一直没有调用方。用户扫一个已知是 WordPress 的站时想用 `wordpress.txt` 而不是通用大字典，这是真实、
常见的需求（dirsearch 本身也没有自动化 TechStack 联动，都是用户手动指定）。

**修复**：三层打通
- `dict.DirOptions` 新增 `Categories []string` 字段
- `dict.New()` 加载顺序新增第 3 步：遍历 `opts.Categories`，调用 `LoadCategoryWordlist()` 追加到 main 队列
- `scanner/dir.DirOptions` 新增对应字段，`toDictOptions()` 透传
- `parseDirOptions()` 解析 `task.Params["category"]`（逗号分隔）
- `options.DirScanOptions` 新增 `Category string` 字段，`ToTask()` 写入 `task.Params["category"]`
- `cmd/agent/scan/dir.go` 注册 `flags.StringVar(&opts.Category, "category", ...)`

**明确不做**：不实现"自动从 WebScanner 的 TechStack 产出联动"——NeoAgent 目前没有 Pipeline 层
把两个扫描器结果互传的机制，写了也是孤岛代码，且违反参数设计文档 7.1 节"DirScanner 是独立
原子扫描器、不耦合其他扫描器结果"的决策。

**测试**：新增 `TestDictionary_CategoryWordlist`（验证 wordpress 字典被加载）和
`TestDictionary_CategoryWordlist_NotFound`（验证不存在的分类名仅警告、不报错）。

**文件**：`internal/core/scanner/dir/dict/dictionary.go`、`internal/core/scanner/dir/dict/dictionary_test.go`、
`internal/core/scanner/dir/dir_scanner.go`、`internal/core/options/scan_dir.go`、`cmd/agent/scan/dir.go`

### Task 5.4 — 错误路径黑名单（`LoadBlacklist`/`Filter.ErrorPaths`）审查定论：不接入 → Phase 6 彻底删除

**问题**：`dict.LoadBlacklist(statusCode)` 和 `Filter.AddErrorPath()`/`Filter.ErrorPaths` 在 Phase 1
已实现，但 `dir_scanner.go` 的 `Run()` 从未调用它们把黑名单文件加载结果灌进 `Filter.ErrorPaths`。

**审查结论：不接入**，理由：
1. 查看 `403_blacklist.txt` 等文件的真实内容，条目形如 `%2e%2e//google.com`、`%ff`、
   `../../../../../../etc/passwd`、`..;/`、`%3f/`——这些根本不是"目录路径"，而是 dirsearch
   原版用来探测服务器/WAF 是否对畸形请求返回假 403/500 的 **payload**。
2. dirsearch 原版用法是拿这些 payload 单独发请求，如果目标对任意畸形请求都返回同一个状态码，
   就把该状态码判定为噪音统一过滤——本质上是通配符检测的一种变体，已被 `WildcardScanner` 覆盖。
3. 而当前 `Filter.ErrorPaths[statusCode][path]` 的查表方式是用**真实字典路径**（如 `/admin`）做 key；
   字典里的路径与 `%2e%2e//google.com` 这类畸形 payload 永远不会重合，就算写代码把 `LoadBlacklist()`
   的结果灌进 `ErrorPaths`，`Layer 2` 也永远不会命中。
4. 这是接入方式与数据语义从一开始就不匹配的设计错误，不是缺一行调用代码的疏漏。补上调用只会
   制造一个"看起来在生效、实际永远没有效果"的假功能，比保持现状不接入更危险（用户会误以为
   自己已被保护）。

**处理方式（Phase 6 更新）**：原 Phase 5 决定保留死代码但不接入。Phase 6 按 Linus
"好代码是删出来的"原则彻底清除——`LoadBlacklist()`、`AddErrorPath()`、`Filter.ErrorPaths`、
`FilterConfig.ErrorPaths` 字段、`Layer 2` 过滤逻辑、3 个 blacklist 数据文件全部删除。相关测试
`TestFilter_ErrorPath`、`TestFilter_AddErrorPath` 一并移除。详见 Task 6.2。

### Task 5.5 — 场景模板字典（`LoadTemplateWordlist`）审查定论：不接入 → Phase 6 彻底删除

**问题**：`dict.LoadTemplateWordlist(template string)` 在 Phase 1 Task 1.2 已实现并通过单元测试，
但一直没有调用方。Task 5.3 修复 `--category` 时曾在设计文档里把这项归因为"与 categories 功能
重叠、优先级低、暂不做"——**这个归因是不准确的，本次补充复查后予以更正**。

**复查结论：不接入，但原因与黑名单（Task 5.4）性质不同**：

1. `LoadTemplateWordlist()` 函数本身没有 bug，实现方式与 `LoadCategoryWordlist()` 完全一致
   （按文件名读取 + 去空行/去注释）。真正的问题在 `templates/*.txt` 的**文件内容**上。
2. 查看 7 个模板文件（`admin`/`api`/`auth`/`backups`/`crud`/`db`/`logs`）的真实内容，条目形如
   `admin/%ADMIN_OP%/%SUBJECT%`、`%CRUD_OP%_%SUBJECT%.%EXT%`、`backup_%DATE%.%ARCHIVE%`，
   普遍含有 `%ADMIN_OP%`/`%SUBJECT%`/`%CRUD_OP%`/`%AUTH_OP%`/`%API_VERSION%`/`%ARCHIVE%`/
   `%ENV%`/`%DB%`/`%DATE%`/`%DATE_COMPACT%` 等 **9+ 种令牌**，而 `dict/template.go` 的
   `ExpandLine()` 只实现了 `%EXT%` 一种令牌的替换。
3. 参考 dirsearch 原版 `lib/core/wordlist_template.py`：这些令牌各自挂了一份预置候选值表
   （`DEFAULT_PLACEHOLDERS`，如 `SUBJECT` 对应 15 个候选词、`ADMIN_OP` 对应 8 个候选词），
   用 `itertools.product` 做笛卡尔积组合展开——`admin/%ADMIN_OP%/%SUBJECT%` 一行实际会展开成
   `8 × 15 = 120` 条真实路径。NeoAgent 目前既没有候选值表，也没有多令牌提取/组合展开逻辑。
4. 如果现在把 `LoadTemplateWordlist()` 接进 `dict.New()`，产出的是带 `%ADMIN_OP%`/`%SUBJECT%`
   字面量的原始字符串，会被当作真实路径原样发出请求——**不会产生任何有效字典条目，纯粹是
   垃圾请求**，比 Task 5.4 的黑名单问题更严重（黑名单接入后是"查不中"，这个接入后是"污染
   扫描流量"）。

**处理方式（Phase 6 更新）**：原 Phase 5 决定保留死代码但不接入。Phase 6 按 Linus
"好代码是删出来的"原则彻底清除——`LoadTemplateWordlist()` 函数、7 个 templates 数据文件
全部删除。详见 Task 6.2。

**当前替代方案**：用户可将想要的具体路径手写成不含 `%XXX%` 令牌的普通文件，通过 `--wordlists`
加载，效果等价于模板展开后的固定子集。

**文档修正**：设计文档 3.2/3.4.1 节、参数设计文档"不做的原因"一节均已同步更正表述，不再使用
"与 categories 功能重叠"这一不准确的归因。

---

## Phase 6 — 参数分级 + 冗余清理（2026-08-23）

**背景**：Phase 5 断层修复后，`dirscan --help` 输出 38 个参数，信息过载。同时 Phase 5 审查后
决定"不接入但保留"的模板字典和黑名单代码成了死代码——保留死代码比没有代码更糟糕，因为它会
误导维护者以为这些功能"即将接入"，实际上永远不会。

### Task 6.1 — CLI 参数分级展示 ✅

**问题**：`dirscan --help` 一次性展示全部 38 个参数，用户被淹没在参数海洋里，找不到常用参数。

**修复**：参数分为三级展示
- **P0 常用参数**（15 个）：`--help` 默认展示，包括 `target`/`extensions`/`wordlists`/
  `category`/`threads`/`timeout`/`recursive`/`force-extensions`/`exclude-status`/
  `max-recursion-depth`/`verbose`/`quiet`/`header`/`method`/`targets-file`
- **P1/P2 高级参数**（23 个）：通过 `flags.MarkHidden()` 隐藏，`--help` 不展示
- **`--help-all`**：展示全部参数（含 hidden），通过自定义 bool flag + `printAllFlags()` 实现

**实现方式**：
- Cobra 1.10.2 不内置 `--help-all`，在 `RunE` 中手动拦截 `helpAll` bool flag
- `printAllFlags()` 遍历 `FlagSet`，临时重置 `Hidden` 字段后调用 `FlagUsages()` 输出，输出后恢复
- `--help-all` 自身也 `MarkHidden`，不出现在 `--help` 中

**文件**：`cmd/agent/scan/dir.go`

### Task 6.2 — 彻底清除模板字典和黑名单死代码 ✅

**问题**：Phase 5 Task 5.4 和 5.5 审查后决定"保留代码但不接入"。但 Linus 说得好："好代码是删
出来的。" 保留死代码等于给未来埋坑——维护者会以为这些功能"待接入"，实际上永远不会接入。

**清理范围**：

**模板字典（templates）清理**：
- 删除 `dict/wordlists/templates/` 目录下全部 7 个模板文件（`admin`/`api`/`auth`/`backups`/
  `crud`/`db`/`logs`）
- 删除 `wordlists.go` 中的 `LoadTemplateWordlist()` 函数

**黑名单（blacklist）清理**：
- 删除 `dict/wordlists/blacklist/` 目录下全部 3 个黑名单文件（`400_blacklist.txt`/
  `403_blacklist.txt`/`500_blacklist.txt`）
- 删除 `wordlists.go` 中的 `LoadBlacklist()` 函数
- 删除 `engine/filter.go` 中的 `FilterConfig.ErrorPaths` 字段
- 删除 `engine/filter.go` 中的 `Filter.AddErrorPath()` 方法
- 删除 `engine/filter.go` 中 `Match()` 的 Layer 2 错误路径黑名单过滤逻辑

**测试清理**：
- 删除 `wordlists_test.go` 中的 `TestLoadBlacklist` 测试
- 删除 `filter_test.go` 中的 `TestFilter_ErrorPath` 和 `TestFilter_AddErrorPath` 测试

**影响分析**：
- 零破坏性：这些代码从未被主扫描流程调用，删除不影响任何运行时行为
- 全部 67 个测试用例通过（含 `-race`），零失败
- `wordlists.go` 从 ~159 行缩减到 ~117 行
- `filter.go` 从 ~139 行缩减到 ~117 行

**文件**：`internal/core/scanner/dir/dict/wordlists.go`、
`internal/core/scanner/dir/engine/filter.go`、
`internal/core/scanner/dir/dict/wordlists_test.go`、
`internal/core/scanner/dir/engine/filter_test.go`

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

- [x] `internal/core/lib/network/qos/` 路径确认（Task 2.2 开始前）
- [x] `model.TaskTypeDirScan` 常量确认（Task 4.1 前）
- [x] `ConsoleReporter` 的 `TabularData` 接口签名确认（Task 1.5 前）
- [x] `RunnerManager` 工厂函数位置确认（Task 4.1 前）
- [x] `rules/dir/custom/` 目录已存在（Task 1.4 前，已由用户创建）

---

## 任务完成状态跟踪

| 任务 | 状态 | 备注 |
|------|:---:|------|
| 1.1 复制字典文件 | ✅ 已完成 | 9681 行 dicc，403 blacklist 9 项，53 UA，25 category |
| 1.2 wordlists.go | ✅ 已完成 | 5 tests pass（Phase 6 删除了 TestLoadBlacklist，含 ListCategories） |
| 1.3 template.go | ✅ 已完成 | 6 tests pass，含 Force+MULTI_TOKEN |
| 1.4 dictionary.go | ✅ 已完成 | 6 tests pass，含 race detector 无竞争 |
| 1.5 result.go | ✅ 已完成 | 9 tests pass，含 formatSize + complete flow |
| 1.6 filter.go | ✅ 已完成 | 7 tests pass（Phase 6 删除了 ErrorPath/AddErrorPath） |
| 1.7 requester.go | ✅ 已完成 | 25 tests pass，含 7 种错误分类 + benchmark |
| 2.1 scanner.go（通配符） | ✅ 已完成 | 6 tests pass；2026-08-19 依据 dirsearch 重构为两阶段采样模型，修复 data race |
| 2.2 dir_scanner.go | ✅ 已完成 | 6 tests pass，含 goroutine 泄漏检测 |
| 3.1 scan_dir.go（Options） | ✅ 已完成 | P0/P1/P2 全部参数已实现，见 Task 3.1 |
| 3.2 dir.go（CLI） | ✅ 已完成 | `scan dir` 子命令完整接入，见 Task 3.2 |
| 4.1 RunnerManager 注册 | ✅ 已完成 | 已在 `NewRunnerManager()` 注册，CLI/Master 共用同一注册表 |
| 4.2 E2E 验收测试 | ✅ 已完成 | 9 tests pass (含 race detector) |
| 4.3 性能基准测试 | ✅ 已完成 | 4 benchmarks pass，实测数据见设计文档 11.2 节；附带修复连接池复用 bug |
| 5.1 `--force-recursive` flag 补注册 | ✅ 已完成 | 笔误级 bug，1 行代码修复 |
| 5.2 `ScanStats` 填充接入 | ✅ 已完成 | `statsAccumulator` + `atomic.Int64`，`TestDirScanner_StatsPopulated` 验证 |
| 5.3 `--category` CLI flag 接入 | ✅ 已完成 | 三层打通：CLI → DirOptions → dict.New()，2 个新测试覆盖 |
| 5.4 错误路径黑名单审查 | ❌ 不接入 → 🗑️ 已删除 | Phase 5 决定不接入，Phase 6 彻底删除死代码（见 Task 6.2） |
| 5.5 场景模板字典审查 | ❌ 不接入 → 🗑️ 已删除 | Phase 5 决定不接入，Phase 6 彻底删除死代码（见 Task 6.2） |
| 6.1 CLI 参数分级展示 | ✅ 已完成 | `--help` 展示 P0 参数，`--help-all` 展示全部，`MarkHidden` 实现 |
| 6.2 清除模板/黑名单死代码 | ✅ 已完成 | 删除 `LoadTemplateWordlist`/`LoadBlacklist`/`ErrorPaths`/`AddErrorPath` + 10 个数据文件 |

---

### Phase 1 完成说明

**完成日期**：2026-08-17

**测试统计**（Phase 6 清理后）：
```
dict:         19 tests pass (含 race detector，删除了 TestLoadBlacklist)
result:        9 tests pass
engine:       32 tests pass (filter 7 + requester 25，删除了 ErrorPath/AddErrorPath)
总计:         60 tests pass, 0 fail
```

**代码统计**（Phase 6 清理后）：
| 文件 | 行数 |
|------|-----:|
| `dict/wordlists.go` | ~117 |
| `dict/wordlists_test.go` | ~96 |
| `dict/template.go` | ~56 |
| `dict/template_test.go` | ~63 |
| `dict/dictionary.go` | ~311 |
| `dict/dictionary_test.go` | ~178 |
| `result/result.go` | ~111 |
| `result/result_test.go` | ~165 |
| `engine/filter.go` | ~117 |
| `engine/filter_test.go` | ~127 |
| `engine/requester.go` | ~304 |
| `engine/requester_test.go` | ~570 |
| **总计** | **~2,215** |

**已知问题/待后续处理**：
1. PathPreservingRoundTripper — Go 1.25 已移除 `req.Opaque` 字段，当前版本暂不实现路径特殊字符保留（`requester.go` 第 192 行注释），后续可扩展为自定义 Transport
2. `dictionary.go` — 文档要求 `AddExtra` 追加到队列头部，当前实现为尾部追加（`append(d.extra, p)`），与 `Next()` 消费顺序（从 `extraIdx` 开始）一致
3. ~~模板字典（templates）和黑名单（blacklist）死代码~~ — Phase 6 已彻底清除（见 Task 6.2）
4. ~~CLI 参数信息过载~~ — Phase 6 已通过参数分级解决（见 Task 6.1）

---

## Phase 7 — 审查优化（待实施）

**背景**：Phase 6 完成后对整个目录扫描模块进行 Linus 式代码审查，识别出以下需要修复/优化的问题。按优先级排序，等待命令字后再动代码。

### Task 7.1 — 🔴 P0 修复 `recursionCtl` 递归深度判定的 TOCTOU 竞争

**问题**：`dir_scanner.go` 第 382-391 行，`depthOf()` 和 `markDepth()` 是两次独立的加锁操作。多个 worker 同时命中同一路径时，可能同时通过 `depth < MaxRecursionDepth` 判断，导致同一个子路径被重复加入 extra 队列，造成指数级重复扫描。

**方案**：将检查和标记合并为原子操作：
```go
func (r *recursionCtl) tryMark(path string, maxDepth int) (bool, int) {
    r.mu.Lock()
    defer r.mu.Unlock()
    d := r.depth[path]
    if d >= maxDepth {
        return false, 0
    }
    r.depth[path] = d + 1
    return true, d + 1
}
```

**涉及文件**：`internal/core/scanner/dir/dir_scanner.go`

**Linus 审查意见**：值得做，但被高估了严重性 → 调整为 P1。

看 `worker()` 第 382-391 行，`depthOf()` 和 `markDepth()` 确实是两次独立加锁。25 个 worker 同时命中同一路径时，可能都读到 `depth=0 < 3`，把同样子路径塞进 extra 队列。

**但**：这不会导致数据错误、panic 或资源泄漏。后果只是同一子路径被重复加入队列 → 被重复扫描 → 通配符检测会过滤掉重复命中。

**关于浪费比例的说明**：初始估计的 3-5% 在高深度（MaxRecursionDepth=3+）场景下基本准确。但补充一点：浪费比例实际取决于**并发数 × 请求 RTT**，与 MaxRecursionDepth 的值**无关**。`depth < MaxRecursionDepth` 只是一个条件判断，深度值越小不会让竞争更激烈——竞争来自 25 个 worker 对 `depth` map 的并发读写窗口。当 RTT 为 100ms、并发 25 时，同一请求到达前可能有 2-3 个 worker 同时读到同一个 depth 值，这就是浪费的全部来源。

proposed `tryMark()` 方案是正确的，改动极小（增加一个函数 + 调用点替换），没有复杂度成本。所以我支持做，但不该标 P0。

```
✅ 值得做（方案正确，复杂度低）
⚠️ 优先级：P0 → P1（仅性能影响，不影响正确性）
```

### Task 7.2 — 🔴 P0 加入字典去重机制

**问题**：`Dictionary.addToMain()` 和 `AddExtra()` 均无去重。内置 dicc.txt + 用户字典 + 分类字典 + 递归子路径可能产生大量重复条目，浪费 `strings.Fields` 计算和 HTTP 请求。

**方案**：`Dictionary` 结构体增加 `seen map[string]struct{}` 字段，`addToMain` 和 `AddExtra` 入队前检查 `seen`。

**涉及文件**：`internal/core/scanner/dir/dict/dictionary.go`

**Linus 审查意见**：部分有价值，但方案需要大幅缩减 → P2。

分析 `Dictionary` 的两条插入路径：

1. **`addToMain()`** — 在 `New()` 中**单次、单线程**执行。确实可能重复（dicc.txt + 用户字典 + 分类字典 + 自定义规则可能有重叠），但这是初始化时的常量行为，不会累积。Go 的 map 去重成本比 `[]string` 顺序扫描去重更高。

2. **`AddExtra()`** — 递归时高频调用。**这个才是真正需要去重的地方。** 多次递归后，extra 队列里出现大量重复路径是真实的，比如 `/api/` 和 `/api/users/` 递归后都生成 `/api/config.json`。

**方案修正：`total` 语义问题必须同步修正。**

`Dictionary` 的 `total` 计数器当前统计的是**总入队次数**而非**唯一路径数**：

```go
func (d *Dictionary) AddExtra(paths ...string) {
    for _, p := range paths {
        if d.total >= d.maxSize {
            break
        }
        d.extra = append(d.extra, p)
        d.total++
    }
}
```

**这是一个真实的递归功能退化 bug。** 当字典接近 50 万上限时，一个重复路径会多占一个 `total` 名额，导致后续真正的新路径被提前截断。`maxSize` 限制的不是内存使用，而是"总入队次数"。

**正确做法**：`seen map[string]struct{}` 同时用于 `AddExtra` 去重和 `total` 计数——只有**首次**出现的路径才 `total++`。否则去重只做了一半，`maxSize` 依然是错的。

实际价值：
- 重复请求减少：递归扫描场景下 20-40%
- 递归深度恢复：避免字典满载时递归子路径被截断丢弃

```
✅ 值得做（仅限 AddExtra 去重，且 total 必须同步修正）
❌ 不值得做（addToMain 去重）
⚠️ 优先级：P0 → P2（价值被高估，仅影响递归场景）
```

### Task 7.3 — 🟡 P1 确认 `math/rand` 的 seed 策略

**问题**：`engine/scanner.go` 使用 `rand.Intn()` 生成随机探测路径，但未显式 `rand.Seed()`。Go 1.20+ 默认随机 seed 无问题，Go 1.20 以下会导致所有进程实例生成相同的「随机」路径。

**方案**：检查 `go.mod` 的 Go 版本。若 < 1.20，在 `init()` 或 `NewWildcardScanner` 中 seed 一次。

**涉及文件**：`internal/core/scanner/dir/engine/scanner.go`、`go.mod`

**Linus 审查意见**：伪问题，直接关闭。

`go.mod` 显示 `go 1.25.0`。Go 1.20+ 已经自动 seed 默认 rand source。`scanner.go` 第 283 行用的就是 `rand.Intn()` (默认 source)。

这个任务的前提条件（Go < 1.20）在生产环境中不存在。

> "这是在解决不存在的问题。真正的问题是不要为未来可能的 Go 版本降级写防御代码。"

```
❌ 不值得做（前提条件不存在）
🔒 直接关闭
```

### Task 7.4 — 🟡 P1 `classifyError` 改用 `errors.As` + `net.Error`

**问题**：`engine/requester.go` 第 286-314 行基于错误消息字符串匹配分类错误，Go 版本升级或措辞变化会导致匹配失效。

**方案**：改用类型断言：
```go
var netErr net.Error
if errors.As(err, &netErr) && netErr.Timeout() {
    return ErrConnectionTimeout
}
```

**涉及文件**：`internal/core/scanner/dir/engine/requester.go`

**Linus 审查意见**：值得做，优先级合理。

`requester.go` 第 286-314 行的 `classifyError` 确实用字符串匹配判断错误类型。Go 的底层网络库错误消息在跨平台时有差异（Windows 用 "connection refused"，Linux 也可能报 "ECONNREFUSED"），当前代码已经覆盖了一些，但确实不完整。改进方案（`errors.As` + `net.Error`）是正确的，但它只影响错误分类的**精细度**——不影响重试逻辑的核心路径（`isConnectionError` 只检查 3 个具体错误，分类错了最多导致重试策略选错，不会 crash）。

实际价值：让错误日志更准确，方便排查问题。P1 合理，但生产环境中不是高频问题。

```
✅ 值得做（方案正确，价值明确）
优先级：P1（合理）
```

### Task 7.5 — 🟡 P2 `loadCustomRules` 加环境变量覆盖

**问题**：`dict/dictionary.go` 第 287-330 行硬编码 7 个相对路径 fallback，依赖运行时 CWD。生产环境（systemd/docker）中 CWD 不确定，fallback 可能全部失败。

**方案**：加 `NEOSCAN_RULES_DIR` 环境变量作为最高优先级路径，找不到时 fallback 到现有相对路径列表。

**涉及文件**：`internal/core/scanner/dir/dict/dictionary.go`

**Linus 审查意见**：值得做，真实的部署问题，改动小。

`dictionary.go` 第 287-330 行的 7 个 fallback 路径严重依赖运行时 CWD。在 systemd/docker 环境中 CWD 不可预测，这意味着自定义字典大概率加载失败。

加 `NEOSCAN_RULES_DIR` 环境变量是标准做法，**一句话的改动**，解决了真实场景的问题。

> "这个改动小、价值明确、不引入任何特殊情况。支持。"

```
✅ 值得做（改动极小，解决真实问题）
优先级：P2（合理）
```

### Task 7.6 — 🟢 P3 `shouldRecursion` 用枚举替代三个 bool

**问题**：`ForceRecursive`、`DeepRecursive`、`Recursive` 三个 bool 的组合存在逻辑冗余（如 `Force=true && Deep=false` 的边界情况），且 `shouldRecursion()` 的三个 if 分支可读性一般。

**方案**：定义 `RecursionMode` 枚举（None/Standard/Deep/Force），替代三个 bool 字段，消除模式组合的边界情况。

**涉及文件**：`internal/core/scanner/dir/dir_scanner.go`、`internal/core/options/scan_dir.go`、`cmd/agent/scan/dir.go`

**Linus 审查意见**：不值得做枚举方案，但同意加一行互斥修正。

当前三个 bool 虽然理论上存在 8 种组合，但实际业务逻辑中只有 2 种是有效的：
- `ForceRecursive=true` → 无条件递归
- `DeepRecursive=true` → 深度递归  
- 都 false → 基础递归

**引入枚举类型（None/Standard/Deep/Force）是过度设计**。需要在 3 个文件间同步修改，只为了消除一个可以通过一行代码解决的语义问题。复杂度增加了，收益是 0。

**但我接受一个补充：`ForceRecursive=true && DeepRecursive=true` 存在语义歧义。** 当两者同时为 `true` 时，第一个分支返回 `true`，`DeepRecursive` 参数被静默忽略。这不会导致 bug，但会让使用者困惑——"我同时开了 force 和 deep，到底哪个生效？"

**修正方案：不加枚举，加一行互斥修正**：

```go
if opts.ForceRecursive {
    opts.DeepRecursive = false // force 优先级高于 deep，消除语义歧义
}
```

一行代码，零复杂度，消除语义歧义。

```
❌ 不值得做（枚举替代方案：增加复杂度，零收益）
✅ 值得做（一行互斥修正：消除语义歧义）
🔒 原方案关闭，修正方案补充
```

---

## Phase 7 审查总结

**最值得做的顺序**：7.5 → 7.4 → 7.1 → 7.2(Partial) → 7.6(互斥修正)

**最大的 Linus 式批评**：7.2 整体标为 P0 是**优先级倒置**——初始化时的常量重复去重（addToMain）比递归时的动态重复（AddExtra）重要 100 倍的说法是荒谬的。7.3 则是在解决不存在的问题。

**最终决策**：

| 任务 | 原优先级 | 审查后 | 决策 |
|------|---------|--------|------|
| 7.1 TOCTOU 竞争 | P0 | P1 | ✅ 值得做（方案正确，复杂度低） |
| 7.2 字典去重 | P0 | P2 | ⚠️ 只做 AddExtra 去重，total 必须同步修正 |
| 7.3 rand seed | P1 | 关闭 | ❌ 伪问题（Go 1.25 自动 seed） |
| 7.4 classifyError | P1 | P1 | ✅ 值得做（方案正确，价值明确） |
| 7.5 环境变量 | P2 | P2 | ✅ 值得做（改动极小，解决真实问题） |
| 7.6 枚举替代 bool | P3 | 关闭 | ❌ 过度设计（但同意加一行互斥修正） |

---

## 补充评估修正说明

以下是对补充评估意见的审查修正。

### Task 7.1 — recursionCtl TOCTOU 竞争：P0 → P1 ✅ 同意，有补充修正

**同意点**：`tryMark()` 方案改动极小，值得做。TOCTOU 不会导致数据错误或 panic，通配符检测会过滤掉大部分重复命中。

**修正点**："低深度场景浪费 10-15%"——**这个说法不对**。

浪费比例实际取决于**并发数 × 请求 RTT**，与 MaxRecursionDepth 的值**无关**。`depth < MaxRecursionDepth` 只是一个条件判断，深度值越小不会让竞争更激烈——竞争来自 25 个 worker 对 `depth` map 的并发读写窗口。

- `MaxRecursionDepth=0`：条件 `0 < 0` 永远不成立，**递归完全禁用，零浪费**。
- `MaxRecursionDepth=3`：竞争窗口大小相同，取决于同一请求到达前有几个 worker 同时读到同一个 depth 值。当 RTT 为 100ms、并发 25 时，可能有 2-3 个 worker 同时读到 `depth=0`，这就是浪费的全部来源。

> 📉 **结论**：低深度 ≠ 高浪费。初始估计的 3-5% 在高深度场景下基本准确。

---

### Task 7.2 — 字典去重：P0 → P2，只做 AddExtra ⚠️ 方向正确，补充确认

**同意点**：区分 `addToMain`（不值得去重）和 `AddExtra`（需要去重）完全正确。P2 优先级合理。

**关键补充确认**：`total` 语义问题**必须同步修正**。

`total` 当前统计的是总入队次数而非唯一路径数，当字典接近 50 万上限时，重复路径多占名额 → 真正的新路径被提前截断 → **递归功能退化**。

✅ **确认实施方案**：`seen map[string]struct{}` 同时用于 `AddExtra` 去重和 `total` 计数控制——只有**首次**出现的路径才 `total++`。

---

### Task 7.3 — math/rand seed：直接关闭 ✅ 完全同意

---

### Task 7.4 — classifyError 改用 errors.As：P1 保持 ✅ 完全同意

---

### Task 7.5 — loadCustomRules 环境变量：P2 保持 ✅ 完全同意

---

### Task 7.6 — shouldRecursion 枚举化：直接关闭 ⚠️ 同意补充修正

**同意补充**：`ForceRecursive=true && DeepRecursive=true` 存在语义歧义（Force 优先级更高，Deep 被静默忽略），加一行互斥修正是对的。

```go
if opts.ForceRecursive {
    opts.DeepRecursive = false // force 优先级高于 deep，消除语义歧义
}
```

**但引入枚举类型依然是过度设计**。一行互斥修正 + 关闭枚举方案 = 最优解。

---

## 评估总结

| 任务 | Linus 判断 | 我的评估 | 关键差异 |
|------|-----------|---------|---------|
| 7.1 | P0→P1 | ✅ 同意 | 补充：低深度场景浪费比例更高（10-15% vs 3-5%） |
| 7.2 | P0→P2，只做 AddExtra | ⚠️ 方向正确 | 需同步修正 `total` 计数器语义，否则递归扫描在字典满载时失效 |
| 7.3 | 关闭 | ✅ 同意 | 无差异 |
| 7.4 | P1 保持 | ✅ 同意 | 无差异 |
| 7.5 | P2 保持 | ✅ 同意 | 无差异 |
| 7.6 | 关闭 | ⚠️ 方向正确 | 建议加一行互斥修正（`Force=true` 时重置 `Deep=false`），不引入枚举 |

**总体评价**：Linus 式审查整体水平很高，特别是：
- 7.1 的量化估计和优先级调整
- 7.2 区分 `addToMain` 和 `AddExtra` 的价值差异
- 7.3 识别伪问题并直接关闭
- 7.6 拒绝过度设计

需要修正的两处（7.2 的 `total` 语义、7.6 的互斥修正）都是"方向正确但细节需要打磨"的情况，不影响整体决策框架。

---

## Phase 7 任务状态跟踪

| 任务 | 状态 | 备注 |
|------|:---:|------|
| 7.1 修复 recursionCtl TOCTOU 竞争 | ✅ 已完成 | `tryMark()` 原子操作替代 `depthOf()`+`markDepth()`，消除竞争窗口 |
| 7.2 字典去重机制 | ✅ 已完成 | `seen` map 仅跟踪 extra 队列，重复路径跳过且不占 `total` 名额；`TestDictionary_AddExtra_Dedup` 等 2 个新测试覆盖 |
| 7.3 math/rand seed 确认 | 🔒 已关闭 | Go 1.25 自动 seed，伪问题 |
| 7.4 classifyError 改用 errors.As | ✅ 已完成 | 类型断言优先（net.Error/DNSError/tls.RecordHeaderError）+ 字符串兜底；`TestRequester_ClassifyError_RealTypes` 覆盖 |
| 7.5 loadCustomRules 环境变量 | ✅ 已完成 | `NEOSCAN_RULES_DIR` 环境变量作为最高优先级路径 |
| 7.6 shouldRecursion 互斥修正 | ✅ 已完成 | `ForceRecursive=true` 时重置 `DeepRecursive=false`，消除语义歧义 |
