package setup

import (
	"neoagent/internal/core/runner"
)

// SetupCore 初始化核心扫描模块
func SetupCore() *CoreModule {
	// 初始化扫描引擎和Runner
	// NewRunnerManager 内部已经使用 factory 包统一加载了所有标准扫描器
	// 包括：Alive, Port, Service, OS, Brute
	runnerMgr := runner.NewRunnerManager()

	return &CoreModule{
		RunnerManager: runnerMgr,
	}
}
