/*
 * @author: sun977
 * @date: 2025.09.04
 * @description: 用户服务
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

	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/repository/mysql"
	"neomaster/internal/repository/redis"
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

	// 创建用户请求
	createUserReq := &model.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
	}

	// 创建用户
	user, err := s.userRepo.CreateUser(ctx, createUserReq)
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
		Roles:     []string{}, // 新注册用户暂无角色
		Permissions: []string{}, // 新注册用户暂无权限
		Remark:    user.Remark,
	}

	// 构造响应
	response := &model.RegisterResponse{
		User:    userInfo,
		Message: "注册成功",
	}

	return response, nil
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
