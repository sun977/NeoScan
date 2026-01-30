/**
 * 认证相关模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端认证和注册相关的数据模型，遵循"好品味"原则
 * @func: 定义认证、注册相关的请求和响应结构
 */
package client

import "time"

// ==================== 认证相关 ====================

// AuthData 认证数据
// 遵循Linus原则：认证数据结构简洁，字段职责明确
type AuthData struct {
	AgentID   string            `json:"agent_id"`  // Agent ID
	Token     string            `json:"token"`     // 认证令牌
	Signature string            `json:"signature"` // 签名
	Timestamp time.Time         `json:"timestamp"` // 时间戳
	Extra     map[string]string `json:"extra"`     // 额外认证信息
}

// AuthResponse 认证响应
// 遵循"好品味"原则：响应结构包含所有必要信息，避免多次请求
type AuthResponse struct {
	Success      bool      `json:"success"`       // 认证是否成功
	AccessToken  string    `json:"access_token"`  // 访问令牌
	RefreshToken string    `json:"refresh_token"` // 刷新令牌
	ExpiresAt    time.Time `json:"expires_at"`    // 过期时间
	Permissions  []string  `json:"permissions"`   // 权限列表
	Message      string    `json:"message"`       // 响应消息
	Timestamp    time.Time `json:"timestamp"`     // 响应时间戳
}

// RegisterResponse 注册响应
// 遵循单一职责原则：专门处理Agent注册后的响应数据
type RegisterResponse struct {
	Success   bool        `json:"success"`   // 注册是否成功
	AgentID   string      `json:"agent_id"`  // 分配的Agent ID
	Token     string      `json:"token"`     // 认证令牌
	Config    AgentConfig `json:"config"`    // 初始配置
	Message   string      `json:"message"`   // 响应消息
	Timestamp time.Time   `json:"timestamp"` // 响应时间戳
}