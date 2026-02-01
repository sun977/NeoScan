/**
 * Agent任务管理服务
 * @author: sun977
 * @date: 2025.10.21
 * @description: 处理Agent与Master端的任务交互（Outbound）和本地任务管理（Inbound）
 * @func:
 *  1. Outbound: 轮询Master任务 -> 转换 -> 执行 -> 上报结果
 *  2. Inbound: 响应API请求 -> 控制任务状态
 */
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"neoagent/internal/config"
	"neoagent/internal/core/runner"
	modelComm "neoagent/internal/model/client"
	"neoagent/internal/pkg/logger"
	"neoagent/internal/service/adapter"
	"neoagent/internal/service/client"
)

// AgentTaskService Agent任务管理服务接口
type AgentTaskService interface {
	// ==================== Lifecycle Methods (Outbound 能力) ====================
	// StartWorker 启动任务轮询工作者，负责从Master拉取任务并执行
	StartWorker(ctx context.Context, interval time.Duration)

	// ==================== Agent任务管理（Inbound 能力 - 响应Master端/本地API命令） ====================
	GetTaskList(ctx context.Context) ([]*Task, error)          // 获取Agent任务列表 [响应Master端GET /:id/tasks]
	CreateTask(ctx context.Context, task *Task) (*Task, error) // 创建新任务 [响应Master端POST /:id/tasks]
	GetTask(ctx context.Context, taskID string) (*Task, error) // 获取特定任务信息 [响应Master端GET /:id/tasks/:task_id]
	DeleteTask(ctx context.Context, taskID string) error       // 删除任务 [响应Master端DELETE /:id/tasks/:task_id]

	// ==================== 任务执行控制（Inbound & Internal） ====================
	StartTask(ctx context.Context, taskID string) error                    // 启动任务执行
	StopTask(ctx context.Context, taskID string) error                     // 停止任务执行
	PauseTask(ctx context.Context, taskID string) error                    // 暂停任务执行
	ResumeTask(ctx context.Context, taskID string) error                   // 恢复任务执行
	GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) // 获取任务执行状态

	// ==================== 任务结果管理（Inbound） ====================
	GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) // 获取任务执行结果
	GetTaskLog(ctx context.Context, taskID string) ([]string, error)       // 获取任务执行日志
	CleanupTask(ctx context.Context, taskID string) error                  // 清理任务资源
}

// agentTaskService Agent任务管理服务实现
type agentTaskService struct {
	masterService client.MasterService
	runnerManager *runner.RunnerManager
	translator    *adapter.TaskTranslator
	config        *config.Config

	// runningTasks 维护正在运行的任务的取消函数
	// Key: TaskID, Value: CancelFunc
	runningTasks map[string]context.CancelFunc
	mu           sync.RWMutex
}

// NewAgentTaskService 创建Agent任务管理服务实例
// 注入必要的依赖：Master通信服务、Runner管理器、任务转换器、配置
func NewAgentTaskService(
	masterService client.MasterService, // Master通信服务
	runnerManager *runner.RunnerManager, // Runner管理器
	translator *adapter.TaskTranslator, // 任务转换器
	cfg *config.Config,
) AgentTaskService {
	return &agentTaskService{
		masterService: masterService,
		runnerManager: runnerManager,
		translator:    translator,
		config:        cfg,
		runningTasks:  make(map[string]context.CancelFunc),
	}
}

// ==================== Lifecycle Methods (Outbound 能力) ====================

// StartWorker 启动任务轮询工作者
// 这是一个阻塞调用（通常在goroutine中运行），直到ctx被取消
func (s *agentTaskService) StartWorker(ctx context.Context, interval time.Duration) {
	logger.LogSystemEvent("TaskService", "Worker", "Starting task worker loop...", logger.InfoLevel, nil)

	// 1. 启动 Poller 获取任务通道
	taskChan := s.masterService.StartTaskPoller(ctx, interval)

	// 2. 消费任务
	for {
		select {
		case <-ctx.Done():
			logger.LogSystemEvent("TaskService", "Worker", "Task worker loop stopped", logger.InfoLevel, nil)
			return
		case tasks, ok := <-taskChan:
			if !ok {
				return
			}
			// 处理一批任务
			for _, task := range tasks {
				// 并发处理任务
				go s.processTask(ctx, task)
			}
		}
	}
}

// processTask 处理单个任务（Outbound 核心逻辑）
func (s *agentTaskService) processTask(parentCtx context.Context, task modelComm.Task) {
	taskID := task.TaskID
	logger.LogSystemEvent("TaskService", "ProcessTask", fmt.Sprintf("Processing task: %s (%s)", taskID, task.TaskType), logger.InfoLevel, nil)

	// 1. 上报状态：Running
	if err := s.masterService.ReportTask(parentCtx, taskID, "running", "", ""); err != nil {
		logger.LogSystemEvent("TaskService", "ReportTask", fmt.Sprintf("Failed to report running status for task %s: %v", taskID, err), logger.ErrorLevel, nil)
		// 即使上报失败，也尝试继续执行，或者选择终止
	}

	// 2. 创建任务上下文（用于支持取消）
	ctx, cancel := context.WithCancel(parentCtx)
	s.mu.Lock()
	s.runningTasks[taskID] = cancel
	s.mu.Unlock()

	// 确保任务结束时清理
	defer func() {
		s.mu.Lock()
		delete(s.runningTasks, taskID)
		s.mu.Unlock()
		cancel()
	}()

	// 3. 转换任务模型 (Master Model -> Core Model)
	coreTask, err := s.translator.ToCoreTask(&task)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to translate task: %v", err)
		logger.LogSystemEvent("TaskService", "TranslateTask", fmt.Sprintf("%s: %v", errMsg, err), logger.ErrorLevel, nil)
		s.masterService.ReportTask(parentCtx, taskID, "failed", "", errMsg)
		return
	}

	// 4. 执行任务
	results, err := s.runnerManager.Execute(ctx, coreTask)

	// 5. 处理结果并上报
	if err != nil {
		// 任务执行失败
		errMsg := fmt.Sprintf("Task execution failed: %v", err)
		logger.LogSystemEvent("TaskService", "ExecuteTask", fmt.Sprintf("%s: %v", errMsg, err), logger.ErrorLevel, nil)
		s.masterService.ReportTask(parentCtx, taskID, "failed", "", errMsg)
	} else {
		// 任务执行成功
		// 序列化结果
		resultJSON, _ := json.Marshal(results)
		// 注意：ReportTask 的 result 字段可能需要根据 Master 的期望格式进行调整
		// 这里简单将 coreModel.TaskResult 数组序列化后上报
		if err := s.masterService.ReportTask(parentCtx, taskID, "completed", string(resultJSON), ""); err != nil {
			logger.LogSystemEvent("TaskService", "ReportResult", fmt.Sprintf("Failed to report completion for task %s: %v", taskID, err), logger.ErrorLevel, nil)
		} else {
			logger.LogSystemEvent("TaskService", "TaskCompleted", fmt.Sprintf("Task %s completed successfully", taskID), logger.InfoLevel, nil)
		}
	}
}

// ==================== Agent任务管理实现 (Inbound 能力) ====================

// GetTaskList 获取Agent任务列表
func (s *agentTaskService) GetTaskList(ctx context.Context) ([]*Task, error) {
	// TODO: 这里目前仅返回内存中正在运行的任务，后续如果有持久化存储，需要查询数据库
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.runningTasks))
	for id := range s.runningTasks {
		tasks = append(tasks, &Task{
			ID:        id,
			Status:    "running",
			CreatedAt: time.Now(), // 这是一个估算值，因为 map 没存时间
		})
	}
	return tasks, nil
}

// CreateTask 创建新任务 (通常用于 Debug 或 本地触发)
func (s *agentTaskService) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	// 实际场景中，Agent 主要被动接收 Master 任务。
	// 如果支持本地创建，逻辑类似 processTask，但需要先构造 modelComm.Task
	return nil, fmt.Errorf("CreateTask not fully implemented for local trigger yet")
}

// GetTask 获取特定任务信息
func (s *agentTaskService) GetTask(ctx context.Context, taskID string) (*Task, error) {
	s.mu.RLock()
	_, ok := s.runningTasks[taskID]
	s.mu.RUnlock()

	if ok {
		return &Task{
			ID:     taskID,
			Status: "running",
		}, nil
	}
	return nil, fmt.Errorf("task not found or not running: %s", taskID)
}

// DeleteTask 删除任务 (通常意味着停止并清理)
func (s *agentTaskService) DeleteTask(ctx context.Context, taskID string) error {
	return s.StopTask(ctx, taskID)
}

// ==================== 任务执行控制实现 ====================

// StartTask 启动任务执行 (用于重试或手动启动，暂未实现)
func (s *agentTaskService) StartTask(ctx context.Context, taskID string) error {
	return fmt.Errorf("StartTask not implemented, use Master distribution for new tasks")
}

// StopTask 停止任务执行 (Inbound -> Outbound Control)
func (s *agentTaskService) StopTask(ctx context.Context, taskID string) error {
	s.mu.Lock()
	cancel, ok := s.runningTasks[taskID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("task not running: %s", taskID)
	}

	// 调用 cancel 函数，这将导致 processTask 中的 runnerManager.Execute 接收到 context done 信号
	cancel()
	logger.LogSystemEvent("TaskService", "StopTask", fmt.Sprintf("Stop signal sent to task: %s", taskID), logger.InfoLevel, nil)
	return nil
}

// PauseTask 暂停任务执行
func (s *agentTaskService) PauseTask(ctx context.Context, taskID string) error {
	return fmt.Errorf("PauseTask functionality not supported by current runners")
}

// ResumeTask 恢复任务执行
func (s *agentTaskService) ResumeTask(ctx context.Context, taskID string) error {
	return fmt.Errorf("ResumeTask functionality not supported by current runners")
}

// GetTaskStatus 获取任务执行状态
func (s *agentTaskService) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	s.mu.RLock()
	_, ok := s.runningTasks[taskID]
	s.mu.RUnlock()

	status := "unknown"
	if ok {
		status = "running"
	}

	return &TaskStatus{
		TaskID:    taskID,
		Status:    status,
		Timestamp: time.Now(),
	}, nil
}

// ==================== 任务结果管理实现 ====================

// GetTaskResult 获取任务执行结果
func (s *agentTaskService) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	return nil, fmt.Errorf("local result storage not implemented, please check Master")
}

// GetTaskLog 获取任务执行日志
func (s *agentTaskService) GetTaskLog(ctx context.Context, taskID string) ([]string, error) {
	return []string{}, fmt.Errorf("local log storage not implemented")
}

// CleanupTask 清理任务资源
func (s *agentTaskService) CleanupTask(ctx context.Context, taskID string) error {
	// StopTask 已经包含了基本的清理（通过 defer）
	// 如果有额外的磁盘文件清理，可以在这里实现
	return nil
}

// ==================== 数据模型定义 (Service Level DTOs) ====================

// Task 任务信息 (API Response Object)
type Task struct {
	ID          string                 `json:"id"`           // 任务ID
	Name        string                 `json:"name"`         // 任务名称
	Type        string                 `json:"type"`         // 任务类型：scan, monitor, update等
	Status      string                 `json:"status"`       // 任务状态：pending, running, completed, failed, paused
	Priority    int                    `json:"priority"`     // 任务优先级
	Config      map[string]interface{} `json:"config"`       // 任务配置参数
	CreatedAt   time.Time              `json:"created_at"`   // 创建时间
	StartedAt   *time.Time             `json:"started_at"`   // 开始时间
	CompletedAt *time.Time             `json:"completed_at"` // 完成时间
	// TODO: 添加更多任务字段
	// Target      string    `json:"target"`       // 扫描目标
	// Progress    int       `json:"progress"`     // 执行进度
	// ErrorMsg    string    `json:"error_msg"`    // 错误信息
}

// TaskStatus 任务执行状态
type TaskStatus struct {
	TaskID    string    `json:"task_id"`   // 任务ID
	Status    string    `json:"status"`    // 执行状态
	Progress  int       `json:"progress"`  // 执行进度（0-100）
	Message   string    `json:"message"`   // 状态描述
	Timestamp time.Time `json:"timestamp"` // 状态更新时间
	// TODO: 添加更多状态字段
	// CPUUsage    float64 `json:"cpu_usage"`    // CPU使用率
	// MemoryUsage float64 `json:"memory_usage"` // 内存使用率
	// NetworkIO   int64   `json:"network_io"`   // 网络IO
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID    string    `json:"task_id"`   // 任务ID
	Status    string    `json:"status"`    // 执行状态：success, failed
	Message   string    `json:"message"`   // 结果描述
	Data      any       `json:"data"`      // 结果数据
	Timestamp time.Time `json:"timestamp"` // 结果时间戳
	// TODO: 添加更多结果字段
	// Duration    time.Duration `json:"duration"`     // 执行耗时
	// ResultCount int           `json:"result_count"` // 结果数量
	// ErrorCount  int           `json:"error_count"`  // 错误数量
}
