/**
 * 模型:响应模型
 * @author: sun977
 * @date: 2025.08.29
 * @description: API响应数据模型，包含各种业务操作的响应结构体
 * @func: 各种Response结构体定义
 */
package model

import (
	"time"
)

// LoginResponse 登录响应结构
type LoginResponse struct {
	User         *User  `json:"user"`          // 用户信息
	AccessToken  string `json:"access_token"`  // 访问令牌
	RefreshToken string `json:"refresh_token"` // 刷新令牌
	ExpiresIn    int64  `json:"expires_in"`    // 令牌过期时间（秒）
}

// RefreshTokenResponse 刷新令牌响应结构
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`  // 新的访问令牌
	RefreshToken string `json:"refresh_token"` // 新的刷新令牌
	ExpiresIn    int64  `json:"expires_in"`    // 令牌过期时间（秒）
	TokenType    string `json:"token_type"`    // 令牌类型，通常为"Bearer"
}

// RegisterResponse 用户注册响应结构
type RegisterResponse struct {
	User *UserInfo `json:"user"` // 用户信息
	// AccessToken  string    `json:"access_token"`  // 访问令牌
	// RefreshToken string    `json:"refresh_token"` // 刷新令牌
	// ExpiresIn    int64     `json:"expires_in"`    // 令牌过期时间（秒）
	Message string `json:"message"` // 注册成功消息
}

// UserInfo 用户信息响应结构
type UserInfo struct {
	ID          uint       `json:"id"`            // 用户ID
	Username    string     `json:"username"`      // 用户名
	Email       string     `json:"email"`         // 邮箱地址
	Nickname    string     `json:"nickname"`      // 用户昵称
	Avatar      string     `json:"avatar"`        // 用户头像URL
	Phone       string     `json:"phone"`         // 手机号码
	Status      UserStatus `json:"status"`        // 用户状态
	LastLoginAt *time.Time `json:"last_login_at"` // 最后登录时间
	CreatedAt   time.Time  `json:"created_at"`    // 创建时间
	Roles       []string   `json:"roles"`         // 用户角色名称列表
	Permissions []string   `json:"permissions"`   // 用户权限名称列表
	Remark      string     `json:"remark"`        // 备注
}

// APIResponse 通用API响应结构
type APIResponse struct {
	Code    int               `json:"code,omitempty"`   // 响应状态码，可选
	Status  string            `json:"status"`           // 响应状态："success" 或 "error"
	Message string            `json:"message"`          // 响应消息
	Data    interface{}       `json:"data,omitempty"`   // 响应数据，可选
	Error   string            `json:"error,omitempty"`  // 错误信息，可选
	Errors  []ValidationError `json:"errors,omitempty"` // 验证错误列表，可选
}

// PaginationResponse 分页响应结构
type PaginationResponse struct {
	Total       int64       `json:"total"`        // 总记录数
	Page        int         `json:"page"`         // 当前页码
	PageSize    int         `json:"page_size"`    // 每页大小
	TotalPages  int         `json:"total_pages"`  // 总页数
	HasNext     bool        `json:"has_next"`     // 是否有下一页
	HasPrevious bool        `json:"has_previous"` // 是否有上一页
	Data        interface{} `json:"data"`         // 分页数据
}

// UserListResponse 用户列表响应结构
type UserListResponse struct {
	Users      []UserInfo          `json:"users"`                // 用户列表
	Pagination *PaginationResponse `json:"pagination,omitempty"` // 分页信息，可选
}

// RoleListResponse 角色列表响应结构
type RoleListResponse struct {
	Roles      []Role              `json:"roles"`                // 角色列表
	Pagination *PaginationResponse `json:"pagination,omitempty"` // 分页信息，可选
}

// PermissionListResponse 权限列表响应结构
type PermissionListResponse struct {
	Permissions []Permission        `json:"permissions"`          // 权限列表
	Pagination  *PaginationResponse `json:"pagination,omitempty"` // 分页信息，可选
}
