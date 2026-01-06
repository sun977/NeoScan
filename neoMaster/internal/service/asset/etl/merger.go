// AssetMerger 资产合并器接口
// 职责: 实现 StageResult -> Asset 的 Upsert 逻辑
// 核心逻辑:
// - 发现 IP -> 查找或创建 AssetHost
// - 发现 Port -> 更新 AssetPort
// - 发现 Service -> 更新 AssetService
// - 发现 Web -> 更新 AssetWeb
package etl

import (
	"context"
	"fmt"
	"time"

	assetModel "neomaster/internal/model/asset"
	assetRepo "neomaster/internal/repo/mysql/asset"
)

// AssetMerger 资产合并器接口
type AssetMerger interface {
	// Merge 将资产包合并到数据库
	Merge(ctx context.Context, bundle *AssetBundle) error
}

// assetMerger 默认实现
type assetMerger struct {
	hostRepo *assetRepo.AssetHostRepository
	webRepo  *assetRepo.AssetWebRepository
}

// NewAssetMerger 创建资产合并器
func NewAssetMerger(
	hostRepo *assetRepo.AssetHostRepository,
	webRepo *assetRepo.AssetWebRepository,
) AssetMerger {
	return &assetMerger{
		hostRepo: hostRepo,
		webRepo:  webRepo,
	}
}

// Merge 将资产包合并到数据库
func (m *assetMerger) Merge(ctx context.Context, bundle *AssetBundle) error {
	if bundle == nil {
		return nil
	}

	// 1. 处理 Host (必选)
	if bundle.Host == nil {
		return fmt.Errorf("missing host info in bundle")
	}

	// Upsert Host
	hostID, err := m.upsertHost(ctx, bundle.Host)
	if err != nil {
		return fmt.Errorf("failed to upsert host: %w", err)
	}

	// 2. 处理 Services
	if len(bundle.Services) > 0 {
		if err := m.upsertServices(ctx, hostID, bundle.Services); err != nil {
			return fmt.Errorf("failed to upsert services: %w", err)
		}
	}

	// 3. 处理 Webs
	if len(bundle.Webs) > 0 {
		if err := m.upsertWebs(ctx, hostID, bundle.Webs); err != nil {
			return fmt.Errorf("failed to upsert webs: %w", err)
		}
	}

	// TODO: 处理 WebDetails 和 Vulns

	return nil
}

// upsertHost 更新或插入主机
func (m *assetMerger) upsertHost(ctx context.Context, host *assetModel.AssetHost) (uint64, error) {
	existing, err := m.hostRepo.GetHostByIP(ctx, host.IP)
	if err != nil {
		return 0, fmt.Errorf("check host existence failed: %w", err)
	}

	now := time.Now()
	if existing != nil {
		// Update
		existing.LastSeenAt = &now
		if host.Hostname != "" {
			existing.Hostname = host.Hostname
		}
		if host.OS != "" {
			existing.OS = host.OS
		}
		// TODO: 合并 Tags 和 SourceStageIDs

		if err := m.hostRepo.UpdateHost(ctx, existing); err != nil {
			return 0, fmt.Errorf("update host failed: %w", err)
		}
		return existing.ID, nil
	}

	// Create
	if host.LastSeenAt == nil {
		host.LastSeenAt = &now
	}
	if err := m.hostRepo.CreateHost(ctx, host); err != nil {
		return 0, fmt.Errorf("create host failed: %w", err)
	}
	return host.ID, nil
}

// upsertServices 更新或插入服务列表
func (m *assetMerger) upsertServices(ctx context.Context, hostID uint64, services []*assetModel.AssetService) error {
	for _, svc := range services {
		svc.HostID = hostID
		existing, err := m.hostRepo.GetServiceByHostIDAndPort(ctx, hostID, svc.Port, svc.Proto)
		if err != nil {
			return fmt.Errorf("check service existence failed: %w", err)
		}

		now := time.Now()
		if existing != nil {
			// Update
			existing.LastSeenAt = &now
			if svc.Name != "" {
				existing.Name = svc.Name
			}
			if svc.Version != "" {
				existing.Version = svc.Version
			}
			if svc.CPE != "" {
				existing.CPE = svc.CPE
			}
			// Product field is not present in AssetService model, Name covers it.
			if svc.Fingerprint != "" {
				existing.Fingerprint = svc.Fingerprint
			}

			if err := m.hostRepo.UpdateService(ctx, existing); err != nil {
				return fmt.Errorf("update service failed: %w", err)
			}
		} else {
			// Create
			if svc.LastSeenAt == nil {
				svc.LastSeenAt = &now
			}
			if err := m.hostRepo.CreateService(ctx, svc); err != nil {
				return fmt.Errorf("create service failed: %w", err)
			}
		}
	}
	return nil
}

// upsertWebs 更新或插入 Web 站点列表
func (m *assetMerger) upsertWebs(ctx context.Context, hostID uint64, webs []*assetModel.AssetWeb) error {
	// TODO: 实现 Web Upsert 逻辑
	// 暂时留空，等待 Web Repo 方法确认
	return nil
}
