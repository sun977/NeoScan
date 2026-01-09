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

type AssetCPEService struct {
	repo       assetrepo.AssetCPERepository
	tagService tagservice.TagService
}

func NewAssetCPEService(repo assetrepo.AssetCPERepository, tagService tagservice.TagService) *AssetCPEService {
	return &AssetCPEService{repo: repo, tagService: tagService}
}

func (s *AssetCPEService) CreateCPERule(ctx context.Context, rule *assetmodel.AssetCPE) error {
	if rule == nil {
		return errors.New("rule data cannot be nil")
	}
	if strings.TrimSpace(rule.MatchStr) == "" {
		return errors.New("match_str cannot be empty")
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_cpe_rule", "SERVICE", map[string]interface{}{
			"operation": "create_cpe_rule",
			"name":      rule.Name,
			"vendor":    rule.Vendor,
			"product":   rule.Product,
		})
		return err
	}
	return nil
}

func (s *AssetCPEService) GetCPERule(ctx context.Context, id uint64) (*assetmodel.AssetCPE, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_cpe_rule", "SERVICE", map[string]interface{}{
			"operation": "get_cpe_rule",
			"id":        id,
		})
		return nil, err
	}
	if rule == nil {
		return nil, errors.New("cpe rule not found")
	}
	return rule, nil
}

func (s *AssetCPEService) UpdateCPERule(ctx context.Context, rule *assetmodel.AssetCPE) error {
	if rule == nil || rule.ID == 0 {
		return errors.New("invalid rule or id")
	}
	if strings.TrimSpace(rule.MatchStr) == "" {
		return errors.New("match_str cannot be empty")
	}

	existing, err := s.repo.GetByID(ctx, rule.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("cpe rule not found")
	}

	if err := s.repo.Update(ctx, rule); err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_cpe_rule", "SERVICE", map[string]interface{}{
			"operation": "update_cpe_rule",
			"id":        rule.ID,
		})
		return err
	}
	return nil
}

func (s *AssetCPEService) DeleteCPERule(ctx context.Context, id uint64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("cpe rule not found")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_cpe_rule", "SERVICE", map[string]interface{}{
			"operation": "delete_cpe_rule",
			"id":        id,
		})
		return err
	}
	return nil
}

func (s *AssetCPEService) ListCPERules(ctx context.Context, page, pageSize int, name, vendor, product string) ([]*assetmodel.AssetCPE, int64, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	list, total, err := s.repo.List(ctx, page, pageSize, name, vendor, product)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_cpe_rules", "SERVICE", map[string]interface{}{
			"operation": "list_cpe_rules",
			"page":      page,
			"page_size": pageSize,
			"name":      name,
			"vendor":    vendor,
			"product":   product,
		})
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return list, total, totalPages, nil
}

func (s *AssetCPEService) AddTagToCPERule(ctx context.Context, ruleID uint64, tagID uint64) error {
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return errors.New("cpe rule not found")
	}

	if err := s.tagService.AddEntityTag(ctx, "fingers_cpe", strconv.FormatUint(ruleID, 10), tagID, "manual", 0); err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_cpe_rule", "SERVICE", map[string]interface{}{
			"operation": "add_tag_to_cpe_rule",
			"rule_id":   ruleID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

func (s *AssetCPEService) RemoveTagFromCPERule(ctx context.Context, ruleID uint64, tagID uint64) error {
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return errors.New("cpe rule not found")
	}

	if err := s.tagService.RemoveEntityTag(ctx, "fingers_cpe", strconv.FormatUint(ruleID, 10), tagID); err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_cpe_rule", "SERVICE", map[string]interface{}{
			"operation": "remove_tag_from_cpe_rule",
			"rule_id":   ruleID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

func (s *AssetCPEService) GetCPERuleTags(ctx context.Context, ruleID uint64) ([]tagsystem.SysTag, error) {
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, errors.New("cpe rule not found")
	}

	entityTags, err := s.tagService.GetEntityTags(ctx, "fingers_cpe", strconv.FormatUint(ruleID, 10))
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_cpe_rule_tags", "SERVICE", map[string]interface{}{
			"operation": "get_cpe_rule_tags",
			"rule_id":   ruleID,
		})
		return nil, err
	}

	if len(entityTags) == 0 {
		return []tagsystem.SysTag{}, nil
	}

	tagIDs := make([]uint64, 0, len(entityTags))
	for _, et := range entityTags {
		tagIDs = append(tagIDs, et.TagID)
	}

	tags, err := s.tagService.GetTagsByIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	return tags, nil
}
