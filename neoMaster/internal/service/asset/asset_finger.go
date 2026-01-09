// 除了基础的CRUD之外
// 还需要：数据库指纹表转换成供agent下载的指纹文件
// 还需要：用户的指纹文件加载到系统指纹库

package asset

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	assetmodel "neomaster/internal/model/asset"
	tagsystem "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
	tagservice "neomaster/internal/service/tag_system"
)

// AssetFingerService 资产指纹规则服务
// 负责处理 asset_finger 表的增删改查业务逻辑
type AssetFingerService struct {
	repo       assetrepo.AssetFingerRepository
	tagService tagservice.TagService
}

// NewAssetFingerService 创建 AssetFingerService 实例
func NewAssetFingerService(repo assetrepo.AssetFingerRepository, tagService tagservice.TagService) *AssetFingerService {
	return &AssetFingerService{repo: repo, tagService: tagService}
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
func (s *AssetFingerService) ListFingerRules(ctx context.Context, page, pageSize int, name string, tagID uint64) ([]*assetmodel.AssetFinger, int64, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	list, total, err := s.repo.List(ctx, page, pageSize, name, tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_finger_rules", "SERVICE", map[string]interface{}{
			"operation": "list_finger_rules",
			"page":      page,
			"page_size": pageSize,
			"name":      name,
			"tag_id":    tagID,
		})
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return list, total, totalPages, nil
}

// AddTagToFingerRule 为指纹规则添加标签
func (s *AssetFingerService) AddTagToFingerRule(ctx context.Context, ruleID uint64, tagID uint64) error {
	// 检查指纹规则是否存在
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return errors.New("fingerprint rule not found")
	}

	if err := s.tagService.AddEntityTag(ctx, "fingers_cms", strconv.FormatUint(ruleID, 10), tagID, "manual", 0); err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_finger_rule", "SERVICE", map[string]interface{}{
			"operation": "add_tag_to_finger_rule",
			"rule_id":   ruleID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromFingerRule 从指纹规则移除标签
func (s *AssetFingerService) RemoveTagFromFingerRule(ctx context.Context, ruleID uint64, tagID uint64) error {
	// 检查指纹规则是否存在
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return errors.New("fingerprint rule not found")
	}

	if err := s.tagService.RemoveEntityTag(ctx, "fingers_cms", strconv.FormatUint(ruleID, 10), tagID); err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_finger_rule", "SERVICE", map[string]interface{}{
			"operation": "remove_tag_from_finger_rule",
			"rule_id":   ruleID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

// GetFingerRuleTags 获取指纹规则标签
func (s *AssetFingerService) GetFingerRuleTags(ctx context.Context, ruleID uint64) ([]tagsystem.SysTag, error) {
	// 检查指纹规则是否存在
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, errors.New("fingerprint rule not found")
	}

	entityTags, err := s.tagService.GetEntityTags(ctx, "fingers_cms", strconv.FormatUint(ruleID, 10))
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_finger_rule_tags", "SERVICE", map[string]interface{}{
			"operation": "get_finger_rule_tags",
			"rule_id":   ruleID,
		})
		return nil, err
	}

	if len(entityTags) == 0 {
		return []tagsystem.SysTag{}, nil
	}

	var tagIDs []uint64
	for _, et := range entityTags {
		tagIDs = append(tagIDs, et.TagID)
	}

	tags, err := s.tagService.GetTagsByIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	return tags, nil
}
