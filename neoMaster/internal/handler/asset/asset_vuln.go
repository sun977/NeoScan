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

// AssetVulnHandler 漏洞资产处理器
type AssetVulnHandler struct {
	service *assetservice.AssetVulnService
}

// NewAssetVulnHandler 创建 AssetVulnHandler 实例
func NewAssetVulnHandler(service *assetservice.AssetVulnService) *AssetVulnHandler {
	return &AssetVulnHandler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// AssetVuln Handlers
// -----------------------------------------------------------------------------

// CreateVuln 创建漏洞
func (h *AssetVulnHandler) CreateVuln(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var vuln assetmodel.AssetVuln
	if err := c.ShouldBindJSON(&vuln); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_vuln",
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

	if err := h.service.CreateVuln(c.Request.Context(), &vuln); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_vuln",
			"cve":       vuln.CVE,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create vulnerability",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_vuln", 0, "", clientIP, XRequestID, "success", "Vulnerability created successfully", map[string]interface{}{
		"id":         vuln.ID,
		"cve":        vuln.CVE,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Vulnerability created successfully",
		Data:    vuln,
	})
}

// GetVuln 获取漏洞详情
func (h *AssetVulnHandler) GetVuln(c *gin.Context) {
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

	vuln, err := h.service.GetVulnByID(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_vuln",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get vulnerability",
			Error:   err.Error(),
		})
		return
	}

	if vuln == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "Vulnerability not found",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Vulnerability retrieved successfully",
		Data:    vuln,
	})
}

// UpdateVuln 更新漏洞信息
func (h *AssetVulnHandler) UpdateVuln(c *gin.Context) {
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

	var vuln assetmodel.AssetVuln
	if err := c.ShouldBindJSON(&vuln); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_vuln",
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
	vuln.ID = id

	if err := h.service.UpdateVuln(c.Request.Context(), &vuln); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_vuln",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update vulnerability",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_vuln", 0, "", clientIP, XRequestID, "success", "Vulnerability updated successfully", map[string]interface{}{
		"id":         id,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Vulnerability updated successfully",
	})
}

// DeleteVuln 删除漏洞
func (h *AssetVulnHandler) DeleteVuln(c *gin.Context) {
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

	if err := h.service.DeleteVuln(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_vuln",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to delete vulnerability",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_vuln", 0, "", clientIP, XRequestID, "success", "Vulnerability deleted successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Vulnerability deleted successfully",
	})
}

// ListVulns 获取漏洞列表
func (h *AssetVulnHandler) ListVulns(c *gin.Context) {
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

	targetType := c.Query("target_type")
	targetRefIDStr := c.Query("target_ref_id")
	var targetRefID uint64
	if targetRefIDStr != "" {
		targetRefID, _ = strconv.ParseUint(targetRefIDStr, 10, 64)
	}
	status := c.Query("status")
	severity := c.Query("severity")
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

	vulns, total, err := h.service.ListVulns(c.Request.Context(), page, pageSize, targetType, targetRefID, status, severity, tagIDs)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_vulns",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list vulnerabilities",
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
		Data:        vulns,
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Vulnerabilities retrieved successfully",
		Data:    pagination,
	})
}

// -----------------------------------------------------------------------------
// AssetVulnPoc Handlers
// -----------------------------------------------------------------------------

// CreatePoc 创建PoC
func (h *AssetVulnHandler) CreatePoc(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var poc assetmodel.AssetVulnPoc
	if err := c.ShouldBindJSON(&poc); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_poc",
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

	if err := h.service.CreatePoc(c.Request.Context(), &poc); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_poc",
			"name":      poc.Name,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to create PoC",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_poc", 0, "", clientIP, XRequestID, "success", "PoC created successfully", map[string]interface{}{
		"id":         poc.ID,
		"name":       poc.Name,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "PoC created successfully",
		Data:    poc,
	})
}

// GetPoc 获取PoC详情
func (h *AssetVulnHandler) GetPoc(c *gin.Context) {
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

	poc, err := h.service.GetPocByID(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_poc",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get PoC",
			Error:   err.Error(),
		})
		return
	}

	if poc == nil {
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "PoC not found",
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "PoC retrieved successfully",
		Data:    poc,
	})
}

// UpdatePoc 更新PoC
func (h *AssetVulnHandler) UpdatePoc(c *gin.Context) {
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

	var poc assetmodel.AssetVulnPoc
	if err := c.ShouldBindJSON(&poc); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation":  "update_poc",
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
	poc.ID = id

	if err := h.service.UpdatePoc(c.Request.Context(), &poc); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_poc",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to update PoC",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_poc", 0, "", clientIP, XRequestID, "success", "PoC updated successfully", map[string]interface{}{
		"id":         id,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "PoC updated successfully",
	})
}

// DeletePoc 删除PoC
func (h *AssetVulnHandler) DeletePoc(c *gin.Context) {
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

	if err := h.service.DeletePoc(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_poc",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to delete PoC",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_poc", 0, "", clientIP, XRequestID, "success", "PoC deleted successfully", map[string]interface{}{
		"id": id,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "PoC deleted successfully",
	})
}

// ListPocsByVuln 获取漏洞关联的PoC列表
func (h *AssetVulnHandler) ListPocsByVuln(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	vulnIDStr := c.Param("id")
	vulnID, err := strconv.ParseUint(vulnIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Vulnerability ID",
			Error:   err.Error(),
		})
		return
	}

	validOnly := c.Query("valid_only") == "true"
	tagIDsStr := c.Query("tag_ids")
	var tagIDs []uint64
	if tagIDsStr != "" {
		parts := strings.Split(tagIDsStr, ",")
		for _, part := range parts {
			id, err1 := strconv.ParseUint(part, 10, 64)
			if err1 != nil {
				continue
			}
			tagIDs = append(tagIDs, id)
		}
	}

	var pocs []*assetmodel.AssetVulnPoc
	if validOnly {
		pocs, err = h.service.GetValidPocsByVulnID(c.Request.Context(), vulnID)
	} else {
		pocs, err = h.service.ListAllPocsByVulnID(c.Request.Context(), vulnID, tagIDs)
	}

	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_pocs_by_vuln",
			"vuln_id":   vulnID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list PoCs",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "PoCs retrieved successfully",
		Data:    pocs,
	})
}

// -----------------------------------------------------------------------------
// Tag Management Handlers
// -----------------------------------------------------------------------------

// GetVulnTags 获取漏洞标签
func (h *AssetVulnHandler) GetVulnTags(c *gin.Context) {
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

	tags, err := h.service.GetVulnTags(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_vuln_tags",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get vulnerability tags",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Vulnerability tags retrieved successfully",
		Data:    tags,
	})
}

// AddVulnTag 为漏洞添加标签
func (h *AssetVulnHandler) AddVulnTag(c *gin.Context) {
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

	if err := h.service.AddVulnTag(c.Request.Context(), id, req.TagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "add_vuln_tag",
			"id":        id,
			"tag_id":    req.TagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to add tag to vulnerability",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("add_vuln_tag", 0, "", clientIP, XRequestID, "success", "Tag added to vulnerability successfully", map[string]interface{}{
		"id":     id,
		"tag_id": req.TagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag added to vulnerability successfully",
	})
}

// RemoveVulnTag 为漏洞移除标签
func (h *AssetVulnHandler) RemoveVulnTag(c *gin.Context) {
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

	if err := h.service.RemoveVulnTag(c.Request.Context(), id, tagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "remove_vuln_tag",
			"id":        id,
			"tag_id":    tagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to remove tag from vulnerability",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("remove_vuln_tag", 0, "", clientIP, XRequestID, "success", "Tag removed from vulnerability successfully", map[string]interface{}{
		"id":     id,
		"tag_id": tagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag removed from vulnerability successfully",
	})
}

// GetPocTags 获取PoC标签
func (h *AssetVulnHandler) GetPocTags(c *gin.Context) {
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

	tags, err := h.service.GetPocTags(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_poc_tags",
			"id":        id,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to get PoC tags",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "PoC tags retrieved successfully",
		Data:    tags,
	})
}

// AddPocTag 为PoC添加标签
func (h *AssetVulnHandler) AddPocTag(c *gin.Context) {
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

	if err := h.service.AddPocTag(c.Request.Context(), id, req.TagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "add_poc_tag",
			"id":        id,
			"tag_id":    req.TagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to add tag to PoC",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("add_poc_tag", 0, "", clientIP, XRequestID, "success", "Tag added to PoC successfully", map[string]interface{}{
		"id":     id,
		"tag_id": req.TagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag added to PoC successfully",
	})
}

// RemovePocTag 为PoC移除标签
func (h *AssetVulnHandler) RemovePocTag(c *gin.Context) {
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

	if err := h.service.RemovePocTag(c.Request.Context(), id, tagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "remove_poc_tag",
			"id":        id,
			"tag_id":    tagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to remove tag from PoC",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("remove_poc_tag", 0, "", clientIP, XRequestID, "success", "Tag removed from PoC successfully", map[string]interface{}{
		"id":     id,
		"tag_id": tagID,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag removed from PoC successfully",
	})
}
