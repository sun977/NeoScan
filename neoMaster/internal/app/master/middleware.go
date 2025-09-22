package master

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// MiddlewareManager 中间件管理器
// 负责管理所有Gin框架的中间件，提供统一的中间件接口
type MiddlewareManager struct {
	sessionService *auth.SessionService // 会话服务，用于JWT令牌验证
	rbacService    *auth.RBACService    // RBAC服务，用于角色和权限验证
	jwtService     *auth.JWTService     // JWT服务，用于令牌管理
}

// NewMiddlewareManager 创建中间件管理器
// 参数:
//   - sessionService: 会话服务实例
//   - rbacService: RBAC服务实例
//   - jwtService: JWT服务实例
//
// 返回: 中间件管理器实例
func NewMiddlewareManager(sessionService *auth.SessionService, rbacService *auth.RBACService, jwtService *auth.JWTService) *MiddlewareManager {
	return &MiddlewareManager{
		sessionService: sessionService,
		rbacService:    rbacService,
		jwtService:     jwtService,
	}
}

// =============================================================================
// Gin框架中间件实现
// =============================================================================

// GinJWTAuthMiddleware Gin JWT认证中间件
// 验证请求头中的JWT令牌，并将用户信息存储到Gin上下文中
// 使用方式: router.Use(middlewareManager.GinJWTAuthMiddleware())
func (m *MiddlewareManager) GinJWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中提取访问令牌
		accessToken, err := m.extractTokenFromGinHeader(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "error",
				Message: "missing or invalid authorization header",
				Error:   err.Error(),
			})
			c.Abort()
			return // 认证失败，直接返回
		}

		// 验证令牌 accessToken
		claims, err := m.sessionService.ValidateSession(c.Request.Context(), accessToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "error",
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
			logger.LogError(err, "", uint(claims.ID), claims.Username, "password_version_check", "GET", map[string]interface{}{
				"operation":    "password_version_check",
				"token_prefix": accessToken[:10] + "...",
				"client_ip":    c.ClientIP(),
				"user_agent":   c.GetHeader("User-Agent"),
				"timestamp":    logger.NowFormatted(),
			})

			// 根据不同的错误类型返回不同的响应
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				// 超时错误，可能是网络问题
				logger.LogError(err, "", uint(claims.ID), claims.Username, "password_version_timeout", "GET", map[string]interface{}{
					"operation":    "password_version_check",
					"error_type":   "network_timeout",
					"token_prefix": accessToken[:10] + "...",
					"timestamp":    logger.NowFormatted(),
				})

			case errors.Is(err, redis.Nil):
				// Redis键不存在，可能需要特殊处理
				logger.LogError(err, "", uint(claims.ID), claims.Username, "password_version_not_found", "GET", map[string]interface{}{
					"operation":    "password_version_check",
					"error_type":   "not_found",
					"token_prefix": accessToken[:10] + "...",
					"timestamp":    logger.NowFormatted(),
				})

			default:
				// 其他未知错误
				logger.LogError(err, "", uint(claims.ID), claims.Username, "password_version_unknown_error", "GET", map[string]interface{}{
					"operation":    "password_version_check",
					"error_type":   "unknown",
					"token_prefix": accessToken[:10] + "...",
					"timestamp":    logger.NowFormatted(),
				})

			}

			// 如果是临时性错误（如Redis连接问题），可以允许请求继续
			// 但应该记录警告并考虑通知管理员
		} else if !validVersion {
			// 密码版本不匹配，令牌已失效
			logger.LogBusinessOperation("password_version_mismatch", uint(claims.ID), claims.Username, "", "", "warning", "令牌因密码版本不匹配被拒绝", map[string]interface{}{
				"operation":    "password_version_check",
				"token_prefix": accessToken[:10] + "...",
				"timestamp":    logger.NowFormatted(),
				"client_ip":    c.ClientIP(),
				"user_agent":   c.GetHeader("User-Agent"),
			})
			c.JSON(http.StatusUnauthorized, model.APIResponse{
				Code:    http.StatusUnauthorized,
				Status:  "error",
				Message: "token version mismatch, please login again",
			})
			c.Abort()
			return
		}

		// 将用户信息添加到Gin上下文
		c.Set("user_id", claims.ID)
		c.Set("username", claims.Username)
		c.Set("roles", []string{})       // User模型中没有直接的Roles字段
		c.Set("permissions", []string{}) // User模型中没有直接的Permissions字段
		c.Set("claims", claims)

		// 继续处理请求
		c.Next()
	}
}

// GinUserActiveMiddleware Gin用户激活状态中间件
// 验证用户账户是否处于激活状态
// 使用方式: router.Use(middlewareManager.GinUserActiveMiddleware())
func (m *MiddlewareManager) GinUserActiveMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
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
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "invalid user ID type",
			})
			c.Abort()
			return
		}
		isActive, err := m.rbacService.IsUserActive(c.Request.Context(), userIDUint)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "failed to check user status",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		if !isActive {
			c.JSON(http.StatusForbidden, model.APIResponse{
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

// GinAdminRoleMiddleware Gin管理员角色中间件
// 验证用户是否具有管理员角色
// 使用方式: router.Use(middlewareManager.GinAdminRoleMiddleware())
func (m *MiddlewareManager) GinAdminRoleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
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
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "invalid user ID type",
			})
			c.Abort()
			return
		}
		hasRole, err := m.rbacService.CheckRole(c.Request.Context(), userIDUint, "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "failed to check role",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, model.APIResponse{
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

// GinRequireAnyRole Gin任意角色验证中间件
// 支持多角色验证，用户只需要拥有其中任意一个角色即可通过验证
// 参数: roles - 允许的角色列表
// 使用方式: router.Use(middlewareManager.GinRequireAnyRole("admin", "moderator"))
func (m *MiddlewareManager) GinRequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.APIResponse{
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
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "invalid user ID type",
			})
			c.Abort()
			return
		}
		hasAnyRole, err := m.rbacService.CheckAnyRole(c.Request.Context(), userIDUint, roles)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "failed",
				Message: "failed to check role",
				Error:   err.Error(),
			})
			c.Abort()
			return
		}

		if !hasAnyRole {
			c.JSON(http.StatusForbidden, model.APIResponse{
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

// GinCORSMiddleware Gin CORS中间件
// 处理跨域资源共享(CORS)请求
// 使用方式: router.Use(middlewareManager.GinCORSMiddleware())
func (m *MiddlewareManager) GinCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置CORS头
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// GinSecurityHeadersMiddleware Gin安全头中间件
// 设置各种安全相关的HTTP响应头
// 使用方式: router.Use(middlewareManager.GinSecurityHeadersMiddleware())
func (m *MiddlewareManager) GinSecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全头
		c.Header("X-Content-Type-Options", "nosniff")                                // 防止MIME类型嗅探
		c.Header("X-Frame-Options", "DENY")                                          // 防止点击劫持
		c.Header("X-XSS-Protection", "1; mode=block")                                // XSS保护
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains") // 强制HTTPS
		c.Header("Content-Security-Policy", "default-src 'self'")                    // 内容安全策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")               // 引用策略

		// 继续处理请求
		c.Next()
	}
}

// GinLoggingMiddleware Gin日志中间件
// 记录所有HTTP请求的访问日志和错误日志
// 使用方式: router.Use(middlewareManager.GinLoggingMiddleware())
func (m *MiddlewareManager) GinLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 提取必要请求信息
		clientIPRaw := c.GetHeader("X-Forwarded-For")
		if clientIPRaw == "" {
			clientIPRaw = c.GetHeader("X-Real-IP")
		}
		if clientIPRaw == "" {
			clientIPRaw = c.ClientIP()
		}
		clientIP := utils.NormalizeIP(clientIPRaw)

		// 存储到Gin上下文
		c.Set("client_ip", clientIP) // 这个是标准化后的可以用作业务使用的客户端IP
		// Gin上下文通过c.Set()方式存储值，后续可以通过c.Get("xx_key")获取

		// 存储到标准上下文
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, "client_ip", clientIP)
		// c.Request.Context()返回是标准的context.Context上下文，不包含gin的上下文
		// 可以使用WithValue方法将自定义的上下文值存储到标准上下文中
		// 这样后续使用标准上下文为参数的函数就可以安全获取自定义的上下文值
		// 获取方式：clientIP, _ := ctx.Value("client_ip").(string)
		c.Request = c.Request.WithContext(ctx)
		// 本项目只有handler中使用了Gin上下文，剩下的逻辑都在service中使用的标准上下文
		// 所以这里需要将Gin上下文的client_ip也存储到标准上下文

		// 处理请求
		c.Next()

		// 记录访问日志
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// 获取用户信息（如果已认证）
		userID := ""
		username := ""
		if uid, exists := c.Get("user_id"); exists {
			if uidUint, ok := uid.(uint); ok {
				userID = fmt.Sprintf("%d", uidUint)
			}
		}
		if uname, exists := c.Get("username"); exists {
			if unameStr, ok := uname.(string); ok {
				username = unameStr
			}
		}

		// 使用日志格式化器记录API请求
		userIDUint := uint(0)
		if userID != "" {
			if id, err := strconv.ParseUint(userID, 10, 32); err == nil {
				userIDUint = uint(id)
			}
		}
		logger.LogBusinessOperation("http_request", userIDUint, username, "", "", "success", "API Request", map[string]interface{}{
			"operation":     "http_request",
			"method":        c.Request.Method,
			"url":           c.Request.URL.String(),
			"status_code":   statusCode,
			"duration":      duration.Milliseconds(),
			"client_ip":     c.ClientIP(),
			"user_agent":    c.Request.UserAgent(),
			"referer":       c.Request.Referer(),
			"request_size":  c.Request.ContentLength,
			"response_size": int64(c.Writer.Size()),
			"timestamp":     logger.NowFormatted(),
		})

		// 如果是错误状态码，记录错误日志
		if statusCode >= 400 {
			errorMsg := ""
			if errors := c.Errors; len(errors) > 0 {
				errorMsg = errors.String()
			}

			logger.LogError(fmt.Errorf("HTTP %d: %s", statusCode, errorMsg), "", userIDUint, username, "http_request", c.Request.Method, map[string]interface{}{
				"operation":   "http_request",
				"method":      c.Request.Method,
				"url":         c.Request.URL.String(),
				"status_code": statusCode,
				"client_ip":   c.ClientIP(),
				"timestamp":   logger.NowFormatted(),
			})
		}
	}
}

// GinRateLimitMiddleware Gin限流中间件
// 实现API请求频率限制（当前为占位实现）
// 使用方式: router.Use(middlewareManager.GinRateLimitMiddleware())
func (m *MiddlewareManager) GinRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现基于IP或用户的限流逻辑
		// 这里可以集成Redis或内存缓存来实现限流
		// 可以考虑使用以下策略：
		// 1. 基于IP的限流：防止单个IP过度请求
		// 2. 基于用户的限流：防止单个用户过度请求
		// 3. 基于API端点的限流：对不同端点设置不同的限流策略

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
		return "", &model.ValidationError{Field: "authorization", Message: "authorization header is required"}
	}

	// 检查Bearer前缀
	if !strings.HasPrefix(authorization, "Bearer ") {
		return "", &model.ValidationError{Field: "authorization", Message: "authorization header must start with 'Bearer '"}
	}

	// 提取令牌
	token := strings.TrimPrefix(authorization, "Bearer ")
	if token == "" {
		return "", &model.ValidationError{Field: "authorization", Message: "access token cannot be empty"}
	}

	return token, nil
}
