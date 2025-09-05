package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/logger"
)

// PasswordService 密码服务
// 负责密码相关的业务逻辑，包括密码修改、重置、验证等
type PasswordService struct {
	userService     *UserService          // 用户业务服务
	sessionService  *SessionService       // 会话服务
	passwordManager *auth.PasswordManager // 密码管理器
	cacheExpiry     time.Duration
}

// NewPasswordService 创建密码管理服务实例
func NewPasswordService(
	userService *UserService,
	sessionService *SessionService,
	passwordManager *auth.PasswordManager,
	cacheExpiry time.Duration,
) *PasswordService {
	return &PasswordService{
		userService:     userService,
		sessionService:  sessionService,
		passwordManager: passwordManager,
		cacheExpiry:     cacheExpiry,
	}
}

// ChangePassword 修改用户密码并更新密码版本
// 包含完整的参数验证、密码验证、日志记录和会话清理逻辑
func (s *PasswordService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	// 参数验证
	if userID == 0 {
		logger.LogError(errors.New("user ID is zero"), "", userID, "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户ID不能为0")
	}

	if oldPassword == "" {
		logger.LogError(errors.New("old password is empty"), "", userID, "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("原密码不能为空")
	}

	if newPassword == "" {
		logger.LogError(errors.New("new password is empty"), "", userID, "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("新密码不能为空")
	}

	// 验证新密码强度
	if len(newPassword) < 8 {
		logger.LogError(errors.New("new password must be at least 8 characters long"), "", userID, "", "password_change", "PUT", map[string]interface{}{
			"operation":       "change_password",
			"password_length": len(newPassword),
			"timestamp":       logger.NowFormatted(),
		})
		return errors.New("新密码长度至少为8位")
	}

	// 获取用户信息
	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		logger.LogError(errors.New("user not found"), "", userID, "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户不存在")
	}

	// 验证旧密码
	isValid, err := s.passwordManager.VerifyPassword(oldPassword, user.Password)
	if err != nil {
		logger.LogError(err, "", userID, user.Username, "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("密码验证失败: %w", err)
	}

	if !isValid {
		logger.LogError(errors.New("old password is incorrect"), "", userID, user.Username, "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("原密码错误")
	}

	// 生成新密码哈希
	newPasswordHash, err := s.passwordManager.HashPassword(newPassword)
	if err != nil {
		logger.LogError(err, "", userID, user.Username, "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("新密码哈希失败: %w", err)
	}

	// 更新密码和版本号（原子操作，确保旧token失效）
	err = s.userService.UpdatePasswordWithVersionHashed(ctx, userID, newPasswordHash)
	if err != nil {
		logger.LogError(err, "", userID, user.Username, "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("更新密码失败: %w", err)
	}

	// 获取新的密码版本
	newPasswordV, err := s.userService.GetUserPasswordVersion(ctx, userID)
	if err != nil {
		logger.LogError(err, "", userID, user.Username, "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取新密码版本失败: %w", err)
	}

	// 更新缓存中的密码版本
	err = s.sessionService.StorePasswordVersion(ctx, userID, newPasswordV, s.cacheExpiry)
	if err != nil {
		// 缓存更新失败不应该影响密码修改，只记录错误
		logger.LogError(err, "", userID, user.Username, "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"step":      "update_cache",
			"timestamp": logger.NowFormatted(),
		})
	}

	// 删除用户所有会话（强制重新登录）
	err = s.sessionService.DeleteAllUserSessions(ctx, userID)
	if err != nil {
		// 会话删除失败不应该影响密码修改，只记录错误
		logger.LogError(err, "", userID, user.Username, "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"step":      "delete_sessions",
			"timestamp": logger.NowFormatted(),
		})
	}

	// 记录成功修改密码的业务日志
	logger.LogBusinessOperation("password_change", userID, user.Username, "", "", "success", "用户修改密码成功", map[string]interface{}{
		"old_password_version": newPasswordV - 1,
		"new_password_version": newPasswordV,
		"timestamp":            logger.NowFormatted(),
	})

	return nil
}

// ResetPassword 重置用户密码（管理员操作）
func (s *PasswordService) ResetPassword(ctx context.Context, userID uint, newPassword string) error {
	// 获取用户信息
	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	// 生成新密码哈希
	newPasswordHash, err := s.passwordManager.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// 更新密码和版本号（原子操作）
	err = s.userService.UpdatePasswordWithVersionHashed(ctx, userID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	// 获取更新后的密码版本号
	newPasswordV, err := s.userService.GetUserPasswordVersion(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get new password version: %w", err)
	}

	// 更新缓存中的密码版本
	err = s.sessionService.StorePasswordVersion(ctx, userID, newPasswordV, s.cacheExpiry)
	if err != nil {
		// 缓存更新失败不应该影响密码重置，只记录错误
		fmt.Printf("Warning: failed to update password version cache: %v\n", err)
	}

	// 删除用户所有会话（强制重新登录）
	err = s.sessionService.DeleteAllUserSessions(ctx, userID)
	if err != nil {
		// 会话删除失败不应该影响密码重置，只记录错误
		fmt.Printf("Warning: failed to delete user sessions: %v\n", err)
	}

	return nil
}

// SyncPasswordVersionToCache 同步密码版本到缓存
func (s *PasswordService) SyncPasswordVersionToCache(ctx context.Context, userID uint) error {
	// 从数据库获取密码版本
	passwordV, err := s.userService.GetUserPasswordVersion(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get password version from database: %w", err)
	}

	// 存储到缓存
	err = s.sessionService.StorePasswordVersion(ctx, userID, passwordV, s.cacheExpiry)
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
