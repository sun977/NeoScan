# Global Scanner Factory

`neoAgent/internal/core/factory`

本包作为 Agent 核心能力的统一构建工厂，旨在解决 "能力的构建" 与 "能力的使用" 分离的问题，确保 Agent 在 CLI 模式、Pipeline 编排模式以及 Server 集群模式下获得一致的扫描能力。

## 1. 设计目标

*   **Single Source of Truth**: 所有的 Scanner 实例化逻辑收敛于此。
*   **Consistency**: 确保所有入口（RunnerManager, AutoRunner）获取的 Scanner 实例配置一致（如默认超时、限流策略、注册的插件）。
*   **Decoupling**: 使用者无需关心 Scanner 的具体初始化细节（如依赖注入、配置解析）。

## 2. 工厂方法

### 2.1 Brute Scanner Factory
`brute_factory.go`

*   `NewFullBruteScanner() *brute.BruteScanner`
    *   创建一个注册了所有支持协议（SSH, RDP, MySQL, Redis, SMB 等 15+ 种）的完整爆破扫描器。
    *   预配置了全局限流器 (QoS)。

### 2.2 Alive Scanner Factory
`alive_factory.go`

*   `NewAliveScanner() *alive.IpAliveScanner`
    *   创建一个 IP 存活扫描器。
    *   配置了默认的 RTT 估算器和自适应限流器。

### 2.3 Port Scanner Factory
`port_factory.go`

*   `NewPortScanner() *port_service.PortServiceScanner`
    *   创建一个端口服务扫描器（支持 TCP Connect 和 Gonmap 指纹识别）。
    *   配置了默认的 Embed 规则库加载逻辑。

### 2.4 OS Scanner Factory
`os_factory.go`

*   `NewOsScanner() *os.Scanner`
    *   创建一个操作系统扫描器。
    *   自动注册所有可用引擎（TTL, Nmap Stack, Service Banner）。

## 3. 使用示例

### 在 RunnerManager 中使用 (Server/CLI Atom Mode)

```go
func NewRunnerManager() *RunnerManager {
    m := &RunnerManager{...}

    // 统一使用工厂创建
    m.Register(factory.NewFullBruteScanner())
    m.Register(factory.NewAliveScanner())
    m.Register(factory.NewPortScanner())
    // ...
    return m
}
```

### 在 AutoRunner 中使用 (Pipeline Mode)

```go
func NewAutoRunner(...) *AutoRunner {
    return &AutoRunner{
        aliveScanner: factory.NewAliveScanner(),
        portScanner:  factory.NewPortScanner(),
        osScanner:    factory.NewOsScanner(),
        // ...
    }
}
```

## 4. 扩展指南

当添加新的 Scanner 类型（如 WebScanner）时：
1.  在 `factory` 包下新建 `web_factory.go`。
2.  实现 `NewWebScanner()` 方法，完成所有必要的初始化配置。
3.  在 `RunnerManager` 和 `AutoRunner` 中替换直接实例化的代码。
