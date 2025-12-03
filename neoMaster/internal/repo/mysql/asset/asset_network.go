package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetNetworkRepository 资产网段仓库
// 负责 AssetNetwork 的数据访问
type AssetNetworkRepository struct {
	db *gorm.DB
}

// NewAssetNetworkRepository 创建 AssetNetworkRepository 实例
func NewAssetNetworkRepository(db *gorm.DB) *AssetNetworkRepository {
	return &AssetNetworkRepository{db: db}
}

// -----------------------------------------------------------------------------
// AssetNetwork (网段资产) CRUD
// -----------------------------------------------------------------------------

// CreateNetwork 创建网段
func (r *AssetNetworkRepository) CreateNetwork(ctx context.Context, network *asset.AssetNetwork) error {
	if network == nil {
		return errors.New("network is nil")
	}
	err := r.db.WithContext(ctx).Create(network).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_network", "REPO", map[string]interface{}{
			"operation": "create_network",
			"cidr":      network.CIDR,
		})
		return err
	}
	return nil
}

// GetNetworkByID 根据ID获取网段
func (r *AssetNetworkRepository) GetNetworkByID(ctx context.Context, id uint64) (*asset.AssetNetwork, error) {
	var network asset.AssetNetwork
	err := r.db.WithContext(ctx).First(&network, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_network_by_id", "REPO", map[string]interface{}{
			"operation": "get_network_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &network, nil
}

// GetNetworkByCIDR 根据CIDR获取网段
func (r *AssetNetworkRepository) GetNetworkByCIDR(ctx context.Context, cidr string) (*asset.AssetNetwork, error) {
	var network asset.AssetNetwork
	err := r.db.WithContext(ctx).Where("cidr = ?", cidr).First(&network).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_network_by_cidr", "REPO", map[string]interface{}{
			"operation": "get_network_by_cidr",
			"cidr":      cidr,
		})
		return nil, err
	}
	return &network, nil
}

// UpdateNetwork 更新网段
func (r *AssetNetworkRepository) UpdateNetwork(ctx context.Context, network *asset.AssetNetwork) error {
	if network == nil || network.ID == 0 {
		return errors.New("invalid network or id")
	}
	err := r.db.WithContext(ctx).Save(network).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_network", "REPO", map[string]interface{}{
			"operation": "update_network",
			"id":        network.ID,
		})
		return err
	}
	return nil
}

// DeleteNetwork 删除网段 (软删除)
func (r *AssetNetworkRepository) DeleteNetwork(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&asset.AssetNetwork{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_network", "REPO", map[string]interface{}{
			"operation": "delete_network",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListNetworks 获取网段列表 (分页 + 筛选)
func (r *AssetNetworkRepository) ListNetworks(ctx context.Context, page, pageSize int, cidr, networkType, status string) ([]*asset.AssetNetwork, int64, error) {
	var networks []*asset.AssetNetwork
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetNetwork{})

	if cidr != "" {
		query = query.Where("cidr LIKE ?", "%"+cidr+"%")
	}
	if networkType != "" {
		query = query.Where("network_type = ?", networkType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_networks_count", "REPO", map[string]interface{}{
			"operation": "list_networks_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("priority desc, id desc").Find(&networks).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_networks_find", "REPO", map[string]interface{}{
			"operation": "list_networks_find",
		})
		return nil, 0, err
	}

	return networks, total, nil
}

// UpdateScanStatus 更新扫描状态
func (r *AssetNetworkRepository) UpdateScanStatus(ctx context.Context, id uint64, status string) error {
	err := r.db.WithContext(ctx).Model(&asset.AssetNetwork{}).Where("id = ?", id).Update("scan_status", status).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_network_scan_status", "REPO", map[string]interface{}{
			"operation": "update_network_scan_status",
			"id":        id,
			"status":    status,
		})
		return err
	}
	return nil
}
