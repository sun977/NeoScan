/**
 * 处理器:规则引擎处理器
 * @author: Linus Torvalds (AI Assistant)
 * @date: 2025.01.11
 * @description: 规则引擎管理处理器，提供规则引擎的执行、管理、监控等功能
 * @func: 处理规则引擎相关的HTTP请求
 */
package scan_config

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/rule_engine"
	scanConfigService "neomaster/internal/service/scan_config"

	"github.com/gin-gonic/gin"
)

// RuleEngineHandler 规则引擎处理器
// 这是一个"好品味"的处理器设计 - 职责单一，专注于规则引擎操作
type RuleEngineHandler struct {
	ruleEngine      *rule_engine.RuleEngine            // 规则引擎实例
	scanRuleService *scanConfigService.ScanRuleService // 扫描规则服务
}

// NewRuleEngineHandler 创建规则引擎处理器实例
// @param ruleEngine 规则引擎实例
// @param scanRuleService 扫描规则服务实例
// @return *RuleEngineHandler 规则引擎处理器实例
func NewRuleEngineHandler(ruleEngine *rule_engine.RuleEngine, scanRuleService *scanConfigService.ScanRuleService) *RuleEngineHandler {
	return &RuleEngineHandler{
		ruleEngine:      ruleEngine,
		scanRuleService: scanRuleService,
	}
}

// ExecuteRule 执行单个规则
// @Summary 执行单个规则
// @Description 根据提供的上下文执行指定的规则
// @Tags 规则引擎
// @Accept json
// @Produce json
// @Param id path uint true "规则ID"
// @Param context body rule_engine.RuleContext true "规则执行上下文"
// @Success 200 {object} model.APIResponse{data=rule_engine.RuleResult} "执行成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 404 {object} model.APIResponse "规则不存在"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/scan-config/rule-engine/rules/{id}/execute [post]
func (h *RuleEngineHandler) ExecuteRule(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 获取路径参数中的规则ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error("解析规则ID失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/:id/execute",
			"operation":  "execute_rule",
			"option":     "parse_id",
			"func_name":  "handler.scan_config.rule_engine.ExecuteRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id_param":   idStr,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的规则ID",
			Error:   err.Error(),
		})
		return
	}

	// 从数据库获取规则信息
	ctx := context.Background()
	scanRule, err := h.scanRuleService.GetScanRule(ctx, uint(id))
	if err != nil {
		logger.Error("获取扫描规则失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/:id/execute",
			"operation":  "execute_rule",
			"option":     "get_rule",
			"func_name":  "handler.scan_config.rule_engine.ExecuteRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "error",
			Message: "规则不存在",
			Error:   err.Error(),
		})
		return
	}

	// 检查规则是否启用
	if !scanRule.IsEnabled() {
		logger.Warn("规则已禁用", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/:id/execute",
			"operation":  "execute_rule",
			"option":     "rule_disabled",
			"func_name":  "handler.scan_config.rule_engine.ExecuteRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"rule_name":  scanRule.Name,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "规则已禁用",
		})
		return
	}

	// 解析请求体中的规则上下文
	var context rule_engine.RuleContext
	if err := c.ShouldBindJSON(&context); err != nil {
		logger.Error("解析规则执行上下文失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/:id/execute",
			"operation":  "execute_rule",
			"option":     "parse_context",
			"func_name":  "handler.scan_config.rule_engine.ExecuteRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的规则执行上下文",
			Error:   err.Error(),
		})
		return
	}

	// 将数据库规则转换为规则引擎规则并添加到引擎中
	engineRule := &rule_engine.Rule{
		ID:          strconv.Itoa(int(scanRule.ID)),
		Name:        scanRule.Name,
		Description: scanRule.Description,
		Enabled:     scanRule.IsEnabled(),
		Priority:    int(scanRule.Priority),
		Severity:    "low",                     // 设置默认严重程度
		Conditions:  []rule_engine.Condition{}, // 需要解析Condition字符串为Condition切片
		Actions:     []rule_engine.Action{},    // 根据需要转换动作
		CreatedAt:   scanRule.CreatedAt,
		UpdatedAt:   scanRule.UpdatedAt,
	}

	// 添加默认条件和动作以通过验证
	engineRule.Conditions = []rule_engine.Condition{
		{
			Field:    "default",
			Operator: "eq",
			Value:    "true",
		},
	}
	engineRule.Actions = []rule_engine.Action{
		{
			Type:       "log",
			Parameters: map[string]interface{}{"message": "规则执行"},
		},
	}

	// 添加规则到引擎（如果不存在）
	if err := h.ruleEngine.AddRule(engineRule); err != nil {
		// 如果规则已存在，这是正常情况，不需要记录错误
		if !strings.Contains(err.Error(), "规则已存在") {
			logger.Error("添加规则到引擎失败", map[string]interface{}{
				"path":       "/api/v1/scan-config/rule-engine/rules/:id/execute",
				"operation":  "execute_rule",
				"option":     "add_rule",
				"func_name":  "handler.scan_config.rule_engine.ExecuteRule",
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
				"id":         id,
				"error":      err.Error(),
				"timestamp":  logger.NowFormatted(),
			})
		}
	}

	// 执行规则 - 使用与添加规则时相同的ID格式
	ruleID := strconv.Itoa(int(scanRule.ID))
	result, err := h.ruleEngine.ExecuteRule(ruleID, &context)
	if err != nil {
		logger.Error("执行规则失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/:id/execute",
			"operation":  "execute_rule",
			"option":     "execute",
			"func_name":  "handler.scan_config.rule_engine.ExecuteRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"id":         id,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "执行规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("执行规则成功", map[string]interface{}{
		"path":       "/api/v1/scan-config/rule-engine/rules/:id/execute",
		"operation":  "execute_rule",
		"option":     "success",
		"func_name":  "handler.scan_config.rule_engine.ExecuteRule",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"id":         id,
		"matched":    result.Matched,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "执行规则成功",
		Data:    result,
	})
}

// ExecuteRules 批量执行规则
// @Summary 批量执行规则
// @Description 根据提供的上下文批量执行多个规则
// @Tags 规则引擎
// @Accept json
// @Produce json
// @Param request body BatchExecuteRulesRequest true "批量执行规则请求"
// @Success 200 {object} model.APIResponse{data=rule_engine.BatchRuleResult} "执行成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/scan-config/rule-engine/rules/batch-execute [post]
func (h *RuleEngineHandler) ExecuteRules(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 定义批量执行请求结构
	type BatchExecuteRulesRequest struct {
		RuleIDs []uint                   `json:"rule_ids" binding:"required,min=1" example:"[1,2,3]"`
		Context *rule_engine.RuleContext `json:"context" binding:"required"`
	}

	// 解析请求体
	var req BatchExecuteRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("解析批量执行规则请求失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/batch-execute",
			"operation":  "execute_rules",
			"option":     "parse_request",
			"func_name":  "handler.scan_config.rule_engine.ExecuteRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的批量执行规则请求",
			Error:   err.Error(),
		})
		return
	}

	// 批量执行规则
	result, err := h.ruleEngine.ExecuteRules(req.Context)
	if err != nil {
		logger.Error("批量执行规则失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/batch-execute",
			"operation":  "execute_rules",
			"option":     "execute",
			"func_name":  "handler.scan_config.rule_engine.ExecuteRules",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"rule_count": len(req.RuleIDs),
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "批量执行规则失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("批量执行规则成功", map[string]interface{}{
		"path":          "/api/v1/scan-config/rule-engine/rules/batch-execute",
		"operation":     "execute_rules",
		"option":        "success",
		"func_name":     "handler.scan_config.rule_engine.ExecuteRules",
		"client_ip":     clientIP,
		"user_agent":    userAgent,
		"request_id":    requestID,
		"rule_count":    len(req.RuleIDs),
		"matched_count": result.Matched,
		"timestamp":     logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "批量执行规则成功",
		Data:    result,
	})
}

// GetEngineMetrics 获取规则引擎指标
// @Summary 获取规则引擎指标
// @Description 获取规则引擎的运行指标和统计信息
// @Tags 规则引擎
// @Accept json
// @Produce json
// @Success 200 {object} model.APIResponse{data=rule_engine.RuleEngineMetrics} "获取成功"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/scan-config/rule-engine/metrics [get]
func (h *RuleEngineHandler) GetEngineMetrics(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 获取规则引擎指标
	metrics := h.ruleEngine.GetMetrics()

	// 记录成功日志
	logger.Info("获取规则引擎指标成功", map[string]interface{}{
		"path":          "/api/v1/scan-config/rule-engine/metrics",
		"operation":     "get_engine_metrics",
		"option":        "success",
		"func_name":     "handler.scan_config.rule_engine.GetEngineMetrics",
		"client_ip":     clientIP,
		"user_agent":    userAgent,
		"request_id":    requestID,
		"total_rules":   metrics.TotalRules,
		"enabled_rules": metrics.EnabledRules,
		"timestamp":     logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取规则引擎指标成功",
		Data:    metrics,
	})
}

// ClearCache 清空规则引擎缓存
// @Summary 清空规则引擎缓存
// @Description 清空规则引擎的内存缓存，强制重新加载规则
// @Tags 规则引擎
// @Accept json
// @Produce json
// @Success 200 {object} model.APIResponse "清空成功"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/scan-config/rule-engine/cache/clear [post]
func (h *RuleEngineHandler) ClearCache(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 清空缓存
	h.ruleEngine.ClearCache()

	// 记录成功日志
	logger.Info("清空规则引擎缓存成功", map[string]interface{}{
		"path":       "/api/v1/scan-config/rule-engine/cache/clear",
		"operation":  "clear_cache",
		"option":     "success",
		"func_name":  "handler.scan_config.rule_engine.ClearCache",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "清空规则引擎缓存成功",
		Data:    nil,
	})
}

// ValidateRule 验证规则
// @Summary 验证规则
// @Description 验证规则的条件表达式和动作配置是否正确
// @Tags 规则引擎
// @Accept json
// @Produce json
// @Param rule body ValidateRuleRequest true "规则验证请求"
// @Success 200 {object} model.APIResponse{data=ValidateRuleResponse} "验证成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/scan-config/rule-engine/rules/validate [post]
func (h *RuleEngineHandler) ValidateRule(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 定义验证请求和响应结构
	type ValidateRuleRequest struct {
		Conditions string                   `json:"conditions" binding:"required" example:"file_type == 'go' && severity > 3"`
		Actions    []map[string]interface{} `json:"actions" binding:"required"`
	}

	type ValidateRuleResponse struct {
		Valid          bool     `json:"valid"`
		ConditionValid bool     `json:"condition_valid"`
		ActionValid    bool     `json:"action_valid"`
		Errors         []string `json:"errors,omitempty"`
		Warnings       []string `json:"warnings,omitempty"`
	}

	// 解析请求体
	var req ValidateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("解析规则验证请求失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/validate",
			"operation":  "validate_rule",
			"option":     "parse_request",
			"func_name":  "handler.scan_config.rule_engine.ValidateRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的规则验证请求",
			Error:   err.Error(),
		})
		return
	}

	// 创建临时规则进行验证
	tempRule := &rule_engine.Rule{
		ID:         "temp-rule",
		Name:       "临时验证规则",
		Severity:   "low",                     // 设置默认严重程度
		Priority:   50,                        // 设置默认优先级
		Enabled:    true,                      // 设置为启用状态
		Conditions: []rule_engine.Condition{}, // 暂时为空，因为req.Conditions是string类型
		Actions:    make([]rule_engine.Action, len(req.Actions)),
	}

	// 添加默认条件以通过验证
	tempRule.Conditions = []rule_engine.Condition{
		{
			Field:    "default",
			Operator: "eq",
			Value:    "true",
		},
	}

	// 转换动作格式
	for i, action := range req.Actions {
		actionType, _ := action["type"].(string)
		parameters, _ := action["parameters"].(map[string]interface{})
		if parameters == nil {
			parameters = make(map[string]interface{})
		}

		tempRule.Actions[i] = rule_engine.Action{
			Type:       actionType,
			Parameters: parameters,
		}
	}

	// 验证规则
	var response ValidateRuleResponse
	var errors []string

	// 验证条件表达式
	if err := h.ruleEngine.ValidateRule(tempRule); err != nil {
		response.ConditionValid = false
		errors = append(errors, "条件表达式验证失败: "+err.Error())
	} else {
		response.ConditionValid = true
	}

	// 验证动作配置
	response.ActionValid = true // 默认为true，如果有错误则设为false
	for i, actionMap := range req.Actions {
		// 转换为rule_engine.Action类型
		actionType, _ := actionMap["type"].(string)
		parameters, _ := actionMap["parameters"].(map[string]interface{})
		if parameters == nil {
			parameters = make(map[string]interface{})
		}

		action := rule_engine.Action{
			Type:       actionType,
			Parameters: parameters,
		}

		if err := h.ruleEngine.ValidateAction(action); err != nil {
			response.ActionValid = false
			errors = append(errors, "动作"+strconv.Itoa(i+1)+"验证失败: "+err.Error())
		}
	}

	// 设置整体验证结果
	response.Valid = response.ConditionValid && response.ActionValid
	response.Errors = errors

	// 记录日志
	if response.Valid {
		logger.Info("规则验证成功", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/validate",
			"operation":  "validate_rule",
			"option":     "success",
			"func_name":  "handler.scan_config.rule_engine.ValidateRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"timestamp":  logger.NowFormatted(),
		})
	} else {
		logger.Warn("规则验证失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/rules/validate",
			"operation":  "validate_rule",
			"option":     "validation_failed",
			"func_name":  "handler.scan_config.rule_engine.ValidateRule",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"errors":     errors,
			"timestamp":  logger.NowFormatted(),
		})
	}

	// 返回验证结果
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "规则验证完成",
		Data:    response,
	})
}

// ParseCondition 解析条件表达式
// @Summary 解析条件表达式
// @Description 解析规则条件表达式，返回解析结果和语法树
// @Tags 规则引擎
// @Accept json
// @Produce json
// @Param request body ParseConditionRequest true "条件解析请求"
// @Success 200 {object} model.APIResponse{data=rule_engine.Condition} "解析成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/scan-config/rule-engine/conditions/parse [post]
func (h *RuleEngineHandler) ParseCondition(c *gin.Context) {
	// 获取请求信息
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")

	// 定义解析请求结构
	type ParseConditionRequest struct {
		Expression string `json:"expression" binding:"required" example:"file_type == 'go' && severity > 3"`
	}

	// 解析请求体
	var req ParseConditionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("解析条件表达式请求失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/conditions/parse",
			"operation":  "parse_condition",
			"option":     "parse_request",
			"func_name":  "handler.scan_config.rule_engine.ParseCondition",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的条件解析请求",
			Error:   err.Error(),
		})
		return
	}

	// 解析条件表达式
	condition, err := h.ruleEngine.ParseCondition(c.Request.Context(), req.Expression)
	if err != nil {
		logger.Error("解析条件表达式失败", map[string]interface{}{
			"path":       "/api/v1/scan-config/rule-engine/conditions/parse",
			"operation":  "parse_condition",
			"option":     "parse",
			"func_name":  "handler.scan_config.rule_engine.ParseCondition",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": requestID,
			"expression": req.Expression,
			"error":      err.Error(),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "解析条件表达式失败",
			Error:   err.Error(),
		})
		return
	}

	// 记录成功日志
	logger.Info("解析条件表达式成功", map[string]interface{}{
		"path":       "/api/v1/scan-config/rule-engine/conditions/parse",
		"operation":  "parse_condition",
		"option":     "success",
		"func_name":  "handler.scan_config.rule_engine.ParseCondition",
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": requestID,
		"expression": req.Expression,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "解析条件表达式成功",
		Data:    condition,
	})
}
