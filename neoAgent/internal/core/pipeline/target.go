package pipeline

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"neoagent/internal/pkg/logger"
)

// TargetGenerator 目标生成器
// 将用户输入 (CIDR, IP, Range, File, List) 转换为流式的 IP 通道
func GenerateTargets(input string) <-chan string {
	out := make(chan string, 100) // 带缓冲的 Channel

	go func() {
		defer close(out)

		// 1. 尝试作为文件读取
		if _, err := os.Stat(input); err == nil {
			file, err := os.Open(input)
			if err == nil {
				defer file.Close()
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line != "" {
						parseAndSend(line, out)
					}
				}
				return
			}
		}

		// 2. 尝试作为逗号分隔的列表
		if strings.Contains(input, ",") {
			parts := strings.Split(input, ",")
			for _, part := range parts {
				parseAndSend(strings.TrimSpace(part), out)
			}
			return
		}

		// 3. 处理单个条目 (CIDR, Range, IP, Domain)
		parseAndSend(input, out)
	}()

	return out
}

func parseAndSend(target string, out chan<- string) {
	// 忽略空行和注释
	if target == "" || strings.HasPrefix(target, "#") {
		return
	}

	// 1. CIDR (e.g., 192.168.1.0/24)
	if _, ipNet, err := net.ParseCIDR(target); err == nil {
		for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
			// 简单的过滤网络地址和广播地址逻辑
			// 这里为了简化，全部发送，由后续 Alive 模块去过滤
			out <- ip.String()
		}
		return
	}

	// 2. IP Range (e.g., 192.168.1.1-192.168.1.10)
	if strings.Contains(target, "-") {
		parts := strings.Split(target, "-")
		if len(parts) == 2 {
			startIP := net.ParseIP(strings.TrimSpace(parts[0]))
			endIP := net.ParseIP(strings.TrimSpace(parts[1]))
			if startIP != nil && endIP != nil {
				for ip := startIP; bytesCompare(ip, endIP) <= 0; inc(ip) {
					out <- ip.String()
				}
				return
			}
		}
	}

	// 3. Single IP
	if ip := net.ParseIP(target); ip != nil {
		out <- ip.String()
		return
	}

	// 4. Domain (解析为 IP)
	if ips, err := net.LookupHost(target); err == nil {
		for _, ip := range ips {
			out <- ip
		}
		return
	}

	// 无法解析
	logger.Warn(fmt.Sprintf("Skipping invalid target: %s", target))
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func bytesCompare(a, b []byte) int {
	if len(a) != len(b) {
		return len(a) - len(b)
	}
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
