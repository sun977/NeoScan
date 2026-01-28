package nmap_service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/dialer"
)

// Engine Gonmap 扫描引擎
type Engine struct {
	Probes       map[string]*Probe
	ProbeSort    []string
	PortProbeMap map[int][]string

	initOnce sync.Once
}

func NewEngine() *Engine {
	return &Engine{
		Probes:       make(map[string]*Probe),
		PortProbeMap: make(map[int][]string),
	}
}

// LoadRules 加载 Nmap 服务探测规则
func (e *Engine) LoadRules(content string) error {
	var err error
	e.Probes, e.ProbeSort, err = ParseNmapServiceProbes(content)
	if err != nil {
		return err
	}

	// 构建 Port -> Probe 映射
	for _, probe := range e.Probes {
		for _, port := range probe.Ports {
			e.PortProbeMap[port] = append(e.PortProbeMap[port], probe.Name)
		}
		for _, port := range probe.SslPorts {
			e.PortProbeMap[port] = append(e.PortProbeMap[port], probe.Name)
		}
	}

	// 按照 Rarity 排序
	for port, probes := range e.PortProbeMap {
		sort.Slice(probes, func(i, j int) bool {
			p1 := e.Probes[probes[i]]
			p2 := e.Probes[probes[j]]
			return p1.Rarity < p2.Rarity
		})
		e.PortProbeMap[port] = probes
	}

	return nil
}

// Scan 扫描指定端口的服务
func (e *Engine) Scan(ctx context.Context, ip string, port int, timeout time.Duration) (*FingerPrint, error) {
	// 1. 获取候选探针列表
	// 默认包含 NULL 探针和 Generic 探针
	probeNames := []string{"NULL", "GetRequest"} // Nmap 默认探针名可能不同，需确认

	// 添加端口特定探针
	if specificProbes, ok := e.PortProbeMap[port]; ok {
		probeNames = append(probeNames, specificProbes...)
	}

	// 去重
	probeNames = uniqueStrings(probeNames)

	// 2. 依次执行探针
	// 使用索引遍历，以便支持动态重排序
	for i := 0; i < len(probeNames); i++ {
		name := probeNames[i]

		// 检查 Context
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		probe, ok := e.Probes[name]
		if !ok {
			continue
		}

		// 发送探针并获取响应
		response, err := e.sendProbe(ctx, ip, port, probe, timeout)
		if err != nil {
			continue // 连接失败或超时，尝试下一个
		}

		// 3. 匹配指纹
		fp, isSoft := e.matchResponse(response, probe)
		if fp != nil {
			// 如果是硬匹配，直接返回结果
			if !isSoft {
				return fp, nil
			}

			// 如果是软匹配 (SoftMatch)，这只是一个线索 (Hint)
			// 利用这个线索去加速后续的探测。
			// 策略：如果识别出 Service (e.g. "http", "ssl")，则把相关探针移到当前位置之后
			if fp.Service != "" {
				// 动态调整 probeNames 顺序
				probeNames = e.prioritizeProbes(probeNames, i+1, fp.Service)
			}
		}
	}

	return nil, nil
}

// prioritizeProbes 将与 service 相关的探针移动到 startIndex 位置
func (e *Engine) prioritizeProbes(probes []string, startIndex int, service string) []string {
	if startIndex >= len(probes) {
		return probes
	}

	// 简单的关键词匹配策略
	// Nmap 探针名称通常包含线索，例如 "GetRequest", "HTTPOptions", "SSLSessionReq"
	// 这里做一个简单的映射或模糊匹配
	var keywords []string
	switch strings.ToLower(service) {
	case "http":
		keywords = []string{"GetRequest", "HTTPOptions", "RTSPRequest", "Help"}
	case "ssl", "tls":
		keywords = []string{"SSL", "TLS"}
	case "ftp":
		keywords = []string{"FTP"}
	case "smtp":
		keywords = []string{"SMTP"}
	case "ssh":
		keywords = []string{"SSH"}
	default:
		return probes
	}

	// 分离出优先级高的探针和剩余探针
	var highPriority []string
	var others []string

	// 只处理尚未执行的探针 (从 startIndex 开始)
	for j := startIndex; j < len(probes); j++ {
		name := probes[j]
		isHit := false
		for _, kw := range keywords {
			if strings.Contains(name, kw) {
				isHit = true
				break
			}
		}
		if isHit {
			highPriority = append(highPriority, name)
		} else {
			others = append(others, name)
		}
	}

	// 重新组合: [已执行] + [高优先级] + [其他]
	newProbes := make([]string, 0, len(probes))
	newProbes = append(newProbes, probes[:startIndex]...)
	newProbes = append(newProbes, highPriority...)
	newProbes = append(newProbes, others...)

	return newProbes
}

func (e *Engine) sendProbe(ctx context.Context, ip string, port int, probe *Probe, timeout time.Duration) ([]byte, error) {
	address := fmt.Sprintf("%s:%d", ip, port)
	d := dialer.Get() // 使用核心网络库

	// 优化超时策略：连接超时短，读写超时长
	// 总超时由 ctx 或 timeout 参数控制，但我们希望连接失败能快速返回
	connectTimeout := 2 * time.Second
	if timeout < connectTimeout {
		connectTimeout = timeout
	}

	// 创建带超时的连接 Context
	connCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	conn, err := d.DialContext(connCtx, "tcp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 设置读写超时
	// 如果 probe 有特定等待时间，使用它；否则使用剩余时间或默认读超时
	readTimeout := timeout
	if probe.Wait > 0 {
		readTimeout = probe.Wait
	} else {
		// 默认读超时至少给 3s，等待服务吐 Banner
		if readTimeout < 3*time.Second {
			readTimeout = 3 * time.Second
		}
	}

	conn.SetDeadline(time.Now().Add(readTimeout))

	// 发送 Payload (如果是 TCP，NULL 探针不发送数据)
	if len(probe.ProbeString) > 0 {
		_, err = conn.Write([]byte(probe.ProbeString))
		if err != nil {
			return nil, err
		}
	}

	// 读取响应
	// 简单实现：读取最多 4KB
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

func (e *Engine) matchResponse(response []byte, probe *Probe) (*FingerPrint, bool) {
	respStr := string(response)

	// 遍历 MatchGroup (硬匹配)
	for _, match := range probe.MatchGroup {
		if match.PatternRegexp == nil {
			continue
		}

		// Use regexp2 MatchString (returns bool, error)
		isMatch, _ := match.PatternRegexp.MatchString(respStr)
		if isMatch {
			// 匹配成功！提取版本信息
			// 返回 isSoft = false
			return extractFingerPrint(match, respStr), false
		}
	}

	// 遍历 SoftMatchGroup (软匹配)
	for _, match := range probe.SoftMatchGroup {
		if match.PatternRegexp == nil {
			continue
		}
		isMatch, _ := match.PatternRegexp.MatchString(respStr)
		if isMatch {
			// 软匹配命中
			// 返回 isSoft = true
			return extractFingerPrint(match, respStr), true
		}
	}

	return nil, false
}

func extractFingerPrint(match *Match, response string) *FingerPrint {
	fp := &FingerPrint{
		Service:          match.Service,
		MatchRegexString: match.Pattern,
	}

	// Find submatches using regexp2
	m, err := match.PatternRegexp.FindStringMatch(response)
	if err != nil || m == nil {
		// Should not happen if MatchString returned true, but safe check
		return fp
	}

	// Convert regexp2 groups to string slice for compatibility with replacePlaceholders
	var submatches []string
	for _, g := range m.Groups() {
		submatches = append(submatches, g.String())
	}

	// Parse VersionInfoTemplate if available
	if match.VersionInfoTemplate != "" {
		parseVersionInfo(fp, match.VersionInfoTemplate, submatches)
	}

	return fp
}

func parseVersionInfo(fp *FingerPrint, template string, submatches []string) {
	// Nmap version info format: p/vendor_product/ v/version/ ...
	// The delimiter can be any char, usually /

	// Simple state machine parser
	input := template
	for len(input) > 0 {
		// Expect tag like " p" or "v"
		input = strings.TrimSpace(input)
		if len(input) < 2 {
			break
		}

		tag := ""
		if strings.HasPrefix(input, "cpe:") {
			tag = "cpe:"
			input = input[4:]
		} else {
			tag = input[:1]
			input = input[1:]
		}

		if len(input) == 0 {
			break
		}

		delimiter := input[:1]
		input = input[1:]

		// Find closing delimiter
		endIdx := strings.Index(input, delimiter)
		if endIdx == -1 {
			break // Malformed
		}

		val := input[:endIdx]
		input = input[endIdx+1:]

		// Replace placeholders $1, $2...
		val = replacePlaceholders(val, submatches)

		switch tag {
		case "p":
			fp.ProductName = val
		case "v":
			fp.Version = val
		case "i":
			fp.Info = val
		case "h":
			fp.Hostname = val
		case "o":
			fp.OperatingSystem = val
		case "d":
			fp.DeviceType = val
		case "cpe:":
			fp.CPE = val // TODO: Handle multiple CPEs
		}
	}
}

func replacePlaceholders(s string, submatches []string) string {
	if !strings.Contains(s, "$") {
		return s
	}
	for i, match := range submatches {
		placeholder := fmt.Sprintf("$%d", i)
		s = strings.ReplaceAll(s, placeholder, match)
	}
	return s
}

func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
