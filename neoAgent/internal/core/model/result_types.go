package model

import (
	"fmt"
	"time"
)

// IpAliveResult IP存活扫描结果
type IpAliveResult struct {
	IP       string        `json:"ip"`
	Alive    bool          `json:"alive"`
	Latency  time.Duration `json:"latency,omitempty"`
	TTL      int           `json:"ttl,omitempty"`
	Hostname string        `json:"hostname,omitempty"`
	OS       string        `json:"os,omitempty"`
}

// Headers 实现 TabularData 接口
// IP        | Status | Latency | TTL | Hostname | OS
// 127.0.0.1 | UP     | 10ms    | 64  | localhost| Linux
func (r IpAliveResult) Headers() []string {
	// 表头列
	return []string{"IP", "Status", "Latency", "TTL", "Hostname", "OS"}
}

// Rows 实现 TabularData 接口
func (r IpAliveResult) Rows() [][]string {
	status := "DOWN"
	if r.Alive {
		status = "UP"
	}

	latency := "N/A"
	if r.Latency > 0 {
		latency = r.Latency.String()
	}

	ttl := "N/A"
	if r.TTL > 0 {
		ttl = fmt.Sprintf("%d", r.TTL)
	}

	return [][]string{{r.IP, status, latency, ttl, r.Hostname, r.OS}}
}
