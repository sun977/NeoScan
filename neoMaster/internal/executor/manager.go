/**
 * 扫描工具执行器管理器实现
 * @author: Sun977
 * @date: 2025.10.11
 * @description: 管理所有扫描工具执行器，提供统一的任务调度和状态管理
 * @func: 执行器注册、任务执行、状态监控等
 */
package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"neomaster/internal/pkg/logger"
)

// DefaultExecutorManager 默认执行器管理器实现
type DefaultExecutorManager struct {
	executors map[string]ScanExecutor // 执行器映射 toolName -> executor
	tasks     map[string]*TaskInfo    // 任务映射 taskID -> taskInfo
	mutex     sync.RWMutex            // 读写锁
}

// TaskInfo 任务信息
type TaskInfo struct {
	Request   *ScanRequest       `json:"request"`    // 扫描请求
	Executor  ScanExecutor       `json:"-"`          // 执行器实例
	Status    *ScanStatus        `json:"status"`     // 任务状态
	Result    *ScanResult        `json:"result"`     // 扫描结果
	CancelCtx context.Context    `json:"-"`          // 取消上下文
	Cancel    context.CancelFunc `json:"-"`          // 取消函数
	CreatedAt time.Time          `json:"created_at"` // 创建时间
}

// NewExecutorManager 创建新的执行器管理器
func NewExecutorManager() *DefaultExecutorManager {
	return &DefaultExecutorManager{
		executors: make(map[string]ScanExecutor),
		tasks:     make(map[string]*TaskInfo),
	}
}

// RegisterExecutor 注册执行器
func (m *DefaultExecutorManager) RegisterExecutor(executor ScanExecutor) error {
	if executor == nil {
		return fmt.Errorf("executor cannot be nil")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	executorName := executor.GetName()
	if executorName == "" {
		return fmt.Errorf("executor name cannot be empty")
	}

	// 检查是否已注册
	if _, exists := m.executors[executorName]; exists {
		return fmt.Errorf("executor %s already registered", executorName)
	}

	// 注册支持的工具
	supportedTools := executor.GetSupportedTools()
	for _, toolName := range supportedTools {
		if existing, exists := m.executors[toolName]; exists {
			logger.LogWarn(fmt.Sprintf("Tool %s already supported by executor %s, will be overridden by %s",
				toolName, existing.GetName(), executorName), "executor.manager.RegisterExecutor", 0, "", "", "", map[string]interface{}{
				"operation": "register_executor",
				"option":    "check_tool_conflict",
				"func_name": "executor.manager.RegisterExecutor",
			})
		}
		m.executors[toolName] = executor
	}

	logger.LogInfo(fmt.Sprintf("Registered executor %s with tools: %v", executorName, supportedTools), "executor.manager.RegisterExecutor", 0, "", "", "", map[string]interface{}{
		"operation": "register_executor",
		"option":    "register_success",
		"func_name": "executor.manager.RegisterExecutor",
	})

	return nil
}

// GetExecutor 获取指定工具的执行器
func (m *DefaultExecutorManager) GetExecutor(toolName string) (ScanExecutor, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	executor, exists := m.executors[toolName]
	if !exists {
		return nil, fmt.Errorf("no executor found for tool: %s", toolName)
	}

	return executor, nil
}

// GetAllExecutors 获取所有执行器
func (m *DefaultExecutorManager) GetAllExecutors() []ScanExecutor {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	executorMap := make(map[string]ScanExecutor)
	for _, executor := range m.executors {
		executorMap[executor.GetName()] = executor
	}

	executors := make([]ScanExecutor, 0, len(executorMap))
	for _, executor := range executorMap {
		executors = append(executors, executor)
	}

	return executors
}

// UnregisterExecutor 注销执行器
func (m *DefaultExecutorManager) UnregisterExecutor(executorName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 查找并移除所有相关的工具映射
	var removedTools []string
	for toolName, executor := range m.executors {
		if executor.GetName() == executorName {
			delete(m.executors, toolName)
			removedTools = append(removedTools, toolName)
		}
	}

	if len(removedTools) == 0 {
		return fmt.Errorf("executor %s not found", executorName)
	}

	logger.LogInfo(fmt.Sprintf("Unregistered executor %s, removed tools: %v", executorName, removedTools), "executor.manager.UnregisterExecutor", 0, "", "", "", map[string]interface{}{
		"operation": "unregister_executor",
		"option":    "unregister_success",
		"func_name": "executor.manager.UnregisterExecutor",
	})

	return nil
}

// ExecuteTask 执行扫描任务
func (m *DefaultExecutorManager) ExecuteTask(ctx context.Context, request *ScanRequest) (*ScanResult, error) {
	if request == nil {
		return nil, fmt.Errorf("scan request cannot be nil")
	}

	if request.TaskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	if request.Tool == nil {
		return nil, fmt.Errorf("scan tool cannot be nil")
	}

	// 获取执行器
	executor, err := m.GetExecutor(request.Tool.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor for tool %s: %w", request.Tool.Name, err)
	}

	// 验证工具配置
	if err := executor.ValidateConfig(request.Tool); err != nil {
		return nil, fmt.Errorf("invalid tool config: %w", err)
	}

	// 创建任务上下文
	taskCtx, cancel := context.WithCancel(ctx)
	if request.Timeout > 0 {
		taskCtx, cancel = context.WithTimeout(ctx, request.Timeout)
	}

	// 创建任务信息
	taskInfo := &TaskInfo{
		Request:  request,
		Executor: executor,
		Status: &ScanStatus{
			TaskID:    request.TaskID,
			Status:    ScanTaskStatusPending,
			Message:   "Task created",
			StartTime: time.Now(),
		},
		CancelCtx: taskCtx,
		Cancel:    cancel,
		CreatedAt: time.Now(),
	}

	// 注册任务
	m.mutex.Lock()
	if _, exists := m.tasks[request.TaskID]; exists {
		m.mutex.Unlock()
		cancel()
		return nil, fmt.Errorf("task %s already exists", request.TaskID)
	}
	m.tasks[request.TaskID] = taskInfo
	m.mutex.Unlock()

	logger.LogInfo(fmt.Sprintf("Created task %s for tool %s, target: %s", request.TaskID, request.Tool.Name, request.Target), "executor.manager.ExecuteTask", 0, "", "", "", map[string]interface{}{
		"operation": "execute_task",
		"option":    "task_created",
		"func_name": "executor.manager.ExecuteTask",
	})

	// 异步执行任务
	go m.executeTaskAsync(taskInfo)

	return &ScanResult{
		TaskID:    request.TaskID,
		Status:    ScanTaskStatusPending,
		StartTime: time.Now(),
	}, nil
}

// executeTaskAsync 异步执行任务
func (m *DefaultExecutorManager) executeTaskAsync(taskInfo *TaskInfo) {
	defer func() {
		if r := recover(); r != nil {
			logger.LogError(fmt.Errorf("Task %s panicked: %v", taskInfo.Request.TaskID, r), "executor.manager.executeTaskAsync", 0, "", "", "", map[string]interface{}{
				"operation": "execute_task",
				"option":    "panic_recovery",
				"func_name": "executor.manager.executeTaskAsync",
			})

			m.updateTaskStatus(taskInfo.Request.TaskID, ScanTaskStatusFailed, fmt.Sprintf("Task panicked: %v", r))
		}
	}()

	// 更新状态为运行中
	m.updateTaskStatus(taskInfo.Request.TaskID, ScanTaskStatusRunning, "Task started")

	// 执行扫描
	result, err := taskInfo.Executor.Execute(taskInfo.CancelCtx, taskInfo.Request)

	m.mutex.Lock()
	taskInfo.Result = result
	m.mutex.Unlock()

	if err != nil {
		logger.LogError(fmt.Errorf("Task %s failed: %v", taskInfo.Request.TaskID, err), "executor.manager.executeTaskAsync", 0, "", "", "", map[string]interface{}{
			"operation": "execute_task",
			"option":    "execution_failed",
			"func_name": "executor.manager.executeTaskAsync",
		})

		m.updateTaskStatus(taskInfo.Request.TaskID, ScanTaskStatusFailed, fmt.Sprintf("Execution failed: %v", err))
		return
	}

	// 检查上下文是否被取消
	select {
	case <-taskInfo.CancelCtx.Done():
		if taskInfo.CancelCtx.Err() == context.DeadlineExceeded {
			m.updateTaskStatus(taskInfo.Request.TaskID, ScanTaskStatusTimeout, "Task timeout")
		} else {
			m.updateTaskStatus(taskInfo.Request.TaskID, ScanTaskStatusCancelled, "Task cancelled")
		}
		return
	default:
	}

	// 任务完成
	m.updateTaskStatus(taskInfo.Request.TaskID, ScanTaskStatusCompleted, "Task completed successfully")

	logger.LogInfo(fmt.Sprintf("Task %s completed successfully", taskInfo.Request.TaskID), "executor.manager.executeTaskAsync", 0, "", "", "", map[string]interface{}{
		"operation": "execute_task",
		"option":    "execution_completed",
		"func_name": "executor.manager.executeTaskAsync",
	})
}

// updateTaskStatus 更新任务状态
func (m *DefaultExecutorManager) updateTaskStatus(taskID string, status ScanTaskStatus, message string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	taskInfo, exists := m.tasks[taskID]
	if !exists {
		return
	}

	taskInfo.Status.Status = status
	taskInfo.Status.Message = message
	taskInfo.Status.ElapsedTime = time.Since(taskInfo.Status.StartTime)
}

// StopTask 停止扫描任务
func (m *DefaultExecutorManager) StopTask(ctx context.Context, taskID string) error {
	m.mutex.RLock()
	taskInfo, exists := m.tasks[taskID]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// 取消任务上下文
	taskInfo.Cancel()

	// 调用执行器的停止方法
	if err := taskInfo.Executor.Stop(ctx, taskID); err != nil {
		logger.LogWarn(fmt.Sprintf("Failed to stop task %s via executor: %v", taskID, err), "executor.manager.StopTask", 0, "", "", "", map[string]interface{}{
			"operation": "stop_task",
			"option":    "executor_stop_failed",
			"func_name": "executor.manager.StopTask",
		})
	}

	m.updateTaskStatus(taskID, ScanTaskStatusCancelled, "Task stopped by user")

	logger.LogInfo(fmt.Sprintf("Task %s stopped", taskID), "executor.manager.StopTask", 0, "", "", "", map[string]interface{}{
		"operation": "stop_task",
		"option":    "task_stopped",
		"func_name": "executor.manager.StopTask",
	})

	return nil
}

// GetTaskStatus 获取任务状态
func (m *DefaultExecutorManager) GetTaskStatus(ctx context.Context, taskID string) (*ScanStatus, error) {
	m.mutex.RLock()
	taskInfo, exists := m.tasks[taskID]
	m.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	// 尝试从执行器获取更详细的状态
	if executorStatus, err := taskInfo.Executor.GetStatus(ctx, taskID); err == nil {
		return executorStatus, nil
	}

	// 返回本地状态
	return taskInfo.Status, nil
}

// Shutdown 关闭管理器
func (m *DefaultExecutorManager) Shutdown() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 停止所有正在运行的任务
	for taskID, taskInfo := range m.tasks {
		if taskInfo.Status.Status == ScanTaskStatusRunning {
			taskInfo.Cancel()
			logger.LogInfo(fmt.Sprintf("Cancelled running task %s during shutdown", taskID), "executor.manager.Shutdown", 0, "", "", "", map[string]interface{}{
				"operation": "shutdown",
				"option":    "cancel_running_task",
				"func_name": "executor.manager.Shutdown",
			})
		}
	}

	// 清理所有执行器
	for _, executor := range m.executors {
		if err := executor.Cleanup(); err != nil {
			logger.LogWarn(fmt.Sprintf("Failed to cleanup executor %s: %v", executor.GetName(), err), "executor.manager.Shutdown", 0, "", "", "", map[string]interface{}{
				"operation": "shutdown",
				"option":    "executor_cleanup_failed",
				"func_name": "executor.manager.Shutdown",
			})
		}
	}

	// 清空映射
	m.executors = make(map[string]ScanExecutor)
	m.tasks = make(map[string]*TaskInfo)

	logger.LogInfo("Executor manager shutdown completed", "executor.manager.Shutdown", 0, "", "", "", map[string]interface{}{
		"operation": "shutdown",
		"option":    "shutdown_completed",
		"func_name": "executor.manager.Shutdown",
	})

	return nil
}
