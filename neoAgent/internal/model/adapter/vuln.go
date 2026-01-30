package adapter

// VulnFindingAttributes 对应 vuln_finding 契约
type VulnFindingAttributes struct {
	Findings []VulnInfo `json:"findings"`
}

type VulnInfo struct {
	IP          string `json:"ip"`
	Port        int    `json:"port,omitempty"`
	URL         string `json:"url,omitempty"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	CVE         string `json:"cve,omitempty"`
	Severity    string `json:"severity"` // critical, high, medium, low, info
	Confidence  string `json:"confidence,omitempty"`
	Description string `json:"description,omitempty"`
	Solution    string `json:"solution,omitempty"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

// PocScanAttributes 对应 poc_scan 契约
type PocScanAttributes struct {
	PocResults []PocResult `json:"poc_results"`
}

type PocResult struct {
	IP               string `json:"ip"`
	PocID            string `json:"poc_id"`
	Target           string `json:"target"`
	Status           string `json:"status"` // confirmed, not_vulnerable, failed
	Severity         string `json:"severity,omitempty"`
	Payload          string `json:"payload,omitempty"`
	ResponseSnapshot string `json:"response_snapshot,omitempty"`
}

// PasswordAuditAttributes 对应 password_audit 契约
type PasswordAuditAttributes struct {
	Accounts []AccountInfo   `json:"accounts"`
	Policy   *PasswordPolicy `json:"policy,omitempty"`
}

type AccountInfo struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Service    string `json:"service"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Success    bool   `json:"success"`
	RootAccess bool   `json:"root_access,omitempty"`
}

type PasswordPolicy struct {
	MaxAttempts int `json:"max_attempts"`
}
