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

// ProjectService 项目服务
// 负责处理项目的业务逻辑
type ProjectService struct {
	repo       *orcrepo.ProjectRepository
	tagService tag_system.TagService
}

// NewProjectService 创建 ProjectService 实例
func NewProjectService(repo *orcrepo.ProjectRepository, tagService tag_system.TagService) *ProjectService {
	return &ProjectService{
		repo:       repo,
		tagService: tagService,
	}
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
func (s *ProjectService) ListProjects(ctx context.Context, page, pageSize int, status string, name string, tagID uint64) ([]*orcmodel.Project, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	projects, total, err := s.repo.ListProjects(ctx, page, pageSize, status, name, tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_projects", "SERVICE", map[string]interface{}{
			"operation": "list_projects",
		})
		return nil, 0, err
	}
	return projects, total, nil
}

// AddWorkflowToProject 关联工作流到项目
func (s *ProjectService) AddWorkflowToProject(ctx context.Context, projectID, workflowID uint64, sortOrder int) error {
	// 检查项目是否存在
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project == nil {
		return errors.New("project not found")
	}

	// 这里也可以检查 workflow 是否存在，但由于没有注入 WorkflowRepo，
	// 且数据库层可能有外键约束，或者单纯信任输入（由前端保证有效性，或数据库报错），
	// 暂时直接调用 Repo。若数据库无外键，可能会插入无效 workflow_id，需注意。
	// 更好的做法是在 Service 初始化时注入 WorkflowRepo，或者在 ProjectRepo 中提供 CheckWorkflowExist。
	// 鉴于当前架构，我们先直接尝试添加。

	err = s.repo.AddWorkflowToProject(ctx, projectID, workflowID, sortOrder)
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

// RemoveWorkflowFromProject 从项目中移除工作流
func (s *ProjectService) RemoveWorkflowFromProject(ctx context.Context, projectID, workflowID uint64) error {
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

// GetProjectWorkflows 获取项目关联的所有工作流
func (s *ProjectService) GetProjectWorkflows(ctx context.Context, projectID uint64) ([]*orcmodel.Workflow, error) {
	workflows, err := s.repo.GetWorkflowsByProjectID(ctx, projectID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_project_workflows", "SERVICE", map[string]interface{}{
			"operation":  "get_project_workflows",
			"project_id": projectID,
		})
		return nil, err
	}
	return workflows, nil
}

// AddTagToProject 为项目添加标签
func (s *ProjectService) AddTagToProject(ctx context.Context, projectID uint64, tagID uint64) error {
	// 1. 检查项目是否存在
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project == nil {
		return errors.New("project not found")
	}

	// 2. 调用 TagService 添加标签
	// entityType="project", source="manual" (假设手动添加), ruleID=0 (非规则自动添加)
	entityIDStr := strconv.FormatUint(projectID, 10)
	err = s.tagService.AddEntityTag(ctx, "project", entityIDStr, tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_project", "SERVICE", map[string]interface{}{
			"operation":  "add_tag_to_project",
			"project_id": projectID,
			"tag_id":     tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromProject 从项目移除标签
func (s *ProjectService) RemoveTagFromProject(ctx context.Context, projectID uint64, tagID uint64) error {
	// 1. 检查项目是否存在
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project == nil {
		return errors.New("project not found")
	}

	// 2. 调用 TagService 移除标签
	entityIDStr := strconv.FormatUint(projectID, 10)
	err = s.tagService.RemoveEntityTag(ctx, "project", entityIDStr, tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_project", "SERVICE", map[string]interface{}{
			"operation":  "remove_tag_from_project",
			"project_id": projectID,
			"tag_id":     tagID,
		})
		return err
	}
	return nil
}

// GetProjectTags 获取项目的所有标签
func (s *ProjectService) GetProjectTags(ctx context.Context, projectID uint64) ([]tagmodel.SysEntityTag, error) {
	// 1. 检查项目是否存在
	project, err := s.repo.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, errors.New("project not found")
	}

	// 2. 调用 TagService 获取标签
	entityIDStr := strconv.FormatUint(projectID, 10)
	tags, err := s.tagService.GetEntityTags(ctx, "project", entityIDStr)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_project_tags", "SERVICE", map[string]interface{}{
			"operation":  "get_project_tags",
			"project_id": projectID,
		})
		return nil, err
	}
	return tags, nil
}
