/*
 * 角色仓库层:角色数据访问
 * @author: sun977
 * @date: 2025.09.11
 * @description: 单纯数据访问,不应该包含业务逻辑
 * @func:
 * 1.创建角色
 * 2.更新角色
 * 3.删除角色
 * 4.角色状态变更等
 */

//  基础CRUD操作:
//  	CreateRole - 创建角色
//  	GetRoleByID - 根据ID获取角色
//  	GetRoleByName - 根据角色名获取角色
//  	UpdateRole - 更新角色信息
//  	DeleteRole - 软删除角色
//  高级查询功能:
//  	GetRoleList - 分页获取角色列表
//  	GetRoleWithPermissions - 获取角色及其权限
//  	GetRolePermissions - 获取角色权限
//  	RoleExists - 检查角色是否存在
//  权限管理:
//  	AssignPermissionToRole - 为角色分配权限
//  	RemovePermissionFromRole - 移除角色权限
//  事务支持:
//  	BeginTx - 开始事务
//  	UpdateRoleWithTx - 事务更新角色
//  	DeleteRoleWithTx - 事务删除角色
//  	DeleteRolePermissionsByRoleID - 事务删除角色权限关联
//  字段更新:
//  	UpdateRoleFields - 使用map更新特定字段

package mysql

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// RoleRepository 角色仓库结构体
// 负责处理角色相关的数据访问，不包含业务逻辑
type RoleRepository struct {
	db *gorm.DB // 数据库连接
}

// NewRoleRepository 创建角色仓库实例
// 注入数据库连接，专注于数据访问操作
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{
		db: db,
	}
}

// CreateRole 创建角色（纯数据访问）
// 直接将角色数据插入数据库，不包含业务逻辑验证
func (r *RoleRepository) CreateRole(ctx context.Context, role *model.Role) error {
	result := r.db.WithContext(ctx).Create(role)
	return result.Error
}

// GetRoleByID 根据ID获取角色
func (r *RoleRepository) GetRoleByID(ctx context.Context, id uint) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("role not found"), "", id, "", "role_get", "GET", map[string]interface{}{
				"operation": "get_role_by_id",
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", id, "", "role_get", "GET", map[string]interface{}{
			"operation": "get_role_by_id",
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &role, nil
}

// GetRoleByName 根据角色名获取角色
func (r *RoleRepository) GetRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("role not found"), "", 0, "", "role_get", "GET", map[string]interface{}{
				"operation": "get_role_by_name",
				"name":      name,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "role_get", "GET", map[string]interface{}{
			"operation": "get_role_by_name",
			"name":      name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &role, nil
}

// UpdateRole 更新角色信息
func (r *RoleRepository) UpdateRole(ctx context.Context, role *model.Role) error {
	role.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(role).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(role.ID), "", "role_update", "PUT", map[string]interface{}{
			"operation": "update_role",
			"name":      role.Name,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// UpdateRoleFields 使用 map 更新角色特定字段
// 主要用于原子更新操作，如状态变更
func (r *RoleRepository) UpdateRoleFields(ctx context.Context, roleID uint, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Role{}).
		Where("id = ?", roleID).
		Updates(fields).Error
}

// DeleteRole 软删除角色
func (r *RoleRepository) DeleteRole(ctx context.Context, roleID uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Role{}, roleID)
	if result.Error != nil {
		// 记录数据库错误日志
		logger.LogError(result.Error, "", uint(roleID), "", "role_delete", "DELETE", map[string]interface{}{
			"operation": "delete_role",
			"timestamp": logger.NowFormatted(),
		})
		return result.Error
	}
	// 删除操作具有幂等性，即使没有找到记录也不应该返回错误
	// 这符合数据访问层的设计原则
	return nil
}

// GetRoleWithPermissions 获取角色及其权限
func (r *RoleRepository) GetRoleWithPermissions(ctx context.Context, roleID uint) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Preload("Permissions").First(&role, roleID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetRoleList 获取角色列表
func (r *RoleRepository) GetRoleList(ctx context.Context, offset, limit int) ([]*model.Role, int64, error) {
	var roles []*model.Role
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.Role{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&roles).Error
	return roles, total, err
}

// RoleExistsByName 检查角色是否存在
func (r *RoleRepository) RoleExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Role{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

// RoleExistsByID 根据ID判断角色是否存在
func (r *RoleRepository) RoleExistsByID(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Role{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// BeginTx 开始事务
func (r *RoleRepository) BeginTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Begin()
}

// DeleteRolePermissionsByRoleID 删除角色的所有权限关联（事务版本）
func (r *RoleRepository) DeleteRolePermissionsByRoleID(ctx context.Context, tx *gorm.DB, roleID uint) error {
	result := tx.WithContext(ctx).Where("role_id = ?", roleID).Delete(&model.RolePermission{})
	if result.Error != nil {
		logger.LogError(result.Error, "", roleID, "", "delete_role_permissions", "DELETE", map[string]interface{}{
			"operation": "delete_role_permissions_by_role_id",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return result.Error
	}
	return nil
}

// DeleteRoleWithTx 使用事务软删除角色
func (r *RoleRepository) DeleteRoleWithTx(ctx context.Context, tx *gorm.DB, roleID uint) error {
	result := tx.WithContext(ctx).Delete(&model.Role{}, roleID)
	if result.Error != nil {
		logger.LogError(result.Error, "", roleID, "", "delete_role_with_tx", "DELETE", map[string]interface{}{
			"operation": "delete_role_with_transaction",
			"role_id":   roleID,
			"timestamp": logger.NowFormatted(),
		})
		return result.Error
	}
	return nil
}

// UpdateRoleWithTx 使用事务更新角色信息
// @param ctx 上下文
// @param tx 事务对象
// @param role 角色对象
// @return 错误信息
func (r *RoleRepository) UpdateRoleWithTx(ctx context.Context, tx *gorm.DB, role *model.Role) error {
	role.UpdatedAt = time.Now()
	err := tx.WithContext(ctx).Save(role).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(role.ID), "", "role_update_with_tx", "PUT", map[string]interface{}{
			"operation": "update_role_with_transaction",
			"name":      role.Name,
			"role_id":   role.ID,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// AssignPermissionToRole 为角色分配权限
func (r *RoleRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID uint) error {
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return err
	}

	var permission model.Permission
	if err := r.db.WithContext(ctx).First(&permission, permissionID).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&role).Association("Permissions").Append(&permission)
}

// RemovePermissionFromRole 移除角色权限
func (r *RoleRepository) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uint) error {
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return err
	}

	var permission model.Permission
	if err := r.db.WithContext(ctx).First(&permission, permissionID).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&role).Association("Permissions").Delete(&permission)
}

// GetRolePermissions 获取角色权限
func (r *RoleRepository) GetRolePermissions(ctx context.Context, roleID uint) ([]*model.Permission, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Preload("Permissions").First(&role, roleID).Error
	if err != nil {
		return nil, err
	}

	// 转换 []model.Permission 为 []*model.Permission
	permissions := make([]*model.Permission, len(role.Permissions))
	for i := range role.Permissions {
		permissions[i] = &role.Permissions[i]
	}

	return permissions, nil
}
