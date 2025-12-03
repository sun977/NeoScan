package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
)

// ProjectService 项目服务
// 负责处理项目的业务逻辑
type ProjectService struct {
	repo *orcrepo.ProjectRepository
}

// NewProjectService 创建 ProjectService 实例
func NewProjectService(repo *orcrepo.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

// CreateProject 创建项目
func (s *ProjectService) CreateProject(ctx context.Context, project *orcmodel.Project) error {
	if project == nil {
		return errors.New("project data cannot be nil")
	}

	err := s.repo.CreateProject(ctx, project)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_project", "SERVICE", map[string]interface{}{
			"operation": "create_project",
			"name":      project.Name,
		})
		return err
	}
	return nil
}

// GetProject 获取项目详情
func (s *ProjectService) GetProject(ctx context.Context, id uint64) (*orcmodel.Project, error) {
	project, err := s.repo.GetProjectByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_project", "SERVICE", map[string]interface{}{
			"operation": "get_project",
			"id":        id,
		})
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}
	return project, nil
}

// UpdateProject 更新项目
func (s *ProjectService) UpdateProject(ctx context.Context, project *orcmodel.Project) error {
	if project == nil {
		return errors.New("project data cannot be nil")
	}
	// 检查是否存在
	existing, err := s.repo.GetProjectByID(ctx, project.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("project not found")
	}

	err = s.repo.UpdateProject(ctx, project)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_project", "SERVICE", map[string]interface{}{
			"operation": "update_project",
			"id":        project.ID,
		})
		return err
	}
	return nil
}

// DeleteProject 删除项目
func (s *ProjectService) DeleteProject(ctx context.Context, id uint64) error {
	// 检查是否存在
	existing, err := s.repo.GetProjectByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("project not found")
	}

	err = s.repo.DeleteProject(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_project", "SERVICE", map[string]interface{}{
			"operation": "delete_project",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListProjects 获取项目列表
func (s *ProjectService) ListProjects(ctx context.Context, page, pageSize int, status string, name string) ([]*orcmodel.Project, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	projects, total, err := s.repo.ListProjects(ctx, page, pageSize, status, name)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_projects", "SERVICE", map[string]interface{}{
			"operation": "list_projects",
		})
		return nil, 0, err
	}
	return projects, total, nil
}
