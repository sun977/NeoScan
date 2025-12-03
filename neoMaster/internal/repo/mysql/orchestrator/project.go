package orchestrator

import (
	"context"
	"errors"
	"gorm.io/gorm"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
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

// ListProjects 获取项目列表 (分页 + 筛选)
func (r *ProjectRepository) ListProjects(ctx context.Context, page, pageSize int, status string, name string) ([]*orcmodel.Project, int64, error) {
	var projects []*orcmodel.Project
	var total int64

	query := r.db.WithContext(ctx).Model(&orcmodel.Project{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_projects_count", "REPO", map[string]interface{}{
			"operation": "list_projects_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&projects).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_projects_find", "REPO", map[string]interface{}{
			"operation": "list_projects_find",
		})
		return nil, 0, err
	}

	return projects, total, nil
}
