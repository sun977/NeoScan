package router

import (
	"github.com/gin-gonic/gin"
)

// setupTagSystemRoutes 注册标签系统相关路由
func (r *Router) setupTagSystemRoutes(rg *gin.RouterGroup) {
	// 使用 JWT 中间件保护
	tags := rg.Group("/tags")
	tags.Use(r.middlewareManager.GinJWTAuthMiddleware())
	tags.Use(r.middlewareManager.GinUserActiveMiddleware())
	{
		// 标签 CRUD
		tags.POST("", r.tagHandler.CreateTag)
		tags.GET("/:id", r.tagHandler.GetTag)
		tags.PUT("/:id", r.tagHandler.UpdateTag)
		tags.PUT("/:id/move", r.tagHandler.MoveTag) // 移动标签 修改标签的层级关系(待定后续可能删除)
		tags.DELETE("/:id", r.tagHandler.DeleteTag)
		tags.GET("", r.tagHandler.ListTags) // 支持 ?parent_id=xxx

		// 规则 CRUD
		rules := tags.Group("/rules")
		rules.POST("", r.tagHandler.CreateRule)
		rules.GET("", r.tagHandler.ListRules) // 支持 ?entity_type=xxx
		rules.PUT("/:id", r.tagHandler.UpdateRule)
		rules.DELETE("/:id", r.tagHandler.DeleteRule)
		rules.POST("/:id/apply", r.tagHandler.ApplyRule) // 手动触发规则执行 ?action=add|remove
	}
}
