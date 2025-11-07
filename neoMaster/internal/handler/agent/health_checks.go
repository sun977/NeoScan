/**
 * Agent健康检查控制器（占位）
 * 作者: Sun977
 * 日期: 2025-11-07
 * 说明: 与Agent健康检查相关的 Handler 方法占位，未来承载健康状态检查与Ping接口。
 * - HealthCheckAgent
 * - PingAgent
 */
package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

// HealthCheckAgent 健康检查（占位实现）
func (h *AgentHandler) HealthCheckAgent(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	agentID := c.Param("id")

	logger.LogBusinessOperation(
		"health_check_agent",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"Agent健康检查占位返回",
		map[string]interface{}{
			"func_name":  "handler.agent.HealthCheckAgent",
			"option":     "placeholder",
			"path":       pathUrl,
			"method":     "GET",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Health check - placeholder",
		Data: map[string]interface{}{
			"agent_id": agentID,
			"healthy":  true,
		},
	})
}

// PingAgent Ping（占位实现）
func (h *AgentHandler) PingAgent(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	agentID := c.Param("id")

	logger.LogBusinessOperation(
		"ping_agent",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"Agent Ping占位返回",
		map[string]interface{}{
			"func_name":  "handler.agent.PingAgent",
			"option":     "placeholder",
			"path":       pathUrl,
			"method":     "GET",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Ping agent - placeholder",
		Data: map[string]interface{}{
			"agent_id": agentID,
			"pong":     true,
		},
	})
}
