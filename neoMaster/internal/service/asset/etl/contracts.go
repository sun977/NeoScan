/**
 * ETL 数据契约定义
 * @author: sun977
 * @date: 2025.01.06
 * @description: 定义 StageResult.Attributes 的预期 JSON 结构。
 * 这些结构体作为 Agent 和 Master 之间的数据交换协议 (Contract)。
 */
package etl

// IPAliveAttributes 探活阶段属性
type IPAliveAttributes struct {
	Alive     bool     `json:"alive"`
	Protocols []string `json:"protocols"` // e.g. ["icmp", "tcp"]
}

// PortScanAttributes 端口扫描属性 (Fast/Full)
type PortScanAttributes struct {
	Ports []struct {
		Port        int    `json:"port"`
		Proto       string `json:"proto"`
		State       string `json:"state"`        // open, closed, filtered
		ServiceHint string `json:"service_hint"` // 简单服务猜测 (http, ssh)
		Banner      string `json:"banner"`       // 服务横幅信息
	} `json:"ports"`
	Summary struct {
		OpenCount    int    `json:"open_count"`
		ScanStrategy string `json:"scan_strategy"`
		ElapsedMs    int    `json:"elapsed_ms"`
	} `json:"summary"`
}

// ServiceFingerprintAttributes 服务指纹属性
type ServiceFingerprintAttributes struct {
	Services []struct {
		Port    int    `json:"port"`
		Proto   string `json:"proto"`
		Name    string `json:"name"`    // e.g. OpenSSH
		Version string `json:"version"` // e.g. 7.9p1
		CPE     string `json:"cpe"`     // e.g. cpe:/a:openbsd:openssh:7.9p1
		Banner  string `json:"banner"`  // 服务横幅信息
	} `json:"services"`
}

// WebEndpointAttributes Web端点属性
type WebEndpointAttributes struct {
	Endpoints []struct {
		URL        string            `json:"url"`
		IP         string            `json:"ip"`
		Title      string            `json:"title"`
		Headers    map[string]string `json:"headers"`
		Screenshot string            `json:"screenshot"`
		TechStack  []string          `json:"tech_stack"`
		StatusCode int               `json:"status_code"`
		Favicon    string            `json:"favicon"`
	} `json:"endpoints"`
}

// VulnFindingAttributes 漏洞发现属性
type VulnFindingAttributes struct {
	Findings []struct {
		ID          string  `json:"id"`          // 漏洞ID (e.g., Scanner-ID)
		CVE         string  `json:"cve"`         // CVE编号 (e.g., CVE-2021-44228)
		Name        string  `json:"name"`        // 漏洞名称
		Type        string  `json:"type"`        // 漏洞类型
		Severity    string  `json:"severity"`    // 严重程度
		Description string  `json:"description"` // 描述
		Solution    string  `json:"solution"`    // 修复建议
		Confidence  float64 `json:"confidence"`  // 置信度
		Reference   string  `json:"reference"`   // 参考链接
		TargetType  string  `json:"target_type"` // host/service/web
		Port        int     `json:"port"`        // 关联端口 (for service target)
		URL         string  `json:"url"`         // 关联URL (for web target)
		Evidence    string  `json:"evidence"`    // 证据
	} `json:"findings"`
}

// PocScanAttributes PoC验证属性
type PocScanAttributes struct {
	PocResults []struct {
		PocID       string `json:"poc_id"`
		Target      string `json:"target"`
		Status      string `json:"status"` // confirmed, not_vulnerable
		Severity    string `json:"severity"`
		EvidenceRef string `json:"evidence_ref"`
	} `json:"poc_results"`
}

// PasswordAuditAttributes 密码审计属性
type PasswordAuditAttributes struct {
	Accounts []struct {
		Username     string `json:"username"`
		Service      string `json:"service"`
		Host         string `json:"host"`
		Port         int    `json:"port"`
		WeakPassword bool   `json:"weak_password"`
		Credential   string `json:"credential"` // masked or hash
		Success      bool   `json:"success"`
	} `json:"accounts"`
}

// SubdomainDiscoveryAttributes 子域发现属性
type SubdomainDiscoveryAttributes struct {
	Subdomains []struct {
		Host   string `json:"host"`
		IP     string `json:"ip"`
		Source string `json:"source"`
	} `json:"subdomains"`
}

// DirectoryScanAttributes 目录扫描属性
type DirectoryScanAttributes struct {
	Paths []struct {
		URL       string `json:"url"`
		Status    int    `json:"status"`
		Length    int    `json:"length"`
		Sensitive bool   `json:"sensitive"`
	} `json:"paths"`
}
