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

	// 标签筛选参数（可选）：与 orchestrator 其他列表接口保持一致，使用 tag_id。
	// tag_id=0 或缺失代表不筛选，保持向后兼容。
	tagIDStr := c.Query("tag_id")
	var tagID uint64
	if tagIDStr != "" {
		// 这里不强制返回 400，以避免客户端传入非数字时导致行为破坏；
		// 与 workflow/project 的 tag_id 解析策略保持一致：解析失败则视为未传入筛选。
		tagID, _ = strconv.ParseUint(tagIDStr, 10, 64)
	}

	var stages []*orcmodel.ScanStage
	if tagID > 0 {
		// 标签筛选：调用服务层获取带标签的阶段列表。
		stages, err = h.service.ListStagesByWorkflowIDWithTag(c.Request.Context(), workflowID, tagID)
	} else {
		// 无标签筛选：直接返回所有阶段。
		stages, err = h.service.ListStagesByWorkflowID(c.Request.Context(), workflowID)
	}
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

// AddStageTagRequest 添加阶段标签请求
type AddStageTagRequest struct {
	TagID uint64 `json:"tag_id" binding:"required"`
}

// AddStageTag 为扫描阶段添加标签
func (h *ScanStageHandler) AddStageTag(c *gin.Context) {
	idStr := c.Param("id")
	stageID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid stage ID",
			Error:   err.Error(),
		})
		return
	}

	var req AddStageTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.AddTagToStage(c.Request.Context(), stageID, req.TagID); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to add tag to stage",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "add_tag_to_stage",
		"stage_id":  stageID,
		"tag_id":    req.TagID,
		"func_name": "handler.orchestrator.scan_stage.AddStageTag",
	}).Info("扫描阶段标签添加成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag added to stage successfully",
	})
}

// RemoveStageTag 从扫描阶段移除标签
func (h *ScanStageHandler) RemoveStageTag(c *gin.Context) {
	idStr := c.Param("id")
	stageID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid stage ID",
			Error:   err.Error(),
		})
		return
	}

	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid tag ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.RemoveTagFromStage(c.Request.Context(), stageID, tagID); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to remove tag from stage",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "remove_tag_from_stage",
		"stage_id":  stageID,
		"tag_id":    tagID,
		"func_name": "handler.orchestrator.scan_stage.RemoveStageTag",
	}).Info("扫描阶段标签移除成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Tag removed from stage successfully",
	})
}

// GetStageTags 获取扫描阶段标签列表
func (h *ScanStageHandler) GetStageTags(c *gin.Context) {
	idStr := c.Param("id")
	stageID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid stage ID",
			Error:   err.Error(),
		})
		return
	}

	tags, err := h.service.GetStageTags(c.Request.Context(), stageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get stage tags",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Success",
		Data:    tags,
	})
}
