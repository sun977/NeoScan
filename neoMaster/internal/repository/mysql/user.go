/**
 * 数据访问层:用户数据访问层
 * @author: sun977
 * @date: 2025.08.29
 * @description: 用户数据访问层
 * @func:
 * 	1.创建用户
 * 	2.根据ID获取用户
 * 	3.根据用户名获取用户
 */
package mysql

import (
	"context"
	"time"

	"neomaster/internal/model"

	"gorm.io/gorm"
)

// UserRepository 用户数据访问层
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据访问层实例
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(user).Error
}

// GetUserByID 根据ID获取用户
func (r *UserRepository) GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdatePasswordWithVersion 更新用户密码并递增密码版本号
func (r *UserRepository) UpdatePasswordWithVersion(ctx context.Context, userID uint, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash": passwordHash,
		"password_v":    gorm.Expr("password_v + 1"),
		"updated_at":    time.Now(),
	}).Error
}

// UpdateLastLogin 更新用户最后登录时间
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_login":   time.Now(),
		"login_count":  gorm.Expr("login_count + 1"),
		"updated_at":   time.Now(),
	}).Error
}

// GetUserPasswordVersion 获取用户密码版本号
func (r *UserRepository) GetUserPasswordVersion(ctx context.Context, userID uint) (int64, error) {
	var passwordV int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Select("password_v").Scan(&passwordV).Error
	return passwordV, err
}

// IncrementPasswordVersion 递增用户密码版本号
func (r *UserRepository) IncrementPasswordVersion(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("password_v", gorm.Expr("password_v + 1")).Error
}

// DeleteUser 软删除用户
func (r *UserRepository) DeleteUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, userID).Error
}

// GetUserWithRolesAndPermissions 获取用户及其角色和权限
func (r *UserRepository) GetUserWithRolesAndPermissions(ctx context.Context, userID uint) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserRoles 获取用户角色
func (r *UserRepository) GetUserRoles(ctx context.Context, userID uint) ([]*model.Role, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	
	return user.Roles, nil
}

// GetUserPermissions 获取用户权限
func (r *UserRepository) GetUserPermissions(ctx context.Context, userID uint) ([]*model.Permission, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	
	permissionMap := make(map[uint]*model.Permission)
	for _, role := range user.Roles {
		for _, permission := range role.Permissions {
			permissionMap[permission.ID] = &permission
		}
	}
	
	permissions := make([]*model.Permission, 0, len(permissionMap))
	for _, permission := range permissionMap {
		permissions = append(permissions, permission)
	}
	return permissions, nil
}

// AssignRoleToUser 为用户分配角色
func (r *UserRepository) AssignRoleToUser(ctx context.Context, userID, roleID uint) error {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return err
	}
	
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return err
	}
	
	return r.db.WithContext(ctx).Model(&user).Association("Roles").Append(&role)
}

// RemoveRoleFromUser 移除用户角色
func (r *UserRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID uint) error {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return err
	}
	
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return err
	}
	
	return r.db.WithContext(ctx).Model(&user).Association("Roles").Delete(&role)
}

// ListUsers 获取用户列表
func (r *UserRepository) ListUsers(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64
	
	if err := r.db.WithContext(ctx).Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

// UserExists 检查用户是否存在
func (r *UserRepository) UserExists(ctx context.Context, username, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("username = ? OR email = ?", username, email).Count(&count).Error
	return count > 0, err
}
