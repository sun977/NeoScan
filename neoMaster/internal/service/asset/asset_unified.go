package asset

import (
	"context"
	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
)

// AssetUnifiedService 统一资产服务层
// 处理资产视图的业务逻辑
type AssetUnifiedService struct {
	repo *assetrepo.AssetUnifiedRepository
}

// NewAssetUnifiedService 创建 AssetUnifiedService 实例
func NewAssetUnifiedService(repo *assetrepo.AssetUnifiedRepository) *AssetUnifiedService {
	return &AssetUnifiedService{repo: repo}
}

// CreateUnifiedAsset 创建统一资产
func (s *AssetUnifiedService) CreateUnifiedAsset(ctx context.Context, asset *assetmodel.AssetUnified) error {
	err := s.repo.CreateUnifiedAsset(ctx, asset)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.unified.CreateUnifiedAsset", "SERVICE", map[string]interface{}{
			"ip":   asset.IP,
			"port": asset.Port,
		})
		return err
	}
	return nil
}

// GetUnifiedAssetByID 根据ID获取统一资产
func (s *AssetUnifiedService) GetUnifiedAssetByID(ctx context.Context, id uint64) (*assetmodel.AssetUnified, error) {
	return s.repo.GetUnifiedAssetByID(ctx, id)
}

// UpdateUnifiedAsset 更新统一资产
func (s *AssetUnifiedService) UpdateUnifiedAsset(ctx context.Context, asset *assetmodel.AssetUnified) error {
	err := s.repo.UpdateUnifiedAsset(ctx, asset)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.unified.UpdateUnifiedAsset", "SERVICE", map[string]interface{}{
			"id": asset.ID,
		})
		return err
	}
	return nil
}

// DeleteUnifiedAsset 删除统一资产
func (s *AssetUnifiedService) DeleteUnifiedAsset(ctx context.Context, id uint64) error {
	err := s.repo.DeleteUnifiedAsset(ctx, id)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.unified.DeleteUnifiedAsset", "SERVICE", map[string]interface{}{
			"id": id,
		})
		return err
	}
	return nil
}

// ListUnifiedAssets 获取统一资产列表
func (s *AssetUnifiedService) ListUnifiedAssets(ctx context.Context, page, pageSize int, filter assetrepo.UnifiedAssetFilter) ([]*assetmodel.AssetUnified, int64, error) {
	return s.repo.ListUnifiedAssets(ctx, page, pageSize, filter)
}

// UpsertUnifiedAsset 插入或更新统一资产
func (s *AssetUnifiedService) UpsertUnifiedAsset(ctx context.Context, asset *assetmodel.AssetUnified) error {
	err := s.repo.UpsertUnifiedAsset(ctx, asset)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.unified.UpsertUnifiedAsset", "SERVICE", map[string]interface{}{
			"ip":   asset.IP,
			"port": asset.Port,
		})
		return err
	}
	return nil
}
