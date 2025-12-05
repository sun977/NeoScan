package orchestrator

import (
	"fmt"
	orchestratorService "neomaster/internal/service/orchestrator"
	"net/http"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

// AgentTaskHandler 处理 Agent 任务相关的 HTTP 请求
// 属于 Orchestrator 模块，负责任务的下发与状态接收
type AgentTaskHandler struct {
	service orchestratorService.AgentTaskService
}

// NewAgentTaskHandler 创建 AgentTaskHandler 实例
func NewAgentTaskHandler(service orchestratorService.AgentTaskService) *AgentTaskHandler {
	return &AgentTaskHandler{
		service: service,
	}
}

// FetchTasks Agent 拉取任务接口
// 路由: GET /api/v1/orchestrator/agents/:agent_id/tasks
// 或保留原路由: GET /api/v1/agent/:id/tasks (由路由配置决定)
func (h *AgentTaskHandler) FetchTasks(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 支持从 path param "agent_id" 或 "id" 获取
	agentID := c.Param("agent_id")
	if agentID == "" {
		agentID = c.Param("id")
	}
	if agentID == "" {
		agentID = c.Query("agent_id")
	}

	if agentID == "" {
		logger.LogBusinessError(
			fmt.Errorf("agent_id is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation": "fetch_tasks",
				"option":    "paramValidation",
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "agent_id is required",
		})
		return
	}

	tasks, err := h.service.FetchTasks(c.Request.Context(), agentID)
	if err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation": "fetch_tasks",
				"option":    "service.FetchTasks",
				"agent_id":  agentID,
			},
		)
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to fetch tasks",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tasks fetched successfully",
		Data:    tasks,
	})
}

// UpdateTaskStatus 更新任务状态接口
// 路由: POST /api/v1/orchestrator/tasks/:task_id/status
func (h *AgentTaskHandler) UpdateTaskStatus(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "task_id is required",
		})
		return
	}

	var req struct {
		Status   string `json:"status" binding:"required"`
		Result   string `json:"result"`
		ErrorMsg string `json:"error_msg"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	err := h.service.UpdateTaskStatus(c.Request.Context(), taskID, req.Status, req.Result, req.ErrorMsg)
	if err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation": "update_task_status",
				"task_id":   taskID,
				"status":    req.Status,
			},
		)
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update task status",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Task status updated successfully",
	})
}
