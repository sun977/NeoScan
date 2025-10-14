/*
 * 数据库迁移工具
 * @author: Sun977
 * @date: 2025.10.14
 * @description: 执行数据库自动迁移，创建所有必要的表结构
 * @func: 自动迁移所有模型，确保数据库结构与代码模型一致
 */

package main

import (
	"fmt"
	"log"
	"os"

	"neomaster/internal/config"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"

	// 导入所有需要迁移的模型
	agentModel "neomaster/internal/model/agent"
	orchestratorModel "neomaster/internal/model/orchestrator"
	systemModel "neomaster/internal/model/system"

	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("", "development")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志管理器
	_, err = logger.InitLogger(&cfg.Log)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// 连接数据库
	db, err := database.NewMySQLConnection(&cfg.Database.MySQL)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// 获取底层数据库连接以便关闭
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	fmt.Println("开始执行数据库迁移...")

	// 执行自动迁移
	err = db.AutoMigrate(
		// 系统模型
		&systemModel.User{},
		&systemModel.Role{},
		&systemModel.Permission{},
		&systemModel.UserRole{},
		&systemModel.RolePermission{},

		// Agent模型
		&agentModel.Agent{},
		&agentModel.AgentVersion{},
		&agentModel.AgentConfig{},
		&agentModel.AgentMetrics{},
		&agentModel.AgentGroup{},
		&agentModel.AgentGroupMember{},

		// 编排器模型
		&orchestratorModel.ScanRule{},
		&orchestratorModel.ProjectConfig{},
		&orchestratorModel.WorkflowConfig{},
		&orchestratorModel.ScanTool{},
	)

	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	fmt.Println("✅ 数据库迁移完成！")

	// 验证表是否创建成功
	if err := verifyTables(db); err != nil {
		log.Printf("⚠️ 表验证失败: %v", err)
		os.Exit(1)
	}

	fmt.Println("✅ 所有表结构验证通过！")
}

// verifyTables 验证关键表是否存在
func verifyTables(db *gorm.DB) error {
	tables := []string{
		"users", "roles", "permissions", "user_roles", "role_permissions",
		"agents", "agent_versions", "agent_configs", "agent_metrics",
		"agent_groups", "agent_group_members",
		"scan_rules", "project_configs", "workflow_configs", "scan_tools",
	}

	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			return fmt.Errorf("表 %s 不存在", table)
		}
		fmt.Printf("✓ 表 %s 存在\n", table)
	}

	return nil
}
