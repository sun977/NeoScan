package asset

import (
	"context"
	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
)

// AssetWebService Web资产服务层
// 处理Web服务及其详细信息的业务逻辑
type AssetWebService struct {
	repo *assetrepo.AssetWebRepository
}

// NewAssetWebService 创建 AssetWebService 实例
func NewAssetWebService(repo *assetrepo.AssetWebRepository) *AssetWebService {
	return &AssetWebService{repo: repo}
}

// -----------------------------------------------------------------------------
// AssetWeb 业务逻辑
// -----------------------------------------------------------------------------

// CreateWeb 创建Web资产
func (s *AssetWebService) CreateWeb(ctx context.Context, web *assetmodel.AssetWeb) error {
	err := s.repo.CreateWeb(ctx, web)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.web.CreateWeb", "SERVICE", map[string]interface{}{
			"url": web.URL,
		})
		return err
	}
	return nil
}

// GetWebByID 根据ID获取Web资产
func (s *AssetWebService) GetWebByID(ctx context.Context, id uint64) (*assetmodel.AssetWeb, error) {
	return s.repo.GetWebByID(ctx, id)
}

// GetWebByURL 根据URL获取Web资产
func (s *AssetWebService) GetWebByURL(ctx context.Context, url string) (*assetmodel.AssetWeb, error) {
	return s.repo.GetWebByURL(ctx, url)
}

// UpdateWeb 更新Web资产
func (s *AssetWebService) UpdateWeb(ctx context.Context, web *assetmodel.AssetWeb) error {
	err := s.repo.UpdateWeb(ctx, web)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.web.UpdateWeb", "SERVICE", map[string]interface{}{
			"id": web.ID,
		})
		return err
	}
	return nil
}

// DeleteWeb 删除Web资产
func (s *AssetWebService) DeleteWeb(ctx context.Context, id uint64) error {
	err := s.repo.DeleteWeb(ctx, id)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.web.DeleteWeb", "SERVICE", map[string]interface{}{
			"id": id,
		})
		return err
	}
	return nil
}

// ListWebs 获取Web资产列表
func (s *AssetWebService) ListWebs(ctx context.Context, page, pageSize int, domain string, assetType string, status string) ([]*assetmodel.AssetWeb, int64, error) {
	return s.repo.ListWebs(ctx, page, pageSize, domain, assetType, status)
}

// -----------------------------------------------------------------------------
// AssetWebDetail 业务逻辑
// -----------------------------------------------------------------------------

// SaveWebDetail 保存Web详细信息 (自动处理创建或更新)
func (s *AssetWebService) SaveWebDetail(ctx context.Context, detail *assetmodel.AssetWebDetail) error {
	err := s.repo.CreateOrUpdateDetail(ctx, detail)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.asset.web.SaveWebDetail", "SERVICE", map[string]interface{}{
			"web_id": detail.AssetWebID,
		})
		return err
	}
	return nil
}

// GetWebDetail 获取Web详细信息
func (s *AssetWebService) GetWebDetail(ctx context.Context, webID uint64) (*assetmodel.AssetWebDetail, error) {
	return s.repo.GetDetailByWebID(ctx, webID)
}
