/**
 * Agent应用程序核心逻辑
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent应用的核心逻辑，负责初始化各种组件和服务
 * @architecture: 参考Master的架构模式，将应用逻辑从main函数中分离
 */

package agent

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"neoagent/internal/app/agent/middleware"
	"neoagent/internal/app/agent/router"
	"neoagent/internal/config"
	"neoagent/internal/pkg/logger"
)

// App Agent应用程序结构体
type App struct {
	router     *router.Router
	httpServer *http.Server
	config     *config.Config
	logger     *logger.LoggerManager
}

// NewApp 创建新的Agent应用程序实例
func NewApp() (*App, error) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 初始化日志管理器
	loggerManager, err := logger.InitLogger(cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	// 设置全局日志实例
	logger.LoggerInstance = loggerManager

	// 记录应用启动日志
	logger.Info("NeoAgent application initializing...")

	// 创建路由器配置
	routerConfig := &router.RouterConfig{
		Debug:            cfg.App.Debug,
		APIVersion:       cfg.Server.APIVersion,
		Prefix:           cfg.Server.Prefix,
		EnableMiddleware: true,
		MiddlewareConfig: createMiddlewareConfig(cfg),
	}

	r := router.NewRouter(routerConfig)

	// 初始化HTTP服务器
	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        r.GetEngine(),
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(cfg.Server.IdleTimeout) * time.Second,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	return &App{
		router:     r,
		httpServer: httpServer,
		config:     cfg,
		logger:     loggerManager,
	}, nil
}

// createMiddlewareConfig 创建中间件配置
// 将全局配置转换为中间件特定的配置结构体
func createMiddlewareConfig(cfg *config.Config) *router.MiddlewareConfig {
	middlewareConfig := &router.MiddlewareConfig{}

	// 检查并设置认证中间件配置
	if cfg.Middleware != nil && cfg.Middleware.Auth != nil {
		middlewareConfig.Auth = &middleware.AuthConfig{
			APIKey:       "", // 从环境变量或配置文件中获取
			APIKeyHeader: "X-API-Key",
			AuthMethod:   cfg.Middleware.Auth.AuthMethod,
			WhitelistIPs: cfg.Middleware.Auth.WhitelistIPs,
			SkipPaths:    cfg.Middleware.Auth.SkipPaths,
		}
	} else {
		// 默认认证配置
		middlewareConfig.Auth = &middleware.AuthConfig{
			APIKeyHeader: "X-API-Key",
			AuthMethod:   "api_key",
			SkipPaths:    []string{"/health", "/ping", "/metrics"},
		}
	}

	// 检查并设置日志中间件配置
	if cfg.Middleware != nil && cfg.Middleware.Logging != nil {
		middlewareConfig.Logging = &middleware.LoggingConfig{
			EnableRequestLog:     cfg.Middleware.Logging.EnableRequestLog,
			EnableResponseLog:    cfg.Middleware.Logging.EnableResponseLog,
			LogRequestBody:       cfg.Middleware.Logging.LogRequestBody,
			LogResponseBody:      cfg.Middleware.Logging.LogResponseBody,
			SlowRequestThreshold: cfg.Middleware.Logging.SlowRequestThreshold,
			MaxRequestBodySize:   cfg.Middleware.Logging.MaxBodySize,
			MaxResponseBodySize:  cfg.Middleware.Logging.MaxBodySize,
			SkipPaths:            cfg.Middleware.Logging.SkipPaths,
		}
	} else {
		// 默认日志配置
		middlewareConfig.Logging = &middleware.LoggingConfig{
			EnableRequestLog:     true,
			EnableResponseLog:    false,
			LogRequestBody:       false,
			LogResponseBody:      false,
			SlowRequestThreshold: 2 * time.Second,
			MaxRequestBodySize:   1024 * 1024, // 1MB
			MaxResponseBodySize:  1024 * 1024, // 1MB
			SkipPaths:            []string{"/health", "/ping"},
		}
	}

	// 检查并设置CORS中间件配置
	if cfg.Middleware != nil && cfg.Middleware.CORS != nil {
		middlewareConfig.CORS = &middleware.CORSConfig{
			Enabled:          cfg.Middleware.CORS.Enabled,
			AllowAllOrigins:  cfg.Middleware.CORS.AllowAllOrigins,
			AllowOrigins:     cfg.Middleware.CORS.AllowOrigins,
			AllowMethods:     cfg.Middleware.CORS.AllowMethods,
			AllowHeaders:     cfg.Middleware.CORS.AllowHeaders,
			ExposeHeaders:    cfg.Middleware.CORS.ExposeHeaders,
			AllowCredentials: cfg.Middleware.CORS.AllowCredentials,
			MaxAge:           time.Duration(cfg.Middleware.CORS.MaxAge) * time.Second,
		}
	} else {
		// 默认CORS配置
		middlewareConfig.CORS = &middleware.CORSConfig{
			Enabled:         true,
			AllowAllOrigins: true,
			AllowOrigins:    []string{"*"},
			AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:    []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
			MaxAge:          12 * time.Hour,
		}
	}

	// 检查并设置限流中间件配置
	if cfg.Middleware != nil && cfg.Middleware.RateLimit != nil {
		middlewareConfig.RateLimit = &middleware.RateLimitConfig{
			RequestsPerSecond: cfg.Middleware.RateLimit.RequestsPerSecond,
			BurstSize:         cfg.Middleware.RateLimit.BurstSize,
		}
	} else {
		// 默认限流配置
		middlewareConfig.RateLimit = &middleware.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         200,
		}
	}

	return middlewareConfig
}

// GetRouter 获取路由器实例
func (a *App) GetRouter() *router.Router {
	return a.router
}

// GetConfig 获取配置实例
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetHTTPServer 获取HTTP服务器实例
func (a *App) GetHTTPServer() *http.Server {
	return a.httpServer
}

// Start 启动Agent应用程序
func (a *App) Start() error {
	logger.Info("Starting NeoAgent server...")

	// 启动HTTP服务器
	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start HTTP server: ", err)
		}
	}()

	logger.Infof("NeoAgent started successfully on port %d", a.config.Server.Port)

	// TODO: 启动其他后台服务
	// 1. 任务执行器管理器
	// 2. 监控数据收集器
	// 3. 与Master的通信客户端
	// 4. 心跳服务

	return nil
}

// Stop 停止Agent应用程序
func (a *App) Stop(ctx context.Context) error {
	logger.Info("Stopping NeoAgent server...")

	// 停止HTTP服务器
	if err := a.httpServer.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown HTTP server: ", err)
		return err
	}

	// TODO: 停止其他服务
	// 1. 停止任务执行器
	// 2. 停止监控服务
	// 3. 停止与Master的连接

	logger.Info("NeoAgent stopped successfully")
	return nil
}
