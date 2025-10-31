/**
 * 路由:管理员路由
 * @author: sun977
 * @date: 2025.10.10
 * @description: 定义管理员路由和相关的结构体
 * @func:
 */
package router

import (
	"github.com/gin-gonic/gin"
)

// setupAdminRoutes 设置管理员路由
func (r *Router) setupAdminRoutes(v1 *gin.RouterGroup) {
	// 管理员路由（需要JWT认证、用户激活状态检查和管理员权限检查）
	admin := v1.Group("/admin")
	admin.Use(r.middlewareManager.GinJWTAuthMiddleware())    // JWT认证中间件
	admin.Use(r.middlewareManager.GinUserActiveMiddleware()) // 用户激活状态检查中间件
	admin.Use(r.middlewareManager.GinAdminRoleMiddleware())  // 管理员权限检查中间件
	{
		// 用户管理
		users := admin.Group("/users")
		{
			users.GET("/list", r.userHandler.GetUserList)               // 获取用户列表
			users.POST("/create", r.userHandler.CreateUser)             // 系统管理员创建用户(包含角色分配)
			users.GET("/:id", r.userHandler.GetUserByID)                // 获取用户详情(users表)
			users.GET("/:id/info", r.userHandler.GetUserInfoByID)       // 获取用户全量信息(包含权限和角色信息)
			users.POST("/:id", r.userHandler.UpdateUserByID)            // 包含用户角色更新
			users.DELETE("/:id", r.userHandler.DeleteUser)              // 删除用户(同时删除用户角色关系)
			users.POST("/:id/activate", r.userHandler.ActivateUser)     // 激活用户
			users.POST("/:id/deactivate", r.userHandler.DeactivateUser) // 禁用用户
			if r.config.App.Features.PasswordReset {                    // 检查配置文件密码重置功能开关
				users.POST("/:id/reset-password", r.userHandler.ResetUserPassword) // 重置用户密码
			}
			// users.POST("/:id/reset-password", r.userHandler.ResetUserPassword) // 重置用户密码
		}

		// 角色管理
		roles := admin.Group("/roles")
		{
			roles.GET("/list", r.roleHandler.GetRoleList)               // 获取角色列表
			roles.POST("/create", r.roleHandler.CreateRole)             // 创建角色(包含权限分配)
			roles.GET("/:id", r.roleHandler.GetRoleByID)                // 获取角色详情
			roles.POST("/:id", r.roleHandler.UpdateRole)                // 更新角色(包含权限更新)[Status字段可用于启用/禁用角色]
			roles.DELETE("/:id", r.roleHandler.DeleteRole)              // 删除角色(硬删除)
			roles.POST("/:id/activate", r.roleHandler.ActivateRole)     // 激活角色
			roles.POST("/:id/deactivate", r.roleHandler.DeactivateRole) // 禁用角色
		}

		// 权限管理
		permissions := admin.Group("/permissions")
		{
			permissions.GET("/list", r.permissionHandler.GetPermissionList)   // handler\system\permission.go
			permissions.POST("/create", r.permissionHandler.CreatePermission) // 创建权限(权限状态默认为启用)
			permissions.GET("/:id", r.permissionHandler.GetPermissionByID)    // 获取权限详情(包含关联角色)
			permissions.POST("/:id", r.permissionHandler.UpdatePermission)    // 更新权限(包含角色更新)[Status字段可用于启用/禁用权限]
			permissions.DELETE("/:id", r.permissionHandler.DeletePermission)  // 删除权限(同时删除权限角色关系)
		}

		// 会话管理
		sessionMgmt := admin.Group("/sessions")
		{
			sessionMgmt.GET("/user/list", r.sessionHandler.ListActiveSessions)                   // 使用 Query 参数指定 userId 来查询用户的会话列表
			sessionMgmt.POST("/user/:userId/revoke", r.sessionHandler.RevokeSession)             // 撤销用户会话 Param 路径传参
			sessionMgmt.POST("/user/:userId/revoke-all", r.sessionHandler.RevokeAllUserSessions) // 撤销用户所有会话
		}

	}
}
