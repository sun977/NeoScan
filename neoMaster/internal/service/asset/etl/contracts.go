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
	} `json:"services"`
}

// WebEndpointAttributes Web端点属性
type WebEndpointAttributes struct {
	Endpoints []struct {
		URL       string `json:"url"`
		Status    int    `json:"status"`
		Tech      string `json:"tech"`      // e.g. Node.js
		Framework string `json:"framework"` // e.g. Express
	} `json:"endpoints"`
}

// VulnFindingAttributes 漏洞发现属性
type VulnFindingAttributes struct {
	Findings []struct {
		ID          string `json:"id"`
		CVE         string `json:"cve"`
		Severity    string `json:"severity"`
		Confidence  string `json:"confidence"`
		EvidenceRef string `json:"evidence_ref"`
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
