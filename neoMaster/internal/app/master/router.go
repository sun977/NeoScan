package master

import (
	"net/http"
	"time"

	authHandler "neomaster/internal/handler/auth"
	systemHandler "neomaster/internal/handler/system"
	authPkg "neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/repository/mysql"
	redisRepo "neomaster/internal/repository/redis"
	authService "neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Router 路由管理器
type Router struct {
	engine            *gin.Engine
	middlewareManager *MiddlewareManager
	loginHandler      *authHandler.LoginHandler
	logoutHandler     *authHandler.LogoutHandler
	refreshHandler    *authHandler.RefreshHandler
	registerHandler   *authHandler.RegisterHandler
	userHandler       *systemHandler.UserHandler
}

// NewRouter 创建路由管理器实例
func NewRouter(db *gorm.DB, redisClient *redis.Client, jwtSecret string) *Router {
	// 初始化工具包
	jwtManager := authPkg.NewJWTManager(jwtSecret, time.Hour, 24*time.Hour)
	passwordConfig := &authPkg.PasswordConfig{
		Memory:      64 * 1024, // 64MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  32,
		KeyLength:   32,
	}
	userRepo := mysql.NewUserRepository(db) // 纯数据访问层
	passwordManager := authPkg.NewPasswordManager(passwordConfig)
	sessionRepo := redisRepo.NewSessionRepository(redisClient)

	// 初始化UserService
	userService := authService.NewUserService(userRepo, sessionRepo, passwordManager, jwtManager)

	// 初始化RBAC服务（不依赖其他服务）
	rbacService := authService.NewRBACService(userService)

	// 初始化SessionService（作为TokenBlacklistService的实现）
	// 注意：这里先创建一个临时的JWTService，后面会重新创建
	tempJWTService := authService.NewJWTService(jwtManager, userService, nil)
	sessionService := authService.NewSessionService(userService, passwordManager, tempJWTService, rbacService, sessionRepo)

	// 重新创建JWTService，注入SessionService作为TokenBlacklistService
	jwtService := authService.NewJWTService(jwtManager, userService, sessionService)

	// 更新SessionService中的JWTService引用
	// 注意：这里需要重新创建SessionService以避免循环依赖
	sessionService = authService.NewSessionService(userService, passwordManager, jwtService, rbacService, sessionRepo)

	// 初始化PasswordService（密码管理服务）
	passwordService := authService.NewPasswordService(userService, sessionService, passwordManager, time.Hour*24)

	// 初始化中间件管理器（传入jwtService用于密码版本验证）
	middlewareManager := NewMiddlewareManager(sessionService, rbacService, jwtService)

	// 初始化处理器(控制器是服务集合,先初始化服务,然后服务装填成控制器)
	loginHandler := authHandler.NewLoginHandler(sessionService)
	logoutHandler := authHandler.NewLogoutHandler(sessionService)
	refreshHandler := authHandler.NewRefreshHandler(sessionService)
	registerHandler := authHandler.NewRegisterHandler(userService)
	userHandler := systemHandler.NewUserHandler(userService, passwordService)

	// 创建Gin引擎
	gin.SetMode(gin.ReleaseMode) // 设置为生产模式
	engine := gin.New()

	return &Router{
		engine:            engine,
		middlewareManager: middlewareManager,
		loginHandler:      loginHandler,
		logoutHandler:     logoutHandler,
		refreshHandler:    refreshHandler,
		registerHandler:   registerHandler,
		userHandler:       userHandler,
	}
}

// SetupRoutes 设置路由
func (r *Router) SetupRoutes() {
	// 设置全局中间件
	r.engine.Use(r.middlewareManager.GinCORSMiddleware())
	r.engine.Use(r.middlewareManager.GinSecurityHeadersMiddleware())
	r.engine.Use(r.middlewareManager.GinLoggingMiddleware()) // 日志中间件注册
	r.engine.Use(r.middlewareManager.GinRateLimitMiddleware())

	// API版本路由组
	// /api/v1
	api := r.engine.Group("/api")
	v1 := api.Group("/v1")

	// 公共路由（不需要认证）
	r.setupPublicRoutes(v1)

	// 认证路由（需要JWT认证）
	r.setupAuthRoutes(v1)

	// 管理员路由（需要管理员权限）
	r.setupAdminRoutes(v1)

	// 健康检查路由
	r.setupHealthRoutes(api)
}

// setupPublicRoutes 设置公共路由
func (r *Router) setupPublicRoutes(v1 *gin.RouterGroup) {
	// 认证相关公共路由
	auth := v1.Group("/auth")
	{
		// 用户注册
		auth.POST("/register", r.registerHandler.Register) // handler\auth\register.go 没有权限校验的接口
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

// setupAuthRoutes 设置认证路由
func (r *Router) setupAuthRoutes(v1 *gin.RouterGroup) {
	// 认证相关路由（需要JWT认证和用户激活状态检查）
	auth := v1.Group("/auth")
	auth.Use(r.middlewareManager.GinJWTAuthMiddleware())
	auth.Use(r.middlewareManager.GinUserActiveMiddleware())
	{
		// 用户登出
		auth.POST("/logout", r.logoutHandler.Logout) // handler\auth\logout.go
		// 用户全部登出
		auth.POST("/logout-all", r.logoutHandler.LogoutAll) // handler\auth\logout.go
	}

	// 用户相关路由（需要JWT认证和用户激活状态检查）
	user := v1.Group("/user")
	user.Use(r.middlewareManager.GinJWTAuthMiddleware())
	user.Use(r.middlewareManager.GinUserActiveMiddleware())
	{
		// 获取当前用户信息(不是用户所有信息,仅用户信息)
		user.GET("/profile", r.userHandler.GetUserInfoByID) // handler\system\user.go
		// 修改用户密码
		user.POST("/change-password", r.userHandler.ChangePassword) // handler\system\user.go
		// 获取用户权限
		user.GET("/permissions", r.userHandler.GetUserPermission) // handler\system\user.go
		// 获取用户角色
		user.GET("/roles", r.userHandler.GetUserRoles) // handler\system\user.go
	}
}

// setupAdminRoutes 设置管理员路由
func (r *Router) setupAdminRoutes(v1 *gin.RouterGroup) {
	// 管理员路由组（需要JWT认证、用户激活状态检查和管理员权限）
	admin := v1.Group("/admin")
	admin.Use(r.middlewareManager.GinJWTAuthMiddleware())
	admin.Use(r.middlewareManager.GinUserActiveMiddleware())
	admin.Use(r.middlewareManager.GinAdminRoleMiddleware()) // 这里已经添加了管理员权限检查

	// 用户管理
	userMgmt := admin.Group("/users")
	{
		userMgmt.GET("/list", r.userHandler.GetUserList)           // handler\system\user.go
		userMgmt.POST("/create", r.userHandler.CreateUser)         // handler\system\user.go
		userMgmt.GET("/:id", r.userHandler.GetUserByID)            // handler\system\user.go
		userMgmt.GET("/:id/info", r.userHandler.GetUserInfoByID)   // handler\system\user.go 获取用户全量信息
		userMgmt.POST("/:id", r.userHandler.UpdateUserByID)        // handler\system\user.go
		userMgmt.DELETE("/:id", r.userHandler.DeleteUser)          // handler\system\user.go
		userMgmt.POST("/:id/activate", r.userHandler.ActivateUser) // handler\system\user.go
		userMgmt.POST("/:id/deactivate", r.deactivateUser)
	}

	// 角色管理
	roleMgmt := admin.Group("/roles")
	{
		roleMgmt.GET("/list", r.listRoles)
		roleMgmt.POST("/create", r.createRole)
		roleMgmt.GET("/:id", r.getRoleByID)
		roleMgmt.PUT("/:id", r.updateRole)
		roleMgmt.DELETE("/:id", r.deleteRole)
	}

	// 权限管理
	permMgmt := admin.Group("/permissions")
	{
		permMgmt.GET("/list", r.listPermissions)
		permMgmt.POST("/create", r.createPermission)
		permMgmt.GET("/:id", r.getPermissionByID)
		permMgmt.PUT("/:id", r.updatePermission)
		permMgmt.DELETE("/:id", r.deletePermission)
	}

	// 会话管理
	sessionMgmt := admin.Group("/sessions")
	{
		sessionMgmt.GET("/list", r.listActiveSessions)
		sessionMgmt.POST("/:sessionId/revoke", r.revokeSession)
		sessionMgmt.POST("/user/:userId/revoke-all", r.revokeAllUserSessions)
	}
}

// setupHealthRoutes 设置健康检查路由
func (r *Router) setupHealthRoutes(api *gin.RouterGroup) {
	// 健康检查
	api.GET("/health", r.healthCheck)
	// 就绪检查
	api.GET("/ready", r.readinessCheck)
	// 存活检查
	api.GET("/live", r.livenessCheck)
}

// GetEngine 获取Gin引擎实例
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// 处理器方法（这些方法需要在后续实现）

// 管理员用户管理处理器

func (r *Router) deactivateUser(c *gin.Context) {
	// TODO: 实现停用用户
	c.JSON(http.StatusOK, gin.H{"message": "deactivate user - not implemented yet"})
}

// 角色管理处理器
func (r *Router) listRoles(c *gin.Context) {
	// TODO: 实现角色列表
	c.JSON(http.StatusOK, gin.H{"message": "list roles - not implemented yet"})
}

func (r *Router) createRole(c *gin.Context) {
	// TODO: 实现创建角色
	c.JSON(http.StatusOK, gin.H{"message": "create role - not implemented yet"})
}

func (r *Router) getRoleByID(c *gin.Context) {
	// TODO: 实现根据ID获取角色
	c.JSON(http.StatusOK, gin.H{"message": "get role by id - not implemented yet"})
}

func (r *Router) updateRole(c *gin.Context) {
	// TODO: 实现更新角色
	c.JSON(http.StatusOK, gin.H{"message": "update role - not implemented yet"})
}

func (r *Router) deleteRole(c *gin.Context) {
	// TODO: 实现删除角色
	c.JSON(http.StatusOK, gin.H{"message": "delete role - not implemented yet"})
}

// 权限管理处理器
func (r *Router) listPermissions(c *gin.Context) {
	// TODO: 实现权限列表
	c.JSON(http.StatusOK, gin.H{"message": "list permissions - not implemented yet"})
}

func (r *Router) createPermission(c *gin.Context) {
	// TODO: 实现创建权限
	c.JSON(http.StatusOK, gin.H{"message": "create permission - not implemented yet"})
}

func (r *Router) getPermissionByID(c *gin.Context) {
	// TODO: 实现根据ID获取权限
	c.JSON(http.StatusOK, gin.H{"message": "get permission by id - not implemented yet"})
}

func (r *Router) updatePermission(c *gin.Context) {
	// TODO: 实现更新权限
	c.JSON(http.StatusOK, gin.H{"message": "update permission - not implemented yet"})
}

func (r *Router) deletePermission(c *gin.Context) {
	// TODO: 实现删除权限
	c.JSON(http.StatusOK, gin.H{"message": "delete permission - not implemented yet"})
}

// 会话管理处理器
func (r *Router) listActiveSessions(c *gin.Context) {
	// TODO: 实现活跃会话列表
	c.JSON(http.StatusOK, gin.H{"message": "list active sessions - not implemented yet"})
}

func (r *Router) revokeSession(c *gin.Context) {
	// TODO: 实现撤销会话
	c.JSON(http.StatusOK, gin.H{"message": "revoke session - not implemented yet"})
}

func (r *Router) revokeAllUserSessions(c *gin.Context) {
	// TODO: 实现撤销用户所有会话
	c.JSON(http.StatusOK, gin.H{"message": "revoke all user sessions - not implemented yet"})
}

// 健康检查处理器
func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": logger.NowFormatted(),
	})
}

func (r *Router) readinessCheck(c *gin.Context) {
	// TODO: 检查依赖服务（数据库、Redis等）是否就绪
	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": logger.NowFormatted(),
	})
}

func (r *Router) livenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": logger.NowFormatted(),
	})
}
