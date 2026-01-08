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

// AssetVulnService 漏洞资产服务层
// 处理漏洞及其PoC的业务逻辑
type AssetVulnService struct {
	repo       *assetrepo.AssetVulnRepository
	tagService tagservice.TagService
}

// NewAssetVulnService 创建 AssetVulnService 实例
func NewAssetVulnService(repo *assetrepo.AssetVulnRepository, tagService tagservice.TagService) *AssetVulnService {
	return &AssetVulnService{
		repo:       repo,
		tagService: tagService,
	}
}

// -----------------------------------------------------------------------------
// AssetVuln 业务逻辑
// -----------------------------------------------------------------------------

// CreateVuln 创建漏洞
func (s *AssetVulnService) CreateVuln(ctx context.Context, vuln *assetmodel.AssetVuln) error {
	err := s.repo.CreateVuln(ctx, vuln)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.vuln.CreateVuln", "SERVICE", map[string]interface{}{
			"target": vuln.TargetRefID,
			"cve":    vuln.CVE,
		})
		return err
	}
	return nil
}

// GetVulnByID 根据ID获取漏洞
func (s *AssetVulnService) GetVulnByID(ctx context.Context, id uint64) (*assetmodel.AssetVuln, error) {
	return s.repo.GetVulnByID(ctx, id)
}

// UpdateVuln 更新漏洞信息
func (s *AssetVulnService) UpdateVuln(ctx context.Context, vuln *assetmodel.AssetVuln) error {
	err := s.repo.UpdateVuln(ctx, vuln)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.vuln.UpdateVuln", "SERVICE", map[string]interface{}{
			"id": vuln.ID,
		})
		return err
	}
	return nil
}

// DeleteVuln 删除漏洞
func (s *AssetVulnService) DeleteVuln(ctx context.Context, id uint64) error {
	err := s.repo.DeleteVuln(ctx, id)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.vuln.DeleteVuln", "SERVICE", map[string]interface{}{
			"id": id,
		})
		return err
	}
	return nil
}

// ListVulns 获取漏洞列表
func (s *AssetVulnService) ListVulns(ctx context.Context, page, pageSize int, targetType string, targetRefID uint64, status string, severity string, tagIDs []uint64) ([]*assetmodel.AssetVuln, int64, error) {
	var vulnIDs []uint64
	if len(tagIDs) > 0 {
		entityIDsStr, err := s.tagService.GetEntityIDsByTagIDs(ctx, "vuln", tagIDs)
		if err != nil {
			return nil, 0, err
		}
		if len(entityIDsStr) == 0 {
			return []*assetmodel.AssetVuln{}, 0, nil
		}
		for _, idStr := range entityIDsStr {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			vulnIDs = append(vulnIDs, id)
		}
		if len(vulnIDs) == 0 {
			return []*assetmodel.AssetVuln{}, 0, nil
		}
	}

	return s.repo.ListVulns(ctx, page, pageSize, targetType, targetRefID, status, severity, vulnIDs)
}

// -----------------------------------------------------------------------------
// AssetVulnPoc 业务逻辑
// -----------------------------------------------------------------------------

// CreatePoc 创建PoC
func (s *AssetVulnService) CreatePoc(ctx context.Context, poc *assetmodel.AssetVulnPoc) error {
	err := s.repo.CreatePoc(ctx, poc)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.vuln.CreatePoc", "SERVICE", map[string]interface{}{
			"vuln_id": poc.VulnID,
			"name":    poc.Name,
		})
		return err
	}
	return nil
}

// GetPocByID 根据ID获取PoC
func (s *AssetVulnService) GetPocByID(ctx context.Context, id uint64) (*assetmodel.AssetVulnPoc, error) {
	return s.repo.GetPocByID(ctx, id)
}

// UpdatePoc 更新PoC
func (s *AssetVulnService) UpdatePoc(ctx context.Context, poc *assetmodel.AssetVulnPoc) error {
	err := s.repo.UpdatePoc(ctx, poc)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.vuln.UpdatePoc", "SERVICE", map[string]interface{}{
			"id": poc.ID,
		})
		return err
	}
	return nil
}

// DeletePoc 删除PoC
func (s *AssetVulnService) DeletePoc(ctx context.Context, id uint64) error {
	err := s.repo.DeletePoc(ctx, id)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.vuln.DeletePoc", "SERVICE", map[string]interface{}{
			"id": id,
		})
		return err
	}
	return nil
}

// GetValidPocsByVulnID 获取指定漏洞的所有有效PoC (用于执行调度)
func (s *AssetVulnService) GetValidPocsByVulnID(ctx context.Context, vulnID uint64) ([]*assetmodel.AssetVulnPoc, error) {
	return s.repo.ListPocsByVulnID(ctx, vulnID, true, nil)
}

// ListAllPocsByVulnID 获取指定漏洞的所有PoC (包括无效的，用于管理)
func (s *AssetVulnService) ListAllPocsByVulnID(ctx context.Context, vulnID uint64, tagIDs []uint64) ([]*assetmodel.AssetVulnPoc, error) {
	var pocIDs []uint64
	if len(tagIDs) > 0 {
		entityIDsStr, err := s.tagService.GetEntityIDsByTagIDs(ctx, "vuln_poc", tagIDs)
		if err != nil {
			return nil, err
		}
		if len(entityIDsStr) == 0 {
			return []*assetmodel.AssetVulnPoc{}, nil
		}
		for _, idStr := range entityIDsStr {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			pocIDs = append(pocIDs, id)
		}
		if len(pocIDs) == 0 {
			return []*assetmodel.AssetVulnPoc{}, nil
		}
	}
	return s.repo.ListPocsByVulnID(ctx, vulnID, false, pocIDs)
}

// -----------------------------------------------------------------------------
// Tag Management
// -----------------------------------------------------------------------------

// AddVulnTag 为漏洞添加标签
func (s *AssetVulnService) AddVulnTag(ctx context.Context, vulnID uint64, tagID uint64) error {
	return s.tagService.AddEntityTag(ctx, "vuln", strconv.FormatUint(vulnID, 10), tagID, "manual", 0)
}

// RemoveVulnTag 为漏洞移除标签
func (s *AssetVulnService) RemoveVulnTag(ctx context.Context, vulnID uint64, tagID uint64) error {
	return s.tagService.RemoveEntityTag(ctx, "vuln", strconv.FormatUint(vulnID, 10), tagID)
}

// GetVulnTags 获取漏洞的标签
func (s *AssetVulnService) GetVulnTags(ctx context.Context, vulnID uint64) ([]tagsystem.SysTag, error) {
	entityTags, err := s.tagService.GetEntityTags(ctx, "vuln", strconv.FormatUint(vulnID, 10))
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

// AddPocTag 为PoC添加标签
func (s *AssetVulnService) AddPocTag(ctx context.Context, pocID uint64, tagID uint64) error {
	return s.tagService.AddEntityTag(ctx, "vuln_poc", strconv.FormatUint(pocID, 10), tagID, "manual", 0)
}

// RemovePocTag 为PoC移除标签
func (s *AssetVulnService) RemovePocTag(ctx context.Context, pocID uint64, tagID uint64) error {
	return s.tagService.RemoveEntityTag(ctx, "vuln_poc", strconv.FormatUint(pocID, 10), tagID)
}

// GetPocTags 获取PoC的标签
func (s *AssetVulnService) GetPocTags(ctx context.Context, pocID uint64) ([]tagsystem.SysTag, error) {
	entityTags, err := s.tagService.GetEntityTags(ctx, "vuln_poc", strconv.FormatUint(pocID, 10))
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
