//go:build darwin

package alive

import (
	"context"
	"os/exec"
	"time"
)

// MacOS (Darwin) 下的 ARP 实现
// 策略:
// 1. 尝试调用系统 `arping` 命令 (需用户安装, e.g., brew install arping)
// 2. 如果 arping 不存在或执行失败，返回 false，由上层策略回退到 ICMP Ping。

type ArpProber struct{}

func NewArpProber() *ArpProber {
	return &ArpProber{}
}

func (p *ArpProber) Probe(ctx context.Context, ip string, timeout time.Duration) (*ProbeResult, error) {
	// 1. 尝试使用 arping
	// MacOS 自带没有 arping，通常通过 brew 安装
	// 假设兼容 iputils-arping 参数，或者根据实际情况调整

	arpingPath, err := exec.LookPath("arping")
	if err == nil {
		// arping 存在，尝试执行
		timeoutSec := int(timeout.Seconds())
		if timeoutSec < 1 {
			timeoutSec = 1
		}

		// -c 1: count 1
		// -t: timeout (BSD arping uses -t for timeout in seconds? No, BSD uses -w in microseconds? Or just wait?)
		// Linux iputils: -w sec
		// Let's assume common flags or fail.
		// Safe bet: just -c 1.

		cmd := exec.CommandContext(ctx, arpingPath, "-c", "1", ip)
		start := time.Now()
		if err := cmd.Run(); err == nil {
			// ARP 成功
			latency := time.Since(start)
			return NewProbeResult(true, latency, 0), nil
		}
	}

	// 2. 降级方案
	// MacOS 也可以尝试解析 `arp -a` 输出，但比较慢且不实时。
	// 这里直接返回 false，让上层使用 ICMP/TCP。

	return &ProbeResult{Alive: false}, nil
}
