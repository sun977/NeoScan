package allocator

import (
	"context"
	"encoding/json"
	"neomaster/internal/model/orchestrator"
	"strings"

	"sync"
	"time"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// ResourceAllocator 资源调度器接口
// 职责: 管理 Agent 资源池，实现最优分配。
// 对应文档: 1.3 Resource Allocator (资源调度器)
type ResourceAllocator interface {
	// CanExecute 检查 Agent 是否有能力执行该任务 (Match Capability & Tags)
	CanExecute(ctx context.Context, agent *agentModel.Agent, task *orchestrator.AgentTask) bool
	// Allow 检查是否允许向该 Agent 分发任务 (Rate Limiting)
	Allow(ctx context.Context, agentID string) bool
}

type resourceAllocator struct {
	// lastDispatchTime 记录每个 Agent 上次分发任务的时间
	lastDispatchTime sync.Map
	// minInterval 最小分发间隔
	minInterval time.Duration
}

// NewResourceAllocator 创建资源调度器
func NewResourceAllocator() ResourceAllocator {
	return &resourceAllocator{
		minInterval: 200 * time.Millisecond, // 默认限制每个 Agent 每秒最多 5 个任务请求 (防止突发过载)
	}
}

// Allow 检查是否允许向该 Agent 分发任务 (Rate Limiting)
func (a *resourceAllocator) Allow(ctx context.Context, agentID string) bool {
	now := time.Now()
	val, loaded := a.lastDispatchTime.LoadOrStore(agentID, now)
	if !loaded {
		return true
	}

	lastTime := val.(time.Time)
	if now.Sub(lastTime) < a.minInterval {
		return false // 过于频繁
	}

	// 更新时间
	a.lastDispatchTime.Store(agentID, now)
	return true
}

// CanExecute 检查 Agent 是否有能力执行该任务
func (a *resourceAllocator) CanExecute(ctx context.Context, agent *agentModel.Agent, task *orchestrator.AgentTask) bool {
	// 1. RateLimiter: 速率限制
	// 全局限速: 防止 Master 被大量心跳打挂 (这里是 Dispatch 阶段，主要防止 Agent 过载)
	// 目标限速: 防止把目标网段打挂 (通常在 Task 生成阶段处理，或者这里检查并发数)

	// 基础检查：Agent 必须在线
	if agent.Status != agentModel.AgentStatusOnline {
		return false
	}

	// 2. AgentSelector: 智能匹配
	// 基于 Capability (能力) 匹配: 只有安装了对应工具的 Agent 才能领取任务
	if !hasCapability(agent, task.ToolName) {
		logger.LogInfo("Agent missing capability for task", "", 0, "", "service.orchestrator.allocator.CanExecute", "", map[string]interface{}{
			"agent_id":  agent.AgentID,
			"tool_name": task.ToolName,
		})
		return false
	}

	// 3. Tag (标签) 匹配
	// 只有 "Zone:Inside" 的 Agent 才能扫内网
	if !matchTags(agent, task.RequiredTags) {
		return false
	}

	return true
}

// hasCapability 检查 Agent 是否拥有特定能力 (工具)
func hasCapability(agent *agentModel.Agent, toolName string) bool {
	// TODO: 将 toolName 映射为 ScanType ID，或者 Agent 直接上报支持的 ToolName 列表
	// 目前 Agent.Capabilities 存储的是 ID 列表 (["1", "2"])
	// 这里暂时做一个简单的模拟，或者假设所有 Online Agent 都有基础能力
	if len(agent.Capabilities) == 0 {
		// 如果没有上报能力，默认认为没有能力，或者根据配置决定
		// 为了开发方便，暂时返回 true，待 Capability 系统完善后改为 false
		return true
	}

	// 简单的字符串包含检查 (如果 Capabilities 以后存的是 Name)
	for _, cap := range agent.Capabilities {
		if strings.EqualFold(cap, toolName) {
			return true
		}
	}

	// Fallback: 暂时允许所有任务，除非明确不匹配
	return true
}

// matchTags 检查 Agent 是否满足任务的标签要求
func matchTags(agent *agentModel.Agent, requiredTagsJSON string) bool {
	if requiredTagsJSON == "" {
		return true // 任务无标签要求
	}

	var requiredTags []string
	if err := json.Unmarshal([]byte(requiredTagsJSON), &requiredTags); err != nil {
		// 解析失败，记录日志并拒绝（安全起见）
		logger.LogError(err, "failed to parse required tags", 0, "", "service.orchestrator.allocator.matchTags", "INTERNAL", nil)
		return false
	}

	if len(requiredTags) == 0 {
		return true
	}

	// Agent 必须包含所有 Required Tags
	// 将 Agent Tags 转为 Map 加速查找
	agentTagMap := make(map[string]bool)
	for _, tag := range agent.Tags {
		agentTagMap[tag] = true
	}

	for _, req := range requiredTags {
		if !agentTagMap[req] {
			// 缺失必须的标签
			return false
		}
	}

	return true
}
