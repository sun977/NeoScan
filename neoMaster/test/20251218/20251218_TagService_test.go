package test_20251218

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"neomaster/internal/model/tag_system"
	repo "neomaster/internal/repo/mysql/tag_system"
	service "neomaster/internal/service/tag_system"
)

// setupServiceTestDB 初始化测试数据库连接
func setupServiceTestDB(t *testing.T) *gorm.DB {
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 确保表存在
	if !db.Migrator().HasTable(&tag_system.SysTag{}) {
		db.AutoMigrate(&tag_system.SysTag{})
	}
	if !db.Migrator().HasTable(&tag_system.SysMatchRule{}) {
		db.AutoMigrate(&tag_system.SysMatchRule{})
	}
	if !db.Migrator().HasTable(&tag_system.SysEntityTag{}) {
		db.AutoMigrate(&tag_system.SysEntityTag{})
	}

	return db
}

// TestTagService_CRUD 测试标签的基本增删改查
func TestTagService_CRUD(t *testing.T) {
	db := setupServiceTestDB(t)
	tagRepo := repo.NewTagRepository(db)
	tagSvc := service.NewTagService(tagRepo, db)
	ctx := context.Background()

	// 0. 清理旧数据
	db.Exec("DELETE FROM sys_tags WHERE name LIKE 'TestService_%'")

	// 1. Create Root Tag
	rootTag := &tag_system.SysTag{
		Name:        "TestService_Root",
		Description: "Root Tag for Service Test",
	}
	if err := tagSvc.CreateTag(ctx, rootTag); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}
	t.Logf("Created Root Tag: %d, Path: %s", rootTag.ID, rootTag.Path)

	if rootTag.Path != "/" {
		t.Errorf("Expected root path '/', got '%s'", rootTag.Path)
	}

	// 2. Create Child Tag
	childTag := &tag_system.SysTag{
		Name:        "TestService_Child",
		ParentID:    rootTag.ID,
		Description: "Child Tag",
	}
	if err := tagSvc.CreateTag(ctx, childTag); err != nil {
		t.Fatalf("CreateTag (child) failed: %v", err)
	}
	t.Logf("Created Child Tag: %d, Path: %s", childTag.ID, childTag.Path)

	expectedPath := fmt.Sprintf("/%d/", rootTag.ID)
	if childTag.Path != expectedPath {
		t.Errorf("Expected child path '%s', got '%s'", expectedPath, childTag.Path)
	}

	// 3. Get Tag By ID
	gotTag, err := tagSvc.GetTag(ctx, childTag.ID)
	if err != nil {
		t.Fatalf("GetTag failed: %v", err)
	}
	if gotTag.Name != childTag.Name {
		t.Errorf("GetTag name mismatch")
	}

	// 4. Update Tag
	childTag.Description = "Updated Description"
	if err1 := tagSvc.UpdateTag(ctx, childTag); err1 != nil {
		t.Fatalf("UpdateTag failed: %v", err1)
	}
	gotTagUpdated, _ := tagSvc.GetTag(ctx, childTag.ID)
	if gotTagUpdated.Description != "Updated Description" {
		t.Errorf("UpdateTag failed to persist description")
	}

	// 5. List Tags (Search)
	listReq := &tag_system.ListTagsRequest{
		Keyword: "TestService_Child",
	}
	tags, total, err := tagSvc.ListTags(ctx, listReq)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if total != 1 || len(tags) != 1 {
		t.Errorf("ListTags expected 1 result, got %d", len(tags))
	}

	// 6. Delete Tag
	if err1 := tagSvc.DeleteTag(ctx, rootTag.ID, true); err1 != nil { // Force delete root
		t.Fatalf("DeleteTag failed: %v", err1)
	}

	// Verify Deletion
	_, err = tagSvc.GetTag(ctx, childTag.ID)
	if err == nil {
		t.Error("Child tag should be deleted via cascade")
	}
}

// TestTagService_AutoTag 测试自动打标功能
func TestTagService_AutoTag(t *testing.T) {
	db := setupServiceTestDB(t)
	tagRepo := repo.NewTagRepository(db)
	tagSvc := service.NewTagService(tagRepo, db)
	ctx := context.Background()

	// 清理
	db.Exec("DELETE FROM sys_tags WHERE name LIKE 'TestAuto_%'")
	db.Exec("DELETE FROM sys_match_rules WHERE name LIKE 'TestRule_%'")
	db.Exec("DELETE FROM sys_entity_tags WHERE entity_id = 'test_host_001'")

	// 1. 准备标签
	tag := &tag_system.SysTag{Name: "TestAuto_Linux"}
	tagSvc.CreateTag(ctx, tag)

	// 2. 创建规则 (OS == "linux")
	ruleJSON := `{"field": "os_type", "operator": "equals", "value": "linux"}`
	rule := &tag_system.SysMatchRule{
		Name:       "TestRule_Linux",
		EntityType: "host",
		TagID:      tag.ID,
		Priority:   10,
		IsEnabled:  true,
		RuleJSON:   ruleJSON,
	}
	if err := tagSvc.CreateRule(ctx, rule); err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	// 3. 验证缓存是否加载 (CreateRule 应该自动触发 Reload)
	// 我们通过执行 AutoTag 来间接验证
	attributes := map[string]interface{}{
		"os_type":  "linux",
		"hostname": "web-01",
	}
	entityID := "test_host_001"

	if err := tagSvc.AutoTag(ctx, "host", entityID, attributes); err != nil {
		t.Fatalf("AutoTag failed: %v", err)
	}

	// 4. 验证标签是否打上
	entityTags, err := tagSvc.GetEntityTags(ctx, "host", entityID)
	if err != nil {
		t.Fatalf("GetEntityTags failed: %v", err)
	}

	found := false
	for _, et := range entityTags {
		if et.TagID == tag.ID && et.Source == "auto" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AutoTag failed to apply tag")
	}

	// 5. 测试属性不匹配的情况
	attributesMismatch := map[string]interface{}{
		"os_type": "windows",
	}
	if err := tagSvc.AutoTag(ctx, "host", entityID, attributesMismatch); err != nil {
		t.Fatalf("AutoTag (mismatch) failed: %v", err)
	}

	// 验证标签是否被移除 (因为是 AutoTag，全量计算后不匹配的会移除)
	entityTags, _ = tagSvc.GetEntityTags(ctx, "host", entityID)
	for _, et := range entityTags {
		if et.TagID == tag.ID && et.Source == "auto" {
			t.Error("AutoTag should have removed the tag on mismatch")
		}
	}
}

// TestTagService_SyncEntityTags 测试实体标签同步 (Agent Report 场景)
func TestTagService_SyncEntityTags(t *testing.T) {
	db := setupServiceTestDB(t)
	tagRepo := repo.NewTagRepository(db)
	tagSvc := service.NewTagService(tagRepo, db)
	ctx := context.Background()

	entityID := "test_agent_sync_001"
	db.Exec("DELETE FROM sys_entity_tags WHERE entity_id = ?", entityID)
	db.Exec("DELETE FROM sys_tags WHERE name LIKE 'TestSync_%'")

	// 准备标签
	t1 := &tag_system.SysTag{Name: "TestSync_1"}
	t2 := &tag_system.SysTag{Name: "TestSync_2"}
	tagSvc.CreateTag(ctx, t1)
	tagSvc.CreateTag(ctx, t2)

	// 1. 初始同步: [t1]
	targetTags := []uint64{t1.ID}
	if err := tagSvc.SyncEntityTags(ctx, "agent", entityID, targetTags, "agent_report", 0); err != nil {
		t.Fatalf("SyncEntityTags (init) failed: %v", err)
	}

	tags, _ := tagSvc.GetEntityTags(ctx, "agent", entityID)
	if len(tags) != 1 || tags[0].TagID != t1.ID {
		t.Errorf("Sync init failed. Got %v", tags)
	}

	// 2. 更新同步: [t2] (t1 应该被移除)
	targetTags2 := []uint64{t2.ID}
	if err := tagSvc.SyncEntityTags(ctx, "agent", entityID, targetTags2, "agent_report", 0); err != nil {
		t.Fatalf("SyncEntityTags (update) failed: %v", err)
	}

	tags, _ = tagSvc.GetEntityTags(ctx, "agent", entityID)
	if len(tags) != 1 || tags[0].TagID != t2.ID {
		t.Errorf("Sync update failed. Got %v", tags)
	}

	// 3. 混合 Source 测试: 手动添加 t1 (Source=manual)，然后 Sync [t2]
	// 预期: t1 保留 (因为是 manual), t2 保留 (因为在 sync list)
	tagSvc.AddEntityTag(ctx, "agent", entityID, t1.ID, "manual", 0)

	if err := tagSvc.SyncEntityTags(ctx, "agent", entityID, targetTags2, "agent_report", 0); err != nil {
		t.Fatalf("SyncEntityTags (mixed) failed: %v", err)
	}

	tags, _ = tagSvc.GetEntityTags(ctx, "agent", entityID)
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags (1 manual + 1 agent), got %d", len(tags))
	}
}
