package orchestrator

import (
	"context"
	"fmt"
	"time"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
	agentRepository "neomaster/internal/repo/mysql/agent"
	orchestratorRepository "neomaster/internal/repo/mysql/orchestrator"
	taskDispatcher "neomaster/internal/service/orchestrator/core/task_dispatcher"
)

// AgentTaskService Agent任务服务接口
// 专门负责Agent的任务相关功能，遵循单一职责原则
type AgentTaskService interface {
	// Agent任务管理
	AssignTask(req *agentModel.AgentTaskAssignRequest) (*agentModel.AgentTaskAssignmentResponse, error)
	FetchTasks(ctx context.Context, agentID string) ([]*agentModel.AgentTaskAssignmentResponse, error)
	UpdateTaskStatus(ctx context.Context, taskID string, status string, result string, errorMsg string) error // 更新任务状态
	CancelTask(ctx context.Context, taskID string) error                                                      // 取消任务
}

// agentTaskService Agent任务服务实现
type agentTaskService struct {
	agentRepo  agentRepository.AgentRepository       // Agent数据访问层
	taskRepo   orchestratorRepository.TaskRepository // 任务数据访问层
	dispatcher taskDispatcher.TaskDispatcher         // 任务分发器
}

// NewAgentTaskService 创建Agent任务服务实例
// 遵循依赖注入原则，保持代码的可测试性
func NewAgentTaskService(
	agentRepo agentRepository.AgentRepository,
	taskRepo orchestratorRepository.TaskRepository,
	dispatcher taskDispatcher.TaskDispatcher,
) AgentTaskService {
	return &agentTaskService{
		agentRepo:  agentRepo,
		taskRepo:   taskRepo,
		dispatcher: dispatcher,
	}
}

// AssignTask 分配任务给Agent服务
func (s *agentTaskService) AssignTask(req *agentModel.AgentTaskAssignRequest) (*agentModel.AgentTaskAssignmentResponse, error) {
	// 1. 验证任务是否存在 (逻辑上由调用方保证，但这里可以double check)
	ctx := context.Background()
	task, err := s.taskRepo.GetTaskByID(ctx, req.TaskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", req.TaskID)
	}

	// 2. 模拟下发任务 (实际应调用gRPC/HTTP Client)
	// TODO: Implement actual dispatch logic
	logger.LogInfo("模拟下发任务到Agent", "", 0, "", "service.agent.task.AssignTask", "", map[string]interface{}{
		"task_id":   req.TaskID,
		"agent_id":  task.AgentID,
		"task_type": req.TaskType,
	})

	// 3. 更新任务状态为 Assigned
	if err := s.taskRepo.UpdateTaskStatus(ctx, req.TaskID, "assigned"); err != nil {
		return nil, err
	}

	return &agentModel.AgentTaskAssignmentResponse{
		AgentID:    task.AgentID,
		TaskID:     req.TaskID,
		TaskType:   req.TaskType,
		Status:     "assigned",
		AssignedAt: time.Now(),
		Message:    "Task assigned successfully",
	}, nil
}

// FetchTasks 获取Agent任务列表服务
func (s *agentTaskService) FetchTasks(ctx context.Context, agentID string) ([]*agentModel.AgentTaskAssignmentResponse, error) {
	// 0. 验证 Agent
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent info: %v", err)
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// 1. 获取当前已分配给该Agent的任务（assigned, running）
	tasks, err := s.taskRepo.GetTasksByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// 2. 调用任务分发器尝试获取新任务
	newTasks, err := s.dispatcher.Dispatch(ctx, agent, len(tasks))
	if err != nil {
		// 分发失败仅记录日志，不影响返回已有任务
		logger.LogError(err, "failed to dispatch tasks", 0, "", "service.agent.task.FetchTasks", "INTERNAL", nil)
	} else if len(newTasks) > 0 {
		tasks = append(tasks, newTasks...)
	}

	// 3. 转换为响应格式
	var response []*agentModel.AgentTaskAssignmentResponse
	for _, t := range tasks {
		var assignedAt time.Time
		if t.AssignedAt != nil {
			assignedAt = *t.AssignedAt
		}

		response = append(response, &agentModel.AgentTaskAssignmentResponse{
			AgentID:     t.AgentID,
			TaskID:      t.TaskID,
			TaskType:    t.ToolName, // Use ToolName as TaskType
			Status:      agentModel.AgentTaskStatus(t.Status),
			AssignedAt:  assignedAt,
			ToolName:    t.ToolName,
			ToolParams:  t.ToolParams,
			InputTarget: t.InputTarget,
			Message:     "Task fetched successfully",
		})
	}

	return response, nil
}

// UpdateTaskStatus 更新任务状态服务
func (s *agentTaskService) UpdateTaskStatus(ctx context.Context, taskID string, status string, result string, errorMsg string) error {
	// 1. 验证任务是否存在
	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 2. 状态机检查 (Simple State Machine)
	// Valid transitions:
	// pending -> assigned (handled by ClaimTask, but here checking current status)
	// assigned -> running
	// running -> completed
	// running -> failed
	// * -> cancelled

	currentStatus := task.Status

	if status == "cancelled" {
		// Always allow cancellation unless already completed
		if currentStatus == "completed" {
			return fmt.Errorf("cannot cancel completed task")
		}
	} else {
		switch currentStatus {
		case "assigned":
			if status != "running" && status != "failed" {
				return fmt.Errorf("invalid transition from assigned to %s", status)
			}
		case "running":
			if status != "completed" && status != "failed" {
				return fmt.Errorf("invalid transition from running to %s", status)
			}
		case "completed", "failed":
			return fmt.Errorf("task already in terminal state: %s", currentStatus)
		case "pending":
			return fmt.Errorf("task must be claimed (assigned) before updates")
		default:
			// unknown state
		}
	}

	// 3. 更新状态和结果
	if status == "completed" || status == "failed" {
		return s.taskRepo.UpdateTaskResult(ctx, taskID, result, errorMsg, status)
	}

	return s.taskRepo.UpdateTaskStatus(ctx, taskID, status)
}

// CancelTask 取消任务服务
func (s *agentTaskService) CancelTask(ctx context.Context, taskID string) error {
	return s.taskRepo.UpdateTaskStatus(ctx, taskID, "cancelled")
}
