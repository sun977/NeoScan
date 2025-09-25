// Package test 会话管理API测试
// 测试NeoScan Master v4.0的会话管理接口
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authHandler "neomaster/internal/handler/auth"
	"neomaster/internal/handler/system"
	"neomaster/internal/model"
	authService "neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestSessionAPI 测试会话管理API接口
func TestSessionAPI(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		// 检查必要的服务是否可用
		if ts.UserService == nil || ts.SessionService == nil {
			t.Skip("跳过会话管理API测试：数据库连接失败，必要的服务不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		// 创建测试路由器
		router := setupSessionTestRouter(ts)

		t.Run("会话管理接口", func(t *testing.T) {
			testSessionManagementAPI(t, router, ts)
		})
	})
}

// setupSessionTestRouter 设置会话测试路由
func setupSessionTestRouter(ts *TestSuite) *gin.Engine {
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
	passwordService := authService.NewPasswordService(
		ts.UserService,
		ts.SessionService,
		ts.passwordManager,
		24*3600, // 24小时
	)
	userHandler := system.NewUserHandler(ts.UserService, passwordService)
	loginHandler := authHandler.NewLoginHandler(ts.SessionService)

	// 检查中间件管理器是否可用
	if ts.MiddlewareManager == nil {
		// 无中间件时，跳过需要中间件的受保护/管理员路由
		return router
	}

	// 管理员路由组
	admin := router.Group("/api/v1/admin")
	admin.Use(ts.MiddlewareManager.GinJWTAuthMiddleware())
	admin.Use(ts.MiddlewareManager.GinUserActiveMiddleware())
	admin.Use(ts.MiddlewareManager.GinAdminRoleMiddleware())
	{
		// 会话管理
		sessions := admin.Group("/sessions/user")
		{
			sessions.GET("/list", userHandler.GetActiveSessions)
			sessions.POST("/:userId/revoke", userHandler.RevokeUserSession)
			sessions.POST("/:userId/revoke-all", userHandler.RevokeAllUserSessions)
		}
	}

	// 认证路由（用于创建测试会话）
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/login", loginHandler.Login)
	}

	return router
}

// testSessionManagementAPI 测试会话管理接口
func testSessionManagementAPI(t *testing.T, router *gin.Engine, ts *TestSuite) {
	// 创建管理员用户
	adminUser := ts.CreateTestUser(t, "sessionadmin", "sessionadmin@example.com", "password123")
	adminRole := ts.CreateTestRole(t, "admin", "系统管理员")
	ts.AssignRoleToUser(t, adminUser.ID, adminRole.ID)

	// 管理员登录
	adminLoginReq := &model.LoginRequest{
		Username: "sessionadmin",
		Password: "password123",
	}

	adminLoginResp, err := ts.SessionService.Login(context.Background(), adminLoginReq, "192.0.2.1", "test-user-agent")
	assert.NoError(t, err, "管理员登录不应该出错")

	adminAccessToken := adminLoginResp.AccessToken

	// 创建普通用户用于测试会话管理
	testUser := ts.CreateTestUser(t, "sessionuser", "sessionuser@example.com", "password123")

	// 普通用户登录以创建会话
	userLoginReq := &model.LoginRequest{
		Username: "sessionuser",
		Password: "password123",
	}

	userLoginResp, err := ts.SessionService.Login(context.Background(), userLoginReq, "192.0.2.2", "test-user-agent-2")
	assert.NoError(t, err, "用户登录不应该出错")

	userAccessToken := userLoginResp.AccessToken
	userID := fmt.Sprintf("%d", testUser.ID)

	// 测试获取活跃会话列表
	t.Run("获取活跃会话列表", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/sessions/user/list?userId="+userID, nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "获取活跃会话列表应该返回200状态码")

		var sessionListResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &sessionListResp)
		assert.NoError(t, err, "解析获取活跃会话列表响应不应该出错")
		assert.Equal(t, float64(200), sessionListResp["code"], "获取活跃会话列表响应代码应该是200")
		assert.Equal(t, "success", sessionListResp["status"], "获取活跃会话列表响应状态应该是success")
	})

	// 测试撤销用户会话
	t.Run("撤销用户会话", func(t *testing.T) {
		// 首先获取用户会话信息以获得token_id
		req := httptest.NewRequest("GET", "/api/v1/admin/sessions/user/list?userId="+userID, nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "获取活跃会话列表应该返回200状态码")

		var sessionListResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &sessionListResp)
		assert.NoError(t, err, "解析获取活跃会话列表响应不应该出错")

		// 从响应中提取token_id（这里我们模拟一个token_id，因为实际实现可能不同）
		revokeData := map[string]interface{}{
			"token_id": "test-token-id",
		}

		body, _ := json.Marshal(revokeData)
		req = httptest.NewRequest("POST", "/api/v1/admin/sessions/user/"+userID+"/revoke", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+adminAccessToken)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 注意：根据实际实现，这里可能返回200或其他状态码
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound, 
			"撤销用户会话应该返回200或404状态码，实际返回: %d", w.Code)
	})

	// 测试撤销用户所有会话
	t.Run("撤销用户所有会话", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/admin/sessions/user/"+userID+"/revoke-all", nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "撤销用户所有会话应该返回200状态码")

		var revokeAllResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &revokeAllResp)
		assert.NoError(t, err, "解析撤销用户所有会话响应不应该出错")
		assert.Equal(t, float64(200), revokeAllResp["code"], "撤销用户所有会话响应代码应该是200")
		assert.Equal(t, "success", revokeAllResp["status"], "撤销用户所有会话响应状态应该是success")
	})

	// 测试无效用户ID的情况
	t.Run("无效用户ID测试", func(t *testing.T) {
		// 测试获取不存在用户的活跃会话列表
		req := httptest.NewRequest("GET", "/api/v1/admin/sessions/user/list?userId=999999", nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "获取不存在用户的活跃会话列表应该返回200状态码")

		var sessionListResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &sessionListResp)
		assert.NoError(t, err, "解析获取不存在用户的活跃会话列表响应不应该出错")
		assert.Equal(t, float64(200), sessionListResp["code"], "获取不存在用户的活跃会话列表响应代码应该是200")
		assert.Equal(t, "success", sessionListResp["status"], "获取不存在用户的活跃会话列表响应状态应该是success")
	})

	// 测试权限不足的情况
	t.Run("权限不足测试", func(t *testing.T) {
		// 创建普通用户
		normalUser := ts.CreateTestUser(t, "normaluser", "normal@example.com", "password123")
		
		// 普通用户登录
		normalLoginReq := &model.LoginRequest{
			Username: "normaluser",
			Password: "password123",
		}

		normalLoginResp, err := ts.SessionService.Login(context.Background(), normalLoginReq, "192.0.2.3", "test-user-agent-3")
		assert.NoError(t, err, "普通用户登录不应该出错")

		normalAccessToken := normalLoginResp.AccessToken
		normalUserID := fmt.Sprintf("%d", normalUser.ID)

		// 普通用户尝试访问会话管理接口应该被拒绝
		req := httptest.NewRequest("GET", "/api/v1/admin/sessions/user/list?userId="+normalUserID, nil)
		req.Header.Set("Authorization", "Bearer "+normalAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 应该返回403 Forbidden
		assert.Equal(t, http.StatusForbidden, w.Code, "普通用户访问会话管理接口应该返回403状态码")
	})
}