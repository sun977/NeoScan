package alive

import (
	"time"
)

// ProbeResult 探测结果
type ProbeResult struct {
	Alive   bool
	Latency time.Duration
	TTL     int
}

// NewProbeResult 创建一个新的 ProbeResult
func NewProbeResult(alive bool, latency time.Duration, ttl int) *ProbeResult {
	return &ProbeResult{
		Alive:   alive,
		Latency: latency,
		TTL:     ttl,
	}
}
