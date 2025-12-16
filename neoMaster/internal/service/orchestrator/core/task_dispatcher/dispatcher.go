// TaskDispatcher 任务分发器接口
// 负责将任务分配给合适的 Agent (Push/Pull 模式)
package task_dispatcher

import (
	"context"
	"neomaster/internal/model/orchestrator"
	agentRepo "neomaster/internal/repo/mysql/orchestrator"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/orchestrator/allocator"
	"neomaster/internal/service/orchestrator/policy"
)

// TaskDispatcher 任务分发器接口
// 负责将任务分配给合适的 Agent (Push/Pull 模式)
// 对应文档: 1.1 Orchestrator - TaskDispatcher
type TaskDispatcher interface {
	// Dispatch 为指定 Agent 分配任务 (Pull 模式)
	// 检查 Agent 当前负载，从队列中获取待执行任务并分配
	// 结合 Resource Allocator 和 Policy Enforcer 进行决策
	Dispatch(ctx context.Context, agent *agentModel.Agent, currentLoad int) ([]*orchestrator.AgentTask, error)
}

type taskDispatcher struct {
	taskRepo  agentRepo.TaskRepository
	policy    policy.PolicyEnforcer       // 策略执行器注入
	allocator allocator.ResourceAllocator // 资源分配器注入
}

// NewTaskDispatcher 创建任务分发器实例
func NewTaskDispatcher(
	taskRepo agentRepo.TaskRepository,
	policy policy.PolicyEnforcer,
	allocator allocator.ResourceAllocator,
) TaskDispatcher {
	return &taskDispatcher{
		taskRepo:  taskRepo,
		policy:    policy,
		allocator: allocator,
	}
}

// Dispatch 为指定 Agent 分配任务
func (d *taskDispatcher) Dispatch(ctx context.Context, agent *agentModel.Agent, currentLoad int) ([]*orchestrator.AgentTask, error) {
	// 0. Resource Allocator: Rate Limiting Check
	// 防止单个 Agent 请求过于频繁
	if !d.allocator.Allow(ctx, agent.AgentID) {
		return nil, nil
	}

	// TODO: 从配置中获取最大并发任务数 (PerformanceSettings)
	maxTasks := 5

	if currentLoad >= maxTasks {
		return nil, nil // 负载已满，不分配新任务
	}

	needed := maxTasks - currentLoad

	// 1. 获取待执行任务
	// 这里获取比 needed 更多的任务，因为有些任务可能被 Allocator 或 Policy 过滤掉
	// TODO: 优化查询，支持按优先级排序
	pendingTasks, err := d.taskRepo.GetPendingTasks(ctx, "agent", needed*3)
	if err != nil {
		logger.LogError(err, "failed to get pending tasks", 0, "", "service.orchestrator.dispatcher.Dispatch", "REPO", nil)
		return nil, err
	}

	if len(pendingTasks) == 0 {
		return nil, nil
	}

	var assignedTasks []*orchestrator.AgentTask
	assignedCount := 0

	// 2. 遍历任务进行分配
	for _, task := range pendingTasks {
		if assignedCount >= needed {
			break
		}

		// 2.1 Resource Allocator: 资源调度检查
		// 检查 Agent 是否有能力执行该任务 (Match Capability & Tags)
		if !d.allocator.CanExecute(ctx, agent, task) {
			// Agent 不匹配，跳过此任务 (继续寻找下一个)
			continue
		}

		// 2.2 Policy Enforcer: 策略检查
		// 最后一道防线：检查任务是否合规 (Whitelist, Scope)
		if err := d.policy.Enforce(ctx, task); err != nil {
			// 策略违规！
			logger.LogInfo("Task policy violation, marking as failed", "", 0, "", "service.orchestrator.dispatcher.Dispatch", "", map[string]interface{}{
				"task_id": task.TaskID,
				"reason":  err.Error(),
			})
			// 标记任务失败，避免反复调度
			d.taskRepo.UpdateTaskResult(ctx, task.TaskID, "", "Policy Violation: "+err.Error(), "failed")
			continue
		}

		// 2.3 尝试领取任务 (CAS / Transaction)
		// ClaimTask 应该是原子操作 (UPDATE ... WHERE status='pending')
		if err := d.taskRepo.ClaimTask(ctx, task.TaskID, agent.AgentID); err != nil {
			// 领取失败（可能被其他 Agent 抢占），记录日志但继续尝试下一个
			logger.LogInfo("failed to claim task (race condition?)", "", 0, "", "service.orchestrator.dispatcher.Dispatch", "", map[string]interface{}{
				"task_id":  task.TaskID,
				"agent_id": agent.AgentID,
				"error":    err.Error(),
			})
			continue
		}

		logger.LogInfo("Task assigned to Agent", "", 0, "", "service.orchestrator.dispatcher.Dispatch", "", map[string]interface{}{
			"task_id":  task.TaskID,
			"agent_id": agent.AgentID,
		})

		// 重新获取任务详情（确保状态最新）
		if t, err := d.taskRepo.GetTaskByID(ctx, task.TaskID); err == nil {
			assignedTasks = append(assignedTasks, t)
			assignedCount++
		}
	}

	return assignedTasks, nil
}
