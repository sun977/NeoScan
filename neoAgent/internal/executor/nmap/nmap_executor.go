/**
 * Nmap扫描工具执行器
 * @author: sun977
 * @date: 2025.10.21
 * @description: Nmap扫描工具执行器实现，支持端口扫描、服务识别、OS检测等
 * @func: 占位符实现，待后续完善
 */
package nmap

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"neoagent/internal/executor/base"
)

// NmapExecutor Nmap执行器
type NmapExecutor struct {
	config     *base.ExecutorConfig
	status     *base.ExecutorStatus
	tasks      map[string]*base.Task
	tasksMutex sync.RWMutex
	running    bool
	startTime  time.Time
	metrics    *base.ExecutorMetrics

	// Nmap特定配置
	nmapPath    string
	defaultArgs []string
	maxTargets  int

	// 任务管理
	taskQueue   chan *base.Task
	cancelFuncs map[string]context.CancelFunc
	cancelMutex sync.RWMutex
}

// NmapConfig Nmap配置
type NmapConfig struct {
	NmapPath     string            `json:"nmap_path"`     // nmap可执行文件路径
	DefaultArgs  []string          `json:"default_args"`  // 默认参数
	MaxTargets   int               `json:"max_targets"`   // 最大目标数量
	ScanProfiles map[string]string `json:"scan_profiles"` // 扫描配置文件
}

// NmapScanResult Nmap扫描结果
type NmapScanResult struct {
	Host     string                 `json:"host"`
	Status   string                 `json:"status"`
	Ports    []NmapPort             `json:"ports"`
	OS       *NmapOS                `json:"os,omitempty"`
	Services []NmapService          `json:"services,omitempty"`
	Scripts  map[string]interface{} `json:"scripts,omitempty"`
}

// NmapPort 端口信息
type NmapPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service"`
	Version  string `json:"version,omitempty"`
}

// NmapOS 操作系统信息
type NmapOS struct {
	Name     string  `json:"name"`
	Accuracy float64 `json:"accuracy"`
	Family   string  `json:"family"`
}

// NmapService 服务信息
type NmapService struct {
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Name      string `json:"name"`
	Product   string `json:"product,omitempty"`
	Version   string `json:"version,omitempty"`
	ExtraInfo string `json:"extra_info,omitempty"`
}

// NewNmapExecutor 创建Nmap执行器实例
func NewNmapExecutor() base.Executor {
	return &NmapExecutor{
		tasks:       make(map[string]*base.Task),
		taskQueue:   make(chan *base.Task, 100),
		cancelFuncs: make(map[string]context.CancelFunc),
		nmapPath:    "nmap", // 默认路径
		defaultArgs: []string{"-sS", "-O", "-sV", "--version-intensity", "5"},
		maxTargets:  1000,
		metrics: &base.ExecutorMetrics{
			Timestamp: time.Now(),
		},
	}
}

// ==================== 基础信息实现 ====================

// GetType 获取执行器类型
func (e *NmapExecutor) GetType() base.ExecutorType {
	return base.ExecutorTypeNmap
}

// GetName 获取执行器名称
func (e *NmapExecutor) GetName() string {
	return "Nmap Executor"
}

// GetVersion 获取执行器版本
func (e *NmapExecutor) GetVersion() string {
	return "1.0.0"
}

// GetDescription 获取执行器描述
func (e *NmapExecutor) GetDescription() string {
	return "Nmap网络扫描工具执行器，支持端口扫描、服务识别、OS检测等功能"
}

// GetCapabilities 获取执行器能力列表
func (e *NmapExecutor) GetCapabilities() []string {
	return []string{
		"port_scan",          // 端口扫描
		"service_detection",  // 服务检测
		"os_detection",       // 操作系统检测
		"script_scan",        // 脚本扫描
		"stealth_scan",       // 隐蔽扫描
		"version_detection",  // 版本检测
		"vulnerability_scan", // 漏洞扫描
		"network_discovery",  // 网络发现
	}
}

// ==================== 生命周期管理实现 ====================

// Initialize 初始化执行器
func (e *NmapExecutor) Initialize(config *base.ExecutorConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// TODO: 实现Nmap执行器初始化逻辑
	// 1. 验证nmap可执行文件
	// 2. 检查nmap版本
	// 3. 验证权限（某些扫描需要root权限）
	// 4. 初始化扫描配置
	// 5. 设置资源限制

	e.config = config

	// 解析Nmap特定配置
	if nmapConfig, ok := config.Settings["nmap"]; ok {
		if err := e.parseNmapConfig(nmapConfig); err != nil {
			return fmt.Errorf("parse nmap config: %w", err)
		}
	}

	// 验证nmap可执行文件
	if err := e.validateNmapInstallation(); err != nil {
		return fmt.Errorf("validate nmap installation: %w", err)
	}

	e.status = &base.ExecutorStatus{
		Type:      base.ExecutorTypeNmap,
		Name:      e.GetName(),
		Status:    "initialized",
		IsRunning: false,
		Timestamp: time.Now(),
	}

	return nil
}

// Start 启动执行器
func (e *NmapExecutor) Start() error {
	if e.running {
		return fmt.Errorf("executor is already running")
	}

	// TODO: 实现Nmap执行器启动逻辑
	// 1. 启动任务处理goroutine
	// 2. 启动指标收集goroutine
	// 3. 启动健康检查goroutine
	// 4. 预热nmap进程池（如果需要）

	e.running = true
	e.startTime = time.Now()

	// 启动任务处理器
	go e.taskProcessor()

	// 更新状态
	e.status.Status = "running"
	e.status.IsRunning = true
	e.status.StartTime = e.startTime
	e.status.Timestamp = time.Now()

	return nil
}

// Stop 停止执行器
func (e *NmapExecutor) Stop() error {
	if !e.running {
		return fmt.Errorf("executor is not running")
	}

	// TODO: 实现Nmap执行器停止逻辑
	// 1. 停止接收新任务
	// 2. 等待当前扫描完成或强制终止
	// 3. 清理临时文件
	// 4. 释放资源

	e.running = false

	// 取消所有运行中的任务
	e.cancelMutex.Lock()
	for taskID, cancelFunc := range e.cancelFuncs {
		cancelFunc()
		delete(e.cancelFuncs, taskID)
	}
	e.cancelMutex.Unlock()

	// 关闭任务队列
	close(e.taskQueue)

	// 更新状态
	e.status.Status = "stopped"
	e.status.IsRunning = false
	e.status.Timestamp = time.Now()

	return nil
}

// Restart 重启执行器
func (e *NmapExecutor) Restart() error {
	if err := e.Stop(); err != nil {
		return fmt.Errorf("stop executor: %w", err)
	}

	time.Sleep(2 * time.Second) // 等待nmap进程完全退出

	if err := e.Start(); err != nil {
		return fmt.Errorf("start executor: %w", err)
	}

	return nil
}

// IsRunning 检查执行器是否运行中
func (e *NmapExecutor) IsRunning() bool {
	return e.running
}

// GetStatus 获取执行器状态
func (e *NmapExecutor) GetStatus() *base.ExecutorStatus {
	e.status.Timestamp = time.Now()
	if e.running {
		e.status.Uptime = time.Since(e.startTime)
	}

	// 更新任务统计
	e.tasksMutex.RLock()
	e.status.TaskCount = len(e.tasks)
	e.tasksMutex.RUnlock()

	return e.status
}

// ==================== 任务执行实现 ====================

// Execute 执行扫描任务
func (e *NmapExecutor) Execute(ctx context.Context, task *base.Task) (*base.TaskResult, error) {
	if !e.running {
		return nil, fmt.Errorf("executor is not running")
	}

	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	// TODO: 实现Nmap扫描任务执行逻辑
	// 1. 解析扫描参数
	// 2. 构建nmap命令
	// 3. 执行扫描
	// 4. 解析扫描结果
	// 5. 格式化输出

	// 设置任务状态
	task.Status = base.TaskStatusRunning
	task.StartTime = time.Now()
	task.UpdatedAt = time.Now()

	// 保存任务
	e.tasksMutex.Lock()
	e.tasks[task.ID] = task
	e.tasksMutex.Unlock()

	// 创建取消上下文
	taskCtx, cancel := context.WithCancel(ctx)
	e.cancelMutex.Lock()
	e.cancelFuncs[task.ID] = cancel
	e.cancelMutex.Unlock()

	// 执行扫描
	result := &base.TaskResult{
		TaskID:    task.ID,
		StartTime: task.StartTime,
		Timestamp: time.Now(),
	}

	// 执行nmap扫描
	if err := e.executeNmapScan(taskCtx, task, result); err != nil {
		result.Status = base.TaskStatusFailed
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Status = base.TaskStatusCompleted
		result.Success = true
	}

	// 更新任务状态
	task.Status = result.Status
	task.EndTime = time.Now()
	task.UpdatedAt = time.Now()
	result.EndTime = task.EndTime
	result.Duration = task.EndTime.Sub(task.StartTime)

	// 清理取消函数
	e.cancelMutex.Lock()
	delete(e.cancelFuncs, task.ID)
	e.cancelMutex.Unlock()

	// 更新指标
	e.updateMetrics(result)

	return result, nil
}

// executeNmapScan 执行nmap扫描
func (e *NmapExecutor) executeNmapScan(ctx context.Context, task *base.Task, result *base.TaskResult) error {
	// TODO: 实现具体的nmap扫描逻辑
	// 1. 解析目标和参数
	// 2. 构建nmap命令行
	// 3. 执行扫描
	// 4. 解析XML输出
	// 5. 转换为标准格式

	// 获取扫描目标
	targets, ok := task.Config["targets"].([]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid targets in task config")
	}

	// 获取扫描参数
	scanType := "default"
	if st, ok := task.Config["scan_type"].(string); ok {
		scanType = st
	}

	// 构建nmap命令
	args := e.buildNmapCommand(targets, scanType, task.Config)

	// 执行nmap命令
	cmd := exec.CommandContext(ctx, e.nmapPath, args...)

	// 设置输出格式为XML
	cmd.Args = append(cmd.Args, "-oX", "-") // 输出XML到stdout

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = err.Error()
		result.Logs = []string{string(output)}
		return err
	}

	// 解析nmap输出
	scanResults, err := e.parseNmapOutput(string(output))
	if err != nil {
		result.Error = fmt.Sprintf("parse nmap output: %v", err)
		result.Logs = []string{string(output)}
		return err
	}

	// 转换为标准格式
	standardResults := e.convertToStandardResults(scanResults)

	// 设置结果
	result.Output = &base.TaskOutput{
		Results: standardResults,
		Summary: &base.ScanSummary{
			TotalTargets:   len(targets),
			ScannedTargets: len(scanResults),
			Duration:       result.Duration,
			StartTime:      result.StartTime,
			EndTime:        result.EndTime,
		},
	}

	result.Logs = []string{string(output)}

	return nil
}

// buildNmapCommand 构建nmap命令
func (e *NmapExecutor) buildNmapCommand(targets []interface{}, scanType string, config map[string]interface{}) []string {
	args := make([]string, 0)

	// 添加默认参数
	args = append(args, e.defaultArgs...)

	// 根据扫描类型添加参数
	switch scanType {
	case "tcp_syn":
		args = append(args, "-sS")
	case "tcp_connect":
		args = append(args, "-sT")
	case "udp":
		args = append(args, "-sU")
	case "ping":
		args = append(args, "-sn")
	case "version":
		args = append(args, "-sV")
	case "os_detection":
		args = append(args, "-O")
	case "aggressive":
		args = append(args, "-A")
	case "stealth":
		args = append(args, "-sS", "-f", "-D", "RND:10")
	}

	// 添加端口范围
	if ports, ok := config["ports"].(string); ok {
		args = append(args, "-p", ports)
	}

	// 添加时间控制
	if timing, ok := config["timing"].(string); ok {
		args = append(args, "-T"+timing)
	}

	// 添加脚本扫描
	if scripts, ok := config["scripts"].(string); ok {
		args = append(args, "--script", scripts)
	}

	// 添加目标
	for _, target := range targets {
		if targetStr, ok := target.(string); ok {
			args = append(args, targetStr)
		}
	}

	return args
}

// parseNmapOutput 解析nmap输出
func (e *NmapExecutor) parseNmapOutput(output string) ([]NmapScanResult, error) {
	// TODO: 实现XML解析逻辑
	// 1. 解析XML格式的nmap输出
	// 2. 提取主机信息
	// 3. 提取端口信息
	// 4. 提取服务信息
	// 5. 提取OS信息

	// 占位符实现 - 简单的文本解析
	results := make([]NmapScanResult, 0)

	lines := strings.Split(output, "\n")
	var currentHost *NmapScanResult

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析主机信息
		if strings.Contains(line, "Nmap scan report for") {
			if currentHost != nil {
				results = append(results, *currentHost)
			}

			// 提取主机地址
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				host := parts[4]
				currentHost = &NmapScanResult{
					Host:   host,
					Status: "up",
					Ports:  make([]NmapPort, 0),
				}
			}
		}

		// 解析端口信息
		if currentHost != nil && strings.Contains(line, "/tcp") {
			// 简单的端口解析
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				portInfo := parts[0]
				state := parts[1]
				service := parts[2]

				// 提取端口号
				portParts := strings.Split(portInfo, "/")
				if len(portParts) >= 2 {
					port := portParts[0]
					protocol := portParts[1]

					currentHost.Ports = append(currentHost.Ports, NmapPort{
						Port:     0, // TODO: 转换为int
						Protocol: protocol,
						State:    state,
						Service:  service,
					})
				}
			}
		}
	}

	// 添加最后一个主机
	if currentHost != nil {
		results = append(results, *currentHost)
	}

	return results, nil
}

// convertToStandardResults 转换为标准结果格式
func (e *NmapExecutor) convertToStandardResults(nmapResults []NmapScanResult) []base.ScanResult {
	results := make([]base.ScanResult, 0, len(nmapResults))

	for _, nmapResult := range nmapResults {
		// 统计开放端口
		openPorts := 0
		for _, port := range nmapResult.Ports {
			if port.State == "open" {
				openPorts++
			}
		}

		// 判断是否有漏洞（占位符逻辑）
		hasVulnerabilities := false
		vulnerabilities := make([]base.Vulnerability, 0)

		// 检查常见的不安全端口
		for _, port := range nmapResult.Ports {
			if port.State == "open" {
				switch port.Port {
				case 21, 23, 53, 135, 139, 445, 1433, 3389:
					hasVulnerabilities = true
					vulnerabilities = append(vulnerabilities, base.Vulnerability{
						ID:          fmt.Sprintf("OPEN_PORT_%d", port.Port),
						Title:       fmt.Sprintf("Open %s port %d", port.Protocol, port.Port),
						Description: fmt.Sprintf("Port %d/%s is open and may pose security risks", port.Port, port.Protocol),
						Severity:    "medium",
						CVSS:        5.0,
						References:  []string{},
						Solution:    "Consider closing unnecessary ports or implementing proper access controls",
						Timestamp:   time.Now(),
					})
				}
			}
		}

		result := base.ScanResult{
			Target:             nmapResult.Host,
			Status:             nmapResult.Status,
			HasVulnerabilities: hasVulnerabilities,
			Vulnerabilities:    vulnerabilities,
			Extra: map[string]interface{}{
				"ports":      nmapResult.Ports,
				"open_ports": openPorts,
				"os":         nmapResult.OS,
				"services":   nmapResult.Services,
				"scripts":    nmapResult.Scripts,
			},
			Timestamp: time.Now(),
		}

		results = append(results, result)
	}

	return results
}

// Cancel 取消扫描任务
func (e *NmapExecutor) Cancel(taskID string) error {
	e.cancelMutex.Lock()
	defer e.cancelMutex.Unlock()

	cancelFunc, exists := e.cancelFuncs[taskID]
	if !exists {
		return fmt.Errorf("task %s not found or not running", taskID)
	}

	// 取消任务
	cancelFunc()

	// 更新任务状态
	e.tasksMutex.Lock()
	if task, exists := e.tasks[taskID]; exists {
		task.Status = base.TaskStatusCanceled
		task.EndTime = time.Now()
		task.UpdatedAt = time.Now()
	}
	e.tasksMutex.Unlock()

	return nil
}

// Pause 暂停扫描任务
func (e *NmapExecutor) Pause(taskID string) error {
	// TODO: 实现任务暂停逻辑
	// nmap可能不支持暂停，可以考虑终止后重新开始
	return fmt.Errorf("pause operation not supported by nmap executor")
}

// Resume 恢复扫描任务
func (e *NmapExecutor) Resume(taskID string) error {
	// TODO: 实现任务恢复逻辑
	// nmap可能不支持恢复，可以考虑重新开始扫描
	return fmt.Errorf("resume operation not supported by nmap executor")
}

// ==================== 任务管理实现 ====================

// GetTask 获取任务信息
func (e *NmapExecutor) GetTask(taskID string) (*base.Task, error) {
	e.tasksMutex.RLock()
	defer e.tasksMutex.RUnlock()

	task, exists := e.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return task, nil
}

// GetTasks 获取所有任务
func (e *NmapExecutor) GetTasks() ([]*base.Task, error) {
	e.tasksMutex.RLock()
	defer e.tasksMutex.RUnlock()

	tasks := make([]*base.Task, 0, len(e.tasks))
	for _, task := range e.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskStatus 获取任务状态
func (e *NmapExecutor) GetTaskStatus(taskID string) (base.TaskStatus, error) {
	task, err := e.GetTask(taskID)
	if err != nil {
		return "", err
	}

	return task.Status, nil
}

// GetTaskResult 获取任务结果
func (e *NmapExecutor) GetTaskResult(taskID string) (*base.TaskResult, error) {
	task, err := e.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	result := &base.TaskResult{
		TaskID:    task.ID,
		Status:    task.Status,
		Success:   task.Status == base.TaskStatusCompleted,
		Output:    task.Output,
		Error:     task.ErrorMessage,
		Logs:      task.Logs,
		StartTime: task.StartTime,
		EndTime:   task.EndTime,
		Duration:  task.EndTime.Sub(task.StartTime),
		Timestamp: time.Now(),
	}

	return result, nil
}

// GetTaskLogs 获取任务日志
func (e *NmapExecutor) GetTaskLogs(taskID string) ([]string, error) {
	task, err := e.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	return task.Logs, nil
}

// ==================== 配置管理实现 ====================

// UpdateConfig 更新配置
func (e *NmapExecutor) UpdateConfig(config *base.ExecutorConfig) error {
	if err := e.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	e.config = config
	e.config.UpdatedAt = time.Now()

	// 重新解析Nmap配置
	if nmapConfig, ok := config.Settings["nmap"]; ok {
		if err := e.parseNmapConfig(nmapConfig); err != nil {
			return fmt.Errorf("parse nmap config: %w", err)
		}
	}

	return nil
}

// GetConfig 获取当前配置
func (e *NmapExecutor) GetConfig() *base.ExecutorConfig {
	return e.config
}

// ValidateConfig 验证配置
func (e *NmapExecutor) ValidateConfig(config *base.ExecutorConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Type != base.ExecutorTypeNmap {
		return fmt.Errorf("invalid executor type: %s", config.Type)
	}

	if config.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be greater than 0")
	}

	if config.TaskTimeout <= 0 {
		return fmt.Errorf("task_timeout must be greater than 0")
	}

	return nil
}

// ==================== 健康检查实现 ====================

// HealthCheck 健康检查
func (e *NmapExecutor) HealthCheck() *base.HealthStatus {
	checks := make(map[string]base.CheckResult)
	isHealthy := true

	// 检查nmap可执行文件
	if err := e.validateNmapInstallation(); err != nil {
		checks["nmap_installation"] = base.CheckResult{
			Name:      "Nmap Installation",
			Status:    "error",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
		isHealthy = false
	} else {
		checks["nmap_installation"] = base.CheckResult{
			Name:      "Nmap Installation",
			Status:    "ok",
			Message:   "Nmap is properly installed",
			Timestamp: time.Now(),
		}
	}

	// 检查执行器状态
	checks["executor_status"] = base.CheckResult{
		Name:      "Executor Status",
		Status:    "ok",
		Message:   fmt.Sprintf("Executor is %s", e.status.Status),
		Timestamp: time.Now(),
	}

	// 检查任务队列
	queueSize := len(e.taskQueue)
	if queueSize > 80 {
		checks["task_queue"] = base.CheckResult{
			Name:      "Task Queue",
			Status:    "warning",
			Message:   fmt.Sprintf("Queue size: %d (high)", queueSize),
			Timestamp: time.Now(),
		}
	} else {
		checks["task_queue"] = base.CheckResult{
			Name:      "Task Queue",
			Status:    "ok",
			Message:   fmt.Sprintf("Queue size: %d", queueSize),
			Timestamp: time.Now(),
		}
	}

	status := "healthy"
	if !isHealthy {
		status = "unhealthy"
	}

	return &base.HealthStatus{
		IsHealthy: isHealthy,
		Status:    status,
		Checks:    checks,
		LastCheck: time.Now(),
	}
}

// GetMetrics 获取执行器指标
func (e *NmapExecutor) GetMetrics() *base.ExecutorMetrics {
	e.metrics.Timestamp = time.Now()
	return e.metrics
}

// ==================== 资源管理实现 ====================

// GetResourceUsage 获取资源使用情况
func (e *NmapExecutor) GetResourceUsage() *base.ResourceUsage {
	// TODO: 实现nmap特定的资源使用情况收集
	return &base.ResourceUsage{
		CPUUsage:     0.0,
		MemoryUsage:  0,
		DiskUsage:    0,
		NetworkUsage: 0,
		FileCount:    0,
		ProcessCount: 1,
		ThreadCount:  1,
		Timestamp:    time.Now(),
	}
}

// CleanupResources 清理资源
func (e *NmapExecutor) CleanupResources() error {
	// TODO: 实现nmap特定的资源清理
	// 1. 清理临时扫描文件
	// 2. 终止僵尸nmap进程
	// 3. 清理扫描结果缓存

	// 清理已完成的任务
	e.tasksMutex.Lock()
	for taskID, task := range e.tasks {
		if task.Status == base.TaskStatusCompleted ||
			task.Status == base.TaskStatusFailed ||
			task.Status == base.TaskStatusCanceled {
			delete(e.tasks, taskID)
		}
	}
	e.tasksMutex.Unlock()

	return nil
}

// ==================== 内部方法 ====================

// parseNmapConfig 解析Nmap配置
func (e *NmapExecutor) parseNmapConfig(config interface{}) error {
	// TODO: 实现Nmap配置解析
	// 1. 解析nmap路径
	// 2. 解析默认参数
	// 3. 解析扫描配置文件
	// 4. 验证配置有效性

	// 占位符实现
	if configMap, ok := config.(map[string]interface{}); ok {
		if path, ok := configMap["nmap_path"].(string); ok {
			e.nmapPath = path
		}

		if maxTargets, ok := configMap["max_targets"].(float64); ok {
			e.maxTargets = int(maxTargets)
		}
	}

	return nil
}

// validateNmapInstallation 验证nmap安装
func (e *NmapExecutor) validateNmapInstallation() error {
	// TODO: 实现nmap安装验证
	// 1. 检查nmap可执行文件是否存在
	// 2. 检查nmap版本
	// 3. 检查必要的权限

	// 占位符实现 - 简单检查命令是否可用
	cmd := exec.Command(e.nmapPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nmap not found or not executable: %w", err)
	}

	return nil
}

// taskProcessor 任务处理器
func (e *NmapExecutor) taskProcessor() {
	for task := range e.taskQueue {
		if !e.running {
			break
		}

		// 处理任务
		go func(t *base.Task) {
			ctx := context.Background()
			if t.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, t.Timeout)
				defer cancel()
			}

			_, err := e.Execute(ctx, t)
			if err != nil {
				t.ErrorMessage = err.Error()
				t.Status = base.TaskStatusFailed
			}
		}(task)
	}
}

// updateMetrics 更新指标
func (e *NmapExecutor) updateMetrics(result *base.TaskResult) {
	e.metrics.TasksTotal++

	if result.Success {
		e.metrics.TasksCompleted++
	} else {
		e.metrics.TasksFailed++
	}

	// 计算平均任务时间
	if e.metrics.TasksTotal > 0 {
		totalTime := e.metrics.AverageTaskTime * time.Duration(e.metrics.TasksTotal-1)
		e.metrics.AverageTaskTime = (totalTime + result.Duration) / time.Duration(e.metrics.TasksTotal)
	} else {
		e.metrics.AverageTaskTime = result.Duration
	}

	// 计算错误率
	if e.metrics.TasksTotal > 0 {
		e.metrics.ErrorRate = float64(e.metrics.TasksFailed) / float64(e.metrics.TasksTotal)
	}

	e.metrics.Timestamp = time.Now()
}
