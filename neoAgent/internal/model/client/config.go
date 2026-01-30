/**
 * 配置相关模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent配置管理相关的数据模型，遵循"好品味"原则
 * @func: 定义Agent配置、插件配置和安全配置的数据结构
 */
package communication

import "time"

// ==================== 配置相关 ====================

// AgentConfig Agent配置
// 遵循Linus原则：配置结构层次清晰，避免嵌套过深
type AgentConfig struct {
	Version           string                 `json:"version"`            // 配置版本
	HeartbeatInterval time.Duration          `json:"heartbeat_interval"` // 心跳间隔
	TaskTimeout       time.Duration          `json:"task_timeout"`       // 任务超时时间
	MaxTasks          int                    `json:"max_tasks"`          // 最大任务数
	LogLevel          string                 `json:"log_level"`          // 日志级别
	Plugins           []PluginConfig         `json:"plugins"`            // 插件配置
	Security          SecurityConfig         `json:"security"`           // 安全配置
	Custom            map[string]interface{} `json:"custom"`             // 自定义配置
	UpdatedAt         time.Time              `json:"updated_at"`         // 更新时间
}

// PluginConfig 插件配置
// 遵循单一职责原则：专门负责插件的配置管理
type PluginConfig struct {
	Name    string                 `json:"name"`    // 插件名称
	Version string                 `json:"version"` // 插件版本
	Enabled bool                   `json:"enabled"` // 是否启用
	Config  map[string]interface{} `json:"config"`  // 插件配置
}

// SecurityConfig 安全配置
// 遵循"好品味"原则：安全配置独立管理，避免与业务配置混合
type SecurityConfig struct {
	TLSEnabled  bool          `json:"tls_enabled"`  // 是否启用TLS
	CertFile    string        `json:"cert_file"`    // 证书文件
	KeyFile     string        `json:"key_file"`     // 密钥文件
	CAFile      string        `json:"ca_file"`      // CA文件
	TokenExpiry time.Duration `json:"token_expiry"` // 令牌过期时间
	MaxRetries  int           `json:"max_retries"`  // 最大重试次数
}

// ConfigSyncRequest 配置同步请求
// 遵循Linus原则：请求结构简洁，包含必要的版本信息
type ConfigSyncRequest struct {
	AgentID        string    `json:"agent_id"`        // Agent ID
	CurrentVersion string    `json:"current_version"` // 当前配置版本
	Timestamp      time.Time `json:"timestamp"`       // 请求时间戳
}

// ConfigSyncResponse 配置同步响应
// 遵循"好品味"原则：响应包含完整的配置数据和重启标识
type ConfigSyncResponse struct {
	Success       bool                   `json:"success"`        // 同步是否成功
	ConfigVersion string                 `json:"config_version"` // 配置版本
	Configs       map[string]interface{} `json:"configs"`        // 配置数据
	NeedRestart   bool                   `json:"need_restart"`   // 是否需要重启
	Message       string                 `json:"message"`        // 响应消息
	Timestamp     time.Time              `json:"timestamp"`      // 响应时间戳
}