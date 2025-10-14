/**
 * 处理器:扫描规则处理器
 * @author: Linus Torvalds (AI Assistant)
 * @date: 2025.10.11
 * @description: 扫描规则管理处理器，提供扫描规则的CRUD操作、状态管理、规则测试等功能
 * @func: 处理扫描规则相关的HTTP请求
 */
package orchestrator

import (
	"encoding/json"
	"fmt"
	"neomaster/internal/model/system"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	scanConfigService "neomaster/internal/service/orchestrator"
)

// ScanRuleHandler 扫描规则处理器
// 这是一个"好品味"的处理器设计 - 职责单一，接口清晰
type ScanRuleHandler struct {
	scanRuleService scanConfigService.ScanRuleService // 扫描规则服务
}

// NewScanRuleHandler 创建扫描规则处理器实例
// @param scanRuleService 扫描规则服务接口
// @return *ScanRuleHandler 扫描规则处理器实例
func NewScanRuleHandler(scanRuleService scanConfigService.ScanRuleService) *ScanRuleHandler {
	return &ScanRuleHandler{
		scanRuleService: scanRuleService,
	}
}

// CreateScanRule 创建扫描规则
// @route POST /api/v1/orchestrator/rules
// @param c Gin上下文
func (h *ScanRuleHandler) CreateScanRule(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析请求参数
	var req orchestrator.CreateScanRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("创建扫描规则请求参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules",
			"operation":  "create_scan_rule",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.orchestrator.scan_rule.CreateScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
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
	if err := h.validateScanRuleRequest(&req); err != nil {
		logger.Error("创建扫描规则请求参数验证失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules",
			"operation":  "create_scan_rule",
			"option":     "validateScanRuleRequest",
			"func_name":  "handler.orchestrator.scan_rule.CreateScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数验证失败",
			Error:   err.Error(),
		})
		return
	}

	// 转换请求结构体为实体结构体
	rule := &orchestrator.ScanRule{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Category:    req.Category,
		Severity:    orchestrator.ScanRuleSeverity(req.Severity),
		Priority:    req.Priority,
		Status:      req.Status,
		IsBuiltIn:   req.IsBuiltIn,
	}

	// 处理config中的enabled字段
	if req.Config != nil {
		if enabled, exists := req.Config["enabled"]; exists {
			if enabledBool, ok := enabled.(bool); ok && enabledBool {
				rule.Status = orchestrator.ScanRuleStatusEnabled
			} else {
				rule.Status = orchestrator.ScanRuleStatusDisabled
			}
		}
	}

	// 序列化配置、条件和动作为JSON字符串
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			logger.Error("序列化规则配置失败", map[string]interface{}{
				"path":       "/api/v1/orchestrator/rules",
				"operation":  "create_scan_rule",
				"option":     "marshal_config",
				"func_name":  "handler.orchestrator.scan_rule.CreateScanRule",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"error":      err.Error(),
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "规则配置格式错误",
				Error:   err.Error(),
			})
			return
		}
		rule.Parameters = string(configJSON)
	}

	if len(req.Conditions) > 0 {
		conditionsJSON, err := json.Marshal(req.Conditions)
		if err != nil {
			logger.Error("序列化规则条件失败", map[string]interface{}{
				"path":       "/api/v1/orchestrator/rules",
				"operation":  "create_scan_rule",
				"option":     "marshal_conditions",
				"func_name":  "handler.orchestrator.scan_rule.CreateScanRule",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"error":      err.Error(),
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "规则条件格式错误",
				Error:   err.Error(),
			})
			return
		}
		rule.Condition = string(conditionsJSON)
	}

	if len(req.Actions) > 0 {
		actionsJSON, err := json.Marshal(req.Actions)
		if err != nil {
			logger.Error("序列化规则动作失败", map[string]interface{}{
				"path":       "/api/v1/orchestrator/rules",
				"operation":  "create_scan_rule",
				"option":     "marshal_actions",
				"func_name":  "handler.orchestrator.scan_rule.CreateScanRule",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"error":      err.Error(),
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "规则动作格式错误",
				Error:   err.Error(),
			})
			return
		}
		rule.Action = string(actionsJSON)
	}

	// 调用服务层创建扫描规则
	createdRule, err := h.scanRuleService.CreateScanRule(c.Request.Context(), rule)
	if err != nil {
		logger.Error("创建扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules",
			"operation":  "create_scan_rule",
			"option":     "scanRuleService.CreateScanRule",
			"func_name":  "handler.orchestrator.scan_rule.CreateScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"rule_name":  req.Name,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "创建扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("创建扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules",
		"operation":  "create_scan_rule",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.CreateScanRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"rule_id":    createdRule.ID,
		"rule_name":  createdRule.Name,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "创建扫描规则成功",
		Data:    createdRule,
	})
}

// GetScanRule 获取扫描规则详情
// @route GET /api/v1/orchestrator/rules/:id
// @param c Gin上下文
func (h *ScanRuleHandler) GetScanRule(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("获取扫描规则ID参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "get_scan_rule",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层获取扫描规则
	rule, err := h.scanRuleService.GetScanRule(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error("获取扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "get_scan_rule",
			"option":     "scanRuleService.GetScanRule",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("获取扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/:id",
		"operation":  "get_scan_rule",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.GetScanRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"rule_name":  rule.Name,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取扫描规则成功",
		Data:    rule,
	})
}

// GetScanRuleList 获取扫描规则列表（别名方法，兼容路由）
// @Summary 获取扫描规则列表
// @Description 分页获取扫描规则列表，支持按类型、状态、严重程度过滤
// @Tags 扫描规则管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认10"
// @Param type query string false "规则类型过滤"
// @Param status query string false "状态过滤"
// @Param severity query string false "严重程度过滤"
// @Param category query string false "分类过滤"
// @Param keyword query string false "关键词搜索"
// @Param is_built_in query bool false "是否内置规则"
// @Param created_by query uint false "创建者ID"
// @Success 200 {object} model.APIResponse{data=model.PaginatedResponse{items=[]orchestrator.ScanRule}} "获取成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/orchestrator/rules [get]
func (h *ScanRuleHandler) GetScanRuleList(c *gin.Context) {
	// 直接调用ListScanRules方法
	h.ListScanRules(c)
}

// BatchImportScanRules 批量导入扫描规则
// @Summary 批量导入扫描规则
// @Description 批量导入扫描规则，支持JSON、YAML、XML格式
// @Tags 扫描规则管理
// @Accept json
// @Produce json
// @Param request body orchestrator.ImportScanRulesRequest true "导入请求参数"
// @Success 200 {object} model.APIResponse{data=map[string]interface{}} "导入成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/admin/orchestrator/rules/batch-import [post]
func (h *ScanRuleHandler) BatchImportScanRules(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析请求参数
	var req orchestrator.ImportScanRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("解析批量导入扫描规则请求参数失败", map[string]interface{}{
			"path":       "/api/v1/admin/orchestrator/rules/batch-import",
			"operation":  "batch_import_scan_rules",
			"option":     "parse_request",
			"func_name":  "handler.orchestrator.scan_rule.BatchImportScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
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

	// 调用服务层批量导入扫描规则
	result, err := h.scanRuleService.BatchImportScanRules(c.Request.Context(), &req)
	if err != nil {
		logger.Error("批量导入扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/admin/orchestrator/rules/batch-import",
			"operation":  "batch_import_scan_rules",
			"option":     "scanRuleService.BatchImportScanRules",
			"func_name":  "handler.orchestrator.scan_rule.BatchImportScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "批量导入扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("批量导入扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/admin/orchestrator/rules/batch-import",
		"operation":  "batch_import_scan_rules",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.BatchImportScanRules",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "批量导入扫描规则成功",
		Data:    result,
	})
}

// BatchEnableScanRules 批量启用扫描规则
// @Summary 批量启用扫描规则
// @Description 批量启用指定的扫描规则
// @Tags 扫描规则管理
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "规则ID列表"
// @Success 200 {object} model.APIResponse{data=map[string]interface{}} "启用成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/admin/orchestrator/rules/batch-enable [post]
func (h *ScanRuleHandler) BatchEnableScanRules(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析请求参数
	var req struct {
		RuleIDs []uint `json:"rule_ids" validate:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("解析批量启用扫描规则请求参数失败", map[string]interface{}{
			"path":       "/api/v1/admin/orchestrator/rules/batch-enable",
			"operation":  "batch_enable_scan_rules",
			"option":     "parse_request",
			"func_name":  "handler.orchestrator.scan_rule.BatchEnableScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
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

	// 调用服务层批量启用规则
	result, err := h.scanRuleService.BatchUpdateScanRuleStatus(c.Request.Context(), req.RuleIDs, orchestrator.ScanRuleStatusEnabled)
	if err != nil {
		logger.Error("批量启用扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/admin/orchestrator/rules/batch-enable",
			"operation":  "batch_enable_scan_rules",
			"option":     "scanRuleService.BatchUpdateScanRuleStatus",
			"func_name":  "handler.orchestrator.scan_rule.BatchEnableScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"rule_ids":   req.RuleIDs,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "批量启用扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("批量启用扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/admin/orchestrator/rules/batch-enable",
		"operation":  "batch_enable_scan_rules",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.BatchEnableScanRules",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"rule_ids":   req.RuleIDs,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "批量启用扫描规则成功",
		Data:    result,
	})
}

// BatchDisableScanRules 批量禁用扫描规则
// @Summary 批量禁用扫描规则
// @Description 批量禁用指定的扫描规则
// @Tags 扫描规则管理
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "规则ID列表"
// @Success 200 {object} model.APIResponse{data=map[string]interface{}} "禁用成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/admin/orchestrator/rules/batch-disable [post]
func (h *ScanRuleHandler) BatchDisableScanRules(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析请求参数
	var req struct {
		RuleIDs []uint `json:"rule_ids" validate:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("解析批量禁用扫描规则请求参数失败", map[string]interface{}{
			"path":       "/api/v1/admin/orchestrator/rules/batch-disable",
			"operation":  "batch_disable_scan_rules",
			"option":     "parse_request",
			"func_name":  "handler.orchestrator.scan_rule.BatchDisableScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
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

	// 调用服务层批量禁用规则
	result, err := h.scanRuleService.BatchUpdateScanRuleStatus(c.Request.Context(), req.RuleIDs, orchestrator.ScanRuleStatusDisabled)
	if err != nil {
		logger.Error("批量禁用扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/admin/orchestrator/rules/batch-disable",
			"operation":  "batch_disable_scan_rules",
			"option":     "scanRuleService.BatchUpdateScanRuleStatus",
			"func_name":  "handler.orchestrator.scan_rule.BatchDisableScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"rule_ids":   req.RuleIDs,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "批量禁用扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("批量禁用扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/admin/orchestrator/rules/batch-disable",
		"operation":  "batch_disable_scan_rules",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.BatchDisableScanRules",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"rule_ids":   req.RuleIDs,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "批量禁用扫描规则成功",
		Data:    result,
	})
}

// UpdateScanRule 更新扫描规则
// @route PUT /api/v1/orchestrator/rules/:id
// @param c Gin上下文
func (h *ScanRuleHandler) UpdateScanRule(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("更新扫描规则ID参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "update_scan_rule",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.scan_rule.UpdateScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 解析请求体
	var req orchestrator.UpdateScanRuleRequest
	if err1 := c.ShouldBindJSON(&req); err1 != nil {
		logger.Error("更新扫描规则请求参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "update_scan_rule",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.orchestrator.scan_rule.UpdateScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err1.Error(),
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

	// 转换请求结构体为实体结构体
	rule := &orchestrator.ScanRule{}

	// 只更新非空字段
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Type != nil {
		rule.Type = *req.Type
	}
	if req.Category != nil {
		rule.Category = *req.Category
	}
	if req.Severity != nil {
		rule.Severity = orchestrator.ScanRuleSeverity(*req.Severity)
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Status != nil {
		rule.Status = *req.Status
	}

	// 序列化配置、条件和动作为JSON字符串
	if req.Config != nil {
		configJSON, err2 := json.Marshal(req.Config)
		if err2 != nil {
			logger.Error("序列化规则配置失败", map[string]interface{}{
				"path":       "/api/v1/orchestrator/rules/:id",
				"operation":  "update_scan_rule",
				"option":     "marshal_config",
				"func_name":  "handler.orchestrator.scan_rule.UpdateScanRule",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"id":         id,
				"error":      err2.Error(),
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "规则配置格式错误",
				Error:   err2.Error(),
			})
			return
		}
		rule.Parameters = string(configJSON)
	}

	if len(req.Conditions) > 0 {
		conditionsJSON, err3 := json.Marshal(req.Conditions)
		if err3 != nil {
			logger.Error("序列化规则条件失败", map[string]interface{}{
				"path":       "/api/v1/orchestrator/rules/:id",
				"operation":  "update_scan_rule",
				"option":     "marshal_conditions",
				"func_name":  "handler.orchestrator.scan_rule.UpdateScanRule",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"id":         id,
				"error":      err3.Error(),
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "规则条件格式错误",
				Error:   err3.Error(),
			})
			return
		}
		rule.Condition = string(conditionsJSON)
	}

	if len(req.Actions) > 0 {
		actionsJSON, err4 := json.Marshal(req.Actions)
		if err4 != nil {
			logger.Error("序列化规则动作失败", map[string]interface{}{
				"path":       "/api/v1/orchestrator/rules/:id",
				"operation":  "update_scan_rule",
				"option":     "marshal_actions",
				"func_name":  "handler.orchestrator.scan_rule.UpdateScanRule",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"id":         id,
				"error":      err4.Error(),
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "规则动作格式错误",
				Error:   err4.Error(),
			})
			return
		}
		rule.Action = string(actionsJSON)
	}

	// 调用服务层更新扫描规则
	updatedRule, err := h.scanRuleService.UpdateScanRule(c.Request.Context(), uint(id), rule)
	if err != nil {
		logger.Error("更新扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "update_scan_rule",
			"option":     "scanRuleService.UpdateScanRule",
			"func_name":  "handler.orchestrator.scan_rule.UpdateScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "更新扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("更新扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/:id",
		"operation":  "update_scan_rule",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.UpdateScanRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"rule_id":    updatedRule.ID,
		"rule_name":  updatedRule.Name,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "更新扫描规则成功",
		Data:    updatedRule,
	})
}

// DeleteScanRule 删除扫描规则
// @route DELETE /api/v1/orchestrator/rules/:id
// @param c Gin上下文
func (h *ScanRuleHandler) DeleteScanRule(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("删除扫描规则ID参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "delete_scan_rule",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.scan_rule.DeleteScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层删除扫描规则
	err = h.scanRuleService.DeleteScanRule(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error("删除扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "delete_scan_rule",
			"option":     "scanRuleService.DeleteScanRule",
			"func_name":  "handler.orchestrator.scan_rule.DeleteScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "删除扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("删除扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/:id",
		"operation":  "delete_scan_rule",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.DeleteScanRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "删除扫描规则成功",
	})
}

// ListScanRules 获取扫描规则列表
// @route GET /api/v1/orchestrator/rules
// @param c Gin上下文
func (h *ScanRuleHandler) ListScanRules(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析查询参数
	var req orchestrator.ListScanRulesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Error("获取扫描规则列表查询参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules",
			"operation":  "list_scan_rules",
			"option":     "ShouldBindQuery",
			"func_name":  "handler.orchestrator.scan_rule.ListScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "查询参数格式错误",
			Error:   err.Error(),
		})
		return
	}

	// 计算offset和limit
	offset := (req.Page - 1) * req.PageSize
	limit := req.PageSize

	// 转换Severity字符串为ScanRuleSeverity类型
	var severity *orchestrator.ScanRuleSeverity
	if req.Severity != nil {
		switch *req.Severity {
		case "low":
			s := orchestrator.ScanRuleSeverityLow
			severity = &s
		case "medium":
			s := orchestrator.ScanRuleSeverityMedium
			severity = &s
		case "high":
			s := orchestrator.ScanRuleSeverityHigh
			severity = &s
		case "critical":
			s := orchestrator.ScanRuleSeverityCritical
			severity = &s
		}
	}

	// 调用服务层获取扫描规则列表
	rules, total, err := h.scanRuleService.ListScanRules(c.Request.Context(), offset, limit, req.Type, req.Status, severity)
	if err != nil {
		logger.Error("获取扫描规则列表失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules",
			"operation":  "list_scan_rules",
			"option":     "scanRuleService.ListScanRules",
			"func_name":  "handler.orchestrator.scan_rule.ListScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取扫描规则列表失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("获取扫描规则列表成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules",
		"operation":  "list_scan_rules",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.ListScanRules",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"total":      total,
		"count":      len(rules),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取扫描规则列表成功",
		Data: map[string]interface{}{
			"rules": rules,
			"total": total,
		},
	})
}

// EnableScanRule 启用扫描规则
// @route POST /api/v1/orchestrator/rules/:id/enable
// @param c Gin上下文
func (h *ScanRuleHandler) EnableScanRule(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("启用扫描规则ID参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/enable",
			"operation":  "enable_scan_rule",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.scan_rule.EnableScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层启用扫描规则
	err = h.scanRuleService.EnableScanRule(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error("启用扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/enable",
			"operation":  "enable_scan_rule",
			"option":     "scanRuleService.EnableScanRule",
			"func_name":  "handler.orchestrator.scan_rule.EnableScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "启用扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("启用扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/:id/enable",
		"operation":  "enable_scan_rule",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.EnableScanRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "启用扫描规则成功",
	})
}

// DisableScanRule 禁用扫描规则
// @route POST /api/v1/orchestrator/rules/:id/disable
// @param c Gin上下文
func (h *ScanRuleHandler) DisableScanRule(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("禁用扫描规则ID参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/disable",
			"operation":  "disable_scan_rule",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.scan_rule.DisableScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层禁用扫描规则
	err = h.scanRuleService.DisableScanRule(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error("禁用扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/disable",
			"operation":  "disable_scan_rule",
			"option":     "scanRuleService.DisableScanRule",
			"func_name":  "handler.orchestrator.scan_rule.DisableScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "禁用扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("禁用扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/:id/disable",
		"operation":  "disable_scan_rule",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.DisableScanRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "禁用扫描规则成功",
	})
}

// MatchScanRules 匹配扫描规则
// @route POST /api/v1/orchestrator/rules/match
// @param c Gin上下文
func (h *ScanRuleHandler) MatchScanRules(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析请求体
	var req orchestrator.MatchScanRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogError(err, c.Request.RequestURI, 0, "", "match_scan_rules", "HANDLER", map[string]interface{}{
			"func_name":  "handler.orchestrator.scan_rule_handler.MatchScanRules",
			"operation":  "match_scan_rules",
			"option":     "bind_request",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
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

	// 调用服务层匹配扫描规则
	rules, err := h.scanRuleService.MatchScanRules(c.Request.Context(), &req)
	if err != nil {
		logger.Error("匹配扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/match",
			"operation":  "match_scan_rule",
			"option":     "scanRuleService.MatchScanRule",
			"func_name":  "handler.orchestrator.scan_rule.MatchScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "匹配扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("匹配扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/match",
		"operation":  "match_scan_rule",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.MatchScanRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"count":      len(rules),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "匹配扫描规则成功",
		Data:    rules,
	})
}

// ImportScanRules 导入扫描规则
// @route POST /api/v1/orchestrator/rules/import
// @param c Gin上下文
func (h *ScanRuleHandler) ImportScanRules(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析请求体
	var req orchestrator.ImportScanRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("导入扫描规则请求参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/import",
			"operation":  "import_scan_rules",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.orchestrator.scan_rule.ImportScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
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

	// 解析导入数据
	var rules []*orchestrator.ScanRule
	switch req.Format {
	case "json":
		if err := json.Unmarshal([]byte(req.Data), &rules); err != nil {
			logger.Error("解析JSON格式扫描规则数据失败", map[string]interface{}{
				"path":       "/api/v1/orchestrator/rules/import",
				"operation":  "import_scan_rules",
				"option":     "json.Unmarshal",
				"func_name":  "handler.orchestrator.scan_rule.ImportScanRules",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"error":      err.Error(),
				"timestamp":  logger.NowFormatted(),
			})
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: "JSON格式数据解析失败",
				Error:   err.Error(),
			})
			return
		}
	default:
		logger.Error("不支持的导入格式", map[string]interface{}{
			"path":      "/api/v1/orchestrator/rules/import",
			"operation": "import_scan_rules",
			"option":    "unsupported_format",
			"func_name": "handler.orchestrator.scan_rule.ImportScanRules",
			"format":    req.Format,
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "不支持的导入格式",
			Error:   fmt.Sprintf("格式 %s 暂不支持", req.Format),
		})
		return
	}

	// 调用服务层导入扫描规则
	result, err := h.scanRuleService.ImportScanRules(c.Request.Context(), rules)
	if err != nil {
		logger.Error("导入扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/import",
			"operation":  "import_scan_rules",
			"option":     "scanRuleService.ImportScanRules",
			"func_name":  "handler.orchestrator.scan_rule.ImportScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "导入扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("导入扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/import",
		"operation":  "import_scan_rules",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.ImportScanRules",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "导入扫描规则成功",
		Data:    result,
	})
}

// ExportScanRules 导出扫描规则
// @route GET /api/v1/orchestrator/rules/export
// @param c Gin上下文
func (h *ScanRuleHandler) ExportScanRules(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析查询参数
	var req orchestrator.ExportScanRulesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Error("导出扫描规则查询参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/export",
			"operation":  "export_scan_rules",
			"option":     "ShouldBindQuery",
			"func_name":  "handler.orchestrator.scan_rule.ExportScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "查询参数格式错误",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层导出扫描规则
	rules, err := h.scanRuleService.ExportScanRules(c.Request.Context(), req.RuleType, req.Status)
	if err != nil {
		logger.Error("导出扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/export",
			"operation":  "export_scan_rules",
			"option":     "scanRuleService.ExportScanRules",
			"func_name":  "handler.orchestrator.scan_rule.ExportScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "导出扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("导出扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/export",
		"operation":  "export_scan_rules",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.ExportScanRules",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 根据格式返回数据
	switch req.Format {
	case "json":
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=scan_rules.json")
		c.JSON(http.StatusOK, rules)
	case "yaml":
		c.Header("Content-Type", "application/x-yaml")
		c.Header("Content-Disposition", "attachment; filename=scan_rules.yaml")
		// 这里需要将rules转换为YAML格式
		c.JSON(http.StatusOK, rules) // 临时使用JSON格式
	case "xml":
		c.Header("Content-Type", "application/xml")
		c.Header("Content-Disposition", "attachment; filename=scan_rules.xml")
		// 这里需要将rules转换为XML格式
		c.JSON(http.StatusOK, rules) // 临时使用JSON格式
	default:
		c.JSON(http.StatusOK, system.APIResponse{
			Code:    http.StatusOK,
			Status:  "success",
			Message: "导出扫描规则成功",
			Data:    rules,
		})
	}
}

// TestScanRule 测试扫描规则
// @route POST /api/v1/orchestrator/rules/:id/test
// @param c Gin上下文
func (h *ScanRuleHandler) TestScanRule(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("测试扫描规则ID参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/test",
			"operation":  "test_scan_rule",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.scan_rule.TestScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 解析请求体
	var req orchestrator.TestScanRuleRequest
	if err1 := c.ShouldBindJSON(&req); err1 != nil {
		logger.Error("测试扫描规则请求参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/test",
			"operation":  "test_scan_rule",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.orchestrator.scan_rule.TestScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err1.Error(),
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

	// 首先获取规则信息
	rule, err := h.scanRuleService.GetScanRule(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error("获取扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/test",
			"operation":  "test_scan_rule",
			"option":     "scanRuleService.GetScanRule",
			"func_name":  "handler.orchestrator.scan_rule.TestScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "error",
			Message: "扫描规则不存在",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层测试扫描规则
	result, err := h.scanRuleService.TestScanRule(c.Request.Context(), rule, req.Target)
	if err != nil {
		logger.Error("测试扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/test",
			"operation":  "test_scan_rule",
			"option":     "scanRuleService.TestScanRule",
			"func_name":  "handler.orchestrator.scan_rule.TestScanRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "测试扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("测试扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/:id/test",
		"operation":  "test_scan_rule",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.TestScanRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "测试扫描规则成功",
		Data:    result,
	})
}

// GetScanRulesByType 按类型获取扫描规则
// @route GET /api/v1/orchestrator/rules/type/:type
// @param c Gin上下文
func (h *ScanRuleHandler) GetScanRulesByType(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	ruleType := c.Param("type")
	if ruleType == "" {
		logger.Error("按类型获取扫描规则类型参数为空", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/type/:type",
			"operation":  "get_scan_rules_by_type",
			"option":     "param_validation",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRulesByType",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "扫描规则类型不能为空",
		})
		return
	}

	// 调用服务层按类型获取扫描规则
	rules, err := h.scanRuleService.GetScanRulesByType(c.Request.Context(), orchestrator.ScanRuleType(ruleType))
	if err != nil {
		logger.Error("按类型获取扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/type/:type",
			"operation":  "get_scan_rules_by_type",
			"option":     "scanRuleService.GetScanRulesByType",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRulesByType",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"rule_type":  ruleType,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "按类型获取扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("按类型获取扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/type/:type",
		"operation":  "get_scan_rules_by_type",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.GetScanRulesByType",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"rule_type":  ruleType,
		"count":      len(rules),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "按类型获取扫描规则成功",
		Data:    rules,
	})
}

// GetScanRulesBySeverity 按严重程度获取扫描规则
// @route GET /api/v1/orchestrator/rules/severity/:severity
// @param c Gin上下文
func (h *ScanRuleHandler) GetScanRulesBySeverity(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	severity := c.Param("severity")
	if severity == "" {
		logger.Error("按严重程度获取扫描规则严重程度参数为空", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/severity/:severity",
			"operation":  "get_scan_rules_by_severity",
			"option":     "param_validation",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRulesBySeverity",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "扫描规则严重程度不能为空",
		})
		return
	}

	// 调用服务层按严重程度获取扫描规则
	rules, err := h.scanRuleService.GetScanRulesBySeverity(c.Request.Context(), orchestrator.ScanRuleSeverity(severity))
	if err != nil {
		logger.Error("按严重程度获取扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/severity/:severity",
			"operation":  "get_scan_rules_by_severity",
			"option":     "scanRuleService.GetScanRulesBySeverity",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRulesBySeverity",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"severity":   severity,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "按严重程度获取扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("按严重程度获取扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/severity/:severity",
		"operation":  "get_scan_rules_by_severity",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.GetScanRulesBySeverity",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"severity":   severity,
		"count":      len(rules),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "按严重程度获取扫描规则成功",
		Data:    rules,
	})
}

// GetActiveScanRules 获取活跃扫描规则
// @route GET /api/v1/orchestrator/rules/active
// @param c Gin上下文
func (h *ScanRuleHandler) GetActiveScanRules(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 调用服务层获取活跃扫描规则
	rules, err := h.scanRuleService.GetActiveScanRules(c.Request.Context())
	if err != nil {
		logger.Error("获取活跃扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/active",
			"operation":  "get_active_scan_rules",
			"option":     "scanRuleService.GetActiveScanRules",
			"func_name":  "handler.orchestrator.scan_rule.GetActiveScanRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取活跃扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("获取活跃扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/active",
		"operation":  "get_active_scan_rules",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.GetActiveScanRules",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"count":      len(rules),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取活跃扫描规则成功",
		Data:    rules,
	})
}

// GetScanRuleMetrics 获取扫描规则指标
// @route GET /api/v1/orchestrator/rules/:id/metrics
// @param c Gin上下文
func (h *ScanRuleHandler) GetScanRuleMetrics(c *gin.Context) {
	// 获取请求上下文信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("获取扫描规则指标ID参数解析失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/metrics",
			"operation":  "get_scan_rule_metrics",
			"option":     "ParseUint",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRuleMetrics",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层获取扫描规则指标
	metrics, err := h.scanRuleService.GetScanRuleMetrics(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error("获取扫描规则指标失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id/metrics",
			"operation":  "get_scan_rule_metrics",
			"option":     "scanRuleService.GetScanRuleMetrics",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRuleMetrics",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取扫描规则指标失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("获取扫描规则指标成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/:id/metrics",
		"operation":  "get_scan_rule_metrics",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.GetScanRuleMetrics",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取扫描规则指标成功",
		Data:    metrics,
	})
}

// ==================== 私有辅助方法 ====================

// validateScanRuleRequest 验证扫描规则请求参数
// @param req 请求参数
// @return error 验证错误
func (h *ScanRuleHandler) validateScanRuleRequest(req *orchestrator.CreateScanRuleRequest) error {
	// 验证规则名称
	if req.Name == "" {
		return system.NewValidationError("扫描规则名称不能为空")
	}

	// 验证规则类型
	if req.Type == "" {
		return system.NewValidationError("扫描规则类型不能为空")
	}

	// 验证规则配置
	if len(req.Config) == 0 {
		return system.NewValidationError("扫描规则配置不能为空")
	}

	return nil
}

// GetScanRuleByID 根据ID获取扫描规则详情
// @Summary 根据ID获取扫描规则详情
// @Description 根据规则ID获取扫描规则的详细信息
// @Tags 扫描规则管理
// @Accept json
// @Produce json
// @Param id path uint true "规则ID"
// @Success 200 {object} model.APIResponse{data=orchestrator.ScanRule} "获取成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 404 {object} model.APIResponse "规则不存在"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/orchestrator/rules/{id} [get]
func (h *ScanRuleHandler) GetScanRuleByID(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 获取路径参数中的ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("解析扫描规则ID失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "get_scan_rule_by_id",
			"option":     "parse_id",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRuleByID",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的扫描规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层获取扫描规则
	rule, err := h.scanRuleService.GetScanRule(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error("获取扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/orchestrator/rules/:id",
			"operation":  "get_scan_rule_by_id",
			"option":     "scanRuleService.GetScanRule",
			"func_name":  "handler.orchestrator.scan_rule.GetScanRuleByID",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取扫描规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("获取扫描规则成功", map[string]interface{}{
		"path":       "/api/v1/orchestrator/rules/:id",
		"operation":  "get_scan_rule_by_id",
		"option":     "success",
		"func_name":  "handler.orchestrator.scan_rule.GetScanRuleByID",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"rule_name":  rule.Name,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取扫描规则成功",
		Data:    rule,
	})
}
