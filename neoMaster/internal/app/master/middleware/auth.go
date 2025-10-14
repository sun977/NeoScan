/**
 * 中间件:认证相关中间件
 * @author: sun977
 * @date: 2025.10.10
 * @description: 定义认证相关中间件
 * @func:
 *   - GinJWTAuthMiddleware: Gin JWT认证中间件
 *   - GinUserActiveMiddleware: 检查用户是否活跃中间件
 *   - GinAdminRoleMiddleware: 检查用户是否具有管理员角色中间件
 *   - GinRequireAnyRole: 检查用户是否具有任意角色中间件[未使用]
 *   - extractTokenFromGinHeader: 从Gin请求头中提取JWT令牌
 */
package middleware

import (
	"context"
	"errors"
	"neomaster/internal/model/system"
	"net/http"
	"strings"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// =============================================================================
// JWT认证相关中间件
// =============================================================================

// GinJWTAuthMiddleware Gin JWT认证中间件
// 验证请求头中的JWT令牌，并将用户信息存储到Gin上下文中
// 使用方式: router.Use(middlewareManager.GinJWTAuthMiddleware())
func (m *MiddlewareManager) GinJWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 提取参数
		clientIP := utils.GetClientIP(c)
		XRequestID := c.GetHeader("X-Request-ID")
		userAgent := c.GetHeader("User-Agent")

		// 从请求头中提取访问令牌
		accessToken, err := m.extractTokenFromGinHeader(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "missing or invalid authorization header",
				Error:   err.Error(),
			})
			c.Abort()
			return // 认证失败，直接返回
		}

		// 验证令牌 accessToken
		claims, err := m.sessionService.ValidateSession(c.Request.Context(), accessToken)
		if err != nil {
			// 记录错误日志
			logger.LogError(err, XRequestID, 0, clientIP, "token_validation", "GET", map[string]interface{}{
				"operation":    "token_validation",
				"token_prefix": accessToken[:10] + "...",
				"client_ip":    clientIP,
				"user_agent":   userAgent,
				"X-Request-ID": XRequestID,
				"timestamp":    logger.NowFormatted(),
			})
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "invalid or expired token",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		// 验证密码版本（确保修改密码后旧token失效）
		// 注意：由于Redis连接问题，此功能可能会失败，但不会导致崩溃
		validVersion, err := m.jwtService.ValidatePasswordVersion(c.Request.Context(), accessToken)
		if err != nil {
			// 记录错误日志
			logger.LogError(err, XRequestID, uint(claims.ID), clientIP, "password_version_check", "GET", map[string]interface{}{
				"operation":    "password_version_check",
				"token_prefix": accessToken[:10] + "...",
				"client_ip":    clientIP,
				"username":     claims.Username,
				"user_agent":   userAgent,
				"X-Request-ID": XRequestID,
				"timestamp":    logger.NowFormatted(),
			})

			// 根据不同的错误类型返回不同的响应
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				// 超时错误，可能是网络问题，允许请求继续
				logger.LogError(err, XRequestID, uint(claims.ID), clientIP, "password_version_timeout", "GET", map[string]interface{}{
					"operation":    "password_version_check",
					"error_type":   "network_timeout",
					"username":     claims.Username,
					"token_prefix": accessToken[:10] + "...",
					"X-Request-ID": XRequestID,
					"timestamp":    logger.NowFormatted(),
				})

			case errors.Is(err, redis.Nil):
				// Redis键不存在，可能是测试环境或首次登录，允许请求继续
				logger.LogError(err, XRequestID, uint(claims.ID), clientIP, "password_version_not_found", "GET", map[string]interface{}{
					"operation":    "password_version_check",
					"error_type":   "not_found",
					"username":     claims.Username,
					"X-Request-ID": XRequestID,
					"token_prefix": accessToken[:10] + "...",
					"timestamp":    logger.NowFormatted(),
				})

			default:
				// 其他未知错误，记录但允许请求继续
				logger.LogError(err, XRequestID, uint(claims.ID), clientIP, "password_version_unknown_error", "GET", map[string]interface{}{
					"operation":    "password_version_check",
					"error_type":   "unknown",
					"token_prefix": accessToken[:10] + "...",
					"username":     claims.Username,
					"X-Request-ID": XRequestID,
					"timestamp":    logger.NowFormatted(),
				})

			}

			// 如果是临时性错误（如Redis连接问题），允许请求继续
			// 但应该记录警告并考虑通知管理员
			// 重要：即使密码版本验证失败，也要设置用户上下文，让后续中间件能正常工作
		} else if !validVersion {
			// 密码版本不匹配，令牌已失效
			logger.LogBusinessOperation("password_version_mismatch", uint(claims.ID), claims.Username, clientIP, XRequestID, "warning", "令牌因密码版本不匹配被拒绝", map[string]interface{}{
				"operation":    "password_version_check",
				"token_prefix": accessToken[:10] + "...",
				"client_ip":    clientIP,
				"username":     claims.Username,
				"user_agent":   userAgent,
				"X-Request-ID": XRequestID,
				"timestamp":    logger.NowFormatted(),
			})
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "token version mismatch, please login again",
			})
			c.Abort()
			return
		}

		// 将用户信息添加到Gin上下文
		// 无论密码版本验证是否成功，都要设置用户上下文，让后续中间件能正常工作
		c.Set("user_id", claims.ID)
		c.Set("username", claims.Username)
		c.Set("roles", []string{})       // User模型中没有直接的Roles字段
		c.Set("permissions", []string{}) // User模型中没有直接的Permissions字段
		c.Set("claims", claims)

		// 继续处理请求
		c.Next()
	}
}

// =============================================================================
// 用户状态验证中间件
// =============================================================================

// GinUserActiveMiddleware Gin用户激活状态中间件
// 验证用户账户是否处于激活状态
// 使用方式: router.Use(middlewareManager.GinUserActiveMiddleware())
func (m *MiddlewareManager) GinUserActiveMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "user not authenticated",
			})
			c.Abort()
			return
		}

		// 检查用户是否处于活跃状态
		// 修复类型转换问题：JWT中间件设置的user_id是uint类型
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "invalid user ID type",
			})
			c.Abort()
			return
		}
		isActive, err := m.rbacService.IsUserActive(c.Request.Context(), userIDUint)
		if err != nil {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "failed to check user status",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		if !isActive {
			c.JSON(http.StatusForbidden, system.APIResponse{
				Code:    http.StatusForbidden,
				Status:  "failed",
				Message: "user account is inactive",
			})
			c.Abort()
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// =============================================================================
// 角色权限验证中间件
// =============================================================================

// GinAdminRoleMiddleware Gin管理员角色中间件
// 验证用户是否具有管理员角色
// 使用方式: router.Use(middlewareManager.GinAdminRoleMiddleware())
func (m *MiddlewareManager) GinAdminRoleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "user not authenticated",
			})
			c.Abort()
			return
		}

		// 检查用户角色
		// 修复类型转换问题：JWT中间件设置的user_id是uint类型
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "invalid user ID type",
			})
			c.Abort()
			return
		}
		hasRole, err := m.rbacService.CheckRole(c.Request.Context(), userIDUint, "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "failed to check role",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, system.APIResponse{
				Code:    http.StatusForbidden,
				Status:  "failed",
				Message: "admin role required",
			})
			c.Abort()
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// GinRequireAnyRole Gin任意角色验证中间件
// 支持多角色验证，用户只需要拥有其中任意一个角色即可通过验证
// 参数: roles - 允许的角色列表
// 使用方式: router.Use(middlewareManager.GinRequireAnyRole("admin", "moderator"))
func (m *MiddlewareManager) GinRequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, system.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "failed",
				Message: "user not authenticated",
			})
			c.Abort()
			return
		}

		// 检查用户是否拥有任意一个角色
		// 修复类型转换问题：JWT中间件设置的user_id是uint类型
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "invalid user ID type",
			})
			c.Abort()
			return
		}
		hasAnyRole, err := m.rbacService.CheckAnyRole(c.Request.Context(), userIDUint, roles)
		if err != nil {
			c.JSON(http.StatusInternalServerError, system.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "failed to check role",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		if !hasAnyRole {
			c.JSON(http.StatusForbidden, system.APIResponse{
				Code:    http.StatusForbidden,
				Status:  "failed",
				Message: "insufficient role privileges",
			})
			c.Abort()
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// =============================================================================
// 辅助方法
// =============================================================================

// extractTokenFromGinHeader 从Gin请求头中提取访问令牌
// 参数: c - Gin上下文
// 返回: 访问令牌字符串和可能的错误
func (m *MiddlewareManager) extractTokenFromGinHeader(c *gin.Context) (string, error) {
	authorization := c.GetHeader("Authorization")
	if authorization == "" {
		return "", &system.ValidationError{Field: "authorization", Message: "authorization header is required"}
	}

	// 检查Bearer前缀
	if !strings.HasPrefix(authorization, "Bearer ") {
		return "", &system.ValidationError{Field: "authorization", Message: "authorization header must start with 'Bearer '"}
	}

	// 提取令牌
	token := strings.TrimPrefix(authorization, "Bearer ")
	if token == "" {
		return "", &system.ValidationError{Field: "authorization", Message: "access token cannot be empty"}
	}

	return token, nil
}
