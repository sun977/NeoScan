// ResourceAllocator 资源分配器接口
// 职责: 管理 Agent 资源池，实现最优分配。
// 对应文档: 1.3 Resource Allocator (资源调度器)
package allocator

import (
	"context"
	"encoding/json"
	"neomaster/internal/model/orchestrator"

	"sync"
	"time"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/matcher"
	"neomaster/internal/service/tag_system"
)

// ResourceAllocator 资源调度器接口
// 职责: 管理 Agent 资源池，实现最优分配。
// 对应文档: 1.3 Resource Allocator (资源调度器)
type ResourceAllocator interface {
	// CanExecute 检查 Agent 是否有能力执行该任务 (Match TaskSupport/Feature & Tags)
	CanExecute(ctx context.Context, agent *agentModel.Agent, task *orchestrator.AgentTask) bool
	// Allow 检查是否允许向该 Agent 分发任务 (Rate Limiting)
	Allow(ctx context.Context, agentID string) bool
}

type resourceAllocator struct {
	// lastDispatchTime 记录每个 Agent 上次分发任务的时间
	lastDispatchTime sync.Map
	// minInterval 最小分发间隔
	minInterval time.Duration
	// tagService 标签服务
	tagService tag_system.TagService
}

// NewResourceAllocator 创建资源调度器
func NewResourceAllocator(tagService tag_system.TagService) ResourceAllocator {
	return &resourceAllocator{
		minInterval: 200 * time.Millisecond, // 默认限制每个 Agent 每秒最多 5 个任务请求 (防止突发过载)
		tagService:  tagService,
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

// CanExecute 检查 Agent 是否有能力执行该任务 (Match TaskSupport/Feature & Tags)
func (a *resourceAllocator) CanExecute(ctx context.Context, agent *agentModel.Agent, task *orchestrator.AgentTask) bool {
	// 1. RateLimiter: 速率限制
	// 全局限速: 防止 Master 被大量心跳打挂 (这里是 Dispatch 阶段，主要防止 Agent 过载)
	// 目标限速: 防止把目标网段打挂 (通常在 Task 生成阶段处理，或者这里检查并发数)

	// 基础检查：Agent 必须在线
	if agent.Status != agentModel.AgentStatusOnline {
		return false
	}

	// 2. AgentSelector: 智能匹配
	// 基于 TaskSupport 匹配: 只有安装了对应工具的 Agent 才能领取任务
	if !hasTaskSupport(agent, task.ToolName) {
		logger.LogInfo("Agent missing task_support for task", "", 0, "", "service.orchestrator.allocator.CanExecute", "", map[string]interface{}{
			"agent_id":  agent.AgentID,
			"tool_name": task.ToolName,
		})
		return false
	}

	// 3. Tag (标签) 匹配
	// 只有 "Zone:Inside" 的 Agent 才能扫内网
	if !a.matchTags(agent, task.RequiredTags) {
		return false
	}

	return true
}

// hasTaskSupport 检查 Agent 是否拥有支持的 TaskSupport
func hasTaskSupport(agent *agentModel.Agent, toolName string) bool {
	// Agent.TaskSupport 存储的是支持的工具名称列表 (例如 ["ipAliveScan", "pocScan"])
	// 直接检查 toolName 是否在列表中
	if len(agent.TaskSupport) == 0 {
		// 如果没有上报能力，默认认为没有能力
		// 为了开发方便，暂时返回 true，待 TaskSupport 系统完善后改为 false
		// return true
		return false
	}

	// 使用 Matcher 引擎进行检查
	agentData := map[string]interface{}{
		"task_support": agent.TaskSupport,
	}

	// 规则: task_support 包含 toolName (忽略大小写)
	rule := matcher.MatchRule{
		Field:      "task_support",
		Operator:   "list_contains",
		Value:      toolName,
		IgnoreCase: true,
	}

	matched, err := matcher.Match(agentData, rule)
	if err != nil {
		logger.LogWarn("Matcher error in hasTaskSupport", "", 0, "", "allocator.hasTaskSupport", "", map[string]interface{}{
			"error":     err.Error(),
			"agent_id":  agent.AgentID,
			"tool_name": toolName,
		})
		return false
	}
	return matched
}

// matchTags 检查 Agent 是否满足任务的标签要求
func (a *resourceAllocator) matchTags(agent *agentModel.Agent, requiredTagsJSON string) bool {
	if requiredTagsJSON == "" || requiredTagsJSON == "[]" {
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

	// 获取Agent标签
	// 1. 获取实体标签关联
	entityTags, err := a.tagService.GetEntityTags(context.Background(), "agent", agent.AgentID)
	if err != nil {
		logger.LogError(err, "failed to get agent entity tags", 0, "", "service.orchestrator.allocator.matchTags", "INTERNAL", nil)
		return false
	}

	if len(entityTags) == 0 {
		// 没有标签，但如果有 requiredTags，则不匹配 (除非 requiredTags 为空，上面已处理)
		return false
	}

	// 2. 提取 Tag IDs
	tagIDs := make([]uint64, 0, len(entityTags))
	for _, et := range entityTags {
		tagIDs = append(tagIDs, et.TagID)
	}

	// 3. 获取标签详情 (获取名称)
	tags, err := a.tagService.GetTagsByIDs(context.Background(), tagIDs)
	if err != nil {
		logger.LogError(err, "failed to get tag details", 0, "", "service.orchestrator.allocator.matchTags", "INTERNAL", nil)
		return false
	}

	// 转换为标签名称列表
	tagNames := make([]string, len(tags))
	for i, t := range tags {
		tagNames[i] = t.Name
	}

	// 使用 Matcher 引擎进行检查
	agentData := map[string]interface{}{
		"tags": tagNames,
	}

	// 构造规则: 必须包含所有 Required Tags (AND)
	var subRules []matcher.MatchRule
	for _, reqTag := range requiredTags {
		subRules = append(subRules, matcher.MatchRule{
			Field:    "tags",
			Operator: "list_contains",
			Value:    reqTag,
			// IgnoreCase: false, // 默认保持大小写敏感，与旧行为一致
		})
	}

	rule := matcher.MatchRule{
		And: subRules,
	}

	// 匹配器匹配的是字符串，所以需要将标签列表转换为字符串
	matched, err := matcher.Match(agentData, rule)
	if err != nil {
		logger.LogWarn("Matcher error in matchTags", "", 0, "", "allocator.matchTags", "", map[string]interface{}{
			"error":    err.Error(),
			"agent_id": agent.AgentID,
			"tags":     requiredTagsJSON,
		})
		return false
	}

	return matched
}
