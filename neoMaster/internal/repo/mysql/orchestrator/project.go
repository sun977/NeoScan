package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// ProjectRepository 项目仓库
// 负责 Project 的数据访问
type ProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository 创建 ProjectRepository 实例
func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// -----------------------------------------------------------------------------
// Project (项目) CRUD
// -----------------------------------------------------------------------------

// CreateProject 创建项目
func (r *ProjectRepository) CreateProject(ctx context.Context, project *orcmodel.Project) error {
	if project == nil {
		return errors.New("project is nil")
	}

	// Ensure JSON fields are valid JSON objects if empty
	if project.NotifyConfig == "" {
		project.NotifyConfig = "{}"
	}
	if project.ExportConfig == "" {
		project.ExportConfig = "{}"
	}
	if project.ExtendedData == "" {
		project.ExtendedData = "{}"
	}

	err := r.db.WithContext(ctx).Create(project).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_project", "REPO", map[string]interface{}{
			"operation": "create_project",
			"name":      project.Name,
		})
		return err
	}
	return nil
}

// GetRunningProjects 获取所有正在运行的项目
func (r *ProjectRepository) GetRunningProjects(ctx context.Context) ([]*orcmodel.Project, error) {
	var projects []*orcmodel.Project
	err := r.db.WithContext(ctx).Where("status = ?", "running").Find(&projects).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "get_running_projects", "REPO", map[string]interface{}{
			"operation": "get_running_projects",
		})
		return nil, err
	}
	return projects, nil
}

// GetScheduledProjects 获取所有配置了Cron调度的空闲项目
func (r *ProjectRepository) GetScheduledProjects(ctx context.Context) ([]*orcmodel.Project, error) {
	var projects []*orcmodel.Project
	err := r.db.WithContext(ctx).Where("status = ? AND enabled = ? AND schedule_type = ?", "idle", true, "cron").Find(&projects).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "get_scheduled_projects", "REPO", map[string]interface{}{
			"operation": "get_scheduled_projects",
		})
		return nil, err
	}
	return projects, nil
}

// GetProjectByID 根据ID获取项目
func (r *ProjectRepository) GetProjectByID(ctx context.Context, id uint64) (*orcmodel.Project, error) {
	var project orcmodel.Project
	err := r.db.WithContext(ctx).First(&project, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_project_by_id", "REPO", map[string]interface{}{
			"operation": "get_project_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &project, nil
}

// UpdateProject 更新项目
func (r *ProjectRepository) UpdateProject(ctx context.Context, project *orcmodel.Project) error {
	if project == nil || project.ID == 0 {
		return errors.New("invalid project or id")
	}
	err := r.db.WithContext(ctx).Save(project).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_project", "REPO", map[string]interface{}{
			"operation": "update_project",
			"id":        project.ID,
		})
		return err
	}
	return nil
}

// DeleteProject 删除项目 (软删除)
func (r *ProjectRepository) DeleteProject(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&orcmodel.Project{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_project", "REPO", map[string]interface{}{
			"operation": "delete_project",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListProjects 获取项目列表 (分页 + 筛选 + 标签)
func (r *ProjectRepository) ListProjects(ctx context.Context, page, pageSize int, status string, name string, tagID uint64) ([]*orcmodel.Project, int64, error) {
	var projects []*orcmodel.Project
	var total int64

	query := r.db.WithContext(ctx).Model(&orcmodel.Project{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if tagID > 0 {
		// 联表查询 sys_entity_tags 表来筛选具有特定标签的项目
		// 注意：entity_type = 'project'
		query = query.Joins("JOIN sys_entity_tags ON projects.id = sys_entity_tags.entity_id").
			Where("sys_entity_tags.entity_type = ? AND sys_entity_tags.tag_id = ?", "project", tagID)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_projects_count", "REPO", map[string]interface{}{
			"operation": "list_projects_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	// 注意：如果使用了 Joins，可能会有重复记录（一个项目多个标签），这里因为限定了 tag_id，一般不会重复。
	// 但如果后续有多个 tag 筛选，可能需要 Distinct。
	// 这里加上 Distinct 以防万一。
	err = query.Distinct("projects.*").Offset(offset).Limit(pageSize).Order("projects.id desc").Find(&projects).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_projects_find", "REPO", map[string]interface{}{
			"operation": "list_projects_find",
		})
		return nil, 0, err
	}

	return projects, total, nil
}

// -----------------------------------------------------------------------------
// ProjectWorkflow (项目-工作流关联)
// -----------------------------------------------------------------------------

// AddWorkflowToProject 关联工作流到项目
func (r *ProjectRepository) AddWorkflowToProject(ctx context.Context, projectID, workflowID uint64, sortOrder int) error {
	association := &orcmodel.ProjectWorkflow{
		ProjectID:  projectID,
		WorkflowID: workflowID,
		SortOrder:  sortOrder,
	}
	err := r.db.WithContext(ctx).Create(association).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "add_workflow_to_project", "REPO", map[string]interface{}{
			"operation":   "add_workflow_to_project",
			"project_id":  projectID,
			"workflow_id": workflowID,
		})
		return err
	}
	return nil
}

// RemoveWorkflowFromProject 从项目中移除工作流
func (r *ProjectRepository) RemoveWorkflowFromProject(ctx context.Context, projectID, workflowID uint64) error {
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND workflow_id = ?", projectID, workflowID).
		Delete(&orcmodel.ProjectWorkflow{}).Error
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

// GetWorkflowsByProjectID 获取项目关联的所有工作流
func (r *ProjectRepository) GetWorkflowsByProjectID(ctx context.Context, projectID uint64) ([]*orcmodel.Workflow, error) {
	var workflows []*orcmodel.Workflow
	// 联表查询：通过 project_workflows 中间表连接 workflows 表
	err := r.db.WithContext(ctx).
		Table("workflows").
		Select("workflows.*").
		Joins("JOIN project_workflows ON workflows.id = project_workflows.workflow_id").
		Where("project_workflows.project_id = ?", projectID).
		Order("project_workflows.sort_order ASC").
		Scan(&workflows).Error

	if err != nil {
		logger.LogError(err, "", 0, "", "get_workflows_by_project_id", "REPO", map[string]interface{}{
			"operation":  "get_workflows_by_project_id",
			"project_id": projectID,
		})
		return nil, err
	}
	return workflows, nil
}
