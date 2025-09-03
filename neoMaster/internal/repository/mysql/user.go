/**
 * 用户仓库层:用户数据访问和业务逻辑层
 * @author: sun977
 * @date: 2025.08.29
 * @description: 用户数据访问和业务逻辑处理
 * @func:
 * 	1.创建用户（包含密码哈希等业务逻辑）
 * 	2.更新用户
 * 	3.删除用户
 * 	4.获取用户信息
 * 	5.用户认证相关操作
 */
package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// UserRepository 用户仓库结构体
// 负责处理用户相关的数据访问和业务逻辑，包括密码哈希等安全操作
type UserRepository struct {
	db              *gorm.DB              // 数据库连接
	passwordManager *auth.PasswordManager // 密码管理器
}

// NewUserRepository 创建用户仓库实例
// 注入数据库连接和密码管理器，支持完整的用户业务逻辑处理
func NewUserRepository(db *gorm.DB, passwordManager *auth.PasswordManager) *UserRepository {
	return &UserRepository{
		db:              db,
		passwordManager: passwordManager,
	}
}

// SetPasswordManager 设置密码管理器（用于测试或动态配置）
func (r *UserRepository) SetPasswordManager(passwordManager *auth.PasswordManager) {
	r.passwordManager = passwordManager
}

// CreateUser 创建用户（包含业务逻辑）
// 处理用户创建的完整流程，包括参数验证、重复检查、密码哈希等
func (r *UserRepository) CreateUser(ctx context.Context, req *model.CreateUserRequest) (*model.User, error) {
	// 参数验证
	if req == nil {
		logger.LogError(errors.New("request is nil"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user",
			"error":     "request is nil",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("创建用户请求不能为空")
	}

	if req.Username == "" {
		logger.LogError(errors.New("username is empty"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user",
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("用户名不能为空")
	}

	if req.Email == "" {
		logger.LogError(errors.New("email is empty"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user",
			"username":  req.Username,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("邮箱不能为空")
	}

	if req.Password == "" {
		logger.LogError(errors.New("password is empty"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("密码不能为空")
	}

	// 检查用户名是否已存在
	existingUser, err := r.GetUserByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		logger.LogError(errors.New("username already exists"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation":        "create_user",
			"username":         req.Username,
			"email":            req.Email,
			"existing_user_id": existingUser.ID,
			"timestamp":        logger.NowFormatted(),
		})
		return nil, errors.New("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingUser, err = r.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		logger.LogError(errors.New("email already exists"), "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation":        "create_user",
			"username":         req.Username,
			"email":            req.Email,
			"existing_user_id": existingUser.ID,
			"timestamp":        logger.NowFormatted(),
		})
		return nil, errors.New("邮箱已存在")
	}

	// 哈希密码（业务逻辑处理）
	hashedPassword, err := r.passwordManager.HashPassword(req.Password)
	if err != nil {
		logger.LogError(err, "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "hash_password",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 创建用户模型
	user := &model.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword, // 使用哈希后的密码
		Status:    model.UserStatusEnabled,
		PasswordV: 1, // 设置密码版本
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 存储到数据库
	err = r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "user_create", "POST", map[string]interface{}{
			"operation": "create_user_db",
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 记录成功创建用户的业务日志
	logger.LogBusinessOperation("create_user", user.ID, user.Username, "", "", "success", "User created successfully", map[string]interface{}{
		"email":            user.Email,
		"status":           user.Status,
		"password_version": user.PasswordV,
		"timestamp":        logger.NowFormatted(),
	})

	return user, nil
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
			logger.LogError(fmt.Errorf("user not found"), "", id, "", "user_get", "GET", map[string]interface{}{
				"operation": "get_user_by_id",
				"timestamp": logger.NowFormatted(),
			})
			return nil, fmt.Errorf("用户不存在")
		}
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
			logger.LogError(fmt.Errorf("user not found"), "", 0, "", "user_get", "GET", map[string]interface{}{
				"operation": "get_user_by_username",
				"username":  username,
				"timestamp": logger.NowFormatted(),
			})
			return nil, fmt.Errorf("用户不存在")
		}
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
			logger.LogError(fmt.Errorf("user not found"), "", 0, "", "user_get", "GET", map[string]interface{}{
				"operation": "get_user_by_email",
				"email":     email,
				"timestamp": logger.NowFormatted(),
			})
			return nil, fmt.Errorf("用户不存在")
		}
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
		logger.LogError(err, "", uint(user.ID), "", "user_update", "PUT", map[string]interface{}{
			"operation": "update_user",
			"username":  user.Username,
			"email":     user.Email,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}

	// 记录成功更新用户的业务日志
	logger.LogBusinessOperation("update_user", uint(user.ID), user.Username, "", "", "success", "用户更新成功", map[string]interface{}{
		"email":     user.Email,
		"status":    user.Status,
		"timestamp": logger.NowFormatted(),
	})

	return nil
}

// UpdatePasswordWithVersion 更新用户密码并递增密码版本号
func (r *UserRepository) UpdatePasswordWithVersion(ctx context.Context, userID uint, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password":   passwordHash,
		"password_v": gorm.Expr("password_v + 1"),
		"updated_at": time.Now(),
	}).Error
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
		logger.LogError(result.Error, "", uint(userID), "", "user_delete", "DELETE", map[string]interface{}{
			"operation": "delete_user",
			"timestamp": logger.NowFormatted(),
		})
		return result.Error
	}
	if result.RowsAffected == 0 {
		logger.LogError(fmt.Errorf("user not found"), "", uint(userID), "", "user_delete", "DELETE", map[string]interface{}{
			"operation": "delete_user",
			"timestamp": logger.NowFormatted(),
		})
		return gorm.ErrRecordNotFound
	}

	// 记录成功删除用户的业务日志
	logger.LogBusinessOperation("delete_user", uint(userID), "", "", "", "success", "用户删除成功", map[string]interface{}{
		"rows_affected": result.RowsAffected,
		"timestamp":     logger.NowFormatted(),
	})

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

// ===== 业务逻辑方法（原UserService方法） =====

// UpdateUserWithBusinessLogic 更新用户信息（包含业务逻辑）
// 处理用户更新的完整流程，包括参数验证、重复检查、密码哈希等
func (r *UserRepository) UpdateUserWithBusinessLogic(ctx context.Context, userID uint, req *model.UpdateUserRequest) (*model.User, error) {
	// 参数验证
	if userID == 0 {
		return nil, errors.New("用户ID不能为0")
	}

	if req == nil {
		return nil, errors.New("更新用户请求不能为空")
	}

	// 获取现有用户
	user, err := r.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 更新字段
	if req.Email != "" && req.Email != user.Email {
		// 检查新邮箱是否已存在
		var existingUser *model.User
		existingUser, err = r.GetUserByEmail(ctx, req.Email)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			return nil, errors.New("邮箱已存在")
		}
		user.Email = req.Email
	}

	if req.Status != nil {
		user.Status = *req.Status
	}

	// 如果需要更新密码
	if req.Password != "" {
		var hashedPassword string
		hashedPassword, err = r.passwordManager.HashPassword(req.Password)
		if err != nil {
			return nil, fmt.Errorf("密码哈希失败: %w", err)
		}
		user.Password = hashedPassword
		user.PasswordV++ // 增加密码版本
	}

	user.UpdatedAt = time.Now()

	// 更新到数据库
	err = r.UpdateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	return user, nil
}

// DeleteUserWithBusinessLogic 删除用户（包含业务逻辑）
// 处理用户删除的完整流程，包括参数验证、存在性检查等
func (r *UserRepository) DeleteUserWithBusinessLogic(ctx context.Context, userID uint) error {
	if userID == 0 {
		return errors.New("用户ID不能为0")
	}

	// 检查用户是否存在
	user, err := r.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		return errors.New("用户不存在")
	}

	// 调用数据层删除用户
	return r.DeleteUser(ctx, userID)
}

// GetUserByIDWithBusinessLogic 根据ID获取用户（包含业务逻辑验证）
func (r *UserRepository) GetUserByIDWithBusinessLogic(ctx context.Context, userID uint) (*model.User, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为0")
	}

	return r.GetUserByID(ctx, userID)
}

// GetUserByUsernameWithBusinessLogic 根据用户名获取用户（包含业务逻辑验证）
func (r *UserRepository) GetUserByUsernameWithBusinessLogic(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, errors.New("用户名不能为空")
	}

	return r.GetUserByUsername(ctx, username)
}

// GetUserByEmailWithBusinessLogic 根据邮箱获取用户（包含业务逻辑验证）
func (r *UserRepository) GetUserByEmailWithBusinessLogic(ctx context.Context, email string) (*model.User, error) {
	if email == "" {
		return nil, errors.New("邮箱不能为空")
	}

	return r.GetUserByEmail(ctx, email)
}

// ListUsersWithBusinessLogic 获取用户列表（包含业务逻辑验证）
func (r *UserRepository) ListUsersWithBusinessLogic(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	if offset < 0 {
		offset = 0
	}

	if limit <= 0 || limit > 100 {
		limit = 20 // 默认每页20条
	}

	return r.ListUsers(ctx, offset, limit)
}

// ChangePassword 修改密码（包含完整的业务逻辑）
// 处理密码修改的完整流程，包括原密码验证、新密码哈希、版本更新等
func (r *UserRepository) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	if userID == 0 {
		logger.LogError(errors.New("user ID is zero"), "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户ID不能为0")
	}

	if oldPassword == "" {
		logger.LogError(errors.New("old password is empty"), "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("原密码不能为空")
	}

	if newPassword == "" {
		logger.LogError(errors.New("new password is empty"), "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("新密码不能为空")
	}

	// 获取用户信息
	user, err := r.GetUserByID(ctx, userID)
	if err != nil {
		logger.LogError(err, "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取用户失败: %w", err)
	}

	if user == nil {
		logger.LogError(errors.New("user is nil"), "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("用户不存在")
	}

	// 验证原密码
	valid, err := r.passwordManager.VerifyPassword(oldPassword, user.Password)
	if err != nil {
		logger.LogError(err, "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("密码验证失败: %w", err)
	}

	if !valid {
		logger.LogError(errors.New("old password is incorrect"), "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("原密码错误")
	}

	// 哈希新密码
	hashedPassword, err := r.passwordManager.HashPassword(newPassword)
	if err != nil {
		logger.LogError(err, "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("新密码哈希失败: %w", err)
	}

	// 更新密码和版本
	user.Password = hashedPassword
	user.PasswordV++
	user.UpdatedAt = time.Now()

	// 调用数据层更新
	err = r.UpdateUser(ctx, user)
	if err != nil {
		logger.LogError(err, "", uint(userID), "", "password_change", "PUT", map[string]interface{}{
			"operation": "change_password",
			"username":  user.Username,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("更新密码失败: %w", err)
	}

	// 记录成功修改密码的业务日志
	logger.LogBusinessOperation("change_password", uint(userID), user.Username, "", "", "success", "密码修改成功", map[string]interface{}{
		"old_password_version": user.PasswordV - 1,
		"new_password_version": user.PasswordV,
		"timestamp":            logger.NowFormatted(),
	})

	return nil
}
