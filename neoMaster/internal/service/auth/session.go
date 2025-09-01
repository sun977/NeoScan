package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/repository/mysql"
)

// SessionService 会话管理服务
type SessionService struct {
	userRepo        *mysql.UserRepository
	passwordManager *auth.PasswordManager
	jwtService      *JWTService
	rbacService     *RBACService
}

// NewSessionService 创建会话服务实例
func NewSessionService(
	userRepo *mysql.UserRepository,
	passwordManager *auth.PasswordManager,
	jwtService *JWTService,
	rbacService *RBACService,
) *SessionService {
	return &SessionService{
		userRepo:        userRepo,
		passwordManager: passwordManager,
		jwtService:      jwtService,
		rbacService:     rbacService,
	}
}

// Login 用户登录
func (s *SessionService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	if req == nil {
		return nil, errors.New("login request cannot be nil")
	}

	if req.Username == "" {
		return nil, errors.New("username cannot be empty")
	}

	if req.Password == "" {
		return nil, errors.New("password cannot be empty")
	}

	// 根据用户名或邮箱查找用户
	var user *model.User
	var err error

	// 尝试通过用户名查找
	user, err = s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		// 如果通过用户名没找到，尝试通过邮箱查找
		user, err = s.userRepo.GetUserByEmail(ctx, req.Username)
		if err != nil {
			// 用户不存在，返回统一的错误信息以保护隐私
			return nil, errors.New("invalid username or password")
		}
	}

	if user == nil {
		return nil, errors.New("invalid username or password")
	}

	// 检查用户是否激活
	if !user.IsActive() {
		return nil, errors.New("user account is inactive")
	}

	// 验证密码
	isValid, err := s.passwordManager.VerifyPassword(req.Password, user.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}
	if !isValid {
		return nil, errors.New("invalid username or password")
	}

	// 生成令牌
	tokenPair, err := s.jwtService.GenerateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// 更新最后登录时间
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// 记录错误但不影响登录流程
		fmt.Printf("Warning: failed to update last login time: %v\n", err)
	}

	// 获取用户角色和权限信息
	userWithPerms, err := s.userRepo.GetUserWithRolesAndPermissions(ctx, user.ID)
	if err != nil {
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
		return errors.New("access token cannot be empty")
	}

	// 撤销令牌
	if err := s.jwtService.RevokeToken(ctx, accessToken); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

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

// GetCurrentUser 获取当前用户信息
func (s *SessionService) GetCurrentUser(ctx context.Context, accessToken string) (*model.UserInfo, error) {
	if accessToken == "" {
		return nil, errors.New("access token cannot be empty")
	}

	// 从令牌获取用户信息
	user, err := s.jwtService.GetUserFromToken(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from token: %w", err)
	}

	// 获取用户角色和权限
	userWithPerms, err := s.userRepo.GetUserWithRolesAndPermissions(ctx, user.ID)
	if err != nil {
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

	return &model.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Phone:       user.Phone,
		Status:      user.Status,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		Roles:       roles,
		Permissions: permissions,
	}, nil
}

// ChangePassword 修改密码
func (s *SessionService) ChangePassword(ctx context.Context, userID uint, req *model.ChangePasswordRequest) error {
	if req == nil {
		return errors.New("change password request cannot be nil")
	}

	if userID == 0 {
		return errors.New("invalid user ID")
	}

	if req.OldPassword == "" {
		return errors.New("old password cannot be empty")
	}

	if req.NewPassword == "" {
		return errors.New("new password cannot be empty")
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return errors.New("user not found")
	}

	// 验证旧密码
	isValid, err := s.passwordManager.VerifyPassword(req.OldPassword, user.Password)
	if err != nil {
		return fmt.Errorf("failed to verify password: %w", err)
	}
	if !isValid {
		return errors.New("old password is incorrect")
	}

	// 验证新密码强度（需要实现密码强度验证函数）
	if len(req.NewPassword) < 8 {
		return errors.New("new password must be at least 8 characters long")
	}

	// 生成新密码哈希
	newPasswordHash, err := s.passwordManager.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// 更新密码和版本号（原子操作，确保旧token失效）
	if err := s.userRepo.UpdatePasswordWithVersion(ctx, userID, newPasswordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
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
