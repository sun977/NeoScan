package nmap_service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"neoagent/internal/pkg/logger"

	"github.com/dlclark/regexp2" // 用于原生支持 PCRE 正则表达式，兼容 Nmap 规则
	// 若使用regexp的话会报错，Qscan的gonmap的处理逻辑是使用repairNMAPString()函数进行了强制转换和清洗
)

// Regexps for parsing nmap-service-probes
var (
	probeRegexp  = regexp.MustCompile(`^Probe ([a-zA-Z0-9]+) ([^ ]+) q\|([^|]*)\|`)
	matchRegexps = []*regexp.Regexp{
		regexp.MustCompile(`^([a-zA-Z0-9-_./]+) m\|([^|]+)\|([is]{0,2})(?: (.*))?$`),
		regexp.MustCompile(`^([a-zA-Z0-9-_./]+) m=([^=]+)=([is]{0,2})(?: (.*))?$`),
		regexp.MustCompile(`^([a-zA-Z0-9-_./]+) m%([^%]+)%([is]{0,2})(?: (.*))?$`),
		regexp.MustCompile(`^([a-zA-Z0-9-_./]+) m@([^@]+)@([is]{0,2})(?: (.*))?$`),
	}
	softMatchRegexps = []*regexp.Regexp{
		regexp.MustCompile(`^([a-zA-Z0-9-_./]+) m\|([^|]+)\|([is]{0,2})(?: (.*))?$`),
		regexp.MustCompile(`^([a-zA-Z0-9-_./]+) m=([^=]+)=([is]{0,2})(?: (.*))?$`),
		regexp.MustCompile(`^([a-zA-Z0-9-_./]+) m%([^%]+)%([is]{0,2})(?: (.*))?$`),
		regexp.MustCompile(`^([a-zA-Z0-9-_./]+) m@([^@]+)@([is]{0,2})(?: (.*))?$`),
	}
	portsRegexp    = regexp.MustCompile(`^ports ([0-9,-]+)`)
	sslportsRegexp = regexp.MustCompile(`^sslports ([0-9,-]+)`)
	rarityRegexp   = regexp.MustCompile(`^rarity ([0-9]+)`)
	fallbackRegexp = regexp.MustCompile(`^fallback ([a-zA-Z0-9,]+)`)
)

// ParseNmapServiceProbes 解析 Nmap 服务探测规则内容
func ParseNmapServiceProbes(content string) (map[string]*Probe, []string, error) {
	lines := strings.Split(content, "\n")
	probes := make(map[string]*Probe)
	var probeSort []string
	var currentProbe *Probe

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "Probe ") {
			if currentProbe != nil {
				probes[currentProbe.Name] = currentProbe
				probeSort = append(probeSort, currentProbe.Name)
			}
			p, err := parseProbeLine(line)
			if err != nil {
				logger.Warn(fmt.Sprintf("Failed to parse probe line: %s, error: %v", line, err))
				continue
			}
			currentProbe = p
			continue
		}

		if currentProbe == nil {
			continue
		}

		if strings.HasPrefix(line, "match ") {
			m := parseMatchLine(line[6:], false)
			if m != nil {
				currentProbe.MatchGroup = append(currentProbe.MatchGroup, m)
			}
		} else if strings.HasPrefix(line, "softmatch ") {
			m := parseMatchLine(line[10:], true)
			if m != nil {
				currentProbe.SoftMatchGroup = append(currentProbe.SoftMatchGroup, m)
			}
		} else if strings.HasPrefix(line, "ports ") {
			currentProbe.Ports = ParsePortList(line[6:])
		} else if strings.HasPrefix(line, "sslports ") {
			currentProbe.SslPorts = ParsePortList(line[9:])
		} else if strings.HasPrefix(line, "rarity ") {
			r, _ := strconv.Atoi(line[7:])
			currentProbe.Rarity = r
		} else if strings.HasPrefix(line, "fallback ") {
			currentProbe.Fallback = strings.Split(line[9:], ",")
		}
	}

	// Add last probe
	if currentProbe != nil {
		probes[currentProbe.Name] = currentProbe
		probeSort = append(probeSort, currentProbe.Name)
	}

	return probes, probeSort, nil
}

func parseProbeLine(line string) (*Probe, error) {
	matches := probeRegexp.FindStringSubmatch(line)
	if len(matches) != 4 {
		return nil, errors.New("invalid probe format")
	}

	// Unescape the probe string
	rawString := matches[3]
	// TODO: Implement full unescaping logic (like qscan's repairNMAPString)
	// For now, simple replacements
	rawString = strings.ReplaceAll(rawString, `\r`, "\r")
	rawString = strings.ReplaceAll(rawString, `\n`, "\n")
	rawString = strings.ReplaceAll(rawString, `\0`, "\000")
	// Add more as needed...

	return &Probe{
		Protocol:    matches[1],
		Name:        matches[2],
		ProbeString: rawString,
		Wait:        6 * time.Second, // Default wait time
	}, nil
}

func parseMatchLine(line string, isSoft bool) *Match {
	var regx *regexp.Regexp
	for _, r := range matchRegexps {
		if r.MatchString(line) {
			regx = r
			break
		}
	}
	if regx == nil {
		return nil
	}

	args := regx.FindStringSubmatch(line)
	service := args[1]
	pattern := args[2]
	opt := args[3]
	info := args[4] // Version info template

	// Compile regex
	re, err := compilePattern(pattern, opt)
	if err != nil {
		logger.Debug(fmt.Sprintf("Invalid regex pattern: %s, error: %v", pattern, err))
		return nil
	}

	return &Match{
		IsSoft:              isSoft,
		Service:             service,
		Pattern:             pattern,
		PatternRegexp:       re,
		VersionInfoTemplate: info,
	}
}

func compilePattern(pattern, opt string) (*regexp2.Regexp, error) {
	// Simple wrapper for Go regexp
	// Note: Nmap uses PCRE, Go uses RE2. Some patterns might fail.
	// We should try to adapt or ignore failed ones.

	if strings.Contains(opt, "i") {
		pattern = "(?i)" + pattern
	}
	if strings.Contains(opt, "s") {
		pattern = "(?s)" + pattern
	}

	// Use implicit options: None (flags handled in pattern)
	// We should use 0 as options for now, flags are in pattern.
	re, err := regexp2.Compile(pattern, 0)
	if err != nil {
		return nil, err
	}

	// Set a reasonable timeout to prevent ReDoS (e.g., 100ms)
	re.MatchTimeout = time.Millisecond * 100
	return re, nil
}

func ParsePortList(s string) []int {
	var ports []int
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Alias support
		switch strings.ToLower(part) {
		case "top100":
			ports = append(ports, Top100Ports...)
			continue
		case "top1000":
			ports = append(ports, Top1000Ports...)
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, _ := strconv.Atoi(rangeParts[0])
				end, _ := strconv.Atoi(rangeParts[1])
				for i := start; i <= end; i++ {
					ports = append(ports, i)
				}
			}
		} else {
			p, _ := strconv.Atoi(part)
			if p > 0 {
				ports = append(ports, p)
			}
		}
	}
	return ports
}
