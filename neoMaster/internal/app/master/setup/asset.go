/**
 * 初始化:资产管理模块
 * @author: sun977
 * @date: 2025.12.04
 * @description: 资产管理模块初始化
 */
package setup

import (
	"neomaster/internal/pkg/logger"

	"neomaster/internal/config"
	assetHandler "neomaster/internal/handler/asset"
	assetRepo "neomaster/internal/repo/mysql/asset"
	assetService "neomaster/internal/service/asset"
	"neomaster/internal/service/asset/enrichment"
	"neomaster/internal/service/asset/etl"
	"neomaster/internal/service/fingerprint"
	"neomaster/internal/service/fingerprint/engines/http"
	"neomaster/internal/service/fingerprint/engines/service"
	tagService "neomaster/internal/service/tag_system"

	"gorm.io/gorm"
)

// BuildAssetModule 构建资产管理模块
func BuildAssetModule(db *gorm.DB, config *config.Config, tagSystem tagService.TagService, etlProcessor etl.ResultProcessor) *AssetModule {
	logger.WithFields(map[string]interface{}{
		"path":      "setup.asset",
		"operation": "build_module",
		"func_name": "setup.BuildAssetModule",
	}).Info("开始初始化资产管理模块")

	// 1. Repository 初始化
	rawRepo := assetRepo.NewRawAssetRepository(db)
	hostRepo := assetRepo.NewAssetHostRepository(db)
	networkRepo := assetRepo.NewAssetNetworkRepository(db)
	policyRepo := assetRepo.NewAssetPolicyRepository(db)
	ruleRepo := assetRepo.NewRuleRepository(db) // 规则仓库,规则包括指纹(CPE、CMS指纹) 和 POC等
	fingerCmsRepo := assetRepo.NewAssetFingerRepository(db)
	fingerServiceRepo := assetRepo.NewAssetCPERepository(db)
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)
	scanRepo := assetRepo.NewAssetScanRepository(db)
	etlErrorRepo := assetRepo.NewETLErrorRepository(db)

	// 2. Service 初始化
	rawService := assetService.NewRawAssetService(rawRepo, tagSystem)                     // 原始资产管理服务
	hostService := assetService.NewAssetHostService(hostRepo, tagSystem)                  // 主机资产服务
	networkService := assetService.NewAssetNetworkService(networkRepo, tagSystem)         // 网络资产服务
	policyService := assetService.NewAssetPolicyService(policyRepo, tagSystem)            // 策略执行服务
	fingerCmsService := assetService.NewAssetFingerService(fingerCmsRepo, tagSystem)      // CMS指纹服务
	fingerServiceService := assetService.NewAssetCPEService(fingerServiceRepo, tagSystem) // CPE指纹服务
	webService := assetService.NewAssetWebService(webRepo, tagSystem)                     // Web资产服务
	vulnService := assetService.NewAssetVulnService(vulnRepo, tagSystem)                  // 漏洞资产服务
	unifiedService := assetService.NewAssetUnifiedService(unifiedRepo, tagSystem)         // 汇总资产服务
	scanService := assetService.NewAssetScanService(scanRepo, networkRepo)                // 扫描记录服务(记录扫描记录)
	etlErrorService := assetService.NewAssetETLErrorService(etlErrorRepo, etlProcessor)   // ETL错误处理服务

	// 2.1 指纹规则管理
	// 从配置中获取规则加密密钥，如果未配置则默认为空
	ruleEncryptionKey := ""
	if config != nil {
		ruleEncryptionKey = config.Security.Agent.RuleEncryptionKey
	}
	fingerprintRuleManager := fingerprint.NewRuleManager(ruleRepo, fingerCmsRepo, fingerServiceRepo, ruleEncryptionKey, config)

	// 2.2 初始化指纹识别服务 (Runtime Identifier)
	// Master 本地运行时指纹识别服务，用于资产二次指纹识别
	httpEngine := http.NewHTTPEngine(fingerCmsRepo)
	serviceEngine := service.NewServiceEngine(fingerServiceRepo)
	fpService := fingerprint.NewFingerprintService(httpEngine, serviceEngine)

	// 加载规则 (尝试从默认目录加载)
	// TODO: 路径应从配置中获取，与 RuleManager 保持一致
	ruleDir := "rules/fingerprint"
	if config != nil && config.App.Rules.RootPath != "" {
		// 这里简单拼接，实际应保持一致
		// 暂且使用硬编码的 rules/fingerprint，因为 RuleManager 默认也是发布到这里
	}

	// 尝试加载规则，如果失败仅记录日志（可能是初次启动无规则）
	if err := fpService.LoadRules(ruleDir); err != nil {
		logger.LogBusinessError(err, "system", 0, "localhost", "", "load_rules", map[string]interface{}{
			"path": ruleDir,
			"msg":  "failed to load fingerprint rules (non-fatal)",
		})
	}

	// 2.3 初始化指纹治理服务 (Governance)
	fingerprintGovernance := enrichment.NewFingerprintMatcher(hostRepo, fpService, tagSystem)

	// 3. Handler 初始化
	rawHandler := assetHandler.NewRawAssetHandler(rawService)
	hostHandler := assetHandler.NewAssetHostHandler(hostService)
	networkHandler := assetHandler.NewAssetNetworkHandler(networkService)
	policyHandler := assetHandler.NewAssetPolicyHandler(policyService)
	fingerCmsHandler := assetHandler.NewAssetFingerHandler(fingerCmsService)
	fingerServiceHandler := assetHandler.NewAssetCPEHandler(fingerServiceService)
	webHandler := assetHandler.NewAssetWebHandler(webService)
	vulnHandler := assetHandler.NewAssetVulnHandler(vulnService)
	unifiedHandler := assetHandler.NewAssetUnifiedHandler(unifiedService)
	scanHandler := assetHandler.NewAssetScanHandler(scanService)
	fingerprintRuleHandler := assetHandler.NewFingerprintRuleHandler(fingerprintRuleManager)
	etlErrorHandler := assetHandler.NewETLErrorHandler(etlErrorService)

	logger.WithFields(map[string]interface{}{
		"path":      "setup.asset",
		"operation": "build_module",
		"func_name": "setup.BuildAssetModule",
	}).Info("资产管理模块初始化完成")

	return &AssetModule{
		AssetRawHandler:           rawHandler,             // 原始资产Handler - 用于处理原始资产数据
		AssetHostHandler:          hostHandler,            // 主机资产Handler - 用于处理主机资产数据
		AssetNetworkHandler:       networkHandler,         // 网络资产Handler - 用于处理网络资产数据
		AssetPolicyHandler:        policyHandler,          // 策略执行Handler - 用于处理策略执行数据
		AssetFingerCmsHandler:     fingerCmsHandler,       // CMS指纹Handler - 用于处理CMS指纹数据
		AssetFingerServiceHandler: fingerServiceHandler,   // CPE指纹Handler - 用于处理CPE指纹数据
		AssetWebHandler:           webHandler,             // Web资产Handler - 用于处理Web资产数据
		AssetVulnHandler:          vulnHandler,            // 漏洞资产Handler - 用于处理漏洞资产数据
		AssetUnifiedHandler:       unifiedHandler,         // 汇总资产Handler - 用于处理汇总资产数据
		AssetScanHandler:          scanHandler,            // 扫描记录Handler - 用于处理扫描记录数据
		FingerprintRuleHandler:    fingerprintRuleHandler, // 添加指纹规则管理Handler - 用于资产指纹规则管理(指纹规则下发给Agent)
		ETLErrorHandler:           etlErrorHandler,        // 添加 ETL 错误处理Handler - 用于处理资产 ETL 过程中的错误

		AssetRawService:           rawService,
		AssetHostService:          hostService,
		AssetNetworkService:       networkService,
		AssetPolicyService:        policyService,
		AssetFingerCmsService:     fingerCmsService,
		AssetFingerServiceService: fingerServiceService,
		AssetWebService:           webService,
		AssetVulnService:          vulnService,
		AssetUnifiedService:       unifiedService,
		AssetScanService:          scanService,
		FingerprintRuleManager:    fingerprintRuleManager, // 添加指纹规则管理服务 - 用于资产指纹规则管理(指纹规则下发给Agent)
		AssetETLErrorService:      etlErrorService,        // 添加 ETL 错误处理服务 - 用于处理资产 ETL 过程中的错误
		FingerprintGovernance:     fingerprintGovernance,  // 添加指纹治理服务 - 用于资产二次指纹识别(Master本地运行时)
	}
}
