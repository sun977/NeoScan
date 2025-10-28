package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"neomaster/internal/app/master/middleware"
	"neomaster/internal/config"
	"neomaster/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestMiddlewareManagerWithConfig 测试中间件管理器使用配置文件
func TestMiddlewareManagerWithConfig(t *testing.T) {
	// 初始化日志
	logger.InitLogger(&config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	})

	// 创建测试配置
	securityConfig := &config.SecurityConfig{
		RateLimit: config.RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 10,
			WindowSize:        "1m",
		},
		Logging: config.LoggingConfig{
			EnableRequestLog:     true,
			EnableResponseLog:    true,
			LogRequestBody:       true,
			LogResponseBody:      false,
			SlowRequestThreshold: 500 * time.Millisecond,
			SkipPaths:            []string{"/health"},
			MaxRequestBodySize:   1024,
			MaxResponseBodySize:  1024,
		},
	}

	// 创建中间件管理器 (需要传入nil参数，因为这是配置测试)
	manager := middleware.NewMiddlewareManager(nil, nil, nil, securityConfig)

	// 创建测试路由
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 手动应用中间件 (因为没有ApplyMiddlewares方法)
	router.Use(manager.GinLoggingMiddleware())
	router.Use(manager.GinRateLimitMiddleware())

	// 添加测试路由
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})
	
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/slow", func(c *gin.Context) {
		time.Sleep(600 * time.Millisecond) // 模拟慢请求
		c.JSON(http.StatusOK, gin.H{"message": "slow response"})
	})

	// 测试正常请求
	t.Run("Normal Request", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test", response["message"])
	})

	// 测试跳过路径
	t.Run("Skip Path", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 测试慢请求
	t.Run("Slow Request", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/slow", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 测试带请求体的请求
	t.Run("Request With Body", func(t *testing.T) {
		body := map[string]interface{}{
			"test": "data",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestRateLimitConfig 测试限流配置
func TestRateLimitConfig(t *testing.T) {
	// 初始化日志
	logger.InitLogger(&config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	})

	// 测试限流配置
	securityConfig := &config.SecurityConfig{
		RateLimit: config.RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 2, // 设置很小的限制便于测试
			WindowSize:        "1s",
		},
		Logging: config.LoggingConfig{
			EnableRequestLog: false, // 关闭日志避免干扰测试
		},
	}

	// 创建中间件管理器
	manager := middleware.NewMiddlewareManager(nil, nil, nil, securityConfig)

	// 创建测试路由
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(manager.GinRateLimitMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// 测试限流功能
	t.Run("Rate Limit Test", func(t *testing.T) {
		// 前两个请求应该成功
		for i := 0; i < 2; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.100") // 设置固定IP
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		}

		// 第三个请求应该被限流
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.100") // 同一个IP
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "Third request should be rate limited")
	})
}

// TestLoggingConfig 测试日志配置
func TestLoggingConfig(t *testing.T) {
	// 初始化日志
	logger.InitLogger(&config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	})

	// 测试禁用日志
	t.Run("Logging Disabled", func(t *testing.T) {
		securityConfig := &config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				Enabled: false,
			},
			Logging: config.LoggingConfig{
				EnableRequestLog:  false,
				EnableResponseLog: false,
			},
		}

		manager := middleware.NewMiddlewareManager(nil, nil, nil, securityConfig)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(manager.GinLoggingMiddleware())

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 测试启用日志
	t.Run("Logging Enabled", func(t *testing.T) {
		securityConfig := &config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				Enabled: false,
			},
			Logging: config.LoggingConfig{
				EnableRequestLog:     true,
				EnableResponseLog:    true,
				LogRequestBody:       true,
				LogResponseBody:      true,
				SlowRequestThreshold: 100 * time.Millisecond,
				MaxRequestBodySize:   1024,
				MaxResponseBodySize:  1024,
			},
		}

		manager := middleware.NewMiddlewareManager(nil, nil, nil, securityConfig)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(manager.GinLoggingMiddleware())

		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		body := map[string]interface{}{
			"test": "data",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestMiddlewareOrder 测试中间件顺序
func TestMiddlewareOrder(t *testing.T) {
	// 初始化日志
	logger.InitLogger(&config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	})

	securityConfig := &config.SecurityConfig{
		RateLimit: config.RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 10,
			WindowSize:        "1m",
		},
		Logging: config.LoggingConfig{
			EnableRequestLog: true,
		},
	}

	manager := middleware.NewMiddlewareManager(nil, nil, nil, securityConfig)
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 记录中间件执行顺序
	var executionOrder []string
	
	// 添加测试中间件来验证执行顺序
	router.Use(func(c *gin.Context) {
		executionOrder = append(executionOrder, "before_middleware_manager")
		c.Next()
		executionOrder = append(executionOrder, "after_middleware_manager")
	})

	router.Use(manager.GinLoggingMiddleware())

	router.Use(func(c *gin.Context) {
		executionOrder = append(executionOrder, "after_apply_middlewares")
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		executionOrder = append(executionOrder, "handler")
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// 重置执行顺序
	executionOrder = []string{}

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证中间件执行顺序
	assert.Contains(t, executionOrder, "before_middleware_manager")
	assert.Contains(t, executionOrder, "handler")
	assert.Contains(t, executionOrder, "after_apply_middlewares")
}
