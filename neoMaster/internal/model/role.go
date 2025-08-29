/**
 * 模型:角色模型
 * @author: sun977
 * @date: 2025.08.29
 * @description: 角色数据模型，包含角色基本信息、状态管理和权限关联
 * @func: Role 结构体及相关方法
 */
package model

import (
	"time"
)

// Role 角色模型
type Role struct {
	ID          uint       `json:"id" gorm:"primaryKey;autoIncrement"`                           // 角色唯一标识ID，主键自增
	Name        string     `json:"name" gorm:"uniqueIndex;not null;size:50" validate:"required"` // 角色名称，唯一索引，必填
	DisplayName string     `json:"display_name" gorm:"size:100"`                                 // 角色显示名称，用于前端展示
	Description string     `json:"description" gorm:"size:255"`                                  // 角色描述信息，最大255字符
	Status      RoleStatus `json:"status" gorm:"default:1;comment:角色状态:0-禁用,1-启用"`               // 角色状态，默认启用
	CreatedAt   time.Time  `json:"created_at"`                                                   // 创建时间，自动管理
	UpdatedAt   time.Time  `json:"updated_at"`                                                   // 更新时间，自动管理
	DeletedAt   *time.Time `json:"-" gorm:"index"`                                               // 软删除时间，不在JSON中返回

	// 关联关系
	Users       []User       `json:"-" gorm:"many2many:user_roles;"`                 // 拥有此角色的用户，多对多关系
	Permissions []Permission `json:"permissions" gorm:"many2many:role_permissions;"` // 角色拥有的权限，多对多关系
}

// RoleStatus 角色状态枚举
type RoleStatus int

const (
	RoleStatusDisabled RoleStatus = 0 // 禁用状态
	RoleStatusEnabled  RoleStatus = 1 // 启用状态
)

// RolePermission 角色权限关联表
type RolePermission struct {
	RoleID       uint      `json:"role_id" gorm:"primaryKey"`       // 角色ID，联合主键
	PermissionID uint      `json:"permission_id" gorm:"primaryKey"` // 权限ID，联合主键
	CreatedAt    time.Time `json:"created_at"`                      // 关联创建时间
}

// TableName 指定角色表名
func (Role) TableName() string {
	return "roles"
}

// TableName 指定角色权限关联表名
func (RolePermission) TableName() string {
	return "role_permissions"
}

// IsActive 检查角色是否处于活跃状态
func (r *Role) IsActive() bool {
	return r.Status == RoleStatusEnabled
}

// HasPermission 检查角色是否拥有指定权限
func (r *Role) HasPermission(permissionName string) bool {
	for _, permission := range r.Permissions {
		if permission.Name == permissionName {
			return true
		}
	}
	return false
}
