package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/repository/mysql"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService JWT认证服务
type JWTService struct {
	jwtManager *auth.JWTManager
	userRepo   *mysql.UserRepository
}

// NewJWTService 创建JWT服务实例
func NewJWTService(jwtManager *auth.JWTManager, userRepo *mysql.UserRepository) *JWTService {
	return &JWTService{
		jwtManager: jwtManager,
		userRepo:   userRepo,
	}
}

// GenerateTokens 生成访问令牌和刷新令牌
func (s *JWTService) GenerateTokens(ctx context.Context, user *model.User) (*auth.TokenPair, error) {
	if user == nil {
		return nil, errors.New("user cannot be nil")
	}

	// 获取用户角色和权限
	userWithPerms, err := s.userRepo.GetUserWithRolesAndPermissions(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// 构建权限列表（通过角色获取）
	permissions := make([]string, 0)
	for _, role := range userWithPerms.Roles {
		for _, perm := range role.Permissions {
			permissions = append(permissions, fmt.Sprintf("%s:%s", perm.Resource, perm.Action))
		}
	}

	// 构建角色列表
	roles := make([]string, 0, len(userWithPerms.Roles))
	for _, role := range userWithPerms.Roles {
		roles = append(roles, role.Name)
	}

	// 生成令牌对
	tokenPair, err := s.jwtManager.GenerateTokenPair(
		userWithPerms.ID,
		userWithPerms.Username,
		userWithPerms.Email,
		userWithPerms.PasswordV, // 添加密码版本号
		roles,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	return tokenPair, nil
}

// ValidateAccessToken 验证访问令牌
func (s *JWTService) ValidateAccessToken(tokenString string) (*auth.JWTClaims, error) {
	claims, err := s.jwtManager.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	return claims, nil
}

// ValidateRefreshToken 验证刷新令牌
func (s *JWTService) ValidateRefreshToken(tokenString string) (*jwt.RegisteredClaims, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	return claims, nil
}

// RefreshTokens 刷新令牌
func (s *JWTService) RefreshTokens(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// 验证刷新令牌
	_, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 从Subject中获取用户ID
	userID, err := s.jwtManager.GetUserIDFromToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID from token: %w", err)
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive() {
		return nil, errors.New("user is inactive")
	}

	// 生成新的令牌对
	return s.GenerateTokens(ctx, user)
}

// GetUserFromToken 从令牌中获取用户信息
func (s *JWTService) GetUserFromToken(ctx context.Context, tokenString string) (*model.User, error) {
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}

// CheckTokenExpiry 检查令牌是否即将过期
func (s *JWTService) CheckTokenExpiry(tokenString string, threshold time.Duration) (bool, error) {
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		return false, err
	}

	// 检查是否在阈值时间内过期
	if claims.ExpiresAt == nil {
		return false, errors.New("token has no expiry time")
	}
	expiryTime := claims.ExpiresAt.Time
	return time.Until(expiryTime) <= threshold, nil
}

// RevokeToken 撤销令牌（通过黑名单机制）
func (s *JWTService) RevokeToken(ctx context.Context, tokenString string) error {
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		return err
	}

	// 这里可以实现令牌黑名单机制
	// 例如将令牌ID存储到Redis中，直到令牌过期
	// 暂时返回nil，表示撤销成功
	_ = claims
	return nil
}

// GetTokenClaims 获取令牌声明信息
func (s *JWTService) GetTokenClaims(tokenString string) (*auth.JWTClaims, error) {
	return s.ValidateAccessToken(tokenString)
}

// IsTokenValid 检查令牌是否有效
func (s *JWTService) IsTokenValid(tokenString string) bool {
	_, err := s.ValidateAccessToken(tokenString)
	return err == nil
}

// GetTokenRemainingTime 获取令牌剩余有效时间
func (s *JWTService) GetTokenRemainingTime(tokenString string) (time.Duration, error) {
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		return 0, err
	}

	if claims.ExpiresAt == nil {
		return 0, errors.New("token has no expiry time")
	}
	expiryTime := claims.ExpiresAt.Time
	remaining := time.Until(expiryTime)

	if remaining < 0 {
		return 0, errors.New("token has expired")
	}

	return remaining, nil
}

// ValidateUserPermission 验证用户是否具有特定权限
func (s *JWTService) ValidateUserPermission(ctx context.Context, userID uint, resource, action string) (bool, error) {
	permissions, err := s.userRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	for _, perm := range permissions {
		if perm.Resource == resource && perm.Action == action {
			return true, nil
		}
	}

	return false, nil
}

// ValidateUserRole 验证用户是否具有特定角色
func (s *JWTService) ValidateUserRole(ctx context.Context, userID uint, roleName string) (bool, error) {
	roles, err := s.userRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	for _, role := range roles {
		if role.Name == roleName {
			return true, nil
		}
	}

	return false, nil
}

// ValidatePasswordVersion 验证令牌中的密码版本是否与用户当前密码版本匹配
func (s *JWTService) ValidatePasswordVersion(ctx context.Context, tokenString string) (bool, error) {
	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		return false, err
	}

	// 优先从缓存获取密码版本
	currentPasswordV, err := s.userRepo.GetUserPasswordVersion(ctx, uint(claims.UserID))
	if err != nil {
		return false, fmt.Errorf("failed to get user password version: %w", err)
	}

	// 检查密码版本是否匹配
	return claims.PasswordV == currentPasswordV, nil
}
