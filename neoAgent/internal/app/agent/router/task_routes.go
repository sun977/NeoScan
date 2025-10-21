/**
 * 路由:任务管理路由
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端任务管理路由，包含任务创建、查询、控制等需要认证的路由
 * @func: 任务管理相关路由注册
 */
package router

import (
	"github.com/gin-gonic/gin"
	"neoagent/internal/handler/task"
	"neoagent/internal/pkg/logger"
)

// setupTaskRoutes 设置任务管理路由
func setupTaskRoutes(apiGroup *gin.RouterGroup) {
	logger.Info("注册任务管理路由开始")
	
	// 创建任务处理器实例
	taskHandler := task.NewAgentTaskHandler()
	
	// 任务管理路由组
	taskGroup := apiGroup.Group("/task")
	{
		// 任务生命周期管理
		taskGroup.POST("/create", taskHandler.CreateTask)           // 创建任务
		taskGroup.POST("/start", taskHandler.StartTask)             // 启动任务
		taskGroup.POST("/pause", taskHandler.PauseTask)             // 暂停任务
		taskGroup.POST("/resume", taskHandler.ResumeTask)           // 恢复任务
		taskGroup.POST("/stop", taskHandler.StopTask)               // 停止任务
		taskGroup.POST("/cancel", taskHandler.CancelTask)           // 取消任务
		
		// 任务查询和状态
		taskGroup.GET("/list", taskHandler.ListTasks)               // 获取任务列表
		taskGroup.GET("/:id", taskHandler.GetTask)                  // 获取任务详情
		taskGroup.GET("/:id/status", taskHandler.GetTaskStatus)     // 获取任务状态
		taskGroup.GET("/:id/progress", taskHandler.GetTaskProgress) // 获取任务进度
		taskGroup.GET("/:id/result", taskHandler.GetTaskResult)     // 获取任务结果
		taskGroup.GET("/:id/logs", taskHandler.GetTaskLogs)         // 获取任务日志
		
		// 任务配置和更新
		taskGroup.PUT("/:id/config", taskHandler.UpdateTaskConfig)  // 更新任务配置
		taskGroup.PUT("/:id/priority", taskHandler.UpdateTaskPriority) // 更新任务优先级
		
		// 任务队列管理
		taskGroup.GET("/queue", taskHandler.GetTaskQueue)           // 获取任务队列状态
		taskGroup.POST("/queue/clear", taskHandler.ClearTaskQueue)  // 清空任务队列
		
		// 任务统计和监控
		taskGroup.GET("/stats", taskHandler.GetTaskStats)           // 获取任务统计信息
		taskGroup.GET("/metrics", taskHandler.GetTaskMetrics)       // 获取任务性能指标
	}
	
	logger.Info("任务管理路由注册完成")
}
