/**
 * 扫描工具执行器接口
 * @author: Sun977
 * @date: 2025.10.11
 * @description: 第三方扫描工具执行器接口，遵循"Never break userspace"原则
 * @func: 定义统一的扫描工具执行接口，支持nmap、masscan、nuclei等工具
 */
package executor_drop

import (
	"context"
	"time"

	"neomaster/internal/model/orchestrator_drop"
)

// ScanExecutor 扫描工具执行器接口
// 所有扫描工具都必须实现此接口，确保统一的执行标准
type ScanExecutor interface {
	// GetName 获取执行器名称
	GetName() string

	// GetVersion 获取执行器版本
	GetVersion() string

	// GetSupportedTools 获取支持的扫描工具列表
	GetSupportedTools() []string

	// IsToolSupported 检查是否支持指定工具
	IsToolSupported(toolName string) bool

	// ValidateConfig 验证工具配置是否正确
	ValidateConfig(tool *orchestrator_drop.ScanTool) error

	// Execute 执行扫描任务
	Execute(ctx context.Context, request *ScanRequest) (*ScanResult, error)

	// Stop 停止正在执行的扫描任务
	Stop(ctx context.Context, taskID string) error

	// GetStatus 获取扫描任务状态
	GetStatus(ctx context.Context, taskID string) (*ScanStatus, error)

	// Cleanup 清理资源
	Cleanup() error
}

// ScanRequest 扫描请求结构
type ScanRequest struct {
	TaskID      string                      `json:"task_id"`     // 任务ID，唯一标识
	Tool        *orchestrator_drop.ScanTool `json:"tool"`        // 扫描工具配置
	Target      string                      `json:"target"`      // 扫描目标（IP、域名、URL等）
	Options     map[string]interface{}      `json:"options"`     // 扫描选项
	Timeout     time.Duration               `json:"timeout"`     // 超时时间
	OutputPath  string                      `json:"output_path"` // 输出文件路径
	WorkingDir  string                      `json:"working_dir"` // 工作目录
	Environment map[string]string           `json:"environment"` // 环境变量
}

// ScanResult 扫描结果结构
type ScanResult struct {
	TaskID        string                 `json:"task_id"`        // 任务ID
	Status        ScanTaskStatus         `json:"status"`         // 执行状态
	StartTime     time.Time              `json:"start_time"`     // 开始时间
	EndTime       time.Time              `json:"end_time"`       // 结束时间
	Duration      time.Duration          `json:"duration"`       // 执行时长
	ExitCode      int                    `json:"exit_code"`      // 退出码
	Output        string                 `json:"output"`         // 标准输出
	Error         string                 `json:"error"`          // 错误输出
	OutputFiles   []string               `json:"output_files"`   // 输出文件列表
	Metadata      map[string]interface{} `json:"metadata"`       // 扩展元数据
	ResourceUsage *ResourceUsage         `json:"resource_usage"` // 资源使用情况
}

// ScanStatus 扫描状态结构
type ScanStatus struct {
	TaskID        string         `json:"task_id"`        // 任务ID
	Status        ScanTaskStatus `json:"status"`         // 当前状态
	Progress      float64        `json:"progress"`       // 进度百分比 (0-100)
	Message       string         `json:"message"`        // 状态消息
	StartTime     time.Time      `json:"start_time"`     // 开始时间
	ElapsedTime   time.Duration  `json:"elapsed_time"`   // 已用时间
	EstimatedTime time.Duration  `json:"estimated_time"` // 预计剩余时间
}

// ScanTaskStatus 扫描任务状态枚举
type ScanTaskStatus int

const (
	ScanTaskStatusPending   ScanTaskStatus = 0 // 等待中
	ScanTaskStatusRunning   ScanTaskStatus = 1 // 运行中
	ScanTaskStatusCompleted ScanTaskStatus = 2 // 已完成
	ScanTaskStatusFailed    ScanTaskStatus = 3 // 失败
	ScanTaskStatusCancelled ScanTaskStatus = 4 // 已取消
	ScanTaskStatusTimeout   ScanTaskStatus = 5 // 超时
)

// String 实现Stringer接口
func (s ScanTaskStatus) String() string {
	switch s {
	case ScanTaskStatusPending:
		return "pending"
	case ScanTaskStatusRunning:
		return "running"
	case ScanTaskStatusCompleted:
		return "completed"
	case ScanTaskStatusFailed:
		return "failed"
	case ScanTaskStatusCancelled:
		return "cancelled"
	case ScanTaskStatusTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUPercent   float64 `json:"cpu_percent"`    // CPU使用率
	MemoryMB     float64 `json:"memory_mb"`      // 内存使用量(MB)
	DiskReadMB   float64 `json:"disk_read_mb"`   // 磁盘读取量(MB)
	DiskWriteMB  float64 `json:"disk_write_mb"`  // 磁盘写入量(MB)
	NetworkInMB  float64 `json:"network_in_mb"`  // 网络接收量(MB)
	NetworkOutMB float64 `json:"network_out_mb"` // 网络发送量(MB)
}

// ExecutorManager 执行器管理器接口
type ExecutorManager interface {
	// RegisterExecutor 注册执行器
	RegisterExecutor(executor ScanExecutor) error

	// GetExecutor 获取指定工具的执行器
	GetExecutor(toolName string) (ScanExecutor, error)

	// GetAllExecutors 获取所有执行器
	GetAllExecutors() []ScanExecutor

	// UnregisterExecutor 注销执行器
	UnregisterExecutor(executorName string) error

	// ExecuteTask 执行扫描任务
	ExecuteTask(ctx context.Context, request *ScanRequest) (*ScanResult, error)

	// StopTask 停止扫描任务
	StopTask(ctx context.Context, taskID string) error

	// GetTaskStatus 获取任务状态
	GetTaskStatus(ctx context.Context, taskID string) (*ScanStatus, error)

	// Shutdown 关闭管理器
	Shutdown() error
}
