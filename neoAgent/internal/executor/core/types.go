package core

// ToolScanResult 工具执行的标准化中间结果
// Parser 的职责是将 XML/JSON/Text 转换为这个结构
type ToolScanResult struct {
	ToolName  string `json:"tool_name"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    string `json:"status"` // success, failed

	// 原始输出 (可选，用于 Debug)
	RawOutput string `json:"raw_output,omitempty"`

	// --- 标准化资产数据 (Flattened) ---
	// Parser 必须尽力将结果映射到以下切片中

	Hosts []HostInfo `json:"hosts,omitempty"` // 存活主机
	Ports []PortInfo `json:"ports,omitempty"` // 开放端口
	Webs  []WebInfo  `json:"webs,omitempty"`  // Web 服务
	Vulns []VulnInfo `json:"vulns,omitempty"` // 漏洞/风险
}

type HostInfo struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
}

type PortInfo struct {
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	Proto   string `json:"proto"`   // tcp/udp
	Service string `json:"service"` // http, ssh
	Product string `json:"product"` // nginx
	Version string `json:"version"` // 1.14.2
	Banner  string `json:"banner"`
}

type WebInfo struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	StatusCode int    `json:"status_code"`
	TechStack  string `json:"tech_stack"` // JSON string of detected technologies
}

type VulnInfo struct {
	ID          string `json:"id"`           // CVE-2021-44228
	Name        string `json:"name"`         // Log4j RCE
	Severity    string `json:"severity"`     // critical, high
	Description string `json:"description"`
	Evidence    string `json:"evidence"` // Proof of concept or output
}
