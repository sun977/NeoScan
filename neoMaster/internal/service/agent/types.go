/**
 * Agent服务层类型定义
 * @author: Sun977
 * @date: 2025.01.20
 * @description: Agent服务层的请求和响应类型定义
 * @func: 统一的数据结构定义，消除特殊情况
 */
package agent

import (
	"neomaster/internal/model/agent"
	"time"
)

// RegisterAgentRequest Agent注册请求
type RegisterAgentRequest struct {
	Name        string              `json:"name" binding:"required"`      // Agent名称
	Host        string              `json:"host" binding:"required"`      // Agent主机地址
	Port        int                 `json:"port" binding:"required"`      // Agent端口
	ScanType    agent.AgentScanType `json:"scan_type" binding:"required"` // 扫描类型
	Version     string              `json:"version"`                      // Agent版本
	Tags        []string            `json:"tags"`                         // Agent标签
	Description string              `json:"description"`                  // Agent描述
	Config      RegisterAgentConfig `json:"config"`                       // Agent配置
}

// RegisterAgentConfig Agent注册时的配置
type RegisterAgentConfig struct {
	MaxConcurrentTasks int      `json:"max_concurrent_tasks"` // 最大并发任务数
	HeartbeatInterval  int      `json:"heartbeat_interval"`   // 心跳间隔（秒）
	TaskTimeout        int      `json:"task_timeout"`         // 任务超时时间（秒）
	EnabledTools       []string `json:"enabled_tools"`        // 启用的工具列表
}

// RegisterAgentResponse Agent注册响应
type RegisterAgentResponse struct {
	ID      uint              `json:"id"`      // Agent ID
	Name    string            `json:"name"`    // Agent名称
	Status  agent.AgentStatus `json:"status"`  // Agent状态
	Message string            `json:"message"` // 响应消息
}

// AgentInfo Agent信息
type AgentInfo struct {
	ID            uint                `json:"id"`             // Agent ID
	Name          string              `json:"name"`           // Agent名称
	Host          string              `json:"host"`           // Agent主机地址
	Port          int                 `json:"port"`           // Agent端口
	Status        agent.AgentStatus   `json:"status"`         // Agent状态
	ScanType      agent.AgentScanType `json:"scan_type"`      // 扫描类型
	Version       string              `json:"version"`        // Agent版本
	Tags          []string            `json:"tags"`           // Agent标签
	Description   string              `json:"description"`    // Agent描述
	Config        agent.AgentConfig   `json:"config"`         // Agent配置
	Metrics       agent.AgentMetrics  `json:"metrics"`        // Agent性能指标
	LastHeartbeat time.Time           `json:"last_heartbeat"` // 最后心跳时间
	CreatedAt     time.Time           `json:"created_at"`     // 创建时间
	UpdatedAt     time.Time           `json:"updated_at"`     // 更新时间
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	AgentID uint                `json:"agent_id" binding:"required"` // Agent ID
	Metrics *agent.AgentMetrics `json:"metrics"`                     // 性能指标（可选）
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Status  string `json:"status"`  // 响应状态
	Message string `json:"message"` // 响应消息
}

// GetAgentListRequest 获取Agent列表请求
type GetAgentListRequest struct {
	Page     int                 `json:"page" form:"page"`           // 页码
	PageSize int                 `json:"page_size" form:"page_size"` // 每页大小
	Status   agent.AgentStatus   `json:"status" form:"status"`       // 状态过滤（可选）
	ScanType agent.AgentScanType `json:"scan_type" form:"scan_type"` // 扫描类型过滤（可选）
}

// GetAgentListResponse 获取Agent列表响应
type GetAgentListResponse struct {
	Agents   []*AgentInfo `json:"agents"`    // Agent列表
	Total    int64        `json:"total"`     // 总数
	Page     int          `json:"page"`      // 当前页码
	PageSize int          `json:"page_size"` // 每页大小
}

// UpdateAgentStatusRequest 更新Agent状态请求
type UpdateAgentStatusRequest struct {
	Status agent.AgentStatus `json:"status" binding:"required"` // 新状态
}

// UpdateAgentStatusResponse 更新Agent状态响应
type UpdateAgentStatusResponse struct {
	Status  string `json:"status"`  // 响应状态
	Message string `json:"message"` // 响应消息
}

// DeleteAgentResponse 删除Agent响应
type DeleteAgentResponse struct {
	Status  string `json:"status"`  // 响应状态
	Message string `json:"message"` // 响应消息
}
