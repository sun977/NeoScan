/**
 * 模型:权限模型
 * @author: sun977
 * @date: 2025.08.29
 * @description: 权限数据模型，包含权限基本信息、资源操作定义和角色关联
 * @func: Permission 结构体及相关方法
 */
package model

import (
	"time"
)

// Permission 权限模型
type Permission struct {
	ID          uint             `json:"id" gorm:"primaryKey;autoIncrement"`                            // 权限唯一标识ID，主键自增
	Name        string           `json:"name" gorm:"uniqueIndex;not null;size:100" validate:"required"` // 权限名称，唯一索引，必填
	DisplayName string           `json:"display_name" gorm:"size:100;comment:权限显示名称"`                   // 权限显示名称，用于前端展示
	Description string           `json:"description" gorm:"size:255;comment:权限描述信息"`                    // 权限描述信息，最大255字符
	Resource    string           `json:"resource" gorm:"size:100;comment:资源标识"`                         // 资源标识，如user、role、system等
	Status      PermissionStatus `json:"status" gorm:"size:20;default:1;comment:状态1启用0禁用"`              // 状态，默认1启用，0禁用
	Action      string           `json:"action" gorm:"size:50;comment:操作标识"`                            // 操作标识，如create、read、update、delete等
	CreatedAt   time.Time        `json:"created_at"`                                                    // 创建时间，自动管理
	UpdatedAt   time.Time        `json:"updated_at"`                                                    // 更新时间，自动管理

	// 关联关系
	Roles []Role `json:"-" gorm:"many2many:role_permissions;"` // 拥有此权限的角色，多对多关系
}

// PermissionStatus 权限状态枚举
type PermissionStatus int

const (
	PermissionStatusDisabled PermissionStatus = 0 // 禁用状态
	PermissionStatusEnabled  PermissionStatus = 1 // 启用状态
)

// TableName 指定权限表名
func (Permission) TableName() string {
	return "permissions"
}

// GetFullName 获取权限的完整名称（资源:操作）
func (p *Permission) GetFullName() string {
	if p.Resource != "" && p.Action != "" {
		return p.Resource + ":" + p.Action
	}
	return p.Name
}

// IsSystemPermission 检查是否为系统级权限
func (p *Permission) IsSystemPermission() bool {
	return p.Resource == "system"
}
