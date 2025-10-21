/**
 * 认证中间件
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端认证中间件，用于验证Master端的请求权限和身份
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
	"neoagent/internal/pkg/utils"
)

// AuthConfig 认证配置
type AuthConfig struct {
	// API Key认证
	APIKey       string        `json:"api_key"`
	APIKeyHeader string        `json:"api_key_header"`
	
	// JWT认证
	JWTSecret    string        `json:"jwt_secret"`
	JWTExpiry    time.Duration `json:"jwt_expiry"`
	
	// 白名单IP
	WhitelistIPs []string      `json:"whitelist_ips"`
	
	// 认证方式
	AuthMethod   string        `json:"auth_method"` // "api_key", "jwt", "both"
	
	// 跳过认证的路径
	SkipPaths    []string      `json:"skip_paths"`
}

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	config *AuthConfig
	logger *logger.LoggerManager
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(config *AuthConfig) *AuthMiddleware {
	if config == nil {
		config = &AuthConfig{
			APIKeyHeader: "X-API-Key",
			AuthMethod:   "api_key",
			SkipPaths: []string{
				"/health",
				"/ping",
				"/metrics",
			},
		}
	}
	
	return &AuthMiddleware{
		config: config,
		logger: logger.LoggerInstance,
	}
}

// Handler 认证处理器
func (m *AuthMiddleware) Handler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// TODO: 实现认证逻辑
		// 1. 检查是否需要跳过认证
		// 2. 验证IP白名单
		// 3. 根据配置进行API Key或JWT认证
		// 4. 记录认证日志
		
		path := c.Request.URL.Path
		
		// 检查是否跳过认证
		if m.shouldSkipAuth(path) {
			c.Next()
			return
		}
		
		// 验证IP白名单
		if !m.validateIPWhitelist(utils.GetClientIP(c)) {
			logger.Warn("IP not in whitelist")
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "IP not allowed",
			})
			c.Abort()
			return
		}
		
		// 根据认证方式进行验证
		var authenticated bool
		var authError string
		
		switch m.config.AuthMethod {
		case "api_key":
			authenticated, authError = m.validateAPIKey(c)
		case "jwt":
			authenticated, authError = m.validateJWT(c)
		case "both":
			// 两种方式都支持，任一通过即可
			apiKeyAuth, _ := m.validateAPIKey(c)
			jwtAuth, _ := m.validateJWT(c)
			authenticated = apiKeyAuth || jwtAuth
			if !authenticated {
				authError = "invalid api key or jwt token"
			}
		default:
			authenticated = false
			authError = "unsupported auth method"
		}
		
		if !authenticated {
			logger.Warn("Authentication failed")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": authError,
			})
			c.Abort()
			return
		}
		
		// 认证成功，继续处理
		logger.Debug("Authentication successful")
		c.Next()
	})
}

// shouldSkipAuth 检查是否应该跳过认证
func (m *AuthMiddleware) shouldSkipAuth(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// validateIPWhitelist 验证IP白名单
func (m *AuthMiddleware) validateIPWhitelist(clientIP string) bool {
	// TODO: 实现IP白名单验证逻辑
	// 1. 如果白名单为空，允许所有IP
	// 2. 支持CIDR格式的IP范围
	// 3. 支持单个IP地址
	
	if len(m.config.WhitelistIPs) == 0 {
		return true // 没有配置白名单，允许所有IP
	}
	
	for _, allowedIP := range m.config.WhitelistIPs {
		if clientIP == allowedIP {
			return true
		}
		
		// TODO: 支持CIDR格式验证
		// 例如: 192.168.1.0/24
	}
	
	return false
}

// validateAPIKey 验证API Key
func (m *AuthMiddleware) validateAPIKey(c *gin.Context) (bool, string) {
	// TODO: 实现API Key验证逻辑
	// 1. 从Header中获取API Key
	// 2. 验证API Key是否有效
	// 3. 可以支持多个API Key
	
	if m.config.APIKey == "" {
		return false, "api key not configured"
	}
	
	apiKey := c.GetHeader(m.config.APIKeyHeader)
	if apiKey == "" {
		return false, "missing api key"
	}
	
	if apiKey != m.config.APIKey {
		return false, "invalid api key"
	}
	
	return true, ""
}

// validateJWT 验证JWT Token
func (m *AuthMiddleware) validateJWT(c *gin.Context) (bool, string) {
	// TODO: 实现JWT验证逻辑
	// 1. 从Header中获取JWT Token
	// 2. 验证JWT签名
	// 3. 检查JWT是否过期
	// 4. 提取用户信息
	
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false, "missing authorization header"
	}
	
	// 检查Bearer格式
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false, "invalid authorization header format"
	}
	
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return false, "missing jwt token"
	}
	
	// TODO: 使用JWT库验证token
	// 这里是占位符实现
	if m.config.JWTSecret == "" {
		return false, "jwt secret not configured"
	}
	
	// 占位符验证逻辑
	if len(token) < 10 {
		return false, "invalid jwt token"
	}
	
	return true, ""
}

// UpdateConfig 更新认证配置
func (m *AuthMiddleware) UpdateConfig(config *AuthConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	m.config = config
	
	logger.Info("Auth middleware config updated")
	
	return nil
}

// GetConfig 获取当前配置
func (m *AuthMiddleware) GetConfig() *AuthConfig {
	return m.config
}