// 自定义日志格式化器
package logger

import (
	"fmt"
	"net/http"
	"time"

	"neoagent/internal/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FormatTimestamp 格式化时间戳为统一的毫秒精度格式
// 返回格式："2006-01-02 15:04:05.000"
func FormatTimestamp(t time.Time) string {
	// 除了日志管理器之外的其他模块使用的时间戳格式
	return t.Format("2006-01-02 15:04:05.000")
}

// NowFormatted 返回当前时间的格式化字符串
// 返回格式："2006-01-02 15:04:05.000"
func NowFormatted() string {
	return FormatTimestamp(time.Now())
}

// LogType 日志类型枚举
type LogType string

const (
	// AccessLog 访问日志 - 记录HTTP请求和API调用
	AccessLog LogType = "access"
	// BusinessLog 业务日志 - 记录业务操作（登录、注册等）
	BusinessLog LogType = "business"
	// ErrorLog 错误日志 - 记录系统错误和异常
	ErrorLog LogType = "error"
	// SystemLog 系统日志 - 记录系统运行状态
	SystemLog LogType = "system"
	// DebugLog 调试日志 - 记录开发调试信息
	DebugLog LogType = "debug"
	// AuditLog 审计日志 - 记录安全相关操作
	AuditLog LogType = "audit"
	// ScanLog 扫描日志 - 记录扫描任务执行情况（Agent特有）
	ScanLog LogType = "scan"
	// PluginLog 插件日志 - 记录插件操作（Agent特有）
	PluginLog LogType = "plugin"
	// SecurityLog 安全日志 - 记录安全事件（Agent特有）
	SecurityLog LogType = "security"
)

// AccessLogEntry 访问日志条目结构
type AccessLogEntry struct {
	Timestamp    time.Time `json:"timestamp"`     // 请求时间
	Method       string    `json:"method"`        // HTTP方法
	Path         string    `json:"path"`          // 请求路径
	Query        string    `json:"query"`         // 查询参数
	StatusCode   int       `json:"status_code"`   // 响应状态码
	ResponseTime int64     `json:"response_time"` // 响应时间(毫秒)
	ClientIP     string    `json:"client_ip"`     // 客户端IP
	UserAgent    string    `json:"user_agent"`    // 用户代理
	UserID       uint      `json:"user_id"`       // 用户ID（如果已认证）
	RequestID    string    `json:"request_id"`    // 请求追踪ID
	RequestSize  int64     `json:"request_size"`  // 请求大小
	ResponseSize int64     `json:"response_size"` // 响应大小
}

// BusinessLogEntry 业务日志条目结构
type BusinessLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`    // 操作时间
	Operation   string                 `json:"operation"`    // 操作类型（login, register, logout等）
	UserID      uint                   `json:"user_id"`      // 操作用户ID
	Username    string                 `json:"username"`     // 用户名
	ClientIP    string                 `json:"client_ip"`    // 客户端IP
	Result      string                 `json:"result"`       // 操作结果（success, failed）
	Message     string                 `json:"message"`      // 详细信息
	RequestID   string                 `json:"request_id"`   // 请求追踪ID
	ExtraFields map[string]interface{} `json:"extra_fields"` // 额外字段
}

// ErrorLogEntry 错误日志条目结构
type ErrorLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`    // 错误时间
	Level       string                 `json:"level"`        // 错误级别
	Error       string                 `json:"error"`        // 错误信息
	StackTrace  string                 `json:"stack_trace"`  // 堆栈跟踪
	RequestID   string                 `json:"request_id"`   // 请求追踪ID
	UserID      uint                   `json:"user_id"`      // 用户ID
	ClientIP    string                 `json:"client_ip"`    // 客户端IP
	Path        string                 `json:"path"`         // 请求路径
	Method      string                 `json:"method"`       // HTTP方法
	ExtraFields map[string]interface{} `json:"extra_fields"` // 额外字段
}

// SystemLogEntry 系统日志条目结构
type SystemLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`    // 时间
	Component   string                 `json:"component"`    // 系统组件（database, redis, grpc等）
	Event       string                 `json:"event"`        // 事件类型（startup, shutdown, error等）
	Message     string                 `json:"message"`      // 详细信息
	Level       string                 `json:"level"`        // 日志级别
	ExtraFields map[string]interface{} `json:"extra_fields"` // 额外字段
}

// AuditLogEntry 审计日志条目结构
type AuditLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`    // 操作时间
	UserID      uint                   `json:"user_id"`      // 操作用户ID
	Username    string                 `json:"username"`     // 用户名
	Action      string                 `json:"action"`       // 操作动作
	Resource    string                 `json:"resource"`     // 操作资源
	Result      string                 `json:"result"`       // 操作结果
	ClientIP    string                 `json:"client_ip"`    // 客户端IP
	UserAgent   string                 `json:"user_agent"`   // 用户代理
	RequestID   string                 `json:"request_id"`   // 请求追踪ID
	ExtraFields map[string]interface{} `json:"extra_fields"` // 额外字段
}

// ScanLogEntry 扫描日志条目结构（Agent特有）
type ScanLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`    // 扫描时间
	TaskID      string                 `json:"task_id"`      // 任务ID
	ScanType    string                 `json:"scan_type"`    // 扫描类型（asset, web, poc, dir等）
	Target      string                 `json:"target"`       // 扫描目标
	Status      string                 `json:"status"`       // 扫描状态（running, completed, failed）
	Progress    int                    `json:"progress"`     // 扫描进度（0-100）
	Result      string                 `json:"result"`       // 扫描结果摘要
	Duration    int64                  `json:"duration"`     // 扫描耗时（毫秒）
	ExtraFields map[string]interface{} `json:"extra_fields"` // 额外字段
}

// PluginLogEntry 插件日志条目结构（Agent特有）
type PluginLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`    // 时间
	PluginName  string                 `json:"plugin_name"`  // 插件名称
	PluginType  string                 `json:"plugin_type"`  // 插件类型（shell, file, monitor）
	Action      string                 `json:"action"`       // 操作动作（load, execute, unload）
	Status      string                 `json:"status"`       // 状态（success, failed）
	Message     string                 `json:"message"`      // 详细信息
	ExtraFields map[string]interface{} `json:"extra_fields"` // 额外字段
}

// SecurityLogEntry 安全日志条目结构（Agent特有）
type SecurityLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`    // 时间
	EventType   string                 `json:"event_type"`   // 事件类型（virus_detected, quarantine, threat等）
	Severity    string                 `json:"severity"`     // 严重程度（low, medium, high, critical）
	Source      string                 `json:"source"`       // 事件源
	Target      string                 `json:"target"`       // 目标对象
	Action      string                 `json:"action"`       // 采取的行动
	Result      string                 `json:"result"`       // 处理结果
	ExtraFields map[string]interface{} `json:"extra_fields"` // 额外字段
}

// LogHTTPRequest 记录标准HTTP请求日志（非Gin框架）
// 用于记录标准HTTP处理器的请求日志
func LogHTTPRequest(r *http.Request, statusCode int, responseTime time.Duration, requestID string, userID uint) {
	if LoggerInstance == nil {
		return
	}

	// 构建访问日志条目（移除未使用的Timestamp字段）
	entry := AccessLogEntry{
		Method:       r.Method,
		Path:         r.URL.Path,
		Query:        r.URL.RawQuery,
		StatusCode:   statusCode,
		ResponseTime: responseTime.Milliseconds(),
		ClientIP:     utils.GetClientIPFromRequest(r),
		UserAgent:    r.UserAgent(),
		UserID:       userID,
		RequestID:    requestID,
		RequestSize:  r.ContentLength,
	}

	// 记录日志（移除重复的timestamp字段，使用logrus自带的时间戳）
	LoggerInstance.logger.WithFields(logrus.Fields{
		"type":          AccessLog,
		"method":        entry.Method,
		"path":          entry.Path,
		"query":         entry.Query,
		"status_code":   entry.StatusCode,
		"response_time": entry.ResponseTime,
		"client_ip":     entry.ClientIP,
		"user_agent":    entry.UserAgent,
		"user_id":       entry.UserID,
		"request_id":    entry.RequestID,
		"request_size":  entry.RequestSize,
	}).Info("HTTP request processed")
}

// LogAccessRequest 记录HTTP访问日志
// 用于记录所有HTTP请求的详细信息，包括请求参数、响应时间、状态码等
func LogAccessRequest(c *gin.Context, startTime time.Time, requestID string, userID uint) {
	if LoggerInstance == nil {
		return
	}

	// 计算响应时间
	responseTime := time.Since(startTime).Milliseconds()

	// 构建访问日志条目（移除未使用的Timestamp字段）
	entry := AccessLogEntry{
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		Query:        c.Request.URL.RawQuery,
		StatusCode:   c.Writer.Status(),
		ResponseTime: responseTime,
		ClientIP:     utils.GetClientIP(c),
		UserAgent:    c.Request.UserAgent(),
		UserID:       userID,
		RequestID:    requestID,
		RequestSize:  c.Request.ContentLength,
		ResponseSize: int64(c.Writer.Size()),
	}

	// 记录日志（移除重复的timestamp字段，使用logrus自带的时间戳）
	LoggerInstance.logger.WithFields(logrus.Fields{
		"type":          AccessLog,
		"method":        entry.Method,
		"path":          entry.Path,
		"query":         entry.Query,
		"status_code":   entry.StatusCode,
		"response_time": entry.ResponseTime,
		"client_ip":     entry.ClientIP,
		"user_agent":    entry.UserAgent,
		"user_id":       entry.UserID,
		"request_id":    entry.RequestID,
		"request_size":  entry.RequestSize,
		"response_size": entry.ResponseSize,
	}).Info("HTTP request processed")
}

// LogBusinessOperation 记录业务操作日志
// 用于记录用户的业务操作，如登录、注册、权限变更等
func LogBusinessOperation(operation string, userID uint, username, clientIP, requestID, result, message string, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	// 构建业务日志条目（移除未使用的Timestamp字段）
	entry := BusinessLogEntry{
		Operation: operation,
		UserID:    userID,
		Username:  username,
		ClientIP:  clientIP,
		Result:    result,
		Message:   message,
		RequestID: requestID,
	}

	// 构建日志字段（移除重复的timestamp字段，使用logrus自带的时间戳）
	fields := logrus.Fields{
		"type":       BusinessLog,
		"operation":  entry.Operation,
		"user_id":    entry.UserID,
		"username":   entry.Username,
		"client_ip":  entry.ClientIP,
		"result":     entry.Result,
		"message":    entry.Message,
		"request_id": entry.RequestID,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// 根据结果选择日志级别
	if result == "success" {
		LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("Business operation: %s", operation))
	} else {
		LoggerInstance.logger.WithFields(fields).Warn(fmt.Sprintf("Business operation failed: %s", operation))
	}
}

// LogError 记录错误日志
// 用于记录系统错误、异常和业务错误
func LogError(err error, requestID string, userID uint, clientIP, path, method string, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	if err == nil {
		return
	}

	// 构建错误日志条目（移除未使用的Timestamp字段）
	entry := ErrorLogEntry{
		Level:     "error",
		Error:     err.Error(),
		RequestID: requestID,
		UserID:    userID,
		ClientIP:  clientIP,
		Path:      path,
		Method:    method,
	}

	// 构建日志字段（移除重复的timestamp字段，使用logrus自带的时间戳）
	fields := logrus.Fields{
		"type":       ErrorLog,
		"level":      entry.Level,
		"error":      entry.Error,
		"request_id": entry.RequestID,
		"user_id":    entry.UserID,
		"client_ip":  entry.ClientIP,
		"path":       entry.Path,
		"method":     entry.Method,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// // 记录错误日志
	// LoggerInstance.logger.WithFields(fields).Error("System error occurred")
	// 记录错误日志，包含具体的错误信息
	LoggerInstance.logger.WithFields(fields).Errorf("System error occurred: %s", err.Error())
}

// LogInfo 记录信息日志
// 用于记录一般性信息、成功操作和状态更新
func LogInfo(message string, requestID string, userID uint, clientIP, path, method string, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	if message == "" {
		return
	}

	// 构建日志字段
	fields := logrus.Fields{
		"type":       "info",
		"message":    message,
		"request_id": requestID,
		"user_id":    userID,
		"client_ip":  clientIP,
		"path":       path,
		"method":     method,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// 记录信息日志
	LoggerInstance.logger.WithFields(fields).Info(message)
}

// LogWarn 记录警告日志
// 用于记录警告信息、潜在问题和需要关注的状态
func LogWarn(message string, requestID string, userID uint, clientIP, path, method string, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	if message == "" {
		return
	}

	// 构建日志字段
	fields := logrus.Fields{
		"type":       "warn",
		"message":    message,
		"request_id": requestID,
		"user_id":    userID,
		"client_ip":  clientIP,
		"path":       path,
		"method":     method,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// 记录警告日志
	LoggerInstance.logger.WithFields(fields).Warn(message)
}

// LogSystemEvent 记录系统事件日志
// 用于记录系统启动、关闭、组件状态变化等系统级事件
func LogSystemEvent(component, event, message string, level LogLevel, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	// 转换为logrus级别
	logrusLevel := toLogrusLevel(level)

	// 构建系统日志条目（移除未使用的Timestamp字段）
	entry := SystemLogEntry{
		Component: component,
		Event:     event,
		Message:   message,
		Level:     logrusLevel.String(),
	}

	// 构建日志字段（移除重复的timestamp字段，使用logrus自带的时间戳）
	fields := logrus.Fields{
		"type":      SystemLog,
		"component": entry.Component,
		"event":     entry.Event,
		"message":   entry.Message,
		"level":     entry.Level,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// 根据级别记录日志
	switch logrusLevel {
	case logrus.DebugLevel:
		LoggerInstance.logger.WithFields(fields).Debug(fmt.Sprintf("System event: %s - %s", component, event))
	case logrus.InfoLevel:
		LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("System event: %s - %s", component, event))
	case logrus.WarnLevel:
		LoggerInstance.logger.WithFields(fields).Warn(fmt.Sprintf("System event: %s - %s", component, event))
	case logrus.ErrorLevel:
		LoggerInstance.logger.WithFields(fields).Error(fmt.Sprintf("System event: %s - %s", component, event))
	case logrus.FatalLevel:
		LoggerInstance.logger.WithFields(fields).Fatal(fmt.Sprintf("System event: %s - %s", component, event))
	default:
		LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("System event: %s - %s", component, event))
	}
}

// LogAuditOperation 记录审计日志
// 用于记录安全相关的操作，满足审计和合规要求
func LogAuditOperation(userID uint, username, action, resource, result, clientIP, userAgent, requestID string, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	// 构建审计日志条目（移除未使用的Timestamp字段）
	entry := AuditLogEntry{
		UserID:    userID,
		Username:  username,
		Action:    action,
		Resource:  resource,
		Result:    result,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		RequestID: requestID,
	}

	// 构建日志字段（移除重复的timestamp字段，使用logrus自带的时间戳）
	fields := logrus.Fields{
		"type":       AuditLog,
		"user_id":    entry.UserID,
		"username":   entry.Username,
		"action":     entry.Action,
		"resource":   entry.Resource,
		"result":     entry.Result,
		"client_ip":  entry.ClientIP,
		"user_agent": entry.UserAgent,
		"request_id": entry.RequestID,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// 记录审计日志
	LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("Audit: %s performed %s on %s", username, action, resource))
}

// LogScanOperation 记录扫描操作日志（Agent特有）
// 用于记录各种扫描任务的执行情况
func LogScanOperation(taskID, scanType, target, status string, progress int, result string, duration int64, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	// 构建扫描日志条目
	entry := ScanLogEntry{
		TaskID:   taskID,
		ScanType: scanType,
		Target:   target,
		Status:   status,
		Progress: progress,
		Result:   result,
		Duration: duration,
	}

	// 构建日志字段
	fields := logrus.Fields{
		"type":      ScanLog,
		"task_id":   entry.TaskID,
		"scan_type": entry.ScanType,
		"target":    entry.Target,
		"status":    entry.Status,
		"progress":  entry.Progress,
		"result":    entry.Result,
		"duration":  entry.Duration,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// 根据状态选择日志级别
	switch status {
	case "completed":
		LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("Scan completed: %s on %s", scanType, target))
	case "failed":
		LoggerInstance.logger.WithFields(fields).Error(fmt.Sprintf("Scan failed: %s on %s", scanType, target))
	case "running":
		LoggerInstance.logger.WithFields(fields).Debug(fmt.Sprintf("Scan running: %s on %s (%d%%)", scanType, target, progress))
	default:
		LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("Scan %s: %s on %s", status, scanType, target))
	}
}

// LogPluginOperation 记录插件操作日志（Agent特有）
// 用于记录插件的加载、执行、卸载等操作
func LogPluginOperation(pluginName, pluginType, action, status, message string, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	// 构建插件日志条目
	entry := PluginLogEntry{
		PluginName: pluginName,
		PluginType: pluginType,
		Action:     action,
		Status:     status,
		Message:    message,
	}

	// 构建日志字段
	fields := logrus.Fields{
		"type":        PluginLog,
		"plugin_name": entry.PluginName,
		"plugin_type": entry.PluginType,
		"action":      entry.Action,
		"status":      entry.Status,
		"message":     entry.Message,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// 根据状态选择日志级别
	if status == "success" {
		LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("Plugin %s: %s %s", action, pluginName, status))
	} else {
		LoggerInstance.logger.WithFields(fields).Error(fmt.Sprintf("Plugin %s failed: %s - %s", action, pluginName, message))
	}
}

// LogSecurityEvent 记录安全事件日志（Agent特有）
// 用于记录病毒检测、威胁处理等安全相关事件
func LogSecurityEvent(eventType, severity, source, target, action, result string, extraFields map[string]interface{}) {
	if LoggerInstance == nil {
		return
	}

	// 构建安全日志条目
	entry := SecurityLogEntry{
		EventType: eventType,
		Severity:  severity,
		Source:    source,
		Target:    target,
		Action:    action,
		Result:    result,
	}

	// 构建日志字段
	fields := logrus.Fields{
		"type":       SecurityLog,
		"event_type": entry.EventType,
		"severity":   entry.Severity,
		"source":     entry.Source,
		"target":     entry.Target,
		"action":     entry.Action,
		"result":     entry.Result,
	}

	// 添加额外字段
	for k, v := range extraFields {
		fields[k] = v
	}

	// 根据严重程度选择日志级别
	switch severity {
	case "critical":
		LoggerInstance.logger.WithFields(fields).Fatal(fmt.Sprintf("Critical security event: %s - %s", eventType, result))
	case "high":
		LoggerInstance.logger.WithFields(fields).Error(fmt.Sprintf("High security event: %s - %s", eventType, result))
	case "medium":
		LoggerInstance.logger.WithFields(fields).Warn(fmt.Sprintf("Medium security event: %s - %s", eventType, result))
	case "low":
		LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("Low security event: %s - %s", eventType, result))
	default:
		LoggerInstance.logger.WithFields(fields).Info(fmt.Sprintf("Security event: %s - %s", eventType, result))
	}
}

// LogLevel 日志级别类型，封装logrus.Level避免Handler层直接依赖logrus
type LogLevel int

const (
	// DebugLevel 调试级别
	DebugLevel LogLevel = iota
	// InfoLevel 信息级别
	InfoLevel
	// WarnLevel 警告级别
	WarnLevel
	// ErrorLevel 错误级别
	ErrorLevel
	// FatalLevel 致命错误级别
	FatalLevel
)

// toLogrusLevel 将封装的LogLevel转换为logrus.Level
// 这是内部函数，外部不应该直接使用logrus
func toLogrusLevel(level LogLevel) logrus.Level {
	switch level {
	case DebugLevel:
		return logrus.DebugLevel
	case InfoLevel:
		return logrus.InfoLevel
	case WarnLevel:
		return logrus.WarnLevel
	case ErrorLevel:
		return logrus.ErrorLevel
	case FatalLevel:
		return logrus.FatalLevel
	default:
		return logrus.InfoLevel
	}
}
