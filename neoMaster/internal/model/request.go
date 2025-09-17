/**
 * 模型:请求模型
 * @author: sun977
 * @date: 2025.08.29
 * @description: API请求数据模型，包含各种业务操作的请求结构体
 * @func: 各种Request结构体定义
 */
package model

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" validate:"required"` // 用户名，必填
	Password string `json:"password" validate:"required"` // 密码，必填
}

// RefreshTokenRequest 刷新令牌请求结构
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"` // 刷新令牌，必填
}

// RegisterRequest 用户注册请求结构
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"` // 用户名，必填，3-50字符
	Email    string `json:"email" validate:"required,email"`           // 邮箱地址，必填，必须符合邮箱格式
	Password string `json:"password" validate:"required,min=6"`        // 密码，必填，最少6字符
	Nickname string `json:"nickname"`                                  // 用户昵称，可选
	Phone    string `json:"phone"`                                     // 手机号码，可选
}

// CreateUserRequest 创建用户请求结构
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"` // 用户名，必填，3-50字符
	Email    string `json:"email" validate:"required,email"`           // 邮箱地址，必填，必须符合邮箱格式
	Password string `json:"password" validate:"required,min=6"`        // 密码，必填，最少6字符
	Nickname string `json:"nickname"`                                  // 用户昵称，可选
	Phone    string `json:"phone"`                                     // 手机号码，可选
	RoleIDs  []uint `json:"role_ids"`                                  // 角色ID列表，可选
	Remark   string `json:"remark"`                                    // 用户备注，可选
}

// UpdateUserRequest 更新用户请求结构 【userID 不许修改，其他字段可选】
type UpdateUserRequest struct {
	Username string      `json:"username" validate:"omitempty,min=3,max=50"` // 用户名，可选，3-50字符
	Nickname string      `json:"nickname"`                                   // 用户昵称，可选
	Email    string      `json:"email" validate:"omitempty,email"`           // 邮箱地址，可选，如果提供必须符合邮箱格式
	Phone    string      `json:"phone"`                                      // 手机号码，可选
	Password string      `json:"password" validate:"omitempty,min=6"`        // 密码，可选，如果提供最少6字符
	Status   *UserStatus `json:"status"`                                     // 用户状态，可选，使用指针以区分零值和未设置(激活|禁用)
	Avatar   string      `json:"avatar"`                                     // 用户头像，可选
	SocketID string      `json:"socket_id"`                                  // 套接字ID，可选
	RoleIDs  []uint      `json:"role_ids"`                                   // 角色ID列表，可选(角色修改单独处理)
	Remark   string      `json:"remark"`                                     // 用户备注，可选
}

// ChangePasswordRequest 修改密码请求结构
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`       // 旧密码，必填
	NewPassword string `json:"new_password" validate:"required,min=6"` // 新密码，必填，最少6字符
}

// CreateRoleRequest 创建角色请求结构
type CreateRoleRequest struct {
	Name          string `json:"name" validate:"required"` // 角色名称，必填
	DisplayName   string `json:"display_name"`             // 角色显示名称，可选
	Description   string `json:"description"`              // 角色描述，可选
	PermissionIDs []uint `json:"permission_ids"`           // 权限ID列表，可选
}

// UpdateRoleRequest 更新角色请求结构
type UpdateRoleRequest struct {
	Name          string      `json:"name"`           // 角色名称，可选
	DisplayName   string      `json:"display_name"`   // 角色显示名称，可选
	Description   string      `json:"description"`    // 角色描述，可选
	Status        *RoleStatus `json:"status"`         // 角色状态，可选，使用指针以区分零值和未设置
	PermissionIDs []uint      `json:"permission_ids"` // 权限ID列表，可选
}

// CreatePermissionRequest 创建权限请求结构
type CreatePermissionRequest struct {
	Name        string `json:"name" validate:"required"` // 权限名称，必填
	DisplayName string `json:"display_name"`             // 权限显示名称，可选
	Description string `json:"description"`              // 权限描述，可选
	Resource    string `json:"resource"`                 // 资源标识，可选
	Action      string `json:"action"`                   // 操作标识，可选
}

// UpdatePermissionRequest 更新权限请求结构
type UpdatePermissionRequest struct {
	DisplayName string `json:"display_name"` // 权限显示名称，可选
	Description string `json:"description"`  // 权限描述，可选
	Resource    string `json:"resource"`     // 资源标识，可选
	Action      string `json:"action"`       // 操作标识，可选
}
