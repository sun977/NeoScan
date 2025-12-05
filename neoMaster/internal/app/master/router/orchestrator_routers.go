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
}
