/**
 * 执行器基础接口
 * @author: sun977
 * @date: 2025.10.21
 * @description: 定义所有扫描工具执行器的统一接口
 * @func: 占位符实现，待后续完善
 */
package base

import (
	"context"
	"time"
)

// ExecutorType 执行器类型
type ExecutorType string

const (
	ExecutorTypeSystem  ExecutorType = "system"  // 系统通用执行器
	ExecutorTypeNmap    ExecutorType = "nmap"    // Nmap执行器
	ExecutorTypeNuclei  ExecutorType = "nuclei"  // Nuclei执行器
	ExecutorTypeMasscan ExecutorType = "masscan" // Masscan执行器
	ExecutorTypeCustom  ExecutorType = "custom"  // 自定义执行器
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 等待执行
	TaskStatusRunning   TaskStatus = "running"   // 正在执行
	TaskStatusCompleted TaskStatus = "completed" // 执行完成
	TaskStatusFailed    TaskStatus = "failed"    // 执行失败
	TaskStatusCanceled  TaskStatus = "canceled"  // 已取消
	TaskStatusTimeout   TaskStatus = "timeout"   // 执行超时
)

// Executor 执行器接口
type Executor interface {
	// ==================== 基础信息 ====================
	GetType() ExecutorType     // 获取执行器类型
	GetName() string           // 获取执行器名称
	GetVersion() string        // 获取执行器版本
	GetDescription() string    // 获取执行器描述
	GetCapabilities() []string // 获取执行器能力列表

	// ==================== 生命周期管理 ====================
	Initialize(config *ExecutorConfig) error // 初始化执行器
	Start() error                            // 启动执行器
	Stop() error                             // 停止执行器
	Restart() error                          // 重启执行器
	IsRunning() bool                         // 检查执行器是否运行中
	GetStatus() *ExecutorStatus              // 获取执行器状态

	// ==================== 任务执行 ====================
	Execute(ctx context.Context, task *Task) (*TaskResult, error) // 执行任务
	Cancel(taskID string) error                                   // 取消任务
	Pause(taskID string) error                                    // 暂停任务
	Resume(taskID string) error                                   // 恢复任务

	// ==================== 任务管理 ====================
	GetTask(taskID string) (*Task, error)             // 获取任务信息
	GetTasks() ([]*Task, error)                       // 获取所有任务
	GetTaskStatus(taskID string) (TaskStatus, error)  // 获取任务状态
	GetTaskResult(taskID string) (*TaskResult, error) // 获取任务结果
	GetTaskLogs(taskID string) ([]string, error)      // 获取任务日志

	// ==================== 配置管理 ====================
	UpdateConfig(config *ExecutorConfig) error   // 更新配置
	GetConfig() *ExecutorConfig                  // 获取当前配置
	ValidateConfig(config *ExecutorConfig) error // 验证配置

	// ==================== 健康检查 ====================
	HealthCheck() *HealthStatus   // 健康检查
	GetMetrics() *ExecutorMetrics // 获取执行器指标

	// ==================== 资源管理 ====================
	GetResourceUsage() *ResourceUsage // 获取资源使用情况
	CleanupResources() error          // 清理资源
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	Type           ExecutorType           `json:"type"`            // 执行器类型
	Name           string                 `json:"name"`            // 执行器名称
	Version        string                 `json:"version"`         // 版本
	MaxConcurrency int                    `json:"max_concurrency"` // 最大并发数
	TaskTimeout    time.Duration          `json:"task_timeout"`    // 任务超时时间
	RetryCount     int                    `json:"retry_count"`     // 重试次数
	RetryDelay     time.Duration          `json:"retry_delay"`     // 重试延迟
	WorkDir        string                 `json:"work_dir"`        // 工作目录
	TempDir        string                 `json:"temp_dir"`        // 临时目录
	LogLevel       string                 `json:"log_level"`       // 日志级别
	EnableMetrics  bool                   `json:"enable_metrics"`  // 是否启用指标收集
	Custom         map[string]interface{} `json:"custom"`          // 自定义配置
	ToolPath       string                 `json:"tool_path"`       // 工具路径
	ToolArgs       []string               `json:"tool_args"`       // 工具参数
	Environment    map[string]string      `json:"environment"`     // 环境变量
	ResourceLimits *ResourceLimits        `json:"resource_limits"` // 资源限制
	CreatedAt      time.Time              `json:"created_at"`      // 创建时间
	UpdatedAt      time.Time              `json:"updated_at"`      // 更新时间
}

// ResourceLimits 资源限制
type ResourceLimits struct {
	MaxCPU    float64 `json:"max_cpu"`    // 最大CPU使用率
	MaxMemory int64   `json:"max_memory"` // 最大内存使用量(字节)
	MaxDisk   int64   `json:"max_disk"`   // 最大磁盘使用量(字节)
	MaxFiles  int     `json:"max_files"`  // 最大文件数
}

// ExecutorStatus 执行器状态
type ExecutorStatus struct {
	Type           ExecutorType  `json:"type"`            // 执行器类型
	Name           string        `json:"name"`            // 执行器名称
	Status         string        `json:"status"`          // 状态: running, stopped, error
	IsRunning      bool          `json:"is_running"`      // 是否运行中
	StartTime      time.Time     `json:"start_time"`      // 启动时间
	LastActivity   time.Time     `json:"last_activity"`   // 最后活动时间
	TaskCount      int           `json:"task_count"`      // 当前任务数
	CompletedTasks int           `json:"completed_tasks"` // 已完成任务数
	FailedTasks    int           `json:"failed_tasks"`    // 失败任务数
	ErrorMessage   string        `json:"error_message"`   // 错误消息
	Uptime         time.Duration `json:"uptime"`          // 运行时间
	Timestamp      time.Time     `json:"timestamp"`       // 状态时间戳
}

// Task 任务定义
type Task struct {
	ID           string                 `json:"id"`            // 任务ID
	Name         string                 `json:"name"`          // 任务名称
	Type         string                 `json:"type"`          // 任务类型
	ExecutorType ExecutorType           `json:"executor_type"` // 执行器类型
	Status       TaskStatus             `json:"status"`        // 任务状态
	Priority     int                    `json:"priority"`      // 优先级
	Config       map[string]interface{} `json:"config"`        // 任务配置
	Input        *TaskInput             `json:"input"`         // 任务输入
	Output       *TaskOutput            `json:"output"`        // 任务输出
	Progress     float64                `json:"progress"`      // 执行进度 (0-100)
	StartTime    time.Time              `json:"start_time"`    // 开始时间
	EndTime      time.Time              `json:"end_time"`      // 结束时间
	Timeout      time.Duration          `json:"timeout"`       // 超时时间
	RetryCount   int                    `json:"retry_count"`   // 重试次数
	MaxRetries   int                    `json:"max_retries"`   // 最大重试次数
	ErrorMessage string                 `json:"error_message"` // 错误消息
	Logs         []string               `json:"logs"`          // 执行日志
	Metrics      *TaskMetrics           `json:"metrics"`       // 任务指标
	CreatedAt    time.Time              `json:"created_at"`    // 创建时间
	UpdatedAt    time.Time              `json:"updated_at"`    // 更新时间
}

// TaskInput 任务输入
type TaskInput struct {
	Targets      []string               `json:"targets"`       // 扫描目标
	Ports        []string               `json:"ports"`         // 端口范围
	Options      map[string]interface{} `json:"options"`       // 扫描选项
	Scripts      []string               `json:"scripts"`       // 脚本列表
	OutputFormat string                 `json:"output_format"` // 输出格式
	OutputFile   string                 `json:"output_file"`   // 输出文件
	Custom       map[string]interface{} `json:"custom"`        // 自定义输入
}

// TaskOutput 任务输出
type TaskOutput struct {
	Results    []ScanResult           `json:"results"`    // 扫描结果
	Summary    *ScanSummary           `json:"summary"`    // 扫描摘要
	Files      []string               `json:"files"`      // 输出文件列表
	Errors     []string               `json:"errors"`     // 错误列表
	Warnings   []string               `json:"warnings"`   // 警告列表
	Statistics map[string]interface{} `json:"statistics"` // 统计信息
	Custom     map[string]interface{} `json:"custom"`     // 自定义输出
}

// ScanResult 扫描结果
type ScanResult struct {
	Target          string                 `json:"target"`          // 扫描目标
	Port            int                    `json:"port"`            // 端口
	Protocol        string                 `json:"protocol"`        // 协议
	Service         string                 `json:"service"`         // 服务
	Version         string                 `json:"version"`         // 版本
	State           string                 `json:"state"`           // 状态
	Banner          string                 `json:"banner"`          // 横幅信息
	Vulnerabilities []Vulnerability        `json:"vulnerabilities"` // 漏洞列表
	Extra           map[string]interface{} `json:"extra"`           // 额外信息
	Timestamp       time.Time              `json:"timestamp"`       // 扫描时间
}

// Vulnerability 漏洞信息
type Vulnerability struct {
	ID          string                 `json:"id"`          // 漏洞ID
	Name        string                 `json:"name"`        // 漏洞名称
	Severity    string                 `json:"severity"`    // 严重程度
	CVSS        float64                `json:"cvss"`        // CVSS评分
	CVE         string                 `json:"cve"`         // CVE编号
	Description string                 `json:"description"` // 漏洞描述
	Solution    string                 `json:"solution"`    // 解决方案
	References  []string               `json:"references"`  // 参考链接
	Tags        []string               `json:"tags"`        // 标签
	Extra       map[string]interface{} `json:"extra"`       // 额外信息
}

// ScanSummary 扫描摘要
type ScanSummary struct {
	TotalTargets    int           `json:"total_targets"`   // 总目标数
	ScannedTargets  int           `json:"scanned_targets"` // 已扫描目标数
	OpenPorts       int           `json:"open_ports"`      // 开放端口数
	ClosedPorts     int           `json:"closed_ports"`    // 关闭端口数
	FilteredPorts   int           `json:"filtered_ports"`  // 过滤端口数
	Services        int           `json:"services"`        // 发现服务数
	Vulnerabilities int           `json:"vulnerabilities"` // 发现漏洞数
	Duration        time.Duration `json:"duration"`        // 扫描耗时
	StartTime       time.Time     `json:"start_time"`      // 开始时间
	EndTime         time.Time     `json:"end_time"`        // 结束时间
}

// TaskResult 任务结果
type TaskResult struct {
	TaskID    string        `json:"task_id"`    // 任务ID
	Status    TaskStatus    `json:"status"`     // 执行状态
	Success   bool          `json:"success"`    // 是否成功
	Output    *TaskOutput   `json:"output"`     // 任务输出
	Error     string        `json:"error"`      // 错误信息
	Logs      []string      `json:"logs"`       // 执行日志
	Metrics   *TaskMetrics  `json:"metrics"`    // 任务指标
	StartTime time.Time     `json:"start_time"` // 开始时间
	EndTime   time.Time     `json:"end_time"`   // 结束时间
	Duration  time.Duration `json:"duration"`   // 执行耗时
	Timestamp time.Time     `json:"timestamp"`  // 结果时间戳
}

// TaskMetrics 任务指标
type TaskMetrics struct {
	CPUTime      time.Duration `json:"cpu_time"`      // CPU时间
	MemoryPeak   int64         `json:"memory_peak"`   // 内存峰值
	DiskRead     int64         `json:"disk_read"`     // 磁盘读取
	DiskWrite    int64         `json:"disk_write"`    // 磁盘写入
	NetworkRead  int64         `json:"network_read"`  // 网络读取
	NetworkWrite int64         `json:"network_write"` // 网络写入
	ProcessCount int           `json:"process_count"` // 进程数
	ThreadCount  int           `json:"thread_count"`  // 线程数
	FileCount    int           `json:"file_count"`    // 文件数
}

// HealthStatus 健康状态
type HealthStatus struct {
	IsHealthy    bool                   `json:"is_healthy"`    // 是否健康
	Status       string                 `json:"status"`        // 状态描述
	Checks       map[string]CheckResult `json:"checks"`        // 检查结果
	LastCheck    time.Time              `json:"last_check"`    // 最后检查时间
	ErrorMessage string                 `json:"error_message"` // 错误消息
}

// CheckResult 检查结果
type CheckResult struct {
	Name      string        `json:"name"`      // 检查名称
	Status    string        `json:"status"`    // 检查状态
	Message   string        `json:"message"`   // 检查消息
	Duration  time.Duration `json:"duration"`  // 检查耗时
	Timestamp time.Time     `json:"timestamp"` // 检查时间
}

// ExecutorMetrics 执行器指标
type ExecutorMetrics struct {
	TasksTotal      int64         `json:"tasks_total"`       // 总任务数
	TasksCompleted  int64         `json:"tasks_completed"`   // 完成任务数
	TasksFailed     int64         `json:"tasks_failed"`      // 失败任务数
	TasksCanceled   int64         `json:"tasks_canceled"`    // 取消任务数
	TasksRunning    int64         `json:"tasks_running"`     // 运行中任务数
	AverageTaskTime time.Duration `json:"average_task_time"` // 平均任务时间
	TotalCPUTime    time.Duration `json:"total_cpu_time"`    // 总CPU时间
	TotalMemoryUsed int64         `json:"total_memory_used"` // 总内存使用
	ErrorRate       float64       `json:"error_rate"`        // 错误率
	Throughput      float64       `json:"throughput"`        // 吞吐量
	Timestamp       time.Time     `json:"timestamp"`         // 指标时间戳
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUUsage     float64   `json:"cpu_usage"`     // CPU使用率
	MemoryUsage  int64     `json:"memory_usage"`  // 内存使用量
	DiskUsage    int64     `json:"disk_usage"`    // 磁盘使用量
	NetworkUsage int64     `json:"network_usage"` // 网络使用量
	FileCount    int       `json:"file_count"`    // 文件数量
	ProcessCount int       `json:"process_count"` // 进程数量
	ThreadCount  int       `json:"thread_count"`  // 线程数量
	Timestamp    time.Time `json:"timestamp"`     // 使用情况时间戳
}
