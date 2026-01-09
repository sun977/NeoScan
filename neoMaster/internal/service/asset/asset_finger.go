package asset

import (
	"context"
	"errors"
	"math"
	"strings"

	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
)

// AssetFingerService 资产指纹规则服务
// 负责处理 asset_finger 表的增删改查业务逻辑
type AssetFingerService struct {
	repo assetrepo.AssetFingerRepository
}

// NewAssetFingerService 创建 AssetFingerService 实例
func NewAssetFingerService(repo assetrepo.AssetFingerRepository) *AssetFingerService {
	return &AssetFingerService{repo: repo}
}

// CreateFingerRule 创建指纹规则
func (s *AssetFingerService) CreateFingerRule(ctx context.Context, rule *assetmodel.AssetFinger) error {
	if rule == nil {
		return errors.New("rule data cannot be nil")
	}
	if strings.TrimSpace(rule.Name) == "" {
		return errors.New("fingerprint name cannot be empty")
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_finger_rule", "SERVICE", map[string]interface{}{
			"operation": "create_finger_rule",
			"name":      rule.Name,
		})
		return err
	}
	return nil
}

// GetFingerRule 获取指纹规则详情
func (s *AssetFingerService) GetFingerRule(ctx context.Context, id uint64) (*assetmodel.AssetFinger, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_finger_rule", "SERVICE", map[string]interface{}{
			"operation": "get_finger_rule",
			"id":        id,
		})
		return nil, err
	}
	if rule == nil {
		return nil, errors.New("fingerprint rule not found")
	}
	return rule, nil
}

// UpdateFingerRule 更新指纹规则
func (s *AssetFingerService) UpdateFingerRule(ctx context.Context, rule *assetmodel.AssetFinger) error {
	if rule == nil || rule.ID == 0 {
		return errors.New("invalid rule or id")
	}
	if strings.TrimSpace(rule.Name) == "" {
		return errors.New("fingerprint name cannot be empty")
	}

	existing, err := s.repo.GetByID(ctx, rule.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("fingerprint rule not found")
	}

	if err := s.repo.Update(ctx, rule); err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_finger_rule", "SERVICE", map[string]interface{}{
			"operation": "update_finger_rule",
			"id":        rule.ID,
		})
		return err
	}
	return nil
}

// DeleteFingerRule 删除指纹规则
func (s *AssetFingerService) DeleteFingerRule(ctx context.Context, id uint64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("fingerprint rule not found")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_finger_rule", "SERVICE", map[string]interface{}{
			"operation": "delete_finger_rule",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListFingerRules 获取指纹规则列表
func (s *AssetFingerService) ListFingerRules(ctx context.Context, page, pageSize int, name string) ([]*assetmodel.AssetFinger, int64, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	list, total, err := s.repo.List(ctx, page, pageSize, name)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_finger_rules", "SERVICE", map[string]interface{}{
			"operation": "list_finger_rules",
			"page":      page,
			"page_size": pageSize,
			"name":      name,
		})
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return list, total, totalPages, nil
}

