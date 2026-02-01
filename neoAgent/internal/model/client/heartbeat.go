package client

import "time"

// PluginStatus 插件状态
type PluginStatus map[string]interface{}

// HeartbeatMetrics 心跳指标
// 对应 Master 端的 AgentMetrics
type HeartbeatMetrics struct {
	AgentID           string       `json:"agent_id"`
	CPUUsage          float64      `json:"cpu_usage"`
	MemoryUsage       float64      `json:"memory_usage"`
	DiskUsage         float64      `json:"disk_usage"`
	NetworkBytesSent  int64        `json:"network_bytes_sent"`
	NetworkBytesRecv  int64        `json:"network_bytes_recv"`
	ActiveConnections int          `json:"active_connections"`
	RunningTasks      int          `json:"running_tasks"`
	CompletedTasks    int          `json:"completed_tasks"`
	FailedTasks       int          `json:"failed_tasks"`
	WorkStatus        string       `json:"work_status"` // idle, working, exception
	ScanType          string       `json:"scan_type"`
	PluginStatus      PluginStatus `json:"plugin_status"`
	Timestamp         time.Time    `json:"timestamp"`
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	AgentID string            `json:"agent_id"`
	Status  string            `json:"status"`
	Metrics *HeartbeatMetrics `json:"metrics,omitempty"`
}

// HeartbeatResponseData 心跳响应数据
type HeartbeatResponseData struct {
	AgentID      string            `json:"agent_id"`
	Status       string            `json:"status"`
	Message      string            `json:"message"`
	Timestamp    time.Time         `json:"timestamp"`
	RuleVersions map[string]string `json:"rule_versions,omitempty"` // 规则版本信息
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Code    int                   `json:"code"`
	Status  string                `json:"status"`
	Message string                `json:"message"`
	Data    HeartbeatResponseData `json:"data"`
}
