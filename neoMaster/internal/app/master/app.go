package master

import (
	"fmt"
	"log"

	"neomaster/internal/config"
	"neomaster/internal/pkg/database"

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

	// 初始化数据库连接
	db, err := database.NewMySQLConnection(&cfg.Database.MySQL)
	if err != nil {
		log.Printf("Warning: Failed to connect to MySQL: %v", err)
		// 在开发阶段，如果数据库连接失败，我们继续运行但使用nil
		db = nil
	}

	// 初始化Redis连接
	redisClient, err := database.NewRedisConnection(&cfg.Database.Redis)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		// 在开发阶段，如果Redis连接失败，我们继续运行但使用nil
		redisClient = nil
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
