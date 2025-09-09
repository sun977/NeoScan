package utils

import (
	"net"
	"strings"
)

// NormalizeIP 标准化IP地址：
// - 若是带端口的地址，去掉端口
// - 若是 X-Forwarded-For 列表，取第一个
// - 若是 IPv4-mapped IPv6 (::ffff:192.0.2.1)，转成纯 IPv4
// - 否则按原样返回（包括真 IPv6）
func NormalizeIP(input string) string {
	if input == "" {
		return ""
	}

	// 先按逗号切分（X-Forwarded-For 可能是列表）
	ip := strings.TrimSpace(strings.Split(input, ",")[0])

	// 去掉端口（host:port 或 [ipv6]:port）
	if h, _, err := net.SplitHostPort(ip); err == nil {
		ip = h
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}

	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}

	return parsed.String()
}
