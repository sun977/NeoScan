/*
 * 权限仓库层:权限数据访问
 * @author: sun977
 * @date: 2025.09.11
 * @description: 单纯数据访问,不应该包含业务逻辑
 * @func:
 * 1.创建权限
 * 2.更新权限
 * 3.删除权限
 * 4.权限基础查询
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

// PermissionRepository 权限仓库结构体
// 负责处理权限相关的数据访问，不包含业务逻辑
type PermissionRepository struct {
	db *gorm.DB // 数据库连接
}

// NewPermissionRepository 创建权限仓库实例
// 注入数据库连接，专注于数据访问操作
func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// CreatePermission 创建权限（纯数据访问）
// 直接将权限数据插入数据库，不包含业务逻辑验证
func (r *PermissionRepository) CreatePermission(ctx context.Context, permission *model.Permission) error {
	result := r.db.WithContext(ctx).Create(permission)
	return result.Error
}

// GetPermissionByID 根据ID获取权限
func (r *PermissionRepository) GetPermissionByID(ctx context.Context, id uint) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogError(fmt.Errorf("permission not found"), "", id, "", "permission_get", "GET", map[string]interface{}{
				"operation": "get_permission_by_id",
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil
		}
		logger.LogError(err, "", id, "", "permission_get", "GET", map[string]interface{}{
			"operation": "get_permission_by_id",
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &permission, nil
}

// GetPermissionByName 根据名称获取权限
func (r *PermissionRepository) GetPermissionByName(ctx context.Context, name string) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&permission).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogError(fmt.Errorf("permission not found"), "", 0, "", "permission_get", "GET", map[string]interface{}{
				"operation": "get_permission_by_name",
				"name":      name,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "permission_get", "GET", map[string]interface{}{
			"operation": "get_permission_by_name",
			"name":      name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &permission, nil
}

// UpdatePermission 更新权限信息
func (r *PermissionRepository) UpdatePermission(ctx context.Context, permission *model.Permission) error {
	permission.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Save(permission).Error; err != nil {
		logger.LogError(err, "", uint(permission.ID), "", "permission_update", "PUT", map[string]interface{}{
			"operation": "update_permission",
			"name":      permission.Name,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// UpdatePermissionFields 使用 map 更新权限特定字段
func (r *PermissionRepository) UpdatePermissionFields(ctx context.Context, permissionID uint, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.Permission{}).Where("id = ?", permissionID).Updates(fields).Error
}

// DeletePermission 软删除权限
func (r *PermissionRepository) DeletePermission(ctx context.Context, permissionID uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Permission{}, permissionID)
	if result.Error != nil {
		logger.LogError(result.Error, "", uint(permissionID), "", "permission_delete", "DELETE", map[string]interface{}{
			"operation": "delete_permission",
			"timestamp": logger.NowFormatted(),
		})
		return result.Error
	}
	return nil
}

// GetPermissionList 获取权限列表
func (r *PermissionRepository) GetPermissionList(ctx context.Context, offset, limit int) ([]*model.Permission, int64, error) {
	var permissions []*model.Permission
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.Permission{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&permissions).Error; err != nil {
		return nil, 0, err
	}
	return permissions, total, nil
}

// PermissionExists 检查权限是否存在
func (r *PermissionRepository) PermissionExists(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Permission{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// BeginTx 开始事务
func (r *PermissionRepository) BeginTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Begin()
}

// DeleteRolePermissionsByPermissionID 删除与指定权限关联的角色关系（事务版）
func (r *PermissionRepository) DeleteRolePermissionsByPermissionID(ctx context.Context, tx *gorm.DB, permissionID uint) error {
	result := tx.WithContext(ctx).Where("permission_id = ?", permissionID).Delete(&model.RolePermission{})
	if result.Error != nil {
		logger.LogError(result.Error, "", permissionID, "", "delete_role_permissions_by_permission", "DELETE", map[string]interface{}{
			"operation":     "delete_role_permissions_by_permission_id",
			"permission_id": permissionID,
			"timestamp":     logger.NowFormatted(),
		})
		return result.Error
	}
	return nil
}

// GetPermissionWithRoles 获取权限及其关联角色
func (r *PermissionRepository) GetPermissionWithRoles(ctx context.Context, permissionID uint) (*model.Permission, error) {
	var permission model.Permission
	if err := r.db.WithContext(ctx).Preload("Roles").First(&permission, permissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &permission, nil
}

// GetPermissionRoles 获取权限关联的角色
func (r *PermissionRepository) GetPermissionRoles(ctx context.Context, permissionID uint) ([]*model.Role, error) {
	var permission model.Permission
	if err := r.db.WithContext(ctx).Preload("Roles").First(&permission, permissionID).Error; err != nil {
		return nil, err
	}
	roles := make([]*model.Role, len(permission.Roles))
	for i := range permission.Roles {
		roles[i] = &permission.Roles[i]
	}
	return roles, nil
}
