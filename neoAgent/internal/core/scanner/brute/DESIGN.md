# Brute Scanner 开发方案 (Phase 3.7)

## 1. 核心目标
实现一个**轻量、快速、安全**的弱口令爆破模块。
该模块应作为 `internal/core/scanner/brute` 的一部分，提供原生的协议支持，不依赖外部工具。

## 2. 架构设计

### 2.1 目录结构
```text
internal/core/scanner/brute/
├── scanner.go           # BruteScanner 主逻辑 (Task调度, 字典管理)
├── cracker.go           # Cracker 接口定义
├── dict.go              # 内置字典 (Top 100)
├── ssh_cracker.go       # SSH 协议实现
├── mysql_cracker.go     # MySQL 协议实现
└── redis_cracker.go     # Redis 协议实现
```

### 2.2 接口定义 (Cracker)
```go
// Cracker 定义特定协议的爆破逻辑
type Cracker interface {
    // Name 返回协议名称 (e.g. "ssh", "mysql")
    Name() string
    
    // Crack 尝试验证一对用户名/密码
    // 返回: success, error (网络错误等)
    Crack(ctx context.Context, host string, port int, user, pass string) (bool, error)
}
```

### 2.3 字典策略 (Dictionary Strategy)
1.  **内置字典 (Built-in)**: 编译进二进制的 Top 100 弱口令，保证开箱即用。
2.  **运行时覆盖 (Runtime Override)**: 优先使用 Task 参数中传入的 User/Pass 列表。
3.  **组合逻辑**: 双重循环 (User x Pass)，支持无用户名的协议 (Redis)。

### 2.4 并发与安全 (Concurrency & Safety)
*   **单机串行**: 对同一 Target:Port 的尝试必须是**串行**的。并发爆破极易导致服务锁定或崩溃。
*   **全局并发**: `BruteScanner` 可以同时处理多个不同的 Target (由上层 Runner 调度)。
*   **超时控制**: 每次 `Crack` 调用必须有严格的 Timeout (e.g. 3s)，防止挂死。

## 3. 实现细节

### 3.1 SSH Cracker
*   **库**: `golang.org/x/crypto/ssh`
*   **关键点**:
    *   `HostKeyCallback`: 必须设为 `ssh.InsecureIgnoreHostKey()`，否则无法连接未知主机。
    *   `Timeout`: 设置 Dial Timeout。
    *   **错误处理**: 区分 "认证失败" (继续尝试) 和 "网络超时/连接重置" (中断或重试)。

### 3.2 MySQL Cracker
*   **库**: `github.com/go-sql-driver/mysql`
*   **关键点**:
    *   DSN 格式: `user:pass@tcp(host:port)/dbname?timeout=3s`
    *   `Ping()`: 使用 `db.PingContext(ctx)` 验证连接。

### 3.3 Redis Cracker
*   **库**: `github.com/redis/go-redis/v9` (或 v8)
*   **关键点**:
    *   Redis 通常只有密码 (User 为空或 default)。
    *   Redis 6+ 支持 ACL (多用户)，需保留 User 扩展性。

## 4. 字典清单 (Top 10 示例)
我们将在 `dict.go` 中内置以下级别的字典：
*   **Users**: root, admin, test, user, guest, postgres, mysql, oracle
*   **Passwords**: 123456, password, admin, root, 12345678, 12345, 123123, qweasd

## 5. 开发步骤
1.  **Step 1**: 创建 `cracker.go` 和 `dict.go`，定义接口和内置字典。
2.  **Step 2**: 实现 `ssh_cracker.go`。
3.  **Step 3**: 完善 `scanner.go` 的调度逻辑，支持从 `Task.Params` 读取自定义字典。
4.  **Step 4**: 编写单元测试 `ssh_test.go` (需 Mock 或 Docker 环境)。

## 6. 决策点
请确认是否开始 **Step 1 & Step 2** (基础架构 + SSH 实现)？
