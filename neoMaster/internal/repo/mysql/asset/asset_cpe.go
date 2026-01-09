package asset

import (
	"context"
	"errors"

	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetCPERepository 接口定义
type AssetCPERepository interface {
	// FindAll 获取所有 CPE 指纹规则
	FindAll(ctx context.Context) ([]*asset.AssetCPE, error)
	// Create 创建 CPE 指纹规则
	Create(ctx context.Context, rule *asset.AssetCPE) error
	// GetByID 根据ID获取 CPE 指纹规则
	GetByID(ctx context.Context, id uint64) (*asset.AssetCPE, error)
	// Update 更新 CPE 指纹规则
	Update(ctx context.Context, rule *asset.AssetCPE) error
	// Delete 删除 CPE 指纹规则
	Delete(ctx context.Context, id uint64) error
	// List 获取 CPE 指纹规则列表(分页 + 简单筛选)
	List(ctx context.Context, page, pageSize int, name, vendor, product string) ([]*asset.AssetCPE, int64, error)
}

// assetCPERepository 实现
type assetCPERepository struct {
	db *gorm.DB
}

// NewAssetCPERepository 创建实例
func NewAssetCPERepository(db *gorm.DB) AssetCPERepository {
	return &assetCPERepository{db: db}
}

// FindAll 获取所有 CPE 指纹规则
func (r *assetCPERepository) FindAll(ctx context.Context) ([]*asset.AssetCPE, error) {
	var rules []*asset.AssetCPE
	if err := r.db.WithContext(ctx).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *assetCPERepository) Create(ctx context.Context, rule *asset.AssetCPE) error {
	if rule == nil {
		return errors.New("rule is nil")
	}
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		logger.LogError(err, "", 0, "", "create_asset_cpe", "REPO", map[string]interface{}{
			"operation": "create_asset_cpe",
			"name":      rule.Name,
		})
		return err
	}
	return nil
}

func (r *assetCPERepository) GetByID(ctx context.Context, id uint64) (*asset.AssetCPE, error) {
	var rule asset.AssetCPE
	if err := r.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_asset_cpe_by_id", "REPO", map[string]interface{}{
			"operation": "get_asset_cpe_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &rule, nil
}

func (r *assetCPERepository) Update(ctx context.Context, rule *asset.AssetCPE) error {
	if rule == nil || rule.ID == 0 {
		return errors.New("invalid rule or id")
	}
	if err := r.db.WithContext(ctx).Model(rule).Updates(rule).Error; err != nil {
		logger.LogError(err, "", 0, "", "update_asset_cpe", "REPO", map[string]interface{}{
			"operation": "update_asset_cpe",
			"id":        rule.ID,
		})
		return err
	}
	return nil
}

func (r *assetCPERepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&asset.AssetCPE{}, id).Error; err != nil {
		logger.LogError(err, "", 0, "", "delete_asset_cpe", "REPO", map[string]interface{}{
			"operation": "delete_asset_cpe",
			"id":        id,
		})
		return err
	}
	return nil
}

func (r *assetCPERepository) List(ctx context.Context, page, pageSize int, name, vendor, product string) ([]*asset.AssetCPE, int64, error) {
	var rules []*asset.AssetCPE
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetCPE{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if vendor != "" {
		query = query.Where("vendor LIKE ?", "%"+vendor+"%")
	}
	if product != "" {
		query = query.Where("product LIKE ?", "%"+product+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "list_asset_cpe_count", "REPO", map[string]interface{}{
			"operation": "list_asset_cpe_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id desc").Find(&rules).Error; err != nil {
		logger.LogError(err, "", 0, "", "list_asset_cpe_find", "REPO", map[string]interface{}{
			"operation": "list_asset_cpe_find",
		})
		return nil, 0, err
	}

	return rules, total, nil
}
