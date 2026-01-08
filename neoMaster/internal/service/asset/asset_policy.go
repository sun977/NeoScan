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

// AssetPolicyService 资产策略服务
// 负责处理资产白名单和跳过策略的业务逻辑
type AssetPolicyService struct {
	repo       *assetrepo.AssetPolicyRepository
	tagService tagservice.TagService
}

// NewAssetPolicyService 创建 AssetPolicyService 实例
func NewAssetPolicyService(repo *assetrepo.AssetPolicyRepository, tagService tagservice.TagService) *AssetPolicyService {
	return &AssetPolicyService{
		repo:       repo,
		tagService: tagService,
	}
}

// -----------------------------------------------------------------------------
// AssetWhitelist 业务逻辑
// -----------------------------------------------------------------------------

// CreateWhitelist 创建白名单
func (s *AssetPolicyService) CreateWhitelist(ctx context.Context, whitelist *asset.AssetWhitelist) error {
	if whitelist == nil {
		return errors.New("whitelist data cannot be nil")
	}
	// TODO: 可以在这里添加业务校验，例如检查名称是否重复 (如果需要 Repo 提供 CheckName 接口)

	err := s.repo.CreateWhitelist(ctx, whitelist)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_whitelist", "SERVICE", map[string]interface{}{
			"operation": "create_whitelist",
			"name":      whitelist.WhitelistName,
		})
		return err
	}
	return nil
}

// GetWhitelist 获取白名单详情
func (s *AssetPolicyService) GetWhitelist(ctx context.Context, id uint64) (*asset.AssetWhitelist, error) {
	whitelist, err := s.repo.GetWhitelistByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_whitelist", "SERVICE", map[string]interface{}{
			"operation": "get_whitelist",
			"id":        id,
		})
		return nil, err
	}
	if whitelist == nil {
		return nil, errors.New("whitelist not found")
	}
	return whitelist, nil
}

// UpdateWhitelist 更新白名单
func (s *AssetPolicyService) UpdateWhitelist(ctx context.Context, whitelist *asset.AssetWhitelist) error {
	// 检查是否存在
	existing, err := s.repo.GetWhitelistByID(ctx, whitelist.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("whitelist not found")
	}

	err = s.repo.UpdateWhitelist(ctx, whitelist)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_whitelist", "SERVICE", map[string]interface{}{
			"operation": "update_whitelist",
			"id":        whitelist.ID,
		})
		return err
	}
	return nil
}

// DeleteWhitelist 删除白名单
func (s *AssetPolicyService) DeleteWhitelist(ctx context.Context, id uint64) error {
	// 检查是否存在
	existing, err := s.repo.GetWhitelistByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("whitelist not found")
	}

	err = s.repo.DeleteWhitelist(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_whitelist", "SERVICE", map[string]interface{}{
			"operation": "delete_whitelist",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListWhitelists 获取白名单列表
func (s *AssetPolicyService) ListWhitelists(ctx context.Context, page, pageSize int, name, targetType string, tagIDs []uint64) ([]*asset.AssetWhitelist, int64, error) {
	var whitelistIDs []uint64

	// 如果指定了标签，先从标签系统获取对应的 WhitelistID 列表
	if len(tagIDs) > 0 {
		entityIDsStr, err := s.tagService.GetEntityIDsByTagIDs(ctx, "whitelist", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "list_whitelists_get_tags", "SERVICE", map[string]interface{}{
				"operation": "list_whitelists_get_tags",
				"tag_ids":   tagIDs,
			})
			return nil, 0, err
		}

		if len(entityIDsStr) == 0 {
			// 筛选了标签但没找到对应的资源，直接返回空列表
			return []*asset.AssetWhitelist{}, 0, nil
		}

		// 转换 ID 类型
		for _, idStr := range entityIDsStr {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			whitelistIDs = append(whitelistIDs, id)
		}

		if len(whitelistIDs) == 0 {
			return []*asset.AssetWhitelist{}, 0, nil
		}
	}

	list, total, err := s.repo.ListWhitelists(ctx, page, pageSize, name, targetType, whitelistIDs)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_whitelists", "SERVICE", map[string]interface{}{
			"operation":   "list_whitelists",
			"page":        page,
			"page_size":   pageSize,
			"name_filter": name,
			"tag_ids":     tagIDs,
		})
		return nil, 0, err
	}
	return list, total, nil
}

// AddTagToWhitelist 添加标签到白名单
func (s *AssetPolicyService) AddTagToWhitelist(ctx context.Context, whitelistID uint64, tagID uint64) error {
	// 检查白名单是否存在
	whitelist, err := s.repo.GetWhitelistByID(ctx, whitelistID)
	if err != nil {
		return err
	}
	if whitelist == nil {
		return errors.New("whitelist not found")
	}

	// 添加标签 (Source=manual)
	err = s.tagService.AddEntityTag(ctx, "whitelist", strconv.FormatUint(whitelistID, 10), tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_whitelist", "SERVICE", map[string]interface{}{
			"operation":    "add_tag_to_whitelist",
			"whitelist_id": whitelistID,
			"tag_id":       tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromWhitelist 从白名单移除标签
func (s *AssetPolicyService) RemoveTagFromWhitelist(ctx context.Context, whitelistID uint64, tagID uint64) error {
	// 检查白名单是否存在
	whitelist, err := s.repo.GetWhitelistByID(ctx, whitelistID)
	if err != nil {
		return err
	}
	if whitelist == nil {
		return errors.New("whitelist not found")
	}

	err = s.tagService.RemoveEntityTag(ctx, "whitelist", strconv.FormatUint(whitelistID, 10), tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_whitelist", "SERVICE", map[string]interface{}{
			"operation":    "remove_tag_from_whitelist",
			"whitelist_id": whitelistID,
			"tag_id":       tagID,
		})
		return err
	}
	return nil
}

// GetWhitelistTags 获取白名单标签
func (s *AssetPolicyService) GetWhitelistTags(ctx context.Context, whitelistID uint64) ([]tagsystem.SysTag, error) {
	// 检查白名单是否存在
	whitelist, err := s.repo.GetWhitelistByID(ctx, whitelistID)
	if err != nil {
		return nil, err
	}
	if whitelist == nil {
		return nil, errors.New("whitelist not found")
	}

	// 1. 获取实体关联关系
	entityTags, err := s.tagService.GetEntityTags(ctx, "whitelist", strconv.FormatUint(whitelistID, 10))
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_whitelist_tags", "SERVICE", map[string]interface{}{
			"operation":    "get_whitelist_tags",
			"whitelist_id": whitelistID,
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
// AssetSkipPolicy 业务逻辑
// -----------------------------------------------------------------------------

// CreateSkipPolicy 创建跳过策略
func (s *AssetPolicyService) CreateSkipPolicy(ctx context.Context, policy *asset.AssetSkipPolicy) error {
	if policy == nil {
		return errors.New("policy data cannot be nil")
	}

	err := s.repo.CreateSkipPolicy(ctx, policy)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_skip_policy", "SERVICE", map[string]interface{}{
			"operation": "create_skip_policy",
			"name":      policy.PolicyName,
		})
		return err
	}
	return nil
}

// GetSkipPolicy 获取跳过策略详情
func (s *AssetPolicyService) GetSkipPolicy(ctx context.Context, id uint64) (*asset.AssetSkipPolicy, error) {
	policy, err := s.repo.GetSkipPolicyByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_skip_policy", "SERVICE", map[string]interface{}{
			"operation": "get_skip_policy",
			"id":        id,
		})
		return nil, err
	}
	if policy == nil {
		return nil, errors.New("policy not found")
	}
	return policy, nil
}

// UpdateSkipPolicy 更新跳过策略
func (s *AssetPolicyService) UpdateSkipPolicy(ctx context.Context, policy *asset.AssetSkipPolicy) error {
	// 检查是否存在
	existing, err := s.repo.GetSkipPolicyByID(ctx, policy.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("policy not found")
	}

	err = s.repo.UpdateSkipPolicy(ctx, policy)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_skip_policy", "SERVICE", map[string]interface{}{
			"operation": "update_skip_policy",
			"id":        policy.ID,
		})
		return err
	}
	return nil
}

// DeleteSkipPolicy 删除跳过策略
func (s *AssetPolicyService) DeleteSkipPolicy(ctx context.Context, id uint64) error {
	// 检查是否存在
	existing, err := s.repo.GetSkipPolicyByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("policy not found")
	}

	err = s.repo.DeleteSkipPolicy(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_skip_policy", "SERVICE", map[string]interface{}{
			"operation": "delete_skip_policy",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListSkipPolicies 获取跳过策略列表
func (s *AssetPolicyService) ListSkipPolicies(ctx context.Context, page, pageSize int, name, policyType string, tagIDs []uint64) ([]*asset.AssetSkipPolicy, int64, error) {
	var policyIDs []uint64

	// 如果指定了标签，先从标签系统获取对应的 PolicyID 列表
	if len(tagIDs) > 0 {
		entityIDsStr, err := s.tagService.GetEntityIDsByTagIDs(ctx, "skip_policy", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "list_skip_policies_get_tags", "SERVICE", map[string]interface{}{
				"operation": "list_skip_policies_get_tags",
				"tag_ids":   tagIDs,
			})
			return nil, 0, err
		}

		if len(entityIDsStr) == 0 {
			// 筛选了标签但没找到对应的资源，直接返回空列表
			return []*asset.AssetSkipPolicy{}, 0, nil
		}

		// 转换 ID 类型
		for _, idStr := range entityIDsStr {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			policyIDs = append(policyIDs, id)
		}

		if len(policyIDs) == 0 {
			return []*asset.AssetSkipPolicy{}, 0, nil
		}
	}

	list, total, err := s.repo.ListSkipPolicies(ctx, page, pageSize, name, policyType, policyIDs)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_skip_policies", "SERVICE", map[string]interface{}{
			"operation":   "list_skip_policies",
			"page":        page,
			"page_size":   pageSize,
			"name_filter": name,
			"tag_ids":     tagIDs,
		})
		return nil, 0, err
	}
	return list, total, nil
}

// AddTagToSkipPolicy 添加标签到跳过策略
func (s *AssetPolicyService) AddTagToSkipPolicy(ctx context.Context, policyID uint64, tagID uint64) error {
	// 检查策略是否存在
	policy, err := s.repo.GetSkipPolicyByID(ctx, policyID)
	if err != nil {
		return err
	}
	if policy == nil {
		return errors.New("skip policy not found")
	}

	// 添加标签 (Source=manual)
	err = s.tagService.AddEntityTag(ctx, "skip_policy", strconv.FormatUint(policyID, 10), tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_skip_policy", "SERVICE", map[string]interface{}{
			"operation": "add_tag_to_skip_policy",
			"policy_id": policyID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromSkipPolicy 从跳过策略移除标签
func (s *AssetPolicyService) RemoveTagFromSkipPolicy(ctx context.Context, policyID uint64, tagID uint64) error {
	// 检查策略是否存在
	policy, err := s.repo.GetSkipPolicyByID(ctx, policyID)
	if err != nil {
		return err
	}
	if policy == nil {
		return errors.New("skip policy not found")
	}

	err = s.tagService.RemoveEntityTag(ctx, "skip_policy", strconv.FormatUint(policyID, 10), tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_skip_policy", "SERVICE", map[string]interface{}{
			"operation": "remove_tag_from_skip_policy",
			"policy_id": policyID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

// GetSkipPolicyTags 获取跳过策略标签
func (s *AssetPolicyService) GetSkipPolicyTags(ctx context.Context, policyID uint64) ([]tagsystem.SysTag, error) {
	// 检查策略是否存在
	policy, err := s.repo.GetSkipPolicyByID(ctx, policyID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, errors.New("skip policy not found")
	}

	// 1. 获取实体关联关系
	entityTags, err := s.tagService.GetEntityTags(ctx, "skip_policy", strconv.FormatUint(policyID, 10))
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_skip_policy_tags", "SERVICE", map[string]interface{}{
			"operation": "get_skip_policy_tags",
			"policy_id": policyID,
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
