package orchestrator

import (
	"context"
	"errors"
	"strconv"

	orcmodel "neomaster/internal/model/orchestrator"
	tagmodel "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/tag_system"
)

// WorkflowService 工作流服务
type WorkflowService struct {
	repo       *orcrepo.WorkflowRepository
	tagService tag_system.TagService
}

// NewWorkflowService 创建 WorkflowService 实例
func NewWorkflowService(repo *orcrepo.WorkflowRepository, tagService tag_system.TagService) *WorkflowService {
	return &WorkflowService{
		repo:       repo,
		tagService: tagService,
	}
}

// CreateWorkflow 创建工作流
func (s *WorkflowService) CreateWorkflow(ctx context.Context, workflow *orcmodel.Workflow) error {
	if workflow == nil {
		return errors.New("workflow data cannot be nil")
	}
	// 校验 Name 唯一性等逻辑可以在这里加，或者依赖 DB 唯一索引
	err := s.repo.CreateWorkflow(ctx, workflow)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_workflow", "SERVICE", map[string]interface{}{
			"operation": "create_workflow",
			"name":      workflow.Name,
		})
		return err
	}
	return nil
}

// GetWorkflow 获取工作流详情
func (s *WorkflowService) GetWorkflow(ctx context.Context, id uint64) (*orcmodel.Workflow, error) {
	workflow, err := s.repo.GetWorkflowByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_workflow", "SERVICE", map[string]interface{}{
			"operation": "get_workflow",
			"id":        id,
		})
		return nil, err
	}
	if workflow == nil {
		return nil, errors.New("workflow not found")
	}
	return workflow, nil
}

// UpdateWorkflow 更新工作流
func (s *WorkflowService) UpdateWorkflow(ctx context.Context, workflow *orcmodel.Workflow) error {
	if workflow == nil {
		return errors.New("workflow data cannot be nil")
	}
	existing, err := s.repo.GetWorkflowByID(ctx, workflow.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("workflow not found")
	}

	err = s.repo.UpdateWorkflow(ctx, workflow)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_workflow", "SERVICE", map[string]interface{}{
			"operation": "update_workflow",
			"id":        workflow.ID,
		})
		return err
	}
	return nil
}

// DeleteWorkflow 删除工作流
func (s *WorkflowService) DeleteWorkflow(ctx context.Context, id uint64) error {
	existing, err := s.repo.GetWorkflowByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("workflow not found")
	}

	err = s.repo.DeleteWorkflow(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_workflow", "SERVICE", map[string]interface{}{
			"operation": "delete_workflow",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListWorkflows 获取工作流列表
func (s *WorkflowService) ListWorkflows(ctx context.Context, page, pageSize int, name string, enabled *bool, tagID uint64) ([]*orcmodel.Workflow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	workflows, total, err := s.repo.ListWorkflows(ctx, page, pageSize, name, enabled, tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_workflows", "SERVICE", map[string]interface{}{
			"operation": "list_workflows",
		})
		return nil, 0, err
	}
	return workflows, total, nil
}

// -----------------------------------------------------------------------------
// Association Logic
// -----------------------------------------------------------------------------

// AddWorkflowToProject 将工作流添加到项目
func (s *WorkflowService) AddWorkflowToProject(ctx context.Context, projectID, workflowID uint64, sortOrder int) error {
	relation := &orcmodel.ProjectWorkflow{
		ProjectID:  projectID,
		WorkflowID: workflowID,
		SortOrder:  sortOrder,
	}
	err := s.repo.AddWorkflowToProject(ctx, relation)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_workflow_to_project", "SERVICE", map[string]interface{}{
			"operation":   "add_workflow_to_project",
			"project_id":  projectID,
			"workflow_id": workflowID,
		})
		return err
	}
	return nil
}

// RemoveWorkflowFromProject 从项目移除工作流
func (s *WorkflowService) RemoveWorkflowFromProject(ctx context.Context, projectID, workflowID uint64) error {
	err := s.repo.RemoveWorkflowFromProject(ctx, projectID, workflowID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_workflow_from_project", "SERVICE", map[string]interface{}{
			"operation":   "remove_workflow_from_project",
			"project_id":  projectID,
			"workflow_id": workflowID,
		})
		return err
	}
	return nil
}

// ListWorkflowsByProjectID 获取项目的工作流
func (s *WorkflowService) ListWorkflowsByProjectID(ctx context.Context, projectID uint64) ([]*orcmodel.Workflow, error) {
	workflows, err := s.repo.ListWorkflowsByProjectID(ctx, projectID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_workflows_by_project_id", "SERVICE", map[string]interface{}{
			"operation":  "list_workflows_by_project_id",
			"project_id": projectID,
		})
		return nil, err
	}
	return workflows, nil
}

// AddTagToWorkflow 为工作流添加标签
func (s *WorkflowService) AddTagToWorkflow(ctx context.Context, workflowID uint64, tagID uint64) error {
	// 1. 检查工作流是否存在
	workflow, err := s.repo.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		return err
	}
	if workflow == nil {
		return errors.New("workflow not found")
	}

	// 2. 调用 TagService 添加标签
	entityIDStr := strconv.FormatUint(workflowID, 10)
	err = s.tagService.AddEntityTag(ctx, "workflow", entityIDStr, tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_workflow", "SERVICE", map[string]interface{}{
			"operation":   "add_tag_to_workflow",
			"workflow_id": workflowID,
			"tag_id":      tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromWorkflow 从工作流移除标签
func (s *WorkflowService) RemoveTagFromWorkflow(ctx context.Context, workflowID uint64, tagID uint64) error {
	// 1. 检查工作流是否存在
	workflow, err := s.repo.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		return err
	}
	if workflow == nil {
		return errors.New("workflow not found")
	}

	// 2. 调用 TagService 移除标签
	entityIDStr := strconv.FormatUint(workflowID, 10)
	err = s.tagService.RemoveEntityTag(ctx, "workflow", entityIDStr, tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_workflow", "SERVICE", map[string]interface{}{
			"operation":   "remove_tag_from_workflow",
			"workflow_id": workflowID,
			"tag_id":      tagID,
		})
		return err
	}
	return nil
}

// GetWorkflowTags 获取工作流的所有标签
func (s *WorkflowService) GetWorkflowTags(ctx context.Context, workflowID uint64) ([]tagmodel.SysEntityTag, error) {
	// 1. 检查工作流是否存在
	workflow, err := s.repo.GetWorkflowByID(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, errors.New("workflow not found")
	}

	// 2. 调用 TagService 获取标签
	entityIDStr := strconv.FormatUint(workflowID, 10)
	tags, err := s.tagService.GetEntityTags(ctx, "workflow", entityIDStr)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_workflow_tags", "SERVICE", map[string]interface{}{
			"operation":   "get_workflow_tags",
			"workflow_id": workflowID,
		})
		return nil, err
	}
	return tags, nil
}
