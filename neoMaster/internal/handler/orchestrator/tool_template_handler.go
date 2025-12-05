package orchestrator

import (
	"math"
	"net/http"
	"strconv"

	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/orchestrator"

	"github.com/gin-gonic/gin"
)

// ScanToolTemplateHandler 工具模板处理器
type ScanToolTemplateHandler struct {
	service *orchestrator.ScanToolTemplateService
}

// NewScanToolTemplateHandler 创建 ScanToolTemplateHandler
func NewScanToolTemplateHandler(service *orchestrator.ScanToolTemplateService) *ScanToolTemplateHandler {
	return &ScanToolTemplateHandler{
		service: service,
	}
}

// CreateTemplate 创建工具模板
func (h *ScanToolTemplateHandler) CreateTemplate(c *gin.Context) {
	var tmpl orcmodel.ScanToolTemplate
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 补充审计信息
	userID := c.GetUint("user_id")
	tmpl.CreatedBy = strconv.FormatUint(uint64(userID), 10)

	if err := h.service.CreateTemplate(c.Request.Context(), &tmpl); err != nil {
		logger.LogBusinessError(err, c.Request.URL.String(), userID, "", "CreateTemplate", "HANDLER", nil)
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to create tool template",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Tool template created successfully",
		Data:    map[string]interface{}{"id": tmpl.ID},
	})
}

// GetTemplate 获取工具模板详情
func (h *ScanToolTemplateHandler) GetTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid template ID",
		})
		return
	}

	tmpl, err := h.service.GetTemplate(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get tool template",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Success",
		Data:    tmpl,
	})
}

// UpdateTemplate 更新工具模板
func (h *ScanToolTemplateHandler) UpdateTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid template ID",
		})
		return
	}

	var tmpl orcmodel.ScanToolTemplate
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	tmpl.ID = uint64(id)

	if err := h.service.UpdateTemplate(c.Request.Context(), &tmpl); err != nil {
		logger.LogBusinessError(err, c.Request.URL.String(), c.GetUint("user_id"), "", "UpdateTemplate", "HANDLER", nil)
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to update tool template",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tool template updated successfully",
	})
}

// DeleteTemplate 删除工具模板
func (h *ScanToolTemplateHandler) DeleteTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid template ID",
		})
		return
	}

	if err := h.service.DeleteTemplate(c.Request.Context(), id); err != nil {
		logger.LogBusinessError(err, c.Request.URL.String(), c.GetUint("user_id"), "", "DeleteTemplate", "HANDLER", nil)
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to delete tool template",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tool template deleted successfully",
	})
}

// ListTemplates 获取工具模板列表
func (h *ScanToolTemplateHandler) ListTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	toolName := c.Query("tool_name")
	category := c.Query("category")

	var isPublic *bool
	if val := c.Query("is_public"); val != "" {
		b, err := strconv.ParseBool(val)
		if err == nil {
			isPublic = &b
		}
	}

	tmpls, total, err := h.service.ListTemplates(c.Request.Context(), page, pageSize, toolName, category, isPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to list tool templates",
			Error:   err.Error(),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Success",
		Data: system.PaginationResponse{
			Data:       tmpls,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}
