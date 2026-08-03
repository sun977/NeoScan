package model

import (
	"fmt"
	"time"
)

// IpAliveResult IP存活扫描结果
type IpAliveResult struct {
	IP       string        `json:"ip"`
	Alive    bool          `json:"alive"`
	RTT      time.Duration `json:"rtt,omitempty"`
	TTL      int           `json:"ttl,omitempty"`
	Hostname string        `json:"hostname,omitempty"`
	OS       string        `json:"os,omitempty"`
}

// Headers 实现 TabularData 接口
// RTT 单位 毫秒 ms
// IP        | Status | OS    | RTT  | TTL | Hostname
// 127.0.0.1 | UP     | Linux | 10ms | 64  | localhost
func (r IpAliveResult) Headers() []string {
	// 表头列
	return []string{"IP", "Status", "OS", "RTT", "TTL", "Hostname"}
}

// Rows 实现 TabularData 接口
func (r IpAliveResult) Rows() [][]string {
	status := "DOWN"
	if r.Alive {
		status = "UP"
	}

	rtt := "N/A"
	if r.RTT > 0 {
		// 统一使用 ms 单位，保留两位小数
		// 不要使用 r.RTT.String()，因为它会自动切换单位(µs/ms/s)，导致列表对齐混乱
		rtt = fmt.Sprintf("%.2fms", float64(r.RTT.Microseconds())/1000.0)
	}

	ttl := "N/A"
	if r.TTL > 0 {
		ttl = fmt.Sprintf("%d", r.TTL)
	}

	return [][]string{{r.IP, status, r.OS, rtt, ttl, r.Hostname}}
}

// PortServiceResult 端口服务扫描结果
type PortServiceResult struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	Status     string `json:"status"` // Open/Closed
	Service    string `json:"service"`
	Product    string `json:"product,omitempty"`
	Version    string `json:"version,omitempty"`
	Info       string `json:"info,omitempty"`
	Hostname   string `json:"hostname,omitempty"`
	OS         string `json:"os,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	CPE        string `json:"cpe,omitempty"`
	Banner     string `json:"banner,omitempty"`
}

func (r PortServiceResult) Headers() []string {
	return []string{"IP", "Port", "Proto", "State", "Service", "Version", "OS"}
}

func (r PortServiceResult) Rows() [][]string {
	version := r.Product
	if r.Version != "" {
		version += " " + r.Version
	}
	if r.Info != "" {
		version += " (" + r.Info + ")"
	}
	return [][]string{{r.IP, fmt.Sprintf("%d", r.Port), r.Protocol, r.Status, r.Service, version, r.OS}}
}

// OsInfo 操作系统识别结果
type OsInfo struct {
	Name           string `json:"name"`            // OS名称 (Windows, Linux, etc.)
	Family         string `json:"family"`          // OS家族 (Windows, Unix, Cisco, etc.)
	Version        string `json:"version"`         // 版本号
	Accuracy       int    `json:"accuracy"`        // 置信度 (0-100)
	Fingerprint    string `json:"fingerprint"`     // 指纹摘要 (用于 CLI 展示)
	RawFingerprint string `json:"raw_fingerprint"` // 完整指纹数据 (用于导出/调试)
	Source         string `json:"source"`          // 识别来源 (TTL, Service, Stack)
}

func (r OsInfo) Headers() []string {
	return []string{"Name", "Family", "Version", "Accuracy", "Source", "Fingerprint"}
}

func (r OsInfo) Rows() [][]string {
	return [][]string{{r.Name, r.Family, r.Version, fmt.Sprintf("%d%%", r.Accuracy), r.Source, r.Fingerprint}}
}

// WebResult Web扫描结果
type WebResult struct {
	URL             string            `json:"url"`
	IP              string            `json:"ip"`
	Port            int               `json:"port"`
	Title           string            `json:"title"`
	StatusCode      int               `json:"status_code"`
	ContentLength   int64             `json:"content_length"`
	ResponseHeaders map[string]string `json:"headers,omitempty"`
	TechStack       []string          `json:"tech_stack,omitempty"` // 识别到的技术栈
	Screenshot      string            `json:"screenshot,omitempty"` // Base64
	Favicon         string            `json:"favicon,omitempty"`    // Base64

	// --- 以下为爬虫功能新增字段，均为 omitempty，不影响现有序列化 ---
	Depth  int        `json:"depth,omitempty"`  // 爬取深度，0 = 首页
	Forms  []FormInfo `json:"forms,omitempty"`  // 表单/输入点
	Params []string   `json:"params,omitempty"` // URL Query 参数名集合
	Leaks  []LeakInfo `json:"leaks,omitempty"`  // 被动泄露检测结果

	// --- 以下为边缘网络组件识别功能新增字段，均为 omitempty，不影响现有序列化 ---
	EdgeComponents []EdgeComponent `json:"edge_components,omitempty"` // 命中的边缘网络组件列表（CDN/WAF/...），一个目标可能同时命中多个
}

// IsEdgeNode 判断是否命中任意边缘网络组件。调用方（如 Web 扫描器判断要不要
// 跳过截图/深度爬取）通常不关心具体命中的是 CDN 还是 WAF，只关心"这是不是
// 一个边缘节点"，用这个方法即可，不需要遍历 EdgeComponents。
func (r WebResult) IsEdgeNode() bool {
	return len(r.EdgeComponents) > 0
}

// FormInfo 表单信息（攻击面输入点）
type FormInfo struct {
	Action string   `json:"action"`
	Method string   `json:"method"`
	Fields []string `json:"fields"`
}

// LeakInfo 被动泄露检测命中信息
type LeakInfo struct {
	Type    string `json:"type"`              // aws_ak / aliyun_ak / jwt / internal_ip / ...
	Match   string `json:"match"`             // 脱敏后的命中内容，禁止存储明文密钥
	Context string `json:"context,omitempty"` // 命中上下文片段（可选）
}

// EdgeComponent 描述命中的一个边缘网络组件（CDN/WAF/反 DDoS 等）。
// 一个 WebResult 可以同时命中多个（如 Cloudflare 常见 CDN+WAF 二合一），
// 因此 WebResult 里用切片承载，不用一个组件类型对应一对标量字段。
type EdgeComponent struct {
	Type     string `json:"type"`     // "cdn" / "waf"，字符串而非枚举类型，避免跨包引入类型依赖
	Provider string `json:"provider"` // 厂商名，如 "Cloudflare"
}

// Headers 实现 TabularData 接口
func (r WebResult) Headers() []string {
	return []string{"URL", "IP", "Port", "Status", "Len", "Title", "Server", "TechStack"}
}

// Rows 实现 TabularData 接口
func (r WebResult) Rows() [][]string {
	stack := ""
	if len(r.TechStack) > 0 {
		stack = fmt.Sprintf("%v", r.TechStack)
		// 简单的截断显示
		if len(stack) > 50 {
			stack = stack[:47] + "..."
		}
	}

	server := ""
	if r.ResponseHeaders != nil {
		if s, ok := r.ResponseHeaders["Server"]; ok {
			server = s
		} else if s, ok := r.ResponseHeaders["server"]; ok {
			server = s
		}
	}
	if len(server) > 30 {
		server = server[:27] + "..."
	}

	return [][]string{{
		r.URL,
		r.IP,
		fmt.Sprintf("%d", r.Port),
		fmt.Sprintf("%d", r.StatusCode),
		fmt.Sprintf("%d", r.ContentLength),
		r.Title,
		server,
		stack,
	}}
}

// VulnResult 漏洞扫描结果
type VulnResult struct {
	ID          string `json:"id"` // CVE-202X-XXXX
	Name        string `json:"name"`
	Severity    string `json:"severity"` // critical, high, medium, low
	Description string `json:"description"`
	Reference   string `json:"reference"`
}

// Headers 实现 TabularData 接口
func (r VulnResult) Headers() []string {
	return []string{"ID", "Name", "Severity"}
}

// Rows 实现 TabularData 接口
func (r VulnResult) Rows() [][]string {
	return [][]string{{r.ID, r.Name, r.Severity}}
}

// BruteResult 爆破结果
type BruteResult struct {
	Service  string `json:"service"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Success  bool   `json:"success"`
}

// Headers 实现 TabularData 接口
func (r BruteResult) Headers() []string {
	return []string{"Service", "Host", "Port", "Username", "Password"}
}

// Rows 实现 TabularData 接口
func (r BruteResult) Rows() [][]string {
	return [][]string{{
		r.Service,
		r.Host,
		fmt.Sprintf("%d", r.Port),
		r.Username,
		r.Password,
	}}
}

// BruteResults 结果集合，用于实现 TabularData 接口以便一次性打印所有结果
type BruteResults []BruteResult

// Headers 实现 TabularData 接口
func (rs BruteResults) Headers() []string {
	return []string{"Service", "Host", "Port", "Username", "Password"}
}

// Rows 实现 TabularData 接口
func (rs BruteResults) Rows() [][]string {
	var rows [][]string
	for _, r := range rs {
		rows = append(rows, r.Rows()...)
	}
	return rows
}
