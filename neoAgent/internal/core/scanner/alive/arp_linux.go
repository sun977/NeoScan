//go:build !windows && !darwin

package alive

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Linux/Unix 下的 ARP 实现
// 策略:
// 1. 尝试调用系统 `arping` 命令 (最准确，但可能不存在)
// 2. 如果 arping 不存在或执行失败，返回 false，由上层策略回退到 ICMP Ping。

type ArpProber struct{}

func NewArpProber() *ArpProber {
	return &ArpProber{}
}

func (p *ArpProber) Probe(ctx context.Context, ip string, timeout time.Duration) (*ProbeResult, error) {
	// 1. 尝试使用 arping
	// 假设是标准的 iputils-arping: arping -c 1 -w 1 <IP>

	arpingPath, err := exec.LookPath("arping")
	if err == nil {
		// arping 存在，尝试执行
		timeoutSec := int(timeout.Seconds())
		if timeoutSec < 1 {
			timeoutSec = 1
		}

		// -f: quit on first reply
		// -c 1: count 1
		// -w: timeout
		cmd := exec.CommandContext(ctx, arpingPath, "-f", "-c", "1", "-w", fmt.Sprint(timeoutSec), ip)
		start := time.Now()
		if err := cmd.Run(); err == nil {
			// ARP 成功
			// Latency: 粗略计算，包含进程启动开销
			latency := time.Since(start)
			return NewProbeResult(true, latency, 0), nil
		}
	}

	// 2. 降级方案
	// 如果 arping 失败，我们直接返回 false。
	// 在 IpAliveScanner 的 Auto 策略中，我们会并发执行 IcmpProber 作为兜底。
	// 所以这里不需要再手动调用 ICMP。

	return &ProbeResult{Alive: false}, nil
}
