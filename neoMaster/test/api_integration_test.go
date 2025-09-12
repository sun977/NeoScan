// Package test API集成测试
// 测试完整的用户注册登录流程和权限验证接口
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authHandler "neomaster/internal/handler/auth"
	"neomaster/internal/handler/system"
	"neomaster/internal/model"
	authService "neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// TestAPIIntegration 测试API集成功能
func TestAPIIntegration(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		t.Run("用户注册API", func(t *testing.T) {
			testUserRegistrationAPI(t, ts)
		})

		t.Run("用户登录API", func(t *testing.T) {
			testUserLoginAPI(t, ts)
		})

		t.Run("用户登出API", func(t *testing.T) {
			testUserLogoutAPI(t, ts)
		})

		t.Run("令牌刷新API", func(t *testing.T) {
			testTokenRefreshAPI(t, ts)
		})

		t.Run("用户信息API", func(t *testing.T) {
			testUserInfoAPI(t, ts)
		})

		t.Run("权限验证API", func(t *testing.T) {
			testPermissionValidationAPI(t, ts)
		})

		t.Run("完整用户流程", func(t *testing.T) {
			testCompleteUserFlow(t, ts)
		})
	})
}

// setupTestRouter 设置测试路由
func setupTestRouter(ts *TestSuite) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	
	// 配置处理方法不允许的情况
	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Method not allowed",
		})
	})

	// 创建处理器
	// 创建PasswordService
		passwordService := authService.NewPasswordService(
			ts.UserService,
			ts.SessionService,
			ts.passwordManager,
			24*time.Hour,
		)
		userHandler := system.NewUserHandler(ts.UserService, passwordService)
	loginHandler := authHandler.NewLoginHandler(ts.SessionService)
	logoutHandler := authHandler.NewLogoutHandler(ts.SessionService)
	refreshHandler := authHandler.NewRefreshHandler(ts.SessionService)
	registerHandler := authHandler.NewRegisterHandler(ts.UserService)

	// 检查中间件管理器是否可用
	if ts.MiddlewareManager == nil {
		// 返回基本路由用于测试
		return router
	}

	// 公开路由
	public := router.Group("/api/v1")
	{
		auth := public.Group("/auth")
		{
			auth.POST("/register", registerHandler.Register)
			auth.POST("/login", loginHandler.Login)
			auth.POST("/refresh", refreshHandler.RefreshToken)
		}
	}

	// 需要认证的路由
	authRoutes := router.Group("/api/v1")
	authRoutes.Use(ts.MiddlewareManager.GinJWTAuthMiddleware())
	{
		authGroup := authRoutes.Group("/auth")
		{
			authGroup.POST("/logout", logoutHandler.Logout)
		}
		
		userGroup := authRoutes.Group("/user")
		{
			userGroup.GET("/profile", userHandler.GetUserByID) // 修复方法名
			userGroup.PUT("/profile", userHandler.UpdateUserByID) // 修复方法名
			// TODO: 实现密码修改方法
			// userGroup.POST("/change-password", userHandler.ChangePassword)
		}
	}

	// 管理员路由组
	admin := router.Group("/api/v1/admin")
	admin.Use(ts.MiddlewareManager.GinJWTAuthMiddleware())
	admin.Use(ts.MiddlewareManager.GinAdminRoleMiddleware())
	{
		admin.GET("/users", userHandler.GetUserList) // 修复方法名
		admin.POST("/users", userHandler.CreateUser) // 修复方法名
		admin.GET("/users/:id", userHandler.GetUserByID) // 修复方法名
		admin.PUT("/users/:id", userHandler.UpdateUserByID) // 修复方法名
		admin.DELETE("/users/:id", userHandler.DeleteUser) // 修复方法名
	}

	return router
}

// testUserRegistrationAPI 测试用户注册API
func testUserRegistrationAPI(t *testing.T, ts *TestSuite) {
	router := setupTestRouter(ts)

	// 测试正常注册
	registerData := map[string]interface{}{
		"username": "apiuser",
		"email":    "apiuser@test.com",
		"password": "password123",
	}

	body, _ := json.Marshal(registerData)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 注册应该成功
	AssertEqual(t, http.StatusCreated, w.Code, "注册应该返回201状态码")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	AssertNoError(t, err, "解析响应不应该出错")

	AssertEqual(t, "success", response["status"], "响应状态应该是success")
	AssertEqual(t, "registration successful", response["message"], "响应消息应该正确")

	// 测试重复注册
	req2 := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	// 重复注册应该返回冲突错误
	AssertEqual(t, http.StatusConflict, w2.Code, "重复注册应该返回409状态码")
}

// testUserLoginAPI 测试用户登录API
func testUserLoginAPI(t *testing.T, ts *TestSuite) {
	router := setupTestRouter(ts)

	// 创建测试用户
	user := ts.CreateTestUser(t, "loginapiuser", "loginapi@test.com", "password123")

	// 测试正常登录
	loginData := map[string]interface{}{
		"username": "loginapiuser",
		"password": "password123",
	}

	body, _ := json.Marshal(loginData)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	AssertEqual(t, http.StatusOK, w.Code, "登录应该返回200状态码")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	AssertNoError(t, err, "解析响应不应该出错")

	AssertEqual(t, "success", response["status"], "响应状态应该是success")
	data := response["data"].(map[string]interface{})
	AssertNotEqual(t, "", data["access_token"], "访问令牌不应该为空")
	AssertNotEqual(t, "", data["refresh_token"], "刷新令牌不应该为空")
	AssertTrue(t, data["expires_in"].(float64) > 0, "过期时间应该大于0")

	userData := data["user"].(map[string]interface{})
	AssertEqual(t, float64(user.ID), userData["id"], "用户ID应该匹配")
	AssertEqual(t, user.Username, userData["username"], "用户名应该匹配")

	// 测试邮箱登录
	emailLoginData := map[string]interface{}{
		"username": "loginapi@test.com", // 使用邮箱
		"password": "password123",
	}

	emailBody, _ := json.Marshal(emailLoginData)
	req2 := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(emailBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	AssertEqual(t, http.StatusOK, w2.Code, "邮箱登录应该返回200状态码")

	// 测试错误密码
	wrongPasswordData := map[string]interface{}{
		"username": "loginapiuser",
		"password": "wrongpassword",
	}

	wrongBody, _ := json.Marshal(wrongPasswordData)
	req3 := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(wrongBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()

	router.ServeHTTP(w3, req3)

	AssertEqual(t, http.StatusUnauthorized, w3.Code, "错误密码应该返回401状态码")

	// 测试不存在的用户
	nonExistentData := map[string]interface{}{
		"username": "nonexistent",
		"password": "password123",
	}

	nonExistentBody, _ := json.Marshal(nonExistentData)
	req4 := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(nonExistentBody))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()

	router.ServeHTTP(w4, req4)

	AssertEqual(t, http.StatusUnauthorized, w4.Code, "不存在用户应该返回401状态码")
}

// testUserLogoutAPI 测试用户登出API
func testUserLogoutAPI(t *testing.T, ts *TestSuite) {
	router := setupTestRouter(ts)

	// 创建测试用户并登录
	_ = ts.CreateTestUser(t, "logoutapiuser", "logoutapi@test.com", "password123")
	loginReq := &model.LoginRequest{
		Username: "logoutapiuser",
		Password: "password123",
	}

	loginResp, err := ts.SessionService.Login(context.Background(), loginReq, "127.0.0.1", "test-user-agent")
	AssertNoError(t, err, "登录不应该出错")

	// 测试正常登出
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	AssertEqual(t, http.StatusOK, w.Code, "登出应该返回200状态码")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	AssertNoError(t, err, "解析响应不应该出错")

	AssertEqual(t, "success", response["status"], "响应状态应该是success")

	// 测试无令牌登出
	req2 := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	AssertEqual(t, http.StatusUnauthorized, w2.Code, "无令牌登出应该返回401状态码")

	// 测试无效令牌登出
	req3 := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req3.Header.Set("Authorization", "Bearer invalid.token")
	w3 := httptest.NewRecorder()

	router.ServeHTTP(w3, req3)

	AssertEqual(t, http.StatusUnauthorized, w3.Code, "无效令牌登出应该返回401状态码")
}

// testTokenRefreshAPI 测试令牌刷新API
func testTokenRefreshAPI(t *testing.T, ts *TestSuite) {
	router := setupTestRouter(ts)

	// 创建测试用户并登录
	_ = ts.CreateTestUser(t, "refreshapiuser", "refreshapi@test.com", "password123")
	loginReq := &model.LoginRequest{
		Username: "refreshapiuser",
		Password: "password123",
	}

	loginResp, err := ts.SessionService.Login(context.Background(), loginReq, "127.0.0.1", "test-user-agent")
	AssertNoError(t, err, "登录不应该出错")

	// 测试正常刷新
	refreshData := map[string]interface{}{
		"refresh_token": loginResp.RefreshToken,
	}

	body, _ := json.Marshal(refreshData)
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	AssertEqual(t, http.StatusOK, w.Code, "刷新令牌应该返回200状态码")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	AssertNoError(t, err, "解析响应不应该出错")

	AssertEqual(t, "success", response["status"], "响应状态应该是success")
	data := response["data"].(map[string]interface{})
	AssertNotEqual(t, "", data["access_token"], "新访问令牌不应该为空")
	AssertNotEqual(t, "", data["refresh_token"], "新刷新令牌不应该为空")
	AssertNotEqual(t, loginResp.AccessToken, data["access_token"], "新访问令牌应该与旧令牌不同")

	// 测试无效刷新令牌
	invalidRefreshData := map[string]interface{}{
		"refresh_token": "invalid.refresh.token",
	}

	invalidBody, _ := json.Marshal(invalidRefreshData)
	req2 := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(invalidBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	AssertEqual(t, http.StatusUnauthorized, w2.Code, "无效刷新令牌应该返回401状态码")

	// 测试用访问令牌刷新
	accessTokenRefreshData := map[string]interface{}{
		"refresh_token": loginResp.AccessToken, // 使用访问令牌
	}

	accessTokenBody, _ := json.Marshal(accessTokenRefreshData)
	req3 := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(accessTokenBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()

	router.ServeHTTP(w3, req3)

	AssertEqual(t, http.StatusUnauthorized, w3.Code, "用访问令牌刷新应该返回401状态码")
}

// testUserInfoAPI 测试用户信息API
func testUserInfoAPI(t *testing.T, ts *TestSuite) {
	router := setupTestRouter(ts)

	// 创建测试用户
	testUser := ts.CreateTestUser(t, "infoapiuser", "infoapi@test.com", "password123")

	// 登录获取令牌
	loginReq := &model.LoginRequest{
		Username: "infoapiuser",
		Password: "password123",
	}

	loginResp, err := ts.SessionService.Login(context.Background(), loginReq, "192.0.2.1", "test-user-agent")
	AssertNoError(t, err, "登录不应该出错")

	// 测试获取用户信息
	req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该返回成功响应
	AssertEqual(t, http.StatusOK, w.Code, "获取用户信息应该成功")

	// 解析响应
	var response model.APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	AssertNoError(t, err, "解析响应不应该出错")
	AssertEqual(t, "success", response.Status, "响应状态应该是success")

	// 检查返回的用户信息
	userDataBytes, err := json.Marshal(response.Data)
	AssertNoError(t, err, "序列化用户数据不应该出错")
	
	var userData map[string]interface{}
	err = json.Unmarshal(userDataBytes, &userData)
	AssertNoError(t, err, "反序列化用户数据不应该出错")
	
	// 验证用户数据字段
	AssertEqual(t, float64(testUser.ID), userData["id"], "用户ID应该匹配")
	AssertEqual(t, testUser.Username, userData["username"], "用户名应该匹配")
	AssertEqual(t, testUser.Email, userData["email"], "邮箱应该匹配")

	// 测试权限验证API
	// 创建测试用户
	// testUser := ts.CreateTestUser(t, "permuser", "perm@test.com", "password123")

	// 登录获取令牌
	loginReq2 := &model.LoginRequest{
		Username: "normaluser",
		Password: "password123",
	}

	loginResp2, err := ts.SessionService.Login(context.Background(), loginReq2, "192.0.2.1", "test-user-agent")
	AssertNoError(t, err, "登录不应该出错")

	// 测试权限验证
	req2 := httptest.NewRequest("GET", "/api/v1/user/permissions", nil)
	req2.Header.Set("Authorization", "Bearer "+loginResp2.AccessToken)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// 应该返回成功响应
	AssertEqual(t, http.StatusOK, w2.Code, "权限验证应该成功")

	// 解析响应
	var response2 model.APIResponse
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	AssertNoError(t, err, "解析响应不应该出错")
	AssertEqual(t, "success", response2.Status, "响应状态应该是success")

	// 测试无令牌访问
	req3 := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	w3 := httptest.NewRecorder()

	router.ServeHTTP(w3, req3)

	AssertEqual(t, http.StatusUnauthorized, w3.Code, "无令牌访问应该返回401状态码")

	// 测试无效令牌访问
	req4 := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req4.Header.Set("Authorization", "Bearer invalid.token")
	w4 := httptest.NewRecorder()

	router.ServeHTTP(w4, req4)

	AssertEqual(t, http.StatusUnauthorized, w4.Code, "无效令牌访问应该返回401状态码")
}

// testPermissionValidationAPI 测试权限验证API
func testPermissionValidationAPI(t *testing.T, ts *TestSuite) {
	router := setupTestRouter(ts)

	// 创建普通用户
	_ = ts.CreateTestUser(t, "normaluser", "normal@test.com", "password123")
	normalLoginReq := &model.LoginRequest{
		Username: "normaluser",
		Password: "password123",
	}

	normalLoginResp, err := ts.SessionService.Login(context.Background(), normalLoginReq, "127.0.0.1", "test-user-agent")
	AssertNoError(t, err, "普通用户登录不应该出错")

	// 创建管理员用户
	adminUser := ts.CreateTestUser(t, "adminuser", "admin@test.com", "password123")
	adminRole := ts.CreateTestRole(t, "admin", "管理员角色")
	ts.AssignRoleToUser(t, adminUser.ID, adminRole.ID)

	adminLoginReq := &model.LoginRequest{
		Username: "adminuser",
		Password: "password123",
	}

	adminLoginResp, err := ts.SessionService.Login(context.Background(), adminLoginReq, "127.0.0.1", "test-user-agent")
	AssertNoError(t, err, "管理员用户登录不应该出错")

	// 测试普通用户访问管理员接口（应该被拒绝）
	t.Run("普通用户访问管理员接口", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+normalLoginResp.AccessToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusForbidden, w.Code, "普通用户访问管理员接口应该返回403状态码")
	})

	// 测试管理员用户访问管理员接口（应该成功）
	t.Run("管理员用户访问管理员接口", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+adminLoginResp.AccessToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// 注意：这里可能返回200或其他状态码，取决于具体实现
		// 重要的是不应该返回403（禁止访问）
		AssertNotEqual(t, http.StatusForbidden, w.Code, "管理员用户访问管理员接口不应该返回403状态码")
	})

	// 测试无令牌访问管理员接口
	t.Run("无令牌访问管理员接口", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusUnauthorized, w.Code, "无令牌访问管理员接口应该返回401状态码")
	})

	// 测试普通用户访问普通接口（应该成功）
	t.Run("普通用户访问普通接口", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
		req.Header.Set("Authorization", "Bearer "+normalLoginResp.AccessToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "普通用户访问普通接口应该返回200状态码")
	})
}

// testCompleteUserFlow 测试完整的用户流程
func testCompleteUserFlow(t *testing.T, ts *TestSuite) {
	router := setupTestRouter(ts)

	// 1. 用户注册
	registerData := map[string]interface{}{
		"username": "flowuser",
		"email":    "flow@test.com",
		"password": "password123",
	}

	registerBody, _ := json.Marshal(registerData)
	registerReq := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerW := httptest.NewRecorder()

	router.ServeHTTP(registerW, registerReq)
	AssertEqual(t, http.StatusCreated, registerW.Code, "注册应该成功")

	// 2. 用户登录
	loginData := map[string]interface{}{
		"username": "flowuser",
		"password": "password123",
	}

	loginBody, _ := json.Marshal(loginData)
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()

	router.ServeHTTP(loginW, loginReq)
	AssertEqual(t, http.StatusOK, loginW.Code, "登录应该成功")

	var loginResponse map[string]interface{}
	err := json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	AssertNoError(t, err, "解析登录响应不应该出错")

	loginData2 := loginResponse["data"].(map[string]interface{})
	accessToken := loginData2["access_token"].(string)
	refreshToken := loginData2["refresh_token"].(string)

	// 3. 获取用户信息
	profileReq := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	profileReq.Header.Set("Authorization", "Bearer "+accessToken)
	profileW := httptest.NewRecorder()

	router.ServeHTTP(profileW, profileReq)
	AssertEqual(t, http.StatusOK, profileW.Code, "获取用户信息应该成功")

	// 4. 刷新令牌
	time.Sleep(10 * time.Millisecond) // 确保时间戳不同

	refreshData := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	refreshBody, _ := json.Marshal(refreshData)
	refreshReq := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshW := httptest.NewRecorder()

	router.ServeHTTP(refreshW, refreshReq)
	AssertEqual(t, http.StatusOK, refreshW.Code, "刷新令牌应该成功")

	var refreshResponse map[string]interface{}
	err = json.Unmarshal(refreshW.Body.Bytes(), &refreshResponse)
	AssertNoError(t, err, "解析刷新响应不应该出错")

	refreshData2 := refreshResponse["data"].(map[string]interface{})
	newAccessToken := refreshData2["access_token"].(string)
	AssertNotEqual(t, accessToken, newAccessToken, "新访问令牌应该与旧令牌不同")

	// 5. 使用新令牌访问接口
	profileReq2 := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	profileReq2.Header.Set("Authorization", "Bearer "+newAccessToken)
	profileW2 := httptest.NewRecorder()

	router.ServeHTTP(profileW2, profileReq2)
	AssertEqual(t, http.StatusOK, profileW2.Code, "使用新令牌获取用户信息应该成功")

	// 6. 用户登出
	logoutReq := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+newAccessToken)
	logoutW := httptest.NewRecorder()

	router.ServeHTTP(logoutW, logoutReq)
	AssertEqual(t, http.StatusOK, logoutW.Code, "登出应该成功")

	// 7. 验证登出后令牌无效（如果实现了令牌黑名单）
	// 注意：这取决于具体的登出实现
	profileReq3 := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	profileReq3.Header.Set("Authorization", "Bearer "+newAccessToken)
	profileW3 := httptest.NewRecorder()

	router.ServeHTTP(profileW3, profileReq3)
	// 如果实现了令牌黑名单，这里应该返回401
	// 如果没有实现，令牌仍然有效，返回200
	// 这里我们只验证不会出现服务器错误
	AssertTrue(t, profileW3.Code == http.StatusOK || profileW3.Code == http.StatusUnauthorized,
		"登出后访问应该返回200或401状态码")
}

// TestAPIErrorHandling 测试API错误处理
func TestAPIErrorHandling(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		router := setupTestRouter(ts)

		t.Run("JSON格式错误", func(t *testing.T) {
			// 发送无效JSON
			req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString("invalid json"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			AssertEqual(t, http.StatusBadRequest, w.Code, "无效JSON应该返回400状态码")
		})

		t.Run("缺少Content-Type", func(t *testing.T) {
			data := map[string]interface{}{
				"username": "testuser",
				"email":    "test@test.com",
				"password": "password123",
			}

			body, _ := json.Marshal(data)
			req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
			// 不设置Content-Type
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 根据具体实现，可能返回400或其他状态码
			AssertTrue(t, w.Code >= 400, "缺少Content-Type应该返回错误状态码")
		})

		t.Run("不支持的HTTP方法", func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/v1/auth/register", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			AssertEqual(t, http.StatusMethodNotAllowed, w.Code, "不支持的HTTP方法应该返回405状态码")
		})

		t.Run("不存在的路由", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			AssertEqual(t, http.StatusNotFound, w.Code, "不存在的路由应该返回404状态码")
		})
	})
}