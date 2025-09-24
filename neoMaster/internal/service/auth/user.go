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
	"strings"
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
// 增加注册源头IP
func (s *UserService) Register(ctx context.Context, req *model.RegisterRequest, clientIP string) (*model.RegisterResponse, error) {
	// 参数验证
	if req == nil {
		logger.LogError(errors.New("register request is nil"), "", 0, clientIP, "user_register", "POST", map[string]interface{}{
			"operation": "register",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("注册请求不能为空")
	}

	if req.Username == "" {
		logger.LogError(errors.New("username is empty"), "", 0, clientIP, "user_register", "POST", map[string]interface{}{
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
		Username:    req.Username,
		Email:       req.Email,
		Nickname:    req.Nickname,
		Password:    hashedPassword, // 使用哈希后的密码
		Phone:       req.Phone,
		Status:      model.UserStatusEnabled,
		PasswordV:   1,        // 设置密码版本
		LastLoginIP: clientIP, // 注册时记录注册IP到 LastLoginIP 字段
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
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Phone:     user.Phone,
		Status:    model.UserStatus(user.Status),
		CreatedAt: user.CreatedAt,
		// Roles:       []string{}, // 新注册用户暂无角色
		// Permissions: []string{}, // 新注册用户暂无权限
		// Remark: user.Remark,
	}

	// 构造响应
	response := &model.RegisterResponse{
		User:    userInfo,
		Message: "registration successful",
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
		Phone:     req.Phone,
		Remark:    req.Remark,
		// // 将角色ID转换为角色对象切片
		// Roles: func() []*model.Role {
		// 	roles := make([]*model.Role, len(req.RoleIDs))
		// 	for i, id := range req.RoleIDs {
		// 		roles[i] = &model.Role{ID: id}
		// 	}
		// 	return roles
		// }(),
	}

	// 处理角色关联(将角色ID转换为角色对象切片)
	if req.RoleIDs != nil {
		roles := make([]*model.Role, len(req.RoleIDs))
		for i, id := range req.RoleIDs {
			roles[i] = &model.Role{ID: id}
		}
		user.Roles = roles
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

// GetUserIDFromToken 从JWT令牌中获取用户ID
// 通过解析JWT访问令牌获取用户ID，用于身份验证
func (s *UserService) GetUserIDFromToken(ctx context.Context, accessToken string) (uint, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// 参数验证：访问令牌不能为空
	if accessToken == "" {
		logger.LogError(errors.New("access token is empty"), "", 0, "", "get_user_id_from_token", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"timestamp": logger.NowFormatted(),
		})
		return 0, errors.New("访问令牌不能为空")
	}

	// 解析并验证JWT令牌
	claims, err := s.jwtManager.ValidateAccessToken(accessToken)
	if err != nil {
		// 记录JWT验证失败日志
		logger.LogError(err, "", 0, "", "get_user_id_from_token", "SERVICE", map[string]interface{}{
			"operation": "jwt_validation",
			"timestamp": logger.NowFormatted(),
		})
		return 0, fmt.Errorf("无效的访问令牌: %w", err)
	}

	// 提取用户ID
	userID := claims.UserID
	if userID == 0 {
		// 记录用户ID为空的错误
		logger.LogError(errors.New("user ID is zero in token claims"), "", 0, "", "get_user_id_from_token", "SERVICE", map[string]interface{}{
			"operation": "claims_validation",
			"timestamp": logger.NowFormatted(),
		})
		return 0, errors.New("令牌中用户ID无效")
	}

	return userID, nil
}

// GetCurrentUserInfo 获取当前用户信息（从访问令牌获取用户ID）
// 通过访问令牌获取当前登录用户的详细信息
func (s *UserService) GetCurrentUserInfo(ctx context.Context, accessToken string) (*model.UserInfo, error) {
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

// GetUserInfoByID 根据用户ID获取用户完整信息（包含角色和权限）
// 直接通过用户ID获取用户详细信息，跳过会话验证
// 用于已通过中间件验证的场景
func (s *UserService) GetUserInfoByID(ctx context.Context, userID uint) (*model.UserInfo, error) {
	// 参数验证：用户ID必须有效
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID: cannot be zero"), "", 0, "", "get_user_info_by_id", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户ID不能为0")
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_user_info_by_id", "SERVICE", map[string]interface{}{
			"operation": "get_user_info_by_id",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		logger.LogError(errors.New("user not found"), "", userID, "", "get_user_info_by_id", "SERVICE", map[string]interface{}{
			"operation": "get_user_info_by_id",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户不存在")
	}

	// 获取用户角色和权限
	roles, err := s.userRepo.GetUserRoles(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_user_info_by_id", "SERVICE", map[string]interface{}{
			"operation": "get_user_info_by_id",
			"user_id":   userID,
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户角色失败: %w", err)
	}

	permissions, err := s.userRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_user_info_by_id", "SERVICE", map[string]interface{}{
			"operation": "get_user_info_by_id",
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
	logger.LogBusinessOperation("get_user_info_by_id", userID, user.Username, "", "", "success", "根据用户ID获取用户信息成功", map[string]interface{}{
		"user_id":   userID,
		"username":  user.Username,
		"timestamp": logger.NowFormatted(),
	})

	return userInfo, nil
}

// UpdateUserByID 更新用户信息
// 处理用户更新的完整流程，包括参数验证、重复检查、密码哈希、事务处理等
// @param ctx 上下文
// @param userID 用户ID
// @param req 更新用户请求（管理员使用）
// @return 更新后的用户信息和错误
func (s *UserService) UpdateUserByID(ctx context.Context, userID uint, req *model.UpdateUserRequest) (*model.User, error) {
	// 第一层：参数验证层
	if err := s.validateUpdateUserParams(userID, req); err != nil {
		// userID 不为 0
		// 请求包 req 不为空
		// 邮箱字段格式验证
		// 密码字段强度验证(6<PASS<128 包含一个字母一个数字)
		// 用户状态值校验(激活|禁用)
		return nil, err
	}

	// 第二层：业务规则验证层
	user, err := s.validateUserForUpdate(ctx, userID, req)
	// 验证用户是否存在
	// 验证用户是否被删除
	// 管理员角色不能被禁用(userID != 1)
	// 用户名字段满足唯一性
	// 邮箱字段满足唯一性
	// 角色id的有效性校验
	if err != nil {
		return nil, err
	}

	// 第三层：事务处理层
	return s.executeUserUpdate(ctx, user, req)
}

// validateUpdateUserParams 验证更新用户的参数
func (s *UserService) validateUpdateUserParams(userID uint, req *model.UpdateUserRequest) error {
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID for update"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"error":     "user_id_zero",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户ID不能为0")
	}

	if req == nil {
		logger.LogError(errors.New("update request is nil"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"error":     "request_nil",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("更新用户请求不能为空")
	}

	// 验证邮箱格式
	if req.Email != "" {
		if !s.isValidEmail(req.Email) {
			logger.LogError(errors.New("invalid email format"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "parameter_validation",
				"user_id":   userID,
				"email":     req.Email,
				"error":     "invalid_email_format",
				"timestamp": logger.NowFormatted(),
			})
			return errors.New("邮箱格式无效")
		}
	}

	// 验证密码强度
	if req.Password != "" {
		if err := auth.ValidatePasswordStrength(req.Password); err != nil {
			logger.LogError(err, "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "parameter_validation",
				"user_id":   userID,
				"error":     "password_strength_validation_failed",
				"timestamp": logger.NowFormatted(),
			})
			return fmt.Errorf("密码强度验证失败: %w", err)
		}
	}

	// 验证状态值(激活|禁用)
	if req.Status != nil {
		if *req.Status < 0 || *req.Status > 2 {
			logger.LogError(errors.New("invalid status value"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "parameter_validation",
				"user_id":   userID,
				"status":    *req.Status,
				"error":     "invalid_status_value",
				"timestamp": logger.NowFormatted(),
			})
			return errors.New("用户状态值无效，必须为0(禁用)、1(启用)")
		}
	}

	return nil
}

// validateUserForUpdate 验证用户是否可以更新
func (s *UserService) validateUserForUpdate(ctx context.Context, userID uint, req *model.UpdateUserRequest) (*model.User, error) {
	// 检查用户是否存在(User 模型字段都可以获得)
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "user_existence_check",
			"user_id":   userID,
			"error":     "database_query_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("get user failed: %w", err)
	}

	if user == nil {
		logger.LogError(errors.New("user not found for update"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "user_existence_check",
			"user_id":   userID,
			"error":     "user_not_found",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("user not found")
	}

	// 检查用户状态 - 已删除的用户不能更新
	if user.DeletedAt != nil {
		logger.LogError(errors.New("user already deleted"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "user_status_check",
			"user_id":   userID,
			"error":     "user_already_deleted",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("user already deleted, cannot update")
	}

	// 业务规则：系统管理员账户的特殊限制
	if userID == 1 {
		// 系统管理员不能被禁用或锁定
		if req.Status != nil && *req.Status != 1 {
			logger.LogError(errors.New("cannot disable system admin"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "business_rule_check",
				"user_id":   userID,
				"status":    *req.Status,
				"error":     "system_admin_status_change_forbidden",
				"timestamp": logger.NowFormatted(),
			})
			return nil, errors.New("cannot disable or lock system admin account")
		}
	}

	// 用户名唯一性
	if req.Username != "" && req.Username != user.Username {
		existingUser, err := s.userRepo.GetUserByUsername(ctx, req.Username)
		if err != nil {
			logger.LogError(err, "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "username_uniqueness_check",
				"user_id":   userID,
				"username":  req.Username,
				"error":     "database_query_failed",
				"timestamp": logger.NowFormatted(),
			})
			// 只记录错误，不返回，返回会终结服务，新用户名在数据库中可以不存在
			// return nil, fmt.Errorf("check username uniqueness failed: %w", err)
		}
		if existingUser != nil && existingUser.ID != userID {
			logger.LogError(errors.New("username already exists"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation":        "username_uniqueness_check",
				"user_id":          userID,
				"username":         req.Username,
				"existing_user_id": existingUser.ID,
				"error":            "username_already_exists",
				"timestamp":        logger.NowFormatted(),
			})
			return nil, errors.New("username already exists")
		}
		return user, nil
	}

	// 检查邮箱唯一性
	if req.Email != "" && req.Email != user.Email {
		existingUser, err := s.userRepo.GetUserByEmail(ctx, req.Email)
		if err != nil {
			logger.LogError(err, "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "email_uniqueness_check",
				"user_id":   userID,
				"email":     req.Email,
				"error":     "database_query_failed",
				"timestamp": logger.NowFormatted(),
			})
			// 只记录错误，不返回，返回会终结服务，新邮箱在数据库中可以不存在
			// return nil, fmt.Errorf("check email uniqueness failed: %w", err)
		}
		if existingUser != nil && existingUser.ID != userID {
			logger.LogError(errors.New("email already exists"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation":        "email_uniqueness_check",
				"user_id":          userID,
				"email":            req.Email,
				"existing_user_id": existingUser.ID,
				"error":            "email_already_exists",
				"timestamp":        logger.NowFormatted(),
			})
			return nil, errors.New("email already exists")
		}
	}

	// 如果角色字段不空，则验证角色ID有效性[判断roleID是否存在]
	if req.RoleIDs != nil {
		for _, roleID := range req.RoleIDs {
			roleExists, err := s.userRepo.UserRoleExistsByID(ctx, roleID)
			if err != nil {
				logger.LogError(err, "", 0, "", "update_user", "SERVICE", map[string]interface{}{
					"operation": "role_existence_check",
					"user_id":   userID,
					"role_id":   roleID,
					"error":     "database_query_failed",
					"timestamp": logger.NowFormatted(),
				})
				// 角色不能在数据库中不存在，这里要返回终结服务
				return nil, fmt.Errorf("check role existence failed: %w", err)
			}
			if !roleExists {
				logger.LogError(errors.New("role not found"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
					"operation": "role_existence_check",
					"user_id":   userID,
					"role_id":   roleID,
					"error":     "role_not_found",
					"timestamp": logger.NowFormatted(),
				})
				return nil, errors.New("role not found")
			}
		}
	}

	return user, nil
}

// executeUserUpdate 执行用户更新操作（包含事务处理）
func (s *UserService) executeUserUpdate(ctx context.Context, user *model.User, req *model.UpdateUserRequest) (*model.User, error) {
	// 开始事务
	tx := s.userRepo.BeginTx(ctx)
	if tx == nil {
		logger.LogError(errors.New("failed to begin transaction"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "transaction_begin",
			"user_id":   user.ID,
			"error":     "transaction_begin_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("开始事务失败")
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.LogError(fmt.Errorf("panic during user update: %v", r), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "panic_recovery",
				"user_id":   user.ID,
				"error":     "panic_occurred",
				"panic":     r,
				"timestamp": logger.NowFormatted(),
			})
		}
	}()

	// 记录更新前的状态
	oldEmail := user.Email
	oldStatus := user.Status
	passwordChanged := false

	// 应用更新
	if req.Username != "" && req.Username != user.Username {
		user.Username = req.Username
	}

	if req.Nickname != "" && req.Nickname != user.Nickname {
		user.Nickname = req.Nickname
	}

	if req.Phone != "" && req.Phone != user.Phone {
		user.Phone = req.Phone
	}

	if req.Email != "" && req.Email != user.Email {
		user.Email = req.Email
	}

	if req.Avatar != "" && req.Avatar != user.Avatar {
		user.Avatar = req.Avatar
	}

	if req.Remark != "" && req.Remark != user.Remark {
		user.Remark = req.Remark
	}

	if req.Status != nil && *req.Status != user.Status {
		user.Status = *req.Status
	}

	// 如果需要更新密码
	if req.Password != "" {
		hashedPassword, err := s.passwordManager.HashPassword(req.Password)
		if err != nil {
			tx.Rollback()
			logger.LogError(err, "", 0, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "password_hash",
				"user_id":   user.ID,
				"error":     "password_hash_failed",
				"timestamp": logger.NowFormatted(),
			})
			return nil, fmt.Errorf("密码哈希失败: %w", err)
		}
		user.Password = hashedPassword
		user.PasswordV++ // 增加密码版本
		passwordChanged = true
	}

	// 更新用户角色信息(如果有指定角色ID)
	if req.RoleIDs != nil {
		// 获取用户角色信息封装到结构体中
		roles := make([]*model.Role, len(req.RoleIDs))
		for i, id := range req.RoleIDs {
			roles[i] = &model.Role{ID: id}
		}
		user.Roles = roles

		// 先删除旧的角色关联
		if err := s.userRepo.DeleteUserRolesByUserID(ctx, tx, user.ID); err != nil {
			tx.Rollback()
			logger.LogError(err, "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
				"operation": "cascade_delete_user_roles",
				"user_id":   user.ID,
				"error":     "delete_user_roles_failed",
				"timestamp": logger.NowFormatted(),
			})
			return nil, fmt.Errorf("删除用户角色关联失败: %w", err)
		}

		// 然后更新为新的角色关联(后续更新操作 UpdateUserWithTx 会创建新的关联)

	}

	// socket_id 更新
	if req.SocketID != "" && req.SocketID != user.SocketId {
		user.SocketId = req.SocketID
	}

	// 更新到数据库
	if err := s.userRepo.UpdateUserWithTx(ctx, tx, user); err != nil {
		tx.Rollback()
		logger.LogError(err, "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "database_update",
			"user_id":   user.ID,
			"error":     "update_user_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.LogError(err, "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "transaction_commit",
			"user_id":   user.ID,
			"error":     "transaction_commit_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 记录成功更新日志
	changes := make(map[string]interface{})
	if req.Email != "" && req.Email != oldEmail {
		changes["email_changed"] = map[string]string{"from": oldEmail, "to": req.Email}
	}
	if req.Status != nil && *req.Status != oldStatus {
		changes["status_changed"] = map[string]int{"from": int(oldStatus), "to": int(*req.Status)}
	}
	if passwordChanged {
		changes["password_changed"] = true
		changes["password_version"] = user.PasswordV
	}

	logger.LogBusinessOperation("update_user", user.ID, user.Username, "", "", "success", "用户更新成功", map[string]interface{}{
		"operation":  "user_update_success",
		"user_id":    user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"status":     user.Status,
		"changes":    changes,
		"updated_at": logger.NowFormatted(),
		"timestamp":  logger.NowFormatted(),
	})

	return user, nil
}

// UserUpdateInfoByID 更新用户信息
// 处理用户更新的完整流程，包括参数验证、重复检查、密码哈希、事务处理等
// @param ctx 上下文
// @param userID 用户ID
// @param req 更新用户请求（用户专用）
// @return 更新后的用户信息和错误
func (s *UserService) UserUpdateInfoByID(ctx context.Context, userID uint, req *model.UpdateUserRequest) (*model.User, error) {
	// 第一层：参数验证层
	if err := s.validateUserUpdateInfoParams(userID, req); err != nil {
		// userID 不为 0
		// 请求包 req 不为空
		// 邮箱字段格式验证
		// 密码字段强度验证(6<PASS<128 包含一个字母一个数字)
		// 用户状态值校验(激活|禁用)
		return nil, err
	}

	// 第二层：业务规则验证层
	user, err := s.validateUserUpdateInfo(ctx, userID, req)
	// 验证用户是否存在
	// 验证用户是否被删除
	// 管理员角色不能被禁用(userID != 1)
	// 用户名字段满足唯一性
	// 邮箱字段满足唯一性
	// 角色id的有效性校验
	if err != nil {
		return nil, err
	}

	// 第三层：事务处理层 返回 user
	return s.executeUserUpdateInfo(ctx, user, req)
}

// validateUpdateUserParams 验证更新用户的参数
func (s *UserService) validateUserUpdateInfoParams(userID uint, req *model.UpdateUserRequest) error {
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID for update"), "", 0, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"error":     "user_id_zero",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("user id cannot be zero")
	}

	if req == nil {
		logger.LogError(errors.New("update request is nil"), "", userID, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"error":     "request_nil",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("update user request cannot be empty")
	}

	// 验证邮箱格式
	if req.Email != "" {
		if !s.isValidEmail(req.Email) {
			logger.LogError(errors.New("invalid email format"), "", userID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "parameter_validation",
				"user_id":   userID,
				"email":     req.Email,
				"error":     "invalid_email_format",
				"timestamp": logger.NowFormatted(),
			})
			return errors.New("email format is invalid")
		}
	}

	// 验证密码强度
	if req.Password != "" {
		if err := auth.ValidatePasswordStrength(req.Password); err != nil {
			logger.LogError(err, "", userID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "parameter_validation",
				"user_id":   userID,
				"error":     "password_strength_validation_failed",
				"timestamp": logger.NowFormatted(),
			})
			return fmt.Errorf("password strength validation failed: %w", err)
		}
	}

	// 验证状态值(激活|禁用)
	if req.Status != nil {
		if *req.Status < 0 || *req.Status > 2 {
			logger.LogError(errors.New("invalid status value"), "", userID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "parameter_validation",
				"user_id":   userID,
				"status":    *req.Status,
				"error":     "invalid_status_value",
				"timestamp": logger.NowFormatted(),
			})
			return errors.New("invalid status value, must be 0-disabled, 1-enabled")
		}
	}

	return nil
}

// validateUserForUpdate 验证用户是否可以更新
func (s *UserService) validateUserUpdateInfo(ctx context.Context, userID uint, req *model.UpdateUserRequest) (*model.User, error) {
	// 检查用户是否存在(User 模型字段都可以获得)
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "user_existence_check",
			"user_id":   userID,
			"error":     "database_query_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("get user failed: %w", err)
	}

	if user == nil {
		logger.LogError(errors.New("user not found for update"), "", userID, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "user_existence_check",
			"user_id":   userID,
			"error":     "user_not_found",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("user not found")
	}

	// 检查用户状态 - 已删除的用户不能更新
	if user.DeletedAt != nil {
		logger.LogError(errors.New("user already deleted"), "", userID, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "user_status_check",
			"user_id":   userID,
			"error":     "user_already_deleted",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("user already deleted, cannot update")
	}

	// 业务规则：系统管理员账户的特殊限制
	if userID == 1 {
		// 系统管理员不能被禁用或锁定
		if req.Status != nil && *req.Status != 1 {
			logger.LogError(errors.New("cannot disable system admin"), "", userID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "business_rule_check",
				"user_id":   userID,
				"status":    *req.Status,
				"error":     "system_admin_status_change_forbidden",
				"timestamp": logger.NowFormatted(),
			})
			return nil, errors.New("cannot disable or lock system admin account")
		}
	}

	// 用户名唯一性
	if req.Username != "" && req.Username != user.Username {
		existingUser, err := s.userRepo.GetUserByUsername(ctx, req.Username)
		if err != nil {
			logger.LogError(err, "", userID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "username_uniqueness_check",
				"user_id":   userID,
				"username":  req.Username,
				"error":     "database_query_failed",
				"timestamp": logger.NowFormatted(),
			})
			// 新用户名可以在数据库中不存在，这里不能返回，会终结服务
			// return nil, fmt.Errorf("check username uniqueness failed: %w", err)
		}
		if existingUser != nil && existingUser.ID != userID {
			logger.LogError(errors.New("username already exists"), "", userID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation":        "username_uniqueness_check",
				"user_id":          userID,
				"username":         req.Username,
				"existing_user_id": existingUser.ID,
				"error":            "username_already_exists",
				"timestamp":        logger.NowFormatted(),
			})
			return nil, errors.New("username already exists")
		}
		return user, nil
	}

	// 检查邮箱唯一性
	if req.Email != "" && req.Email != user.Email {
		existingUser, err := s.userRepo.GetUserByEmail(ctx, req.Email)
		if err != nil {
			logger.LogError(err, "", userID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "email_uniqueness_check",
				"user_id":   userID,
				"email":     req.Email,
				"error":     "database_query_failed",
				"timestamp": logger.NowFormatted(),
			})
			// 新邮箱可以在数据库中不存在，这里不能返回，会终结服务
			// return nil, fmt.Errorf("check email uniqueness failed: %w", err)
		}

		if existingUser != nil && existingUser.ID != userID {
			logger.LogError(errors.New("email already exists"), "", userID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation":        "email_uniqueness_check",
				"user_id":          userID,
				"email":            req.Email,
				"existing_user_id": existingUser.ID,
				"error":            "email_already_exists",
				"timestamp":        logger.NowFormatted(),
			})
			return nil, errors.New("email already exists")
		}
	}
	return user, nil
}

// executeUserUpdate 执行用户更新操作（包含事务处理）
func (s *UserService) executeUserUpdateInfo(ctx context.Context, user *model.User, req *model.UpdateUserRequest) (*model.User, error) {
	// 开始事务
	tx := s.userRepo.BeginTx(ctx)
	if tx == nil {
		logger.LogError(errors.New("failed to begin transaction"), "", user.ID, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "transaction_begin",
			"user_id":   user.ID,
			"error":     "transaction_begin_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("开始事务失败")
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.LogError(fmt.Errorf("panic during user update: %v", r), "", user.ID, "", "update_user", "SERVICE", map[string]interface{}{
				"operation": "panic_recovery",
				"user_id":   user.ID,
				"error":     "panic_occurred",
				"panic":     r,
				"timestamp": logger.NowFormatted(),
			})
		}
	}()

	// 记录更新前的状态
	oldEmail := user.Email
	oldUsername := user.Username
	oldNickname := user.Nickname
	oldPhone := user.Phone
	oldAvatar := user.Avatar
	oldRemark := user.Remark
	oldSocketID := user.SocketId

	// 应用更新
	if req.Username != "" && req.Username != user.Username {
		user.Username = req.Username
	}

	if req.Nickname != "" && req.Nickname != user.Nickname {
		user.Nickname = req.Nickname
	}

	if req.Phone != "" && req.Phone != user.Phone {
		user.Phone = req.Phone
	}

	if req.Email != "" && req.Email != user.Email {
		user.Email = req.Email
	}

	if req.Avatar != "" && req.Avatar != user.Avatar {
		user.Avatar = req.Avatar
	}

	if req.Remark != "" && req.Remark != user.Remark {
		user.Remark = req.Remark
	}

	// 不允许修改自己的状态

	// 密码修改有专门的接口，这里不允许修改

	// 更新用户角色信息（用户不允许修改自己的角色）

	// socket_id 更新（暂时保留）
	if req.SocketID != "" && req.SocketID != user.SocketId {
		user.SocketId = req.SocketID
	}

	// 更新到数据库
	if err := s.userRepo.UpdateUserWithTx(ctx, tx, user); err != nil {
		tx.Rollback()
		logger.LogError(err, "", user.ID, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "database_update",
			"user_id":   user.ID,
			"error":     "update_user_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.LogError(err, "", user.ID, "", "update_user", "SERVICE", map[string]interface{}{
			"operation": "transaction_commit",
			"user_id":   user.ID,
			"error":     "transaction_commit_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 记录成功更新日志
	changes := make(map[string]interface{})
	if req.Email != "" && req.Email != oldEmail {
		changes["email_changed"] = map[string]string{"from": oldEmail, "to": req.Email}
	}
	if req.Username != "" && req.Username != oldUsername {
		changes["username_changed"] = map[string]string{"from": oldUsername, "to": req.Username}
	}
	if req.Nickname != "" && req.Nickname != oldNickname {
		changes["nickname_changed"] = map[string]string{"from": oldNickname, "to": req.Nickname}
	}
	if req.Phone != "" && req.Phone != oldPhone {
		changes["phone_changed"] = map[string]string{"from": oldPhone, "to": req.Phone}
	}
	if req.Avatar != "" && req.Avatar != oldAvatar {
		changes["avatar_changed"] = map[string]string{"from": oldAvatar, "to": req.Avatar}
	}
	if req.Remark != "" && req.Remark != oldRemark {
		changes["remark_changed"] = map[string]string{"from": oldRemark, "to": req.Remark}
	}
	if req.SocketID != "" && req.SocketID != oldSocketID {
		changes["socket_id_changed"] = map[string]string{"from": oldSocketID, "to": req.SocketID}
	}

	logger.LogBusinessOperation("update_user", user.ID, user.Username, "", "", "success", "用户更新用户信息成功", map[string]interface{}{
		"operation":  "user_update_success",
		"user_id":    user.ID,
		"username":   user.Username,
		"changes":    changes,
		"updated_at": logger.NowFormatted(),
		"timestamp":  logger.NowFormatted(),
	})

	return user, nil
}

// isValidEmail 验证邮箱格式
func (s *UserService) isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	if len(email) < 5 || len(email) > 254 {
		return false
	}
	// 检查是否包含@符号
	if !strings.Contains(email, "@") {
		return false
	}
	// 检查@符号的位置
	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	// 检查域名部分是否包含点
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

// DeleteUser 删除用户
// 完整的业务逻辑包括：参数验证、业务规则检查、级联删除、事务处理、审计日志
func (s *UserService) DeleteUser(ctx context.Context, userID uint) error {
	// 第一层：参数验证层
	if err := s.validateDeleteUserParams(userID); err != nil {
		return err
	}

	// 第二层：业务规则验证层
	user, err := s.validateUserForDeletion(ctx, userID)
	if err != nil {
		return err
	}

	// 第三层：事务处理层
	return s.executeUserDeletion(ctx, user)
}

// validateDeleteUserParams 验证删除用户的参数
func (s *UserService) validateDeleteUserParams(userID uint) error {
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID for deletion"), "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"error":     "user_id_zero",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户ID不能为0")
	}
	return nil
}

// validateUserForDeletion 验证用户是否可以删除
func (s *UserService) validateUserForDeletion(ctx context.Context, userID uint) (*model.User, error) {
	// 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "user_existence_check",
			"user_id":   userID,
			"error":     "database_query_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		logger.LogError(errors.New("user not found for deletion"), "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "user_existence_check",
			"user_id":   userID,
			"error":     "user_not_found",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户不存在")
	}

	// 检查用户状态 - 已删除的用户不能再次删除
	if user.DeletedAt != nil {
		logger.LogError(errors.New("user already deleted"), "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "user_status_check",
			"user_id":   userID,
			"error":     "user_already_deleted",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户已被删除")
	}

	// 业务规则：检查是否为系统管理员（ID为1的用户不能删除）
	if userID == 1 {
		logger.LogError(errors.New("cannot delete system admin"), "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "business_rule_check",
			"user_id":   userID,
			"error":     "system_admin_deletion_forbidden",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("不能删除系统管理员账户")
	}

	return user, nil
}

// executeUserDeletion 执行用户删除操作（包含事务处理）
func (s *UserService) executeUserDeletion(ctx context.Context, user *model.User) error {
	// 开始事务
	tx := s.userRepo.BeginTx(ctx)
	if tx == nil {
		logger.LogError(errors.New("failed to begin transaction"), "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "transaction_begin",
			"user_id":   user.ID,
			"error":     "transaction_begin_failed",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("开始事务失败")
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.LogError(fmt.Errorf("panic during user deletion: %v", r), "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
				"operation": "panic_recovery",
				"user_id":   user.ID,
				"error":     "panic_occurred",
				"panic":     r,
				"timestamp": logger.NowFormatted(),
			})
		}
	}()

	// 1. 删除用户角色关联
	if err := s.userRepo.DeleteUserRolesByUserID(ctx, tx, user.ID); err != nil {
		tx.Rollback()
		logger.LogError(err, "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "cascade_delete_user_roles",
			"user_id":   user.ID,
			"error":     "delete_user_roles_failed",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("删除用户角色关联失败: %w", err)
	}

	// 2. 软删除用户
	if err := s.userRepo.DeleteUserWithTx(ctx, tx, user.ID); err != nil {
		tx.Rollback()
		logger.LogError(err, "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "soft_delete_user",
			"user_id":   user.ID,
			"error":     "delete_user_failed",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("删除用户失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		logger.LogError(err, "", 0, "", "delete_user", "SERVICE", map[string]interface{}{
			"operation": "transaction_commit",
			"user_id":   user.ID,
			"error":     "transaction_commit_failed",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 记录成功删除日志
	logger.LogBusinessOperation("delete_user", user.ID, user.Username, "", "", "success", "用户删除成功", map[string]interface{}{
		"operation":  "user_deletion_success",
		"user_id":    user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"deleted_at": logger.NowFormatted(),
		"timestamp":  logger.NowFormatted(),
	})

	return nil
}

// GetUserByID 根据用户ID获取用户
// 完整的业务逻辑包括：参数验证、上下文检查、数据获取、状态验证、日志记录
func (s *UserService) GetUserByID(ctx context.Context, userID uint) (*model.User, error) {
	// 参数验证：用户ID必须有效
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID: cannot be zero"), "", 0, "", "get_user_by_id", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户ID不能为0")
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 从数据库获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		// 记录数据库查询失败日志
		logger.LogError(err, "", userID, "", "get_user_by_id", "SERVICE", map[string]interface{}{
			"operation": "database_query",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 检查用户是否存在
	if user == nil {
		// 记录用户不存在日志
		logger.LogError(errors.New("user not found"), "", userID, "", "get_user_by_id", "SERVICE", map[string]interface{}{
			"operation": "user_not_found",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户不存在")
	}

	// 记录成功获取用户信息的业务日志
	logger.LogBusinessOperation("get_user_by_id", userID, user.Username, "", "", "success", "用户信息获取成功", map[string]interface{}{
		"operation":   "get_user_success",
		"user_id":     userID,
		"username":    user.Username,
		"user_status": user.Status,
		"timestamp":   logger.NowFormatted(),
	})

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

// GetUserList 获取用户列表
// 提供分页查询功能，包含完整的参数验证和错误处理
// 参数:
//   - ctx: 上下文，用于超时控制和取消操作
//   - offset: 偏移量，必须 >= 0
//   - limit: 每页数量，范围 [1, 100]，默认20
//
// 返回:
//   - []*model.User: 用户列表
//   - int64: 总记录数
//   - error: 错误信息
func (s *UserService) GetUserList(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	// 保存原始参数值用于日志记录
	originalOffset := offset
	originalLimit := limit

	// 参数验证：偏移量不能为负数
	if offset < 0 {
		logger.LogError(fmt.Errorf("invalid offset parameter: %d", offset), "", 0, "", "get_user_list", "SERVICE", map[string]interface{}{
			"operation": "get_user_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		offset = 0 // 自动修正为0
	}

	// 参数验证：限制每页数量的合理范围
	if limit <= 0 {
		limit = 20 // 默认每页20条
	} else if limit > 100 {
		limit = 100 // 最大每页100条，防止查询过大数据集
	}

	// 记录参数修正日志（如果发生了修正）
	if originalLimit != limit || originalOffset != offset {
		logger.LogBusinessOperation("get_user_list", 0, "system", "", "", "parameter_corrected", "分页参数已自动修正", map[string]interface{}{
			"operation":        "get_user_list",
			"original_offset":  originalOffset,
			"original_limit":   originalLimit,
			"corrected_offset": offset,
			"corrected_limit":  limit,
			"timestamp":        logger.NowFormatted(),
		})
	}

	// 上下文检查：确保请求未被取消
	select {
	case <-ctx.Done():
		return nil, 0, fmt.Errorf("request cancelled: %w", ctx.Err())
	default:
		// 继续执行
	}

	// 调用repository层获取数据
	users, total, err := s.userRepo.GetUserList(ctx, offset, limit)
	if err != nil {
		// 记录数据库查询错误
		logger.LogError(err, "", 0, "", "get_user_list", "SERVICE", map[string]interface{}{
			"operation": "get_user_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, fmt.Errorf("failed to get user list from repository: %w", err)
	}

	// 数据完整性检查
	if users == nil {
		users = make([]*model.User, 0) // 确保返回空切片而不是nil
	}

	// 记录成功操作日志
	logger.LogBusinessOperation("get_user_list", 0, "system", "", "", "success", "获取用户列表成功", map[string]interface{}{
		"operation":    "get_user_list",
		"offset":       offset,
		"limit":        limit,
		"total":        total,
		"result_count": len(users),
		"timestamp":    logger.NowFormatted(),
	})

	return users, total, nil
}

// GetUserPermissions 获取用户权限
// 通过用户角色关联查询获取用户的所有权限，自动去重
// 注意：此函数应该只在已通过JWT中间件验证的上下文中调用
// 参数:
//   - ctx: 请求上下文，用于超时控制和链路追踪
//   - userID: 用户唯一标识ID，必须大于0
//
// 返回:
//   - []*model.Permission: 用户权限列表，已去重
//   - error: 错误信息，包含参数验证、用户存在性检查和数据库操作错误
func (s *UserService) GetUserPermissions(ctx context.Context, userID uint) ([]*model.Permission, error) {
	// 参数验证：用户ID必须有效
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID: cannot be zero"), "", 0, "", "get_user_permissions", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户ID不能为0")
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 首先验证用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_user_permissions", "SERVICE", map[string]interface{}{
			"operation": "check_user_existence",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 用户不存在
	if user == nil {
		logger.LogError(errors.New("user not found"), "", userID, "", "get_user_permissions", "SERVICE", map[string]interface{}{
			"operation": "user_not_found",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户不存在")
	}

	// 检查用户状态是否有效（只有启用状态的用户才能获取权限）
	if user.Status != model.UserStatusEnabled {
		logger.LogError(errors.New("user status invalid"), "", userID, user.Username, "get_user_permissions", "SERVICE", map[string]interface{}{
			"operation":   "check_user_status",
			"user_id":     userID,
			"username":    user.Username,
			"user_status": user.Status,
			"timestamp":   logger.NowFormatted(),
		})
		return nil, errors.New("用户状态无效，无法获取权限")
	}

	// 获取用户权限
	permissions, err := s.userRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, user.Username, "get_user_permissions", "SERVICE", map[string]interface{}{
			"operation": "get_permissions_from_repo",
			"user_id":   userID,
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户权限失败: %w", err)
	}

	// 记录成功获取权限的业务日志
	permissionNames := make([]string, len(permissions))
	for i, perm := range permissions {
		permissionNames[i] = perm.GetFullName()
	}

	logger.LogBusinessOperation("get_user_permissions", userID, user.Username, "", "", "success",
		fmt.Sprintf("成功获取用户权限，共%d个权限", len(permissions)), map[string]interface{}{
			"user_id":          userID,
			"username":         user.Username,
			"permission_count": len(permissions),
			"permissions":      permissionNames,
			"timestamp":        logger.NowFormatted(),
		})

	return permissions, nil
}

// GetUserRoles 获取用户角色
// 通过用户角色关联查询获取用户的所有角色信息
// 注意：此函数应该只在已通过JWT中间件验证的上下文中调用
// 参数:
//   - ctx: 请求上下文，用于超时控制和链路追踪
//   - userID: 用户唯一标识ID，必须大于0
//
// 返回:
//   - []*model.Role: 用户角色列表，包含角色的完整信息
//   - error: 错误信息，包含参数验证、用户存在性检查和数据库操作错误
func (s *UserService) GetUserRoles(ctx context.Context, userID uint) ([]*model.Role, error) {
	// 参数验证：用户ID必须有效
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID: cannot be zero"), "", 0, "", "get_user_roles", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户ID不能为0")
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 首先验证用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_user_roles", "SERVICE", map[string]interface{}{
			"operation": "check_user_existence",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 用户不存在
	if user == nil {
		logger.LogError(errors.New("user not found"), "", userID, "", "get_user_roles", "SERVICE", map[string]interface{}{
			"operation": "user_not_found",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户不存在")
	}

	// 检查用户状态是否有效（只有启用状态的用户才能获取角色）
	if user.Status != model.UserStatusEnabled {
		logger.LogError(errors.New("user status invalid"), "", userID, user.Username, "get_user_roles", "SERVICE", map[string]interface{}{
			"operation":   "check_user_status",
			"user_id":     userID,
			"username":    user.Username,
			"user_status": user.Status,
			"timestamp":   logger.NowFormatted(),
		})
		return nil, errors.New("用户状态无效，无法获取角色")
	}

	// 获取用户角色
	roles, err := s.userRepo.GetUserRoles(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, user.Username, "get_user_roles", "SERVICE", map[string]interface{}{
			"operation": "get_roles_from_repo",
			"user_id":   userID,
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取用户角色失败: %w", err)
	}

	// 记录成功获取角色的业务日志
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	logger.LogBusinessOperation("get_user_roles", userID, user.Username, "", "", "success",
		fmt.Sprintf("成功获取用户角色，共%d个角色", len(roles)), map[string]interface{}{
			"user_id":    userID,
			"username":   user.Username,
			"role_count": len(roles),
			"roles":      roleNames,
			"timestamp":  logger.NowFormatted(),
		})

	return roles, nil
}

// UpdatePasswordWithVersion 更新用户密码并递增密码版本号(没有使用)
// 这是一个原子操作，确保密码更新和版本号递增同时完成
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
		"password":   hashedPassword,
		"password_v": gorm.Expr("password_v + ?", 1),
		"updated_at": time.Now(),
	})

	if err != nil {
		return fmt.Errorf("更新密码和版本号失败: %w", err)
	}

	return nil
}

// UpdatePasswordWithVersionHashed 使用已哈希的密码更新用户密码并递增密码版本号
// 这是一个原子操作，确保密码更新和版本号递增同时完成，用于修改用户密码
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
		"password":   passwordHash,
		"password_v": gorm.Expr("password_v + ?", 1),
		"updated_at": time.Now(),
	})

	if err != nil {
		return fmt.Errorf("更新密码和版本号失败: %w", err)
	}

	return nil
}

// GetUserPasswordVersion 获取用户密码版本号
// 用于密码版本控制，确保修改密码后旧token失效
// 注意：此方法接收已哈希的密码，主要供内部服务调用
func (s *UserService) GetUserPasswordVersion(ctx context.Context, userID uint) (int64, error) {
	// 参数验证
	if userID == 0 {
		return 0, errors.New("用户ID不能为0")
	}

	// 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		return 0, errors.New("用户不存在")
	}

	// 调用数据访问层获取密码版本号
	return s.userRepo.GetUserPasswordVersion(ctx, userID)
}

// UpdateUserPasswordVersion 更新用户密码版本号
// 用于密码版本控制，确保修改密码后旧token失效
// 注意：此方法接收密码版本号，主要供内部服务调用
func (s *UserService) UpdateUserPasswordVersion(ctx context.Context, userID uint, passwordV int64) error {
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}

	if passwordV < 0 {
		return errors.New("密码版本号不能小于0")
	}

	// 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		return errors.New("用户不存在")
	}

	return s.userRepo.UpdatePasswordVersion(ctx, userID, passwordV)
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

// UpdateLastLogin 更新用户最后登录时间（包含客户端IP）
func (s *UserService) UpdateLastLogin(ctx context.Context, userID uint, clientIP string) error {
	// 参数验证
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}

	// 调用数据访问层更新最后登录时间与IP
	return s.userRepo.UpdateLastLogin(ctx, userID, clientIP)
}

// UpdateUserStatus 更新用户状态 - 通用状态管理函数
// 将指定用户的状态设置为启用或禁用状态，消除重复代码，体现"好品味"原则
// @param ctx 上下文
// @param userID 用户ID
// @param status 目标状态 (1: 启用, 0: 禁用) 【禁止禁用userID=1的管理员】
// @return 错误信息
func (s *UserService) UpdateUserStatus(ctx context.Context, userID uint, status model.UserStatus) error {
	// 参数验证层 - 消除特殊情况
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID"), "", 0, "", "update_user_status", "SERVICE", map[string]interface{}{
			"operation": "update_user_status",
			"error":     "invalid_user_id",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户ID不能为0")
	}

	// 验证状态值有效性 - 严格的参数检查
	if status != model.UserStatusEnabled && status != model.UserStatusDisabled {
		logger.LogError(errors.New("invalid status value"), "", 0, "", "update_user_status", "SERVICE", map[string]interface{}{
			"operation": "update_user_status",
			"error":     "invalid_status_value",
			"user_id":   userID,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户状态值无效,必须为0(禁用)或1(启用)")
	}

	// 业务规则验证层 - 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "update_user_status", "SERVICE", map[string]interface{}{
			"operation": "update_user_status",
			"error":     "get_user_failed",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		logger.LogError(errors.New("user not found"), "", userID, "", "update_user_status", "SERVICE", map[string]interface{}{
			"operation": "update_user_status",
			"error":     "user_not_found",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户不存在")
	}

	// 幂等性检查 - 避免无意义操作
	if user.Status == status {
		statusText := "禁用"
		if status == model.UserStatusEnabled {
			statusText = "启用"
		}

		logger.LogBusinessOperation("update_user_status", userID, user.Username, "", "", "success",
			fmt.Sprintf("用户已处于%s状态", statusText), map[string]interface{}{
				"operation":      "update_user_status",
				"user_id":        userID,
				"username":       user.Username,
				"current_status": status,
				"target_status":  status,
				"timestamp":      logger.NowFormatted(),
			})
		return nil
	}

	// 业务规则：系统管理员保护机制
	if userID == 1 && status == model.UserStatusDisabled {
		logger.LogError(errors.New("cannot disable system admin"), "", 0, "", "update_user_status", "SERVICE", map[string]interface{}{
			"operation": "business_rule_check",
			"user_id":   userID,
			"status":    status,
			"error":     "system_admin_status_change_forbidden",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("不能禁用系统管理员账户")
	}

	// 数据操作层 - 执行状态更新
	// 使用 UpdateUserFields 进行原子更新操作
	updateFields := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	err = s.userRepo.UpdateUserFields(ctx, userID, updateFields)
	if err != nil {
		statusText := "禁用"
		if status == model.UserStatusEnabled {
			statusText = "启用"
		}

		logger.LogError(err, "", userID, "", "update_user_status", "SERVICE", map[string]interface{}{
			"operation": "update_user_status",
			"error":     fmt.Sprintf("%s_failed", statusText),
			"user_id":   userID,
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("%s用户失败: %w", statusText, err)
	}

	// 审计日志层 - 记录成功操作
	statusText := "禁用"
	statusTextOpposite := "启用"
	if status == model.UserStatusEnabled {
		statusText = "启用"
		statusTextOpposite = "禁用"
	}

	logger.LogBusinessOperation("update_user_status", userID, user.Username, "", "", "success",
		fmt.Sprintf("用户%s成功", statusText), map[string]interface{}{
			"operation":       "update_user_status",
			"user_id":         userID,
			"username":        user.Username,
			"previous_status": statusTextOpposite,
			"new_status":      statusText,
			"target_status":   int(status),
			"timestamp":       logger.NowFormatted(),
		})

	return nil
}

// ActivateUser 激活用户 - 语义化包装函数，保持向后兼容
// 将指定用户的状态设置为启用状态
// @param ctx 上下文
// @param userID 用户ID
// @return 错误信息
func (s *UserService) ActivateUser(ctx context.Context, userID uint) error {
	// 调用通用状态更新函数，体现"好品味"原则：消除特殊情况
	return s.UpdateUserStatus(ctx, userID, model.UserStatusEnabled)
}

// DeactivateUser 禁用用户 - 语义化包装函数
// 将指定用户的状态设置为禁用状态
// @param ctx 上下文
// @param userID 用户ID
// @return 错误信息
func (s *UserService) DeactivateUser(ctx context.Context, userID uint) error {
	// 调用通用状态更新函数，体现"好品味"原则：消除特殊情况
	return s.UpdateUserStatus(ctx, userID, model.UserStatusDisabled)
}

// 重置用户密码(管理员操作)
func (s *UserService) ResetUserPassword(ctx context.Context, userID uint, newPassword string) error {
	// 参数验证
	if userID == 0 {
		logger.LogError(errors.New("invalid user ID for password reset"), "", 0, "", "reset_user_password", "SERVICE", map[string]interface{}{
			"operation": "parameter_validation",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户ID不能为0")
	}

	// 固定为安全的简单默认密码（满足最小要求）
	const defaultPassword = "123456"

	// 获取用户以进行存在性和日志校验
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "reset_user_password", "SERVICE", map[string]interface{}{
			"operation": "get_user_for_reset",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取用户失败: %w", err)
	}
	if user == nil {
		logger.LogError(errors.New("user not found for password reset"), "", userID, "", "reset_user_password", "SERVICE", map[string]interface{}{
			"operation": "user_existence_check",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户不存在")
	}

	// 生成新密码哈希
	passwordHash, err := s.passwordManager.HashPassword(defaultPassword)
	if err != nil {
		logger.LogError(err, "", userID, user.Username, "reset_user_password", "SERVICE", map[string]interface{}{
			"operation": "hash_password",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("新密码哈希失败: %w", err)
	}

	// 使用原子方法更新密码并递增版本号
	if err = s.UpdatePasswordWithVersionHashed(ctx, userID, passwordHash); err != nil {
		logger.LogError(err, "", userID, user.Username, "reset_user_password", "SERVICE", map[string]interface{}{
			"operation": "update_password_with_version",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("重置密码失败: %w", err)
	}

	// 获取新的密码版本（用于日志或后续同步）
	newPasswordV, err := s.GetUserPasswordVersion(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, user.Username, "reset_user_password", "SERVICE", map[string]interface{}{
			"operation": "get_new_password_version",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		// 不影响主流程
	}

	// 删除用户所有会话（尽力而为，不影响主流程）
	if derr := s.redisRepo.DeleteAllUserSessions(ctx, uint64(userID)); derr != nil {
		logger.LogError(derr, "", userID, user.Username, "reset_user_password", "SERVICE", map[string]interface{}{
			"operation": "delete_user_sessions",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
	}

	// 记录成功操作
	logger.LogBusinessOperation("reset_user_password", userID, user.Username, "", "", "success", "用户密码重置成功", map[string]interface{}{
		"user_id":              userID,
		"username":             user.Username,
		"new_password_version": newPasswordV,
		"timestamp":            logger.NowFormatted(),
	})

	return nil
}
