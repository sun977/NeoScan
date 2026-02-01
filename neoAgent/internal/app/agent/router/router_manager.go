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
	"neoagent/internal/app/agent/middleware"
	"neoagent/internal/handler/client"
	"neoagent/internal/handler/monitor"
	handlerTask "neoagent/internal/handler/task"
	"neoagent/internal/pkg/logger"
	serviceTask "neoagent/internal/service/task"

	"github.com/gin-gonic/gin"
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
	engine *gin.Engine
	config *RouterConfig
	logger *logger.LoggerManager

	// 中间件
	authMiddleware      *middleware.AuthMiddleware
	loggingMiddleware   *middleware.LoggingMiddleware
	corsMiddleware      *middleware.CORSMiddleware
	rateLimitMiddleware *middleware.RateLimitMiddleware

	// 处理器
	taskHandler          handlerTask.AgentTaskHandler
	monitorHandler       monitor.AgentMonitorHandler
	communicationHandler client.MasterCommunicationHandler
}

// NewRouter 创建新的路由器
func NewRouter(config *RouterConfig, taskService serviceTask.AgentTaskService) *Router {
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
	router.initHandlers(taskService)

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
func (r *Router) initHandlers(taskService serviceTask.AgentTaskService) {
	// TODO: 初始化各种处理器
	// 1. 任务处理器
	// 2. 监控处理器
	// 3. 通信处理器

	// 这里使用占位符实现，实际应该通过依赖注入
	r.taskHandler = handlerTask.NewAgentTaskHandler(taskService)
	r.monitorHandler = monitor.NewAgentMonitorHandler()
	r.communicationHandler = client.NewMasterCommunicationHandler()
}

// registerRoutes 注册路由
func (r *Router) registerRoutes() {
	logger.Info("开始注册路由")

	// 注册全局中间件
	r.registerGlobalMiddleware()

	// 注册健康检查路由
	r.setupHealthRoutes()

	// 注册API路由组
	apiGroup := r.engine.Group(r.config.Prefix + "/" + r.config.APIVersion)

	// 注册任务管理路由
	setupTaskRoutes(apiGroup, r.taskHandler)

	// 注册监控路由
	setupMonitorRoutes(apiGroup)

	// 注册通信路由
	setupCommunicationRoutes(r.engine)

	logger.Info("路由注册完成")
}

// registerGlobalMiddleware 注册全局中间件
func (r *Router) registerGlobalMiddleware() {
	logger.Info("开始注册全局中间件")

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

	logger.Info("全局中间件注册完成")
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
