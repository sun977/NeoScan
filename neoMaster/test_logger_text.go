package main

import (
	"fmt"
	"neomaster/internal/config"
	"neomaster/internal/pkg/logger"

	"github.com/sirupsen/logrus"
)

func main() {
	// 配置日志为文本格式输出到控制台（更易读）
	cfg := &config.LogConfig{
		Level:  "info",
		Format: "text", // 使用文本格式，更易读
		Output: "stdout",
		Caller: false, // 关闭调用者信息，输出更简洁
	}

	// 初始化日志管理器
	logger.InitLogger(cfg)

	fmt.Println("=== 日志模块使用示例（文本格式） ===")
	fmt.Println("")

	// 1. 业务操作日志示例
	fmt.Println("1. 业务操作日志（用户登录）:")
	logger.LogBusinessOperation(
		"login",         // 操作类型
		1001,            // 用户ID
		"testuser",      // 用户名
		"192.168.1.100", // 客户端IP
		"req-12345",     // 请求ID
		"success",       // 操作结果
		"用户登录成功",        // 消息
		map[string]interface{}{ // 额外字段
			"login_method": "password",
			"device":       "web",
		},
	)

	fmt.Println("")
	fmt.Println("2. 业务操作日志（登录失败）:")
	logger.LogBusinessOperation(
		"login",
		0, // 未知用户ID
		"unknown",
		"192.168.1.101",
		"req-12346",
		"failed", // 失败状态
		"用户名或密码错误",
		map[string]interface{}{
			"login_method":  "password",
			"attempt_count": 3,
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
		"database",       // 组件
		"startup",        // 事件
		"数据库连接池初始化完成",    // 消息
		logrus.InfoLevel, // 日志级别
		map[string]interface{}{
			"max_connections":  100,
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
		},
	)

	fmt.Println("")
	fmt.Println("=== 日志输出完成 ===")
}
