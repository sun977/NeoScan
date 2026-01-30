package adapter

// IpAliveAttributes 对应 ip_alive 契约
type IpAliveAttributes struct {
	Hosts   []HostInfo     `json:"hosts"`
	Summary *IpAliveSummary `json:"summary,omitempty"`
}

type HostInfo struct {
	IP       string  `json:"ip"`
	RTT      float64 `json:"rtt,omitempty"`
	TTL      int     `json:"ttl,omitempty"`
	Hostname string  `json:"hostname,omitempty"`
	OS       string  `json:"os,omitempty"`
	Mac      string  `json:"mac,omitempty"`
}

type IpAliveSummary struct {
	AliveCount   int   `json:"alive_count"`
	TotalScanned int   `json:"total_scanned"`
	ElapsedMs    int64 `json:"elapsed_ms"`
}
