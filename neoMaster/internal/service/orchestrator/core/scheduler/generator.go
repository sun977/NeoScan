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
	// 如果没有目标，则返回 nil
	if len(targets) == 0 {
		return nil, nil
	}

	// 解析 PerformanceSettings
	chunkSize := 50
	timeout := 3600
	if stage.PerformanceSettings != "" {
		var perf map[string]interface{}
		if err := json.Unmarshal([]byte(stage.PerformanceSettings), &perf); err == nil {
			if cs, ok := perf["chunk_size"].(float64); ok && cs > 0 {
				chunkSize = int(cs)
			}
			if to, ok := perf["timeout"].(float64); ok && to > 0 {
				timeout = int(to)
			}
		}
	}

	// 解析 ExecutionPolicy
	priority := 0
	if stage.ExecutionPolicy != "" {
		var exec map[string]interface{}
		if err := json.Unmarshal([]byte(stage.ExecutionPolicy), &exec); err == nil {
			if p, ok := exec["priority"].(float64); ok {
				priority = int(p)
			}
		}
	}

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

		task := &orcModel.AgentTask{
			TaskID:       taskID,
			ProjectID:    projectID,
			WorkflowID:   stage.WorkflowID,
			StageID:      uint64(stage.ID),
			Status:       "pending",
			Priority:     priority,
			TaskType:     "tool", // Explicitly set default
			ToolName:     stage.ToolName,
			ToolParams:   stage.ToolParams,
			InputTarget:  string(targetsJSON),
			RequiredTags: "[]", // Default empty JSON array
			OutputResult: "{}", // Default empty JSON object
			Timeout:      timeout,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
