package runner

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/core/model"
)

// RunnerManager 管理所有的 Runner
type RunnerManager struct {
	runners map[model.TaskType]Runner
	mu      sync.RWMutex
}

func NewRunnerManager() *RunnerManager {
	return &RunnerManager{
		runners: make(map[model.TaskType]Runner),
	}
}

// Register 注册一个 Runner
func (m *RunnerManager) Register(runner Runner) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runners[runner.Name()] = runner
}

// Get 获取指定类型的 Runner
func (m *RunnerManager) Get(taskType model.TaskType) (Runner, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if runner, ok := m.runners[taskType]; ok {
		return runner, nil
	}
	return nil, fmt.Errorf("no runner found for task type: %s", taskType)
}

// Execute 执行任务
func (m *RunnerManager) Execute(ctx context.Context, task *model.Task) ([]*model.TaskResult, error) {
	runner, err := m.Get(task.Type)
	if err != nil {
		return nil, err
	}
	
	return runner.Run(ctx, task)
}
