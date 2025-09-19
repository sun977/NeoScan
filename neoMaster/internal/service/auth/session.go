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
	"neomaster/internal/pkg/utils"
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
// clientIP: 客户端IP地址，从HTTP请求中获取
// userAgent: 用户代理信息，从HTTP请求头中获取
func (s *SessionService) Login(ctx context.Context, req *model.LoginRequest, clientIP, userAgent string) (*model.LoginResponse, error) {
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
		// 如果是用户不存在的错误，尝试通过邮箱查找
		if err.Error() == "用户不存在" {
			user, err = s.userService.GetUserByEmail(ctx, req.Username)
			if err != nil {
				// 邮箱查找也失败，记录日志并返回错误
				logger.LogError(err, "", 0, "", "user_login", "POST", map[string]interface{}{
					"operation": "login",
					"username":  req.Username,
					"error":     "user_not_found",
					"timestamp": logger.NowFormatted(),
				})
				return nil, errors.New("invalid username or password")
			}
		} else {
			// 其他数据库错误
			logger.LogError(err, "", 0, "", "user_login", "POST", map[string]interface{}{
				"operation": "login",
				"username":  req.Username,
				"error":     "database_error",
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

	// 标准化IP，并更新最后登录时间与IP
	normalizedIP := utils.NormalizeIP(clientIP)
	err = s.userService.UpdateLastLogin(ctx, user.ID, normalizedIP)
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
		ClientIP:    normalizedIP, // 经过标准化的客户端IP
		UserAgent:   userAgent,    // 从请求上下文获取的用户代理
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
			LastLoginIP: user.LastLoginIP, // 添加最后登录IP
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

	// 应该是解析accessToken获取user信息
	claims, err := s.jwtService.ValidateAccessToken(accessToken)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_logout", "POST", map[string]interface{}{
			"operation":    "logout",
			"token_prefix": accessToken[:10] + "...",
			"timestamp":    logger.NowFormatted(),
		})
		return fmt.Errorf("failed to validate access token: %w", err)
	}

	user, err := s.userService.GetUserByID(ctx, claims.UserID)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_logout", "POST", map[string]interface{}{
			"operation":    "logout",
			"token_prefix": accessToken[:10] + "...",
			"timestamp":    logger.NowFormatted(),
		})
		// 继续执行撤销操作，即使获取用户信息失败
	}

	// 撤销令牌（将令牌添加到黑名单--添加到redis缓存中--"revoked:token:20250919173619-856583300"）
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

	// 刷新令牌(包含验证刷新令牌)
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
	// 验证令牌是否为空
	if accessToken == "" {
		return nil, errors.New("access token cannot be empty")
	}

	// 验证令牌有效性
	// 1.这里会检查令牌格式、签名有效性、是否过期等
	// 2.检查令牌是否在黑名单中（已被撤销）
	// 验证成功返回解析出的JWT声明信息，失败返回错误信息
	claims, err := s.jwtService.ValidateAccessToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// 使用JWT声明中的用户ID获取用户信息
	user, err := s.userService.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
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

// StorePasswordVersion 存储用户密码版本到缓存
func (s *SessionService) StorePasswordVersion(ctx context.Context, userID uint, passwordVersion int64, expiration time.Duration) error {
	return s.sessionRepo.StorePasswordVersion(ctx, uint64(userID), passwordVersion, expiration)
}

// DeleteAllUserSessions 删除用户的所有会话
func (s *SessionService) DeleteAllUserSessions(ctx context.Context, userID uint) error {
	return s.sessionRepo.DeleteAllUserSessions(ctx, uint64(userID))
}

// RevokeToken 撤销令牌（添加到黑名单）
// 实现TokenBlacklistService接口
// 参数:
//   - ctx: 请求上下文
//   - jti: JWT ID（令牌唯一标识符）
//   - expiration: 黑名单过期时间
//
// 返回: 错误信息
func (s *SessionService) RevokeToken(ctx context.Context, jti string, expiration time.Duration) error {
	if jti == "" {
		logger.LogError(errors.New("token JTI cannot be empty"), "", 0, "", "revoke_token", "POST", map[string]interface{}{
			"operation": "revoke_token",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("token JTI cannot be empty")
	}

	// 调用SessionRepository的RevokeToken方法
	// 这里遵循了层级调用关系：Service → Repository
	err := s.sessionRepo.RevokeToken(ctx, jti, expiration)
	if err != nil {
		logger.LogError(err, "", 0, "", "revoke_token", "POST", map[string]interface{}{
			"operation": "revoke_token",
			"jti":       jti,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// 记录令牌撤销的业务日志
	logger.LogBusinessOperation("revoke_token", 0, "", "", "", "success", "令牌撤销成功", map[string]interface{}{
		"jti":       jti,
		"timestamp": logger.NowFormatted(),
	})

	return nil
}

// IsTokenRevoked 检查令牌是否已被撤销
// 实现TokenBlacklistService接口
// 参数:
//   - ctx: 请求上下文
//   - jti: JWT ID（令牌唯一标识符）
//
// 返回: 是否已撤销, 错误信息
func (s *SessionService) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, errors.New("token JTI cannot be empty")
	}

	// 调用SessionRepository的IsTokenRevoked方法
	// 这里遵循了层级调用关系：Service → Repository
	isRevoked, err := s.sessionRepo.IsTokenRevoked(ctx, jti)
	if err != nil {
		// 记录错误日志，但不记录业务日志（这是一个查询操作）
		logger.LogError(err, "", 0, "", "check_token_revoked", "GET", map[string]interface{}{
			"operation": "check_token_revoked",
			"jti":       jti,
			"timestamp": logger.NowFormatted(),
		})
		return false, fmt.Errorf("failed to check token revocation status: %w", err)
	}

	return isRevoked, nil
}

// 主要用于handler层sessionHandler的实现
// GetUserSessions 获取指定用户的所有会话
func (s *SessionService) GetUserSessions(ctx context.Context, userID uint) ([]*model.SessionData, error) {
	if userID == 0 {
		return nil, errors.New("userID cannot be zero")
	}
	sessions, err := s.sessionRepo.GetUserSessions(ctx, uint64(userID))
	if err != nil {
		logger.LogError(err, "", userID, "", "get_user_sessions", "GET", map[string]interface{}{
			"operation": "get_user_sessions",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	return sessions, nil
}

// DeleteUserSession 撤销指定用户当前会话
func (s *SessionService) DeleteUserSession(ctx context.Context, userID uint) error {
	if userID == 0 {
		return errors.New("userID cannot be zero")
	}
	if err := s.sessionRepo.DeleteSession(ctx, uint64(userID)); err != nil {
		logger.LogError(err, "", userID, "", "delete_user_session", "POST", map[string]interface{}{
			"operation": "delete_user_session",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("failed to delete user session: %w", err)
	}
	logger.LogBusinessOperation("delete_user_session", userID, "", "", "", "success", "用户会话撤销成功", map[string]interface{}{
		"user_id":   userID,
		"timestamp": logger.NowFormatted(),
	})
	return nil
}
