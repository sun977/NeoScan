//go:build darwin

package netraw

import (
	"fmt"
	"net"
	"time"
)

// Darwin Stub Implementation
// MacOS Raw Socket 支持有限，且需要特殊权限/BPF。
// 暂时降级为不支持，Agent 将回退到 Connect Scan 和 ICMP Ping。

type RawSocket struct{}

func NewRawSocket(protocol int) (*RawSocket, error) {
	return nil, fmt.Errorf("raw socket not supported on darwin")
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
