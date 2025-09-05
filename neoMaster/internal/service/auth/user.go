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
