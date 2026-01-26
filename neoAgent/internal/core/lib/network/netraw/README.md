# NetRaw - Raw Socket & Packet Crafting Library

## 模块概述
`netraw` 是 NeoAgent 的底层网络库，提供跨平台的 **Raw Socket (原始套接字)** 操作接口和 **Packet Crafting (数据包构建)** 工具。

它的核心目标是支持高级网络扫描任务（如 OS 协议栈指纹识别、TCP SYN 扫描、防火墙探测等），这些任务通常需要绕过操作系统的 TCP/IP 协议栈，直接发送自定义构造的 IP/TCP 数据包。

## 目录结构
```text
netraw/
├── socket_linux.go    # Linux 平台 Raw Socket 实现 (AF_INET, SOCK_RAW)
├── socket_windows.go  # Windows 平台 Stub (占位符，返回 Not Supported)
├── packet_builder.go  # IP/TCP 数据包构建器与校验和计算
└── README.md          # 本文档
```

## 功能特性

### 1. Raw Socket 操作
提供统一的 `RawSocket` 结构体和接口，但在不同平台上表现不同：
- **Linux**: 完整支持。
  - 使用 `syscall.Socket(AF_INET, SOCK_RAW, protocol)` 创建。
  - 自动开启 `IP_HDRINCL`，允许用户自定义构建 IP 头部。
  - 支持 `Send` (Sendto) 和 `Receive` (Recvfrom)。
  - 支持 `BindToInterface` 绑定特定网卡。
- **Windows**: **不支持**。
  - 由于 Windows Winsock2 对 Raw Socket 的限制（无法发送 TCP 数据包），本模块在 Windows 下所有方法均返回 "not supported" 错误。
  - Windows Agent 应降级使用应用层扫描（如 Connect Scan）。

### 2. 数据包构建 (Packet Crafting)
提供构建标准 TCP/IP 数据包的工具函数：
- **`BuildIPv4Packet`**: 构建包含 IPv4 头部的完整数据包。
- **`BuildTCPHeader`**: 构建 TCP 头部（含 Checksum 计算位）。
- **`Checksum`**: 通用的互联网校验和算法 (RFC 1071)。

## 使用示例 (Linux Only)

```go
// 1. 创建 TCP Raw Socket
socket, err := netraw.NewRawSocket(syscall.IPPROTO_TCP)
if err != nil {
    log.Fatal(err)
}
defer socket.Close()

// 2. 构建 TCP SYN 包
// ... (构建 payload) ...
packet, err := netraw.BuildIPv4Packet(srcIP, dstIP, syscall.IPPROTO_TCP, payload)

// 3. 发送
err = socket.Send(dstIP, packet)

// 4. 接收响应
buf := make([]byte, 1500)
n, from, err := socket.Receive(buf, 2*time.Second)
```

## 设计哲学
- **Don't fight the OS**: 不强行在 Windows 上通过 WinPcap/Npcap 等驱动实现 Raw Socket，避免引入复杂的外部依赖和不稳定性。
- **Native First**: 优先使用 Go 原生 syscall 实现，零 CGO 依赖。
- **KISS**: 仅提供最基础的 Send/Receive 和 Packet Build，不包含复杂的协议状态机。
