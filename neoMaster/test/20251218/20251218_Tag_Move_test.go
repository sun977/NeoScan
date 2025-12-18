package test_20251218

import (
	"fmt"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"neomaster/internal/model/tag_system"
	tagRepo "neomaster/internal/repo/mysql/tag_system"
)

// setupTestDB 初始化测试数据库连接
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 确保表存在
	if !db.Migrator().HasTable(&tag_system.SysTag{}) {
		err = db.AutoMigrate(&tag_system.SysTag{})
		if err != nil {
			t.Fatalf("Failed to migrate SysTag table: %v", err)
		}
	}

	return db
}

// TestMoveTagSafety 验证 MoveTag 的安全性和正确性
// 对应 Linus 的核心哲学: "Never break userspace" (数据一致性是神圣的)
func TestMoveTagSafety(t *testing.T) {
	db := setupTestDB(t)
	repo := tagRepo.NewTagRepository(db)

	// 0. 清理旧数据，确保环境干净
	db.Exec("DELETE FROM sys_tags WHERE name LIKE 'TestMove_%'")

	// 1. 构建初始树结构
	// Root (A)
	//   |-- Child (B)
	//        |-- Grandchild (C)
	// TargetRoot (D)

	// A: Root
	rootA := &tag_system.SysTag{
		Name:     "TestMove_RootA",
		ParentID: 0,
		Path:     "/",
		Level:    0,
	}
	if err := repo.CreateTag(rootA); err != nil {
		t.Fatalf("Failed to create RootA: %v", err)
	}

	// B: Child of A
	childB := &tag_system.SysTag{
		Name:     "TestMove_ChildB",
		ParentID: rootA.ID,
		Path:     fmt.Sprintf("/%d/", rootA.ID),
		Level:    1,
	}
	if err := repo.CreateTag(childB); err != nil {
		t.Fatalf("Failed to create ChildB: %v", err)
	}

	// C: Child of B (Grandchild of A)
	grandChildC := &tag_system.SysTag{
		Name:     "TestMove_GrandChildC",
		ParentID: childB.ID,
		Path:     fmt.Sprintf("/%d/%d/", rootA.ID, childB.ID),
		Level:    2,
	}
	if err := repo.CreateTag(grandChildC); err != nil {
		t.Fatalf("Failed to create GrandChildC: %v", err)
	}

	// D: Target Root
	targetRootD := &tag_system.SysTag{
		Name:     "TestMove_TargetRootD",
		ParentID: 0,
		Path:     "/",
		Level:    0,
	}
	if err := repo.CreateTag(targetRootD); err != nil {
		t.Fatalf("Failed to create TargetRootD: %v", err)
	}

	t.Log("Initial Tree Created:")
	t.Logf("A: ID=%d, Path=%s", rootA.ID, rootA.Path)
	t.Logf("B: ID=%d, Path=%s", childB.ID, childB.Path)
	t.Logf("C: ID=%d, Path=%s", grandChildC.ID, grandChildC.Path)
	t.Logf("D: ID=%d, Path=%s", targetRootD.ID, targetRootD.Path)

	// --- 测试场景 1: 正常移动 (Move B to D) ---
	// 预期:
	// B.ParentID -> D.ID
	// B.Path -> "/D.ID/"
	// C.Path -> "/D.ID/B.ID/" (自动级联更新)
	// C.Level -> 2 (依然是2, 因为 D(0)->B(1)->C(2))

	t.Log("\n--- Scenario 1: Moving B to under D ---")
	if err := repo.MoveTag(childB.ID, targetRootD.ID); err != nil {
		t.Fatalf("MoveTag failed: %v", err)
	}

	// 验证 B
	var updatedB tag_system.SysTag
	db.First(&updatedB, childB.ID)
	expectedBPath := fmt.Sprintf("/%d/", targetRootD.ID)
	if updatedB.ParentID != targetRootD.ID || updatedB.Path != expectedBPath {
		t.Errorf("B update failed. Got ParentID=%d, Path=%s; Expected ParentID=%d, Path=%s",
			updatedB.ParentID, updatedB.Path, targetRootD.ID, expectedBPath)
	}

	// 验证 C (级联更新的核心验证)
	var updatedC tag_system.SysTag
	db.First(&updatedC, grandChildC.ID)
	expectedCPath := fmt.Sprintf("/%d/%d/", targetRootD.ID, childB.ID)
	if updatedC.Path != expectedCPath {
		t.Errorf("C cascade update failed. Got Path=%s; Expected Path=%s", updatedC.Path, expectedCPath)
	}

	t.Log("Scenario 1 Passed: Tree structure and paths updated correctly.")

	// --- 测试场景 2: 循环依赖检测 (Try to move B to C) ---
	// B 现在是 C 的父亲。尝试把 B 移到 C 下面，应该报错。
	t.Log("\n--- Scenario 2: Circular Dependency Check (Move B to C) ---")
	err := repo.MoveTag(childB.ID, grandChildC.ID)
	if err == nil {
		t.Error("Expected error for circular dependency, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	// --- 测试场景 3: 移动到根节点 (Move B to Root) ---
	// 预期:
	// B.ParentID -> 0
	// B.Path -> "/"
	// C.Path -> "/B.ID/"
	t.Log("\n--- Scenario 3: Moving B to Root ---")
	if err := repo.MoveTag(childB.ID, 0); err != nil {
		t.Fatalf("MoveTag to root failed: %v", err)
	}

	// 验证 B
	db.First(&updatedB, childB.ID)
	if updatedB.ParentID != 0 || updatedB.Path != "/" {
		t.Errorf("B update to root failed. Got ParentID=%d, Path=%s", updatedB.ParentID, updatedB.Path)
	}

	// 验证 C
	db.First(&updatedC, grandChildC.ID)
	expectedCPathRoot := fmt.Sprintf("/%d/", childB.ID)
	if updatedC.Path != expectedCPathRoot {
		t.Errorf("C cascade update (root) failed. Got Path=%s; Expected Path=%s", updatedC.Path, expectedCPathRoot)
	}

	t.Log("Scenario 3 Passed: Moved to root successfully.")

	// 清理数据
	// db.Exec("DELETE FROM sys_tags WHERE name LIKE 'TestMove_%'")
}
