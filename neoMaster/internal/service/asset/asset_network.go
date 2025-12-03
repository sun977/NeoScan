package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
)

// AssetNetworkService 资产网段服务
// 负责处理网段资产的业务逻辑
type AssetNetworkService struct {
	repo *assetrepo.AssetNetworkRepository
}

// NewAssetNetworkService 创建 AssetNetworkService 实例
func NewAssetNetworkService(repo *assetrepo.AssetNetworkRepository) *AssetNetworkService {
	return &AssetNetworkService{repo: repo}
}

// -----------------------------------------------------------------------------
// AssetNetwork 业务逻辑
// -----------------------------------------------------------------------------

// CreateNetwork 创建网段
func (s *AssetNetworkService) CreateNetwork(ctx context.Context, network *asset.AssetNetwork) error {
	if network == nil {
		return errors.New("network data cannot be nil")
	}
	// 检查CIDR是否已存在
	existing, err := s.repo.GetNetworkByCIDR(ctx, network.CIDR)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("network with this CIDR already exists")
	}

	err = s.repo.CreateNetwork(ctx, network)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_network", "SERVICE", map[string]interface{}{
			"operation": "create_network",
			"cidr":      network.CIDR,
		})
		return err
	}
	return nil
}

// GetNetwork 获取网段详情
func (s *AssetNetworkService) GetNetwork(ctx context.Context, id uint64) (*asset.AssetNetwork, error) {
	network, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_network", "SERVICE", map[string]interface{}{
			"operation": "get_network",
			"id":        id,
		})
		return nil, err
	}
	if network == nil {
		return nil, errors.New("network not found")
	}
	return network, nil
}

// UpdateNetwork 更新网段
func (s *AssetNetworkService) UpdateNetwork(ctx context.Context, network *asset.AssetNetwork) error {
	// 检查是否存在
	existing, err := s.repo.GetNetworkByID(ctx, network.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("network not found")
	}

	err = s.repo.UpdateNetwork(ctx, network)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_network", "SERVICE", map[string]interface{}{
			"operation": "update_network",
			"id":        network.ID,
		})
		return err
	}
	return nil
}

// DeleteNetwork 删除网段
func (s *AssetNetworkService) DeleteNetwork(ctx context.Context, id uint64) error {
	// 检查是否存在
	existing, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("network not found")
	}

	err = s.repo.DeleteNetwork(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_network", "SERVICE", map[string]interface{}{
			"operation": "delete_network",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListNetworks 获取网段列表
func (s *AssetNetworkService) ListNetworks(ctx context.Context, page, pageSize int, cidr, networkType, status string) ([]*asset.AssetNetwork, int64, error) {
	list, total, err := s.repo.ListNetworks(ctx, page, pageSize, cidr, networkType, status)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_networks", "SERVICE", map[string]interface{}{
			"operation": "list_networks",
			"page":      page,
			"page_size": pageSize,
		})
		return nil, 0, err
	}
	return list, total, nil
}

// UpdateScanStatus 更新扫描状态
func (s *AssetNetworkService) UpdateScanStatus(ctx context.Context, id uint64, status string) error {
	// 检查是否存在
	existing, err := s.repo.GetNetworkByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("network not found")
	}

	// TODO: 这里可以添加状态流转的校验逻辑，例如不能从 finished 直接变回 scanning (取决于业务)

	err = s.repo.UpdateScanStatus(ctx, id, status)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_network_scan_status", "SERVICE", map[string]interface{}{
			"operation": "update_network_scan_status",
			"id":        id,
			"status":    status,
		})
		return err
	}
	return nil
}
