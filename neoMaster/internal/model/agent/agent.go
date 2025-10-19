/**
 * 模型:Agent 核心模型
 * @author: sun977
 * @date: 2025.09.26
 * @description: Agent 核心模型定义，遵循单一职责原则
 * @func: 定义 Agent 实体及其核心业务方法
 */
package agent

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"neomaster/internal/model/basemodel"
	"neomaster/internal/pkg/utils"
)

// ============================================================================
// 基础类型定义
// ============================================================================

// StringSlice 自定义字符串切片类型，用于处理JSON数组字段
// 支持MySQL JSON格式和PostgreSQL数组格式的转换
// 现在用于存储ScanType和TagType的ID列表，格式为 ["2", "3"]
type StringSlice []string

// Scan 实现sql.Scanner接口，用于从数据库读取数据
// 参数: value - 数据库中的值
// 返回: 错误信息
// - Agent 表中有 capabilities 和 tags 字段，类型为 JSON (["2", "3"])，存储字符串形式的ID
// - 每次查询 Agent 数据时，GORM 都会调用 Scan 方法
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("无法将 %T 转换为 StringSlice", value)
	}

	// 使用convert.go中的转换函数
	slice, err := utils.JSONArrayToStringSlice(str)
	if err != nil {
		return fmt.Errorf("StringSlice.Scan 失败: %v", err)
	}

	*s = StringSlice(slice)
	return nil
}

// Value 实现driver.Valuer接口，用于向数据库写入数据
// 返回: 数据库值和错误信息
func (s StringSlice) Value() (driver.Value, error) {
	// 使用convert.go中的转换函数
	jsonStr, err := utils.StringSliceToJSONArray([]string(s))
	if err != nil {
		return nil, fmt.Errorf("StringSlice.Value 失败: %v", err)
	}
	return jsonStr, nil
}

// PluginStatusJSON 插件状态JSON类型
type PluginStatusJSON map[string]interface{}

// Scan 实现sql.Scanner接口
func (p *PluginStatusJSON) Scan(value interface{}) error {
	if value == nil {
		*p = PluginStatusJSON{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("无法将 %T 转换为 PluginStatusJSON", value)
	}

	if str == "" {
		*p = PluginStatusJSON{}
		return nil
	}

	return json.Unmarshal([]byte(str), p)
}

// Value 实现driver.Valuer接口
func (p PluginStatusJSON) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "{}", nil
	}
	return json.Marshal(p)
}

// ConfigTemplateJSON 配置模板JSON类型
type ConfigTemplateJSON map[string]interface{}

// Scan 实现sql.Scanner接口
func (c *ConfigTemplateJSON) Scan(value interface{}) error {
	if value == nil {
		*c = ConfigTemplateJSON{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("无法将 %T 转换为 ConfigTemplateJSON", value)
	}

	if str == "" {
		*c = ConfigTemplateJSON{}
		return nil
	}

	return json.Unmarshal([]byte(str), c)
}

// Value 实现driver.Valuer接口
func (c ConfigTemplateJSON) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "{}", nil
	}
	return json.Marshal(c)
}

// ============================================================================
// 枚举常量定义
// ============================================================================

// AgentStatus Agent状态枚举
type AgentStatus string

const (
	AgentStatusOnline      AgentStatus = "online"      // 在线
	AgentStatusOffline     AgentStatus = "offline"     // 离线
	AgentStatusException   AgentStatus = "exception"   // 异常
	AgentStatusMaintenance AgentStatus = "maintenance" // 维护
)

// AgentWorkStatus Agent工作状态枚举
type AgentWorkStatus string

const (
	AgentWorkStatusIdle      AgentWorkStatus = "idle"      // 空闲 [工作完成回归到此状态]
	AgentWorkStatusWorking   AgentWorkStatus = "working"   // 工作中
	AgentWorkStatusException AgentWorkStatus = "exception" // 异常
)

// Agent 内置扫描类型状态常量枚举[系统默认内置扫描类型,自定义扫描类型在数据表中定义(结构体模型:ScanType)]
type AgentScanType string

const (
	AgentScanTypeIdle          AgentScanType = "idle"          // 空闲 [工作完成回归到此状态,表示当前agent没有扫描任务在执行]
	AgentScanTypeIpAliveScan   AgentScanType = "ipAliveScan"   // IP探活阶段 [默认类型:探测网段内存活IP]
	AgentScanTypeFastPortScan  AgentScanType = "fastPortScan"  // 快速端口扫描 [可选类型:默认端口的快速扫描]
	AgentScanTypeFullPortScan  AgentScanType = "fullPortScan"  // 全量端口扫描 [默认类型:全端口扫描] --- 会带有端口对应的服务信息
	AgentScanTypeServiceScan   AgentScanType = "serviceScan"   // 服务扫描 [可选类型:服务扫描] --- 如果端口识别不携带服务识别,这一步单独做一次服务识别
	AgentScanTypeVulnScan      AgentScanType = "vulnScan"      // 漏洞扫描 [可选类型:漏洞扫描]
	AgentScanTypePocScan       AgentScanType = "pocScan"       // POC扫描 [可选类型:POC扫描] --- 结合给定的POC工具或者脚本识别(poc支持自定义,属于高精度的vulnScan,单独一类)
	AgentScanTypeWebScan       AgentScanType = "webScan"       // Web扫描 [可选类型:Web扫描,目标是网址] --- 识别出有web服务或者web框架cms等执行web扫描,爬虫,浏览器驱动,AI爬取等
	AgentScanTypePassScan      AgentScanType = "passScan"      // 弱密码扫描 [可选类型:弱密码扫描] --- 识别出有密码的服务后探测默认/弱口令检查,如数据库,ssh和其他协议等
	AgentScanTypeProxyScan     AgentScanType = "proxyScan"     // 代理服务探测扫描 [可选类型:代理扫描] --- 识别出有代理服务后,进行代理扫描,如http,https,socks等
	AgentScanTypeDirScan       AgentScanType = "dirScan"       // 目录扫描 [可选类型:目录扫描,目标是网址] --- 识别出有web系统后，对系统进行目录扫描,如dirsearch等
	AgentScanTypeSubDomainScan AgentScanType = "subDomainScan" // 子域名扫描 [可选类型:子域名扫描,目标是网址] --- 识别出有web系统后，对系统进行子域名扫描
	AgentScanTypeApiScan       AgentScanType = "apiScan"       // API资产扫描 [可选类型:API扫描,目标是网址] --- 对需要探测的系统所暴露的API进行API资产扫描
	AgentScanTypeFileScan      AgentScanType = "fileScan"      // 文件扫描 [特殊可选类型:文件扫描,目标是本机文件] --- 后续补充,webshell发现，病毒查杀，基于YARA的模块可能会用(预留)
	AgentScanTypeOtherScan     AgentScanType = "otherScan"     // 其他扫描 [可选类型:其他扫描] --- 其他自定义的扫描类型,如自定义的脚本扫描,自定义的模块扫描等(不同于用户定义的扫描类型)
)

// AgentTaskStatus 任务状态枚举
type AgentTaskStatus string

const (
	AgentTaskStatusAssigned  AgentTaskStatus = "assigned"  // 已分配/待执行
	AgentTaskStatusRunning   AgentTaskStatus = "running"   // 运行中
	AgentTaskStatusCompleted AgentTaskStatus = "completed" // 已完成
	AgentTaskStatusFailed    AgentTaskStatus = "failed"    // 已失败
)

// ============================================================================
// 核心实体：Agent - 相对静态，注册时确定
// ============================================================================

// Agent 核心实体 - Agent基础信息
type Agent struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	basemodel.BaseModel

	// 基本标识信息
	AgentID   string      `json:"agent_id" gorm:"uniqueIndex;not null;size:100;comment:Agent唯一标识ID"`
	Hostname  string      `json:"hostname" gorm:"size:255;comment:主机名"`
	IPAddress string      `json:"ip_address" gorm:"size:45;comment:IP地址，支持IPv6"`
	Port      int         `json:"port" gorm:"default:5772;comment:Agent服务端口"`
	Version   string      `json:"version" gorm:"size:50;comment:Agent版本号"`
	Status    AgentStatus `json:"status" gorm:"default:offline;size:20;comment:Agent状态:online-在线,offline-离线,exception-异常,maintenance-维护"`

	// 静态系统信息
	OS          string `json:"os" gorm:"size:50;comment:操作系统"`
	Arch        string `json:"arch" gorm:"size:20;comment:系统架构"`
	CPUCores    int    `json:"cpu_cores" gorm:"comment:CPU核心数"`
	MemoryTotal int64  `json:"memory_total" gorm:"comment:总内存大小(字节)"`
	DiskTotal   int64  `json:"disk_total" gorm:"comment:总磁盘大小(字节)"`

	// 能力和标签(存储ScanType和TagType的ID) - 内容格式:["2","3"] (字符串形式的ID列表)
	Capabilities StringSlice `json:"capabilities" gorm:"type:json;comment:Agent支持的扫描类型ID列表，对应ScanType表的ID"`
	Tags         StringSlice `json:"tags" gorm:"type:json;comment:Agent标签ID列表，对应TagType表的ID"`

	// 安全认证
	GRPCToken   string    `json:"grpc_token" gorm:"column:grpc_token;size:500;comment:gRPC通信Token"`
	TokenExpiry time.Time `json:"token_expiry" gorm:"comment:Token过期时间"`

	// 时间戳
	ResultLatestTime *time.Time `json:"result_latest_time" gorm:"comment:最新返回结果时间"`
	LastHeartbeat    time.Time  `json:"last_heartbeat" gorm:"comment:最后心跳时间"`

	// 扩展字段
	Remark      string `json:"remark" gorm:"size:500;comment:备注信息"`
	ContainerID string `json:"container_id" gorm:"size:100;comment:容器ID"`
	PID         int    `json:"pid" gorm:"column:pid;comment:进程ID"`
}

// TableName 定义表名
func (Agent) TableName() string {
	return "agents"
}

// ============================================================================
// Agent 状态管理方法
// ============================================================================

// IsActive 检查Agent是否处于在线活跃状态
// Agent 结构体的方法 - 检查Agent是否处于在线活跃状态
func (a *Agent) IsActive() bool {
	return a.Status == AgentStatusOnline
}

// IsMaintenance 检查Agent是否处于维护状态
// Agent 结构体的方法 - 检查Agent是否处于维护状态
func (a *Agent) IsMaintenance() bool {
	return a.Status == AgentStatusMaintenance
}

// SetStatus 设置Agent状态
// Agent 结构体的方法 - 设置Agent状态
func (a *Agent) SetStatus(status AgentStatus) {
	a.Status = status
}

// GetStatus 获取Agent当前状态
// Agent 结构体的方法 - 获取Agent当前状态
func (a *Agent) GetStatus() AgentStatus {
	return a.Status
}

// IsOnline 判断Agent是否在线（基于状态和心跳时间）
func (a *Agent) IsOnline() bool {
	return a.Status == AgentStatusOnline && time.Since(a.LastHeartbeat) < 5*time.Minute
}

// UpdateHeartbeat 更新心跳时间
func (a *Agent) UpdateHeartbeat() {
	a.LastHeartbeat = time.Now()
}

// ============================================================================
// Agent 能力管理方法
// ============================================================================

// AddCapability 添加能力（避免重复）
// 参数: capabilityID - 扫描类型ID（字符串形式）
func (a *Agent) AddCapability(capabilityID string) {
	for _, c := range a.Capabilities {
		if c == capabilityID {
			return // 避免重复添加
		}
	}
	a.Capabilities = append(a.Capabilities, capabilityID)
}

// RemoveCapability 移除能力
// Agent 结构体的方法 - 移除指定能力
// 参数: capabilityID - 扫描类型ID（字符串形式）
func (a *Agent) RemoveCapability(capabilityID string) {
	for i, c := range a.Capabilities {
		if c == capabilityID {
			a.Capabilities = append(a.Capabilities[:i], a.Capabilities[i+1:]...)
			return
		}
	}
}

// HasCapability 检查是否具有指定能力
// Agent 结构体的方法 - 检查是否具有指定能力
// 参数: capabilityID - 扫描类型ID（字符串形式）
func (a *Agent) HasCapability(capabilityID string) bool {
	for _, c := range a.Capabilities {
		if c == capabilityID {
			return true
		}
	}
	return false
}

// CanAcceptTask 判断Agent是否可以接受指定类型的任务
// 参数: taskTypeID - 任务类型ID（字符串形式）
func (a *Agent) CanAcceptTask(taskTypeID string) bool {
	return a.IsOnline() && a.HasCapability(taskTypeID)
}

// ============================================================================
// Agent 标签管理方法
// ============================================================================

// AddTag 添加标签（避免重复）
// 参数: tagID - 标签类型ID（字符串形式）
func (a *Agent) AddTag(tagID string) {
	for _, t := range a.Tags {
		if t == tagID {
			return // 避免重复添加
		}
	}
	a.Tags = append(a.Tags, tagID)
}

// RemoveTag 移除标签
// Agent 结构体的方法 - 移除指定标签
// 参数: tagID - 标签类型ID（字符串形式）
func (a *Agent) RemoveTag(tagID string) {
	for i, t := range a.Tags {
		if t == tagID {
			a.Tags = append(a.Tags[:i], a.Tags[i+1:]...)
			return
		}
	}
}

// HasTag 检查是否具有指定标签
// Agent 结构体的方法 - 检查是否具有指定标签
// 参数: tagID - 标签类型ID（字符串形式）
func (a *Agent) HasTag(tagID string) bool {
	for _, t := range a.Tags {
		if t == tagID {
			return true
		}
	}
	return false
}

// ============================================================================
// Agent 安全认证方法
// ============================================================================

// IsTokenValid 检查Token是否有效
// Agent 结构体的方法 - 检查Token是否有效
func (a *Agent) IsTokenValid() bool {
	return time.Now().Before(a.TokenExpiry)
}

// ============================================================================
// 相关实体：AgentVersion
// ============================================================================

// AgentVersion Agent版本信息
type AgentVersion struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	basemodel.BaseModel

	Version     string    `json:"version" gorm:"not null;size:50;comment:版本号"`
	ReleaseDate time.Time `json:"release_date" gorm:"comment:发布日期"`
	Changelog   string    `json:"changelog" gorm:"type:text;comment:版本更新日志"`
	DownloadURL string    `json:"download_url" gorm:"size:500;comment:下载地址"`
	IsActive    bool      `json:"is_active" gorm:"default:true;comment:是否激活"`
	IsLatest    bool      `json:"is_latest" gorm:"default:false;comment:是否为最新版本"`
}

// TableName 定义表名
func (AgentVersion) TableName() string {
	return "agent_versions"
}

// IsActiveVersion 检查版本是否激活
// AgentVersion 结构体的方法 - 检查版本是否激活
func (av *AgentVersion) IsActiveVersion() bool {
	return av.IsActive
}

// IsLatestVersion 检查是否为最新版本
// AgentVersion 结构体的方法 - 检查是否为最新版本
func (av *AgentVersion) IsLatestVersion() bool {
	return av.IsLatest
}

// ============================================================================
// 相关实体：AgentConfig
// ============================================================================

// AgentConfig Agent配置信息
type AgentConfig struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	basemodel.BaseModel

	AgentID             string                 `json:"agent_id" gorm:"uniqueIndex;size:100;comment:Agent唯一标识ID"`
	Version             int                    `json:"version" gorm:"default:1;comment:配置版本号"`
	HeartbeatInterval   int                    `json:"heartbeat_interval" gorm:"default:30;comment:心跳间隔(秒)"`
	TaskPollInterval    int                    `json:"task_poll_interval" gorm:"default:10;comment:任务轮询间隔(秒)"`
	MaxConcurrentTasks  int                    `json:"max_concurrent_tasks" gorm:"default:5;comment:最大并发任务数"`
	PluginConfig        map[string]interface{} `json:"plugin_config" gorm:"type:json;comment:插件配置信息"`
	LogLevel            string                 `json:"log_level" gorm:"default:info;size:20;comment:日志级别"`
	Timeout             int                    `json:"timeout" gorm:"default:300;comment:超时时间(秒)"`
	TokenExpiryDuration int                    `json:"token_expiry_duration" gorm:"default:86400;comment:Token过期时间(秒)"`
	TokenNeverExpire    bool                   `json:"token_never_expire" gorm:"default:false;comment:Token是否永不过期"`
	IsActive            bool                   `json:"is_active" gorm:"default:true;comment:是否激活"`
}

// TableName 定义表名
func (AgentConfig) TableName() string {
	return "agent_configs"
}

// IsActiveConfig 检查配置是否激活
// AgentConfig 结构体的方法 - 检查配置是否激活
func (ac *AgentConfig) IsActiveConfig() bool {
	return ac.IsActive
}

// IncrementVersion 增加配置版本号
// AgentConfig 结构体的方法 - 增加配置版本号并更新时间
func (ac *AgentConfig) IncrementVersion() {
	ac.Version++
	ac.UpdatedAt = time.Now()
}

// ============================================================================
// 相关实体：AgentMetrics
// ============================================================================

// AgentMetrics Agent负载信息（动态数据）
type AgentMetrics struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	basemodel.BaseModel

	AgentID           string           `json:"agent_id" gorm:"uniqueIndex;size:100;comment:Agent唯一标识ID,唯一索引"`
	CPUUsage          float64          `json:"cpu_usage" gorm:"comment:CPU使用率(百分比)"`
	MemoryUsage       float64          `json:"memory_usage" gorm:"comment:内存使用率(百分比)"`
	DiskUsage         float64          `json:"disk_usage" gorm:"comment:磁盘使用率(百分比)"`
	NetworkBytesSent  int64            `json:"network_bytes_sent" gorm:"comment:网络发送字节数"`
	NetworkBytesRecv  int64            `json:"network_bytes_recv" gorm:"comment:网络接收字节数"`
	ActiveConnections int              `json:"active_connections" gorm:"comment:活动连接数"`
	RunningTasks      int              `json:"running_tasks" gorm:"comment:正在运行的任务数"`
	CompletedTasks    int              `json:"completed_tasks" gorm:"comment:已完成任务数"`
	FailedTasks       int              `json:"failed_tasks" gorm:"comment:失败任务数"`
	WorkStatus        AgentWorkStatus  `json:"work_status" gorm:"size:20;comment:工作状态:idle-空闲,working-工作中,exception-异常"`
	ScanType          string           `json:"scan_type" gorm:"size:50;comment:当前扫描类型"`
	PluginStatus      PluginStatusJSON `json:"plugin_status" gorm:"type:json;comment:插件状态信息"`
	Timestamp         time.Time        `json:"timestamp" gorm:"index;comment:指标时间戳"`
}

// TableName 定义表名
func (AgentMetrics) TableName() string {
	return "agent_metrics"
}

// GetAgentLoad 获取Agent负载
// AgentMetrics 结构体的方法 - 获取Agent负载，基于CPU和内存使用率计算
func (am *AgentMetrics) GetAgentLoad() float64 {
	// 简单的负载计算方式，可以基于CPU使用率、内存使用率和任务数综合计算
	return (am.CPUUsage + am.MemoryUsage) / 2.0
}

// IsOverloaded 检查Agent是否过载
// AgentMetrics 结构体的方法 - 检查Agent是否过载
func (am *AgentMetrics) IsOverloaded() bool {
	// 当CPU或内存使用率超过80%时认为过载
	return am.CPUUsage > 80.0 || am.MemoryUsage > 80.0
}

// IsWorking 检查Agent是否正在工作中
// AgentMetrics 结构体的方法 - 检查Agent是否正在工作中
func (am *AgentMetrics) IsWorking() bool {
	return am.WorkStatus == AgentWorkStatusWorking
}

// UpdateTimestamp 更新时间戳
// AgentMetrics 结构体的方法 - 更新时间戳为当前时间
func (am *AgentMetrics) UpdateTimestamp() {
	am.Timestamp = time.Now()
}

// 统一的度量接口
// 注意：此接口应该在Agent端实现，用于收集Agent自身的系统指标
// Master端通过gRPC等通信方式调用Agent端的实现来获取指标数据
type MetricsCollector interface {
	GetCPUUsage() float64
	GetMemoryUsage() float64
	GetDiskUsage() float64
	GetNetworkStats() (sent, recv int64)
	GetTaskStats() (running, completed, failed int)
}

// ============================================================================
// 相关实体：AgentGroup
// ============================================================================

// AgentGroup Agent分组
type AgentGroup struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	basemodel.BaseModel

	GroupID     string   `json:"group_id" gorm:"not null;size:100;comment:分组ID"`
	Name        string   `json:"name" gorm:"not null;size:100;comment:分组名称"`
	Description string   `json:"description" gorm:"size:500;comment:分组描述"`
	Tags        []string `json:"tags" gorm:"type:json;comment:分组标签列表"`
}

// TableName 定义表名
func (AgentGroup) TableName() string {
	return "agent_groups"
}

// IsValid 检查分组是否有效
// AgentGroup 结构体的方法 - 检查分组是否有效
func (ag *AgentGroup) IsValid() bool {
	return ag.Name != ""
}

// UpdateTimestamp 更新时间戳
// AgentGroup 结构体的方法 - 更新时间戳为当前时间
func (ag *AgentGroup) UpdateTimestamp() {
	ag.UpdatedAt = time.Now()
}

// ============================================================================
// 相关实体：AgentGroupMember
// ============================================================================

// AgentGroupMember Agent分组成员关系
// AgentGroupMember 结构体定义Agent分组成员关系[联合分组是一对一关系,不能使用联合分组]
// 注意：一个Agent可以属于多个分组,一个分组可以包含多个Agent
type AgentGroupMember struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	basemodel.BaseModel

	AgentID  string    `json:"agent_id" gorm:"not null;size:100;comment:Agent业务ID"` // Agent业务ID,外键关联agents表
	GroupID  string    `json:"group_id" gorm:"not null;size:100;comment:分组ID"`      // 分组ID,外键关联agent_groups表
	JoinedAt time.Time `json:"joined_at" gorm:"comment:加入时间"`
}

// TableName 指定AgentGroupMember表名
// AgentGroupMember 结构体的方法 - 指定AgentGroupMember表名
func (AgentGroupMember) TableName() string {
	return "agent_group_members"
}

// ============================================================================
// 相关实体：AgentTaskAssignment
// ============================================================================

// AgentTaskAssignment 任务分发记录
type AgentTaskAssignment struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	basemodel.BaseModel

	AgentID     string          `json:"agent_id" gorm:"index;size:100;comment:Agent业务ID"`
	TaskID      string          `json:"task_id" gorm:"index;size:100;comment:任务ID"`
	TaskType    string          `json:"task_type" gorm:"size:50;comment:任务类型"`
	AssignedAt  time.Time       `json:"assigned_at" gorm:"comment:任务分配时间"`
	StartedAt   *time.Time      `json:"started_at" gorm:"comment:任务开始时间"`
	CompletedAt *time.Time      `json:"completed_at" gorm:"comment:任务完成时间"`
	Status      AgentTaskStatus `json:"status" gorm:"size:20;comment:任务状态:assigned-已分配,running-运行中,completed-已完成,failed-已失败"`
	Result      string          `json:"result" gorm:"type:text;comment:任务执行结果"`
}

// TableName 定义表名
func (AgentTaskAssignment) TableName() string {
	return "agent_task_assignments"
}

// IsAssigned 检查任务是否已分配
// AgentTaskAssignment 结构体的方法 - 检查任务是否已分配(未执行/即将执行任务)
func (ata *AgentTaskAssignment) IsAssigned() bool {
	return ata.Status == AgentTaskStatusAssigned
}

// IsRunning 检查任务是否正在运行
// AgentTaskAssignment 结构体的方法 - 检查任务是否正在运行
func (ata *AgentTaskAssignment) IsRunning() bool {
	return ata.Status == AgentTaskStatusRunning
}

// IsCompleted 检查任务是否已完成
// AgentTaskAssignment 结构体的方法 - 检查任务是否已完成
func (ata *AgentTaskAssignment) IsCompleted() bool {
	return ata.Status == AgentTaskStatusCompleted
}

// IsFailed 检查任务是否失败
// AgentTaskAssignment 结构体的方法 - 检查任务是否失败
func (ata *AgentTaskAssignment) IsFailed() bool {
	return ata.Status == AgentTaskStatusFailed
}

// AssignTask 标记任务已分配
// AgentTaskAssignment 结构体的方法 - 标记任务已分配
func (ata *AgentTaskAssignment) AssignTask() {
	ata.Status = AgentTaskStatusAssigned
	ata.AssignedAt = time.Now()
}

// StartTask 标记任务开始
// AgentTaskAssignment 结构体的方法 - 标记任务开始
func (ata *AgentTaskAssignment) StartTask() {
	ata.Status = AgentTaskStatusRunning
	now := time.Now()
	ata.StartedAt = &now
}

// CompleteTask 标记任务完成
// AgentTaskAssignment 结构体的方法 - 标记任务完成并记录结果
func (ata *AgentTaskAssignment) CompleteTask(result string) {
	ata.Status = AgentTaskStatusCompleted
	now := time.Now()
	ata.CompletedAt = &now
	ata.Result = result
}

// FailTask 标记任务失败
// AgentTaskAssignment 结构体的方法 - 标记任务失败并记录错误信息
func (ata *AgentTaskAssignment) FailTask(result string) {
	ata.Status = AgentTaskStatusFailed
	now := time.Now()
	ata.CompletedAt = &now
	ata.Result = result
}

// ============================================================================
// 相关实体：ScanType
// ============================================================================

// ScanType 扫描类型定义 (用来给Agent指定扫描类型和能力标注 - Agent-Capabilities)
type ScanType struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	// ID字段作为主键和业务标识，统一使用BaseModel.ID(uint64)
	basemodel.BaseModel

	Name           string             `json:"name" gorm:"not null;size:100;comment:扫描类型名称"`
	DisplayName    string             `json:"display_name" gorm:"not null;size:100;comment:扫描类型显示名称"`
	Description    string             `json:"description" gorm:"size:500;comment:扫描类型描述"`
	Category       string             `json:"category" gorm:"size:50;comment:扫描类型分类"`
	IsActive       bool               `json:"is_active" gorm:"default:true;comment:是否激活"`
	ConfigTemplate ConfigTemplateJSON `json:"config_template" gorm:"type:json;comment:配置模板"`
}

// TableName 定义表名
func (ScanType) TableName() string {
	return "agent_scan_types"
}

// IsActiveType 检查扫描类型是否激活
// ScanType 结构体的方法 - 检查扫描类型是否激活
func (st *ScanType) IsActiveType() bool {
	return st.IsActive
}

// TagType 标签类型定义 (用来给Agent指定标签 - Agent-Tags)
type TagType struct {
	// 引用基类 (ID, CreatedAt, UpdatedAt)
	// ID字段作为主键和业务标识，统一使用BaseModel.ID(uint64)
	basemodel.BaseModel

	Name        string `json:"name" gorm:"not null;size:100;comment:标签类型名称，唯一"`
	DisplayName string `json:"display_name" gorm:"not null;size:100;comment:标签类型显示名称"`
	Description string `json:"description" gorm:"size:500;comment:标签类型描述"`
	Remarks     string `json:"remarks" gorm:"size:500;comment:标签类型备注"`
}

func (TagType) TableName() string {
	return "agent_tag_types"
}
