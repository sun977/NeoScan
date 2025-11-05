package setup

import (
	orchestratorHandler "neomaster/internal/handler/orchestrator"
	"neomaster/internal/pkg/logger"
	orchestratorRepo "neomaster/internal/repo/mysql/orchestrator"
	orchestratorService "neomaster/internal/service/orchestrator"

	"gorm.io/gorm"
)

// BuildOrchestratorModule 构建扫描编排器模块（项目配置/工作流/扫描工具/扫描规则/规则引擎）
// 责任边界：
// - 初始化 orchestrator 相关的仓库与服务（ProjectConfig/Workflow/ScanTool/ScanRule）。
// - 聚合对应的 Handler（包含 RuleEngineHandler），供 router_manager 进行路由注册。
// - setup 层仅负责“依赖装配”，不在此处编写业务逻辑，实现与测试保持在各自的 package 中。
//
// 参数：
// - db：数据库连接（gorm.DB），用于构建各业务仓库（基于 gorm）。
//
// 返回：
// - *OrchestratorModule：聚合后的扫描编排器模块输出（包含 Handlers 与具体 Services）。
func BuildOrchestratorModule(db *gorm.DB) *OrchestratorModule {
	// 结构化日志：记录模块初始化关键步骤，便于问题定位与审计
	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.orchestrator.BuildOrchestratorModule",
		"operation": "setup",
		"option":    "setup.orchestrator.begin",
		"func_name": "setup.orchestrator.BuildOrchestratorModule",
	}).Info("开始构建扫描编排器模块")

	// 1) 初始化仓库（统一由 gorm 管理数据库连接与事务）
	projectConfigRepo := orchestratorRepo.NewProjectConfigRepository(db)
	workflowConfigRepo := orchestratorRepo.NewWorkflowConfigRepository(db)
	scanToolRepo := orchestratorRepo.NewScanToolRepository(db)
	scanRuleRepo := orchestratorRepo.NewScanRuleRepository(db)

	// 2) 初始化服务（遵循 Handler → Service → Repository 层级调用约束）
	projectConfigService := orchestratorService.NewProjectConfigService(projectConfigRepo, workflowConfigRepo, scanToolRepo)
	workflowService := orchestratorService.NewWorkflowService(workflowConfigRepo, projectConfigRepo, scanToolRepo, scanRuleRepo)
	scanToolService := orchestratorService.NewScanToolService(scanToolRepo)
	scanRuleService := orchestratorService.NewScanRuleService(scanRuleRepo)

	// 3) 初始化处理器（控制器）：
	projectConfigHandler := orchestratorHandler.NewProjectConfigHandler(projectConfigService)
	workflowHandler := orchestratorHandler.NewWorkflowHandler(workflowService)
	scanToolHandler := orchestratorHandler.NewScanToolHandler(scanToolService)
	// 注意：ScanRuleHandler 的构造函数接收非指针类型的 ScanRuleService，需要传入解引用后的值
	scanRuleHandler := orchestratorHandler.NewScanRuleHandler(*scanRuleService)
	// 规则引擎Handler现在完全通过ScanRuleService管理规则引擎，不再需要单独的规则引擎实例
	ruleEngineHandler := orchestratorHandler.NewRuleEngineHandler(nil, scanRuleService)

	// 4) 聚合输出模块，便于路由层与其他模块按需使用
	module := &OrchestratorModule{
		// Handlers
		ProjectConfigHandler: projectConfigHandler,
		WorkflowHandler:      workflowHandler,
		ScanToolHandler:      scanToolHandler,
		ScanRuleHandler:      scanRuleHandler,
		RuleEngineHandler:    ruleEngineHandler,
		// Services
		ProjectConfigService: projectConfigService,
		WorkflowService:      workflowService,
		ScanToolService:      scanToolService,
		ScanRuleService:      scanRuleService,
	}

	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.orchestrator.BuildOrchestratorModule",
		"operation": "setup",
		"option":    "setup.orchestrator.done",
		"func_name": "setup.orchestrator.BuildOrchestratorModule",
	}).Info("扫描编排器模块构建完成")

	return module
}
