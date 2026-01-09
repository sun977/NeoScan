/**
 * 路由:路由管理器
 * @author: sun977
 * @date: 2025.10.10
 * @description: 路由管理器，包含Router结构体、NewRouter函数和SetupRoutes主函数
 * @func:
 */
package router

import (
	setup "neomaster/internal/app/master/setup"
	"neomaster/internal/service/asset/etl"
	"neomaster/internal/service/orchestrator/core/scheduler"
	"neomaster/internal/service/orchestrator/local_agent"

	"neomaster/internal/app/master/middleware"
	"neomaster/internal/config"
	agentHandler "neomaster/internal/handler/agent"
	assetHandler "neomaster/internal/handler/asset"
	authHandler "neomaster/internal/handler/auth"
	orchestratorHandler "neomaster/internal/handler/orchestrator"
	systemHandler "neomaster/internal/handler/system"
	tagHandler "neomaster/internal/handler/tag_system"

	// 统一使用项目封装的日志模块，便于采集规范字段与统一输出
	"neomaster/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Router 路由管理器
type Router struct {
	config            *config.Config
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
	// 资产管理相关Handler
	assetRawHandler     *assetHandler.RawAssetHandler
	assetHostHandler    *assetHandler.AssetHostHandler
	assetNetworkHandler *assetHandler.AssetNetworkHandler
	assetPolicyHandler  *assetHandler.AssetPolicyHandler
	assetWebHandler     *assetHandler.AssetWebHandler
	assetVulnHandler    *assetHandler.AssetVulnHandler
	assetUnifiedHandler *assetHandler.AssetUnifiedHandler
	assetScanHandler    *assetHandler.AssetScanHandler

	// 编排器相关Handler
	projectHandler          *orchestratorHandler.ProjectHandler
	workflowHandler         *orchestratorHandler.WorkflowHandler
	scanStageHandler        *orchestratorHandler.ScanStageHandler
	scanToolTemplateHandler *orchestratorHandler.ScanToolTemplateHandler
	agentTaskHandler        *orchestratorHandler.AgentTaskHandler

	// 标签系统相关Handler
	tagHandler *tagHandler.TagHandler

	// 调度服务
	schedulerService scheduler.SchedulerService
	// 本地Agent (原系统任务执行器)
	localAgent *local_agent.LocalAgent
	// ETL 处理器
	etlProcessor etl.ResultProcessor
}

// NewRouter 创建路由管理器实例
func NewRouter(db *gorm.DB, redisClient *redis.Client, config *config.Config) *Router {
	// 内部提取需要的配置
	securityConfig := &config.Security

	// 通过 setup.BuildAuthModule 进行认证模块的初始化与聚合输出
	authModule, err := setup.BuildAuthModule(db, redisClient, config)
	if err != nil {
		// 不指定日志类型会默认输出到app.log
		logger.WithFields(map[string]interface{}{
			"path":      "router_manager.NewRouter",
			"operation": "setup",
			"option":    "setup.auth.error",
			"func_name": "router.NewRouter",
			"error":     err.Error(),
		}).Error("认证模块初始化失败")
		// 初始化失败时，直接返回一个基础 Router；调用方可根据返回值判断并处理
		gin.SetMode(gin.ReleaseMode)
		engine := gin.New()
		return &Router{config: config, engine: engine}
	}

	// 通过 setup.BuildSystemRBACModule 初始化系统RBAC模块（角色与权限管理）
	rbacModule := setup.BuildSystemRBACModule(db)

	// 初始化中间件管理器（传入jwtService用于密码版本验证）
	middlewareManager := middleware.NewMiddlewareManager(authModule.SessionService, authModule.RBACService, authModule.JWTService, securityConfig)

	// 初始化处理器(控制器是服务集合,先初始化服务,然后服务装填成控制器)
	loginHandler := authModule.LoginHandler
	logoutHandler := authModule.LogoutHandler
	refreshHandler := authModule.RefreshHandler
	registerHandler := authModule.RegisterHandler
	userHandler := systemHandler.NewUserHandler(authModule.UserService, authModule.PasswordService)
	roleHandler := rbacModule.RoleHandler
	permissionHandler := rbacModule.PermissionHandler
	sessionHandler := systemHandler.NewSessionHandler(authModule.SessionService)

	// 通过 setup.BuildTagSystemModule 初始化标签系统模块
	tagModule := setup.BuildTagSystemModule(db)

	// 通过 setup.BuildAssetModule 初始化资产管理模块
	assetModule := setup.BuildAssetModule(db, tagModule.TagService)

	// 通过 setup.BuildOrchestratorModule 初始化扫描编排器模块
	orchestratorModule := setup.BuildOrchestratorModule(db, config, tagModule.TagService)

	// 通过 setup.BuildAgentModule 初始化 Agent 管理模块（Manager/Monitor/Config/Task 服务聚合）
	// TaskDispatcher 现已完全由 Orchestrator 管理，AgentModule 不再需要注入
	agentModule := setup.BuildAgentModule(db, config, tagModule.TagService)

	// 从 OrchestratorModule 中获取聚合后的处理器
	projectHandler := orchestratorModule.ProjectHandler
	workflowHandler := orchestratorModule.WorkflowHandler
	scanStageHandler := orchestratorModule.ScanStageHandler
	scanToolTemplateHandler := orchestratorModule.ScanToolTemplateHandler
	agentTaskHandler := orchestratorModule.AgentTaskHandler

	// 从 AgentModule 中获取聚合后的 Handler（分组功能已合并到 ManagerService 内部）
	assetRawHandler := assetModule.AssetRawHandler
	agentMgmtHandler := agentModule.AgentHandler
	assetHostHandler := assetModule.AssetHostHandler
	assetNetworkHandler := assetModule.AssetNetworkHandler
	assetPolicyHandler := assetModule.AssetPolicyHandler
	assetWebHandler := assetModule.AssetWebHandler
	assetVulnHandler := assetModule.AssetVulnHandler
	assetUnifiedHandler := assetModule.AssetUnifiedHandler
	assetScanHandler := assetModule.AssetScanHandler

	// 从 TagModule 中获取处理器
	tagHandler := tagModule.TagHandler

	// 创建Gin引擎
	gin.SetMode(gin.ReleaseMode) // 设置为生产模式
	engine := gin.New()

	return &Router{
		config:            config,
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
		agentHandler: agentMgmtHandler,
		// 资产管理相关Handler
		assetRawHandler:     assetRawHandler,
		assetHostHandler:    assetHostHandler,
		assetNetworkHandler: assetNetworkHandler,
		assetPolicyHandler:  assetPolicyHandler,
		assetWebHandler:     assetWebHandler,
		assetVulnHandler:    assetVulnHandler,
		assetUnifiedHandler: assetUnifiedHandler,
		assetScanHandler:    assetScanHandler,

		// 扫描编排器相关Handler
		projectHandler:          projectHandler,
		workflowHandler:         workflowHandler,
		scanStageHandler:        scanStageHandler,
		scanToolTemplateHandler: scanToolTemplateHandler,
		agentTaskHandler:        agentTaskHandler,

		// 标签系统Handler
		tagHandler: tagHandler,

		// 扫描任务调度服务
		schedulerService: orchestratorModule.SchedulerService,
		// 本地Agent
		localAgent: orchestratorModule.LocalAgent,
		// ETL 处理器
		etlProcessor: orchestratorModule.ETLProcessor,
	}
}

// SetupRoutes 设置全局中间件和路由
// 在这里配置调用各个路由模块
func (r *Router) SetupRoutes() {
	// 1) 先注册全局中间件；2) 再注册各模块路由。

	// 1) 全局中间件注册
	r.registerGlobalMiddleware()

	// 2) 路由注册
	r.registerRoutes()
}

// GetEngine 获取Gin引擎实例
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// GetSchedulerService 获取调度服务实例
func (r *Router) GetSchedulerService() scheduler.SchedulerService {
	return r.schedulerService
}

// GetLocalAgent 获取本地Agent实例
func (r *Router) GetLocalAgent() *local_agent.LocalAgent {
	return r.localAgent
}

// GetETLProcessor 获取ETL处理器实例
func (r *Router) GetETLProcessor() etl.ResultProcessor {
	return r.etlProcessor
}

// registerGlobalMiddleware 注册全局中间件（对齐 neoAgent 的风格）
// 设计与原因：
// - 将全局中间件的挂载集中在一个方法中，便于统一管理与测试（只需在此处验证链条顺序）。
// - 保持现有中间件的顺序与行为不变，同时补充 gin.Recovery() 以增强健壮性（如需要可按配置开关）。
func (r *Router) registerGlobalMiddleware() {
	logger.WithFields(map[string]interface{}{
		"path":      "router_manager.registerGlobalMiddleware",
		"operation": "register_global_middleware",
		"option":    "middlewareManager.attach",
		"func_name": "router.registerGlobalMiddleware",
	}).Info("开始注册全局中间件")

	// 系统恢复中间件，防止 panic 直接导致进程崩溃（与 neoAgent 一致的防护策略）
	r.engine.Use(gin.Recovery())

	if r.middlewareManager != nil {
		// CORS 中间件
		r.engine.Use(r.middlewareManager.GinCORSMiddleware())
		// 安全响应头中间件
		r.engine.Use(r.middlewareManager.GinSecurityHeadersMiddleware())
		// 统一日志中间件
		r.engine.Use(r.middlewareManager.GinLoggingMiddleware())
		// 限流中间件
		r.engine.Use(r.middlewareManager.GinRateLimitMiddleware())
	}

	logger.WithFields(map[string]interface{}{
		"path":      "router_manager.registerGlobalMiddleware",
		"operation": "register_global_middleware",
		"option":    "middlewareManager.attach.done",
		"func_name": "router.registerGlobalMiddleware",
	}).Info("全局中间件注册完成")
}

// registerRoutes 注册路由（对齐 neoAgent 的风格）
// 设计与原因：
// - 将“中间件注册”和“各模块路由注册”的步骤分离，提升可维护性与可测试性。
// - 保留现有的分组与路径，不改动对外行为；仅将 SetupRoutes 的实现迁移到此处集中管理。
func (r *Router) registerRoutes() {
	logger.WithFields(map[string]interface{}{
		"path":      "router_manager.registerRoutes",
		"operation": "register_routes",
		"option":    "routes.attach.begin",
		"func_name": "router.registerRoutes",
	}).Info("开始注册路由")

	// 1) 全局中间件注册
	// 全局中间件注册移除函数外，在SetupRoutes中调用

	// 2) API 版本路由组：/api/v1（保持现有风格，不使用前缀配置化）
	api := r.engine.Group("/api")
	v1 := api.Group("/v1")

	// 3) 具体模块路由注册（保持原有调用顺序与权限边界）
	// 公共路由（不需要认证）
	r.setupPublicRoutes(v1)
	// 用户认证路由（需要 JWT 认证）
	r.setupUserRoutes(v1)
	// 管理员路由（需要管理员权限）
	r.setupAdminRoutes(v1)
	// 扫描编排器配置路由（需要 JWT 认证）
	r.setupOrchestratorRoutes(v1)
	// Agent 管理路由（需要 JWT 认证）
	r.setupAgentRoutes(v1)
	// 资产管理路由（需要 JWT 认证）
	r.setupAssetRoutes(v1)
	// 标签系统路由（需要 JWT 认证）
	r.setupTagSystemRoutes(v1)
	// 健康检查路由
	r.setupHealthRoutes(api)

	logger.WithFields(map[string]interface{}{
		"path":      "router_manager.registerRoutes",
		"operation": "register_routes",
		"option":    "routes.attach.done",
		"func_name": "router.registerRoutes",
	}).Info("路由注册完成")
}
