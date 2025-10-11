package main

import (
	"fmt"
	"neomaster/internal/config"
	"neomaster/internal/pkg/logger"
	"github.com/sirupsen/logrus"
)

func main() {
	// 配置日志为JSON格式输出到控制台
	cfg := &config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
		Caller: true,
	}

	// 初始化日志管理器
	logger.InitLogger(cfg)

	fmt.Println("=== 日志模块使用示例 ===")
	fmt.Println("")

	// 1. 业务操作日志示例
	fmt.Println("1. 业务操作日志（用户登录）:")
	logger.LogBusinessOperation(
		"login",                    // 操作类型
		1001,                       // 用户ID
		"testuser",                 // 用户名
		"192.168.1.100",            // 客户端IP
		"req-12345",                // 请求ID
		"success",                  // 操作结果
		"用户登录成功",                // 消息
		map[string]interface{}{     // 额外字段
			"login_method": "password",
			"device":       "web",
			"browser":      "Chrome",
		},
	)

	fmt.Println("")
	fmt.Println("2. 业务操作日志（用户注册）:")
	logger.LogBusinessOperation(
		"register",
		1002,
		"newuser",
		"192.168.1.101",
		"req-12346",
		"success",
		"用户注册成功",
		map[string]interface{}{
			"email":        "newuser@example.com",
			"registration": "email",
		},
	)

	fmt.Println("")
	fmt.Println("3. 错误日志示例:")
	logger.LogError(
		fmt.Errorf("数据库连接失败: connection timeout"),
		"req-12347",
		1001,
		"192.168.1.100",
		"/api/users",
		"GET",
		map[string]interface{}{
			"database": "mysql",
			"timeout":  "30s",
		},
	)

	fmt.Println("")
	fmt.Println("4. 系统事件日志示例:")
	logger.LogSystemEvent(
		"database",     // 组件
		"connection",   // 事件
		"数据库连接池初始化完成", // 消息
		logrus.InfoLevel, // 日志级别
		map[string]interface{}{
			"max_connections": 100,
			"idle_connections": 10,
		},
	)

	fmt.Println("")
	fmt.Println("5. 审计日志示例:")
	logger.LogAuditOperation(
		1001,                    // 用户ID
		"admin",                 // 用户名
		"delete_user",           // 操作
		"user:1002",             // 资源
		"success",               // 结果
		"192.168.1.100",         // 客户端IP
		"Mozilla/5.0 Chrome/91", // User-Agent
		"req-12348",             // 请求ID
		map[string]interface{}{
			"reason": "违规用户",
			"admin_level": "super",
		},
	)

	fmt.Println("")
	fmt.Println("=== 日志输出完成 ===")
}