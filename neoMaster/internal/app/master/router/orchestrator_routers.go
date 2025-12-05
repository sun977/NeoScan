/**
 * 路由:扫描编排器路由
 * @author: sun977
 * @date: 2025.12.04
 * @description: 扫描编排器路由模块
 * @func: 注册 Project, Workflow, ScanStage, ScanToolTemplate 相关路由
 */
package router

import "github.com/gin-gonic/gin"

func (r *Router) setupOrchestratorRoutes(v1 *gin.RouterGroup) {
	orchestratorGroup := v1.Group("/orchestrator")
	// 使用 JWT 中间件进行认证
	if r.middlewareManager != nil {
		orchestratorGroup.Use(r.middlewareManager.GinJWTAuthMiddleware())
		orchestratorGroup.Use(r.middlewareManager.GinUserActiveMiddleware())
	}

	// 1. 项目管理 (Project Management)
	projects := orchestratorGroup.Group("/projects")
	{
		projects.POST("", r.projectHandler.CreateProject)
		projects.GET("", r.projectHandler.ListProjects)
		projects.GET("/:id", r.projectHandler.GetProject)
		projects.PUT("/:id", r.projectHandler.UpdateProject)
		projects.DELETE("/:id", r.projectHandler.DeleteProject)

		// 项目关联工作流
		projects.POST("/:id/workflows", r.projectHandler.AddWorkflow)
		projects.DELETE("/:id/workflows/:workflow_id", r.projectHandler.RemoveWorkflow)
		projects.GET("/:id/workflows", r.projectHandler.GetProjectWorkflows)
	}

	// 2. 工作流管理 (Workflow Management)
	workflows := orchestratorGroup.Group("/workflows")
	{
		workflows.POST("", r.workflowHandler.CreateWorkflow)
		workflows.GET("", r.workflowHandler.ListWorkflows)
		workflows.GET("/:id", r.workflowHandler.GetWorkflow)
		workflows.PUT("/:id", r.workflowHandler.UpdateWorkflow)
		workflows.DELETE("/:id", r.workflowHandler.DeleteWorkflow)
	}

	// 3. 扫描阶段管理 (Scan Stage Management)
	stages := orchestratorGroup.Group("/stages")
	{
		stages.POST("", r.scanStageHandler.CreateStage)
		stages.GET("", r.scanStageHandler.ListStages)
		stages.GET("/:id", r.scanStageHandler.GetStage)
		stages.PUT("/:id", r.scanStageHandler.UpdateStage)
		stages.DELETE("/:id", r.scanStageHandler.DeleteStage)
	}

	// 4. 工具模板管理 (Tool Template Management)
	templates := orchestratorGroup.Group("/tool-templates")
	{
		templates.POST("", r.scanToolTemplateHandler.CreateTemplate)
		templates.GET("", r.scanToolTemplateHandler.ListTemplates)
		templates.GET("/:id", r.scanToolTemplateHandler.GetTemplate)
		templates.PUT("/:id", r.scanToolTemplateHandler.UpdateTemplate)
		templates.DELETE("/:id", r.scanToolTemplateHandler.DeleteTemplate)
	}

	// 5. Agent 任务管理 (Agent Task Management)
	// 迁移至 Orchestrator 路径下: /orchestrator/agent/...
	agentTaskGroup := orchestratorGroup.Group("/agent")
	{
		agentTaskGroup.GET("/:id/tasks", r.agentTaskHandler.FetchTasks)                        // 获取Agent当前任务
		agentTaskGroup.POST("/:id/tasks/:task_id/status", r.agentTaskHandler.UpdateTaskStatus) // 更新任务状态 [Agent端上报任务状态]
	}

	// ============== Agent任务管理路由（🔴 需要Agent端配合实现 - Agent端执行任务） ====================
	// 	agentManageGroup.GET("/:id/tasks", r.agentHandler.FetchTasks)                        // 🔴 获取Agent当前任务 [需要Agent端返回正在执行的任务状态]
	// 	agentManageGroup.POST("/:id/tasks/:task_id/status", r.agentHandler.UpdateTaskStatus) // 🔴 更新任务状态 [Agent端上报任务状态]
	// 	agentManageGroup.POST("/:id/tasks", r.agentCreateTaskPlaceholder)                    // 🔴 分配任务给Agent [需要Master->Agent通信，下发扫描任务]
	// 	agentManageGroup.GET("/:id/tasks/:task_id", r.agentGetTaskPlaceholder)               // 🔴 获取任务执行状态 [需要Agent端返回任务执行进度和结果]
	// 	agentManageGroup.DELETE("/:id/tasks/:task_id", r.agentDeleteTaskPlaceholder)         // 🔴 取消Agent任务 [需要Master->Agent通信，取消正在执行的任务]
}
