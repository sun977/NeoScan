# OS 扫描模块 (OsScanner) 设计文档

## 1. 核心设计哲学
本模块采用 **"多引擎并发竞速 (Multi-Engine Racing)"** 架构。
操作系统识别本质上是一种"推断 (Inference)"过程，没有绝对的单一真理来源。因此，我们通过同时运行多个识别引擎，收集不同维度的证据（TTL、协议栈指纹、服务Banner等），最终选择置信度（Accuracy）最高的结果。

## 2. 模块文件结构与职责 (`internal/core/scanner/os`)

| 文件名 | 职责描述 | 平台限制 |
| :--- | :--- | :--- |
| **`os_scanner.go`** | **主控调度器**。定义了 `Scanner` 结构体和 `OsScanEngine` 接口。负责管理所有注册的引擎，根据扫描模式 (`fast`/`deep`/`auto`) 调度具体的引擎执行，并汇总最终结果。 | 通用 |
| **`ttl_engine.go`** | **TTL 估算引擎**。基于 ICMP Echo Reply 的 TTL 值进行粗略的 OS 类型推断（Windows/Linux/Network Device）。适用于所有平台，速度极快但精度较低。 | 通用 |
| **`nmap_engine.go`** | **Nmap 协议栈指纹引擎 (核心逻辑)**。实现了 `NmapStackEngine`。负责加载 Nmap OS 数据库，编排探测流程（寻找开放/关闭端口 -> 执行探测 -> 指纹匹配）。 | **Linux Only** (`//go:build linux`) |
| **`nmap_probes.go`** | **Nmap 探测包构建与解析**。封装了底层的发包逻辑。实现了 Nmap 第二代指纹识别所需的全部探测包构造 (SEQ, ECN, T2-T7, IE, U1) 以及响应包的解析与指纹行生成。 | **Linux Only** (`//go:build linux`) |
| **`service_engine.go`** | **服务推断引擎**。主动探测关键端口获取 Banner，通过正则匹配推断 OS。是 Windows 平台除 TTL 外的核心识别手段。 | 通用 |
| **`nmap_engine_others.go`** | **Nmap 引擎存根 (Stub)**。为非 Linux 平台（如 Windows/macOS）提供 `NmapStackEngine` 的空实现，确保代码可以跨平台编译。 | **Non-Linux** (`//go:build !linux`) |

## 3. 跨平台兼容性设计 (Cross-Platform Compatibility)

由于 **Nmap Stack Engine** 深度依赖 `Raw Socket` (原始套接字) 来发送畸形 TCP/IP 数据包，而 Windows 系统的 `Raw Socket` 支持受到严格限制，因此我们采用了**条件编译**策略：

- **Linux 环境**:
  - `NmapStackEngine` 具备完整功能，支持 Deep Scan。
  - `ServiceEngine` 作为补充，提升识别准确率（特别是针对 OpenSSH 等通用服务）。
- **Windows/其他环境**:
  - `NmapStackEngine` 自动降级，不可用。
  - 主要依赖 `ServiceEngine` (Banner 识别) 和 `TTLEngine`。

## 4. 当前实现能力 (Atomic Capability)

### 4.1 引擎详情

#### A. TTL Engine (`fast` mode)
- **原理**: 基于 ICMP TTL 值推断 (Windows <= 128, Linux <= 64)。
- **精度**: Accuracy ~80%。
- **特点**: 极快，无侵入。

#### B. Nmap Stack Engine (`deep` mode)
- **原理**: 完整复刻 Nmap OS Fingerprinting 2nd Generation。
- **匹配算法**: **加权评分制 (Weighted Scoring)**。将测试项打散为细粒度属性进行匹配，大幅提升了在畸形网络环境（如 Loopback）下的识别率。
- **数据源**: 内置 `nmap-os-db`。
- **输出**: 支持提取 OS Generation (e.g. "Linux 3.X|4.X")。

#### C. Service Inference Engine (`service_engine.go`)
- **原理**: "Sniper" 模式，主动探测 22, 80, 443 等端口。
- **匹配逻辑**:
  - 精确匹配: `Microsoft-IIS`, `Ubuntu`, `CentOS`, `FreeBSD` 等关键字。
  - 模糊匹配: `OpenSSH` + `Windows` -> Windows; `OpenSSH` -> Linux/Unix (置信度 85%)。
- **价值**: 在 Auto 模式下，通常能提供比 Nmap Stack 更直观的发行版信息（如 "Ubuntu" vs "Linux 3.X"）。

## 5. 差距分析与未来规划 (Gap Analysis & Roadmap)

### 5.1 为什么与 Nmap 官方结果仍有差距？

尽管我们拥有 Nmap 的指纹库，但在某些场景下（特别是冷门设备或复杂网络环境）识别效果仍不如 Nmap，原因如下：

1.  **探测包微操 (Packet Crafting Nuances)**:
    - Nmap 的探测包在 TCP 选项顺序、畸形校验和、标志位组合上经过了20年的打磨。
    - 我们的 Go `net` 库或 Raw Socket 实现可能在某些细微的协议字段上（如 IP ID 生成策略）与 Nmap 不完全一致，导致目标回包不同，进而导致指纹匹配失败。

2.  **权重算法 (Weighted Scoring)**:
    - Nmap 使用复杂的 `MatchPoints` 权重表，不同测试项的权重动态变化。
    - 我们目前采用了**平均加权算法**，虽然比“一票否决”强，但缺乏针对特定测试项（如 GCD, ISR）的精细化权重控制。

3.  **服务指纹库的完整性**:
    - 我们虽然引入了 `nmap-service-probes` 文件，但目前的 `ServiceEngine` 仅使用了硬编码的正则（为了轻量化）。
    - Nmap 拥有强大的服务指纹匹配引擎，能解析并执行复杂的探测指令（Probe/Match/Fallback）。这是我们目前最大的短板。

### 5.2 未来规划 (Roadmap)

#### Phase 3.5: 真正的服务扫描器 (Real PortServiceScanner)
- **目标**: 实现对 `nmap-service-probes` 的完整解析与执行。
- **内容**:
  - 实现 DSL 解析器，加载 Probe 和 Match 指令。
  - 实现基于状态机的探测调度（Null Probe -> Fallback Probes）。
  - 支持 `softmatch` 和 `hardmatch` 逻辑。

#### Phase 3.6: 算法精调
- **目标**: 引入 `MatchPoints` 权重逻辑。
- **内容**:
  - 解析 `nmap-os-db` 中的 `MatchPoints` 指令。
  - 在 `calculateScore` 中应用动态权重。

#### Phase 4.0: 自动化编排
- **目标**: `scan run`。
- **内容**: 自动串联 Alive -> Port -> Service -> OS 流程，让 Service Engine 的结果自动喂给 OS Engine 进行综合决策。

### 5.3 模块划分
由于OS模块不像PortService模块一样是一起运行的，所以不再执着于在 neoAgent scan run -t 192.168.1.1 -p 1-65535 -s 中添加-o进行os识别。而是在最终的流水线中统一实现原子扫描能力整合。保留os识别模块的原子性，只是在最终的结果中进行整合。
