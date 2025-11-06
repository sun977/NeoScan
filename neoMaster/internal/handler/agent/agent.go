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
	"net"
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
	agentManagerService agentService.AgentManagerService // Agent管理服务（包含分组功能）
	agentMonitorService agentService.AgentMonitorService // Agent监控服务
	agentConfigService  agentService.AgentConfigService  // Agent配置服务
	agentTaskService    agentService.AgentTaskService    // Agent任务服务
}

// NewAgentHandler 创建Agent处理器实例
func NewAgentHandler(
	agentManagerService agentService.AgentManagerService,
	agentMonitorService agentService.AgentMonitorService,
	agentConfigService agentService.AgentConfigService,
	agentTaskService agentService.AgentTaskService,
) *AgentHandler {
	return &AgentHandler{
		agentManagerService: agentManagerService,
		agentMonitorService: agentMonitorService,
		agentConfigService:  agentConfigService,
		agentTaskService:    agentTaskService,
	}
}

// validateRegisterRequest 验证Agent注册请求参数
func (h *AgentHandler) validateRegisterRequest(req *agentModel.RegisterAgentRequest) error {
	if req.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	// 验证hostname长度
	if len(req.Hostname) > 255 {
		return fmt.Errorf("hostname too long")
	}
	if req.IPAddress == "" {
		return fmt.Errorf("ip_address is required")
	}
	// 验证IP地址格式
	if net.ParseIP(req.IPAddress) == nil {
		return fmt.Errorf("invalid ip_address format")
	}
	if req.Port <= 0 || req.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if req.Version == "" {
		return fmt.Errorf("version is required")
	}
	// 验证version长度
	if len(req.Version) > 50 {
		return fmt.Errorf("version too long")
	}
	if req.OS == "" {
		return fmt.Errorf("os is required")
	}
	if req.Arch == "" {
		return fmt.Errorf("arch is required")
	}
	// 验证CPU核心数不能为负数或零
	if req.CPUCores <= 0 {
		return fmt.Errorf("invalid CPU cores")
	}
	// 验证memory_total不能为负数
	if req.MemoryTotal < 0 {
		return fmt.Errorf("invalid memory total")
	}
	// 验证disk_total不能为负数
	if req.DiskTotal < 0 {
		return fmt.Errorf("invalid disk total")
	}
	// 验证capabilities不能为空
	if len(req.Capabilities) == 0 {
		return fmt.Errorf("at least one capability is required")
	}
	// 验证capabilities包含有效值(根据CapabilityID验证) - 委托Service层处理业务逻辑
	for _, capabilityID := range req.Capabilities {
		if !h.agentManagerService.IsValidCapabilityId(capabilityID) {
			return fmt.Errorf("invalid capability ID: %s", capabilityID)
		}
	}
	return nil
}

// validateHeartbeatRequest 验证Agent心跳请求参数
func (h *AgentHandler) validateHeartbeatRequest(req *agentModel.HeartbeatRequest) error {
	if req.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if req.Status == "" {
		return fmt.Errorf("status is required")
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
		return fmt.Errorf("invalid status")
	}

	// 验证性能指标数据（如果提供）
	if req.Metrics != nil {
		if req.Metrics.CPUUsage < 0 || req.Metrics.CPUUsage > 100 {
			return fmt.Errorf("invalid CPU usage")
		}
		if req.Metrics.MemoryUsage < 0 || req.Metrics.MemoryUsage > 100 {
			return fmt.Errorf("invalid memory usage")
		}
		if req.Metrics.DiskUsage < 0 || req.Metrics.DiskUsage > 100 {
			return fmt.Errorf("invalid disk usage")
		}
		if req.Metrics.NetworkBytesSent < 0 {
			return fmt.Errorf("invalid network bytes sent")
		}
		if req.Metrics.NetworkBytesRecv < 0 {
			return fmt.Errorf("invalid network bytes received")
		}
		if req.Metrics.RunningTasks < 0 {
			return fmt.Errorf("invalid running tasks")
		}
		if req.Metrics.CompletedTasks < 0 {
			return fmt.Errorf("invalid completed tasks")
		}
		if req.Metrics.FailedTasks < 0 {
			return fmt.Errorf("invalid failed tasks")
		}
	}
	return nil
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
// 支持分页、status 状态过滤、keyword 关键字模糊查询、tags 标签过滤、capabilities 功能模块过滤
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
		Code:   http.StatusOK,
		Status: "success",
		Data: map[string]interface{}{
			"agent_id":   agentID,
			"new_status": req.Status,
		},
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
		if statusCode == http.StatusNotFound {
			message = "Agent not found"
		} else {
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
		Data: map[string]interface{}{
			"agent_id": agentID,
		},
	})
}

// GetAgentMetrics 获取指定Agent的最新性能快照（只读，来自Master端数据库 agent_metrics 表）
// 路由：GET /api/v1/agent/:id/metrics
// 说明：遵循项目分层规范，Handler → Service → Repository → DB；统一日志和错误返回格式
func (h *AgentHandler) GetAgentMetrics(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID并校验
	agentID := c.Param("id")
	if agentID == "" {
		// 按项目日志规范记录业务错误
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":  "get_agent_metrics",
				"option":     "paramValidation",
				"func_name":  "handler.agent.GetAgentMetrics",
				"user_agent": userAgent,
			},
		)
		// 返回统一错误响应
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 调用服务层，从数据库获取该Agent的最新性能快照
	metrics, err := h.agentMonitorService.GetAgentMetricsFromDB(agentID)
	if err != nil {
		// 映射错误为HTTP状态码
		statusCode := h.getErrorStatusCode(err)

		// 记录业务错误日志，包含关键字段
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":   "get_agent_metrics",
				"option":      "agentMonitorService.GetAgentMetricsFromDB",
				"func_name":   "handler.agent.GetAgentMetrics",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)

		// 针对404返回更清晰的消息，其它情况统一失败描述
		message := "Failed to get agent metrics"
		if statusCode == http.StatusNotFound {
			message = "Agent metrics not found"
		} else if statusCode == http.StatusBadRequest {
			message = "Invalid request"
		}

		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: message,
			Error:   err.Error(),
		})
		return
	}

	// 成功日志
	logger.LogInfo(
		"获取Agent性能快照成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"GET",
		map[string]interface{}{
			"operation":  "get_agent_metrics",
			"option":     "success",
			"func_name":  "handler.agent.GetAgentMetrics",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	// 返回统一成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent metrics retrieved successfully",
		Data:    metrics,
	})
}

// GetAgentListAllMetrics 获取所有Agent的最新性能快照列表（只读，来自Master端数据库 agent_metrics 表）
// 路由：GET /api/v1/agent/metrics
func (h *AgentHandler) GetAgentListAllMetrics(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 解析分页参数（保持与 GetAgentList 中的解析风格一致）
	// page 和 page_size 为可选参数，未提供时使用合理默认值
	page := 1
	pageSize := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	// 解析过滤参数：work_status、scan_type、keyword（agent_id关键词）
	var workStatusPtr *agentModel.AgentWorkStatus
	if wsStr := c.Query("work_status"); wsStr != "" {
		ws := agentModel.AgentWorkStatus(wsStr)
		workStatusPtr = &ws
	}

	var scanTypePtr *agentModel.AgentScanType
	if stStr := c.Query("scan_type"); stStr != "" {
		st := agentModel.AgentScanType(stStr)
		scanTypePtr = &st
	}

	var keywordPtr *string
	if kw := c.Query("keyword"); kw != "" {
		keywordPtr = &kw
	}

	// 调用服务层，分页获取所有Agent的最新性能快照（仓储层SQL分页 + 过滤条件）
	list, total, err := h.agentMonitorService.GetAgentListAllMetricsFromDB(page, pageSize, workStatusPtr, scanTypePtr, keywordPtr)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)

		// 记录业务错误日志
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":   "get_agent_list_all_metrics",
				"option":      "agentMonitorService.GetAgentListAllMetricsFromDB",
				"func_name":   "handler.agent.GetAgentListAllMetrics",
				"user_agent":  userAgent,
				"status_code": statusCode,
				"page":        page,
				"page_size":   pageSize,
				"work_status": func() interface{} {
					if workStatusPtr != nil {
						return *workStatusPtr
					}
					return nil
				}(),
				"scan_type": func() interface{} {
					if scanTypePtr != nil {
						return *scanTypePtr
					}
					return nil
				}(),
				"keyword": func() interface{} {
					if keywordPtr != nil {
						return *keywordPtr
					}
					return nil
				}(),
			},
		)

		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to get all agents metrics",
			Error:   err.Error(),
		})
		return
	}

	// Service 已进行分页查询，这里直接使用返回的当前页数据
	pageItems := list

	// 计算总页数（注意类型转换，total 为 int64）
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	// 构造分页响应结构
	resp := &agentModel.AgentMetricsListResponse{
		Metrics: pageItems,
		Pagination: &agentModel.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	// 成功日志（补充分页信息）
	logger.LogInfo(
		"获取所有Agent性能快照列表成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"GET",
		map[string]interface{}{
			"operation":  "get_agent_list_all_metrics",
			"option":     "success",
			"func_name":  "handler.agent.GetAgentListAllMetrics",
			"user_agent": userAgent,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"work_status": func() interface{} {
				if workStatusPtr != nil {
					return *workStatusPtr
				}
				return nil
			}(),
			"scan_type": func() interface{} {
				if scanTypePtr != nil {
					return *scanTypePtr
				}
				return nil
			}(),
			"keyword": func() interface{} {
				if keywordPtr != nil {
					return *keywordPtr
				}
				return nil
			}(),
		},
	)

	// 返回统一成功响应（包含分页）
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "All agents metrics retrieved successfully",
		Data:    resp,
	})
}

// CreateAgentMetrics 创建/上报Agent性能指标快照（Master端数据库插入）
// 路由：POST /api/v1/agent/:id/metrics
// 设计：
// - 严格遵守分层：Handler → Service → Repository → DB；不直接操作数据库
// - 统一日志与错误返回；校验基础数值边界，防止脏数据入库
func (h *AgentHandler) CreateAgentMetrics(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID并校验
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "create_agent_metrics",
				"option":     "paramValidation",
				"func_name":  "handler.agent.CreateAgentMetrics",
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

	// 解析请求体为 AgentMetrics
	var metrics agentModel.AgentMetrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "create_agent_metrics",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.CreateAgentMetrics",
				"user_agent": userAgent,
				"agent_id":   agentID,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid metrics request format",
			Error:   err.Error(),
		})
		return
	}

	// 统一以路径中的 agentID 为准，避免用户伪造
	metrics.AgentID = agentID

	// 基础字段校验（防御性编程）
	if metrics.CPUUsage < 0 || metrics.CPUUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid CPU usage", Error: "invalid CPU usage"})
		return
	}
	if metrics.MemoryUsage < 0 || metrics.MemoryUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid memory usage", Error: "invalid memory usage"})
		return
	}
	if metrics.DiskUsage < 0 || metrics.DiskUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid disk usage", Error: "invalid disk usage"})
		return
	}
	if metrics.NetworkBytesSent < 0 || metrics.NetworkBytesRecv < 0 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid network bytes", Error: "invalid network bytes"})
		return
	}
	if metrics.RunningTasks < 0 || metrics.CompletedTasks < 0 || metrics.FailedTasks < 0 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid task counters", Error: "invalid task counters"})
		return
	}
	// WorkStatus 枚举校验（允许空值按默认）
	if metrics.WorkStatus != "" {
		valid := metrics.WorkStatus == agentModel.AgentWorkStatusIdle ||
			metrics.WorkStatus == agentModel.AgentWorkStatusWorking ||
			metrics.WorkStatus == agentModel.AgentWorkStatusException
		if !valid {
			c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid work status", Error: fmt.Sprintf("invalid work_status: %s", metrics.WorkStatus)})
			return
		}
	}
	// ScanType 可选校验：若提供，需为非空字符串（业务允许自定义类型，暂不强校验）
	// 时间戳：若未提供则自动补齐当前时间
	if metrics.Timestamp.IsZero() {
		metrics.UpdateTimestamp()
	}

	// 调用服务层创建指标
	if err := h.agentMonitorService.CreateAgentMetrics(agentID, &metrics); err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":   "create_agent_metrics",
				"option":      "agentMonitorService.CreateAgentMetrics",
				"func_name":   "handler.agent.CreateAgentMetrics",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to create agent metrics",
			Error:   err.Error(),
		})
		return
	}

	// 成功日志与响应
	logger.LogInfo(
		"创建Agent性能指标成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"POST",
		map[string]interface{}{
			"operation":  "create_agent_metrics",
			"option":     "success",
			"func_name":  "handler.agent.CreateAgentMetrics",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Agent metrics created successfully",
		Data: map[string]interface{}{
			"agent_id":  agentID,
			"timestamp": metrics.Timestamp,
		},
	})
}

// UpdateAgentMetrics 更新Agent性能指标快照（Master端数据库更新）
// 路由：PUT /api/v1/agent/:id/metrics
func (h *AgentHandler) UpdateAgentMetrics(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID并校验
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
				"operation":  "update_agent_metrics",
				"option":     "paramValidation",
				"func_name":  "handler.agent.UpdateAgentMetrics",
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

	// 解析请求体为 AgentMetrics
	var metrics agentModel.AgentMetrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":  "update_agent_metrics",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.UpdateAgentMetrics",
				"user_agent": userAgent,
				"agent_id":   agentID,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid metrics request format",
			Error:   err.Error(),
		})
		return
	}

	// 统一以路径中的 agentID 为准
	metrics.AgentID = agentID

	// 基础字段校验（与创建一致）
	if metrics.CPUUsage < 0 || metrics.CPUUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid CPU usage", Error: "invalid CPU usage"})
		return
	}
	if metrics.MemoryUsage < 0 || metrics.MemoryUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid memory usage", Error: "invalid memory usage"})
		return
	}
	if metrics.DiskUsage < 0 || metrics.DiskUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid disk usage", Error: "invalid disk usage"})
		return
	}
	if metrics.NetworkBytesSent < 0 || metrics.NetworkBytesRecv < 0 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid network bytes", Error: "invalid network bytes"})
		return
	}
	if metrics.RunningTasks < 0 || metrics.CompletedTasks < 0 || metrics.FailedTasks < 0 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid task counters", Error: "invalid task counters"})
		return
	}
	if metrics.WorkStatus != "" {
		valid := metrics.WorkStatus == agentModel.AgentWorkStatusIdle ||
			metrics.WorkStatus == agentModel.AgentWorkStatusWorking ||
			metrics.WorkStatus == agentModel.AgentWorkStatusException
		if !valid {
			c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid work status", Error: fmt.Sprintf("invalid work_status: %s", metrics.WorkStatus)})
			return
		}
	}
	if metrics.Timestamp.IsZero() {
		metrics.UpdateTimestamp()
	}

	// 调用服务层更新指标
	if err := h.agentMonitorService.UpdateAgentMetrics(agentID, &metrics); err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":   "update_agent_metrics",
				"option":      "agentMonitorService.UpdateAgentMetrics",
				"func_name":   "handler.agent.UpdateAgentMetrics",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)
		// 针对404返回更清晰的消息，其它情况统一失败描述
		message := "Failed to update agent metrics"
		if statusCode == http.StatusNotFound {
			message = "Agent metrics not found"
		}
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: message,
			Error:   err.Error(),
		})
		return
	}

	// 成功日志与响应
	logger.LogInfo(
		"更新Agent性能指标成功",
		XRequestID,
		0,
		clientIP,
		pathUrl,
		"PUT",
		map[string]interface{}{
			"operation":  "update_agent_metrics",
			"option":     "success",
			"func_name":  "handler.agent.UpdateAgentMetrics",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent metrics updated successfully",
		Data: map[string]interface{}{
			"agent_id":  agentID,
			"timestamp": metrics.Timestamp,
		},
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
	// 统一处理“未找到”类错误
	if errMsg == "Agent not found" || errMsg == "agent not found" || errMsg == "Agent不存在" ||
		strings.Contains(errMsg, "未找到") || strings.Contains(errMsg, "not found") {
		return http.StatusNotFound
	}

	// 参数缺失或不合法
	if strings.Contains(errMsg, "不能为空") || strings.Contains(errMsg, "is required") || strings.Contains(errMsg, "invalid") {
		return http.StatusBadRequest
	}

	// 默认返回内部服务器错误
	return http.StatusInternalServerError
}
