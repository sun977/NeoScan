/**
 * 路由:扫描配置路由
 * @author: Linus Torvalds (AI Assistant)
 * @date: 2025.10.11
 * @description: 扫描配置模块路由，包含项目配置、工作流、扫描工具、扫描规则的路由定义
 * @func: 提供扫描配置相关的API路由配置
 */
package router

import (
	"github.com/gin-gonic/gin"
)

// setupScanConfigRoutes 设置扫描配置路由
// 这是一个"好品味"的路由设计 - 消除了特殊情况，统一了权限控制
func (r *Router) setupScanConfigRoutes(v1 *gin.RouterGroup) {
	// 扫描配置路由组 - 需要JWT认证和用户激活状态检查
	scanConfig := v1.Group("/scan-config")
	scanConfig.Use(r.middlewareManager.GinJWTAuthMiddleware())    // JWT认证中间件
	scanConfig.Use(r.middlewareManager.GinUserActiveMiddleware()) // 用户激活状态检查中间件
	{
		// ==================== 项目配置管理路由 ====================
		// 项目配置是扫描配置的核心 - 所有其他配置都依赖于项目
		projects := scanConfig.Group("/projects")
		{
			// 基础CRUD操作
		projects.GET("", r.projectConfigHandler.ListProjectConfigs)       // 获取项目配置列表
		projects.GET("/:id", r.projectConfigHandler.GetProjectConfig)   // 获取项目配置详情
		projects.POST("", r.projectConfigHandler.CreateProjectConfig)       // 创建项目配置
		projects.PUT("/:id", r.projectConfigHandler.UpdateProjectConfig)    // 更新项目配置
		projects.DELETE("/:id", r.projectConfigHandler.DeleteProjectConfig) // 删除项目配置

		// 状态管理操作
		projects.POST("/:id/enable", r.projectConfigHandler.EnableProjectConfig)   // 启用项目配置
		projects.POST("/:id/disable", r.projectConfigHandler.DisableProjectConfig) // 禁用项目配置

		// 配置同步和热重载
		projects.POST("/:id/sync", r.projectConfigHandler.SyncProjectConfig)     // 同步项目配置
		projects.POST("/:id/reload", r.projectConfigHandler.ReloadProjectConfig) // 热重载项目配置
		}

		// ==================== 工作流配置管理路由 ====================
		// 工作流编排扫描任务的执行流程
		workflows := scanConfig.Group("/workflows")
		{
			// 基础CRUD操作
		workflows.GET("", r.workflowHandler.ListWorkflows)       // 获取工作流列表
		workflows.GET("/:id", r.workflowHandler.GetWorkflow)     // 获取工作流详情
		workflows.POST("", r.workflowHandler.CreateWorkflow)       // 创建工作流
		workflows.PUT("/:id", r.workflowHandler.UpdateWorkflow)    // 更新工作流
		workflows.DELETE("/:id", r.workflowHandler.DeleteWorkflow) // 删除工作流

		// 工作流执行控制
		workflows.POST("/:id/execute", r.workflowHandler.ExecuteWorkflow) // 执行工作流
		workflows.POST("/:id/stop", r.workflowHandler.StopWorkflow)       // 停止工作流
		workflows.POST("/:id/pause", r.workflowHandler.PauseWorkflow)     // 暂停工作流
		workflows.POST("/:id/resume", r.workflowHandler.ResumeWorkflow)   // 恢复工作流
		workflows.POST("/:id/retry", r.workflowHandler.RetryWorkflow)     // 重试工作流

		// 状态管理操作
		workflows.POST("/:id/enable", r.workflowHandler.EnableWorkflow)   // 启用工作流
		workflows.POST("/:id/disable", r.workflowHandler.DisableWorkflow) // 禁用工作流

		// 工作流监控和日志
		workflows.GET("/:id/status", r.workflowHandler.GetWorkflowStatus)   // 获取工作流状态
		workflows.GET("/:id/logs", r.workflowHandler.GetWorkflowLogs)       // 获取工作流日志
		workflows.GET("/:id/metrics", r.workflowHandler.GetWorkflowMetrics) // 获取工作流指标

		// 按项目获取工作流
		workflows.GET("/project/:project_id", r.workflowHandler.GetWorkflowsByProject) // 按项目获取工作流
		}

		// ==================== 扫描工具管理路由 ====================
		// 扫描工具是执行具体扫描任务的组件
		tools := scanConfig.Group("/tools")
		{
			// 基础CRUD操作
		tools.GET("", r.scanToolHandler.ListScanTools)       // 获取扫描工具列表
		tools.GET("/:id", r.scanToolHandler.GetScanTool)     // 获取扫描工具详情
		tools.POST("", r.scanToolHandler.CreateScanTool)       // 创建扫描工具
		tools.PUT("/:id", r.scanToolHandler.UpdateScanTool)    // 更新扫描工具
		tools.DELETE("/:id", r.scanToolHandler.DeleteScanTool) // 删除扫描工具

		// 状态管理操作
		tools.POST("/:id/enable", r.scanToolHandler.EnableScanTool)   // 启用扫描工具
		tools.POST("/:id/disable", r.scanToolHandler.DisableScanTool) // 禁用扫描工具

		// 工具管理操作
		tools.POST("/:id/install", r.scanToolHandler.InstallScanTool)     // 安装扫描工具
		tools.POST("/:id/uninstall", r.scanToolHandler.UninstallScanTool) // 卸载扫描工具
		tools.GET("/:id/health", r.scanToolHandler.HealthCheckScanTool)   // 检查工具健康状态

		// 工具查询和指标
		tools.GET("/available", r.scanToolHandler.GetAvailableScanTools) // 获取可用扫描工具
		tools.GET("/type/:type", r.scanToolHandler.GetScanToolsByType)   // 按类型获取扫描工具
		tools.GET("/:id/metrics", r.scanToolHandler.GetScanToolMetrics)  // 获取工具指标
		}

		// ==================== 扫描规则管理路由 ====================
		// 扫描规则定义具体的扫描检查逻辑
		rules := scanConfig.Group("/rules")
		{
			// 基础CRUD操作
			rules.GET("", r.scanRuleHandler.GetScanRuleList)       // 获取扫描规则列表
			rules.GET("/:id", r.scanRuleHandler.GetScanRuleByID)   // 获取扫描规则详情
			rules.POST("", r.scanRuleHandler.CreateScanRule)       // 创建扫描规则
			rules.PUT("/:id", r.scanRuleHandler.UpdateScanRule)    // 更新扫描规则
			rules.DELETE("/:id", r.scanRuleHandler.DeleteScanRule) // 删除扫描规则

			// 状态管理操作
			rules.POST("/:id/enable", r.scanRuleHandler.EnableScanRule)   // 启用扫描规则
			rules.POST("/:id/disable", r.scanRuleHandler.DisableScanRule) // 禁用扫描规则

			// 规则测试和验证
			rules.POST("/:id/test", r.scanRuleHandler.TestScanRule) // 测试扫描规则

			// 规则查询和分类
			rules.GET("/type/:type", r.scanRuleHandler.GetScanRulesByType)             // 按类型获取扫描规则
			rules.GET("/severity/:severity", r.scanRuleHandler.GetScanRulesBySeverity) // 按严重程度获取扫描规则
			rules.GET("/active", r.scanRuleHandler.GetActiveScanRules)                 // 获取活跃扫描规则

			// 规则导入导出
			rules.POST("/import", r.scanRuleHandler.ImportScanRules) // 导入扫描规则
			rules.GET("/export", r.scanRuleHandler.ExportScanRules)  // 导出扫描规则

			// 规则指标
		rules.GET("/:id/metrics", r.scanRuleHandler.GetScanRuleMetrics) // 获取规则指标
		}

		// ==================== 规则引擎管理路由 ====================
		// 规则引擎负责规则的执行、验证和管理
		ruleEngine := scanConfig.Group("/rule-engine")
		{
			// 规则执行相关
			ruleEngine.POST("/rules/:id/execute", r.ruleEngineHandler.ExecuteRule)      // 执行单个规则
			ruleEngine.POST("/rules/batch-execute", r.ruleEngineHandler.ExecuteRules)   // 批量执行规则
			
			// 规则验证相关
			ruleEngine.POST("/rules/validate", r.ruleEngineHandler.ValidateRule)        // 验证规则
			ruleEngine.POST("/conditions/parse", r.ruleEngineHandler.ParseCondition)    // 解析条件表达式
			
			// 引擎管理相关
			ruleEngine.GET("/metrics", r.ruleEngineHandler.GetEngineMetrics)            // 获取引擎指标
			ruleEngine.POST("/cache/clear", r.ruleEngineHandler.ClearCache)             // 清空缓存
		}
	}

	// ==================== 管理员专用扫描配置路由 ====================
	// 需要管理员权限的高级配置操作
	adminScanConfig := v1.Group("/admin/scan-config")
	adminScanConfig.Use(r.middlewareManager.GinJWTAuthMiddleware())    // JWT认证中间件
	adminScanConfig.Use(r.middlewareManager.GinUserActiveMiddleware()) // 用户激活状态检查中间件
	adminScanConfig.Use(r.middlewareManager.GinAdminRoleMiddleware())  // 管理员权限检查中间件
	{
		// 系统级配置管理
		adminScanConfig.GET("/system/config", r.projectConfigHandler.GetSystemScanConfig)    // 获取系统扫描配置
		adminScanConfig.PUT("/system/config", r.projectConfigHandler.UpdateSystemScanConfig) // 更新系统扫描配置

		// 全局工具管理
		adminScanConfig.POST("/tools/batch-install", r.scanToolHandler.BatchInstallScanTools)     // 批量安装扫描工具
		adminScanConfig.POST("/tools/batch-uninstall", r.scanToolHandler.BatchUninstallScanTools) // 批量卸载扫描工具
		adminScanConfig.GET("/tools/system-status", r.scanToolHandler.GetSystemToolStatus)        // 获取系统工具状态

		// 全局规则管理
		adminScanConfig.POST("/rules/batch-import", r.scanRuleHandler.BatchImportScanRules)   // 批量导入扫描规则
		adminScanConfig.POST("/rules/batch-enable", r.scanRuleHandler.BatchEnableScanRules)   // 批量启用扫描规则
		adminScanConfig.POST("/rules/batch-disable", r.scanRuleHandler.BatchDisableScanRules) // 批量禁用扫描规则

		// 系统监控和统计
		adminScanConfig.GET("/statistics", r.workflowHandler.GetSystemScanStatistics) // 获取系统扫描统计
		adminScanConfig.GET("/performance", r.workflowHandler.GetSystemPerformance)   // 获取系统性能指标
	}
}
