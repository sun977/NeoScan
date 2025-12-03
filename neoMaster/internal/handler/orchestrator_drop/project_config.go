/*
 * 项目配置处理器：项目配置HTTP接口处理
 * @author: Sun977
 * @date: 2025.01.27
 * @description: 处理项目配置相关的HTTP请求和响应
 * @func:
 * 1.项目配置CRUD操作接口
 * 2.项目配置状态管理接口
 * 3.项目配置查询和列表接口
 * 4.项目配置同步和热重载接口
 */

//  核心HTTP接口:
//  	POST   /api/v1/orchestrator/projects - 创建项目配置
//  	GET    /api/v1/orchestrator/projects/:id - 获取项目配置详情
//  	PUT    /api/v1/orchestrator/projects/:id - 更新项目配置
//  	DELETE /api/v1/orchestrator/projects/:id - 删除项目配置
//  	GET    /api/v1/orchestrator/projects - 获取项目配置列表
//  状态管理接口:
//  	POST   /api/v1/orchestrator/projects/:id/enable - 启用项目配置
//  	POST   /api/v1/orchestrator/projects/:id/disable - 禁用项目配置
//  配置管理接口:
//  	POST   /api/v1/orchestrator/projects/:id/reload - 热重载项目配置
//  	POST   /api/v1/orchestrator/projects/:id/sync - 同步项目配置

package orchestrator

import (
	"errors"
	"neomaster/internal/model/system"
	"net/http"
	"strconv"
	"strings"

	"neomaster/internal/model/orchestrator_drop"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	scanConfigService "neomaster/internal/service/orchestrator_drop"

	"github.com/gin-gonic/gin"
)

// ProjectConfigHandler 项目配置处理器结构体
// 负责处理项目配置相关的HTTP请求
type ProjectConfigHandler struct {
	projectConfigService *scanConfigService.ProjectConfigService // 项目配置服务
}

// NewProjectConfigHandler 创建项目配置处理器实例
// 注入必要的Service依赖，遵循依赖注入原则
func NewProjectConfigHandler(projectConfigService *scanConfigService.ProjectConfigService) *ProjectConfigHandler {
	return &ProjectConfigHandler{
		projectConfigService: projectConfigService,
	}
}

// CreateProjectConfig 创建项目配置
// @route POST /api/v1/orchestrator/projects
// @param c Gin上下文
func (h *ProjectConfigHandler) CreateProjectConfig(c *gin.Context) {
	// 获取请求上下文信息 - Linus式：统一处理，消除重复
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析请求体
	var req orchestrator_drop.ProjectConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "create_project_config",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.orchestrator.project_config.CreateProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数格式错误",
			Error:   err.Error(),
		})
		return
	}

	// 验证请求参数
	if err := h.validateProjectConfigRequest(&req); err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":    "create_project_config",
			"option":       "validateProjectConfigRequest",
			"func_name":    "handler.orchestrator.project_config.CreateProjectConfig",
			"client_ip":    clientIP,
			"user_agent":   userAgent,
			"request_id":   requestID,
			"project_name": req.Name,
			"timestamp":    logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数验证失败",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层创建项目配置
	createdConfig, err := h.projectConfigService.CreateProjectConfig(c.Request.Context(), &req)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":    "create_project_config",
			"option":       "projectConfigService.CreateProjectConfig",
			"func_name":    "handler.orchestrator.project_config.CreateProjectConfig",
			"client_ip":    clientIP,
			"user_agent":   userAgent,
			"request_id":   requestID,
			"project_name": req.Name,
			"timestamp":    logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "已存在") {
			c.JSON(http.StatusConflict, system.APIResponse{
				Code:    http.StatusConflict,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "创建项目配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "create_project_config", "创建项目配置成功", logger.InfoLevel, map[string]interface{}{
		"operation":    "create_project_config",
		"option":       "success",
		"func_name":    "handler.orchestrator.project_config.CreateProjectConfig",
		"client_ip":    clientIP,
		"user_agent":   userAgent,
		"request_id":   requestID,
		"project_id":   createdConfig.ID,
		"project_name": createdConfig.Name,
		"timestamp":    logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "项目配置创建成功",
		Data:    createdConfig,
	})
}

// GetProjectConfig 获取项目配置详情
// @route GET /api/v1/orchestrator/projects/:id
// @param c Gin上下文
func (h *ProjectConfigHandler) GetProjectConfig(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_project_config",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.project_config.GetProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的项目配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层获取项目配置
	config, err := h.projectConfigService.GetProjectConfig(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogBusinessError(err, requestID, uint(id), clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_project_config",
			"option":     "projectConfigService.GetProjectConfig",
			"func_name":  "handler.orchestrator.project_config.GetProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"project_id": id,
			"timestamp":  logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, system.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "获取项目配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "get_project_config", "获取项目配置成功", logger.InfoLevel, map[string]interface{}{
		"operation":    "get_project_config",
		"option":       "success",
		"func_name":    "handler.orchestrator.project_config.GetProjectConfig",
		"client_ip":    clientIP,
		"user_agent":   userAgent,
		"request_id":   requestID,
		"project_id":   id,
		"project_name": config.Name,
		"timestamp":    logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取项目配置成功",
		Data:    config,
	})
}

// UpdateProjectConfig 更新项目配置
// @route PUT /api/v1/orchestrator/projects/:id
// @param c Gin上下文
func (h *ProjectConfigHandler) UpdateProjectConfig(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":  "update_project_config",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.project_config.UpdateProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的项目配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 解析请求体
	var req orchestrator_drop.ProjectConfig
	if err1 := c.ShouldBindJSON(&req); err1 != nil {
		logger.LogBusinessError(err1, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":  "update_project_config",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.orchestrator.project_config.UpdateProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"project_id": id,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数格式错误",
			Error:   err1.Error(),
		})
		return
	}

	// 验证请求参数
	if err2 := h.validateProjectConfigRequest(&req); err2 != nil {
		logger.LogBusinessError(err2, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":    "update_project_config",
			"option":       "validateProjectConfigRequest",
			"func_name":    "handler.orchestrator.project_config.UpdateProjectConfig",
			"client_ip":    clientIP,
			"user_agent":   userAgent,
			"request_id":   requestID,
			"project_id":   id,
			"project_name": req.Name,
			"timestamp":    logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数验证失败",
			Error:   err2.Error(),
		})
		return
	}

	// 调用Service层更新项目配置
	updatedConfig, err := h.projectConfigService.UpdateProjectConfig(c.Request.Context(), uint(id), &req)
	if err != nil {
		logger.LogBusinessError(err, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":    "update_project_config",
			"option":       "projectConfigService.UpdateProjectConfig",
			"func_name":    "handler.orchestrator.project_config.UpdateProjectConfig",
			"client_ip":    clientIP,
			"user_agent":   userAgent,
			"request_id":   requestID,
			"project_id":   id,
			"project_name": req.Name,
			"timestamp":    logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, system.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else if strings.Contains(err.Error(), "已存在") {
			c.JSON(http.StatusConflict, system.APIResponse{
				Code:    http.StatusConflict,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "更新项目配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "update_project_config", "更新项目配置成功", logger.InfoLevel, map[string]interface{}{
		"operation":    "update_project_config",
		"option":       "success",
		"func_name":    "handler.orchestrator.project_config.UpdateProjectConfig",
		"client_ip":    clientIP,
		"user_agent":   userAgent,
		"request_id":   requestID,
		"project_id":   id,
		"project_name": updatedConfig.Name,
		"timestamp":    logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "项目配置更新成功",
		Data:    updatedConfig,
	})
}

// DeleteProjectConfig 删除项目配置
// @route DELETE /api/v1/orchestrator/projects/:id
// @param c Gin上下文
func (h *ProjectConfigHandler) DeleteProjectConfig(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "DELETE", map[string]interface{}{
			"operation":  "delete_project_config",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.project_config.DeleteProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的项目配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层删除项目配置
	err = h.projectConfigService.DeleteProjectConfig(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogBusinessError(err, requestID, uint(id), clientIP, urlPath, "DELETE", map[string]interface{}{
			"operation":  "delete_project_config",
			"option":     "projectConfigService.DeleteProjectConfig",
			"func_name":  "handler.orchestrator.project_config.DeleteProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"project_id": id,
			"timestamp":  logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, system.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "删除项目配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "delete_project_config", "删除项目配置成功", logger.InfoLevel, map[string]interface{}{
		"operation":  "delete_project_config",
		"option":     "success",
		"func_name":  "handler.orchestrator.project_config.DeleteProjectConfig",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"project_id": id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "项目配置删除成功",
	})
}

// ListProjectConfigs 获取项目配置列表
// @route GET /api/v1/orchestrator/projects
// @param c Gin上下文
func (h *ProjectConfigHandler) ListProjectConfigs(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析查询参数
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")
	status := c.Query("status")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// 解析状态过滤参数
	var statusFilter *orchestrator_drop.ProjectConfigStatus
	if status != "" {
		var s orchestrator_drop.ProjectConfigStatus
		switch status {
		case "inactive", "0":
			s = orchestrator_drop.ProjectConfigStatusInactive
		case "active", "1":
			s = orchestrator_drop.ProjectConfigStatusActive
		case "archived", "2":
			s = orchestrator_drop.ProjectConfigStatusArchived
		default:
			// 无效状态，忽略过滤
			statusFilter = nil
		}
		if statusFilter == nil {
			statusFilter = &s
		}
	}

	// 调用Service层获取项目配置列表
	configs, total, err := h.projectConfigService.ListProjectConfigs(c.Request.Context(), offset, limit, statusFilter, nil)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "list_project_configs",
			"option":     "projectConfigService.ListProjectConfigs",
			"func_name":  "handler.orchestrator.project_config.ListProjectConfigs",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"offset":     offset,
			"limit":      limit,
			"status":     status,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取项目配置列表失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "list_project_configs", "获取项目配置列表成功", logger.InfoLevel, map[string]interface{}{
		"operation":  "list_project_configs",
		"option":     "success",
		"func_name":  "handler.orchestrator.project_config.ListProjectConfigs",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"offset":     offset,
		"limit":      limit,
		"status":     status,
		"total":      total,
		"count":      len(configs),
		"timestamp":  logger.NowFormatted(),
	})

	// 构建分页响应
	response := map[string]interface{}{
		"items":  configs,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	}

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取项目配置列表成功",
		Data:    response,
	})
}

// EnableProjectConfig 启用项目配置
// @route POST /api/v1/orchestrator/projects/:id/enable
// @param c Gin上下文
func (h *ProjectConfigHandler) EnableProjectConfig(c *gin.Context) {
	h.updateProjectConfigStatus(c, orchestrator_drop.ProjectConfigStatusActive, "enable_project_config", "启用项目配置")
}

// DisableProjectConfig 禁用项目配置
// @route POST /api/v1/orchestrator/projects/:id/disable
// @param c Gin上下文
func (h *ProjectConfigHandler) DisableProjectConfig(c *gin.Context) {
	h.updateProjectConfigStatus(c, orchestrator_drop.ProjectConfigStatusInactive, "disable_project_config", "禁用项目配置")
}

// ReloadProjectConfig 热重载项目配置
// @route POST /api/v1/orchestrator/projects/:id/reload
// @param c Gin上下文
func (h *ProjectConfigHandler) ReloadProjectConfig(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "reload_project_config",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.project_config.ReloadProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的项目配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层热重载项目配置
	err = h.projectConfigService.ReloadProjectConfig(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogBusinessError(err, requestID, uint(id), clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "reload_project_config",
			"option":     "projectConfigService.ReloadProjectConfig",
			"func_name":  "handler.orchestrator.project_config.ReloadProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"project_id": id,
			"timestamp":  logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, system.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "热重载项目配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "reload_project_config", "重新加载项目配置成功", logger.InfoLevel, map[string]interface{}{
		"operation":  "reload_project_config",
		"option":     "success",
		"func_name":  "handler.orchestrator.project_config.ReloadProjectConfig",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"project_id": id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "项目配置热重载成功",
	})
}

// SyncProjectConfig 同步项目配置
// @route POST /api/v1/orchestrator/projects/:id/sync
// @param c Gin上下文
func (h *ProjectConfigHandler) SyncProjectConfig(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "sync_project_config",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.project_config.SyncProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的项目配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层同步项目配置
	err = h.projectConfigService.SyncProjectConfig(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogBusinessError(err, requestID, uint(id), clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "sync_project_config",
			"option":     "projectConfigService.SyncProjectConfig",
			"func_name":  "handler.orchestrator.project_config.SyncProjectConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"project_id": id,
			"timestamp":  logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, system.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "同步项目配置失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "sync_project_config", "同步项目配置成功", logger.InfoLevel, map[string]interface{}{
		"operation":  "sync_project_config",
		"option":     "success",
		"func_name":  "handler.orchestrator.project_config.SyncProjectConfig",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"project_id": id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "项目配置同步成功",
	})
}

// 私有方法：更新项目配置状态
func (h *ProjectConfigHandler) updateProjectConfigStatus(c *gin.Context, status orchestrator_drop.ProjectConfigStatus, operation, message string) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  operation,
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.project_config.updateProjectConfigStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的项目配置ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层更新状态
	var serviceErr error
	if status == orchestrator_drop.ProjectConfigStatusActive {
		serviceErr = h.projectConfigService.EnableProjectConfig(c.Request.Context(), uint(id))
	} else {
		serviceErr = h.projectConfigService.DisableProjectConfig(c.Request.Context(), uint(id))
	}

	if serviceErr != nil {
		logger.LogBusinessError(serviceErr, requestID, uint(id), clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  operation,
			"option":     "projectConfigService.UpdateStatus",
			"func_name":  "handler.orchestrator.project_config.updateProjectConfigStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"project_id": id,
			"status":     status,
			"timestamp":  logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(serviceErr.Error(), "不存在") {
			c.JSON(http.StatusNotFound, system.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: serviceErr.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: message + "失败",
				Error:   serviceErr.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "update_project_config_status", "更新项目配置状态成功", logger.InfoLevel, map[string]interface{}{
		"operation":  operation,
		"option":     "success",
		"func_name":  "handler.orchestrator.project_config.updateProjectConfigStatus",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"project_id": id,
		"status":     status,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: message + "成功",
	})
}

// 私有方法：验证项目配置请求参数
func (h *ProjectConfigHandler) validateProjectConfigRequest(req *orchestrator_drop.ProjectConfig) error {
	// 基础字段验证
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("项目名称不能为空")
	}

	if len(req.Name) > 100 {
		return errors.New("项目名称长度不能超过100个字符")
	}

	if len(req.Description) > 500 {
		return errors.New("项目描述长度不能超过500个字符")
	}

	// 目标配置验证
	if strings.TrimSpace(req.TargetScope) == "" {
		return errors.New("目标配置不能为空")
	}

	// TODO: 可以添加更多的业务验证逻辑
	// 1. 验证目标配置JSON格式
	// 2. 验证扫描配置JSON格式
	// 3. 验证通知配置JSON格式
	// 4. 验证调度配置JSON格式

	return nil
}

// GetSystemScanConfig 获取系统扫描配置
// @route GET /api/v1/orchestrator/system
// @param c Gin上下文
func (h *ProjectConfigHandler) GetSystemScanConfig(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 调用Service层获取系统配置
	config, err := h.projectConfigService.GetSystemScanConfig(c.Request.Context())
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_system_scan_config",
			"option":     "projectConfigService.GetSystemScanConfig",
			"func_name":  "handler.orchestrator.project_config.GetSystemScanConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取系统扫描配置失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "get_system_scan_config", "获取系统扫描配置成功", logger.InfoLevel, map[string]interface{}{
		"operation":  "get_system_scan_config",
		"option":     "success",
		"func_name":  "handler.orchestrator.project_config.GetSystemScanConfig",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取系统扫描配置成功",
		Data:    config,
	})
}

// UpdateSystemScanConfig 更新系统扫描配置
// @route PUT /api/v1/orchestrator/system
// @param c Gin上下文
func (h *ProjectConfigHandler) UpdateSystemScanConfig(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析请求体
	var configData map[string]interface{}
	if err := c.ShouldBindJSON(&configData); err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":  "update_system_scan_config",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.orchestrator.project_config.UpdateSystemScanConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数格式错误",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层更新系统配置
	err := h.projectConfigService.UpdateSystemScanConfig(c.Request.Context(), configData)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":  "update_system_scan_config",
			"option":     "projectConfigService.UpdateSystemScanConfig",
			"func_name":  "handler.orchestrator.project_config.UpdateSystemScanConfig",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "更新系统扫描配置失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("project_config_handler", "update_system_scan_config", "更新系统扫描配置成功", logger.InfoLevel, map[string]interface{}{
		"operation":  "update_system_scan_config",
		"option":     "success",
		"func_name":  "handler.orchestrator.project_config.UpdateSystemScanConfig",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "更新系统扫描配置成功",
	})
}
