package master

import (
	"fmt"
)

// App 应用程序结构体
type App struct {
	router *Router
}

// NewApp 创建新的应用程序实例
func NewApp() (*App, error) {
	// 初始化路由器（传入nil作为占位符，实际项目中需要连接真实数据库和Redis）
	router := NewRouter(nil, nil, "your-jwt-secret-key")

	// 设置路由
	router.SetupRoutes()

	return &App{
		router: router,
	}, nil
}

// GetRouter 获取路由器实例
func (a *App) GetRouter() *Router {
	return a.router
}

// Start 启动应用程序（可选方法，用于未来扩展）
func (a *App) Start() error {
	// 这里可以添加应用程序启动逻辑
	// 比如数据库连接、Redis连接等
	fmt.Println("Application started successfully")
	return nil
}

// Stop 停止应用程序（可选方法，用于未来扩展）
func (a *App) Stop() error {
	// 这里可以添加应用程序停止逻辑
	// 比如关闭数据库连接、Redis连接等
	fmt.Println("Application stopped successfully")
	return nil
}