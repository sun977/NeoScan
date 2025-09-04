package test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"neomaster/internal/config"
	"neomaster/internal/pkg/database"
)

func TestMySQLConnection(t *testing.T) {
	fmt.Println("开始测试MySQL连接...")

	// 加载配置 - 使用正确的配置路径
	configPath := filepath.Join("..", "configs")
	if _, err := os.Stat("configs"); err == nil {
		configPath = "configs"
	}
	cfg, err := config.LoadConfig(configPath, "development")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	fmt.Printf("MySQL配置信息:\n")
	fmt.Printf("  Host: %s\n", cfg.Database.MySQL.Host)
	fmt.Printf("  Port: %d\n", cfg.Database.MySQL.Port)
	fmt.Printf("  Username: %s\n", cfg.Database.MySQL.Username)
	fmt.Printf("  Database: %s\n", cfg.Database.MySQL.Database)
	fmt.Printf("  Charset: %s\n", cfg.Database.MySQL.Charset)

	// 尝试连接MySQL
	fmt.Println("\n尝试连接MySQL数据库...")
	db, err := database.NewMySQLConnection(&cfg.Database.MySQL)
	if err != nil {
		t.Fatalf("MySQL连接失败: %v", err)
	}

	fmt.Println("✅ MySQL连接成功!")

	// 测试数据库操作
	fmt.Println("\n测试数据库操作...")
	var version string
	err = db.Raw("SELECT VERSION()").Scan(&version).Error
	if err != nil {
		t.Fatalf("查询MySQL版本失败: %v", err)
	}

	fmt.Printf("✅ MySQL版本: %s\n", version)

	// 测试数据库是否存在
	fmt.Println("\n检查数据库是否存在...")
	var dbExists int
	err = db.Raw("SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", cfg.Database.MySQL.Database).Scan(&dbExists).Error
	if err != nil {
		t.Fatalf("检查数据库存在性失败: %v", err)
	}

	if dbExists > 0 {
		fmt.Printf("✅ 数据库 '%s' 存在\n", cfg.Database.MySQL.Database)
	} else {
		fmt.Printf("❌ 数据库 '%s' 不存在\n", cfg.Database.MySQL.Database)
	}

	// 测试表是否存在
	fmt.Println("\n检查用户表是否存在...")
	var tableExists int
	err = db.Raw("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = 'users'", cfg.Database.MySQL.Database).Scan(&tableExists).Error
	if err != nil {
		t.Fatalf("检查用户表存在性失败: %v", err)
	}

	if tableExists > 0 {
		fmt.Println("✅ 用户表存在")
		
		// 查询用户数量
		var userCount int64
		err = db.Raw("SELECT COUNT(*) FROM users").Scan(&userCount).Error
		if err != nil {
			log.Printf("查询用户数量失败: %v", err)
		} else {
			fmt.Printf("✅ 用户表中有 %d 条记录\n", userCount)
		}
	} else {
		fmt.Println("❌ 用户表不存在")
	}

	fmt.Println("\n🎉 MySQL连接测试完成!")
}