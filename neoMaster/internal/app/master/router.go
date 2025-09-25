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
	roleHandler       *systemHandler.RoleHandler
	permissionHandler *systemHandler.PermissionHandler
	sessionHandler    *systemHandler.SessionHandler
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
	passwordManager := authPkg.NewPasswordManager(passwordConfig)
	sessionRepo := redisRepo.NewSessionRepository(redisClient)

	// 初始化用户服务UserService
	userRepo := mysql.NewUserRepository(db) // 纯数据访问层
	userService := authService.NewUserService(userRepo, sessionRepo, passwordManager, jwtManager)

	// 初始化角色服务RoleService
	roleRepo := mysql.NewRoleRepository(db)
	roleService := authService.NewRoleService(roleRepo)

	// 初始化权限服务PermissionService
	permissionRepo := mysql.NewPermissionRepository(db)
	permissionService := authService.NewPermissionService(permissionRepo)

	// 初始化RBAC服务（不依赖其他服务）
	rbacService := authService.NewRBACService(userService)

	// 先创建SessionService（不传入JWTService）
	sessionService := authService.NewSessionService(userService, passwordManager, rbacService, sessionRepo)

	// 再创建JWTService
	jwtService := authService.NewJWTService(jwtManager, userService, sessionRepo)

	// 设置SessionService的TokenGenerator（解决循环依赖）
	sessionService.SetTokenGenerator(jwtService)

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
	roleHandler := systemHandler.NewRoleHandler(roleService)
	permissionHandler := systemHandler.NewPermissionHandler(permissionService)
	sessionHandler := systemHandler.NewSessionHandler(sessionService)

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
		roleHandler:       roleHandler,
		permissionHandler: permissionHandler,
		sessionHandler:    sessionHandler,
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
		auth.POST("/register", r.registerHandler.Register) // handler\auth\register.go 没有权限校验的接口，默认角色为普通用户 role_id = 2
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

// setupAdminRoutes 设置管理员路由
func (r *Router) setupAdminRoutes(v1 *gin.RouterGroup) {
	// 管理员路由组（需要JWT认证、用户激活状态检查和管理员权限）
	admin := v1.Group("/admin")
	admin.Use(r.middlewareManager.GinJWTAuthMiddleware())    // JWT认证中间件
	admin.Use(r.middlewareManager.GinUserActiveMiddleware()) // 用户激活状态检查中间件
	admin.Use(r.middlewareManager.GinAdminRoleMiddleware())  // 管理员权限检查中间件

	// 用户管理(系统管理员管理用户)
	userMgmt := admin.Group("/users")
	{
		userMgmt.GET("/list", r.userHandler.GetUserList)                      // 获取用户列表
		userMgmt.POST("/create", r.userHandler.CreateUser)                    // 系统管理员创建用户(包含角色分配)
		userMgmt.GET("/:id", r.userHandler.GetUserByID)                       // 获取用户详情(users表)
		userMgmt.GET("/:id/info", r.userHandler.GetUserInfoByID)              // 获取用户全量信息(包含权限和角色信息)
		userMgmt.POST("/:id", r.userHandler.UpdateUserByID)                   // 包含用户角色更新
		userMgmt.DELETE("/:id", r.userHandler.DeleteUser)                     // 删除用户(同时删除用户角色关系)
		userMgmt.POST("/:id/activate", r.userHandler.ActivateUser)            // 激活用户
		userMgmt.POST("/:id/deactivate", r.userHandler.DeactivateUser)        // 禁用用户
		userMgmt.POST("/:id/reset-password", r.userHandler.ResetUserPassword) // 重置用户密码
	}

	// 角色管理
	roleMgmt := admin.Group("/roles")
	{
		roleMgmt.GET("/list", r.roleHandler.GetRoleList)               // 获取角色列表
		roleMgmt.POST("/create", r.roleHandler.CreateRole)             // 创建角色(包含权限分配)
		roleMgmt.GET("/:id", r.roleHandler.GetRoleByID)                // 获取角色详情
		roleMgmt.POST("/:id", r.roleHandler.UpdateRole)                // 更新角色(包含权限更新)[Status字段可用于启用/禁用角色]
		roleMgmt.DELETE("/:id", r.roleHandler.DeleteRole)              // 删除角色(硬删除)
		roleMgmt.POST("/:id/activate", r.roleHandler.ActivateRole)     // 激活角色
		roleMgmt.POST("/:id/deactivate", r.roleHandler.DeactivateRole) // 禁用角色
	}

	// 权限管理
	permMgmt := admin.Group("/permissions")
	{
		permMgmt.GET("/list", r.permissionHandler.GetPermissionList)   // handler\system\permission.go
		permMgmt.POST("/create", r.permissionHandler.CreatePermission) // 创建权限(权限状态默认为启用)
		permMgmt.GET("/:id", r.permissionHandler.GetPermissionByID)    // 获取权限详情(包含关联角色)
		permMgmt.POST("/:id", r.permissionHandler.UpdatePermission)    // 更新权限(包含角色更新)[Status字段可用于启用/禁用权限]
		permMgmt.DELETE("/:id", r.permissionHandler.DeletePermission)  // 删除权限(同时删除权限角色关系)
	}

	// 会话管理
	sessionMgmt := admin.Group("/sessions")
	{
		sessionMgmt.GET("/user/list", r.sessionHandler.ListActiveSessions)                   // 使用 Query 参数指定 userId 来查询用户的会话列表
		sessionMgmt.POST("/user/:userId/revoke", r.sessionHandler.RevokeSession)             // 撤销用户会话 Param 路径传参
		sessionMgmt.POST("/user/:userId/revoke-all", r.sessionHandler.RevokeAllUserSessions) // 撤销用户所有会话
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
