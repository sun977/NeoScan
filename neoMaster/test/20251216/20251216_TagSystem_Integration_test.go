package test_20251216

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	assetModel "neomaster/internal/model/asset"
	"neomaster/internal/model/orchestrator"
	"neomaster/internal/model/tag_system"
	tagRepo "neomaster/internal/repo/mysql/tag_system"
	tagService "neomaster/internal/service/tag_system"
)

// setupTestDB 初始化测试数据库连接
// 使用 neoscan_dev 数据库
func setupTestDB(t *testing.T) *gorm.DB {
	// 尝试使用环境变量或默认配置，这里为了测试方便直接硬编码，但建议确认密码
	// 如果连接失败，请检查本地 MySQL 配置
	// dsn := "root:123456@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
	// 如果你的密码是空或其他，请修改这里
	// dsn := "root:@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移相关表
	err = db.AutoMigrate(
		&tag_system.SysTag{},
		&tag_system.SysEntityTag{},
		&tag_system.SysMatchRule{},
		&orchestrator.AgentTask{},
		&assetModel.AssetHost{},
		&assetModel.AssetNetwork{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate tables: %v", err)
	}

	return db
}

// TestTagSystemIntegration 标签系统完整集成测试
// 涵盖:
// 1. 标签层级创建 (CRUD)
// 2. 规则创建与 AutoTag
// 3. 手动标签传播 (Network -> Host)
func TestTagSystemIntegration(t *testing.T) {
	db := setupTestDB(t)
	repo := tagRepo.NewTagRepository(db)
	svc := tagService.NewTagService(repo, db)
	ctx := context.Background()

	// 清理测试数据
	defer func() {
		// db.Exec("DELETE FROM sys_tags WHERE name LIKE 'TestTag%'")
		// db.Exec("DELETE FROM sys_match_rules WHERE rule_json LIKE '%TestRule%'")
		// db.Exec("DELETE FROM sys_entity_tags WHERE source = 'auto'")
		// db.Exec("DELETE FROM agent_tasks WHERE task_type = 'sys_tag_propagation'")
		// db.Exec("DELETE FROM asset_hosts WHERE ip LIKE '192.168.Test.%'")
		// db.Exec("DELETE FROM asset_networks WHERE cidr = '192.168.Test.0/24'")
	}()

	t.Log("=== Step 1: Creating Hierarchical Tags ===")
	// 目标结构: /R/Location/IDC/IDC CZTT
	// 我们模拟: /TestTag/Location/IDC/IDC_CZTT

	// 先尝试删除根标签及其子标签 (如果有)
	// 注意：在实际测试中，可能需要递归删除，这里简单处理
	db.Where("name LIKE ?", "TestTag%").Delete(&tag_system.SysTag{})
	db.Where("name LIKE ?", "TestRule%").Delete(&tag_system.SysMatchRule{})
	db.Where("network LIKE ?", "192.168.Test%").Delete(&assetModel.AssetNetwork{})

	// 1.1 Root Tag
	rootTag := &tag_system.SysTag{
		Name:        "TestTag_Root",
		Description: "Root Tag for Testing",
		Color:       "#000000",
	}
	if err := svc.CreateTag(ctx, rootTag); err != nil {
		t.Fatalf("Failed to create root tag: %v", err)
	}
	t.Logf("Created Root Tag: ID=%d, Path=%s", rootTag.ID, rootTag.Path)

	// 1.2 Level 1: Location
	locTag := &tag_system.SysTag{
		Name:     "TestTag_Location",
		ParentID: rootTag.ID,
	}
	if err := svc.CreateTag(ctx, locTag); err != nil {
		t.Fatalf("Failed to create location tag: %v", err)
	}
	t.Logf("Created Location Tag: ID=%d, Path=%s", locTag.ID, locTag.Path)

	// 1.3 Level 2: IDC
	idcTag := &tag_system.SysTag{
		Name:     "TestTag_IDC",
		ParentID: locTag.ID,
	}
	if err := svc.CreateTag(ctx, idcTag); err != nil {
		t.Fatalf("Failed to create IDC tag: %v", err)
	}
	t.Logf("Created IDC Tag: ID=%d, Path=%s", idcTag.ID, idcTag.Path)

	// 1.4 Level 3: IDC_CZTT (最终标签)
	czttTag := &tag_system.SysTag{
		Name:     "TestTag_IDC_CZTT",
		ParentID: idcTag.ID,
	}
	if err := svc.CreateTag(ctx, czttTag); err != nil {
		t.Fatalf("Failed to create CZTT tag: %v", err)
	}
	t.Logf("Created CZTT Tag: ID=%d, Path=%s", czttTag.ID, czttTag.Path)

	t.Log("=== Step 2: Testing AutoTag (Scenario 1 & 3) ===")
	// 场景: 如果端口包含 8080, 自动打上 "TestTag_IDC_CZTT" (模拟 Web 标签场景)

	// 2.1 创建规则
	ruleJSON := `{"field": "port", "operator": "in", "value": [80, 443, 8080]}`
	rule := &tag_system.SysMatchRule{
		Name:       "TestRule_WebPort",
		TagID:      czttTag.ID,
		EntityType: "host",
		Priority:   10,
		RuleJSON:   ruleJSON,
		IsEnabled:  true,
	}
	if err := svc.CreateRule(ctx, rule); err != nil {
		t.Fatalf("Failed to create rule: %v", err)
	}
	t.Logf("Created Rule: ID=%d", rule.ID)

	// 2.2 模拟资产入库并触发 AutoTag
	// Case A: 命中规则
	hostID_A := "host_test_A"
	attrs_A := map[string]interface{}{
		"port": 8080,
		"ip":   "192.168.1.10",
	}
	if err := svc.AutoTag(ctx, "host", hostID_A, attrs_A); err != nil {
		t.Fatalf("AutoTag failed for Case A: %v", err)
	}

	// 验证 Case A 是否有标签
	tagsA, err := repo.GetEntityTags("host", hostID_A)
	if err != nil {
		t.Fatalf("Failed to get entity tags: %v", err)
	}
	if len(tagsA) == 0 || tagsA[0].TagID != czttTag.ID {
		t.Errorf("Case A should match rule. Got tags: %v", tagsA)
	} else {
		t.Log("Case A AutoTag Success: Matched rule")
	}

	// Case B: 不命中规则
	hostID_B := "host_test_B"
	attrs_B := map[string]interface{}{
		"port": 22,
		"ip":   "192.168.1.11",
	}
	if err1 := svc.AutoTag(ctx, "host", hostID_B, attrs_B); err1 != nil {
		t.Fatalf("AutoTag failed for Case B: %v", err1)
	}
	tagsB, _ := repo.GetEntityTags("host", hostID_B)
	if len(tagsB) > 0 {
		t.Errorf("Case B should NOT match rule. Got tags: %v", tagsB)
	} else {
		t.Log("Case B AutoTag Success: Not matched")
	}

	t.Log("=== Step 3: Testing Manual Propagation (Scenario 2) ===")
	// 场景: 网段打标 -> 扩散到 IP

	// 3.1 准备数据: 创建一个 Network 和一个属于该 Network 的 Host
	testCIDR := "192.168.Test.0/24"
	network := &assetModel.AssetNetwork{
		CIDR:    testCIDR,
		Network: testCIDR, // Network 字段也填上，避免非空约束
		Tags:    "[]",     // Tags 字段是 JSON 类型，不能为空字符串
		// Name: "Test Network", // AssetNetwork 结构体中没有 Name 字段
	}
	if err2 := db.Create(network).Error; err2 != nil {
		t.Fatalf("Failed to create test network: %v", err2)
	}

	// 3.2 触发传播
	// 我们想把 rootTag 传播给该网段下的所有主机
	taskID, err := svc.SubmitEntityPropagationTask(ctx, "network", uint64(network.ID), []uint64{rootTag.ID}, "add")
	if err != nil {
		t.Fatalf("Failed to submit propagation task: %v", err)
	}
	t.Logf("Propagation Task Submitted: TaskID=%s", taskID)

	// 3.3 验证任务是否生成
	var task orchestrator.AgentTask
	if err := db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		t.Fatalf("Failed to find task: %v", err)
	}

	// 验证 Payload
	var payload tagService.TagPropagationPayload
	if err := json.Unmarshal([]byte(task.ToolParams), &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	// 检查 Payload 内容
	// Rule 应该是: ip cidr 192.168.Test.0/24
	if payload.TargetType != "host" {
		t.Errorf("Expected TargetType host, got %s", payload.TargetType)
	}
	if payload.Rule.Field != "ip" || payload.Rule.Operator != "cidr" || payload.Rule.Value != testCIDR {
		t.Errorf("Virtual Rule mismatch. Got: %+v", payload.Rule)
	}
	if len(payload.Tags) == 0 || payload.Tags[0] != rootTag.Name {
		t.Errorf("Tags mismatch. Expected %s, got %v", rootTag.Name, payload.Tags)
	}

	t.Log("Manual Propagation Task Verified Success")
}
