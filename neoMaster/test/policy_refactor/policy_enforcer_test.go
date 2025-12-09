package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"neomaster/internal/config"
	"neomaster/internal/model/asset"
	"neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"
	assetRepo "neomaster/internal/repo/mysql/asset"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/policy"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// SetupPolicyTestEnv 初始化策略测试环境
func SetupPolicyTestEnv() (*gorm.DB, policy.PolicyEnforcer, error) {
	// 初始化简易日志
	logger.InitLogger(&config.LogConfig{Level: "error", Output: "console"})

	// 连接到测试数据库 (neoscan_dev)
	// 注意: 这里假设运行测试的环境有本地 MySQL
	// 如果没有，测试会 fail，符合预期
	dbConfig := &config.MySQLConfig{
		Host:      "localhost",
		Port:      3306,
		Username:  "root",
		Password:  "ROOT",
		Database:  "neoscan_dev",
		Charset:   "utf8mb4",
		ParseTime: true,
		Loc:       "Local",
	}
	db, err := database.NewMySQLConnection(dbConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("connect db failed: %v", err)
	}

	// 自动迁移相关表
	if err := db.AutoMigrate(&asset.AssetWhitelist{}, &asset.AssetSkipPolicy{}, &orchestrator.Project{}); err != nil {
		return nil, nil, err
	}

	// 清理旧数据
	db.Exec("TRUNCATE TABLE asset_whitelists")
	db.Exec("TRUNCATE TABLE asset_skip_policies")
	db.Exec("TRUNCATE TABLE projects")

	// 初始化 Repo
	pRepo := orcRepo.NewProjectRepository(db)
	aRepo := assetRepo.NewAssetPolicyRepository(db)

	// 初始化 Enforcer
	enforcer := policy.NewPolicyEnforcer(pRepo, aRepo)

	return db, enforcer, nil
}

func TestPolicyEnforcer_Whitelist(t *testing.T) {
	db, enforcer, err := SetupPolicyTestEnv()
	if err != nil {
		t.Skipf("Skipping test due to DB setup failure: %v", err)
		return
	}

	ctx := context.Background()

	// 1. 插入白名单规则
	rules := []asset.AssetWhitelist{
		{WhitelistName: "Block Localhost", TargetType: "ip", TargetValue: "127.0.0.1", Enabled: true, Tags: "[]", Scope: "{}"},
		{WhitelistName: "Block Range", TargetType: "ip", TargetValue: "192.168.1.1-192.168.1.5", Enabled: true, Tags: "[]", Scope: "{}"},
		{WhitelistName: "Block CIDR", TargetType: "cidr", TargetValue: "10.0.0.0/8", Enabled: true, Tags: "[]", Scope: "{}"},
		{WhitelistName: "Block Domain Suffix", TargetType: "domain", TargetValue: ".gov.cn", Enabled: true, Tags: "[]", Scope: "{}"},
		{WhitelistName: "Block Domain Pattern", TargetType: "domain_pattern", TargetValue: "*.bad.com", Enabled: true, Tags: "[]", Scope: "{}"},
		{WhitelistName: "Block Specific Domain", TargetType: "domain", TargetValue: "forbidden.com", Enabled: true, Tags: "[]", Scope: "{}"},
		{WhitelistName: "Block URL Prefix", TargetType: "url", TargetValue: "http://malicious.com/api", Enabled: true, Tags: "[]", Scope: "{}"},
		{WhitelistName: "Block Keyword", TargetType: "keyword", TargetValue: "sensitive", Enabled: true, Tags: "[]", Scope: "{}"},
	}
	for _, r := range rules {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("Failed to create rule: %v", err)
		}
	}

	// 2. 准备项目 (Project)
	// TargetScope 必须包含我们要测试的目标，否则会先被 Scope 校验拦截
	project := &orchestrator.Project{
		Name:         "Test Project Whitelist",
		TargetScope:  "127.0.0.1,192.168.1.1,192.168.1.3,192.168.1.6,google.com,test.gov.cn,forbidden.com,mysensitive.com,safe.com,10.1.1.1,api.bad.com,http://malicious.com/api/v1,http://malicious.com/other,http://127.0.0.1/admin",
		Status:       "running",
		NotifyConfig: "{}",
		ExportConfig: "{}",
		ExtendedData: "{}",
		Tags:         "[]",
	}
	if err := db.Create(project).Error; err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	tests := []struct {
		name        string
		target      string
		shouldBlock bool
	}{
		{"Blocked Localhost", "127.0.0.1", true},
		{"Blocked Range Start", "192.168.1.1", true},
		{"Blocked Range Mid", "192.168.1.3", true},
		{"Allowed Range Out", "192.168.1.6", false},
		{"Blocked CIDR", "10.1.1.1", true},
		{"Blocked Domain Suffix", "test.gov.cn", true},
		{"Blocked Specific Domain", "forbidden.com", true},
		{"Blocked Domain Pattern", "api.bad.com", true},
		{"Allowed Domain", "google.com", false},
		{"Blocked Keyword", "mysensitive.com", true},
		{"Allowed Safe", "safe.com", false},
		{"Blocked URL Prefix", "http://malicious.com/api/v1", true},
		{"Allowed URL Other Path", "http://malicious.com/other", false},
		{"Blocked URL Host (matches IP whitelist)", "http://127.0.0.1/admin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &orchestrator.AgentTask{
				TaskID:      "task-" + tt.target,
				ProjectID:   project.ID,
				InputTarget: tt.target,
			}
			err := enforcer.Enforce(ctx, task)
			if tt.shouldBlock {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "whitelisted")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPolicyEnforcer_SkipLogic(t *testing.T) {
	db, enforcer, err := SetupPolicyTestEnv()
	if err != nil {
		t.Skipf("Skipping test due to DB setup failure: %v", err)
		return
	}
	ctx := context.Background()

	// 1. 插入跳过策略
	// 阻止带有 "prod" 标签的项目
	rulesEnv := map[string][]string{
		"block_env_tags": {"prod"},
	}
	rulesEnvJson, _ := json.Marshal(rulesEnv)

	db.Create(&asset.AssetSkipPolicy{
		PolicyName:     "Block Prod Env",
		ConditionRules: string(rulesEnvJson),
		Enabled:        true,
		ActionConfig:   "{}",
		Scope:          "{}",
		Tags:           "[]",
	})

	// 2. 创建项目
	projProd := &orchestrator.Project{
		Name:         "Prod Project",
		TargetScope:  "1.1.1.1",
		Tags:         `["prod", "web"]`,
		NotifyConfig: "{}",
		ExportConfig: "{}",
		ExtendedData: "{}",
	}
	db.Create(projProd)

	projDev := &orchestrator.Project{
		Name:         "Dev Project",
		TargetScope:  "1.1.1.1",
		Tags:         `["dev"]`,
		NotifyConfig: "{}",
		ExportConfig: "{}",
		ExtendedData: "{}",
	}
	db.Create(projDev)

	// 测试 Prod 项目应被跳过
	err = enforcer.Enforce(ctx, &orchestrator.AgentTask{ProjectID: projProd.ID, InputTarget: "1.1.1.1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skipped")

	// 测试 Dev 项目应通过
	err = enforcer.Enforce(ctx, &orchestrator.AgentTask{ProjectID: projDev.ID, InputTarget: "1.1.1.1"})
	assert.NoError(t, err)
}
