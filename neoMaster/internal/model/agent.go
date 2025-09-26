/**
 * 模型:Agent 模型
 * @author: sun977
 * @date: 2025.09.26
 * @description: 定义 Agent 模型和相关的结构体
 * @func:
 */
package model

import (
	"neomaster/internal/pkg/utils"
	"time"
)

// Agent 状态枚举常量
type AgentStatus string

const (
	AgentStatusOnline      AgentStatus = "online"      // 在线
	AgentStatusOffline     AgentStatus = "offline"     // 离线
	AgentStatusException   AgentStatus = "exception"   // 异常
	AgentStatusMaintenance AgentStatus = "maintenance" // 维护
)

// Agent 工作状态常量枚举
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

// 1. Agent基础信息 - 相对静态，注册时确定
type Agent struct {
	// 基本信息
	ID        string      `json:"id" gorm:"primaryKey;autoIncrement"`
	AgentID   string      `json:"agent_id" gorm:"index"`
	Hostname  string      `json:"hostname"`
	IPAddress string      `json:"ip_address"`
	Port      int         `json:"port" gorm:"default:5772"`
	Version   string      `json:"version"`
	Status    AgentStatus `json:"status" gorm:"default:offline"` // online, offline, error, maintenance

	// 静态系统信息
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	CPUCores    int    `json:"cpu_cores"`
	MemoryTotal int64  `json:"memory_total"`
	DiskTotal   int64  `json:"disk_total"`

	// 能力和标签
	Capabilities []string `json:"capabilities" gorm:"type:json"` // 表示agent支持的功能模块
	Tags         []string `json:"tags" gorm:"type:json"`

	// 安全认证字段
	GRPCToken   string    `json:"grpc_token"`   // 用于gRPC通信的Token
	TokenExpiry time.Time `json:"token_expiry"` // Token过期时间

	// 时间戳
	ResultLatestTime *time.Time `json:"result_latest_time"` // 最新的返回结果时间
	LastHeartbeat    time.Time  `json:"last_heartbeat"`
	RegisteredAt     time.Time  `json:"registered_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// 扩展字段
	Remark string `json:"remark"` // 备注信息
}

// IsOnline 判断Agent是否在线
func (a *Agent) IsOnline() bool {
	return a.Status == AgentStatusOnline &&
		time.Since(a.LastHeartbeat) < 5*time.Minute
}

// CanAcceptTask 判断Agent是否可以接受指定类型的任务
func (a *Agent) CanAcceptTask(taskType string) bool {
	if !a.IsOnline() {
		return false
	}
	for _, capability := range a.Capabilities {
		if capability == taskType {
			return true
		}
	}
	return false
}

// TableName 定义Agent的数据库表名
func (Agent) TableName() string {
	return "agents"
}

// 2. Agent配置 - 独立管理，支持版本控制
type AgentConfig struct {
	ID                  string                 `json:"id" gorm:"primaryKey;autoIncrement"`
	AgentID             string                 `json:"agent_id" gorm:"index"`
	Version             int                    `json:"version" gorm:"default:1"`
	HeartbeatInterval   int                    `json:"heartbeat_interval"`             // 心跳间隔
	TaskPollInterval    int                    `json:"task_poll_interval"`             // 任务轮询间隔
	MaxConcurrentTasks  int                    `json:"max_concurrent_tasks"`           // 最大并发任务数
	PluginConfig        map[string]interface{} `json:"plugin_config" gorm:"type:json"` // 插件配置
	LogLevel            string                 `json:"log_level"`                      // 日志级别
	Timeout             int                    `json:"timeout"`                        // 超时时间
	TokenExpiryDuration int                    `json:"token_expiry_duration"`          // Token过期时间（秒）
	TokenNeverExpire    bool                   `json:"token_never_expire"`             // Token是否永不过期 true 表示永不过期
	IsActive            bool                   `json:"is_active" gorm:"default:true"`  // 是否激活
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// 3. Agent负载信息(动态) - 高频更新，独立存储
type AgentMetrics struct {
	ID                string                 `json:"id" gorm:"primaryKey;autoIncrement"`
	AgentID           string                 `json:"agent_id" gorm:"index"`
	CPUUsage          float64                `json:"cpu_usage"`
	MemoryUsage       float64                `json:"memory_usage"`
	DiskUsage         float64                `json:"disk_usage"`
	NetworkBytesSent  int64                  `json:"network_bytes_sent"`
	NetworkBytesRecv  int64                  `json:"network_bytes_recv"`
	ActiveConnections int                    `json:"active_connections"` // 活动连接数
	RunningTasks      int                    `json:"running_tasks"`      // 正在运行的任务数
	CompletedTasks    int                    `json:"completed_tasks"`    // 已完成任务数
	FailedTasks       int                    `json:"failed_tasks"`       // 失败任务数
	WorkStatus        AgentWorkStatus        `json:"work_status"`        // 工作状态：空闲/工作中/异常
	ScanType          string                 `json:"scan_type"`          // 扫描类型：空闲/IP探活/快速扫描/端口扫描/漏洞扫描等 [使用string为了内置扫描类型和自定义扫描类型的兼容]
	PluginStatus      map[string]interface{} `json:"plugin_status"`      // 插件状态信息 key: 插件名称, value: 插件状态详情【第三方工具都可以使用这一个字段】
	Timestamp         time.Time              `json:"timestamp" gorm:"index"`
}

// 统一的度量接口
type MetricsCollector interface {
	GetCPUUsage() float64
	GetMemoryUsage() float64
	GetDiskUsage() float64
	GetNetworkStats() (sent, recv int64)
	GetTaskStats() (running, completed, failed int)
}

// Agent实现度量收集
func (a *Agent) CollectAgentMetrics() *AgentMetrics {
	// 生成UUID [agent_550e8400-e29b-41d4-a716-446655440000]
	prefix := "agent_"
	uuid, _ := utils.GenerateUUIDWithPrefix(prefix)
	if uuid == "" {
		// 如果生成失败，使用时间戳
		uuid = time.Now().Format("20060102150405")
	}

	// TODO 收集Agent的其他指标

	return &AgentMetrics{
		AgentID: uuid,
		// ... 统一的度量收集逻辑
		Timestamp: time.Now(),
	}
}

// TableName 定义AgentMetrics的数据库表名
func (AgentMetrics) TableName() string {
	return "agent_metrics"
}

// 4. Agent分组
type AgentGroup struct {
	ID          string    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags" gorm:"type:json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AgentGroupMember struct {
	AgentID  string    `json:"agent_id" gorm:"primaryKey"`
	GroupID  string    `json:"group_id" gorm:"primaryKey"`
	JoinedAt time.Time `json:"joined_at"`
}

// 4. 添加任务分发记录
type AgentTaskAssignment struct {
	ID          string     `json:"id" gorm:"primaryKey;autoIncrement"`
	AgentID     string     `json:"agent_id" gorm:"index"`
	TaskID      string     `json:"task_id" gorm:"index"`
	TaskType    string     `json:"task_type"`
	AssignedAt  time.Time  `json:"assigned_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Status      string     `json:"status"` // assigned, running, completed, failed
	Result      string     `json:"result" gorm:"type:text"`
}

// Agent版本信息
type AgentVersion struct {
	ID          string    `json:"id" gorm:"primaryKey;autoIncrement"`
	Version     string    `json:"version" gorm:"not null"`
	ReleaseDate time.Time `json:"release_date"`
	Changelog   string    `json:"changelog"`
	DownloadURL string    `json:"download_url"`
	IsActive    bool      `json:"is_active"`
	IsLatest    bool      `json:"is_latest"`
}

// 扫描类型结构体 [为自定义扫描类型预留,系统默认内置扫描类型在代码中定义]
type ScanType struct {
	ID             string                 `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string                 `json:"name" gorm:"unique;not null"`
	DisplayName    string                 `json:"display_name" gorm:"not null"`
	Description    string                 `json:"description"`
	Category       string                 `json:"category"` // 扫描类型分类
	IsActive       bool                   `json:"is_active" gorm:"default:true"`
	ConfigTemplate map[string]interface{} `json:"config_template" gorm:"type:json"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// TableName 定义ScanType的数据库表名
func (ScanType) TableName() string {
	return "agent_scan_types"
}
