/**
 * 路由:公共路由
 * @author: sun977
 * @date: 2025.10.10
 * @description: 公共路由，包含注册、登录等不需要认证的路由
 * @func:
 */
package router

import (
	"github.com/gin-gonic/gin"
)

// setupPublicRoutes 设置公共路由
func (r *Router) setupPublicRoutes(v1 *gin.RouterGroup) {
	// 认证相关公共路由
	auth := v1.Group("/auth")
	{
		// 检查配置文件用户注册功能开关
		if r.config.App.Features.UserRegistration {
			// 用户注册
			auth.POST("/register", r.registerHandler.Register) // handler\auth\register.go 没有权限校验的接口，默认角色为普通用户 role_id = 2
		}
		// auth.POST("/register", r.registerHandler.Register) // handler\auth\register.go 没有权限校验的接口，默认角色为普通用户 role_id = 2
		// 用户登录
		auth.POST("/login", r.loginHandler.Login) // handler\auth\login.go
		// 获取登录表单页面（可选）
		// auth.GET("/login", r.loginHandler.GetLoginForm)
		// 刷新令牌(从body中传递传递refresh_token)
		auth.POST("/refresh", r.refreshHandler.RefreshToken) // handler\auth\refresh.go
		// 从请求头刷新令牌(从请求头Authorization传递refresh token)
		auth.POST("/refresh-header", r.refreshHandler.RefreshTokenFromHeader) // handler\auth\refresh.go
		// 检查令牌过期时间(从请求头中获取access token)
		auth.POST("/check-expiry", r.refreshHandler.CheckTokenExpiry) // handler\auth\refresh.go
	}
}
