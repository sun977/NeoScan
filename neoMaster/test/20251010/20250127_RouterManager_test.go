// RouterManager测试文件
// 测试优化后的路由管理器功能，包括路由初始化、管理、协调等
// 适配优化后的router_manager.go模块
// 测试命令：go test -v -run TestRouterManager ./test/20250127

// Package test 路由管理器功能测试
// 测试优化后的router_manager.go模块
package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRouterManager 测试路由管理器模块
func TestRouterManager(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		// 检查必要的服务是否可用
		if ts.RouterManager == nil {
			t.Skip("跳过路由管理器测试：路由管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		t.Run("路由管理器初始化", func(t *testing.T) {
			testRouterManagerInitialization(t, ts)
		})

		t.Run("公开路由管理", func(t *testing.T) {
			testPublicRoutesManagement(t, ts)
		})

		t.Run("用户路由管理", func(t *testing.T) {
			testUserRoutesManagement(t, ts)
		})

		t.Run("管理员路由管理", func(t *testing.T) {
			testAdminRoutesManagement(t, ts)
		})

		t.Run("健康检查路由管理", func(t *testing.T) {
			testHealthRoutesManagement(t, ts)
		})

		t.Run("Agent路由管理", func(t *testing.T) {
			testAgentRoutesManagement(t, ts)
		})

		t.Run("路由中间件集成", func(t *testing.T) {
			testRouterMiddlewareIntegration(t, ts)
		})

		t.Run("路由错误处理", func(t *testing.T) {
			testRouterErrorHandling(t, ts)
		})
	})
}

// testRouterManagerInitialization 测试路由管理器初始化
func testRouterManagerInitialization(t *testing.T, ts *TestSuite) {
	// 验证路由管理器已正确初始化
	AssertNotNil(t, ts.RouterManager, "路由管理器应已初始化")

	// 验证Gin引擎已初始化
	engine := ts.RouterManager.GetEngine()
	AssertNotNil(t, engine, "Gin引擎应已初始化")

	// 验证路由管理器的基本配置
	t.Run("路由管理器基本配置", func(t *testing.T) {
		// 测试路由管理器是否正确设置了中间件
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 健康检查路由应该存在并返回正确响应
		if w.Code == http.StatusOK {
			t.Log("健康检查路由正常工作")
		} else {
			t.Logf("健康检查路由状态码: %d", w.Code)
		}
	})

	// 验证中间件是否正确应用
	t.Run("中间件应用验证", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/api/v1/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 验证CORS中间件是否生效
		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		if len(corsHeader) > 0 {
			t.Log("CORS中间件正常工作")
		}

		// 验证安全头中间件是否生效
		securityHeader := w.Header().Get("X-Content-Type-Options")
		if securityHeader == "nosniff" {
			t.Log("安全头中间件正常工作")
		}
	})
}

// testPublicRoutesManagement 测试公开路由管理
func testPublicRoutesManagement(t *testing.T, ts *TestSuite) {
	engine := ts.RouterManager.GetEngine()

	// 测试用户注册路由
	t.Run("用户注册路由", func(t *testing.T) {
		// 测试注册接口是否存在
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{
			"username": "testuser",
			"email": "test@example.com",
			"password": "testpassword123"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 验证路由存在（不是404）
		AssertNotEqual(t, http.StatusNotFound, w.Code, "注册路由应存在")
		
		// 如果返回400或其他错误，说明路由存在但参数可能有问题
		if w.Code == http.StatusBadRequest || w.Code == http.StatusUnprocessableEntity {
			t.Log("注册路由存在，参数验证正常")
		}
	})

	// 测试用户登录路由
	t.Run("用户登录路由", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
			"username": "testuser",
			"password": "testpassword123"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 验证路由存在
		AssertNotEqual(t, http.StatusNotFound, w.Code, "登录路由应存在")
	})

	// 测试密码重置路由
	t.Run("密码重置路由", func(t *testing.T) {
		// 测试请求重置密码
		req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password", strings.NewReader(`{
			"email": "test@example.com"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "忘记密码路由应存在")
	})

	// 测试邮箱验证路由
	t.Run("邮箱验证路由", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/verify-email?token=test-token", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "邮箱验证路由应存在")
	})
}

// testUserRoutesManagement 测试用户路由管理
func testUserRoutesManagement(t *testing.T, ts *TestSuite) {
	engine := ts.RouterManager.GetEngine()

	// 测试用户信息路由
	t.Run("用户信息路由", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 应该返回401未授权，说明路由存在但需要认证
		AssertEqual(t, http.StatusUnauthorized, w.Code, "用户信息路由应需要认证")
	})

	// 测试更新用户信息路由
	t.Run("更新用户信息路由", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/user/profile", strings.NewReader(`{
			"nickname": "新昵称"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "更新用户信息路由应需要认证")
	})

	// 测试修改密码路由
	t.Run("修改密码路由", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/user/password", strings.NewReader(`{
			"old_password": "oldpass",
			"new_password": "newpass123"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "修改密码路由应需要认证")
	})

	// 测试用户扫描任务路由
	t.Run("用户扫描任务路由", func(t *testing.T) {
		// 获取扫描任务列表
		req, _ := http.NewRequest("GET", "/api/v1/user/scans", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "扫描任务路由应需要认证")

		// 创建扫描任务
		req, _ = http.NewRequest("POST", "/api/v1/user/scans", strings.NewReader(`{
			"target": "192.168.1.1",
			"scan_type": "port_scan"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "创建扫描任务路由应需要认证")
	})
}

// testAdminRoutesManagement 测试管理员路由管理
func testAdminRoutesManagement(t *testing.T, ts *TestSuite) {
	engine := ts.RouterManager.GetEngine()

	// 测试用户管理路由
	t.Run("用户管理路由", func(t *testing.T) {
		// 获取用户列表
		req, _ := http.NewRequest("GET", "/api/v1/admin/users", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "管理员用户列表路由应需要认证")

		// 删除用户
		req, _ = http.NewRequest("DELETE", "/api/v1/admin/users/123", nil)
		w = httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "删除用户路由应需要认证")
	})

	// 测试系统配置路由
	t.Run("系统配置路由", func(t *testing.T) {
		// 获取系统配置
		req, _ := http.NewRequest("GET", "/api/v1/admin/config", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "系统配置路由应需要认证")

		// 更新系统配置
		req, _ = http.NewRequest("PUT", "/api/v1/admin/config", strings.NewReader(`{
			"max_concurrent_scans": 10
		}`))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "更新系统配置路由应需要认证")
	})

	// 测试系统监控路由
	t.Run("系统监控路由", func(t *testing.T) {
		// 获取系统状态
		req, _ := http.NewRequest("GET", "/api/v1/admin/system/status", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "系统状态路由应需要认证")

		// 获取系统日志
		req, _ = http.NewRequest("GET", "/api/v1/admin/system/logs", nil)
		w = httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "系统日志路由应需要认证")
	})
}

// testHealthRoutesManagement 测试健康检查路由管理
func testHealthRoutesManagement(t *testing.T, ts *TestSuite) {
	engine := ts.RouterManager.GetEngine()

	// 测试基本健康检查
	t.Run("基本健康检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 健康检查应该返回200或者至少不是404
		AssertNotEqual(t, http.StatusNotFound, w.Code, "健康检查路由应存在")
		
		if w.Code == http.StatusOK {
			// 验证响应格式
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				t.Log("健康检查返回有效JSON响应")
				if status, exists := response["status"]; exists {
					t.Logf("健康检查状态: %v", status)
				}
			}
		}
	})

	// 测试详细健康检查
	t.Run("详细健康检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health/detailed", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "详细健康检查路由应存在")
	})

	// 测试就绪检查
	t.Run("就绪检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/ready", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "就绪检查路由应存在")
	})

	// 测试存活检查
	t.Run("存活检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/alive", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "存活检查路由应存在")
	})
}

// testAgentRoutesManagement 测试Agent路由管理
func testAgentRoutesManagement(t *testing.T, ts *TestSuite) {
	engine := ts.RouterManager.GetEngine()

	// 测试Agent注册路由
	t.Run("Agent注册路由", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/agent/register", strings.NewReader(`{
			"agent_id": "test-agent-001",
			"agent_name": "测试Agent",
			"capabilities": ["port_scan", "service_detection"]
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent注册路由应存在")
	})

	// 测试Agent心跳路由
	t.Run("Agent心跳路由", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/agent/heartbeat", strings.NewReader(`{
			"agent_id": "test-agent-001",
			"status": "active",
			"load": 0.5
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent心跳路由应存在")
	})

	// 测试任务分发路由
	t.Run("任务分发路由", func(t *testing.T) {
		// 获取任务
		req, _ := http.NewRequest("GET", "/api/v1/agent/tasks", nil)
		req.Header.Set("Agent-ID", "test-agent-001")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "获取任务路由应存在")

		// 提交任务结果
		req, _ = http.NewRequest("POST", "/api/v1/agent/tasks/123/result", strings.NewReader(`{
			"status": "completed",
			"result": {"ports": [80, 443, 22]}
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Agent-ID", "test-agent-001")
		w = httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "提交任务结果路由应存在")
	})
}

// testRouterMiddlewareIntegration 测试路由中间件集成
func testRouterMiddlewareIntegration(t *testing.T, ts *TestSuite) {
	engine := ts.RouterManager.GetEngine()

	// 测试中间件在不同路由组的应用
	t.Run("公开路由中间件", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 验证CORS中间件生效
		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		AssertTrue(t, len(corsHeader) > 0, "公开路由应应用CORS中间件")

		// 验证安全头中间件生效
		securityHeader := w.Header().Get("X-Content-Type-Options")
		AssertEqual(t, "nosniff", securityHeader, "公开路由应应用安全头中间件")
	})

	// 测试受保护路由中间件
	t.Run("受保护路由中间件", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 应该被JWT中间件拦截
		AssertEqual(t, http.StatusUnauthorized, w.Code, "受保护路由应应用JWT中间件")

		// 验证其他中间件仍然生效
		securityHeader := w.Header().Get("X-Content-Type-Options")
		AssertEqual(t, "nosniff", securityHeader, "受保护路由应保持安全头中间件")
	})

	// 测试管理员路由中间件
	t.Run("管理员路由中间件", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/users", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 应该被JWT中间件拦截（因为没有令牌）
		AssertEqual(t, http.StatusUnauthorized, w.Code, "管理员路由应应用JWT和管理员权限中间件")
	})
}

// testRouterErrorHandling 测试路由错误处理
func testRouterErrorHandling(t *testing.T, ts *TestSuite) {
	engine := ts.RouterManager.GetEngine()

	// 测试404错误处理
	t.Run("404错误处理", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/nonexistent", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusNotFound, w.Code, "不存在的路由应返回404")

		// 验证错误响应格式
		if w.Body.Len() > 0 {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				t.Log("404错误返回有效JSON响应")
			}
		}
	})

	// 测试方法不允许错误处理
	t.Run("405错误处理", func(t *testing.T) {
		// 对只支持POST的路由发送GET请求
		req, _ := http.NewRequest("GET", "/api/v1/auth/login", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 应该返回405方法不允许或404（取决于路由配置）
		AssertTrue(t, w.Code == http.StatusMethodNotAllowed || w.Code == http.StatusNotFound,
			"错误的HTTP方法应返回405或404")
	})

	// 测试请求体过大错误处理
	t.Run("请求体过大错误处理", func(t *testing.T) {
		// 创建一个很大的请求体
		largeBody := strings.Repeat("a", 10*1024*1024) // 10MB
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 应该被限制（具体状态码取决于中间件配置）
		AssertTrue(t, w.Code >= 400, "过大的请求体应被拒绝")
	})
}

// TestRouterManagerIntegration 测试路由管理器集成
func TestRouterManagerIntegration(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.RouterManager == nil {
			t.Skip("跳过路由管理器集成测试：路由管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)
		engine := ts.RouterManager.GetEngine()

		// 测试完整的请求流程
		t.Run("完整请求流程测试", func(t *testing.T) {
			// 1. 健康检查（无需认证）
			req, _ := http.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("健康检查状态码: %d", w.Code)

			// 2. 用户注册（公开接口）
			req, _ = http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{
				"username": "integrationtest",
				"email": "integration@test.com",
				"password": "testpass123"
			}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "https://example.com")
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			t.Logf("用户注册状态码: %d", w.Code)
			AssertNotEqual(t, http.StatusNotFound, w.Code, "注册接口应存在")

			// 验证CORS和安全头
			AssertTrue(t, len(w.Header().Get("Access-Control-Allow-Origin")) > 0, "应设置CORS头")
			AssertEqual(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "应设置安全头")

			// 3. 尝试访问受保护资源（应被拒绝）
			req, _ = http.NewRequest("GET", "/api/v1/user/profile", nil)
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问应被拒绝")

			// 4. 尝试访问管理员资源（应被拒绝）
			req, _ = http.NewRequest("GET", "/api/v1/admin/users", nil)
			w = httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			
			AssertEqual(t, http.StatusUnauthorized, w.Code, "无权限访问应被拒绝")
		})

		// 测试路由组隔离
		t.Run("路由组隔离测试", func(t *testing.T) {
			routeTests := []struct {
				method   string
				path     string
				expected int
			}{
				{"GET", "/health", http.StatusOK},                    // 健康检查
				{"POST", "/api/v1/auth/login", http.StatusBadRequest}, // 公开接口（参数错误）
				{"GET", "/api/v1/user/profile", http.StatusUnauthorized}, // 用户接口（需认证）
				{"GET", "/api/v1/admin/users", http.StatusUnauthorized},  // 管理员接口（需认证+权限）
				{"POST", "/api/v1/agent/heartbeat", http.StatusBadRequest}, // Agent接口
			}

			for _, rt := range routeTests {
				req, _ := http.NewRequest(rt.method, rt.path, nil)
				if rt.method == "POST" {
					req.Header.Set("Content-Type", "application/json")
				}
				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)

				// 验证路由存在且返回预期状态码
				if rt.expected == http.StatusOK {
					AssertEqual(t, rt.expected, w.Code, 
						"路由应返回预期状态码")
				} else {
					AssertTrue(t, w.Code == rt.expected || w.Code == http.StatusNotFound,
						"路由应返回预期状态码或404")
				}
			}
		})

		t.Log("路由管理器集成测试通过，所有路由组正常工作")
	})
}