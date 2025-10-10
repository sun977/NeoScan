/**
 * 路由:健康检查路由
 * @author: sun977
 * @date: 2025.10.10
 * @description: 包含健康检查路由
 * @func:
 */

package router

import (
	"net/http"

	"neomaster/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

// setupHealthRoutes 设置健康检查路由
func (r *Router) setupHealthRoutes(api *gin.RouterGroup) {
	// 健康检查
	api.GET("/health", r.healthCheck)
	// 就绪检查
	api.GET("/ready", r.readinessCheck)
	// 存活检查
	api.GET("/live", r.livenessCheck)
}

// 健康检查处理器
func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": logger.NowFormatted(),
	})
}

// readinessCheck 就绪检查处理器
func (r *Router) readinessCheck(c *gin.Context) {
	// TODO: 检查依赖服务（数据库、Redis等）是否就绪
	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": logger.NowFormatted(),
	})
}

// livenessCheck 存活检查处理器
func (r *Router) livenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": logger.NowFormatted(),
	})
}
