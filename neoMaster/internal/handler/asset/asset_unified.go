package asset

import (
	"net/http"
	"strconv"
	"strings"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	assetService "neomaster/internal/service/asset"

	assetmodel "neomaster/internal/model/asset"
	assetRepo "neomaster/internal/repo/mysql/asset"

	"github.com/gin-gonic/gin"
)

// AssetUnifiedHandler 统一资产处理器
// 负责处理统一资产视图的 HTTP 请求
type AssetUnifiedHandler struct {
	service *assetService.AssetUnifiedService
}

// NewAssetUnifiedHandler 创建 AssetUnifiedHandler 实例
func NewAssetUnifiedHandler(service *assetService.AssetUnifiedService) *AssetUnifiedHandler {
	return &AssetUnifiedHandler{service: service}
}

// CreateUnifiedAsset 创建统一资产
func (h *AssetUnifiedHandler) CreateUnifiedAsset(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	userAgent := c.GetHeader("User-Agent")

	var asset assetmodel.AssetUnified
	if err := c.ShouldBindJSON(&asset); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_unified_asset",
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

	if err := h.service.CreateUnifiedAsset(c.Request.Context(), &asset); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_unified_asset",
			"ip":         asset.IP,
			"user_agent": userAgent,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create unified asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_unified_asset", 0, "", clientIP, XRequestID, "success", "Unified asset created successfully", map[string]interface{}{
		"path":       pathUrl,
		"method":     "POST",
		"asset_id":   asset.ID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Unified asset created successfully",
		Data:    asset,
	})
}

// GetUnifiedAsset 获取统一资产详情
func (h *AssetUnifiedHandler) GetUnifiedAsset(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Asset ID",
			Error:   err.Error(),
		})
		return
	}

	asset, err := h.service.GetUnifiedAssetByID(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_unified_asset",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to retrieve unified asset",
			Error:   err.Error(),
		})
		return
	}

	if asset == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Unified asset not found",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Unified asset retrieved successfully",
		Data:    asset,
	})
}

// UpdateUnifiedAsset 更新统一资产
func (h *AssetUnifiedHandler) UpdateUnifiedAsset(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	userAgent := c.GetHeader("User-Agent")

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Asset ID",
			Error:   err.Error(),
		})
		return
	}

	var asset assetmodel.AssetUnified
	if err := c.ShouldBindJSON(&asset); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_unified_asset",
			"id":         id,
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

	asset.ID = id
	if err := h.service.UpdateUnifiedAsset(c.Request.Context(), &asset); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_unified_asset",
			"id":         id,
			"user_agent": userAgent,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update unified asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_unified_asset", 0, "", clientIP, XRequestID, "success", "Unified asset updated successfully", map[string]interface{}{
		"path":       pathUrl,
		"method":     "PUT",
		"id":         id,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Unified asset updated successfully",
		Data:    asset,
	})
}

// DeleteUnifiedAsset 删除统一资产
func (h *AssetUnifiedHandler) DeleteUnifiedAsset(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	userAgent := c.GetHeader("User-Agent")

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Asset ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.DeleteUnifiedAsset(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation":  "delete_unified_asset",
			"id":         id,
			"user_agent": userAgent,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to delete unified asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_unified_asset", 0, "", clientIP, XRequestID, "success", "Unified asset deleted successfully", map[string]interface{}{
		"path":       pathUrl,
		"method":     "DELETE",
		"id":         id,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Unified asset deleted successfully",
	})
}

// ListUnifiedAssets 获取统一资产列表
func (h *AssetUnifiedHandler) ListUnifiedAssets(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	var filter assetRepo.UnifiedAssetFilter
	if val := c.Query("project_id"); val != "" {
		if pid, err1 := strconv.ParseUint(val, 10, 64); err1 == nil {
			filter.ProjectID = pid
		}
	}
	filter.IP = c.Query("ip")
	if val := c.Query("port"); val != "" {
		if p, err2 := strconv.Atoi(val); err2 == nil {
			filter.Port = p
		}
	}
	filter.Service = c.Query("service")
	filter.Product = c.Query("product")
	filter.Component = c.Query("component")
	if val := c.Query("is_web"); val != "" {
		isWeb := val == "true"
		filter.IsWeb = &isWeb
	}
	filter.Keyword = c.Query("keyword")

	tagIDsStr := c.Query("tag_ids")
	var tagIDs []uint64
	if tagIDsStr != "" {
		parts := strings.Split(tagIDsStr, ",")
		for _, part := range parts {
			id, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				continue
			}
			tagIDs = append(tagIDs, id)
		}
	}

	assets, total, err := h.service.ListUnifiedAssets(c.Request.Context(), page, pageSize, filter, tagIDs)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_unified_assets",
			"page":      page,
			"page_size": pageSize,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list unified assets",
			Error:   err.Error(),
		})
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Unified assets retrieved successfully",
		Data: system.PaginationResponse{
			Data:        assets,
			Total:       total,
			Page:        page,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			HasNext:     page < totalPages,
			HasPrevious: page > 1,
		},
	})
}

// UpsertUnifiedAsset 插入或更新统一资产
func (h *AssetUnifiedHandler) UpsertUnifiedAsset(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	userAgent := c.GetHeader("User-Agent")

	var asset assetmodel.AssetUnified
	if err := c.ShouldBindJSON(&asset); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "upsert_unified_asset",
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

	if err := h.service.UpsertUnifiedAsset(c.Request.Context(), &asset); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "upsert_unified_asset",
			"ip":         asset.IP,
			"user_agent": userAgent,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to upsert unified asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("upsert_unified_asset", 0, "", clientIP, XRequestID, "success", "Unified asset upserted successfully", map[string]interface{}{
		"path":       pathUrl,
		"method":     "POST",
		"asset_id":   asset.ID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Unified asset upserted successfully",
		Data:    asset,
	})
}

// -----------------------------------------------------------------------------
// Tag Management Handlers
// -----------------------------------------------------------------------------

// GetUnifiedAssetTags 获取统一资产标签
func (h *AssetUnifiedHandler) GetUnifiedAssetTags(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Asset ID",
			Error:   err.Error(),
		})
		return
	}

	tags, err := h.service.GetUnifiedAssetTags(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_unified_asset_tags",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get unified asset tags",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Unified asset tags retrieved successfully",
		Data:    tags,
	})
}

// AddUnifiedAssetTag 为统一资产添加标签
func (h *AssetUnifiedHandler) AddUnifiedAssetTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Asset ID",
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

	if err := h.service.AddUnifiedAssetTag(c.Request.Context(), id, req.TagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "add_unified_asset_tag",
			"id":        id,
			"tag_id":    req.TagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to add tag to unified asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("add_unified_asset_tag", 0, "", clientIP, XRequestID, "success", "Tag added to unified asset successfully", map[string]interface{}{
		"id":     id,
		"tag_id": req.TagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag added to unified asset successfully",
	})
}

// RemoveUnifiedAssetTag 为统一资产移除标签
func (h *AssetUnifiedHandler) RemoveUnifiedAssetTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Asset ID",
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

	if err := h.service.RemoveUnifiedAssetTag(c.Request.Context(), id, tagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "remove_unified_asset_tag",
			"id":        id,
			"tag_id":    tagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to remove tag from unified asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("remove_unified_asset_tag", 0, "", clientIP, XRequestID, "success", "Tag removed from unified asset successfully", map[string]interface{}{
		"id":     id,
		"tag_id": tagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag removed from unified asset successfully",
	})
}
