/**
 * Agent处理器层:Agent HTTP请求处理
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent控制器层，处理HTTP请求和响应，遵循RESTful API设计
 * @func: HTTP请求处理，遵循"好品味"原则 - 统一的错误处理，简洁的响应格式
 */
package agent

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

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
	pathUrl := c.Request.URL.String()

	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		logger.LogError(
			fmt.Errorf("missing Content-Type header"),
			XRequestID,
			0, // AgentID - 在注册阶段还没有agent ID
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "register_agent",
				"option":     "contentTypeCheck",
				"func_name":  "handler.agent.RegisterAgent",
				"user_agent": userAgent,
			},
		)
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
		logger.LogError(
			err,
			XRequestID,
			0, // AgentID - 在注册阶段还没有agent ID
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":    "register_agent",
				"option":       "ShouldBindJSON",
				"func_name":    "handler.agent.RegisterAgent",
				"user_agent":   userAgent,
				"content_type": contentType,
			},
		)
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
		logger.LogError(
			err,
			XRequestID,
			0, // AgentID - 在注册阶段还没有agent ID
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":   "register_agent",
				"option":      "agentService.RegisterAgent",
				"func_name":   "handler.agent.RegisterAgent",
				"user_agent":  userAgent,
				"hostname":    req.Hostname,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Agent registration failed",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.LogInfo(
		"Agent注册成功",
		XRequestID,
		0, // AgentID - 在注册阶段还没有agent ID
		clientIP,
		pathUrl,
		"POST",
		map[string]interface{}{
			"operation":  "register_agent",
			"option":     "success",
			"func_name":  "handler.agent.RegisterAgent",
			"user_agent": userAgent,
			"agent_id":   response.AgentID,
			"hostname":   req.Hostname,
		},
	)

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
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":  "get_agent_info",
				"option":     "paramValidation",
				"func_name":  "handler.agent.GetAgentInfo",
				"user_agent": userAgent,
			},
		)
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
		logger.LogError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":   "get_agent_info",
				"option":      "agentService.GetAgentInfo",
				"func_name":   "handler.agent.GetAgentInfo",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to get agent info",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.LogInfo(
		"获取Agent信息成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"GET",
		map[string]interface{}{
			"operation":  "get_agent_info",
			"option":     "success",
			"func_name":  "handler.agent.GetAgentInfo",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

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
	pathUrl := c.Request.URL.String()

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

	// 过滤参数 status: offline / online
	req.Status = agentModel.AgentStatus(c.Query("status"))

	// 标签过滤参数处理 - 支持逗号分隔的标签值
	// 例如: tags=2,7 或 tags=2&tags=7 两种格式都支持
	tagsArray := c.QueryArray("tags")
	if len(tagsArray) > 1 {
		// 处理多个tags参数: tags=2&tags=7
		req.Tags = tagsArray
	} else if len(tagsArray) == 1 && strings.Contains(tagsArray[0], ",") {
		// 处理逗号分隔的标签值: tags=2,7
		req.Tags = strings.Split(tagsArray[0], ",")
		// 去除空白字符
		for i, tag := range req.Tags {
			req.Tags[i] = strings.TrimSpace(tag)
		}
	} else if len(tagsArray) == 1 {
		// 单个标签: tags=2
		req.Tags = tagsArray
	}

	// 功能模块过滤参数处理 - 支持逗号分隔的功能模块值
	// 例如: capabilities=1,2 或 capabilities=1&capabilities=2 两种格式都支持
	capabilitiesArray := c.QueryArray("capabilities")
	if len(capabilitiesArray) > 1 {
		// 处理多个capabilities参数: capabilities=1&capabilities=2
		req.Capabilities = capabilitiesArray
	} else if len(capabilitiesArray) == 1 && strings.Contains(capabilitiesArray[0], ",") {
		// 处理逗号分隔的功能模块值: capabilities=1,2
		req.Capabilities = strings.Split(capabilitiesArray[0], ",")
		// 去除空白字符
		for i, capability := range req.Capabilities {
			req.Capabilities[i] = strings.TrimSpace(capability)
		}
	} else if len(capabilitiesArray) == 1 {
		// 单个功能模块: capabilities=scan
		req.Capabilities = capabilitiesArray
	}

	// 调用服务层获取Agent列表
	response, err := h.agentService.GetAgentList(&req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":   "get_agent_list",
				"option":      "agentService.GetAgentList",
				"func_name":   "handler.agent.GetAgentList",
				"user_agent":  userAgent,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to get agent list",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.LogInfo(
		"获取Agent列表成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"GET",
		map[string]interface{}{
			"operation":  "get_agent_list",
			"option":     "success",
			"func_name":  "handler.agent.GetAgentList",
			"user_agent": userAgent,
			"total":      response.Pagination.Total,
		},
	)

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
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":  "update_agent_status",
				"option":     "paramValidation",
				"func_name":  "handler.agent.UpdateAgentStatus",
				"user_agent": userAgent,
			},
		)
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
		logger.LogError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":  "update_agent_status",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.UpdateAgentStatus",
				"user_agent": userAgent,
				"agent_id":   agentID,
			},
		)
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
		logger.LogError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":   "update_agent_status",
				"option":      "agentService.UpdateAgentStatus",
				"func_name":   "handler.agent.UpdateAgentStatus",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status":      string(req.Status),
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to update agent status",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.LogInfo(
		"更新Agent状态成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"PUT",
		map[string]interface{}{
			"operation":  "update_agent_status",
			"option":     "success",
			"func_name":  "handler.agent.UpdateAgentStatus",
			"user_agent": userAgent,
			"agent_id":   agentID,
			"status":     string(req.Status),
		},
	)

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
	pathUrl := c.Request.URL.String()

	// 解析请求体
	var req agentModel.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogError(
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

	// 调用服务层处理心跳
	response, err := h.agentService.ProcessHeartbeat(&req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogError(
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
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to process heartbeat",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.LogInfo(
		"处理Agent心跳成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"POST",
		map[string]interface{}{
			"operation":  "process_heartbeat",
			"option":     "success",
			"func_name":  "handler.agent.ProcessHeartbeat",
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

// 注意：ListAgents 和 UpdateAgent 方法已删除，因为它们在服务层接口中不存在
// 如需要这些功能，请先在服务层接口中定义相应的方法

// DeleteAgent 删除Agent处理器
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"DELETE",
			map[string]interface{}{
				"operation":  "delete_agent",
				"option":     "paramValidation",
				"func_name":  "handler.agent.DeleteAgent",
				"user_agent": userAgent,
			},
		)
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
		logger.LogError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"DELETE",
			map[string]interface{}{
				"operation":   "delete_agent",
				"option":      "agentService.DeleteAgent",
				"func_name":   "handler.agent.DeleteAgent",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to delete agent",
			Error:   err.Error(),
		})
		return
	}

	// 成功响应
	logger.LogInfo(
		"Agent删除成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"DELETE",
		map[string]interface{}{
			"operation":  "delete_agent",
			"option":     "success",
			"func_name":  "handler.agent.DeleteAgent",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

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
