package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
)

// AssetPolicyService 资产策略服务
// 负责处理资产白名单和跳过策略的业务逻辑
type AssetPolicyService struct {
	repo *assetrepo.AssetPolicyRepository
}

// NewAssetPolicyService 创建 AssetPolicyService 实例
func NewAssetPolicyService(repo *assetrepo.AssetPolicyRepository) *AssetPolicyService {
	return &AssetPolicyService{repo: repo}
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
func (s *AssetPolicyService) ListWhitelists(ctx context.Context, page, pageSize int, name, targetType string) ([]*asset.AssetWhitelist, int64, error) {
	list, total, err := s.repo.ListWhitelists(ctx, page, pageSize, name, targetType)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_whitelists", "SERVICE", map[string]interface{}{
			"operation":   "list_whitelists",
			"page":        page,
			"page_size":   pageSize,
			"name_filter": name,
		})
		return nil, 0, err
	}
	return list, total, nil
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
func (s *AssetPolicyService) ListSkipPolicies(ctx context.Context, page, pageSize int, name, policyType string) ([]*asset.AssetSkipPolicy, int64, error) {
	list, total, err := s.repo.ListSkipPolicies(ctx, page, pageSize, name, policyType)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_skip_policies", "SERVICE", map[string]interface{}{
			"operation":   "list_skip_policies",
			"page":        page,
			"page_size":   pageSize,
			"name_filter": name,
		})
		return nil, 0, err
	}
	return list, total, nil
}
