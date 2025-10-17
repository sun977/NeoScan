/**
 * 路由:路由管理器
 * @author: sun977
 * @date: 2025.10.10
 * @description: 路由管理器，包含Router结构体、NewRouter函数和SetupRoutes主函数
 * @func:
 */
package router

import (
	agentRepo "neomaster/internal/repository/mysql/agent"
	"neomaster/internal/repository/mysql/system"
	"time"

	"neomaster/internal/app/master/middleware"
	agentHandler "neomaster/internal/handler/agent"
	authHandler "neomaster/internal/handler/auth"
	scanConfigHandler "neomaster/internal/handler/orchestrator"
	systemHandler "neomaster/internal/handler/system"
	authPkg "neomaster/internal/pkg/auth"
	scanConfigRepo "neomaster/internal/repository/orchestrator"
	redisRepo "neomaster/internal/repository/redis"
	agentService "neomaster/internal/service/agent"
	authService "neomaster/internal/service/auth"
	scanConfigService "neomaster/internal/service/orchestrator"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Router 路由管理器
type Router struct {
	engine            *gin.Engine
	middlewareManager *middleware.MiddlewareManager
	loginHandler      *authHandler.LoginHandler
	logoutHandler     *authHandler.LogoutHandler
	refreshHandler    *authHandler.RefreshHandler
	registerHandler   *authHandler.RegisterHandler
	userHandler       *systemHandler.UserHandler
	roleHandler       *systemHandler.RoleHandler
	permissionHandler *systemHandler.PermissionHandler
	sessionHandler    *systemHandler.SessionHandler
	// Agent管理相关Handler
	agentHandler *agentHandler.AgentHandler
	// 扫描配置相关Handler
	projectConfigHandler *scanConfigHandler.ProjectConfigHandler
	workflowHandler      *scanConfigHandler.WorkflowHandler
	scanToolHandler      *scanConfigHandler.ScanToolHandler
	scanRuleHandler      *scanConfigHandler.ScanRuleHandler
	ruleEngineHandler    *scanConfigHandler.RuleEngineHandler
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
	// // 根据配置选择会话存储方式【后续待补充】
	// var sessionRepo authService.SessionRepository
	// if cfg.Session.Store == "memory" {
	// 	sessionRepo = memory.NewSessionRepository()
	// } else {
	// 	sessionRepo = redisRepo.NewSessionRepository(redisClient)
	// }
	sessionRepo := redisRepo.NewSessionRepository(redisClient)

	// 初始化用户服务UserService
	userRepo := system.NewUserRepository(db) // 纯数据访问层
	userService := authService.NewUserService(userRepo, sessionRepo, passwordManager, jwtManager)

	// 初始化角色服务RoleService
	roleRepo := system.NewRoleRepository(db)
	roleService := authService.NewRoleService(roleRepo)

	// 初始化权限服务PermissionService
	permissionRepo := system.NewPermissionRepository(db)
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
	middlewareManager := middleware.NewMiddlewareManager(sessionService, rbacService, jwtService)

	// 初始化处理器(控制器是服务集合,先初始化服务,然后服务装填成控制器)
	loginHandler := authHandler.NewLoginHandler(sessionService)
	logoutHandler := authHandler.NewLogoutHandler(sessionService)
	refreshHandler := authHandler.NewRefreshHandler(sessionService)
	registerHandler := authHandler.NewRegisterHandler(userService)
	userHandler := systemHandler.NewUserHandler(userService, passwordService)
	roleHandler := systemHandler.NewRoleHandler(roleService)
	permissionHandler := systemHandler.NewPermissionHandler(permissionService)
	sessionHandler := systemHandler.NewSessionHandler(sessionService)

	// 初始化扫描配置相关Repository
	projectConfigRepo := scanConfigRepo.NewProjectConfigRepository(db)
	workflowConfigRepo := scanConfigRepo.NewWorkflowConfigRepository(db)
	scanToolRepo := scanConfigRepo.NewScanToolRepository(db)
	scanRuleRepo := scanConfigRepo.NewScanRuleRepository(db)

	// 初始化Agent相关Repository和Service
	agentRepository := agentRepo.NewAgentRepository(db)

	// 初始化Agent服务（分组功能已合并到Manager中）
	agentManagerService := agentService.NewAgentManagerService(agentRepository)
	agentMonitorService := agentService.NewAgentMonitorService(agentRepository)
	agentConfigService := agentService.NewAgentConfigService(agentRepository)
	agentTaskService := agentService.NewAgentTaskService(agentRepository)

	// 初始化扫描配置相关Service
	projectConfigService := scanConfigService.NewProjectConfigService(projectConfigRepo, workflowConfigRepo, scanToolRepo)
	workflowService := scanConfigService.NewWorkflowService(workflowConfigRepo, projectConfigRepo, scanToolRepo, scanRuleRepo)
	scanToolService := scanConfigService.NewScanToolService(scanToolRepo)
	scanRuleService := scanConfigService.NewScanRuleService(scanRuleRepo)

	// 初始化Agent相关Handler（分组功能已合并到AgentManagerService）
	agentHdl := agentHandler.NewAgentHandler(
		agentManagerService,
		agentMonitorService,
		agentConfigService,
		agentTaskService,
	)

	// 初始化扫描配置相关Handler
	projectConfigHandler := scanConfigHandler.NewProjectConfigHandler(projectConfigService)
	workflowHandler := scanConfigHandler.NewWorkflowHandler(workflowService)
	scanToolHandler := scanConfigHandler.NewScanToolHandler(scanToolService)
	scanRuleHandler := scanConfigHandler.NewScanRuleHandler(*scanRuleService)
	// 规则引擎Handler现在完全通过ScanRuleService管理规则引擎，不再需要单独的规则引擎实例
	ruleEngineHandler := scanConfigHandler.NewRuleEngineHandler(nil, scanRuleService)

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
		// Agent管理相关Handler
		agentHandler: agentHdl,
		// 扫描配置相关Handler
		projectConfigHandler: projectConfigHandler,
		workflowHandler:      workflowHandler,
		scanToolHandler:      scanToolHandler,
		scanRuleHandler:      scanRuleHandler,
		ruleEngineHandler:    ruleEngineHandler,
	}
}

// SetupRoutes 设置路由
// 在这里配置调用各个路由模块
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

	// 用户认证路由（需要JWT认证）
	r.setupUserRoutes(v1)

	// 管理员路由（需要管理员权限）
	r.setupAdminRoutes(v1)

	// 扫描编排器配置路由（需要JWT认证）
	r.setupOrchestratorRoutes(v1)

	// Agent管理路由（需要JWT认证）
	r.setupAgentRoutes(v1)

	// 健康检查路由
	r.setupHealthRoutes(api)
}

// GetEngine 获取Gin引擎实例
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
