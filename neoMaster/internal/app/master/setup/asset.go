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
	hostRepo := assetRepo.NewAssetHostRepository(db)
	networkRepo := assetRepo.NewAssetNetworkRepository(db)
	policyRepo := assetRepo.NewAssetPolicyRepository(db)
	rawRepo := assetRepo.NewRawAssetRepository(db)

	// 2. Service 初始化
	hostService := assetService.NewAssetHostService(hostRepo)
	networkService := assetService.NewAssetNetworkService(networkRepo)
	policyService := assetService.NewAssetPolicyService(policyRepo)
	rawService := assetService.NewRawAssetService(rawRepo)

	// 3. Handler 初始化
	hostHandler := assetHandler.NewAssetHostHandler(hostService)
	networkHandler := assetHandler.NewAssetNetworkHandler(networkService)
	policyHandler := assetHandler.NewAssetPolicyHandler(policyService)
	rawHandler := assetHandler.NewRawAssetHandler(rawService)

	logger.WithFields(map[string]interface{}{
		"path":      "setup.asset",
		"operation": "build_module",
		"func_name": "setup.BuildAssetModule",
	}).Info("资产管理模块初始化完成")

	return &AssetModule{
		// Handlers
		AssetHostHandler:    hostHandler,
		AssetNetworkHandler: networkHandler,
		AssetPolicyHandler:  policyHandler,
		AssetRawHandler:     rawHandler,

		// Services
		AssetHostService:    hostService,
		AssetNetworkService: networkService,
		AssetPolicyService:  policyService,
		AssetRawService:     rawService,
	}
}
