/**
 * 测试套件辅助工具
 * @author: Sun977
 * @date: 2025.10.14
 * @description: 测试环境初始化和清理工具，遵循"好品味"原则 - 简洁、可复用、可靠
 * @func: 提供数据库连接、路由初始化、测试数据创建等功能
 */
package test

import (
	"context"
	"fmt"
	agentRepo "neomaster/internal/repo/mysql/agent"
	system2 "neomaster/internal/repo/mysql/system"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"neomaster/internal/app/master/router"
	"neomaster/internal/config"
	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/utils"
	agentService "neomaster/internal/service/agent"
)

// TestSuite 测试套件结构体
type TestSuite struct {
	DB              *gorm.DB
	RouterManager   *router.Router
	AgentRepository agentRepo.AgentRepository
	UserRepository  *system2.UserRepository
	RoleRepository  *system2.RoleRepository
	AgentService    agentService.AgentManagerService
	SessionService  *MockSessionService
	t               *testing.T
}

// NewTestSuite 创建新的测试套件
func NewTestSuite(t *testing.T) *TestSuite {
	// 加载测试配置
	cfg, err := config.LoadConfig("../../configs", "test")
	require.NoError(t, err, "加载测试配置失败")

	// 使用neoscan_test数据库连接
	db, err := database.NewMySQLConnection(&cfg.Database.MySQL)
	require.NoError(t, err, "连接neoscan_test数据库失败")

	// 自动迁移数据库表 - 只迁移测试需要的基础表
	err = db.AutoMigrate(
		&agentModel.Agent{},
		&system.User{},
		&system.Role{},
		&system.Permission{},
		&system.UserRole{},
		&system.RolePermission{},
		// 暂时移除orchestrator相关表，避免依赖问题
		// &orchestrator.Project{},
		// &orchestrator.ProjectConfig{},
		// &orchestrator.WorkflowConfig{},
		// &orchestrator.WorkflowExecution{},
		// &orchestrator.WorkflowStep{},
		// &orchestrator.WorkflowStepExecution{},
	)
	require.NoError(t, err, "数据库表迁移失败")

	// 初始化Repository
	agentRepository := agentRepo.NewAgentRepository(db)
	userRepository := system2.NewUserRepository(db)
	roleRepository := system2.NewRoleRepository(db)

	// 初始化Service - 简化版本，只创建必要的服务用于测试
	agentSvc := agentService.NewAgentService(agentRepository)

	// 为测试创建简化的服务，避免复杂的依赖关系
	// sessionSvc := authService.NewSessionService(userRepository, roleRepository)
	sessionSvc := NewMockSessionService(userRepository, roleRepository)

	// 初始化路由管理器 - 需要数据库、Redis和JWT密钥
	// 为测试创建一个简单的Redis客户端（可以是mock）
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // 测试用Redis地址
		DB:   0,                // 使用默认数据库
	})
	jwtSecret := "test-jwt-secret-key-for-testing-only"
	routerManager := router.NewRouter(db, redisClient, jwtSecret)
	
	// 设置路由 - 这是关键步骤，必须调用才能注册所有路由
	routerManager.SetupRoutes()

	// 创建基础角色和权限
	ts := &TestSuite{
		DB:              db,
		RouterManager:   routerManager,
		AgentRepository: agentRepository,
		UserRepository:  userRepository,
		RoleRepository:  roleRepository,
		AgentService:    agentSvc,
		SessionService:  sessionSvc,
		t:               t,
	}

	// 初始化基础数据
	ts.initBaseData()

	return ts
}

// initBaseData 初始化基础数据（角色、权限等）
func (ts *TestSuite) initBaseData() {
	ctx := context.Background()

	// 先清理现有数据，避免重复插入
	ts.cleanupData()

	// 创建基础角色数据
	adminRole := &system.Role{
		Name:        "admin",
		DisplayName: "管理员",
		Description: "系统管理员，拥有所有权限",
		Status:      system.RoleStatusEnabled,
	}

	operatorRole := &system.Role{
		Name:        "operator",
		DisplayName: "操作员",
		Description: "系统操作员，拥有操作权限",
		Status:      system.RoleStatusEnabled,
	}

	viewerRole := &system.Role{
		Name:        "viewer",
		DisplayName: "查看者",
		Description: "只读用户，仅有查看权限",
		Status:      system.RoleStatusEnabled,
	}

	guestRole := &system.Role{
		Name:        "guest",
		DisplayName: "访客",
		Description: "访客用户，最低权限",
		Status:      system.RoleStatusEnabled,
	}

	err := ts.RoleRepository.CreateRole(ctx, adminRole)
	require.NoError(ts.t, err)

	err = ts.RoleRepository.CreateRole(ctx, operatorRole)
	require.NoError(ts.t, err)

	err = ts.RoleRepository.CreateRole(ctx, viewerRole)
	require.NoError(ts.t, err)

	err = ts.RoleRepository.CreateRole(ctx, guestRole)
	require.NoError(ts.t, err)

	// 创建普通用户角色
	userRole := &system.Role{
		Name:        "user",
		DisplayName: "普通用户",
		Description: "普通用户角色",
		Status:      system.RoleStatusEnabled,
	}
	err = ts.RoleRepository.CreateRole(ctx, userRole)
	require.NoError(ts.t, err)

	// 创建基础权限
	permissions := []*system.Permission{
		{Name: "agent:read", DisplayName: "查看Agent", Description: "查看Agent信息"},
		{Name: "agent:write", DisplayName: "管理Agent", Description: "创建、更新、删除Agent"},
		{Name: "user:read", DisplayName: "查看用户", Description: "查看用户信息"},
		{Name: "user:write", DisplayName: "管理用户", Description: "创建、更新、删除用户"},
	}

	for _, perm := range permissions {
		err := ts.DB.Create(perm).Error
		require.NoError(ts.t, err)

		// 给管理员角色分配所有权限
		rolePermission := &system.RolePermission{
			RoleID:       adminRole.ID,
			PermissionID: perm.ID,
		}
		err = ts.DB.Create(rolePermission).Error
		require.NoError(ts.t, err)
	}
}

// CreateTestUser 创建测试用户
func (ts *TestSuite) CreateTestUser(t *testing.T, username, email, password string) *system.User {
	// 生成密码哈希
	hashedPassword, err := auth.HashPasswordWithDefaultConfig(password)
	require.NoError(t, err)

	user := &system.User{
		Username:    username,
		Email:       email,
		Password:    hashedPassword,
		Nickname:    username + "_nickname",
		Status:      system.UserStatusEnabled,
		LastLoginIP: "127.0.0.1",
	}

	err = ts.UserRepository.CreateUser(context.Background(), user)
	require.NoError(t, err)

	return user
}

// AssignRoleToUser 为用户分配角色
func (ts *TestSuite) AssignRoleToUser(t *testing.T, userID uint, roleName string) {
	// 查找角色
	role, err := ts.RoleRepository.GetRoleByName(context.Background(), roleName)
	require.NoError(t, err)

	// 创建用户角色关联
	userRole := &system.UserRole{
		UserID: userID,
		RoleID: role.ID,
	}

	err = ts.DB.Create(userRole).Error
	require.NoError(t, err)
}

// CreateTestAgent 创建测试Agent
func (ts *TestSuite) CreateTestAgent(t *testing.T, hostname, ipAddress string, port int) *agentModel.Agent {
	// 生成唯一的AgentID
	agentID, err := utils.GenerateUUID()
	require.NoError(t, err)

	// 设置Token过期时间为24小时后
	tokenExpiry := time.Now().Add(24 * time.Hour)

	agent := &agentModel.Agent{
		AgentID:       agentID,
		Hostname:      hostname,
		IPAddress:     ipAddress,
		Port:          port,
		Version:       "1.0.0",
		OS:            "Linux",
		Arch:          "x86_64",
		CPUCores:      4,
		MemoryTotal:   8589934592,   // 8GB
		DiskTotal:     107374182400, // 100GB
		Status:        agentModel.AgentStatusOnline,
		Capabilities:  agentModel.StringSlice{"port_scan", "vuln_scan"},
		Tags:          agentModel.StringSlice{"test", "development"},
		TokenExpiry:   tokenExpiry,  // 设置Token过期时间
		Remark:        "测试Agent",
		LastHeartbeat: time.Now(),
	}

	err = ts.AgentRepository.Create(agent)
	require.NoError(t, err)

	return agent
}

// CreateTestAgentWithCustomData 创建自定义数据的测试Agent
func (ts *TestSuite) CreateTestAgentWithCustomData(t *testing.T, data map[string]interface{}) *agentModel.Agent {
	// 设置默认值
	defaults := map[string]interface{}{
		"hostname":     "test-agent",
		"ip_address":   "192.168.1.100",
		"port":         8080,
		"version":      "1.0.0",
		"os":           "Linux",
		"arch":         "x86_64",
		"cpu_cores":    4,
		"memory_total": 8589934592,
		"disk_total":   107374182400,
		"status":       agentModel.AgentStatusOnline,
		"capabilities": agentModel.StringSlice{"port_scan"},
		"tags":         agentModel.StringSlice{"test"},
		"remark":       "测试Agent",
	}

	// 合并自定义数据
	for key, value := range data {
		defaults[key] = value
	}

	// 生成Agent ID
	agentID, err := utils.GenerateUUID()
	if err != nil {
		require.NoError(t, err)
	}

	// 设置Token过期时间为24小时后
	tokenExpiry := time.Now().Add(24 * time.Hour)

	agent := &agentModel.Agent{
		AgentID:       agentID,
		Hostname:      defaults["hostname"].(string),
		IPAddress:     defaults["ip_address"].(string),
		Port:          defaults["port"].(int),
		Version:       defaults["version"].(string),
		OS:            defaults["os"].(string),
		Arch:          defaults["arch"].(string),
		CPUCores:      defaults["cpu_cores"].(int),
		MemoryTotal:   defaults["memory_total"].(int64),
		DiskTotal:     defaults["disk_total"].(int64),
		Status:        defaults["status"].(agentModel.AgentStatus),
		Capabilities:  defaults["capabilities"].(agentModel.StringSlice),
		Tags:          defaults["tags"].(agentModel.StringSlice),
		TokenExpiry:   tokenExpiry,  // 设置Token过期时间
		Remark:        defaults["remark"].(string),
		LastHeartbeat: time.Now(),
	}

	err = ts.AgentRepository.Create(agent)
	require.NoError(t, err)

	return agent
}

// UpdateAgentStatus 更新Agent状态
func (ts *TestSuite) UpdateAgentStatus(t *testing.T, agentID string, status agentModel.AgentStatus) {
	err := ts.AgentRepository.UpdateStatus(agentID, status)
	require.NoError(t, err)
}

// UpdateAgentHeartbeat 更新Agent心跳时间
func (ts *TestSuite) UpdateAgentHeartbeat(t *testing.T, agentID string) {
	err := ts.AgentRepository.UpdateLastHeartbeat(agentID)
	require.NoError(t, err)
}

// GetAgentCount 获取Agent总数
func (ts *TestSuite) GetAgentCount() int64 {
	var count int64
	ts.DB.Model(&agentModel.Agent{}).Count(&count)
	return count
}

// GetUserCount 获取用户总数
func (ts *TestSuite) GetUserCount() int64 {
	var count int64
	ts.DB.Model(&system.User{}).Count(&count)
	return count
}

// ExecuteSQL 执行原生SQL（用于复杂测试场景）
func (ts *TestSuite) ExecuteSQL(sql string, args ...interface{}) error {
	return ts.DB.Exec(sql, args...).Error
}

// Cleanup 清理测试环境
func (ts *TestSuite) Cleanup() {
	if ts.DB != nil {
		ts.cleanupData()

		// 关闭数据库连接
		sqlDB, err := ts.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

// cleanupData 清理数据库中的测试数据
func (ts *TestSuite) cleanupData() {
	// 清理所有表数据 - 按照外键依赖顺序删除
	ts.DB.Exec("DELETE FROM user_roles")
	ts.DB.Exec("DELETE FROM role_permissions")
	ts.DB.Exec("DELETE FROM users")
	ts.DB.Exec("DELETE FROM roles")
	ts.DB.Exec("DELETE FROM permissions")
	ts.DB.Exec("DELETE FROM agents")
}

// AssertAgentExists 断言Agent存在
func (ts *TestSuite) AssertAgentExists(t *testing.T, agentID string) *agentModel.Agent {
	agent, err := ts.AgentRepository.GetByID(agentID)
	require.NoError(t, err)
	require.NotNil(t, agent)
	return agent
}

// AssertAgentNotExists 断言Agent不存在
func (ts *TestSuite) AssertAgentNotExists(t *testing.T, agentID string) {
	_, err := ts.AgentRepository.GetByID(agentID)
	require.Error(t, err)
}

// AssertUserExists 断言用户存在
func (ts *TestSuite) AssertUserExists(t *testing.T, userID uint) *system.User {
	user, err := ts.UserRepository.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, user)
	return user
}

// WaitForCondition 等待条件满足（用于异步操作测试）
func (ts *TestSuite) WaitForCondition(condition func() bool, timeout time.Duration, message string) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("condition not met within timeout: %s", message)
}

// CreateContext 创建测试上下文
func (ts *TestSuite) CreateContext() context.Context {
	return context.Background()
}
