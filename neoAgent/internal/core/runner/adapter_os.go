package runner

import (
	"context"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/os"
	"time"
)

// OsRunner OS扫描器适配器
type OsRunner struct {
	scanner *os.Scanner
}

// NewOsRunner 创建新的OS扫描器适配器
func NewOsRunner(scanner *os.Scanner) *OsRunner {
	return &OsRunner{scanner: scanner}
}

// Name 返回Runner名称
func (r *OsRunner) Name() model.TaskType {
	return model.TaskTypeOsScan
}

// Run 执行OS扫描
func (r *OsRunner) Run(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	startTime := time.Now()
	
	mode := "auto"
	if m, ok := task.Params["mode"].(string); ok {
		mode = m
	}

	info, err := r.scanner.Scan(ctx, task.Target, mode)
	
	result := &model.TaskResult{
		TaskID:      task.ID,
		Status:      model.TaskStatusSuccess,
		ExecutedAt:  startTime,
		CompletedAt: time.Now(),
	}

	if err != nil {
		result.Status = model.TaskStatusFailed
		result.Error = err.Error()
	} else {
		result.Result = info
	}

	return []*model.TaskResult{result}, nil
}
