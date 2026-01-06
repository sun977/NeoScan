// 项目配置文件加载逻辑
// 1. 确定配置目录：
//    - 环境变量(.env) NEOSCAN_CONFIG_PATH 指定的路径
//    - 或默认 "configs" 目录[二进制web服务文件需与configs同目录]
// 2. 根据环境选择文件：
//    - "production"/"prod" → config.prod.yaml  →  neoscan_prod 数据库
//    - "test"/"testing"    → config.test.yaml  →  neoscan_test 数据库
//    - 其他/默认           → config.yaml  →  neoscan_dev 数据库
// 3. 文件存在性检查：
//    - 如果目标文件不存在，回退到 config.yaml
//    - 确保系统始终有配置文件可用
// 开发：我没有使用.env 文件，而是使用 config.yaml 文件运行的项目

package master

import (
	"context"
	"fmt"
	"log"
	"neomaster/internal/service/asset/etl"
	"neomaster/internal/service/orchestrator/core/scheduler"
	"neomaster/internal/service/orchestrator/local_agent"

	"neomaster/internal/app/master/router"
	"neomaster/internal/config"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// App 应用程序结构体
type App struct {
	router     *router.Router
	db         *gorm.DB
	redis      *redis.Client
	config     *config.Config
	scheduler  scheduler.SchedulerService
	localAgent *local_agent.LocalAgent
	etl        etl.ResultProcessor
}

// NewApp 创建新的应用程序实例
func NewApp() (*App, error) {
	// 加载配置文件，默认使用 development 开发环境，test 是测试环境，prod 是生产环境
	// cfg, err := config.LoadConfig("", "test")
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
		logger.LogBusinessError(err, "", 0, "", "db_connect", "CONNECT", map[string]interface{}{
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
	// 	logger.LogBusinessError(err, "", 0, "", "redis_connect", "CONNECT", map[string]interface{}{
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
			logger.LogBusinessError(err, "", 0, "", "redis_connect", "CONNECT", map[string]interface{}{
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
	// router := router.NewRouter(db, redisClient, cfg.JWT.Secret) 【这里可以直接引入配置文件而不是配置文件部分】
	// router := router.NewRouter(db, redisClient, cfg.Security.JWT.Secret, &cfg.Security)
	router := router.NewRouter(db, redisClient, cfg)

	// 设置路由
	router.SetupRoutes()

	// 初始化调度引擎
	// 通过 Router 获取 OrchestratorModule 中初始化的 SchedulerService
	// 避免重复初始化和多实例问题
	schedulerService := router.GetSchedulerService()
	localAgent := router.GetLocalAgent()
	etlProcessor := router.GetETLProcessor()

	return &App{
		router:     router,
		db:         db,
		redis:      redisClient,
		config:     cfg,
		scheduler:  schedulerService,
		localAgent: localAgent,
		etl:        etlProcessor,
	}, nil
}

// GetRouter 获取路由器实例
func (a *App) GetRouter() *router.Router { // 返回类型使用router包中的Router类型
	return a.router
}

// GetConfig 获取配置实例
func (a *App) GetConfig() *config.Config {
	return a.config
}

// StartScheduler 启动调度引擎及后台服务
func (a *App) StartScheduler(ctx context.Context) {
	if a.scheduler != nil {
		a.scheduler.Start(ctx)
	}
	if a.localAgent != nil {
		a.localAgent.Start()
	}
	if a.etl != nil {
		a.etl.Start(ctx)
	}
}

// StopScheduler 停止调度引擎及后台服务
func (a *App) StopScheduler() {
	if a.scheduler != nil {
		a.scheduler.Stop()
	}
	if a.localAgent != nil {
		a.localAgent.Stop()
	}
	if a.etl != nil {
		a.etl.Stop()
	}
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
