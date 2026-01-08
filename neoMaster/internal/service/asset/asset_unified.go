package asset

import (
	"context"
	assetmodel "neomaster/internal/model/asset"
	tagsystem "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
	tagservice "neomaster/internal/service/tag_system"
	"strconv"
)

// AssetUnifiedService 统一资产服务层
// 处理资产视图的业务逻辑
type AssetUnifiedService struct {
	repo       *assetrepo.AssetUnifiedRepository
	tagService tagservice.TagService
}

// NewAssetUnifiedService 创建 AssetUnifiedService 实例
func NewAssetUnifiedService(repo *assetrepo.AssetUnifiedRepository, tagService tagservice.TagService) *AssetUnifiedService {
	return &AssetUnifiedService{
		repo:       repo,
		tagService: tagService,
	}
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
func (s *AssetUnifiedService) ListUnifiedAssets(ctx context.Context, page, pageSize int, filter assetrepo.UnifiedAssetFilter, tagIDs []uint64) ([]*assetmodel.AssetUnified, int64, error) {
	if len(tagIDs) > 0 {
		entityIDsStr, err := s.tagService.GetEntityIDsByTagIDs(ctx, "unified_asset", tagIDs)
		if err != nil {
			return nil, 0, err
		}
		if len(entityIDsStr) == 0 {
			return []*assetmodel.AssetUnified{}, 0, nil
		}

		var ids []uint64
		for _, idStr := range entityIDsStr {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return []*assetmodel.AssetUnified{}, 0, nil
		}
		filter.IDs = ids
	}

	return s.repo.ListUnifiedAssets(ctx, page, pageSize, filter)
}

// -----------------------------------------------------------------------------
// Tag Management
// -----------------------------------------------------------------------------

// AddUnifiedAssetTag 为统一资产添加标签
func (s *AssetUnifiedService) AddUnifiedAssetTag(ctx context.Context, assetID uint64, tagID uint64) error {
	return s.tagService.AddEntityTag(ctx, "unified_asset", strconv.FormatUint(assetID, 10), tagID, "manual", 0)
}

// RemoveUnifiedAssetTag 为统一资产移除标签
func (s *AssetUnifiedService) RemoveUnifiedAssetTag(ctx context.Context, assetID uint64, tagID uint64) error {
	return s.tagService.RemoveEntityTag(ctx, "unified_asset", strconv.FormatUint(assetID, 10), tagID)
}

// GetUnifiedAssetTags 获取统一资产的标签
func (s *AssetUnifiedService) GetUnifiedAssetTags(ctx context.Context, assetID uint64) ([]tagsystem.SysTag, error) {
	entityTags, err := s.tagService.GetEntityTags(ctx, "unified_asset", strconv.FormatUint(assetID, 10))
	if err != nil {
		return nil, err
	}
	if len(entityTags) == 0 {
		return []tagsystem.SysTag{}, nil
	}
	var tagIDs []uint64
	for _, et := range entityTags {
		tagIDs = append(tagIDs, et.TagID)
	}
	return s.tagService.GetTagsByIDs(ctx, tagIDs)
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
