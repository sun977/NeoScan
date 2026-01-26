//go:build linux
// +build linux

package netraw

import (
	"fmt"
	"net"
	"syscall"
	"time"
)

// RawSocket 封装 Linux 下的 Raw Socket 操作
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
		return nil, fmt.Errorf("failed to create raw socket: %v", err)
	}

	// 设置 IP_HDRINCL，告诉内核我们自己构建 IP 头部
	// 注意：如果只想发 TCP 包而不构建 IP 头，可以不设置这个，
	// 但为了完全控制 (如 Nmap OS 探测)，通常需要自己构建 IP 头
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
	addr := syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{dst[0], dst[1], dst[2], dst[3]},
	}

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

// BindToInterface 绑定到指定网卡 (可选)
func (s *RawSocket) BindToInterface(ifaceName string) error {
	return syscall.SetsockoptString(s.fd, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifaceName)
}
