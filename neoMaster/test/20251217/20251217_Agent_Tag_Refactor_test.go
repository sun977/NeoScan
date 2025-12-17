package test_20251217

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/tag_system"
	agentRepo "neomaster/internal/repo/mysql/agent"
	tagRepo "neomaster/internal/repo/mysql/tag_system"
	agentService "neomaster/internal/service/agent"
	tagService "neomaster/internal/service/tag_system"
)

// setupTestDB 初始化测试数据库连接
// 使用 neoscan_dev 数据库
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移相关表
	// 注意：由于外键约束，这里可能无法轻易修改表结构，所以只在必要时迁移
	// 实际开发中，应由迁移脚本管理 Schema
	// 这里为了简单，如果表存在则不强制 AutoMigrate
	if !db.Migrator().HasTable(&agentModel.Agent{}) {
		err = db.AutoMigrate(&agentModel.Agent{})
		if err != nil {
			t.Fatalf("Failed to migrate Agent table: %v", err)
		}
	}
	if !db.Migrator().HasTable(&tag_system.SysTag{}) {
		err = db.AutoMigrate(&tag_system.SysTag{})
		if err != nil {
			t.Fatalf("Failed to migrate SysTag table: %v", err)
		}
	}
	// SysEntityTag 可能会被多次引用，也检查一下
	if !db.Migrator().HasTable(&tag_system.SysEntityTag{}) {
		err = db.AutoMigrate(&tag_system.SysEntityTag{})
		if err != nil {
			t.Fatalf("Failed to migrate SysEntityTag table: %v", err)
		}
	}

	return db
}

// TestAgentTagRefactor 测试Agent标签管理重构 (使用ID而非Name)
// 验证:
// 1. AddAgentTag (ID)
// 2. RemoveAgentTag (ID)
// 3. UpdateAgentTags (IDs)
// 4. GetAgentTags (返回Names, 内部逻辑正确)
func TestAgentTagRefactor(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// 初始化 Repository 和 Service
	tagRepoInst := tagRepo.NewTagRepository(db)
	tagSvc := tagService.NewTagService(tagRepoInst, db)
	agentRepoInst := agentRepo.NewAgentRepository(db)
	agentSvc := agentService.NewAgentManagerService(agentRepoInst, tagSvc)

	// 清理旧数据
	db.Exec("DELETE FROM sys_tags WHERE name LIKE 'TestAgentTag%'")
	db.Exec("DELETE FROM agents WHERE agent_id = 'test_agent_001'")
	db.Exec("DELETE FROM sys_entity_tags WHERE entity_type = 'agent' AND entity_id = 'test_agent_001'")

	// 准备测试数据
	// 1. 创建两个测试标签
	tag1 := &tag_system.SysTag{
		Name:        "TestAgentTag_1",
		Description: "For Agent Testing 1",
	}
	if err := tagSvc.CreateTag(ctx, tag1); err != nil {
		t.Fatalf("Failed to create tag 1: %v", err)
	}
	t.Logf("Created Tag 1: ID=%d, Name=%s", tag1.ID, tag1.Name)

	tag2 := &tag_system.SysTag{
		Name:        "TestAgentTag_2",
		Description: "For Agent Testing 2",
	}
	if err := tagSvc.CreateTag(ctx, tag2); err != nil {
		t.Fatalf("Failed to create tag 2: %v", err)
	}
	t.Logf("Created Tag 2: ID=%d, Name=%s", tag2.ID, tag2.Name)

	// 2. 创建一个测试 Agent (仅需最小字段)
	// 注意：RegisterAgent 逻辑较多，这里直接操作 DB 插入一条记录以简化依赖
	testAgent := &agentModel.Agent{
		AgentID:       "test_agent_001",
		Hostname:      "test-host",
		IPAddress:     "127.0.0.1",
		Status:        agentModel.AgentStatusOnline,
		TokenExpiry:   time.Now().Add(24 * time.Hour),
		LastHeartbeat: time.Now(),
	}
	if err := db.Create(testAgent).Error; err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}
	t.Logf("Created Test Agent: ID=%s", testAgent.AgentID)

	// === 测试 1: AddAgentTag (使用 TagID) ===
	t.Log("=== Test 1: AddAgentTag ===")
	reqAdd := &agentModel.AgentTagRequest{
		AgentID: testAgent.AgentID,
		TagID:   tag1.ID,
	}
	if err := agentSvc.AddAgentTag(reqAdd); err != nil {
		t.Fatalf("AddAgentTag failed: %v", err)
	}

	// 验证
	tags, err := agentSvc.GetAgentTags(testAgent.AgentID)
	if err != nil {
		t.Fatalf("GetAgentTags failed: %v", err)
	}
	if len(tags) != 1 || tags[0] != tag1.Name {
		t.Errorf("Expected tags [%s], got %v", tag1.Name, tags)
	}
	t.Logf("AddAgentTag Success. Current Tags: %v", tags)

	// === 测试 2: UpdateAgentTags (覆盖更新, 使用 IDs) ===
	t.Log("=== Test 2: UpdateAgentTags ===")
	// 将 Tag1 替换为 Tag2
	// 注意: UpdateAgentTags 现在接收 []uint64
	newTagIDs := []uint64{tag2.ID}
	oldTags, newTagsResp, err := agentSvc.UpdateAgentTags(testAgent.AgentID, newTagIDs)
	if err != nil {
		t.Fatalf("UpdateAgentTags failed: %v", err)
	}

	// 验证返回值
	t.Logf("Update Return - Old: %v, New: %v", oldTags, newTagsResp)
	if len(oldTags) != 1 || oldTags[0] != tag1.Name {
		t.Errorf("Old tags incorrect. Expected [%s], got %v", tag1.Name, oldTags)
	}
	if len(newTagsResp) != 1 || newTagsResp[0] != tag2.Name {
		t.Errorf("New tags incorrect. Expected [%s], got %v", tag2.Name, newTagsResp)
	}

	// 验证实际状态
	currentTags, err := agentSvc.GetAgentTags(testAgent.AgentID)
	if err != nil {
		t.Fatalf("GetAgentTags failed: %v", err)
	}
	if len(currentTags) != 1 || currentTags[0] != tag2.Name {
		t.Errorf("Expected current tags [%s], got %v", tag2.Name, currentTags)
	}
	t.Logf("UpdateAgentTags Success. Current Tags: %v", currentTags)

	// === 测试 3: UpdateAgentTags (多标签) ===
	t.Log("=== Test 3: UpdateAgentTags (Multiple) ===")
	// 更新为 Tag1 + Tag2
	multiTagIDs := []uint64{tag1.ID, tag2.ID}
	_, _, err = agentSvc.UpdateAgentTags(testAgent.AgentID, multiTagIDs)
	if err != nil {
		t.Fatalf("UpdateAgentTags (Multiple) failed: %v", err)
	}

	currentTags, err = agentSvc.GetAgentTags(testAgent.AgentID)
	if err != nil {
		t.Fatalf("GetAgentTags failed: %v", err)
	}
	// 顺序可能不保证，检查包含性
	if len(currentTags) != 2 {
		t.Errorf("Expected 2 tags, got %d: %v", len(currentTags), currentTags)
	} else {
		// 简单检查
		hasTag1 := false
		hasTag2 := false
		for _, name := range currentTags {
			if name == tag1.Name {
				hasTag1 = true
			}
			if name == tag2.Name {
				hasTag2 = true
			}
		}
		if !hasTag1 || !hasTag2 {
			t.Errorf("Expected both tags present. Got %v", currentTags)
		}
	}
	t.Logf("UpdateAgentTags (Multiple) Success. Current Tags: %v", currentTags)

	// === 测试 4: RemoveAgentTag (使用 TagID) ===
	t.Log("=== Test 4: RemoveAgentTag ===")
	reqRemove := &agentModel.AgentTagRequest{
		AgentID: testAgent.AgentID,
		TagID:   tag1.ID,
	}
	if err := agentSvc.RemoveAgentTag(reqRemove); err != nil {
		t.Fatalf("RemoveAgentTag failed: %v", err)
	}

	// 验证
	currentTags, err = agentSvc.GetAgentTags(testAgent.AgentID)
	if err != nil {
		t.Fatalf("GetAgentTags failed: %v", err)
	}
	if len(currentTags) != 1 || currentTags[0] != tag2.Name {
		t.Errorf("Expected tags [%s], got %v", tag2.Name, currentTags)
	}
	t.Logf("RemoveAgentTag Success. Current Tags: %v", currentTags)

	// 清理
	t.Log("=== Cleaning up ===")
	// db.Exec("DELETE FROM sys_tags WHERE name LIKE 'TestAgentTag%'")
	// db.Exec("DELETE FROM agents WHERE agent_id = 'test_agent_001'")
	// db.Exec("DELETE FROM sys_entity_tags WHERE entity_type = 'agent' AND entity_id = 'test_agent_001'")
}
