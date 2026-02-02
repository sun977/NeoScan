/**
 * Masscan高速端口扫描工具执行器
 * @author: sun977
 * @date: 2025.10.21
 * @description: Masscan高速端口扫描工具执行器实现，支持大规模高速端口扫描
 * @func: 占位符实现，待后续完善
 */
package masscan

import (
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"neoagent/internal/executor/base"
)

// MasscanExecutor Masscan执行器
type MasscanExecutor struct {
	config     *base.ExecutorConfig
	status     *base.ExecutorStatus
	tasks      map[string]*base.Task
	tasksMutex sync.RWMutex
	running    bool
	startTime  time.Time
	metrics    *base.ExecutorMetrics
	
	// Masscan特定配置
	masscanPath   string
	defaultArgs   []string
	maxRate       int    // 最大扫描速率（包/秒）
	maxTargets    int    // 最大目标数量
	excludeFile   string // 排除文件路径
	
	// 任务管理
	taskQueue    chan *base.Task
	cancelFuncs  map[string]context.CancelFunc
	cancelMutex  sync.RWMutex
}

// MasscanConfig Masscan配置
type MasscanConfig struct {
	MasscanPath   string   `json:"masscan_path"`   // masscan可执行文件路径
	DefaultArgs   []string `json:"default_args"`   // 默认参数
	MaxRate       int      `json:"max_rate"`       // 最大扫描速率
	MaxTargets    int      `json:"max_targets"`    // 最大目标数量
	ExcludeFile   string   `json:"exclude_file"`   // 排除文件路径
	Interface     string   `json:"interface"`      // 网络接口
	SourceIP      string   `json:"source_ip"`      // 源IP地址
	SourcePort    string   `json:"source_port"`    // 源端口范围
}

// MasscanResult Masscan扫描结果（XML格式）
type MasscanResult struct {
	XMLName xml.Name      `xml:"nmaprun"`
	Hosts   []MasscanHost `xml:"host"`
}

// MasscanHost 主机信息
type MasscanHost struct {
	Address MasscanAddress `xml:"address"`
	Ports   []MasscanPort  `xml:"ports>port"`
}

// MasscanAddress 地址信息
type MasscanAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

// MasscanPort 端口信息
type MasscanPort struct {
	Protocol string            `xml:"protocol,attr"`
	PortID   int               `xml:"portid,attr"`
	State    MasscanPortState  `xml:"state"`
	Service  MasscanService    `xml:"service"`
}

// MasscanPortState 端口状态
type MasscanPortState struct {
	State  string `xml:"state,attr"`
	Reason string `xml:"reason,attr"`
}

// MasscanService 服务信息
type MasscanService struct {
	Name    string `xml:"name,attr"`
	Banner  string `xml:"banner,attr"`
}

// MasscanScanResult 标准化的扫描结果
type MasscanScanResult struct {
	Host     string                 `json:"host"`
	Status   string                 `json:"status"`
	Ports    []MasscanPortResult   `json:"ports"`
	ScanTime time.Time             `json:"scan_time"`
}

// MasscanPortResult 端口结果
type MasscanPortResult struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service,omitempty"`
	Banner   string `json:"banner,omitempty"`
}

// NewMasscanExecutor 创建Masscan执行器实例
func NewMasscanExecutor() base.Executor {
	return &MasscanExecutor{
		tasks:       make(map[string]*base.Task),
		taskQueue:   make(chan *base.Task, 100),
		cancelFuncs: make(map[string]context.CancelFunc),
		masscanPath: "masscan", // 默认路径
		defaultArgs: []string{"--open-only", "--banners"},
		maxRate:     1000,  // 默认1000包/秒
		maxTargets:  10000, // 默认最大10000个目标
		metrics: &base.ExecutorMetrics{
			Timestamp: time.Now(),
		},
	}
}

// ==================== 基础信息实现 ====================

// GetType 获取执行器类型
func (e *MasscanExecutor) GetType() base.ExecutorType {
	return base.ExecutorTypeMasscan
}

// GetName 获取执行器名称
func (e *MasscanExecutor) GetName() string {
	return "Masscan Executor"
}

// GetVersion 获取执行器版本
func (e *MasscanExecutor) GetVersion() string {
	return "1.0.0"
}

// GetDescription 获取执行器描述
func (e *MasscanExecutor) GetDescription() string {
	return "Masscan高速端口扫描工具执行器，支持大规模高速端口扫描"
}

// GetTaskSupport 获取执行器能力列表 (原GetCapabilities)
func (e *MasscanExecutor) GetTaskSupport() []string {
	return []string{
		"high_speed_scan",      // 高速扫描
		"large_scale_scan",     // 大规模扫描
		"port_discovery",       // 端口发现
		"banner_grabbing",      // Banner抓取
		"tcp_syn_scan",         // TCP SYN扫描
		"udp_scan",            // UDP扫描
		"range_scan",          // 范围扫描
		"exclude_targets",     // 排除目标
		"rate_limiting",       // 速率限制
	}
}

// ==================== 生命周期管理实现 ====================

// Initialize 初始化执行器
func (e *MasscanExecutor) Initialize(config *base.ExecutorConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	// TODO: 实现Masscan执行器初始化逻辑
	// 1. 验证masscan可执行文件
	// 2. 检查masscan版本
	// 3. 验证权限（需要root权限）
	// 4. 检查网络接口
	// 5. 设置资源限制
	
	e.config = config
	
	// 解析Masscan特定配置
	if masscanConfig, ok := config.Custom["masscan"]; ok {
		if err := e.parseMasscanConfig(masscanConfig); err != nil {
			return fmt.Errorf("parse masscan config: %w", err)
		}
	}
	
	// 验证masscan安装
	if err := e.validateMasscanInstallation(); err != nil {
		return fmt.Errorf("validate masscan installation: %w", err)
	}
	
	e.status = &base.ExecutorStatus{
		Type:      base.ExecutorTypeMasscan,
		Name:      e.GetName(),
		Status:    "initialized",
		IsRunning: false,
		Timestamp: time.Now(),
	}
	
	return nil
}

// Start 启动执行器
func (e *MasscanExecutor) Start() error {
	if e.running {
		return fmt.Errorf("executor is already running")
	}
	
	// TODO: 实现Masscan执行器启动逻辑
	// 1. 启动任务处理goroutine
	// 2. 启动指标收集goroutine
	// 3. 启动健康检查goroutine
	// 4. 检查网络权限
	
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
func (e *MasscanExecutor) Stop() error {
	if !e.running {
		return fmt.Errorf("executor is not running")
	}
	
	// TODO: 实现Masscan执行器停止逻辑
	// 1. 停止接收新任务
	// 2. 等待当前扫描完成或强制终止
	// 3. 清理临时文件
	// 4. 释放网络资源
	
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
func (e *MasscanExecutor) Restart() error {
	if err := e.Stop(); err != nil {
		return fmt.Errorf("stop executor: %w", err)
	}
	
	time.Sleep(2 * time.Second) // 等待masscan进程完全退出
	
	if err := e.Start(); err != nil {
		return fmt.Errorf("start executor: %w", err)
	}
	
	return nil
}

// IsRunning 检查执行器是否运行中
func (e *MasscanExecutor) IsRunning() bool {
	return e.running
}

// GetStatus 获取执行器状态
func (e *MasscanExecutor) GetStatus() *base.ExecutorStatus {
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
func (e *MasscanExecutor) Execute(ctx context.Context, task *base.Task) (*base.TaskResult, error) {
	if !e.running {
		return nil, fmt.Errorf("executor is not running")
	}
	
	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}
	
	// TODO: 实现Masscan扫描任务执行逻辑
	// 1. 解析扫描参数
	// 2. 构建masscan命令
	// 3. 执行扫描
	// 4. 解析XML结果
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
	
	// 执行masscan扫描
	if err := e.executeMasscanScan(taskCtx, task, result); err != nil {
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

// executeMasscanScan 执行masscan扫描
func (e *MasscanExecutor) executeMasscanScan(ctx context.Context, task *base.Task, result *base.TaskResult) error {
	// TODO: 实现具体的masscan扫描逻辑
	// 1. 解析目标和参数
	// 2. 构建masscan命令行
	// 3. 执行扫描
	// 4. 解析XML输出
	// 5. 转换为标准格式
	
	// 获取扫描目标
	targets, ok := task.Config["targets"].([]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid targets in task config")
	}
	
	// 获取端口范围
	ports := "1-65535" // 默认全端口
	if p, ok := task.Config["ports"].(string); ok {
		ports = p
	}
	
	// 构建masscan命令
	args := e.buildMasscanCommand(targets, ports, task.Config)
	
	// 执行masscan命令
	cmd := exec.CommandContext(ctx, e.masscanPath, args...)
	
	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = err.Error()
		result.Logs = []string{string(output)}
		return err
	}
	
	// 解析masscan输出
	masscanResults, err := e.parseMasscanOutput(string(output))
	if err != nil {
		result.Error = fmt.Sprintf("parse masscan output: %v", err)
		result.Logs = []string{string(output)}
		return err
	}
	
	// 转换为标准格式
	standardResults := e.convertToStandardResults(masscanResults)
	
	// 设置结果
	result.Output = &base.TaskOutput{
		Results: standardResults,
		Summary: &base.ScanSummary{
			TotalTargets:   len(targets),
			ScannedTargets: len(masscanResults),
			Duration:       result.Duration,
			StartTime:      result.StartTime,
			EndTime:        result.EndTime,
		},
	}
	
	result.Logs = []string{string(output)}
	
	return nil
}

// buildMasscanCommand 构建masscan命令
func (e *MasscanExecutor) buildMasscanCommand(targets []interface{}, ports string, config map[string]interface{}) []string {
	args := make([]string, 0)
	
	// 添加默认参数
	args = append(args, e.defaultArgs...)
	
	// 添加端口范围
	args = append(args, "-p", ports)
	
	// 添加扫描速率
	if rate, ok := config["rate"].(float64); ok {
		args = append(args, "--rate", fmt.Sprintf("%.0f", rate))
	} else {
		args = append(args, "--rate", fmt.Sprintf("%d", e.maxRate))
	}
	
	// 添加网络接口
	if iface, ok := config["interface"].(string); ok {
		args = append(args, "-e", iface)
	}
	
	// 添加源IP
	if sourceIP, ok := config["source_ip"].(string); ok {
		args = append(args, "--source-ip", sourceIP)
	}
	
	// 添加源端口
	if sourcePort, ok := config["source_port"].(string); ok {
		args = append(args, "--source-port", sourcePort)
	}
	
	// 添加排除文件
	if e.excludeFile != "" {
		args = append(args, "--excludefile", e.excludeFile)
	}
	
	// 添加输出格式
	args = append(args, "-oX", "-") // 输出XML到stdout
	
	// 添加目标
	for _, target := range targets {
		if targetStr, ok := target.(string); ok {
			args = append(args, targetStr)
		}
	}
	
	return args
}

// parseMasscanOutput 解析masscan输出
func (e *MasscanExecutor) parseMasscanOutput(output string) ([]MasscanScanResult, error) {
	// TODO: 实现XML解析逻辑
	// 1. 解析XML格式的masscan输出
	// 2. 提取主机信息
	// 3. 提取端口信息
	// 4. 提取服务信息
	
	var xmlResult MasscanResult
	if err := xml.Unmarshal([]byte(output), &xmlResult); err != nil {
		// 如果XML解析失败，尝试简单的文本解析
		return e.parseTextOutput(output)
	}
	
	results := make([]MasscanScanResult, 0, len(xmlResult.Hosts))
	
	for _, host := range xmlResult.Hosts {
		ports := make([]MasscanPortResult, 0, len(host.Ports))
		
		for _, port := range host.Ports {
			portResult := MasscanPortResult{
				Port:     port.PortID,
				Protocol: port.Protocol,
				State:    port.State.State,
				Service:  port.Service.Name,
				Banner:   port.Service.Banner,
			}
			ports = append(ports, portResult)
		}
		
		result := MasscanScanResult{
			Host:     host.Address.Addr,
			Status:   "up",
			Ports:    ports,
			ScanTime: time.Now(),
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// parseTextOutput 解析文本输出（备用方法）
func (e *MasscanExecutor) parseTextOutput(output string) ([]MasscanScanResult, error) {
	// TODO: 实现简单的文本解析
	// 处理masscan的简单文本输出格式
	
	results := make([]MasscanScanResult, 0)
	hostPorts := make(map[string][]MasscanPortResult)
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 解析格式: "Discovered open port 80/tcp on 192.168.1.1"
		if strings.Contains(line, "Discovered open port") {
			parts := strings.Fields(line)
			if len(parts) >= 6 {
				portProto := parts[3]
				host := parts[5]
				
				// 解析端口和协议
				portProtoParts := strings.Split(portProto, "/")
				if len(portProtoParts) == 2 {
					port, err := strconv.Atoi(portProtoParts[0])
					if err != nil {
						continue
					}
					
					protocol := portProtoParts[1]
					
					portResult := MasscanPortResult{
						Port:     port,
						Protocol: protocol,
						State:    "open",
					}
					
					hostPorts[host] = append(hostPorts[host], portResult)
				}
			}
		}
	}
	
	// 转换为结果格式
	for host, ports := range hostPorts {
		result := MasscanScanResult{
			Host:     host,
			Status:   "up",
			Ports:    ports,
			ScanTime: time.Now(),
		}
		results = append(results, result)
	}
	
	return results, nil
}

// convertToStandardResults 转换为标准结果格式
func (e *MasscanExecutor) convertToStandardResults(masscanResults []MasscanScanResult) []base.ScanResult {
	results := make([]base.ScanResult, 0, len(masscanResults))
	
	for _, masscanResult := range masscanResults {
		// 统计开放端口
		openPorts := len(masscanResult.Ports)
		
		// 判断是否有潜在的安全风险（占位符逻辑）
		vulnerabilities := make([]base.Vulnerability, 0)
		
		// 检查常见的高风险端口
		for _, port := range masscanResult.Ports {
			if port.State == "open" {
				switch port.Port {
				case 21, 22, 23, 25, 53, 80, 110, 135, 139, 143, 443, 445, 993, 995, 1433, 3306, 3389, 5432, 5900:
					// 这些是常见的服务端口，可能需要进一步检查
					if e.isHighRiskPort(port.Port) {
						vulnerabilities = append(vulnerabilities, base.Vulnerability{
							ID:          fmt.Sprintf("OPEN_PORT_%d", port.Port),
							Name:        fmt.Sprintf("High-risk port %d is open", port.Port),
							Description: fmt.Sprintf("Port %d/%s is open and may pose security risks", port.Port, port.Protocol),
							Severity:    e.getPortRiskLevel(port.Port),
							CVSS:        e.getPortCVSS(port.Port),
							References:  []string{},
							Solution:    "Consider closing unnecessary ports or implementing proper access controls",
						})
					}
				}
			}
		}
		
		result := base.ScanResult{
			Target:          masscanResult.Host,
			Vulnerabilities: vulnerabilities,
			Extra: map[string]interface{}{
				"ports":         masscanResult.Ports,
				"open_ports":    openPorts,
				"scan_time":     masscanResult.ScanTime,
				"scan_type":     "masscan",
			},
			Timestamp: time.Now(),
		}
		
		results = append(results, result)
	}
	
	return results
}

// isHighRiskPort 判断是否为高风险端口
func (e *MasscanExecutor) isHighRiskPort(port int) bool {
	highRiskPorts := []int{21, 23, 135, 139, 445, 1433, 3389, 5900}
	for _, riskPort := range highRiskPorts {
		if port == riskPort {
			return true
		}
	}
	return false
}

// getPortRiskLevel 获取端口风险等级
func (e *MasscanExecutor) getPortRiskLevel(port int) string {
	switch port {
	case 21, 23, 135, 139, 445, 1433, 3389, 5900:
		return "high"
	case 22, 25, 53, 80, 110, 143, 443, 993, 995, 3306, 5432:
		return "medium"
	default:
		return "low"
	}
}

// getPortCVSS 获取端口CVSS分数
func (e *MasscanExecutor) getPortCVSS(port int) float64 {
	switch port {
	case 21, 23, 135, 139, 445, 1433, 3389, 5900:
		return 7.5
	case 22, 25, 53, 80, 110, 143, 443, 993, 995, 3306, 5432:
		return 5.0
	default:
		return 3.0
	}
}

// Cancel 取消扫描任务
func (e *MasscanExecutor) Cancel(taskID string) error {
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
func (e *MasscanExecutor) Pause(taskID string) error {
	// TODO: 实现任务暂停逻辑
	// masscan可能不支持暂停，可以考虑终止后重新开始
	return fmt.Errorf("pause operation not supported by masscan executor")
}

// Resume 恢复扫描任务
func (e *MasscanExecutor) Resume(taskID string) error {
	// TODO: 实现任务恢复逻辑
	// masscan可能不支持恢复，可以考虑重新开始扫描
	return fmt.Errorf("resume operation not supported by masscan executor")
}

// ==================== 任务管理实现 ====================

// GetTask 获取任务信息
func (e *MasscanExecutor) GetTask(taskID string) (*base.Task, error) {
	e.tasksMutex.RLock()
	defer e.tasksMutex.RUnlock()
	
	task, exists := e.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	
	return task, nil
}

// GetTasks 获取所有任务
func (e *MasscanExecutor) GetTasks() ([]*base.Task, error) {
	e.tasksMutex.RLock()
	defer e.tasksMutex.RUnlock()
	
	tasks := make([]*base.Task, 0, len(e.tasks))
	for _, task := range e.tasks {
		tasks = append(tasks, task)
	}
	
	return tasks, nil
}

// GetTaskStatus 获取任务状态
func (e *MasscanExecutor) GetTaskStatus(taskID string) (base.TaskStatus, error) {
	task, err := e.GetTask(taskID)
	if err != nil {
		return "", err
	}
	
	return task.Status, nil
}

// GetTaskResult 获取任务结果
func (e *MasscanExecutor) GetTaskResult(taskID string) (*base.TaskResult, error) {
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
func (e *MasscanExecutor) GetTaskLogs(taskID string) ([]string, error) {
	task, err := e.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	
	return task.Logs, nil
}

// ==================== 配置管理实现 ====================

// UpdateConfig 更新配置
func (e *MasscanExecutor) UpdateConfig(config *base.ExecutorConfig) error {
	if err := e.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	
	e.config = config
	e.config.UpdatedAt = time.Now()
	
	// 重新解析Masscan配置
	if masscanConfig, ok := config.Custom["masscan"]; ok {
		if err := e.parseMasscanConfig(masscanConfig); err != nil {
			return fmt.Errorf("parse masscan config: %w", err)
		}
	}
	
	return nil
}

// GetConfig 获取当前配置
func (e *MasscanExecutor) GetConfig() *base.ExecutorConfig {
	return e.config
}

// ValidateConfig 验证配置
func (e *MasscanExecutor) ValidateConfig(config *base.ExecutorConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	if config.Type != base.ExecutorTypeMasscan {
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
func (e *MasscanExecutor) HealthCheck() *base.HealthStatus {
	checks := make(map[string]base.CheckResult)
	isHealthy := true
	
	// 检查masscan可执行文件
	if err := e.validateMasscanInstallation(); err != nil {
		checks["masscan_installation"] = base.CheckResult{
			Name:      "Masscan Installation",
			Status:    "error",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
		isHealthy = false
	} else {
		checks["masscan_installation"] = base.CheckResult{
			Name:      "Masscan Installation",
			Status:    "ok",
			Message:   "Masscan is properly installed",
			Timestamp: time.Now(),
		}
	}
	
	// 检查权限
	if err := e.checkPermissions(); err != nil {
		checks["permissions"] = base.CheckResult{
			Name:      "Permissions",
			Status:    "warning",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	} else {
		checks["permissions"] = base.CheckResult{
			Name:      "Permissions",
			Status:    "ok",
			Message:   "Sufficient permissions for scanning",
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
func (e *MasscanExecutor) GetMetrics() *base.ExecutorMetrics {
	e.metrics.Timestamp = time.Now()
	return e.metrics
}

// ==================== 资源管理实现 ====================

// GetResourceUsage 获取资源使用情况
func (e *MasscanExecutor) GetResourceUsage() *base.ResourceUsage {
	// TODO: 实现masscan特定的资源使用情况收集
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
func (e *MasscanExecutor) CleanupResources() error {
	// TODO: 实现masscan特定的资源清理
	// 1. 清理临时扫描文件
	// 2. 终止僵尸masscan进程
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

// parseMasscanConfig 解析Masscan配置
func (e *MasscanExecutor) parseMasscanConfig(config interface{}) error {
	// TODO: 实现Masscan配置解析
	// 1. 解析masscan路径
	// 2. 解析默认参数
	// 3. 解析扫描速率限制
	// 4. 验证配置有效性
	
	// 占位符实现
	if configMap, ok := config.(map[string]interface{}); ok {
		if path, ok := configMap["masscan_path"].(string); ok {
			e.masscanPath = path
		}
		
		if maxRate, ok := configMap["max_rate"].(float64); ok {
			e.maxRate = int(maxRate)
		}
		
		if maxTargets, ok := configMap["max_targets"].(float64); ok {
			e.maxTargets = int(maxTargets)
		}
		
		if excludeFile, ok := configMap["exclude_file"].(string); ok {
			e.excludeFile = excludeFile
		}
	}
	
	return nil
}

// validateMasscanInstallation 验证masscan安装
func (e *MasscanExecutor) validateMasscanInstallation() error {
	// TODO: 实现masscan安装验证
	// 1. 检查masscan可执行文件是否存在
	// 2. 检查masscan版本
	// 3. 检查必要的权限
	
	// 占位符实现 - 简单检查命令是否可用
	cmd := exec.Command(e.masscanPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("masscan not found or not executable: %w", err)
	}
	
	return nil
}

// checkPermissions 检查权限
func (e *MasscanExecutor) checkPermissions() error {
	// TODO: 实现权限检查
	// 1. 检查是否有root权限（某些扫描需要）
	// 2. 检查网络接口访问权限
	// 3. 检查原始套接字权限
	
	// 占位符实现
	return nil
}

// taskProcessor 任务处理器
func (e *MasscanExecutor) taskProcessor() {
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
func (e *MasscanExecutor) updateMetrics(result *base.TaskResult) {
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