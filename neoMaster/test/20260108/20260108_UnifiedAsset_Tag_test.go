package test_20260108

import (
	"context"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	assetModel "neomaster/internal/model/asset"
	tagsystem "neomaster/internal/model/tag_system"
	assetRepo "neomaster/internal/repo/mysql/asset"
	tagRepo "neomaster/internal/repo/mysql/tag_system"
	assetService "neomaster/internal/service/asset"
	tagService "neomaster/internal/service/tag_system"
)

func setupDB(t *testing.T) *gorm.DB {
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	// 确保 AssetUnified 表存在
	db.AutoMigrate(&assetModel.AssetUnified{})
	return db
}

func TestUnifiedAssetTagging(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// 1. 初始化 Services
	tagR := tagRepo.NewTagRepository(db)
	tagS := tagService.NewTagService(tagR, db)

	unifiedR := assetRepo.NewAssetUnifiedRepository(db)
	unifiedS := assetService.NewAssetUnifiedService(unifiedR, tagS)

	// 2. 准备测试数据
	// 2.1 创建测试标签
	testTag := &tagsystem.SysTag{
		Name:        "TestTag_Unified_01",
		Description: "Tag for Unified Asset Test",
		Color:       "#FF0000",
	}
	// 清理旧标签
	db.Where("name = ?", testTag.Name).Delete(&tagsystem.SysTag{})
	if err := tagS.CreateTag(ctx, testTag); err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}
	t.Logf("Created Tag: ID=%d, Name=%s", testTag.ID, testTag.Name)

	// 2.2 创建测试 Unified Asset
	testAsset := &assetModel.AssetUnified{
		ProjectID: 999,
		IP:        "1.1.1.1",
		Port:      8080,
		Service:   "http",
		IsWeb:     true,
		Source:    "test",
		TechStack: "{}",
	}
	// 清理旧资产
	db.Where("ip = ? AND port = ?", testAsset.IP, testAsset.Port).Delete(&assetModel.AssetUnified{})
	if err := unifiedS.CreateUnifiedAsset(ctx, testAsset); err != nil {
		t.Fatalf("Failed to create unified asset: %v", err)
	}
	t.Logf("Created Asset: ID=%d, IP=%s", testAsset.ID, testAsset.IP)

	// 清理函数
	defer func() {
		tagS.DeleteTag(ctx, testTag.ID, true)
		unifiedS.DeleteUnifiedAsset(ctx, testAsset.ID)
	}()

	// 3. 测试 AddUnifiedAssetTag
	err := unifiedS.AddUnifiedAssetTag(ctx, testAsset.ID, testTag.ID)
	if err != nil {
		t.Fatalf("AddUnifiedAssetTag failed: %v", err)
	}
	t.Log("AddUnifiedAssetTag success")

	// 4. 测试 GetUnifiedAssetTags
	tags, err := unifiedS.GetUnifiedAssetTags(ctx, testAsset.ID)
	if err != nil {
		t.Fatalf("GetUnifiedAssetTags failed: %v", err)
	}
	found := false
	for _, tag := range tags {
		if tag.ID == testTag.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Tag not found in asset tags. Got: %v", tags)
	} else {
		t.Log("GetUnifiedAssetTags verified tag presence")
	}

	// 5. 测试 ListUnifiedAssets with Tag Filtering
	assets, _, err := unifiedS.ListUnifiedAssets(ctx, 1, 10, assetRepo.UnifiedAssetFilter{}, []uint64{testTag.ID})
	if err != nil {
		t.Fatalf("ListUnifiedAssets failed: %v", err)
	}
	foundAsset := false
	for _, a := range assets {
		if a.ID == testAsset.ID {
			foundAsset = true
			break
		}
	}
	if !foundAsset {
		t.Errorf("Asset not found when filtering by tag. Got %d assets", len(assets))
	} else {
		t.Log("ListUnifiedAssets filtering verified")
	}

	// 6. 测试 RemoveUnifiedAssetTag
	err = unifiedS.RemoveUnifiedAssetTag(ctx, testAsset.ID, testTag.ID)
	if err != nil {
		t.Fatalf("RemoveUnifiedAssetTag failed: %v", err)
	}
	t.Log("RemoveUnifiedAssetTag success")

	// 7. Verify Removal
	tags, err = unifiedS.GetUnifiedAssetTags(ctx, testAsset.ID)
	if err != nil {
		t.Fatalf("GetUnifiedAssetTags failed: %v", err)
	}
	for _, tag := range tags {
		if tag.ID == testTag.ID {
			t.Errorf("Tag should have been removed but was found")
		}
	}
	t.Log("Verify Removal success")
}
