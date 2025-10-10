/**
 * 中间件:安全中间件
 * @author: sun977
 * @date: 2025.10.10
 * @description: 定义安全中间件
 * @func:
 *   - GinCORSMiddleware CORS跨域资源共享中间件,处理跨域请求，设置必要的CORS头部信息
 *   - GinSecurityHeadersMiddleware 安全头部中间件,设置必要的安全头部信息，防止常见的安全漏洞
 *   - GinNoIndexMiddleware 禁用索引中间件,防止搜索引擎索引网站内容
 *   - GinRequestIDMiddleware 请求ID中间件,为每个请求添加唯一的请求ID,方便日志跟踪和调试
 */
package middleware

import (
	"neomaster/internal/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GinCORSMiddleware CORS跨域资源共享中间件
// 处理跨域请求，设置必要的CORS头部信息
func (m *MiddlewareManager) GinCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录日志
		logrus.WithFields(logrus.Fields{
			"path":      c.Request.URL.Path,
			"operation": "cors_middleware",
			"option":    "handle_cors_request",
			"func_name": "middleware.security.GinCORSMiddleware",
			"method":    c.Request.Method,
			"origin":    c.Request.Header.Get("Origin"),
		}).Debug("Processing CORS request")

		// 获取请求来源
		origin := c.Request.Header.Get("Origin")

		// 设置CORS头部
		// 允许的来源（生产环境应该配置具体的域名）
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// 允许的HTTP方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		// 允许的请求头
		c.Header("Access-Control-Allow-Headers",
			"Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")

		// 允许发送凭据（cookies等）
		c.Header("Access-Control-Allow-Credentials", "true")

		// 预检请求的缓存时间（秒）
		c.Header("Access-Control-Max-Age", "86400")

		// 允许客户端访问的响应头
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")

		// 处理预检请求（OPTIONS方法）
		if c.Request.Method == "OPTIONS" {
			logrus.WithFields(logrus.Fields{
				"path":      c.Request.URL.Path,
				"operation": "cors_preflight",
				"option":    "handle_options_request",
				"func_name": "middleware.security.GinCORSMiddleware",
				"origin":    origin,
			}).Debug("Handling CORS preflight request")

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// GinSecurityHeadersMiddleware 安全头中间件
// 添加各种安全相关的HTTP头部，提高应用安全性
func (m *MiddlewareManager) GinSecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录日志
		logrus.WithFields(logrus.Fields{
			"path":      c.Request.URL.Path,
			"operation": "security_headers",
			"option":    "set_security_headers",
			"func_name": "middleware.security.GinSecurityHeadersMiddleware",
		}).Debug("Setting security headers")

		// X-Content-Type-Options: 防止MIME类型嗅探攻击
		c.Header("X-Content-Type-Options", "nosniff")

		// X-Frame-Options: 防止点击劫持攻击
		c.Header("X-Frame-Options", "DENY")

		// X-XSS-Protection: 启用浏览器XSS过滤器
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy: 控制Referer头的发送策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content-Security-Policy: 内容安全策略（根据实际需求调整）
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none';"
		c.Header("Content-Security-Policy", csp)

		// Strict-Transport-Security: 强制HTTPS（仅在HTTPS环境下设置）
		if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Permissions-Policy: 控制浏览器功能权限
		c.Header("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=(), usb=(), magnetometer=(), gyroscope=()")

		// Server: 隐藏服务器信息
		c.Header("Server", "NeoScan-Master")

		// 继续处理请求
		c.Next()
	}
}

// GinNoIndexMiddleware 防止搜索引擎索引中间件
// 添加robots标签，防止敏感页面被搜索引擎索引
func (m *MiddlewareManager) GinNoIndexMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录日志
		logrus.WithFields(logrus.Fields{
			"path":      c.Request.URL.Path,
			"operation": "no_index",
			"option":    "set_no_index_headers",
			"func_name": "middleware.security.GinNoIndexMiddleware",
		}).Debug("Setting no-index headers")

		// 防止搜索引擎索引
		c.Header("X-Robots-Tag", "noindex, nofollow, nosnippet, noarchive")

		// 继续处理请求
		c.Next()
	}
}

// GinRequestIDMiddleware 请求ID中间件
// 为每个请求生成唯一ID，便于日志追踪和问题排查
func (m *MiddlewareManager) GinRequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否已有请求ID（可能来自负载均衡器或代理）
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 生成新的请求ID（使用简单的时间戳+随机数方案） 550e8400-e29b-41d4-a716-446655440000
			requestID, _ = utils.GenerateUUID()
		}

		// 设置请求ID到上下文中
		c.Set("request_id", requestID)

		// 设置响应头
		c.Header("X-Request-ID", requestID)

		// 记录日志
		logrus.WithFields(logrus.Fields{
			"path":       c.Request.URL.Path,
			"operation":  "request_id",
			"option":     "generate_request_id",
			"func_name":  "middleware.security.GinRequestIDMiddleware",
			"request_id": requestID,
		}).Debug("Generated request ID")

		// 继续处理请求
		c.Next()
	}
}

// // generateRequestID 生成请求ID
// // 使用时间戳和随机数生成唯一的请求标识符
// func generateRequestID() string {
// 	// 简单的请求ID生成策略：时间戳(纳秒) + 随机后缀
// 	// 生产环境建议使用更强的UUID生成算法
// 	requestID, _ := utils.GenerateUUID()
// 	return requestID
// }
