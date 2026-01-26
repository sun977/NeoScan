package os

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

// TTLEngine 基于 Ping TTL 的轻量级 OS 识别
type TTLEngine struct{}

func NewTTLEngine() *TTLEngine {
	return &TTLEngine{}
}

func (e *TTLEngine) Name() string {
	return "ttl"
}

func (e *TTLEngine) Scan(ctx context.Context, target string) (*OsInfo, error) {
	ttl, err := e.getPingTTL(ctx, target)
	if err != nil {
		return nil, err
	}

	if ttl <= 0 {
		return nil, fmt.Errorf("no response")
	}

	return e.guessOS(ttl), nil
}

func (e *TTLEngine) getPingTTL(ctx context.Context, ip string) (int, error) {
	var cmd *exec.Cmd
	var stdout bytes.Buffer

	// 构造 Ping 命令 (参考 alive/icmp.go)
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", "1000", ip)
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ip)
	}

	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	return e.parseTTL(stdout.String(), runtime.GOOS), nil
}

func (e *TTLEngine) parseTTL(output string, osType string) int {
	var reTTL *regexp.Regexp
	if osType == "windows" {
		// Windows: TTL=128
		reTTL = regexp.MustCompile(`(?i)TTL=(\d+)`)
	} else {
		// Linux: ttl=64
		reTTL = regexp.MustCompile(`ttl=(\d+)`)
	}

	matches := reTTL.FindStringSubmatch(output)
	if len(matches) > 1 {
		if t, err := strconv.Atoi(matches[1]); err == nil {
			return t
		}
	}
	return 0
}

func (e *TTLEngine) guessOS(ttl int) *OsInfo {
	// 简单的 TTL 推断规则
	// Windows: 128 (通常在 65-128 之间)
	// Linux/Unix: 64 (通常在 33-64 之间)
	// Solaris/Cisco: 255 (通常在 129-255 之间)
	
	info := &OsInfo{
		Source:   "TTL",
		Accuracy: 80, // TTL 并不是 100% 准确
	}

	if ttl <= 32 {
		info.Name = "Unknown (Old/Embedded)"
		info.Family = "Unknown"
		info.Accuracy = 50
	} else if ttl <= 64 {
		info.Name = "Linux/Unix"
		info.Family = "Unix"
		info.Fingerprint = "TTL<=64"
	} else if ttl <= 128 {
		info.Name = "Windows"
		info.Family = "Windows"
		info.Fingerprint = "TTL<=128"
	} else {
		info.Name = "Solaris/Network Device"
		info.Family = "Unix/Cisco"
		info.Fingerprint = "TTL<=255"
	}

	return info
}
