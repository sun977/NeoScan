/**
 * 模型:用户模型
 * @author: sun977
 * @date: 2025.08.29
 * @description: 用户数据模型，包含用户基本信息、状态管理和关联关系
 * @func: User 结构体及相关方法
 */
package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID          uint       `json:"id" gorm:"primaryKey;autoIncrement"`                    // 用户唯一标识ID，主键自增
	Username    string     `json:"username" gorm:"uniqueIndex;not null;size:50" validate:"required,min=3,max=50"` // 用户名，唯一索引，3-50字符
	Email       string     `json:"email" gorm:"uniqueIndex;not null;size:100" validate:"required,email"`        // 邮箱地址，唯一索引，必须符合邮箱格式
	Password    string     `json:"-" gorm:"not null;size:255"`                            // 用户密码，加密存储，不在JSON中返回
	PasswordV   int64      `json:"-" gorm:"default:1;comment:密码版本号,用于使旧token失效"`        // 密码版本控制，用于token失效机制
	Nickname    string     `json:"nickname" gorm:"size:50"`                               // 用户昵称，最大50字符
	Avatar      string     `json:"avatar" gorm:"size:255"`                                // 用户头像URL，最大255字符
	Phone       string     `json:"phone" gorm:"size:20"`                                  // 手机号码，最大20字符
	SocketId    string     `json:"socket_id" gorm:"size:100;comment:WebSocket连接ID"`       // WebSocket连接标识，用于实时通信功能
	Remark      string     `json:"remark" gorm:"size:500;comment:管理员备注"`                // 管理员对用户的备注说明，最大500字符
	Status      UserStatus `json:"status" gorm:"default:1;comment:用户状态:0-禁用,1-启用"`       // 用户状态，默认启用
	LastLoginAt *time.Time `json:"last_login_at" gorm:"comment:最后登录时间"`                 // 最后登录时间，可为空
	LastLoginIP string     `json:"last_login_ip" gorm:"size:45;comment:最后登录IP"`          // 最后登录IP地址，支持IPv6
	CreatedAt   time.Time  `json:"created_at"`                                            // 创建时间，自动管理
	UpdatedAt   time.Time  `json:"updated_at"`                                            // 更新时间，自动管理
	DeletedAt   *time.Time `json:"-" gorm:"index"`                                        // 软删除时间，不在JSON中返回

	// 关联关系
	Roles []*Role `json:"roles" gorm:"many2many:user_roles;"` // 用户角色，多对多关系
}

// UserStatus 用户状态枚举
type UserStatus int

const (
	UserStatusDisabled UserStatus = 0 // 禁用状态
	UserStatusEnabled  UserStatus = 1 // 启用状态
)

// UserRole 用户角色关联表
type UserRole struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey"` // 用户ID，联合主键
	RoleID    uint      `json:"role_id" gorm:"primaryKey"` // 角色ID，联合主键
	CreatedAt time.Time `json:"created_at"`                // 关联创建时间
}

// TableName 指定用户表名
// User 结构体的方法 - 指定用户表名
func (User) TableName() string {
	return "users"
}

// TableName 指定用户角色关联表名
// UserRole 结构体的方法 - 指定用户角色关联表名
func (UserRole) TableName() string {
	return "user_roles"
}

// HasRole 检查用户是否拥有指定角色
// User 结构体的方法 - 检查用户是否拥有指定角色
// 指针接受者  方法名   参数列表  返回值列表
func (u *User) HasRole(roleName string) bool {
	// 遍历用户拥有的角色，_ 表示忽略索引，role 表示角色对象，u.Roles 表示用户拥有的角色列表
	for _, role := range u.Roles {
		// Role 结构体中有 Name 字段，这里是对比输入和角色名称是否一致
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// HasPermission 检查用户是否拥有指定权限
// User 结构体的方法 - 检查用户是否拥有指定权限
func (u *User) HasPermission(permissionName string) bool {
	for _, role := range u.Roles {
		for _, permission := range role.Permissions {
			if permission.Name == permissionName {
				return true
			}
		}
	}
	return false
}

// IsActive 检查用户是否处于活跃状态
// User 结构体的方法 - 检查用户是否处于活跃状态
func (u *User) IsActive() bool {
	return u.Status == UserStatusEnabled
}
