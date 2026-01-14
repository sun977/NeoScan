package asset

import (
	"math"
	"net/http"
	"strconv"
	"strings"

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

	// 处理标签过滤
	var tagIDs []uint64
	tagIDsStr := c.Query("tag_ids")
	if tagIDsStr != "" {
		ids := strings.Split(tagIDsStr, ",")
		for _, idStr := range ids {
			id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64)
			if err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	rawAssets, total, err := h.service.ListRawAssets(c.Request.Context(), page, pageSize, batchID, status, tagIDs)
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

// AddRawAssetTagRequest 添加标签请求
type AddRawAssetTagRequest struct {
	TagID uint64 `json:"tag_id" binding:"required"`
}

// AddRawAssetTag 添加原始资产标签
func (h *RawAssetHandler) AddRawAssetTag(c *gin.Context) {
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

	var req AddRawAssetTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.AddTagToRawAsset(c.Request.Context(), id, req.TagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "add_raw_asset_tag",
			"id":        id,
			"tag_id":    req.TagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to add tag to raw asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("add_raw_asset_tag", 0, "", clientIP, XRequestID, "success", "Tag added to raw asset successfully", map[string]interface{}{
		"id":     id,
		"tag_id": req.TagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag added to raw asset successfully",
	})
}

// RemoveRawAssetTag 删除原始资产标签
func (h *RawAssetHandler) RemoveRawAssetTag(c *gin.Context) {
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

	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Tag ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.RemoveTagFromRawAsset(c.Request.Context(), id, tagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "remove_raw_asset_tag",
			"id":        id,
			"tag_id":    tagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to remove tag from raw asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("remove_raw_asset_tag", 0, "", clientIP, XRequestID, "success", "Tag removed from raw asset successfully", map[string]interface{}{
		"id":     id,
		"tag_id": tagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag removed from raw asset successfully",
	})
}

// GetRawAssetTags 获取原始资产标签
func (h *RawAssetHandler) GetRawAssetTags(c *gin.Context) {
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

	tags, err := h.service.GetRawAssetTags(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_raw_asset_tags",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get raw asset tags",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw asset tags retrieved successfully",
		Data:    tags,
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

	// 处理标签过滤
	var tagIDs []uint64
	tagIDsStr := c.Query("tag_ids")
	if tagIDsStr != "" {
		ids := strings.Split(tagIDsStr, ",")
		for _, idStr := range ids {
			id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64)
			if err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	networks, total, err := h.service.ListRawNetworks(c.Request.Context(), page, pageSize, status, sourceType, tagIDs)
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

// AddRawNetworkTagRequest 添加标签请求
type AddRawNetworkTagRequest struct {
	TagID uint64 `json:"tag_id" binding:"required"`
}

// AddRawNetworkTag 添加待确认网段标签
func (h *RawAssetHandler) AddRawNetworkTag(c *gin.Context) {
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

	var req AddRawNetworkTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.AddTagToRawNetwork(c.Request.Context(), id, req.TagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "add_raw_network_tag",
			"id":        id,
			"tag_id":    req.TagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to add tag to raw network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("add_raw_network_tag", 0, "", clientIP, XRequestID, "success", "Tag added to raw network successfully", map[string]interface{}{
		"id":     id,
		"tag_id": req.TagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag added to raw network successfully",
	})
}

// RemoveRawNetworkTag 删除待确认网段标签
func (h *RawAssetHandler) RemoveRawNetworkTag(c *gin.Context) {
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

	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Tag ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.RemoveTagFromRawNetwork(c.Request.Context(), id, tagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "remove_raw_network_tag",
			"id":        id,
			"tag_id":    tagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to remove tag from raw network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("remove_raw_network_tag", 0, "", clientIP, XRequestID, "success", "Tag removed from raw network successfully", map[string]interface{}{
		"id":     id,
		"tag_id": tagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag removed from raw network successfully",
	})
}

// GetRawNetworkTags 获取待确认网段标签
func (h *RawAssetHandler) GetRawNetworkTags(c *gin.Context) {
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

	tags, err := h.service.GetRawNetworkTags(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_raw_network_tags",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get raw network tags",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Raw network tags retrieved successfully",
		Data:    tags,
	})
}
