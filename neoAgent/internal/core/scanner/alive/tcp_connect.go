package alive

import (
	"context"
	"fmt"
	"neoagent/internal/core/lib/network/dialer"
	"time"
)

// TcpConnectProber 基于 TCP Full Connect 的探测器
type TcpConnectProber struct {
	Ports []int
}

func NewTcpConnectProber(ports []int) *TcpConnectProber {
	return &TcpConnectProber{Ports: ports}
}

func (p *TcpConnectProber) Probe(ctx context.Context, ip string, timeout time.Duration) (*ProbeResult, error) {
	// 针对每个端口并发探测，只要有一个通就算活
	resultChan := make(chan time.Duration, len(p.Ports))

	for _, port := range p.Ports {
		go func(port int) {
			address := fmt.Sprintf("%s:%d", ip, port)
			// 使用全局 Dialer
			d := dialer.Get()
			start := time.Now()
			conn, err := d.DialContext(ctx, "tcp", address)
			if err == nil {
				conn.Close()
				resultChan <- time.Since(start)
			} else {
				resultChan <- 0
			}
		}(port)
	}

	// 只要有一个成功即可
	for i := 0; i < len(p.Ports); i++ {
		select {
		case latency := <-resultChan:
			if latency > 0 {
				return NewProbeResult(true, latency, 0), nil
			}
		case <-ctx.Done():
			return &ProbeResult{Alive: false}, ctx.Err()
		}
	}

	return &ProbeResult{Alive: false}, nil
}
