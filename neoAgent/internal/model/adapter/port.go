package adapter

// PortScanAttributes 对应 port_scan 契约
type PortScanAttributes struct {
	Ports   []PortInfo       `json:"ports"`
	Summary *PortScanSummary `json:"summary,omitempty"`
}

type PortInfo struct {
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	Proto       string `json:"proto"` // tcp, udp
	State       string `json:"state"` // open, closed, filtered
	ServiceHint string `json:"service_hint,omitempty"`
	Banner      string `json:"banner,omitempty"`
}

type PortScanSummary struct {
	OpenCount    int    `json:"open_count"`
	ScanStrategy string `json:"scan_strategy,omitempty"`
	ElapsedMs    int64  `json:"elapsed_ms"`
}
