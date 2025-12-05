/**
 * 初始化:扫描编排器模块
 * @author: sun977
 * @date: 2025.12.05
 * @description: 扫描编排器模块初始化
 */
package setup

import (
	"neomaster/internal/pkg/logger"

	orchestratorHandler "neomaster/internal/handler/orchestrator"
	orchestratorRepo "neomaster/internal/repo/mysql/orchestrator"
	orchestratorService "neomaster/internal/service/orchestrator"

	"gorm.io/gorm"
)

// BuildOrchestratorModule 构建扫描编排器模块
func BuildOrchestratorModule(db *gorm.DB) *OrchestratorModule {
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

	// 2. Service 初始化
	projectService := orchestratorService.NewProjectService(projectRepo)
	workflowService := orchestratorService.NewWorkflowService(workflowRepo)
	scanStageService := orchestratorService.NewScanStageService(scanStageRepo)
	scanToolTemplateService := orchestratorService.NewScanToolTemplateService(scanToolTemplateRepo)

	// 3. Handler 初始化
	projectHandler := orchestratorHandler.NewProjectHandler(projectService)
	workflowHandler := orchestratorHandler.NewWorkflowHandler(workflowService)
	scanStageHandler := orchestratorHandler.NewScanStageHandler(scanStageService)
	scanToolTemplateHandler := orchestratorHandler.NewScanToolTemplateHandler(scanToolTemplateService)

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

		ProjectService:          projectService,
		WorkflowService:         workflowService,
		ScanStageService:        scanStageService,
		ScanToolTemplateService: scanToolTemplateService,
	}
}
