# Qscan Hydra (爆破模块) 代码分析报告

## 1. 架构概述

Qscan 的 `hydra` 包实现了一个多协议弱口令爆破框架。其核心思想是将"调度逻辑"与"具体的协议验证逻辑"解耦。

### 核心组件
1.  **Cracker (hydra/cracker.go)**: 爆破任务的主控者。
    *   管理并发池 (`pool.Pool`)。
    *   管理字典 (`AuthList`) 和目标信息 (`AuthInfo`)。
    *   负责任务分发 (`dispatcher`) 和结果收集 (`success`)。
2.  **Auth (hydra/type-auth.go)**: 定义了认证凭据模型 (Username, Password, Other)。
3.  **Protocol Impl (hydra/ssh, hydra/mysql...)**: 具体的协议验证函数，通常只暴露一个 `Check` 函数。

---

## 2. 优点与可借鉴之处 (Pros)

### 2.1 简洁的接口设计
每个协议的实现非常纯粹，只关注"验证一对账号密码是否正确"。
例如 `ssh.Check`：
```go
func Check(Host, Username, Password string, Port int) error
```
*   **借鉴**: 我们的 `Cracker` 接口也应如此设计，不要让具体的协议实现感知到 Task、Pipeline 等复杂上下文。

### 2.2 统一的调度模型
`Cracker.Run()` 统一处理了三种爆破模式：
*   `UsernameAndPassword`: 用户名+密码组合 (e.g. SSH, MySQL)。
*   `OnlyPassword`: 仅密码 (e.g. Redis, VNC)。
*   `UnauthorizedAccessVulnerability`: 未授权访问 (e.g. Telnet 无需认证)。
*   **借鉴**: 这种分类非常清晰，我们在设计 Task 参数时也应考虑这三种情况。

### 2.3 动态字典生成
`Auth.MakePassword()` 支持 `%user%` 占位符，可以在密码中使用用户名（例如检测 `root:root`）。
*   **借鉴**: 这是一个非常实用的功能，建议在我们的 `dict.go` 中实现。

### 2.4 协程池 (Worker Pool)
使用了 `pool.Pool` 来控制爆破的并发度。
*   **借鉴**: 爆破是 IO 密集型任务，协程池是必须的。

---

## 3. 缺点与改进空间 (Cons)

### 3.1 错误处理过于粗糙
在 `generateWorker` 中：
```go
if strings.Contains(err.Error(), "timeout") == true {
    continue
}
```
*   **问题**: 仅仅通过字符串匹配错误信息是不够健壮的。网络超时、连接重置、认证失败应该有明确的类型区分。
*   **改进**: 定义明确的 `ErrAuthFailed`, `ErrConnectionFailed`，根据错误类型决定是重试、跳过还是标记为成功。

### 3.2 硬编码的重试逻辑
`c.retries = 3` 是硬编码的。
*   **改进**: 应该作为配置项暴露给用户。

### 3.3 缺乏全局 QoS
虽然有 `Pool` 控制线程数，但没有针对单个目标的速率限制（Rate Limiting）。
*   **风险**: 如果对同一 IP 并发 50 个 SSH 连接，极易触发封禁。
*   **改进**: 引入我们已经实现的 `AdaptiveLimiter`，或者强制对同一 Target 串行爆破。

### 3.4 SSH 实现细节
在 `ssh.Check` 中：
```go
HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
    return nil
},
```
*   **评价**: 这是正确的做法（忽略 HostKey 检查），否则无法连接未知主机。我们将沿用此配置。

---

## 4. 对 NeoAgent 的具体建议

### 4.1 接口设计
建议定义如下接口：
```go
type Cracker interface {
    Name() string
    // 返回: success, error (仅网络错误/协议错误返回 error，认证失败返回 false, nil)
    Crack(ctx context.Context, target string, port int, user, pass string) (bool, error)
}
```

### 4.2 目录结构
```text
internal/core/scanner/brute/
├── scanner.go           # 调度器 (类似 Qscan 的 Cracker + Pool)
├── cracker.go           # 接口定义
├── dict.go              # 字典管理 (支持 %user% 替换)
├── ssh/ssh.go           # SSH 实现
├── mysql/mysql.go       # MySQL 实现
└── ...
```

### 4.3 字典加载
Qscan 的 `DefaultAuthMap` 硬编码了大量字典。
*   **建议**: NeoAgent 保持二进制轻量化，只内置 Top 100。主要依赖外部文件加载或运行时参数。

### 4.4 爆破逻辑
参考 Qscan 的 `dispatcher`，实现 User/Pass 的笛卡尔积组合，并支持 `StopOnSuccess`（爆破成功一个即停止）。

---

## 5. 结论
Qscan 的爆破模块结构清晰，逻辑简单直接，非常适合作为 NeoAgent 爆破模块的原型。
我们将吸取其"接口解耦"的优点，并改进其"错误处理"和"并发控制"的不足。
