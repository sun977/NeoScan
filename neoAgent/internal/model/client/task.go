package client

import (
	"encoding/json"
	"neoagent/internal/core/model"
	"time"
)

// Task Master分配的任务
type Task struct {
	TaskID      string `json:"task_id"`
	ProjectID   int    `json:"project_id"`
	TaskType    string `json:"task_type"`
	ToolName    string `json:"tool_name"`
	ToolParams  string `json:"tool_params"`
	InputTarget string `json:"input_target"` // JSON string
	Timeout     int    `json:"timeout"`
}

// Target 任务目标 (从InputTarget解析)
type Target struct {
	Type   string            `json:"type"`
	Value  string            `json:"value"`
	Source string            `json:"source"`
	Meta   map[string]string `json:"meta"`
}

// FetchTasksResponse 拉取任务响应
type FetchTasksResponse struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
	Data   []Task `json:"data"`
}

// TaskStatusReport 任务状态上报
type TaskStatusReport struct {
	Status   string `json:"status"`
	Result   string `json:"result"` // JSON string
	ErrorMsg string `json:"error_msg"`
}

// TaskStatusResponse 状态上报响应
type TaskStatusResponse struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
}

// ToCoreTask 转换为核心任务模型
func (t *Task) ToCoreTask() (*model.Task, error) {
	// 解析 InputTarget
	var targets []Target
	if err := json.Unmarshal([]byte(t.InputTarget), &targets); err != nil {
		return nil, err
	}

	// 假设一个任务对应一个主要目标，或者需要在CoreTask中处理多个目标
	// 这里简化处理，取第一个目标
	targetValue := ""
	params := make(map[string]interface{})
	
	if len(targets) > 0 {
		targetValue = targets[0].Value
		// 将Meta信息放入Params
		for k, v := range targets[0].Meta {
			params[k] = v
		}
	}

	// 转换TaskType
	// 需要根据Master定义的TaskType映射到Core的TaskType
	// 这里假设Master直接传递了对应的Type或者需要Mapper
	// 暂时直接使用
	
	params["tool_name"] = t.ToolName
	params["tool_params"] = t.ToolParams
	
	return &model.Task{
		ID:        t.TaskID,
		Type:      model.TaskType(t.TaskType),
		Target:    targetValue,
		Params:    params,
		Timeout:   time.Duration(t.Timeout) * time.Second,
		CreatedAt: time.Now(),
	}, nil
}
