/**
 * 路由:用户路由
 * @author: sun977
 * @date: 2025.10.10
 * @description: 包含需要JWT认证的用户相关路由
 * @func:
 */
package router

import (
	"github.com/gin-gonic/gin"
)

// setupUserRoutes 设置用户认证路由
func (r *Router) setupUserRoutes(v1 *gin.RouterGroup) {
	// 认证相关路由（需要JWT认证和用户激活状态检查）
	auth := v1.Group("/auth")
	auth.Use(r.middlewareManager.GinJWTAuthMiddleware())
	auth.Use(r.middlewareManager.GinUserActiveMiddleware())
	{
		// 登出只能一次
		// 用户全部登出(更新密码版本,所有类型token失效,不再使用redis撤销黑名单的方式)
		auth.POST("/logout-all", r.logoutHandler.LogoutAll)
	}

	// 用户相关路由（需要JWT认证和用户激活状态检查）
	user := v1.Group("/user")
	user.Use(r.middlewareManager.GinJWTAuthMiddleware())
	user.Use(r.middlewareManager.GinUserActiveMiddleware())
	{
		// 获取当前用户全量信息(包含权限和角色信息)
		user.GET("/profile", r.userHandler.GetUserInfoByIDforUser) // 获取当前用户全量信息
		// 修改用户密码
		user.POST("/change-password", r.userHandler.ChangePassword) // 修改用户密码
		// 更新用户信息（需要补充）
		user.POST("/update", r.userHandler.UserUpdateInfoByID) // 允许用户自己修改自己的信息（仅user表，不能修改角色和权限等）
		// 获取用户权限
		user.GET("/permissions", r.userHandler.GetUserPermission) // 获取用户权限(permissions表)
		// 获取用户角色
		user.GET("/roles", r.userHandler.GetUserRoles) // 获取用户角色(roles表)
	}
}
