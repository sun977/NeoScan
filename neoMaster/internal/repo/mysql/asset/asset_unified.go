package asset

import (
	"context"
	"errors"
	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetUnifiedRepository 统一资产仓库
// 负责 AssetUnified (读模型/宽表) 的数据访问
type AssetUnifiedRepository struct {
	db *gorm.DB
}

// NewAssetUnifiedRepository 创建 AssetUnifiedRepository 实例
func NewAssetUnifiedRepository(db *gorm.DB) *AssetUnifiedRepository {
	return &AssetUnifiedRepository{db: db}
}

// -----------------------------------------------------------------------------
// AssetUnified (统一资产视图) CRUD
// -----------------------------------------------------------------------------

// CreateUnifiedAsset 创建统一资产记录 (通常由同步Worker调用)
func (r *AssetUnifiedRepository) CreateUnifiedAsset(ctx context.Context, asset *assetmodel.AssetUnified) error {
	if asset == nil {
		return errors.New("asset is nil")
	}
	err := r.db.WithContext(ctx).Create(asset).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_unified_asset", "REPO", map[string]interface{}{
			"operation": "create_unified_asset",
			"ip":        asset.IP,
			"port":      asset.Port,
		})
		return err
	}
	return nil
}

// GetUnifiedAssetByID 根据ID获取统一资产记录
func (r *AssetUnifiedRepository) GetUnifiedAssetByID(ctx context.Context, id uint64) (*assetmodel.AssetUnified, error) {
	var asset assetmodel.AssetUnified
	err := r.db.WithContext(ctx).First(&asset, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_unified_asset_by_id", "REPO", map[string]interface{}{
			"operation": "get_unified_asset_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &asset, nil
}

// UpdateUnifiedAsset 更新统一资产记录
func (r *AssetUnifiedRepository) UpdateUnifiedAsset(ctx context.Context, asset *assetmodel.AssetUnified) error {
	if asset == nil || asset.ID == 0 {
		return errors.New("invalid asset or id")
	}
	err := r.db.WithContext(ctx).Save(asset).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_unified_asset", "REPO", map[string]interface{}{
			"operation": "update_unified_asset",
			"id":        asset.ID,
		})
		return err
	}
	return nil
}

// DeleteUnifiedAsset 删除统一资产记录
func (r *AssetUnifiedRepository) DeleteUnifiedAsset(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&assetmodel.AssetUnified{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_unified_asset", "REPO", map[string]interface{}{
			"operation": "delete_unified_asset",
			"id":        id,
		})
		return err
	}
	return nil
}

// UnifiedAssetFilter 统一资产查询过滤器
type UnifiedAssetFilter struct {
	ProjectID uint64
	IP        string
	Port      int
	Service   string
	Product   string
	Component string
	IsWeb     *bool
	Keyword   string // 模糊搜索 (IP/Hostname/Title/Service)
}

// ListUnifiedAssets 获取统一资产列表 (高级搜索)
func (r *AssetUnifiedRepository) ListUnifiedAssets(ctx context.Context, page, pageSize int, filter UnifiedAssetFilter) ([]*assetmodel.AssetUnified, int64, error) {
	var assets []*assetmodel.AssetUnified
	var total int64

	query := r.db.WithContext(ctx).Model(&assetmodel.AssetUnified{})

	// 精确过滤
	if filter.ProjectID > 0 {
		query = query.Where("project_id = ?", filter.ProjectID)
	}
	if filter.Port > 0 {
		query = query.Where("port = ?", filter.Port)
	}
	if filter.IsWeb != nil {
		query = query.Where("is_web = ?", *filter.IsWeb)
	}

	// 模糊过滤
	if filter.IP != "" {
		query = query.Where("ip LIKE ?", "%"+filter.IP+"%")
	}
	if filter.Service != "" {
		query = query.Where("service LIKE ?", "%"+filter.Service+"%")
	}
	if filter.Product != "" {
		query = query.Where("product LIKE ?", "%"+filter.Product+"%")
	}
	if filter.Component != "" {
		query = query.Where("component LIKE ?", "%"+filter.Component+"%")
	}

	// 关键词综合搜索 (模拟搜索引擎体验)
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("ip LIKE ? OR host_name LIKE ? OR title LIKE ? OR service LIKE ? OR product LIKE ?",
			keyword, keyword, keyword, keyword, keyword)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_unified_assets_count", "REPO", map[string]interface{}{
			"operation": "list_unified_assets_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&assets).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_unified_assets_find", "REPO", map[string]interface{}{
			"operation": "list_unified_assets_find",
		})
		return nil, 0, err
	}

	return assets, total, nil
}

// UpsertUnifiedAsset 插入或更新 (基于 IP + Port + ProjectID)
// 用于同步Worker，如果存在则更新，不存在则插入
func (r *AssetUnifiedRepository) UpsertUnifiedAsset(ctx context.Context, asset *assetmodel.AssetUnified) error {
	// 注意：MySQL 8.0+ 支持 ON DUPLICATE KEY UPDATE
	// 这里假设 ip, port, project_id 建立了唯一索引或联合唯一索引
	// 如果没有唯一索引，这个 Upsert 可能会产生重复数据，请确保数据库层面有约束

	// 简单实现：先查后写 (对于高并发场景建议用原生 SQL 的 Upsert)
	var existing assetmodel.AssetUnified
	err := r.db.WithContext(ctx).Where("project_id = ? AND ip = ? AND port = ?", asset.ProjectID, asset.IP, asset.Port).First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不存在，创建
			return r.CreateUnifiedAsset(ctx, asset)
		}
		return err // 其他错误
	}

	// 存在，更新ID并保存
	asset.ID = existing.ID
	return r.UpdateUnifiedAsset(ctx, asset)
}
