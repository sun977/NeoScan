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

// AssetWebHandler Web资产处理器
type AssetWebHandler struct {
	service *assetservice.AssetWebService
}

// NewAssetWebHandler 创建 AssetWebHandler 实例
func NewAssetWebHandler(service *assetservice.AssetWebService) *AssetWebHandler {
	return &AssetWebHandler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// AssetWeb Handlers
// -----------------------------------------------------------------------------

// CreateWeb 创建Web资产
func (h *AssetWebHandler) CreateWeb(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var web assetmodel.AssetWeb
	if err := c.ShouldBindJSON(&web); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_web",
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

	if err := h.service.CreateWeb(c.Request.Context(), &web); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_web",
			"url":       web.URL,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create web asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_web", 0, "", clientIP, XRequestID, "success", "Web asset created successfully", map[string]interface{}{
		"id":         web.ID,
		"url":        web.URL,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Web asset created successfully",
		Data:    web,
	})
}

// GetWeb 获取Web资产详情
func (h *AssetWebHandler) GetWeb(c *gin.Context) {
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

	web, err := h.service.GetWebByID(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_web",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get web asset",
			Error:   err.Error(),
		})
		return
	}

	if web == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Web asset not found",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Web asset retrieved successfully",
		Data:    web,
	})
}

// UpdateWeb 更新Web资产
func (h *AssetWebHandler) UpdateWeb(c *gin.Context) {
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
			Error:   err.Error(),
		})
		return
	}

	var web assetmodel.AssetWeb
	if err := c.ShouldBindJSON(&web); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_web",
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
	web.ID = id

	if err := h.service.UpdateWeb(c.Request.Context(), &web); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_web",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update web asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_web", 0, "", clientIP, XRequestID, "success", "Web asset updated successfully", map[string]interface{}{
		"id":         id,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Web asset updated successfully",
	})
}

// DeleteWeb 删除Web资产
func (h *AssetWebHandler) DeleteWeb(c *gin.Context) {
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

	if err := h.service.DeleteWeb(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_web",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to delete web asset",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_web", 0, "", clientIP, XRequestID, "success", "Web asset deleted successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Web asset deleted successfully",
	})
}

// ListWebs 获取Web资产列表
func (h *AssetWebHandler) ListWebs(c *gin.Context) {
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

	domain := c.Query("domain")
	assetType := c.Query("asset_type")
	status := c.Query("status")

	webs, total, err := h.service.ListWebs(c.Request.Context(), page, pageSize, domain, assetType, status)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_webs",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list web assets",
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
		Data:        webs,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Web assets retrieved successfully",
		Data:    pagination,
	})
}

// -----------------------------------------------------------------------------
// AssetWebDetail Handlers
// -----------------------------------------------------------------------------

// GetWebDetail 获取Web详细信息
func (h *AssetWebHandler) GetWebDetail(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	webID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Web ID",
			Error:   err.Error(),
		})
		return
	}

	detail, err := h.service.GetWebDetail(c.Request.Context(), webID)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_web_detail",
			"web_id":    webID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get web detail",
			Error:   err.Error(),
		})
		return
	}

	if detail == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Web detail not found",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Web detail retrieved successfully",
		Data:    detail,
	})
}

// SaveWebDetail 保存Web详细信息
func (h *AssetWebHandler) SaveWebDetail(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	webID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Web ID",
			Error:   err.Error(),
		})
		return
	}

	var detail assetmodel.AssetWebDetail
	if err := c.ShouldBindJSON(&detail); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "save_web_detail",
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
	detail.AssetWebID = webID // Ensure ID matches path param

	if err := h.service.SaveWebDetail(c.Request.Context(), &detail); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "save_web_detail",
			"web_id":    webID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to save web detail",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("save_web_detail", 0, "", clientIP, XRequestID, "success", "Web detail saved successfully", map[string]interface{}{
		"web_id":     webID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Web detail saved successfully",
	})
}
