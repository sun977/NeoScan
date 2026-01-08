package asset

import (
	"context"
	"errors"
	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetWebRepository Web资产仓库
// 负责 AssetWeb 和 AssetWebDetail 的数据访问
type AssetWebRepository struct {
	db *gorm.DB
}

// NewAssetWebRepository 创建 AssetWebRepository 实例
func NewAssetWebRepository(db *gorm.DB) *AssetWebRepository {
	return &AssetWebRepository{db: db}
}

// -----------------------------------------------------------------------------
// AssetWeb (Web资产主表) CRUD
// -----------------------------------------------------------------------------

// CreateWeb 创建Web资产
func (r *AssetWebRepository) CreateWeb(ctx context.Context, web *assetmodel.AssetWeb) error {
	if web == nil {
		return errors.New("web is nil")
	}
	err := r.db.WithContext(ctx).Create(web).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_web", "REPO", map[string]interface{}{
			"operation": "create_web",
			"url":       web.URL,
			"domain":    web.Domain,
		})
		return err
	}
	return nil
}

// GetWebByID 根据ID获取Web资产
func (r *AssetWebRepository) GetWebByID(ctx context.Context, id uint64) (*assetmodel.AssetWeb, error) {
	var web assetmodel.AssetWeb
	err := r.db.WithContext(ctx).First(&web, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_web_by_id", "REPO", map[string]interface{}{
			"operation": "get_web_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &web, nil
}

// GetWebByURL 根据URL获取Web资产 (精确匹配)
func (r *AssetWebRepository) GetWebByURL(ctx context.Context, url string) (*assetmodel.AssetWeb, error) {
	var web assetmodel.AssetWeb
	err := r.db.WithContext(ctx).Where("url = ?", url).First(&web).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_web_by_url", "REPO", map[string]interface{}{
			"operation": "get_web_by_url",
			"url":       url,
		})
		return nil, err
	}
	return &web, nil
}

// UpdateWeb 更新Web资产
func (r *AssetWebRepository) UpdateWeb(ctx context.Context, web *assetmodel.AssetWeb) error {
	if web == nil || web.ID == 0 {
		return errors.New("invalid web or id")
	}
	// 使用 Updates 而不是 Save，以支持部分更新并避免覆盖 CreatedAt 等字段
	err := r.db.WithContext(ctx).Model(web).Updates(web).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_web", "REPO", map[string]interface{}{
			"operation": "update_web",
			"id":        web.ID,
		})
		return err
	}
	return nil
}

// DeleteWeb 删除Web资产
func (r *AssetWebRepository) DeleteWeb(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&assetmodel.AssetWeb{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_web", "REPO", map[string]interface{}{
			"operation": "delete_web",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListWebs 获取Web资产列表 (分页 + 筛选)
func (r *AssetWebRepository) ListWebs(ctx context.Context, page, pageSize int, domain string, assetType string, status string, webIDs []uint64) ([]*assetmodel.AssetWeb, int64, error) {
	var webs []*assetmodel.AssetWeb
	var total int64

	query := r.db.WithContext(ctx).Model(&assetmodel.AssetWeb{})

	if domain != "" {
		query = query.Where("domain LIKE ?", "%"+domain+"%")
	}
	if assetType != "" {
		query = query.Where("asset_type = ?", assetType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if len(webIDs) > 0 {
		query = query.Where("id IN ?", webIDs)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_webs_count", "REPO", map[string]interface{}{
			"operation": "list_webs_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&webs).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_webs_find", "REPO", map[string]interface{}{
			"operation": "list_webs_find",
		})
		return nil, 0, err
	}

	return webs, total, nil
}

// -----------------------------------------------------------------------------
// AssetWebDetail (Web详细信息) CRUD
// -----------------------------------------------------------------------------

// CreateOrUpdateDetail 创建或更新Web详细信息 (One-to-One 关系)
func (r *AssetWebRepository) CreateOrUpdateDetail(ctx context.Context, detail *assetmodel.AssetWebDetail) error {
	if detail == nil || detail.AssetWebID == 0 {
		return errors.New("invalid detail or asset_web_id")
	}

	// 使用 Upsert 逻辑：如果存在则更新，不存在则插入
	// MySQL: ON DUPLICATE KEY UPDATE
	// GORM Clause: OnConflict
	// 这里假设 asset_web_id 是唯一索引
	var existing assetmodel.AssetWebDetail
	err := r.db.WithContext(ctx).Where("asset_web_id = ?", detail.AssetWebID).First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不存在，创建
			return r.db.WithContext(ctx).Create(detail).Error
		}
		return err
	}

	// 存在，更新
	detail.ID = existing.ID
	// 使用 Updates 而不是 Save，以支持部分更新并避免覆盖 CreatedAt 等字段
	return r.db.WithContext(ctx).Model(detail).Updates(detail).Error
}

// GetDetailByWebID 根据WebID获取详细信息
func (r *AssetWebRepository) GetDetailByWebID(ctx context.Context, webID uint64) (*assetmodel.AssetWebDetail, error) {
	var detail assetmodel.AssetWebDetail
	err := r.db.WithContext(ctx).Where("asset_web_id = ?", webID).First(&detail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_detail_by_web_id", "REPO", map[string]interface{}{
			"operation": "get_detail_by_web_id",
			"web_id":    webID,
		})
		return nil, err
	}
	return &detail, nil
}
