package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// ScanToolTemplateRepository 扫描工具模板仓库
// 负责 ScanToolTemplate 的数据访问
type ScanToolTemplateRepository struct {
	db *gorm.DB
}

// NewScanToolTemplateRepository 创建 ScanToolTemplateRepository 实例
func NewScanToolTemplateRepository(db *gorm.DB) *ScanToolTemplateRepository {
	return &ScanToolTemplateRepository{db: db}
}

// -----------------------------------------------------------------------------
// ScanToolTemplate (工具模板) CRUD
// -----------------------------------------------------------------------------

// CreateTemplate 创建模板
func (r *ScanToolTemplateRepository) CreateTemplate(ctx context.Context, tmpl *orcmodel.ScanToolTemplate) error {
	if tmpl == nil {
		return errors.New("template is nil")
	}
	err := r.db.WithContext(ctx).Create(tmpl).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_template", "REPO", map[string]interface{}{
			"operation": "create_template",
			"name":      tmpl.Name,
			"tool":      tmpl.ToolName,
		})
		return err
	}
	return nil
}

// GetTemplateByID 根据ID获取模板
func (r *ScanToolTemplateRepository) GetTemplateByID(ctx context.Context, id uint64) (*orcmodel.ScanToolTemplate, error) {
	var tmpl orcmodel.ScanToolTemplate
	err := r.db.WithContext(ctx).First(&tmpl, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_template_by_id", "REPO", map[string]interface{}{
			"operation": "get_template_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &tmpl, nil
}

// UpdateTemplate 更新模板
func (r *ScanToolTemplateRepository) UpdateTemplate(ctx context.Context, tmpl *orcmodel.ScanToolTemplate) error {
	if tmpl == nil || tmpl.ID == 0 {
		return errors.New("invalid template or id")
	}
	err := r.db.WithContext(ctx).Save(tmpl).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_template", "REPO", map[string]interface{}{
			"operation": "update_template",
			"id":        tmpl.ID,
		})
		return err
	}
	return nil
}

// DeleteTemplate 删除模板
func (r *ScanToolTemplateRepository) DeleteTemplate(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&orcmodel.ScanToolTemplate{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_template", "REPO", map[string]interface{}{
			"operation": "delete_template",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListTemplates 获取模板列表 (支持按工具名和分类筛选)
func (r *ScanToolTemplateRepository) ListTemplates(ctx context.Context, page, pageSize int, toolName string, category string, isPublic *bool) ([]*orcmodel.ScanToolTemplate, int64, error) {
	var tmpls []*orcmodel.ScanToolTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&orcmodel.ScanToolTemplate{})

	if toolName != "" {
		query = query.Where("tool_name = ?", toolName)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if isPublic != nil {
		query = query.Where("is_public = ?", *isPublic)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_templates_count", "REPO", map[string]interface{}{
			"operation": "list_templates_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&tmpls).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_templates_find", "REPO", map[string]interface{}{
			"operation": "list_templates_find",
		})
		return nil, 0, err
	}

	return tmpls, total, nil
}
