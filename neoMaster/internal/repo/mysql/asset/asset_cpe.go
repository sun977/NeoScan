package asset

import (
	"context"

	"neomaster/internal/model/asset"

	"gorm.io/gorm"
)

// AssetCPERepository 接口定义
type AssetCPERepository interface {
	// FindAll 获取所有 CPE 指纹规则
	FindAll(ctx context.Context) ([]*asset.AssetCPE, error)
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
