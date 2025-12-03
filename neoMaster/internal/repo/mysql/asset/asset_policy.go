package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetPolicyRepository 资产策略仓库
// 负责 AssetWhitelist 和 AssetSkipPolicy 的数据访问
type AssetPolicyRepository struct {
	db *gorm.DB
}

// NewAssetPolicyRepository 创建 AssetPolicyRepository 实例
func NewAssetPolicyRepository(db *gorm.DB) *AssetPolicyRepository {
	return &AssetPolicyRepository{db: db}
}

// -----------------------------------------------------------------------------
// AssetWhitelist (资产白名单) CRUD
// -----------------------------------------------------------------------------

// CreateWhitelist 创建白名单
func (r *AssetPolicyRepository) CreateWhitelist(ctx context.Context, whitelist *asset.AssetWhitelist) error {
	if whitelist == nil {
		return errors.New("whitelist is nil")
	}
	err := r.db.WithContext(ctx).Create(whitelist).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_whitelist", "REPO", map[string]interface{}{
			"operation": "create_whitelist",
			"name":      whitelist.WhitelistName,
		})
		return err
	}
	return nil
}

// GetWhitelistByID 根据ID获取白名单
func (r *AssetPolicyRepository) GetWhitelistByID(ctx context.Context, id uint64) (*asset.AssetWhitelist, error) {
	var whitelist asset.AssetWhitelist
	err := r.db.WithContext(ctx).First(&whitelist, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error in business logic usually, but let caller decide
		}
		logger.LogError(err, "", 0, "", "get_whitelist_by_id", "REPO", map[string]interface{}{
			"operation": "get_whitelist_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &whitelist, nil
}

// UpdateWhitelist 更新白名单
func (r *AssetPolicyRepository) UpdateWhitelist(ctx context.Context, whitelist *asset.AssetWhitelist) error {
	if whitelist == nil || whitelist.ID == 0 {
		return errors.New("invalid whitelist or id")
	}
	err := r.db.WithContext(ctx).Save(whitelist).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_whitelist", "REPO", map[string]interface{}{
			"operation": "update_whitelist",
			"id":        whitelist.ID,
		})
		return err
	}
	return nil
}

// DeleteWhitelist 删除白名单 (软删除)
func (r *AssetPolicyRepository) DeleteWhitelist(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&asset.AssetWhitelist{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_whitelist", "REPO", map[string]interface{}{
			"operation": "delete_whitelist",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListWhitelists 获取白名单列表 (分页 + 筛选)
func (r *AssetPolicyRepository) ListWhitelists(ctx context.Context, page, pageSize int, name, targetType string) ([]*asset.AssetWhitelist, int64, error) {
	var whitelists []*asset.AssetWhitelist
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetWhitelist{})

	if name != "" {
		query = query.Where("whitelist_name LIKE ?", "%"+name+"%")
	}
	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_whitelists_count", "REPO", map[string]interface{}{
			"operation": "list_whitelists_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&whitelists).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_whitelists_find", "REPO", map[string]interface{}{
			"operation": "list_whitelists_find",
		})
		return nil, 0, err
	}

	return whitelists, total, nil
}

// -----------------------------------------------------------------------------
// AssetSkipPolicy (资产跳过策略) CRUD
// -----------------------------------------------------------------------------

// CreateSkipPolicy 创建跳过策略
func (r *AssetPolicyRepository) CreateSkipPolicy(ctx context.Context, policy *asset.AssetSkipPolicy) error {
	if policy == nil {
		return errors.New("policy is nil")
	}
	err := r.db.WithContext(ctx).Create(policy).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_skip_policy", "REPO", map[string]interface{}{
			"operation": "create_skip_policy",
			"name":      policy.PolicyName,
		})
		return err
	}
	return nil
}

// GetSkipPolicyByID 根据ID获取跳过策略
func (r *AssetPolicyRepository) GetSkipPolicyByID(ctx context.Context, id uint64) (*asset.AssetSkipPolicy, error) {
	var policy asset.AssetSkipPolicy
	err := r.db.WithContext(ctx).First(&policy, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_skip_policy_by_id", "REPO", map[string]interface{}{
			"operation": "get_skip_policy_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &policy, nil
}

// UpdateSkipPolicy 更新跳过策略
func (r *AssetPolicyRepository) UpdateSkipPolicy(ctx context.Context, policy *asset.AssetSkipPolicy) error {
	if policy == nil || policy.ID == 0 {
		return errors.New("invalid policy or id")
	}
	err := r.db.WithContext(ctx).Save(policy).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_skip_policy", "REPO", map[string]interface{}{
			"operation": "update_skip_policy",
			"id":        policy.ID,
		})
		return err
	}
	return nil
}

// DeleteSkipPolicy 删除跳过策略
func (r *AssetPolicyRepository) DeleteSkipPolicy(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&asset.AssetSkipPolicy{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_skip_policy", "REPO", map[string]interface{}{
			"operation": "delete_skip_policy",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListSkipPolicies 获取跳过策略列表 (分页 + 筛选)
func (r *AssetPolicyRepository) ListSkipPolicies(ctx context.Context, page, pageSize int, name, policyType string) ([]*asset.AssetSkipPolicy, int64, error) {
	var policies []*asset.AssetSkipPolicy
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetSkipPolicy{})

	if name != "" {
		query = query.Where("policy_name LIKE ?", "%"+name+"%")
	}
	if policyType != "" {
		query = query.Where("policy_type = ?", policyType)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_skip_policies_count", "REPO", map[string]interface{}{
			"operation": "list_skip_policies_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&policies).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_skip_policies_find", "REPO", map[string]interface{}{
			"operation": "list_skip_policies_find",
		})
		return nil, 0, err
	}

	return policies, total, nil
}
