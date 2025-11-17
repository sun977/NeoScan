/**
 * 模型:Agent响应模型
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent相关的响应数据模型，遵循"好品味"原则 - 数据结构优先
 * @func: 各种Agent Response结构体定义
 */
package agent

import (
	"time"
)

// RegisterAgentResponse Agent注册响应结构
// 返回注册成功后的Agent信息和认证Token
type RegisterAgentResponse struct {
	AgentID     string    `json:"agent_id"`     // Agent唯一标识ID
	GRPCToken   string    `json:"grpc_token"`   // gRPC通信Token
	TokenExpiry time.Time `json:"token_expiry"` // Token过期时间
	Status      string    `json:"status"`       // 注册状态
	Message     string    `json:"message"`      // 响应消息
}

// AgentInfo Agent信息结构
// 用于返回Agent的详细信息，包含基础信息和状态
type AgentInfo struct {
	ID               uint        `json:"id"`                 // 数据库主键ID
	AgentID          string      `json:"agent_id"`           // Agent唯一标识ID
	Hostname         string      `json:"hostname"`           // 主机名
	IPAddress        string      `json:"ip_address"`         // IP地址
	Port             int         `json:"port"`               // Agent服务端口
	Version          string      `json:"version"`            // Agent版本号
	Status           AgentStatus `json:"status"`             // Agent状态
	OS               string      `json:"os"`                 // 操作系统
	Arch             string      `json:"arch"`               // 系统架构
	CPUCores         int         `json:"cpu_cores"`          // CPU核心数
	MemoryTotal      int64       `json:"memory_total"`       // 总内存大小(字节)
	DiskTotal        int64       `json:"disk_total"`         // 总磁盘大小(字节)
	Capabilities     []string    `json:"capabilities"`       // Agent支持的功能模块列表
	Tags             []string    `json:"tags"`               // Agent标签列表
	LastHeartbeat    time.Time   `json:"last_heartbeat"`     // 最后心跳时间
	ResultLatestTime *time.Time  `json:"result_latest_time"` // 最新返回结果时间
	Remark           string      `json:"remark"`             // 备注信息
	ContainerID      string      `json:"container_id"`       // 容器ID
	PID              int         `json:"pid"`                // 进程ID
	CreatedAt        time.Time   `json:"created_at"`         // 创建时间
	UpdatedAt        time.Time   `json:"updated_at"`         // 更新时间
}

// GetAgentListResponse 获取Agent列表响应结构
// 包含Agent列表和分页信息
type GetAgentListResponse struct {
	Agents     []*AgentInfo        `json:"agents"`     // Agent列表
	Pagination *PaginationResponse `json:"pagination"` // 分页信息
}

// PaginationResponse 分页响应结构
// 通用的分页信息结构
type PaginationResponse struct {
	Page       int   `json:"page"`        // 当前页码
	PageSize   int   `json:"page_size"`   // 每页大小
	Total      int64 `json:"total"`       // 总记录数
	TotalPages int   `json:"total_pages"` // 总页数
}

// AgentMetricsResponse Agent性能指标响应结构
// 返回Agent的实时性能数据
type AgentMetricsResponse struct {
	AgentID           string                 `json:"agent_id"`           // Agent唯一标识ID
	CPUUsage          float64                `json:"cpu_usage"`          // CPU使用率(百分比)
	MemoryUsage       float64                `json:"memory_usage"`       // 内存使用率(百分比)
	DiskUsage         float64                `json:"disk_usage"`         // 磁盘使用率(百分比)
	NetworkBytesSent  int64                  `json:"network_bytes_sent"` // 网络发送字节数
	NetworkBytesRecv  int64                  `json:"network_bytes_recv"` // 网络接收字节数
	ActiveConnections int                    `json:"active_connections"` // 活动连接数
	RunningTasks      int                    `json:"running_tasks"`      // 正在运行的任务数
	CompletedTasks    int                    `json:"completed_tasks"`    // 已完成任务数
	FailedTasks       int                    `json:"failed_tasks"`       // 失败任务数
	WorkStatus        AgentWorkStatus        `json:"work_status"`        // 工作状态
	ScanType          string                 `json:"scan_type"`          // 当前扫描类型
	PluginStatus      map[string]interface{} `json:"plugin_status"`      // 插件状态信息
	Timestamp         time.Time              `json:"timestamp"`          // 指标时间戳
}

// AgentMetricsListResponse 获取Agent性能指标列表响应结构
// 包含指标列表和分页信息，保持与Agent列表返回结构一致的风格
type AgentMetricsListResponse struct {
	Metrics    []*AgentMetricsResponse `json:"metrics"`    // 指标列表（当前页）
	Pagination *PaginationResponse     `json:"pagination"` // 分页信息
}

// AgentConfigResponse Agent配置响应结构
// 返回Agent的配置信息
type AgentConfigResponse struct {
	AgentID             string                 `json:"agent_id"`              // Agent唯一标识ID
	Version             int                    `json:"version"`               // 配置版本号
	HeartbeatInterval   int                    `json:"heartbeat_interval"`    // 心跳间隔(秒)
	TaskPollInterval    int                    `json:"task_poll_interval"`    // 任务轮询间隔(秒)
	MaxConcurrentTasks  int                    `json:"max_concurrent_tasks"`  // 最大并发任务数
	PluginConfig        map[string]interface{} `json:"plugin_config"`         // 插件配置信息
	LogLevel            string                 `json:"log_level"`             // 日志级别
	Timeout             int                    `json:"timeout"`               // 超时时间(秒)
	TokenExpiryDuration int                    `json:"token_expiry_duration"` // Token过期时间(秒)
	TokenNeverExpire    bool                   `json:"token_never_expire"`    // Token是否永不过期
	IsActive            bool                   `json:"is_active"`             // 是否激活
	CreatedAt           time.Time              `json:"created_at"`            // 创建时间
	UpdatedAt           time.Time              `json:"updated_at"`            // 更新时间
}

// AgentStatusResponse Agent状态响应结构
// 返回Agent状态更新结果
type AgentStatusResponse struct {
	AgentID   string      `json:"agent_id"`   // Agent唯一标识ID
	Status    AgentStatus `json:"status"`     // 当前状态
	Message   string      `json:"message"`    // 响应消息
	UpdatedAt time.Time   `json:"updated_at"` // 更新时间
}

// AgentTaskAssignmentResponse Agent任务分配响应结构
// 返回任务分配结果
type AgentTaskAssignmentResponse struct {
	AgentID    string          `json:"agent_id"`    // Agent唯一标识ID
	TaskID     string          `json:"task_id"`     // 任务ID
	TaskType   string          `json:"task_type"`   // 任务类型
	Status     AgentTaskStatus `json:"status"`      // 任务状态
	AssignedAt time.Time       `json:"assigned_at"` // 任务分配时间
	Message    string          `json:"message"`     // 响应消息
}

// AgentGroupResponse Agent分组响应结构
// 返回分组信息
type AgentGroupResponse struct {
	ID          uint      `json:"id"`          // 数据库主键ID
	GroupID     string    `json:"group_id"`    // 分组ID
	Name        string    `json:"name"`        // 分组名称
	Description string    `json:"description"` // 分组描述
	Tags        []string  `json:"tags"`        // 分组标签列表
	CreatedAt   time.Time `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`  // 更新时间
}

// AgentGroupListResponse 获取分组列表响应结构
type AgentGroupListResponse struct {
	Groups     []*AgentGroupResponse `json:"groups"`
	Pagination *PaginationResponse   `json:"pagination"`
}

// AgentVersionResponse Agent版本响应结构
// 返回Agent版本信息
type AgentVersionResponse struct {
	ID          uint      `json:"id"`           // 数据库主键ID
	Version     string    `json:"version"`      // 版本号
	ReleaseDate time.Time `json:"release_date"` // 发布日期
	Changelog   string    `json:"changelog"`    // 版本更新日志
	DownloadURL string    `json:"download_url"` // 下载地址
	IsActive    bool      `json:"is_active"`    // 是否激活
	IsLatest    bool      `json:"is_latest"`    // 是否为最新版本
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`   // 更新时间
}

// HeartbeatResponse 心跳响应结构
// 返回心跳处理结果
type HeartbeatResponse struct {
	AgentID   string    `json:"agent_id"`  // Agent唯一标识ID
	Status    string    `json:"status"`    // 处理状态
	Message   string    `json:"message"`   // 响应消息
	Timestamp time.Time `json:"timestamp"` // 响应时间戳
}

// AgentDeleteResponse Agent删除响应结构
// 返回删除操作结果
type AgentDeleteResponse struct {
	AgentID   string    `json:"agent_id"`   // Agent唯一标识ID
	Status    string    `json:"status"`     // 删除状态
	Message   string    `json:"message"`    // 响应消息
	DeletedAt time.Time `json:"deleted_at"` // 删除时间
}

// ==================== 高级统计与分析响应结构 ====================

// AggregatedPerformance 性能聚合统计
// 说明：对当前 agent_metrics 快照做整体聚合，体现分布水平
type AggregatedPerformance struct {
	CPUAvg  float64 `json:"cpu_avg"`    // CPU平均使用率
	CPUMax  float64 `json:"cpu_max"`    // CPU最大使用率
	CPUMin  float64 `json:"cpu_min"`    // CPU最小使用率
	MemAvg  float64 `json:"memory_avg"` // 内存平均使用率
	MemMax  float64 `json:"memory_max"` // 内存最大使用率
	MemMin  float64 `json:"memory_min"` // 内存最小使用率
	DiskAvg float64 `json:"disk_avg"`   // 磁盘平均使用率
	DiskMax float64 `json:"disk_max"`   // 磁盘最大使用率
	DiskMin float64 `json:"disk_min"`   // 磁盘最小使用率

	RunningTasksTotal   int64 `json:"running_tasks_total"`   // 正在运行任务总数
	CompletedTasksTotal int64 `json:"completed_tasks_total"` // 已完成任务总数
	FailedTasksTotal    int64 `json:"failed_tasks_total"`    // 失败任务总数
}

// AgentStatisticsResponse Agent总体统计响应
type AgentStatisticsResponse struct {
	TotalAgents            int64                 `json:"total_agents"`             // metrics快照中的Agent数量
	OnlineAgents           int64                 `json:"online_agents"`            // 近窗口内在线的Agent数
	OfflineAgents          int64                 `json:"offline_agents"`           // 离线Agent数（=Total-Online）
	WorkStatusDistribution map[string]int64      `json:"work_status_distribution"` // 工作状态分布
	ScanTypeDistribution   map[string]int64      `json:"scan_type_distribution"`   // 扫描类型分布
	Performance            AggregatedPerformance `json:"performance"`              // 性能聚合统计
}

// AgentLoadItem 单个Agent负载项
type AgentLoadItem struct {
	AgentID      string          `json:"agent_id"`
	CPUUsage     float64         `json:"cpu_usage"`
	MemoryUsage  float64         `json:"memory_usage"`
	RunningTasks int             `json:"running_tasks"`
	LoadScore    float64         `json:"load_score"` // 负载评分：0.5*CPU + 0.5*Mem + 5*RunningTasks
	WorkStatus   AgentWorkStatus `json:"work_status"`
	ScanType     string          `json:"scan_type"`
	Timestamp    time.Time       `json:"timestamp"`
}

// AgentLoadBalanceResponse 负载均衡分析响应
type AgentLoadBalanceResponse struct {
	TopBusyAgents []AgentLoadItem `json:"top_busy_agents"` // 负载最高的若干Agent
	TopIdleAgents []AgentLoadItem `json:"top_idle_agents"` // 负载最低的若干Agent
	Advice        string          `json:"advice"`          // 简要调度建议
}

// AgentPerformanceItem 性能排名项
type AgentPerformanceItem struct {
	AgentID          string    `json:"agent_id"`
	CPUUsage         float64   `json:"cpu_usage"`
	MemoryUsage      float64   `json:"memory_usage"`
	DiskUsage        float64   `json:"disk_usage"`
	NetworkBytesSent int64     `json:"network_bytes_sent"`
	NetworkBytesRecv int64     `json:"network_bytes_recv"`
	FailedTasks      int       `json:"failed_tasks"`
	Timestamp        time.Time `json:"timestamp"`
}

// AgentPerformanceAnalysisResponse 性能分析响应
type AgentPerformanceAnalysisResponse struct {
	Aggregated     AggregatedPerformance  `json:"aggregated"`  // 聚合统计
	TopCPU         []AgentPerformanceItem `json:"top_cpu"`     // CPU使用率TopN
	TopMemory      []AgentPerformanceItem `json:"top_memory"`  // 内存使用率TopN
	TopNetwork     []AgentPerformanceItem `json:"top_network"` // 网络字节总量TopN（发送+接收）
	TopFailedTasks []AgentPerformanceItem `json:"top_failed"`  // 失败任务数TopN
}

// AgentCapacityItem 过载明细
type AgentCapacityItem struct {
	AgentID     string    `json:"agent_id"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
	Reason      string    `json:"reason"` // 过载原因：cpu/memory/disk
	Timestamp   time.Time `json:"timestamp"`
}

// AgentCapacityAnalysisResponse 容量分析响应
type AgentCapacityAnalysisResponse struct {
	OnlineAgents    int64               `json:"online_agents"`     // 在线Agent数
	Overloaded      int64               `json:"overloaded_agents"` // 过载Agent数
	CPUThreshold    float64             `json:"cpu_threshold"`
	MemThreshold    float64             `json:"memory_threshold"`
	DiskThreshold   float64             `json:"disk_threshold"`
	AverageHeadroom float64             `json:"average_headroom"` // 平均余量（(100-max(cpu,mem,disk))的均值）
	CapacityScore   float64             `json:"capacity_score"`   // 0-100：平均余量的简化映射
	Bottlenecks     map[string]int64    `json:"bottlenecks"`      // 各瓶颈计数
	Recommendations string              `json:"recommendations"`  // 扩容建议
	OverloadedList  []AgentCapacityItem `json:"overloaded_list"`
}
