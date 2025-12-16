package tag_system

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model/system"
	"neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/matcher"
	"neomaster/internal/pkg/utils"
	service "neomaster/internal/service/tag_system"
)

type TagHandler struct {
	service service.TagService
}

func NewTagHandler(service service.TagService) *TagHandler {
	return &TagHandler{service: service}
}

// --- Tag CRUD ---

// CreateTag 创建标签
func (h *TagHandler) CreateTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var req tag_system.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_tag",
			"error":      "invalid_json",
			"user_agent": userAgent,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	tag := &tag_system.SysTag{
		Name:        req.Name,
		ParentID:    req.ParentID,
		Color:       req.Color,
		Category:    req.Category,
		Description: req.Description,
	}

	if err := h.service.CreateTag(c.Request.Context(), tag); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_tag",
			"name":      req.Name,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_tag", 0, "", clientIP, XRequestID, "success", "Tag created successfully", map[string]interface{}{
		"id":   tag.ID,
		"name": tag.Name,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag created successfully",
		Data:    gin.H{"id": tag.ID},
	})
}

// GetTag 获取标签详情
func (h *TagHandler) GetTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid ID",
			Error:   "invalid id format",
		})
		return
	}

	tag, err := h.service.GetTag(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_tag",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	resp := tag_system.TagResponse{
		ID:          tag.ID,
		Name:        tag.Name,
		ParentID:    tag.ParentID,
		Path:        tag.Path,
		Level:       tag.Level,
		Color:       tag.Color,
		Category:    tag.Category,
		Description: tag.Description,
		CreatedAt:   tag.CreatedAt,
		UpdatedAt:   tag.UpdatedAt,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag retrieved successfully",
		Data:    resp,
	})
}

// UpdateTag 更新标签
func (h *TagHandler) UpdateTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid ID",
			Error:   "invalid id format",
		})
		return
	}

	var req tag_system.UpdateTagRequest
	if err1 := c.ShouldBindJSON(&req); err1 != nil {
		logger.LogBusinessError(err1, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_tag",
			"error":      "invalid_json",
			"user_agent": userAgent,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err1.Error(),
		})
		return
	}

	// 先获取旧标签
	tag, err := h.service.GetTag(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_tag",
			"step":      "get_old_tag",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	// 更新字段
	if req.Name != "" {
		tag.Name = req.Name
	}
	if req.Color != "" {
		tag.Color = req.Color
	}
	if req.Description != "" {
		tag.Description = req.Description
	}

	if err := h.service.UpdateTag(c.Request.Context(), tag); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_tag",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_tag", 0, "", clientIP, XRequestID, "success", "Tag updated successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag updated successfully",
	})
}

// DeleteTag 删除标签
func (h *TagHandler) DeleteTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid ID",
			Error:   "invalid id format",
		})
		return
	}

	if err := h.service.DeleteTag(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_tag",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_tag", 0, "", clientIP, XRequestID, "success", "Tag deleted successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag deleted successfully",
	})
}

// ListTags 获取标签列表 (支持分页和关键字筛选)
func (h *TagHandler) ListTags(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var req tag_system.ListTagsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid query parameters",
			Error:   err.Error(),
		})
		return
	}

	tags, total, err := h.service.ListTags(c.Request.Context(), &req)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_tags",
			"params":    req,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	var respList []tag_system.TagResponse
	for _, tag := range tags {
		respList = append(respList, tag_system.TagResponse{
			ID:          tag.ID,
			Name:        tag.Name,
			ParentID:    tag.ParentID,
			Path:        tag.Path,
			Level:       tag.Level,
			Color:       tag.Color,
			Category:    tag.Category,
			Description: tag.Description,
			CreatedAt:   tag.CreatedAt,
			UpdatedAt:   tag.UpdatedAt,
		})
	}
	// 确保返回空数组而不是 null
	if respList == nil {
		respList = []tag_system.TagResponse{}
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tags retrieved successfully",
		Data: gin.H{
			"list":      respList,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// --- Rule CRUD ---

// CreateRule 创建规则
func (h *TagHandler) CreateRule(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var req tag_system.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_rule",
			"error":      "invalid_json",
			"user_agent": userAgent,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 序列化 RuleJSON
	ruleJSONBytes, err := json.Marshal(req.RuleJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid rule json format",
			Error:   err.Error(),
		})
		return
	}

	rule := &tag_system.SysMatchRule{
		TagID:      req.TagID,
		Name:       req.Name,
		EntityType: req.EntityType,
		Priority:   req.Priority,
		RuleJSON:   string(ruleJSONBytes),
		IsEnabled:  req.IsEnabled,
	}

	if err := h.service.CreateRule(c.Request.Context(), rule); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_rule",
			"name":      req.Name,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_rule", 0, "", clientIP, XRequestID, "success", "Rule created successfully", map[string]interface{}{
		"id":   rule.ID,
		"name": rule.Name,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Rule created successfully",
		Data:    gin.H{"id": rule.ID},
	})
}

// ListRules 获取规则列表 (支持分页和筛选)
func (h *TagHandler) ListRules(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var req tag_system.ListRulesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid query parameters",
			Error:   err.Error(),
		})
		return
	}

	rules, total, err := h.service.ListRules(c.Request.Context(), &req)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_rules",
			"params":    req,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	var respList []tag_system.RuleResponse
	for _, r := range rules {
		var mr matcher.MatchRule
		// 尝试解析 JSON，如果失败则返回空规则或跳过
		if err := json.Unmarshal([]byte(r.RuleJSON), &mr); err != nil {
			// 记录日志或忽略，这里简单处理
		}

		respList = append(respList, tag_system.RuleResponse{
			ID:         r.ID,
			TagID:      r.TagID,
			Name:       r.Name,
			EntityType: r.EntityType,
			Priority:   r.Priority,
			RuleJSON:   mr,
			IsEnabled:  r.IsEnabled,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
		})
	}
	if respList == nil {
		respList = []tag_system.RuleResponse{}
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Rules retrieved successfully",
		Data: gin.H{
			"list":      respList,
			"total":     total,
			"page":      req.Page,
			"page_size": req.PageSize,
		},
	})
}

// UpdateRule 更新规则
func (h *TagHandler) UpdateRule(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid ID",
			Error:   "invalid id format",
		})
		return
	}

	var req tag_system.UpdateRuleRequest
	if err1 := c.ShouldBindJSON(&req); err1 != nil {
		logger.LogBusinessError(err1, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_rule",
			"error":      "invalid_json",
			"user_agent": userAgent,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err1.Error(),
		})
		return
	}

	// 1. 获取旧规则
	rule, err := h.service.GetRule(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_rule",
			"step":      "get_old_rule",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	// 2. 更新字段
	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Priority != 0 {
		rule.Priority = req.Priority
	}
	if req.RuleJSON != nil {
		ruleJSONBytes, err := json.Marshal(req.RuleJSON)
		if err != nil {
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "failed",
				Message: "Invalid rule json format",
				Error:   err.Error(),
			})
			return
		}
		rule.RuleJSON = string(ruleJSONBytes)
	}
	// IsEnabled 是 bool，不能简单判断零值。
	// 但 Request struct 指针字段可以判断 nil。
	// 这里假设 Request 结构体字段不是指针，所以 bool 无法区分 "false" 和 "unset"。
	// 暂时只支持显式更新 Name/Priority/RuleJSON。
	// 如果需要支持 bool 更新，建议 Request 使用 *bool。
	// 检查 request.go 定义。
	if req.IsEnabled != nil {
		rule.IsEnabled = *req.IsEnabled
	}

	// 3. 保存
	if err := h.service.UpdateRule(c.Request.Context(), rule); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_rule",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_rule", 0, "", clientIP, XRequestID, "success", "Rule updated successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Rule updated successfully",
	})
}

// DeleteRule 删除规则
func (h *TagHandler) DeleteRule(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid ID",
			Error:   "invalid id format",
		})
		return
	}

	if err := h.service.DeleteRule(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_rule",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_rule", 0, "", clientIP, XRequestID, "success", "Rule deleted successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Rule deleted successfully",
	})
}

// ApplyRule 手动触发规则执行 (添加/移除标签)
func (h *TagHandler) ApplyRule(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid ID",
			Error:   "invalid id format",
		})
		return
	}

	// 获取 action 参数，默认为 "add"
	action := c.DefaultQuery("action", "add")
	if action != "add" && action != "remove" {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid action. Must be 'add' or 'remove'",
		})
		return
	}

	taskID, err := h.service.SubmitPropagationTask(c.Request.Context(), id, action)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "apply_rule",
			"rule_id":   id,
			"action":    action,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to submit propagation task",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("apply_rule", 0, "", clientIP, XRequestID, "success", "Rule application task submitted", map[string]interface{}{
		"rule_id": id,
		"action":  action,
		"task_id": taskID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Rule application task submitted successfully",
		Data: gin.H{
			"task_id": taskID,
			"action":  action,
		},
	})
}
