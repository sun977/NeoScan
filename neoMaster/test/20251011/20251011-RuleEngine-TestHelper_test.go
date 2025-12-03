/**
 * 规则引擎测试辅助函数
 * @author: Sun977
 * @date: 2025.10.11
 * @description: 规则引擎测试的辅助函数和工具，提供测试环境设置、数据准备等功能
 * @scope: 测试环境初始化、测试数据创建、JWT Token生成等
 */
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"neomaster/internal/model/system"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"neomaster/internal/model/orchestrator_drop"
)

// TestEnvironment 测试环境结构体
type TestEnvironment struct {
	DB          *gorm.DB
	RedisClient *redis.Client
	CleanupFunc func()
}

// setupTestEnvironment 设置测试环境
func setupTestEnvironment(t *testing.T) (*gorm.DB, *redis.Client, func()) {
	// 设置测试数据库连接
	testDBDSN := getTestDatabaseDSN()
	db, err := gorm.Open(mysql.Open(testDBDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 测试时静默日志
	})
	require.NoError(t, err, "测试数据库连接失败")

	// 设置测试Redis连接
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getTestRedisAddr(),
		Password: "",
		DB:       1, // 使用测试专用数据库
	})

	// 验证Redis连接
	_, err = redisClient.Ping(redisClient.Context()).Result()
	require.NoError(t, err, "测试Redis连接失败")

	// 初始化测试数据
	initTestData(t, db)

	// 返回清理函数
	cleanup := func() {
		cleanupTestData(t, db, redisClient)
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		redisClient.Close()
	}

	return db, redisClient, cleanup
}

// getTestDatabaseDSN 获取测试数据库DSN
func getTestDatabaseDSN() string {
	// 优先使用环境变量，否则使用默认测试配置
	if dsn := os.Getenv("TEST_DATABASE_DSN"); dsn != "" {
		return dsn
	}

	// 默认测试数据库配置 - 使用正确的MySQL密码ROOT
	return "root:ROOT@tcp(localhost:3306)/neoscan_test?charset=utf8mb4&parseTime=True&loc=Local"
}

// getTestRedisAddr 获取测试Redis地址
func getTestRedisAddr() string {
	if addr := os.Getenv("TEST_REDIS_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

// initTestData 初始化测试数据
func initTestData(t *testing.T, db *gorm.DB) {
	// 自动迁移测试表结构
	err := db.AutoMigrate(
		&system.User{},
		&system.Role{},
		&system.Permission{},
		&system.UserRole{},
		&system.RolePermission{},
		&orchestrator_drop.ScanRule{},
		&orchestrator_drop.ProjectConfig{},
		&orchestrator_drop.WorkflowConfig{},
		&orchestrator_drop.ScanTool{},
	)
	require.NoError(t, err, "测试数据库迁移失败")

	// 创建测试用户
	createTestUsers(t, db)

	// 创建测试角色和权限
	createTestRolesAndPermissions(t, db)

	// 创建测试扫描规则
	createTestScanRules(t, db)
}

// createTestUsers 创建测试用户
func createTestUsers(t *testing.T, db *gorm.DB) {
	testUsers := []system.User{
		{
			Username: "admin",
			Email:    "admin@test.com",
			Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
			Status:   system.UserStatusEnabled,
		},
		{
			Username: "testuser",
			Email:    "testuser@test.com",
			Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
			Status:   system.UserStatusEnabled,
		},
	}

	for _, user := range testUsers {
		// 检查用户是否已存在
		var existingUser system.User
		result := db.Where("username = ?", user.Username).First(&existingUser)
		if result.Error == gorm.ErrRecordNotFound {
			err := db.Create(&user).Error
			require.NoError(t, err, "创建测试用户失败: %s", user.Username)
		}
	}
}

// createTestRolesAndPermissions 创建测试角色和权限
func createTestRolesAndPermissions(t *testing.T, db *gorm.DB) {
	// 创建测试角色
	testRoles := []system.Role{
		{
			Name:        "admin",
			Description: "管理员角色",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "user",
			Description: "普通用户角色",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, role := range testRoles {
		var existingRole system.Role
		result := db.Where("name = ?", role.Name).First(&existingRole)
		if result.Error == gorm.ErrRecordNotFound {
			err := db.Create(&role).Error
			require.NoError(t, err, "创建测试角色失败: %s", role.Name)
		}
	}

	// 创建测试权限
	testPermissions := []system.Permission{
		{
			Name:        "rule_engine:execute",
			Description: "执行规则引擎",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "rule_engine:manage",
			Description: "管理规则引擎",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, permission := range testPermissions {
		var existingPermission system.Permission
		result := db.Where("name = ?", permission.Name).First(&existingPermission)
		if result.Error == gorm.ErrRecordNotFound {
			err := db.Create(&permission).Error
			require.NoError(t, err, "创建测试权限失败: %s", permission.Name)
		}
	}
}

// createTestScanRules 创建测试扫描规则
func createTestScanRules(t *testing.T, db *gorm.DB) {
	testRules := []orchestrator_drop.ScanRule{
		{
			Name:        "Test IP Filter Rule",
			Description: "测试IP过滤规则",
			Type:        orchestrator_drop.ScanRuleTypeFilter,
			Severity:    orchestrator_drop.ScanRuleSeverityMedium,
			Condition:   "request_ip == '192.168.1.1'",
			Action:      "log",
			Parameters:  "{}", // 提供有效的JSON字符串而不是空值
			Metadata:    "{}", // 提供有效的JSON字符串而不是空值
			Status:      orchestrator_drop.ScanRuleStatusEnabled,
		},
		{
			Name:        "Test User Agent Rule",
			Description: "测试User-Agent规则",
			Type:        orchestrator_drop.ScanRuleTypeFilter,
			Severity:    orchestrator_drop.ScanRuleSeverityLow,
			Condition:   "user_agent =~ '^Mozilla.*'",
			Action:      "allow",
			Parameters:  "{}", // 提供有效的JSON字符串而不是空值
			Metadata:    "{}", // 提供有效的JSON字符串而不是空值
			Status:      orchestrator_drop.ScanRuleStatusEnabled,
		},
	}

	for _, rule := range testRules {
		var existingRule orchestrator_drop.ScanRule
		result := db.Where("name = ?", rule.Name).First(&existingRule)
		if result.Error == gorm.ErrRecordNotFound {
			err := db.Create(&rule).Error
			require.NoError(t, err, "创建测试扫描规则失败: %s", rule.Name)
		}
	}
}

// cleanupTestData 清理测试数据
func cleanupTestData(t *testing.T, db *gorm.DB, redisClient *redis.Client) {
	// 清理数据库测试数据
	testTables := []string{
		"user_roles", "role_permissions", "scan_rules",
		"project_configs", "workflow_configs", "scan_tools",
		"users", "roles", "permissions",
	}

	for _, table := range testTables {
		err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE 1=1", table)).Error
		if err != nil {
			t.Logf("清理测试表 %s 失败: %v", table, err)
		}
	}

	// 清理Redis测试数据
	err := redisClient.FlushDB(redisClient.Context()).Err()
	if err != nil {
		t.Logf("清理Redis测试数据失败: %v", err)
	}
}

// createTestUserAndGetToken 创建测试用户并获取JWT Token
func createTestUserAndGetToken(t *testing.T, engine *gin.Engine) string {
	// 注册测试用户
	registerReq := map[string]interface{}{
		"username": "test_rule_engine_user",
		"email":    "rule_engine@test.com",
		"password": "test123456",
	}

	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 如果用户已存在，直接登录
	if w.Code != http.StatusOK {
		return loginAndGetToken(t, engine, "test_rule_engine_user", "test123456")
	}

	// 解析注册响应获取token
	var registerResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
	require.NoError(t, err, "注册响应解析失败")

	if data, ok := registerResponse["data"].(map[string]interface{}); ok {
		if token, ok := data["token"].(string); ok {
			return token
		}
	}

	// 如果注册响应中没有token，尝试登录获取
	return loginAndGetToken(t, engine, "test_rule_engine_user", "test123456")
}

// loginAndGetToken 登录并获取JWT Token
func loginAndGetToken(t *testing.T, engine *gin.Engine, username, password string) string {
	loginReq := map[string]interface{}{
		"username": username,
		"password": password,
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 打印响应内容以便调试
	t.Logf("登录响应状态码: %d", w.Code)
	t.Logf("登录响应内容: %s", w.Body.String())

	require.Equal(t, http.StatusOK, w.Code, "登录应该成功")

	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err, "登录响应解析失败")

	data, ok := loginResponse["data"].(map[string]interface{})
	require.True(t, ok, "登录响应应该包含data字段")

	// 根据实际响应结构，token字段名为access_token
	token, ok := data["access_token"].(string)
	require.True(t, ok, "登录响应应该包含access_token字段")
	require.NotEmpty(t, token, "Token不应该为空")

	return token
}

// createTestRuleViaAPI 通过API创建测试规则
func createTestRuleViaAPI(t *testing.T, engine *gin.Engine, token string, rule map[string]interface{}) uint {
	body, _ := json.Marshal(rule)
	req := httptest.NewRequest("POST", "/api/v1/orchestrator/rules", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 打印响应内容以便调试
	t.Logf("创建规则响应状态码: %d", w.Code)
	t.Logf("创建规则响应内容: %s", w.Body.String())

	require.Equal(t, http.StatusCreated, w.Code, "创建规则应该成功")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "创建规则响应解析失败")

	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "响应应该包含data字段")

	id, ok := data["id"].(float64)
	require.True(t, ok, "响应应该包含id字段")

	return uint(id)
}

// waitForCondition 等待条件满足（用于异步操作测试）
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timeoutChan:
			t.Fatalf("等待条件超时: %s", message)
		}
	}
}

// assertJSONResponse 断言JSON响应格式并返回解析后的数据
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	// 如果状态码不匹配，先输出响应内容以便调试
	if w.Code != expectedStatus {
		t.Logf("期望状态码: %d, 实际状态码: %d", expectedStatus, w.Code)
		t.Logf("响应内容: %s", w.Body.String())
	}

	require.Equal(t, expectedStatus, w.Code, "HTTP状态码不匹配")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "响应应该是有效的JSON格式")

	return response
}

// generateTestContext 生成测试上下文数据
func generateTestContext(index int) map[string]interface{} {
	return map[string]interface{}{
		"request_ip":     fmt.Sprintf("192.168.1.%d", 100+index),
		"request_method": "GET",
		"request_path":   fmt.Sprintf("/api/test/%d", index),
		"user_agent":     "Mozilla/5.0 (Test Agent)",
		"timestamp":      time.Now().Unix(),
		"request_size":   1024 + index*100,
		"response_time":  500 + index*50,
	}
}
