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
	coreModel "neoagent/internal/core/model"
	"neoagent/internal/core/runner"
	modelComm "neoagent/internal/model/client"
	"neoagent/internal/pkg/logger"
	"neoagent/internal/pkg/monitor"
	"neoagent/internal/service/adapter"
	"neoagent/internal/service/client"
)

// App Agent应用程序结构体
type App struct {
	router        *router.Router
	httpServer    *http.Server
	config        *config.Config
	logger        *logger.LoggerManager
	masterService client.MasterService
	runnerManager *runner.RunnerManager
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
	serverModule := setup.SetupServer(cfg)

	return &App{
		router:        serverModule.Router,
		httpServer:    serverModule.HTTPServer,
		config:        cfg,
		logger:        loggerManager,
		masterService: clientModule.MasterService,
		runnerManager: coreModule.RunnerManager,
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
		Hostname:     hostInfo.Hostname,
		IPAddress:    a.config.Server.Host, // 优先使用配置的Host，如果为空可能需要获取真实IP
		Port:         a.config.Server.Port,
		Version:      a.config.Agent.Version,
		OS:           hostInfo.OS,
		Arch:         hostInfo.Arch,
		CPUCores:     hostInfo.CPUCores,
		MemoryTotal:  hostInfo.MemoryTotal,
		DiskTotal:    hostInfo.DiskTotal,
		Capabilities: []string{"scan", "monitor"}, // TODO: 动态获取能力
		Tags:         a.config.Agent.Tags,
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
	logger.Info("Starting task poller...")
	taskChan := a.masterService.StartTaskPoller(ctx, taskInterval)

	// 6. 处理任务
	go a.handleTasks(taskChan)
}

// handleTasks 处理从Master拉取的任务
func (a *App) handleTasks(taskChan <-chan []modelComm.Task) {
	translator := adapter.NewTaskTranslator()

	for tasks := range taskChan {
		for _, clientTask := range tasks {
			logger.Infof("Received task: %s (Type: %s)", clientTask.TaskID, clientTask.TaskType)

			// 1. 转换任务
			coreTask, err := translator.ToCoreTask(&clientTask)
			if err != nil {
				logger.Errorf("Failed to translate task %s: %v", clientTask.TaskID, err)
				a.reportTaskError(context.Background(), clientTask.TaskID, "translation_failed", err.Error())
				continue
			}

			// 2. 更新任务状态为 Running
			a.masterService.ReportTask(context.Background(), coreTask.ID, string(coreModel.TaskStatusRunning), "", "")

			// 3. 执行任务
			// TODO: 使用 Worker Pool 或协程并发执行，目前简单起见同步执行（会阻塞轮询）
			// 实际上 TaskPoller 是独立的，这里应该 spawn goroutine
			go func(task *coreModel.Task) {
				ctx := context.Background() // TODO: 使用带超时的 Context

				logger.Infof("Executing task: %s", task.ID)
				results, err := a.runnerManager.Execute(ctx, task)

				// 4. 处理结果并上报
				if err != nil {
					logger.Errorf("Task %s execution failed: %v", task.ID, err)
					a.reportTaskError(ctx, task.ID, string(coreModel.TaskStatusFailed), err.Error())
					return
				}

				// 聚合结果并上报
				// 目前 Master 协议可能期望一个聚合的 JSON 结果
				// 我们取第一个结果作为主要结果，或者需要 adapter 支持聚合
				// TaskTranslator.ToTaskStatusReport 似乎处理了 []Result

				// 构造一个包含所有结果的 TaskResult
				aggResult := &coreModel.TaskResult{
					TaskID: task.ID,
					Status: coreModel.TaskStatusSuccess,
				}

				// 将 []*TaskResult 的 Result 部分提取出来
				// 注意：Core 的 Runner 返回的是 []*TaskResult，其中每个 Result.Result 可能是具体的 struct
				// Adapter 需要处理 []interface{} 或特定类型

				// 简单起见，我们假设 Adapter 能处理 []interface{} 或者我们需要解包
				// 查看 Adapter: switch res := internalRes.Result.(type)
				// case []model.IpAliveResult: ...
				// 所以我们需要把 []*TaskResult 中的具体 Result 聚合到一个 Slice 中

				var aggData interface{}

				if len(results) > 0 {
					// 尝试推断类型并聚合
					firstRes := results[0].Result
					switch firstRes.(type) {
					case coreModel.IpAliveResult:
						var list []coreModel.IpAliveResult
						for _, r := range results {
							if v, ok := r.Result.(coreModel.IpAliveResult); ok {
								list = append(list, v)
							}
						}
						aggData = list
					case *coreModel.IpAliveResult:
						var list []coreModel.IpAliveResult
						for _, r := range results {
							if v, ok := r.Result.(*coreModel.IpAliveResult); ok {
								list = append(list, *v)
							}
						}
						aggData = list
					case coreModel.PortServiceResult:
						var list []coreModel.PortServiceResult
						for _, r := range results {
							if v, ok := r.Result.(coreModel.PortServiceResult); ok {
								list = append(list, v)
							}
						}
						aggData = list
					case *coreModel.PortServiceResult:
						var list []coreModel.PortServiceResult
						for _, r := range results {
							if v, ok := r.Result.(*coreModel.PortServiceResult); ok {
								list = append(list, *v)
							}
						}
						aggData = list
					case *coreModel.OsInfo:
						// OS Scan 通常只返回一个结果
						aggData = firstRes
					default:
						// Fallback: list of whatever
						var list []interface{}
						for _, r := range results {
							list = append(list, r.Result)
						}
						aggData = list
					}
				}

				aggResult.Result = aggData

				report, err := translator.ToTaskStatusReport(task.ID, aggResult)
				if err != nil {
					logger.Errorf("Failed to translate result for task %s: %v", task.ID, err)
					// Report error?
				} else {
					logger.Infof("Reporting success for task %s", task.ID)
					a.masterService.ReportTask(ctx, task.ID, report.Status, report.Result, report.ErrorMsg)
				}

			}(coreTask)

		}
	}
}

func (a *App) reportTaskError(ctx context.Context, taskID, status, errorMsg string) {
	a.masterService.ReportTask(ctx, taskID, status, "", errorMsg)
}
