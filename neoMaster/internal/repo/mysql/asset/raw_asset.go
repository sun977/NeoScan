package asset

import (
	"context"
	"errors"
	"gorm.io/gorm"
	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
)

// RawAssetRepository 原始资产仓库
// 负责 RawAsset 和 RawAssetNetwork 的数据访问
type RawAssetRepository struct {
	db *gorm.DB
}

// NewRawAssetRepository 创建 RawAssetRepository 实例
func NewRawAssetRepository(db *gorm.DB) *RawAssetRepository {
	return &RawAssetRepository{db: db}
}

// -----------------------------------------------------------------------------
// RawAsset (原始导入记录) CRUD
// -----------------------------------------------------------------------------

// CreateRawAsset 创建原始资产记录
func (r *RawAssetRepository) CreateRawAsset(ctx context.Context, raw *assetmodel.RawAsset) error {
	if raw == nil {
		return errors.New("raw asset is nil")
	}
	err := r.db.WithContext(ctx).Create(raw).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_raw_asset", "REPO", map[string]interface{}{
			"operation": "create_raw_asset",
			"source":    raw.SourceType,
			"batch_id":  raw.ImportBatchID,
		})
		return err
	}
	return nil
}

// GetRawAssetByID 根据ID获取原始资产记录
func (r *RawAssetRepository) GetRawAssetByID(ctx context.Context, id uint64) (*assetmodel.RawAsset, error) {
	var raw assetmodel.RawAsset
	err := r.db.WithContext(ctx).First(&raw, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_raw_asset_by_id", "REPO", map[string]interface{}{
			"operation": "get_raw_asset_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &raw, nil
}

// UpdateRawAssetStatus 更新原始资产状态 (常用于Worker处理后)
func (r *RawAssetRepository) UpdateRawAssetStatus(ctx context.Context, id uint64, status string, errMsg string) error {
	updates := map[string]interface{}{
		"normalize_status": status,
		"normalize_error":  errMsg,
	}
	err := r.db.WithContext(ctx).Model(&assetmodel.RawAsset{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_raw_asset_status", "REPO", map[string]interface{}{
			"operation": "update_raw_asset_status",
			"id":        id,
			"status":    status,
		})
		return err
	}
	return nil
}

// ListRawAssets 获取原始资产列表 (支持按批次和状态筛选)
func (r *RawAssetRepository) ListRawAssets(ctx context.Context, page, pageSize int, batchID string, status string) ([]*assetmodel.RawAsset, int64, error) {
	var raws []*assetmodel.RawAsset
	var total int64

	query := r.db.WithContext(ctx).Model(&assetmodel.RawAsset{})

	if batchID != "" {
		query = query.Where("import_batch_id = ?", batchID)
	}
	if status != "" {
		query = query.Where("normalize_status = ?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_raw_assets_count", "REPO", map[string]interface{}{
			"operation": "list_raw_assets_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&raws).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_raw_assets_find", "REPO", map[string]interface{}{
			"operation": "list_raw_assets_find",
		})
		return nil, 0, err
	}

	return raws, total, nil
}

// -----------------------------------------------------------------------------
// RawAssetNetwork (待确认网段) CRUD
// -----------------------------------------------------------------------------

// CreateRawNetwork 创建待确认网段
func (r *RawAssetRepository) CreateRawNetwork(ctx context.Context, network *assetmodel.RawAssetNetwork) error {
	if network == nil {
		return errors.New("raw network is nil")
	}
	err := r.db.WithContext(ctx).Create(network).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_raw_network", "REPO", map[string]interface{}{
			"operation": "create_raw_network",
			"network":   network.Network,
		})
		return err
	}
	return nil
}

// GetRawNetworkByID 根据ID获取待确认网段
func (r *RawAssetRepository) GetRawNetworkByID(ctx context.Context, id uint64) (*assetmodel.RawAssetNetwork, error) {
	var network assetmodel.RawAssetNetwork
	err := r.db.WithContext(ctx).First(&network, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_raw_network_by_id", "REPO", map[string]interface{}{
			"operation": "get_raw_network_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &network, nil
}

// UpdateRawNetworkStatus 更新待确认网段状态 (Approved/Rejected)
func (r *RawAssetRepository) UpdateRawNetworkStatus(ctx context.Context, id uint64, status string) error {
	err := r.db.WithContext(ctx).Model(&assetmodel.RawAssetNetwork{}).Where("id = ?", id).Update("status", status).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_raw_network_status", "REPO", map[string]interface{}{
			"operation": "update_raw_network_status",
			"id":        id,
			"status":    status,
		})
		return err
	}
	return nil
}

// ListRawNetworks 获取待确认网段列表
func (r *RawAssetRepository) ListRawNetworks(ctx context.Context, page, pageSize int, status string, sourceType string) ([]*assetmodel.RawAssetNetwork, int64, error) {
	var networks []*assetmodel.RawAssetNetwork
	var total int64

	query := r.db.WithContext(ctx).Model(&assetmodel.RawAssetNetwork{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_raw_networks_count", "REPO", map[string]interface{}{
			"operation": "list_raw_networks_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&networks).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_raw_networks_find", "REPO", map[string]interface{}{
			"operation": "list_raw_networks_find",
		})
		return nil, 0, err
	}

	return networks, total, nil
}
