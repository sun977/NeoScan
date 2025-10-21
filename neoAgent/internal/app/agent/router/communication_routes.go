/**
 * 路由:通信管理路由
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端通信管理路由，包含与Master的注册、认证、心跳、数据上报、配置同步、命令处理等路由
 * @func: 通信管理相关路由注册
 */
package router

import (
	"neoagent/internal/handler/communication"
	"neoagent/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

// setupCommunicationRoutes 设置通信管理路由
func setupCommunicationRoutes(engine *gin.Engine) {
	logger.Info("注册通信管理路由开始")

	// 创建通信处理器实例
	commHandler := communication.NewMasterCommunicationHandler()

	// 通信管理路由组
	commGroup := engine.Group("/agent/communication")
	{
		// Agent注册和认证
		commGroup.POST("/register", commHandler.RegisterToMaster)   // 向Master注册Agent
		commGroup.POST("/auth", commHandler.AuthenticateWithMaster) // 与Master进行认证

		// 心跳和状态同步
		commGroup.POST("/heartbeat", commHandler.SendHeartbeat) // 发送心跳到Master
		commGroup.POST("/sync-status", commHandler.SyncStatus)  // 同步状态到Master

		// 数据上报
		commGroup.POST("/report-metrics", commHandler.ReportMetrics) // 上报性能指标
		commGroup.POST("/report-task", commHandler.ReportTaskResult) // 上报任务结果
		commGroup.POST("/report-alert", commHandler.ReportAlert)     // 上报告警信息

		// 配置同步
		commGroup.POST("/sync-config", commHandler.SyncConfig)   // 从Master同步配置
		commGroup.POST("/apply-config", commHandler.ApplyConfig) // 应用Master下发的配置

		// 命令接收和响应
		commGroup.POST("/receive-command", commHandler.ReceiveCommand)       // 接收Master命令
		commGroup.POST("/command-response", commHandler.SendCommandResponse) // 发送命令执行结果

		// 连接管理
		commGroup.GET("/check-connection", commHandler.CheckConnection) // 检查与Master的连接
		commGroup.POST("/reconnect", commHandler.ReconnectToMaster)     // 重连到Master
	}

	logger.Info("通信管理路由注册完成")
}
