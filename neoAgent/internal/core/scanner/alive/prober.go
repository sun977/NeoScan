package alive

import (
	"context"
	"time"
)

// Prober 定义探测器接口
type Prober interface {
	// Probe 执行探测
	// ip: 目标IP
	// timeout: 超时时间
	// 返回: (是否存活, 错误)
	Probe(ctx context.Context, ip string, timeout time.Duration) (bool, error)
}

// MultiProber 组合探测器
type MultiProber struct {
	probers []Prober
}

func NewMultiProber(probers ...Prober) *MultiProber {
	return &MultiProber{probers: probers}
}

// Probe 并发执行所有探测器，只要有一个成功即返回 True
func (m *MultiProber) Probe(ctx context.Context, ip string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan bool, len(m.probers))

	for _, p := range m.probers {
		go func(prober Prober) {
			alive, _ := prober.Probe(ctx, ip, timeout)
			resultChan <- alive
		}(p)
	}

	// 等待结果，只要有一个为 true 就返回
	// 如果所有都返回 false，则最终返回 false
	failureCount := 0
	for i := 0; i < len(m.probers); i++ {
		select {
		case alive := <-resultChan:
			if alive {
				return true, nil
			}
			failureCount++
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	return false, nil
}
