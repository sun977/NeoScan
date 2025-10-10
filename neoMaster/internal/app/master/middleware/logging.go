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
	"context"
	"fmt"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GinLoggingMiddleware Gin日志中间件
// 记录所有HTTP请求的访问日志和错误日志
// 使用方式: router.Use(middlewareManager.GinLoggingMiddleware())
func (m *MiddlewareManager) GinLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 提取并格式化客户端IP
		clientIP := utils.GetClientIP(c)
		XRequestID := c.GetHeader("X-Request-ID")
		userAgent := c.GetHeader("User-Agent")

		// 存储到Gin上下文
		c.Set("client_ip", clientIP) // 这个是标准化后的可以用作业务使用的客户端IP
		// Gin上下文通过c.Set()方式存储值，后续可以通过c.Get("xx_key")获取

		// 存储到标准上下文
		ctx := c.Request.Context()
		type clientIPKeyType struct{}
		// 定义一个常量作为上下文键,避免使用空的匿名结构体
		var clientIPKey = clientIPKeyType{}
		ctx = context.WithValue(ctx, clientIPKey, clientIP)
		// c.Request.Context()返回是标准的context.Context上下文，不包含gin的上下文
		// 可以使用WithValue方法将自定义的上下文值存储到标准上下文中
		// 这样后续使用标准上下文为参数的函数就可以安全获取自定义的上下文值
		// 获取方式：clientIP, _ := ctx.Value("client_ip").(string)
		c.Request = c.Request.WithContext(ctx)
		// 本项目只有handler中使用了Gin上下文，剩下的逻辑都在service中使用的标准上下文
		// 所以这里需要将Gin上下文的client_ip也存储到标准上下文

		// 处理请求
		c.Next()

		// 记录访问日志
		duration := time.Since(start)
		statusCode := c.Writer.Status()

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

		// 使用日志格式化器记录API请求
		userIDUint := uint(0)
		if userID != "" {
			if id, err := strconv.ParseUint(userID, 10, 32); err == nil {
				userIDUint = uint(id)
			}
		}
		logger.LogBusinessOperation("http_request", userIDUint, username, clientIP, XRequestID, "success", "API Request", map[string]interface{}{
			"operation":     "http_request",
			"method":        c.Request.Method,
			"url":           c.Request.URL.String(),
			"status_code":   statusCode,
			"duration":      duration.Milliseconds(),
			"client_ip":     clientIP,
			"username":      username,
			"user_agent":    userAgent,
			"X-Request-ID":  XRequestID,
			"referer":       c.Request.Referer(),
			"request_size":  c.Request.ContentLength,
			"response_size": int64(c.Writer.Size()),
			"timestamp":     logger.NowFormatted(),
		})

		// 如果是错误状态码，记录错误日志
		if statusCode >= 400 {
			errorMsg := ""
			if errors := c.Errors; len(errors) > 0 {
				errorMsg = errors.String()
			} else {
				// 如果没有详细错误信息，则根据状态码提供默认错误描述
				switch statusCode {
				case 400:
					errorMsg = "Bad Request"
				case 401:
					errorMsg = "Unauthorized"
				case 403:
					errorMsg = "Forbidden"
				case 404:
					errorMsg = "Not Found"
				case 405:
					errorMsg = "Method Not Allowed"
				case 500:
					errorMsg = "Internal Server Error"
				case 502:
					errorMsg = "Bad Gateway"
				case 503:
					errorMsg = "Service Unavailable"
				default:
					errorMsg = http.StatusText(statusCode)
				}
			}

			logger.LogError(fmt.Errorf("HTTP %d: %s", statusCode, errorMsg), XRequestID, userIDUint, clientIP, "http_request", c.Request.Method, map[string]interface{}{
				"operation":    "http_request",
				"method":       c.Request.Method,
				"url":          c.Request.URL.String(),
				"status_code":  statusCode,
				"username":     username,
				"client_ip":    clientIP,
				"user_agent":   userAgent,
				"X-Request-ID": XRequestID,
				"timestamp":    logger.NowFormatted(),
			})
		}
	}
}
