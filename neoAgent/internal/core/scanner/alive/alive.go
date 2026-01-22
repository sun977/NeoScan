package alive

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"neoagent/internal/core/model"
	"neoagent/internal/core/options"
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
	// 1. 解析目标
	ips, err := parseTarget(task.Target)
	if err != nil {
		return nil, err
	}

	// 2. 解析参数
	opts := parseOptions(task.Params)

	var results []*model.TaskResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 并发控制
	sem := make(chan struct{}, 100)

	// 获取本地 IP 用于拓扑判断 (缓存一下)
	localAddrs, _ := getLocalAddrs()

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}

		go func(targetIP string) {
			defer wg.Done()
			defer func() { <-sem }()

			// 3. 根据策略选择探测器
			prober := s.getProber(targetIP, opts, localAddrs)

			// 4. 执行探测
			// 超时时间取 task.Timeout 或默认短超时 (例如单IP探测给 2s足够了，总超时由 task.Timeout 控制)
			// 但这里的 task.Timeout 是整个任务的。
			// 单个 IP 的探测不宜过长。
			probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			alive, _ := prober.Probe(probeCtx, targetIP, 3*time.Second)

			if alive {
				resultData := model.IpAliveResult{
					IP:    targetIP,
					Alive: true,
				}

				result := &model.TaskResult{
					TaskID:      task.ID,
					Status:      model.TaskStatusSuccess,
					Result:      resultData,
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

// parseOptions 从任务参数中解析选项
func parseOptions(params map[string]interface{}) *options.IpAliveScanOptions {
	opts := options.NewIpAliveScanOptions()

	if v, ok := params["strategy"].(string); ok {
		opts.Strategy = v
	}
	if v, ok := params["enable_arp"].(bool); ok {
		opts.EnableArp = v
	}
	if v, ok := params["enable_icmp"].(bool); ok {
		opts.EnableIcmp = v
	}
	if v, ok := params["enable_tcp"].(bool); ok {
		opts.EnableTcp = v
	}
	if v, ok := params["enable_tcp_syn"].(bool); ok {
		opts.EnableTcpSyn = v
	}

	// 处理端口列表，注意类型转换
	if v, ok := params["tcp_ports"]; ok {
		// 可能是 []int 或 []interface{} (JSON反序列化后)
		if ports, ok := v.([]int); ok {
			opts.TcpPorts = ports
		} else if portsIf, ok := v.([]interface{}); ok {
			var ports []int
			for _, p := range portsIf {
				if f, ok := p.(float64); ok {
					ports = append(ports, int(f))
				} else if i, ok := p.(int); ok {
					ports = append(ports, int(i))
				}
			}
			if len(ports) > 0 {
				opts.TcpPorts = ports
			}
		}
	}

	return opts
}

// getProber 根据策略构建探测器
func (s *IpAliveScanner) getProber(targetIP string, opts *options.IpAliveScanOptions, localAddrs []net.Addr) Prober {
	var probers []Prober

	if opts.Strategy == "manual" {
		if opts.EnableArp {
			probers = append(probers, NewArpProber())
		}
		if opts.EnableIcmp {
			probers = append(probers, NewIcmpProber())
		}
		if opts.EnableTcp {
			probers = append(probers, NewTcpConnectProber(opts.TcpPorts))
		}
		if opts.EnableTcpSyn {
			probers = append(probers, NewTcpSynProber(opts.TcpPorts))
		}
		// 如果什么都没选，默认回退到 Ping
		if len(probers) == 0 {
			probers = append(probers, NewIcmpProber())
		}
	} else {
		// Auto Strategy
		isLocal := isLocalIP(targetIP, localAddrs)

		if isLocal {
			// 同广播域：优先 ARP
			// 为了保险，也可以组合 ICMP，但 ARP 只要通了就是通了
			probers = append(probers, NewArpProber())
		} else {
			// 跨网段：ICMP + TCP Connect
			probers = append(probers, NewIcmpProber())
			probers = append(probers, NewTcpConnectProber(opts.TcpPorts))
		}
	}

	return NewMultiProber(probers...)
}

func getLocalAddrs() ([]net.Addr, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var addrs []net.Addr
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 {
			continue
		}
		if i.Flags&net.FlagLoopback != 0 {
			continue
		}
		as, err := i.Addrs()
		if err == nil {
			addrs = append(addrs, as...)
		}
	}
	return addrs, nil
}

func isLocalIP(targetIP string, localAddrs []net.Addr) bool {
	ip := net.ParseIP(targetIP)
	if ip == nil {
		return false
	}

	for _, addr := range localAddrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.Contains(ip) {
				return true
			}
		}
	}
	return false
}

// parseTarget 解析目标 IP (简化版)
func parseTarget(target string) ([]string, error) {
	// 如果是 CIDR
	if _, ipNet, err := net.ParseCIDR(target); err == nil {
		var ips []string
		for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
			ips = append(ips, ip.String())
		}
		// 移除网络地址和广播地址
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
