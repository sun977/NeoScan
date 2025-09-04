/*
 * @author: sun977
 * @date: 2025.09.04
 * @description: 会话管理服务
 * @func:
 * 1.登录
 * 2.注销
 * 3.刷新会话
 * 4.获取会话信息
 * 5.会话状态检查
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
	"neomaster/internal/repository/redis"
)

// SessionService 会话管理服务
type SessionService struct {
	userService     *UserService
	passwordManager *auth.PasswordManager
	jwtService      *JWTService
	rbacService     *RBACService
	sessionRepo     *redis.SessionRepository
}

// NewSessionService 创建会话服务实例
func NewSessionService(
	userService *UserService,
	passwordManager *auth.PasswordManager,
	jwtService *JWTService,
	rbacService *RBACService,
	sessionRepo *redis.SessionRepository,
) *SessionService {
	return &SessionService{
		userService:     userService,
		passwordManager: passwordManager,
		jwtService:      jwtService,
		rbacService:     rbacService,
		sessionRepo:     sessionRepo,
	}
}

// Login 用户登录
func (s *SessionService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	if req == nil {
		logger.LogError(errors.New("login request cannot be nil"), "", 0, "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("login request cannot be nil")
	}

	if req.Username == "" {
		logger.LogError(errors.New("username cannot be empty"), "", 0, "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("username cannot be empty")
	}

	if req.Password == "" {
		logger.LogError(errors.New("password cannot be empty"), "", 0, "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("password cannot be empty")
	}

	// 根据用户名或邮箱查找用户
	var user *model.User
	var err error

	// 尝试通过用户名查找
	user, err = s.userService.GetUserByUsername(ctx, req.Username)
	if err != nil {
		// 数据库查询出错
		logger.LogError(err, "", 0, "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"username":  req.Username,
			"error":     "database_error_username",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("invalid username or password")
	}

	// 如果通过用户名没找到，尝试通过邮箱查找
	if user == nil {
		user, err = s.userService.GetUserByEmail(ctx, req.Username)
		if err != nil {
			// 数据库查询出错
			logger.LogError(err, "", 0, "", "user_login", "POST", map[string]interface{}{
				"operation": "login",
				"username":  req.Username,
				"error":     "database_error_email",
				"timestamp": logger.NowFormatted(),
			})
			return nil, errors.New("invalid username or password")
		}
	}

	// 如果用户不存在（两种方式都没找到）
	if user == nil {
		logger.LogError(fmt.Errorf("user not found"), "", 0, "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"username":  req.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("invalid username or password")
	}

	// 检查用户是否激活
	if !user.IsActive() {
		logger.LogError(errors.New("user account is not active"), "", uint(user.ID), "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"username":  user.Username,
			"status":    user.Status,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("user account is inactive")
	}

	// 验证密码
	isValid, err := s.passwordManager.VerifyPassword(req.Password, user.Password)
	if err != nil {
		logger.LogError(err, "", uint(user.ID), "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}
	if !isValid {
		logger.LogError(errors.New("password is incorrect"), "", uint(user.ID), "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("invalid username or password")
	}

	// 生成令牌
	tokenPair, err := s.jwtService.GenerateTokens(ctx, user)
	if err != nil {
		logger.LogError(err, "", uint(user.ID), "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// 更新最后登录时间
	err = s.userService.UpdateLastLogin(ctx, user.ID)
	if err != nil {
		// 记录错误但不影响登录流程
		fmt.Printf("Warning: failed to update last login time: %v\n", err)
	}

	// 获取用户角色和权限信息
	userWithPerms, err := s.userService.GetUserWithRolesAndPermissions(ctx, user.ID)
	if err != nil {
		logger.LogError(err, "", uint(user.ID), "", "user_login", "POST", map[string]interface{}{
			"operation": "login",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// 构建角色列表
	roles := make([]string, 0, len(userWithPerms.Roles))
	for _, role := range userWithPerms.Roles {
		roles = append(roles, role.Name)
	}

	// 构建权限列表（通过角色获取）
	permissions := make([]string, 0)
	for _, role := range userWithPerms.Roles {
		for _, perm := range role.Permissions {
			permissions = append(permissions, fmt.Sprintf("%s:%s", perm.Resource, perm.Action))
		}
	}

	// 存储会话信息到Redis
	sessionData := &model.SessionData{
		UserID:      user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Roles:       roles,
		Permissions: permissions,
		LoginTime:   time.Now(),
		LastActive:  time.Now(),
		ClientIP:    "", // TODO: 从请求上下文获取客户端IP
		UserAgent:   "", // TODO: 从请求上下文获取用户代理
	}

	// 设置会话过期时间（与访问令牌过期时间一致）
	sessionExpiration := time.Duration(tokenPair.ExpiresIn) * time.Second
	err = s.sessionRepo.StoreSession(ctx, uint64(user.ID), sessionData, sessionExpiration)
	if err != nil {
		logger.LogError(err, "", uint(user.ID), "", "user_login", "POST", map[string]interface{}{
			"operation": "store_session",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		// 会话存储失败不影响登录，但记录警告
		fmt.Printf("Warning: failed to store session: %v\n", err)
	}

	// 记录成功登录的业务日志
	logger.LogBusinessOperation("user_login", uint(user.ID), user.Username, "", "", "success", "用户登录成功", map[string]interface{}{
		"email":       user.Email,
		"roles":       roles,
		"permissions": permissions,                        // 添加权限信息到日志中
		"session_id":  tokenPair.AccessToken[:10] + "...", // 只记录token前缀
		"timestamp":   logger.NowFormatted(),
	})

	return &model.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User: &model.User{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			Nickname:    user.Nickname,
			Avatar:      user.Avatar,
			Phone:       user.Phone,
			Status:      user.Status,
			LastLoginAt: user.LastLoginAt,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Roles:       userWithPerms.Roles,
		},
	}, nil
}

// Logout 用户登出
func (s *SessionService) Logout(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		logger.LogError(errors.New("access token cannot be empty"), "", 0, "", "user_logout", "POST", map[string]interface{}{
			"operation": "logout",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("access token cannot be empty")
	}

	// 获取用户信息用于日志记录
	user, err := s.jwtService.GetUserFromToken(ctx, accessToken)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_logout", "POST", map[string]interface{}{
			"operation":    "logout",
			"token_prefix": accessToken[:10] + "...",
			"timestamp":    logger.NowFormatted(),
		})
		// 继续执行撤销操作，即使获取用户信息失败
	}

	// 撤销令牌
	if err := s.jwtService.RevokeToken(ctx, accessToken); err != nil {
		logger.LogError(err, "", 0, "", "user_logout", "POST", map[string]interface{}{
			"operation":    "logout",
			"token_prefix": accessToken[:10] + "...",
			"timestamp":    logger.NowFormatted(),
		})
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// 记录成功登出的业务日志
	logData := map[string]interface{}{
		"token_prefix": accessToken[:10] + "...",
		"timestamp":    time.Now(),
	}
	if user != nil {
		logData["user_id"] = user.ID
		logData["username"] = user.Username
	}
	var userID uint = 0
	var username string = ""
	if user != nil {
		userID = uint(user.ID)
		username = user.Username
	}
	logger.LogBusinessOperation("user_logout", userID, username, "", "", "success", "用户登出成功", logData)

	return nil
}

// RefreshToken 刷新令牌
func (s *SessionService) RefreshToken(ctx context.Context, req *model.RefreshTokenRequest) (*model.RefreshTokenResponse, error) {
	if req == nil {
		return nil, errors.New("refresh token request cannot be nil")
	}

	if req.RefreshToken == "" {
		return nil, errors.New("refresh token cannot be empty")
	}

	// 刷新令牌
	tokenPair, err := s.jwtService.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tokens: %w", err)
	}

	return &model.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

// ValidateSession 验证会话
func (s *SessionService) ValidateSession(ctx context.Context, accessToken string) (*model.User, error) {
	if accessToken == "" {
		return nil, errors.New("access token cannot be empty")
	}

	// 验证令牌并获取用户信息
	user, err := s.jwtService.GetUserFromToken(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// 检查用户是否仍然活跃
	if !user.IsActive() {
		return nil, errors.New("user account is inactive")
	}

	return user, nil
}

// CheckPermission 检查用户权限
func (s *SessionService) CheckPermission(ctx context.Context, userID uint, resource, action string) (bool, error) {
	return s.rbacService.CheckPermission(ctx, userID, resource, action)
}

// CheckRole 检查用户角色
func (s *SessionService) CheckRole(ctx context.Context, userID uint, roleName string) (bool, error) {
	return s.rbacService.CheckRole(ctx, userID, roleName)
}

// IsTokenExpiringSoon 检查令牌是否即将过期
func (s *SessionService) IsTokenExpiringSoon(accessToken string, threshold time.Duration) (bool, error) {
	return s.jwtService.CheckTokenExpiry(accessToken, threshold)
}

// GetTokenRemainingTime 获取令牌剩余时间
func (s *SessionService) GetTokenRemainingTime(accessToken string) (time.Duration, error) {
	return s.jwtService.GetTokenRemainingTime(accessToken)
}
