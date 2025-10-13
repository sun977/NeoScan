// DataInitialization测试文件
// 测试了测试数据的初始化功能，包括创建测试用户、创建测试角色和为用户分配角色等功能
// 测试命令：无独立测试命令，为基础测试框架文件

// Package test 测试数据初始化
// 提供测试环境所需的基础数据创建功能
package test

import (
	"context"
	"fmt"
	"neomaster/internal/model/system"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"neomaster/internal/pkg/auth"
)

// TestData 测试数据结构
type TestData struct {
	// 用户测试数据
	Users    []*system.User
	Roles    []*system.Role
	UserRoles []*system.UserRole

	// JWT测试数据
	TokenData []*system.TokenData
	Claims    []*auth.JWTClaims

	// 会话测试数据
	Sessions []*system.SessionData
}

// NewTestData 创建新的测试数据实例
func NewTestData() *TestData {
	return &TestData{
		Users:      make([]*system.User, 0),
		Roles:      make([]*system.Role, 0),
		UserRoles:  make([]*system.UserRole, 0),
		TokenData: make([]*system.TokenData, 0),
		Claims:    make([]*auth.JWTClaims, 0),
		Sessions:  make([]*system.SessionData, 0),
	}
}

// CreateTestUser 创建测试用户数据
func (td *TestData) CreateTestUser(username, email, password string) *system.User {
	passwordManager := auth.NewPasswordManager(nil)
	hashedPassword, _ := passwordManager.HashPassword(password)
	user := &system.User{
		ID:        uint(len(td.Users) + 1),
		Username:  username,
		Email:     email,
		Password:  hashedPassword,
		Status:    system.UserStatusEnabled,
		PasswordV: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	td.Users = append(td.Users, user)
	return user
}

// CreateTestRole 创建测试角色数据
func (td *TestData) CreateTestRole(name, description string) *system.Role {
	role := &system.Role{
		ID:          uint(len(td.Roles) + 1),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	td.Roles = append(td.Roles, role)
	return role
}

// AssignRoleToUser 为用户分配角色
func (td *TestData) AssignRoleToUser(userID, roleID uint) *system.UserRole {
	userRole := &system.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		CreatedAt: time.Now(),
	}
	td.UserRoles = append(td.UserRoles, userRole)
	return userRole
}

// CreateTestTokenData 创建测试令牌数据
func (td *TestData) CreateTestTokenData(accessToken, refreshToken string, expiresAt time.Time) *system.TokenData {
	tokenData := &system.TokenData{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}
	td.TokenData = append(td.TokenData, tokenData)
	return tokenData
}

// CreateTestClaims 创建测试JWT声明
func (td *TestData) CreateTestClaims(userID uint, username, email string, roles []string) *auth.JWTClaims {
	claims := &auth.JWTClaims{
		UserID:    userID,
		Username:  username,
		Email:     email,
		Roles:     roles,
		PasswordV: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "neoscan",
			Subject:   username,
			Audience:  []string{"neoscan-api"},
		},
	}
	td.Claims = append(td.Claims, claims)
	return claims
}

// CreateTestSession 创建测试会话
func (td *TestData) CreateTestSession(userID uint, username, email string) *system.SessionData {
	session := &system.SessionData{
		UserID:     userID,
		Username:   username,
		Email:      email,
		Roles:      []string{"user"},
		LoginTime:  time.Now(),
		LastActive: time.Now(),
		ClientIP:   "127.0.0.1",
		UserAgent:  "test-agent",
	}
	td.Sessions = append(td.Sessions, session)
	return session
}

// GetUserByID 根据ID获取用户
func (td *TestData) GetUserByID(id uint) *system.User {
	for _, user := range td.Users {
		if user.ID == id {
			return user
		}
	}
	return nil
}

// GetUserByUsername 根据用户名获取用户
func (td *TestData) GetUserByUsername(username string) *system.User {
	for _, user := range td.Users {
		if user.Username == username {
			return user
		}
	}
	return nil
}

// GetUserByEmail 根据邮箱获取用户
func (td *TestData) GetUserByEmail(email string) *system.User {
	for _, user := range td.Users {
		if user.Email == email {
			return user
		}
	}
	return nil
}

// GetRoleByName 根据名称获取角色
func (td *TestData) GetRoleByName(name string) *system.Role {
	for _, role := range td.Roles {
		if role.Name == name {
			return role
		}
	}
	return nil
}

// GetUserRoles 获取用户的所有角色
func (td *TestData) GetUserRoles(userID uint) []*system.Role {
	var roles []*system.Role
	for _, userRole := range td.UserRoles {
		if userRole.UserID == userID {
			for _, role := range td.Roles {
				if role.ID == userRole.RoleID {
					roles = append(roles, role)
					break
				}
			}
		}
	}
	return roles
}

// MockUserRepository Mock用户仓库
type MockUserRepository struct {
	data *TestData
}

// NewMockUserRepository 创建Mock用户仓库
func NewMockUserRepository(data *TestData) *MockUserRepository {
	return &MockUserRepository{data: data}
}

// CreateUser 创建用户
func (m *MockUserRepository) CreateUser(ctx context.Context, user *system.User) error {
	if m.data.GetUserByUsername(user.Username) != nil {
		return fmt.Errorf("用户名已存在")
	}
	if m.data.GetUserByEmail(user.Email) != nil {
		return fmt.Errorf("邮箱已存在")
	}

	user.ID = uint(len(m.data.Users) + 1)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.data.Users = append(m.data.Users, user)
	return nil
}

// GetUserByID 根据ID获取用户
func (m *MockUserRepository) GetUserByID(ctx context.Context, id uint) (*system.User, error) {
	user := m.data.GetUserByID(id)
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*system.User, error) {
	user := m.data.GetUserByUsername(username)
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*system.User, error) {
	user := m.data.GetUserByEmail(email)
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return user, nil
}

// UpdateUser 更新用户
func (m *MockUserRepository) UpdateUser(ctx context.Context, user *system.User) (*system.User, error) {
	for i, u := range m.data.Users {
		if u.ID == user.ID {
			user.UpdatedAt = time.Now()
			m.data.Users[i] = user
			return user, nil
		}
	}
	return nil, fmt.Errorf("用户不存在")
}

// DeleteUser 删除用户
func (m *MockUserRepository) DeleteUser(ctx context.Context, id uint) error {
	for i, user := range m.data.Users {
		if user.ID == id {
			m.data.Users = append(m.data.Users[:i], m.data.Users[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("用户不存在")
}

// GetUserWithRolesAndPermissions 获取用户及其角色权限信息
func (m *MockUserRepository) GetUserWithRolesAndPermissions(ctx context.Context, userID uint) (*system.User, error) {
	user := m.data.GetUserByID(userID)
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 获取用户角色
	roles := m.data.GetUserRoles(userID)
	user.Roles = roles

	return user, nil
}

// ListUsers 列出用户
func (m *MockUserRepository) ListUsers(ctx context.Context, offset, limit int) ([]*system.User, int64, error) {
	total := int64(len(m.data.Users))
	if offset >= len(m.data.Users) {
		return []*system.User{}, total, nil
	}

	end := offset + limit
	if end > len(m.data.Users) {
		end = len(m.data.Users)
	}

	return m.data.Users[offset:end], total, nil
}

// MockRoleRepository Mock角色仓库
type MockRoleRepository struct {
	data *TestData
}

// NewMockRoleRepository 创建Mock角色仓库
func NewMockRoleRepository(data *TestData) *MockRoleRepository {
	return &MockRoleRepository{data: data}
}

// CreateRole 创建角色
func (m *MockRoleRepository) CreateRole(ctx context.Context, role *system.Role) (*system.Role, error) {
	if m.data.GetRoleByName(role.Name) != nil {
		return nil, fmt.Errorf("角色名已存在")
	}

	role.ID = uint(len(m.data.Roles) + 1)
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	m.data.Roles = append(m.data.Roles, role)
	return role, nil
}

// GetRoleByID 根据ID获取角色
func (m *MockRoleRepository) GetRoleByID(ctx context.Context, id uint) (*system.Role, error) {
	for _, role := range m.data.Roles {
		if role.ID == id {
			return role, nil
		}
	}
	return nil, fmt.Errorf("角色不存在")
}

// GetRoleByName 根据名称获取角色
func (m *MockRoleRepository) GetRoleByName(ctx context.Context, name string) (*system.Role, error) {
	role := m.data.GetRoleByName(name)
	if role == nil {
		return nil, fmt.Errorf("角色不存在")
	}
	return role, nil
}

// AssignRoleToUser 为用户分配角色
func (m *MockRoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	// 检查用户和角色是否存在
	if m.data.GetUserByID(userID) == nil {
		return fmt.Errorf("用户不存在")
	}

	found := false
	for _, role := range m.data.Roles {
		if role.ID == roleID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("角色不存在")
	}

	// 检查是否已经分配
	for _, userRole := range m.data.UserRoles {
		if userRole.UserID == userID && userRole.RoleID == roleID {
			return fmt.Errorf("角色已分配")
		}
	}

	// 分配角色
	m.data.AssignRoleToUser(userID, roleID)
	return nil
}

// RemoveRoleFromUser 移除用户角色
func (m *MockRoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	for i, userRole := range m.data.UserRoles {
		if userRole.UserID == userID && userRole.RoleID == roleID {
			m.data.UserRoles = append(m.data.UserRoles[:i], m.data.UserRoles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("用户角色关系不存在")
}

// MockSessionRepository Mock会话仓库
type MockSessionRepository struct {
	data *TestData
}

// NewMockSessionRepository 创建Mock会话仓库
func NewMockSessionRepository(data *TestData) *MockSessionRepository {
	return &MockSessionRepository{data: data}
}

// CreateSession 创建会话
func (m *MockSessionRepository) CreateSession(ctx context.Context, session *system.SessionData) (*system.SessionData, error) {
	session.LoginTime = time.Now()
	session.LastActive = time.Now()
	m.data.Sessions = append(m.data.Sessions, session)
	return session, nil
}

// GetSessionByUserID 根据用户ID获取会话
func (m *MockSessionRepository) GetSessionByUserID(ctx context.Context, userID uint) (*system.SessionData, error) {
	for _, session := range m.data.Sessions {
		if session.UserID == userID {
			return session, nil
		}
	}
	return nil, fmt.Errorf("会话不存在")
}

// UpdateSession 更新会话
func (m *MockSessionRepository) UpdateSession(ctx context.Context, session *system.SessionData) (*system.SessionData, error) {
	for i, s := range m.data.Sessions {
		if s.UserID == session.UserID {
			session.LastActive = time.Now()
			m.data.Sessions[i] = session
			return session, nil
		}
	}
	return nil, fmt.Errorf("会话不存在")
}

// DeleteSession 删除会话
func (m *MockSessionRepository) DeleteSession(ctx context.Context, userID uint) error {
	for i, session := range m.data.Sessions {
		if session.UserID == userID {
			m.data.Sessions = append(m.data.Sessions[:i], m.data.Sessions[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("会话不存在")
}

// DeleteSessionByUserID 根据用户ID删除会话
func (m *MockSessionRepository) DeleteSessionByUserID(ctx context.Context, userID uint) error {
	for i, session := range m.data.Sessions {
		if session.UserID == userID {
			m.data.Sessions = append(m.data.Sessions[:i], m.data.Sessions[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("会话不存在")
}

// ValidateSession 验证会话
func (m *MockSessionRepository) ValidateSession(ctx context.Context, userID uint) (bool, error) {
	for _, session := range m.data.Sessions {
		if session.UserID == userID {
			// 检查会话是否活跃
			if !session.IsActive(time.Hour * 24) {
				return false, fmt.Errorf("会话已过期")
			}
			return true, nil
		}
	}
	return false, fmt.Errorf("会话不存在")
}

// TestDataBuilder 测试数据构建器
type TestDataBuilder struct {
	data *TestData
}

// NewTestDataBuilder 创建测试数据构建器
func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{
		data: NewTestData(),
	}
}

// WithUsers 添加用户数据
func (b *TestDataBuilder) WithUsers(users ...*system.User) *TestDataBuilder {
	b.data.Users = append(b.data.Users, users...)
	return b
}

// WithRoles 添加角色数据
func (b *TestDataBuilder) WithRoles(roles ...*system.Role) *TestDataBuilder {
	b.data.Roles = append(b.data.Roles, roles...)
	return b
}

// WithUserRoles 添加用户角色关系数据
func (b *TestDataBuilder) WithUserRoles(userRoles ...*system.UserRole) *TestDataBuilder {
	b.data.UserRoles = append(b.data.UserRoles, userRoles...)
	return b
}

// WithSessions 添加会话数据
func (b *TestDataBuilder) WithSessions(sessions ...*system.SessionData) *TestDataBuilder {
	b.data.Sessions = append(b.data.Sessions, sessions...)
	return b
}

// WithDefaultUsers 添加默认用户数据
func (b *TestDataBuilder) WithDefaultUsers() *TestDataBuilder {
	// 创建管理员用户
	adminUser := b.data.CreateTestUser("admin", "admin@test.com", "admin123")
	adminRole := b.data.CreateTestRole("admin", "系统管理员")
	b.data.AssignRoleToUser(adminUser.ID, adminRole.ID)

	// 创建普通用户
	normalUser := b.data.CreateTestUser("user", "user@test.com", "user123")
	userRole := b.data.CreateTestRole("user", "普通用户")
	b.data.AssignRoleToUser(normalUser.ID, userRole.ID)

	// 创建版主用户
	moderatorUser := b.data.CreateTestUser("moderator", "moderator@test.com", "moderator123")
	moderatorRole := b.data.CreateTestRole("moderator", "版主")
	b.data.AssignRoleToUser(moderatorUser.ID, moderatorRole.ID)

	return b
}

// WithDefaultRoles 添加默认角色数据
func (b *TestDataBuilder) WithDefaultRoles() *TestDataBuilder {
	b.data.CreateTestRole("admin", "系统管理员")
	b.data.CreateTestRole("user", "普通用户")
	b.data.CreateTestRole("moderator", "版主")
	b.data.CreateTestRole("guest", "访客")
	return b
}

// Build 构建测试数据
func (b *TestDataBuilder) Build() *TestData {
	return b.data
}

// TestScenarios 测试场景
type TestScenarios struct{}

// NewTestScenarios 创建测试场景
func NewTestScenarios() *TestScenarios {
	return &TestScenarios{}
}

// UserRegistrationScenario 用户注册场景
func (ts *TestScenarios) UserRegistrationScenario() *TestData {
	return NewTestDataBuilder().Build()
}

// UserLoginScenario 用户登录场景
func (ts *TestScenarios) UserLoginScenario() *TestData {
	return NewTestDataBuilder().
		WithDefaultUsers().
		Build()
}

// PermissionTestScenario 权限测试场景
func (ts *TestScenarios) PermissionTestScenario() *TestData {
	return NewTestDataBuilder().
		WithDefaultUsers().
		WithDefaultRoles().
		Build()
}

// SessionManagementScenario 会话管理场景
func (ts *TestScenarios) SessionManagementScenario() *TestData {
	data := NewTestDataBuilder().
		WithDefaultUsers().
		Build()

	// 为用户创建会话
	for _, user := range data.Users {
		data.CreateTestSession(user.ID, user.Username, user.Email)
	}

	return data
}

// ComplexScenario 复杂测试场景
func (ts *TestScenarios) ComplexScenario() *TestData {
	data := NewTestDataBuilder().
		WithDefaultUsers().
		WithDefaultRoles().
		Build()

	// 创建更多用户和角色关系
	for i := 1; i <= 10; i++ {
		user := data.CreateTestUser(
			fmt.Sprintf("testuser%d", i),
			fmt.Sprintf("testuser%d@test.com", i),
			"password123",
		)

		// 随机分配角色
		if i%3 == 0 {
			// 管理员
			adminRole := data.GetRoleByName("admin")
			if adminRole != nil {
				data.AssignRoleToUser(user.ID, adminRole.ID)
			}
		} else if i%2 == 0 {
			// 版主
			moderatorRole := data.GetRoleByName("moderator")
			if moderatorRole != nil {
				data.AssignRoleToUser(user.ID, moderatorRole.ID)
			}
		} else {
			// 普通用户
			userRole := data.GetRoleByName("user")
			if userRole != nil {
				data.AssignRoleToUser(user.ID, userRole.ID)
			}
		}

		// 创建会话
		data.CreateTestSession(
			user.ID,
			user.Username,
			user.Email,
		)
	}

	return data
}

// TestConstants 测试常量
var TestConstants = struct {
	// 默认密码
	DefaultPassword string
	// 默认令牌过期时间
	DefaultTokenExpiry time.Duration
	// 默认刷新令牌过期时间
	DefaultRefreshTokenExpiry time.Duration
	// 测试JWT密钥
	TestJWTSecret string
}{
	DefaultPassword:           "password123",
	DefaultTokenExpiry:        time.Hour,
	DefaultRefreshTokenExpiry: 24 * time.Hour,
	TestJWTSecret:             "test-jwt-secret-key-for-testing-only",
}

// TestHelpers 测试辅助函数
type TestHelpers struct{}

// NewTestHelpers 创建测试辅助函数实例
func NewTestHelpers() *TestHelpers {
	return &TestHelpers{}
}

// GenerateTestEmail 生成测试邮箱
func (h *TestHelpers) GenerateTestEmail(prefix string) string {
	return fmt.Sprintf("%s_%d@test.com", prefix, time.Now().UnixNano())
}

// GenerateTestUsername 生成测试用户名
func (h *TestHelpers) GenerateTestUsername(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// GenerateTestToken 生成测试令牌
func (h *TestHelpers) GenerateTestToken(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), time.Now().Unix())
}

// IsValidEmail 验证邮箱格式
func (h *TestHelpers) IsValidEmail(email string) bool {
	// 简单的邮箱格式验证
	return len(email) > 0 && 
		len(email) <= 254 && 
		contains(email, "@") && 
		contains(email, ".")
}

// IsValidUsername 验证用户名格式
func (h *TestHelpers) IsValidUsername(username string) bool {
	// 简单的用户名格式验证
	return len(username) >= 3 && len(username) <= 50
}

// IsValidPassword 验证密码强度
func (h *TestHelpers) IsValidPassword(password string) bool {
	// 简单的密码强度验证
	return len(password) >= 6 && len(password) <= 128
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}