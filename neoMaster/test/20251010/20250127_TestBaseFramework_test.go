// TestBaseFramework测试文件
// 测试了测试基础框架功能，包括测试环境设置、数据库清理、Redis清理和测试用户/角色创建等
// 适配优化后的中间件模块结构
// 测试命令：无独立测试命令，为基础测试框架文件

// Package test 测试基础框架
// 提供测试环境设置和基础功能
package test

import (
	"context"
	"neomaster/internal/model/system"
	system2 "neomaster/internal/repository/mysql/system"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"neomaster/internal/app/master/middleware"
	"neomaster/internal/app/master/router"
	"neomaster/internal/config"
	pkgAuth "neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/database"
	redisRepo "neomaster/internal/repository/redis"
	authService "neomaster/internal/service/auth"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// TestConfig 测试配置结构
type TestConfig struct {
	DB    *gorm.DB         // 测试数据库连接
	Redis *redis.Client    // 测试Redis连接
	Cfg   *config.Config   // 测试配置
	JWT   *pkgAuth.JWTManager // JWT管理器
}

// TestSuite 测试套件，包含所有测试需要的依赖
// 适配优化后的中间件结构
type TestSuite struct {
	*TestConfig
	UserRepo          *system2.UserRepository       // 用户仓库
	SessionRepo       *redisRepo.SessionRepository  // 会话仓库
	UserService       *authService.UserService      // 用户服务
	JWTService        *authService.JWTService       // JWT服务
	SessionService    *authService.SessionService   // 会话服务
	RBACService       *authService.RBACService      // RBAC服务
	passwordManager   *pkgAuth.PasswordManager      // 密码管理器
	MiddlewareManager *middleware.MiddlewareManager // 中间件管理器，使用优化后的middleware包
	RouterManager     *router.Router                // 路由管理器
}

// SetupTestEnvironment 设置测试环境
// 初始化测试数据库、Redis连接和相关服务
// 适配优化后的中间件结构
func SetupTestEnvironment(t *testing.T) *TestSuite {
	// 设置测试环境变量
	os.Setenv("GO_ENV", "test")

	// 确保不使用环境变量重写配置，强制使用 configs/config.test.yaml
	unsetConfigEnvOverrides()

	// 加载配置文件 - 使用绝对路径
	configPath := filepath.Join("..", "..", "configs")
	if _, err := os.Stat("../../configs"); err == nil {
		configPath = "../../configs"
	}
	cfg, err := config.LoadConfig(configPath, "test")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 修改数据库配置为测试环境
	cfg.Database.MySQL.Database = "neoscan_test"
	// 保持原有密码配置，不修改密码

	// 尝试连接测试数据库，如果失败则跳过需要数据库的测试
	db, err := database.NewMySQLConnection(&cfg.Database.MySQL)
	if err != nil {
		// 如果数据库连接失败，创建一个nil的数据库连接用于单元测试
		t.Logf("警告: 无法连接到测试数据库，将跳过数据库相关测试: %v", err)
		db = nil
	}

	// 连接测试Redis
	redisClient, err := database.NewRedisConnection(&cfg.Database.Redis)
	if err != nil {
		t.Fatalf("连接测试Redis失败: %v", err)
	}

	// 创建JWT管理器
	// 注意：配置文件中的过期时间已经是time.Duration格式（如24h），不需要再乘以time.Hour
	jwtManager := pkgAuth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpire,
		cfg.JWT.RefreshTokenExpire,
	)

	// 初始化测试配置
	testConfig := &TestConfig{
		DB:    db,
		Redis: redisClient,
		Cfg:   cfg,
		JWT:   jwtManager,
	}

	// 如果数据库连接成功，设置测试数据库
	if db != nil {
		testConfig.SetupTestDatabase(t)
	}

	// 清理测试Redis
	testConfig.CleanupTestRedis(t)

	// 创建密码管理器
	passwordManager := pkgAuth.NewPasswordManager(nil)

	// 初始化仓库层
	var userRepo *system2.UserRepository
	var sessionRepo *redisRepo.SessionRepository

	if db != nil {
		userRepo = system2.NewUserRepository(db)
	}
	sessionRepo = redisRepo.NewSessionRepository(redisClient)

	// 创建服务实例 - 只有在userRepo不为nil时才创建
	var userService *authService.UserService
	var jwtService *authService.JWTService
	var authSvc *authService.SessionService
	var rbacService *authService.RBACService
	if userRepo != nil {
		// 首先创建用户服务
		userService = authService.NewUserService(
			userRepo,
			sessionRepo,
			passwordManager,
			jwtManager,
		)
		
		// 然后创建依赖用户服务的其他服务
		rbacService = authService.NewRBACService(userService)

		// 先创建SessionService（不传入JWTService）
		authSvc = authService.NewSessionService(
			userService,
			passwordManager,
			rbacService,
			sessionRepo,
		)

		// 再创建JWTService
		jwtService = authService.NewJWTService(jwtManager, userService, sessionRepo)

		// 设置SessionService的TokenGenerator（解决循环依赖）
		authSvc.SetTokenGenerator(jwtService)
	}

	// 创建中间件管理器 - 只有在所有服务都可用时才创建
	var middlewareManager *middleware.MiddlewareManager
	if authSvc != nil && rbacService != nil && jwtService != nil && userService != nil {
		middlewareManager = middleware.NewMiddlewareManager(authSvc, rbacService, jwtService, userService)
	}

	// 创建路由管理器 - 只有在中间件管理器可用时才创建
	var routerManager *router.Router
	if middlewareManager != nil {
		// 不创建应用实例，直接创建路由管理器
		routerManager = router.NewRouter(testConfig.DB, testConfig.Redis, testConfig.Cfg.JWT.Secret)
		routerManager.SetupRoutes()
	}

	return &TestSuite{
		TestConfig:        testConfig,
		UserRepo:          userRepo,
		SessionRepo:       sessionRepo,
		UserService:       userService,
		JWTService:        jwtService,
		SessionService:    authSvc,
		RBACService:       rbacService,
		passwordManager:   passwordManager,
		MiddlewareManager: middlewareManager,
		RouterManager:     routerManager,
	}
}

// unsetConfigEnvOverrides 取消设置可能影响配置加载的环境变量
func unsetConfigEnvOverrides() {
	envVars := []string{
		"MYSQL_HOST", "MYSQL_PORT", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_DATABASE",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"JWT_SECRET", "JWT_ACCESS_TOKEN_EXPIRE", "JWT_REFRESH_TOKEN_EXPIRE",
		"SERVER_HOST", "SERVER_PORT", "SERVER_MODE",
		"LOG_LEVEL", "LOG_FORMAT", "LOG_OUTPUT",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

// SetupTestDatabase 设置测试数据库
// 执行数据库迁移和创建默认角色
func (tc *TestConfig) SetupTestDatabase(t *testing.T) {
	if tc.DB == nil {
		t.Skip("跳过数据库设置：数据库连接不可用")
		return
	}

	// 执行数据库迁移
	if err := tc.migrateTestDatabase(); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 清理测试数据
	tc.CleanupTestDatabase(t)

	// 创建默认角色
	tc.createDefaultRoles(t)

	t.Logf("测试数据库设置完成")
}

// createDefaultRoles 创建默认角色
func (tc *TestConfig) createDefaultRoles(t *testing.T) {
	if tc.DB == nil {
		return
	}

	// 创建默认角色
	roles := []system.Role{
		{Name: "admin", Description: "系统管理员"},
		{Name: "user", Description: "普通用户"},
		{Name: "guest", Description: "访客用户"},
	}

	for _, role := range roles {
		var existingRole system.Role
		if err := tc.DB.Where("name = ?", role.Name).First(&existingRole).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tc.DB.Create(&role).Error; err != nil {
					t.Logf("创建角色 %s 失败: %v", role.Name, err)
				}
			}
		}
	}
}

// migrateTestDatabase 执行数据库迁移
func (tc *TestConfig) migrateTestDatabase() error {
	return tc.DB.AutoMigrate(
		&system.User{},
		&system.Role{},
		&system.Permission{},
	)
}

// CleanupTestDatabase 清理测试数据库
// 删除测试过程中创建的数据
func (tc *TestConfig) CleanupTestDatabase(t *testing.T) {
	if tc.DB == nil {
		return
	}

	// 清理用户角色关联表
	if err := tc.DB.Exec("DELETE FROM user_roles WHERE user_id > 0").Error; err != nil {
		t.Logf("清理用户角色关联表失败: %v", err)
	}

	// 清理角色权限关联表
	if err := tc.DB.Exec("DELETE FROM role_permissions WHERE role_id > 0").Error; err != nil {
		t.Logf("清理角色权限关联表失败: %v", err)
	}

	// 清理用户表（保留默认角色）
	if err := tc.DB.Where("id > 0").Delete(&system.User{}).Error; err != nil {
		t.Logf("清理用户表失败: %v", err)
	}

	// 清理权限表
	if err := tc.DB.Where("id > 0").Delete(&system.Permission{}).Error; err != nil {
		t.Logf("清理权限表失败: %v", err)
	}

	t.Logf("测试数据库清理完成")
}

// CleanupTestRedis 清理测试Redis
// 删除测试过程中创建的缓存数据
func (tc *TestConfig) CleanupTestRedis(t *testing.T) {
	if tc.Redis == nil {
		return
	}

	ctx := context.Background()

	// 清理会话相关的键
	sessionKeys, err := tc.Redis.Keys(ctx, "session:*").Result()
	if err == nil && len(sessionKeys) > 0 {
		tc.Redis.Del(ctx, sessionKeys...)
	}

	// 清理JWT相关的键
	jwtKeys, err := tc.Redis.Keys(ctx, "jwt:*").Result()
	if err == nil && len(jwtKeys) > 0 {
		tc.Redis.Del(ctx, jwtKeys...)
	}

	// 清理限流相关的键
	rateLimitKeys, err := tc.Redis.Keys(ctx, "rate_limit:*").Result()
	if err == nil && len(rateLimitKeys) > 0 {
		tc.Redis.Del(ctx, rateLimitKeys...)
	}

	t.Logf("测试Redis清理完成")
}

// TeardownTestEnvironment 清理测试环境
// 关闭数据库连接和Redis连接
func (ts *TestSuite) TeardownTestEnvironment(t *testing.T) {
	// 清理测试数据
	if ts.DB != nil {
		ts.CleanupTestDatabase(t)
	}
	ts.CleanupTestRedis(t)

	// 关闭数据库连接
	if ts.DB != nil {
		if sqlDB, err := ts.DB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// 关闭Redis连接
	if ts.Redis != nil {
		ts.Redis.Close()
	}

	t.Logf("测试环境清理完成")
}

// CreateTestUser 创建测试用户
// 用于测试环境中快速创建用户，跳过复杂的业务逻辑验证
func (ts *TestSuite) CreateTestUser(t *testing.T, username, email, password string) *system.User {
	// 如果数据库连接不可用，返回nil
	if ts.UserRepo == nil {
		t.Skip("跳过创建测试用户：数据库连接不可用")
		return nil
	}

	ctx := context.Background()

	// 哈希密码
	hashedPassword, err := ts.passwordManager.HashPassword(password)
	if err != nil {
		t.Fatalf("哈希密码失败: %v", err)
	}

	// 创建测试用户
	user := &system.User{
		Username:  username,
		Email:     email,
		Password:  hashedPassword, // 使用哈希后的密码
		Status:    system.UserStatusEnabled,
		PasswordV: 1,
	}

	// 保存到数据库（使用直接创建方法，不包含业务逻辑验证）
	err = ts.UserRepo.CreateUserDirect(ctx, user)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 确保将密码版本存储到Redis缓存中
	if ts.SessionRepo != nil {
		// 多次尝试存储密码版本到Redis
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			err := ts.SessionRepo.StorePasswordVersion(ctx, uint64(user.ID), user.PasswordV, time.Hour*24)
			if err == nil {
				break // 成功存储，退出重试循环
			}
			if i == maxRetries-1 {
				// 最后一次尝试失败，记录错误
				t.Logf("警告：经过%d次尝试后仍无法将密码版本存储到Redis: %v", maxRetries, err)
			} else {
				// 等待一段时间后重试
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	t.Logf("✅ 创建测试用户: %s (ID: %d)", username, user.ID)
	return user
}

// CreateTestRole 创建测试角色
// 用于测试中需要角色数据的场景
func (ts *TestSuite) CreateTestRole(t *testing.T, name, description string) *system.Role {
	if ts.DB == nil {
		t.Skip("跳过角色创建：数据库连接不可用")
		return nil
	}

	role := &system.Role{
		Name:        name,
		Description: description,
	}

	if err := ts.DB.Create(role).Error; err != nil {
		t.Fatalf("创建测试角色失败: %v", err)
	}

	t.Logf("创建测试角色成功: %s", name)
	return role
}

// AssignRoleToUser 为用户分配角色
// 用于测试中需要用户角色关联的场景
func (ts *TestSuite) AssignRoleToUser(t *testing.T, userID, roleID uint) {
	if ts.DB == nil {
		t.Skip("跳过角色分配：数据库连接不可用")
		return
	}

	// 创建用户角色关联
	if err := ts.DB.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error; err != nil {
		t.Fatalf("分配用户角色失败: %v", err)
	}

	t.Logf("用户角色分配成功: userID=%d, roleID=%d", userID, roleID)
}

// RunWithTestEnvironment 运行测试环境
// 提供统一的测试环境设置和清理
func RunWithTestEnvironment(t *testing.T, testFunc func(*TestSuite)) {
	ts := SetupTestEnvironment(t)
	defer ts.TeardownTestEnvironment(t)
	testFunc(ts)
}

// 测试断言辅助函数
func AssertNoError(t *testing.T, err error, message string) {
	if err != nil {
		t.Fatalf("%s: %v", message, err)
	}
}

func AssertError(t *testing.T, err error, message string) {
	if err == nil {
		t.Fatalf("%s: 期望有错误但没有错误", message)
	}
}

func AssertEqual(t *testing.T, expected, actual interface{}, message string) {
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("%s: 期望 %v, 实际 %v", message, expected, actual)
	}
}

func AssertNotEqual(t *testing.T, expected, actual interface{}, message string) {
	if reflect.DeepEqual(expected, actual) {
		t.Fatalf("%s: 期望不等于 %v, 但实际相等", message, expected)
	}
}

func AssertTrue(t *testing.T, value bool, message string) {
	if !value {
		t.Fatalf("%s: 期望为true但为false", message)
	}
}

func AssertFalse(t *testing.T, value bool, message string) {
	if value {
		t.Fatalf("%s: 期望为false但为true", message)
	}
}

func AssertNotNil(t *testing.T, value interface{}, message string) {
	if value == nil || (reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil()) {
		t.Fatalf("%s: 期望非nil但为nil", message)
	}
}

func AssertNil(t *testing.T, value interface{}, message string) {
	if value != nil && !(reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil()) {
		t.Fatalf("%s: 期望为nil但为非nil", message)
	}
}