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

type AssetCPEHandler struct {
	service *assetservice.AssetCPEService
}

func NewAssetCPEHandler(service *assetservice.AssetCPEService) *AssetCPEHandler {
	return &AssetCPEHandler{service: service}
}

func (h *AssetCPEHandler) CreateCPERule(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	var rule assetmodel.AssetCPE
	if err := c.ShouldBindJSON(&rule); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation":  "create_cpe_rule",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.asset.asset_cpe.CreateCPERule",
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

	if err := h.service.CreateCPERule(c.Request.Context(), &rule); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "create_cpe_rule",
			"name":      rule.Name,
			"vendor":    rule.Vendor,
			"product":   rule.Product,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to create CPE rule",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("create_cpe_rule", 0, "", clientIP, XRequestID, "success", "CPE rule created successfully", map[string]interface{}{
		"func_name":  "handler.asset.asset_cpe.CreateCPERule",
		"path":       pathUrl,
		"method":     "POST",
		"name":       rule.Name,
		"id":         rule.ID,
		"user_agent": userAgent,
	})

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "CPE rule created successfully",
		Data:    rule,
	})
}

func (h *AssetCPEHandler) GetCPERule(c *gin.Context) {
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

	rule, err := h.service.GetCPERule(c.Request.Context(), id)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_cpe_rule",
			"id":        id,
		})
		c.JSON(http.StatusNotFound, system.APIResponse{
			Code:    http.StatusNotFound,
			Status:  "failed",
			Message: "CPE rule not found",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "CPE rule retrieved successfully",
		Data:    rule,
	})
}

func (h *AssetCPEHandler) UpdateCPERule(c *gin.Context) {
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

	var rule assetmodel.AssetCPE
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

	if err := h.service.UpdateCPERule(c.Request.Context(), &rule); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "PUT", map[string]interface{}{
			"operation": "update_cpe_rule",
			"id":        id,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to update CPE rule",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("update_cpe_rule", 0, "", clientIP, XRequestID, "success", "CPE rule updated successfully", map[string]interface{}{
		"id":        id,
		"func_name": "handler.asset.asset_cpe.UpdateCPERule",
		"path":      pathUrl,
		"method":    "PUT",
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "CPE rule updated successfully",
	})
}

func (h *AssetCPEHandler) DeleteCPERule(c *gin.Context) {
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

	if err := h.service.DeleteCPERule(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "delete_cpe_rule",
			"id":        id,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to delete CPE rule",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("delete_cpe_rule", 0, "", clientIP, XRequestID, "success", "CPE rule deleted successfully", map[string]interface{}{
		"id":        id,
		"func_name": "handler.asset.asset_cpe.DeleteCPERule",
		"path":      pathUrl,
		"method":    "DELETE",
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "CPE rule deleted successfully",
	})
}

func (h *AssetCPEHandler) ListCPERules(c *gin.Context) {
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
	vendor := c.Query("vendor")
	product := c.Query("product")
	tagIDStr := strings.TrimSpace(c.Query("tag_id"))
	var tagID uint64
	if tagIDStr != "" {
		parsed, err := strconv.ParseUint(tagIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, system.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "failed",
				Message: "Invalid Tag ID",
				Error:   err.Error(),
			})
			return
		}
		tagID = parsed
	}

	list, total, _, err := h.service.ListCPERules(c.Request.Context(), page, pageSize, name, vendor, product, tagID)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "list_cpe_rules",
			"page":      page,
			"page_size": pageSize,
			"name":      name,
			"vendor":    vendor,
			"product":   product,
			"tag_id":    tagID,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "Failed to list CPE rules",
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
		Message: "CPE rules retrieved successfully",
		Data:    pagination,
	})
}

func (h *AssetCPEHandler) AddCPERuleTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	ruleID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Rule ID",
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

	if err := h.service.AddTagToCPERule(c.Request.Context(), ruleID, req.TagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "POST", map[string]interface{}{
			"operation": "add_cpe_rule_tag",
			"rule_id":   ruleID,
			"tag_id":    req.TagID,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to add tag to CPE rule",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("add_cpe_rule_tag", 0, "", clientIP, XRequestID, "success", "Tag added to CPE rule successfully", map[string]interface{}{
		"rule_id":   ruleID,
		"tag_id":    req.TagID,
		"func_name": "handler.asset.asset_cpe.AddCPERuleTag",
		"path":      pathUrl,
		"method":    "POST",
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag added to CPE rule successfully",
	})
}

func (h *AssetCPEHandler) RemoveCPERuleTag(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	ruleID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Rule ID",
			Error:   err.Error(),
		})
		return
	}

	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(strings.TrimSpace(tagIDStr), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Tag ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.RemoveTagFromCPERule(c.Request.Context(), ruleID, tagID); err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "DELETE", map[string]interface{}{
			"operation": "remove_cpe_rule_tag",
			"rule_id":   ruleID,
			"tag_id":    tagID,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to remove tag from CPE rule",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("remove_cpe_rule_tag", 0, "", clientIP, XRequestID, "success", "Tag removed from CPE rule successfully", map[string]interface{}{
		"rule_id":   ruleID,
		"tag_id":    tagID,
		"func_name": "handler.asset.asset_cpe.RemoveCPERuleTag",
		"path":      pathUrl,
		"method":    "DELETE",
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag removed from CPE rule successfully",
	})
}

func (h *AssetCPEHandler) GetCPERuleTags(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	idStr := c.Param("id")
	ruleID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid Rule ID",
			Error:   err.Error(),
		})
		return
	}

	tags, err := h.service.GetCPERuleTags(c.Request.Context(), ruleID)
	if err != nil {
		logger.LogBusinessError(err, XRequestID, 0, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation": "get_cpe_rule_tags",
			"rule_id":   ruleID,
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Failed to get CPE rule tags",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "CPE rule tags retrieved successfully",
		Data:    tags,
	})
}
