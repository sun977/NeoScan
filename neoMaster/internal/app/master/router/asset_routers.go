/**
 * 路由:资产路由
 * @author: sun977
 * @date: 2025.12.04
 * @description: 资产路由模块
 * @func: 已完成
 */
package router

import (
	"neomaster/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

func (r *Router) setupAssetRoutes(v1 *gin.RouterGroup) {
	assetGroup := v1.Group("/asset")

	// 使用 JWT 中间件保护
	if r.middlewareManager != nil {
		assetGroup.Use(r.middlewareManager.GinJWTAuthMiddleware())
		assetGroup.Use(r.middlewareManager.GinUserActiveMiddleware())
	}

	{
		// 原始资产管理
		rawAssets := assetGroup.Group("/raw-assets")
		{
			rawAssets.POST("", r.assetRawHandler.CreateRawAsset)                   // 创建原始资产
			rawAssets.GET("/:id", r.assetRawHandler.GetRawAsset)                   // 获取原始资产详情
			rawAssets.PATCH("/:id/status", r.assetRawHandler.UpdateRawAssetStatus) // 更新原始资产状态
			rawAssets.GET("", r.assetRawHandler.ListRawAssets)                     // 获取原始资产列表
		}

		// 待处理网段管理
		rawNetworks := assetGroup.Group("/raw-networks")
		{
			rawNetworks.POST("", r.assetRawHandler.CreateRawNetwork)              // 创建待处理网段
			rawNetworks.GET("/:id", r.assetRawHandler.GetRawNetwork)              // 获取待处理网段详情
			rawNetworks.POST("/:id/approve", r.assetRawHandler.ApproveRawNetwork) // 批准待处理网段
			rawNetworks.POST("/:id/reject", r.assetRawHandler.RejectRawNetwork)   // 拒绝待处理网段
			rawNetworks.GET("", r.assetRawHandler.ListRawNetworks)                // 获取待处理网段列表
		}

		// 网段管理
		networks := assetGroup.Group("/networks")
		{
			networks.POST("", r.assetNetworkHandler.CreateNetwork)                     // 创建网段
			networks.GET("/:id", r.assetNetworkHandler.GetNetwork)                     // 获取网段详情
			networks.PUT("/:id", r.assetNetworkHandler.UpdateNetwork)                  // 更新网段
			networks.DELETE("/:id", r.assetNetworkHandler.DeleteNetwork)               // 删除网段
			networks.GET("", r.assetNetworkHandler.ListNetworks)                       // 获取网段列表
			networks.PATCH("/:id/scan-status", r.assetNetworkHandler.UpdateScanStatus) // 更新网段扫描状态

			// 网段标签管理
			networks.GET("/:id/tags", r.assetNetworkHandler.GetNetworkTags)              // 获取网段标签
			networks.POST("/:id/tags", r.assetNetworkHandler.AddNetworkTag)              // 添加网段标签
			networks.DELETE("/:id/tags/:tag_id", r.assetNetworkHandler.RemoveNetworkTag) // 删除网段标签
		}

		// 主机资产管理
		hosts := assetGroup.Group("/hosts")
		{
			hosts.POST("", r.assetHostHandler.CreateHost)       // 创建主机
			hosts.GET("/:id", r.assetHostHandler.GetHost)       // 获取主机详情
			hosts.PUT("/:id", r.assetHostHandler.UpdateHost)    // 更新主机
			hosts.DELETE("/:id", r.assetHostHandler.DeleteHost) // 删除主机
			hosts.GET("", r.assetHostHandler.ListHosts)         // 获取主机列表

			// 主机服务列表
			hosts.GET("/:id/services", r.assetHostHandler.ListServicesByHost)

			// 主机标签管理
			hosts.GET("/:id/tags", r.assetHostHandler.GetHostTags)              // 获取主机标签
			hosts.POST("/:id/tags", r.assetHostHandler.AddHostTag)              // 添加主机标签
			hosts.DELETE("/:id/tags/:tag_id", r.assetHostHandler.RemoveHostTag) // 删除主机标签

			// 主机服务标签管理
			services := hosts.Group("/:id/services")
			{
				services.GET("/:service_id/tags", r.assetHostHandler.GetServiceTags)              // 获取服务标签
				services.POST("/:service_id/tags", r.assetHostHandler.AddServiceTag)              // 添加服务标签
				services.DELETE("/:service_id/tags/:tag_id", r.assetHostHandler.RemoveServiceTag) // 删除服务标签
			}
		}

		// Web资产管理
		webs := assetGroup.Group("/webs")
		{
			webs.POST("", r.assetWebHandler.CreateWeb)       // 创建Web资产
			webs.GET("/:id", r.assetWebHandler.GetWeb)       // 获取Web资产详情
			webs.PUT("/:id", r.assetWebHandler.UpdateWeb)    // 更新Web资产
			webs.DELETE("/:id", r.assetWebHandler.DeleteWeb) // 删除Web资产
			webs.GET("", r.assetWebHandler.ListWebs)         // 获取Web资产列表

			// Web详细信息
			webs.GET("/:id/detail", r.assetWebHandler.GetWebDetail)  // 获取Web详细信息
			webs.PUT("/:id/detail", r.assetWebHandler.SaveWebDetail) // 保存Web详细信息

			// Web标签管理
			webs.GET("/:id/tags", r.assetWebHandler.GetWebTags)              // 获取Web资产标签
			webs.POST("/:id/tags", r.assetWebHandler.AddWebTag)              // 添加Web资产标签
			webs.DELETE("/:id/tags/:tag_id", r.assetWebHandler.RemoveWebTag) // 删除Web资产标签
		}

		// 资产策略管理
		policies := assetGroup.Group("/policies")
		{
			// 白名单管理
			whitelists := policies.Group("/whitelists")
			{
				whitelists.POST("", r.assetPolicyHandler.CreateWhitelist)       // 创建白名单
				whitelists.GET("/:id", r.assetPolicyHandler.GetWhitelist)       // 获取白名单详情
				whitelists.PUT("/:id", r.assetPolicyHandler.UpdateWhitelist)    // 更新白名单
				whitelists.DELETE("/:id", r.assetPolicyHandler.DeleteWhitelist) // 删除白名单
				whitelists.GET("", r.assetPolicyHandler.ListWhitelists)         // 获取白名单列表
			}

			// 跳过策略管理
			skipPolicies := policies.Group("/skip-policies")
			{
				skipPolicies.POST("", r.assetPolicyHandler.CreateSkipPolicy)       // 创建跳过策略
				skipPolicies.GET("/:id", r.assetPolicyHandler.GetSkipPolicy)       // 获取跳过策略详情
				skipPolicies.PUT("/:id", r.assetPolicyHandler.UpdateSkipPolicy)    // 更新跳过策略
				skipPolicies.DELETE("/:id", r.assetPolicyHandler.DeleteSkipPolicy) // 删除跳过策略
				skipPolicies.GET("", r.assetPolicyHandler.ListSkipPolicies)        // 获取跳过策略列表
			}
		}

		// 漏洞资产管理
		vulns := assetGroup.Group("/vulns")
		{
			// 漏洞情报管理
			vulns.POST("", r.assetVulnHandler.CreateVuln)       // 创建漏洞
			vulns.GET("/:id", r.assetVulnHandler.GetVuln)       // 获取漏洞详情
			vulns.PUT("/:id", r.assetVulnHandler.UpdateVuln)    // 更新漏洞信息
			vulns.DELETE("/:id", r.assetVulnHandler.DeleteVuln) // 删除漏洞
			vulns.GET("", r.assetVulnHandler.ListVulns)         // 获取漏洞列表

			// PoC管理
			vulns.POST("/pocs", r.assetVulnHandler.CreatePoc)         // 创建PoC
			vulns.GET("/pocs/:id", r.assetVulnHandler.GetPoc)         // 获取PoC详情
			vulns.PUT("/pocs/:id", r.assetVulnHandler.UpdatePoc)      // 更新PoC
			vulns.DELETE("/pocs/:id", r.assetVulnHandler.DeletePoc)   // 删除PoC
			vulns.GET("/:id/pocs", r.assetVulnHandler.ListPocsByVuln) // 获取漏洞关联的PoC列表
		}

		// 统一资产视图
		unified := assetGroup.Group("/unified")
		{
			unified.POST("", r.assetUnifiedHandler.CreateUnifiedAsset)        // 创建统一资产
			unified.POST("/upsert", r.assetUnifiedHandler.UpsertUnifiedAsset) // 插入或更新统一资产
			unified.GET("/:id", r.assetUnifiedHandler.GetUnifiedAsset)        // 获取统一资产详情
			unified.PUT("/:id", r.assetUnifiedHandler.UpdateUnifiedAsset)     // 更新统一资产
			unified.DELETE("/:id", r.assetUnifiedHandler.DeleteUnifiedAsset)  // 删除统一资产
			unified.GET("", r.assetUnifiedHandler.ListUnifiedAssets)          // 获取统一资产列表
		}

		// 资产扫描记录管理
		scans := assetGroup.Group("/scans")
		{
			scans.POST("", r.assetScanHandler.CreateScan)                                 // 创建扫描记录
			scans.GET("/:id", r.assetScanHandler.GetScan)                                 // 获取扫描记录详情
			scans.PUT("/:id", r.assetScanHandler.UpdateScan)                              // 更新扫描记录
			scans.DELETE("/:id", r.assetScanHandler.DeleteScan)                           // 删除扫描记录
			scans.GET("", r.assetScanHandler.ListScans)                                   // 获取扫描记录列表
			scans.GET("/latest/:network_id", r.assetScanHandler.GetLatestScanByNetworkID) // 获取指定网络ID的最新扫描记录
		}
	}

	logger.WithFields(map[string]interface{}{
		"path": "router.asset",
		"func": "setupAssetRoutes",
	}).Info("资产管理路由注册完成")
}
