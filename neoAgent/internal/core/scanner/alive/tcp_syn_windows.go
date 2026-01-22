//go:build windows

package alive

import (
	"context"
	"time"
)

// Windows 下不支持 Raw Socket 发送 SYN 包 (微软限制)
// 降级为 TCP Connect
type TcpSynProber struct {
	Ports []int
}

func NewTcpSynProber(ports []int) *TcpSynProber {
	return &TcpSynProber{Ports: ports}
}

func (p *TcpSynProber) Probe(ctx context.Context, ip string, timeout time.Duration) (bool, error) {
	// Windows 自动降级为 TCP Connect
	delegate := NewTcpConnectProber(p.Ports)
	return delegate.Probe(ctx, ip, timeout)
}
