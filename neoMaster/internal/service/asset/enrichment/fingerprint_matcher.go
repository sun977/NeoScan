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
	assetRepo  *repo.AssetHostRepository // 资产仓库示例
	fpService  fingerprint.Service       // 指纹识别服务示例
	tagService tag_system.TagService     // 标签服务示例
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
	// 这其实信息补齐,找到所有Product为空,但Banner不为空的服务然后二次识别并补充一些缺失的信息 或 打标签
	services, err := s.assetRepo.GetServicesPendingFingerprint(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending services: %w", err)
	}

	// 检查是否有待处理的服务
	if len(services) == 0 {
		return 0, nil
	}

	// 初始化已处理计数器
	processedCount := 0

	// 遍历所有待处理服务
	for _, svc := range services {
		// 2. 获取 Host IP (用于指纹识别输入)
		host, err := s.assetRepo.GetHostByID(ctx, svc.HostID) // 根据 Host.ID 获取Host信息
		if err != nil {
			logger.LogError(err, "", 0, "", "fingerprint_matcher.process", "GOVERNANCE", map[string]interface{}{
				"service_id": svc.ID,
				"host_id":    svc.HostID,
			})
			continue
		}
		if host == nil {
			// 主机不存在? 可能是孤儿数据，跳过
			continue
		}

		// 3. 构造识别输入
		// Input 结构体定义在 fingerprint/input.go 中，包含了指纹识别所需的所有字段，后续可以根据需要添加其他字段
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
			// 有最佳匹配结果
			best := result.Best

			// 更新服务字段 - 填充识别出的产品、版本、CPE 等信息
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

			// 增加已处理计数器
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
			// 如果指纹识别失败或没有匹配结果
			// 防止无限循环处理未识别的服务，将其标记为"unknown"

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
