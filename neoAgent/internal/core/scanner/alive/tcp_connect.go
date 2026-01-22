package alive

import (
	"context"
	"fmt"
	"net"
	"time"
)

// TcpConnectProber 基于 TCP Full Connect 的探测器
type TcpConnectProber struct {
	Ports []int
}

func NewTcpConnectProber(ports []int) *TcpConnectProber {
	return &TcpConnectProber{Ports: ports}
}

func (p *TcpConnectProber) Probe(ctx context.Context, ip string, timeout time.Duration) (bool, error) {
	// 针对每个端口并发探测，只要有一个通就算活
	resultChan := make(chan bool, len(p.Ports))
	
	for _, port := range p.Ports {
		go func(port int) {
			address := fmt.Sprintf("%s:%d", ip, port)
			d := net.Dialer{Timeout: timeout}
			conn, err := d.DialContext(ctx, "tcp", address)
			if err == nil {
				conn.Close()
				resultChan <- true
			} else {
				resultChan <- false
			}
		}(port)
	}

	// 只要有一个成功即可
	for i := 0; i < len(p.Ports); i++ {
		select {
		case success := <-resultChan:
			if success {
				return true, nil
			}
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	return false, nil
}
