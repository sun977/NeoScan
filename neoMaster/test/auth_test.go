package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"neomaster/internal/app/master"
	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/repository/redis"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// AuthTestSuite 认证测试套件
type AuthTestSuite struct {
	suite.Suite
	db          *gorm.DB
	redisClient *redis.Client
	router      *master.Router
	jwtSecret   string
	testUser    *model.User
}

// SetupSuite 测试套件初始化
func (suite *AuthTestSuite) SetupSuite() {
	// 初始化测试数据库（SQLite内存数据库）
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	// 自动迁移数据库表
	err = db.AutoMigrate(&model.User{}, &model.Role{}, &model.Permission{}, &model.UserRole{}, &model.RolePermission{})
	suite.Require().NoError(err)

	// 初始化测试Redis客户端（使用miniredis或mock）
	// 这里使用真实的Redis客户端，但连接到测试数据库
	suite.redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // 使用测试数据库
	})

	// 清空测试Redis数据库
	suite.redisClient.FlushDB(context.Background())

	// JWT密钥
	suite.jwtSecret = "test-jwt-secret-key-for-testing-only"

	// 初始化路由
	suite.router = master.NewRouter(suite.db, suite.redisClient, suite.jwtSecret)
	suite.router.SetupRoutes()

	// 创建测试用户
	suite.createTestUser()
}

// TearDownSuite 测试套件清理
func (suite *AuthTestSuite) TearDownSuite() {
	// 清空Redis测试数据
	suite.redisClient.FlushDB(context.Background())
	suite.redisClient.Close()
}

// createTestUser 创建测试用户
func (suite *AuthTestSuite) createTestUser() {
	passwordManager := password.NewPasswordManager()
	hashedPassword, err := passwordManager.HashPassword("testpassword123")
	suite.Require().NoError(err)

	testUser := &model.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  hashedPassword,
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = suite.db.Create(testUser).Error
	suite.Require().NoError(err)
	suite.testUser = testUser

	// 创建测试角色和权限
	suite.createTestRolesAndPermissions()
}

// createTestRolesAndPermissions 创建测试角色和权限
func (suite *AuthTestSuite) createTestRolesAndPermissions() {
	// 创建权限
	permissions := []model.Permission{
		{Name: "user:read", Description: "读取用户信息"},
		{Name: "user:write", Description: "修改用户信息"},
		{Name: "admin:read", Description: "管理员读取权限"},
		{Name: "admin:write", Description: "管理员写入权限"},
	}

	for _, perm := range permissions {
		err := suite.db.Create(&perm).Error
		suite.Require().NoError(err)
	}

	// 创建角色
	userRole := model.Role{
		Name:        "user",
		Description: "普通用户角色",
	}
	adminRole := model.Role{
		Name:        "admin",
		Description: "管理员角色",
	}

	err := suite.db.Create(&userRole).Error
	suite.Require().NoError(err)
	err = suite.db.Create(&adminRole).Error
	suite.Require().NoError(err)

	// 为角色分配权限
	// 用户角色权限
	userPermissions := []string{"user:read", "user:write"}
	for _, permName := range userPermissions {
		var perm model.Permission
		err := suite.db.Where("name = ?", permName).First(&perm).Error
		suite.Require().NoError(err)

		rolePermission := model.RolePermission{
			RoleID:       userRole.ID,
			PermissionID: perm.ID,
		}
		err = suite.db.Create(&rolePermission).Error
		suite.Require().NoError(err)
	}

	// 管理员角色权限
	adminPermissions := []string{"user:read", "user:write", "admin:read", "admin:write"}
	for _, permName := range adminPermissions {
		var perm model.Permission
		err := suite.db.Where("name = ?", permName).First(&perm).Error
		suite.Require().NoError(err)

		rolePermission := model.RolePermission{
			RoleID:       adminRole.ID,
			PermissionID: perm.ID,
		}
		err = suite.db.Create(&rolePermission).Error
		suite.Require().NoError(err)
	}

	// 为测试用户分配用户角色
	userRoleAssignment := model.UserRole{
		UserID: suite.testUser.ID,
		RoleID: userRole.ID,
	}
	err = suite.db.Create(&userRoleAssignment).Error
	suite.Require().NoError(err)
}

// TestUserLogin 测试用户登录
func (suite *AuthTestSuite) TestUserLogin() {
	// 准备登录请求
	loginReq := map[string]interface{}{
		"username": "testuser",
		"password": "testpassword123",
	}

	reqBody, err := json.Marshal(loginReq)
	suite.Require().NoError(err)

	// 发送登录请求
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应
	suite.Equal(http.StatusOK, w.Code)

	var response model.APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.True(response.Success)
	suite.NotNil(response.Data)

	// 验证返回的令牌
	data, ok := response.Data.(map[string]interface{})
	suite.True(ok)
	suite.NotEmpty(data["access_token"])
	suite.NotEmpty(data["refresh_token"])
}

// TestUserLoginInvalidCredentials 测试无效凭据登录
func (suite *AuthTestSuite) TestUserLoginInvalidCredentials() {
	// 准备无效登录请求
	loginReq := map[string]interface{}{
		"username": "testuser",
		"password": "wrongpassword",
	}

	reqBody, err := json.Marshal(loginReq)
	suite.Require().NoError(err)

	// 发送登录请求
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response model.APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.False(response.Success)
	suite.Contains(response.Message, "invalid")
}

// TestJWTAuthMiddleware 测试JWT认证中间件
func (suite *AuthTestSuite) TestJWTAuthMiddleware() {
	// 首先登录获取令牌
	accessToken := suite.loginAndGetToken()

	// 使用令牌访问受保护的端点
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应（应该成功通过认证）
	suite.Equal(http.StatusOK, w.Code)
}

// TestJWTAuthMiddlewareInvalidToken 测试无效令牌
func (suite *AuthTestSuite) TestJWTAuthMiddlewareInvalidToken() {
	// 使用无效令牌访问受保护的端点
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应（应该返回401）
	suite.Equal(http.StatusUnauthorized, w.Code)
}

// TestJWTAuthMiddlewareMissingToken 测试缺少令牌
func (suite *AuthTestSuite) TestJWTAuthMiddlewareMissingToken() {
	// 不提供令牌访问受保护的端点
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应（应该返回401）
	suite.Equal(http.StatusUnauthorized, w.Code)
}

// TestTokenRefresh 测试令牌刷新
func (suite *AuthTestSuite) TestTokenRefresh() {
	// 首先登录获取刷新令牌
	_, refreshToken := suite.loginAndGetTokens()

	// 准备刷新请求
	refreshReq := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	reqBody, err := json.Marshal(refreshReq)
	suite.Require().NoError(err)

	// 发送刷新请求
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应
	suite.Equal(http.StatusOK, w.Code)

	var response model.APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.True(response.Success)
	suite.NotNil(response.Data)

	// 验证返回的新令牌
	data, ok := response.Data.(map[string]interface{})
	suite.True(ok)
	suite.NotEmpty(data["access_token"])
}

// TestUserLogout 测试用户登出
func (suite *AuthTestSuite) TestUserLogout() {
	// 首先登录获取令牌
	accessToken := suite.loginAndGetToken()

	// 发送登出请求
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应
	suite.Equal(http.StatusOK, w.Code)

	var response model.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.True(response.Success)
}

// TestPermissionMiddleware 测试权限中间件
func (suite *AuthTestSuite) TestPermissionMiddleware() {
	// 这个测试需要创建一个需要特定权限的端点
	// 由于当前路由配置中的管理员端点需要admin角色，我们测试用户角色访问应该被拒绝
	accessToken := suite.loginAndGetToken()

	// 尝试访问管理员端点
	req := httptest.NewRequest("GET", "/api/v1/admin/users/list", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应（应该返回403，因为测试用户没有admin角色）
	suite.Equal(http.StatusForbidden, w.Code)
}

// TestHealthCheck 测试健康检查端点
func (suite *AuthTestSuite) TestHealthCheck() {
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)

	// 验证响应
	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	suite.Equal("healthy", response["status"])
}

// 辅助方法

// loginAndGetToken 登录并获取访问令牌
func (suite *AuthTestSuite) loginAndGetToken() string {
	accessToken, _ := suite.loginAndGetTokens()
	return accessToken
}

// loginAndGetTokens 登录并获取访问令牌和刷新令牌
func (suite *AuthTestSuite) loginAndGetTokens() (string, string) {
	loginReq := map[string]interface{}{
		"username": "testuser",
		"password": "testpassword123",
	}

	reqBody, err := json.Marshal(loginReq)
	suite.Require().NoError(err)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.GetRouter().ServeHTTP(w, req)
	suite.Require().Equal(http.StatusOK, w.Code)

	var response model.APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.Require().NoError(err)

	data, ok := response.Data.(map[string]interface{})
	suite.Require().True(ok)

	accessToken, ok := data["access_token"].(string)
	suite.Require().True(ok)

	refreshToken, ok := data["refresh_token"].(string)
	suite.Require().True(ok)

	return accessToken, refreshToken
}

// TestPasswordVersionControl 测试密码版本控制
func (suite *AuthTestSuite) TestPasswordVersionControl() {
	// 1. 用户登录获取token
	accessToken, _ := suite.loginAndGetTokens()

	// 2. 验证token有效
	req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	suite.router.GetRouter().ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// 3. 修改密码
	changePasswordReq := map[string]string{
		"old_password": "testpassword123",
		"new_password": "newpassword123",
	}
	body, _ := json.Marshal(changePasswordReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()
	suite.router.GetRouter().ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// 4. 验证旧token失效
	req = httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()
	suite.router.GetRouter().ServeHTTP(w, req)
	suite.Equal(http.StatusUnauthorized, w.Code)

	// 5. 使用新密码登录
	loginReq := map[string]string{
		"username": "testuser",
		"password": "newpassword123",
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.GetRouter().ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	// 6. 验证新token有效
	var loginResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResp)
	suite.NoError(err)
	newAccessToken := loginResp["data"].(map[string]interface{})["access_token"].(string)

	req = httptest.NewRequest("GET", "/api/v1/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+newAccessToken)
	w = httptest.NewRecorder()
	suite.router.GetRouter().ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)
}

// TestPasswordVersionInToken 测试token中包含密码版本
func (suite *AuthTestSuite) TestPasswordVersionInToken() {
	// 获取JWT管理器
	jwtManager := jwt.NewJWTManager(suite.jwtSecret, time.Hour, time.Hour*24)

	// 生成包含密码版本的token
	token, err := jwtManager.GenerateAccessToken(1, "testuser", "test@example.com", 1, []string{"user"})
	suite.NoError(err)
	suite.NotEmpty(token)

	// 验证token并检查密码版本
	claims, err := jwtManager.ValidateAccessToken(token)
	suite.NoError(err)
	suite.Equal(int64(1), claims.PasswordV)
	suite.Equal(uint(1), claims.UserID)
	suite.Equal("testuser", claims.Username)
}

// TestAuthTestSuite 运行认证测试套件
func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

// 单元测试

// TestJWTManager JWT管理器单元测试
func TestJWTManager(t *testing.T) {
	jwtSecret := "test-secret-key"
	jwtManager := auth.NewJWTManager(jwtSecret, time.Hour, time.Hour*24)

	// 测试生成访问令牌
	token, err := jwtManager.GenerateAccessToken(1, "testuser", "test@example.com", 1, []string{"user"})
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 测试验证访问令牌
	parsedClaims, err := jwtManager.ValidateAccessToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), parsedClaims.UserID)
	assert.Equal(t, "testuser", parsedClaims.Username)
	assert.Equal(t, "test@example.com", parsedClaims.Email)
	assert.Equal(t, int64(1), parsedClaims.PasswordV)
	assert.Equal(t, []string{"user"}, parsedClaims.Roles)

	// 测试生成刷新令牌
	refreshToken, err := jwtManager.GenerateRefreshToken(1, "testuser", 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, refreshToken)

	// 测试验证刷新令牌
	refreshClaims, err := jwtManager.ValidateRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), refreshClaims.UserID)
	assert.Equal(t, "testuser", refreshClaims.Username)
	assert.Equal(t, int64(1), refreshClaims.PasswordV)

	// 测试无效令牌
	_, err = jwtManager.ValidateAccessToken("invalid-token")
	assert.Error(t, err)
}

// TestPasswordManager 密码管理器单元测试
func TestPasswordManager(t *testing.T) {
	passwordManager := password.NewPasswordManager()

	// 测试密码哈希
	plainPassword := "testpassword123"
	hashedPassword, err := passwordManager.HashPassword(plainPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)
	assert.NotEqual(t, plainPassword, hashedPassword)

	// 测试密码验证
	isValid, err := passwordManager.VerifyPassword(hashedPassword, plainPassword)
	assert.NoError(t, err)
	assert.True(t, isValid)

	// 测试错误密码
	isValid, err = passwordManager.VerifyPassword(hashedPassword, "wrongpassword")
	assert.NoError(t, err)
	assert.False(t, isValid)
}

// BenchmarkPasswordHashing 密码哈希性能测试
func BenchmarkPasswordHashing(b *testing.B) {
	passwordManager := password.NewPasswordManager()
	plainPassword := "testpassword123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := passwordManager.HashPassword(plainPassword)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPasswordVerification 密码验证性能测试
func BenchmarkPasswordVerification(b *testing.B) {
	passwordManager := password.NewPasswordManager()
	plainPassword := "testpassword123"
	hashedPassword, _ := passwordManager.HashPassword(plainPassword)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := passwordManager.VerifyPassword(hashedPassword, plainPassword)
		if err != nil {
			b.Fatal(err)
		}
	}
}
