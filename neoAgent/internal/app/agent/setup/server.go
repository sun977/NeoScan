package setup

import (
	"fmt"
	"net/http"
	"time"

	"neoagent/internal/app/agent/middleware"
	"neoagent/internal/app/agent/router"
	"neoagent/internal/config"
)

// SetupServer 初始化服务器模块
func SetupServer(cfg *config.Config) *ServerModule {
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

	return &ServerModule{
		Router:     r,
		HTTPServer: httpServer,
	}
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
