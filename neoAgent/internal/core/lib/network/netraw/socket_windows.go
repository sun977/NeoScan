//go:build windows
// +build windows

package netraw

import (
	"fmt"
	"net"
	"time"
)

// Windows 占位实现
// Windows 下不直接支持 Raw Socket (Winsock2 限制了 TCP Raw Socket 的使用)
// 如需支持，通常需要 WinPcap/Npcap，但这引入了外部 CGO 依赖。
// 根据设计哲学，我们放弃 Windows 上的 Raw Socket 实现。

type RawSocket struct{}

func NewRawSocket(protocol int) (*RawSocket, error) {
	return nil, fmt.Errorf("raw socket not supported on windows")
}

func (s *RawSocket) Close() error {
	return nil
}

func (s *RawSocket) Send(dst net.IP, packet []byte) error {
	return fmt.Errorf("not supported")
}

func (s *RawSocket) Receive(buffer []byte, timeout time.Duration) (int, net.IP, error) {
	return 0, nil, fmt.Errorf("not supported")
}

func (s *RawSocket) BindToInterface(ifaceName string) error {
	return fmt.Errorf("not supported")
}
