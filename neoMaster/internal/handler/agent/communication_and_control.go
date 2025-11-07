/**
 * Agent通信与控制控制器
 * 作者: Sun977
 * 日期: 2025-11-07
 * 说明: 将与Agent通信与控制相关的 Handler 方法集中于此，目前包含：
 * - ProcessHeartbeat（处理Agent心跳）
 * 重构策略: 保持原有业务逻辑与返回格式不变，统一成功日志使用 LogBusinessOperation。
 */
package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

// ProcessHeartbeat 处理Agent心跳处理器
// 路由：POST /api/v1/agent/heartbeat
func (h *AgentHandler) ProcessHeartbeat(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 解析请求体
	var req agentModel.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "process_heartbeat",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.ProcessHeartbeat",
				"user_agent": userAgent,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid heartbeat request format",
			Error:   err.Error(),
		})
		return
	}

	// 验证必填字段
	if err := h.validateHeartbeatRequest(&req); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "process_heartbeat",
				"option":     "validateHeartbeatRequest",
				"func_name":  "handler.agent.ProcessHeartbeat",
				"user_agent": userAgent,
				"agent_id":   req.AgentID,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: err.Error(),
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层处理心跳
	response, err := h.agentMonitorService.ProcessHeartbeat(&req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":   "process_heartbeat",
				"option":      "agentService.ProcessHeartbeat",
				"func_name":   "handler.agent.ProcessHeartbeat",
				"user_agent":  userAgent,
				"agent_id":    req.AgentID,
				"status_code": statusCode,
			},
		)

		// 根据错误类型返回不同的消息
		var message string
		switch statusCode {
		case http.StatusNotFound:
			message = "Agent not found"
		default:
			message = "Failed to process heartbeat"
		}

		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: message,
			Error:   err.Error(),
		})
		return
	}

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"process_heartbeat", // operation
		0,                   // userID
		"",                  // username
		clientIP,
		XRequestID,
		"success",
		"处理Agent心跳成功",
		map[string]interface{}{
			"func_name":  "handler.agent.ProcessHeartbeat",
			"option":     "success",
			"path":       pathUrl,
			"method":     "POST",
			"user_agent": userAgent,
			"agent_id":   req.AgentID,
		},
	)

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Heartbeat processed successfully",
		Data:    response,
	})
}
