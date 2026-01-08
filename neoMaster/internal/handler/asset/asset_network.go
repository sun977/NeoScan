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

// AssetNetworkHandler 资产网段处理器
type AssetNetworkHandler struct {
	service *assetservice.AssetNetworkService
}

// NewAssetNetworkHandler 创建 AssetNetworkHandler 实例
func NewAssetNetworkHandler(service *assetservice.AssetNetworkService) *AssetNetworkHandler {
	return &AssetNetworkHandler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// AssetNetwork Handlers
// -----------------------------------------------------------------------------

// CreateNetwork 创建网段
func (h *AssetNetworkHandler) CreateNetwork(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var network assetmodel.AssetNetwork
	if err := c.ShouldBindJSON(&network); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_network",
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

	if err := h.service.CreateNetwork(c.Request.Context(), &network); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_network",
			"cidr":      network.CIDR,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_network", 0, "", clientIP, XRequestID, "success", "Network created successfully", map[string]interface{}{
		"cidr":       network.CIDR,
		"id":         network.ID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Network created successfully",
		Data:    network,
	})
}

// GetNetwork 获取网段详情
func (h *AssetNetworkHandler) GetNetwork(c *gin.Context) {
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

	network, err := h.service.GetNetwork(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_network",
			"id":        id,
		})
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Network not found",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Network retrieved successfully",
		Data:    network,
	})
}

// UpdateNetwork 更新网段
func (h *AssetNetworkHandler) UpdateNetwork(c *gin.Context) {
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

	var network assetmodel.AssetNetwork
	if err := c.ShouldBindJSON(&network); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}
	network.ID = id

	if err := h.service.UpdateNetwork(c.Request.Context(), &network); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_network",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_network", 0, "", clientIP, XRequestID, "success", "Network updated successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Network updated successfully",
	})
}

// DeleteNetwork 删除网段
func (h *AssetNetworkHandler) DeleteNetwork(c *gin.Context) {
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

	if err := h.service.DeleteNetwork(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_network",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to delete network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_network", 0, "", clientIP, XRequestID, "success", "Network deleted successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Network deleted successfully",
	})
}

// ListNetworks 获取网段列表
func (h *AssetNetworkHandler) ListNetworks(c *gin.Context) {
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

	cidr := c.Query("cidr")
	networkType := c.Query("type")
	status := c.Query("status")
	tagIDsStr := c.Query("tag_ids")

	var tagIDs []uint64
	if tagIDsStr != "" {
		ids := strings.Split(tagIDsStr, ",")
		for _, id := range ids {
			if idInt, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64); err == nil {
				tagIDs = append(tagIDs, idInt)
			}
		}
	}

	networks, total, err := h.service.ListNetworks(c.Request.Context(), page, pageSize, cidr, networkType, status, tagIDs)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_networks",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list networks",
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
		Message: "Networks retrieved successfully",
		Data:    pagination,
	})
}

// UpdateScanStatus 更新网段扫描状态
func (h *AssetNetworkHandler) UpdateScanStatus(c *gin.Context) {
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

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.UpdateScanStatus(c.Request.Context(), id, req.Status); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PATCH", map[string]interface{}{
			"operation": "update_network_scan_status",
			"id":        id,
			"status":    req.Status,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update scan status",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_network_scan_status", 0, "", clientIP, XRequestID, "success", "Network scan status updated", map[string]interface{}{
		"id":     id,
		"status": req.Status,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Network scan status updated successfully",
	})
}

// -----------------------------------------------------------------------------
// Tag Management Handlers
// -----------------------------------------------------------------------------

// AddNetworkTag 为网段添加标签
func (h *AssetNetworkHandler) AddNetworkTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	networkID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Network ID",
			Error:   err.Error(),
		})
		return
	}

	var req struct {
		TagID uint64 `json:"tag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.AddTagToNetwork(c.Request.Context(), networkID, req.TagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "add_network_tag",
			"network_id": networkID,
			"tag_id":     req.TagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to add tag to network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("add_network_tag", 0, "", clientIP, XRequestID, "success", "Tag added to network successfully", map[string]interface{}{
		"network_id": networkID,
		"tag_id":     req.TagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag added to network successfully",
	})
}

// RemoveNetworkTag 从网段移除标签
func (h *AssetNetworkHandler) RemoveNetworkTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	networkIDStr := c.Param("id")
	networkID, err := strconv.ParseUint(networkIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Network ID",
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

	if err := h.service.RemoveTagFromNetwork(c.Request.Context(), networkID, tagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation":  "remove_network_tag",
			"network_id": networkID,
			"tag_id":     tagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to remove tag from network",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("remove_network_tag", 0, "", clientIP, XRequestID, "success", "Tag removed from network successfully", map[string]interface{}{
		"network_id": networkID,
		"tag_id":     tagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag removed from network successfully",
	})
}

// GetNetworkTags 获取网段标签
func (h *AssetNetworkHandler) GetNetworkTags(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	networkID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Network ID",
			Error:   err.Error(),
		})
		return
	}

	tags, err := h.service.GetNetworkTags(c.Request.Context(), networkID)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation":  "get_network_tags",
			"network_id": networkID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get network tags",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Network tags retrieved successfully",
		Data:    tags,
	})
}
