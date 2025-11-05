package setup

import (
	agentHandler "neomaster/internal/handler/agent"
	"neomaster/internal/pkg/logger"
	agentRepo "neomaster/internal/repo/mysql/agent"
	agentService "neomaster/internal/service/agent"

	"gorm.io/gorm"
)

// BuildAgentModule 构建 Agent 管理模块
// 责任边界：
// - 初始化 Agent 相关的仓库与服务（Manager/Monitor/Config/Task）。
// - 聚合为统一的 AgentHandler（内部组合上述服务），供 router_manager 进行路由注册。
// - setup 层仅负责“依赖装配”，不在此处编写业务逻辑，实现与测试保持在各自的 package 中。
//
// 参数：
// - db：数据库连接（gorm.DB），用于构建 AgentRepository（基于 gorm）。
//
// 返回：
// - *AgentModule：聚合后的 Agent 模块输出（包含 Handler 与具体 Service）。
func BuildAgentModule(db *gorm.DB) *AgentModule {
	// 结构化日志：记录模块初始化关键步骤，便于问题定位与审计
	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.agent.BuildAgentModule",
		"operation": "setup",
		"option":    "setup.agent.begin",
		"func_name": "setup.agent.BuildAgentModule",
	}).Info("开始构建 Agent 管理模块")

	// 1) 初始化仓库（统一由 gorm 管理数据库连接与事务）
	agentRepository := agentRepo.NewAgentRepository(db)

	// 2) 初始化服务（遵循 Handler → Service → Repository 层级调用约束）
	managerService := agentService.NewAgentManagerService(agentRepository)
	monitorService := agentService.NewAgentMonitorService(agentRepository)
	configService := agentService.NewAgentConfigService(agentRepository)
	taskService := agentService.NewAgentTaskService(agentRepository)

	// 3) 聚合处理器（控制器）：分组功能已合并到 ManagerService 内部
	agentHandler := agentHandler.NewAgentHandler(
		managerService,
		monitorService,
		configService,
		taskService,
	)

	// 4) 聚合输出模块，便于路由层与其他模块按需使用
	module := &AgentModule{
		AgentHandler:   agentHandler,
		ManagerService: managerService,
		MonitorService: monitorService,
		ConfigService:  configService,
		TaskService:    taskService,
	}

	logger.WithFields(map[string]interface{}{
		"path":      "internal.app.master.setup.agent.BuildAgentModule",
		"operation": "setup",
		"option":    "setup.agent.done",
		"func_name": "setup.agent.BuildAgentModule",
	}).Info("Agent 管理模块构建完成")

	return module
}
