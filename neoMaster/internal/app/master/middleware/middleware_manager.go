package middleware

import (
	"neomaster/internal/config"
	"neomaster/internal/service/auth"
	"sync"
)

// MiddlewareManager 中间件管理器
// 负责管理所有Gin框架的中间件，提供统一的中间件接口
type MiddlewareManager struct {
	sessionService *auth.SessionService   // 会话服务，用于JWT令牌验证
	rbacService    *auth.RBACService      // RBAC服务，用于角色和权限验证
	jwtService     *auth.JWTService       // JWT服务，用于令牌管理
	securityConfig *config.SecurityConfig // 安全配置，用于中间件配置
	rateLimiter    RateLimiter
	rateLimiterOnce sync.Once
}

// NewMiddlewareManager 创建中间件管理器
// 参数:
//   - sessionService: 会话服务实例
//   - rbacService: RBAC服务实例
//   - jwtService: JWT服务实例
//   - securityConfig: 安全配置实例
//
// 返回: 中间件管理器实例
func NewMiddlewareManager(sessionService *auth.SessionService, rbacService *auth.RBACService, jwtService *auth.JWTService, securityConfig *config.SecurityConfig) *MiddlewareManager {
	return &MiddlewareManager{
		sessionService: sessionService,
		rbacService:    rbacService,
		jwtService:     jwtService,
		securityConfig: securityConfig,
	}
}
