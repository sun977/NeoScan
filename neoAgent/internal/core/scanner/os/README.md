# OS 扫描模块 (OsScanner) 设计文档

## 1. 核心设计哲学
本模块采用 **"多引擎并发竞速 (Multi-Engine Racing)"** 架构。
操作系统识别本质上是一种"推断 (Inference)"过程，没有绝对的单一真理来源。因此，我们通过同时运行多个识别引擎，收集不同维度的证据（TTL、协议栈指纹、服务Banner等），最终选择置信度（Accuracy）最高的结果。

## 2. 模块文件结构与职责 (`internal/core/scanner/os`)

| 文件名 | 职责描述 | 平台限制 |
| :--- | :--- | :--- |
| **`os_scanner.go`** | **主控调度器**。定义了 `Scanner` 结构体和 `OsScanEngine` 接口。负责管理所有注册的引擎，根据扫描模式 (`fast`/`deep`/`auto`) 调度具体的引擎执行，并汇总最终结果。 | 通用 |
| **`ttl_engine.go`** | **TTL 估算引擎**。基于 ICMP 响应的 TTL 值进行粗略的 OS 类型推断（Windows/Linux/Network Device）。适用于所有平台，速度极快但精度较低。 | 通用 |
| **`nmap_engine.go`** | **Nmap 协议栈指纹引擎 (核心逻辑)**。实现了 `NmapStackEngine`。负责加载 Nmap OS 数据库，编排探测流程（寻找开放/关闭端口 -> 执行探测 -> 指纹匹配）。 | **Linux Only** (`//go:build linux`) |
| **`nmap_probes.go`** | **Nmap 探测包构建与解析**。封装了底层的发包逻辑。实现了 Nmap 第二代指纹识别所需的全部探测包构造 (SEQ, ECN, T2-T7, IE, U1) 以及响应包的解析与指纹行生成。 | **Linux Only** (`//go:build linux`) |
| **`nmap_engine_others.go`** | **Nmap 引擎存根 (Stub)**。为非 Linux 平台（如 Windows/macOS）提供 `NmapStackEngine` 的空实现，确保代码可以跨平台编译，但调用时会返回错误或不支持提示。 | **Non-Linux** (`//go:build !linux`) |

## 3. 跨平台兼容性设计 (Cross-Platform Compatibility)

由于 **Nmap Stack Engine** 深度依赖 `Raw Socket` (原始套接字) 来发送畸形 TCP/IP 数据包 (如自定义 Flags、Options、Seq 等)，而 Windows 系统的 `Raw Socket` 支持受到严格限制（无法构建完整的 TCP 头），因此我们采用了**条件编译 (Conditional Compilation)** 策略：

- **Linux 环境**:
  - 编译 `nmap_engine.go` 和 `nmap_probes.go`。
  - `NmapStackEngine` 具备完整功能，支持 Deep Scan。
- **Windows/其他环境**:
  - 编译 `nmap_engine_others.go`。
  - `NmapStackEngine` 仅作为占位符存在，调用 `Scan` 方法会直接返回 "Not Supported" 错误。
  - `Scanner` 会自动降级，仅使用 `TTLEngine` (Fast Scan) 或未来实现的 `ServiceInferenceEngine`。

这种设计确保了 `neoAgent` 可以在 Windows 上编译和运行（用于常规扫描），同时在 Linux 上提供最强的 OS 识别能力。

## 4. 当前实现能力 (Atomic Capability)

### 4.1 引擎详情

#### A. TTL Engine (`fast` mode)
- **原理**: 基于 ICMP Echo Reply 的 TTL (Time To Live) 值进行粗略推断。
- **适用性**: Windows / Linux 通用。
- **映射规则**:
  - `TTL <= 64` -> Linux/Unix
  - `TTL <= 128` -> Windows
  - `TTL <= 255` -> Solaris/Network Device
- **优缺点**:
  - ✅ 速度极快，无侵入性，无需 Root 权限。
  - ❌ 准确度较低 (Accuracy ~80%)。

#### B. Nmap Stack Engine (`deep` mode)
- **原理**: 完整实现了 Nmap OS Fingerprinting 2nd Generation 标准。通过发送一系列精心构造的畸形数据包，分析目标主机的 TCP/IP 协议栈响应特征。
- **适用性**: **仅 Linux Agent 可用** (依赖 Raw Socket)。
- **探测能力**:
  - **SEQ**: 6个 TCP SYN 包，分析序列号生成算法 (GCD, ISR, SP) 和 TCP 选项。
  - **ECN**: 分析对显式拥塞通知的支持。
  - **T2-T7**: 6个 TCP 包 (包含不同 Flag 和 Option 组合)，探测对协议规范的边缘实现。
  - **IE**: ICMP Echo 探测。
  - **U1**: UDP 探测 (针对关闭端口)。
- **依赖**:
  - `internal/core/lib/network/netraw`: 提供底层 Raw Socket 发包与抓包能力。
  - `internal/pkg/fingerprint/engines/nmap`: 提供 OS DB 解析与指纹匹配算法。

## 5. 使用方式

### 5.1 代码调用
```go
scanner := os.NewScanner()
// 自动选择策略（推荐）
// Linux: 并发运行 TTL + Nmap Stack
// Windows: 仅运行 TTL
info, err := scanner.Scan(ctx, "192.168.1.1", "auto")
```

### 5.2 模式说明
- **`fast`**: 仅运行 TTL Engine。适用于大规模内网资产粗筛。
- **`deep`**: 强制运行 Nmap Stack Engine (Linux) 或其他高精度引擎。
- **`auto`**: 智能混合模式。

## 6. 未来扩展规划 (Roadmap)

- [x] **阶段一：Raw Socket 基础设施** (Linux 实现完成)
- [x] **阶段二：Nmap Stack Engine** (全量探测 T1-T7/IE/ECN 实现完成)
- [ ] **阶段三：Service Inference Engine (服务推断)**
  - 目标: 利用端口扫描获取的 Service Banner (如 `Server: Microsoft-IIS/10.0`) 直接推断 OS。
  - 价值: 弥补 Windows Agent 无法使用 Raw Socket 进行精确识别的短板。
