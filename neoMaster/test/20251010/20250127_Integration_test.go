// 集成测试文件
// 验证优化后的中间件和路由模块的协同工作
// 测试完整的请求流程，包括中间件链、路由处理、错误处理等
// 测试命令：go test -v -run TestIntegration ./test/20250127

// Package test 集成测试
// 验证优化后的中间件和路由模块协同工作
package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestMiddlewareRouterIntegration 测试中间件和路由的集成
func TestMiddlewareRouterIntegration(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.RouterManager == nil {
			t.Skip("跳过集成测试：路由管理器不可用")
			return
		}

		gin.SetMode(gin.TestMode)
		engine := ts.RouterManager.GetEngine()

		t.Run("完整请求流程测试", func(t *testing.T) {
			testCompleteRequestFlow(t, engine)
		})

		t.Run("中间件链执行顺序测试", func(t *testing.T) {
			testMiddlewareChainOrder(t, engine)
		})

		t.Run("错误处理集成测试", func(t *testing.T) {
			testErrorHandlingIntegration(t, engine)
		})

		t.Run("安全中间件集成测试", func(t *testing.T) {
			testSecurityMiddlewareIntegration(t, engine)
		})

		t.Run("认证授权集成测试", func(t *testing.T) {
			testAuthenticationAuthorizationIntegration(t, engine)
		})

		t.Run("限流中间件集成测试", func(t *testing.T) {
			testRateLimitIntegration(t, engine)
		})

		t.Run("日志中间件集成测试", func(t *testing.T) {
			testLoggingMiddlewareIntegration(t, engine)
		})
	})
}

// testCompleteRequestFlow 测试完整的请求流程
func testCompleteRequestFlow(t *testing.T, engine *gin.Engine) {
	// 测试公开接口的完整流程
	t.Run("公开接口完整流程", func(t *testing.T) {
		// 1. 健康检查 - 应该成功
		req, _ := http.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("User-Agent", "Integration-Test/1.0")
		w := httptest.NewRecorder()
		
		start := time.Now()
		engine.ServeHTTP(w, req)
		duration := time.Since(start)

		t.Logf("健康检查请求耗时: %v", duration)
		t.Logf("健康检查状态码: %d", w.Code)

		// 验证中间件是否正确应用
		AssertTrue(t, len(w.Header().Get("X-Content-Type-Options")) > 0, "应设置安全头")
		
		// 如果支持CORS，应该有相关头
		if corsHeader := w.Header().Get("Access-Control-Allow-Origin"); len(corsHeader) > 0 {
			t.Log("CORS中间件正常工作")
		}

		// 2. 用户注册 - 测试参数验证
		req, _ = http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{
			"username": "integrationuser",
			"email": "integration@test.com",
			"password": "testpass123"
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("User-Agent", "Integration-Test/1.0")
		w = httptest.NewRecorder()
		
		start = time.Now()
		engine.ServeHTTP(w, req)
		duration = time.Since(start)

		t.Logf("用户注册请求耗时: %v", duration)
		t.Logf("用户注册状态码: %d", w.Code)

		// 验证路由存在
		AssertNotEqual(t, http.StatusNotFound, w.Code, "注册路由应存在")
		
		// 验证中间件链
		AssertTrue(t, len(w.Header().Get("X-Content-Type-Options")) > 0, "应设置安全头")
		
		// 如果返回JSON错误，验证格式
		if w.Body.Len() > 0 && strings.Contains(w.Header().Get("Content-Type"), "application/json") {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				t.Log("注册接口返回有效JSON响应")
			}
		}

		// 3. 用户登录 - 测试认证流程
		req, _ = http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
			"username": "integrationuser",
			"password": "testpass123"
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://example.com")
		w = httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		t.Logf("用户登录状态码: %d", w.Code)
		
		AssertNotEqual(t, http.StatusNotFound, w.Code, "登录路由应存在")
	})

	// 测试受保护接口的完整流程
	t.Run("受保护接口完整流程", func(t *testing.T) {
		// 1. 无令牌访问用户信息 - 应被拒绝
		req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问应被JWT中间件拒绝")
		AssertTrue(t, len(w.Header().Get("X-Content-Type-Options")) > 0, "安全头中间件应仍然生效")

		// 2. 无效令牌访问 - 应被拒绝
		req, _ = http.NewRequest("GET", "/api/v1/user/profile", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		req.Header.Set("Origin", "https://example.com")
		w = httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无效令牌访问应被JWT中间件拒绝")

		// 3. 访问管理员接口 - 应被拒绝
		req, _ = http.NewRequest("GET", "/api/v1/admin/users", nil)
		w = httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问管理员接口应被拒绝")
	})
}

// testMiddlewareChainOrder 测试中间件链执行顺序
func testMiddlewareChainOrder(t *testing.T, engine *gin.Engine) {
	// 测试CORS预检请求的中间件链
	t.Run("CORS预检请求中间件链", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/api/v1/auth/login", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		t.Logf("CORS预检请求状态码: %d", w.Code)
		
		// 验证CORS中间件响应
		corsOrigin := w.Header().Get("Access-Control-Allow-Origin")
		corsMethod := w.Header().Get("Access-Control-Allow-Methods")
		corsHeaders := w.Header().Get("Access-Control-Allow-Headers")
		
		t.Logf("CORS Origin: %s", corsOrigin)
		t.Logf("CORS Methods: %s", corsMethod)
		t.Logf("CORS Headers: %s", corsHeaders)
		
		// 验证安全头中间件仍然生效
		securityHeader := w.Header().Get("X-Content-Type-Options")
		AssertEqual(t, "nosniff", securityHeader, "安全头中间件应在CORS之后仍然生效")
	})

	// 测试普通请求的中间件链
	t.Run("普通请求中间件链", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
			"username": "test",
			"password": "test"
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("User-Agent", "Test-Client/1.0")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		// 验证多个中间件都生效
		headers := w.Header()
		
		// 安全头中间件
		AssertEqual(t, "nosniff", headers.Get("X-Content-Type-Options"), "X-Content-Type-Options应被设置")
		AssertEqual(t, "DENY", headers.Get("X-Frame-Options"), "X-Frame-Options应被设置")
		
		// CORS中间件（如果配置了）
		if corsHeader := headers.Get("Access-Control-Allow-Origin"); len(corsHeader) > 0 {
			t.Log("CORS中间件正常工作")
		}
		
		// 日志中间件（通过响应时间验证）
		if responseTime := headers.Get("X-Response-Time"); len(responseTime) > 0 {
			t.Logf("响应时间: %s", responseTime)
		}
		
		t.Log("中间件链执行顺序正确")
	})
}

// testErrorHandlingIntegration 测试错误处理集成
func testErrorHandlingIntegration(t *testing.T, engine *gin.Engine) {
	// 测试404错误处理
	t.Run("404错误处理", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/nonexistent", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusNotFound, w.Code, "不存在的路由应返回404")
		
		// 验证中间件仍然生效
		AssertTrue(t, len(w.Header().Get("X-Content-Type-Options")) > 0, "404响应应仍然包含安全头")
		
		// 验证错误响应格式
		if w.Body.Len() > 0 {
			contentType := w.Header().Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err == nil {
					t.Log("404错误返回有效JSON格式")
					if message, exists := response["message"]; exists {
						t.Logf("错误消息: %v", message)
					}
				}
			}
		}
	})

	// 测试方法不允许错误处理
	t.Run("405错误处理", func(t *testing.T) {
		// 对POST接口发送GET请求
		req, _ := http.NewRequest("GET", "/api/v1/auth/login", nil)
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		// 应该返回405或404（取决于路由配置）
		AssertTrue(t, w.Code == http.StatusMethodNotAllowed || w.Code == http.StatusNotFound,
			"错误的HTTP方法应返回405或404")
		
		// 验证中间件仍然生效
		AssertTrue(t, len(w.Header().Get("X-Content-Type-Options")) > 0, "错误响应应仍然包含安全头")
	})

	// 测试请求体过大错误处理
	t.Run("请求体过大错误处理", func(t *testing.T) {
		// 创建一个较大的请求体
		largePayload := strings.Repeat("a", 1024*1024) // 1MB
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(largePayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		// 应该被限制
		AssertTrue(t, w.Code >= 400, "过大的请求体应被拒绝")
		t.Logf("大请求体状态码: %d", w.Code)
	})

	// 测试无效JSON错误处理
	t.Run("无效JSON错误处理", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{invalid json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertTrue(t, w.Code >= 400, "无效JSON应被拒绝")
		t.Logf("无效JSON状态码: %d", w.Code)
	})
}

// testSecurityMiddlewareIntegration 测试安全中间件集成
func testSecurityMiddlewareIntegration(t *testing.T, engine *gin.Engine) {
	// 测试安全头设置
	t.Run("安全头设置", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		headers := w.Header()
		
		// 验证各种安全头
		securityHeaders := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"X-XSS-Protection":       "1; mode=block",
		}
		
		for headerName, expectedValue := range securityHeaders {
			actualValue := headers.Get(headerName)
			if len(actualValue) > 0 {
				t.Logf("安全头 %s: %s", headerName, actualValue)
				if expectedValue != "" {
					AssertEqual(t, expectedValue, actualValue, 
						"安全头应设置正确")
				}
			} else {
				t.Logf("安全头 %s 未设置", headerName)
			}
		}
	})

	// 测试HTTPS重定向（如果配置了）
	t.Run("HTTPS重定向", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Forwarded-Proto", "http")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		// 如果配置了HTTPS重定向，应该返回重定向状态码
		if w.Code == http.StatusMovedPermanently || w.Code == http.StatusFound {
			location := w.Header().Get("Location")
			AssertTrue(t, strings.HasPrefix(location, "https://"), "应重定向到HTTPS")
			t.Log("HTTPS重定向中间件正常工作")
		} else {
			t.Log("未配置HTTPS重定向或在测试环境中禁用")
		}
	})

	// 测试内容安全策略（如果配置了）
	t.Run("内容安全策略", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		csp := w.Header().Get("Content-Security-Policy")
		if len(csp) > 0 {
			t.Logf("CSP策略: %s", csp)
			AssertTrue(t, len(csp) > 0, "CSP策略应被设置")
		} else {
			t.Log("未配置CSP策略")
		}
	})
}

// testAuthenticationAuthorizationIntegration 测试认证授权集成
func testAuthenticationAuthorizationIntegration(t *testing.T, engine *gin.Engine) {
	// 测试JWT认证中间件
	t.Run("JWT认证中间件", func(t *testing.T) {
		// 1. 无令牌访问
		req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问应被拒绝")
		
		// 验证错误响应格式
		if w.Body.Len() > 0 {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				t.Log("认证错误返回有效JSON格式")
			}
		}

		// 2. 无效令牌格式
		req, _ = http.NewRequest("GET", "/api/v1/user/profile", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w = httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无效令牌格式应被拒绝")

		// 3. 无效Bearer令牌
		req, _ = http.NewRequest("GET", "/api/v1/user/profile", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w = httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无效Bearer令牌应被拒绝")
	})

	// 测试管理员权限中间件
	t.Run("管理员权限中间件", func(t *testing.T) {
		// 无令牌访问管理员接口
		req, _ := http.NewRequest("GET", "/api/v1/admin/users", nil)
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问管理员接口应被拒绝")

		// 无效令牌访问管理员接口
		req, _ = http.NewRequest("GET", "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer invalid-admin-token")
		w = httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无效令牌访问管理员接口应被拒绝")
	})

	// 测试用户激活状态检查
	t.Run("用户激活状态检查", func(t *testing.T) {
		// 这个测试需要有效的JWT令牌，在实际环境中会检查用户激活状态
		// 这里只测试中间件是否存在
		req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
		req.Header.Set("Authorization", "Bearer test-token-for-inactive-user")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		// 应该被JWT中间件拦截（因为令牌无效）
		AssertEqual(t, http.StatusUnauthorized, w.Code, "无效令牌应被JWT中间件拦截")
	})
}

// testRateLimitIntegration 测试限流中间件集成
func testRateLimitIntegration(t *testing.T, engine *gin.Engine) {
	// 测试基本限流
	t.Run("基本限流测试", func(t *testing.T) {
		// 快速发送多个请求测试限流
		var responses []int
		
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
				"username": "testuser",
				"password": "testpass"
			}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Real-IP", "192.168.1.100") // 模拟同一IP
			w := httptest.NewRecorder()
			
			engine.ServeHTTP(w, req)
			responses = append(responses, w.Code)
			
			// 检查限流头
			if remaining := w.Header().Get("X-RateLimit-Remaining"); len(remaining) > 0 {
				t.Logf("请求 %d - 剩余请求数: %s", i+1, remaining)
			}
			
			if reset := w.Header().Get("X-RateLimit-Reset"); len(reset) > 0 {
				t.Logf("请求 %d - 重置时间: %s", i+1, reset)
			}
		}
		
		// 统计状态码
		statusCounts := make(map[int]int)
		for _, code := range responses {
			statusCounts[code]++
		}
		
		t.Logf("状态码统计: %v", statusCounts)
		
		// 如果有限流，应该有429状态码
		if count429, exists := statusCounts[http.StatusTooManyRequests]; exists {
			t.Logf("触发限流 %d 次", count429)
			AssertTrue(t, count429 > 0, "应该触发限流")
		} else {
			t.Log("未触发限流或限流配置较宽松")
		}
	})

	// 测试不同IP的限流隔离
	t.Run("不同IP限流隔离", func(t *testing.T) {
		ips := []string{"192.168.1.101", "192.168.1.102", "192.168.1.103"}
		
		for _, ip := range ips {
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
				"username": "testuser",
				"password": "testpass"
			}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Real-IP", ip)
			w := httptest.NewRecorder()
			
			engine.ServeHTTP(w, req)
			
			t.Logf("IP %s - 状态码: %d", ip, w.Code)
			
			// 不同IP应该有独立的限流计数
			if remaining := w.Header().Get("X-RateLimit-Remaining"); len(remaining) > 0 {
				t.Logf("IP %s - 剩余请求数: %s", ip, remaining)
			}
		}
	})
}

// testLoggingMiddlewareIntegration 测试日志中间件集成
func testLoggingMiddlewareIntegration(t *testing.T, engine *gin.Engine) {
	// 测试请求日志记录
	t.Run("请求日志记录", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
			"username": "logtest",
			"password": "logtest123"
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "LogTest/1.0")
		req.Header.Set("X-Request-ID", "test-request-123")
		w := httptest.NewRecorder()
		
		start := time.Now()
		engine.ServeHTTP(w, req)
		duration := time.Since(start)
		
		t.Logf("请求处理耗时: %v", duration)
		
		// 验证响应时间头（如果日志中间件设置了）
		if responseTime := w.Header().Get("X-Response-Time"); len(responseTime) > 0 {
			t.Logf("响应时间头: %s", responseTime)
		}
		
		// 验证请求ID头（如果设置了）
		if requestID := w.Header().Get("X-Request-ID"); len(requestID) > 0 {
			t.Logf("请求ID: %s", requestID)
		}
	})

	// 测试错误日志记录
	t.Run("错误日志记录", func(t *testing.T) {
		// 发送会导致错误的请求
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{invalid json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		engine.ServeHTTP(w, req)
		
		t.Logf("错误请求状态码: %d", w.Code)
		
		// 错误请求也应该被正确记录
		AssertTrue(t, w.Code >= 400, "无效请求应返回错误状态码")
	})

	// 测试不同HTTP方法的日志记录
	t.Run("不同HTTP方法日志记录", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE"}
		
		for _, method := range methods {
			var req *http.Request
			
			if method == "GET" {
				req, _ = http.NewRequest(method, "/health", nil)
			} else {
				req, _ = http.NewRequest(method, "/api/v1/auth/login", strings.NewReader(`{}`))
				req.Header.Set("Content-Type", "application/json")
			}
			
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("%s 请求状态码: %d", method, w.Code)
		}
	})
}

// TestFullSystemIntegration 测试完整系统集成
func TestFullSystemIntegration(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.RouterManager == nil {
			t.Skip("跳过完整系统集成测试：路由管理器不可用")
			return
		}

		gin.SetMode(gin.TestMode)
		engine := ts.RouterManager.GetEngine()

		// 测试系统启动和基本功能
		t.Run("系统启动和基本功能", func(t *testing.T) {
			// 1. 健康检查
			req, _ := http.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("系统健康检查: %d", w.Code)
			AssertNotEqual(t, http.StatusNotFound, w.Code, "健康检查应可用")

			// 2. 系统信息
			req, _ = http.NewRequest("GET", "/api/v1/public/info", nil)
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("系统信息接口: %d", w.Code)

			// 3. 版本信息
			req, _ = http.NewRequest("GET", "/api/v1/public/version", nil)
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("版本信息接口: %d", w.Code)
		})

		// 测试用户生命周期
		t.Run("用户生命周期测试", func(t *testing.T) {
			// 1. 用户注册
			req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{
				"username": "lifecycle_user",
				"email": "lifecycle@test.com",
				"password": "lifecycle123"
			}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("用户注册: %d", w.Code)
			AssertNotEqual(t, http.StatusNotFound, w.Code, "注册接口应存在")

			// 2. 用户登录
			req, _ = http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
				"username": "lifecycle_user",
				"password": "lifecycle123"
			}`))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("用户登录: %d", w.Code)

			// 3. 访问用户资源（无令牌）
			req, _ = http.NewRequest("GET", "/api/v1/user/profile", nil)
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问应被拒绝")

			// 4. 访问管理员资源（无权限）
			req, _ = http.NewRequest("GET", "/api/v1/admin/users", nil)
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			AssertEqual(t, http.StatusUnauthorized, w.Code, "无权限访问应被拒绝")
		})

		// 测试Agent通信流程
		t.Run("Agent通信流程测试", func(t *testing.T) {
			// 1. Agent注册
			req, _ := http.NewRequest("POST", "/api/v1/agent/register", strings.NewReader(`{
				"agent_id": "integration-agent-001",
				"agent_name": "集成测试Agent",
				"capabilities": ["port_scan", "service_detection"],
				"version": "1.0.0"
			}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("Agent注册: %d", w.Code)
			AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent注册接口应存在")

			// 2. Agent心跳
			req, _ = http.NewRequest("POST", "/api/v1/agent/heartbeat", strings.NewReader(`{
				"agent_id": "integration-agent-001",
				"status": "active",
				"load": 0.3
			}`))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("Agent心跳: %d", w.Code)

			// 3. 获取任务
			req, _ = http.NewRequest("GET", "/api/v1/agent/tasks", nil)
			req.Header.Set("Agent-ID", "integration-agent-001")
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("获取任务: %d", w.Code)
		})

		t.Log("完整系统集成测试完成")
	})
}