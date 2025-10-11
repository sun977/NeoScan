/*
 * 工作流处理器：工作流HTTP接口处理
 * @author: Linus-inspired AI
 * @date: 2025.01.27
 * @description: 处理工作流相关的HTTP请求和响应
 * @func:
 * 1.工作流CRUD操作接口
 * 2.工作流执行和控制接口
 * 3.工作流状态监控接口
 * 4.工作流日志和统计接口
 */

//  核心HTTP接口:
//  	POST   /api/v1/scan-config/workflows - 创建工作流配置
//  	GET    /api/v1/scan-config/workflows/:id - 获取工作流配置详情
//  	PUT    /api/v1/scan-config/workflows/:id - 更新工作流配置
//  	DELETE /api/v1/scan-config/workflows/:id - 删除工作流配置
//  	GET    /api/v1/scan-config/workflows - 获取工作流配置列表
//  执行控制接口:
//  	POST   /api/v1/scan-config/workflows/:id/execute - 执行工作流
//  	POST   /api/v1/scan-config/workflows/:id/stop - 停止工作流
//  	POST   /api/v1/scan-config/workflows/:id/pause - 暂停工作流
//  	POST   /api/v1/scan-config/workflows/:id/resume - 恢复工作流
//  	POST   /api/v1/scan-config/workflows/:id/retry - 重试工作流
//  状态管理接口:
//  	POST   /api/v1/scan-config/workflows/:id/enable - 启用工作流
//  	POST   /api/v1/scan-config/workflows/:id/disable - 禁用工作流
//  	GET    /api/v1/scan-config/workflows/:id/status - 获取工作流状态
//  	GET    /api/v1/scan-config/workflows/:id/logs - 获取工作流日志
//  	GET    /api/v1/scan-config/workflows/:id/metrics - 获取工作流指标

package scan_config

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"neomaster/internal/model"
	"neomaster/internal/model/scan_config"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	scanConfigService "neomaster/internal/service/scan_config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// WorkflowHandler 工作流处理器结构体
// 负责处理工作流相关的HTTP请求
type WorkflowHandler struct {
	workflowService *scanConfigService.WorkflowService // 工作流服务
}

// NewWorkflowHandler 创建工作流处理器实例
// 注入必要的Service依赖，遵循依赖注入原则
func NewWorkflowHandler(workflowService *scanConfigService.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{
		workflowService: workflowService,
	}
}

// CreateWorkflow 创建工作流配置
// @route POST /api/v1/scan-config/workflows
// @param c Gin上下文
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	// 获取请求上下文信息 - Linus式：统一处理，消除重复
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析请求体
	var req scan_config.WorkflowConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "create_workflow",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.scan_config.workflow.CreateWorkflow",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数格式错误",
			Error:   err.Error(),
		})
		return
	}

	// 验证请求参数
	if err := h.validateWorkflowRequest(&req); err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":     "create_workflow",
			"option":        "validateWorkflowRequest",
			"func_name":     "handler.scan_config.workflow.CreateWorkflow",
			"client_ip":     clientIP,
			"user_agent":    userAgent,
			"request_id":    requestID,
			"workflow_name": req.Name,
			"timestamp":     logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数验证失败",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层创建工作流配置
	createdWorkflow, err := h.workflowService.CreateWorkflowConfig(c.Request.Context(), &req)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":     "create_workflow",
			"option":        "workflowService.CreateWorkflowConfig",
			"func_name":     "handler.scan_config.workflow.CreateWorkflow",
			"client_ip":     clientIP,
			"user_agent":    userAgent,
			"request_id":    requestID,
			"workflow_name": req.Name,
			"timestamp":     logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "已存在") {
			c.JSON(http.StatusConflict, model.APIResponse{
				Code:    http.StatusConflict,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "创建工作流配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "create_workflow", "创建工作流成功", logrus.InfoLevel, map[string]interface{}{
		"operation":     "create_workflow",
		"option":        "success",
		"func_name":     "handler.scan_config.workflow.CreateWorkflow",
		"client_ip":     clientIP,
		"user_agent":    userAgent,
		"request_id":    requestID,
		"workflow_id":   createdWorkflow.ID,
		"workflow_name": createdWorkflow.Name,
		"timestamp":     logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusCreated, model.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "工作流配置创建成功",
		Data:    createdWorkflow,
	})
}

// GetWorkflow 获取工作流配置详情
// @route GET /api/v1/scan-config/workflows/:id
// @param c Gin上下文
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_workflow",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.workflow.GetWorkflow",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的工作流配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层获取工作流配置
	workflow, err := h.workflowService.GetWorkflowConfig(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "GET", map[string]interface{}{
			"operation":   "get_workflow",
			"option":      "workflowService.GetWorkflowConfig",
			"func_name":   "handler.scan_config.workflow.GetWorkflow",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  requestID,
			"workflow_id": id,
			"timestamp":   logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "获取工作流配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "get_workflow", "获取工作流成功", logrus.InfoLevel, map[string]interface{}{
		"operation":     "get_workflow",
		"option":        "success",
		"func_name":     "handler.scan_config.workflow.GetWorkflow",
		"client_ip":     clientIP,
		"user_agent":    userAgent,
		"request_id":    requestID,
		"workflow_id":   id,
		"workflow_name": workflow.Name,
		"timestamp":     logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取工作流配置成功",
		Data:    workflow,
	})
}

// UpdateWorkflow 更新工作流配置
// @route PUT /api/v1/scan-config/workflows/:id
// @param c Gin上下文
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":  "update_workflow",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.workflow.UpdateWorkflow",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的工作流配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 解析请求体
	var req scan_config.WorkflowConfig
	if err1 := c.ShouldBindJSON(&req); err1 != nil {
		logger.LogError(err1, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":   "update_workflow",
			"option":      "ShouldBindJSON",
			"func_name":   "handler.scan_config.workflow.UpdateWorkflow",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  requestID,
			"workflow_id": id,
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数格式错误",
			Error:   err1.Error(),
		})
		return
	}

	// 验证请求参数
	if err2 := h.validateWorkflowRequest(&req); err2 != nil {
		logger.LogError(err2, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":     "update_workflow",
			"option":        "validateWorkflowRequest",
			"func_name":     "handler.scan_config.workflow.UpdateWorkflow",
			"client_ip":     clientIP,
			"user_agent":    userAgent,
			"request_id":    requestID,
			"workflow_id":   id,
			"workflow_name": req.Name,
			"timestamp":     logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数验证失败",
			Error:   err2.Error(),
		})
		return
	}

	// 调用Service层更新工作流配置
	updatedWorkflow, err := h.workflowService.UpdateWorkflowConfig(c.Request.Context(), uint(id), &req)
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":     "update_workflow",
			"option":        "workflowService.UpdateWorkflowConfig",
			"func_name":     "handler.scan_config.workflow.UpdateWorkflow",
			"client_ip":     clientIP,
			"user_agent":    userAgent,
			"request_id":    requestID,
			"workflow_id":   id,
			"workflow_name": req.Name,
			"timestamp":     logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else if strings.Contains(err.Error(), "已存在") {
			c.JSON(http.StatusConflict, model.APIResponse{
				Code:    http.StatusConflict,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "更新工作流配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "update_workflow", "更新工作流成功", logrus.InfoLevel, map[string]interface{}{
		"operation":     "update_workflow",
		"option":        "success",
		"func_name":     "handler.scan_config.workflow.UpdateWorkflow",
		"client_ip":     clientIP,
		"user_agent":    userAgent,
		"request_id":    requestID,
		"workflow_id":   id,
		"workflow_name": updatedWorkflow.Name,
		"timestamp":     logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "工作流配置更新成功",
		Data:    updatedWorkflow,
	})
}

// DeleteWorkflow 删除工作流配置
// @route DELETE /api/v1/scan-config/workflows/:id
// @param c Gin上下文
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "DELETE", map[string]interface{}{
			"operation":  "delete_workflow",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.workflow.DeleteWorkflow",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的工作流配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层删除工作流配置
	err = h.workflowService.DeleteWorkflowConfig(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "DELETE", map[string]interface{}{
			"operation":   "delete_workflow",
			"option":      "workflowService.DeleteWorkflowConfig",
			"func_name":   "handler.scan_config.workflow.DeleteWorkflow",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  requestID,
			"workflow_id": id,
			"timestamp":   logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "删除工作流配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "delete_workflow", "删除工作流成功", logrus.InfoLevel, map[string]interface{}{
		"operation":   "delete_workflow",
		"option":      "success",
		"func_name":   "handler.scan_config.workflow.DeleteWorkflow",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"request_id":  requestID,
		"workflow_id": id,
		"timestamp":   logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "工作流配置删除成功",
	})
}

// ListWorkflows 获取工作流配置列表
// @route GET /api/v1/scan-config/workflows
// @param c Gin上下文
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析查询参数
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")
	status := c.Query("status")
	projectIDStr := c.Query("project_id")
	triggerType := c.Query("trigger_type")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// 解析项目ID过滤参数
	var projectID *uint
	if projectIDStr != "" {
		if pid, err1 := strconv.ParseUint(projectIDStr, 10, 32); err1 == nil {
			id := uint(pid)
			projectID = &id
		}
	}

	// 解析状态过滤参数
	var statusFilter *scan_config.WorkflowStatus
	if status != "" {
		// 根据字符串值转换为对应的WorkflowStatus枚举
		switch strings.ToLower(status) {
		case "draft", "0":
			s := scan_config.WorkflowStatusDraft
			statusFilter = &s
		case "active", "1":
			s := scan_config.WorkflowStatusActive
			statusFilter = &s
		case "inactive", "2":
			s := scan_config.WorkflowStatusInactive
			statusFilter = &s
		case "archived", "3":
			s := scan_config.WorkflowStatusArchived
			statusFilter = &s
		}
	}

	// 解析触发类型过滤参数
	var triggerTypeFilter *scan_config.WorkflowTriggerType
	if triggerType != "" {
		t := scan_config.WorkflowTriggerType(triggerType)
		triggerTypeFilter = &t
	}

	// 调用Service层获取工作流配置列表
	workflows, total, err := h.workflowService.ListWorkflowConfigs(c.Request.Context(), offset, limit, statusFilter, projectID, triggerTypeFilter)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":    "list_workflows",
			"option":       "workflowService.ListWorkflowConfigs",
			"func_name":    "handler.scan_config.workflow.ListWorkflows",
			"client_ip":    clientIP,
			"user_agent":   userAgent,
			"request_id":   requestID,
			"offset":       offset,
			"limit":        limit,
			"status":       status,
			"project_id":   projectIDStr,
			"trigger_type": triggerType,
			"timestamp":    logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取工作流配置列表失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "list_workflows", "获取工作流列表成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "list_workflows",
		"option":     "success",
		"func_name":  "handler.scan_config.workflow.ListWorkflows",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"offset":     offset,
		"limit":      limit,
		"status":     status,
		"total":      total,
		"count":      len(workflows),
		"timestamp":  logger.NowFormatted(),
	})

	// 构建分页响应
	response := map[string]interface{}{
		"items":  workflows,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取工作流配置列表成功",
		Data:    response,
	})
}

// ExecuteWorkflow 执行工作流
// @route POST /api/v1/scan-config/workflows/:id/execute
// @param c Gin上下文
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	h.controlWorkflow(c, "execute", "执行工作流")
}

// StopWorkflow 停止工作流
// @route POST /api/v1/scan-config/workflows/:id/stop
// @param c Gin上下文
func (h *WorkflowHandler) StopWorkflow(c *gin.Context) {
	h.controlWorkflow(c, "stop", "停止工作流")
}

// PauseWorkflow 暂停工作流
// @route POST /api/v1/scan-config/workflows/:id/pause
// @param c Gin上下文
func (h *WorkflowHandler) PauseWorkflow(c *gin.Context) {
	h.controlWorkflow(c, "pause", "暂停工作流")
}

// ResumeWorkflow 恢复工作流
// @route POST /api/v1/scan-config/workflows/:id/resume
// @param c Gin上下文
func (h *WorkflowHandler) ResumeWorkflow(c *gin.Context) {
	h.controlWorkflow(c, "resume", "恢复工作流")
}

// RetryWorkflow 重试工作流
// @route POST /api/v1/scan-config/workflows/:id/retry
// @param c Gin上下文
func (h *WorkflowHandler) RetryWorkflow(c *gin.Context) {
	h.controlWorkflow(c, "retry", "重试工作流")
}

// EnableWorkflow 启用工作流
// @route POST /api/v1/scan-config/workflows/:id/enable
// @param c Gin上下文
func (h *WorkflowHandler) EnableWorkflow(c *gin.Context) {
	h.updateWorkflowStatus(c, scan_config.WorkflowStatusActive, "enable_workflow", "启用工作流")
}

// DisableWorkflow 禁用工作流
// @route POST /api/v1/scan-config/workflows/:id/disable
// @param c Gin上下文
func (h *WorkflowHandler) DisableWorkflow(c *gin.Context) {
	h.updateWorkflowStatus(c, scan_config.WorkflowStatusInactive, "disable_workflow", "禁用工作流")
}

// GetWorkflowStatus 获取工作流状态
// @route GET /api/v1/scan-config/workflows/:id/status
// @param c Gin上下文
func (h *WorkflowHandler) GetWorkflowStatus(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_workflow_status",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.workflow.GetWorkflowStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的工作流配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层获取工作流状态
	status, err := h.workflowService.GetWorkflowStatus(c.Request.Context(), idStr)
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "GET", map[string]interface{}{
			"operation":   "get_workflow_status",
			"option":      "workflowService.GetWorkflowStatus",
			"func_name":   "handler.scan_config.workflow.GetWorkflowStatus",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  requestID,
			"workflow_id": id,
			"timestamp":   logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "获取工作流状态失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "get_workflow_status", "获取工作流状态成功", logrus.InfoLevel, map[string]interface{}{
		"operation":   "get_workflow_status",
		"option":      "success",
		"func_name":   "handler.scan_config.workflow.GetWorkflowStatus",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"request_id":  requestID,
		"workflow_id": id,
		"status":      status,
		"timestamp":   logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取工作流状态成功",
		Data:    map[string]interface{}{"status": status},
	})
}

// GetWorkflowLogs 获取工作流日志
// @route GET /api/v1/scan-config/workflows/:id/logs
// @param c Gin上下文
func (h *WorkflowHandler) GetWorkflowLogs(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_workflow_logs",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.workflow.GetWorkflowLogs",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的工作流配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 解析查询参数
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "100")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100
	}

	// 调用Service层获取工作流日志
	logs, err := h.workflowService.GetWorkflowLogs(c.Request.Context(), idStr, offset, limit)
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "GET", map[string]interface{}{
			"operation":   "get_workflow_logs",
			"option":      "workflowService.GetWorkflowLogs",
			"func_name":   "handler.scan_config.workflow.GetWorkflowLogs",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  requestID,
			"workflow_id": id,
			"offset":      offset,
			"limit":       limit,
			"timestamp":   logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "获取工作流日志失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "get_workflow_logs", "获取工作流日志成功", logrus.InfoLevel, map[string]interface{}{
		"operation":   "get_workflow_logs",
		"option":      "success",
		"func_name":   "handler.scan_config.workflow.GetWorkflowLogs",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"request_id":  requestID,
		"workflow_id": id,
		"offset":      offset,
		"limit":       limit,
		"log_count":   len(logs),
		"timestamp":   logger.NowFormatted(),
	})

	// 构建响应
	response := map[string]interface{}{
		"logs":   logs,
		"offset": offset,
		"limit":  limit,
		"count":  len(logs),
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取工作流日志成功",
		Data:    response,
	})
}

// GetWorkflowMetrics 获取工作流指标
// @route GET /api/v1/scan-config/workflows/:id/metrics
// @param c Gin上下文
func (h *WorkflowHandler) GetWorkflowMetrics(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_workflow_metrics",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.workflow.GetWorkflowMetrics",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的工作流配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层获取工作流指标
	metrics, err := h.workflowService.GetWorkflowPerformance(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "GET", map[string]interface{}{
			"operation":   "get_workflow_metrics",
			"option":      "workflowService.GetWorkflowPerformance",
			"func_name":   "handler.scan_config.workflow.GetWorkflowMetrics",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  requestID,
			"workflow_id": id,
			"timestamp":   logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "获取工作流指标失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "get_workflow_metrics", "获取工作流指标成功", logrus.InfoLevel, map[string]interface{}{
		"operation":   "get_workflow_metrics",
		"option":      "success",
		"func_name":   "handler.scan_config.workflow.GetWorkflowMetrics",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"request_id":  requestID,
		"workflow_id": id,
		"timestamp":   logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取工作流指标成功",
		Data:    metrics,
	})
}

// 私有方法：工作流控制操作
func (h *WorkflowHandler) controlWorkflow(c *gin.Context, action, message string) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  action + "_workflow",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.workflow.controlWorkflow",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"action":     action,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的工作流配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 根据操作类型调用不同的Service方法
	var serviceErr error
	var result interface{}

	switch action {
	case "execute":
		result, serviceErr = h.workflowService.ExecuteWorkflow(c.Request.Context(), uint(id), nil)
	case "stop":
		serviceErr = h.workflowService.StopWorkflow(c.Request.Context(), idStr)
	case "pause":
		serviceErr = h.workflowService.PauseWorkflow(c.Request.Context(), idStr)
	case "resume":
		serviceErr = h.workflowService.ResumeWorkflow(c.Request.Context(), idStr)
	case "retry":
		result, serviceErr = h.workflowService.RetryWorkflow(c.Request.Context(), idStr)
	default:
		serviceErr = errors.New("不支持的操作类型")
	}

	if serviceErr != nil {
		logger.LogError(serviceErr, requestID, uint(id), clientIP, urlPath, "POST", map[string]interface{}{
			"operation":   action + "_workflow",
			"option":      "workflowService." + strings.Title(action) + "Workflow",
			"func_name":   "handler.scan_config.workflow.controlWorkflow",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  requestID,
			"workflow_id": id,
			"action":      action,
			"timestamp":   logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(serviceErr.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: serviceErr.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: message + "失败",
				Error:   serviceErr.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", action+"_workflow", message+"成功", logrus.InfoLevel, map[string]interface{}{
		"operation":   action + "_workflow",
		"option":      "success",
		"func_name":   "handler.scan_config.workflow.controlWorkflow",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"request_id":  requestID,
		"workflow_id": id,
		"action":      action,
		"timestamp":   logger.NowFormatted(),
	})

	// 构建响应
	response := model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: message + "成功",
	}

	if result != nil {
		response.Data = result
	}

	// 返回成功响应
	c.JSON(http.StatusOK, response)
}

// 私有方法：更新工作流状态
func (h *WorkflowHandler) updateWorkflowStatus(c *gin.Context, status scan_config.WorkflowStatus, operation, message string) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  operation,
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.workflow.updateWorkflowStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的工作流配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层更新状态
	var serviceErr error
	if status == scan_config.WorkflowStatusActive {
		serviceErr = h.workflowService.EnableWorkflowConfig(c.Request.Context(), uint(id))
	} else {
		serviceErr = h.workflowService.DisableWorkflowConfig(c.Request.Context(), uint(id))
	}

	if serviceErr != nil {
		logger.LogError(serviceErr, requestID, uint(id), clientIP, urlPath, "POST", map[string]interface{}{
			"operation":   operation,
			"option":      "workflowService.UpdateStatus",
			"func_name":   "handler.scan_config.workflow.updateWorkflowStatus",
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  requestID,
			"workflow_id": id,
			"status":      status,
			"timestamp":   logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(serviceErr.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: serviceErr.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: message + "失败",
				Error:   serviceErr.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", operation, message+"成功", logrus.InfoLevel, map[string]interface{}{
		"operation":   operation,
		"option":      "success",
		"func_name":   "handler.scan_config.workflow.updateWorkflowStatus",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"request_id":  requestID,
		"workflow_id": id,
		"status":      status,
		"timestamp":   logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: message + "成功",
	})
}

// 私有方法：验证工作流请求参数
func (h *WorkflowHandler) validateWorkflowRequest(req *scan_config.WorkflowConfig) error {
	// 基础字段验证
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("工作流名称不能为空")
	}

	if len(req.Name) > 100 {
		return errors.New("工作流名称长度不能超过100个字符")
	}

	if len(req.Description) > 500 {
		return errors.New("工作流描述长度不能超过500个字符")
	}

	// 项目ID验证
	if req.ProjectID == 0 {
		return errors.New("项目ID不能为空")
	}

	// 触发类型验证
	if req.TriggerType == "" {
		return errors.New("触发类型不能为空")
	}

	// 步骤配置验证
	if strings.TrimSpace(req.Steps) == "" {
		return errors.New("步骤配置不能为空")
	}

	// TODO: 可以添加更多的业务验证逻辑
	// 1. 验证步骤配置JSON格式
	// 2. 验证触发配置JSON格式
	// 3. 验证通知配置JSON格式
	// 4. 验证调度配置JSON格式

	return nil
}

// GetWorkflowsByProject 按项目获取工作流
func (h *WorkflowHandler) GetWorkflowsByProject(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		logger.Error("项目ID格式错误", map[string]interface{}{
			"path":      "/api/v1/scan-config/workflows/project/:project_id",
			"operation": "get_workflows_by_project",
			"option":    "parse_project_id",
			"func_name": "handler.scan_config.workflow.GetWorkflowsByProject",
			"error":     err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目ID格式错误"})
		return
	}

	workflows, err := h.workflowService.GetWorkflowsByProject(c.Request.Context(), uint(projectID))
	if err != nil {
		logger.Error("获取项目工作流失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/workflows/project/:project_id",
			"operation":  "get_workflows_by_project",
			"option":     "workflowService.GetWorkflowsByProject",
			"func_name":  "handler.scan_config.workflow.GetWorkflowsByProject",
			"project_id": projectID,
			"error":      err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取项目工作流失败"})
		return
	}

	logger.Info("获取项目工作流成功", map[string]interface{}{
		"path":       "/api/v1/scan-config/workflows/project/:project_id",
		"operation":  "get_workflows_by_project",
		"option":     "success",
		"func_name":  "handler.scan_config.workflow.GetWorkflowsByProject",
		"project_id": projectID,
		"count":      len(workflows),
	})

	c.JSON(http.StatusOK, gin.H{
		"data":  workflows,
		"count": len(workflows),
	})
}

// GetSystemScanStatistics 获取系统扫描统计信息
// @route GET /api/v1/scan-config/workflows/system-statistics
// @param c Gin上下文
func (h *WorkflowHandler) GetSystemScanStatistics(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 调用Service层获取系统扫描统计信息
	statistics, err := h.workflowService.GetSystemScanStatistics(c.Request.Context())
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_system_scan_statistics",
			"option":     "workflowService.GetSystemScanStatistics",
			"func_name":  "handler.scan_config.workflow.GetSystemScanStatistics",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取系统扫描统计信息失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "get_system_scan_statistics", "获取系统扫描统计信息成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "get_system_scan_statistics",
		"option":     "success",
		"func_name":  "handler.scan_config.workflow.GetSystemScanStatistics",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取系统扫描统计信息成功",
		Data:    statistics,
	})
}

// GetSystemPerformance 获取系统性能信息
// @route GET /api/v1/scan-config/workflows/system-performance
// @param c Gin上下文
func (h *WorkflowHandler) GetSystemPerformance(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 调用Service层获取系统性能信息
	performance, err := h.workflowService.GetSystemPerformance(c.Request.Context())
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_system_performance",
			"option":     "workflowService.GetSystemPerformance",
			"func_name":  "handler.scan_config.workflow.GetSystemPerformance",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取系统性能信息失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("workflow_handler", "get_system_performance", "获取系统性能信息成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "get_system_performance",
		"option":     "success",
		"func_name":  "handler.scan_config.workflow.GetSystemPerformance",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取系统性能信息成功",
		Data:    performance,
	})
}
