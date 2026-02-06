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
}

// Headers 实现 TabularData 接口
func (r WebResult) Headers() []string {
	return []string{"URL", "Status", "Title", "TechStack"}
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
	return [][]string{{r.URL, fmt.Sprintf("%d", r.StatusCode), r.Title, stack}}
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
