/**
 * 执行器管理器
 * @author: sun977
 * @date: 2025.10.21
 * @description: 统一管理所有类型的扫描工具执行器，提供执行器的注册、获取、调度等功能
 * @func: 占位符实现，待后续完善
 */
package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"neoagent/internal/executor/base"
	"neoagent/internal/executor/masscan"
	"neoagent/internal/executor/nmap"
	"neoagent/internal/executor/nuclei"
	"neoagent/internal/executor/system"
)

// ExecutorManager 执行器管理器
type ExecutorManager struct {
	executors      map[base.ExecutorType]base.Executor
	executorsMutex sync.RWMutex

	// 任务调度
	taskQueue     chan *ScheduledTask
	taskScheduler *TaskScheduler

	// 状态管理
	isRunning bool
	startTime time.Time

	// 配置
	config *ManagerConfig

	// 指标
	metrics      *ManagerMetrics
	metricsMutex sync.RWMutex
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	MaxConcurrentTasks  int                                        `json:"max_concurrent_tasks"`
	TaskTimeout         time.Duration                              `json:"task_timeout"`
	HealthCheckInterval time.Duration                              `json:"health_check_interval"`
	MetricsInterval     time.Duration                              `json:"metrics_interval"`
	ExecutorConfigs     map[base.ExecutorType]*base.ExecutorConfig `json:"executor_configs"`
	EnabledExecutors    []base.ExecutorType                        `json:"enabled_executors"`
	ResourceLimits      *ResourceLimits                            `json:"resource_limits"`
}

// ResourceLimits 资源限制
type ResourceLimits struct {
	MaxCPUUsage     float64 `json:"max_cpu_usage"`
	MaxMemoryUsage  int64   `json:"max_memory_usage"`
	MaxDiskUsage    int64   `json:"max_disk_usage"`
	MaxNetworkUsage int64   `json:"max_network_usage"`
}

// ScheduledTask 调度任务
type ScheduledTask struct {
	Task         *base.Task
	ExecutorType base.ExecutorType
	Priority     int
	ScheduledAt  time.Time
	Deadline     time.Time
	Retries      int
	MaxRetries   int
}

// TaskScheduler 任务调度器
type TaskScheduler struct {
	manager    *ExecutorManager
	taskQueue  chan *ScheduledTask
	workerPool chan struct{}
	isRunning  bool

	// 调度策略
	strategy SchedulingStrategy

	// 统计
	totalTasks     int64
	completedTasks int64
	failedTasks    int64
}

// SchedulingStrategy 调度策略
type SchedulingStrategy string

const (
	StrategyFIFO        SchedulingStrategy = "fifo"         // 先进先出
	StrategyPriority    SchedulingStrategy = "priority"     // 优先级调度
	StrategyRoundRobin  SchedulingStrategy = "round_robin"  // 轮询调度
	StrategyLoadBalance SchedulingStrategy = "load_balance" // 负载均衡
)

// ManagerMetrics 管理器指标
type ManagerMetrics struct {
	// 执行器统计
	TotalExecutors   int                                         `json:"total_executors"`
	RunningExecutors int                                         `json:"running_executors"`
	HealthyExecutors int                                         `json:"healthy_executors"`
	ExecutorMetrics  map[base.ExecutorType]*base.ExecutorMetrics `json:"executor_metrics"`

	// 任务统计
	TotalTasks      int64         `json:"total_tasks"`
	CompletedTasks  int64         `json:"completed_tasks"`
	FailedTasks     int64         `json:"failed_tasks"`
	RunningTasks    int64         `json:"running_tasks"`
	QueuedTasks     int64         `json:"queued_tasks"`
	AverageTaskTime time.Duration `json:"average_task_time"`
	TaskThroughput  float64       `json:"task_throughput"`

	// 资源统计
	TotalCPUUsage     float64 `json:"total_cpu_usage"`
	TotalMemoryUsage  int64   `json:"total_memory_usage"`
	TotalDiskUsage    int64   `json:"total_disk_usage"`
	TotalNetworkUsage int64   `json:"total_network_usage"`

	// 时间戳
	Timestamp time.Time     `json:"timestamp"`
	Uptime    time.Duration `json:"uptime"`
}

// ExecutorInfo 执行器信息
type ExecutorInfo struct {
	Type         base.ExecutorType     `json:"type"`
	Name         string                `json:"name"`
	Version      string                `json:"version"`
	Description  string                `json:"description"`
	Capabilities []string              `json:"capabilities"`
	Status       *base.ExecutorStatus  `json:"status"`
	Health       *base.HealthStatus    `json:"health"`
	Metrics      *base.ExecutorMetrics `json:"metrics"`
	Config       *base.ExecutorConfig  `json:"config"`
}

// NewExecutorManager 创建执行器管理器
func NewExecutorManager(config *ManagerConfig) *ExecutorManager {
	if config == nil {
		config = &ManagerConfig{
			MaxConcurrentTasks:  10,
			TaskTimeout:         30 * time.Minute,
			HealthCheckInterval: 30 * time.Second,
			MetricsInterval:     10 * time.Second,
			EnabledExecutors: []base.ExecutorType{
				base.ExecutorTypeSystem,
				base.ExecutorTypeNmap,
				base.ExecutorTypeNuclei,
				base.ExecutorTypeMasscan,
			},
			ResourceLimits: &ResourceLimits{
				MaxCPUUsage:     80.0,
				MaxMemoryUsage:  2 * 1024 * 1024 * 1024,  // 2GB
				MaxDiskUsage:    10 * 1024 * 1024 * 1024, // 10GB
				MaxNetworkUsage: 100 * 1024 * 1024,       // 100MB/s
			},
		}
	}

	manager := &ExecutorManager{
		executors: make(map[base.ExecutorType]base.Executor),
		taskQueue: make(chan *ScheduledTask, 1000),
		config:    config,
		metrics: &ManagerMetrics{
			ExecutorMetrics: make(map[base.ExecutorType]*base.ExecutorMetrics),
			Timestamp:       time.Now(),
		},
	}

	// 创建任务调度器
	manager.taskScheduler = &TaskScheduler{
		manager:    manager,
		taskQueue:  manager.taskQueue,
		workerPool: make(chan struct{}, config.MaxConcurrentTasks),
		strategy:   StrategyPriority,
	}

	return manager
}

// ==================== 生命周期管理 ====================

// Initialize 初始化管理器
func (m *ExecutorManager) Initialize() error {
	// TODO: 实现管理器初始化逻辑
	// 1. 注册所有启用的执行器
	// 2. 初始化各个执行器
	// 3. 启动健康检查
	// 4. 启动指标收集

	// 注册执行器
	if err := m.registerExecutors(); err != nil {
		return fmt.Errorf("register executors: %w", err)
	}

	// 初始化执行器
	if err := m.initializeExecutors(); err != nil {
		return fmt.Errorf("initialize executors: %w", err)
	}

	return nil
}

// Start 启动管理器
func (m *ExecutorManager) Start() error {
	if m.isRunning {
		return fmt.Errorf("manager is already running")
	}

	// TODO: 实现管理器启动逻辑
	// 1. 启动所有执行器
	// 2. 启动任务调度器
	// 3. 启动健康检查goroutine
	// 4. 启动指标收集goroutine

	// 启动所有执行器
	if err := m.startExecutors(); err != nil {
		return fmt.Errorf("start executors: %w", err)
	}

	// 启动任务调度器
	if err := m.taskScheduler.Start(); err != nil {
		return fmt.Errorf("start task scheduler: %w", err)
	}

	m.isRunning = true
	m.startTime = time.Now()

	// 启动后台goroutines
	go m.healthCheckLoop()
	go m.metricsCollectionLoop()

	return nil
}

// Stop 停止管理器
func (m *ExecutorManager) Stop() error {
	if !m.isRunning {
		return fmt.Errorf("manager is not running")
	}

	// TODO: 实现管理器停止逻辑
	// 1. 停止接收新任务
	// 2. 等待当前任务完成
	// 3. 停止任务调度器
	// 4. 停止所有执行器

	m.isRunning = false

	// 停止任务调度器
	if err := m.taskScheduler.Stop(); err != nil {
		return fmt.Errorf("stop task scheduler: %w", err)
	}

	// 停止所有执行器
	if err := m.stopExecutors(); err != nil {
		return fmt.Errorf("stop executors: %w", err)
	}

	// 关闭任务队列
	close(m.taskQueue)

	return nil
}

// Restart 重启管理器
func (m *ExecutorManager) Restart() error {
	if err := m.Stop(); err != nil {
		return fmt.Errorf("stop manager: %w", err)
	}

	time.Sleep(2 * time.Second)

	if err := m.Start(); err != nil {
		return fmt.Errorf("start manager: %w", err)
	}

	return nil
}

// IsRunning 检查管理器是否运行中
func (m *ExecutorManager) IsRunning() bool {
	return m.isRunning
}

// ==================== 执行器管理 ====================

// RegisterExecutor 注册执行器
func (m *ExecutorManager) RegisterExecutor(executorType base.ExecutorType, executor base.Executor) error {
	m.executorsMutex.Lock()
	defer m.executorsMutex.Unlock()

	if _, exists := m.executors[executorType]; exists {
		return fmt.Errorf("executor type %s already registered", executorType)
	}

	m.executors[executorType] = executor
	return nil
}

// UnregisterExecutor 注销执行器
func (m *ExecutorManager) UnregisterExecutor(executorType base.ExecutorType) error {
	m.executorsMutex.Lock()
	defer m.executorsMutex.Unlock()

	executor, exists := m.executors[executorType]
	if !exists {
		return fmt.Errorf("executor type %s not found", executorType)
	}

	// 停止执行器
	if executor.IsRunning() {
		if err := executor.Stop(); err != nil {
			return fmt.Errorf("stop executor: %w", err)
		}
	}

	delete(m.executors, executorType)
	return nil
}

// GetExecutor 获取执行器
func (m *ExecutorManager) GetExecutor(executorType base.ExecutorType) (base.Executor, error) {
	m.executorsMutex.RLock()
	defer m.executorsMutex.RUnlock()

	executor, exists := m.executors[executorType]
	if !exists {
		return nil, fmt.Errorf("executor type %s not found", executorType)
	}

	return executor, nil
}

// GetExecutors 获取所有执行器
func (m *ExecutorManager) GetExecutors() map[base.ExecutorType]base.Executor {
	m.executorsMutex.RLock()
	defer m.executorsMutex.RUnlock()

	executors := make(map[base.ExecutorType]base.Executor)
	for executorType, executor := range m.executors {
		executors[executorType] = executor
	}

	return executors
}

// GetExecutorInfo 获取执行器信息
func (m *ExecutorManager) GetExecutorInfo(executorType base.ExecutorType) (*ExecutorInfo, error) {
	executor, err := m.GetExecutor(executorType)
	if err != nil {
		return nil, err
	}

	info := &ExecutorInfo{
		Type:         executor.GetType(),
		Name:         executor.GetName(),
		Version:      executor.GetVersion(),
		Description:  executor.GetDescription(),
		Capabilities: executor.GetCapabilities(),
		Status:       executor.GetStatus(),
		Health:       executor.HealthCheck(),
		Metrics:      executor.GetMetrics(),
		Config:       executor.GetConfig(),
	}

	return info, nil
}

// GetAllExecutorInfo 获取所有执行器信息
func (m *ExecutorManager) GetAllExecutorInfo() ([]*ExecutorInfo, error) {
	m.executorsMutex.RLock()
	defer m.executorsMutex.RUnlock()

	infos := make([]*ExecutorInfo, 0, len(m.executors))

	for _, executor := range m.executors {
		info := &ExecutorInfo{
			Type:         executor.GetType(),
			Name:         executor.GetName(),
			Version:      executor.GetVersion(),
			Description:  executor.GetDescription(),
			Capabilities: executor.GetCapabilities(),
			Status:       executor.GetStatus(),
			Health:       executor.HealthCheck(),
			Metrics:      executor.GetMetrics(),
			Config:       executor.GetConfig(),
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// ==================== 任务管理 ====================

// SubmitTask 提交任务
func (m *ExecutorManager) SubmitTask(task *base.Task, executorType base.ExecutorType) error {
	if !m.isRunning {
		return fmt.Errorf("manager is not running")
	}

	// 验证执行器类型
	if _, err := m.GetExecutor(executorType); err != nil {
		return fmt.Errorf("invalid executor type: %w", err)
	}

	// 创建调度任务
	scheduledTask := &ScheduledTask{
		Task:         task,
		ExecutorType: executorType,
		Priority:     m.calculateTaskPriority(task),
		ScheduledAt:  time.Now(),
		Deadline:     time.Now().Add(m.config.TaskTimeout),
		Retries:      0,
		MaxRetries:   3,
	}

	// 提交到任务队列
	select {
	case m.taskQueue <- scheduledTask:
		m.updateTaskMetrics(1, 0, 0, 0, 1)
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// ExecuteTask 执行任务
func (m *ExecutorManager) ExecuteTask(ctx context.Context, task *base.Task, executorType base.ExecutorType) (*base.TaskResult, error) {
	executor, err := m.GetExecutor(executorType)
	if err != nil {
		return nil, fmt.Errorf("get executor: %w", err)
	}

	// 检查执行器健康状态
	health := executor.HealthCheck()
	if !health.IsHealthy {
		return nil, fmt.Errorf("executor %s is not healthy: %s", executorType, health.Status)
	}

	// 执行任务
	startTime := time.Now()
	result, err := executor.Execute(ctx, task)
	duration := time.Since(startTime)

	// 更新指标
	if err != nil {
		m.updateTaskMetrics(0, 0, 1, -1, -1)
	} else {
		m.updateTaskMetrics(0, 1, 0, -1, -1)
	}

	// 更新平均任务时间
	m.updateAverageTaskTime(duration)

	return result, err
}

// CancelTask 取消任务
func (m *ExecutorManager) CancelTask(taskID string, executorType base.ExecutorType) error {
	executor, err := m.GetExecutor(executorType)
	if err != nil {
		return fmt.Errorf("get executor: %w", err)
	}

	return executor.Cancel(taskID)
}

// GetTaskStatus 获取任务状态
func (m *ExecutorManager) GetTaskStatus(taskID string, executorType base.ExecutorType) (base.TaskStatus, error) {
	executor, err := m.GetExecutor(executorType)
	if err != nil {
		return "", fmt.Errorf("get executor: %w", err)
	}

	return executor.GetTaskStatus(taskID)
}

// GetTaskResult 获取任务结果
func (m *ExecutorManager) GetTaskResult(taskID string, executorType base.ExecutorType) (*base.TaskResult, error) {
	executor, err := m.GetExecutor(executorType)
	if err != nil {
		return nil, fmt.Errorf("get executor: %w", err)
	}

	return executor.GetTaskResult(taskID)
}

// ==================== 配置管理 ====================

// UpdateConfig 更新配置
func (m *ExecutorManager) UpdateConfig(config *ManagerConfig) error {
	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	m.config = config

	// 更新执行器配置
	for executorType, executorConfig := range config.ExecutorConfigs {
		if executor, err := m.GetExecutor(executorType); err == nil {
			if err := executor.UpdateConfig(executorConfig); err != nil {
				return fmt.Errorf("update executor %s config: %w", executorType, err)
			}
		}
	}

	return nil
}

// GetConfig 获取当前配置
func (m *ExecutorManager) GetConfig() *ManagerConfig {
	return m.config
}

// ==================== 健康检查 ====================

// HealthCheck 健康检查
func (m *ExecutorManager) HealthCheck() map[base.ExecutorType]*base.HealthStatus {
	m.executorsMutex.RLock()
	defer m.executorsMutex.RUnlock()

	healthStatus := make(map[base.ExecutorType]*base.HealthStatus)

	for executorType, executor := range m.executors {
		healthStatus[executorType] = executor.HealthCheck()
	}

	return healthStatus
}

// GetMetrics 获取管理器指标
func (m *ExecutorManager) GetMetrics() *ManagerMetrics {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()

	// 更新执行器指标
	m.executorsMutex.RLock()
	for executorType, executor := range m.executors {
		m.metrics.ExecutorMetrics[executorType] = executor.GetMetrics()
	}
	m.executorsMutex.RUnlock()

	// 更新基本统计
	m.metrics.TotalExecutors = len(m.executors)
	m.metrics.Timestamp = time.Now()
	if m.isRunning {
		m.metrics.Uptime = time.Since(m.startTime)
	}

	return m.metrics
}

// ==================== 内部方法 ====================

// registerExecutors 注册执行器
func (m *ExecutorManager) registerExecutors() error {
	// TODO: 根据配置注册启用的执行器

	for _, executorType := range m.config.EnabledExecutors {
		var executor base.Executor

		switch executorType {
		case base.ExecutorTypeSystem:
			executor = system.NewSystemExecutor()
		case base.ExecutorTypeNmap:
			executor = nmap.NewNmapExecutor()
		case base.ExecutorTypeNuclei:
			executor = nuclei.NewNucleiExecutor()
		case base.ExecutorTypeMasscan:
			executor = masscan.NewMasscanExecutor()
		default:
			return fmt.Errorf("unsupported executor type: %s", executorType)
		}

		if err := m.RegisterExecutor(executorType, executor); err != nil {
			return fmt.Errorf("register executor %s: %w", executorType, err)
		}
	}

	return nil
}

// initializeExecutors 初始化执行器
func (m *ExecutorManager) initializeExecutors() error {
	m.executorsMutex.RLock()
	defer m.executorsMutex.RUnlock()

	for executorType, executor := range m.executors {
		config := m.config.ExecutorConfigs[executorType]
		if config == nil {
			// 使用默认配置
			config = &base.ExecutorConfig{
				Type:           executorType,
				MaxConcurrency: 5,
				TaskTimeout:    30 * time.Minute,
				Custom:         make(map[string]interface{}),
			}
		}

		if err := executor.Initialize(config); err != nil {
			return fmt.Errorf("initialize executor %s: %w", executorType, err)
		}
	}

	return nil
}

// startExecutors 启动执行器
func (m *ExecutorManager) startExecutors() error {
	m.executorsMutex.RLock()
	defer m.executorsMutex.RUnlock()

	for executorType, executor := range m.executors {
		if err := executor.Start(); err != nil {
			return fmt.Errorf("start executor %s: %w", executorType, err)
		}
	}

	return nil
}

// stopExecutors 停止执行器
func (m *ExecutorManager) stopExecutors() error {
	m.executorsMutex.RLock()
	defer m.executorsMutex.RUnlock()

	for executorType, executor := range m.executors {
		if err := executor.Stop(); err != nil {
			return fmt.Errorf("stop executor %s: %w", executorType, err)
		}
	}

	return nil
}

// calculateTaskPriority 计算任务优先级
func (m *ExecutorManager) calculateTaskPriority(task *base.Task) int {
	// TODO: 实现任务优先级计算逻辑
	// 1. 根据任务类型设置基础优先级
	// 2. 根据任务紧急程度调整优先级
	// 3. 根据资源使用情况调整优先级

	priority := 5 // 默认优先级

	// 根据任务类型调整优先级
	if taskType, ok := task.Config["type"].(string); ok {
		switch taskType {
		case "urgent":
			priority = 10
		case "high":
			priority = 8
		case "normal":
			priority = 5
		case "low":
			priority = 2
		}
	}

	return priority
}

// updateTaskMetrics 更新任务指标
func (m *ExecutorManager) updateTaskMetrics(total, completed, failed, running, queued int64) {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()

	m.metrics.TotalTasks += total
	m.metrics.CompletedTasks += completed
	m.metrics.FailedTasks += failed
	m.metrics.RunningTasks += running
	m.metrics.QueuedTasks += queued

	// 计算吞吐量
	if m.isRunning {
		uptime := time.Since(m.startTime)
		if uptime > 0 {
			m.metrics.TaskThroughput = float64(m.metrics.CompletedTasks) / uptime.Seconds()
		}
	}
}

// updateAverageTaskTime 更新平均任务时间
func (m *ExecutorManager) updateAverageTaskTime(duration time.Duration) {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()

	if m.metrics.CompletedTasks > 0 {
		totalTime := m.metrics.AverageTaskTime * time.Duration(m.metrics.CompletedTasks-1)
		m.metrics.AverageTaskTime = (totalTime + duration) / time.Duration(m.metrics.CompletedTasks)
	} else {
		m.metrics.AverageTaskTime = duration
	}
}

// validateConfig 验证配置
func (m *ExecutorManager) validateConfig(config *ManagerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.MaxConcurrentTasks <= 0 {
		return fmt.Errorf("max_concurrent_tasks must be greater than 0")
	}

	if config.TaskTimeout <= 0 {
		return fmt.Errorf("task_timeout must be greater than 0")
	}

	if len(config.EnabledExecutors) == 0 {
		return fmt.Errorf("at least one executor must be enabled")
	}

	return nil
}

// healthCheckLoop 健康检查循环
func (m *ExecutorManager) healthCheckLoop() {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !m.isRunning {
			return
		}

		// 执行健康检查
		healthStatus := m.HealthCheck()

		// 统计健康的执行器数量
		healthyCount := 0
		runningCount := 0

		for _, health := range healthStatus {
			if health.IsHealthy {
				healthyCount++
			}
		}

		m.executorsMutex.RLock()
		for _, executor := range m.executors {
			if executor.IsRunning() {
				runningCount++
			}
		}
		m.executorsMutex.RUnlock()

		// 更新指标
		m.metricsMutex.Lock()
		m.metrics.HealthyExecutors = healthyCount
		m.metrics.RunningExecutors = runningCount
		m.metricsMutex.Unlock()
	}
}

// metricsCollectionLoop 指标收集循环
func (m *ExecutorManager) metricsCollectionLoop() {
	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !m.isRunning {
			return
		}

		// 收集资源使用情况
		m.collectResourceMetrics()
	}
}

// collectResourceMetrics 收集资源指标
func (m *ExecutorManager) collectResourceMetrics() {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()

	// TODO: 实现资源指标收集
	// 1. 收集所有执行器的资源使用情况
	// 2. 汇总系统资源使用情况
	// 3. 更新指标

	var totalCPU float64
	var totalMemory int64
	var totalDisk int64
	var totalNetwork int64

	m.executorsMutex.RLock()
	for _, executor := range m.executors {
		usage := executor.GetResourceUsage()
		if usage != nil {
			totalCPU += usage.CPUUsage
			totalMemory += usage.MemoryUsage
			totalDisk += usage.DiskUsage
			totalNetwork += usage.NetworkUsage
		}
	}
	m.executorsMutex.RUnlock()

	m.metrics.TotalCPUUsage = totalCPU
	m.metrics.TotalMemoryUsage = totalMemory
	m.metrics.TotalDiskUsage = totalDisk
	m.metrics.TotalNetworkUsage = totalNetwork
}

// ==================== 任务调度器实现 ====================

// Start 启动任务调度器
func (ts *TaskScheduler) Start() error {
	if ts.isRunning {
		return fmt.Errorf("task scheduler is already running")
	}

	ts.isRunning = true

	// 启动工作协程池
	for i := 0; i < cap(ts.workerPool); i++ {
		go ts.worker()
	}

	// 启动调度协程
	go ts.scheduler()

	return nil
}

// Stop 停止任务调度器
func (ts *TaskScheduler) Stop() error {
	if !ts.isRunning {
		return fmt.Errorf("task scheduler is not running")
	}

	ts.isRunning = false

	// 等待所有工作协程完成
	for i := 0; i < cap(ts.workerPool); i++ {
		ts.workerPool <- struct{}{}
	}

	return nil
}

// scheduler 调度器主循环
func (ts *TaskScheduler) scheduler() {
	for ts.isRunning {
		scheduledTask := <-ts.taskQueue
		// 获取工作协程
		ts.workerPool <- struct{}{}

		// 分配任务给工作协程
		go func(task *ScheduledTask) {
			defer func() { <-ts.workerPool }()
			ts.executeScheduledTask(task)
		}(scheduledTask)
	}
}

// worker 工作协程
func (ts *TaskScheduler) worker() {
	// 工作协程在scheduler中动态创建和管理
}

// executeScheduledTask 执行调度任务
func (ts *TaskScheduler) executeScheduledTask(scheduledTask *ScheduledTask) {
	ctx := context.Background()
	if scheduledTask.Task.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, scheduledTask.Task.Timeout)
		defer cancel()
	}

	// 执行任务
	result, err := ts.manager.ExecuteTask(ctx, scheduledTask.Task, scheduledTask.ExecutorType)

	// 处理执行结果
	if err != nil {
		// 任务失败，检查是否需要重试
		if scheduledTask.Retries < scheduledTask.MaxRetries {
			scheduledTask.Retries++
			scheduledTask.ScheduledAt = time.Now()

			// 重新提交任务
			select {
			case ts.taskQueue <- scheduledTask:
				// 重新提交成功
			default:
				// 队列满，放弃重试
				ts.failedTasks++
			}
		} else {
			// 超过最大重试次数，任务失败
			ts.failedTasks++
		}
	} else {
		// 任务成功
		ts.completedTasks++

		// TODO: 处理任务结果
		_ = result
	}

	ts.totalTasks++
}
