//go:build !windows

package alive

import (
	"context"
	"time"
)

// Linux 下可以使用 Raw Socket 实现 SYN 扫描
// 这里先做一个简单的实现或者 Stub，随后完善
// 真正的 SYN 扫描需要构造 IP/TCP 头部
type TcpSynProber struct {
	Ports []int
}

func NewTcpSynProber(ports []int) *TcpSynProber {
	return &TcpSynProber{Ports: ports}
}

func (p *TcpSynProber) Probe(ctx context.Context, ip string, timeout time.Duration) (bool, error) {
	// TODO: Implement Raw Socket SYN scan
	// 这是一个复杂功能，需要 root 权限
	// 暂时降级为 TCP Connect 以保证项目可编译运行
	delegate := NewTcpConnectProber(p.Ports)
	return delegate.Probe(ctx, ip, timeout)
}
