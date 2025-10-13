// DeprecatedAPI测试文件
// 测试了已弃用的API接口，包括登出接口的功能和错误处理
// 测试命令：go test -v -run TestDeprecatedAPI ./test

// Package test 已弃用API测试
// 测试NeoScan Master v4.0中已弃用的API接口
package test

import (
	"context"
	"encoding/json"
	"neomaster/internal/model/system"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	authHandler "neomaster/internal/handler/auth"
)

// TestDeprecatedAPI 测试已弃用的API接口
func TestDeprecatedAPI(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		// 检查必要的服务是否可用
		if ts.UserService == nil || ts.SessionService == nil {
			t.Skip("跳过已弃用API测试：数据库连接失败，必要的服务不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		// 创建测试路由器
		router := setupDeprecatedTestRouter(ts)

		t.Run("已弃用的登出接口", func(t *testing.T) {
			testDeprecatedLogoutAPI(t, router, ts)
		})
	})
}

// setupDeprecatedTestRouter 设置已弃用API测试路由
func setupDeprecatedTestRouter(ts *TestSuite) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// 配置处理方法不允许的情况
	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Method not allowed",
		})
	})

	// 检查必要的服务是否可用
	if ts.UserService == nil || ts.SessionService == nil {
		// 返回一个基本的路由器，不注册任何需要服务的路由
		return router
	}

	// 创建处理器
	loginHandler := authHandler.NewLoginHandler(ts.SessionService)
	logoutHandler := authHandler.NewLogoutHandler(ts.SessionService)

	// 认证路由
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/login", loginHandler.Login)
		// 已弃用的登出接口
		auth.POST("/logout", logoutHandler.Logout)
	}

	return router
}

// testDeprecatedLogoutAPI 测试已弃用的登出接口
func testDeprecatedLogoutAPI(t *testing.T, router *gin.Engine, ts *TestSuite) {
	// 创建测试用户并登录
	_ = ts.CreateTestUser(t, "deprecateduser", "deprecated@example.com", "password123")
	loginReq := &system.LoginRequest{
		Username: "deprecateduser",
		Password: "password123",
	}

	loginResp, err := ts.SessionService.Login(context.Background(), loginReq, "192.0.2.1", "test-user-agent")
	assert.NoError(t, err, "登录不应该出错")

	accessToken := loginResp.AccessToken

	// 测试已弃用的登出接口
	t.Run("测试已弃用的登出接口", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 根据API文档，已弃用的登出接口应该仍然可以工作
		// 但我们期望它返回200状态码
		assert.Equal(t, http.StatusOK, w.Code, "已弃用的登出接口应该返回200状态码")

		var logoutResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &logoutResp)
		assert.NoError(t, err, "解析已弃用登出接口响应不应该出错")
		
		// 检查响应格式是否符合v4.0规范
		if code, ok := logoutResp["code"]; ok {
			assert.Equal(t, float64(200), code, "已弃用登出接口响应代码应该是200")
		}
		
		if status, ok := logoutResp["status"]; ok {
			assert.Equal(t, "success", status, "已弃用登出接口响应状态应该是success")
		}
		
		if message, ok := logoutResp["message"]; ok {
			assert.Equal(t, "logout successful", message, "已弃用登出接口响应消息应该是'logout successful'")
		}
	})

	// 测试无令牌访问已弃用的登出接口
	t.Run("无令牌访问已弃用登出接口", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 无令牌访问应该返回401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code, "无令牌访问已弃用登出接口应该返回401状态码")
	})

	// 测试无效令牌访问已弃用的登出接口
	t.Run("无效令牌访问已弃用登出接口", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.string")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 无效令牌访问应该返回401 Unauthorized或500 Internal Server Error（取决于实现）
		assert.Contains(t, []int{http.StatusUnauthorized, http.StatusInternalServerError}, w.Code, 
			"无效令牌访问已弃用登出接口应该返回401或500状态码，实际返回: %d", w.Code)
	})
}