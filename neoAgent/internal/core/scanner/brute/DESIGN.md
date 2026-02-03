# Brute Scanner 设计方案 v3.0 (Developable Spec)

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
│   └── redis.go
└── DESIGN.md            # 本文档
```

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

### 4.2 伪代码实现

```go
func (s *BruteScanner) Scan(ctx context.Context, task *model.Task) {
    // 1. 根据 Service 获取对应的 Cracker
    cracker := s.getCracker(task.Service)
    
    // 2. 生成字典 (根据 AuthMode)
    authList := s.dictManager.Generate(task.Params, cracker.Mode())
    
    // 3. 获取全局令牌 (QoS)
    if err := s.limiter.Acquire(ctx); err != nil {
        return
    }
    defer s.limiter.Release()

    // 4. Target Worker: 串行遍历
    for _, auth := range authList {
        // 快速失败检查
        select {
        case <-ctx.Done():
            return
        default:
        }

        // 执行检查 (带超时)
        checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
        success, err := cracker.Check(checkCtx, task.IP, task.Port, auth)
        cancel()

        if success {
            s.reportSuccess(task, auth)
            if s.config.StopOnSuccess {
                return // 爆破成功即停止
            }
        }

        // 错误处理与速率调整
        if err != nil {
             // 如果是网络错误，limiter.OnFailure() 降低全局并发
             // 如果是认证错误，limiter.OnSuccess() 保持或增加并发
             s.handleError(err)
        }
        
        // 避免 CPU 100% 的微小休眠 (可选)
        time.Sleep(10 * time.Millisecond)
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

### 6.1 SSH (`protocol/ssh.go`)
*   [ ] 库: `golang.org/x/crypto/ssh`
*   [ ] 配置: `HostKeyCallback: ssh.InsecureIgnoreHostKey()` (必须忽略 HostKey)
*   [ ] 模式: `AuthModeUserPass`
*   [ ] 判据: `ssh.NewClientConn` 返回 nil error 即为成功。

### 6.2 MySQL (`protocol/mysql.go`)
*   [ ] 库: `github.com/go-sql-driver/mysql`
*   [ ] DSN: `user:pass@tcp(host:port)/dbname?timeout=3s`
*   [ ] 模式: `AuthModeUserPass`
*   [ ] 判据: `db.Ping()` 成功。

### 6.3 Redis (`protocol/redis.go`)
*   [ ] 库: `github.com/redis/go-redis/v9`
*   [ ] 配置: `Client` 连接
*   [ ] 模式: `AuthModeOnlyPass`
*   [ ] 判据: `client.Ping()` 成功 (注意区分 NOAUTH 错误)。

---

## 7. 任务集成 (Integration)

*   **Runner 注册**:
    在 `internal/core/runner/runner_manager.go` 中，将 `BruteScanner` 注册为处理 `TaskTypeBrute` 的 Runner。
*   **输入参数映射**:
    Task Params 需要包含:
    *   `service`: 协议名 (ssh, mysql...)
    *   `users`: (可选) 自定义用户字典
    *   `passwords`: (可选) 自定义密码字典

## 8. 开发计划 (Roadmap)

1.  **Phase 1 (基础)**:
    *   定义 `cracker.go` 接口。
    *   实现 `dict.go` 及 `%user%` 逻辑。
2.  **Phase 2 (协议)**:
    *   实现 SSH, MySQL, Redis 三个 Cracker。
3.  **Phase 3 (调度)**:
    *   实现 `scanner.go`，集成 `AdaptiveLimiter`。
    *   编写单元测试验证并发控制。
4.  **Phase 4 (集成)**:
    *   对接 RunnerManager，进行端到端测试。
