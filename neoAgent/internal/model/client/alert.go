/**
 * 告警相关模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent告警机制相关的数据模型，遵循"好品味"原则
 * @func: 定义告警信息的数据结构
 */
package client

import "time"

// ==================== 告警相关 ====================

// Alert 告警信息
// 遵循Linus原则：告警结构包含完整的告警信息和解决状态
type Alert struct {
	ID         string                 `json:"id"`          // 告警ID
	AgentID    string                 `json:"agent_id"`    // Agent ID
	Type       string                 `json:"type"`        // 告警类型
	Level      string                 `json:"level"`       // 告警级别: info, warning, error, critical
	Title      string                 `json:"title"`       // 告警标题
	Message    string                 `json:"message"`     // 告警消息
	Source     string                 `json:"source"`      // 告警源
	Tags       map[string]string      `json:"tags"`        // 告警标签
	Metrics    map[string]interface{} `json:"metrics"`     // 相关指标
	Resolved   bool                   `json:"resolved"`    // 是否已解决
	ResolvedAt time.Time              `json:"resolved_at"` // 解决时间
	CreatedAt  time.Time              `json:"created_at"`  // 创建时间
	UpdatedAt  time.Time              `json:"updated_at"`  // 更新时间
}