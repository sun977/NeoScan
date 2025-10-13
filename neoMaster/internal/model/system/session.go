/**
 * 模型:会话模型
 * @author: sun977
 * @date: 2025.08.29
 * @description: 会话和令牌数据模型，包含用户会话信息和令牌管理
 * @func: SessionData、TokenData 结构体定义
 */
package system

import (
	"time"
)

// SessionData 会话数据结构
type SessionData struct {
	UserID      uint      `json:"user_id"`      // 用户ID
	Username    string    `json:"username"`     // 用户名
	Email       string    `json:"email"`        // 邮箱地址
	Roles       []string  `json:"roles"`        // 用户角色名称列表
	Permissions []string  `json:"permissions"`  // 用户权限名称列表
	LoginTime   time.Time `json:"login_time"`   // 登录时间
	LastActive  time.Time `json:"last_active"`  // 最后活跃时间
	ClientIP    string    `json:"client_ip"`    // 客户端IP地址
	UserAgent   string    `json:"user_agent"`   // 用户代理信息
}

// TokenData 令牌数据结构
type TokenData struct {
	AccessToken  string    `json:"access_token"`  // 访问令牌
	RefreshToken string    `json:"refresh_token"` // 刷新令牌
	ExpiresAt    time.Time `json:"expires_at"`    // 令牌过期时间
	CreatedAt    time.Time `json:"created_at"`    // 令牌创建时间
}

// IsExpired 检查令牌是否已过期
func (t *TokenData) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsActive 检查会话是否活跃（最后活跃时间在指定时间内）
func (s *SessionData) IsActive(timeout time.Duration) bool {
	return time.Since(s.LastActive) <= timeout
}

// UpdateLastActive 更新最后活跃时间
func (s *SessionData) UpdateLastActive() {
	s.LastActive = time.Now()
}

// HasRole 检查会话用户是否拥有指定角色
func (s *SessionData) HasRole(roleName string) bool {
	for _, role := range s.Roles {
		if role == roleName {
			return true
		}
	}
	return false
}

// HasPermission 检查会话用户是否拥有指定权限
func (s *SessionData) HasPermission(permissionName string) bool {
	for _, permission := range s.Permissions {
		if permission == permissionName {
			return true
		}
	}
	return false
}