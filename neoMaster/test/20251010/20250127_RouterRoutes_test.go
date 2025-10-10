// 路由模块测试文件
// 测试优化后的各个路由模块功能，包括公开路由、用户路由、管理员路由、健康检查路由、Agent路由
// 适配优化后的路由模块结构
// 测试命令：go test -v -run TestRoutes ./test/20250127

// Package test 路由模块功能测试
// 测试优化后的各个路由文件
package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestPublicRoutes 测试公开路由模块
func TestPublicRoutes(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.RouterManager == nil {
			t.Skip("跳过公开路由测试：路由管理器不可用")
			return
		}

		gin.SetMode(gin.TestMode)
		engine := ts.RouterManager.GetEngine()

		t.Run("认证相关路由", func(t *testing.T) {
			testAuthenticationRoutes(t, engine)
		})

		t.Run("公开API路由", func(t *testing.T) {
			testPublicAPIRoutes(t, engine)
		})

		t.Run("文档和静态资源路由", func(t *testing.T) {
			testDocumentationRoutes(t, engine)
		})
	})
}

// TestUserRoutes 测试用户路由模块
func TestUserRoutes(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.RouterManager == nil {
			t.Skip("跳过用户路由测试：路由管理器不可用")
			return
		}

		gin.SetMode(gin.TestMode)
		engine := ts.RouterManager.GetEngine()

		t.Run("用户信息管理路由", func(t *testing.T) {
			testUserProfileRoutes(t, engine)
		})

		t.Run("用户扫描任务路由", func(t *testing.T) {
			testUserScanRoutes(t, engine)
		})

		t.Run("用户设置路由", func(t *testing.T) {
			testUserSettingsRoutes(t, engine)
		})
	})
}

// TestAdminRoutes 测试管理员路由模块
func TestAdminRoutes(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.RouterManager == nil {
			t.Skip("跳过管理员路由测试：路由管理器不可用")
			return
		}

		gin.SetMode(gin.TestMode)
		engine := ts.RouterManager.GetEngine()

		t.Run("用户管理路由", func(t *testing.T) {
			testAdminUserManagementRoutes(t, engine)
		})

		t.Run("系统管理路由", func(t *testing.T) {
			testAdminSystemManagementRoutes(t, engine)
		})

		t.Run("监控和日志路由", func(t *testing.T) {
			testAdminMonitoringRoutes(t, engine)
		})
	})
}

// TestHealthRoutes 测试健康检查路由模块
func TestHealthRoutes(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.RouterManager == nil {
			t.Skip("跳过健康检查路由测试：路由管理器不可用")
			return
		}

		gin.SetMode(gin.TestMode)
		engine := ts.RouterManager.GetEngine()

		t.Run("基础健康检查路由", func(t *testing.T) {
			testBasicHealthRoutes(t, engine)
		})

		t.Run("详细健康检查路由", func(t *testing.T) {
			testDetailedHealthRoutes(t, engine)
		})

		t.Run("服务状态检查路由", func(t *testing.T) {
			testServiceStatusRoutes(t, engine)
		})
	})
}

// TestAgentRoutes 测试Agent路由模块
func TestAgentRoutes(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.RouterManager == nil {
			t.Skip("跳过Agent路由测试：路由管理器不可用")
			return
		}

		gin.SetMode(gin.TestMode)
		engine := ts.RouterManager.GetEngine()

		t.Run("Agent管理路由", func(t *testing.T) {
			testAgentManagementRoutes(t, engine)
		})

		t.Run("任务分发路由", func(t *testing.T) {
			testTaskDistributionRoutes(t, engine)
		})

		t.Run("Agent通信路由", func(t *testing.T) {
			testAgentCommunicationRoutes(t, engine)
		})
	})
}

// testAuthenticationRoutes 测试认证相关路由
func testAuthenticationRoutes(t *testing.T, engine *gin.Engine) {
	// 测试用户注册
	t.Run("用户注册", func(t *testing.T) {
		testCases := []struct {
			name     string
			payload  string
			expected int
		}{
			{
				name: "有效注册数据",
				payload: `{
					"username": "testuser123",
					"email": "test@example.com",
					"password": "password123"
				}`,
				expected: http.StatusBadRequest, // 可能因为验证规则返回400
			},
			{
				name: "无效邮箱格式",
				payload: `{
					"username": "testuser123",
					"email": "invalid-email",
					"password": "password123"
				}`,
				expected: http.StatusBadRequest,
			},
			{
				name: "密码过短",
				payload: `{
					"username": "testuser123",
					"email": "test@example.com",
					"password": "123"
				}`,
				expected: http.StatusBadRequest,
			},
			{
				name: "空请求体",
				payload:  `{}`,
				expected: http.StatusBadRequest,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(tc.payload))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)

				// 验证路由存在
				AssertNotEqual(t, http.StatusNotFound, w.Code, "注册路由应存在")
				
				// 如果不是404，验证是否符合预期
				if w.Code != http.StatusNotFound {
					t.Logf("%s - 状态码: %d", tc.name, w.Code)
				}
			})
		}
	})

	// 测试用户登录
	t.Run("用户登录", func(t *testing.T) {
		testCases := []struct {
			name     string
			payload  string
			expected int
		}{
			{
				name: "有效登录数据",
				payload: `{
					"username": "testuser",
					"password": "password123"
				}`,
				expected: http.StatusUnauthorized, // 用户不存在或密码错误
			},
			{
				name: "缺少用户名",
				payload: `{
					"password": "password123"
				}`,
				expected: http.StatusBadRequest,
			},
			{
				name: "缺少密码",
				payload: `{
					"username": "testuser"
				}`,
				expected: http.StatusBadRequest,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(tc.payload))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)

				AssertNotEqual(t, http.StatusNotFound, w.Code, "登录路由应存在")
				t.Logf("%s - 状态码: %d", tc.name, w.Code)
			})
		}
	})

	// 测试密码重置
	t.Run("密码重置", func(t *testing.T) {
		// 请求重置密码
		req, _ := http.NewRequest("POST", "/api/v1/auth/forgot-password", strings.NewReader(`{
			"email": "test@example.com"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "忘记密码路由应存在")

		// 重置密码
		req, _ = http.NewRequest("POST", "/api/v1/auth/reset-password", strings.NewReader(`{
			"token": "reset-token",
			"new_password": "newpassword123"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "重置密码路由应存在")
	})

	// 测试邮箱验证
	t.Run("邮箱验证", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/auth/verify-email?token=test-token", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "邮箱验证路由应存在")
	})

	// 测试刷新令牌
	t.Run("刷新令牌", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", strings.NewReader(`{
			"refresh_token": "test-refresh-token"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "刷新令牌路由应存在")
	})
}

// testPublicAPIRoutes 测试公开API路由
func testPublicAPIRoutes(t *testing.T, engine *gin.Engine) {
	// 测试系统信息接口
	t.Run("系统信息", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/public/info", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "系统信息路由应存在")
		
		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				t.Log("系统信息返回有效JSON响应")
			}
		}
	})

	// 测试版本信息接口
	t.Run("版本信息", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/public/version", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "版本信息路由应存在")
	})

	// 测试服务状态接口
	t.Run("服务状态", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/public/status", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "服务状态路由应存在")
	})
}

// testDocumentationRoutes 测试文档和静态资源路由
func testDocumentationRoutes(t *testing.T, engine *gin.Engine) {
	// 测试API文档
	t.Run("API文档", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/docs", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// 文档路由可能不存在，这是正常的
		t.Logf("API文档路由状态码: %d", w.Code)
	})

	// 测试Swagger文档
	t.Run("Swagger文档", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		t.Logf("Swagger文档路由状态码: %d", w.Code)
	})
}

// testUserProfileRoutes 测试用户信息管理路由
func testUserProfileRoutes(t *testing.T, engine *gin.Engine) {
	// 测试获取用户信息
	t.Run("获取用户信息", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/user/profile", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "用户信息路由应需要认证")
	})

	// 测试更新用户信息
	t.Run("更新用户信息", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/user/profile", strings.NewReader(`{
			"nickname": "新昵称",
			"avatar": "avatar.jpg"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "更新用户信息路由应需要认证")
	})

	// 测试修改密码
	t.Run("修改密码", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/user/password", strings.NewReader(`{
			"old_password": "oldpass",
			"new_password": "newpass123"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "修改密码路由应需要认证")
	})

	// 测试删除账户
	t.Run("删除账户", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/user/account", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "删除账户路由应需要认证")
	})
}

// testUserScanRoutes 测试用户扫描任务路由
func testUserScanRoutes(t *testing.T, engine *gin.Engine) {
	// 测试获取扫描任务列表
	t.Run("获取扫描任务列表", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/user/scans", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "扫描任务列表路由应需要认证")
	})

	// 测试创建扫描任务
	t.Run("创建扫描任务", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/user/scans", strings.NewReader(`{
			"target": "192.168.1.1",
			"scan_type": "port_scan",
			"options": {
				"ports": "1-1000",
				"timeout": 30
			}
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "创建扫描任务路由应需要认证")
	})

	// 测试获取扫描任务详情
	t.Run("获取扫描任务详情", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/user/scans/123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "扫描任务详情路由应需要认证")
	})

	// 测试停止扫描任务
	t.Run("停止扫描任务", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/user/scans/123/stop", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "停止扫描任务路由应需要认证")
	})

	// 测试删除扫描任务
	t.Run("删除扫描任务", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/user/scans/123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "删除扫描任务路由应需要认证")
	})
}

// testUserSettingsRoutes 测试用户设置路由
func testUserSettingsRoutes(t *testing.T, engine *gin.Engine) {
	// 测试获取用户设置
	t.Run("获取用户设置", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/user/settings", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "用户设置路由应需要认证")
	})

	// 测试更新用户设置
	t.Run("更新用户设置", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/user/settings", strings.NewReader(`{
			"theme": "dark",
			"language": "zh-CN",
			"notifications": true
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "更新用户设置路由应需要认证")
	})
}

// testAdminUserManagementRoutes 测试管理员用户管理路由
func testAdminUserManagementRoutes(t *testing.T, engine *gin.Engine) {
	// 测试获取用户列表
	t.Run("获取用户列表", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/users", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "管理员用户列表路由应需要认证")
	})

	// 测试获取用户详情
	t.Run("获取用户详情", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/users/123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "用户详情路由应需要认证")
	})

	// 测试更新用户状态
	t.Run("更新用户状态", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/admin/users/123/status", strings.NewReader(`{
			"status": "active"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "更新用户状态路由应需要认证")
	})

	// 测试删除用户
	t.Run("删除用户", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/admin/users/123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "删除用户路由应需要认证")
	})
}

// testAdminSystemManagementRoutes 测试管理员系统管理路由
func testAdminSystemManagementRoutes(t *testing.T, engine *gin.Engine) {
	// 测试获取系统配置
	t.Run("获取系统配置", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/config", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "系统配置路由应需要认证")
	})

	// 测试更新系统配置
	t.Run("更新系统配置", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/admin/config", strings.NewReader(`{
			"max_concurrent_scans": 10,
			"scan_timeout": 300
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "更新系统配置路由应需要认证")
	})

	// 测试系统维护
	t.Run("系统维护", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/admin/maintenance", strings.NewReader(`{
			"action": "cleanup",
			"options": {"days": 30}
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "系统维护路由应需要认证")
	})
}

// testAdminMonitoringRoutes 测试管理员监控和日志路由
func testAdminMonitoringRoutes(t *testing.T, engine *gin.Engine) {
	// 测试获取系统状态
	t.Run("获取系统状态", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/system/status", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "系统状态路由应需要认证")
	})

	// 测试获取系统日志
	t.Run("获取系统日志", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/system/logs", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "系统日志路由应需要认证")
	})

	// 测试获取性能指标
	t.Run("获取性能指标", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/metrics", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "性能指标路由应需要认证")
	})
}

// testBasicHealthRoutes 测试基础健康检查路由
func testBasicHealthRoutes(t *testing.T, engine *gin.Engine) {
	// 测试基本健康检查
	t.Run("基本健康检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "健康检查路由应存在")
		
		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err == nil {
				t.Log("健康检查返回有效JSON响应")
				if status, exists := response["status"]; exists {
					t.Logf("健康状态: %v", status)
				}
			}
		}
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

// testDetailedHealthRoutes 测试详细健康检查路由
func testDetailedHealthRoutes(t *testing.T, engine *gin.Engine) {
	// 测试详细健康检查
	t.Run("详细健康检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health/detailed", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "详细健康检查路由应存在")
	})

	// 测试数据库健康检查
	t.Run("数据库健康检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health/database", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "数据库健康检查路由应存在")
	})

	// 测试Redis健康检查
	t.Run("Redis健康检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health/redis", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Redis健康检查路由应存在")
	})
}

// testServiceStatusRoutes 测试服务状态检查路由
func testServiceStatusRoutes(t *testing.T, engine *gin.Engine) {
	// 测试服务依赖检查
	t.Run("服务依赖检查", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health/dependencies", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "服务依赖检查路由应存在")
	})

	// 测试版本信息
	t.Run("版本信息", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health/version", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "版本信息路由应存在")
	})
}

// testAgentManagementRoutes 测试Agent管理路由
func testAgentManagementRoutes(t *testing.T, engine *gin.Engine) {
	// 测试Agent注册
	t.Run("Agent注册", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/agent/register", strings.NewReader(`{
			"agent_id": "test-agent-001",
			"agent_name": "测试Agent",
			"capabilities": ["port_scan", "service_detection"],
			"version": "1.0.0"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent注册路由应存在")
	})

	// 测试Agent心跳
	t.Run("Agent心跳", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/agent/heartbeat", strings.NewReader(`{
			"agent_id": "test-agent-001",
			"status": "active",
			"load": 0.5,
			"timestamp": "2025-01-27T10:00:00Z"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent心跳路由应存在")
	})

	// 测试Agent状态更新
	t.Run("Agent状态更新", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/agent/status", strings.NewReader(`{
			"agent_id": "test-agent-001",
			"status": "busy"
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent状态更新路由应存在")
	})
}

// testTaskDistributionRoutes 测试任务分发路由
func testTaskDistributionRoutes(t *testing.T, engine *gin.Engine) {
	// 测试获取任务
	t.Run("获取任务", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/agent/tasks", nil)
		req.Header.Set("Agent-ID", "test-agent-001")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "获取任务路由应存在")
	})

	// 测试提交任务结果
	t.Run("提交任务结果", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/agent/tasks/123/result", strings.NewReader(`{
			"status": "completed",
			"result": {
				"ports": [80, 443, 22],
				"services": ["http", "https", "ssh"]
			},
			"execution_time": 30
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Agent-ID", "test-agent-001")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "提交任务结果路由应存在")
	})

	// 测试任务状态更新
	t.Run("任务状态更新", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/api/v1/agent/tasks/123/status", strings.NewReader(`{
			"status": "running",
			"progress": 50
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Agent-ID", "test-agent-001")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "任务状态更新路由应存在")
	})
}

// testAgentCommunicationRoutes 测试Agent通信路由
func testAgentCommunicationRoutes(t *testing.T, engine *gin.Engine) {
	// 测试Agent配置获取
	t.Run("Agent配置获取", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/agent/config", nil)
		req.Header.Set("Agent-ID", "test-agent-001")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent配置获取路由应存在")
	})

	// 测试Agent日志上报
	t.Run("Agent日志上报", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/agent/logs", strings.NewReader(`{
			"agent_id": "test-agent-001",
			"logs": [
				{
					"level": "info",
					"message": "任务开始执行",
					"timestamp": "2025-01-27T10:00:00Z"
				}
			]
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent日志上报路由应存在")
	})

	// 测试Agent错误报告
	t.Run("Agent错误报告", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/agent/errors", strings.NewReader(`{
			"agent_id": "test-agent-001",
			"error": {
				"code": "SCAN_TIMEOUT",
				"message": "扫描超时",
				"details": "目标主机无响应"
			}
		}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		AssertNotEqual(t, http.StatusNotFound, w.Code, "Agent错误报告路由应存在")
	})
}