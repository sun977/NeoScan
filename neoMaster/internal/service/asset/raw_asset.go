package asset

import (
	"context"
	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
)

// RawAssetService 原始资产服务层
// 处理原始数据导入和清洗的业务逻辑
type RawAssetService struct {
	repo *assetrepo.RawAssetRepository
}

// NewRawAssetService 创建 RawAssetService 实例
func NewRawAssetService(repo *assetrepo.RawAssetRepository) *RawAssetService {
	return &RawAssetService{repo: repo}
}

// -----------------------------------------------------------------------------
// RawAsset 业务逻辑
// -----------------------------------------------------------------------------

// CreateRawAsset 创建原始资产
func (s *RawAssetService) CreateRawAsset(ctx context.Context, raw *assetmodel.RawAsset) error {
	err := s.repo.CreateRawAsset(ctx, raw)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.raw.CreateRawAsset", "SERVICE", map[string]interface{}{
			"source": raw.SourceType,
		})
		return err
	}
	return nil
}

// GetRawAssetByID 根据ID获取原始资产
func (s *RawAssetService) GetRawAssetByID(ctx context.Context, id uint64) (*assetmodel.RawAsset, error) {
	return s.repo.GetRawAssetByID(ctx, id)
}

// UpdateRawAssetStatus 更新原始资产状态
func (s *RawAssetService) UpdateRawAssetStatus(ctx context.Context, id uint64, status string, errMsg string) error {
	err := s.repo.UpdateRawAssetStatus(ctx, id, status, errMsg)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.raw.UpdateRawAssetStatus", "SERVICE", map[string]interface{}{
			"id":     id,
			"status": status,
		})
		return err
	}
	return nil
}

// ListRawAssets 获取原始资产列表
func (s *RawAssetService) ListRawAssets(ctx context.Context, page, pageSize int, batchID string, status string) ([]*assetmodel.RawAsset, int64, error) {
	return s.repo.ListRawAssets(ctx, page, pageSize, batchID, status)
}

// -----------------------------------------------------------------------------
// RawAssetNetwork 业务逻辑
// -----------------------------------------------------------------------------

// CreateRawNetwork 创建待确认网段
func (s *RawAssetService) CreateRawNetwork(ctx context.Context, network *assetmodel.RawAssetNetwork) error {
	err := s.repo.CreateRawNetwork(ctx, network)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.raw.CreateRawNetwork", "SERVICE", map[string]interface{}{
			"network": network.Network,
		})
		return err
	}
	return nil
}

// GetRawNetworkByID 根据ID获取待确认网段
func (s *RawAssetService) GetRawNetworkByID(ctx context.Context, id uint64) (*assetmodel.RawAssetNetwork, error) {
	return s.repo.GetRawNetworkByID(ctx, id)
}

// ApproveRawNetwork 批准待确认网段 (状态流转)
func (s *RawAssetService) ApproveRawNetwork(ctx context.Context, id uint64) error {
	err := s.repo.UpdateRawNetworkStatus(ctx, id, "approved")
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.raw.ApproveRawNetwork", "SERVICE", map[string]interface{}{
			"id": id,
		})
		return err
	}
	return nil
}

// RejectRawNetwork 拒绝待确认网段
func (s *RawAssetService) RejectRawNetwork(ctx context.Context, id uint64) error {
	err := s.repo.UpdateRawNetworkStatus(ctx, id, "rejected")
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.raw.RejectRawNetwork", "SERVICE", map[string]interface{}{
			"id": id,
		})
		return err
	}
	return nil
}

// ListRawNetworks 获取待确认网段列表
func (s *RawAssetService) ListRawNetworks(ctx context.Context, page, pageSize int, status string, sourceType string) ([]*assetmodel.RawAssetNetwork, int64, error) {
	return s.repo.ListRawNetworks(ctx, page, pageSize, status, sourceType)
}
