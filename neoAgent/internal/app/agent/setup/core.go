package setup

import (
	"neoagent/internal/core/runner"
	"neoagent/internal/core/scanner/alive"
	"neoagent/internal/core/scanner/os"
	"neoagent/internal/core/scanner/port_service"
)

// SetupCore 初始化核心扫描模块
func SetupCore() *CoreModule {
	// 初始化扫描引擎和Runner
	runnerMgr := runner.NewRunnerManager()

	// 1. Alive Scanner
	aliveScanner := alive.NewIpAliveScanner()
	runnerMgr.Register(aliveScanner)

	// 2. Port Service Scanner
	portScanner := port_service.NewPortServiceScanner()
	runnerMgr.Register(portScanner)
	// Service Scan 也使用 PortScanner (通过适配器)
	runnerMgr.Register(runner.NewServiceRunner(portScanner))

	// 3. OS Scanner
	osScanner := os.NewScanner()
	osScanner.Register(os.NewTTLEngine())
	osScanner.Register(os.NewNmapServiceEngine())
	runnerMgr.Register(runner.NewOsRunner(osScanner))

	return &CoreModule{
		RunnerManager: runnerMgr,
	}
}
