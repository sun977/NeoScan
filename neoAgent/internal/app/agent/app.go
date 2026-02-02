/**
 * Agent应用程序核心逻辑
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent应用的核心逻辑，负责初始化各种组件和服务
 * @architecture: 参考Master的架构模式，将应用逻辑从main函数中分离
 */

package agent

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"neoagent/internal/app/agent/router"
	"neoagent/internal/app/agent/setup"
	"neoagent/internal/config"
	"neoagent/internal/core/runner"
	modelComm "neoagent/internal/model/client"
	"neoagent/internal/pkg/logger"
	"neoagent/internal/pkg/monitor"
	"neoagent/internal/service/adapter"
	"neoagent/internal/service/client"
	"neoagent/internal/service/task"
)

// App Agent应用程序结构体
type App struct {
	router        *router.Router
	httpServer    *http.Server
	config        *config.Config
	logger        *logger.LoggerManager
	masterService client.MasterService
	runnerManager *runner.RunnerManager
	taskService   task.AgentTaskService
}

// NewApp 创建新的Agent应用程序实例
func NewApp() (*App, error) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 初始化日志管理器
	loggerManager, err := logger.InitLogger(cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	// 设置全局日志实例
	logger.LoggerInstance = loggerManager

	// 记录应用启动日志
	logger.Info("NeoAgent application initializing...")

	// 初始化各模块
	clientModule := setup.SetupClient(cfg)
	coreModule := setup.SetupCore()

	// 初始化任务服务（因为ServerModule依赖它）
	taskService := task.NewAgentTaskService(
		clientModule.MasterService,
		coreModule.RunnerManager,
		adapter.NewTaskTranslator(),
		cfg,
	)

	serverModule := setup.SetupServer(cfg, taskService)

	return &App{
		router:        serverModule.Router,
		httpServer:    serverModule.HTTPServer,
		config:        cfg,
		logger:        loggerManager,
		masterService: clientModule.MasterService,
		runnerManager: coreModule.RunnerManager,
		taskService:   taskService,
	}, nil
}

// GetRouter 获取路由器实例
func (a *App) GetRouter() *router.Router {
	return a.router
}

// GetConfig 获取配置实例
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetHTTPServer 获取HTTP服务器实例
func (a *App) GetHTTPServer() *http.Server {
	return a.httpServer
}

// Start 启动Agent应用程序
func (a *App) Start() error {
	logger.Info("Starting NeoAgent server...")

	// 启动HTTP服务器
	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start HTTP server: ", err)
		}
	}()

	logger.Infof("NeoAgent started successfully on port %d", a.config.Server.Port)

	// 启动Master服务交互（后台运行）
	if a.masterService != nil && a.config.Agent != nil && a.config.Agent.AutoRegister {
		go a.startMasterService(context.Background())
	}

	// TODO: 启动其他后台服务
	// 1. 任务执行器管理器
	// 2. 监控数据收集器

	return nil
}

// Stop 停止Agent应用程序
func (a *App) Stop(ctx context.Context) error {
	logger.Info("Stopping NeoAgent server...")

	// 停止HTTP服务器
	if err := a.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop HTTP server: %w", err)
	}

	logger.Info("NeoAgent stopped successfully")
	return nil
}

// startMasterService 启动Master服务交互
func (a *App) startMasterService(ctx context.Context) {
	// 1. 获取主机信息
	hostInfo, err := monitor.GetHostInfo()
	if err != nil {
		logger.Error("Failed to get host info for registration: ", err)
		// 即使获取失败，也尝试注册，使用空值或默认值
		if hostInfo == nil {
			hostInfo = &monitor.HostInfo{}
		}
	}

	// 2. 构建注册请求
	req := &modelComm.AgentRegisterRequest{
		Hostname:    hostInfo.Hostname,
		IPAddress:   a.config.Server.Host, // 优先使用配置的Host，如果为空可能需要获取真实IP
		Port:        a.config.Server.Port,
		Version:     a.config.Agent.Version,
		OS:          hostInfo.OS,
		Arch:        hostInfo.Arch,
		CPUCores:    hostInfo.CPUCores,
		MemoryTotal: hostInfo.MemoryTotal,
		DiskTotal:   hostInfo.DiskTotal,
		TaskSupport: []string{"ipAliveScan", "fastPortScan", "fullPortScan", "serviceScan", "webScan"}, // 使用 Master 定义的有效 ScanType 这里应该先获取本机agent的能力类型
		Tags:        a.config.Agent.Tags,
		TokenSecret: a.config.Master.TokenSecret,
	}

	// 3. 注册重试循环
	retryCount := 0
	maxRetries := a.config.Master.MaxReconnectAttempts
	if maxRetries <= 0 {
		maxRetries = 10 // 默认重试10次
	}
	retryInterval := a.config.Master.ReconnectInterval
	if retryInterval <= 0 {
		retryInterval = 5 * time.Second
	}

	for {
		err := a.masterService.Register(ctx, req)
		if err == nil {
			break
		}

		retryCount++
		logger.Errorf("Failed to register with Master (attempt %d/%d): %v", retryCount, maxRetries, err)

		if retryCount >= maxRetries {
			logger.Error("Max registration retries reached. Agent will run in standalone mode or wait for manual intervention.")
			return
		}

		time.Sleep(retryInterval)
	}

	// 4. 注册成功，开启心跳
	logger.Info("Successfully registered with Master. Starting heartbeat...")
	a.masterService.StartHeartbeat(ctx)

	// 5. 开启任务轮询
	// TODO: 这里的interval应该从Master获取或者配置
	taskInterval := 5 * time.Second
	logger.Info("Starting task poller worker...")

	// 启动任务服务的工作者循环（Outbound能力）
	go a.taskService.StartWorker(ctx, taskInterval)
}
