// Package models 定义工具适配层的标准化中间结果结构
// 所有的工具 Parser 都必须将原始输出转换为此结构
//
// Design Philosophy:
// 1. Flattened (扁平化): 避免深层嵌套，便于序列化和数据库映射
// 2. Strongly Typed (强类型): 核心字段强类型，扩展字段使用 JSON
// 3. Lossless (无损): 保留 RawOutput 以便 Debug
package models

// ToolScanResult 工具执行的标准化中间结果
type ToolScanResult struct {
	ToolName  string `json:"tool_name"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    string `json:"status"` // success, failed

	// 原始输出 (可选，用于 Debug 或存储原始证据)
	RawOutput string `json:"raw_output,omitempty"`

	// 错误信息 (如果 Status == failed)
	Error string `json:"error,omitempty"`

	// --- 标准化资产数据 ---

	Hosts []HostInfo `json:"hosts,omitempty"` // 存活主机
	Ports []PortInfo `json:"ports,omitempty"` // 开放端口
	Webs  []WebInfo  `json:"webs,omitempty"`  // Web 服务
	Vulns []VulnInfo `json:"vulns,omitempty"` // 漏洞/风险
}

// HostInfo 主机信息
type HostInfo struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Status   string `json:"status"` // up, down
	TTL      int    `json:"ttl,omitempty"`
}

// PortInfo 端口信息
type PortInfo struct {
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	Proto   string `json:"proto"`   // tcp, udp
	State   string `json:"state"`   // open, closed, filtered
	Service string `json:"service"` // http, ssh, mysql
	Product string `json:"product"` // nginx
	Version string `json:"version"` // 1.14.2
	Banner  string `json:"banner"`
	CPE     string `json:"cpe"`
}

// WebInfo Web 服务信息
type WebInfo struct {
	URL        string              `json:"url"`
	IP         string              `json:"ip"`
	Port       int                 `json:"port"`
	Title      string              `json:"title"`
	StatusCode int                 `json:"status_code"`
	TechStack  []string            `json:"tech_stack"` // 指纹列表: ["jQuery", "Nginx"]
	Headers    map[string][]string `json:"headers,omitempty"`
}

// VulnInfo 漏洞/风险信息
type VulnInfo struct {
	IP          string  `json:"ip"`
	Port        int     `json:"port"`
	URL         string  `json:"url,omitempty"`
	TemplateID  string  `json:"template_id"` // 扫描插件ID (e.g., CVE-2023-xxxx)
	Name        string  `json:"name"`
	Severity    string  `json:"severity"` // low, medium, high, critical
	Description string  `json:"description"`
	Proof       string  `json:"proof"` // 验证证据 (Request/Response snippet)
	Reference   string  `json:"reference"`
}
