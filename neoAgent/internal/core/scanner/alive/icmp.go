package alive

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

type IcmpProber struct{}

func NewIcmpProber() *IcmpProber {
	return &IcmpProber{}
}

func (p *IcmpProber) Probe(ctx context.Context, ip string, timeout time.Duration) (*ProbeResult, error) {
	// 简单的 ICMP Ping 实现
	// 为了避免 Raw Socket 权限问题，我们优先尝试调用系统 ping 命令
	// 这是一种 "Pragmatic" 的做法。

	var cmd *exec.Cmd
	var stdout bytes.Buffer

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

	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return &ProbeResult{Alive: false}, nil
	}

	// 解析输出获取 TTL 和 Latency
	output := stdout.String()
	latency, ttl := parsePingOutput(output, runtime.GOOS)

	return NewProbeResult(true, latency, ttl), nil
}

func parsePingOutput(output string, osType string) (time.Duration, int) {
	var latency time.Duration
	var ttl int

	if osType == "windows" {
		// Windows: "Reply from 1.1.1.1: bytes=32 time=13ms TTL=56"
		// 兼容中文: "来自 127.0.0.1 的回复: 字节=32 时间<1ms TTL=128"
		// 忽略 "time" 或 "时间" 前缀，直接匹配 = 或 < 后面的数字 + ms
		reTime := regexp.MustCompile(`[<>=]([\d\.]+) ?ms`)
		reTTL := regexp.MustCompile(`TTL=(\d+)`)

		if matches := reTime.FindStringSubmatch(output); len(matches) > 1 {
			if ms, err := strconv.ParseFloat(matches[1], 64); err == nil {
				latency = time.Duration(ms * float64(time.Millisecond))
			}
		}
		if matches := reTTL.FindStringSubmatch(output); len(matches) > 1 {
			if t, err := strconv.Atoi(matches[1]); err == nil {
				ttl = t
			}
		}
	} else {
		// Linux: "64 bytes from 1.1.1.1: icmp_seq=1 ttl=56 time=13.5 ms"
		reTime := regexp.MustCompile(`time=([\d\.]+) ms`)
		reTTL := regexp.MustCompile(`ttl=(\d+)`)

		if matches := reTime.FindStringSubmatch(output); len(matches) > 1 {
			if ms, err := strconv.ParseFloat(matches[1], 64); err == nil {
				latency = time.Duration(ms * float64(time.Millisecond))
			}
		}
		if matches := reTTL.FindStringSubmatch(output); len(matches) > 1 {
			if t, err := strconv.Atoi(matches[1]); err == nil {
				ttl = t
			}
		}
	}

	return latency, ttl
}
