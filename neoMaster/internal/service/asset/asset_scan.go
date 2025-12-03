package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
)

// AssetScanService 资产扫描服务
// 负责处理扫描记录的业务逻辑
type AssetScanService struct {
	scanRepo    *assetrepo.AssetScanRepository
	networkRepo *assetrepo.AssetNetworkRepository // 依赖 Network Repo 检查网段存在性
}

// NewAssetScanService 创建 AssetScanService 实例
func NewAssetScanService(scanRepo *assetrepo.AssetScanRepository, networkRepo *assetrepo.AssetNetworkRepository) *AssetScanService {
	return &AssetScanService{
		scanRepo:    scanRepo,
		networkRepo: networkRepo,
	}
}

// -----------------------------------------------------------------------------
// AssetNetworkScan 业务逻辑
// -----------------------------------------------------------------------------

// CreateScan 创建扫描记录
func (s *AssetScanService) CreateScan(ctx context.Context, scan *asset.AssetNetworkScan) error {
	if scan == nil {
		return errors.New("scan data cannot be nil")
	}
	// 检查网段是否存在
	if s.networkRepo != nil {
		network, err := s.networkRepo.GetNetworkByID(ctx, scan.NetworkID)
		if err != nil {
			return err
		}
		if network == nil {
			return errors.New("associated network not found")
		}
	}

	err := s.scanRepo.CreateScan(ctx, scan)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_scan", "SERVICE", map[string]interface{}{
			"operation":  "create_scan",
			"network_id": scan.NetworkID,
			"agent_id":   scan.AgentID,
		})
		return err
	}
	return nil
}

// GetScan 获取扫描记录详情
func (s *AssetScanService) GetScan(ctx context.Context, id uint64) (*asset.AssetNetworkScan, error) {
	scan, err := s.scanRepo.GetScanByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_scan", "SERVICE", map[string]interface{}{
			"operation": "get_scan",
			"id":        id,
		})
		return nil, err
	}
	if scan == nil {
		return nil, errors.New("scan record not found")
	}
	return scan, nil
}

// UpdateScan 更新扫描记录
func (s *AssetScanService) UpdateScan(ctx context.Context, scan *asset.AssetNetworkScan) error {
	// 检查是否存在
	existing, err := s.scanRepo.GetScanByID(ctx, scan.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("scan record not found")
	}

	err = s.scanRepo.UpdateScan(ctx, scan)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_scan", "SERVICE", map[string]interface{}{
			"operation": "update_scan",
			"id":        scan.ID,
		})
		return err
	}
	return nil
}

// DeleteScan 删除扫描记录
func (s *AssetScanService) DeleteScan(ctx context.Context, id uint64) error {
	// 检查是否存在
	existing, err := s.scanRepo.GetScanByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("scan record not found")
	}

	err = s.scanRepo.DeleteScan(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_scan", "SERVICE", map[string]interface{}{
			"operation": "delete_scan",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListScans 获取扫描记录列表
func (s *AssetScanService) ListScans(ctx context.Context, page, pageSize int, networkID, agentID uint64, status string) ([]*asset.AssetNetworkScan, int64, error) {
	list, total, err := s.scanRepo.ListScans(ctx, page, pageSize, networkID, agentID, status)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_scans", "SERVICE", map[string]interface{}{
			"operation":  "list_scans",
			"page":       page,
			"page_size":  pageSize,
			"network_id": networkID,
		})
		return nil, 0, err
	}
	return list, total, nil
}

// GetLatestScanByNetworkID 获取指定网段的最后一次扫描记录
func (s *AssetScanService) GetLatestScanByNetworkID(ctx context.Context, networkID uint64) (*asset.AssetNetworkScan, error) {
	scan, err := s.scanRepo.GetLatestScanByNetworkID(ctx, networkID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_latest_scan_by_network_id", "SERVICE", map[string]interface{}{
			"operation":  "get_latest_scan_by_network_id",
			"network_id": networkID,
		})
		return nil, err
	}
	// 这里返回 nil 是合法的，表示还没有扫描记录
	return scan, nil
}
