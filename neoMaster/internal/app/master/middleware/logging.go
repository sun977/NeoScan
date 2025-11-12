/**
 * 中间件:日志相关中间件
 * @author: sun977
 * @date: 2025.10.10
 * @description: 定义日志中间件
 * @func:
 *   - GinLoggingMiddleware Gin日志中间件[同时把客户端IP存储到Gin上下文和标准上下文,供后续使用]
 */
package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GinLoggingMiddleware Gin日志中间件
// 记录所有HTTP请求的访问日志和错误日志
// 使用方式: router.Use(middlewareManager.GinLoggingMiddleware())
func (m *MiddlewareManager) GinLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用请求日志
		if !m.securityConfig.Logging.EnableRequestLog {
			c.Next()
			return
		}

		// 检查是否跳过此路径
		if m.shouldSkipLogging(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// 提取并格式化客户端IP
		clientIP := utils.GetClientIP(c)
		XRequestID := c.GetHeader("X-Request-ID")
		userAgent := c.GetHeader("User-Agent")

		// 存储到Gin上下文
		c.Set("client_ip", clientIP) // 这个是标准化后的可以用作业务使用的客户端IP
		// Gin上下文通过c.Set()方式存储值，后续可以通过c.Get("xx_key")获取

		// // 存储到标准上下文
		// ctx := c.Request.Context()
		// type clientIPKeyType struct{}
		// // 定义一个常量作为上下文键,避免使用空的匿名结构体
		// var clientIPKey = clientIPKeyType{}
		// ctx = context.WithValue(ctx, clientIPKey, clientIP)
		// // c.Request.Context()返回是标准的context.Context上下文，不包含gin的上下文
		// // 可以使用WithValue方法将自定义的上下文值存储到标准上下文中
		// // 这样后续使用标准上下文为参数的函数就可以安全获取自定义的上下文值
		// // 获取方式：clientIP, _ := ctx.Value("client_ip").(string)
		// c.Request = c.Request.WithContext(ctx)

		// 存储到标准上下文（双写：兼容旧键 + 推进统一键）
		// 1) 统一键：跨包一致读取（推荐）
		// 2) 旧键：维持现有代码路径（局部匿名类型，短期兼容）
		ctx := c.Request.Context()
		// 统一键写入（utils.ContextKeyClientIP）
		ctx = context.WithValue(ctx, utils.ContextKeyClientIP, clientIP)
		// 旧键写入（局部匿名类型），供仍旧使用 clientIPKeyType{} 的代码读取
		type clientIPKeyType struct{}
		var clientIPKey = clientIPKeyType{}
		ctx = context.WithValue(ctx, clientIPKey, clientIP)
		// 关联到请求对象
		c.Request = c.Request.WithContext(ctx)
		// 本项目只有handler中使用了Gin上下文，剩下的逻辑都在service中使用的标准上下文
		// 所以这里需要将Gin上下文的client_ip也存储到标准上下文

		// 记录请求体（如果配置启用）
		var requestBody string
		if m.securityConfig.Logging.LogRequestBody && c.Request.ContentLength > 0 && c.Request.ContentLength <= int64(m.securityConfig.Logging.MaxRequestBodySize) {
			requestBody = m.readRequestBody(c)
		}

		// 处理请求
		c.Next()

		// 记录访问日志
		duration := time.Since(start)
		// statusCode := c.Writer.Status()

		// 检查是否为慢请求
		isSlowRequest := duration > m.securityConfig.Logging.SlowRequestThreshold

		// 获取用户信息（如果已认证）
		userID := ""
		username := ""
		if uid, exists := c.Get("user_id"); exists {
			if uidUint, ok := uid.(uint); ok {
				userID = fmt.Sprintf("%d", uidUint)
			}
		}
		if uname, exists := c.Get("username"); exists {
			if unameStr, ok := uname.(string); ok {
				username = unameStr
			}
		}

		// 构建日志数据
		logData := map[string]interface{}{
			"operation":     "http_request",
			"duration":      duration.Milliseconds(),
			"client_ip":     clientIP,
			"username":      username,
			"user_agent":    userAgent,
			"X-Request-ID":  XRequestID,
			"referer":       c.Request.Referer(),
			"request_size":  c.Request.ContentLength,
			"response_size": int64(c.Writer.Size()),
			"timestamp":     logger.NowFormatted(),
			"is_slow":       isSlowRequest,
		}

		// 添加请求体到日志（如果启用）
		if requestBody != "" {
			logData["request_body"] = requestBody
		}

		// 记录响应体（如果配置启用且响应不太大）
		if m.securityConfig.Logging.LogResponseBody && m.securityConfig.Logging.EnableResponseLog {
			if responseBody := m.getResponseBody(c); responseBody != "" {
				logData["response_body"] = responseBody
			}
		}

		// 使用日志格式化器记录API请求
		userIDUint := uint(0)
		if userID != "" {
			if id, err := strconv.ParseUint(userID, 10, 32); err == nil {
				userIDUint = uint(id)
			}
		}

		// 添加错误信息到日志数据（如果存在错误）
		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			errorMsg := getErrorMessage(statusCode, c.Errors)
			logData["error_message"] = errorMsg
		}

		// 记录访问日志
		logger.LogAccessRequest(c, start, XRequestID, userIDUint, logData)
	}
}

// shouldSkipLogging 检查是否应该跳过日志记录
func (m *MiddlewareManager) shouldSkipLogging(path string) bool {
	for _, skipPath := range m.securityConfig.Logging.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// readRequestBody 读取请求体内容
func (m *MiddlewareManager) readRequestBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	// 读取请求体
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}

	// 恢复请求体，以便后续处理器可以读取
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// 检查大小限制
	if len(bodyBytes) > m.securityConfig.Logging.MaxRequestBodySize {
		return fmt.Sprintf("[Request body too large: %d bytes]", len(bodyBytes))
	}

	return string(bodyBytes)
}

// getResponseBody 获取响应体内容
func (m *MiddlewareManager) getResponseBody(c *gin.Context) string {
	// 注意：这个方法需要配合响应写入器包装器使用
	// 由于当前实现的复杂性，这里返回空字符串
	// 如果需要记录响应体，需要在中间件开始时包装ResponseWriter
	return ""
}

// getErrorMessage 获取错误消息
func getErrorMessage(statusCode int, errors []*gin.Error) string {
	if len(errors) > 0 {
		var messages []string
		for _, err := range errors {
			messages = append(messages, err.Error())
		}
		return strings.Join(messages, "; ")
	}

	// 如果没有详细错误信息，则根据状态码提供默认错误描述
	switch statusCode {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	default:
		return http.StatusText(statusCode)
	}
}
