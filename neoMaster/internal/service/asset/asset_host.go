package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	tagsystem "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	assetrepo "neomaster/internal/repo/mysql/asset"
	tagservice "neomaster/internal/service/tag_system"
	"strconv"
)

// // AssetHostServiceInterface 资产主机服务接口
// // 定义资产主机服务的方法，用于处理主机和服务资产的业务逻辑
// type AssetHostServiceInterface interface {
// 	CreateHost(ctx context.Context, host *asset.AssetHost) error
// 	GetHost(ctx context.Context, id uint64) (*asset.AssetHost, error)
// 	UpdateHost(ctx context.Context, host *asset.AssetHost) error
// }

// AssetHostService 资产主机服务
// 负责处理主机和服务资产的业务逻辑
type AssetHostService struct {
	repo       *assetrepo.AssetHostRepository
	tagService tagservice.TagService
}

// NewAssetHostService 创建 AssetHostService 实例
func NewAssetHostService(repo *assetrepo.AssetHostRepository, tagService tagservice.TagService) *AssetHostService {
	return &AssetHostService{
		repo:       repo,
		tagService: tagService,
	}
}

// -----------------------------------------------------------------------------
// AssetHost 业务逻辑
// -----------------------------------------------------------------------------

// CreateHost 创建主机
func (s *AssetHostService) CreateHost(ctx context.Context, host *asset.AssetHost) error {
	if host == nil {
		return errors.New("host data cannot be nil")
	}
	// 检查IP是否已存在
	existing, err := s.repo.GetHostByIP(ctx, host.IP)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("host with this IP already exists")
	}

	err = s.repo.CreateHost(ctx, host)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_host", "SERVICE", map[string]interface{}{
			"operation": "create_host",
			"ip":        host.IP,
		})
		return err
	}
	return nil
}

// GetHost 获取主机详情
func (s *AssetHostService) GetHost(ctx context.Context, id uint64) (*asset.AssetHost, error) {
	host, err := s.repo.GetHostByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_host", "SERVICE", map[string]interface{}{
			"operation": "get_host",
			"id":        id,
		})
		return nil, err
	}
	if host == nil {
		return nil, errors.New("host not found")
	}
	return host, nil
}

// UpdateHost 更新主机
func (s *AssetHostService) UpdateHost(ctx context.Context, host *asset.AssetHost) error {
	// 检查是否存在
	existing, err := s.repo.GetHostByID(ctx, host.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("host not found")
	}

	err = s.repo.UpdateHost(ctx, host)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_host", "SERVICE", map[string]interface{}{
			"operation": "update_host",
			"id":        host.ID,
		})
		return err
	}
	return nil
}

// DeleteHost 删除主机
func (s *AssetHostService) DeleteHost(ctx context.Context, id uint64) error {
	// 检查是否存在
	existing, err := s.repo.GetHostByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("host not found")
	}

	err = s.repo.DeleteHost(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_host", "SERVICE", map[string]interface{}{
			"operation": "delete_host",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListHosts 获取主机列表
func (s *AssetHostService) ListHosts(ctx context.Context, page, pageSize int, ip, hostname, os string) ([]*asset.AssetHost, int64, error) {
	list, total, err := s.repo.ListHosts(ctx, page, pageSize, ip, hostname, os)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_hosts", "SERVICE", map[string]interface{}{
			"operation": "list_hosts",
			"page":      page,
			"page_size": pageSize,
		})
		return nil, 0, err
	}
	return list, total, nil
}

// AddTagToHost 添加标签到主机

// RemoveTagFromHost 从主机移除标签

// GetHostTags 获取主机标签

// -----------------------------------------------------------------------------
// AssetService (服务资产) 业务逻辑
// -----------------------------------------------------------------------------

// CreateService 创建服务
func (s *AssetHostService) CreateService(ctx context.Context, service *asset.AssetService) error {
	if service == nil {
		return errors.New("service data cannot be nil")
	}
	// 校验关联的主机是否存在
	host, err := s.repo.GetHostByID(ctx, service.HostID)
	if err != nil {
		return err
	}
	if host == nil {
		return errors.New("associated host not found")
	}

	err = s.repo.CreateService(ctx, service)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_service", "SERVICE", map[string]interface{}{
			"operation": "create_service",
			"host_id":   service.HostID,
			"port":      service.Port,
		})
		return err
	}
	return nil
}

// GetService 获取服务详情
func (s *AssetHostService) GetService(ctx context.Context, id uint64) (*asset.AssetService, error) {
	service, err := s.repo.GetServiceByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_service", "SERVICE", map[string]interface{}{
			"operation": "get_service",
			"id":        id,
		})
		return nil, err
	}
	if service == nil {
		return nil, errors.New("service not found")
	}
	return service, nil
}

// UpdateService 更新服务
func (s *AssetHostService) UpdateService(ctx context.Context, service *asset.AssetService) error {
	// 检查是否存在
	existing, err := s.repo.GetServiceByID(ctx, service.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("service not found")
	}

	err = s.repo.UpdateService(ctx, service)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_service", "SERVICE", map[string]interface{}{
			"operation": "update_service",
			"id":        service.ID,
		})
		return err
	}
	return nil
}

// DeleteService 删除服务
func (s *AssetHostService) DeleteService(ctx context.Context, id uint64) error {
	// 检查是否存在
	existing, err := s.repo.GetServiceByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("service not found")
	}

	err = s.repo.DeleteService(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_service", "SERVICE", map[string]interface{}{
			"operation": "delete_service",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListServicesByHostID 获取指定主机的服务列表
func (s *AssetHostService) ListServicesByHostID(ctx context.Context, hostID uint64) ([]*asset.AssetService, error) {
	// 校验主机是否存在
	host, err := s.repo.GetHostByID(ctx, hostID)
	if err != nil {
		return nil, err
	}
	if host == nil {
		return nil, errors.New("host not found")
	}

	list, err := s.repo.ListServicesByHostID(ctx, hostID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_services_by_host", "SERVICE", map[string]interface{}{
			"operation": "list_services_by_host",
			"host_id":   hostID,
		})
		return nil, err
	}
	return list, nil
}

// ListServices 获取服务列表
func (s *AssetHostService) ListServices(ctx context.Context, page, pageSize int, port int, name, proto string) ([]*asset.AssetService, int64, error) {
	list, total, err := s.repo.ListServices(ctx, page, pageSize, port, name, proto)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_services", "SERVICE", map[string]interface{}{
			"operation": "list_services",
			"page":      page,
			"page_size": pageSize,
		})
		return nil, 0, err
	}
	return list, total, nil
}

// AddTagToService 添加标签到服务
func (s *AssetHostService) AddTagToService(ctx context.Context, serviceID uint64, tagID uint64) error {
	// 检查服务是否存在
	service, err := s.repo.GetServiceByID(ctx, serviceID)
	if err != nil {
		return err
	}
	if service == nil {
		return errors.New("service not found")
	}

	// 添加标签 (Source=manual)
	// 注意：entityType 使用 "service"
	err = s.tagService.AddEntityTag(ctx, "service", strconv.FormatUint(serviceID, 10), tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_service", "SERVICE", map[string]interface{}{
			"operation":  "add_tag_to_service",
			"service_id": serviceID,
			"tag_id":     tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromService 从服务移除标签
func (s *AssetHostService) RemoveTagFromService(ctx context.Context, serviceID uint64, tagID uint64) error {
	// 检查服务是否存在
	service, err := s.repo.GetServiceByID(ctx, serviceID)
	if err != nil {
		return err
	}
	if service == nil {
		return errors.New("service not found")
	}

	err = s.tagService.RemoveEntityTag(ctx, "service", strconv.FormatUint(serviceID, 10), tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_service", "SERVICE", map[string]interface{}{
			"operation":  "remove_tag_from_service",
			"service_id": serviceID,
			"tag_id":     tagID,
		})
		return err
	}
	return nil
}

// GetServiceTags 获取服务标签
func (s *AssetHostService) GetServiceTags(ctx context.Context, serviceID uint64) ([]tagsystem.SysTag, error) {
	// 检查服务是否存在
	service, err := s.repo.GetServiceByID(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, errors.New("service not found")
	}

	// 1. 获取实体关联关系
	entityTags, err := s.tagService.GetEntityTags(ctx, "service", strconv.FormatUint(serviceID, 10))
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_service_tags", "SERVICE", map[string]interface{}{
			"operation":  "get_service_tags",
			"service_id": serviceID,
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
