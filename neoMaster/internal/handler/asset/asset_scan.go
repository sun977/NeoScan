package asset

import (
	"net/http"
	"strconv"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	assetService "neomaster/internal/service/asset"

	assetmodel "neomaster/internal/model/asset"

	"github.com/gin-gonic/gin"
)

// AssetScanHandler 资产扫描记录处理器
// 负责处理资产扫描记录的 HTTP 请求
type AssetScanHandler struct {
	service *assetService.AssetScanService
}

// NewAssetScanHandler 创建 AssetScanHandler 实例
func NewAssetScanHandler(service *assetService.AssetScanService) *AssetScanHandler {
	return &AssetScanHandler{service: service}
}

// CreateScan 创建扫描记录
func (h *AssetScanHandler) CreateScan(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()
	userAgent := c.GetHeader("User-Agent")

	var scan assetmodel.AssetNetworkScan
	if err := c.ShouldBindJSON(&scan); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_scan",
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

	if err := h.service.CreateScan(c.Request.Context(), &scan); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_scan",
			"network_id": scan.NetworkID,
			"agent_id":   scan.AgentID,
			"user_agent": userAgent,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create scan record",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_scan", 0, "", clientIP, XRequestID, "success", "Scan record created successfully", map[string]interface{}{
		"path":       pathUrl,
		"method":     "POST",
		"scan_id":    scan.ID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Scan record created successfully",
		Data:    scan,
	})
}

// GetScan 获取扫描记录详情
func (h *AssetScanHandler) GetScan(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Scan ID",
			Error:   err.Error(),
		})
		return
	}

	scan, err := h.service.GetScan(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_scan",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to retrieve scan record",
			Error:   err.Error(),
		})
		return
	}

	if scan == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Scan record not found",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Scan record retrieved successfully",
		Data:    scan,
	})
}

// UpdateScan 更新扫描记录
func (h *AssetScanHandler) UpdateScan(c *gin.Context) {
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
			Message: "Invalid Scan ID",
			Error:   err.Error(),
		})
		return
	}

	var scan assetmodel.AssetNetworkScan
	if err := c.ShouldBindJSON(&scan); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_scan",
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

	scan.ID = id
	if err := h.service.UpdateScan(c.Request.Context(), &scan); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_scan",
			"id":         id,
			"user_agent": userAgent,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update scan record",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_scan", 0, "", clientIP, XRequestID, "success", "Scan record updated successfully", map[string]interface{}{
		"path":       pathUrl,
		"method":     "PUT",
		"id":         id,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Scan record updated successfully",
		Data:    scan,
	})
}

// DeleteScan 删除扫描记录
func (h *AssetScanHandler) DeleteScan(c *gin.Context) {
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
			Message: "Invalid Scan ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.DeleteScan(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation":  "delete_scan",
			"id":         id,
			"user_agent": userAgent,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to delete scan record",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_scan", 0, "", clientIP, XRequestID, "success", "Scan record deleted successfully", map[string]interface{}{
		"path":       pathUrl,
		"method":     "DELETE",
		"id":         id,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Scan record deleted successfully",
	})
}

// ListScans 获取扫描记录列表
func (h *AssetScanHandler) ListScans(c *gin.Context) {
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

	var networkID uint64
	if val := c.Query("network_id"); val != "" {
		if nid, err1 := strconv.ParseUint(val, 10, 64); err1 == nil {
			networkID = nid
		}
	}

	var agentID uint64
	if val := c.Query("agent_id"); val != "" {
		if aid, err2 := strconv.ParseUint(val, 10, 64); err2 == nil {
			agentID = aid
		}
	}

	status := c.Query("status")

	scans, total, err := h.service.ListScans(c.Request.Context(), page, pageSize, networkID, agentID, status)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation":  "list_scans",
			"page":       page,
			"page_size":  pageSize,
			"network_id": networkID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list scan records",
			Error:   err.Error(),
		})
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Scan records retrieved successfully",
		Data: system.PaginationResponse{
			Data:        scans,
			Total:       total,
			Page:        page,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			HasNext:     page < totalPages,
			HasPrevious: page > 1,
		},
	})
}

// GetLatestScanByNetworkID 获取指定网段的最新扫描记录
func (h *AssetScanHandler) GetLatestScanByNetworkID(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	networkIDStr := c.Param("network_id")
	if networkIDStr == "" {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "network_id is required",
		})
		return
	}

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

	scan, err := h.service.GetLatestScanByNetworkID(c.Request.Context(), networkID)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation":  "get_latest_scan",
			"network_id": networkID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to retrieve latest scan record",
			Error:   err.Error(),
		})
		return
	}

	if scan == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "No scan record found for this network",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Latest scan record retrieved successfully",
		Data:    scan,
	})
}
