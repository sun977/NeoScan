package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"neomaster/internal/pkg/logger"
	repo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/fingerprint"
	"neomaster/internal/service/tag_system"
)

// FingerprintMatcher 离线指纹匹配服务
type FingerprintMatcher struct {
	assetRepo  *repo.AssetHostRepository
	fpService  fingerprint.Service
	tagService tag_system.TagService
}

// NewFingerprintMatcher 创建实例
func NewFingerprintMatcher(
	assetRepo *repo.AssetHostRepository,
	fpService fingerprint.Service,
	tagService tag_system.TagService,
) *FingerprintMatcher {
	return &FingerprintMatcher{
		assetRepo:  assetRepo,
		fpService:  fpService,
		tagService: tagService,
	}
}

// ProcessBatch 处理一批待识别的服务
// limit: 每次处理的数量
// enableTagLinkage: 是否启用标签联动
func (s *FingerprintMatcher) ProcessBatch(ctx context.Context, limit int, enableTagLinkage bool) (int, error) {
	// 1. 获取待处理服务 (Product为空且Banner不为空)
	services, err := s.assetRepo.GetServicesPendingFingerprint(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending services: %w", err)
	}

	if len(services) == 0 {
		return 0, nil
	}

	processedCount := 0

	for _, svc := range services {
		// 2. 获取 Host IP (用于指纹识别输入)
		host, err := s.assetRepo.GetHostByID(ctx, svc.HostID)
		if err != nil {
			logger.LogError(err, "", 0, "", "fingerprint_matcher.process", "GOVERNANCE", map[string]interface{}{
				"service_id": svc.ID,
				"host_id":    svc.HostID,
			})
			continue
		}
		if host == nil {
			// Host not found? 可能是孤儿数据，跳过
			continue
		}

		// 3. 构造识别输入
		input := &fingerprint.Input{
			Target:   host.IP,
			Port:     svc.Port,
			Protocol: svc.Proto,
			Banner:   svc.Banner,
			// TODO: 如果数据库有 Headers/Body 也可以填入，目前 AssetService 只有 Banner
		}

		// 4. 调用指纹服务
		result, err := s.fpService.Identify(ctx, input)
		if err != nil {
			logger.LogError(err, "", 0, "", "fingerprint_matcher.identify", "ENRICHMENT", map[string]interface{}{
				"service_id": svc.ID,
				"ip":         host.IP,
			})
			continue
		}

		// 5. 处理结果
		if result != nil && result.Best != nil {
			best := result.Best

			// 更新服务字段
			svc.Product = best.Product
			svc.Version = best.Version
			svc.CPE = best.CPE // 假设指纹服务返回的是标准CPE

			// 序列化完整指纹信息
			if fingerprintJSON, err := json.Marshal(result); err == nil {
				svc.Fingerprint = string(fingerprintJSON)
			}

			// 保存更新
			if err := s.assetRepo.UpdateService(ctx, svc); err != nil {
				logger.LogError(err, "", 0, "", "fingerprint_matcher.update_service", "GOVERNANCE", map[string]interface{}{
					"service_id": svc.ID,
				})
				continue
			}

			processedCount++

			// 6. 标签联动 (可选)
			if enableTagLinkage && s.tagService != nil {
				// 构造属性用于匹配规则
				attributes := map[string]interface{}{
					"product": svc.Product,
					"version": svc.Version,
					"vendor":  best.Vendor,
					"port":    svc.Port,
					"proto":   svc.Proto,
					"banner":  svc.Banner,
				}

				// 调用 AutoTag
				// 传入 "service" 作为 entityType,系统打标规则中的 entityType 为 "service",
				err := s.tagService.AutoTag(ctx, "service", strconv.FormatUint(svc.ID, 10), attributes)
				if err != nil {
					logger.LogError(err, "", 0, "", "fingerprint_matcher.auto_tag", "GOVERNANCE", map[string]interface{}{
						"service_id": svc.ID,
					})
				}
			}
		} else {
			// 没识别出来?
			// 策略: 标记为已处理，避免重复扫描?
			// 目前查询条件是 product is null. 如果没识别出来，product 还是 null，下次还会查出来。
			// 这是一个死循环风险。
			// 解决方案:
			// A. 增加 `fingerprint_status` 字段 (schema变更)
			// B. 填入 "unknown" 到 product (简单粗暴)
			// C. 依赖 `Fingerprint` 字段非空来判断 (如果指纹服务返回空result，我们存什么?)
			//
			// 既然是离线治理，建议填入一个特殊值或者更新 LastSeenAt 防止立即被再次选中?
			// 或者在 Query 中增加 LastSeenAt < X minutes ago?
			//
			// 这里采用方案 B: 如果识别失败，Product = "unknown" 或保留原样但记录日志?
			// 为了防止死循环，我们应该更新 Product 为 "unknown" 或者 "unidentified"

			svc.Product = "unknown"
			if err := s.assetRepo.UpdateService(ctx, svc); err != nil {
				logger.LogError(err, "", 0, "", "fingerprint_matcher.mark_unknown", "GOVERNANCE", map[string]interface{}{
					"service_id": svc.ID,
				})
			}
		}
	}

	return processedCount, nil
}
