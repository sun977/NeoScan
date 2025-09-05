/**
 * 用户仓库层:用户数据访问
 * @author: sun977
 * @date: 2025.08.29
 * @description: 用户数据访问
 * @func:单纯数据访问,不应该包含业务逻辑
 */
package mysql

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// UserRepository 用户仓库结构体
// 负责处理用户相关的数据访问，不包含业务逻辑
type UserRepository struct {
	db *gorm.DB // 数据库连接
}

// NewUserRepository 创建用户仓库实例
// 注入数据库连接，专注于数据访问操作
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// CreateUser 创建用户（纯数据访问）
// 直接将用户数据插入数据库，不包含业务逻辑验证
func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	result := r.db.WithContext(ctx).Create(user)
	return result.Error
}

// CreateUserDirect 直接创建用户（仅用于内部调用，不包含业务逻辑验证）
// 主要用于测试或特殊场景，密码应该已经被哈希处理
func (r *UserRepository) CreateUserDirect(ctx context.Context, user *model.User) error {
	// 仅负责数据存储，不进行业务逻辑处理
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
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("user not found"), "", id, "", "user_get", "GET", map[string]interface{}{
				"operation": "get_user_by_id",
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", id, "", "user_get", "GET", map[string]interface{}{
			"operation": "get_user_by_id",
			"timestamp": logger.NowFormatted(),
		})
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
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("user not found"), "", 0, "", "user_get", "GET", map[string]interface{}{
				"operation": "get_user_by_username",
				"username":  username,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "user_get", "GET", map[string]interface{}{
			"operation": "get_user_by_username",
			"username":  username,
			"timestamp": logger.NowFormatted(),
		})
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
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("user not found"), "", 0, "", "user_get", "GET", map[string]interface{}{
				"operation": "get_user_by_email",
				"email":     email,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "user_get", "GET", map[string]interface{}{
			"operation": "get_user_by_email",
			"email":     email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(user).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(user.ID), "", "user_update", "PUT", map[string]interface{}{
			"operation": "update_user",
			"username":  user.Username,
			"email":     user.Email,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// UpdateUserFields 使用 map 更新用户特定字段
// 主要用于原子更新操作，如密码和版本号同时更新
func (r *UserRepository) UpdateUserFields(ctx context.Context, userID uint, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(fields).Error
}

// UpdateLastLogin 更新用户最后登录时间
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_login_at": now,
		"last_login_ip": "", // 这里可以传入IP参数，暂时设为空
		"updated_at":    now,
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
	result := r.db.WithContext(ctx).Delete(&model.User{}, userID)
	if result.Error != nil {
		// 记录数据库错误日志
		logger.LogError(result.Error, "", uint(userID), "", "user_delete", "DELETE", map[string]interface{}{
			"operation": "delete_user",
			"timestamp": logger.NowFormatted(),
		})
		return result.Error
	}
	// 删除操作具有幂等性，即使没有找到记录也不应该返回错误
	// 这符合数据访问层的设计原则
	return nil
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

// GetUserList 获取用户列表
func (r *UserRepository) GetUserList(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
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
