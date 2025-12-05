package scheduler

import (
	"encoding/json"
	"fmt"

	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/utils"
)

// TaskGenerator 任务生成器接口
type TaskGenerator interface {
	GenerateTasks(stage *orcModel.ScanStage, projectID uint64, targets []string) ([]*orcModel.AgentTask, error)
}

type taskGenerator struct{}

func NewTaskGenerator() TaskGenerator {
	return &taskGenerator{}
}

// GenerateTasks 根据 Stage 和目标生成任务
func (g *taskGenerator) GenerateTasks(stage *orcModel.ScanStage, projectID uint64, targets []string) ([]*orcModel.AgentTask, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	// 目标分片策略
	// 默认每 50 个目标一个任务，避免单个任务过大
	// TODO: 从 PerformanceSettings 获取分片大小
	chunkSize := 50
	var tasks []*orcModel.AgentTask

	for i := 0; i < len(targets); i += chunkSize {
		end := i + chunkSize
		if end > len(targets) {
			end = len(targets)
		}
		chunk := targets[i:end]

		targetsJSON, err := json.Marshal(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal targets: %v", err)
		}

		taskID, err := utils.GenerateUUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate task ID: %v", err)
		}

		// TODO: 从 ExecutionPolicy 获取优先级
		priority := 0

		// TODO: 从 PerformanceSettings 获取超时
		timeout := 3600

		task := &orcModel.AgentTask{
			TaskID:      taskID,
			ProjectID:   projectID,
			WorkflowID:  stage.WorkflowID,
			StageID:     uint64(stage.ID),
			Status:      "pending",
			Priority:    priority,
			ToolName:    stage.ToolName,
			ToolParams:  stage.ToolParams,
			InputTarget: string(targetsJSON),
			Timeout:     timeout,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
