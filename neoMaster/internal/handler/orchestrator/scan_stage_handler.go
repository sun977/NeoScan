package orchestrator

import (
	"net/http"
	"strconv"

	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/orchestrator"

	"github.com/gin-gonic/gin"
)

// ScanStageHandler 扫描阶段处理器
type ScanStageHandler struct {
	service *orchestrator.ScanStageService
}

// NewScanStageHandler 创建 ScanStageHandler
func NewScanStageHandler(service *orchestrator.ScanStageService) *ScanStageHandler {
	return &ScanStageHandler{
		service: service,
	}
}

// CreateStage 创建扫描阶段
func (h *ScanStageHandler) CreateStage(c *gin.Context) {
	var stage orcmodel.ScanStage
	if err := c.ShouldBindJSON(&stage); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.CreateStage(c.Request.Context(), &stage); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to create scan stage",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "create_stage",
		"option":    "ScanStageService.CreateStage",
		"func_name": "handler.orchestrator.scan_stage.CreateStage",
	}).Info("扫描阶段创建成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Scan stage created successfully",
		Data:    stage,
	})
}

// GetStage 获取扫描阶段详情
func (h *ScanStageHandler) GetStage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid stage ID",
			Error:   err.Error(),
		})
		return
	}

	stage, err := h.service.GetStage(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get scan stage",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Success",
		Data:    stage,
	})
}

// UpdateStage 更新扫描阶段
func (h *ScanStageHandler) UpdateStage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid stage ID",
			Error:   err.Error(),
		})
		return
	}

	var stage orcmodel.ScanStage
	if err := c.ShouldBindJSON(&stage); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}
	stage.ID = id

	if err := h.service.UpdateStage(c.Request.Context(), &stage); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to update scan stage",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "update_stage",
		"option":    "ScanStageService.UpdateStage",
		"func_name": "handler.orchestrator.scan_stage.UpdateStage",
	}).Info("扫描阶段更新成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Scan stage updated successfully",
		Data:    stage,
	})
}

// DeleteStage 删除扫描阶段
func (h *ScanStageHandler) DeleteStage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid stage ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.DeleteStage(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to delete scan stage",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "delete_stage",
		"option":    "ScanStageService.DeleteStage",
		"func_name": "handler.orchestrator.scan_stage.DeleteStage",
	}).Info("扫描阶段删除成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Scan stage deleted successfully",
	})
}

// ListStages 获取工作流的所有阶段
func (h *ScanStageHandler) ListStages(c *gin.Context) {
	workflowIDStr := c.Query("workflow_id")
	if workflowIDStr == "" {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "workflow_id is required",
		})
		return
	}

	workflowID, err := strconv.ParseUint(workflowIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid workflow ID",
			Error:   err.Error(),
		})
		return
	}

	stages, err := h.service.ListStagesByWorkflowID(c.Request.Context(), workflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to list scan stages",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Success",
		Data:    stages,
	})
}
