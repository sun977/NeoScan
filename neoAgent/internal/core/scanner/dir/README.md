# DirScanner — 目录/文件扫描模块

- **版本**: v1.0
- **日期**: 2026-08-25

DirScanner 是 NeoAgent 的目录与敏感文件扫描原子模块，移植自 dirsearch 的核心算法并用 Go 原生重写。内置通配符（动态 404）检测引擎，可有效避免 CDN/WAF 统一错误页造成的误报。

## 实现原理

### 整体流水线

```mermaid
flowchart TD
    A["入口 A（CLI）<br/>neoagent scan dir"] --> A1["DirScanOptions.Validate()<br/>参数预校验"]
    A1 --> A2["ToTask()<br/>组装 model.Task"]
    B["入口 B（Master 下发）<br/>type: dirScan"] --> B1["task_to_core.go<br/>dirScan → TaskTypeDirScan<br/>meta 合并到 Params"]
    A2 --> T["model.Task<br/>Type=dir_scan，Params 下划线命名"]
    B1 --> T
    T --> M["RunnerManager.Execute<br/>CLI/Master 共用注册表"]
    M --> R["DirScanner.Run()<br/>parseDirOptions()"]

    subgraph CORE["执行核心组件"]
        D["Dictionary<br/>双队列字典"]
        Q["Requester<br/>HTTP 请求器"]
        F["Filter<br/>多层过滤"]
        W["WildcardScanner<br/>通配符检测"]
        S["statsAccumulator<br/>原子统计"]
    end

    R --> CORE
    CORE --> POOL["Worker 池（N goroutines）<br/>取路径 → 限速 → 请求 → 过滤<br/>→ 通配符检测 → 命中上报 → 递归展开"]
    POOL --> H["hits channel"]
    H --> RES["DirResult（单线程汇总）"]
    RES --> O1["CLI 路径：终端表格 / JSON / CSV"]
    RES --> O2["Master 路径：ReportTask 上报"]
```

### 核心机制

**1. 双队列字典（dict/dictionary.go）**

- `main` 队列：初始化时预加载（内置字典 + 用户字典 + 分类字典 + 自定义规则），经 %EXT% 展开、大小写变换、前后缀组合后只读
- `extra` 队列：递归扫描动态发现的子路径，**优先于 main 消费**，内部用 `seen` map 去重
- 条目上限 500,000（`maxSize`），限制的是唯一路径数而非总入队次数

**2. 真递归扫描（Phase 8 重构）**

命中目录（如 `/admin/` 返回 200/301/302/403）后，将**整个主字典在该目录前缀下重放**：

```
命中 /admin/ → 展开 /admin/<字典每条目> → /admin/config.php、/admin/login ...
```

- 递归深度 = 路径嵌套层数（`/admin/` 深度 1，`/admin/api/` 深度 2），无需额外状态
- 同一目录通过 `tryMarkExpand` 原子标记，只展开一次
- 三种模式：基础递归（仅目录）、深度递归（含所有父级目录）、强制递归（任意命中路径）

**3. 通配符检测（engine/scanner.go）**

识别"任意路径都返回同一动态内容"的通配符响应（典型如 CDN 统一 404 页）：

- **采样阶段**（每路径前缀一次性，`sync.Once` 保证）：向随机不存在路径发起 5 个探测请求，提取共同静态词作为基准
- **判定阶段**：响应归一化（UUID/长 hex/时间戳/Base64 替换为占位符）后与基准比对，同时检查词数膨胀比例（>1.35 倍判为独立内容）
- **指纹自校准**：同一响应指纹出现超过 10 次自动加入通配符黑名单
- 探测请求纳入并发控制：worker 持有限流槽位直到检测完成 + 内部信号量门控（容量 8）

**4. 多层过滤（engine/filter.go）**

按序执行：状态码白名单 → 状态码黑名单（默认 404/500/502/503）→ 响应大小黑名单 → 关键词排除 → 正则排除

**5. 统计（statsAccumulator）**

全部使用 `atomic.Int64` 无锁计数：总请求/成功响应/被过滤/通配符拦截/错误 + RTT 最大值/最小值/平均值。扫描结束后单线程 `Finalize()` 一次性转换。

## 目录结构

```
internal/core/scanner/dir/
├── dir_scanner.go            # 主扫描器：Worker 池调度、递归控制、统计聚合、参数解析
├── dir_scanner_test.go       # 单元测试（含真递归断言、goroutine 泄漏检测）
├── dir_scanner_bench_test.go # 性能基准测试
├── dir_scanner_e2e_test.go   # 端到端验收测试
├── README.md                 # 本文档
│
├── dict/                     # 字典管理子包
│   ├── dictionary.go         # 双队列字典迭代器（main 只读 + extra 动态去重）
│   ├── template.go           # %EXT% 扩展名展开（Classic/Force 两种模式）
│   ├── wordlists.go          # 内置字典加载（go:embed）
│   └── wordlists/            # 内置字典数据文件
│       ├── dicc.txt          # 主字典（9,681 行，558 条含 %EXT%）
│       ├── categories/       # 技术栈分类字典（25 个，如 wordpress、spring）
│       └── user-agents.txt   # User-Agent 池（53 条）
│
├── engine/                   # 扫描引擎子包
│   ├── requester.go          # HTTP 请求器：连接池复用、重试退避、限速、代理、错误分类
│   ├── filter.go             # 多层响应过滤器 + Response 结构定义
│   └── scanner.go            # 通配符检测引擎（采样 + 指纹自校准）
│
└── result/                   # 结果对象子包
    └── result.go             # DirHit（单次命中）、DirResult（完整结果）、ScanStats（统计）
```

相关外部文件：

```
cmd/agent/scan/dir.go              # CLI 子命令：flag 注册、参数分级（--help/--help-all）
internal/core/options/scan_dir.go  # CLI 参数集合：Validate() 预校验、ToTask() 参数下发
internal/core/factory/dir_factory.go # 扫描器工厂（RunnerManager 注册用）
```

## 参数使用

### 参数分级

- `neoagent scan dir --help`：只显示 P0 常用参数（15 个）
- `neoagent scan dir --help-all`：显示全部参数（40 个，含高级参数）

### P0 常用参数

| 参数 | 短写 | 默认值 | 说明 |
|------|:---:|:---:|------|
| `--target` | `-t` | — | 目标 URL（也可用位置参数） |
| `--extensions` | `-e` | `php,asp,html,js,json,bak,git,env` | 扩展名列表，替换字典中的 `%EXT%` |
| `--wordlists` | `-w` | 内置字典 | 自定义字典文件（逗号分隔多个） |
| `--category` | — | — | 追加技术栈字典，如 `wordpress,spring` |
| `--threads` | — | 25 | 并发线程数（1~500） |
| `--timeout` | — | 10 | 单请求超时（秒，1~300） |
| `--recursive` | `-r` | true | 命中目录后递归扫描 |
| `--force-extensions` | `-f` | false | 不含 %EXT% 的条目也追加扩展名变体 |
| `--exclude-status` | `-x` | `404,500,502,503` | 状态码黑名单 |
| `--max-recursion-depth` | `-R` | 3 | 最大递归深度（0~10） |
| `--verbose` | `-v` | false | 详细输出 |
| `--quiet` | `-q` | false | 静默模式 |
| `--header` | `-H` | — | 自定义请求头，可多次指定 |
| `--method` | `-m` | GET | HTTP 方法 |
| `--targets-file` | `-l` | — | 批量目标文件（每行一个 URL） |

### P1/P2 高级参数（--help-all 可见）

| 参数 | 默认值 | 说明 |
|------|:---:|------|
| `--deep-recursive` | false | 深度递归：命中路径的所有父级目录也展开 |
| `--force-recursive` | false | 强制递归：任意命中路径都展开 ⚠️ 每个命中路径重放整个字典，流量急剧放大 |
| `--proxy` | — | 代理地址 `http://127.0.0.1:8080` |
| `--follow-redirects` | false | 跟随 301/302 重定向 |
| `--max-retries` | 2 | 失败重试次数（指数退避，上限 5s） |
| `--rate-limit` | 0 | 每秒请求数上限（0=不限） |
| `--delay` | 0 | 每次请求间隔（毫秒） |
| `--max-time` | 0 | 整个扫描最长运行时间（秒） |
| `--target-max-time` | 0 | 单目标最长扫描时间（秒） |
| `--random-agent` | false | 内置 UA 池随机轮换 |
| `--prefixes` / `--suffixes` | — | 字典条目追加前/后缀（笛卡尔积） |
| `--uppercase` / `--lowercase` / `--capital` | false | 大小写变换（优先级 U > L > C） |
| `--include-status` | — | 状态码白名单（优先于黑名单） |
| `--exclude-text` | — | 响应体关键词排除，可多次指定 |
| `--exclude-regex` | — | 响应体正则排除 |
| `--exclude-size` | — | 响应体大小黑名单（字节） |
| `--headers-file` | — | 从文件加载请求头 |
| `--auth` | — | Basic Auth（`user:pass`） |
| `--user-agent` | — | 自定义 UA（优先级低于 --random-agent） |
| `--ip` / `--network-interface` | — | 绑定本地出口 IP / 网卡 |

### 使用示例

```bash
# 基本扫描（内置字典 + 默认递归）
neoagent scan dir https://example.com

# 指定扩展名 + 技术栈字典（目标已知是 WordPress）
neoagent scan dir https://example.com -e php,bak --category wordpress

# 自定义字典 + 关闭递归（快速扫一层）
neoagent scan dir https://example.com -w mydict.txt -r=false

# 深度递归，限制深度 2（探测每个父级目录）
neoagent scan dir https://example.com --deep-recursive -R 2

# 低速率扫描（规避 WAF/限速）
neoagent scan dir https://example.com --rate-limit 10 --delay 100

# 带认证和自定义头
neoagent scan dir https://example.com --auth admin:secret -H "X-Token: abc123"

# 批量目标 + 结果导出
neoagent scan dir -l targets.txt -oJ result.json -oC result.csv

# 只看 200 和 403 的结果
neoagent scan dir https://example.com -i 200,403
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `NEOSCAN_RULES_DIR` | 自定义字典目录（`*.txt` 自动加载），优先级高于默认的 `rules/dir/custom/` 相对路径搜索 |

## 编程调用与 Master 下发

DirScanner 实现 `runner.Runner` 接口，已注册到 `RunnerManager`（`manager.go`），CLI 与 Master 集群下发**共用同一个执行核心**，两条路径仅"任务来源和结果去向"不同，扫描行为完全一致。

### Master 下发链路

```
Master 任务 {"type": "dirScan", "target": "http://example.com", "meta": {...}}
    ↓  adapter/task_to_core.go: case "dir_scan"/"dirScan" → TaskTypeDirScan
    ↓  meta 逐条合并到 task.Params
RunnerManager.Execute(ctx, task)
    ↓  factory.NewDirScanner()（已注册）
DirScanner.Run() → parseDirOptions(task.Params)
```

### task.Params 参数契约

Master 下发时 `meta` 中的 key 必须使用**下划线命名**，与 `parseDirOptions()` 的解析约定一一对应。值的类型要求：`bool` 用 JSON 布尔，`int` 用 JSON 数字（`int`/`float64` 均兼容），列表用逗号分隔字符串。

| Params key | 类型 | 对应 CLI 参数 |
|------|:---:|------|
| `wordlists` | string | `--wordlists` |
| `category` | string | `--category` |
| `extensions` | string | `--extensions` |
| `force_extensions` | bool | `--force-extensions` |
| `prefixes` / `suffixes` | string | `--prefixes` / `--suffixes` |
| `uppercase` / `lowercase` / `capital` | bool | `-U` / `-L` / `-C` |
| `threads` | int | `--threads` |
| `timeout` | int（秒） | `--timeout` |
| `max_retries` | int | `--max-retries` |
| `method` | string | `--method` |
| `recursive` | bool | `--recursive` |
| `deep_recursive` | bool | `--deep-recursive` |
| `force_recursive` | bool | `--force-recursive` |
| `max_recursion_depth` | int | `--max-recursion-depth` |
| `rate_limit` | int | `--rate-limit` |
| `delay` | int（毫秒） | `--delay` |
| `max_time` / `target_max_time` | int（秒） | `--max-time` / `--target-max-time` |
| `include_status` / `exclude_status` | string | `--include-status` / `--exclude-status` |
| `exclude_size` | string | `--exclude-size` |
| `exclude_text` | string | `--exclude-text` |
| `exclude_regex` | string | `--exclude-regex` |
| `follow_redirects` | bool | `--follow-redirects` |
| `proxy` | string | `--proxy` |
| `random_agent` | bool | `--random-agent` |
| `user_agent` | string | `--user-agent` |
| `auth` | string（`user:pass`） | `--auth` |
| `headers` | string（多行 `Key: Value`） | `-H` / `--headers-file` |
| `ip` / `network_interface` | string | `--ip` / `--network-interface` |
| `skip_builtin` | bool | （仅测试用，跳过内置字典） |

### 编程调用示例

```go
manager := runner.NewRunnerManager()
task := &model.Task{
    ID:      "task-001",
    Type:    model.TaskTypeDirScan,
    Target:  "http://example.com",
    Timeout: 2 * time.Hour,
    Params: map[string]interface{}{
        "threads":             50,
        "extensions":          "php,bak",
        "category":            "wordpress",
        "recursive":           true,
        "max_recursion_depth": 2,
        "exclude_status":      "404,500",
    },
}
results, err := manager.Execute(ctx, task)
```

## 注意事项

1. **递归流量**：默认 `recursive=true`，真递归下每个命中目录会重放整个主字典（内置字典展开后约 13,600 条）。目标目录结构深时请评估流量，或用 `-r=false` 关闭、`-R 1` 限制深度。
2. **递归上限**：字典条目上限 500,000。内置字典约 13.6K 条，理论上第 36 个命中目录后递归展开会被截断（日志有 Warnf 提示，不中断扫描）。
3. **`--force-recursive` 警告**：命中文件（如 `/admin/config.php`）也会当目录展开，产生大量无效请求，仅建议在确认目标行为后使用。
4. **通配符站点**：目标对任意路径返回 200 时，通配符检测会自动识别并抑制误报，统计中的 `wildcard_reqs` 反映被拦截数量。
5. **HTTPS 证书**：默认跳过证书校验（`InsecureSkipVerify`），支持自签名站点。
