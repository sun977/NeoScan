/**
 * 初始化:资产管理模块
 * @author: sun977
 * @date: 2025.12.04
 * @description: 资产管理模块初始化
 */
package setup

import (
	"neomaster/internal/pkg/logger"

	assetHandler "neomaster/internal/handler/asset"
	assetRepo "neomaster/internal/repo/mysql/asset"
	assetService "neomaster/internal/service/asset"

	"gorm.io/gorm"
)

// BuildAssetModule 构建资产管理模块
func BuildAssetModule(db *gorm.DB) *AssetModule {
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
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)

	// 2. Service 初始化
	rawService := assetService.NewRawAssetService(rawRepo)
	hostService := assetService.NewAssetHostService(hostRepo)
	networkService := assetService.NewAssetNetworkService(networkRepo)
	policyService := assetService.NewAssetPolicyService(policyRepo)
	webService := assetService.NewAssetWebService(webRepo)
	vulnService := assetService.NewAssetVulnService(vulnRepo)
	unifiedService := assetService.NewAssetUnifiedService(unifiedRepo)

	// 3. Handler 初始化
	rawHandler := assetHandler.NewRawAssetHandler(rawService)
	hostHandler := assetHandler.NewAssetHostHandler(hostService)
	networkHandler := assetHandler.NewAssetNetworkHandler(networkService)
	policyHandler := assetHandler.NewAssetPolicyHandler(policyService)
	webHandler := assetHandler.NewAssetWebHandler(webService)
	vulnHandler := assetHandler.NewAssetVulnHandler(vulnService)
	unifiedHandler := assetHandler.NewAssetUnifiedHandler(unifiedService)

	logger.WithFields(map[string]interface{}{
		"path":      "setup.asset",
		"operation": "build_module",
		"func_name": "setup.BuildAssetModule",
	}).Info("资产管理模块初始化完成")

	return &AssetModule{
		AssetRawHandler:     rawHandler,
		AssetHostHandler:    hostHandler,
		AssetNetworkHandler: networkHandler,
		AssetPolicyHandler:  policyHandler,
		AssetWebHandler:     webHandler,
		AssetVulnHandler:    vulnHandler,
		AssetUnifiedHandler: unifiedHandler,

		AssetRawService:     rawService,
		AssetHostService:    hostService,
		AssetNetworkService: networkService,
		AssetPolicyService:  policyService,
		AssetWebService:     webService,
		AssetVulnService:    vulnService,
		AssetUnifiedService: unifiedService,
	}
}
