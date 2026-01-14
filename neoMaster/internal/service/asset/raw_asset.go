package asset

import (
	"context"
	"errors"
	"strconv"

	assetmodel "neomaster/internal/model/asset"
	tagsystem "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
	tagservice "neomaster/internal/service/tag_system"
)

// RawAssetService 原始资产服务层
// 处理原始数据导入和清洗的业务逻辑
type RawAssetService struct {
	repo       *assetrepo.RawAssetRepository
	tagService tagservice.TagService
}

// NewRawAssetService 创建 RawAssetService 实例
func NewRawAssetService(repo *assetrepo.RawAssetRepository, tagService tagservice.TagService) *RawAssetService {
	return &RawAssetService{
		repo:       repo,
		tagService: tagService,
	}
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
func (s *RawAssetService) ListRawAssets(ctx context.Context, page, pageSize int, batchID string, status string, tagIDs []uint64) ([]*assetmodel.RawAsset, int64, error) {
	var rawAssetIDs []uint64

	// 如果指定了标签，先从标签系统获取对应的 RawAssetID 列表
	if len(tagIDs) > 0 {
		entityIDsStr, err := s.tagService.GetEntityIDsByTagIDs(ctx, "raw_asset", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "list_raw_assets_get_tags", "SERVICE", map[string]interface{}{
				"operation": "list_raw_assets_get_tags",
				"tag_ids":   tagIDs,
			})
			return nil, 0, err
		}

		if len(entityIDsStr) == 0 {
			// 筛选了标签但没找到对应的资源，直接返回空列表
			return []*assetmodel.RawAsset{}, 0, nil
		}

		// 转换 ID 类型
		for _, idStr := range entityIDsStr {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			rawAssetIDs = append(rawAssetIDs, id)
		}

		if len(rawAssetIDs) == 0 {
			return []*assetmodel.RawAsset{}, 0, nil
		}
	}

	return s.repo.ListRawAssets(ctx, page, pageSize, batchID, status, rawAssetIDs)
}

// AddTagToRawAsset 添加标签到原始资产
func (s *RawAssetService) AddTagToRawAsset(ctx context.Context, rawAssetID uint64, tagID uint64) error {
	// 检查原始资产是否存在
	rawAsset, err := s.repo.GetRawAssetByID(ctx, rawAssetID)
	if err != nil {
		return err
	}
	if rawAsset == nil {
		return errors.New("raw asset not found")
	}

	// 添加标签 (Source=manual)
	err = s.tagService.AddEntityTag(ctx, "raw_asset", strconv.FormatUint(rawAssetID, 10), tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_raw_asset", "SERVICE", map[string]interface{}{
			"operation":    "add_tag_to_raw_asset",
			"raw_asset_id": rawAssetID,
			"tag_id":       tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromRawAsset 从原始资产移除标签
func (s *RawAssetService) RemoveTagFromRawAsset(ctx context.Context, rawAssetID uint64, tagID uint64) error {
	// 检查原始资产是否存在
	rawAsset, err := s.repo.GetRawAssetByID(ctx, rawAssetID)
	if err != nil {
		return err
	}
	if rawAsset == nil {
		return errors.New("raw asset not found")
	}

	err = s.tagService.RemoveEntityTag(ctx, "raw_asset", strconv.FormatUint(rawAssetID, 10), tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_raw_asset", "SERVICE", map[string]interface{}{
			"operation":    "remove_tag_from_raw_asset",
			"raw_asset_id": rawAssetID,
			"tag_id":       tagID,
		})
		return err
	}
	return nil
}

// GetRawAssetTags 获取原始资产标签
func (s *RawAssetService) GetRawAssetTags(ctx context.Context, rawAssetID uint64) ([]tagsystem.SysTag, error) {
	// 检查原始资产是否存在
	rawAsset, err := s.repo.GetRawAssetByID(ctx, rawAssetID)
	if err != nil {
		return nil, err
	}
	if rawAsset == nil {
		return nil, errors.New("raw asset not found")
	}

	// 1. 获取实体关联关系
	entityTags, err := s.tagService.GetEntityTags(ctx, "raw_asset", strconv.FormatUint(rawAssetID, 10))
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_raw_asset_tags", "SERVICE", map[string]interface{}{
			"operation":    "get_raw_asset_tags",
			"raw_asset_id": rawAssetID,
		})
		return nil, err
	}

	if len(entityTags) == 0 {
		return []tagsystem.SysTag{}, nil
	}

	// 2. 提取TagIDs
	var tagIDs []uint64
	for _, et := range entityTags {
		tagIDs = append(tagIDs, et.TagID)
	}

	// 3. 批量获取标签详情
	tags, err := s.tagService.GetTagsByIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	return tags, nil
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
