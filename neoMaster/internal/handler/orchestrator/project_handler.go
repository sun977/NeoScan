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

// ProjectHandler 项目处理器
type ProjectHandler struct {
	service *orchestrator.ProjectService
}

// NewProjectHandler 创建 ProjectHandler
func NewProjectHandler(service *orchestrator.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		service: service,
	}
}

// CreateProject 创建项目
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var project orcmodel.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 补充审计信息
	userID := c.GetUint("user_id")
	project.CreatedBy = uint64(userID)
	project.UpdatedBy = uint64(userID)

	if err := h.service.CreateProject(c.Request.Context(), &project); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to create project",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "create_project",
		"option":    "ProjectService.CreateProject",
		"func_name": "handler.orchestrator.project.CreateProject",
	}).Info("项目创建成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Project created successfully",
		Data:    project,
	})
}

// GetProject 获取项目详情
func (h *ProjectHandler) GetProject(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid project ID",
			Error:   err.Error(),
		})
		return
	}

	project, err := h.service.GetProject(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get project",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Success",
		Data:    project,
	})
}

// UpdateProject 更新项目
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid project ID",
			Error:   err.Error(),
		})
		return
	}

	var project orcmodel.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}
	project.ID = id

	// 补充审计信息
	userID := c.GetUint("user_id")
	project.UpdatedBy = uint64(userID)

	if err := h.service.UpdateProject(c.Request.Context(), &project); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to update project",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "update_project",
		"option":    "ProjectService.UpdateProject",
		"func_name": "handler.orchestrator.project.UpdateProject",
	}).Info("项目更新成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Project updated successfully",
		Data:    project,
	})
}

// DeleteProject 删除项目
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid project ID",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.DeleteProject(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to delete project",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":      c.Request.URL.String(),
		"operation": "delete_project",
		"option":    "ProjectService.DeleteProject",
		"func_name": "handler.orchestrator.project.DeleteProject",
	}).Info("项目删除成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Project deleted successfully",
	})
}

// ListProjects 获取项目列表
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	name := c.Query("name")
	status := c.Query("status")
	tagIDStr := c.Query("tag_id")
	var tagID uint64
	if tagIDStr != "" {
		tagID, _ = strconv.ParseUint(tagIDStr, 10, 64)
	}

	// 获取当前用户ID，用于过滤项目（如果是普通用户）
	// 暂时不做权限过滤，后续可以加上
	// userID := c.GetUint("user_id")

	projects, total, err := h.service.ListProjects(c.Request.Context(), page, pageSize, status, name, tagID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to list projects",
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
			Data:       projects,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// AddWorkflowRequest 添加工作流请求参数
type AddWorkflowRequest struct {
	WorkflowID uint64 `json:"workflow_id" binding:"required"`
	SortOrder  int    `json:"sort_order"`
}

// AddWorkflow 关联工作流到项目
func (h *ProjectHandler) AddWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	projectID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid project ID",
			Error:   err.Error(),
		})
		return
	}

	var req AddWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if err := h.service.AddWorkflowToProject(c.Request.Context(), projectID, req.WorkflowID, req.SortOrder); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to add workflow to project",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":        c.Request.URL.String(),
		"operation":   "add_workflow_to_project",
		"project_id":  projectID,
		"workflow_id": req.WorkflowID,
		"func_name":   "handler.orchestrator.project.AddWorkflow",
	}).Info("工作流关联成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Workflow added to project successfully",
	})
}

// RemoveWorkflow 从项目中移除工作流
func (h *ProjectHandler) RemoveWorkflow(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid project ID",
			Error:   err.Error(),
		})
		return
	}

	workflowIDStr := c.Param("workflow_id")
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

	if err := h.service.RemoveWorkflowFromProject(c.Request.Context(), projectID, workflowID); err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to remove workflow from project",
			Error:   err.Error(),
		})
		return
	}

	logger.WithFields(map[string]interface{}{
		"path":        c.Request.URL.String(),
		"operation":   "remove_workflow_from_project",
		"project_id":  projectID,
		"workflow_id": workflowID,
		"func_name":   "handler.orchestrator.project.RemoveWorkflow",
	}).Info("工作流移除成功")

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Workflow removed from project successfully",
	})
}

// GetProjectWorkflows 获取项目关联的工作流列表
func (h *ProjectHandler) GetProjectWorkflows(c *gin.Context) {
	idStr := c.Param("id")
	projectID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Invalid project ID",
			Error:   err.Error(),
		})
		return
	}

	workflows, err := h.service.GetProjectWorkflows(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get project workflows",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Success",
		Data:    workflows,
	})
}
