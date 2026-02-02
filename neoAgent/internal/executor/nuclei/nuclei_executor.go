/**
 * Nuclei漏洞扫描工具执行器
 * @author: sun977
 * @date: 2025.10.21
 * @description: Nuclei漏洞扫描工具执行器实现，支持基于模板的漏洞检测
 * @func: 占位符实现，待后续完善
 */
package nuclei

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"neoagent/internal/executor/base"
)

// NucleiExecutor Nuclei执行器
type NucleiExecutor struct {
	config     *base.ExecutorConfig
	status     *base.ExecutorStatus
	tasks      map[string]*base.Task
	tasksMutex sync.RWMutex
	running    bool
	startTime  time.Time
	metrics    *base.ExecutorMetrics

	// Nuclei特定配置
	nucleiPath    string
	templatesPath string
	defaultArgs   []string
	maxTargets    int
	rateLimit     int

	// 任务管理
	taskQueue   chan *base.Task
	cancelFuncs map[string]context.CancelFunc
	cancelMutex sync.RWMutex
}

// NucleiConfig Nuclei配置
type NucleiConfig struct {
	NucleiPath      string   `json:"nuclei_path"`      // nuclei可执行文件路径
	TemplatesPath   string   `json:"templates_path"`   // 模板路径
	DefaultArgs     []string `json:"default_args"`     // 默认参数
	MaxTargets      int      `json:"max_targets"`      // 最大目标数量
	RateLimit       int      `json:"rate_limit"`       // 速率限制
	CustomTemplates []string `json:"custom_templates"` // 自定义模板
	Severity        []string `json:"severity"`         // 严重程度过滤
}

// NucleiResult Nuclei扫描结果
type NucleiResult struct {
	TemplateID       string          `json:"template-id"`
	TemplatePath     string          `json:"template-path"`
	Info             NucleiInfo      `json:"info"`
	Type             string          `json:"type"`
	Host             string          `json:"host"`
	MatchedAt        string          `json:"matched-at"`
	ExtractedResults []string        `json:"extracted-results,omitempty"`
	Request          string          `json:"request,omitempty"`
	Response         string          `json:"response,omitempty"`
	IP               string          `json:"ip,omitempty"`
	Timestamp        time.Time       `json:"timestamp"`
	CurlCommand      string          `json:"curl-command,omitempty"`
	Matcher          []NucleiMatcher `json:"matcher,omitempty"`
}

// NucleiInfo 模板信息
type NucleiInfo struct {
	Name           string               `json:"name"`
	Author         []string             `json:"author"`
	Tags           []string             `json:"tags"`
	Description    string               `json:"description"`
	Reference      []string             `json:"reference,omitempty"`
	Severity       string               `json:"severity"`
	Metadata       map[string]string    `json:"metadata,omitempty"`
	Classification NucleiClassification `json:"classification,omitempty"`
}

// NucleiClassification 分类信息
type NucleiClassification struct {
	CVEID     []string `json:"cve-id,omitempty"`
	CWEID     []string `json:"cwe-id,omitempty"`
	CVSS      string   `json:"cvss-metrics,omitempty"`
	CVSSScore float64  `json:"cvss-score,omitempty"`
}

// NucleiMatcher 匹配器信息
type NucleiMatcher struct {
	Name   string   `json:"name,omitempty"`
	Type   string   `json:"type"`
	Part   string   `json:"part,omitempty"`
	Words  []string `json:"words,omitempty"`
	Regex  []string `json:"regex,omitempty"`
	Status []int    `json:"status,omitempty"`
}

// NewNucleiExecutor 创建Nuclei执行器实例
func NewNucleiExecutor() base.Executor {
	return &NucleiExecutor{
		tasks:         make(map[string]*base.Task),
		taskQueue:     make(chan *base.Task, 100),
		cancelFuncs:   make(map[string]context.CancelFunc),
		nucleiPath:    "nuclei", // 默认路径
		templatesPath: "nuclei-templates",
		defaultArgs:   []string{"-json", "-silent", "-no-color"},
		maxTargets:    1000,
		rateLimit:     150, // 默认150请求/秒
		metrics: &base.ExecutorMetrics{
			Timestamp: time.Now(),
		},
	}
}

// ==================== 基础信息实现 ====================

// GetType 获取执行器类型
func (e *NucleiExecutor) GetType() base.ExecutorType {
	return base.ExecutorTypeNuclei
}

// GetName 获取执行器名称
func (e *NucleiExecutor) GetName() string {
	return "Nuclei Executor"
}

// GetVersion 获取执行器版本
func (e *NucleiExecutor) GetVersion() string {
	return "1.0.0"
}

// GetDescription 获取执行器描述
func (e *NucleiExecutor) GetDescription() string {
	return "Nuclei漏洞扫描工具执行器，支持基于模板的快速漏洞检测"
}

// GetTaskSupport 获取执行器能力列表 (原GetCapabilities)
func (e *NucleiExecutor) GetTaskSupport() []string {
	return []string{
		"vulnerability_scan",    // 漏洞扫描
		"template_based_scan",   // 基于模板的扫描
		"web_application_scan",  // Web应用扫描
		"network_service_scan",  // 网络服务扫描
		"subdomain_takeover",    // 子域名接管检测
		"misconfiguration_scan", // 错误配置检测
		"exposed_panel_scan",    // 暴露面板检测
		"technology_detection",  // 技术栈检测
		"custom_template_scan",  // 自定义模板扫描
	}
}

// ==================== 生命周期管理实现 ====================

// Initialize 初始化执行器
func (e *NucleiExecutor) Initialize(config *base.ExecutorConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// TODO: 实现Nuclei执行器初始化逻辑
	// 1. 验证nuclei可执行文件
	// 2. 检查nuclei版本
	// 3. 更新模板库
	// 4. 验证模板路径
	// 5. 设置资源限制

	e.config = config

	// 解析Nuclei特定配置
	if nucleiConfig, ok := config.Custom["nuclei"]; ok {
		if err := e.parseNucleiConfig(nucleiConfig); err != nil {
			return fmt.Errorf("parse nuclei config: %w", err)
		}
	}

	// 验证nuclei安装
	if err := e.validateNucleiInstallation(); err != nil {
		return fmt.Errorf("validate nuclei installation: %w", err)
	}

	// 更新模板库
	if err := e.updateTemplates(); err != nil {
		// 模板更新失败不应该阻止初始化，只记录警告
		fmt.Printf("Warning: failed to update templates: %v\n", err)
	}

	e.status = &base.ExecutorStatus{
		Type:      base.ExecutorTypeNuclei,
		Name:      e.GetName(),
		Status:    "initialized",
		IsRunning: false,
		Timestamp: time.Now(),
	}

	return nil
}

// Start 启动执行器
func (e *NucleiExecutor) Start() error {
	if e.running {
		return fmt.Errorf("executor is already running")
	}

	// TODO: 实现Nuclei执行器启动逻辑
	// 1. 启动任务处理goroutine
	// 2. 启动指标收集goroutine
	// 3. 启动健康检查goroutine
	// 4. 预热nuclei进程（如果需要）

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
func (e *NucleiExecutor) Stop() error {
	if !e.running {
		return fmt.Errorf("executor is not running")
	}

	// TODO: 实现Nuclei执行器停止逻辑
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
func (e *NucleiExecutor) Restart() error {
	if err := e.Stop(); err != nil {
		return fmt.Errorf("stop executor: %w", err)
	}

	time.Sleep(2 * time.Second) // 等待nuclei进程完全退出

	if err := e.Start(); err != nil {
		return fmt.Errorf("start executor: %w", err)
	}

	return nil
}

// IsRunning 检查执行器是否运行中
func (e *NucleiExecutor) IsRunning() bool {
	return e.running
}

// GetStatus 获取执行器状态
func (e *NucleiExecutor) GetStatus() *base.ExecutorStatus {
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
func (e *NucleiExecutor) Execute(ctx context.Context, task *base.Task) (*base.TaskResult, error) {
	if !e.running {
		return nil, fmt.Errorf("executor is not running")
	}

	if task == nil {
		return nil, fmt.Errorf("task cannot be nil")
	}

	// TODO: 实现Nuclei扫描任务执行逻辑
	// 1. 解析扫描参数
	// 2. 构建nuclei命令
	// 3. 执行扫描
	// 4. 解析JSON结果
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

	// 执行nuclei扫描
	if err := e.executeNucleiScan(taskCtx, task, result); err != nil {
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

// executeNucleiScan 执行nuclei扫描
func (e *NucleiExecutor) executeNucleiScan(ctx context.Context, task *base.Task, result *base.TaskResult) error {
	// TODO: 实现具体的nuclei扫描逻辑
	// 1. 解析目标和参数
	// 2. 构建nuclei命令行
	// 3. 执行扫描
	// 4. 解析JSON输出
	// 5. 转换为标准格式

	// 获取扫描目标
	targets, ok := task.Config["targets"].([]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid targets in task config")
	}

	// 获取扫描参数
	templates := []string{}
	if t, ok := task.Config["templates"].([]interface{}); ok {
		for _, template := range t {
			if templateStr, ok := template.(string); ok {
				templates = append(templates, templateStr)
			}
		}
	}

	// 构建nuclei命令
	args := e.buildNucleiCommand(targets, templates, task.Config)

	// 执行nuclei命令
	cmd := exec.CommandContext(ctx, e.nucleiPath, args...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		// nuclei可能在找到漏洞时返回非零退出码，这是正常的
		if ctx.Err() != nil {
			return ctx.Err() // 上下文取消
		}
		// 继续处理输出，可能包含有效结果
	}

	// 解析nuclei输出
	nucleiResults, err := e.parseNucleiOutput(string(output))
	if err != nil {
		result.Error = fmt.Sprintf("parse nuclei output: %v", err)
		result.Logs = []string{string(output)}
		return err
	}

	// 转换为标准格式
	standardResults := e.convertToStandardResults(nucleiResults)

	// 设置结果
	result.Output = &base.TaskOutput{
		Results: standardResults,
		Summary: &base.ScanSummary{
			TotalTargets:   len(targets),
			ScannedTargets: len(standardResults),
			Duration:       result.Duration,
			StartTime:      result.StartTime,
			EndTime:        result.EndTime,
		},
	}

	result.Logs = []string{string(output)}

	return nil
}

// buildNucleiCommand 构建nuclei命令
func (e *NucleiExecutor) buildNucleiCommand(targets []interface{}, templates []string, config map[string]interface{}) []string {
	args := make([]string, 0)

	// 添加默认参数
	args = append(args, e.defaultArgs...)

	// 添加模板
	if len(templates) > 0 {
		args = append(args, "-t", strings.Join(templates, ","))
	} else {
		// 使用默认模板路径
		args = append(args, "-t", e.templatesPath)
	}

	// 添加严重程度过滤
	if severity, ok := config["severity"].([]interface{}); ok {
		severityList := make([]string, 0)
		for _, s := range severity {
			if severityStr, ok := s.(string); ok {
				severityList = append(severityList, severityStr)
			}
		}
		if len(severityList) > 0 {
			args = append(args, "-severity", strings.Join(severityList, ","))
		}
	}

	// 添加标签过滤
	if tags, ok := config["tags"].([]interface{}); ok {
		tagList := make([]string, 0)
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				tagList = append(tagList, tagStr)
			}
		}
		if len(tagList) > 0 {
			args = append(args, "-tags", strings.Join(tagList, ","))
		}
	}

	// 添加排除标签
	if excludeTags, ok := config["exclude_tags"].([]interface{}); ok {
		excludeTagList := make([]string, 0)
		for _, tag := range excludeTags {
			if tagStr, ok := tag.(string); ok {
				excludeTagList = append(excludeTagList, tagStr)
			}
		}
		if len(excludeTagList) > 0 {
			args = append(args, "-exclude-tags", strings.Join(excludeTagList, ","))
		}
	}

	// 添加速率限制
	if rateLimit, ok := config["rate_limit"].(float64); ok {
		args = append(args, "-rate-limit", fmt.Sprintf("%.0f", rateLimit))
	} else {
		args = append(args, "-rate-limit", fmt.Sprintf("%d", e.rateLimit))
	}

	// 添加并发数
	if concurrency, ok := config["concurrency"].(float64); ok {
		args = append(args, "-c", fmt.Sprintf("%.0f", concurrency))
	}

	// 添加超时设置
	if timeout, ok := config["timeout"].(float64); ok {
		args = append(args, "-timeout", fmt.Sprintf("%.0fs", timeout))
	}

	// 添加重试次数
	if retries, ok := config["retries"].(float64); ok {
		args = append(args, "-retries", fmt.Sprintf("%.0f", retries))
	}

	// 添加目标
	for _, target := range targets {
		if targetStr, ok := target.(string); ok {
			args = append(args, "-u", targetStr)
		}
	}

	return args
}

// parseNucleiOutput 解析nuclei输出
func (e *NucleiExecutor) parseNucleiOutput(output string) ([]NucleiResult, error) {
	// TODO: 实现JSON解析逻辑
	// 1. 按行分割输出
	// 2. 解析每行JSON
	// 3. 验证结果格式
	// 4. 过滤无效结果

	results := make([]NucleiResult, 0)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 跳过非JSON行（如进度信息）
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var result NucleiResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			// 跳过无法解析的行
			continue
		}

		// 设置时间戳
		result.Timestamp = time.Now()

		results = append(results, result)
	}

	return results, nil
}

// convertToStandardResults 转换为标准结果格式
func (e *NucleiExecutor) convertToStandardResults(nucleiResults []NucleiResult) []base.ScanResult {
	results := make([]base.ScanResult, 0, len(nucleiResults))

	// 按主机分组结果
	hostResults := make(map[string][]NucleiResult)
	for _, nucleiResult := range nucleiResults {
		host := nucleiResult.Host
		if host == "" {
			host = "unknown"
		}
		hostResults[host] = append(hostResults[host], nucleiResult)
	}

	// 为每个主机创建扫描结果
	for host, hostVulns := range hostResults {
		vulnerabilities := make([]base.Vulnerability, 0, len(hostVulns))

		for _, vuln := range hostVulns {
			// 转换严重程度
			severity := e.convertSeverity(vuln.Info.Severity)

			// 计算CVSS分数
			cvssScore := vuln.Info.Classification.CVSSScore
			if cvssScore == 0 {
				cvssScore = e.getSeverityScore(severity)
			}

			vulnerability := base.Vulnerability{
				ID:          vuln.TemplateID,
				Name:        vuln.Info.Name,
				Description: vuln.Info.Description,
				Severity:    severity,
				CVSS:        cvssScore,
				References:  vuln.Info.Reference,
				Solution:    "", // nuclei通常不提供解决方案
				Extra: map[string]interface{}{
					"template_path":     vuln.TemplatePath,
					"matched_at":        vuln.MatchedAt,
					"type":              vuln.Type,
					"author":            vuln.Info.Author,
					"tags":              vuln.Info.Tags,
					"cve_id":            vuln.Info.Classification.CVEID,
					"cwe_id":            vuln.Info.Classification.CWEID,
					"cvss_metrics":      vuln.Info.Classification.CVSS,
					"extracted_results": vuln.ExtractedResults,
					"request":           vuln.Request,
					"response":          vuln.Response,
					"curl_command":      vuln.CurlCommand,
					"matcher":           vuln.Matcher,
				},
			}

			vulnerabilities = append(vulnerabilities, vulnerability)
		}

		result := base.ScanResult{
			Target:          host,
			Vulnerabilities: vulnerabilities,
			Extra: map[string]interface{}{
				"vulnerability_count": len(vulnerabilities),
				"scan_type":           "nuclei",
			},
			Timestamp: time.Now(),
		}

		results = append(results, result)
	}

	return results
}

// convertSeverity 转换严重程度
func (e *NucleiExecutor) convertSeverity(nucleiSeverity string) string {
	switch strings.ToLower(nucleiSeverity) {
	case "info":
		return "info"
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "critical":
		return "critical"
	default:
		return "unknown"
	}
}

// getSeverityScore 根据严重程度获取CVSS分数
func (e *NucleiExecutor) getSeverityScore(severity string) float64 {
	switch severity {
	case "info":
		return 0.0
	case "low":
		return 3.9
	case "medium":
		return 6.9
	case "high":
		return 8.9
	case "critical":
		return 10.0
	default:
		return 0.0
	}
}

// Cancel 取消扫描任务
func (e *NucleiExecutor) Cancel(taskID string) error {
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
func (e *NucleiExecutor) Pause(taskID string) error {
	// TODO: 实现任务暂停逻辑
	// nuclei可能不支持暂停，可以考虑终止后重新开始
	return fmt.Errorf("pause operation not supported by nuclei executor")
}

// Resume 恢复扫描任务
func (e *NucleiExecutor) Resume(taskID string) error {
	// TODO: 实现任务恢复逻辑
	// nuclei可能不支持恢复，可以考虑重新开始扫描
	return fmt.Errorf("resume operation not supported by nuclei executor")
}

// ==================== 任务管理实现 ====================

// GetTask 获取任务信息
func (e *NucleiExecutor) GetTask(taskID string) (*base.Task, error) {
	e.tasksMutex.RLock()
	defer e.tasksMutex.RUnlock()

	task, exists := e.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return task, nil
}

// GetTasks 获取所有任务
func (e *NucleiExecutor) GetTasks() ([]*base.Task, error) {
	e.tasksMutex.RLock()
	defer e.tasksMutex.RUnlock()

	tasks := make([]*base.Task, 0, len(e.tasks))
	for _, task := range e.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskStatus 获取任务状态
func (e *NucleiExecutor) GetTaskStatus(taskID string) (base.TaskStatus, error) {
	task, err := e.GetTask(taskID)
	if err != nil {
		return "", err
	}

	return task.Status, nil
}

// GetTaskResult 获取任务结果
func (e *NucleiExecutor) GetTaskResult(taskID string) (*base.TaskResult, error) {
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
func (e *NucleiExecutor) GetTaskLogs(taskID string) ([]string, error) {
	task, err := e.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	return task.Logs, nil
}

// ==================== 配置管理实现 ====================

// UpdateConfig 更新配置
func (e *NucleiExecutor) UpdateConfig(config *base.ExecutorConfig) error {
	if err := e.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	e.config = config
	e.config.UpdatedAt = time.Now()

	// 重新解析Nuclei配置
	if nucleiConfig, ok := config.Custom["nuclei"]; ok {
		if err := e.parseNucleiConfig(nucleiConfig); err != nil {
			return fmt.Errorf("parse nuclei config: %w", err)
		}
	}

	return nil
}

// GetConfig 获取当前配置
func (e *NucleiExecutor) GetConfig() *base.ExecutorConfig {
	return e.config
}

// ValidateConfig 验证配置
func (e *NucleiExecutor) ValidateConfig(config *base.ExecutorConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Type != base.ExecutorTypeNuclei {
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
func (e *NucleiExecutor) HealthCheck() *base.HealthStatus {
	checks := make(map[string]base.CheckResult)
	isHealthy := true

	// 检查nuclei可执行文件
	if err := e.validateNucleiInstallation(); err != nil {
		checks["nuclei_installation"] = base.CheckResult{
			Name:      "Nuclei Installation",
			Status:    "error",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
		isHealthy = false
	} else {
		checks["nuclei_installation"] = base.CheckResult{
			Name:      "Nuclei Installation",
			Status:    "ok",
			Message:   "Nuclei is properly installed",
			Timestamp: time.Now(),
		}
	}

	// 检查模板库
	if err := e.validateTemplates(); err != nil {
		checks["templates"] = base.CheckResult{
			Name:      "Templates",
			Status:    "warning",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	} else {
		checks["templates"] = base.CheckResult{
			Name:      "Templates",
			Status:    "ok",
			Message:   "Templates are available",
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
func (e *NucleiExecutor) GetMetrics() *base.ExecutorMetrics {
	e.metrics.Timestamp = time.Now()
	return e.metrics
}

// ==================== 资源管理实现 ====================

// GetResourceUsage 获取资源使用情况
func (e *NucleiExecutor) GetResourceUsage() *base.ResourceUsage {
	// TODO: 实现nuclei特定的资源使用情况收集
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
func (e *NucleiExecutor) CleanupResources() error {
	// TODO: 实现nuclei特定的资源清理
	// 1. 清理临时扫描文件
	// 2. 终止僵尸nuclei进程
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

// parseNucleiConfig 解析Nuclei配置
func (e *NucleiExecutor) parseNucleiConfig(config interface{}) error {
	// TODO: 实现Nuclei配置解析
	// 1. 解析nuclei路径
	// 2. 解析模板路径
	// 3. 解析默认参数
	// 4. 验证配置有效性

	// 占位符实现
	if configMap, ok := config.(map[string]interface{}); ok {
		if path, ok := configMap["nuclei_path"].(string); ok {
			e.nucleiPath = path
		}

		if templatesPath, ok := configMap["templates_path"].(string); ok {
			e.templatesPath = templatesPath
		}

		if maxTargets, ok := configMap["max_targets"].(float64); ok {
			e.maxTargets = int(maxTargets)
		}

		if rateLimit, ok := configMap["rate_limit"].(float64); ok {
			e.rateLimit = int(rateLimit)
		}
	}

	return nil
}

// validateNucleiInstallation 验证nuclei安装
func (e *NucleiExecutor) validateNucleiInstallation() error {
	// TODO: 实现nuclei安装验证
	// 1. 检查nuclei可执行文件是否存在
	// 2. 检查nuclei版本
	// 3. 检查必要的权限

	// 占位符实现 - 简单检查命令是否可用
	cmd := exec.Command(e.nucleiPath, "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nuclei not found or not executable: %w", err)
	}

	return nil
}

// validateTemplates 验证模板库
func (e *NucleiExecutor) validateTemplates() error {
	// TODO: 实现模板库验证
	// 1. 检查模板路径是否存在
	// 2. 检查模板数量
	// 3. 验证模板格式

	// 占位符实现
	return nil
}

// updateTemplates 更新模板库
func (e *NucleiExecutor) updateTemplates() error {
	// TODO: 实现模板库更新
	// 1. 执行nuclei -update-templates
	// 2. 检查更新结果
	// 3. 验证新模板

	// 占位符实现
	cmd := exec.Command(e.nucleiPath, "-update-templates")
	return cmd.Run()
}

// taskProcessor 任务处理器
func (e *NucleiExecutor) taskProcessor() {
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
func (e *NucleiExecutor) updateMetrics(result *base.TaskResult) {
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
