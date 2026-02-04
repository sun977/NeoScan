# RDP Protocol Stack (Ported from sbscan)

## 简介
本模块是 RDP (Remote Desktop Protocol) 协议栈的纯 Go 语言实现，专为 NeoAgent 的 RDP 弱口令爆破功能服务。
它移植自 [SweetBabyScan (sbscan)](https://github.com/inbug-team/SweetBabyScan)，并在其基础上进行了精简和优化，移除了所有与图形界面渲染、外设重定向相关的代码，仅保留核心的连接建立和身份认证逻辑。

## 核心能力
*   **纯 Go 实现**: 无需 CGO，跨平台编译友好。
*   **NLA 支持**: 完整支持 NLA (Network Level Authentication) 认证流程，包含 CredSSP 和 NTLMv2 协议。
*   **Standard RDP 支持**: 支持老旧系统的 Standard RDP Security 认证。
*   **高性能**: 针对爆破场景优化，去除了位图解压、RLE 解码等高 CPU 消耗的无关逻辑。

## 目录结构
*   `grdp.go`: 核心入口，提供 `Login(host, domain, user, pwd)` 接口。
*   `core/`: 底层网络 IO 和 Socket 封装。
*   `protocol/`: RDP 协议分层实现。
    *   `tpkt/`: TPKT (ISO Transport Service on top of TCP) 层。
    *   `x224/`: X.224 (Connection Request/Confirm) 层。
    *   `mcs/`: MCS (Multipoint Communication Service) 层 (T.125)。
    *   `sec/`: RDP Security 层 (处理加密和签名)。
    *   `nla/`: NLA 认证层 (CredSSP, NTLMv2)。
    *   `pdu/`: PDU (Protocol Data Unit) 消息定义与序列化。
    *   `lic/`: 许可证协议 (License Protocol) 存根。
*   `emission/`: 事件发射器，用于处理异步协议交互。

## 移植与修改说明
为了适应 NeoAgent 的架构并极致优化体积与性能，我们对原版代码进行了以下修改：

1.  **移除图形化逻辑**:
    *   删除了 `rle.go` (Run-Length Encoding 解码)。
    *   删除了 `bitmap.go` (位图处理)。
    *   删除了 `orders.go` (绘图指令)。
    *   删除了 `rfb/` 目录 (VNC 相关)。
2.  **移除外设重定向**:
    *   删除了 `cliprdr.go` (剪贴板重定向)。
    *   删除了 `rdpsnd.go` (音频重定向)。
3.  **依赖清理**:
    *   移除了对第三方日志库的复杂依赖，使用内置的轻量级 `glog`。
4.  **Bug 修复**:
    *   修复了部分 PDU 结构体定义缺失的问题 (`data.go`)。
    *   补充了缺失的 `emission` 库实现。

## 使用方法
本模块不直接对外暴露，而是通过 `internal/core/scanner/brute/protocol/rdp.go` 中的 `RDPCracker` 进行封装和调用。

```go
// 调用示例
err := grdp.Login("192.168.1.1:3389", "", "administrator", "password")
if err == nil {
    // 认证成功
} else {
    // 认证失败或连接错误
}
```
