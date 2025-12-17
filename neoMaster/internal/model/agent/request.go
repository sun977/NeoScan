/**
 * 模型:Agent请求模型
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent相关的请求数据模型，遵循"好品味"原则 - 数据结构优先
 * @func: 各种Agent Request结构体定义
 */
package agent

// RegisterAgentRequest Agent注册请求结构
// 遵循Linus原则：简洁明了，消除特殊情况
type RegisterAgentRequest struct {
	Hostname     string   `json:"hostname" validate:"required"`             // 主机名，必填
	IPAddress    string   `json:"ip_address" validate:"required"`           // IP地址，必填
	Port         int      `json:"port" validate:"required,min=1,max=65535"` // 端口，必填，范围1-65535
	Version      string   `json:"version" validate:"required"`              // Agent版本，必填
	OS           string   `json:"os" validate:"required"`                   // 操作系统，必填
	Arch         string   `json:"arch" validate:"required"`                 // 系统架构，必填
	CPUCores     int      `json:"cpu_cores" validate:"min=1"`               // CPU核心数，最少1个
	MemoryTotal  int64    `json:"memory_total" validate:"min=1"`            // 总内存大小(字节)，最少1字节
	DiskTotal    int64    `json:"disk_total" validate:"min=1"`              // 总磁盘大小(字节)，最少1字节
	ContainerID  string   `json:"container_id"`                             // 容器ID，可选
	PID          int      `json:"pid" validate:"min=1"`                     // 进程ID，最少1
	Capabilities []string `json:"capabilities"`                             // Agent支持的扫描类型ID列表 (兼容旧版)
	Tags         []string `json:"tags"`                                     // Agent标签列表 (兼容旧版)
	TaskSupport  []string `json:"task_support"`                             // Agent支持的任务类型列表 (新，对应ScanType)
	Feature      []string `json:"feature"`                                  // Agent具备的特性功能列表 (新，备用)
	Remark       string   `json:"remark"`                                   // 备注信息
}

// HeartbeatRequest Agent心跳请求结构
// 遵循"好品味"原则：心跳状态信息和性能指标数据完全分离
// 心跳请求只负责传递心跳状态和性能指标数据
type HeartbeatRequest struct {
	// 心跳基础信息 - 用于更新agents表的last_heartbeat、updated_at、status字段
	AgentID string      `json:"agent_id" validate:"required"` // Agent唯一标识ID，必填
	Status  AgentStatus `json:"status" validate:"required"`   // Agent状态，必填

	// 性能指标数据 - 可选，用于存储到agent_metrics表
	Metrics *AgentMetrics `json:"metrics,omitempty"` // 性能指标数据，可选
}

// GetAgentListRequest 获取Agent列表请求结构
// 支持分页和过滤条件
type GetAgentListRequest struct {
	Page         int         `json:"page" validate:"min=1"`              // 页码，最少1
	PageSize     int         `json:"page_size" validate:"min=1,max=100"` // 每页大小，1-100
	Status       AgentStatus `json:"status"`                             // 按状态过滤，可选
	ScanType     string      `json:"scan_type"`                          // 按扫描类型过滤，可选
	Keyword      string      `json:"keyword"`                            // 关键词搜索(主机名、IP地址)，可选
	Tags         []string    `json:"tags"`                               // 按标签过滤，可选
	TaskSupport  []string    `json:"task_support"`                       // 按任务支持过滤，可选
}

// UpdateAgentStatusRequest 更新Agent状态请求结构
type UpdateAgentStatusRequest struct {
	Status AgentStatus `json:"status" validate:"required"` // 新状态，必填
}

// AgentConfigUpdateRequest Agent配置更新请求结构
type AgentConfigUpdateRequest struct {
	HeartbeatInterval   int                    `json:"heartbeat_interval" validate:"min=5,max=300"`      // 心跳间隔(秒)，5-300秒
	TaskPollInterval    int                    `json:"task_poll_interval" validate:"min=1,max=60"`       // 任务轮询间隔(秒)，1-60秒
	MaxConcurrentTasks  int                    `json:"max_concurrent_tasks" validate:"min=1,max=50"`     // 最大并发任务数，1-50
	PluginConfig        map[string]interface{} `json:"plugin_config"`                                    // 插件配置信息
	LogLevel            string                 `json:"log_level" validate:"oneof=debug info warn error"` // 日志级别，限定值
	Timeout             int                    `json:"timeout" validate:"min=30,max=3600"`               // 超时时间(秒)，30-3600秒
	TokenExpiryDuration int                    `json:"token_expiry_duration" validate:"min=3600"`        // Token过期时间(秒)，最少1小时
	TokenNeverExpire    bool                   `json:"token_never_expire"`                               // Token是否永不过期
}

// AgentTaskAssignRequest Agent任务分配请求结构
type AgentTaskAssignRequest struct {
	TaskID   string `json:"task_id" validate:"required"`   // 任务ID，必填
	TaskType string `json:"task_type" validate:"required"` // 任务类型，必填
}

// AgentTagRequest Agent标签操作请求结构
type AgentTagRequest struct {
	AgentID string `json:"agent_id" validate:"required"` // Agent业务ID，必填
	TagID   uint64 `json:"tag_id" validate:"required"`   // 标签ID，必填
}

// AgentCapabilityRequest Agent能力操作请求结构
type AgentCapabilityRequest struct {
	AgentID    string `json:"agent_id" validate:"required"`   // Agent业务ID，必填
	Capability string `json:"capability" validate:"required"` // 能力名称，必填
}

// AgentTaskSupportRequest Agent任务支持操作请求结构
// 新增：对应 TaskSupport 字段的操作
type AgentTaskSupportRequest struct {
	AgentID     string `json:"agent_id" validate:"required"`     // Agent业务ID，必填
	TaskSupport string `json:"task_support" validate:"required"` // 任务支持ID/名称，必填
}

// AgentTaskResultRequest Agent任务结果上报请求结构
type AgentTaskResultRequest struct {
	TaskID      string      `json:"task_id" validate:"required"`  // 任务ID，必填
	AgentID     string      `json:"agent_id" validate:"required"` // AgentID，必填
	Status      string      `json:"status" validate:"required"`   // 任务状态 (running, completed, failed)
	Result      interface{} `json:"result"`                       // 任务结果数据 (JSON对象或字符串)
	Error       string      `json:"error"`                        // 错误信息
	CompletedAt int64       `json:"completed_at"`                 // 完成时间戳
}

// // UpdateAgentTagsRequest 更新指定 Agent 标签列表请求结构
// type UpdateAgentTagsRequest struct {
// 	AgentID string   `json:"agent_id" validate:"required"` // Agent业务ID，必填
// 	Tags    []string `json:"tags" validate:"required"`     // 新标签列表，必填
// }

// ==================== 高级统计与分析查询参数 ====================

// AgentStatisticsQuery 统计查询参数
type AgentStatisticsQuery struct {
	WindowSeconds int `json:"window_seconds" validate:"min=1,max=86400"` // 在线判定窗口（秒）
}

// AgentLoadBalanceQuery 负载均衡查询参数
type AgentLoadBalanceQuery struct {
	WindowSeconds int `json:"window_seconds" validate:"min=1,max=86400"` // 在线判定窗口（秒）
	TopN          int `json:"top_n" validate:"min=1,max=100"`            // Top列表数量
}

// AgentPerformanceQuery 性能分析查询参数
type AgentPerformanceQuery struct {
	WindowSeconds int `json:"window_seconds" validate:"min=1,max=86400"` // 在线判定窗口（秒）
	TopN          int `json:"top_n" validate:"min=1,max=100"`            // Top列表数量
}

// AgentCapacityQuery 容量分析查询参数
type AgentCapacityQuery struct {
	WindowSeconds   int     `json:"window_seconds" validate:"min=1,max=86400"` // 在线判定窗口（秒）
	CPUThreshold    float64 `json:"cpu_threshold" validate:"min=0,max=100"`    // CPU过载阈值
	MemoryThreshold float64 `json:"memory_threshold" validate:"min=0,max=100"` // 内存过载阈值
	DiskThreshold   float64 `json:"disk_threshold" validate:"min=0,max=100"`   // 磁盘过载阈值
}
