// Package test 用户模型单元测试
// 测试用户创建、验证、密码哈希等核心功能
package test

import (
	"context"
	"testing"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
)

// TestUserModel 测试用户模型的基本功能
func TestUserModel(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		t.Run("创建用户", func(t *testing.T) {
			testCreateUser(t, ts)
		})

		t.Run("用户密码验证", func(t *testing.T) {
			testUserPasswordValidation(t, ts)
		})

		t.Run("用户状态管理", func(t *testing.T) {
			testUserModelStatusManagement(t, ts)
		})

		t.Run("用户角色管理", func(t *testing.T) {
			testUserRoleManagement(t, ts)
		})

		t.Run("密码版本控制", func(t *testing.T) {
			testPasswordVersionControl(t, ts)
		})
	})
}

// testCreateUser 测试用户创建功能
func testCreateUser(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户创建测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 测试数据
	username := "testuser"
	email := "test@example.com"
	password := "password123"

	// 创建用户请求
	req := &model.CreateUserRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	// 创建密码管理器实例
	passwordManager := auth.NewPasswordManager(nil)
	
	// 为UserRepository注入密码管理器
	ts.UserRepo.SetPasswordManager(passwordManager)

	// 通过UserRepository的业务逻辑方法创建用户
	user, err := ts.UserRepo.CreateUser(ctx, req)
	AssertNoError(t, err, "创建用户应该成功")
	AssertNotEqual(t, uint(0), user.ID, "用户ID应该被设置")
	AssertEqual(t, username, user.Username, "用户名应该匹配")
	AssertEqual(t, email, user.Email, "邮箱应该匹配")
	AssertEqual(t, model.UserStatusEnabled, user.Status, "用户应该是启用状态")
	AssertEqual(t, int64(1), user.PasswordV, "密码版本应该是1")

	// 验证密码已被哈希
	AssertNotEqual(t, password, user.Password, "密码应该被哈希")

	// 验证密码哈希
	valid, err := passwordManager.VerifyPassword(password, user.Password)
	AssertNoError(t, err, "密码验证不应该出错")
	AssertTrue(t, valid, "密码验证应该成功")

	// 验证创建时间
	AssertTrue(t, !user.CreatedAt.IsZero(), "创建时间应该被设置")
	AssertTrue(t, !user.UpdatedAt.IsZero(), "更新时间应该被设置")
}

// testUserPasswordValidation 测试用户密码验证
func testUserPasswordValidation(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户密码验证测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 使用UserRepository创建用户（合并后的架构设计）
	passwordManager := auth.NewPasswordManager(nil)
	ts.UserRepo.SetPasswordManager(passwordManager)
	plainPassword := "testpass123"
	
	// 通过UserRepository创建用户请求
	req := &model.CreateUserRequest{
		Username: "passwordtest",
		Email:    "password@test.com",
		Password: plainPassword, // 传入明文密码，让UserRepository处理哈希
	}
	
	// UserRepository会在内部处理密码哈希
	user, err := ts.UserRepo.CreateUser(ctx, req)
	AssertNoError(t, err, "创建用户不应该出错")

	// 测试正确密码验证
	valid, err := passwordManager.VerifyPassword(plainPassword, user.Password)
	AssertNoError(t, err, "密码验证不应该出错")
	AssertTrue(t, valid, "正确密码应该验证成功")

	// 测试错误密码
	valid, err = passwordManager.VerifyPassword("wrongpassword", user.Password)
	AssertNoError(t, err, "密码验证不应该出错")
	AssertFalse(t, valid, "错误密码应该验证失败")

	// 测试空密码 - 应该返回错误
	valid, err = passwordManager.VerifyPassword("", user.Password)
	AssertError(t, err, "空密码应该返回错误")
	AssertFalse(t, valid, "空密码应该验证失败")

	// 更新密码
	newPassword := "newpassword456"
	hashedPassword, err := passwordManager.HashPassword(newPassword)
	AssertNoError(t, err, "密码哈希不应该出错")

	user.Password = hashedPassword
	user.PasswordV++ // 增加密码版本
	err = ts.UserRepo.UpdateUser(ctx, user)
	AssertNoError(t, err, "更新用户不应该出错")

	// 验证新密码
	valid, err = passwordManager.VerifyPassword(newPassword, user.Password)
	AssertNoError(t, err, "新密码验证不应该出错")
	AssertTrue(t, valid, "新密码应该验证成功")

	// 验证旧密码失效
	valid, err = passwordManager.VerifyPassword("testpass123", user.Password)
	AssertNoError(t, err, "旧密码验证不应该出错")
	AssertFalse(t, valid, "旧密码应该验证失败")
}

// testUserStatusManagement 测试用户状态管理
func testUserModelStatusManagement(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户状态管理测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 创建激活用户
	activeUser := ts.CreateTestUser(t, "activeuser", "active@test.com", "password123")
	AssertEqual(t, model.UserStatusEnabled, activeUser.Status, "用户应该是启用状态")
	AssertTrue(t, activeUser.IsActive(), "IsActive方法应该返回true")

	// 禁用用户
	activeUser.Status = model.UserStatusDisabled
	err := ts.UserRepo.UpdateUser(ctx, activeUser)
	AssertNoError(t, err, "更新用户状态不应该出错")
	AssertEqual(t, model.UserStatusDisabled, activeUser.Status, "用户应该是禁用状态")
	AssertFalse(t, activeUser.IsActive(), "IsActive方法应该返回false")

	// 重新激活用户
	activeUser.Status = model.UserStatusEnabled
	err = ts.UserRepo.UpdateUser(ctx, activeUser)
	AssertNoError(t, err, "重新激活用户不应该出错")
	AssertEqual(t, model.UserStatusEnabled, activeUser.Status, "用户应该重新激活")
	AssertTrue(t, activeUser.IsActive(), "IsActive方法应该返回true")
}

// testUserRoleManagement 测试用户角色管理
func testUserRoleManagement(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户角色管理测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "roleuser", "role@test.com", "password123")

	// 创建测试角色
	adminRole := ts.CreateTestRole(t, "admin", "管理员角色")
	userRole := ts.CreateTestRole(t, "user", "普通用户角色")

	// 为用户分配角色
	ts.AssignRoleToUser(t, user.ID, adminRole.ID)
	ts.AssignRoleToUser(t, user.ID, userRole.ID)

	// 获取用户及其角色
	userWithRoles, err := ts.UserRepo.GetUserWithRolesAndPermissions(ctx, user.ID)
	AssertNoError(t, err, "获取用户角色不应该出错")
	AssertEqual(t, 2, len(userWithRoles.Roles), "用户应该有2个角色")

	// 验证角色名称
	roleNames := make(map[string]bool)
	for _, role := range userWithRoles.Roles {
		roleNames[role.Name] = true
	}
	AssertTrue(t, roleNames["admin"], "用户应该有admin角色")
	AssertTrue(t, roleNames["user"], "用户应该有user角色")
}

// testPasswordVersionControl 测试密码版本控制
func testPasswordVersionControl(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过密码版本控制测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "versionuser", "version@test.com", "password123")
	initialVersion := user.PasswordV
	AssertEqual(t, int64(1), initialVersion, "初始密码版本应该是1")

	// 模拟密码修改
	newPassword := "newpassword456"
	passwordManager := auth.NewPasswordManager(nil)
	hashedPassword, err := passwordManager.HashPassword(newPassword)
	AssertNoError(t, err, "密码哈希不应该出错")

	// 更新密码和版本
	user.Password = hashedPassword
	user.PasswordV = initialVersion + 1
	err = ts.UserRepo.UpdateUser(ctx, user)
	AssertNoError(t, err, "更新用户密码不应该出错")

	// 验证密码版本递增
	AssertEqual(t, initialVersion+1, user.PasswordV, "密码版本应该递增")

	// 再次修改密码
	anotherPassword := "anotherpassword789"
	anotherHashedPassword, err := passwordManager.HashPassword(anotherPassword)
	AssertNoError(t, err, "密码哈希不应该出错")

	user.Password = anotherHashedPassword
	user.PasswordV++
	err = ts.UserRepo.UpdateUser(ctx, user)
	AssertNoError(t, err, "再次更新用户密码不应该出错")

	// 验证密码版本继续递增
	AssertEqual(t, initialVersion+2, user.PasswordV, "密码版本应该继续递增")

	// 验证最新密码有效
	valid, err := passwordManager.VerifyPassword(anotherPassword, user.Password)
	AssertNoError(t, err, "最新密码验证不应该出错")
	AssertTrue(t, valid, "最新密码应该验证成功")

	// 验证旧密码无效
	valid, err = passwordManager.VerifyPassword("password123", user.Password)
	AssertNoError(t, err, "旧密码验证不应该出错")
	AssertFalse(t, valid, "旧密码应该验证失败")

	valid, err = passwordManager.VerifyPassword(newPassword, user.Password)
	AssertNoError(t, err, "中间密码验证不应该出错")
	AssertFalse(t, valid, "中间密码应该验证失败")
}

// TestUserRepository 测试用户仓库的CRUD操作
func TestUserRepository(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		t.Run("创建用户", func(t *testing.T) {
			testCreateUserRepository(t, ts)
		})

		t.Run("获取用户", func(t *testing.T) {
			testGetUserRepository(t, ts)
		})

		t.Run("更新用户", func(t *testing.T) {
			testUpdateUserRepository(t, ts)
		})

		t.Run("删除用户", func(t *testing.T) {
			testDeleteUserRepository(t, ts)
		})

		t.Run("用户查询", func(t *testing.T) {
			testUserQueryRepository(t, ts)
		})
	})
}

// testCreateUserRepository 测试创建用户仓库操作
func testCreateUserRepository(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户仓库创建测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 测试创建用户
	user := &model.User{
		Username: "repotest",
		Email:    "repo@test.com",
		Password: "password123",
		Status:   model.UserStatusEnabled,
	}

	err := ts.UserRepo.CreateUserDirect(ctx, user)
	AssertNoError(t, err, "创建用户不应该出错")
	AssertNotEqual(t, uint(0), user.ID, "用户ID应该被设置")

	// 测试重复用户名
	duplicateUser := &model.User{
		Username: "repotest", // 相同用户名
		Email:    "another@test.com",
		Password: "password123",
		Status:   model.UserStatusEnabled,
	}

	err = ts.UserRepo.CreateUserDirect(ctx, duplicateUser)
	AssertError(t, err, "创建重复用户名应该出错")

	// 测试重复邮箱
	duplicateEmailUser := &model.User{
		Username: "anotheruser",
		Email:    "repo@test.com", // 相同邮箱
		Password: "password123",
		Status:   model.UserStatusEnabled,
	}

	err = ts.UserRepo.CreateUserDirect(ctx, duplicateEmailUser)
	AssertError(t, err, "创建重复邮箱应该出错")
}

// testGetUserRepository 测试获取用户仓库操作
func testGetUserRepository(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户仓库获取测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "gettest", "get@test.com", "password123")

	// 通过ID获取用户
	fetchedUser, err := ts.UserRepo.GetUserByID(ctx, user.ID)
	AssertNoError(t, err, "通过ID获取用户不应该出错")
	AssertEqual(t, user.ID, fetchedUser.ID, "用户ID应该匹配")
	AssertEqual(t, user.Username, fetchedUser.Username, "用户名应该匹配")

	// 通过用户名获取用户
	fetchedByUsername, err := ts.UserRepo.GetUserByUsername(ctx, user.Username)
	AssertNoError(t, err, "通过用户名获取用户不应该出错")
	AssertEqual(t, user.ID, fetchedByUsername.ID, "用户ID应该匹配")

	// 通过邮箱获取用户
	fetchedByEmail, err := ts.UserRepo.GetUserByEmail(ctx, user.Email)
	AssertNoError(t, err, "通过邮箱获取用户不应该出错")
	AssertEqual(t, user.ID, fetchedByEmail.ID, "用户ID应该匹配")

	// 获取不存在的用户
	_, err = ts.UserRepo.GetUserByID(ctx, 99999)
	AssertError(t, err, "获取不存在的用户应该出错")

	_, err = ts.UserRepo.GetUserByUsername(ctx, "nonexistent")
	AssertError(t, err, "获取不存在的用户名应该出错")

	_, err = ts.UserRepo.GetUserByEmail(ctx, "nonexistent@test.com")
	AssertError(t, err, "获取不存在的邮箱应该出错")
}

// testUpdateUserRepository 测试更新用户仓库操作
func testUpdateUserRepository(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户仓库更新测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "updatetest", "update@test.com", "password123")
	originalUpdatedAt := user.UpdatedAt

	// 等待一小段时间确保更新时间不同
	time.Sleep(10 * time.Millisecond)

	// 更新用户信息
	user.Email = "updated@test.com"
	user.Status = model.UserStatusDisabled

	err := ts.UserRepo.UpdateUser(ctx, user)
	AssertNoError(t, err, "更新用户不应该出错")
	AssertEqual(t, "updated@test.com", user.Email, "邮箱应该被更新")
	AssertEqual(t, model.UserStatusDisabled, user.Status, "用户状态应该被更新")
	AssertTrue(t, user.UpdatedAt.After(originalUpdatedAt), "更新时间应该改变")
}

// testDeleteUserRepository 测试删除用户仓库操作
func testDeleteUserRepository(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户仓库删除测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 创建测试用户
	user := ts.CreateTestUser(t, "deletetest", "delete@test.com", "password123")

	// 删除用户
	err := ts.UserRepo.DeleteUser(ctx, user.ID)
	AssertNoError(t, err, "删除用户不应该出错")

	// 验证用户已被删除
	_, err = ts.UserRepo.GetUserByID(ctx, user.ID)
	AssertError(t, err, "获取已删除用户应该出错")

	// 删除不存在的用户
	err = ts.UserRepo.DeleteUser(ctx, 99999)
	AssertError(t, err, "删除不存在用户应该出错")
}

// testUserQueryRepository 测试用户查询仓库操作
func testUserQueryRepository(t *testing.T, ts *TestSuite) {
	// 如果数据库连接不可用，跳过此测试
	if ts.UserRepo == nil {
		t.Skip("跳过用户仓库查询测试：数据库连接不可用")
		return
	}
	
	ctx := context.Background()

	// 创建多个测试用户
	ts.CreateTestUser(t, "query1", "query1@test.com", "password123")
	ts.CreateTestUser(t, "query2", "query2@test.com", "password123")
	ts.CreateTestUser(t, "query3", "query3@test.com", "password123")

	// 获取所有用户
	users, total, err := ts.UserRepo.ListUsers(ctx, 0, 10)
	AssertNoError(t, err, "获取用户列表不应该出错")
	AssertTrue(t, len(users) >= 3, "应该至少有3个用户")
	AssertTrue(t, total >= 3, "总数应该至少有3个用户")

	// 测试分页
	paginatedUsers, paginatedTotal, err := ts.UserRepo.ListUsers(ctx, 0, 2)
	AssertNoError(t, err, "分页获取用户不应该出错")
	AssertEqual(t, 2, len(paginatedUsers), "应该返回2个用户")
	AssertTrue(t, paginatedTotal >= 3, "总数应该至少有3个用户")

	// 测试偏移量
	offsetUsers, offsetTotal, err := ts.UserRepo.ListUsers(ctx, 1, 2)
	AssertNoError(t, err, "偏移量获取用户不应该出错")
	AssertEqual(t, 2, len(offsetUsers), "应该返回2个用户")
	AssertTrue(t, offsetTotal >= 3, "总数应该至少有3个用户")

	// 验证分页结果不同
	if len(paginatedUsers) > 0 && len(offsetUsers) > 0 {
		AssertNotEqual(t, paginatedUsers[0].ID, offsetUsers[0].ID, "分页结果应该不同")
	}
}