# Port Service Scanner (端口服务扫描器)

## 模块概述
本模块 (`internal/core/scanner/port`) 实现了基于 TCP Connect 的端口扫描以及兼容 Nmap `nmap-service-probes` 格式的服务指纹识别。它不依赖外部 Nmap 二进制文件，而是通过内置的解析器和执行引擎直接处理 Nmap 规则。

## 核心组件

### 1. 扫描器 (`port_service_scanner.go`)
- **PortServiceScanner**: 实现了 `runner.Runner` 接口。
- **并发模型**: 使用 Semaphore 控制并发度，支持 Context 取消。
- **网络层**: 集成 `internal/core/lib/network/dialer`，透明支持 SOCKS5 代理。

### 2. Nmap 规则解析 (`parser.go`, `rules.go`)
- **规则源**: 通过 `go:embed` 嵌入 `nmap-service-probes` 文件，实现零依赖部署。
- **解析器**: 自研的流式解析器，支持 Nmap 核心指令。

### 3. 数据模型 (`types.go`)
- **Probe**: 定义探测包结构，包含协议、探测字符串、适用端口等。
- **Match**: 定义正则匹配规则，包含服务名、版本提取模式等。
- **FingerPrint**: 结构化的指纹识别结果 (CPE, OS, Version 等)。

## Nmap 逻辑实现细节

### 支持的指令集
本模块目前支持以下 Nmap 指令：
- `Probe <protocol> <probename> <probestring>`: 定义探测包。
- `match <service> <pattern> <versioninfo>`: 定义硬匹配规则。
- `softmatch <service> <pattern>`: 定义软匹配规则 (用于加速后续匹配)。
- `ports <portlist>`: 定义探针适用的端口。
- `sslports <portlist>`: 定义探针适用的 SSL 端口。
- `rarity <n>`: 定义探针的稀有度 (用于优化扫描速度)。
- `fallback <probename>`: 定义回退探针。

### 服务识别流程
当 TCP 连接建立成功后，进入服务识别阶段 (`scanPort` -> `executeProbe`):

1. **探针选择 (`getProbesForPort`)**:
   - 优先选择与当前端口 (`ports`/`sslports`) 显式关联的探针。
   - 补充低稀有度 (`rarity <= 7`) 的通用探针。
   - 对探针列表进行排序和去重。

2. **交互探测**:
   - **Send**: 解析 `ProbeString` 中的转义字符 (如 `\r\n`, `\x00`) 并发送。
   - **Receive**: 读取响应数据 (默认前 4KB)。
   - **Match**: 
     - 遍历该 Probe 关联的 `MatchGroup`。
     - 使用正则 (`regexp`) 匹配响应数据。
     - 如果匹配成功，解析 `VersionInfo` 提取详细信息 (Service, Product, Version, OS, CPE 等)。

## 限制与不足
- **脚本引擎**: 不支持 Nmap NSE (Lua 脚本)。
- **复杂逻辑**: 部分 Nmap 的复杂指令 (如 `helper`) 尚未实现。
- **UDP**: 目前仅专注于 TCP 协议的服务识别。
- **性能瓶颈**: 使用 TCP Connect (全连接) 扫描，在高并发下容易丢包或被防火墙拦截。建议在全端口扫描时降低并发速率 (e.g., `-r 100`)。

## 性能调优与优化建议 (Performance Tuning)

### 当前问题
当前实现使用 `net.Dial` 进行 TCP 全连接扫描。虽然实现简单且跨平台兼容性好，但在高并发场景下存在以下问题：
1.  **资源消耗大**: 每个端口都需要完成三次握手，占用大量文件描述符和系统资源。
2.  **易被拦截**: 全连接行为特征明显，容易触发 IDS/IPS 或防火墙的速率限制。
3.  **超时敏感**: 默认超时 (2s) 对于网络状况不佳的目标可能过短，导致漏报。

### 优化建议 (Roadmap)

#### 短期优化 (Phase 3.x)
1.  **默认参数调整**: 将默认并发数 (`-r`) 从 1000 降低到 500 或更低，提高扫描稳定性。
2.  **动态超时**: 实现自适应超时机制，根据目标主机的 RTT 动态调整 Connect Timeout。
3.  **重试机制**: 对 Timeout 的端口增加一次重试，减少误报。

#### 长期演进 (Phase 4+)
1.  **SYN Scan (半连接)**: 引入 Raw Socket 实现 TCP SYN 扫描。
    - **优势**: 速度极快，不建立完整连接，隐蔽性高。
    - **挑战**: 需要 Root/Admin 权限，且需处理不同 OS 的 Raw Socket 差异。
2.  **Masscan 集成**: 考虑集成 Masscan 的异步无状态扫描算法，实现百万级 PPS 的扫描速度。

## 未来规划 (Refactoring Roadmap)

为了提升架构的整洁度和复用性，计划进行以下重构 (Phase 3.6+):

### 1. 引擎下沉 (Engine Sinking)
- **目标**: 将 Nmap 解析与匹配逻辑从 `scanner/port` 剥离。
- **行动**: 迁移 `parser.go` 和 `types.go` 到 `internal/pkg/fingerprint/engines/nmap`。

### 2. 接口标准化
- **目标**: 实现通用的指纹识别接口。
- **行动**: 封装 `NmapEngine` 实现 `MatchEngine` 接口：
  ```go
  type NmapEngine struct {
      // ...
  }
  func (e *NmapEngine) Match(ctx context.Context, conn net.Conn) (*Fingerprint, error)
  ```

### 3. 混合指纹管理
- **目标**: 支持多种指纹源的混合匹配。
- **行动**: `PortServiceScanner` 将演变为调度器，协调 `NmapEngine` (协议交互) 和 `WebEngine` (HTTP 特征) 等多种引擎协同工作。
