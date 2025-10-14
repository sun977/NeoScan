/**
 * Agent处理器层:Agent HTTP请求处理
 * @author: Linus-style implementation
 * @date: 2025.10.14
 * @description: Agent控制器层，处理HTTP请求和响应，遵循RESTful API设计
 * @func: HTTP请求处理，遵循"好品味"原则 - 统一的错误处理，简洁的响应格式
 */
package agent

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	agentService "neomaster/internal/service/agent"
)

// AgentHandler Agent处理器
type AgentHandler struct {
	agentService agentService.AgentService // 修复：使用接口类型而不是指针
}

// NewAgentHandler 创建Agent处理器实例
func NewAgentHandler(agentService agentService.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
	}
}

// RegisterAgent Agent注册处理器
func (h *AgentHandler) RegisterAgent(c *gin.Context) {
	// 规范化客户端信息（在全流程统一使用）
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		logger.WithFields(logrus.Fields{
			"path":       "/api/v1/agent/register",
			"operation":  "register_agent",
			"option":     "contentTypeCheck",
			"func_name":  "handler.agent.agent_handler.RegisterAgent",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"error":      "missing Content-Type header",
		}).Error("Content-Type header缺失")
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Content-Type header is required",
			Error:   "missing Content-Type header",
		})
		return
	}

	// 解析请求体
	var req agentModel.RegisterAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithFields(logrus.Fields{
			"path":         "/api/v1/agent/register",
			"operation":    "register_agent",
			"option":       "ShouldBindJSON",
			"func_name":    "handler.agent.agent_handler.RegisterAgent",
			"client_ip":    clientIP,
			"user_agent":   userAgent,
			"request_id":   XRequestID,
			"content_type": contentType,
			"error":        err.Error(),
		}).Error("请求体解析失败")
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层处理业务逻辑
	response, err := h.agentService.RegisterAgent(&req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.WithFields(logrus.Fields{
			"path":        "/api/v1/agent/register",
			"operation":   "register_agent",
			"option":      "agentService.RegisterAgent",
			"func_name":   "handler.agent.agent_handler.RegisterAgent",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"hostname":    req.Hostname,
			"status_code": statusCode,
			"error":       err.Error(),
		}).Error("Agent注册失败")
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Agent registration failed",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.WithFields(logrus.Fields{
		"path":       "/api/v1/agent/register",
		"operation":  "register_agent",
		"option":     "success",
		"func_name":  "handler.agent.agent_handler.RegisterAgent",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": XRequestID,
		"agent_id":   response.AgentID,
		"hostname":   req.Hostname,
	}).Info("Agent注册成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent registered successfully",
		Data:    response,
	})
}

// GetAgentInfo 获取Agent信息处理器
func (h *AgentHandler) GetAgentInfo(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.WithFields(logrus.Fields{
			"path":       "/api/v1/agent/:id",
			"operation":  "get_agent_info",
			"option":     "paramValidation",
			"func_name":  "handler.agent.agent_handler.GetAgentInfo",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"error":      "agent ID is required",
		}).Error("Agent ID参数缺失")
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 调用服务层获取Agent信息
	agentInfo, err := h.agentService.GetAgentInfo(agentID)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.WithFields(logrus.Fields{
			"path":        "/api/v1/agent/:id",
			"operation":   "get_agent_info",
			"option":      "agentService.GetAgentInfo",
			"func_name":   "handler.agent.agent_handler.GetAgentInfo",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"agent_id":    agentID,
			"status_code": statusCode,
			"error":       err.Error(),
		}).Error("获取Agent信息失败")
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to get agent info",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.WithFields(logrus.Fields{
		"path":       "/api/v1/agent/:id",
		"operation":  "get_agent_info",
		"option":     "success",
		"func_name":  "handler.agent.agent_handler.GetAgentInfo",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": XRequestID,
		"agent_id":   agentID,
	}).Info("获取Agent信息成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent info retrieved successfully",
		Data:    agentInfo,
	})
}

// GetAgentList 获取Agent列表处理器
func (h *AgentHandler) GetAgentList(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	// 解析查询参数
	var req agentModel.GetAgentListRequest
	
	// 分页参数
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			req.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
			req.PageSize = pageSize
		}
	}
	
	// 过滤参数
	req.Status = agentModel.AgentStatus(c.Query("status"))
	req.Tags = c.QueryArray("tags")

	// 调用服务层获取Agent列表
	response, err := h.agentService.GetAgentList(&req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.WithFields(logrus.Fields{
			"path":        "/api/v1/agent/list",
			"operation":   "get_agent_list",
			"option":      "agentService.GetAgentList",
			"func_name":   "handler.agent.agent_handler.GetAgentList",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"status_code": statusCode,
			"error":       err.Error(),
		}).Error("获取Agent列表失败")
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to get agent list",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.WithFields(logrus.Fields{
		"path":       "/api/v1/agent/list",
		"operation":  "get_agent_list",
		"option":     "success",
		"func_name":  "handler.agent.agent_handler.GetAgentList",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": XRequestID,
		"total":      response.Pagination.Total,
	}).Info("获取Agent列表成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent list retrieved successfully",
		Data:    response,
	})
}

// UpdateAgentStatus 更新Agent状态处理器
func (h *AgentHandler) UpdateAgentStatus(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.WithFields(logrus.Fields{
			"path":       "/api/v1/agent/:id/status",
			"operation":  "update_agent_status",
			"option":     "paramValidation",
			"func_name":  "handler.agent.agent_handler.UpdateAgentStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"error":      "agent ID is required",
		}).Error("Agent ID参数缺失")
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 解析请求体
	var req agentModel.UpdateAgentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithFields(logrus.Fields{
			"path":       "/api/v1/agent/:id/status",
			"operation":  "update_agent_status",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.agent.agent_handler.UpdateAgentStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"agent_id":   agentID,
			"error":      err.Error(),
		}).Error("请求体解析失败")
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层更新Agent状态
	err := h.agentService.UpdateAgentStatus(agentID, req.Status)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.WithFields(logrus.Fields{
			"path":        "/api/v1/agent/:id/status",
			"operation":   "update_agent_status",
			"option":      "agentService.UpdateAgentStatus",
			"func_name":   "handler.agent.agent_handler.UpdateAgentStatus",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"agent_id":    agentID,
			"status":      string(req.Status),
			"status_code": statusCode,
			"error":       err.Error(),
		}).Error("更新Agent状态失败")
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to update agent status",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.WithFields(logrus.Fields{
		"path":       "/api/v1/agent/:id/status",
		"operation":  "update_agent_status",
		"option":     "success",
		"func_name":  "handler.agent.agent_handler.UpdateAgentStatus",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": XRequestID,
		"agent_id":   agentID,
		"status":     string(req.Status),
	}).Info("更新Agent状态成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent status updated successfully",
	})
}

// ProcessHeartbeat 处理Agent心跳处理器
func (h *AgentHandler) ProcessHeartbeat(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	// 解析请求体
	var req agentModel.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithFields(logrus.Fields{
			"path":       "/api/v1/agent/heartbeat",
			"operation":  "process_heartbeat",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.agent.agent_handler.ProcessHeartbeat",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"error":      err.Error(),
		}).Error("心跳请求体解析失败")
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid heartbeat request format",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层处理心跳
	response, err := h.agentService.ProcessHeartbeat(&req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.WithFields(logrus.Fields{
			"path":        "/api/v1/agent/heartbeat",
			"operation":   "process_heartbeat",
			"option":      "agentService.ProcessHeartbeat",
			"func_name":   "handler.agent.agent_handler.ProcessHeartbeat",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"agent_id":    req.AgentID,
			"status_code": statusCode,
			"error":       err.Error(),
		}).Error("处理Agent心跳失败")
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to process heartbeat",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.WithFields(logrus.Fields{
		"path":       "/api/v1/agent/heartbeat",
		"operation":  "process_heartbeat",
		"option":     "success",
		"func_name":  "handler.agent.agent_handler.ProcessHeartbeat",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": XRequestID,
		"agent_id":   req.AgentID,
	}).Info("处理Agent心跳成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Heartbeat processed successfully",
		Data:    response,
	})
}

// DeleteAgent 删除Agent处理器
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.WithFields(logrus.Fields{
			"path":       "/api/v1/agent/:id",
			"operation":  "delete_agent",
			"option":     "paramValidation",
			"func_name":  "handler.agent.agent_handler.DeleteAgent",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"error":      "agent ID is required",
		}).Error("Agent ID参数缺失")
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 调用服务层删除Agent
	err := h.agentService.DeleteAgent(agentID)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.WithFields(logrus.Fields{
			"path":        "/api/v1/agent/:id",
			"operation":   "delete_agent",
			"option":      "agentService.DeleteAgent",
			"func_name":   "handler.agent.agent_handler.DeleteAgent",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"agent_id":    agentID,
			"status_code": statusCode,
			"error":       err.Error(),
		}).Error("删除Agent失败")
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to delete agent",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.WithFields(logrus.Fields{
		"path":       "/api/v1/agent/:id",
		"operation":  "delete_agent",
		"option":     "success",
		"func_name":  "handler.agent.agent_handler.DeleteAgent",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": XRequestID,
		"agent_id":   agentID,
	}).Info("删除Agent成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent deleted successfully",
	})
}

// getErrorStatusCode 根据错误类型返回HTTP状态码
func (h *AgentHandler) getErrorStatusCode(err error) int {
	// 根据错误类型返回相应的HTTP状态码
	if err == nil {
		return http.StatusOK
	}
	
	// 可以根据具体的错误类型进行更精确的状态码映射
	errMsg := err.Error()
	if errMsg == "Agent不存在" || errMsg == "agent not found" {
		return http.StatusNotFound
	}
	
	// 默认返回内部服务器错误
	return http.StatusInternalServerError
}
