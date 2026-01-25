package dialer

import (
	"context"
	"net"
	"time"
)

// Dialer 定义了网络连接器接口
type Dialer interface {
	// DialContext 建立连接
	// network: 协议 (tcp, udp)
	// address: 目标地址 (ip:port)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// DefaultDialer 默认直连拨号器
type DefaultDialer struct {
	Timeout time.Duration
}

func NewDefaultDialer(timeout time.Duration) *DefaultDialer {
	return &DefaultDialer{
		Timeout: timeout,
	}
}

func (d *DefaultDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: d.Timeout,
	}
	return dialer.DialContext(ctx, network, address)
}
