/**
 * 路由:监控管理路由
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端监控管理路由，包含性能指标、健康检查、告警、日志等需要认证的路由
 * @func: 监控管理相关路由注册
 */
package router

import (
	"github.com/gin-gonic/gin"
	"neoagent/internal/handler/monitor"
	"neoagent/internal/pkg/logger"
)

// setupMonitorRoutes 设置监控路由
func setupMonitorRoutes(apiGroup *gin.RouterGroup) {
	logger.Info("注册监控路由开始")
	
	// 创建监控处理器实例
	monitorHandler := monitor.NewAgentMonitorHandler()
	
	// 监控路由组
	monitorGroup := apiGroup.Group("/monitor")
	{
		// 系统监控
		monitorGroup.GET("/system", monitorHandler.GetSystemMetrics)     // 获取系统指标
		monitorGroup.GET("/cpu", monitorHandler.GetCPUMetrics)           // 获取CPU指标
		monitorGroup.GET("/memory", monitorHandler.GetMemoryMetrics)     // 获取内存指标
		monitorGroup.GET("/disk", monitorHandler.GetDiskMetrics)         // 获取磁盘指标
		monitorGroup.GET("/network", monitorHandler.GetNetworkMetrics)   // 获取网络指标
		
		// 进程监控
		monitorGroup.GET("/processes", monitorHandler.GetProcessList)    // 获取进程列表
		monitorGroup.GET("/process/:pid", monitorHandler.GetProcessInfo) // 获取特定进程信息
		
		// 服务监控
		monitorGroup.GET("/services", monitorHandler.GetServiceStatus)   // 获取服务状态
		monitorGroup.GET("/health", monitorHandler.GetHealthStatus)      // 获取健康状态
		
		// 性能监控
		monitorGroup.GET("/performance", monitorHandler.GetPerformanceMetrics) // 获取性能指标
		monitorGroup.GET("/load", monitorHandler.GetSystemLoad)          // 获取系统负载
		
		// 日志监控
		monitorGroup.GET("/logs", monitorHandler.GetLogMetrics)          // 获取日志指标
		monitorGroup.GET("/alerts", monitorHandler.GetAlerts)            // 获取告警信息
		
		// 监控配置
		monitorGroup.POST("/config", monitorHandler.UpdateMonitorConfig) // 更新监控配置
		monitorGroup.GET("/config", monitorHandler.GetMonitorConfig)     // 获取监控配置
	}
	
	logger.Info("监控路由注册完成")
}
