package asset

import (
	"context"
	"errors"

	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetFingerRepository 接口定义
type AssetFingerRepository interface {
	// FindAll 获取所有 CMS 指纹规则
	FindAll(ctx context.Context) ([]*asset.AssetFinger, error)
	// Create 创建指纹规则
	Create(ctx context.Context, rule *asset.AssetFinger) error
	// GetByID 根据ID获取指纹规则
	GetByID(ctx context.Context, id uint64) (*asset.AssetFinger, error)
	// Update 更新指纹规则
	Update(ctx context.Context, rule *asset.AssetFinger) error
	// Delete 删除指纹规则
	Delete(ctx context.Context, id uint64) error
	// List 获取指纹规则列表(分页 + 简单筛选)
	List(ctx context.Context, page, pageSize int, name string, tagID uint64) ([]*asset.AssetFinger, int64, error)
}

// assetFingerRepository 实现
type assetFingerRepository struct {
	db *gorm.DB
}

// NewAssetFingerRepository 创建实例
func NewAssetFingerRepository(db *gorm.DB) AssetFingerRepository {
	return &assetFingerRepository{db: db}
}

// FindAll 获取所有 CMS 指纹规则
func (r *assetFingerRepository) FindAll(ctx context.Context) ([]*asset.AssetFinger, error) {
	var rules []*asset.AssetFinger
	if err := r.db.WithContext(ctx).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// Create 创建指纹规则
func (r *assetFingerRepository) Create(ctx context.Context, rule *asset.AssetFinger) error {
	if rule == nil {
		return errors.New("rule is nil")
	}
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		logger.LogError(err, "", 0, "", "create_asset_finger", "REPO", map[string]interface{}{
			"operation": "create_asset_finger",
			"name":      rule.Name,
		})
		return err
	}
	return nil
}

// GetByID 根据ID获取指纹规则
func (r *assetFingerRepository) GetByID(ctx context.Context, id uint64) (*asset.AssetFinger, error) {
	var rule asset.AssetFinger
	if err := r.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_asset_finger_by_id", "REPO", map[string]interface{}{
			"operation": "get_asset_finger_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &rule, nil
}

// Update 更新指纹规则
func (r *assetFingerRepository) Update(ctx context.Context, rule *asset.AssetFinger) error {
	if rule == nil || rule.ID == 0 {
		return errors.New("invalid rule or id")
	}
	if err := r.db.WithContext(ctx).Model(rule).Updates(rule).Error; err != nil {
		logger.LogError(err, "", 0, "", "update_asset_finger", "REPO", map[string]interface{}{
			"operation": "update_asset_finger",
			"id":        rule.ID,
		})
		return err
	}
	return nil
}

// Delete 删除指纹规则
func (r *assetFingerRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&asset.AssetFinger{}, id).Error; err != nil {
		logger.LogError(err, "", 0, "", "delete_asset_finger", "REPO", map[string]interface{}{
			"operation": "delete_asset_finger",
			"id":        id,
		})
		return err
	}
	return nil
}

// List 获取指纹规则列表(分页 + 简单筛选)
func (r *assetFingerRepository) List(ctx context.Context, page, pageSize int, name string, tagID uint64) ([]*asset.AssetFinger, int64, error) {
	var rules []*asset.AssetFinger
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetFinger{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if tagID > 0 {
		query = query.Joins("JOIN sys_entity_tags ON asset_finger.id = sys_entity_tags.entity_id").
			Where("sys_entity_tags.entity_type = ? AND sys_entity_tags.tag_id = ?", "fingers_cms", tagID)
	}

	countQuery := query
	if tagID > 0 {
		countQuery = countQuery.Distinct("asset_finger.id")
	}

	if err := countQuery.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "list_asset_finger_count", "REPO", map[string]interface{}{
			"operation": "list_asset_finger_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if tagID > 0 {
		query = query.Distinct("asset_finger.*")
	}
	if err := query.Offset(offset).Limit(pageSize).Order("asset_finger.id desc").Find(&rules).Error; err != nil {
		logger.LogError(err, "", 0, "", "list_asset_finger_find", "REPO", map[string]interface{}{
			"operation": "list_asset_finger_find",
		})
		return nil, 0, err
	}

	return rules, total, nil
}
