/*
 * @author: sun977
 * @date: 2025.09.04
 * @description: 用户服务业务逻辑
 * @func:
 * 1.创建
 * 2.更新
 * 3.删除
 * 4.状态变更等
 */
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/repository/mysql"
	"neomaster/internal/repository/redis"

	"gorm.io/gorm"
)

// UserService 用户服务
// 负责用户相关的业务逻辑，包括用户注册、获取用户信息等
type UserService struct {
	userRepo        *mysql.UserRepository    // 用户数据仓库
	redisRepo       *redis.SessionRepository // Redis缓存仓库
	passwordManager *auth.PasswordManager    // 密码管理器
	jwtManager      *auth.JWTManager         // JWT管理器
}

// NewUserService 创建新的用户服务实例
func NewUserService(
	userRepo *mysql.UserRepository,
	redisRepo *redis.SessionRepository,
	passwordManager *auth.PasswordManager,
	jwtManager *auth.JWTManager,
) *UserService {
	return &UserService{
		userRepo:        userRepo,
		redisRepo:       redisRepo,
		passwordManager: passwordManager,
		jwtManager:      jwtManager,
	}
}

// Register 用户注册
// 处理用户注册请求，包括参数验证、用户名/邮箱唯一性检查、密码哈希等
func (s *UserService) Register(ctx context.Context, req *model.RegisterRequest) (*model.RegisterResponse, error) {
	// 参数验证
	if req == nil {
		logger.LogError(errors.New("register request is nil"), "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation": "register",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("注册请求不能为空")
	}

	if req.Username == "" {
		logger.LogError(errors.New("username is empty"), "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation": "register",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户名不能为空")
	}

	if req.Email == "" {
		logger.LogError(errors.New("email is empty"), "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation": "register",
			"username":  req.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("邮箱不能为空")
	}

	if req.Password == "" {
		logger.LogError(errors.New("password is empty"), "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation": "register",
			"username":  req.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("密码不能为空")
	}

	// 检查用户名和邮箱是否已存在
	exists, err := s.userRepo.UserExists(ctx, req.Username, req.Email)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation": "register",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("检查用户是否存在失败: %w", err)
	}

	if exists {
		logger.LogError(model.ErrUserAlreadyExists, "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation": "register",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, model.ErrUserAlreadyExists
	}

	// 哈希密码
	hashedPassword, err := s.passwordManager.HashPassword(req.Password)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation": "hash_password",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 创建用户对象
	user := &model.User{
		Username:  req.Username,
		Email:     req.Email,
		Nickname:  req.Nickname,
		Password:  hashedPassword, // 使用哈希后的密码
		Status:    model.UserStatusEnabled,
		PasswordV: 1, // 设置密码版本
	}

	// 创建用户
	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation": "register",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 记录成功注册的业务日志
	logger.LogBusinessOperation("user_register", user.ID, user.Username, "", "", "success", "用户注册成功", map[string]interface{}{
		"user_id":   user.ID,
		"username":  user.Username,
		"email":     user.Email,
		"nickname":  user.Nickname,
		"timestamp": logger.NowFormatted(),
	})

	// 构造用户信息
	userInfo := &model.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Phone:       user.Phone,
		Status:      model.UserStatus(user.Status),
		CreatedAt:   user.CreatedAt,
		Roles:       []string{}, // 新注册用户暂无角色
		Permissions: []string{}, // 新注册用户暂无权限
		Remark:      user.Remark,
	}

	// 构造响应
	response := &model.RegisterResponse{
		User:    userInfo,
		Message: "注册成功",
	}

	return response, nil
}

// CreateUser 创建用户
// 处理用户创建的完整流程，包括参数验证、重复检查、密码哈希等
func (s *UserService) CreateUser(ctx context.Context, req *model.CreateUserRequest) (*model.User, error) {
	// 参数验证
	if req == nil {
		logger.LogError(errors.New("request is nil"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user",
			"error":     "request is nil",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("创建用户请求不能为空")
	}

	if req.Username == "" {
		logger.LogError(errors.New("username is empty"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user",
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户名不能为空")
	}

	if req.Email == "" {
		logger.LogError(errors.New("email is empty"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user",
			"username":  req.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("邮箱不能为空")
	}

	if req.Password == "" {
		logger.LogError(errors.New("password is empty"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("密码不能为空")
	}

	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		logger.LogError(errors.New("username already exists"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation":        "create_user",
			"username":         req.Username,
			"email":            req.Email,
			"existing_user_id": existingUser.ID,
			"timestamp":        logger.NowFormatted(),
		})
		return nil, errors.New("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingUser, err = s.userRepo.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		logger.LogError(errors.New("email already exists"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation":        "create_user",
			"username":         req.Username,
			"email":            req.Email,
			"existing_user_id": existingUser.ID,
			"timestamp":        logger.NowFormatted(),
		})
		return nil, errors.New("邮箱已存在")
	}

	// 哈希密码（业务逻辑处理）
	hashedPassword, err := s.passwordManager.HashPassword(req.Password)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "hash_password",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 创建用户模型
	user := &model.User{
		Username:  req.Username,
		Email:     req.Email,
		Nickname:  req.Nickname,
		Password:  hashedPassword, // 使用哈希后的密码
		Status:    model.UserStatusEnabled,
		PasswordV: 1, // 设置密码版本
	}

	// 存储到数据库
	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user_db",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 记录成功创建用户的业务日志
	logger.LogBusinessOperation("create_user", user.ID, user.Username, "", "", "success", "User created successfully", map[string]interface{}{
		"email":            user.Email,
		"status":           user.Status,
		"password_version": user.PasswordV,
		"timestamp":        logger.NowFormatted(),
	})

	return user, nil
}

// GetCurrentUser 获取当前用户信息
// 通过访问令牌获取当前登录用户的详细信息
func (s *UserService) GetCurrentUser(ctx context.Context, accessToken string) (*model.UserInfo, error) {
	// 验证访问令牌
	if accessToken == "" {
		logger.LogError(errors.New("access token is empty"), "", 0, "", "get_current_user", "GET", map[string]interface{}{
			"operation": "get_current_user",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("访问令牌不能为空")
	}

	// 解析JWT令牌
	claims, err := s.jwtManager.ValidateAccessToken(accessToken)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_current_user", "GET", map[string]interface{}{
			"operation": "get_current_user",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("无效的访问令牌: %w", err)
	}

	userID := claims.UserID

	// 检查会话是否有效
	sessionData, err := s.redisRepo.GetSession(ctx, uint64(userID))
	if err != nil {
		logger.LogError(err, "", userID, "", "get_current_user", "GET", map[string]interface{}{
			"operation": "get_current_user",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取会话信息失败: %w", err)
	}

	if sessionData == nil {
		logger.LogError(errors.New("session not found"), "", userID, "", "get_current_user", "GET", map[string]interface{}{
			"operation": "get_current_user",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("会话已过期，请重新登录")
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_current_user", "GET", map[string]interface{}{
			"operation": "get_current_user",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		logger.LogError(errors.New("user not found"), "", userID, "", "get_current_user", "GET", map[string]interface{}{
			"operation": "get_current_user",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户不存在")
	}

	// 获取用户角色和权限
	roles, err := s.userRepo.GetUserRoles(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_current_user", "GET", map[string]interface{}{
			"operation": "get_current_user",
			"user_id":   userID,
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户角色失败: %w", err)
	}

	permissions, err := s.userRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_current_user", "GET", map[string]interface{}{
			"operation": "get_current_user",
			"user_id":   userID,
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户权限失败: %w", err)
	}

	// 转换角色和权限为字符串数组
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	permissionNames := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionNames[i] = permission.Name
	}

	// 构造用户信息响应
	userInfo := &model.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Phone:       user.Phone,
		Status:      model.UserStatus(user.Status),
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		Roles:       roleNames,
		Permissions: permissionNames,
		Remark:      user.Remark,
	}

	// 记录成功获取用户信息的业务日志
	logger.LogBusinessOperation("get_current_user", userID, user.Username, "", "", "success", "获取当前用户信息成功", map[string]interface{}{
		"user_id":   userID,
		"username":  user.Username,
		"timestamp": logger.NowFormatted(),
	})

	return userInfo, nil
}

// UpdateUser 更新用户信息
// 处理用户更新的完整流程，包括参数验证、重复检查、密码哈希等
func (s *UserService) UpdateUser(ctx context.Context, userID uint, req *model.UpdateUserRequest) (*model.User, error) {
	// 参数验证
	if userID == 0 {
		return nil, errors.New("用户ID不能为0")
	}

	if req == nil {
		return nil, errors.New("更新用户请求不能为空")
	}

	// 获取现有用户
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 更新字段
	if req.Email != "" && req.Email != user.Email {
		// 检查新邮箱是否已存在
		var existingUser *model.User
		existingUser, err = s.userRepo.GetUserByEmail(ctx, req.Email)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			return nil, errors.New("邮箱已存在")
		}
		user.Email = req.Email
	}

	if req.Status != nil {
		user.Status = *req.Status
	}

	// 如果需要更新密码
	if req.Password != "" {
		var hashedPassword string
		hashedPassword, err = s.passwordManager.HashPassword(req.Password)
		if err != nil {
			return nil, fmt.Errorf("密码哈希失败: %w", err)
		}
		user.Password = hashedPassword
		user.PasswordV++ // 增加密码版本
	}

	// 更新到数据库
	err = s.userRepo.UpdateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	return user, nil
}

// DeleteUser 删除用户
// 处理用户删除的完整流程，包括参数验证、存在性检查等
func (s *UserService) DeleteUser(ctx context.Context, userID uint) error {
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}

	// 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		return errors.New("用户不存在")
	}

	// 调用数据层删除用户
	return s.userRepo.DeleteUser(ctx, userID)
}

// GetUserByID 根据用户ID获取用户
func (s *UserService) GetUserByID(ctx context.Context, userID uint) (*model.User, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为0")
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, errors.New("用户名不能为空")
	}

	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if email == "" {
		return nil, errors.New("邮箱不能为空")
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	return user, nil
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	if offset < 0 {
		offset = 0
	}

	if limit <= 0 || limit > 100 {
		limit = 20 // 默认每页20条
	}

	return s.userRepo.ListUsers(ctx, offset, limit)
}

// GetUserPermissions 获取用户权限
func (s *UserService) GetUserPermissions(ctx context.Context, userID uint) ([]*model.Permission, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为0")
	}

	return s.userRepo.GetUserPermissions(ctx, userID)
}

// GetUserRoles 获取用户角色
func (s *UserService) GetUserRoles(ctx context.Context, userID uint) ([]*model.Role, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为0")
	}

	return s.userRepo.GetUserRoles(ctx, userID)
}

// UpdatePasswordWithVersion 更新用户密码并递增密码版本号
// 这是一个原子操作，确保密码更新和版本号递增同时完成，用于使旧token失效
func (s *UserService) UpdatePasswordWithVersion(ctx context.Context, userID uint, newPassword string) error {
	// 参数验证
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}

	if newPassword == "" {
		return errors.New("新密码不能为空")
	}

	// 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		return errors.New("用户不存在")
	}

	// 哈希新密码
	hashedPassword, err := s.passwordManager.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	// 直接进行数据库更新操作（原子操作）
	// 同时更新密码哈希和递增密码版本号，确保旧token失效
	err = s.userRepo.UpdateUserFields(ctx, userID, map[string]interface{}{
		"password": hashedPassword,
		"password_v":    gorm.Expr("password_v + ?", 1),
		"updated_at":    time.Now(),
	})

	if err != nil {
		return fmt.Errorf("更新密码和版本号失败: %w", err)
	}

	return nil
}

// UpdatePasswordWithVersionHashed 使用已哈希的密码更新用户密码并递增密码版本号
// 这是一个原子操作，确保密码更新和版本号递增同时完成，用于使旧token失效
// 注意：此方法接收已哈希的密码，主要供内部服务调用
func (s *UserService) UpdatePasswordWithVersionHashed(ctx context.Context, userID uint, passwordHash string) error {
	// 参数验证
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}

	if passwordHash == "" {
		return errors.New("密码哈希不能为空")
	}

	// 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		return errors.New("用户不存在")
	}

	// 直接进行数据库更新操作（原子操作）
	// 同时更新密码哈希和递增密码版本号，确保旧token失效
	err = s.userRepo.UpdateUserFields(ctx, userID, map[string]interface{}{
		"password": passwordHash,
		"password_v":    gorm.Expr("password_v + ?", 1),
		"updated_at":    time.Now(),
	})

	if err != nil {
		return fmt.Errorf("更新密码和版本号失败: %w", err)
	}

	return nil
}

// GetUserPasswordVersion 获取用户密码版本号
// 用于密码版本控制，确保修改密码后旧token失效
func (s *UserService) GetUserPasswordVersion(ctx context.Context, userID uint) (int64, error) {
	// 参数验证
	if userID == 0 {
		return 0, errors.New("用户ID不能为0")
	}

	// 调用数据访问层获取密码版本号
	return s.userRepo.GetUserPasswordVersion(ctx, userID)
}

// GetUserWithRolesAndPermissions 获取用户及其角色和权限
func (s *UserService) GetUserWithRolesAndPermissions(ctx context.Context, userID uint) (*model.User, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为0")
	}

	return s.userRepo.GetUserWithRolesAndPermissions(ctx, userID)
}

// AssignRoleToUser 为用户分配角色
func (s *UserService) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	// 参数验证
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}
	if roleID == 0 {
		return errors.New("角色ID不能为0")
	}

	// 调用数据访问层分配角色
	return s.userRepo.AssignRoleToUser(ctx, userID, roleID)
}

// RemoveRoleFromUser 移除用户角色
func (s *UserService) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	// 参数验证
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}
	if roleID == 0 {
		return errors.New("角色ID不能为0")
	}

	// 调用数据访问层移除角色
	return s.userRepo.RemoveRoleFromUser(ctx, userID, roleID)
}

// UpdateLastLogin 更新用户最后登录时间
func (s *UserService) UpdateLastLogin(ctx context.Context, userID uint) error {
	// 参数验证
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}

	// 调用数据访问层更新最后登录时间
	return s.userRepo.UpdateLastLogin(ctx, userID)
}
