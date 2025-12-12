/*
*
  - 数据库迁移工具
  - @author: Sun977
  - @date: 2025.10.15
  - @description: 数据库模型迁移和测试数据初始化工具
  - @usage: go run main.go -env=test -seed=true -drop=true
    -drop
    是否先删除表（危险操作）
    -env string
    环境标识 (test, dev, prod) (default "test")
    -seed
    是否填充测试数据 (default true)
    -verbose
    是否显示详细日志

示例:
main.exe -env=test -seed=true    # 测试环境迁移并填充数据
main.exe -env=prod -seed=false   # 生产环境仅迁移表结构
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"neomaster/internal/model/orchestrator"
	"os"
	"time"

	"neomaster/internal/config"
	"neomaster/internal/model/agent"
	"neomaster/internal/model/orchestrator_drop"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// MigrateOptions 迁移选项配置
type MigrateOptions struct {
	Environment string // 环境标识: test, dev, prod
	SeedData    bool   // 是否填充测试数据
	DropFirst   bool   // 是否先删除表（危险操作）
	Verbose     bool   // 是否显示详细日志
}

// DataSeeder 测试数据填充器
// 遵循"好品味"原则：简洁的数据结构，无特殊情况
type DataSeeder struct {
	db  *gorm.DB
	env string
	log *logger.LoggerManager
}

// Fields 定义日志字段类型，避免直接依赖logrus
type Fields map[string]interface{}

func main() {
	// 解析命令行参数
	opts := parseFlags()

	// 加载配置
	cfg, err := config.LoadConfig("", opts.Environment)
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 初始化日志管理器
	logManager, err := logger.InitLogger(&cfg.Log)
	if err != nil {
		log.Fatalf("日志初始化失败: %v", err)
	}

	logManager.GetLogger().WithFields(logrus.Fields{
		"path":        "cmd/migrate/main.go",
		"operation":   "database_migration",
		"option":      "migrate.start",
		"func_name":   "main",
		"environment": opts.Environment,
		"seed_data":   opts.SeedData,
		"drop_first":  opts.DropFirst,
	}).Info("开始数据库迁移")

	// 初始化数据库连接
	db, err := database.NewMySQLConnection(&cfg.Database.MySQL)
	if err != nil {
		logManager.GetLogger().WithFields(logrus.Fields{
			"path":      "cmd/migrate/main.go",
			"operation": "database_connection",
			"option":    "database.NewMySQLConnection",
			"func_name": "main",
			"error":     err.Error(),
		}).Fatal("数据库连接失败")
	}

	// 执行迁移
	if err := performMigration(db, opts, logManager); err != nil {
		logManager.GetLogger().WithFields(logrus.Fields{
			"path":      "cmd/migrate/main.go",
			"operation": "database_migration",
			"option":    "performMigration",
			"func_name": "main",
			"error":     err.Error(),
		}).Fatal("数据库迁移失败")
	}

	logManager.GetLogger().WithFields(logrus.Fields{
		"path":      "cmd/migrate/main.go",
		"operation": "database_migration",
		"option":    "migrate.complete",
		"func_name": "main",
	}).Info("数据库迁移完成")
}

// parseFlags 解析命令行参数
// 遵循Unix哲学：做一件事并做好
func parseFlags() *MigrateOptions {
	opts := &MigrateOptions{}

	flag.StringVar(&opts.Environment, "env", "test", "环境标识 (test, dev, prod)")
	flag.BoolVar(&opts.SeedData, "seed", true, "是否填充测试数据")
	flag.BoolVar(&opts.DropFirst, "drop", false, "是否先删除表（危险操作）")
	flag.BoolVar(&opts.Verbose, "verbose", false, "是否显示详细日志")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "NeoScan 数据库迁移工具\n\n")
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s -env=test -seed=true    # 测试环境迁移并填充数据\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -env=prod -seed=false   # 生产环境仅迁移表结构\n", os.Args[0])
	}

	flag.Parse()
	return opts
}

// performMigration 执行数据库迁移
// 遵循"Never break userspace"原则：向后兼容，不破坏现有数据
func performMigration(db *gorm.DB, opts *MigrateOptions, logManager *logger.LoggerManager) error {
	// 1. 删除表（如果指定）
	if opts.DropFirst {
		if err := dropTables(db, logManager); err != nil {
			return fmt.Errorf("删除表失败: %w", err)
		}
	}

	// 2. 执行模型迁移
	if err := migrateModels(db, logManager); err != nil {
		return fmt.Errorf("模型迁移失败: %w", err)
	}

	// 3. 填充测试数据（如果指定）
	if opts.SeedData {
		seeder := NewDataSeeder(db, opts.Environment, logManager)
		if err := seeder.SeedAll(); err != nil {
			return fmt.Errorf("数据填充失败: %w", err)
		}
	}

	return nil
}

// dropTables 删除所有表
// 危险操作，仅用于开发环境重置
func dropTables(db *gorm.DB, logManager *logger.LoggerManager) error {
	logManager.GetLogger().WithFields(logrus.Fields{
		"path":      "cmd/migrate/main.go",
		"operation": "drop_tables",
		"option":    "dropTables",
		"func_name": "dropTables",
	}).Warn("开始删除数据库表")

	// 定义所有模型（按依赖关系逆序）- 只包含实际存在的模型
	models := []interface{}{
		// 关联表先删除
		&system.UserRole{},
		&system.RolePermission{},
		&agent.AgentGroupMember{},

		// 主表后删除
		&system.User{},
		&system.Role{},
		&system.Permission{},
		&agent.Agent{},
		&agent.AgentVersion{},
		&agent.AgentConfig{},
		&agent.AgentMetrics{},
		&agent.AgentGroup{},
		&agent.ScanType{},
		&orchestrator_drop.ProjectConfig{},
		&orchestrator_drop.ScanTool{},
		&orchestrator_drop.ScanRule{},
		&orchestrator_drop.WorkflowConfig{},
	}

	for _, model := range models {
		if err := db.Migrator().DropTable(model); err != nil {
			logManager.GetLogger().WithFields(logrus.Fields{
				"path":      "cmd/migrate/main.go",
				"operation": "drop_table",
				"option":    "db.Migrator().DropTable",
				"func_name": "dropTables",
				"model":     fmt.Sprintf("%T", model),
				"error":     err.Error(),
			}).Error("删除表失败")
		}
	}

	return nil
}

// migrateModels 执行模型迁移
func migrateModels(db *gorm.DB, loggerMgr *logger.LoggerManager) error {
	loggerMgr.GetLogger().Info("开始执行模型迁移...")

	// 定义所有需要迁移的模型
	models := []interface{}{
		// 系统模块
		&system.User{},
		&system.Role{},
		&system.Permission{},
		&system.LoginRequest{},

		// Agent模块
		&agent.Agent{},
		&agent.AgentVersion{},
		&agent.AgentConfig{},
		&agent.AgentMetrics{},
		&agent.AgentGroup{},
		&agent.AgentGroupMember{},
		&agent.ScanType{},

		// Orchestrator模块 (New)
		&orchestrator.Project{},
		&orchestrator.Workflow{},
		&orchestrator.ProjectWorkflow{},
		&orchestrator.ScanStage{},
		&orchestrator.AgentTask{},
		&orchestrator.StageResult{},

		// Orchestrator模块 (Old/Deprecated)
		&orchestrator_drop.ProjectConfig{},
		&orchestrator_drop.WorkflowConfig{},
		&orchestrator_drop.ScanTool{},
		&orchestrator_drop.ScanRule{},
	}

	// 执行自动迁移
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("迁移模型 %T 失败: %w", model, err)
		}
		loggerMgr.GetLogger().WithField("model", fmt.Sprintf("%T", model)).Info("模型迁移成功")
	}

	// 手动处理关联表的特殊字段
	if err := fixAssociationTables(db, loggerMgr); err != nil {
		return fmt.Errorf("修复关联表失败: %w", err)
	}

	loggerMgr.GetLogger().Info("所有模型迁移完成")
	return nil
}

// fixAssociationTables 修复关联表的特殊字段
func fixAssociationTables(db *gorm.DB, loggerMgr *logger.LoggerManager) error {
	loggerMgr.GetLogger().Info("开始修复关联表字段...")

	// 1. 检查并修复 role_permissions 表的 created_at 字段
	if !db.Migrator().HasColumn(&system.RolePermission{}, "created_at") {
		loggerMgr.GetLogger().Info("为 role_permissions 表添加 created_at 字段")
		if err := db.Migrator().AddColumn(&system.RolePermission{}, "created_at"); err != nil {
			return fmt.Errorf("添加 role_permissions.created_at 字段失败: %w", err)
		}
	}

	// 2. 检查 user_roles 表是否存在，如果不存在则让 GORM 自动创建
	if !db.Migrator().HasTable(&system.UserRole{}) {
		loggerMgr.GetLogger().Info("user_roles 表不存在，将由 GORM 自动创建")
	} else {
		// 如果表已存在但没有 id 字段，需要重建表
		if !db.Migrator().HasColumn(&system.UserRole{}, "id") {
			loggerMgr.GetLogger().Info("user_roles 表缺少 id 字段，需要重建表")

			// 先备份数据
			var existingUserRoles []map[string]interface{}
			if err := db.Table("user_roles").Find(&existingUserRoles).Error; err != nil {
				loggerMgr.GetLogger().WithField("error", err.Error()).Warn("备份 user_roles 数据失败")
			}

			// 删除旧表
			if err := db.Migrator().DropTable(&system.UserRole{}); err != nil {
				return fmt.Errorf("删除旧 user_roles 表失败: %w", err)
			}

			// 重新创建表
			if err := db.AutoMigrate(&system.UserRole{}); err != nil {
				return fmt.Errorf("重新创建 user_roles 表失败: %w", err)
			}

			// 恢复数据（如果有的话）
			if len(existingUserRoles) > 0 {
				for _, userRole := range existingUserRoles {
					if err := db.Table("user_roles").Create(&userRole).Error; err != nil {
						loggerMgr.GetLogger().WithField("error", err.Error()).Warn("恢复 user_roles 数据失败")
					}
				}
				loggerMgr.GetLogger().WithField("count", len(existingUserRoles)).Info("恢复 user_roles 数据完成")
			}
		}
	}

	// 3. 强制重新迁移关联表
	associationModels := []interface{}{
		&system.RolePermission{},
	}

	for _, model := range associationModels {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("重新迁移关联表 %T 失败: %w", model, err)
		}
		loggerMgr.GetLogger().WithField("model", fmt.Sprintf("%T", model)).Info("关联表重新迁移成功")
	}

	loggerMgr.GetLogger().Info("关联表字段修复完成")
	return nil
}

// NewDataSeeder 创建数据填充器
func NewDataSeeder(db *gorm.DB, env string, logManager *logger.LoggerManager) *DataSeeder {
	return &DataSeeder{
		db:  db,
		env: env,
		log: logManager,
	}
}

// SeedAll 填充所有测试数据
// 遵循"好品味"原则：统一的处理流程，无特殊情况
func (s *DataSeeder) SeedAll() error {
	s.log.GetLogger().WithFields(logrus.Fields{
		"path":      "cmd/migrate/main.go",
		"operation": "seed_data",
		"option":    "SeedAll",
		"func_name": "DataSeeder.SeedAll",
		"env":       s.env,
	}).Info("开始填充测试数据")

	// 按依赖关系顺序填充数据
	seedFunctions := []struct {
		name string
		fn   func() error
	}{
		{"系统基础数据", s.seedSystemData},
		{"Agent测试数据", s.seedAgentData},
		{"扫描配置数据", s.seedOrchestratorData},
	}

	for _, seed := range seedFunctions {
		s.log.GetLogger().WithFields(logrus.Fields{
			"path":      "cmd/migrate/main.go",
			"operation": "seed_module",
			"option":    seed.name,
			"func_name": "DataSeeder.SeedAll",
		}).Info("填充数据模块")

		if err := seed.fn(); err != nil {
			return fmt.Errorf("填充%s失败: %w", seed.name, err)
		}
	}

	s.log.GetLogger().WithFields(logrus.Fields{
		"path":      "cmd/migrate/main.go",
		"operation": "seed_data",
		"option":    "SeedAll.complete",
		"func_name": "DataSeeder.SeedAll",
	}).Info("测试数据填充完成")

	return nil
}

// seedSystemData 填充系统基础数据（用户权限体系）
func (s *DataSeeder) seedSystemData() error {
	// 1. 创建默认角色
	roles := []system.Role{
		{Name: "admin", DisplayName: "系统管理员", Description: "拥有系统所有权限的超级管理员", Status: 1},
		{Name: "user", DisplayName: "普通用户", Description: "系统普通用户，拥有基础功能权限", Status: 1},
		{Name: "guest", DisplayName: "访客用户", Description: "只读权限的访客用户", Status: 1},
	}

	for _, role := range roles {
		if err := s.db.Where("name = ?", role.Name).FirstOrCreate(&role).Error; err != nil {
			return fmt.Errorf("创建角色失败: %w", err)
		}
	}

	// 2. 创建默认权限
	permissions := []system.Permission{
		{Name: "system:admin", DisplayName: "系统管理", Description: "系统管理权限", Resource: "system", Action: "admin", Status: 1},
		{Name: "user:create", DisplayName: "创建用户", Description: "创建新用户的权限", Resource: "user", Action: "create", Status: 1},
		{Name: "user:read", DisplayName: "查看用户", Description: "查看用户信息的权限", Resource: "user", Action: "read", Status: 1},
		{Name: "user:update", DisplayName: "更新用户", Description: "更新用户信息的权限", Resource: "user", Action: "update", Status: 1},
		{Name: "user:delete", DisplayName: "删除用户", Description: "删除用户的权限", Resource: "user", Action: "delete", Status: 1},
		{Name: "role:create", DisplayName: "创建角色", Description: "创建新角色的权限", Resource: "role", Action: "create", Status: 1},
		{Name: "role:read", DisplayName: "查看角色", Description: "查看角色信息的权限", Resource: "role", Action: "read", Status: 1},
		{Name: "role:update", DisplayName: "更新角色", Description: "更新角色信息的权限", Resource: "role", Action: "update", Status: 1},
		{Name: "role:delete", DisplayName: "删除角色", Description: "删除角色的权限", Resource: "role", Action: "delete", Status: 1},
		{Name: "permission:create", DisplayName: "创建权限", Description: "创建新权限的权限", Resource: "permission", Action: "create", Status: 1},
		{Name: "permission:read", DisplayName: "查看权限", Description: "查看权限信息的权限", Resource: "permission", Action: "read", Status: 1},
		{Name: "permission:update", DisplayName: "更新权限", Description: "更新权限信息的权限", Resource: "permission", Action: "update", Status: 1},
		{Name: "permission:delete", DisplayName: "删除权限", Description: "删除权限的权限", Resource: "permission", Action: "delete", Status: 1},
	}

	for _, perm := range permissions {
		if err := s.db.Where("name = ?", perm.Name).FirstOrCreate(&perm).Error; err != nil {
			return fmt.Errorf("创建权限失败: %w", err)
		}
	}

	// 3. 创建默认管理员用户
	adminUser := system.User{
		Username: "admin",
		Email:    "admin@neoscan.com",
		Password: "$argon2id$v=19$m=65536,t=3,p=2$lMamQlbNnoIXZfszn4jWqw$zVTokU4nXju4CdOR1bH5ABOMbaEagr8mTXrhAh/p0kQ", // 密码: admin123
		Nickname: "系统管理员",
		Status:   1,
	}

	if err := s.db.Where("username = ?", adminUser.Username).FirstOrCreate(&adminUser).Error; err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	// 4. 分配权限（管理员拥有所有权限）
	var adminRole system.Role
	if err := s.db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return fmt.Errorf("查找管理员角色失败: %w", err)
	}

	var allPermissions []system.Permission
	if err := s.db.Find(&allPermissions).Error; err != nil {
		return fmt.Errorf("查找权限列表失败: %w", err)
	}

	// 为管理员角色分配所有权限
	for _, perm := range allPermissions {
		rolePerm := system.RolePermission{
			RoleID:       adminRole.ID,
			PermissionID: perm.ID,
		}
		s.db.Where("role_id = ? AND permission_id = ?", rolePerm.RoleID, rolePerm.PermissionID).FirstOrCreate(&rolePerm)
	}

	// 为管理员用户分配管理员角色
	userRole := system.UserRole{
		UserID: adminUser.ID,
		RoleID: adminRole.ID,
	}
	s.db.Where("user_id = ? AND role_id = ?", userRole.UserID, userRole.RoleID).FirstOrCreate(&userRole)

	return nil
}

// seedAgentData 填充Agent测试数据
func (s *DataSeeder) seedAgentData() error {
	// 1. 创建Agent版本
	versions := []agent.AgentVersion{
		{
			Version:     "v1.0.0",
			ReleaseDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Changelog:   "初始版本发布\n- 基础Agent功能\n- 支持心跳检测\n- 支持任务分发",
			DownloadURL: "https://releases.neoscan.com/agent/v1.0.0/agent-v1.0.0.tar.gz",
			IsActive:    true,
			IsLatest:    false,
		},
		{
			Version:     "v1.1.0",
			ReleaseDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			Changelog:   "功能增强版本\n- 新增插件系统\n- 优化性能监控\n- 修复已知问题",
			DownloadURL: "https://releases.neoscan.com/agent/v1.1.0/agent-v1.1.0.tar.gz",
			IsActive:    true,
			IsLatest:    true,
		},
	}

	for _, version := range versions {
		if err := s.db.Where("version = ?", version.Version).FirstOrCreate(&version).Error; err != nil {
			return fmt.Errorf("创建Agent版本失败: %w", err)
		}
	}

	// 2. 创建Agent分组
	groups := []agent.AgentGroup{
		{GroupID: "ag_001", Name: "default", Description: "默认分组"},
		{GroupID: "ag_002", Name: "production", Description: "生产环境Agent分组"},
		{GroupID: "ag_003", Name: "development", Description: "开发环境Agent分组"},
	}

	for _, group := range groups {
		if err := s.db.Where("group_id = ?", group.GroupID).FirstOrCreate(&group).Error; err != nil {
			return fmt.Errorf("创建Agent分组失败: %w", err)
		}
	}

	// 3. 创建扫描类型测试数据
	scanTypes := []agent.ScanType{
		{
			Name:        "port_scan",
			DisplayName: "端口扫描",
			Description: "对目标主机进行端口扫描，识别开放的端口和服务",
			Category:    "network",
			IsActive:    true,
			ConfigTemplate: agent.ConfigTemplateJSON{
				"timeout":     30,
				"max_threads": 100,
				"port_range":  "1-65535",
				"scan_method": "tcp",
			},
		},
		{
			Name:        "vuln_scan",
			DisplayName: "漏洞扫描",
			Description: "对目标系统进行漏洞扫描，识别已知的安全漏洞",
			Category:    "security",
			IsActive:    true,
			ConfigTemplate: agent.ConfigTemplateJSON{
				"timeout":        60,
				"max_concurrent": 10,
				"severity":       "medium",
				"update_db":      true,
			},
		},
		{
			Name:        "web_scan",
			DisplayName: "Web扫描",
			Description: "对Web应用进行安全扫描，识别Web漏洞",
			Category:    "web",
			IsActive:    true,
			ConfigTemplate: agent.ConfigTemplateJSON{
				"timeout":         120,
				"max_depth":       3,
				"user_agent":      "NeoScan/1.0",
				"follow_redirect": true,
			},
		},
	}

	for _, scanType := range scanTypes {
		if err := s.db.Where("name = ?", scanType.Name).FirstOrCreate(&scanType).Error; err != nil {
			return fmt.Errorf("创建扫描类型失败: %w", err)
		}
	}

	// 4. 创建测试Agent（仅在test环境）
	if s.env == "test" {
		lastHeartbeat1 := time.Now().Add(-5 * time.Minute)
		lastHeartbeat2 := time.Now().Add(-10 * time.Minute)
		tokenExpiry := time.Now().Add(24 * time.Hour) // 设置Token过期时间为24小时后

		agents := []agent.Agent{
			{
				AgentID:       "neoscan-agent-001",
				Hostname:      "dev-scanner-01",
				IPAddress:     "192.168.1.100",
				Port:          5772,
				Version:       "v1.1.0",
				Status:        agent.AgentStatusOnline,
				OS:            "Linux",
				Arch:          "x86_64",
				CPUCores:      8,
				MemoryTotal:   17179869184,
				DiskTotal:     107374182400,
				TokenExpiry:   tokenExpiry,
				LastHeartbeat: lastHeartbeat1,
				Remark:        "开发环境测试Agent",
			},
			{
				AgentID:       "neoscan-agent-002",
				Hostname:      "test-scanner-01",
				IPAddress:     "172.16.0.10",
				Port:          5772,
				Version:       "v1.0.0",
				Status:        agent.AgentStatusOffline,
				OS:            "Windows",
				Arch:          "x86_64",
				CPUCores:      4,
				MemoryTotal:   8589934592,
				DiskTotal:     53687091200,
				TokenExpiry:   tokenExpiry,
				LastHeartbeat: lastHeartbeat2,
				Remark:        "测试环境Windows Agent",
			},
		}

		for _, ag := range agents {
			if err := s.db.Where("agent_id = ?", ag.AgentID).FirstOrCreate(&ag).Error; err != nil {
				return fmt.Errorf("创建测试Agent失败: %w", err)
			}
		}
	}

	return nil
}

// seedOrchestratorData 填充扫描编排模块测试数据
func (s *DataSeeder) seedOrchestratorData() error {
	s.log.GetLogger().Info("开始填充扫描编排模块测试数据...")

	// 1. 创建扫描工具
	scanTools := []orchestrator_drop.ScanTool{
		{
			Name:            "nmap",
			DisplayName:     "Nmap Network Scanner",
			Description:     "Network exploration tool and security scanner",
			Type:            orchestrator_drop.ScanToolTypePortScan,
			Version:         "7.94",
			ExecutablePath:  "/usr/bin/nmap",
			WorkingDir:      "/tmp",
			CommandTemplate: "nmap {{.target}} {{.options}}",
			DefaultParams:   `{"target": "127.0.0.1", "options": "-sS -O"}`,
			ParamSchema: `{
				"type": "object",
				"properties": {
					"target": {"type": "string", "description": "Target IP or hostname"},
					"options": {"type": "string", "description": "Nmap scan options"}
				},
				"required": ["target"]
			}`,
			OutputFormat: "xml",
			OutputParser: "nmap_xml_parser",
			ResultMapping: `{
				"ports": "$.nmaprun.host.ports.port",
				"os": "$.nmaprun.host.os.osmatch",
				"services": "$.nmaprun.host.ports.port.service"
			}`,
			MaxExecutionTime: 1800,
			MaxMemoryMB:      512,
			RequiresSudo:     true,
			Status:           orchestrator_drop.ScanToolStatusEnabled,
			IsBuiltIn:        true,
			SupportedOS:      "linux,windows,macos",
			Dependencies:     "nmap package",
			InstallGuide:     "Install nmap using package manager",
			Tags:             "network,port-scan,discovery",
			Metadata: `{
				"category": "network_scanner",
				"difficulty": "easy",
				"documentation": "https://nmap.org/docs.html"
			}`,
			CreatedBy: 1,
			UpdatedBy: 1,
		},
		{
			Name:            "masscan",
			DisplayName:     "Masscan Port Scanner",
			Description:     "Fast TCP port scanner",
			Type:            orchestrator_drop.ScanToolTypePortScan,
			Version:         "1.3.2",
			ExecutablePath:  "/usr/bin/masscan",
			WorkingDir:      "/tmp",
			CommandTemplate: "masscan {{.target}} {{.ports}} {{.rate}}",
			DefaultParams:   `{"target": "127.0.0.1/24", "ports": "-p1-65535", "rate": "--rate=1000"}`,
			ParamSchema: `{
				"type": "object",
				"properties": {
					"target": {"type": "string", "description": "Target IP range"},
					"ports": {"type": "string", "description": "Port range to scan"},
					"rate": {"type": "string", "description": "Scan rate"}
				},
				"required": ["target"]
			}`,
			OutputFormat: "json",
			OutputParser: "masscan_json_parser",
			ResultMapping: `{
				"open_ports": "$.ports",
				"timestamp": "$.timestamp",
				"ip": "$.ip"
			}`,
			MaxExecutionTime: 3600,
			MaxMemoryMB:      256,
			RequiresSudo:     true,
			Status:           orchestrator_drop.ScanToolStatusEnabled,
			IsBuiltIn:        true,
			SupportedOS:      "linux,windows,macos",
			Dependencies:     "masscan package",
			InstallGuide:     "Install masscan using package manager",
			Tags:             "network,port-scan,fast",
			Metadata: `{
				"category": "fast_scanner",
				"difficulty": "medium",
				"documentation": "https://github.com/robertdavidgraham/masscan"
			}`,
			CreatedBy: 1,
			UpdatedBy: 1,
		},
	}

	for _, tool := range scanTools {
		if err := s.db.Where("name = ?", tool.Name).FirstOrCreate(&tool).Error; err != nil {
			return fmt.Errorf("创建扫描工具失败: %w", err)
		}
		s.log.GetLogger().WithField("tool", tool.Name).Info("扫描工具创建成功")
	}

	// 2. 创建扫描规则
	// 创建扫描规则测试数据
	scanRules := []orchestrator_drop.ScanRule{
		{
			Name:            "port_scan_filter",
			DisplayName:     "端口扫描过滤规则",
			Description:     "过滤常见端口扫描结果",
			Type:            orchestrator_drop.ScanRuleTypeFilter,
			Category:        "network",
			Condition:       `{"field": "port", "operator": "in", "value": [80, 443, 22, 21]}`,
			Action:          `{"type": "allow", "message": "允许常见端口"}`,
			Parameters:      `{"timeout": 30, "retry_count": 3}`,
			ApplicableTools: "nmap,masscan",
			TargetTypes:     "ip,domain",
			ScanPhases:      "discovery,enumeration",
			Severity:        orchestrator_drop.ScanRuleSeverityMedium,
			Priority:        5,
			Confidence:      0.8,
			Status:          orchestrator_drop.ScanRuleStatusEnabled,
			IsBuiltIn:       true,
			Version:         "1.0",
			Tags:            "network,port,filter",
			Metadata:        `{"author": "system", "last_updated": "2024-01-01"}`,
			CreatedBy:       1,
		},
		{
			Name:            "vulnerability_alert",
			DisplayName:     "漏洞告警规则",
			Description:     "检测到高危漏洞时触发告警",
			Type:            orchestrator_drop.ScanRuleTypeAlert,
			Category:        "security",
			Condition:       `{"field": "severity", "operator": "gte", "value": "high"}`,
			Action:          `{"type": "alert", "message": "发现高危漏洞", "parameters": {"notify": true}}`,
			Parameters:      `{"notification_channels": ["email", "webhook"], "escalation_level": "high"}`,
			ApplicableTools: "nessus,openvas,nuclei",
			TargetTypes:     "web,service",
			ScanPhases:      "vulnerability_scan",
			Severity:        orchestrator_drop.ScanRuleSeverityHigh,
			Priority:        9,
			Confidence:      0.95,
			Status:          orchestrator_drop.ScanRuleStatusEnabled,
			IsBuiltIn:       true,
			Version:         "1.0",
			Tags:            "security,vulnerability,alert",
			Metadata:        `{"author": "system", "criticality": "high"}`,
			CreatedBy:       1,
		},
	}

	for _, rule := range scanRules {
		if err := s.db.Where("name = ?", rule.Name).FirstOrCreate(&rule).Error; err != nil {
			return fmt.Errorf("创建扫描规则失败: %w", err)
		}
		s.log.GetLogger().WithField("rule", rule.Name).Info("扫描规则创建成功")
	}

	// 3. 创建项目配置
	projectConfigs := []orchestrator_drop.ProjectConfig{
		{
			Name:          "demo_web_scan",
			DisplayName:   "演示Web扫描项目",
			Description:   "用于演示的Web应用扫描项目配置",
			TargetScope:   "192.168.1.0/24,example.com",
			ExcludeList:   "192.168.1.1,admin.example.com",
			ScanFrequency: 24,
			MaxConcurrent: 5,
			TimeoutSecond: 3600,
			Priority:      5,
			Status:        orchestrator_drop.ProjectConfigStatusActive,
			IsEnabled:     true,
			Tags:          "demo,web,security",
			Metadata:      `{"project_type": "demo", "environment": "test", "owner": "system"}`,
			CreatedBy:     1,
		},
		{
			Name:          "internal_network_scan",
			DisplayName:   "内网扫描项目",
			Description:   "内部网络安全扫描项目",
			TargetScope:   "10.0.0.0/8,172.16.0.0/12",
			ExcludeList:   "10.0.0.1,172.16.0.1",
			ScanFrequency: 12,
			MaxConcurrent: 10,
			TimeoutSecond: 7200,
			Priority:      8,
			Status:        orchestrator_drop.ProjectConfigStatusActive,
			IsEnabled:     true,
			Tags:          "internal,network,compliance",
			Metadata:      `{"project_type": "production", "environment": "internal", "compliance": "required"}`,
			CreatedBy:     1,
		},
	}

	for _, config := range projectConfigs {
		if err := s.db.Where("name = ?", config.Name).FirstOrCreate(&config).Error; err != nil {
			return fmt.Errorf("创建项目配置失败: %w", err)
		}
		s.log.GetLogger().WithField("project", config.Name).Info("项目配置创建成功")
	}

	// 4. 创建工作流配置
	workflowConfigs := []orchestrator_drop.WorkflowConfig{
		{
			Name:           "basic_web_scan_workflow",
			DisplayName:    "基础Web扫描工作流",
			Description:    "包含端口扫描和漏洞扫描的基础工作流",
			Version:        "1.0",
			ProjectID:      1, // 关联到第一个项目
			TriggerType:    orchestrator_drop.WorkflowTriggerScheduled,
			CronExpr:       "0 2 * * *", // 每天凌晨2点执行
			EventFilter:    `{"event_type": "scheduled", "conditions": []}`,
			Steps:          `[{"id":"port_scan","name":"端口扫描","type":"scan_tool","tool_id":1,"timeout":300},{"id":"vuln_scan","name":"漏洞扫描","type":"scan_tool","tool_id":2,"depends_on":["port_scan"],"timeout":600}]`,
			Variables:      `{"scan_timeout": 300, "max_concurrent": 5, "notification_enabled": true}`,
			Environment:    `{"SCAN_ENV": "test", "LOG_LEVEL": "info"}`,
			MaxRetries:     3,
			TimeoutMinutes: 120,
			Status:         orchestrator_drop.WorkflowStatusActive,
			IsEnabled:      true,
			Tags:           "web,security,automated",
			Metadata:       `{"author": "system", "category": "security", "priority": "normal"}`,
			CreatedBy:      1,
		},
	}

	for _, workflow := range workflowConfigs {
		if err := s.db.Where("name = ?", workflow.Name).FirstOrCreate(&workflow).Error; err != nil {
			return fmt.Errorf("创建工作流配置失败: %w", err)
		}
		s.log.GetLogger().WithField("workflow", workflow.Name).Info("工作流配置创建成功")
	}

	s.log.GetLogger().Info("扫描编排模块测试数据填充完成")
	return nil
}
