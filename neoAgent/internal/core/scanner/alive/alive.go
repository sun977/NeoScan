package alive

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"neoagent/internal/core/model"

	probing "github.com/prometheus-community/pro-bing"
)

// IpAliveScanner 实现 IP 存活扫描
type IpAliveScanner struct{}

func NewIpAliveScanner() *IpAliveScanner {
	return &IpAliveScanner{}
}

func (s *IpAliveScanner) Name() model.TaskType {
	return model.TaskTypeIpAliveScan
}

func (s *IpAliveScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	// 1. 解析目标 (支持 CIDR 和 单个IP)
	// 这里简化处理，假设 Target 是单个IP或CIDR
	// 实际生产中需要 IP 解析库
	ips, err := parseTarget(task.Target)
	if err != nil {
		return nil, err
	}

	var results []*model.TaskResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 并发控制 (Semaphore)
	sem := make(chan struct{}, 100) // 限制并发数为100

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}

		go func(targetIP string) {
			defer wg.Done()
			defer func() { <-sem }()

			if isAlive(targetIP) {
				result := &model.TaskResult{
					TaskID:      task.ID,
					Status:      model.TaskStatusSuccess,
					Result:      map[string]interface{}{"ip": targetIP, "alive": true},
					ExecutedAt:  time.Now(),
					CompletedAt: time.Now(),
				}
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()
	return results, nil
}

// parseTarget 解析目标 IP (简化版)
func parseTarget(target string) ([]string, error) {
	// 如果是 CIDR
	if _, ipNet, err := net.ParseCIDR(target); err == nil {
		var ips []string
		for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
			ips = append(ips, ip.String())
		}
		// 移除网络地址和广播地址 (通常是第一个和最后一个)
		if len(ips) > 2 {
			return ips[1 : len(ips)-1], nil
		}
		return ips, nil
	}

	// 如果是单个 IP
	if ip := net.ParseIP(target); ip != nil {
		return []string{ip.String()}, nil
	}

	// 尝试作为域名解析
	addrs, err := net.LookupHost(target)
	if err == nil && len(addrs) > 0 {
		return addrs, nil
	}

	return nil, fmt.Errorf("invalid target: %s", target)
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// isAlive 使用 ICMP Ping 探测
func isAlive(ip string) bool {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return false
	}

	// Windows 下需要特权模式或设置非特权
	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	} else {
		// Linux 下通常也需要 true，取决于 sysctl 设置
		pinger.SetPrivileged(true)
	}

	pinger.Count = 1
	pinger.Timeout = 1 * time.Second

	err = pinger.Run()
	if err != nil {
		return false
	}

	return pinger.Statistics().PacketsRecv > 0
}
