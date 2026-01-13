/**
 * 初始化:扫描编排器模块
 * @author: sun977
 * @date: 2025.12.05
 * @description: 扫描编排器模块初始化
 */
package setup

import (
	"time"

	"neomaster/internal/config"
	"neomaster/internal/pkg/logger"
	agentRepo "neomaster/internal/repo/mysql/agent"
	assetRepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/asset/etl"
	"neomaster/internal/service/fingerprint"
	"neomaster/internal/service/fingerprint/engines/http"
	"neomaster/internal/service/fingerprint/engines/service"
	"neomaster/internal/service/orchestrator/core/scheduler"
	"neomaster/internal/service/orchestrator/core/task_dispatcher"
	"neomaster/internal/service/orchestrator/ingestor"    // 引入ingestor
	"neomaster/internal/service/orchestrator/local_agent" // 本地Agent，用于master模块执行系统任务

	// 引入ETL
	orchestratorHandler "neomaster/internal/handler/orchestrator"
	orchestratorRepo "neomaster/internal/repo/mysql/orchestrator"
	orchestratorService "neomaster/internal/service/orchestrator"
	"neomaster/internal/service/orchestrator/allocator"
	"neomaster/internal/service/orchestrator/policy"
	"neomaster/internal/service/tag_system"

	"gorm.io/gorm"
)

// BuildOrchestratorModule 构建扫描编排器模块
func BuildOrchestratorModule(db *gorm.DB, cfg *config.Config, tagService tag_system.TagService) *OrchestratorModule {
	logger.WithFields(map[string]interface{}{
		"path":      "setup.orchestrator",
		"operation": "build_module",
		"func_name": "setup.BuildOrchestratorModule",
	}).Info("开始初始化扫描编排器模块")

	// 1. Repository 初始化
	projectRepo := orchestratorRepo.NewProjectRepository(db)
	workflowRepo := orchestratorRepo.NewWorkflowRepository(db)
	scanStageRepo := orchestratorRepo.NewScanStageRepository(db)
	scanToolTemplateRepo := orchestratorRepo.NewScanToolTemplateRepository(db)
	// TaskDispatcher 需要 TaskRepository (虽属 Agent 域，但被编排器核心组件使用)
	taskRepo := orchestratorRepo.NewTaskRepository(db)
	// AgentTaskService 需要 AgentRepository
	agentRepository := agentRepo.NewAgentRepository(db)
	// PolicyEnforcer 需要 AssetPolicyRepository
	assetPolicyRepo := assetRepo.NewAssetPolicyRepository(db)

	// 2. Core Components 初始化 (Policy Enforcer, Resource Allocator, Task Dispatcher, Scheduler)
	policyEnforcer := policy.NewPolicyEnforcer(assetPolicyRepo)
	resourceAllocator := allocator.NewResourceAllocator(tagService)
	dispatcher := task_dispatcher.NewTaskDispatcher(cfg, taskRepo, policyEnforcer, resourceAllocator)
	schedulerService := scheduler.NewSchedulerService(
		db,
		cfg,
		projectRepo,
		workflowRepo,
		scanStageRepo,
		taskRepo,
		agentRepository,
		10*time.Second, // 默认轮询间隔
	)
	localAgent := local_agent.NewLocalAgent(db, taskRepo)

	// Ingestor Components 初始化
	// 读取队列配置
	queueCapacity := cfg.App.Master.Queue.Capacity
	resultQueue := ingestor.NewMemoryQueue(queueCapacity)

	resultValidator := ingestor.NewResultValidator(taskRepo)
	// 使用配置中的路径初始化归档器
	archivePath := cfg.App.Master.Archive.Path
	if archivePath == "" {
		archivePath = "data/evidence" // 默认值
	}
	evidenceArchiver := ingestor.NewFileArchiver(archivePath)
	resultIngestor := ingestor.NewResultIngestor(resultQueue, resultValidator, evidenceArchiver)

	// ETL Components 初始化
	// 启动 Worker 消费结果队列
	etlWorkerNum := cfg.App.Master.ETL.WorkerNum
	if etlWorkerNum <= 0 {
		etlWorkerNum = 5 // 默认值
	}

	// 初始化 AssetMerger
	hostRepo := assetRepo.NewAssetHostRepository(db)
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)
	etlErrorRepo := assetRepo.NewETLErrorRepository(db)
	assetMerger := etl.NewAssetMerger(hostRepo, webRepo, vulnRepo, unifiedRepo)

	// 初始化 FingerprintService
	httpEngine := http.NewHTTPEngine(assetRepo.NewAssetFingerRepository(db))
	serviceEngine := service.NewServiceEngine(assetRepo.NewAssetCPERepository(db))
	fpService := fingerprint.NewFingerprintService(httpEngine, serviceEngine)

	// 加载指纹规则
	rulePath := cfg.GetFingerprintRulePath()
	if err := fpService.LoadRules(rulePath); err != nil {
		// 仅记录错误，不阻断启动，因为可能只使用数据库规则
		logger.LogError(err, "", 0, "", "setup.BuildOrchestratorModule", "", map[string]interface{}{
			"msg":  "Failed to load fingerprint rules from file",
			"path": rulePath,
		})
	} else {
		logger.LogInfo("Fingerprint rules loaded", "", 0, "", "setup.BuildOrchestratorModule", "", map[string]interface{}{
			"path": rulePath,
		})
	}

	etlProcessor := etl.NewResultProcessor(resultQueue, assetMerger, etlErrorRepo, etlWorkerNum)
	// TODO: 在应用启动时调用 etlProcessor.Start(ctx)

	// 3. Service 初始化
	projectService := orchestratorService.NewProjectService(projectRepo, tagService)
	workflowService := orchestratorService.NewWorkflowService(workflowRepo, tagService)
	scanStageService := orchestratorService.NewScanStageService(scanStageRepo, tagService)
	scanToolTemplateService := orchestratorService.NewScanToolTemplateService(scanToolTemplateRepo)
	// agentTaskService := orchestratorService.NewAgentTaskService(agentRepository, taskRepo, dispatcher)
	agentTaskService := task_dispatcher.NewAgentTaskService(agentRepository, taskRepo, dispatcher)

	// 4. Handler 初始化
	projectHandler := orchestratorHandler.NewProjectHandler(projectService)
	workflowHandler := orchestratorHandler.NewWorkflowHandler(workflowService)
	scanStageHandler := orchestratorHandler.NewScanStageHandler(scanStageService)
	scanToolTemplateHandler := orchestratorHandler.NewScanToolTemplateHandler(scanToolTemplateService)
	agentTaskHandler := orchestratorHandler.NewAgentTaskHandler(agentTaskService)

	logger.WithFields(map[string]interface{}{
		"path":      "setup.orchestrator",
		"operation": "build_module",
		"func_name": "setup.BuildOrchestratorModule",
	}).Info("扫描编排器模块初始化完成")

	return &OrchestratorModule{
		ProjectHandler:          projectHandler,
		WorkflowHandler:         workflowHandler,
		ScanStageHandler:        scanStageHandler,
		ScanToolTemplateHandler: scanToolTemplateHandler,
		AgentTaskHandler:        agentTaskHandler,

		ProjectService:          projectService,
		WorkflowService:         workflowService,
		ScanStageService:        scanStageService,
		ScanToolTemplateService: scanToolTemplateService,
		AgentTaskService:        agentTaskService,

		// Core Components
		TaskDispatcher:   dispatcher,
		SchedulerService: schedulerService,
		LocalAgent:       localAgent,
		ResultIngestor:   resultIngestor,
		ETLProcessor:     etlProcessor,
	}
}
