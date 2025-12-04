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

// AssetPolicyHandler 资产策略处理器
type AssetPolicyHandler struct {
	service *assetservice.AssetPolicyService
}

// NewAssetPolicyHandler 创建 AssetPolicyHandler 实例
func NewAssetPolicyHandler(service *assetservice.AssetPolicyService) *AssetPolicyHandler {
	return &AssetPolicyHandler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// AssetWhitelist Handlers
// -----------------------------------------------------------------------------

// CreateWhitelist 创建白名单
func (h *AssetPolicyHandler) CreateWhitelist(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var whitelist assetmodel.AssetWhitelist
	if err := c.ShouldBindJSON(&whitelist); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_whitelist",
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

	if err := h.service.CreateWhitelist(c.Request.Context(), &whitelist); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_whitelist",
			"name":      whitelist.WhitelistName,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create whitelist",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_whitelist", 0, "", clientIP, XRequestID, "success", "Whitelist created successfully", map[string]interface{}{
		"name":       whitelist.WhitelistName,
		"id":         whitelist.ID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Whitelist created successfully",
		Data:    whitelist,
	})
}

// GetWhitelist 获取白名单详情
func (h *AssetPolicyHandler) GetWhitelist(c *gin.Context) {
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

	whitelist, err := h.service.GetWhitelist(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_whitelist",
			"id":        id,
		})
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Whitelist not found",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Whitelist retrieved successfully",
		Data:    whitelist,
	})
}

// UpdateWhitelist 更新白名单
func (h *AssetPolicyHandler) UpdateWhitelist(c *gin.Context) {
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

	var whitelist assetmodel.AssetWhitelist
	if err := c.ShouldBindJSON(&whitelist); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}
	whitelist.ID = id

	if err := h.service.UpdateWhitelist(c.Request.Context(), &whitelist); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_whitelist",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update whitelist",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_whitelist", 0, "", clientIP, XRequestID, "success", "Whitelist updated successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Whitelist updated successfully",
	})
}

// DeleteWhitelist 删除白名单
func (h *AssetPolicyHandler) DeleteWhitelist(c *gin.Context) {
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

	if err := h.service.DeleteWhitelist(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_whitelist",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to delete whitelist",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_whitelist", 0, "", clientIP, XRequestID, "success", "Whitelist deleted successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Whitelist deleted successfully",
	})
}

// ListWhitelists 获取白名单列表
func (h *AssetPolicyHandler) ListWhitelists(c *gin.Context) {
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
	targetType := c.Query("type")

	whitelists, total, err := h.service.ListWhitelists(c.Request.Context(), page, pageSize, name, targetType)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_whitelists",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list whitelists",
			Error:   err.Error(),
		})
		return
	}

	// 计算分页信息
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := system.PaginationResponse{
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
		Data:        whitelists,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Whitelists retrieved successfully",
		Data:    pagination,
	})
}

// -----------------------------------------------------------------------------
// AssetSkipPolicy Handlers
// -----------------------------------------------------------------------------

// CreateSkipPolicy 创建跳过策略
func (h *AssetPolicyHandler) CreateSkipPolicy(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var policy assetmodel.AssetSkipPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_skip_policy",
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

	if err := h.service.CreateSkipPolicy(c.Request.Context(), &policy); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_skip_policy",
			"name":      policy.PolicyName,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create skip policy",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_skip_policy", 0, "", clientIP, XRequestID, "success", "Skip policy created successfully", map[string]interface{}{
		"name":       policy.PolicyName,
		"id":         policy.ID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Skip policy created successfully",
		Data:    policy,
	})
}

// GetSkipPolicy 获取跳过策略详情
func (h *AssetPolicyHandler) GetSkipPolicy(c *gin.Context) {
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

	policy, err := h.service.GetSkipPolicy(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_skip_policy",
			"id":        id,
		})
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Skip policy not found",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Skip policy retrieved successfully",
		Data:    policy,
	})
}

// UpdateSkipPolicy 更新跳过策略
func (h *AssetPolicyHandler) UpdateSkipPolicy(c *gin.Context) {
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

	var policy assetmodel.AssetSkipPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}
	policy.ID = id

	if err := h.service.UpdateSkipPolicy(c.Request.Context(), &policy); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_skip_policy",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update skip policy",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_skip_policy", 0, "", clientIP, XRequestID, "success", "Skip policy updated successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Skip policy updated successfully",
	})
}

// DeleteSkipPolicy 删除跳过策略
func (h *AssetPolicyHandler) DeleteSkipPolicy(c *gin.Context) {
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

	if err := h.service.DeleteSkipPolicy(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_skip_policy",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to delete skip policy",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_skip_policy", 0, "", clientIP, XRequestID, "success", "Skip policy deleted successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Skip policy deleted successfully",
	})
}

// ListSkipPolicies 获取跳过策略列表
func (h *AssetPolicyHandler) ListSkipPolicies(c *gin.Context) {
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
	policyType := c.Query("type")

	policies, total, err := h.service.ListSkipPolicies(c.Request.Context(), page, pageSize, name, policyType)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_skip_policies",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list skip policies",
			Error:   err.Error(),
		})
		return
	}

	// 计算分页信息
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	pagination := system.PaginationResponse{
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
		Data:        policies,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Skip policies retrieved successfully",
		Data:    pagination,
	})
}
