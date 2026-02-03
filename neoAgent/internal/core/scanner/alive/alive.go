package alive

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/qos"
	"neoagent/internal/core/model"
	"neoagent/internal/core/options"
)

// IpAliveScanner 实现 IP 存活扫描
type IpAliveScanner struct {
	// QoS 模块
	rttEstimator *qos.RttEstimator
	limiter      *qos.AdaptiveLimiter
}

func NewIpAliveScanner() *IpAliveScanner {
	return &IpAliveScanner{
		rttEstimator: qos.NewRttEstimator(),
		// Alive 扫描通常非常快，可以允许更高的并发
		// Initial 200, Max 5000
		limiter: qos.NewAdaptiveLimiter(200, 20, 5000),
	}
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

	// 并发控制 (QoS)
	// 如果用户指定了 Concurrency，作为 AdaptiveLimiter 的 Initial 和 Max
	if opts.Concurrency > 0 {
		s.limiter = qos.NewAdaptiveLimiter(opts.Concurrency, 10, opts.Concurrency*2)
	}

	// 获取本地 IP 用于拓扑判断 (缓存一下)
	localAddrs, _ := getLocalAddrs()

	for _, ip := range ips {
		wg.Add(1)
		
		// 获取并发令牌 (带上下文超时)
		if err := s.limiter.Acquire(ctx); err != nil {
			wg.Done()
			return nil, err // 上下文取消
		}

		go func(targetIP string) {
			defer wg.Done()
			defer s.limiter.Release()

			// 3. 根据策略选择探测器
			prober := s.getProber(targetIP, opts, localAddrs)

			// 4. 执行探测
			// 使用动态 RTO 作为超时基准
			// Alive 探测通常需要比 RTT 稍长的时间，例如 2*RTT + Buffer
			// 如果 RTO 很短 (e.g. 100ms)，我们至少给 1s 的探测时间 (尤其是 ARP/ICMP 可能被限速)
			// 但也不能太长，否则会拖慢整体进度
			rto := s.rttEstimator.Timeout()
			scanTimeout := rto * 2
			if scanTimeout < 1*time.Second {
				scanTimeout = 1 * time.Second
			}
			if scanTimeout > 3*time.Second {
				scanTimeout = 3 * time.Second // 上限 3s
			}

			probeCtx, cancel := context.WithTimeout(ctx, scanTimeout)
			defer cancel()

			start := time.Now()
			probeRes, err := prober.Probe(probeCtx, targetIP, scanTimeout)
			duration := time.Since(start)

			// QoS 反馈
			if err == nil && probeRes != nil && probeRes.Alive {
				// 成功：更新 RTT，增加并发
				s.rttEstimator.Update(duration)
				s.limiter.OnSuccess()
			} else {
				// 失败
				// 如果是因为超时导致的失败，惩罚并发
				// 注意：Probe 返回的 error 可能为空，只看 probeRes.Alive
				// 如果 probeRes.Alive == false，可能是不可达（很快返回）或超时
				if duration >= scanTimeout {
					s.limiter.OnFailure()
				}
			}

			if probeRes != nil && probeRes.Alive {
				resultData := model.IpAliveResult{
					IP:    targetIP,
					Alive: true,
					RTT:   probeRes.Latency,
					TTL:   probeRes.TTL,
				}

				// OS 猜测
				if probeRes.TTL > 0 {
					resultData.OS = guessOS(probeRes.TTL)
				}

				// Hostname 解析
				if opts.ResolveHostname {
					resultData.Hostname = resolveHostname(targetIP)
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

	if v, ok := params["enable_arp"].(bool); ok {
		opts.EnableArp = v
	}
	if v, ok := params["enable_icmp"].(bool); ok {
		opts.EnableIcmp = v
	}
	if v, ok := params["enable_tcp"].(bool); ok {
		opts.EnableTcp = v
	}

	// 处理端口列表，注意类型转换
	if v, ok := params["tcp_ports"]; ok {
		// 可能是 []int 或 []interface{} (JSON反序列化后)
		if ports, ok := v.([]int); ok {
			// 如果 CLI 传了参数，这里就是具体的列表；如果没传，Cobra 传进来 nil
			// 如果是 nil，我们应该回退到 Default
			if len(ports) > 0 {
				opts.TcpPorts = ports
			}
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

	// 确保 TcpPorts 不为空 (Double Check)
	if len(opts.TcpPorts) == 0 {
		opts.TcpPorts = options.DefaultAliveTcpPorts
	}

	if v, ok := params["concurrency"]; ok {
		if i, ok := v.(int); ok {
			opts.Concurrency = i
		} else if f, ok := v.(float64); ok {
			opts.Concurrency = int(f)
		}
	}

	return opts
}

// getProber 根据策略构建探测器
func (s *IpAliveScanner) getProber(targetIP string, opts *options.IpAliveScanOptions, localAddrs []net.Addr) Prober {
	var probers []Prober

	// 智能策略推断
	// 如果用户指定了任意协议开关，则进入 Manual 模式 ，用户输入优先
	// 优化：如果用户指定了自定义 TcpPorts (即非默认列表)，则隐式认为想要开启 TCP 探测
	isCustomPorts := isCustomTcpPorts(opts.TcpPorts)
	if isCustomPorts && !opts.EnableTcp {
		opts.EnableTcp = true
	}

	isManual := opts.EnableArp || opts.EnableIcmp || opts.EnableTcp

	if isManual {
		if opts.EnableArp {
			probers = append(probers, NewArpProber())
		}
		if opts.EnableIcmp {
			probers = append(probers, NewIcmpProber())
		}
		if opts.EnableTcp {
			probers = append(probers, NewTcpConnectProber(opts.TcpPorts))
		}
	} else {
		// Auto Strategy (默认) 用户没有指定协议的时候
		isLocal := isLocalIP(targetIP, localAddrs)

		if isLocal {
			// 同广播域：优先 ARP
			// 为了防止 Linux 下无 Root/arping 导致 ARP 失败，
			// 我们同时开启 ICMP Ping 作为兜底。
			// MultiProber 会并发执行，只要有一个成功就返回 True。
			probers = append(probers, NewArpProber())
			probers = append(probers, NewIcmpProber())
		} else {
			// 跨网段：ICMP + TCP Connect
			probers = append(probers, NewIcmpProber())
			probers = append(probers, NewTcpConnectProber(opts.TcpPorts))
		}
	}

	return NewMultiProber(probers...)
}

func isCustomTcpPorts(ports []int) bool {
	// 如果长度不同，肯定是自定义
	if len(ports) != len(options.DefaultAliveTcpPorts) {
		return true
	}
	// 逐个比较
	for i, p := range ports {
		if p != options.DefaultAliveTcpPorts[i] {
			return true
		}
	}
	return false
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

// guessOS 根据 TTL 猜测操作系统
func guessOS(ttl int) string {
	// 简单的 TTL 指纹
	// Linux/Unix: 64
	// Windows: 128
	// Solaris/Cisco: 255

	// 允许一定的跳数损耗
	if ttl <= 64 && ttl > 32 {
		return "Linux/Unix"
	}
	if ttl <= 128 && ttl > 64 {
		return "Windows"
	}
	if ttl <= 255 && ttl > 128 {
		return "Solaris/Cisco"
	}
	return "Unknown"
}

// resolveHostname 反向解析 Hostname
func resolveHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		// 通常返回的 name 包含末尾的点，例如 "google.com."
		name := names[0]
		if len(name) > 0 && name[len(name)-1] == '.' {
			return name[:len(name)-1]
		}
		return name
	}
	return ""
}
