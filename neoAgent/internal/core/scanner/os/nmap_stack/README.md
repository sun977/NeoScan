# Nmap Stack OS Fingerprinting 模块

## 1. 概述
本模块命名为 nmap_stack，是因为是基于 TCP/IP 协议栈的指纹识别。
本模块实现了基于 TCP/IP 协议栈指纹的操作系统识别功能，其原理与 Nmap 的 `-O` 选项一致。通过向目标主机发送一系列精心构造的数据包（包括 TCP、UDP 和 ICMP），并分析目标主机的响应特征（如 TCP 窗口大小、TTL、TCP 选项顺序等），来推断目标操作系统的类型和版本。

由于该技术依赖于 Raw Socket (原始套接字) 来构造和接收自定义数据包，因此**仅支持 Linux 平台**且需要 **Root 权限**运行。

## 2. 核心原理
该模块遵循 Nmap 第二代 OS 识别算法，主要包含以下探测步骤：

1.  **SEQ (Sequence Generation)**: 发送 6 个 TCP SYN 包到开放端口，分析序列号生成算法 (GCD, ISR)、TCP 选项 (MSS, WScale, TS, SACK) 等。
2.  **OPS (TCP Options)**: 分析 SYN/ACK 包中的 TCP 选项顺序和值。
3.  **WIN (TCP Window)**: 分析 TCP 窗口大小的变化模式。
4.  **ECN (Explicit Congestion Notification)**: 发送设置了 ECN 标志位的 SYN 包，检测目标对拥塞控制的支持。
5.  **T1-T7 (TCP Probes)**:
    *   T1: 发送 SYN 包到开放端口（也是 SEQ 测试的一部分）。
    *   T2: 发送 NULL 包（无标志位）到开放端口。
    *   T3: 发送 SYN|FIN|URG|PSH 包到开放端口。
    *   T4: 发送 ACK 包到开放端口。
    *   T5: 发送 SYN 包到**关闭**端口。
    *   T6: 发送 ACK 包到**关闭**端口。
    *   T7: 发送 FIN|PSH|URG 包到**关闭**端口。
6.  **IE (ICMP Echo)**: 发送两个 ICMP Echo Request 包，检测 ICMP 响应特征。
7.  **U1 (UDP)**: 发送 UDP 包到关闭端口，期望收到 ICMP Port Unreachable 响应。

## 3. 代码结构

*   **`engine.go`**:
    *   `NmapStackEngine`: 核心引擎结构体，负责协调整个扫描流程。
    *   `Scan()`: 主入口，执行以下步骤：
        1.  检查运行环境（必须是 Linux）。
        2.  寻找一个开放端口（Open Port）和一个关闭端口（Closed Port）。
        3.  调用 `executeProbes` 执行全量探测。
        4.  调用 `db.Match` 将收集到的指纹与数据库进行匹配。
    *   `executeProbes()`: 构造并发送所有探测包，同时启动 `receiverLoop` 接收响应。
    *   `receiverLoop()`: 监听 TCP/UDP/ICMP 原始套接字，捕获目标返回的数据包。

*   **`probes.go`**:
    *   `buildAllProbes()`: 构造上述 1-7 类探测包的具体实现。
    *   `generateFingerprint()`: 将收集到的响应数据（`ProbeResponse`）解析并转换为 Nmap 格式的指纹字符串（例如 `T1(R=Y%DF=Y%W=...)`）。
    *   `parseTCPResponse()`: 解析 TCP 响应头，提取 R, DF, T, W, S, A, F, O 等关键字段。

*   **`matcher.go`**:
    *   `OSDB`: 存储从 `nmap-os-db` 文件加载的指纹库。
    *   `Match()`: 在数据库中查找最佳匹配的指纹。
    *   `calculateScore()`: 计算目标指纹与数据库指纹的匹配分数。采用 Nmap 的加权算法，对比每一项测试的属性值。
    *   `matchValue()`: 处理具体的数值比较逻辑，支持范围 (`10-20`)、逻辑或 (`|`)、大于小于 (`>10`) 等操作符。

*   **`parser.go`**:
    *   `ParseOSDB()`: 解析 `nmap-os-db` 文本文件，将其加载到内存结构 `OSDB` 中。
    *   `OSFingerprint`: 定义了单条指纹的数据结构，包含 Vendor, OSFamily, OSGen 等信息。

*   **`rules.go`**:
    *   使用 `embed` 包将 `nmap-os-db` 数据库文件打包进二进制文件中，确保运行时无外部依赖。

*   **`engine_others.go`**:
    *   非 Linux 平台的存根（Stub）实现，直接返回不支持错误。

## 4. 依赖
*   **`neoagent/internal/core/lib/network/netraw`**: 这是一个封装了底层 Raw Socket 操作的库，用于发送和接收自定义的 IP/TCP/UDP/ICMP 数据包。

## 5. 使用限制
1.  **权限要求**: 必须以 `root` 用户运行，因为 Raw Socket 需要 `CAP_NET_RAW` 权限。
2.  **平台限制**: 目前仅实现了 Linux 版本的 Raw Socket 逻辑。Windows 平台由于对 Raw Socket 支持受限（不支持发送 TCP 包），无法使用此模块。
3.  **网络环境**:
    *   需要在目标主机上找到至少一个**开放的 TCP 端口**。如果目标防火墙全拦，无法进行准确识别。
    *   需要在目标主机上找到一个**关闭的 TCP 端口**。

## 6. 与 Nmap 的对比
*   **实现程度**: 实现了 Nmap OS 探测的核心子集，能够兼容 Nmap 的指纹库。
*   **差异**:
    *   简化了部分复杂的时序分析（如 SEQ 测试中的精确 RTT 计算）。
    *   未实现重传机制（目前假设网络状况良好）。
    *   未实现模糊匹配（Guess）的高级启发式算法，目前仅做精确和次精确匹配。
