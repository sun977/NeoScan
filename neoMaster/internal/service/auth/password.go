package auth

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/pkg/auth"
	"neomaster/internal/repository/mysql"
	"neomaster/internal/repository/redis"
)

// PasswordService 密码管理服务
type PasswordService struct {
	userRepo    *mysql.UserRepository
	sessionRepo *redis.SessionRepository
	passwordMgr *auth.PasswordManager
	cacheExpiry time.Duration
}

// NewPasswordService 创建密码管理服务实例
func NewPasswordService(
	userRepo *mysql.UserRepository,
	sessionRepo *redis.SessionRepository,
	passwordMgr *auth.PasswordManager,
	cacheExpiry time.Duration,
) *PasswordService {
	return &PasswordService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		passwordMgr: passwordMgr,
		cacheExpiry: cacheExpiry,
	}
}

// ChangePassword 修改用户密码并更新密码版本
func (s *PasswordService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	// 验证旧密码
	isValid, err := s.passwordMgr.VerifyPassword(oldPassword, user.Password)
	if err != nil {
		return fmt.Errorf("failed to verify password: %w", err)
	}
	if !isValid {
		return fmt.Errorf("invalid old password")
	}

	// 生成新密码哈希
	newPasswordHash, err := s.passwordMgr.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// 更新密码和版本号（原子操作）
	err = s.userRepo.UpdatePasswordWithVersion(ctx, userID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 获取更新后的密码版本号
	newPasswordV, err := s.userRepo.GetUserPasswordVersion(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get new password version: %w", err)
	}

	// 更新缓存中的密码版本
	err = s.sessionRepo.StorePasswordVersion(ctx, uint64(userID), newPasswordV, s.cacheExpiry)
	if err != nil {
		// 缓存更新失败不应该影响密码修改，只记录错误
		fmt.Printf("Warning: failed to update password version cache: %v\n", err)
	}

	// 删除用户所有会话（强制重新登录）
	err = s.sessionRepo.DeleteAllUserSessions(ctx, uint64(userID))
	if err != nil {
		// 会话删除失败不应该影响密码修改，只记录错误
		fmt.Printf("Warning: failed to delete user sessions: %v\n", err)
	}

	return nil
}

// ResetPassword 重置用户密码（管理员操作）
func (s *PasswordService) ResetPassword(ctx context.Context, userID uint, newPassword string) error {
	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	// 生成新密码哈希
	newPasswordHash, err := s.passwordMgr.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// 更新密码和版本号（原子操作）
	err = s.userRepo.UpdatePasswordWithVersion(ctx, userID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	// 获取更新后的密码版本号
	newPasswordV, err := s.userRepo.GetUserPasswordVersion(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get new password version: %w", err)
	}

	// 更新缓存中的密码版本
	err = s.sessionRepo.StorePasswordVersion(ctx, uint64(userID), newPasswordV, s.cacheExpiry)
	if err != nil {
		// 缓存更新失败不应该影响密码重置，只记录错误
		fmt.Printf("Warning: failed to update password version cache: %v\n", err)
	}

	// 删除用户所有会话（强制重新登录）
	err = s.sessionRepo.DeleteAllUserSessions(ctx, uint64(userID))
	if err != nil {
		// 会话删除失败不应该影响密码重置，只记录错误
		fmt.Printf("Warning: failed to delete user sessions: %v\n", err)
	}

	return nil
}

// SyncPasswordVersionToCache 同步密码版本到缓存
func (s *PasswordService) SyncPasswordVersionToCache(ctx context.Context, userID uint) error {
	// 从数据库获取密码版本
	passwordV, err := s.userRepo.GetUserPasswordVersion(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get password version from database: %w", err)
	}

	// 存储到缓存
	err = s.sessionRepo.StorePasswordVersion(ctx, uint64(userID), passwordV, s.cacheExpiry)
	if err != nil {
		return fmt.Errorf("failed to store password version to cache: %w", err)
	}

	return nil
}

// ValidatePasswordStrength 验证密码强度
func (s *PasswordService) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must be no more than 128 characters long")
	}

	// 可以添加更多密码强度验证规则
	// 例如：必须包含大小写字母、数字、特殊字符等

	return nil
}
