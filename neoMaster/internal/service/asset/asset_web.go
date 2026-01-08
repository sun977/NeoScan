package asset

import (
	"context"
	"errors"
	assetmodel "neomaster/internal/model/asset"
	tagsystem "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
	tagservice "neomaster/internal/service/tag_system"
	"strconv"
)

// AssetWebService Web资产服务层
// 处理Web服务及其详细信息的业务逻辑
type AssetWebService struct {
	repo       *assetrepo.AssetWebRepository
	tagService tagservice.TagService
}

// NewAssetWebService 创建 AssetWebService 实例
func NewAssetWebService(repo *assetrepo.AssetWebRepository, tagService tagservice.TagService) *AssetWebService {
	return &AssetWebService{
		repo:       repo,
		tagService: tagService,
	}
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
func (s *AssetWebService) ListWebs(ctx context.Context, page, pageSize int, domain string, assetType string, status string, tagIDs []uint64) ([]*assetmodel.AssetWeb, int64, error) {
	var webIDs []uint64

	// 如果指定了标签，先从标签系统获取对应的 WebID 列表
	if len(tagIDs) > 0 {
		entityIDsStr, err := s.tagService.GetEntityIDsByTagIDs(ctx, "web", tagIDs)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "list_webs_get_tags", "SERVICE", map[string]interface{}{
				"operation": "list_webs_get_tags",
				"tag_ids":   tagIDs,
			})
			return nil, 0, err
		}

		if len(entityIDsStr) == 0 {
			// 筛选了标签但没找到对应的资源，直接返回空列表
			return []*assetmodel.AssetWeb{}, 0, nil
		}

		// 转换 ID 类型
		for _, idStr := range entityIDsStr {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				continue
			}
			webIDs = append(webIDs, id)
		}

		if len(webIDs) == 0 {
			return []*assetmodel.AssetWeb{}, 0, nil
		}
	}

	return s.repo.ListWebs(ctx, page, pageSize, domain, assetType, status, webIDs)
}

// AddTagToWeb 添加标签到Web资产
func (s *AssetWebService) AddTagToWeb(ctx context.Context, webID uint64, tagID uint64) error {
	// 检查Web资产是否存在
	web, err := s.repo.GetWebByID(ctx, webID)
	if err != nil {
		return err
	}
	if web == nil {
		return errors.New("web asset not found")
	}

	// 添加标签 (Source=manual)
	err = s.tagService.AddEntityTag(ctx, "web", strconv.FormatUint(webID, 10), tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_web", "SERVICE", map[string]interface{}{
			"operation": "add_tag_to_web",
			"web_id":    webID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromWeb 从Web资产移除标签
func (s *AssetWebService) RemoveTagFromWeb(ctx context.Context, webID uint64, tagID uint64) error {
	// 检查Web资产是否存在
	web, err := s.repo.GetWebByID(ctx, webID)
	if err != nil {
		return err
	}
	if web == nil {
		return errors.New("web asset not found")
	}

	err = s.tagService.RemoveEntityTag(ctx, "web", strconv.FormatUint(webID, 10), tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_web", "SERVICE", map[string]interface{}{
			"operation": "remove_tag_from_web",
			"web_id":    webID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

// GetWebTags 获取Web资产标签
func (s *AssetWebService) GetWebTags(ctx context.Context, webID uint64) ([]tagsystem.SysTag, error) {
	// 检查Web资产是否存在
	web, err := s.repo.GetWebByID(ctx, webID)
	if err != nil {
		return nil, err
	}
	if web == nil {
		return nil, errors.New("web asset not found")
	}

	// 1. 获取实体关联关系
	entityTags, err := s.tagService.GetEntityTags(ctx, "web", strconv.FormatUint(webID, 10))
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_web_tags", "SERVICE", map[string]interface{}{
			"operation": "get_web_tags",
			"web_id":    webID,
		})
		return nil, err
	}

	if len(entityTags) == 0 {
		return []tagsystem.SysTag{}, nil
	}

	// 2. 提取TagIDs
	var tagIDs []uint64
	for _, et := range entityTags {
		tagIDs = append(tagIDs, et.TagID)
	}

	// 3. 批量获取标签详情
	tags, err := s.tagService.GetTagsByIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	return tags, nil
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
