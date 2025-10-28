/**
 * IP白名单认证功能测试
 * @author: Linus Torvalds (AI Assistant)
 * @date: 2025.01.27
 * @description: 测试IP白名单认证中间件的各种场景
 * @test_cases:
 *   - TestIPWhitelistUtils: 测试IP白名单工具函数
 *   - TestAuthMiddlewareIPWhitelist: 测试IP白名单认证中间件
 *   - TestAuthMiddlewareSkipPaths: 测试跳过路径功能
 *   - TestAuthMiddlewareMultipleStrategies: 测试多种认证策略组合
 */
package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neomaster/internal/app/master/middleware"
	"neomaster/internal/config"
	"neomaster/internal/pkg/utils"
)

// TestIPWhitelistUtils 测试IP白名单工具函数
func TestIPWhitelistUtils(t *testing.T) {
	tests := []struct {
		name        string
		clientIP    string
		whitelist   []string
		expected    bool
		description string
	}{
		{
			name:        "Single IP Match",
			clientIP:    "192.168.1.100",
			whitelist:   []string{"192.168.1.100", "10.0.0.1"},
			expected:    true,
			description: "客户端IP在单IP白名单中",
		},
		{
			name:        "Single IP No Match",
			clientIP:    "192.168.1.101",
			whitelist:   []string{"192.168.1.100", "10.0.0.1"},
			expected:    false,
			description: "客户端IP不在单IP白名单中",
		},
		{
			name:        "CIDR Match",
			clientIP:    "192.168.1.50",
			whitelist:   []string{"192.168.1.0/24"},
			expected:    true,
			description: "客户端IP在CIDR范围内",
		},
		{
			name:        "CIDR No Match",
			clientIP:    "192.168.2.50",
			whitelist:   []string{"192.168.1.0/24"},
			expected:    false,
			description: "客户端IP不在CIDR范围内",
		},
		{
			name:        "Mixed Whitelist Match Single IP",
			clientIP:    "10.0.0.1",
			whitelist:   []string{"192.168.1.0/24", "10.0.0.1", "172.16.0.0/16"},
			expected:    true,
			description: "混合白名单中匹配单IP",
		},
		{
			name:        "Mixed Whitelist Match CIDR",
			clientIP:    "172.16.5.10",
			whitelist:   []string{"192.168.1.0/24", "10.0.0.1", "172.16.0.0/16"},
			expected:    true,
			description: "混合白名单中匹配CIDR",
		},
		{
			name:        "Empty Whitelist",
			clientIP:    "192.168.1.100",
			whitelist:   []string{},
			expected:    false,
			description: "空白名单拒绝所有IP",
		},
		{
			name:        "IPv6 Match",
			clientIP:    "::1",
			whitelist:   []string{"::1", "192.168.1.0/24"},
			expected:    true,
			description: "IPv6地址匹配",
		},
		{
			name:        "IPv6 CIDR Match",
			clientIP:    "2001:db8::1",
			whitelist:   []string{"2001:db8::/32"},
			expected:    true,
			description: "IPv6 CIDR匹配",
		},
		{
			name:        "Invalid Client IP",
			clientIP:    "invalid-ip",
			whitelist:   []string{"192.168.1.0/24"},
			expected:    false,
			description: "无效客户端IP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsIPInWhitelist(tt.clientIP, tt.whitelist)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestValidateIPWhitelistConfig 测试IP白名单配置验证
func TestValidateIPWhitelistConfig(t *testing.T) {
	tests := []struct {
		name        string
		whitelist   []string
		expectError bool
		description string
	}{
		{
			name:        "Valid Single IPs",
			whitelist:   []string{"192.168.1.1", "10.0.0.1", "::1"},
			expectError: false,
			description: "有效的单IP列表",
		},
		{
			name:        "Valid CIDR",
			whitelist:   []string{"192.168.1.0/24", "10.0.0.0/8", "2001:db8::/32"},
			expectError: false,
			description: "有效的CIDR列表",
		},
		{
			name:        "Mixed Valid",
			whitelist:   []string{"192.168.1.1", "10.0.0.0/8", "::1"},
			expectError: false,
			description: "混合有效格式",
		},
		{
			name:        "Invalid IP",
			whitelist:   []string{"192.168.1.1", "invalid-ip"},
			expectError: true,
			description: "包含无效IP",
		},
		{
			name:        "Invalid CIDR",
			whitelist:   []string{"192.168.1.0/24", "192.168.1.0/99"},
			expectError: true,
			description: "包含无效CIDR",
		},
		{
			name:        "Empty List",
			whitelist:   []string{},
			expectError: false,
			description: "空列表有效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateIPWhitelistConfig(tt.whitelist)
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestAuthMiddlewareIPWhitelist 测试IP白名单认证中间件
func TestAuthMiddlewareIPWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		authConfig     *config.AuthConfig
		clientIP       string
		expectedStatus int
		description    string
	}{
		{
			name: "IP Whitelist Enabled - IP in Whitelist",
			authConfig: &config.AuthConfig{
				AuthMethod:        "jwt",
				EnableIPWhitelist: true,
				WhitelistIPs:      []string{"192.168.1.100", "10.0.0.0/8"},
				APIKey:            "test-secret",
				APIKeyHeader:      "X-API-Key",
			},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
			description:    "IP在白名单中，应该通过认证",
		},
		{
			name: "IP Whitelist Enabled - IP Not in Whitelist",
			authConfig: &config.AuthConfig{
				AuthMethod:        "jwt",
				EnableIPWhitelist: true,
				WhitelistIPs:      []string{"192.168.1.100", "10.0.0.0/8"},
				APIKey:            "test-secret",
				APIKeyHeader:      "X-API-Key",
			},
			clientIP:       "192.168.2.100",
			expectedStatus: http.StatusUnauthorized,
			description:    "IP不在白名单中，应该进行JWT认证但失败",
		},
		{
			name: "IP Whitelist Disabled",
			authConfig: &config.AuthConfig{
				AuthMethod:        "jwt",
				EnableIPWhitelist: false,
				WhitelistIPs:      []string{"192.168.1.100"},
				APIKey:            "test-secret",
				APIKeyHeader:      "X-API-Key",
			},
			clientIP:       "192.168.2.100",
			expectedStatus: http.StatusUnauthorized,
			description:    "IP白名单未启用，应该进行JWT认证但失败",
		},
		{
			name: "CIDR Range Match",
			authConfig: &config.AuthConfig{
				AuthMethod:        "jwt",
				EnableIPWhitelist: true,
				WhitelistIPs:      []string{"192.168.0.0/16"},
				APIKey:            "test-secret",
				APIKeyHeader:      "X-API-Key",
			},
			clientIP:       "192.168.5.10",
			expectedStatus: http.StatusOK,
			description:    "IP在CIDR范围内，应该通过认证",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试路由
			router := gin.New()
			router.Use(middleware.GinAuthMiddleware(tt.authConfig))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// 创建测试请求
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", tt.clientIP)

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
		})
	}
}

// TestAuthMiddlewareSkipPaths 测试跳过路径功能
func TestAuthMiddlewareSkipPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authConfig := &config.AuthConfig{
		AuthMethod:        "jwt",
		EnableIPWhitelist: false,
		SkipPaths:         []string{"/health", "/api/v1/public"},
		APIKey:            "test-secret",
		APIKeyHeader:      "X-API-Key",
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "Skip Path - Health Check",
			path:           "/health",
			expectedStatus: http.StatusOK,
			description:    "健康检查路径应该跳过认证",
		},
		{
			name:           "Skip Path - Public API",
			path:           "/api/v1/public/info",
			expectedStatus: http.StatusOK,
			description:    "公共API路径应该跳过认证",
		},
		{
			name:           "Protected Path",
			path:           "/api/v1/private/data",
			expectedStatus: http.StatusUnauthorized,
			description:    "受保护路径应该需要认证",
		},
		{
			name:           "Root Path",
			path:           "/",
			expectedStatus: http.StatusUnauthorized,
			description:    "根路径应该需要认证",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试路由
			router := gin.New()
			router.Use(middleware.GinAuthMiddleware(authConfig))

			// 添加所有测试路径
			router.GET("/health", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
			router.GET("/api/v1/public/info", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"info": "public"})
			})
			router.GET("/api/v1/private/data", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"data": "private"})
			})
			router.GET("/", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "root"})
			})

			// 创建测试请求
			req := httptest.NewRequest("GET", tt.path, nil)

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
		})
	}
}

// TestAuthMiddlewareAPIKey 测试API Key认证
func TestAuthMiddlewareAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authConfig := &config.AuthConfig{
		AuthMethod:   "api_key",
		APIKey:       "valid-api-key",
		APIKeyHeader: "X-API-Key",
	}

	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
		description    string
	}{
		{
			name:           "Valid API Key",
			apiKey:         "valid-api-key",
			expectedStatus: http.StatusOK,
			description:    "有效的API Key应该通过认证",
		},
		{
			name:           "Invalid API Key",
			apiKey:         "invalid-api-key",
			expectedStatus: http.StatusUnauthorized,
			description:    "无效的API Key应该被拒绝",
		},
		{
			name:           "Missing API Key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
			description:    "缺少API Key应该被拒绝",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试路由
			router := gin.New()
			router.Use(middleware.GinAuthMiddleware(authConfig))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// 创建测试请求
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
		})
	}
}

// TestAuthMiddlewareNoAuth 测试无认证模式
func TestAuthMiddlewareNoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authConfig := &config.AuthConfig{
		AuthMethod: "none",
	}

	// 创建测试路由
	router := gin.New()
	router.Use(middleware.GinAuthMiddleware(authConfig))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 创建测试请求
	req := httptest.NewRequest("GET", "/test", nil)

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code, "无认证模式应该允许所有请求")
}

// TestAuthMiddlewareMultipleStrategies 测试多种认证策略组合
func TestAuthMiddlewareMultipleStrategies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authConfig := &config.AuthConfig{
		AuthMethod:        "api_key",
		EnableIPWhitelist: true,
		WhitelistIPs:      []string{"192.168.1.100"},
		SkipPaths:         []string{"/health"},
		APIKey:            "valid-api-key",
		APIKeyHeader:      "X-API-Key",
	}

	tests := []struct {
		name           string
		path           string
		clientIP       string
		apiKey         string
		expectedStatus int
		description    string
	}{
		{
			name:           "Skip Path Priority",
			path:           "/health",
			clientIP:       "192.168.2.100", // 不在白名单
			apiKey:         "",              // 无API Key
			expectedStatus: http.StatusOK,
			description:    "跳过路径优先级最高，应该直接通过",
		},
		{
			name:           "IP Whitelist Priority",
			path:           "/test",
			clientIP:       "192.168.1.100", // 在白名单
			apiKey:         "",              // 无API Key
			expectedStatus: http.StatusOK,
			description:    "IP白名单优先级高于API Key认证",
		},
		{
			name:           "API Key Fallback",
			path:           "/test",
			clientIP:       "192.168.2.100", // 不在白名单
			apiKey:         "valid-api-key", // 有效API Key
			expectedStatus: http.StatusOK,
			description:    "IP不在白名单时，应该使用API Key认证",
		},
		{
			name:           "All Auth Failed",
			path:           "/test",
			clientIP:       "192.168.2.100",   // 不在白名单
			apiKey:         "invalid-api-key", // 无效API Key
			expectedStatus: http.StatusUnauthorized,
			description:    "所有认证方式都失败时应该被拒绝",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试路由
			router := gin.New()
			router.Use(middleware.GinAuthMiddleware(authConfig))
			router.GET("/health", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// 创建测试请求
			req := httptest.NewRequest("GET", tt.path, nil)
			req.Header.Set("X-Forwarded-For", tt.clientIP)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
		})
	}
}
