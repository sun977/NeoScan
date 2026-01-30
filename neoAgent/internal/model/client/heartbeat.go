/**
 * 心跳相关模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent心跳机制相关的数据模型，遵循"好品味"原则
 * @func: 定义心跳请求和响应的数据结构
 */
package client

import "time"

// ==================== 心跳相关 ====================

// Heartbeat 心跳数据
// 遵循Linus原则：心跳数据结构简洁，包含必要的状态和指标信息
type Heartbeat struct {
	AgentID    string                 `json:"agent_id"`     // Agent ID
	Status     string                 `json:"status"`       // 当前状态
	TaskCount  int                    `json:"task_count"`   // 任务数量
	Metrics    *PerformanceMetrics    `json:"metrics"`      // 性能指标
	LastTaskID string                 `json:"last_task_id"` // 最后执行的任务ID
	Extra      map[string]interface{} `json:"extra"`        // 额外信息
	Timestamp  time.Time              `json:"timestamp"`    // 心跳时间戳
}

// HeartbeatResponse 心跳响应
// 遵循"好品味"原则：响应包含下次心跳时间和待执行命令，减少轮询
type HeartbeatResponse struct {
	Success       bool       `json:"success"`        // 心跳是否成功
	NextHeartbeat time.Time  `json:"next_heartbeat"` // 下次心跳时间
	Commands      []*Command `json:"commands"`       // 待执行命令
	ConfigUpdate  bool       `json:"config_update"`  // 是否有配置更新
	Message       string     `json:"message"`        // 响应消息
	Timestamp     time.Time  `json:"timestamp"`      // 响应时间戳
}