package auth

import (
	"context"
	"errors"
	"fmt"

	"neomaster/internal/model"
)

// RBACService 基于角色的访问控制服务
type RBACService struct {
	userService *UserService // 用户服务，提供用户相关的业务逻辑操作
}

// NewRBACService 创建RBAC服务实例
// 参数:
//   - userService: 用户服务实例，提供用户相关的业务逻辑功能
// 返回: RBACService指针，包含所有RBAC相关的业务方法
func NewRBACService(userService *UserService) *RBACService {
	return &RBACService{
		userService: userService, // 注入用户服务依赖
	}
}

// CheckPermission 检查用户是否具有特定权限
func (s *RBACService) CheckPermission(ctx context.Context, userID uint, resource, action string) (bool, error) {
	if userID == 0 {
		return false, errors.New("invalid user ID")
	}

	if resource == "" || action == "" {
		return false, errors.New("resource and action cannot be empty")
	}

	// 获取用户权限
	permissions, err := s.userService.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// 检查是否有匹配的权限
	for _, perm := range permissions {
		if s.matchPermission(perm, resource, action) {
			return true, nil
		}
	}

	return false, nil
}

// CheckRole 检查用户是否具有特定角色
func (s *RBACService) CheckRole(ctx context.Context, userID uint, roleName string) (bool, error) {
	if userID == 0 {
		return false, errors.New("invalid user ID")
	}

	if roleName == "" {
		return false, errors.New("role name cannot be empty")
	}

	// 获取用户角色
	roles, err := s.userService.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	// 检查是否有匹配的角色
	for _, role := range roles {
		if role.Name == roleName {
			return true, nil
		}
	}

	return false, nil
}

// CheckAnyRole 检查用户是否具有任意一个指定角色
func (s *RBACService) CheckAnyRole(ctx context.Context, userID uint, roleNames []string) (bool, error) {
	if userID == 0 {
		return false, errors.New("invalid user ID")
	}

	if len(roleNames) == 0 {
		return false, errors.New("role names cannot be empty")
	}

	// 获取用户角色
	userRoles, err := s.userService.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	// 创建角色名映射以提高查找效率
	roleMap := make(map[string]bool)
	for _, roleName := range roleNames {
		roleMap[roleName] = true
	}

	// 检查用户是否有任意一个指定角色
	for _, role := range userRoles {
		if roleMap[role.Name] {
			return true, nil
		}
	}

	return false, nil
}

// CheckAllRoles 检查用户是否具有所有指定角色
func (s *RBACService) CheckAllRoles(ctx context.Context, userID uint, roleNames []string) (bool, error) {
	if userID == 0 {
		return false, errors.New("invalid user ID")
	}

	if len(roleNames) == 0 {
		return true, nil // 空列表认为满足条件
	}

	// 获取用户角色
	userRoles, err := s.userService.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	// 创建用户角色映射
	userRoleMap := make(map[string]bool)
	for _, role := range userRoles {
		userRoleMap[role.Name] = true
	}

	// 检查是否拥有所有指定角色
	for _, roleName := range roleNames {
		if !userRoleMap[roleName] {
			return false, nil
		}
	}

	return true, nil
}

// GetUserPermissions 获取用户的所有权限
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uint) ([]*model.Permission, error) {
	if userID == 0 {
		return nil, errors.New("invalid user ID")
	}

	return s.userService.GetUserPermissions(ctx, userID)
}

// GetUserRoles 获取用户的所有角色
func (s *RBACService) GetUserRoles(ctx context.Context, userID uint) ([]*model.Role, error) {
	if userID == 0 {
		return nil, errors.New("invalid user ID")
	}

	return s.userService.GetUserRoles(ctx, userID)
}

// AssignRoleToUser 为用户分配角色
func (s *RBACService) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	if userID == 0 || roleID == 0 {
		return errors.New("invalid user ID or role ID")
	}

	return s.userService.AssignRoleToUser(ctx, userID, roleID)
}

// RemoveRoleFromUser 移除用户角色
func (s *RBACService) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	if userID == 0 || roleID == 0 {
		return errors.New("invalid user ID or role ID")
	}

	return s.userService.RemoveRoleFromUser(ctx, userID, roleID)
}

// IsUserActive 检查用户是否处于活跃状态
func (s *RBACService) IsUserActive(ctx context.Context, userID uint) (bool, error) {
	if userID == 0 {
		return false, errors.New("invalid user ID")
	}

	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return false, errors.New("user not found")
	}

	return user.IsActive(), nil
}

// ValidateResourceAccess 验证用户对资源的访问权限
func (s *RBACService) ValidateResourceAccess(ctx context.Context, userID uint, resource, action string) error {
	// 检查用户是否活跃
	isActive, err := s.IsUserActive(ctx, userID)
	if err != nil {
		return err
	}

	if !isActive {
		return errors.New("user is not active")
	}

	// 检查权限
	hasPermission, err := s.CheckPermission(ctx, userID, resource, action)
	if err != nil {
		return err
	}

	if !hasPermission {
		return fmt.Errorf("access denied: user does not have permission for %s:%s", resource, action)
	}

	return nil
}

// matchPermission 匹配权限
func (s *RBACService) matchPermission(permission *model.Permission, resource, action string) bool {
	// 精确匹配
	if permission.Resource == resource && permission.Action == action {
		return true
	}

	// 通配符匹配
	if permission.Resource == "*" && permission.Action == "*" {
		return true
	}

	if permission.Resource == resource && permission.Action == "*" {
		return true
	}

	if permission.Resource == "*" && permission.Action == action {
		return true
	}

	return false
}

// GetPermissionString 获取权限字符串表示
func (s *RBACService) GetPermissionString(permission *model.Permission) string {
	return fmt.Sprintf("%s:%s", permission.Resource, permission.Action)
}

// ParsePermissionString 解析权限字符串
func (s *RBACService) ParsePermissionString(permStr string) (resource, action string, err error) {
	parts := make([]string, 0, 2)
	for i, part := range []rune(permStr) {
		if part == ':' {
			parts = append(parts, permStr[:i])
			parts = append(parts, permStr[i+1:])
			break
		}
	}

	if len(parts) != 2 {
		return "", "", errors.New("invalid permission string format, expected 'resource:action'")
	}

	return parts[0], parts[1], nil
}

// HasSuperAdminRole 检查用户是否具有超级管理员角色
func (s *RBACService) HasSuperAdminRole(ctx context.Context, userID uint) (bool, error) {
	return s.CheckRole(ctx, userID, "super_admin")
}

// HasAdminRole 检查用户是否具有管理员角色
func (s *RBACService) HasAdminRole(ctx context.Context, userID uint) (bool, error) {
	return s.CheckAnyRole(ctx, userID, []string{"super_admin", "admin"})
}
