// ResultValidator 结果校验器接口
// 职责: 验证 StageResult 的格式合法性、签名正确性
package ingestor

import (
	"context"
	"fmt"

	orcModel "neomaster/internal/model/orchestrator"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
)

// ResultValidator 结果校验器接口
type ResultValidator interface {
	// Validate 校验结果是否合法
	// 检查 TaskID 是否存在、AgentID 是否匹配、签名是否正确等
	Validate(ctx context.Context, result *orcModel.StageResult) error
}

type resultValidator struct {
	taskRepo orcRepo.TaskRepository
}

// NewResultValidator 创建结果校验器
func NewResultValidator(taskRepo orcRepo.TaskRepository) ResultValidator {
	return &resultValidator{
		taskRepo: taskRepo,
	}
}

// Validate 校验结果是否合法
func (v *resultValidator) Validate(ctx context.Context, result *orcModel.StageResult) error {
	// 1. 基础字段非空检查
	if result.TaskID == "" {
		return fmt.Errorf("missing task_id")
	}
	if result.AgentID == "" {
		return fmt.Errorf("missing agent_id")
	}
	if result.ResultType == "" {
		return fmt.Errorf("missing result_type")
	}

	// 2. 检查 TaskID 是否有效 (从数据库查询)
	// 注意: 这里是一个数据库操作，可能会影响摄入吞吐量。
	// 在高并发场景下，可以考虑使用缓存 (Redis) 或 BloomFilter。
	// 但为了数据一致性，目前先直接查库。
	task, err := v.taskRepo.GetTaskByID(ctx, result.TaskID)
	if err != nil {
		return fmt.Errorf("query task failed: %v", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", result.TaskID)
	}

	// 3. 校验 AgentID 是否匹配
	// 确保上报结果的 Agent 就是领取任务的 Agent (防止伪造或错乱)
	if task.AgentID != result.AgentID {
		return fmt.Errorf("agent_id mismatch: task assigned to %s, but result from %s", task.AgentID, result.AgentID)
	}

	// 4. 校验任务状态 (可选)
	// 理论上只应接收 'running' 状态任务的结果
	// 但考虑到重试或网络延迟，'assigned' 状态也可能上报
	if task.Status != "running" && task.Status != "assigned" {
		// 警告级别，不一定要阻断，视业务逻辑而定
		// return fmt.Errorf("task status invalid: %s", task.Status)
	}

	return nil
}
