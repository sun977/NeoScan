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
	// 返回: (结果, 错误)
	Probe(ctx context.Context, ip string, timeout time.Duration) (*ProbeResult, error)
}

// MultiProber 组合探测器
type MultiProber struct {
	probers []Prober
}

func NewMultiProber(probers ...Prober) *MultiProber {
	return &MultiProber{probers: probers}
}

// Probe 并发执行所有探测器，只要有一个成功即返回 True
func (m *MultiProber) Probe(ctx context.Context, ip string, timeout time.Duration) (*ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan *ProbeResult, len(m.probers))

	for _, p := range m.probers {
		go func(prober Prober) {
			res, _ := prober.Probe(ctx, ip, timeout)
			// 确保不返回 nil，即使失败也要返回一个 Alive=false 的结果
			if res == nil {
				res = &ProbeResult{Alive: false}
			}
			resultChan <- res
		}(p)
	}

	// 等待结果，只要有一个为 true 就返回
	// 如果所有都返回 false，则最终返回 false
	// 优先返回包含更多信息（如 TTL/Latency）的结果
	var bestResult *ProbeResult = &ProbeResult{Alive: false}

	for i := 0; i < len(m.probers); i++ {
		select {
		case res := <-resultChan:
			if res.Alive {
				// 如果已经有一个结果，比较一下，取信息量大的
				// 比如 ICMP 有 TTL，TCP 只有 RTT，优先取 ICMP 的
				if !bestResult.Alive {
					bestResult = res
				} else {
					if res.TTL > 0 {
						bestResult.TTL = res.TTL
					}
					if res.Latency > 0 && (bestResult.Latency == 0 || res.Latency < bestResult.Latency) {
						bestResult.Latency = res.Latency
					}
				}
				// 策略确认：速度优先。
				// 只要探测到存活，立即返回，不等待其他可能包含 TTL 的探测结果（如 ICMP）。
				// 如果结果中缺失 TTL，上层逻辑将不显示 OS 信息或标记为 Unknown。
				return bestResult, nil
			}
		case <-ctx.Done():
			return bestResult, ctx.Err()
		}
	}

	return bestResult, nil
}
