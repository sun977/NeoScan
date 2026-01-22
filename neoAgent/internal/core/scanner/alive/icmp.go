package alive

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

type IcmpProber struct{}

func NewIcmpProber() *IcmpProber {
	return &IcmpProber{}
}

func (p *IcmpProber) Probe(ctx context.Context, ip string, timeout time.Duration) (bool, error) {
	// 简单的 ICMP Ping 实现
	// 为了避免 Raw Socket 权限问题，我们优先尝试调用系统 ping 命令
	// 这是一种 "Pragmatic" 的做法。

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// -n 1: 发送1次
		// -w 1000: 超时1000ms (注意 Windows ping -w 单位是毫秒)
		timeoutMs := int(timeout.Milliseconds())
		if timeoutMs < 1 {
			timeoutMs = 1000
		}
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", fmt.Sprint(timeoutMs), ip)
	} else {
		// Linux/Mac
		// -c 1: count 1
		// -W 1: timeout 1 second (Linux ping -W 单位通常是秒，有的版本支持小数)
		timeoutSec := int(timeout.Seconds())
		if timeoutSec < 1 {
			timeoutSec = 1
		}
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", fmt.Sprint(timeoutSec), ip)
	}

	// 既然是探活，我们不关心输出，只关心 Exit Code
	err := cmd.Run()
	return err == nil, nil
}
