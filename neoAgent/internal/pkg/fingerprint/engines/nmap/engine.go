package nmap

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"neoagent/internal/core/lib/network/dialer"
)

// NmapEngine 实现了基于 Nmap 规则的服务指纹识别
type NmapEngine struct {
	probes       []*Probe
	probeMap     map[string]*Probe
	portProbeMap map[int][]*Probe // 端口到探针的映射 (优化查找)
	allProbes    []*Probe         // 所有探针 (按 rarity 排序)

	customRules string // 自定义规则内容

	initOnce sync.Once
	initErr  error
}

// NewNmapEngine 创建一个新的 Nmap 引擎
func NewNmapEngine() *NmapEngine {
	e := &NmapEngine{
		probeMap:     make(map[string]*Probe),
		portProbeMap: make(map[int][]*Probe),
	}
	return e
}

// SetRules 设置自定义规则 (必须在 Scan 之前调用)
func (e *NmapEngine) SetRules(content string) {
	e.customRules = content
}

// ensureInit 确保规则已加载
func (e *NmapEngine) ensureInit() error {
	e.initOnce.Do(func() {
		rules := e.customRules
		if rules == "" {
			rules = NmapServiceProbes
		}

		// 拼接 NmapCustomizeProbes (如果存在)
		// 注意：自定义规则应该在标准规则之后，或者根据 Nmap 逻辑，顺序可能影响匹配
		// Nmap 中自定义规则通常合并在 nmap-service-probes 文件中
		// 这里我们简单拼接
		if NmapCustomizeProbes != "" {
			rules += "\n" + NmapCustomizeProbes
		}

		// 解析 NmapServiceProbes
		if rules == "" {
			e.initErr = fmt.Errorf("nmap rules not found")
			return
		}

		probes, err := ParseNmapProbes(rules)
		if err != nil {
			e.initErr = fmt.Errorf("failed to parse nmap probes: %v", err)
			return
		}

		e.probes = probes

		// 构建索引
		for _, p := range probes {
			e.probeMap[p.Name] = p

			// 关联到 ports
			for _, port := range p.Ports {
				e.portProbeMap[port] = append(e.portProbeMap[port], p)
			}
			// 关联到 sslports
			for _, port := range p.SslPorts {
				e.portProbeMap[port] = append(e.portProbeMap[port], p)
			}
		}

		// 对所有探针按 rarity 排序
		e.allProbes = make([]*Probe, len(probes))
		copy(e.allProbes, probes)
		sort.Slice(e.allProbes, func(i, j int) bool {
			return e.allProbes[i].Rarity < e.allProbes[j].Rarity
		})

		// 对每个端口的探针列表也排序
		for port, list := range e.portProbeMap {
			sort.Slice(list, func(i, j int) bool {
				return list[i].Rarity < list[j].Rarity
			})
			// 去重
			e.portProbeMap[port] = uniqueProbes(list)
		}
	})
	return e.initErr
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

// Scan 执行服务指纹识别
// ip: 目标 IP
// port: 目标端口
// timeout: 单次探测超时时间
func (e *NmapEngine) Scan(ctx context.Context, ip string, port int, timeout time.Duration) (*FingerPrint, error) {
	if err := e.ensureInit(); err != nil {
		return nil, err
	}

	// 1. 获取适用该端口的探针
	probes := e.getProbesForPort(port)

	// 2. 遍历探针执行
	for _, probe := range probes {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		fingerprint := e.executeProbe(ctx, ip, port, probe, timeout)
		if fingerprint != nil {
			return fingerprint, nil
		}
	}

	return nil, nil // 未识别
}

func (e *NmapEngine) getProbesForPort(port int) []*Probe {
	// 1. 获取端口特定探针
	specific := e.portProbeMap[port]

	// 2. 获取通用探针 (Rarity <= 7)
	// 实际 Nmap 逻辑会更复杂，这里简化
	candidates := make([]*Probe, 0, len(specific)+len(e.allProbes))
	candidates = append(candidates, specific...)

	for _, p := range e.allProbes {
		if p.Rarity <= 7 {
			candidates = append(candidates, p)
		}
	}

	return uniqueProbes(candidates)
}

func (e *NmapEngine) executeProbe(ctx context.Context, ip string, port int, probe *Probe, timeout time.Duration) *FingerPrint {
	// 1. 建立连接 (使用全局 Dialer，支持代理)
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := dialer.Get().DialContext(dialCtx, "tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil
	}
	defer conn.Close()

	// 2. 发送探针数据
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
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && n == 0 {
		return nil
	}
	response := buf[:n]

	// 4. 匹配
	// 优先匹配当前 probe 的 MatchGroup
	for _, match := range probe.MatchGroup {
		// 编译正则 (如果在 Parse 时未编译)
		if match.PatternRegexp == nil {
			continue
		}

		// 执行正则匹配
		submatches := match.PatternRegexp.FindSubmatch(response)
		if submatches != nil {
			// 匹配成功，提取信息
			fp := &FingerPrint{
				ProbeName:        probe.Name,
				MatchRegexString: match.Pattern,
				Service:          match.Service,
			}

			// 解析 VersionInfo (e.g., p/OpenSSH/ v/8.2p1/)
			// 这里需要一个 helper 来解析 Nmap 的 Version 字符串格式并填充 fp
			// 暂时简单实现：如果 submatches 有捕获组，尝试填充
			// 真正的 Nmap Version 解析很复杂，涉及到 $1, $2 替换
			e.parseVersionInfo(fp, match.VersionInfo, submatches)

			return fp
		}
	}

	// 软匹配 (SoftMatch) - 暂时略过，因为 SoftMatch 主要是为了加速后续匹配
	// 如果匹配到 SoftMatch，应该限制后续只跑特定服务的探针

	return nil
}

// parseVersionInfo 解析版本信息字符串并填充 FingerPrint
// versionInfo 格式如: p/OpenSSH/ v/8.2p1/ i/Ubuntu/ o/Linux/
// submatches 是正则捕获组，用于替换 $1, $2 等
func (e *NmapEngine) parseVersionInfo(fp *FingerPrint, versionInfo string, submatches [][]byte) {
	// 简单的解析器，支持 p, v, i, h, o, d, cpe
	// 以及 $1, $2 替换

	// Helper to replace $n
	replacePlaceholders := func(s string) string {
		for i := 1; i < len(submatches); i++ {
			placeholder := fmt.Sprintf("$%d", i)
			ifContains := false
			// check if contains
			for j := 0; j < len(s)-1; j++ {
				if s[j] == '$' && s[j+1] == byte('0'+i) {
					ifContains = true
					break
				}
			}
			if ifContains {
				// 简单的字符串替换 (不够严谨，但够用)
				// 注意：Go 的 regexp ReplaceAllString 是基于 $1 的，这里我们可以直接用
				// 但 versionInfo 不是正则，是模板。
				// 我们手动替换
				// 实际上 Nmap 允许 $P(1) 这种 helper，这里暂不支持
				// 仅支持 $1 - $9
				s = replaceAll(s, placeholder, string(submatches[i]))
			}
		}
		return s
	}

	// 遍历 versionInfo
	// Nmap version string is like: p/val/ v/val/ ...
	// 分隔符不一定是 /，通常是第一个字符
	if len(versionInfo) < 3 {
		return
	}

	// 简单的状态机解析
	// 或者直接正则提取: ([pvihod]|cpe):/([^/]*)/
	// 但分隔符是动态的

	// 简化实现：假设都是 p/.../ 格式，且不包含嵌套
	// 真正的解析需要处理转义

	// 这里我们直接硬编码支持常见的
	// p/Product/ v/Version/ i/Info/ h/Hostname/ o/OS/ d/Device/ cpe:/CPE/

	cursor := 0
	length := len(versionInfo)

	for cursor < length {
		// 跳过空格
		if versionInfo[cursor] == ' ' {
			cursor++
			continue
		}

		// 识别类型
		typeChar := ""
		if strings.HasPrefix(versionInfo[cursor:], "cpe:") {
			typeChar = "cpe"
			cursor += 4
		} else {
			typeChar = string(versionInfo[cursor])
			cursor++
		}

		if cursor >= length {
			break
		}

		// 获取分隔符
		delimiter := versionInfo[cursor]
		cursor++

		// 寻找结束分隔符
		end := -1
		// 如果分隔符是 |，则结束符也是 |
		// 注意 Nmap 允许嵌套 |，例如 p|foo|bar||
		// 但通常不嵌套
		// 如果是 match 格式，通常是 p/value/
		// 我们从 cursor 开始找 delimiter
		// 注意：如果 value 中包含 delimiter，需要转义？Nmap 规则通常不转义，而是选择不冲突的 delimiter

		end = strings.IndexByte(versionInfo[cursor:], delimiter)

		if end == -1 {
			// 如果没找到，可能是格式错误，或者到了末尾
			break
		}

		val := versionInfo[cursor : cursor+end]
		val = replacePlaceholders(val)

		switch typeChar {
		case "s":
			// match 指令中的 's' 不是前缀，而是 flag (softmatch)
			// 但在这里 versionInfo 解析中，如果遇到 's' 且不是 p/v/i 等
			// 其实 Nmap 语法中，match <service> <pattern> <versioninfo>
			// versionInfo 是一系列 helper，如 p// v//
			// 有时候会有 s p//，这里的 s 其实是 match line 的一部分？
			// 不，match line 是: match <service> <pattern> [<flags>] [<versioninfo>]
			// 我们的 parser.go 可能把 flags 也当成了 versionInfo 的一部分
			// 让我们看看 parser.go 如何解析 match line
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
		case "cpe":
			fp.CPE = "cpe:" + val // Nmap cpe:/.../ -> cpe:...
		}

		cursor += end + 1
	}
}

func replaceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func unescapeString(s string) string {
	// Nmap probe string has generic C-style escapes
	// \0, \n, \r, \xHH
	// 简单处理：使用 strconv.Unquote (需要加引号)
	// 但 Nmap string 可能包含不合法的 go string 字符
	// 最好手动处理 \xHH

	// 极简实现：仅处理 \r, \n, \t, \0, \xHH
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'r':
				out = append(out, '\r')
			case 'n':
				out = append(out, '\n')
			case 't':
				out = append(out, '\t')
			case '0':
				out = append(out, 0)
			case 'x':
				if i+2 < len(s) {
					hex := s[i+1 : i+3]
					var v byte
					fmt.Sscanf(hex, "%2x", &v)
					out = append(out, v)
					i += 2
				}
			default:
				out = append(out, s[i])
			}
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}
