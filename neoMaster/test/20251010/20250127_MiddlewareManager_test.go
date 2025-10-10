// MiddlewareManager测试文件
// 测试优化后的中间件管理器功能，包括中间件初始化、管理、协调等
// 适配拆分后的middleware_manager.go模块
// 测试命令：go test -v -run TestMiddlewareManager ./test/20250127

// Package test 中间件管理器功能测试
// 测试拆分后的middleware_manager.go模块
package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMiddlewareManager 测试中间件管理器模块
func TestMiddlewareManager(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		// 检查必要的服务是否可用
		if ts.MiddlewareManager == nil {
			t.Skip("跳过中间件管理器测试：中间件管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		t.Run("中间件管理器初始化", func(t *testing.T) {
			testMiddlewareManagerInitialization(t, ts)
		})

		t.Run("认证中间件管理", func(t *testing.T) {
			testAuthMiddlewareManagement(t, ts)
		})

		t.Run("日志中间件管理", func(t *testing.T) {
			testLoggingMiddlewareManagement(t, ts)
		})

		t.Run("安全中间件管理", func(t *testing.T) {
			testSecurityMiddlewareManagement(t, ts)
		})

		t.Run("限流中间件管理", func(t *testing.T) {
			testRateLimitMiddlewareManagement(t, ts)
		})

		t.Run("中间件链协调", func(t *testing.T) {
			testMiddlewareChainCoordination(t, ts)
		})

		t.Run("中间件错误处理", func(t *testing.T) {
			testMiddlewareErrorHandling(t, ts)
		})
	})
}

// testMiddlewareManagerInitialization 测试中间件管理器初始化
func testMiddlewareManagerInitialization(t *testing.T, ts *TestSuite) {
	// 验证中间件管理器已正确初始化
	AssertNotNil(t, ts.MiddlewareManager, "中间件管理器应已初始化")

	// 验证各个中间件方法是否可用
	t.Run("认证中间件方法可用性", func(t *testing.T) {
		// 测试JWT中间件方法
		jwtMiddleware := ts.MiddlewareManager.GinJWTAuthMiddleware()
		AssertNotNil(t, jwtMiddleware, "JWT中间件应可用")

		// 测试用户激活检查中间件
		activeMiddleware := ts.MiddlewareManager.GinUserActiveMiddleware()
		AssertNotNil(t, activeMiddleware, "用户激活检查中间件应可用")

		// 测试管理员权限中间件
		adminMiddleware := ts.MiddlewareManager.GinAdminRoleMiddleware()
		AssertNotNil(t, adminMiddleware, "管理员权限中间件应可用")
	})

	t.Run("日志中间件方法可用性", func(t *testing.T) {
		// 测试日志中间件方法
		loggingMiddleware := ts.MiddlewareManager.GinLoggingMiddleware()
		AssertNotNil(t, loggingMiddleware, "日志中间件应可用")
	})

	t.Run("安全中间件方法可用性", func(t *testing.T) {
		// 测试CORS中间件
		corsMiddleware := ts.MiddlewareManager.GinCORSMiddleware()
		AssertNotNil(t, corsMiddleware, "CORS中间件应可用")

		// 测试安全头中间件
		securityMiddleware := ts.MiddlewareManager.GinSecurityHeadersMiddleware()
		AssertNotNil(t, securityMiddleware, "安全头中间件应可用")
	})

	t.Run("限流中间件方法可用性", func(t *testing.T) {
		// 测试基本限流中间件
		rateLimitMiddleware := ts.MiddlewareManager.GinRateLimitMiddleware()
		AssertNotNil(t, rateLimitMiddleware, "基本限流中间件应可用")

		// 测试API限流中间件
		apiRateLimitMiddleware := ts.MiddlewareManager.GinAPIRateLimitMiddleware()
		AssertNotNil(t, apiRateLimitMiddleware, "API限流中间件应可用")
	})
}

// testAuthMiddlewareManagement 测试认证中间件管理
func testAuthMiddlewareManagement(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	_ = gin.New()
	
	// 测试JWT中间件
	t.Run("JWT中间件管理", func(t *testing.T) {
		jwtRouter := gin.New()
		jwtRouter.Use(ts.MiddlewareManager.GinJWTAuthMiddleware())
		jwtRouter.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "受保护的资源"})
		})

		// 测试无令牌访问
		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		jwtRouter.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问应返回401")
	})

	// 测试用户激活检查中间件
	t.Run("用户激活检查中间件管理", func(t *testing.T) {
		activeRouter := gin.New()
		activeRouter.Use(ts.MiddlewareManager.GinUserActiveMiddleware())
		activeRouter.GET("/active-required", func(t *gin.Context) {
			t.JSON(http.StatusOK, gin.H{"message": "需要激活用户"})
		})

		// 测试未激活用户访问
		req, _ := http.NewRequest("GET", "/active-required", nil)
		w := httptest.NewRecorder()
		activeRouter.ServeHTTP(w, req)

		// 由于没有用户上下文，应该被拦截
		AssertNotEqual(t, http.StatusOK, w.Code, "未激活用户访问应被拦截")
	})

	// 测试管理员权限中间件
	t.Run("管理员权限中间件管理", func(t *testing.T) {
		adminRouter := gin.New()
		adminRouter.Use(ts.MiddlewareManager.GinAdminRoleMiddleware())
		adminRouter.GET("/admin-only", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "管理员专用"})
		})

		// 测试非管理员访问
		req, _ := http.NewRequest("GET", "/admin-only", nil)
		w := httptest.NewRecorder()
		adminRouter.ServeHTTP(w, req)

		// 由于没有管理员权限，应该被拦截
		AssertNotEqual(t, http.StatusOK, w.Code, "非管理员访问应被拦截")
	})
}

// testLoggingMiddlewareManagement 测试日志中间件管理
func testLoggingMiddlewareManagement(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
	router.GET("/log-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "日志测试"})
	})

	// 发送测试请求
	req, _ := http.NewRequest("GET", "/log-test", nil)
	req.Header.Set("User-Agent", "middleware-manager-test")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	AssertEqual(t, http.StatusOK, w.Code, "日志中间件应正常工作")
	AssertTrue(t, strings.Contains(w.Body.String(), "日志测试"), "响应内容应正确")

	// 注意：实际的日志输出验证在日志中间件专门的测试中进行
}

// testSecurityMiddlewareManagement 测试安全中间件管理
func testSecurityMiddlewareManagement(t *testing.T, ts *TestSuite) {
	// 测试CORS中间件管理
	t.Run("CORS中间件管理", func(t *testing.T) {
		corsRouter := gin.New()
		corsRouter.Use(ts.MiddlewareManager.GinCORSMiddleware())
		corsRouter.GET("/cors-test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "CORS测试"})
		})

		// 发送CORS请求
		req, _ := http.NewRequest("GET", "/cors-test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		corsRouter.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "CORS请求应成功")
		AssertEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"), "CORS头应正确设置")
	})

	// 测试安全头中间件管理
	t.Run("安全头中间件管理", func(t *testing.T) {
		securityRouter := gin.New()
		securityRouter.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
		securityRouter.GET("/security-test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "安全头测试"})
		})

		// 发送测试请求
		req, _ := http.NewRequest("GET", "/security-test", nil)
		w := httptest.NewRecorder()
		securityRouter.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "安全头请求应成功")
		AssertEqual(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "安全头应正确设置")
	})
}

// testRateLimitMiddlewareManagement 测试限流中间件管理
func testRateLimitMiddlewareManagement(t *testing.T, ts *TestSuite) {
	// 测试基本限流中间件管理
	t.Run("基本限流中间件管理", func(t *testing.T) {
		rateLimitRouter := gin.New()
		rateLimitRouter.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
		rateLimitRouter.GET("/rate-limit-test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "限流测试"})
		})

		// 发送测试请求
		req, _ := http.NewRequest("GET", "/rate-limit-test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		rateLimitRouter.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "限流请求应成功")
		AssertTrue(t, strings.Contains(w.Body.String(), "限流测试"), "响应内容应正确")
	})

	// 测试API限流中间件管理
	t.Run("API限流中间件管理", func(t *testing.T) {
		apiRateLimitRouter := gin.New()
		apiRateLimitRouter.Use(ts.MiddlewareManager.GinAPIRateLimitMiddleware())
		apiRateLimitRouter.GET("/api-rate-limit-test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "API限流测试"})
		})

		// 发送测试请求
		req, _ := http.NewRequest("GET", "/api-rate-limit-test", nil)
		req.RemoteAddr = "192.168.1.101:12345"
		w := httptest.NewRecorder()
		apiRateLimitRouter.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "API限流请求应成功")
		AssertTrue(t, strings.Contains(w.Body.String(), "API限流测试"), "响应内容应正确")
	})
}

// testMiddlewareChainCoordination 测试中间件链协调
func testMiddlewareChainCoordination(t *testing.T, ts *TestSuite) {
	// 创建包含多个中间件的路由
	router := gin.New()
	
	// 按顺序添加中间件
	router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
	router.Use(ts.MiddlewareManager.GinCORSMiddleware())
	router.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
	router.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
	
	router.GET("/chain-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "中间件链测试成功",
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
		})
	})

	// 发送测试请求
	req, _ := http.NewRequest("GET", "/chain-test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("User-Agent", "chain-test-agent")
	req.RemoteAddr = "192.168.1.102:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	AssertEqual(t, http.StatusOK, w.Code, "中间件链请求应成功")
	
	// 验证各个中间件都生效
	headers := w.Header()
	
	// CORS中间件效果
	AssertEqual(t, "*", headers.Get("Access-Control-Allow-Origin"), "CORS中间件应生效")
	
	// 安全头中间件效果
	AssertEqual(t, "nosniff", headers.Get("X-Content-Type-Options"), "安全头中间件应生效")
	
	// 验证响应内容
	AssertTrue(t, strings.Contains(w.Body.String(), "中间件链测试成功"), "响应内容应正确")
	
	t.Log("中间件链协调测试通过，所有中间件正常协作")
}

// testMiddlewareErrorHandling 测试中间件错误处理
func testMiddlewareErrorHandling(t *testing.T, ts *TestSuite) {
	// 测试中间件错误传播
	t.Run("中间件错误传播", func(t *testing.T) {
		router := gin.New()
		
		// 添加日志中间件来记录错误
		router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
		
		// 添加一个会产生错误的中间件
		router.Use(func(c *gin.Context) {
			// 模拟中间件处理
			c.Next()
			
			// 检查是否有错误
			if len(c.Errors) > 0 {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "中间件处理错误",
					"details": c.Errors.String(),
				})
				c.Abort()
				return
			}
		})
		
		router.GET("/error-test", func(c *gin.Context) {
			// 模拟业务逻辑错误
		c.Error(fmt.Errorf("测试错误"))
		c.JSON(http.StatusOK, gin.H{"message": "这不应该被返回"})
		})

		// 发送测试请求
		req, _ := http.NewRequest("GET", "/error-test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 验证错误处理
		AssertEqual(t, http.StatusInternalServerError, w.Code, "应返回错误状态码")
		AssertTrue(t, strings.Contains(w.Body.String(), "中间件处理错误"), "应包含错误信息")
	})

	// 测试中间件恢复机制
	t.Run("中间件恢复机制", func(t *testing.T) {
		router := gin.New()
		
		// 添加恢复中间件
		router.Use(gin.Recovery())
		router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
		
		router.GET("/panic-test", func(c *gin.Context) {
			// 模拟panic
			panic("测试panic恢复")
		})

		// 发送测试请求
		req, _ := http.NewRequest("GET", "/panic-test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 验证恢复机制
		AssertEqual(t, http.StatusInternalServerError, w.Code, "panic应被恢复并返回500")
	})
}

// TestMiddlewareManagerIntegration 测试中间件管理器集成
func TestMiddlewareManagerIntegration(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.MiddlewareManager == nil {
			t.Skip("跳过中间件管理器集成测试：中间件管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		// 创建完整的中间件栈
		router := gin.New()
		
		// 基础中间件
		router.Use(gin.Recovery())
		router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
		
		// 安全中间件
		router.Use(ts.MiddlewareManager.GinCORSMiddleware())
		router.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
		
		// 限流中间件
		router.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
		
		// 公开路由
		router.GET("/public", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "公开接口", "type": "public"})
		})
		
		// 受保护路由
		protected := router.Group("/protected")
		protected.Use(ts.MiddlewareManager.GinJWTAuthMiddleware())
		protected.GET("/user", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "用户接口", "type": "protected"})
		})
		
		// 管理员路由
		admin := router.Group("/admin")
		admin.Use(ts.MiddlewareManager.GinJWTAuthMiddleware())
		admin.Use(ts.MiddlewareManager.GinAdminRoleMiddleware())
		admin.GET("/dashboard", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "管理员面板", "type": "admin"})
		})

		// 测试公开接口
		t.Run("公开接口集成测试", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/public", nil)
			req.Header.Set("Origin", "https://example.com")
			req.RemoteAddr = "192.168.1.200:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			AssertEqual(t, http.StatusOK, w.Code, "公开接口应成功")
			AssertTrue(t, strings.Contains(w.Body.String(), "公开接口"), "响应内容应正确")
			
			// 验证中间件效果
			AssertEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"), "CORS应生效")
			AssertEqual(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "安全头应生效")
		})

		// 测试受保护接口
		t.Run("受保护接口集成测试", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/protected/user", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 没有JWT令牌应该被拒绝
			AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问应被拒绝")
		})

		// 测试管理员接口
		t.Run("管理员接口集成测试", func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 没有JWT令牌和管理员权限应该被拒绝
			AssertEqual(t, http.StatusUnauthorized, w.Code, "无权限访问应被拒绝")
		})

		t.Log("中间件管理器集成测试通过，所有中间件正常协作")
	})
}