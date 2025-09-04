// Package test 提供测试基础设施和通用测试工具
// 包含测试数据库配置、初始化和清理函数
package test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"neomaster/internal/app/master"
	"neomaster/internal/config"
	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/database"
	"neomaster/internal/repository/mysql"
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
	JWT   *auth.JWTManager // JWT管理器
}

// TestSuite 测试套件，包含所有测试需要的依赖
type TestSuite struct {
	*TestConfig
	UserRepo          *mysql.UserRepository        // 用户仓库（包含业务逻辑）
	SessionRepo       *redisRepo.SessionRepository // 会话仓库
	JWTService        *authService.JWTService      // JWT服务
	AuthService       *authService.SessionService  // 认证服务
	RBACService       *authService.RBACService     // RBAC服务
	passwordManager   *auth.PasswordManager        // 密码管理器
	SessionService    *authService.SessionService  // 会话服务（别名）
	UserService       *authService.UserService     // 用户服务
	MiddlewareManager *master.MiddlewareManager    // 中间件管理器
}

// SetupTestEnvironment 设置测试环境
// 初始化测试数据库、Redis连接和相关服务
func SetupTestEnvironment(t *testing.T) *TestSuite {
	// 设置测试环境变量
	os.Setenv("GO_ENV", "test")

	// 加载配置文件 - 使用默认的config.yaml文件
	cfg, err := config.LoadConfig("../configs", "development")
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
	jwtManager := auth.NewJWTManager(
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

	// 创建仓库实例 - 如果数据库连接失败则创建nil仓库
	var userRepo *mysql.UserRepository
	passwordManager := auth.NewPasswordManager(nil)
	if db != nil {
		userRepo = mysql.NewUserRepository(db)
	} else {
		// 数据库连接失败时，userRepo保持为nil
		userRepo = nil
	}

	sessionRepo := redisRepo.NewSessionRepository(redisClient)

	// 密码管理器已在上面创建

	// 创建RBAC服务 - 只有在userRepo不为nil时才创建
	var rbacService *authService.RBACService
	if userRepo != nil {
		rbacService = authService.NewRBACService(userRepo)
	}

	// 创建服务实例 - 只有在userRepo不为nil时才创建
	var jwtService *authService.JWTService
	var authSvc *authService.SessionService
	var userService *authService.UserService
	if userRepo != nil {
		jwtService = authService.NewJWTService(jwtManager, userRepo)
		authSvc = authService.NewSessionService(
			userRepo,
			passwordManager,
			jwtService,
			rbacService,
			sessionRepo,
		)
		userService = authService.NewUserService(
			userRepo,
			sessionRepo,
			passwordManager,
			testConfig.JWT,
		)
	}

	// 创建中间件管理器 - 只有在所有服务都可用时才创建
	var middlewareManager *master.MiddlewareManager
	if authSvc != nil && rbacService != nil && jwtService != nil {
		middlewareManager = master.NewMiddlewareManager(authSvc, rbacService, jwtService)
	}

	// 返回完整的测试套件
	return &TestSuite{
		TestConfig:        testConfig,
		UserRepo:          userRepo,
		SessionRepo:       sessionRepo,
		JWTService:        jwtService,
		AuthService:       authSvc,
		RBACService:       rbacService,
		passwordManager:   auth.NewPasswordManager(nil), // 初始化密码管理器
		SessionService:    authSvc,                      // 会话服务使用认证服务
		UserService:       userService,                  // 用户服务
		MiddlewareManager: middlewareManager,
	}
}

// SetupTestDatabase 设置测试数据库
// 创建测试数据库并执行数据库迁移
func (tc *TestConfig) SetupTestDatabase(t *testing.T) {
	// 如果数据库连接为nil，跳过数据库设置
	if tc.DB == nil {
		t.Log("跳过数据库设置：数据库连接不可用")
		return
	}

	// 创建测试数据库（如果不存在）
	dbName := tc.Cfg.Database.MySQL.Database
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)

	if err := tc.DB.Exec(createDBSQL).Error; err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 执行数据库迁移
	if err := tc.migrateTestDatabase(); err != nil {
		t.Fatalf("测试数据库迁移失败: %v", err)
	}

	// 清理测试数据，确保每个测试开始时数据库是干净的
	tc.CleanupTestDatabase(t)

	t.Logf("✅ 测试数据库 %s 设置完成", dbName)
}

// migrateTestDatabase 执行测试数据库迁移
func (tc *TestConfig) migrateTestDatabase() error {
	// 自动迁移所有模型
	return tc.DB.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RolePermission{},
	)
}

// CleanupTestDatabase 清理测试数据库
// 删除所有测试数据，保持数据库结构
func (tc *TestConfig) CleanupTestDatabase(t *testing.T) {
	// 如果数据库连接不可用，跳过清理
	if tc.DB == nil {
		t.Log("⚠️ 跳过数据库清理：数据库连接不可用")
		return
	}

	// 清理所有表的数据（按依赖关系顺序）
	tables := []string{
		"role_permissions",
		"user_roles",
		"permissions",
		"roles",
		"users",
	}

	for _, table := range tables {
		if err := tc.DB.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			t.Logf("清理表 %s 失败: %v", table, err)
		}
	}

	// 重置自增ID
	for _, table := range tables {
		if err := tc.DB.Exec(fmt.Sprintf("ALTER TABLE %s AUTO_INCREMENT = 1", table)).Error; err != nil {
			t.Logf("重置表 %s 自增ID失败: %v", table, err)
		}
	}

	t.Log("✅ 测试数据库清理完成")
}

// CleanupTestRedis 清理测试Redis数据
func (tc *TestConfig) CleanupTestRedis(t *testing.T) {
	// 如果Redis连接不可用，跳过清理
	if tc.Redis == nil {
		t.Log("⚠️ 跳过Redis清理：Redis连接不可用")
		return
	}

	ctx := context.Background()

	// 清理所有测试相关的Redis键
	keys, err := tc.Redis.Keys(ctx, "test:*").Result()
	if err != nil {
		t.Logf("获取测试Redis键失败: %v", err)
		return
	}

	if len(keys) > 0 {
		if err := tc.Redis.Del(ctx, keys...).Err(); err != nil {
			t.Logf("清理测试Redis数据失败: %v", err)
			return
		}
	}

	t.Log("✅ 测试Redis数据清理完成")
}

// TeardownTestEnvironment 清理测试环境
// 关闭数据库连接并清理资源
func (ts *TestSuite) TeardownTestEnvironment(t *testing.T) {
	// 清理测试数据
	ts.CleanupTestDatabase(t)
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

	t.Log("✅ 测试环境清理完成")
}

// CreateTestUser 创建测试用户
// 返回创建的用户实例，用于测试
func (ts *TestSuite) CreateTestUser(t *testing.T, username, email, password string) *model.User {
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
	user := &model.User{
		Username:  username,
		Email:     email,
		Password:  hashedPassword, // 使用哈希后的密码
		Status:    model.UserStatusEnabled,
		PasswordV: 1,
	}

	// 保存到数据库（使用直接创建方法，不包含业务逻辑验证）
	err = ts.UserRepo.CreateUserDirect(ctx, user)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	t.Logf("✅ 创建测试用户: %s (ID: %d)", username, user.ID)
	return user
}

// CreateTestRole 创建测试角色，如果角色已存在则返回已存在的角色
func (ts *TestSuite) CreateTestRole(t *testing.T, name, description string) *model.Role {
	// 如果数据库连接不可用，返回nil
	if ts.UserRepo == nil {
		t.Skip("跳过创建测试角色：数据库连接不可用")
		return nil
	}

	ctx := context.Background()

	// 首先检查角色是否已存在
	var existingRole model.Role
	err := ts.DB.WithContext(ctx).Where("name = ?", name).First(&existingRole).Error
	if err == nil {
		// 角色已存在，返回已存在的角色
		t.Logf("🔄 使用已存在的测试角色: %s (ID: %d)", name, existingRole.ID)
		return &existingRole
	}

	// 角色不存在，创建新角色
	role := &model.Role{
		Name:        name,
		Description: description,
		Status:      model.RoleStatusEnabled,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存到数据库
	err = ts.DB.WithContext(ctx).Create(role).Error
	if err != nil {
		t.Fatalf("创建测试角色失败: %v", err)
	}

	t.Logf("✅ 创建测试角色: %s (ID: %d)", name, role.ID)
	return role
}

// AssignRoleToUser 为用户分配角色
func (ts *TestSuite) AssignRoleToUser(t *testing.T, userID, roleID uint) {
	// 如果数据库连接不可用，跳过此操作
	if ts.UserRepo == nil {
		t.Skip("跳过角色分配：数据库连接不可用")
		return
	}

	ctx := context.Background()

	// 调用UserRepository的角色分配方法
	err := ts.UserRepo.AssignRoleToUser(ctx, userID, roleID)
	if err != nil {
		t.Fatalf("为用户分配角色失败: %v", err)
	}

	t.Logf("✅ 为用户 %d 分配角色 %d", userID, roleID)
}

// RunWithTestEnvironment 在测试环境中运行测试函数
// 自动处理环境设置和清理
func RunWithTestEnvironment(t *testing.T, testFunc func(*TestSuite)) {
	// 设置测试环境
	ts := SetupTestEnvironment(t)
	ts.SetupTestDatabase(t)

	// 延迟清理
	defer ts.TeardownTestEnvironment(t)

	// 运行测试函数
	testFunc(ts)
}

// AssertNoError 断言没有错误
func AssertNoError(t *testing.T, err error, message string) {
	if err != nil {
		t.Fatalf("%s: %v", message, err)
	}
}

// AssertError 断言有错误
func AssertError(t *testing.T, err error, message string) {
	if err == nil {
		t.Fatalf("%s: 期望有错误但没有错误", message)
	}
}

// AssertEqual 断言两个值相等
func AssertEqual(t *testing.T, expected, actual interface{}, message string) {
	if expected != actual {
		t.Fatalf("%s: 期望 %v, 实际 %v", message, expected, actual)
	}
}

// AssertNotEqual 断言两个值不相等
func AssertNotEqual(t *testing.T, expected, actual interface{}, message string) {
	if expected == actual {
		t.Fatalf("%s: 期望值不应该等于 %v", message, expected)
	}
}

// AssertTrue 断言值为真
func AssertTrue(t *testing.T, value bool, message string) {
	if !value {
		t.Fatalf("%s: 期望为真但为假", message)
	}
}

// AssertFalse 断言值为假
func AssertFalse(t *testing.T, value bool, message string) {
	if value {
		t.Fatalf("%s: 期望为假但为真", message)
	}
}

// AssertNotNil 断言值不为空
func AssertNotNil(t *testing.T, value interface{}, message string) {
	if value == nil {
		t.Fatalf("%s: 期望不为空但为空", message)
	}
}

// AssertNil 断言值为空
func AssertNil(t *testing.T, value interface{}, message string) {
	// 检查接口值是否为 nil
	if value != nil {
		// 使用反射检查是否为 nil 指针
		if reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil() {
			return // 这是一个 nil 指针，测试通过
		}
		t.Fatalf("%s: 期望为空但不为空，实际值: %v (类型: %T)", message, value, value)
	}
}
