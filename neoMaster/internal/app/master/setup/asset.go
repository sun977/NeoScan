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
	"neomaster/internal/service/asset/etl"
	"neomaster/internal/service/fingerprint"
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
	fingerCmsRepo := assetRepo.NewAssetFingerRepository(db)
	fingerServiceRepo := assetRepo.NewAssetCPERepository(db)
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)
	scanRepo := assetRepo.NewAssetScanRepository(db)
	etlErrorRepo := assetRepo.NewETLErrorRepository(db)

	// 2. Service 初始化
	rawService := assetService.NewRawAssetService(rawRepo, tagSystem)
	hostService := assetService.NewAssetHostService(hostRepo, tagSystem)
	networkService := assetService.NewAssetNetworkService(networkRepo, tagSystem)
	policyService := assetService.NewAssetPolicyService(policyRepo, tagSystem)
	fingerCmsService := assetService.NewAssetFingerService(fingerCmsRepo, tagSystem)
	fingerServiceService := assetService.NewAssetCPEService(fingerServiceRepo, tagSystem)
	webService := assetService.NewAssetWebService(webRepo, tagSystem)
	vulnService := assetService.NewAssetVulnService(vulnRepo, tagSystem)
	unifiedService := assetService.NewAssetUnifiedService(unifiedRepo, tagSystem)
	scanService := assetService.NewAssetScanService(scanRepo, networkRepo)
	etlErrorService := assetService.NewAssetETLErrorService(etlErrorRepo, etlProcessor)

	// 2.1 指纹规则管理
	// 从配置中获取规则加密密钥，如果未配置则默认为空
	ruleEncryptionKey := ""
	if config != nil {
		ruleEncryptionKey = config.Security.Agent.RuleEncryptionKey
	}
	fingerprintRuleManager := fingerprint.NewRuleManager(fingerCmsRepo, fingerServiceRepo, ruleEncryptionKey)

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
		AssetRawHandler:           rawHandler,
		AssetHostHandler:          hostHandler,
		AssetNetworkHandler:       networkHandler,
		AssetPolicyHandler:        policyHandler,
		AssetFingerCmsHandler:     fingerCmsHandler,
		AssetFingerServiceHandler: fingerServiceHandler,
		AssetWebHandler:           webHandler,
		AssetVulnHandler:          vulnHandler,
		AssetUnifiedHandler:       unifiedHandler,
		AssetScanHandler:          scanHandler,
		FingerprintRuleHandler:    fingerprintRuleHandler,
		ETLErrorHandler:           etlErrorHandler,

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
		FingerprintRuleManager:    fingerprintRuleManager,
		AssetETLErrorService:      etlErrorService,
	}
}
