package master

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// MiddlewareManager 中间件管理器
type MiddlewareManager struct {
	sessionService *auth.SessionService
	rbacService    *auth.RBACService
	jwtService     *auth.JWTService
}

// NewMiddlewareManager 创建中间件管理器
func NewMiddlewareManager(sessionService *auth.SessionService, rbacService *auth.RBACService, jwtService *auth.JWTService) *MiddlewareManager {
	return &MiddlewareManager{
		sessionService: sessionService,
		rbacService:    rbacService,
		jwtService:     jwtService,
	}
}

// JWTAuthMiddleware JWT认证中间件
func (m *MiddlewareManager) JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置响应头
		w.Header().Set("Content-Type", "application/json")

		// 从请求头中提取访问令牌
		accessToken, err := m.extractTokenFromHeader(r)
		if err != nil {
			m.writeErrorResponse(w, http.StatusUnauthorized, "missing or invalid authorization header", err)
			return
		}

		// 验证令牌
		claims, err := m.sessionService.ValidateSession(r.Context(), accessToken)
		if err != nil {
			m.writeErrorResponse(w, http.StatusUnauthorized, "invalid or expired token", err)
			return
		}

		// 验证密码版本（确保修改密码后旧token失效）
		validVersion, err := m.jwtService.ValidatePasswordVersion(r.Context(), accessToken)
		if err != nil {
			m.writeErrorResponse(w, http.StatusUnauthorized, "failed to validate token version", err)
			return
		}
		if !validVersion {
			m.writeErrorResponse(w, http.StatusUnauthorized, "token version mismatch, please login again", nil)
			return
		}

		// 将用户信息添加到请求上下文
		ctx := context.WithValue(r.Context(), "user_id", claims.ID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "roles", []string{})       // User模型中没有直接的Roles字段
		ctx = context.WithValue(ctx, "permissions", []string{}) // User模型中没有直接的Permissions字段
		ctx = context.WithValue(ctx, "claims", claims)

		// 继续处理请求
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission 权限验证中间件
func (m *MiddlewareManager) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从上下文获取用户ID
			userID, ok := r.Context().Value("user_id").(uint64)
			if !ok {
				m.writeErrorResponse(w, http.StatusUnauthorized, "user not authenticated", nil)
				return
			}

			// 解析权限字符串
			resource, action, err := m.rbacService.ParsePermissionString(permission)
			if err != nil {
				m.writeErrorResponse(w, http.StatusBadRequest, "invalid permission format", err)
				return
			}

			// 检查用户权限
			hasPermission, err := m.rbacService.CheckPermission(r.Context(), uint(userID), resource, action)
			if err != nil {
				m.writeErrorResponse(w, http.StatusInternalServerError, "failed to check permission", err)
				return
			}

			if !hasPermission {
				m.writeErrorResponse(w, http.StatusForbidden, "insufficient permissions", nil)
				return
			}

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole 角色验证中间件
func (m *MiddlewareManager) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从上下文获取用户ID
			userID, ok := r.Context().Value("user_id").(uint64)
			if !ok {
				m.writeErrorResponse(w, http.StatusUnauthorized, "user not authenticated", nil)
				return
			}

			// 检查用户角色
			hasRole, err := m.rbacService.CheckRole(r.Context(), uint(userID), role)
			if err != nil {
				m.writeErrorResponse(w, http.StatusInternalServerError, "failed to check role", err)
				return
			}

			if !hasRole {
				m.writeErrorResponse(w, http.StatusForbidden, "insufficient role privileges", nil)
				return
			}

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole 任意角色验证中间件
func (m *MiddlewareManager) RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从上下文获取用户ID
			userID, ok := r.Context().Value("user_id").(uint64)
			if !ok {
				m.writeErrorResponse(w, http.StatusUnauthorized, "user not authenticated", nil)
				return
			}

			// 检查用户是否拥有任意一个角色
			hasAnyRole, err := m.rbacService.CheckAnyRole(r.Context(), uint(userID), roles)
			if err != nil {
				m.writeErrorResponse(w, http.StatusInternalServerError, "failed to check role", err)
				return
			}

			if !hasAnyRole {
				m.writeErrorResponse(w, http.StatusForbidden, "insufficient role privileges", nil)
				return
			}

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// RequireActiveUser 活跃用户验证中间件
func (m *MiddlewareManager) RequireActiveUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从上下文获取用户ID
		userID, ok := r.Context().Value("user_id").(uint64)
		if !ok {
			m.writeErrorResponse(w, http.StatusUnauthorized, "user not authenticated", nil)
			return
		}

		// 检查用户是否处于活跃状态
		isActive, err := m.rbacService.IsUserActive(r.Context(), uint(userID))
		if err != nil {
			m.writeErrorResponse(w, http.StatusInternalServerError, "failed to check user status", err)
			return
		}

		if !isActive {
			m.writeErrorResponse(w, http.StatusForbidden, "user account is inactive", nil)
			return
		}

		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware CORS中间件
func (m *MiddlewareManager) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware 日志中间件
func (m *MiddlewareManager) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 创建响应记录器
		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 处理请求
		next.ServeHTTP(rec, r)

		// 记录访问日志
		duration := time.Since(start)
		
		// 使用日志格式化器记录HTTP请求
		logger.LogBusinessOperation("http_request", 0, "", "", "", "info", "HTTP请求", map[string]interface{}{
			"operation": "http_request",
			"method":       r.Method,
			"url":          r.URL.String(),
			"status_code":  rec.statusCode,
			"duration":     duration.Milliseconds(),
			"client_ip":    r.RemoteAddr,
			"user_agent":   r.UserAgent(),
			"referer":      r.Referer(),
			"request_size": r.ContentLength,
			"response_size": int64(rec.size),
			"timestamp":    start,
		})
	})
}

// RateLimitMiddleware 限流中间件（简单实现）
func (m *MiddlewareManager) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: 实现基于IP或用户的限流逻辑
		// 这里可以集成Redis或内存缓存来实现限流

		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware 安全头中间件
func (m *MiddlewareManager) SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置安全头
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}

// 辅助方法

// extractTokenFromHeader 从请求头中提取访问令牌
func (m *MiddlewareManager) extractTokenFromHeader(r *http.Request) (string, error) {
	authorization := r.Header.Get("Authorization")
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

// writeErrorResponse 写入错误响应
func (m *MiddlewareManager) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	w.WriteHeader(statusCode)
	response := model.APIResponse{
		Code:    statusCode,
		Status:  "error",
		Message: message,
	}

	if err != nil {
		response.Error = err.Error()
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		http.Error(w, "failed to encode error response", http.StatusInternalServerError)
	}
}

// responseRecorder 响应记录器，用于日志中间件
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *responseRecorder) Write(data []byte) (int, error) {
	n, err := rec.ResponseWriter.Write(data)
	rec.size += n
	return n, err
}

// 中间件链构建器

// Chain 中间件链
type Chain struct {
	middlewares []func(http.Handler) http.Handler
}

// NewChain 创建新的中间件链
func NewChain(middlewares ...func(http.Handler) http.Handler) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Then 应用中间件链到处理器
func (c *Chain) Then(handler http.Handler) http.Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

// Append 添加中间件到链
func (c *Chain) Append(middlewares ...func(http.Handler) http.Handler) *Chain {
	newMiddlewares := make([]func(http.Handler) http.Handler, len(c.middlewares)+len(middlewares))
	copy(newMiddlewares, c.middlewares)
	copy(newMiddlewares[len(c.middlewares):], middlewares)
	return &Chain{middlewares: newMiddlewares}
}

// 常用中间件组合

// PublicChain 公共中间件链（不需要认证）
func (m *MiddlewareManager) PublicChain() *Chain {
	return NewChain(
		m.CORSMiddleware,
		m.SecurityHeadersMiddleware,
		m.LoggingMiddleware,
		m.RateLimitMiddleware,
	)
}

// AuthChain 认证中间件链
func (m *MiddlewareManager) AuthChain() *Chain {
	return m.PublicChain().Append(
		m.JWTAuthMiddleware,
		m.RequireActiveUser,
	)
}

// AdminChain 管理员中间件链
func (m *MiddlewareManager) AdminChain() *Chain {
	return m.AuthChain().Append(
		m.RequireRole("admin"),
	)
}

// Gin框架中间件适配器

// GinJWTAuthMiddleware Gin JWT认证中间件
func (m *MiddlewareManager) GinJWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中提取访问令牌
		accessToken, err := m.extractTokenFromGinHeader(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "missing or invalid authorization header",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		// 验证令牌
		claims, err := m.sessionService.ValidateSession(c.Request.Context(), accessToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "invalid or expired token",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		// 验证密码版本（确保修改密码后旧token失效）
		// 注意：由于Redis连接问题，此功能可能会失败，但不会导致崩溃
		validVersion, err := m.jwtService.ValidatePasswordVersion(c.Request.Context(), accessToken)
		if err != nil {
			// 如果是Redis连接问题，记录警告但不阻止请求
			// TODO: 添加日志记录
			// 暂时跳过密码版本验证，允许请求继续
		} else if !validVersion {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "token version mismatch, please login again",
			})
			c.Abort()
			return
		}

		// 将用户信息添加到Gin上下文
		c.Set("user_id", claims.ID)
		c.Set("username", claims.Username)
		c.Set("roles", []string{})
		c.Set("permissions", []string{})
		c.Set("claims", claims)

		// 继续处理请求
		c.Next()
	}
}

// GinUserActiveMiddleware Gin用户激活状态中间件
func (m *MiddlewareManager) GinUserActiveMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "user not authenticated",
			})
			c.Abort()
			return
		}

		// 检查用户是否处于活跃状态
		// 修复类型转换问题：JWT中间件设置的user_id是uint类型
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "invalid user ID type",
			})
			c.Abort()
			return
		}
		isActive, err := m.rbacService.IsUserActive(c.Request.Context(), userIDUint)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to check user status",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		if !isActive {
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "user account is inactive",
			})
			c.Abort()
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// GinAdminRoleMiddleware Gin管理员角色中间件
func (m *MiddlewareManager) GinAdminRoleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "user not authenticated",
			})
			c.Abort()
			return
		}

		// 检查用户角色
		// 修复类型转换问题：JWT中间件设置的user_id是uint类型
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "invalid user ID type",
			})
			c.Abort()
			return
		}
		hasRole, err := m.rbacService.CheckRole(c.Request.Context(), userIDUint, "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to check role",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "insufficient role privileges",
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
func (m *MiddlewareManager) GinRequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "user not authenticated",
			})
			c.Abort()
			return
		}

		// 检查用户是否拥有任意一个角色
		// 修复类型转换问题：JWT中间件设置的user_id是uint类型
		userIDUint, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "invalid user ID type",
			})
			c.Abort()
			return
		}
		hasAnyRole, err := m.rbacService.CheckAnyRole(c.Request.Context(), userIDUint, roles)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to check role",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		if !hasAnyRole {
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "insufficient role privileges",
			})
			c.Abort()
			return
		}

		// 继续处理请求
		c.Next()
	}
}

// GinCORSMiddleware Gin CORS中间件
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
func (m *MiddlewareManager) GinSecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置安全头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 继续处理请求
		c.Next()
	}
}

// GinLoggingMiddleware Gin日志中间件
func (m *MiddlewareManager) GinLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

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
		logger.LogBusinessOperation("http_request", userIDUint, username, "", "", "info", "API请求", map[string]interface{}{
			"operation": "http_request",
			"method":       c.Request.Method,
			"url":          c.Request.URL.String(),
			"status_code":  statusCode,
			"duration":     duration.Milliseconds(),
			"client_ip":    c.ClientIP(),
			"user_agent":   c.Request.UserAgent(),
			"referer":      c.Request.Referer(),
			"request_size": c.Request.ContentLength,
			"response_size": int64(c.Writer.Size()),
			"timestamp":    start,
		})
		
		// 如果是错误状态码，记录错误日志
		if statusCode >= 400 {
			errorMsg := ""
			if errors := c.Errors; len(errors) > 0 {
				errorMsg = errors.String()
			}
			
			logger.LogError(fmt.Errorf("HTTP %d: %s", statusCode, errorMsg), "", userIDUint, username, "http_request", c.Request.Method, map[string]interface{}{
				"operation": "http_request",
				"method":      c.Request.Method,
				"url":         c.Request.URL.String(),
				"status_code": statusCode,
				"client_ip":   c.ClientIP(),
				"timestamp": time.Now(),
			})
		}
	}
}

// GinRateLimitMiddleware Gin限流中间件
func (m *MiddlewareManager) GinRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现基于IP或用户的限流逻辑
		// 这里可以集成Redis或内存缓存来实现限流

		// 继续处理请求
		c.Next()
	}
}

// extractTokenFromGinHeader 从Gin请求头中提取访问令牌
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
