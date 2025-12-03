package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetScanRepository 资产扫描仓库
// 负责 AssetNetworkScan 的数据访问
type AssetScanRepository struct {
	db *gorm.DB
}

// NewAssetScanRepository 创建 AssetScanRepository 实例
func NewAssetScanRepository(db *gorm.DB) *AssetScanRepository {
	return &AssetScanRepository{db: db}
}

// -----------------------------------------------------------------------------
// AssetNetworkScan (网段扫描记录) CRUD
// -----------------------------------------------------------------------------

// CreateScan 创建扫描记录
func (r *AssetScanRepository) CreateScan(ctx context.Context, scan *asset.AssetNetworkScan) error {
	if scan == nil {
		return errors.New("scan is nil")
	}
	err := r.db.WithContext(ctx).Create(scan).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_scan", "REPO", map[string]interface{}{
			"operation":  "create_scan",
			"network_id": scan.NetworkID,
			"agent_id":   scan.AgentID,
		})
		return err
	}
	return nil
}

// GetScanByID 根据ID获取扫描记录
func (r *AssetScanRepository) GetScanByID(ctx context.Context, id uint64) (*asset.AssetNetworkScan, error) {
	var scan asset.AssetNetworkScan
	err := r.db.WithContext(ctx).First(&scan, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_scan_by_id", "REPO", map[string]interface{}{
			"operation": "get_scan_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &scan, nil
}

// UpdateScan 更新扫描记录
func (r *AssetScanRepository) UpdateScan(ctx context.Context, scan *asset.AssetNetworkScan) error {
	if scan == nil || scan.ID == 0 {
		return errors.New("invalid scan or id")
	}
	err := r.db.WithContext(ctx).Save(scan).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_scan", "REPO", map[string]interface{}{
			"operation": "update_scan",
			"id":        scan.ID,
		})
		return err
	}
	return nil
}

// DeleteScan 删除扫描记录 (软删除)
func (r *AssetScanRepository) DeleteScan(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&asset.AssetNetworkScan{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_scan", "REPO", map[string]interface{}{
			"operation": "delete_scan",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListScans 获取扫描记录列表 (分页 + 筛选)
func (r *AssetScanRepository) ListScans(ctx context.Context, page, pageSize int, networkID, agentID uint64, status string) ([]*asset.AssetNetworkScan, int64, error) {
	var scans []*asset.AssetNetworkScan
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetNetworkScan{})

	if networkID > 0 {
		query = query.Where("network_id = ?", networkID)
	}
	if agentID > 0 {
		query = query.Where("agent_id = ?", agentID)
	}
	if status != "" {
		query = query.Where("scan_status = ?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_scans_count", "REPO", map[string]interface{}{
			"operation": "list_scans_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&scans).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_scans_find", "REPO", map[string]interface{}{
			"operation": "list_scans_find",
		})
		return nil, 0, err
	}

	return scans, total, nil
}

// GetLatestScanByNetworkID 获取指定网段的最后一次扫描记录
func (r *AssetScanRepository) GetLatestScanByNetworkID(ctx context.Context, networkID uint64) (*asset.AssetNetworkScan, error) {
	var scan asset.AssetNetworkScan
	err := r.db.WithContext(ctx).Where("network_id = ?", networkID).Order("id desc").First(&scan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_latest_scan_by_network_id", "REPO", map[string]interface{}{
			"operation":  "get_latest_scan_by_network_id",
			"network_id": networkID,
		})
		return nil, err
	}
	return &scan, nil
}
