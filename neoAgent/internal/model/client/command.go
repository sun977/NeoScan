/**
 * 命令相关模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: Master发送命令和Agent响应相关的数据模型，遵循"好品味"原则
 * @func: 定义命令、命令响应和命令状态的数据结构
 */
package client

import "time"

// ==================== 命令相关 ====================

// Command Master发送的命令
// 遵循Linus原则：命令结构清晰，包含执行所需的所有信息
type Command struct {
	ID        string                 `json:"id"`        // 命令ID
	Type      string                 `json:"type"`      // 命令类型
	Action    string                 `json:"action"`    // 具体动作
	Payload   map[string]interface{} `json:"payload"`   // 命令载荷
	Priority  int                    `json:"priority"`  // 优先级
	Timeout   time.Duration          `json:"timeout"`   // 超时时间
	Retry     int                    `json:"retry"`     // 重试次数
	Timestamp time.Time              `json:"timestamp"` // 命令时间戳
	ExpireAt  time.Time              `json:"expire_at"` // 过期时间
}

// CommandResponse 命令响应
// 遵循"好品味"原则：响应包含完整的执行结果和性能数据
type CommandResponse struct {
	CommandID string                 `json:"command_id"` // 命令ID
	AgentID   string                 `json:"agent_id"`   // Agent ID
	Success   bool                   `json:"success"`    // 执行是否成功
	Result    map[string]interface{} `json:"result"`     // 执行结果
	Error     string                 `json:"error"`      // 错误信息
	Duration  time.Duration          `json:"duration"`   // 执行耗时
	Timestamp time.Time              `json:"timestamp"`  // 响应时间戳
}

// CommandStatus 命令状态
// 遵循单一职责原则：专门负责命令执行状态的跟踪
type CommandStatus struct {
	CommandID string    `json:"command_id"` // 命令ID
	Status    string    `json:"status"`     // 状态: pending, running, completed, failed, timeout
	Progress  float64   `json:"progress"`   // 执行进度 (0-100)
	Message   string    `json:"message"`    // 状态消息
	StartTime time.Time `json:"start_time"` // 开始时间
	EndTime   time.Time `json:"end_time"`   // 结束时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}