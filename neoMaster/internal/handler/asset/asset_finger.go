package asset

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	assetservice "neomaster/internal/service/asset"
)

// AssetFingerHandler 资产指纹规则管理
// 负责 asset_finger 表(HTTP 指纹规则)的增删改查接口
type AssetFingerHandler struct {
	service *assetservice.AssetFingerService
}

// NewAssetFingerHandler 创建 AssetFingerHandler 实例
func NewAssetFingerHandler(service *assetservice.AssetFingerService) *AssetFingerHandler {
	return &AssetFingerHandler{service: service}
}

// CreateFingerRule 创建指纹规则
func (h *AssetFingerHandler) CreateFingerRule(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var rule assetmodel.AssetFinger
	if err := c.ShouldBindJSON(&rule); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_finger_rule",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.asset.asset_finger.CreateFingerRule",
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

	if err := h.service.CreateFingerRule(c.Request.Context(), &rule); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_finger_rule",
			"name":      rule.Name,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to create fingerprint rule",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_finger_rule", 0, "", clientIP, XRequestID, "success", "Fingerprint rule created successfully", map[string]interface{}{
		"func_name":  "handler.asset.asset_finger.CreateFingerRule",
		"path":       pathUrl,
		"method":     "POST",
		"name":       rule.Name,
		"id":         rule.ID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Fingerprint rule created successfully",
		Data:    rule,
	})
}

// GetFingerRule 获取指纹规则详情
func (h *AssetFingerHandler) GetFingerRule(c *gin.Context) {
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
			Error:   err.Error(),
		})
		return
	}

	rule, err := h.service.GetFingerRule(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_finger_rule",
			"id":        id,
		})
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Fingerprint rule not found",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Fingerprint rule retrieved successfully",
		Data:    rule,
	})
}

// UpdateFingerRule 更新指纹规则
func (h *AssetFingerHandler) UpdateFingerRule(c *gin.Context) {
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
			Error:   err.Error(),
		})
		return
	}

	var rule assetmodel.AssetFinger
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}
	rule.ID = id

	if err := h.service.UpdateFingerRule(c.Request.Context(), &rule); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_finger_rule",
			"id":        id,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to update fingerprint rule",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_finger_rule", 0, "", clientIP, XRequestID, "success", "Fingerprint rule updated successfully", map[string]interface{}{
		"id":        id,
		"func_name": "handler.asset.asset_finger.UpdateFingerRule",
		"path":      pathUrl,
		"method":    "PUT",
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Fingerprint rule updated successfully",
	})
}

// DeleteFingerRule 删除指纹规则
func (h *AssetFingerHandler) DeleteFingerRule(c *gin.Context) {
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
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.DeleteFingerRule(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_finger_rule",
			"id":        id,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to delete fingerprint rule",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_finger_rule", 0, "", clientIP, XRequestID, "success", "Fingerprint rule deleted successfully", map[string]interface{}{
		"id":        id,
		"func_name": "handler.asset.asset_finger.DeleteFingerRule",
		"path":      pathUrl,
		"method":    "DELETE",
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Fingerprint rule deleted successfully",
	})
}

// ListFingerRules 获取指纹规则列表
func (h *AssetFingerHandler) ListFingerRules(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	name := c.Query("name")

	list, total, _, err := h.service.ListFingerRules(c.Request.Context(), page, pageSize, name)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_finger_rules",
			"page":      page,
			"page_size": pageSize,
			"name":      name,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list fingerprint rules",
			Error:   err.Error(),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	pagination := system.PaginationResponse{
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
		Data:        list,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Fingerprint rules retrieved successfully",
		Data:    pagination,
	})
}
