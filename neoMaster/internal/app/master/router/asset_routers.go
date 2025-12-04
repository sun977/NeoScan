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
	}

	// logger.WithFields(map[string]interface{}{
	// 	"path": "router.asset",
	// 	"func": "setupAssetRoutes",
	// }).Info("资产管理路由注册完成")
}
