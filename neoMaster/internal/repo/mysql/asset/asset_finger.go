package asset

import (
	"context"

	"neomaster/internal/model/asset"

	"gorm.io/gorm"
)

// AssetFingerRepository 接口定义
type AssetFingerRepository interface {
	// FindAll 获取所有 CMS 指纹规则
	FindAll(ctx context.Context) ([]*asset.AssetFinger, error)
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
