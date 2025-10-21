/**
 * Agent端路由注册
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端路由注册，统一管理所有路由
 * @func: 占位符实现，待后续完善
 */
package router

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"neoagent/internal/app/agent/middleware"
	"neoagent/internal/handler/communication"
	"neoagent/internal/handler/monitor"
	"neoagent/internal/handler/task"
	"neoagent/internal/pkg/logger"
)

// RouterConfig 路由配置
type RouterConfig struct {
	// 是否启用调试模式
	Debug bool `json:"debug"`
	
	// API版本
	APIVersion string `json:"api_version"`
	
	// 路由前缀
	Prefix string `json:"prefix"`
	
	// 是否启用中间件
	EnableMiddleware bool `json:"enable_middleware"`
	
	// 中间件配置
	MiddlewareConfig *MiddlewareConfig `json:"middleware_config"`
}

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	// 认证中间件配置
	Auth *middleware.AuthConfig `json:"auth"`
	
	// 日志中间件配置
	Logging *middleware.LoggingConfig `json:"logging"`
	
	// CORS中间件配置
	CORS *middleware.CORSConfig `json:"cors"`
	
	// 限流中间件配置
	RateLimit *middleware.RateLimitConfig `json:"rate_limit"`
}

// Router Agent路由器
type Router struct {
	engine   *gin.Engine
	config   *RouterConfig
	logger   *logger.LoggerManager
	
	// 中间件
	authMiddleware      *middleware.AuthMiddleware
	loggingMiddleware   *middleware.LoggingMiddleware
	corsMiddleware      *middleware.CORSMiddleware
	rateLimitMiddleware *middleware.RateLimitMiddleware
	
	// 处理器
	taskHandler         task.AgentTaskHandler
	monitorHandler      monitor.AgentMonitorHandler
	communicationHandler communication.MasterCommunicationHandler
}

// NewRouter 创建新的路由器
func NewRouter(config *RouterConfig) *Router {
	if config == nil {
		config = &RouterConfig{
			Debug:            false,
			APIVersion:       "v1",
			Prefix:           "/api",
			EnableMiddleware: true,
			MiddlewareConfig: &MiddlewareConfig{},
		}
	}
	
	// 设置Gin模式
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	
	engine := gin.New()
	
	router := &Router{
		engine: engine,
		config: config,
		logger: logger.LoggerInstance,
	}
	
	// 初始化中间件
	if config.EnableMiddleware {
		router.initMiddleware()
	}
	
	// 初始化处理器
	router.initHandlers()
	
	// 注册路由
	router.registerRoutes()
	
	return router
}

// initMiddleware 初始化中间件
func (r *Router) initMiddleware() {
	// TODO: 初始化各种中间件
	// 1. 认证中间件
	// 2. 日志中间件
	// 3. CORS中间件
	// 4. 限流中间件
	
	if r.config.MiddlewareConfig.Auth != nil {
		r.authMiddleware = middleware.NewAuthMiddleware(r.config.MiddlewareConfig.Auth)
	}
	
	if r.config.MiddlewareConfig.Logging != nil {
		r.loggingMiddleware = middleware.NewLoggingMiddleware(r.config.MiddlewareConfig.Logging)
	}
	
	if r.config.MiddlewareConfig.CORS != nil {
		r.corsMiddleware = middleware.NewCORSMiddleware(r.config.MiddlewareConfig.CORS)
	}
	
	if r.config.MiddlewareConfig.RateLimit != nil {
		r.rateLimitMiddleware = middleware.NewRateLimitMiddleware(r.config.MiddlewareConfig.RateLimit)
	}
}

// initHandlers 初始化处理器
func (r *Router) initHandlers() {
	// TODO: 初始化各种处理器
	// 1. 任务处理器
	// 2. 监控处理器
	// 3. 通信处理器
	
	// 这里使用占位符实现，实际应该通过依赖注入
	r.taskHandler = task.NewAgentTaskHandler()
	r.monitorHandler = monitor.NewAgentMonitorHandler()
	r.communicationHandler = communication.NewMasterCommunicationHandler()
}

// registerRoutes 注册路由
func (r *Router) registerRoutes() {
	// TODO: 注册所有路由
	// 1. 健康检查路由
	// 2. 任务管理路由
	// 3. 监控路由
	// 4. 通信路由
	
	// 注册全局中间件
	r.registerGlobalMiddleware()
	
	// 注册健康检查路由
	r.registerHealthRoutes()
	
	// 注册API路由组
	apiGroup := r.engine.Group(r.config.Prefix + "/" + r.config.APIVersion)
	
	// 注册任务管理路由
	r.registerTaskRoutes(apiGroup)
	
	// 注册监控路由
	r.registerMonitorRoutes(apiGroup)
	
	// 注册通信路由
	r.registerCommunicationRoutes(apiGroup)
}

// registerGlobalMiddleware 注册全局中间件
func (r *Router) registerGlobalMiddleware() {
	// 恢复中间件
	r.engine.Use(gin.Recovery())
	
	// CORS中间件
	if r.corsMiddleware != nil {
		r.engine.Use(r.corsMiddleware.Handler())
	}
	
	// 日志中间件
	if r.loggingMiddleware != nil {
		r.engine.Use(r.loggingMiddleware.Handler())
	}
	
	// 限流中间件
	if r.rateLimitMiddleware != nil {
		r.engine.Use(r.rateLimitMiddleware.Handler())
	}
}

// registerHealthRoutes 注册健康检查路由
func (r *Router) registerHealthRoutes() {
	// 健康检查路由（不需要认证）
	r.engine.GET("/health", r.handleHealth)
	r.engine.GET("/ping", r.handlePing)
	r.engine.GET("/version", r.handleVersion)
}

// registerTaskRoutes 注册任务管理路由
func (r *Router) registerTaskRoutes(group *gin.RouterGroup) {
	// 任务路由组（需要认证）
	taskGroup := group.Group("/tasks")
	if r.authMiddleware != nil {
		taskGroup.Use(r.authMiddleware.Handler())
	}
	
	// 任务管理接口
	taskGroup.POST("", r.taskHandler.CreateTask)
	taskGroup.GET("/:id", r.taskHandler.GetTask)
	taskGroup.DELETE("/:id", r.taskHandler.DeleteTask)
	taskGroup.GET("", r.taskHandler.GetTaskList)
	
	// 任务控制接口
	taskGroup.POST("/:id/start", r.taskHandler.StartTask)
	taskGroup.POST("/:id/stop", r.taskHandler.StopTask)
	taskGroup.POST("/:id/pause", r.taskHandler.PauseTask)
	taskGroup.POST("/:id/resume", r.taskHandler.ResumeTask)
	
	// 任务结果接口
	taskGroup.GET("/:id/result", r.taskHandler.GetTaskResult)
	taskGroup.GET("/:id/logs", r.taskHandler.GetTaskLog)
	taskGroup.GET("/:id/status", r.taskHandler.GetTaskStatus)
}

// registerMonitorRoutes 注册监控路由
func (r *Router) registerMonitorRoutes(group *gin.RouterGroup) {
	// 监控路由组（需要认证）
	monitorGroup := group.Group("/monitor")
	if r.authMiddleware != nil {
		monitorGroup.Use(r.authMiddleware.Handler())
	}
	
	// 性能指标接口
	monitorGroup.GET("/metrics", r.monitorHandler.GetPerformanceMetrics)
	monitorGroup.GET("/system", r.monitorHandler.GetSystemInfo)
	monitorGroup.GET("/resources", r.monitorHandler.GetResourceUsage)
	
	// 健康检查接口
	monitorGroup.GET("/health", r.monitorHandler.GetHealthStatus)
	monitorGroup.POST("/health/check", r.monitorHandler.PerformHealthCheck)
	
	// 告警接口
	monitorGroup.GET("/alerts", r.monitorHandler.GetAlerts)
	monitorGroup.POST("/alerts", r.monitorHandler.CreateAlert)
	monitorGroup.PUT("/alerts/:id/ack", r.monitorHandler.AcknowledgeAlert)
	
	// 日志接口
	monitorGroup.GET("/logs", r.monitorHandler.GetLogs)
	monitorGroup.PUT("/logs/level", r.monitorHandler.SetLogLevel)
	monitorGroup.POST("/logs/rotate", r.monitorHandler.RotateLogs)
}

// registerCommunicationRoutes 注册通信路由
func (r *Router) registerCommunicationRoutes(group *gin.RouterGroup) {
	// 通信路由组（需要认证）
	commGroup := group.Group("/communication")
	if r.authMiddleware != nil {
		commGroup.Use(r.authMiddleware.Handler())
	}
	
	// Agent注册和认证
	commGroup.POST("/register", r.communicationHandler.RegisterToMaster)
	commGroup.POST("/authenticate", r.communicationHandler.AuthenticateWithMaster)
	
	// 心跳和状态同步
	commGroup.POST("/heartbeat", r.communicationHandler.SendHeartbeat)
	commGroup.PUT("/status", r.communicationHandler.SyncStatus)
	
	// 数据上报
	commGroup.POST("/report/metrics", r.communicationHandler.ReportMetrics)
	commGroup.POST("/report/results", r.communicationHandler.ReportTaskResult)
	commGroup.POST("/report/alerts", r.communicationHandler.ReportAlert)
	
	// 配置同步
	commGroup.GET("/config", r.communicationHandler.SyncConfig)
	commGroup.PUT("/config", r.communicationHandler.ApplyConfig)
	
	// 命令接收和响应
	commGroup.GET("/commands", r.communicationHandler.ReceiveCommand)
	commGroup.POST("/commands/:id/response", r.communicationHandler.SendCommandResponse)
	
	// 连接管理
	commGroup.POST("/connect", r.communicationHandler.CheckConnection)
	commGroup.POST("/reconnect", r.communicationHandler.ReconnectToMaster)
}

// 健康检查处理器
func (r *Router) handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"service":   "neoagent",
	})
}

// Ping处理器
func (r *Router) handlePing(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
		"timestamp": time.Now().Unix(),
	})
}

// 版本处理器
func (r *Router) handleVersion(c *gin.Context) {
	c.JSON(200, gin.H{
		"version":    "1.0.0",
		"build_time": "2025-01-14",
		"git_commit": "unknown",
	})
}

// GetEngine 获取Gin引擎
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// UpdateConfig 更新路由配置
func (r *Router) UpdateConfig(config *RouterConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	r.config = config
	
	// 更新中间件配置
	if r.authMiddleware != nil && config.MiddlewareConfig.Auth != nil {
		r.authMiddleware.UpdateConfig(config.MiddlewareConfig.Auth)
	}
	
	if r.loggingMiddleware != nil && config.MiddlewareConfig.Logging != nil {
		r.loggingMiddleware.UpdateConfig(config.MiddlewareConfig.Logging)
	}
	
	if r.corsMiddleware != nil && config.MiddlewareConfig.CORS != nil {
		r.corsMiddleware.UpdateConfig(config.MiddlewareConfig.CORS)
	}
	
	if r.rateLimitMiddleware != nil && config.MiddlewareConfig.RateLimit != nil {
		r.rateLimitMiddleware.UpdateConfig(config.MiddlewareConfig.RateLimit)
	}
	
	logger.Info("Router config updated")
	
	return nil
}

// GetConfig 获取当前配置
func (r *Router) GetConfig() *RouterConfig {
	return r.config
}