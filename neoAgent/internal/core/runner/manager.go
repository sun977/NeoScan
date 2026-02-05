package runner

import (
	"context"
	"fmt"
	"sync"

	"neoagent/internal/core/factory"
	"neoagent/internal/core/model"
)

// RunnerManager 管理所有的 Runner
type RunnerManager struct {
	runners map[model.TaskType]Runner
	mu      sync.RWMutex
}

func NewRunnerManager() *RunnerManager {
	m := &RunnerManager{
		runners: make(map[model.TaskType]Runner),
	}

	// 初始化并注册 BruteScanner
	// 使用 Factory 获取全功能 BruteScanner
	bs := factory.NewFullBruteScanner()
	m.Register(bs)

	// 注册 AliveScanner
	aliveScanner := factory.NewAliveScanner()
	m.Register(aliveScanner)

	// 注册 PortScanner
	portScanner := factory.NewPortScanner()
	m.Register(portScanner)

	// 注册 ServiceScanner (使用 Adapter)
	// ServiceScanner 复用 PortScanner 的底层逻辑，但通过 Adapter 调整行为
	serviceRunner := NewServiceRunner(factory.NewPortScanner())
	m.Register(serviceRunner)

	// 注册 OsScanner (使用 Adapter)
	osScanner := factory.NewOsScanner()
	osRunner := NewOsRunner(osScanner)
	m.Register(osRunner)

	return m
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
