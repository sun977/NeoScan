package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// WorkflowRepository 工作流仓库
type WorkflowRepository struct {
	db *gorm.DB
}

// NewWorkflowRepository 创建 WorkflowRepository 实例
func NewWorkflowRepository(db *gorm.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// -----------------------------------------------------------------------------
// Workflow CRUD
// -----------------------------------------------------------------------------

// CreateWorkflow 创建工作流
func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, workflow *orcmodel.Workflow) error {
	if workflow == nil {
		return errors.New("workflow is nil")
	}
	err := r.db.WithContext(ctx).Create(workflow).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_workflow", "REPO", map[string]interface{}{
			"operation": "create_workflow",
			"name":      workflow.Name,
		})
		return err
	}
	return nil
}

// GetWorkflowsByProjectID 获取项目关联的所有工作流
func (r *WorkflowRepository) GetWorkflowsByProjectID(ctx context.Context, projectID uint64) ([]*orcmodel.Workflow, error) {
	var workflows []*orcmodel.Workflow
	// 使用 Join 查询关联的工作流
	err := r.db.WithContext(ctx).
		Table("workflows").
		Joins("JOIN project_workflows ON workflows.id = project_workflows.workflow_id").
		Where("project_workflows.project_id = ?", projectID).
		Order("project_workflows.sort_order ASC").
		Find(&workflows).Error

	if err != nil {
		logger.LogError(err, "", 0, "", "get_workflows_by_project_id", "REPO", map[string]interface{}{
			"operation":  "get_workflows_by_project_id",
			"project_id": projectID,
		})
		return nil, err
	}
	return workflows, nil
}

// GetWorkflowByID 根据ID获取工作流
func (r *WorkflowRepository) GetWorkflowByID(ctx context.Context, id uint64) (*orcmodel.Workflow, error) {
	var workflow orcmodel.Workflow
	err := r.db.WithContext(ctx).First(&workflow, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_workflow_by_id", "REPO", map[string]interface{}{
			"operation": "get_workflow_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &workflow, nil
}

// UpdateWorkflow 更新工作流
func (r *WorkflowRepository) UpdateWorkflow(ctx context.Context, workflow *orcmodel.Workflow) error {
	if workflow == nil || workflow.ID == 0 {
		return errors.New("invalid workflow or id")
	}
	err := r.db.WithContext(ctx).Save(workflow).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_workflow", "REPO", map[string]interface{}{
			"operation": "update_workflow",
			"id":        workflow.ID,
		})
		return err
	}
	return nil
}

// DeleteWorkflow 删除工作流 (软删除)
func (r *WorkflowRepository) DeleteWorkflow(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&orcmodel.Workflow{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_workflow", "REPO", map[string]interface{}{
			"operation": "delete_workflow",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListWorkflows 获取工作流列表 (分页 + 筛选)
func (r *WorkflowRepository) ListWorkflows(ctx context.Context, page, pageSize int, name string, enabled *bool) ([]*orcmodel.Workflow, int64, error) {
	var workflows []*orcmodel.Workflow
	var total int64

	query := r.db.WithContext(ctx).Model(&orcmodel.Workflow{})

	if name != "" {
		query = query.Where("name LIKE ? OR display_name LIKE ?", "%"+name+"%", "%"+name+"%")
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_workflows_count", "REPO", map[string]interface{}{
			"operation": "list_workflows_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&workflows).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_workflows_find", "REPO", map[string]interface{}{
			"operation": "list_workflows_find",
		})
		return nil, 0, err
	}

	return workflows, total, nil
}

// -----------------------------------------------------------------------------
// ProjectWorkflow Association
// -----------------------------------------------------------------------------

// AddWorkflowToProject 将工作流关联到项目
func (r *WorkflowRepository) AddWorkflowToProject(ctx context.Context, relation *orcmodel.ProjectWorkflow) error {
	if relation == nil {
		return errors.New("relation is nil")
	}
	err := r.db.WithContext(ctx).Create(relation).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "add_workflow_to_project", "REPO", map[string]interface{}{
			"operation":   "add_workflow_to_project",
			"project_id":  relation.ProjectID,
			"workflow_id": relation.WorkflowID,
		})
		return err
	}
	return nil
}

// RemoveWorkflowFromProject 从项目移除工作流
func (r *WorkflowRepository) RemoveWorkflowFromProject(ctx context.Context, projectID, workflowID uint64) error {
	err := r.db.WithContext(ctx).Where("project_id = ? AND workflow_id = ?", projectID, workflowID).Delete(&orcmodel.ProjectWorkflow{}).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "remove_workflow_from_project", "REPO", map[string]interface{}{
			"operation":   "remove_workflow_from_project",
			"project_id":  projectID,
			"workflow_id": workflowID,
		})
		return err
	}
	return nil
}

// ListWorkflowsByProjectID 获取项目关联的所有工作流
func (r *WorkflowRepository) ListWorkflowsByProjectID(ctx context.Context, projectID uint64) ([]*orcmodel.Workflow, error) {
	var workflows []*orcmodel.Workflow
	// 使用 Join 查询
	err := r.db.WithContext(ctx).Table("workflows").
		Joins("JOIN project_workflows ON project_workflows.workflow_id = workflows.id").
		Where("project_workflows.project_id = ? AND workflows.deleted_at IS NULL", projectID).
		Order("project_workflows.sort_order ASC").
		Find(&workflows).Error

	if err != nil {
		logger.LogError(err, "", 0, "", "list_workflows_by_project_id", "REPO", map[string]interface{}{
			"operation":  "list_workflows_by_project_id",
			"project_id": projectID,
		})
		return nil, err
	}
	return workflows, nil
}
