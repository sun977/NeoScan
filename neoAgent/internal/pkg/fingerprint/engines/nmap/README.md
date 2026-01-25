# Nmap Fingerprint Engine (Nmap 指纹识别引擎)

## 模块概述
本模块 (`internal/pkg/fingerprint/engines/nmap`) 实现了基于 Nmap 规则库的服务指纹识别引擎。它能够解析官方的 `nmap-service-probes` 格式规则，并执行交互式探测（Probe-Match）流程，从而识别目标端口运行的具体服务、版本、OS 和设备类型。

此模块设计为**零依赖**，不需要系统中安装 Nmap 二进制文件，所有逻辑均由 Go 原生实现。

## 核心组件

### 1. 引擎 (`engine.go`)
- **NmapEngine**: 核心控制器，负责管理探针列表、端口索引，并执行扫描任务。
- **并发安全**: 支持并发调用 `Scan` 方法。
- **网络层**: 集成 `internal/core/lib/network/dialer`，支持 SOCKS5 代理。

### 2. 解析器 (`parser.go`)
- **功能**: 解析 Nmap 格式的规则文件。
- **支持指令**: `Probe`, `match`, `softmatch`, `ports`, `sslports`, `rarity`, `fallback`。

### 3. 数据模型 (`types.go`)
- **Probe**: 探测包定义。
- **Match**: 正则匹配规则。
- **FingerPrint**: 识别结果结构体 (CPE, Product, Version 等)。

### 4. 规则数据 (`rules.go` & `embed`)
- **nmap-service-probes**: 官方 Nmap 规则库 (静态 Embed)。
- **nmap-custom-probes**: 用户自定义扩展规则 (静态 Embed)。

## 规则管理与架构决策

### 架构设计：静态基础 + 动态扩展
本模块采用“静态保底，动态增强”的混合架构设计。

#### 1. 静态基础 (Immutable Infrastructure)
我们将规则文件视为 Agent 的**编译期依赖**，而非运行时配置。
- **实现方式**: 使用 `//go:embed` 将规则文件打包进二进制。
- **文件位置**:
    - `nmap-service-probes`: 官方规则，保证基础识别能力。
    - `nmap-custom-probes`: 内部自定义规则，用于补充私有协议或新漏洞指纹。
- **优势**:
    - **零外部依赖**: Agent 单文件部署，无需携带辅助文件。
    - **启动速度快**: 内存直接加载，无 IO 开销。
    - **版本一致性**: 避免因环境差异导致规则文件版本不一致。

#### 2. 关于 `go:embed` 的路径限制
`go:embed` 不支持使用 `..` 回溯路径。因此，我们必须将规则文件**放置在 Go 包目录下**。
- **工程实践**:
    - 规则源文件维护在 `rules/fingerprint/nmap/` 目录。
    - 构建或开发时，需将规则文件**复制**到 `internal/pkg/fingerprint/engines/nmap/` 目录下。
    - 代码中只 Embed 当前目录下的副本。

#### 3. 动态扩展 (Runtime Extension)
虽然基础规则是静态编译的，但引擎支持运行时热更新。
- **接口**: `engine.SetRules(content string)`
- **场景**: Master 下发紧急漏洞指纹规则。
- **逻辑**: 引擎会将 Master 下发的规则与内置规则合并（或覆盖），实现能力的动态升级。

## 自定义规则开发指南
请在 `nmap-custom-probes` 文件中添加自定义规则。格式完全遵循 Nmap 标准：

```bash
# 示例：自定义 SMB 探测
Probe TCP SMB_NEGOTIATE q|\x00\x00\x00\xc0...|
rarity 1
ports 445
match microsoft-ds m|^\0\0...SMB.*|s p/Microsoft DS/
```

## 测试
- `engine_test.go` 包含完整的单元测试，覆盖了规则加载、解析、模拟服务扫描等场景。
- 运行测试前，请确保当前目录下存在规则文件副本。
