# Dialer - 统一网络连接库

## 模块概述
`dialer` 是 NeoAgent 的**统一网络连接层**。
它封装了 Go 原生的 `net.Dialer`，提供了**全局的超时控制**、**代理支持 (SOCKS5)** 和**连接复用**能力。

所有上层扫描模块（如 PortServiceScanner, VulnScanner 等）都**必须**通过本模块发起 TCP/UDP 连接，严禁私自调用 `net.Dial`。

## 目录结构
```text
dialer/
├── dialer.go  # 核心实现 (NewDialer, Dial, DialContext)
├── proxy.go   # SOCKS5 代理支持逻辑
├── global.go  # 全局单例管理 (InitGlobalDialer, GlobalDialer)
└── README.md  # 本文档
```

## 核心功能

### 1. 统一的超时控制
- 提供默认连接超时时间 (DefaultTimeout)。
- 支持通过 `Dialer` 结构体配置自定义超时。

### 2. 透明代理支持 (Proxy)
- 内置 SOCKS5 代理客户端。
- 如果配置了 Proxy 地址，所有通过该 Dialer 发起的连接都会自动走代理。
- 对上层业务透明，上层无需感知代理的存在。

### 3. 全局单例 (Global Instance)
- 提供 `GlobalDialer`，方便整个应用共享同一个连接配置（如全局代理设置）。
- 通过 `dialer.InitGlobalDialer(proxyUrl, timeout)` 初始化。

## 使用示例

### 场景一：使用全局 Dialer (推荐)
```go
// 1. 初始化 (通常在 main.go 或 root command 中)
// 启用 SOCKS5 代理，超时 2 秒
dialer.InitGlobalDialer("socks5://127.0.0.1:1080", 2*time.Second)

// 2. 发起连接
conn, err := dialer.Dial("tcp", "192.168.1.1:80")
if err != nil {
    // 处理错误
}
defer conn.Close()
```

### 场景二：创建独立的 Dialer
```go
// 创建一个不走代理、超时 500ms 的专用 Dialer
d := dialer.NewDialer("", 500*time.Millisecond)
conn, err := d.Dial("tcp", "10.0.0.1:22")
```

## 设计原则
- **Centralized Control**: 所有出站流量的策略（代理、超时）集中管理。
- **Drop-in Replacement**: 接口设计尽量与 `net.Dialer` 保持一致，方便迁移。
