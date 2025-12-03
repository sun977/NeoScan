package asset

import (
	"context"
	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
)

// AssetVulnService 漏洞资产服务层
// 处理漏洞及其PoC的业务逻辑
type AssetVulnService struct {
	repo *assetrepo.AssetVulnRepository
}

// NewAssetVulnService 创建 AssetVulnService 实例
func NewAssetVulnService(repo *assetrepo.AssetVulnRepository) *AssetVulnService {
	return &AssetVulnService{repo: repo}
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
func (s *AssetVulnService) ListVulns(ctx context.Context, page, pageSize int, targetType string, targetRefID uint64, status string, severity string) ([]*assetmodel.AssetVuln, int64, error) {
	return s.repo.ListVulns(ctx, page, pageSize, targetType, targetRefID, status, severity)
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
	return s.repo.ListPocsByVulnID(ctx, vulnID, true)
}

// ListAllPocsByVulnID 获取指定漏洞的所有PoC (包括无效的，用于管理)
func (s *AssetVulnService) ListAllPocsByVulnID(ctx context.Context, vulnID uint64) ([]*assetmodel.AssetVulnPoc, error) {
	return s.repo.ListPocsByVulnID(ctx, vulnID, false)
}
