package nmap

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 正则表达式定义
var (
	// Probe 指令: Probe <protocol> <probename> <probestring>
	// 由于 probestring 包含分隔符，这里只匹配前缀
	probePrefixRegexp = regexp.MustCompile(`^Probe\s+(TCP|UDP)\s+([a-zA-Z0-9-_./]+)\s+q`)

	// Match 指令: match <service> <pattern> [<versioninfo>]
	// 同样只匹配前缀
	matchPrefixRegexp = regexp.MustCompile(`^(match|softmatch)\s+([a-zA-Z0-9-_./]+)\s+m`)

	// 辅助指令
	portsRegexp    = regexp.MustCompile(`^ports\s+(.*)$`)
	sslPortsRegexp = regexp.MustCompile(`^sslports\s+(.*)$`)
	rarityRegexp   = regexp.MustCompile(`^rarity\s+(\d+)$`)
	fallbackRegexp = regexp.MustCompile(`^fallback\s+(.*)$`)
)

// ParseNmapProbes 解析 nmap-service-probes 内容
func ParseNmapProbes(content string) ([]*Probe, error) {
	// 1. 预处理 (参考 gonmap 的 repairNMAPString)
	content = repairNmapString(content)

	lines := strings.Split(content, "\n")
	var probes []*Probe
	var currentProbe *Probe

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 处理 Probe 指令
		if strings.HasPrefix(line, "Probe ") {
			p, err := parseProbeLine(line)
			if err != nil {
				// 记录错误但继续? 或者直接返回错误
				// Nmap 规则文件很大，容错一点比较好
				continue
			}
			currentProbe = p
			probes = append(probes, p)
			continue
		}

		// 如果没有当前 Probe，忽略后续指令 (match/ports等必须依附于 Probe)
		if currentProbe == nil {
			continue
		}

		// 处理 match/softmatch
		if strings.HasPrefix(line, "match ") || strings.HasPrefix(line, "softmatch ") {
			m, err := parseMatchLine(line)
			if err != nil {
				continue
			}
			if strings.HasPrefix(line, "softmatch ") {
				m.IsSoft = true
				currentProbe.SoftMatchGroup = append(currentProbe.SoftMatchGroup, m)
			} else {
				currentProbe.MatchGroup = append(currentProbe.MatchGroup, m)
			}
			continue
		}

		// 处理 ports
		if matches := portsRegexp.FindStringSubmatch(line); len(matches) > 1 {
			currentProbe.Ports = ParsePortList(matches[1])
			continue
		}

		// 处理 sslports
		if matches := sslPortsRegexp.FindStringSubmatch(line); len(matches) > 1 {
			currentProbe.SslPorts = ParsePortList(matches[1])
			continue
		}

		// 处理 rarity
		if matches := rarityRegexp.FindStringSubmatch(line); len(matches) > 1 {
			rarity, _ := strconv.Atoi(matches[1])
			currentProbe.Rarity = rarity
			continue
		}

		// 处理 fallback
		if matches := fallbackRegexp.FindStringSubmatch(line); len(matches) > 1 {
			currentProbe.Fallback = matches[1]
			continue
		}
	}

	return probes, nil
}

// parseProbeLine 解析 Probe 行
// Probe TCP NULL q||
func parseProbeLine(line string) (*Probe, error) {
	// 1. 找到 q 的位置
	qIndex := strings.Index(line, " q")
	if qIndex == -1 {
		return nil, fmt.Errorf("invalid probe line: missing 'q' delimiter")
	}

	// 2. 解析前部分: Probe TCP NULL
	prefix := line[:qIndex]
	parts := strings.Fields(prefix)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid probe prefix: %s", prefix)
	}
	protocol := parts[1]
	name := parts[2]

	// 3. 解析 probe string: q|...|
	// q 后面紧跟的一个字符是分隔符
	remaining := line[qIndex+1:] // q|...|
	if len(remaining) < 3 {
		return nil, fmt.Errorf("invalid probe string format")
	}

	// 提取分隔符 (q后面的第一个字符)
	delimiter := remaining[1] // q|... -> |

	// 查找结束分隔符
	// 注意：内容中可能包含转义的分隔符，但 Nmap 格式通常比较规整
	// 这里简单实现，寻找最后一个分隔符
	endIndex := strings.LastIndexByte(remaining, delimiter)
	if endIndex <= 1 {
		return nil, fmt.Errorf("missing closing delimiter for probe string")
	}

	probeString := remaining[2:endIndex]

	return &Probe{
		Name:        name,
		Protocol:    protocol,
		ProbeString: probeString,
		Ports:       make(PortList, 0),
		SslPorts:    make(PortList, 0),
	}, nil
}

// parseMatchLine 解析 match 行
// match ftp m/^220.*FTP/ p/vsftpd/
func parseMatchLine(line string) (*Match, error) {
	// 1. 识别 service
	// match <service> <pattern> ...
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid match line")
	}
	service := parts[1]
	remaining := parts[2] // m/.../ ...

	// 2. 解析 pattern: m/.../ [is]
	if len(remaining) < 3 || remaining[0] != 'm' {
		return nil, fmt.Errorf("invalid match pattern start")
	}
	delimiter := remaining[1]

	// 寻找模式结束符
	// 这里需要处理转义字符，简单的 LastIndex 可能不够
	// 但 Nmap 规则通常是一行一个 match
	// 简单的做法：找到第二个分隔符
	patternEndIndex := -1
	for i := 2; i < len(remaining); i++ {
		if remaining[i] == delimiter && remaining[i-1] != '\\' {
			patternEndIndex = i
			break
		}
	}
	if patternEndIndex == -1 {
		return nil, fmt.Errorf("missing closing delimiter for match pattern")
	}

	pattern := remaining[2:patternEndIndex]

	// 提取 flags (i, s)
	// pattern 后面可能紧跟 flags，然后是空格
	rest := remaining[patternEndIndex+1:]
	flags := ""
	versionStart := 0

	// 简单的向前扫描空格
	// Nmap 格式：m|pattern|flags versioninfo
	// flags 是紧跟第二个分隔符的字母，例如 m|foo|s p/bar/

	// 如果 rest 为空，则没有 version info
	if len(rest) == 0 {
		// No flags, no version info
	} else {
		// 扫描直到遇到空格
		for i, c := range rest {
			if c == ' ' {
				versionStart = i + 1
				break
			}
			flags += string(c)
		}
		// 如果循环结束还没遇到空格，说明全是 flags，没有 version info (或者 version info 为空)
		if versionStart == 0 && len(flags) > 0 {
			// 检查 flags 是否真的全是字母 (i, s)
			// 有时候可能是 m|pattern| versioninfo 直接跟在后面？
			// 不，Nmap 规范是 m|pattern|[flags] [versioninfo]
			// 如果 rest 只是 "s"，那 flags="s"，versionStart=0
			// 这种情况下 versionInfo 为空
			versionStart = len(rest)
		}
	}

	// 编译正则 (需要处理 flags)
	// Go regexp 不支持 (?s) 这种内嵌 flag 放在外面?
	// Nmap 的 flags 是 i (case-insensitive) 和 s (dot matches newline)
	// 我们需要把它们转换成 Go 的 (?is) 前缀
	regexPrefix := ""
	if strings.Contains(flags, "i") {
		regexPrefix += "i"
	}
	if strings.Contains(flags, "s") {
		regexPrefix += "s"
	}
	if regexPrefix != "" {
		pattern = "(?" + regexPrefix + ")" + pattern
	}

	// 3. 解析 version info
	versionInfo := ""
	if versionStart > 0 && versionStart < len(rest) {
		versionInfo = rest[versionStart:]
	}

	// 尝试编译正则
	re, err := regexp.Compile(pattern)
	if err != nil {
		// 很多 Nmap 正则 Go 不支持 (Perl 语法)，这里可能需要容错或者后续处理
		// 暂时返回错误
		return nil, fmt.Errorf("invalid regex: %v", err)
	}

	return &Match{
		Service:       service,
		Pattern:       pattern,
		PatternRegexp: re,
		VersionInfo:   versionInfo,
	}, nil
}

// ParsePortList 解析端口列表字符串
// 80,443,8000-8010
func ParsePortList(expr string) PortList {
	var list PortList
	parts := strings.Split(expr, ",")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, _ := strconv.Atoi(rangeParts[0])
				end, _ := strconv.Atoi(rangeParts[1])
				for i := start; i <= end; i++ {
					list = append(list, i)
				}
			}
		} else {
			port, _ := strconv.Atoi(part)
			list = append(list, port)
		}
	}
	return list
}

// repairNmapString 修复/清理 Nmap 规则字符串 (参考 gonmap)
func repairNmapString(s string) string {
	s = strings.ReplaceAll(s, "${backquote}", "`")
	// 替换 Go 不支持的正则语法，或者 Nmap 特有的宏
	// 这里可以逐步添加更多替换规则
	s = strings.ReplaceAll(s, `\1`, `$1`) // 反向引用语法转换
	return s
}
