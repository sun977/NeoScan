package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
)

// ScanToolTemplateService 扫描工具模板服务
// 负责处理扫描工具模板的业务逻辑
type ScanToolTemplateService struct {
	repo *orcrepo.ScanToolTemplateRepository
}

// NewScanToolTemplateService 创建 ScanToolTemplateService 实例
func NewScanToolTemplateService(repo *orcrepo.ScanToolTemplateRepository) *ScanToolTemplateService {
	return &ScanToolTemplateService{repo: repo}
}

// CreateTemplate 创建模板
func (s *ScanToolTemplateService) CreateTemplate(ctx context.Context, tmpl *orcmodel.ScanToolTemplate) error {
	if tmpl == nil {
		return errors.New("template data cannot be nil")
	}
	err := s.repo.CreateTemplate(ctx, tmpl)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_template", "SERVICE", map[string]interface{}{
			"operation": "create_template",
			"name":      tmpl.Name,
		})
		return err
	}
	return nil
}

// GetTemplate 获取模板详情
func (s *ScanToolTemplateService) GetTemplate(ctx context.Context, id uint64) (*orcmodel.ScanToolTemplate, error) {
	tmpl, err := s.repo.GetTemplateByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_template", "SERVICE", map[string]interface{}{
			"operation": "get_template",
			"id":        id,
		})
		return nil, err
	}
	if tmpl == nil {
		return nil, errors.New("template not found")
	}
	return tmpl, nil
}

// UpdateTemplate 更新模板
func (s *ScanToolTemplateService) UpdateTemplate(ctx context.Context, tmpl *orcmodel.ScanToolTemplate) error {
	if tmpl == nil {
		return errors.New("template data cannot be nil")
	}
	existing, err := s.repo.GetTemplateByID(ctx, tmpl.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("template not found")
	}

	err = s.repo.UpdateTemplate(ctx, tmpl)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_template", "SERVICE", map[string]interface{}{
			"operation": "update_template",
			"id":        tmpl.ID,
		})
		return err
	}
	return nil
}

// DeleteTemplate 删除模板
func (s *ScanToolTemplateService) DeleteTemplate(ctx context.Context, id uint64) error {
	existing, err := s.repo.GetTemplateByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("template not found")
	}

	err = s.repo.DeleteTemplate(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_template", "SERVICE", map[string]interface{}{
			"operation": "delete_template",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListTemplates 获取模板列表
func (s *ScanToolTemplateService) ListTemplates(ctx context.Context, page, pageSize int, toolName string, category string, isPublic *bool) ([]*orcmodel.ScanToolTemplate, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	tmpls, total, err := s.repo.ListTemplates(ctx, page, pageSize, toolName, category, isPublic)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_templates", "SERVICE", map[string]interface{}{
			"operation": "list_templates",
		})
		return nil, 0, err
	}
	return tmpls, total, nil
}
