/*
 * 扫描工具处理器：扫描工具HTTP接口处理
 * @author: Linus-inspired AI
 * @date: 2025.01.27
 * @description: 处理扫描工具相关的HTTP请求和响应
 * @func:
 * 1.扫描工具CRUD操作接口
 * 2.扫描工具状态管理接口
 * 3.扫描工具健康检查接口
 * 4.扫描工具统计和指标接口
 */

//  核心HTTP接口:
//  	POST   /api/v1/scan-config/tools - 创建扫描工具
//  	GET    /api/v1/scan-config/tools/:id - 获取扫描工具详情
//  	PUT    /api/v1/scan-config/tools/:id - 更新扫描工具
//  	DELETE /api/v1/scan-config/tools/:id - 删除扫描工具
//  	GET    /api/v1/scan-config/tools - 获取扫描工具列表
//  状态管理接口:
//  	POST   /api/v1/scan-config/tools/:id/enable - 启用扫描工具
//  	POST   /api/v1/scan-config/tools/:id/disable - 禁用扫描工具
//  	GET    /api/v1/scan-config/tools/:id/health - 健康检查
//  工具管理接口:
//  	POST   /api/v1/scan-config/tools/:id/install - 安装扫描工具
//  	POST   /api/v1/scan-config/tools/:id/uninstall - 卸载扫描工具
//  	GET    /api/v1/scan-config/tools/:id/metrics - 获取工具指标

package scan_config

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"neomaster/internal/model"
	"neomaster/internal/model/scan_config"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	scan_config_service "neomaster/internal/service/scan_config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ScanToolHandler 扫描工具处理器结构体
// 负责处理扫描工具相关的HTTP请求
type ScanToolHandler struct {
	scanToolService *scan_config_service.ScanToolService // 扫描工具服务
}

// NewScanToolHandler 创建扫描工具处理器实例
// 注入必要的Service依赖，遵循依赖注入原则
func NewScanToolHandler(scanToolService *scan_config_service.ScanToolService) *ScanToolHandler {
	return &ScanToolHandler{
		scanToolService: scanToolService,
	}
}

// CreateScanTool 创建扫描工具
// @route POST /api/v1/scan-config/tools
// @param c Gin上下文
func (h *ScanToolHandler) CreateScanTool(c *gin.Context) {
	// 获取请求上下文信息 - Linus式：统一处理，消除重复
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析请求体
	var req scan_config.ScanTool
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "create_scan_tool",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.scan_config.scan_tool.CreateScanTool",
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
	if err := h.validateScanToolRequest(&req); err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "create_scan_tool",
			"option":     "validateScanToolRequest",
			"func_name":  "handler.scan_config.scan_tool.CreateScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_name":  req.Name,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数验证失败",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层创建扫描工具
	createdTool, err := h.scanToolService.CreateScanTool(c.Request.Context(), &req)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "create_scan_tool",
			"option":     "scanToolService.CreateScanTool",
			"func_name":  "handler.scan_config.scan_tool.CreateScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_name":  req.Name,
			"timestamp":  logger.NowFormatted(),
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
				Message: "创建扫描工具失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "create_scan_tool", "创建扫描工具成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "create_scan_tool",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.CreateScanTool",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"tool_id":    createdTool.ID,
		"tool_name":  createdTool.Name,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusCreated, model.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "扫描工具创建成功",
		Data:    createdTool,
	})
}

// GetScanTool 获取扫描工具详情
// @route GET /api/v1/scan-config/tools/:id
// @param c Gin上下文
func (h *ScanToolHandler) GetScanTool(c *gin.Context) {
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
			"operation":  "get_scan_tool",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.scan_tool.GetScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描工具ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层获取扫描工具
	tool, err := h.scanToolService.GetScanTool(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_scan_tool",
			"option":     "scanToolService.GetScanTool",
			"func_name":  "handler.scan_config.scan_tool.GetScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"timestamp":  logger.NowFormatted(),
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
				Message: "获取扫描工具失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "get_scan_tool", "获取扫描工具成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "get_scan_tool",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.GetScanTool",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"tool_id":    id,
		"tool_name":  tool.Name,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取扫描工具成功",
		Data:    tool,
	})
}

// UpdateScanTool 更新扫描工具
// @route PUT /api/v1/scan-config/tools/:id
// @param c Gin上下文
func (h *ScanToolHandler) UpdateScanTool(c *gin.Context) {
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
			"operation":  "update_scan_tool",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.scan_tool.UpdateScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描工具ID",
			Error:   err.Error(),
		})
		return
	}

	// 解析请求体
	var req scan_config.ScanTool
	if err1 := c.ShouldBindJSON(&req); err1 != nil {
		logger.LogError(err1, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":  "update_scan_tool",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.scan_config.scan_tool.UpdateScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"timestamp":  logger.NowFormatted(),
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
	if err2 := h.validateScanToolRequest(&req); err2 != nil {
		logger.LogError(err2, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":  "update_scan_tool",
			"option":     "validateScanToolRequest",
			"func_name":  "handler.scan_config.scan_tool.UpdateScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"tool_name":  req.Name,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数验证失败",
			Error:   err2.Error(),
		})
		return
	}

	// 调用Service层更新扫描工具
	updatedTool, err := h.scanToolService.UpdateScanTool(c.Request.Context(), uint(id), &req)
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "PUT", map[string]interface{}{
			"operation":  "update_scan_tool",
			"option":     "scanToolService.UpdateScanTool",
			"func_name":  "handler.scan_config.scan_tool.UpdateScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"tool_name":  req.Name,
			"timestamp":  logger.NowFormatted(),
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
				Message: "更新扫描工具失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "update_scan_tool", "更新扫描工具成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "update_scan_tool",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.UpdateScanTool",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"tool_id":    id,
		"tool_name":  updatedTool.Name,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "扫描工具更新成功",
		Data:    updatedTool,
	})
}

// DeleteScanTool 删除扫描工具
// @route DELETE /api/v1/scan-config/tools/:id
// @param c Gin上下文
func (h *ScanToolHandler) DeleteScanTool(c *gin.Context) {
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
			"operation":  "delete_scan_tool",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.scan_tool.DeleteScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描工具ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层删除扫描工具
	err = h.scanToolService.DeleteScanTool(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "DELETE", map[string]interface{}{
			"operation":  "delete_scan_tool",
			"option":     "scanToolService.DeleteScanTool",
			"func_name":  "handler.scan_config.scan_tool.DeleteScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"timestamp":  logger.NowFormatted(),
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
				Message: "删除扫描工具失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "delete_scan_tool", "删除扫描工具成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "delete_scan_tool",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.DeleteScanTool",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"tool_id":    id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "扫描工具删除成功",
	})
}

// ListScanTools 获取扫描工具列表
// @route GET /api/v1/scan-config/tools
// @param c Gin上下文
func (h *ScanToolHandler) ListScanTools(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析查询参数
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")
	status := c.Query("status")
	toolType := c.Query("type")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// 解析状态过滤参数
	var statusFilter *scan_config.ScanToolStatus
	if status != "" {
		var s scan_config.ScanToolStatus
		switch status {
		case "disabled", "0":
			s = scan_config.ScanToolStatusDisabled
		case "enabled", "1":
			s = scan_config.ScanToolStatusEnabled
		case "testing", "2":
			s = scan_config.ScanToolStatusTesting
		default:
			// 无效状态，忽略过滤
			statusFilter = nil
		}
		if statusFilter == nil {
			statusFilter = &s
		}
	}

	// 解析工具类型过滤参数
	var typeFilter *scan_config.ScanToolType
	if toolType != "" {
		t := scan_config.ScanToolType(toolType)
		typeFilter = &t
	}

	// 调用Service层获取扫描工具列表
	tools, total, err := h.scanToolService.ListScanTools(c.Request.Context(), offset, limit, typeFilter, statusFilter)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "list_scan_tools",
			"option":     "scanToolService.ListScanTools",
			"func_name":  "handler.scan_config.scan_tool.ListScanTools",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"offset":     offset,
			"limit":      limit,
			"status":     status,
			"type":       toolType,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取扫描工具列表失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "list_scan_tools", "获取扫描工具列表成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "list_scan_tools",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.ListScanTools",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"offset":     offset,
		"limit":      limit,
		"status":     status,
		"type":       toolType,
		"total":      total,
		"count":      len(tools),
		"timestamp":  logger.NowFormatted(),
	})

	// 构建分页响应
	response := map[string]interface{}{
		"items":  tools,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取扫描工具列表成功",
		Data:    response,
	})
}

// EnableScanTool 启用扫描工具
// @route POST /api/v1/scan-config/tools/:id/enable
// @param c Gin上下文
func (h *ScanToolHandler) EnableScanTool(c *gin.Context) {
	h.updateScanToolStatus(c, scan_config.ScanToolStatusEnabled, "enable_scan_tool", "启用扫描工具")
}

// DisableScanTool 禁用扫描工具
// @route POST /api/v1/scan-config/tools/:id/disable
// @param c Gin上下文
func (h *ScanToolHandler) DisableScanTool(c *gin.Context) {
	h.updateScanToolStatus(c, scan_config.ScanToolStatusDisabled, "disable_scan_tool", "禁用扫描工具")
}

// HealthCheckScanTool 扫描工具健康检查
// @route GET /api/v1/scan-config/tools/:id/health
// @param c Gin上下文
func (h *ScanToolHandler) HealthCheckScanTool(c *gin.Context) {
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
			"operation":  "health_check_scan_tool",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.scan_tool.HealthCheckScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描工具ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层进行健康检查
	healthStatus, err := h.scanToolService.CheckScanToolHealth(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "health_check_scan_tool",
			"option":     "scanToolService.CheckScanToolHealth",
			"func_name":  "handler.scan_config.scan_tool.HealthCheckScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"timestamp":  logger.NowFormatted(),
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
				Message: "扫描工具健康检查失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "health_check_scan_tool", "扫描工具健康检查成功", logrus.InfoLevel, map[string]interface{}{
		"operation":     "health_check_scan_tool",
		"option":        "success",
		"func_name":     "handler.scan_config.scan_tool.HealthCheckScanTool",
		"client_ip":     clientIP,
		"user_agent":    userAgent,
		"request_id":    requestID,
		"tool_id":       id,
		"health_status": healthStatus,
		"timestamp":     logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "扫描工具健康检查成功",
		Data:    map[string]interface{}{"health_status": healthStatus},
	})
}

// InstallScanTool 安装扫描工具
// @route POST /api/v1/scan-config/tools/:id/install
// @param c Gin上下文
func (h *ScanToolHandler) InstallScanTool(c *gin.Context) {
	h.manageScanTool(c, "install", "安装扫描工具")
}

// UninstallScanTool 卸载扫描工具
// @route POST /api/v1/scan-config/tools/:id/uninstall
// @param c Gin上下文
func (h *ScanToolHandler) UninstallScanTool(c *gin.Context) {
	h.manageScanTool(c, "uninstall", "卸载扫描工具")
}

// GetScanToolMetrics 获取扫描工具指标
// @route GET /api/v1/scan-config/tools/:id/metrics
// @param c Gin上下文
func (h *ScanToolHandler) GetScanToolMetrics(c *gin.Context) {
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
			"operation":  "get_scan_tool_metrics",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.scan_tool.GetScanToolMetrics",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描工具ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层获取工具指标
	metrics, err := h.scanToolService.GetScanToolPerformance(c.Request.Context(), uint(id))
	if err != nil {
		logger.LogError(err, requestID, uint(id), clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_scan_tool_metrics",
			"option":     "scanToolService.GetPerformanceMetrics",
			"func_name":  "handler.scan_config.scan_tool.GetScanToolMetrics",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"timestamp":  logger.NowFormatted(),
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
				Message: "获取扫描工具指标失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "get_scan_tool_metrics", "获取扫描工具指标成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "get_scan_tool_metrics",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.GetScanToolMetrics",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"tool_id":    id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取扫描工具指标成功",
		Data:    metrics,
	})
}

// 私有方法：更新扫描工具状态
func (h *ScanToolHandler) updateScanToolStatus(c *gin.Context, status scan_config.ScanToolStatus, operation, message string) {
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
			"func_name":  "handler.scan_config.scan_tool.updateScanToolStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描工具ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用Service层更新状态
	var serviceErr error
	if status == scan_config.ScanToolStatusEnabled {
		serviceErr = h.scanToolService.EnableScanTool(c.Request.Context(), uint(id))
	} else {
		serviceErr = h.scanToolService.DisableScanTool(c.Request.Context(), uint(id))
	}

	if serviceErr != nil {
		logger.LogError(serviceErr, requestID, uint(id), clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  operation,
			"option":     "scanToolService.UpdateStatus",
			"func_name":  "handler.scan_config.scan_tool.updateScanToolStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"status":     status,
			"timestamp":  logger.NowFormatted(),
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
	logger.LogSystemEvent("scan_tool_handler", operation, message+"成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  operation,
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.updateScanToolStatus",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"tool_id":    id,
		"status":     status,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: message + "成功",
	})
}

// 私有方法：管理扫描工具（安装/卸载）
func (h *ScanToolHandler) manageScanTool(c *gin.Context, action, message string) {
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
			"operation":  action + "_scan_tool",
			"option":     "ParseUint",
			"func_name":  "handler.scan_config.scan_tool.manageScanTool",
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
			Message: "无效的扫描工具ID",
			Error:   err.Error(),
		})
		return
	}

	// 根据操作类型调用不同的Service方法
	var serviceErr error

	switch action {
	case "install":
		serviceErr = h.scanToolService.InstallScanTool(c.Request.Context(), uint(id))
	case "uninstall":
		serviceErr = h.scanToolService.UninstallScanTool(c.Request.Context(), uint(id))
	default:
		serviceErr = errors.New("不支持的操作类型")
	}

	if serviceErr != nil {
		logger.LogError(serviceErr, requestID, uint(id), clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  action + "_scan_tool",
			"option":     "scanToolService." + strings.Title(action) + "ScanTool",
			"func_name":  "handler.scan_config.scan_tool.manageScanTool",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"tool_id":    id,
			"action":     action,
			"timestamp":  logger.NowFormatted(),
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
	logger.LogSystemEvent("scan_tool_handler", action+"_scan_tool", message+"成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  action + "_scan_tool",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.manageScanTool",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"tool_id":    id,
		"action":     action,
		"timestamp":  logger.NowFormatted(),
	})

	// 构建响应
	response := model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: message + "成功",
	}

	// 返回成功响应
	c.JSON(http.StatusOK, response)
}

// 私有方法：验证扫描工具请求参数
func (h *ScanToolHandler) validateScanToolRequest(req *scan_config.ScanTool) error {
	// 基础字段验证
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("扫描工具名称不能为空")
	}

	if len(req.Name) > 100 {
		return errors.New("扫描工具名称长度不能超过100个字符")
	}

	if len(req.Description) > 500 {
		return errors.New("扫描工具描述长度不能超过500个字符")
	}

	// 工具类型验证
	if req.Type == "" {
		return errors.New("扫描工具类型不能为空")
	}

	// 版本验证
	if strings.TrimSpace(req.Version) == "" {
		return errors.New("扫描工具版本不能为空")
	}

	if len(req.Version) > 50 {
		return errors.New("扫描工具版本长度不能超过50个字符")
	}

	// 默认参数验证
	if strings.TrimSpace(req.DefaultParams) == "" {
		return errors.New("扫描工具默认参数不能为空")
	}

	// TODO: 可以添加更多的业务验证逻辑
	// 1. 验证配置JSON格式
	// 2. 验证工具类型是否支持
	// 3. 验证版本格式是否正确

	return nil
}

// GetAvailableScanTools 获取可用扫描工具
func (h *ScanToolHandler) GetAvailableScanTools(c *gin.Context) {
	tools, err := h.scanToolService.GetAvailableScanTools(c.Request.Context())
	if err != nil {
		logger.Error("获取可用扫描工具失败", map[string]interface{}{
			"path":      "/api/v1/scan-config/tools/available",
			"operation": "get_available_scan_tools",
			"option":    "scanToolService.GetAvailableScanTools",
			"func_name": "handler.scan_config.scan_tool.GetAvailableScanTools",
			"error":     err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取可用扫描工具失败"})
		return
	}

	logger.Info("获取可用扫描工具成功", map[string]interface{}{
		"path":      "/api/v1/scan-config/tools/available",
		"operation": "get_available_scan_tools",
		"option":    "success",
		"func_name": "handler.scan_config.scan_tool.GetAvailableScanTools",
		"count":     len(tools),
	})

	c.JSON(http.StatusOK, gin.H{
		"data":  tools,
		"count": len(tools),
	})
}

// BatchInstallScanTools 批量安装扫描工具
// @route POST /api/v1/scan-config/tools/batch-install
// @param c Gin上下文
func (h *ScanToolHandler) BatchInstallScanTools(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析请求参数
	var toolIDStrs []string
	if err := c.ShouldBindJSON(&toolIDStrs); err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "batch_install_scan_tools",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.scan_config.scan_tool.BatchInstallScanTools",
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

	// 转换字符串ID为uint类型
	var toolIDs []uint
	for _, idStr := range toolIDStrs {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
				"operation":  "batch_install_scan_tools",
				"option":     "ParseUint",
				"func_name":  "handler.scan_config.scan_tool.BatchInstallScanTools",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"invalid_id": idStr,
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "工具ID格式错误",
				Error:   fmt.Sprintf("无效的工具ID: %s", idStr),
			})
			return
		}
		toolIDs = append(toolIDs, uint(id))
	}

	// 调用Service层批量安装工具
	result, err := h.scanToolService.BatchInstallScanTools(c.Request.Context(), toolIDs)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "batch_install_scan_tools",
			"option":     "scanToolService.BatchInstallScanTools",
			"func_name":  "handler.scan_config.scan_tool.BatchInstallScanTools",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "批量安装扫描工具失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "batch_install_scan_tools", "批量安装扫描工具成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "batch_install_scan_tools",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.BatchInstallScanTools",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "批量安装扫描工具成功",
		Data:    result,
	})
}

// BatchUninstallScanTools 批量卸载扫描工具
// @route POST /api/v1/scan-config/tools/batch-uninstall
// @param c Gin上下文
func (h *ScanToolHandler) BatchUninstallScanTools(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 解析请求参数
	var toolIDStrs []string
	if err := c.ShouldBindJSON(&toolIDStrs); err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "batch_uninstall_scan_tools",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.scan_config.scan_tool.BatchUninstallScanTools",
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

	// 转换字符串ID为uint类型
	var toolIDs []uint
	for _, idStr := range toolIDStrs {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
				"operation":  "batch_uninstall_scan_tools",
				"option":     "ParseUint",
				"func_name":  "handler.scan_config.scan_tool.BatchUninstallScanTools",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"invalid_id": idStr,
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "工具ID格式错误",
				Error:   fmt.Sprintf("无效的工具ID: %s", idStr),
			})
			return
		}
		toolIDs = append(toolIDs, uint(id))
	}

	// 调用Service层批量卸载工具
	result, err := h.scanToolService.BatchUninstallScanTools(c.Request.Context(), toolIDs)
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "batch_uninstall_scan_tools",
			"option":     "scanToolService.BatchUninstallScanTools",
			"func_name":  "handler.scan_config.scan_tool.BatchUninstallScanTools",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "批量卸载扫描工具失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "batch_uninstall_scan_tools", "批量卸载扫描工具成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "batch_uninstall_scan_tools",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.BatchUninstallScanTools",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "批量卸载扫描工具成功",
		Data:    result,
	})
}

// GetSystemToolStatus 获取系统工具状态
// @route GET /api/v1/scan-config/tools/system-status
// @param c Gin上下文
func (h *ScanToolHandler) GetSystemToolStatus(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	// 调用Service层获取系统工具状态
	status, err := h.scanToolService.GetSystemToolStatus(c.Request.Context())
	if err != nil {
		logger.LogError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation":  "get_system_tool_status",
			"option":     "scanToolService.GetSystemToolStatus",
			"func_name":  "handler.scan_config.scan_tool.GetSystemToolStatus",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取系统工具状态失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_tool_handler", "get_system_tool_status", "获取系统工具状态成功", logrus.InfoLevel, map[string]interface{}{
		"operation":  "get_system_tool_status",
		"option":     "success",
		"func_name":  "handler.scan_config.scan_tool.GetSystemToolStatus",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取系统工具状态成功",
		Data:    status,
	})
}

// GetScanToolsByType 按类型获取扫描工具
func (h *ScanToolHandler) GetScanToolsByType(c *gin.Context) {
	toolType := c.Param("type")
	if toolType == "" {
		logger.Error("工具类型不能为空", map[string]interface{}{
			"path":      "/api/v1/scan-config/tools/type/:type",
			"operation": "get_scan_tools_by_type",
			"option":    "validate_type",
			"func_name": "handler.scan_config.scan_tool.GetScanToolsByType",
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "工具类型不能为空"})
		return
	}

	tools, err := h.scanToolService.GetScanToolsByType(c.Request.Context(), scan_config.ScanToolType(toolType))
	if err != nil {
		logger.Error("按类型获取扫描工具失败", map[string]interface{}{
			"path":      "/api/v1/scan-config/tools/type/:type",
			"operation": "get_scan_tools_by_type",
			"option":    "scanToolService.GetScanToolsByType",
			"func_name": "handler.scan_config.scan_tool.GetScanToolsByType",
			"type":      toolType,
			"error":     err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "按类型获取扫描工具失败"})
		return
	}

	logger.Info("按类型获取扫描工具成功", map[string]interface{}{
		"path":      "/api/v1/scan-config/tools/type/:type",
		"operation": "get_scan_tools_by_type",
		"option":    "success",
		"func_name": "handler.scan_config.scan_tool.GetScanToolsByType",
		"type":      toolType,
		"count":     len(tools),
	})

	c.JSON(http.StatusOK, gin.H{
		"data":  tools,
		"count": len(tools),
	})
}
