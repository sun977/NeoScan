/**
 * Agent相关模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent基本信息和状态模型，遵循"好品味"原则 - 数据结构优先
 * @func: 定义Agent信息、状态和性能指标相关的数据结构
 */
package communication

import "time"

// ==================== Agent信息和状态 ====================

// AgentInfo Agent基本信息
// 遵循Linus原则：数据结构清晰，字段职责明确
type AgentInfo struct {
	ID           string            `json:"id"`            // Agent唯一标识
	Name         string            `json:"name"`          // Agent名称
	Version      string            `json:"version"`       // Agent版本
	IP           string            `json:"ip"`            // Agent IP地址
	Port         int               `json:"port"`          // Agent端口
	OS           string            `json:"os"`            // 操作系统
	Arch         string            `json:"arch"`          // 系统架构
	Capabilities []string          `json:"capabilities"`  // Agent能力列表
	Tags         map[string]string `json:"tags"`          // Agent标签
	RegisterTime time.Time         `json:"register_time"` // 注册时间
}

// AgentStatus Agent状态信息
// 遵循"好品味"原则：状态信息和指标数据分离，避免混合职责
type AgentStatus struct {
	ID            string                 `json:"id"`             // Agent ID
	Status        string                 `json:"status"`         // 状态: online, offline, busy, error
	LastSeen      time.Time              `json:"last_seen"`      // 最后活跃时间
	TaskCount     int                    `json:"task_count"`     // 当前任务数量
	CPUUsage      float64                `json:"cpu_usage"`      // CPU使用率
	MemoryUsage   float64                `json:"memory_usage"`   // 内存使用率
	DiskUsage     float64                `json:"disk_usage"`     // 磁盘使用率
	NetworkIO     map[string]int64       `json:"network_io"`     // 网络IO统计
	CustomMetrics map[string]interface{} `json:"custom_metrics"` // 自定义指标
	Timestamp     time.Time              `json:"timestamp"`      // 状态时间戳
}

// ==================== 性能指标 ====================

// PerformanceMetrics 性能指标
// 遵循单一职责原则：专门负责性能数据收集和传输
type PerformanceMetrics struct {
	AgentID         string             `json:"agent_id"`         // Agent ID
	CPUUsage        float64            `json:"cpu_usage"`        // CPU使用率
	MemoryUsage     float64            `json:"memory_usage"`     // 内存使用率
	DiskUsage       float64            `json:"disk_usage"`       // 磁盘使用率
	NetworkIO       NetworkIOMetrics   `json:"network_io"`       // 网络IO
	ProcessCount    int                `json:"process_count"`    // 进程数量
	ThreadCount     int                `json:"thread_count"`     // 线程数量
	FileDescriptors int                `json:"file_descriptors"` // 文件描述符数量
	LoadAverage     []float64          `json:"load_average"`     // 负载平均值
	Uptime          time.Duration      `json:"uptime"`           // 运行时间
	Custom          map[string]float64 `json:"custom"`           // 自定义指标
	Timestamp       time.Time          `json:"timestamp"`        // 指标时间戳
}

// NetworkIOMetrics 网络IO指标
// 遵循"好品味"原则：网络指标独立定义，避免嵌套复杂性
type NetworkIOMetrics struct {
	BytesReceived   int64 `json:"bytes_received"`   // 接收字节数
	BytesSent       int64 `json:"bytes_sent"`       // 发送字节数
	PacketsReceived int64 `json:"packets_received"` // 接收包数
	PacketsSent     int64 `json:"packets_sent"`     // 发送包数
	ErrorsReceived  int64 `json:"errors_received"`  // 接收错误数
	ErrorsSent      int64 `json:"errors_sent"`      // 发送错误数
}