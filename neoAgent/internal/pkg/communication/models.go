/**
 * 通信数据模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端与Master端通信的数据模型定义
 * @func: 占位符实现，待后续完善
 */
package communication

import (
	"errors"
	"time"
)

// ==================== 错误定义 ====================

var (
	ErrNotConnected     = errors.New("not connected to master")
	ErrConnectionFailed = errors.New("failed to connect to master")
	ErrAuthFailed       = errors.New("authentication failed")
	ErrTimeout          = errors.New("operation timeout")
	ErrInvalidResponse  = errors.New("invalid response from master")
)

// ==================== Agent信息和状态 ====================

// AgentInfo Agent基本信息
type AgentInfo struct {
	ID          string            `json:"id"`           // Agent唯一标识
	Name        string            `json:"name"`         // Agent名称
	Version     string            `json:"version"`      // Agent版本
	IP          string            `json:"ip"`           // Agent IP地址
	Port        int               `json:"port"`         // Agent端口
	OS          string            `json:"os"`           // 操作系统
	Arch        string            `json:"arch"`         // 系统架构
	Capabilities []string         `json:"capabilities"` // Agent能力列表
	Tags        map[string]string `json:"tags"`         // Agent标签
	RegisterTime time.Time        `json:"register_time"` // 注册时间
}

// AgentStatus Agent状态信息
type AgentStatus struct {
	ID           string                 `json:"id"`            // Agent ID
	Status       string                 `json:"status"`        // 状态: online, offline, busy, error
	LastSeen     time.Time              `json:"last_seen"`     // 最后活跃时间
	TaskCount    int                    `json:"task_count"`    // 当前任务数量
	CPUUsage     float64                `json:"cpu_usage"`     // CPU使用率
	MemoryUsage  float64                `json:"memory_usage"`  // 内存使用率
	DiskUsage    float64                `json:"disk_usage"`    // 磁盘使用率
	NetworkIO    map[string]int64       `json:"network_io"`    // 网络IO统计
	CustomMetrics map[string]interface{} `json:"custom_metrics"` // 自定义指标
	Timestamp    time.Time              `json:"timestamp"`     // 状态时间戳
}

// ==================== 认证相关 ====================

// AuthData 认证数据
type AuthData struct {
	AgentID   string            `json:"agent_id"`   // Agent ID
	Token     string            `json:"token"`      // 认证令牌
	Signature string            `json:"signature"`  // 签名
	Timestamp time.Time         `json:"timestamp"`  // 时间戳
	Extra     map[string]string `json:"extra"`      // 额外认证信息
}

// AuthResponse 认证响应
type AuthResponse struct {
	Success     bool      `json:"success"`      // 认证是否成功
	AccessToken string    `json:"access_token"` // 访问令牌
	RefreshToken string   `json:"refresh_token"` // 刷新令牌
	ExpiresAt   time.Time `json:"expires_at"`   // 过期时间
	Permissions []string  `json:"permissions"`  // 权限列表
	Message     string    `json:"message"`      // 响应消息
	Timestamp   time.Time `json:"timestamp"`    // 响应时间戳
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	Success   bool      `json:"success"`   // 注册是否成功
	AgentID   string    `json:"agent_id"`  // 分配的Agent ID
	Token     string    `json:"token"`     // 认证令牌
	Config    AgentConfig `json:"config"`  // 初始配置
	Message   string    `json:"message"`   // 响应消息
	Timestamp time.Time `json:"timestamp"` // 响应时间戳
}

// ==================== 心跳相关 ====================

// Heartbeat 心跳数据
type Heartbeat struct {
	AgentID     string                 `json:"agent_id"`     // Agent ID
	Status      string                 `json:"status"`       // 当前状态
	TaskCount   int                    `json:"task_count"`   // 任务数量
	Metrics     *PerformanceMetrics    `json:"metrics"`      // 性能指标
	LastTaskID  string                 `json:"last_task_id"` // 最后执行的任务ID
	Extra       map[string]interface{} `json:"extra"`        // 额外信息
	Timestamp   time.Time              `json:"timestamp"`    // 心跳时间戳
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Success       bool      `json:"success"`        // 心跳是否成功
	NextHeartbeat time.Time `json:"next_heartbeat"` // 下次心跳时间
	Commands      []*Command `json:"commands"`      // 待执行命令
	ConfigUpdate  bool      `json:"config_update"`  // 是否有配置更新
	Message       string    `json:"message"`        // 响应消息
	Timestamp     time.Time `json:"timestamp"`      // 响应时间戳
}

// ==================== 配置相关 ====================

// AgentConfig Agent配置
type AgentConfig struct {
	Version       string                 `json:"version"`        // 配置版本
	HeartbeatInterval time.Duration      `json:"heartbeat_interval"` // 心跳间隔
	TaskTimeout   time.Duration          `json:"task_timeout"`   // 任务超时时间
	MaxTasks      int                    `json:"max_tasks"`      // 最大任务数
	LogLevel      string                 `json:"log_level"`      // 日志级别
	Plugins       []PluginConfig         `json:"plugins"`        // 插件配置
	Security      SecurityConfig         `json:"security"`       // 安全配置
	Custom        map[string]interface{} `json:"custom"`         // 自定义配置
	UpdatedAt     time.Time              `json:"updated_at"`     // 更新时间
}

// PluginConfig 插件配置
type PluginConfig struct {
	Name    string                 `json:"name"`    // 插件名称
	Version string                 `json:"version"` // 插件版本
	Enabled bool                   `json:"enabled"` // 是否启用
	Config  map[string]interface{} `json:"config"`  // 插件配置
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	TLSEnabled    bool   `json:"tls_enabled"`    // 是否启用TLS
	CertFile      string `json:"cert_file"`      // 证书文件
	KeyFile       string `json:"key_file"`       // 密钥文件
	CAFile        string `json:"ca_file"`        // CA文件
	TokenExpiry   time.Duration `json:"token_expiry"`   // 令牌过期时间
	MaxRetries    int    `json:"max_retries"`    // 最大重试次数
}

// ConfigSyncRequest 配置同步请求
type ConfigSyncRequest struct {
	AgentID       string    `json:"agent_id"`       // Agent ID
	CurrentVersion string   `json:"current_version"` // 当前配置版本
	Timestamp     time.Time `json:"timestamp"`      // 请求时间戳
}

// ConfigSyncResponse 配置同步响应
type ConfigSyncResponse struct {
	Success       bool                   `json:"success"`        // 同步是否成功
	ConfigVersion string                 `json:"config_version"` // 配置版本
	Configs       map[string]interface{} `json:"configs"`        // 配置数据
	NeedRestart   bool                   `json:"need_restart"`   // 是否需要重启
	Message       string                 `json:"message"`        // 响应消息
	Timestamp     time.Time              `json:"timestamp"`      // 响应时间戳
}

// ==================== 命令相关 ====================

// Command Master发送的命令
type Command struct {
	ID          string                 `json:"id"`          // 命令ID
	Type        string                 `json:"type"`        // 命令类型
	Action      string                 `json:"action"`      // 具体动作
	Payload     map[string]interface{} `json:"payload"`     // 命令载荷
	Priority    int                    `json:"priority"`    // 优先级
	Timeout     time.Duration          `json:"timeout"`     // 超时时间
	Retry       int                    `json:"retry"`       // 重试次数
	Timestamp   time.Time              `json:"timestamp"`   // 命令时间戳
	ExpireAt    time.Time              `json:"expire_at"`   // 过期时间
}

// CommandResponse 命令响应
type CommandResponse struct {
	CommandID string                 `json:"command_id"` // 命令ID
	AgentID   string                 `json:"agent_id"`   // Agent ID
	Success   bool                   `json:"success"`    // 执行是否成功
	Result    map[string]interface{} `json:"result"`     // 执行结果
	Error     string                 `json:"error"`      // 错误信息
	Duration  time.Duration          `json:"duration"`   // 执行耗时
	Timestamp time.Time              `json:"timestamp"`  // 响应时间戳
}

// CommandStatus 命令状态
type CommandStatus struct {
	CommandID string    `json:"command_id"` // 命令ID
	Status    string    `json:"status"`     // 状态: pending, running, completed, failed, timeout
	Progress  float64   `json:"progress"`   // 执行进度 (0-100)
	Message   string    `json:"message"`    // 状态消息
	StartTime time.Time `json:"start_time"` // 开始时间
	EndTime   time.Time `json:"end_time"`   // 结束时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// ==================== 性能指标 ====================

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	AgentID     string            `json:"agent_id"`     // Agent ID
	CPUUsage    float64           `json:"cpu_usage"`    // CPU使用率
	MemoryUsage float64           `json:"memory_usage"` // 内存使用率
	DiskUsage   float64           `json:"disk_usage"`   // 磁盘使用率
	NetworkIO   NetworkIOMetrics  `json:"network_io"`   // 网络IO
	ProcessCount int              `json:"process_count"` // 进程数量
	ThreadCount int               `json:"thread_count"`  // 线程数量
	FileDescriptors int           `json:"file_descriptors"` // 文件描述符数量
	LoadAverage []float64         `json:"load_average"` // 负载平均值
	Uptime      time.Duration     `json:"uptime"`       // 运行时间
	Custom      map[string]float64 `json:"custom"`      // 自定义指标
	Timestamp   time.Time         `json:"timestamp"`    // 指标时间戳
}

// NetworkIOMetrics 网络IO指标
type NetworkIOMetrics struct {
	BytesReceived int64 `json:"bytes_received"` // 接收字节数
	BytesSent     int64 `json:"bytes_sent"`     // 发送字节数
	PacketsReceived int64 `json:"packets_received"` // 接收包数
	PacketsSent   int64 `json:"packets_sent"`   // 发送包数
	ErrorsReceived int64 `json:"errors_received"` // 接收错误数
	ErrorsSent    int64 `json:"errors_sent"`    // 发送错误数
}

// ==================== 任务相关 ====================

// Task 任务信息
type Task struct {
	ID          string                 `json:"id"`          // 任务ID
	Name        string                 `json:"name"`        // 任务名称
	Type        string                 `json:"type"`        // 任务类型
	Status      string                 `json:"status"`      // 任务状态
	Priority    int                    `json:"priority"`    // 优先级
	Config      map[string]interface{} `json:"config"`      // 任务配置
	Progress    float64                `json:"progress"`    // 执行进度
	StartTime   time.Time              `json:"start_time"`  // 开始时间
	EndTime     time.Time              `json:"end_time"`    // 结束时间
	Timeout     time.Duration          `json:"timeout"`     // 超时时间
	RetryCount  int                    `json:"retry_count"` // 重试次数
	CreatedAt   time.Time              `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time              `json:"updated_at"`  // 更新时间
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID      string                 `json:"task_id"`     // 任务ID
	AgentID     string                 `json:"agent_id"`    // Agent ID
	Status      string                 `json:"status"`      // 执行状态
	Result      map[string]interface{} `json:"result"`      // 执行结果
	Error       string                 `json:"error"`       // 错误信息
	Logs        []string               `json:"logs"`        // 执行日志
	Metrics     *TaskMetrics           `json:"metrics"`     // 任务指标
	StartTime   time.Time              `json:"start_time"`  // 开始时间
	EndTime     time.Time              `json:"end_time"`    // 结束时间
	Duration    time.Duration          `json:"duration"`    // 执行耗时
	Timestamp   time.Time              `json:"timestamp"`   // 结果时间戳
}

// TaskMetrics 任务执行指标
type TaskMetrics struct {
	CPUTime     time.Duration `json:"cpu_time"`     // CPU时间
	MemoryPeak  int64         `json:"memory_peak"`  // 内存峰值
	DiskRead    int64         `json:"disk_read"`    // 磁盘读取
	DiskWrite   int64         `json:"disk_write"`   // 磁盘写入
	NetworkRead int64         `json:"network_read"` // 网络读取
	NetworkWrite int64        `json:"network_write"` // 网络写入
}

// ==================== 告警相关 ====================

// Alert 告警信息
type Alert struct {
	ID          string                 `json:"id"`          // 告警ID
	AgentID     string                 `json:"agent_id"`    // Agent ID
	Type        string                 `json:"type"`        // 告警类型
	Level       string                 `json:"level"`       // 告警级别: info, warning, error, critical
	Title       string                 `json:"title"`       // 告警标题
	Message     string                 `json:"message"`     // 告警消息
	Source      string                 `json:"source"`      // 告警源
	Tags        map[string]string      `json:"tags"`        // 告警标签
	Metrics     map[string]interface{} `json:"metrics"`     // 相关指标
	Resolved    bool                   `json:"resolved"`    // 是否已解决
	ResolvedAt  time.Time              `json:"resolved_at"` // 解决时间
	CreatedAt   time.Time              `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time              `json:"updated_at"`  // 更新时间
}

// ==================== 通用响应 ====================

// SyncResponse 同步响应
type SyncResponse struct {
	Success   bool      `json:"success"`   // 同步是否成功
	Message   string    `json:"message"`   // 响应消息
	Timestamp time.Time `json:"timestamp"` // 响应时间戳
}

// ReportResponse 上报响应
type ReportResponse struct {
	Success   bool      `json:"success"`   // 上报是否成功
	Message   string    `json:"message"`   // 响应消息
	Timestamp time.Time `json:"timestamp"` // 响应时间戳
}