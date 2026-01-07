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

	// 4. 处理 WebDetails
	if len(bundle.WebDetails) > 0 {
		if err := m.upsertWebDetails(ctx, bundle.WebDetails); err != nil {
			return fmt.Errorf("failed to upsert web details: %w", err)
		}
	}

	// TODO: 处理 Vulns

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
	for _, web := range webs {
		web.HostID = hostID // 关联主机
		existing, err := m.webRepo.GetWebByURL(ctx, web.URL)
		if err != nil {
			return fmt.Errorf("check web existence failed: %w", err)
		}

		now := time.Now()
		if existing != nil {
			// Update
			existing.LastSeenAt = &now
			if web.Domain != "" {
				existing.Domain = web.Domain
			}
			if web.TechStack != "" && web.TechStack != "{}" {
				existing.TechStack = web.TechStack
			}
			if web.BasicInfo != "" && web.BasicInfo != "{}" {
				existing.BasicInfo = web.BasicInfo
			}
			// Update Status/Tags if needed

			if err := m.webRepo.UpdateWeb(ctx, existing); err != nil {
				return fmt.Errorf("update web failed: %w", err)
			}
			// 如果有关联的 Detail，需要更新其 AssetWebID
			// 这里假设 bundle.WebDetails 中的 AssetWebID 是空的或者临时的
			// 实际上 Mapper 应该处理好，或者我们在这里通过 URL 匹配
		} else {
			// Create
			if web.LastSeenAt == nil {
				web.LastSeenAt = &now
			}
			if err := m.webRepo.CreateWeb(ctx, web); err != nil {
				return fmt.Errorf("create web failed: %w", err)
			}
		}
	}
	return nil
}

// upsertWebDetails 更新或插入 Web 详细信息
func (m *assetMerger) upsertWebDetails(ctx context.Context, details []*assetModel.AssetWebDetail) error {
	for _, detail := range details {
		// 注意: AssetWebDetail 依赖 AssetWebID。
		// 在 Bundle 中，Detail 必须能关联到 Web。
		// 如果 Web 是新创建的，ID 在 upsertWebs 之后才生成。
		// 这意味着 upsertWebs 应该返回 ID 映射，或者 Bundle 结构需要调整以支持嵌套。
		// 简单方案: 假设 Mapper 已经尽力填充了，或者我们再次查询 URL 获取 ID。
		// 由于 Detail 通常不包含 URL 字段，我们很难反向查找。
		// 最佳实践: 在 Bundle 中，Web 和 Detail 应该是一对一关联的，例如 AssetWeb 包含 Detail 指针。
		// 但目前 Bundle 是扁平列表。
		// 妥协: 假设 Detail.AssetWebID 已经设置 (如果是基于已有 Web 的爬虫任务)。
		// 如果是发现任务同时产生 Web 和 Detail，我们需要更复杂的逻辑。

		if detail.AssetWebID == 0 {
			continue // Skip orphan details
		}

		if err := m.webRepo.CreateOrUpdateDetail(ctx, detail); err != nil {
			return fmt.Errorf("upsert web detail failed: %w", err)
		}
	}
	return nil
}
