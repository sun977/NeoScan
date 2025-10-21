/**
 * 路由:健康检查路由
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端健康检查路由，包含健康检查、存活检查等不需要认证的路由
 * @func: 健康检查相关路由注册和处理器
 */
package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"neoagent/internal/pkg/logger"
)

// setupHealthRoutes 设置健康检查路由
func (r *Router) setupHealthRoutes() {
	logger.Info("注册健康检查路由开始")
	// 健康检查路由（不需要认证）
	r.engine.GET("/health", r.handleHealth)
	r.engine.GET("/ping", r.handlePing)
	r.engine.GET("/version", r.handleVersion)
	logger.Info("健康检查路由注册完成")
}

// handleHealth 健康检查处理器
func (r *Router) handleHealth(c *gin.Context) {
	logger.Info("处理健康检查请求")

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": logger.NowFormatted(),
		"service":   "neoAgent",
		"version":   "1.0.0",
	})
}

// handlePing Ping处理器
func (r *Router) handlePing(c *gin.Context) {
	logger.Info("处理Ping请求")

	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": logger.NowFormatted(),
	})
}

// handleVersion 版本信息处理器
func (r *Router) handleVersion(c *gin.Context) {
	logger.Info("处理版本信息请求")

	c.JSON(http.StatusOK, gin.H{
		"service":    "neoAgent",
		"version":    "1.0.0",
		"build_time": time.Now().Format("2006-01-02 15:04:05"),
		"go_version": fmt.Sprintf("go%s", "1.21"),
		"timestamp":  logger.NowFormatted(),
	})
}
