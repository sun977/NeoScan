package main

import (
	"fmt"
	"log"

	"neomaster/internal/config"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/database"
)

func main() {
	fmt.Println("开始修复用户密码...")

	// 加载配置
	cfg, err := config.LoadConfig("", "development")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	db, err := database.NewMySQLConnection(&cfg.Database.MySQL)
	if err != nil {
		log.Fatalf("MySQL连接失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 创建密码管理器并生成哈希密码
	plainPassword := "admin123"
	passwordManager := auth.NewPasswordManager(nil) // 使用默认配置
	hashedPassword, err := passwordManager.HashPassword(plainPassword)
	if err != nil {
		log.Fatalf("密码哈希失败: %v", err)
	}

	fmt.Printf("✅ 生成密码哈希: %s\n", hashedPassword)

	// 更新admin用户的密码
	result := db.Exec("UPDATE users SET password = ?, password_v = password_v + 1 WHERE username = ?", hashedPassword, "admin")
	if result.Error != nil {
		log.Fatalf("更新密码失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		log.Fatal("没有找到admin用户")
	}

	fmt.Printf("✅ 成功更新了 %d 个用户的密码\n", result.RowsAffected)

	// 验证密码是否正确
	var storedPassword string
	err = db.Raw("SELECT password FROM users WHERE username = ?", "admin").Scan(&storedPassword).Error
	if err != nil {
		log.Fatalf("查询密码失败: %v", err)
	}

	// 验证密码
	isValid, err := passwordManager.VerifyPassword(plainPassword, storedPassword)
	if err != nil {
		log.Fatalf("密码验证失败: %v", err)
	}
	if isValid {
		fmt.Println("✅ 密码验证成功!")
	} else {
		fmt.Println("❌ 密码验证失败!")
	}

	fmt.Println("\n🎉 密码修复完成!")
	fmt.Println("现在可以使用以下凭据登录:")
	fmt.Println("  用户名: admin")
	fmt.Println("  密码: admin123")
}
