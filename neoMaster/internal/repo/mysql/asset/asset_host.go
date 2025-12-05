package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// AssetHostRepository 资产主机仓库
// 负责 AssetHost 和 AssetService 的数据访问
type AssetHostRepository struct {
	db *gorm.DB
}

// NewAssetHostRepository 创建 AssetHostRepository 实例
func NewAssetHostRepository(db *gorm.DB) *AssetHostRepository {
	return &AssetHostRepository{db: db}
}

// -----------------------------------------------------------------------------
// AssetHost (主机资产) CRUD
// -----------------------------------------------------------------------------

// CreateHost 创建主机
func (r *AssetHostRepository) CreateHost(ctx context.Context, host *asset.AssetHost) error {
	if host == nil {
		return errors.New("host is nil")
	}
	err := r.db.WithContext(ctx).Create(host).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_host", "REPO", map[string]interface{}{
			"operation": "create_host",
			"ip":        host.IP,
		})
		return err
	}
	return nil
}

// GetHostByID 根据ID获取主机
func (r *AssetHostRepository) GetHostByID(ctx context.Context, id uint64) (*asset.AssetHost, error) {
	var host asset.AssetHost
	err := r.db.WithContext(ctx).First(&host, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_host_by_id", "REPO", map[string]interface{}{
			"operation": "get_host_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &host, nil
}

// GetHostByIP 根据IP获取主机
func (r *AssetHostRepository) GetHostByIP(ctx context.Context, ip string) (*asset.AssetHost, error) {
	var host asset.AssetHost
	err := r.db.WithContext(ctx).Where("ip = ?", ip).First(&host).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_host_by_ip", "REPO", map[string]interface{}{
			"operation": "get_host_by_ip",
			"ip":        ip,
		})
		return nil, err
	}
	return &host, nil
}

// UpdateHost 更新主机
func (r *AssetHostRepository) UpdateHost(ctx context.Context, host *asset.AssetHost) error {
	if host == nil || host.ID == 0 {
		return errors.New("invalid host or id")
	}
	// 使用 Updates 而不是 Save，以支持部分更新并避免覆盖 CreatedAt 等字段
	err := r.db.WithContext(ctx).Model(host).Updates(host).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_host", "REPO", map[string]interface{}{
			"operation": "update_host",
			"id":        host.ID,
		})
		return err
	}
	return nil
}

// DeleteHost 删除主机 (软删除)
func (r *AssetHostRepository) DeleteHost(ctx context.Context, id uint64) error {
	// 开启事务，因为删除主机通常也应该处理关联的服务(虽然软删除可能保留，但根据业务需求可能需要级联)
	// 这里暂时只删除主机本身，依赖GORM的软删除机制
	err := r.db.WithContext(ctx).Delete(&asset.AssetHost{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_host", "REPO", map[string]interface{}{
			"operation": "delete_host",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListHosts 获取主机列表 (分页 + 筛选)
func (r *AssetHostRepository) ListHosts(ctx context.Context, page, pageSize int, ip, hostname, os string) ([]*asset.AssetHost, int64, error) {
	var hosts []*asset.AssetHost
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetHost{})

	if ip != "" {
		query = query.Where("ip LIKE ?", "%"+ip+"%")
	}
	if hostname != "" {
		query = query.Where("hostname LIKE ?", "%"+hostname+"%")
	}
	if os != "" {
		query = query.Where("os LIKE ?", "%"+os+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_hosts_count", "REPO", map[string]interface{}{
			"operation": "list_hosts_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&hosts).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_hosts_find", "REPO", map[string]interface{}{
			"operation": "list_hosts_find",
		})
		return nil, 0, err
	}

	return hosts, total, nil
}

// -----------------------------------------------------------------------------
// AssetService (服务资产) CRUD
// -----------------------------------------------------------------------------

// CreateService 创建服务
func (r *AssetHostRepository) CreateService(ctx context.Context, service *asset.AssetService) error {
	if service == nil {
		return errors.New("service is nil")
	}
	err := r.db.WithContext(ctx).Create(service).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_service", "REPO", map[string]interface{}{
			"operation": "create_service",
			"host_id":   service.HostID,
			"port":      service.Port,
		})
		return err
	}
	return nil
}

// GetServiceByID 根据ID获取服务
func (r *AssetHostRepository) GetServiceByID(ctx context.Context, id uint64) (*asset.AssetService, error) {
	var service asset.AssetService
	err := r.db.WithContext(ctx).First(&service, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_service_by_id", "REPO", map[string]interface{}{
			"operation": "get_service_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &service, nil
}

// UpdateService 更新服务
func (r *AssetHostRepository) UpdateService(ctx context.Context, service *asset.AssetService) error {
	if service == nil || service.ID == 0 {
		return errors.New("invalid service or id")
	}
	err := r.db.WithContext(ctx).Save(service).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_service", "REPO", map[string]interface{}{
			"operation": "update_service",
			"id":        service.ID,
		})
		return err
	}
	return nil
}

// DeleteService 删除服务
func (r *AssetHostRepository) DeleteService(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&asset.AssetService{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_service", "REPO", map[string]interface{}{
			"operation": "delete_service",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListServicesByHostID 获取指定主机的服务列表
func (r *AssetHostRepository) ListServicesByHostID(ctx context.Context, hostID uint64) ([]*asset.AssetService, error) {
	var services []*asset.AssetService
	err := r.db.WithContext(ctx).Where("host_id = ?", hostID).Order("port asc").Find(&services).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_services_by_host", "REPO", map[string]interface{}{
			"operation": "list_services_by_host",
			"host_id":   hostID,
		})
		return nil, err
	}
	return services, nil
}

// ListServices 获取服务列表 (分页 + 筛选)
func (r *AssetHostRepository) ListServices(ctx context.Context, page, pageSize int, port int, name, proto string) ([]*asset.AssetService, int64, error) {
	var services []*asset.AssetService
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetService{})

	if port > 0 {
		query = query.Where("port = ?", port)
	}
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if proto != "" {
		query = query.Where("proto = ?", proto)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_services_count", "REPO", map[string]interface{}{
			"operation": "list_services_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&services).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_services_find", "REPO", map[string]interface{}{
			"operation": "list_services_find",
		})
		return nil, 0, err
	}

	return services, total, nil
}
