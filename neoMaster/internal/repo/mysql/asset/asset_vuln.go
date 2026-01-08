package asset

import (
	"context"
	"errors"
	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetVulnRepository 漏洞资产仓库
// 负责 AssetVuln 和 AssetVulnPoc 的数据访问
type AssetVulnRepository struct {
	db *gorm.DB
}

// NewAssetVulnRepository 创建 AssetVulnRepository 实例
func NewAssetVulnRepository(db *gorm.DB) *AssetVulnRepository {
	return &AssetVulnRepository{db: db}
}

// -----------------------------------------------------------------------------
// AssetVuln (漏洞资产) CRUD
// -----------------------------------------------------------------------------

// CreateVuln 创建漏洞记录
func (r *AssetVulnRepository) CreateVuln(ctx context.Context, vuln *assetmodel.AssetVuln) error {
	if vuln == nil {
		return errors.New("vuln is nil")
	}
	err := r.db.WithContext(ctx).Create(vuln).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_vuln", "REPO", map[string]interface{}{
			"operation": "create_vuln",
			"target":    vuln.TargetRefID,
			"cve":       vuln.CVE,
		})
		return err
	}
	return nil
}

// GetVulnByID 根据ID获取漏洞记录
func (r *AssetVulnRepository) GetVulnByID(ctx context.Context, id uint64) (*assetmodel.AssetVuln, error) {
	var vuln assetmodel.AssetVuln
	err := r.db.WithContext(ctx).First(&vuln, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_vuln_by_id", "REPO", map[string]interface{}{
			"operation": "get_vuln_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &vuln, nil
}

// UpdateVuln 更新漏洞记录
func (r *AssetVulnRepository) UpdateVuln(ctx context.Context, vuln *assetmodel.AssetVuln) error {
	if vuln == nil || vuln.ID == 0 {
		return errors.New("invalid vuln or id")
	}
	// 使用 Updates 而不是 Save，以支持部分更新并避免覆盖 CreatedAt 等字段
	err := r.db.WithContext(ctx).Model(vuln).Updates(vuln).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_vuln", "REPO", map[string]interface{}{
			"operation": "update_vuln",
			"id":        vuln.ID,
		})
		return err
	}
	return nil
}

// DeleteVuln 删除漏洞记录
func (r *AssetVulnRepository) DeleteVuln(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&assetmodel.AssetVuln{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_vuln", "REPO", map[string]interface{}{
			"operation": "delete_vuln",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListVulns 获取漏洞列表 (支持分页和多条件筛选)
func (r *AssetVulnRepository) ListVulns(ctx context.Context, page, pageSize int, targetType string, targetRefID uint64, status string, severity string, vulnIDs []uint64) ([]*assetmodel.AssetVuln, int64, error) {
	var vulns []*assetmodel.AssetVuln
	var total int64

	query := r.db.WithContext(ctx).Model(&assetmodel.AssetVuln{})

	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if targetRefID > 0 {
		query = query.Where("target_ref_id = ?", targetRefID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if len(vulnIDs) > 0 {
		query = query.Where("id IN ?", vulnIDs)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_vulns_count", "REPO", map[string]interface{}{
			"operation": "list_vulns_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&vulns).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_vulns_find", "REPO", map[string]interface{}{
			"operation": "list_vulns_find",
		})
		return nil, 0, err
	}

	return vulns, total, nil
}

// -----------------------------------------------------------------------------
// AssetVulnPoc (漏洞PoC) CRUD
// -----------------------------------------------------------------------------

// CreatePoc 创建PoC记录
func (r *AssetVulnRepository) CreatePoc(ctx context.Context, poc *assetmodel.AssetVulnPoc) error {
	if poc == nil {
		return errors.New("poc is nil")
	}
	err := r.db.WithContext(ctx).Create(poc).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_poc", "REPO", map[string]interface{}{
			"operation": "create_poc",
			"vuln_id":   poc.VulnID,
			"name":      poc.Name,
		})
		return err
	}
	return nil
}

// GetPocByID 根据ID获取PoC
func (r *AssetVulnRepository) GetPocByID(ctx context.Context, id uint64) (*assetmodel.AssetVulnPoc, error) {
	var poc assetmodel.AssetVulnPoc
	err := r.db.WithContext(ctx).First(&poc, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_poc_by_id", "REPO", map[string]interface{}{
			"operation": "get_poc_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &poc, nil
}

// UpdatePoc 更新PoC记录
func (r *AssetVulnRepository) UpdatePoc(ctx context.Context, poc *assetmodel.AssetVulnPoc) error {
	if poc == nil || poc.ID == 0 {
		return errors.New("invalid poc or id")
	}
	// 使用 Updates 而不是 Save，以支持部分更新并避免覆盖 CreatedAt 等字段
	err := r.db.WithContext(ctx).Model(poc).Updates(poc).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_poc", "REPO", map[string]interface{}{
			"operation": "update_poc",
			"id":        poc.ID,
		})
		return err
	}
	return nil
}

// DeletePoc 删除PoC记录
func (r *AssetVulnRepository) DeletePoc(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&assetmodel.AssetVulnPoc{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_poc", "REPO", map[string]interface{}{
			"operation": "delete_poc",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListPocsByVulnID 获取指定漏洞的所有PoC (支持按优先级排序)
func (r *AssetVulnRepository) ListPocsByVulnID(ctx context.Context, vulnID uint64, onlyValid bool, pocIDs []uint64) ([]*assetmodel.AssetVulnPoc, error) {
	var pocs []*assetmodel.AssetVulnPoc
	query := r.db.WithContext(ctx).Where("vuln_id = ?", vulnID)

	if onlyValid {
		query = query.Where("is_valid = ?", true)
	}
	if len(pocIDs) > 0 {
		query = query.Where("id IN ?", pocIDs)
	}

	// 按优先级升序 (越小越优先)，如果优先级相同则按ID降序
	err := query.Order("priority asc, id desc").Find(&pocs).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_pocs_by_vuln_id", "REPO", map[string]interface{}{
			"operation": "list_pocs_by_vuln_id",
			"vuln_id":   vulnID,
		})
		return nil, err
	}
	return pocs, nil
}
