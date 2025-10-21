/**
 * CORS中间件
 * @author: sun977
 * @date: 2025.01.21
 * @description: Agent端CORS中间件，用于处理跨域请求
 * @func: 占位符实现，待后续完善
 */
package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"neoagent/internal/pkg/logger"
)

// CORSConfig CORS配置
type CORSConfig struct {
	// 允许的源
	AllowOrigins []string `json:"allow_origins"`
	
	// 允许的方法
	AllowMethods []string `json:"allow_methods"`
	
	// 允许的头部
	AllowHeaders []string `json:"allow_headers"`
	
	// 暴露的头部
	ExposeHeaders []string `json:"expose_headers"`
	
	// 是否允许凭证
	AllowCredentials bool `json:"allow_credentials"`
	
	// 预检请求缓存时间
	MaxAge time.Duration `json:"max_age"`
	
	// 是否允许所有源
	AllowAllOrigins bool `json:"allow_all_origins"`
	
	// 是否启用CORS
	Enabled bool `json:"enabled"`
}

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	config *CORSConfig
	logger *logger.LoggerManager
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(config *CORSConfig) *CORSMiddleware {
	if config == nil {
		config = &CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
				http.MethodHead,
			},
			AllowHeaders: []string{
				"Origin",
				"Content-Length",
				"Content-Type",
				"Authorization",
				"X-Requested-With",
				"X-API-Key",
				"X-Agent-ID",
				"X-Request-ID",
			},
			ExposeHeaders: []string{
				"Content-Length",
				"X-Request-ID",
				"X-Response-Time",
			},
			AllowCredentials: false,
			MaxAge:          12 * time.Hour,
			AllowAllOrigins: true,
			Enabled:         true,
		}
	}
	
	return &CORSMiddleware{
		config: config,
		logger: logger.LoggerInstance,
	}
}

// Handler CORS处理器
func (m *CORSMiddleware) Handler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// TODO: 实现CORS处理逻辑
		// 1. 检查是否启用CORS
		// 2. 处理预检请求
		// 3. 设置CORS头部
		// 4. 验证源
		
		if !m.config.Enabled {
			c.Next()
			return
		}
		
		origin := c.GetHeader("Origin")
		
		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			m.handlePreflightRequest(c, origin)
			return
		}
		
		// 设置CORS头部
		m.setCORSHeaders(c, origin)
		
		c.Next()
	})
}

// handlePreflightRequest 处理预检请求
func (m *CORSMiddleware) handlePreflightRequest(c *gin.Context, origin string) {
	// 验证源
	if !m.isOriginAllowed(origin) {
		logger.Warn("CORS preflight request denied")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	
	// 验证请求方法
	requestMethod := c.GetHeader("Access-Control-Request-Method")
	if !m.isMethodAllowed(requestMethod) {
		logger.Warn("CORS preflight method not allowed")
		c.AbortWithStatus(http.StatusMethodNotAllowed)
		return
	}
	
	// 验证请求头部
	requestHeaders := c.GetHeader("Access-Control-Request-Headers")
	if !m.areHeadersAllowed(requestHeaders) {
		logger.Warn("CORS preflight headers not allowed")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	
	// 设置预检响应头部
	m.setPreflightHeaders(c, origin)
	
	logger.Debug("CORS preflight request allowed")
	
	c.AbortWithStatus(http.StatusNoContent)
}

// setCORSHeaders 设置CORS头部
func (m *CORSMiddleware) setCORSHeaders(c *gin.Context, origin string) {
	// 设置允许的源
	if m.config.AllowAllOrigins {
		c.Header("Access-Control-Allow-Origin", "*")
	} else if m.isOriginAllowed(origin) {
		c.Header("Access-Control-Allow-Origin", origin)
	}
	
	// 设置允许的方法
	if len(m.config.AllowMethods) > 0 {
		c.Header("Access-Control-Allow-Methods", strings.Join(m.config.AllowMethods, ", "))
	}
	
	// 设置允许的头部
	if len(m.config.AllowHeaders) > 0 {
		c.Header("Access-Control-Allow-Headers", strings.Join(m.config.AllowHeaders, ", "))
	}
	
	// 设置暴露的头部
	if len(m.config.ExposeHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(m.config.ExposeHeaders, ", "))
	}
	
	// 设置是否允许凭证
	if m.config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}
}

// setPreflightHeaders 设置预检响应头部
func (m *CORSMiddleware) setPreflightHeaders(c *gin.Context, origin string) {
	// 设置基本CORS头部
	m.setCORSHeaders(c, origin)
	
	// 设置缓存时间
	if m.config.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", fmt.Sprintf("%.0f", m.config.MaxAge.Seconds()))
	}
}

// isOriginAllowed 检查源是否被允许
func (m *CORSMiddleware) isOriginAllowed(origin string) bool {
	if m.config.AllowAllOrigins {
		return true
	}
	
	if origin == "" {
		return false
	}
	
	for _, allowedOrigin := range m.config.AllowOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
		
		// 支持通配符匹配
		if m.matchWildcard(allowedOrigin, origin) {
			return true
		}
	}
	
	return false
}

// isMethodAllowed 检查方法是否被允许
func (m *CORSMiddleware) isMethodAllowed(method string) bool {
	if method == "" {
		return false
	}
	
	for _, allowedMethod := range m.config.AllowMethods {
		if allowedMethod == method {
			return true
		}
	}
	
	return false
}

// areHeadersAllowed 检查头部是否被允许
func (m *CORSMiddleware) areHeadersAllowed(headers string) bool {
	if headers == "" {
		return true
	}
	
	requestHeaders := strings.Split(headers, ",")
	for _, header := range requestHeaders {
		header = strings.TrimSpace(header)
		if !m.isHeaderAllowed(header) {
			return false
		}
	}
	
	return true
}

// isHeaderAllowed 检查单个头部是否被允许
func (m *CORSMiddleware) isHeaderAllowed(header string) bool {
	header = strings.ToLower(strings.TrimSpace(header))
	
	for _, allowedHeader := range m.config.AllowHeaders {
		if strings.ToLower(allowedHeader) == header {
			return true
		}
	}
	
	return false
}

// matchWildcard 通配符匹配
func (m *CORSMiddleware) matchWildcard(pattern, str string) bool {
	// TODO: 实现更复杂的通配符匹配逻辑
	// 简单实现：只支持 *.domain.com 格式
	
	if !strings.Contains(pattern, "*") {
		return pattern == str
	}
	
	// 处理 *.domain.com 格式
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(str, suffix)
	}
	
	return false
}

// UpdateConfig 更新CORS配置
func (m *CORSMiddleware) UpdateConfig(config *CORSConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	m.config = config
	
	logger.Info("CORS middleware config updated")
	
	return nil
}

// GetConfig 获取当前配置
func (m *CORSMiddleware) GetConfig() *CORSConfig {
	return m.config
}