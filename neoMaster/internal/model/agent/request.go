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
	Capabilities []string `json:"capabilities"`                             // Agent支持的功能模块列表
	Tags         []string `json:"tags"`                                     // Agent标签列表
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
	Capabilities []string    `json:"capabilities"`                       // 按功能模块过滤，可选
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

// AgentGroupCreateRequest Agent分组创建请求结构
type AgentGroupCreateRequest struct {
	GroupID     string   `json:"group_id" validate:"required"` // 分组ID，必填
	Name        string   `json:"name" validate:"required"`     // 分组名称，必填
	Description string   `json:"description"`                  // 分组描述，可选
	Tags        []string `json:"tags"`                         // 分组标签列表，可选
}

// AgentGroupMemberRequest Agent分组成员操作请求结构
type AgentGroupMemberRequest struct {
	AgentID string `json:"agent_id" validate:"required"` // Agent业务ID，必填
	GroupID string `json:"group_id" validate:"required"` // 分组ID，必填
}
