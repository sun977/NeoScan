/**
 * 路由:资产路由
 * @author: sun977
 * @date: 2025.12.04
 * @description: 资产路由模块
 * @func: 已完成
 */
package router

import (
	"github.com/gin-gonic/gin"
)

func (r *Router) setupAssetRoutes(v1 *gin.RouterGroup) {
	assetGroup := v1.Group("/asset")

	// 使用 JWT 中间件保护
	if r.middlewareManager != nil {
		assetGroup.Use(r.middlewareManager.GinJWTAuthMiddleware())
	}

	{
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
	}

	// logger.WithFields(map[string]interface{}{
	// 	"path": "router.asset",
	// 	"func": "setupAssetRoutes",
	// }).Info("资产管理路由注册完成")
}
