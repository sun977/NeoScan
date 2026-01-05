// IP工具包
// 提供IP地址的标准化、校验和转换功能

package utils

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
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
			if isIPEqual(clientIPParsed, allowedIP) {
				return true
			}
		}
	}

	return false
}

// isIPInCIDR 检查IP是否在CIDR范围内
func isIPInCIDR(clientIP net.IP, cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false // CIDR格式无效
	}
	return ipNet.Contains(clientIP)
}

// isIPEqual 检查两个IP是否相等
func isIPEqual(clientIP net.IP, targetIPStr string) bool {
	targetIP := net.ParseIP(targetIPStr)
	if targetIP == nil {
		return false
	}
	return clientIP.Equal(targetIP)
}

// CIDR2IPs 将 CIDR 转换为 IP 列表
// 示例: "192.168.0.0/30" -> ["192.168.0.0", "192.168.0.1", "192.168.0.2", "192.168.0.3"]
// 注意: 仅支持 IPv4, 且不建议用于过大的网段
func CIDR2IPs(cidr string) ([]string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR format: %w", err)
	}

	// 仅支持 IPv4
	ip = ip.To4()
	if ip == nil {
		return nil, fmt.Errorf("only IPv4 is supported for CIDR expansion")
	}

	var ips []string
	for currentIP := ip.Mask(ipNet.Mask); ipNet.Contains(currentIP); inc(currentIP) {
		ips = append(ips, currentIP.String())
	}

	// 移除网络地址和广播地址? 通常扫描需要保留
	// 这里保留所有地址
	return ips, nil
}

// inc 增加 IP 地址
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Range2IPs 将 IP 范围字符串转换为 IP 列表
// 示例: "192.168.0.1-192.168.0.3" -> ["192.168.0.1", "192.168.0.2", "192.168.0.3"]
func Range2IPs(rangeStr string) ([]string, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid IP range format: %s", rangeStr)
	}

	startIPStr := strings.TrimSpace(parts[0])
	endIPStr := strings.TrimSpace(parts[1])

	startIP := net.ParseIP(startIPStr).To4()
	if startIP == nil {
		return nil, fmt.Errorf("invalid start IP: %s", startIPStr)
	}

	endIP := net.ParseIP(endIPStr).To4()
	if endIP == nil {
		return nil, fmt.Errorf("invalid end IP: %s", endIPStr)
	}

	// 转换为整数进行比较和迭代
	startInt := IP2Int(startIP)
	endInt := IP2Int(endIP)

	if startInt > endInt {
		return nil, fmt.Errorf("start IP must be less than or equal to end IP")
	}

	// 限制生成的 IP 数量，防止内存溢出 (例如限制为 65536)
	count := endInt - startInt + 1
	if count > 65536 {
		return nil, fmt.Errorf("IP range too large: %d addresses (max 65536)", count)
	}

	var ips []string
	for i := startInt; i <= endInt; i++ {
		ips = append(ips, Int2IP(i).String())
	}

	return ips, nil
}

// IP2Int 将 IPv4 地址转换为 uint32 整数
func IP2Int(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// Int2IP 将 uint32 整数转换为 IPv4 地址
func Int2IP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	ip[0] = byte(nn >> 24)
	ip[1] = byte(nn >> 16)
	ip[2] = byte(nn >> 8)
	ip[3] = byte(nn)
	return ip
}

// MergeIPs 合并 IP 列表并去重排序
func MergeIPs(ips []string) []string {
	uniqueIPs := make(map[string]struct{})
	for _, ip := range ips {
		if ip = strings.TrimSpace(ip); ip != "" {
			uniqueIPs[ip] = struct{}{}
		}
	}

	result := make([]string, 0, len(uniqueIPs))
	for ip := range uniqueIPs {
		result = append(result, ip)
	}

	// 排序
	sort.Slice(result, func(i, j int) bool {
		ip1 := net.ParseIP(result[i])
		ip2 := net.ParseIP(result[j])
		if ip1 == nil || ip2 == nil {
			return result[i] < result[j]
		}
		return IP2Int(ip1) < IP2Int(ip2)
	})

	return result
}

// ==========================================
// 验证与解析函数 (Validation & Parsing)
// ==========================================

// ParseIPPairs 解析 IP 范围字符串为 IP 列表
// 支持格式:
// - 完整范围: 192.168.0.1-192.168.2.255
// - 简写范围: 192.168.0.1-255 (等同于 192.168.0.1-192.168.0.255)
// - 混合简写: 192.168.0.1-2.255 (等同于 192.168.0.1-192.168.2.255)
func ParseIPPairs(ipRange string) ([]string, error) {
	parts := strings.Split(ipRange, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid IP range format: %s", ipRange)
	}

	startIPStr := strings.TrimSpace(parts[0])
	endIPStr := strings.TrimSpace(parts[1])

	// 解析起始 IP
	startIP := net.ParseIP(startIPStr).To4()
	if startIP == nil {
		return nil, fmt.Errorf("invalid start IP: %s", startIPStr)
	}

	// 解析结束 IP (处理简写逻辑)
	var endIP net.IP
	if strings.Contains(endIPStr, ".") {
		// 可能是完整 IP 或部分段 (e.g. 2.255)
		dots := strings.Count(endIPStr, ".")
		if dots == 3 {
			// 完整 IP: 192.168.2.255
			endIP = net.ParseIP(endIPStr).To4()
		} else {
			// 部分段: 2.255 -> 补全前缀
			// startIP: 192.168.0.1
			// endIPStr: 2.255
			// result: 192.168.2.255
			startIPParts := strings.Split(startIPStr, ".")
			endIPParts := strings.Split(endIPStr, ".")

			// 需要补全的前缀段数 = 4 - 结束IP的段数
			prefixLen := 4 - len(endIPParts)
			if prefixLen < 0 {
				return nil, fmt.Errorf("invalid end IP format: %s", endIPStr)
			}

			fullEndIPStr := strings.Join(startIPParts[:prefixLen], ".") + "." + endIPStr
			endIP = net.ParseIP(fullEndIPStr).To4()
		}
	} else {
		// 纯数字简写: 255 -> 192.168.0.255
		// 只有最后一段不同
		startIPParts := strings.Split(startIPStr, ".")
		fullEndIPStr := strings.Join(startIPParts[:3], ".") + "." + endIPStr
		endIP = net.ParseIP(fullEndIPStr).To4()
	}

	if endIP == nil {
		return nil, fmt.Errorf("invalid end IP: %s", endIPStr)
	}

	// 转换为整数进行比较和迭代
	startInt := IP2Int(startIP)
	endInt := IP2Int(endIP)

	if startInt > endInt {
		return nil, fmt.Errorf("start IP must be less than or equal to end IP")
	}

	// 限制生成的 IP 数量，防止内存溢出
	count := endInt - startInt + 1
	if count > 65536 {
		return nil, fmt.Errorf("IP range too large: %d addresses (max 65536)", count)
	}

	var ips []string
	for i := startInt; i <= endInt; i++ {
		ips = append(ips, Int2IP(i).String())
	}

	return ips, nil
}

// IsURL 检查字符串是否为 URL
// 格式: protocol://netloc/path
// 支持带端口: protocol://netloc:port/path
func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// IsIPPort 检查字符串是否为 IP:Port 格式
// 示例: 192.168.0.1:8080
func IsIPPort(str string) bool {
	host, port, err := net.SplitHostPort(str)
	if err != nil {
		return false
	}
	return IsIP(host) && IsPort(port)
}

// IsDomainPort 检查字符串是否为 Domain:Port 格式
// 示例: example.com:8080
func IsDomainPort(str string) bool {
	host, port, err := net.SplitHostPort(str)
	if err != nil {
		return false
	}
	return IsDomain(host) && IsPort(port)
}

// IsNetlocPort 检查字符串是否为 [Domain or IP]:Port 格式
// 示例: 192.168.0.1:8080 或 example.com:8080
func IsNetlocPort(str string) bool {
	return IsIPPort(str) || IsDomainPort(str)
}

// IsCIDR 检查字符串是否为有效的 CIDR 格式 (IPv4)
// 示例: 192.168.0.0/24
func IsCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// IsIP 检查字符串是否为有效的 IP 地址 (IPv4 或 IPv6)
func IsIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsIPv4 检查字符串是否为有效的 IPv4 地址
func IsIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() != nil
}

// IsIPv6 检查字符串是否为有效的 IPv6 地址
func IsIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() == nil
}

// IsPort 检查字符串是否为有效的端口号 (0-65535)
func IsPort(port string) bool {
	p, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return p >= 0 && p <= 65535
}

// 域名正则
var domainRegex = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// IsDomain 检查字符串是否为有效的域名
// 注意: 这是一个简化的检查，不覆盖所有 RFC 规则
func IsDomain(domain string) bool {
	if len(domain) > 255 {
		return false
	}
	if IsIP(domain) {
		return false
	}
	return domainRegex.MatchString(domain) || domain == "localhost"
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

// IsIPRange 检查字符串是否为 IP 范围 (CIDR 或 start-end)
func IsIPRange(s string) bool {
	return strings.Contains(s, "-") || strings.Contains(s, "/")
}

// IPCompare 比较两个 IPv4 地址
// 返回 -1 if ip1 < ip2, 0 if equal, 1 if ip1 > ip2
func IPCompare(ip1, ip2 net.IP) int {
	int1 := IP2Int(ip1)
	int2 := IP2Int(ip2)
	if int1 < int2 {
		return -1
	}
	if int1 > int2 {
		return 1
	}
	return 0
}

// CheckIPInRange 检查目标 IP 是否在指定的 IP 范围内
// 支持格式:
// - 单 IP: "192.168.1.1"
// - CIDR: "192.168.1.0/24"
// - 范围: "192.168.1.1-192.168.1.5"
func CheckIPInRange(targetIPStr, rangeStr string) (bool, error) {
	targetIP := net.ParseIP(targetIPStr)
	if targetIP == nil {
		return false, fmt.Errorf("invalid target IP: %s", targetIPStr)
	}

	rangeStr = strings.TrimSpace(rangeStr)

	// 1. CIDR
	if strings.Contains(rangeStr, "/") {
		_, ipNet, err := net.ParseCIDR(rangeStr)
		if err != nil {
			return false, err
		}
		return ipNet.Contains(targetIP), nil
	}

	// 2. Range
	if strings.Contains(rangeStr, "-") {
		parts := strings.Split(rangeStr, "-")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid range format: %s", rangeStr)
		}
		startIP := net.ParseIP(strings.TrimSpace(parts[0]))
		endIP := net.ParseIP(strings.TrimSpace(parts[1]))
		if startIP == nil || endIP == nil {
			return false, fmt.Errorf("invalid IP in range: %s", rangeStr)
		}
		return IPCompare(targetIP, startIP) >= 0 && IPCompare(targetIP, endIP) <= 0, nil
	}

	// 3. Single IP
	checkIP := net.ParseIP(rangeStr)
	if checkIP == nil {
		return false, fmt.Errorf("invalid IP format: %s", rangeStr)
	}
	return targetIP.Equal(checkIP), nil
}

// CheckDomainMatch 检查域名是否匹配规则
// 支持规则:
// - 精确匹配: example.com
// - 通配符前缀: *.example.com (匹配 a.example.com, example.com)
// - 点号前缀: .example.com (匹配 a.example.com)
func CheckDomainMatch(target, rule string) bool {
	target = strings.ToLower(target)
	rule = strings.ToLower(rule)

	if target == rule {
		return true
	}

	if strings.HasPrefix(rule, "*.") {
		// *.example.com matches example.com and api.example.com
		root := rule[2:]
		if target == root {
			return true
		}
		return strings.HasSuffix(target, "."+root)
	}

	if strings.HasPrefix(rule, ".") {
		// .example.com matches api.example.com
		return strings.HasSuffix(target, rule)
	}

	return false
}

// CheckTargetInScope 检查目标是否在范围内 --- 执行器 enforcer 调用
// 自动识别范围类型 (IP/CIDR/Range/Domain)
func CheckTargetInScope(target, scope string) (bool, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false, nil
	}

	// 预处理目标：去掉端口
	targetHost := target
	if h, _, err := net.SplitHostPort(target); err == nil {
		targetHost = h
	}

	// 1. 判断 Scope 类型是否为 IP 相关 (IP, CIDR, Range)
	if IsIP(scope) || IsCIDR(scope) || IsIPRange(scope) {
		// 如果 Scope 是 IP 类型，则 Target 必须是 IP 才能匹配
		// CheckIPInRange 内部会解析 Target IP，如果解析失败返回 error
		match, err := CheckIPInRange(targetHost, scope)
		if err != nil {
			// 如果错误是因为 Target 不是有效 IP，则视为不匹配，不返回错误
			if strings.Contains(err.Error(), "invalid target IP") {
				return false, nil
			}
			return false, err
		}
		return match, nil
	}

	// 2. 否则视为域名规则
	return CheckDomainMatch(targetHost, scope), nil
}
