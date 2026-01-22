package model

import (
	"time"
)

// IpAliveResult IP存活扫描结果
type IpAliveResult struct {
	IP      string        `json:"ip"`
	Alive   bool          `json:"alive"`
	Latency time.Duration `json:"latency,omitempty"` // 暂时没用到，后续加上
}

// Headers 实现 TabularData 接口
// IP        | Status | Latency
// 127.0.0.1 | UP     | N/A
func (r IpAliveResult) Headers() []string {
	return []string{"IP", "Status", "Latency"}
	// 这三个字段分别对应表格的列
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

	return [][]string{{r.IP, status, latency}}
}
