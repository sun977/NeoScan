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
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
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
	hostRepo    *assetRepo.AssetHostRepository
	webRepo     *assetRepo.AssetWebRepository
	vulnRepo    *assetRepo.AssetVulnRepository
	unifiedRepo *assetRepo.AssetUnifiedRepository
}

// NewAssetMerger 创建资产合并器
func NewAssetMerger(
	hostRepo *assetRepo.AssetHostRepository,
	webRepo *assetRepo.AssetWebRepository,
	vulnRepo *assetRepo.AssetVulnRepository,
	unifiedRepo *assetRepo.AssetUnifiedRepository,
) AssetMerger {
	return &assetMerger{
		hostRepo:    hostRepo,
		webRepo:     webRepo,
		vulnRepo:    vulnRepo,
		unifiedRepo: unifiedRepo,
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

	// 3. 处理 WebAssets
	if len(bundle.WebAssets) > 0 {
		if err := m.upsertWebAssets(ctx, hostID, bundle.WebAssets); err != nil {
			return fmt.Errorf("failed to upsert web assets: %w", err)
		}
	}

	// 5. 同步到 Unified 表 (关键步骤: 确保指纹和资产信息落地到统一视图)
	if err := m.syncToUnified(ctx, bundle); err != nil {
		return fmt.Errorf("failed to sync to unified asset: %w", err)
	}

	// 6. 处理 Vulns
	if len(bundle.Vulns) > 0 {
		if err := m.upsertVulns(ctx, hostID, bundle.Vulns); err != nil {
			return fmt.Errorf("failed to upsert vulns: %w", err)
		}
	}

	return nil
}

// syncToUnified 同步资产信息到 Unified 表
func (m *assetMerger) syncToUnified(ctx context.Context, bundle *AssetBundle) error {
	if bundle.Host == nil {
		return nil
	}

	// 缓存已处理的端口
	processedPorts := make(map[int]bool)
	now := time.Now()

	// 1. 处理 Services -> Unified
	for _, svc := range bundle.Services {
		unified := &assetModel.AssetUnified{
			ProjectID:   bundle.ProjectID,
			IP:          bundle.Host.IP,
			Port:        svc.Port,
			HostName:    bundle.Host.Hostname,
			OS:          bundle.Host.OS,
			Protocol:    svc.Proto,
			Service:     svc.Name,
			Version:     svc.Version,
			Fingerprint: svc.Fingerprint, // 同步指纹信息
			SyncTime:    &now,
		}

		// 解析 CPE 获取 Product
		if svc.CPE != "" {
			// cpe:2.3:a:vendor:product:version:...
			parts := strings.Split(svc.CPE, ":")
			if len(parts) >= 5 {
				unified.Product = parts[4]
			}
		}

		// 检查是否有匹配的 Web
		for _, wa := range bundle.WebAssets {
			if wa == nil || wa.Web == nil {
				continue
			}
			web := wa.Web
			if getPortFromURL(web.URL) == svc.Port {
				unified.IsWeb = true
				unified.URL = web.URL
				unified.TechStack = web.TechStack

				// 尝试从 BasicInfo 提取 Title
				if web.BasicInfo != "" && web.BasicInfo != "{}" {
					var info map[string]interface{}
					if err := json.Unmarshal([]byte(web.BasicInfo), &info); err == nil {
						if title, ok := info["title"].(string); ok {
							unified.Title = title
						}
					}
				}
				break
			}
		}

		// Upsert 资产统一表
		if err := m.unifiedRepo.UpsertUnifiedAsset(ctx, unified); err != nil {
			return err
		}
		processedPorts[svc.Port] = true
	}

	// 2. 处理纯 Web (没有 Service 记录的情况)
	for _, wa := range bundle.WebAssets {
		if wa == nil || wa.Web == nil {
			continue
		}
		web := wa.Web
		port := getPortFromURL(web.URL)
		if processedPorts[port] {
			continue
		}

		unified := &assetModel.AssetUnified{
			ProjectID: bundle.ProjectID,
			IP:        bundle.Host.IP,
			Port:      port,
			HostName:  bundle.Host.Hostname,
			OS:        bundle.Host.OS,
			Protocol:  "tcp", // Web 默认为 TCP
			Service:   "http",
			URL:       web.URL,
			TechStack: web.TechStack,
			IsWeb:     true,
			SyncTime:  &now,
		}

		if strings.HasPrefix(web.URL, "https") {
			unified.Service = "https"
		}

		// 尝试从 BasicInfo 提取 Title
		if web.BasicInfo != "" && web.BasicInfo != "{}" {
			var info map[string]interface{}
			if err := json.Unmarshal([]byte(web.BasicInfo), &info); err == nil {
				if title, ok := info["title"].(string); ok {
					unified.Title = title
				}
			}
		}

		if err := m.unifiedRepo.UpsertUnifiedAsset(ctx, unified); err != nil {
			return err
		}
	}

	return nil
}

// getPortFromURL 从 URL 中获取端口
func getPortFromURL(rawURL string) int {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 80
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			return 443
		}
		return 80
	}
	p, _ := strconv.Atoi(port)
	return p
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

// upsertWebAssets 创建或更新 Web 资产(包含web信息和web详情)
func (m *assetMerger) upsertWebAssets(ctx context.Context, hostID uint64, webAssets []*WebAsset) error {
	now := time.Now()
	for _, wa := range webAssets {
		if wa == nil || wa.Web == nil || wa.Web.URL == "" {
			continue
		}

		// 获取Web资产中的web信息部分 - WebAsset.Web
		web := wa.Web
		web.HostID = hostID
		existing, err := m.webRepo.GetWebByURL(ctx, web.URL) // 根据URL获取Web资产
		if err != nil {
			return fmt.Errorf("check web existence failed: %w", err)
		}

		// 如果存在web资产
		if existing != nil {
			// 更新web资产相关的字段信息
			existing.LastSeenAt = &now
			if web.HostID != 0 {
				existing.HostID = web.HostID
			}
			if web.Domain != "" {
				existing.Domain = web.Domain
			}
			if web.TechStack != "" && web.TechStack != "{}" {
				existing.TechStack = web.TechStack
			}
			if web.BasicInfo != "" && web.BasicInfo != "{}" {
				existing.BasicInfo = web.BasicInfo
			}
			// 更新 AssetWeb 资产表
			if err := m.webRepo.UpdateWeb(ctx, existing); err != nil {
				return fmt.Errorf("update web failed: %w", err)
			}
			// 获取 AssetWeb.ID
			web.ID = existing.ID
		} else { // 如果不存在web资产
			if web.LastSeenAt == nil {
				web.LastSeenAt = &now
			}
			// 创建新的 AssetWeb 资产
			if err := m.webRepo.CreateWeb(ctx, web); err != nil {
				return fmt.Errorf("create web failed: %w", err)
			}
		}

		// 处理web资产详情 - WebAsset.Detail
		if wa.Detail == nil {
			continue // 没有web资产详情,就跳过
		}
		// 有web资产详情就赋值然后更新
		detail := wa.Detail
		detail.AssetWebID = web.ID
		// 更新 AssetWebDetail 资产
		if err := m.webRepo.CreateOrUpdateDetail(ctx, detail); err != nil {
			return fmt.Errorf("upsert web detail failed: %w", err)
		}
	}
	return nil
}

// upsertVulns 创建或更新漏洞资产 - AssetVuln
func (m *assetMerger) upsertVulns(ctx context.Context, hostID uint64, vulns []*assetModel.AssetVuln) error {
	now := time.Now()
	for _, v := range vulns {
		if v == nil {
			continue
		}

		// 获取漏洞资产目标类型
		targetType := strings.ToLower(strings.TrimSpace(v.TargetType))
		if targetType == "" {
			targetType = "host" // 默认目标类型为主机
		}

		// 解析漏洞资产目标
		targetRefID, resolvedTargetType, err := m.resolveVulnTarget(ctx, hostID, targetType, v)
		if err != nil {
			return err
		}
		v.TargetType = resolvedTargetType
		v.TargetRefID = targetRefID

		if v.FirstSeenAt == nil {
			v.FirstSeenAt = &now
		}
		v.LastSeenAt = &now
		if err := m.vulnRepo.UpsertVuln(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

// resolveVulnTarget 解析漏洞资产目标
func (m *assetMerger) resolveVulnTarget(ctx context.Context, hostID uint64, targetType string, v *assetModel.AssetVuln) (uint64, string, error) {
	switch targetType {
	case "service":
		// 尝试从VulnAttributes中获取漏洞资产目标端口
		port := extractPortFromVulnAttributes(v.Attributes)
		if port <= 0 {
			return hostID, "host", nil
		}
		// 获取对应端口的Service资产,协议筛选TCP
		svc, err := m.hostRepo.GetServiceByHostIDAndPort(ctx, hostID, port, "tcp")
		if err != nil {
			return 0, "", err
		}
		// 如果没有找到对应的Service资产,则尝试从UDP协议获取
		if svc == nil {
			svc, err = m.hostRepo.GetServiceByHostIDAndPort(ctx, hostID, port, "udp")
			if err != nil {
				return 0, "", err
			}
		}
		// 如果没有找到对应的Service资产,则创建一个新的Service资产
		if svc == nil {
			stub := &assetModel.AssetService{
				HostID:     hostID,
				Port:       port,
				Proto:      "tcp",
				LastSeenAt: timePtr(time.Now()),
			}
			if err := m.hostRepo.CreateService(ctx, stub); err != nil {
				return 0, "", err
			}
			svc = stub
		}
		return svc.ID, "service", nil
	case "web":
		// 尝试从VulnAttributes中获取漏洞资产目标URL
		rawURL := extractURLFromVulnAttributes(v.Attributes)
		if rawURL == "" {
			return hostID, "host", nil
		}
		// 获取对应URL的Web资产
		web, err := m.webRepo.GetWebByURL(ctx, rawURL)
		if err != nil {
			return 0, "", err
		}
		// 如果没有找到对应的Web资产,则创建一个新的Web资产
		if web == nil {
			domain := ""
			if u, err := url.Parse(rawURL); err == nil {
				domain = u.Hostname()
			}
			stub := &assetModel.AssetWeb{
				HostID:     hostID,
				URL:        rawURL,
				Domain:     domain,
				Tags:       "{}",
				LastSeenAt: timePtr(time.Now()),
			}
			if err := m.webRepo.CreateWeb(ctx, stub); err != nil {
				return 0, "", err
			}
			web = stub
		}
		return web.ID, "web", nil
	default:
		return hostID, "host", nil
	}
}

// extractPortFromVulnAttributes 尝试从漏洞资产属性中提取端口
func extractPortFromVulnAttributes(raw string) int {
	if raw == "" {
		return 0
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return 0
	}
	v, ok := m["port"]
	if !ok {
		return 0
	}
	switch vv := v.(type) {
	case float64:
		return int(vv)
	case int:
		return vv
	case string:
		p, _ := strconv.Atoi(vv)
		return p
	default:
		return 0
	}
}

// extractURLFromVulnAttributes 尝试从漏洞资产属性中提取URL
func extractURLFromVulnAttributes(raw string) string {
	if raw == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	if s, ok := m["url"].(string); ok && s != "" {
		return s
	}
	if s, ok := m["target"].(string); ok && s != "" {
		if strings.HasPrefix(strings.ToLower(s), "http://") || strings.HasPrefix(strings.ToLower(s), "https://") {
			return s
		}
	}
	return ""
}

func timePtr(t time.Time) *time.Time {
	return &t
}
