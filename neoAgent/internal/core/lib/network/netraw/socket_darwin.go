//go:build darwin

package netraw

import (
	"fmt"
	"net"
	"syscall"
	"time"
)

// RawSocket 封装 Darwin (macOS) 下的 Raw Socket 操作
// 注意：必须使用 sudo 运行 (Root 权限)
type RawSocket struct {
	fd       int
	protocol int
}

// NewRawSocket 创建一个新的 Raw Socket
// protocol: 协议号 (e.g., syscall.IPPROTO_TCP, syscall.IPPROTO_ICMP)
func NewRawSocket(protocol int) (*RawSocket, error) {
	// 创建 Raw Socket
	// AF_INET: IPv4
	// SOCK_RAW: 原始套接字
	// protocol: 协议类型
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, protocol)
	if err != nil {
		// 检查是否是因为权限不足
		if err == syscall.EPERM || err == syscall.EACCES {
			return nil, fmt.Errorf("permission denied: raw socket requires root privileges (sudo)")
		}
		return nil, fmt.Errorf("failed to create raw socket: %v", err)
	}

	// 设置 IP_HDRINCL，告诉内核我们自己构建 IP 头部
	// macOS 也支持这个选项
	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1)
	if err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to set IP_HDRINCL: %v", err)
	}

	return &RawSocket{
		fd:       fd,
		protocol: protocol,
	}, nil
}

// Close 关闭 Socket
func (s *RawSocket) Close() error {
	return syscall.Close(s.fd)
}

// Send 发送数据包
// dst: 目标 IP 地址
// packet: 完整的 IP 数据包 (含 IP 头)
func (s *RawSocket) Send(dst net.IP, packet []byte) error {
	// 转换目标 IP 为 [4]byte
	dst4 := dst.To4()
	if dst4 == nil {
		return fmt.Errorf("destination must be IPv4")
	}

	addr := syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{dst4[0], dst4[1], dst4[2], dst4[3]},
	}

	// 在 macOS 上，Sendto 与 Linux 略有不同，但 Go 的 syscall 封装处理了大部分差异。
	// 关键差异点：BSD 派系通常要求 IP 头部长度字段为主机字节序，
	// 但如果开启了 IP_HDRINCL，这通常由用户空间负责，或者内核会自动处理。
	// 现代 macOS 内核通常期望 IP 头是网络字节序（Big Endian），与 Linux 一致。

	err := syscall.Sendto(s.fd, packet, 0, &addr)
	if err != nil {
		return fmt.Errorf("sendto failed: %v", err)
	}
	return nil
}

// Receive 接收数据包
// buffer: 用于接收数据的缓冲区
// timeout: 超时时间
// 返回: 读取的字节数, 来源 IP, 错误
func (s *RawSocket) Receive(buffer []byte, timeout time.Duration) (int, net.IP, error) {
	// 设置读取超时
	tv := syscall.NsecToTimeval(timeout.Nanoseconds())
	err := syscall.SetsockoptTimeval(s.fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to set recv timeout: %v", err)
	}

	n, from, err := syscall.Recvfrom(s.fd, buffer, 0)
	if err != nil {
		return 0, nil, err
	}

	var srcIP net.IP
	if addr, ok := from.(*syscall.SockaddrInet4); ok {
		srcIP = net.IP(addr.Addr[:])
	}

	return n, srcIP, nil
}

// BindToInterface 绑定到指定网卡
// 注意：macOS 不支持 SO_BINDTODEVICE (这是 Linux 特有的)
// macOS 需要使用 IP_BOUND_IF 或类似的机制，或者 simply bind 到接口 IP。
// 这里暂时返回不支持，避免运行时错误。
func (s *RawSocket) BindToInterface(ifaceName string) error {
	// macOS/BSD 不支持 SO_BINDTODEVICE
	// 替代方案：syscall.SetsockoptInt(s.fd, syscall.IPPROTO_IP, syscall.IP_BOUND_IF, index)
	// 但这需要获取接口索引，为了简化，这里暂时忽略或返回错误。
	// 对于大多数扫描任务，让路由表决定出口是完全可以接受的。
	return fmt.Errorf("BindToInterface not supported on darwin (SO_BINDTODEVICE is linux only)")
}
