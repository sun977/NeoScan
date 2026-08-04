# Brute Scanner 设计方案 v3.1

- **版本**: v3.1
- **日期**: 2026-08-04
- **状态**: 已实现（Phase 1-4 均已落地，详见第 8 节）

## 1. 核心目标 (Objectives)
实现一个**企业级、高并发、安全**的弱口令爆破模块。
核心设计哲学：**"以 Qscan 为骨架，以 NeoAgent QoS 为灵魂"**。

*   **极简接口**: 借鉴 Qscan 的 `Cracker` 接口设计，协议实现零依赖。
*   **安全并发**: 结合 NeoAgent 的 `AdaptiveLimiter` (AIMD) 实现全局流控，同时强制**单目标串行**以规避锁定。
*   **智能调度**: 支持 UserPass/OnlyPass/None 三种认证模式，支持 `%user%` 动态字典。

---

## 2. 架构设计 (Architecture)

### 2.1 模块结构
```text
internal/core/scanner/brute/
├── scanner.go           # [Scheduler] 扫描器入口, 并发控制, 结果收集
├── cracker.go           # [Interface] 核心接口定义, 错误类型定义
├── dict.go              # [Dict] 字典管理器 (内置 + 动态生成)
├── protocol/            # [Impl] 具体协议实现 (无业务依赖)
│   ├── ssh.go
│   ├── mysql.go
│   ├── redis.go
│   ├── postgres.go
│   ├── ftp.go
│   ├── mongo.go
│   ├── clickhouse.go
│   ├── smb.go
│   ├── mssql.go
│   ├── oracle.go
│   ├── oracle_sid.go
│   ├── telnet.go
│   ├── elasticsearch.go
│   ├── snmp.go
│   ├── rdp/             # RDP 纯 Go 实现 (无需 CGO)
│   └── ...
├── README.md            # 模块说明（面向使用者/扩展者）
└── DESIGN.md            # 本文档（面向设计与实现细节）
```

> 说明: 实际已实现 15 种协议（SSH/MySQL/Redis/PostgreSQL/FTP/MongoDB/ClickHouse/SMB/MSSQL/Oracle/Oracle SID/Telnet/Elasticsearch/SNMP/RDP），早期版本设计草图仅列出前 3 个作为示例，现已补全。

### 2.2 数据流 (Data Flow)
```mermaid
graph TD
    A[Task (Target, Port, Service)] --> B[BruteScanner]
    B --> C{Dict Generator}
    C -->|Generate| D[Auth Pair List]
    B --> E{Global Limiter}
    E -->|Acquire Token| F[Target Worker (Goroutine)]
    F -->|Serial Loop| G[Cracker.Check]
    G -->|Result| H[Result Collector]
```

---

## 3. 核心接口定义 (Core Interfaces)

### 3.1 Cracker 接口
位于 `internal/core/scanner/brute/cracker.go`。
保持纯粹，不引入 `model.Task` 等上层结构。

```go
package brute

import "context"

// AuthMode 定义爆破模式
type AuthMode int

const (
	AuthModeUserPass AuthMode = iota // 需要用户名和密码 (SSH, MySQL)
	AuthModeOnlyPass                 // 仅需要密码 (Redis, VNC)
	AuthModeNone                     // 无需认证/默认凭据 (Telnet, MongoDB)
)

// Auth 认证凭据 (数据传输对象)
type Auth struct {
	Username string
	Password string
	Other    map[string]string // 扩展字段 (e.g. Oracle SID)
}

// Cracker 协议适配器接口
type Cracker interface {
	// Name 返回协议名称 (e.g. "ssh", "mysql")
	Name() string

	// Mode 返回该协议的爆破模式
	Mode() AuthMode

	// Check 验证单个凭据
	// context: 用于控制超时 (通常 3-5秒)
	// host, port: 目标地址
	// auth: 待验证凭据
	// 返回:
	// - bool: true 表示认证成功
	// - error: 见 "错误处理标准" 章节
	Check(ctx context.Context, host string, port int, auth Auth) (bool, error)
}
```

### 3.2 错误处理标准 (Error Handling)
区分"业务失败"与"技术失败"，这对于调度器决定是否重试至关重要。

```go
var (
	// ErrAuthFailed 认证失败 (账号密码错误) -> 继续尝试下一个
	// 实现 Check 时，如果确定是密码错误，返回 (false, nil) 即可，也可以返回此 error 用于日志
	ErrAuthFailed = errors.New("auth failed")

	// ErrConnectionFailed 连接失败 (超时/拒绝/重置) -> 触发重试或熔断
	ErrConnectionFailed = errors.New("connection failed")

	// ErrProtocolError 协议交互错误 (如非预期响应) -> 视为该协议不支持，跳过
	ErrProtocolError = errors.New("protocol error")
)
```

---

## 4. 调度器逻辑 (Scheduler Logic)

### 4.1 并发模型：全局并发 + 单机串行
这是本设计的核心安全保障。

1.  **全局并发 (Global Concurrency)**:
    *   复用 `internal/core/lib/network/qos` 中的 `AdaptiveLimiter`。
    *   限制同时进行的**目标主机数** (Target Workers)，而不是连接数。
2.  **单机串行 (Strict Serial per Target)**:
    *   对于同一个 `(IP, Port)`，**绝对禁止**并发发起多个登录请求。
    *   **实现**: 为每个 Target 启动**一个** Goroutine (Target Worker)，在该 Goroutine 内部串行遍历字典。
    *   **原因**: 防止触发 Account Lockout 或被 IDS 识别为攻击。

### 4.2 实际实现（`scanner.go` Run 方法要点）
以下按真实代码摘录关键步骤，伪代码已与实现对齐：

```go
func (s *BruteScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
    // 1. 从 task.Params["service"] 解析协议名，查找对应 Cracker
    serviceName, _ := task.Params["service"].(string)
    cracker, exists := s.crackers[serviceName]

    // 2. 生成字典 (根据 AuthMode，支持 task.Params["users"]/["passwords"] 覆盖)
    authList := s.dict.Generate(task.Params, cracker.Mode())

    // 3. 获取全局并发令牌 (QoS AdaptiveLimiter)
    s.globalLimit.Acquire(ctx)
    defer s.globalLimit.Release()

    // 4. stop_on_success 默认 true，可通过 task.Params["stop_on_success"] 覆盖
    stopOnSuccess := true
    if v, ok := task.Params["stop_on_success"].(bool); ok {
        stopOnSuccess = v
    }

    // 5. 双重循环：外层端口 (task.PortRange 解析出的端口列表)，内层字典
    //    单个 (IP, Port) 内严格串行，避免触发账号锁定
    for _, port := range ports {
        for _, auth := range authList {
            success, err := cracker.Check(checkCtx, task.Target, port, auth)
            if success {
                // 记录成功凭据
                if stopOnSuccess {
                    return results, nil
                }
            }
        }
    }
}
```

---

## 5. 字典管理 (Dictionary Manager)

### 5.1 动态生成逻辑
参考 Qscan 的 `MakePassword`，支持运行时动态组合。

*   **输入**:
    *   `Users []string`: 来自 Task 参数 (用户指定) 或 内置 TopUser。
    *   `Passs []string`: 来自 Task 参数 (用户指定) 或 内置 TopPass。
*   **占位符替换**:
    *   遍历密码字典，如果发现 `%user%` 字符串，将其替换为**当前正在尝试的用户名**。
    *   场景: 检测 `admin:admin`, `root:root` 等 "账号密码相同" 的弱口令。

### 5.2 组合策略
```go
func Generate(users, passes []string, mode AuthMode) []Auth {
    var list []Auth
    
    switch mode {
    case AuthModeUserPass:
        // 笛卡尔积: User * Pass
        for _, u := range users {
            for _, p := range passes {
                realPass := strings.ReplaceAll(p, "%user%", u)
                list = append(list, Auth{Username: u, Password: realPass})
            }
        }
    case AuthModeOnlyPass:
        // 仅遍历密码
        for _, p := range passes {
            list = append(list, Auth{Password: p})
        }
    case AuthModeNone:
        // 空凭据
        list = append(list, Auth{})
    }
    return list
}
```

---

## 6. 协议实现清单 (Implementation Checklist)

以下协议均已实现（位于 `protocol/` 目录），Checklist 状态更新为已完成：

### 6.1 SSH (`protocol/ssh.go`)
*   [x] 库: `golang.org/x/crypto/ssh`
*   [x] 配置: `HostKeyCallback: ssh.InsecureIgnoreHostKey()` (必须忽略 HostKey)
*   [x] 模式: `AuthModeUserPass`
*   [x] 判据: `ssh.NewClientConn` 返回 nil error 即为成功。

### 6.2 MySQL (`protocol/mysql.go`)
*   [x] 库: `github.com/go-sql-driver/mysql`
*   [x] DSN: `user:pass@tcp(host:port)/dbname?timeout=3s`
*   [x] 模式: `AuthModeUserPass`
*   [x] 判据: `db.Ping()` 成功。

### 6.3 Redis (`protocol/redis.go`)
*   [x] 库: `github.com/redis/go-redis/v9`
*   [x] 配置: `Client` 连接
*   [x] 模式: `AuthModeOnlyPass`
*   [x] 判据: `client.Ping()` 成功 (注意区分 NOAUTH 错误)。

### 6.4 其余协议
PostgreSQL、FTP、MongoDB、ClickHouse、SMB、MSSQL、Oracle、Oracle SID、Telnet、Elasticsearch、SNMP、RDP 均已按同样的 `Cracker` 接口规范实现，具体判据参见各自源文件注释，不再逐一展开。

---

## 7. 任务集成 (Integration)

*   **Runner 注册**（真实路径，非早期草图中的 `runner_manager.go`）:
    * 所有协议的 Cracker 统一在 `internal/core/factory/brute_factory.go` 的 `NewFullBruteScanner()` 中通过 `RegisterCracker` 注册，形成一个"全功能" `*brute.BruteScanner` 实例。
    * `internal/core/runner/manager.go` 的 `NewRunnerManager()` 直接调用 `factory.NewFullBruteScanner()` 获取该实例并注册为 `TaskTypeBrute` 的 Runner，CLI 与 Master 共用同一份注册表。
    * **新增协议时只需修改 `brute_factory.go` 一处**，不要在 `manager.go` 里单独 `RegisterCracker`，否则会导致 CLI/Master 能力不一致。
*   **输入参数映射**（`Task.Params`，由 `internal/core/options/scan_brute.go` 的 `BruteScanOptions.ToTask()` 或 Master `task_to_core.go` 构造）:
    *   `service` (必填): 协议名 (ssh, mysql, redis, postgres, ftp, mongo, clickhouse, smb, mssql, oracle, oracle-sid, telnet, elasticsearch, snmp)
    *   `users`: (可选) 自定义用户字典，缺省使用内置 TopUser
    *   `passwords`: (可选) 自定义密码字典，缺省使用内置 Top100 弱口令
    *   `stop_on_success`: (可选，默认 `true`) 是否找到一个有效凭据后即停止
    *   `Task.Target`: 目标 IP
    *   `Task.PortRange`: 目标端口，支持逗号分隔多端口 (e.g. `"22,2222"`)
*   **CLI 侧的 Options 层**（2026-08-04 补齐，详见 [Agent指令集规范.md](../../../../docs/Agent指令集规范.md) v1.3 更新记录）:
    * `cmd/agent/scan/brute.go` 通过 `options.BruteScanOptions` 完成参数校验和默认端口推断（`Validate()` 中按 `service` 查表得到默认端口），再 `ToTask()` 转换为上述 `Task.Params`，与其余已实现扫描指令（alive/port/os/web）的组织方式统一。

## 8. 开发计划 (Roadmap) —— 已全部完成

1.  **Phase 1 (基础)** ✅
    *   定义 `cracker.go` 接口。
    *   实现 `dict.go` 及 `%user%` 逻辑。
2.  **Phase 2 (协议)** ✅（超出原计划，实际实现 15 种协议而非仅 SSH/MySQL/Redis 三种）
    *   实现 SSH, MySQL, Redis, PostgreSQL, FTP, MongoDB, ClickHouse, SMB, MSSQL, Oracle, Oracle SID, Telnet, Elasticsearch, SNMP, RDP 共 15 个 Cracker。
3.  **Phase 3 (调度)** ✅
    *   实现 `scanner.go`，集成 `AdaptiveLimiter`。
    *   编写单元测试验证并发控制 (`scanner_test.go`)。
4.  **Phase 4 (集成)** ✅
    *   对接 RunnerManager，通过 `factory.NewFullBruteScanner()` 统一注册。
    *   CLI (`neoAgent scan brute`) 与 Master (`brute_force` TaskType) 两条入口共用同一套 Runner，完成端到端联调。
    *   补齐 `internal/core/options/scan_brute.go`，CLI 侧统一为 Options 模式（Phase 4 的收尾项，2026-08-04 完成）。
