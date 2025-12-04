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

// RawAssetHandler 原始资产处理器
type RawAssetHandler struct {
	service *assetservice.RawAssetService
}

// NewRawAssetHandler 创建 RawAssetHandler 实例
func NewRawAssetHandler(service *assetservice.RawAssetService) *RawAssetHandler {
	return &RawAssetHandler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// RawAsset Handlers
// -----------------------------------------------------------------------------

// CreateRawAsset 创建原始资产
func (h *RawAssetHandler) CreateRawAsset(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var raw assetmodel.RawAsset
	if err := c.ShouldBindJSON(&raw); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_raw_asset",
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

	if err := h.service.CreateRawAsset(c.Request.Context(), &raw); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_raw_asset",
			"source":    raw.SourceType,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create raw asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_raw_asset", 0, "", clientIP, XRequestID, "success", "Raw asset created successfully", map[string]interface{}{
		"id":         raw.ID,
		"source":     raw.SourceType,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Raw asset created successfully",
		Data:    raw,
	})
}

// GetRawAsset 获取原始资产详情
func (h *RawAssetHandler) GetRawAsset(c *gin.Context) {
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

	raw, err := h.service.GetRawAssetByID(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_raw_asset",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get raw asset",
			Error:   err.Error(),
		})
		return
	}

	if raw == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Raw asset not found",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw asset retrieved successfully",
		Data:    raw,
	})
}

// UpdateRawAssetStatusRequest 更新原始资产状态请求
type UpdateRawAssetStatusRequest struct {
	Status string `json:"status" binding:"required"`
	ErrMsg string `json:"err_msg"`
}

// UpdateRawAssetStatus 更新原始资产状态
func (h *RawAssetHandler) UpdateRawAssetStatus(c *gin.Context) {
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

	var req UpdateRawAssetStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.UpdateRawAssetStatus(c.Request.Context(), id, req.Status, req.ErrMsg); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PATCH", map[string]interface{}{
			"operation": "update_raw_asset_status",
			"id":        id,
			"status":    req.Status,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update raw asset status",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_raw_asset_status", 0, "", clientIP, XRequestID, "success", "Raw asset status updated successfully", map[string]interface{}{
		"id":     id,
		"status": req.Status,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw asset status updated successfully",
	})
}

// ListRawAssets 获取原始资产列表
func (h *RawAssetHandler) ListRawAssets(c *gin.Context) {
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

	batchID := c.Query("batch_id")
	status := c.Query("status")

	rawAssets, total, err := h.service.ListRawAssets(c.Request.Context(), page, pageSize, batchID, status)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_raw_assets",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list raw assets",
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
		Data:        rawAssets,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw assets retrieved successfully",
		Data:    pagination,
	})
}

// -----------------------------------------------------------------------------
// RawAssetNetwork Handlers
// -----------------------------------------------------------------------------

// CreateRawNetwork 创建待确认网段
func (h *RawAssetHandler) CreateRawNetwork(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var network assetmodel.RawAssetNetwork
	if err := c.ShouldBindJSON(&network); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_raw_network",
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

	if err := h.service.CreateRawNetwork(c.Request.Context(), &network); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_raw_network",
			"network":   network.Network,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create raw network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_raw_network", 0, "", clientIP, XRequestID, "success", "Raw network created successfully", map[string]interface{}{
		"id":         network.ID,
		"network":    network.Network,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Raw network created successfully",
		Data:    network,
	})
}

// GetRawNetwork 获取待确认网段详情
func (h *RawAssetHandler) GetRawNetwork(c *gin.Context) {
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

	network, err := h.service.GetRawNetworkByID(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_raw_network",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get raw network",
			Error:   err.Error(),
		})
		return
	}

	if network == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Raw network not found",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw network retrieved successfully",
		Data:    network,
	})
}

// ApproveRawNetwork 批准待确认网段
func (h *RawAssetHandler) ApproveRawNetwork(c *gin.Context) {
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

	if err := h.service.ApproveRawNetwork(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "approve_raw_network",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to approve raw network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("approve_raw_network", 0, "", clientIP, XRequestID, "success", "Raw network approved successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw network approved successfully",
	})
}

// RejectRawNetwork 拒绝待确认网段
func (h *RawAssetHandler) RejectRawNetwork(c *gin.Context) {
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

	if err := h.service.RejectRawNetwork(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "reject_raw_network",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to reject raw network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("reject_raw_network", 0, "", clientIP, XRequestID, "success", "Raw network rejected successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw network rejected successfully",
	})
}

// ListRawNetworks 获取待确认网段列表
func (h *RawAssetHandler) ListRawNetworks(c *gin.Context) {
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

	status := c.Query("status")
	sourceType := c.Query("source_type")

	networks, total, err := h.service.ListRawNetworks(c.Request.Context(), page, pageSize, status, sourceType)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_raw_networks",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list raw networks",
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
		Data:        networks,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw networks retrieved successfully",
		Data:    pagination,
	})
}
