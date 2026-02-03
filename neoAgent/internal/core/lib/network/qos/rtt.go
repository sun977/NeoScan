package qos

import (
	"sync"
	"time"
)

const (
	defaultInitialRTO = 1 * time.Second      // 默认初始重传超时时间 (RTO)
	minRTO            = 100 * time.Millisecond // 最小 RTO，防止超时过短
	maxRTO            = 10 * time.Second       // 最大 RTO，防止超时过长
	alpha             = 0.125                  // 平滑因子 1/8 (RFC 6298 标准值)
	beta              = 0.25                   // 偏差因子 1/4 (RFC 6298 标准值)
)

// RttEstimator 实现了 RFC 6298 TCP RTO (重传超时) 计算算法
// 用于根据网络状况动态估算合适的超时时间
type RttEstimator struct {
	srtt   time.Duration // Smoothed RTT (平滑往返时间)
	rttvar time.Duration // RTT Variation (RTT 波动值/偏差)
	rto    time.Duration // Retransmission Timeout (计算出的超时时间)
	mu     sync.RWMutex  // 读写锁，保护并发访问
}

// NewRttEstimator 创建一个新的 RTT 估算器
func NewRttEstimator() *RttEstimator {
	return &RttEstimator{
		rto: defaultInitialRTO,
	}
}

// Update 根据新的 RTT 测量值更新估算状态
// rtt: 本次实际测量的往返时间
func (e *RttEstimator) Update(rtt time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.srtt == 0 {
		// 第一次测量 (参考 RFC 6298 2.2)
		// SRTT <- R
		// RTTVAR <- R/2
		e.srtt = rtt
		e.rttvar = rtt / 2
	} else {
		// 后续测量 (参考 RFC 6298 2.3)
		// RTTVAR <- (1 - beta) * RTTVAR + beta * |SRTT - R'|
		// 计算本次测量值与平滑值的偏差绝对值
		delta := e.srtt - rtt
		if delta < 0 {
			delta = -delta
		}
		// 更新 RTT 波动值
		e.rttvar = time.Duration((1-beta)*float64(e.rttvar) + beta*float64(delta))

		// SRTT <- (1 - alpha) * SRTT + alpha * R'
		// 更新平滑 RTT
		e.srtt = time.Duration((1-alpha)*float64(e.srtt) + alpha*float64(rtt))
	}

	// RTO <- SRTT + max(G, K*RTTVAR)
	// K=4 是标准建议值。这里忽略时钟粒度 G 的影响，因为 Go 的时钟精度足够高。
	e.rto = e.srtt + 4*e.rttvar

	// 边界检查 (参考 RFC 6298 2.4 / 2.5)
	// 确保 RTO 不会过小导致误判，也不会过大导致等待太久
	if e.rto < minRTO {
		e.rto = minRTO
	} else if e.rto > maxRTO {
		e.rto = maxRTO
	}
}

// Timeout 获取当前建议的超时时间 (RTO)
// 这个值是动态变化的，反映了最近的网络状况
func (e *RttEstimator) Timeout() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rto
}
