# OS 扫描模块 (OsScanner) 设计文档

## 1. 核心设计哲学
本模块采用 **"多引擎并发竞速 (Multi-Engine Racing)"** 架构。
操作系统识别本质上是一种"推断 (Inference)"过程，没有绝对的单一真理来源。因此，我们通过同时运行多个识别引擎，收集不同维度的证据（TTL、协议栈指纹、服务Banner等），最终选择置信度（Accuracy）最高的结果。

## 2. 当前实现能力 (Atomic Capability)

### 2.1 架构组件
- **Scanner**: 主控调度器，负责并发执行 Engine，并汇总结果。
- **OsScanEngine (Interface)**: 统一的引擎接口，任何识别逻辑只需实现 `Scan(ctx, target)`。
- **Mode**: 扫描模式控制，支持 `fast`, `deep`, `auto`。

### 2.2 已实现引擎

#### A. TTL Engine (`fast` mode)
- **原理**: 基于 ICMP Echo Reply 的 TTL (Time To Live) 值进行粗略推断。
- **实现**: 复用系统 `ping` 命令（为了兼容性和避免 Raw Socket 权限问题）。
- **映射规则**:
  - `TTL <= 64` -> Linux/Unix
  - `TTL <= 128` -> Windows
  - `TTL <= 255` -> Solaris/Network Device
- **优缺点**:
  - ✅ 速度极快，无侵入性。
  - ✅ 不需要 Root 权限。
  - ❌ 准确度较低 (Accuracy ~80%)，容易受中间跳数影响。

#### B. Nmap Stack Engine (`deep` mode - Stub)
- **原理**: 基于 Nmap 的 TCP/IP 协议栈指纹识别技术 (OS Fingerprinting 2nd Generation)。
- **现状**: **占位符 (Stub)**。
- **限制**: **仅 Linux 系统支持**。
  - Windows 系统对 Raw Socket 支持有限，无法精确控制 TCP 标志位和 IP 头部，因此不支持 Deep 模式的 SYN/Stack 扫描。
- **原因**: 完整的协议栈识别需要发送畸形包 (T1-T7, ECN, IE)，必须依赖 **Raw Socket**。在 NeoAgent 的底层网络库 (internal/core/lib/network) 完成 Raw Socket 封装前，无法实现。

## 3. 使用方式

### 3.1 代码调用
```go
scanner := os.NewScanner()
// 自动选择策略（推荐）
info, err := scanner.Scan(ctx, "192.168.1.1", "auto")
```

### 3.2 参数说明
- **`fast`**: 仅运行 TTL Engine。适用于大规模内网资产粗筛。**Windows/Linux 通用**。
- **`deep`**: 运行所有高精度引擎。**仅限 Linux Agent 使用**。在 Windows 上调用会报错或降级。
- **`auto`**: 混合模式。
  - Linux: 并发运行 TTL + Nmap Stack。
  - Windows: 仅运行 TTL (未来可加入 Service Inference)。

## 4. 未来扩展规划 (Roadmap)

### 4.1 阶段一：Raw Socket 基础设施建设 (Linux Only)
- **目标**: 在 `internal/core/lib/network` 中实现 Linux 平台的 Raw Socket 发包与抓包能力。
- **说明**: 放弃在 Windows 上实现 Raw Socket (因系统限制)，Windows 端的 Deep Scan 将依赖应用层指纹 (Service Inference)。

### 4.2 阶段二：Nmap Stack Engine 实现 (Linux Only)
- **目标**: 解析 `nmap-os-db` 指纹库，实现 T1-T7 等探测包的构造与响应分析。
- **集成**: 仅在 Linux 编译版本中启用此引擎。

### 4.3 阶段三：Service Inference Engine (服务推断)
- **目标**: 利用 `PortServiceScanner` 的成果。
- **逻辑**: 如果端口扫描发现 `Server: Microsoft-IIS/10.0`，则直接推断 OS 为 `Windows Server 2016+`。
- **价值**: 这是最"实惠"的识别方式，不需要额外发包，直接复用端口扫描结果。

### 4.4 与 PortServiceScanner 的协同
- 未来 `PortServiceScanner` 增加 `-o` 参数时，底层可以直接调用 `OsScanner` 的引擎，实现能力复用，避免重复造轮子。
