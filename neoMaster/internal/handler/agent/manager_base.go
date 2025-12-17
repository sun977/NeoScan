/**
 * Agent基础管理控制器
 * 作者: Sun977
 * 日期: 2025-11-07
 * 说明: 将与基础管理相关的 Handler 方法集中于此，包括：
 * - RegisterAgent（注册）
 * - GetAgentInfo（获取单个Agent信息）
 * - GetAgentList（获取Agent列表，含分页/过滤）
 * - UpdateAgentStatus（更新Agent状态）
 * - DeleteAgent（删除Agent）
 * 重构策略: 保持原有业务逻辑不变，仅文件拆分与注释增强，确保日志规范统一（LogBusinessOperation/LogBusinessError）。
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
)

// RegisterAgent Agent注册处理器
// 说明: 公开接口，解析请求体与校验后，调用 Service 完成注册，统一记录业务日志。
func (h *AgentHandler) RegisterAgent(c *gin.Context) {
	// 规范化客户端信息（在全流程统一使用）
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		logger.LogBusinessError(
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
		logger.LogBusinessError(
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
			Message: "Invalid JSON format",
			Error:   err.Error(),
		})
		return
	}

	// 验证必填字段
	if err := h.validateRegisterRequest(&req); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0, // AgentID - 在注册阶段还没有agent ID
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "register_agent",
				"option":     "validateRegisterRequest",
				"func_name":  "handler.agent.RegisterAgent",
				"user_agent": userAgent,
				"hostname":   req.Hostname,
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

	// 调用服务层注册Agent
	response, err := h.agentManagerService.RegisterAgent(&req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)

		// 检查是否是重复注册错误 - 修复错误检查逻辑
		if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		}

		logger.LogBusinessError(
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

	// 成功业务日志：使用 LogBusinessOperation 统一记录
	logger.LogBusinessOperation(
		"register_agent", // operation
		0,                // userID - 注册阶段暂无用户信息
		"",               // username - 未认证用户
		clientIP,
		XRequestID,
		"success",
		"Agent注册成功",
		map[string]interface{}{
			"func_name":  "handler.agent.RegisterAgent", // 具体路径的函数名
			"option":     "success",                     // 操作步骤（此处表示成功）
			"path":       pathUrl,                       // 请求URI路径
			"method":     "POST",                        // HTTP方法
			"user_agent": userAgent,                     // 客户端User-Agent
			"agent_id":   response.AgentID,              // 注册成功后返回的AgentID
			"hostname":   req.Hostname,                  // Agent主机名
		},
	)

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent registered successfully",
		Data:    response,
	})
}

// GetAgentInfo 根据ID获取Agent信息
// 说明: 校验路径参数，调用服务层获取信息，统一错误处理与日志记录。
func (h *AgentHandler) GetAgentInfo(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
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
	agentInfo, err := h.agentManagerService.GetAgentInfo(agentID)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
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

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"get_agent_info",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"获取Agent信息成功",
		map[string]interface{}{
			"func_name":  "handler.agent.GetAgentInfo",
			"option":     "success",
			"path":       pathUrl,
			"method":     "GET",
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

// GetAgentList 获取Agent列表
// 说明: 支持分页、关键字、标签、能力过滤，统一错误处理与日志记录。
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

	// 关键字过滤参数 - 支持对agent_id、hostname、ip_address的模糊查询
	req.Keyword = c.Query("keyword")

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

	// 功能模块过滤参数处理 (兼容)
	capabilitiesArray := c.QueryArray("capabilities")
	if len(capabilitiesArray) > 0 {
		if len(capabilitiesArray) > 1 {
			req.Capabilities = capabilitiesArray
		} else if len(capabilitiesArray) == 1 && strings.Contains(capabilitiesArray[0], ",") {
			req.Capabilities = strings.Split(capabilitiesArray[0], ",")
			for i, v := range req.Capabilities {
				req.Capabilities[i] = strings.TrimSpace(v)
			}
		} else {
			req.Capabilities = capabilitiesArray
		}
	}

	// 任务支持过滤参数处理 - 支持逗号分隔的值
	taskSupportArray := c.QueryArray("task_support")
	if len(taskSupportArray) > 0 {
		if len(taskSupportArray) > 1 {
			req.TaskSupport = taskSupportArray
		} else if len(taskSupportArray) == 1 && strings.Contains(taskSupportArray[0], ",") {
			req.TaskSupport = strings.Split(taskSupportArray[0], ",")
			for i, v := range req.TaskSupport {
				req.TaskSupport[i] = strings.TrimSpace(v)
			}
		} else {
			req.TaskSupport = taskSupportArray
		}
	}

	// 调用服务层获取Agent列表
	response, err := h.agentManagerService.GetAgentList(&req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
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

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"get_agent_list",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"获取Agent列表成功",
		map[string]interface{}{
			"func_name":  "handler.agent.GetAgentList",
			"option":     "success",
			"path":       pathUrl,
			"method":     "GET",
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

// UpdateAgentStatus 更新Agent状态
// 说明: 支持 PATCH/PUT 语义中的状态更新，进行校验后调用服务层，记录业务日志。
func (h *AgentHandler) UpdateAgentStatus(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
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
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PATCH",
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

	// 验证状态值是否有效
	validStatuses := []agentModel.AgentStatus{
		agentModel.AgentStatusOnline,
		agentModel.AgentStatusOffline,
		agentModel.AgentStatusException,
		agentModel.AgentStatusMaintenance,
	}
	isValidStatus := false
	for _, validStatus := range validStatuses {
		if req.Status == validStatus {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		logger.LogBusinessError(
			fmt.Errorf("invalid status: %s", req.Status),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PATCH",
			map[string]interface{}{
				"operation":  "update_agent_status",
				"option":     "statusValidation",
				"func_name":  "handler.agent.UpdateAgentStatus",
				"user_agent": userAgent,
				"agent_id":   agentID,
				"status":     string(req.Status),
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid status value",
			Error:   fmt.Sprintf("status '%s' is not valid", req.Status),
		})
		return
	}

	// 调用服务层更新Agent状态
	err := h.agentManagerService.UpdateAgentStatus(agentID, req.Status)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PATCH",
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

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"update_agent_status",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"更新Agent状态成功",
		map[string]interface{}{
			"func_name":  "handler.agent.UpdateAgentStatus",
			"option":     "success",
			"path":       pathUrl,
			"method":     "PUT",
			"user_agent": userAgent,
			"agent_id":   agentID,
			"status":     string(req.Status),
		},
	)

	c.JSON(http.StatusOK, system.APIResponse{
		Code:   http.StatusOK,
		Status: "success",
		Data: map[string]interface{}{
			"agent_id":   agentID,
			"new_status": req.Status,
		},
		Message: "Agent status updated successfully",
	})
}

// DeleteAgent 删除Agent
// 说明: 校验路径参数后，调用服务层删除 Agent，统一日志与响应格式。
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
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
	err := h.agentManagerService.DeleteAgent(agentID)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
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

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"delete_agent",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"Agent删除成功",
		map[string]interface{}{
			"func_name":  "handler.agent.DeleteAgent",
			"option":     "success",
			"path":       pathUrl,
			"method":     "DELETE",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent deleted successfully",
		Data: map[string]interface{}{
			"agent_id": agentID,
		},
	})
}
