package setup

import (
	"gorm.io/gorm"
	"neomaster/internal/pkg/logger"
	agentRepo "neomaster/internal/repo/mysql/agent"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/core/scheduler"
)

// BuildSchedulerService 构建调度引擎服务
func BuildSchedulerService(db *gorm.DB) scheduler.SchedulerService {
	logger.WithFields(map[string]interface{}{
		"path":      "setup.scheduler",
		"operation": "build_service",
		"func_name": "setup.BuildSchedulerService",
	}).Info("开始初始化调度引擎服务")

	// 1. Repository 初始化
	projectRepo := orcRepo.NewProjectRepository(db)
	workflowRepo := orcRepo.NewWorkflowRepository(db)
	stageRepo := orcRepo.NewScanStageRepository(db)
	taskRepo := orcRepo.NewTaskRepository(db)
	agentRepo := agentRepo.NewAgentRepository(db)

	// 2. Scheduler 初始化
	schedulerService := scheduler.NewSchedulerService(
		projectRepo,
		workflowRepo,
		stageRepo,
		taskRepo,
		agentRepo,
	)

	logger.WithFields(map[string]interface{}{
		"path":      "setup.scheduler",
		"operation": "build_service",
		"func_name": "setup.BuildSchedulerService",
	}).Info("调度引擎服务初始化完成")

	return schedulerService
}
