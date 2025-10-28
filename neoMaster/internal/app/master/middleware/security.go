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
	"fmt"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GinCORSMiddleware CORS跨域资源共享中间件
// 处理跨域请求，设置必要的CORS头部信息
func (m *MiddlewareManager) GinCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用CORS
		if !m.securityConfig.CORS.Enabled {
			c.Next()
			return
		}

		// 获取客户端IP和请求来源
		clientIP := utils.GetClientIP(c)
		origin := c.Request.Header.Get("Origin")

		// 记录日志
		logger.LogInfo("Processing CORS request", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"operation": "cors_middleware",
			"option":    "handle_cors_request",
			"func_name": "middleware.security.GinCORSMiddleware",
			"origin":    origin,
		})

		// 设置CORS头部
		m.setCORSHeaders(c, origin)

		// 处理预检请求（OPTIONS方法）
		if c.Request.Method == "OPTIONS" {
			logger.LogInfo("Handling CORS preflight request", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
				"operation": "cors_preflight",
				"option":    "handle_options_request",
				"func_name": "middleware.security.GinCORSMiddleware",
				"origin":    origin,
			})

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
		// 获取客户端IP
		clientIP := utils.GetClientIP(c)

		// 记录日志
		logger.LogInfo("Setting security headers", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"operation": "security_headers",
			"option":    "set_security_headers",
			"func_name": "middleware.security.GinSecurityHeadersMiddleware",
		})

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
		// 获取客户端IP
		clientIP := utils.GetClientIP(c)

		// 记录日志
		logger.LogInfo("Setting no-index headers", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"operation": "no_index",
			"option":    "set_no_index_headers",
			"func_name": "middleware.security.GinNoIndexMiddleware",
		})

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
		// 获取客户端IP
		clientIP := utils.GetClientIP(c)

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
		logger.LogInfo("Generated request ID", "", 0, clientIP, c.Request.URL.Path, c.Request.Method, map[string]interface{}{
			"operation":  "request_id",
			"option":     "generate_request_id",
			"func_name":  "middleware.security.GinRequestIDMiddleware",
			"request_id": requestID,
		})

		// 继续处理请求
		c.Next()
	}
}

// setCORSHeaders 设置CORS头部
func (m *MiddlewareManager) setCORSHeaders(c *gin.Context, origin string) {
	corsConfig := &m.securityConfig.CORS

	// 设置允许的源
	if corsConfig.AllowAllOrigins {
		c.Header("Access-Control-Allow-Origin", "*")
	} else if origin != "" && m.isOriginAllowed(origin) {
		c.Header("Access-Control-Allow-Origin", origin)
	}

	// 设置允许的方法
	if len(corsConfig.AllowMethods) > 0 {
		methods := strings.Join(corsConfig.AllowMethods, ", ")
		c.Header("Access-Control-Allow-Methods", methods)
	}

	// 设置允许的头部
	if len(corsConfig.AllowHeaders) > 0 {
		headers := strings.Join(corsConfig.AllowHeaders, ", ")
		c.Header("Access-Control-Allow-Headers", headers)
	}

	// 设置暴露的头部
	if len(corsConfig.ExposeHeaders) > 0 {
		exposeHeaders := strings.Join(corsConfig.ExposeHeaders, ", ")
		c.Header("Access-Control-Expose-Headers", exposeHeaders)
	}

	// 设置是否允许凭证
	if corsConfig.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	// 设置缓存时间
	if corsConfig.MaxAge > 0 {
		maxAge := int(corsConfig.MaxAge.Seconds())
		c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", maxAge))
	}
}

// isOriginAllowed 检查源是否被允许
func (m *MiddlewareManager) isOriginAllowed(origin string) bool {
	corsConfig := &m.securityConfig.CORS

	if corsConfig.AllowAllOrigins {
		return true
	}

	if origin == "" {
		return false
	}

	for _, allowedOrigin := range corsConfig.AllowOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}

	return false
}
