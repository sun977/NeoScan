package master

import (
	"fmt"
	"log"

	"neomaster/internal/config"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// App 应用程序结构体
type App struct {
	router *Router
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
}

// NewApp 创建新的应用程序实例
func NewApp() (*App, error) {
	// 加载配置
	cfg, err := config.LoadConfig("", "development")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 初始化日志管理器
	_, err = logger.InitLogger(&cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// 记录应用启动日志
	logger.LogBusinessOperation("app_start", 0, "", "", "", "success", "NeoMaster application starting", map[string]interface{}{
		"version":   "1.0.0",
		"env":       "development",
		"timestamp": logger.NowFormatted(),
	})

	// 初始化数据库连接
	db, err := database.NewMySQLConnection(&cfg.Database.MySQL)
	if err != nil {
		log.Printf("Warning: Failed to connect to MySQL: %v", err)
		// 记录数据库连接失败日志
		logger.LogError(err, "", 0, "", "db_connect", "CONNECT", map[string]interface{}{
			"operation": "mysql_connect",
			"host":      cfg.Database.MySQL.Host,
			"port":      cfg.Database.MySQL.Port,
			"database":  cfg.Database.MySQL.Database,
			"timestamp": logger.NowFormatted(),
		})
		// 在开发阶段，如果数据库连接失败，我们继续运行但使用nil
		db = nil
	} else {
		// 记录数据库连接成功日志
		logger.LogBusinessOperation("db_connect", 0, "", "", "", "success", "MySQL database connected successfully", map[string]interface{}{
			"operation": "mysql_connect",
			"host":      cfg.Database.MySQL.Host,
			"database":  cfg.Database.MySQL.Database,
			"timestamp": logger.NowFormatted(),
		})
	}

	// // 初始化Redis连接
	// redisClient, err := database.NewRedisConnection(&cfg.Database.Redis)
	// if err != nil {
	// 	log.Printf("Warning: Failed to connect to Redis: %v", err)
	// 	// 记录Redis连接失败日志
	// 	logger.LogError(err, "", 0, "", "redis_connect", "CONNECT", map[string]interface{}{
	// 		"operation": "redis_connect",
	// 		"host":      cfg.Database.Redis.Host,
	// 		"port":      cfg.Database.Redis.Port,
	// 		"database":  cfg.Database.Redis.Database,
	// 		"timestamp": logger.NowFormatted(),
	// 	})
	// 	// 在开发阶段，如果Redis连接失败，我们继续运行但使用nil
	// 	redisClient = nil
	// } else {
	// 	// 记录Redis连接成功日志
	// 	logger.LogBusinessOperation("redis_connect", 0, "", "", "", "success", "Redis connected successfully", map[string]interface{}{
	// 		"operation": "redis_connect",
	// 		"host":      cfg.Database.Redis.Host,
	// 		"database":  cfg.Database.Redis.Database,
	// 		"timestamp": logger.NowFormatted(),
	// 	})
	// }

	// 初始化Redis连接【后续待补充使用内存存储,目前默认使用redis】
	var redisClient *redis.Client
	if cfg.Session.Store != "memory" {
		redisClient, err = database.NewRedisConnection(&cfg.Database.Redis)
		if err != nil {
			log.Printf("Warning: Failed to connect to Redis: %v", err)
			// 记录Redis连接失败日志
			logger.LogError(err, "", 0, "", "redis_connect", "CONNECT", map[string]interface{}{
				"operation": "redis_connect",
				"host":      cfg.Database.Redis.Host,
				"port":      cfg.Database.Redis.Port,
				"database":  cfg.Database.Redis.Database,
				"timestamp": logger.NowFormatted(),
			})
			// 在开发阶段，如果Redis连接失败，我们继续运行但使用nil
			redisClient = nil
		} else {
			// 记录Redis连接成功日志
			logger.LogBusinessOperation("redis_connect", 0, "", "", "", "success", "Redis connected successfully", map[string]interface{}{
				"operation": "redis_connect",
				"host":      cfg.Database.Redis.Host,
				"database":  cfg.Database.Redis.Database,
				"timestamp": logger.NowFormatted(),
			})
		}
	} else {
		log.Printf("Using memory session store")
		logger.LogBusinessOperation("session_store", 0, "", "", "", "success", "Using memory session store", map[string]interface{}{
			"operation":  "session_store_init",
			"store_type": "memory",
			"timestamp":  logger.NowFormatted(),
		})
	}

	// 初始化路由器
	router := NewRouter(db, redisClient, cfg.JWT.Secret)

	// 设置路由
	router.SetupRoutes()

	return &App{
		router: router,
		db:     db,
		redis:  redisClient,
		config: cfg,
	}, nil
}

// GetRouter 获取路由器实例
func (a *App) GetRouter() *Router {
	return a.router
}

// GetConfig 获取配置实例
func (a *App) GetConfig() *config.Config {
	return a.config
}

// Start 启动应用程序（可选方法，用于未来扩展）
func (a *App) Start() error {
	// 这里可以添加应用程序启动逻辑
	fmt.Println("Application started successfully")
	return nil
}

// Stop 停止应用程序（可选方法，用于未来扩展）
func (a *App) Stop() error {
	// 这里可以添加应用程序停止逻辑
	// 关闭数据库连接
	if a.db != nil {
		if sqlDB, err := a.db.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// 关闭Redis连接
	if a.redis != nil {
		a.redis.Close()
	}

	fmt.Println("Application stopped successfully")
	return nil
}
