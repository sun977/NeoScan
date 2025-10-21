/**
 * 日志中间件
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端日志中间件，用于记录HTTP请求和响应信息
 * @func: 占位符实现，待后续完善
 */
package middleware

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"neoagent/internal/pkg/logger"
	"neoagent/internal/pkg/utils"
)

// LoggingConfig 日志配置
type LoggingConfig struct {
	// 是否启用请求日志
	EnableRequestLog  bool `json:"enable_request_log"`
	
	// 是否启用响应日志
	EnableResponseLog bool `json:"enable_response_log"`
	
	// 是否记录请求体
	LogRequestBody    bool `json:"log_request_body"`
	
	// 是否记录响应体
	LogResponseBody   bool `json:"log_response_body"`
	
	// 最大请求体大小（字节）
	MaxRequestBodySize int64 `json:"max_request_body_size"`
	
	// 最大响应体大小（字节）
	MaxResponseBodySize int64 `json:"max_response_body_size"`
	
	// 跳过日志的路径
	SkipPaths []string `json:"skip_paths"`
	
	// 慢请求阈值
	SlowRequestThreshold time.Duration `json:"slow_request_threshold"`
}

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct {
	config *LoggingConfig
	logger *logger.LoggerManager
}

// responseWriter 响应写入器包装
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 写入响应数据
func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware(config *LoggingConfig) *LoggingMiddleware {
	if config == nil {
		config = &LoggingConfig{
			EnableRequestLog:     true,
			EnableResponseLog:    true,
			LogRequestBody:       false,
			LogResponseBody:      false,
			MaxRequestBodySize:   1024 * 1024, // 1MB
			MaxResponseBodySize:  1024 * 1024, // 1MB
			SlowRequestThreshold: 2 * time.Second,
			SkipPaths: []string{
				"/health",
				"/ping",
			},
		}
	}
	
	return &LoggingMiddleware{
		config: config,
		logger: logger.LoggerInstance,
	}
}

// Handler 日志处理器
func (m *LoggingMiddleware) Handler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// TODO: 实现日志记录逻辑
		// 1. 记录请求开始时间
		// 2. 记录请求信息
		// 3. 包装响应写入器
		// 4. 处理请求
		// 5. 记录响应信息
		// 6. 计算处理时间
		
		startTime := time.Now()
		path := c.Request.URL.Path
		
		// 检查是否跳过日志
		if m.shouldSkipLogging(path) {
			c.Next()
			return
		}
		
		// 记录请求信息
		if m.config.EnableRequestLog {
			m.logRequest(c)
		}
		
		// 包装响应写入器以捕获响应体
		var responseBody *bytes.Buffer
		if m.config.EnableResponseLog && m.config.LogResponseBody {
			responseBody = &bytes.Buffer{}
			writer := &responseWriter{
				ResponseWriter: c.Writer,
				body:          responseBody,
			}
			c.Writer = writer
		}
		
		// 处理请求
		c.Next()
		
		// 计算处理时间
		duration := time.Since(startTime)
		
		// 记录响应信息
		if m.config.EnableResponseLog {
			m.logResponse(c, duration, responseBody)
		}
		
		// 检查慢请求
		if duration > m.config.SlowRequestThreshold {
			m.logSlowRequest(c, duration)
		}
	})
}

// shouldSkipLogging 检查是否应该跳过日志
func (m *LoggingMiddleware) shouldSkipLogging(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// logRequest 记录请求信息
func (m *LoggingMiddleware) logRequest(c *gin.Context) {
	fields := []interface{}{
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"query", c.Request.URL.RawQuery,
		"ip", utils.GetClientIP(c),
		"user_agent", c.GetHeader("User-Agent"),
		"content_type", c.GetHeader("Content-Type"),
		"content_length", c.Request.ContentLength,
	}
	
	// 记录请求头
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			fields = append(fields, "header_"+key, values[0])
		}
	}
	
	// 记录请求体
	if m.config.LogRequestBody && c.Request.ContentLength > 0 && c.Request.ContentLength <= m.config.MaxRequestBodySize {
		if body := m.readRequestBody(c); body != "" {
			fields = append(fields, "request_body", body)
		}
	}
	
	logger.Info("HTTP Request")
}

// logResponse 记录响应信息
func (m *LoggingMiddleware) logResponse(c *gin.Context, duration time.Duration, responseBody *bytes.Buffer) {
	fields := []interface{}{
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"ip", utils.GetClientIP(c),
		"status", c.Writer.Status(),
		"size", c.Writer.Size(),
		"duration", duration,
	}
	
	// 记录响应头
	for key, values := range c.Writer.Header() {
		if len(values) > 0 {
			fields = append(fields, "response_header_"+key, values[0])
		}
	}
	
	// 记录响应体
	if m.config.LogResponseBody && responseBody != nil {
		bodySize := int64(responseBody.Len())
		if bodySize > 0 && bodySize <= m.config.MaxResponseBodySize {
			fields = append(fields, "response_body", responseBody.String())
		}
	}
	
	// 根据状态码选择日志级别
	if c.Writer.Status() >= 500 {
		logger.Error("HTTP Response")
	} else if c.Writer.Status() >= 400 {
		logger.Warn("HTTP Response")
	} else {
		logger.Info("HTTP Response")
	}
}

// logSlowRequest 记录慢请求
func (m *LoggingMiddleware) logSlowRequest(c *gin.Context, duration time.Duration) {
	logger.Warn("Slow Request Detected")
}

// readRequestBody 读取请求体
func (m *LoggingMiddleware) readRequestBody(c *gin.Context) string {
	// TODO: 实现请求体读取逻辑
	// 1. 读取请求体
	// 2. 恢复请求体以供后续处理
	// 3. 处理大文件和二进制数据
	
	if c.Request.Body == nil {
		return ""
	}
	
	// 读取请求体
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("Failed to read request body")
		return ""
	}
	
	// 恢复请求体
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	
	// 检查是否为文本内容
	if m.isTextContent(c.GetHeader("Content-Type")) {
		return string(bodyBytes)
	}
	
	return "[binary data]"
}

// isTextContent 检查是否为文本内容
func (m *LoggingMiddleware) isTextContent(contentType string) bool {
	textTypes := []string{
		"application/json",
		"application/xml",
		"text/",
		"application/x-www-form-urlencoded",
	}
	
	for _, textType := range textTypes {
		if strings.Contains(contentType, textType) {
			return true
		}
	}
	
	return false
}

// UpdateConfig 更新日志配置
func (m *LoggingMiddleware) UpdateConfig(config *LoggingConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	m.config = config
	
	logger.Info("Logging middleware config updated")
	
	return nil
}

// GetConfig 获取当前配置
func (m *LoggingMiddleware) GetConfig() *LoggingConfig {
	return m.config
}