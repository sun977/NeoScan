package adapter

// SubdomainDiscoveryAttributes 对应 subdomain_discovery 契约
type SubdomainDiscoveryAttributes struct {
	Subdomains []SubdomainInfo `json:"subdomains"`
}

type SubdomainInfo struct {
	Host       string `json:"host"`
	IP         string `json:"ip,omitempty"`
	Source     string `json:"source,omitempty"`
	CNAME      string `json:"cname,omitempty"`
	IsWildcard bool   `json:"is_wildcard,omitempty"`
}

// ProxyDetectionAttributes 对应 proxy_detection 契约
type ProxyDetectionAttributes struct {
	Proxies []ProxyInfo `json:"proxies"`
}

type ProxyInfo struct {
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	Type         string `json:"type"` // http, socks4, socks5
	Open         bool   `json:"open"`
	AuthRequired bool   `json:"auth_required,omitempty"`
}

// FileDiscoveryAttributes 对应 file_discovery 契约
type FileDiscoveryAttributes struct {
	Files []FileInfo `json:"files"`
}

type FileInfo struct {
	URL       string `json:"url"`
	Path      string `json:"path,omitempty"`
	Size      int64  `json:"size,omitempty"`
	MIME      string `json:"mime,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

// OtherScanAttributes 对应 other_scan 契约
type OtherScanAttributes struct {
	Summary string                 `json:"summary,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
