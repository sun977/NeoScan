/**
 * 测试用Mock会话服务
 * @author: Linus Torvalds (AI Assistant)
 * @date: 2025.01.14
 * @description: 为测试提供简化的会话服务实现，避免复杂的依赖关系
 * @func: 提供基本的登录功能用于测试
 */
package test

import (
	"context"
	"errors"
	"fmt"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/repository/mysql"
)

// MockSessionService 模拟会话服务，用于测试
type MockSessionService struct {
	userRepo *mysql.UserRepository
	roleRepo *mysql.RoleRepository
}

// NewMockSessionService 创建模拟会话服务
func NewMockSessionService(userRepo *mysql.UserRepository, roleRepo *mysql.RoleRepository) *MockSessionService {
	return &MockSessionService{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

// Login 模拟登录
func (s *MockSessionService) Login(ctx context.Context, req *system.LoginRequest, clientIP, userAgent string) (*system.LoginResponse, error) {
	// 获取用户
	user, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	// 验证密码
	isValid, err := auth.VerifyPasswordWithDefaultConfig(req.Password, user.Password)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New("invalid password")
	}

	// 生成token
	token, err := utils.GenerateUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &system.LoginResponse{
		User:         user,
		AccessToken:  token,
		RefreshToken: token + "_refresh",
		ExpiresIn:    3600,
	}, nil
}

// Logout 模拟登出
func (s *MockSessionService) Logout(ctx context.Context, token string) error {
	// 简单返回成功
	return nil
}

// ValidateToken 模拟token验证
func (s *MockSessionService) ValidateToken(ctx context.Context, token string) (*system.User, error) {
	// 简单返回一个测试用户
	return &system.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Status:   system.UserStatusEnabled,
	}, nil
}