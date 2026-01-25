package port

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"neoagent/internal/core/model"
)

const (
	ScannerName    = "port_service_scanner"
	DefaultTimeout = 2 * time.Second
)

// PortServiceScanner 端口服务扫描器
// 实现了 Scanner 接口，整合了 TCP Connect 扫描与 Nmap 服务识别逻辑
type PortServiceScanner struct {
	probes       []*Probe
	probeMap     map[string]*Probe
	portProbeMap map[int][]*Probe // 端口到探针的映射 (优化查找)
	allProbes    []*Probe         // 所有探针 (按 rarity 排序)

	initOnce sync.Once
	initErr  error
}

func NewPortServiceScanner() *PortServiceScanner {
	return &PortServiceScanner{
		probeMap:     make(map[string]*Probe),
		portProbeMap: make(map[int][]*Probe),
	}
}

func (s *PortServiceScanner) Name() string {
	return ScannerName
}

func (s *PortServiceScanner) Type() model.TaskType {
	return model.TaskTypePortScan
}

// ensureInit 确保规则已加载
func (s *PortServiceScanner) ensureInit() error {
	s.initOnce.Do(func() {
		// 解析 NmapServiceProbes (来自 rules.go)
		if NmapServiceProbes == "" {
			// 如果没有嵌入规则，可能是在开发环境或者文件丢失
			// 这里可以加载一个最小集或者报错，暂时不做处理，Scan 时会发现没有探针
			return
		}

		probes, err := ParseNmapProbes(NmapServiceProbes)
		if err != nil {
			s.initErr = fmt.Errorf("failed to parse nmap probes: %v", err)
			return
		}

		s.probes = probes

		// 构建索引
		for _, p := range probes {
			s.probeMap[p.Name] = p

			// 关联到 ports
			for _, port := range p.Ports {
				s.portProbeMap[port] = append(s.portProbeMap[port], p)
			}
			// 关联到 sslports
			for _, port := range p.SslPorts {
				s.portProbeMap[port] = append(s.portProbeMap[port], p)
			}
		}

		// 对所有探针按 rarity 排序
		s.allProbes = make([]*Probe, len(probes))
		copy(s.allProbes, probes)
		sort.Slice(s.allProbes, func(i, j int) bool {
			return s.allProbes[i].Rarity < s.allProbes[j].Rarity
		})

		// 对每个端口的探针列表也排序
		for port, list := range s.portProbeMap {
			sort.Slice(list, func(i, j int) bool {
				return list[i].Rarity < list[j].Rarity
			})
			// 去重 (同一个探针可能同时在 ports 和 sslports)
			s.portProbeMap[port] = uniqueProbes(list)
		}
	})
	return s.initErr
}

func uniqueProbes(probes []*Probe) []*Probe {
	seen := make(map[string]bool)
	result := make([]*Probe, 0, len(probes))
	for _, p := range probes {
		if !seen[p.Name] {
			seen[p.Name] = true
			result = append(result, p)
		}
	}
	return result
}

func (s *PortServiceScanner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	if err := s.ensureInit(); err != nil {
		return nil, err
	}

	target := task.Target
	portRange := task.PortRange
	if portRange == "" {
		// 默认扫描 Top 1000? 或者报错
		// 这里假设调用方已处理好
		return nil, fmt.Errorf("port range is required")
	}

	// 解析参数
	serviceDetect := false
	if val, ok := task.Params["service_detect"]; ok {
		if v, ok := val.(bool); ok {
			serviceDetect = v
		}
	}

	// 解析端口列表
	ports := ParsePortList(portRange)

	// 并发控制 (使用 Runner 或简单的 WaitGroup)
	// 这里为了简单演示，使用 Semaphore
	concurrency := 100
	if val, ok := task.Params["rate"]; ok {
		if v, ok := val.(int); ok {
			concurrency = v
		}
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []interface{}

	for _, port := range ports {
		wg.Add(1)
		sem <- struct{}{}
		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()

			// 执行扫描
			res := s.scanPort(ctx, target, p, serviceDetect)
			if res != nil {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	return []*model.TaskResult{{
		TaskID: task.TaskID,
		Status: model.TaskStatusCompleted,
		Result: results,
	}}, nil
}

func (s *PortServiceScanner) scanPort(ctx context.Context, ip string, port int, serviceDetect bool) *model.PortServiceResult {
	address := fmt.Sprintf("%s:%d", ip, port)
	timeout := DefaultTimeout

	// 1. TCP Connect 探测端口开放
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil // 端口关闭 (或被过滤)
	}
	defer conn.Close()

	result := &model.PortServiceResult{
		IP:       ip,
		Port:     port,
		Protocol: "tcp",
		Status:   "open",
	}

	// 如果不进行服务识别，直接返回 Open
	if !serviceDetect {
		return result
	}

	// 2. 服务识别 (Service Discovery)
	// 获取适用该端口的探针
	probes := s.getProbesForPort(port)

	// 总是把 NULL 探针放在第一个 (如果适用)
	// 并且把 Generic 探针放在后面

	// 简单的探针执行循环
	for _, probe := range probes {
		select {
		case <-ctx.Done():
			return result
		default:
		}

		// 某些探针可能不需要重新连接，可以复用连接?
		// Nmap 逻辑通常是每个探针建立新连接，除了 NULL 探针可能复用初始连接
		// 这里简单起见，每次都建立新连接 (除了第一个 Probe 可能尝试复用 conn 如果我们没 Close)
		// 但为了代码清晰，我们重新 Dial

		fingerprint := s.executeProbe(ctx, ip, port, probe, timeout)
		if fingerprint != nil {
			// 匹配成功!
			result.Service = fingerprint.Service
			result.Product = fingerprint.ProductName
			result.Version = fingerprint.Version
			result.Info = fingerprint.Info
			result.Hostname = fingerprint.Hostname
			result.OS = fingerprint.OperatingSystem
			result.DeviceType = fingerprint.DeviceType
			result.CPE = fingerprint.CPE
			result.Status = "matched" // 或者保持 open?
			return result
		}
	}

	// 如果所有探针都失败，标记为 unknown service
	result.Service = "unknown"
	return result
}

func (s *PortServiceScanner) getProbesForPort(port int) []*Probe {
	// 1. 获取端口特定探针
	specific := s.portProbeMap[port]

	// 2. 获取所有探针 (按 rarity 排序)
	// 实际 Nmap 逻辑会根据 rarity 阈值过滤
	// 这里简化：如果没有特定探针，或者特定探针没跑出来，会跑 Top 探针
	// 我们合并特定探针和通用探针，去重

	// 为了性能，我们只返回 Top N 探针 + 特定探针
	// 或者直接返回所有 rarity <= 7 的探针

	candidates := make([]*Probe, 0, len(specific)+len(s.allProbes))
	candidates = append(candidates, specific...)

	for _, p := range s.allProbes {
		if p.Rarity <= 7 { // 默认 rarity 限制
			candidates = append(candidates, p)
		}
	}

	return uniqueProbes(candidates)
}

func (s *PortServiceScanner) executeProbe(ctx context.Context, ip string, port int, probe *Probe, timeout time.Duration) *FingerPrint {
	// 1. 建立连接
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
	if err != nil {
		return nil
	}
	defer conn.Close()

	// 2. 发送探针数据
	// 处理 probe string 中的转义字符 (\r, \n, \x00 等)
	// ParseNmapProbes 中应该已经处理了一部分，或者我们需要在这里处理 Unquote
	// 假设 ProbeString 已经是 raw bytes (需要在 parser 中处理好)
	// 简单的 strconv.Unquote 只能处理 quoted string

	// 这里需要一个 helper 来把 `\x00\x00` 转换成实际 bytes
	// 暂时假设 parser 已经做好了，或者我们在 parser 里补上
	payload := unescapeString(probe.ProbeString)

	if len(payload) > 0 {
		conn.SetWriteDeadline(time.Now().Add(timeout))
		_, err = conn.Write([]byte(payload))
		if err != nil {
			return nil
		}
	}

	// 3. 读取响应
	conn.SetReadDeadline(time.Now().Add(timeout))
	// 读取多少? Nmap 通常读取一定量或者直到超时
	// 这里读取前 4KB
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && n == 0 {
		return nil
	}
	response := buf[:n]

	// 4. 匹配
	// 优先匹配当前 probe 的 MatchGroup
	for _, match := range probe.MatchGroup {
		if match.PatternRegexp.Match(response) {
			// 提取指纹
			return extractFingerprint(match, response)
		}
	}

	// 尝试 SoftMatch? (略)

	// 尝试 Fallback? (略，需要递归或循环)

	return nil
}

// unescapeString 处理 probe string 中的转义
// 简单实现，支持 \r \n \t \xHH
func unescapeString(s string) string {
	// 如果包含 \，尝试解析
	if !strings.Contains(s, "\\") {
		return s
	}
	// 这里的实现比较 hacky，理想情况是在 parser 阶段做
	// 简单替换常见字符
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\0`, "\x00")

	// 处理 \xHH (需要正则或循环)
	// 暂时略过复杂 hex 转义，假设 rules 文件比较简单
	return s
}

func extractFingerprint(match *Match, response []byte) *FingerPrint {
	fp := &FingerPrint{
		Service: match.Service,
	}

	// 使用正则提取子组
	matches := match.PatternRegexp.FindSubmatch(response)
	if matches == nil {
		return fp
	}

	// 解析 VersionInfo (p/Vendor/ v/$1/ ...)
	// 这是一个微型的模板解析器
	// 简单实现: 替换 $1, $2 ...

	parseTemplate := func(tmpl string) string {
		for i, sub := range matches {
			if i == 0 {
				continue
			}
			tmpl = strings.ReplaceAll(tmpl, fmt.Sprintf("$%d", i), string(sub))
		}
		return tmpl
	}

	// 简单的字符串分割解析 match.VersionInfo
	// e.g. p/OpenSSH/ v/$1/
	// 需要识别 p/, v/, i/, h/, o/, d/

	parts := strings.Split(match.VersionInfo, " ")
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		prefix := part[:2]               // p/
		content := part[2 : len(part)-1] // content

		val := parseTemplate(content)

		switch prefix {
		case "p/":
			fp.ProductName = val
		case "v/":
			fp.Version = val
		case "i/":
			fp.Info = val
		case "h/":
			fp.Hostname = val
		case "o/":
			fp.OperatingSystem = val
		case "d/":
			fp.DeviceType = val
		case "cpe:/":
			fp.CPE = val
		}
	}

	return fp
}
