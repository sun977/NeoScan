package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	tagsystem "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
	tagservice "neomaster/internal/service/tag_system"
	"strconv"
)

// AssetNetworkService 资产网段服务
// 负责处理网段资产的业务逻辑
type AssetNetworkService struct {
	repo       *assetrepo.AssetNetworkRepository
	tagService tagservice.TagService
}

// NewAssetNetworkService 创建 AssetNetworkService 实例
func NewAssetNetworkService(repo *assetrepo.AssetNetworkRepository, tagService tagservice.TagService) *AssetNetworkService {
	return &AssetNetworkService{
		repo:       repo,
		tagService: tagService,
	}
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
func (s *AssetNetworkService) ListNetworks(ctx context.Context, page, pageSize int, cidr, networkType, status string, tagIDs []uint64) ([]*asset.AssetNetwork, int64, error) {
	var networkIDs []uint64

	// 如果指定了标签，先从标签系统获取对应的 NetworkID 列表
	if len(tagIDs) > 0 {
		entityIDsStr, err := s.tagService.GetEntityIDsByTagIDs(ctx, "network", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "list_networks_get_tags", "SERVICE", map[string]interface{}{
				"operation": "list_networks_get_tags",
				"tag_ids":   tagIDs,
			})
			return nil, 0, err
		}

		if len(entityIDsStr) == 0 {
			// 筛选了标签但没找到对应的资源，直接返回空列表
			return []*asset.AssetNetwork{}, 0, nil
		}

		// 转换 ID 类型
		for _, idStr := range entityIDsStr {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			networkIDs = append(networkIDs, id)
		}

		if len(networkIDs) == 0 {
			return []*asset.AssetNetwork{}, 0, nil
		}
	}

	list, total, err := s.repo.ListNetworks(ctx, page, pageSize, cidr, networkType, status, networkIDs)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_networks", "SERVICE", map[string]interface{}{
			"operation": "list_networks",
			"page":      page,
			"page_size": pageSize,
			"tag_ids":   tagIDs,
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

// AddTagToNetwork 添加标签到网段
func (s *AssetNetworkService) AddTagToNetwork(ctx context.Context, networkID uint64, tagID uint64) error {
	// 检查网段是否存在
	network, err := s.repo.GetNetworkByID(ctx, networkID)
	if err != nil {
		return err
	}
	if network == nil {
		return errors.New("network not found")
	}

	// 添加标签 (Source=manual)
	err = s.tagService.AddEntityTag(ctx, "network", strconv.FormatUint(networkID, 10), tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_network", "SERVICE", map[string]interface{}{
			"operation":  "add_tag_to_network",
			"network_id": networkID,
			"tag_id":     tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromNetwork 从网段移除标签
func (s *AssetNetworkService) RemoveTagFromNetwork(ctx context.Context, networkID uint64, tagID uint64) error {
	// 检查网段是否存在
	network, err := s.repo.GetNetworkByID(ctx, networkID)
	if err != nil {
		return err
	}
	if network == nil {
		return errors.New("network not found")
	}

	err = s.tagService.RemoveEntityTag(ctx, "network", strconv.FormatUint(networkID, 10), tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_network", "SERVICE", map[string]interface{}{
			"operation":  "remove_tag_from_network",
			"network_id": networkID,
			"tag_id":     tagID,
		})
		return err
	}
	return nil
}

// GetNetworkTags 获取网段标签
func (s *AssetNetworkService) GetNetworkTags(ctx context.Context, networkID uint64) ([]tagsystem.SysTag, error) {
	// 检查网段是否存在
	network, err := s.repo.GetNetworkByID(ctx, networkID)
	if err != nil {
		return nil, err
	}
	if network == nil {
		return nil, errors.New("network not found")
	}

	// 1. 获取实体关联关系
	entityTags, err := s.tagService.GetEntityTags(ctx, "network", strconv.FormatUint(networkID, 10))
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_network_tags", "SERVICE", map[string]interface{}{
			"operation":  "get_network_tags",
			"network_id": networkID,
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
