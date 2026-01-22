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
