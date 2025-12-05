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

// WorkflowHandler 工作流处理器
type WorkflowHandler struct {
	service *orchestrator.WorkflowService
}

// NewWorkflowHandler 创建 WorkflowHandler
func NewWorkflowHandler(service *orchestrator.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{
		service: service,
	}
}

// CreateWorkflow 创建工作流
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	var workflow orcmodel.Workflow
	if err := c.ShouldBindJSON(&workflow); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.CreateWorkflow(c.Request.Context(), &workflow); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to create workflow",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "create_workflow",
		"option":    "WorkflowService.CreateWorkflow",
		"func_name": "handler.orchestrator.workflow.CreateWorkflow",
	}).Info("工作流创建成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Workflow created successfully",
		Data:    workflow,
	})
}

// GetWorkflow 获取工作流详情
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid workflow ID",
			Error:   err.Error(),
		})
		return
	}

	workflow, err := h.service.GetWorkflow(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get workflow",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Success",
		Data:    workflow,
	})
}

// UpdateWorkflow 更新工作流
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid workflow ID",
			Error:   err.Error(),
		})
		return
	}

	var workflow orcmodel.Workflow
	if err := c.ShouldBindJSON(&workflow); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}
	workflow.ID = id

	if err := h.service.UpdateWorkflow(c.Request.Context(), &workflow); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to update workflow",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "update_workflow",
		"option":    "WorkflowService.UpdateWorkflow",
		"func_name": "handler.orchestrator.workflow.UpdateWorkflow",
	}).Info("工作流更新成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Workflow updated successfully",
		Data:    workflow,
	})
}

// DeleteWorkflow 删除工作流
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid workflow ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.DeleteWorkflow(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to delete workflow",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "delete_workflow",
		"option":    "WorkflowService.DeleteWorkflow",
		"func_name": "handler.orchestrator.workflow.DeleteWorkflow",
	}).Info("工作流删除成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Workflow deleted successfully",
	})
}

// ListWorkflows 获取工作流列表
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	name := c.Query("name")

	var enabled *bool
	if val := c.Query("enabled"); val != "" {
		b, err := strconv.ParseBool(val)
		if err == nil {
			enabled = &b
		}
	}

	workflows, total, err := h.service.ListWorkflows(c.Request.Context(), page, pageSize, name, enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to list workflows",
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
			Data:       workflows,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}
