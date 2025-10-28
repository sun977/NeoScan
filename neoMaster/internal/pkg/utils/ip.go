package utils

import (
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
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

// 从Gin上下文获取客户端IP
func GetClientIP(c *gin.Context) string {
	clientIPRaw := c.GetHeader("X-Forwarded-For")
	if clientIPRaw == "" {
		clientIPRaw = c.GetHeader("X-Real-IP")
	}
	if clientIPRaw == "" {
		clientIPRaw = c.ClientIP()
	}
	return NormalizeIP(clientIPRaw)
}

// IsIPInWhitelist 检查客户端IP是否在白名单中
// 参数:
//   - clientIP: 客户端IP地址
//   - whitelist: IP白名单列表，支持单IP和CIDR格式
//
// 返回: 如果IP在白名单中返回true，否则返回false
//
// 支持格式:
//   - 单IP: "192.168.1.100"
//   - CIDR: "192.168.1.0/24"
//   - IPv6: "::1", "2001:db8::/32"
func IsIPInWhitelist(clientIP string, whitelist []string) bool {
	if len(whitelist) == 0 {
		return false // 空白名单拒绝所有IP
	}

	// 标准化客户端IP
	normalizedClientIP := NormalizeIP(clientIP)
	if normalizedClientIP == "" {
		return false // 无效IP
	}

	// 解析客户端IP
	clientIPParsed := net.ParseIP(normalizedClientIP)
	if clientIPParsed == nil {
		return false // 无效IP格式
	}

	// 遍历白名单
	for _, allowedIP := range whitelist {
		allowedIP = strings.TrimSpace(allowedIP)
		if allowedIP == "" {
			continue
		}

		if strings.Contains(allowedIP, "/") {
			// CIDR格式：192.168.1.0/24
			if isIPInCIDR(clientIPParsed, allowedIP) {
				return true
			}
		} else {
			// 单IP格式：192.168.1.100
			allowedIPParsed := net.ParseIP(allowedIP)
			if allowedIPParsed != nil && clientIPParsed.Equal(allowedIPParsed) {
				return true
			}
		}
	}

	return false
}

// isIPInCIDR 检查IP是否在CIDR范围内
// 参数:
//   - ip: 要检查的IP地址（已解析）
//   - cidr: CIDR格式的网络范围，如 "192.168.1.0/24"
//
// 返回: 如果IP在CIDR范围内返回true，否则返回false
func isIPInCIDR(ip net.IP, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false // 无效的CIDR格式
	}

	return network.Contains(ip)
}

// ValidateIPWhitelistConfig 验证IP白名单配置格式
// 参数:
//   - whitelist: IP白名单列表
//
// 返回: 如果所有IP格式都有效返回nil，否则返回错误信息
func ValidateIPWhitelistConfig(whitelist []string) error {
	for i, ipStr := range whitelist {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		if strings.Contains(ipStr, "/") {
			// 验证CIDR格式
			_, _, err := net.ParseCIDR(ipStr)
			if err != nil {
				return fmt.Errorf("invalid CIDR format at index %d: %s (%v)", i, ipStr, err)
			}
		} else {
			// 验证单IP格式
			if net.ParseIP(ipStr) == nil {
				return fmt.Errorf("invalid IP format at index %d: %s", i, ipStr)
			}
		}
	}

	return nil
}

// GetIPWhitelistSummary 获取IP白名单摘要信息（用于日志记录）
// 参数:
//   - whitelist: IP白名单列表
//
// 返回: 白名单摘要字符串
func GetIPWhitelistSummary(whitelist []string) string {
	if len(whitelist) == 0 {
		return "empty"
	}

	singleIPs := 0
	cidrRanges := 0

	for _, ip := range whitelist {
		if strings.Contains(ip, "/") {
			cidrRanges++
		} else {
			singleIPs++
		}
	}

	return fmt.Sprintf("%d single IPs, %d CIDR ranges", singleIPs, cidrRanges)
}
