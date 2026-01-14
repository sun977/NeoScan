package test_20260114

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	assetHandler "neomaster/internal/handler/asset"
	assetModel "neomaster/internal/model/asset"
	tagsystem "neomaster/internal/model/tag_system"
	assetRepo "neomaster/internal/repo/mysql/asset"
	tagRepo "neomaster/internal/repo/mysql/tag_system"
	assetService "neomaster/internal/service/asset"
	tagService "neomaster/internal/service/tag_system"
)

// setupRouter 构建测试路由环境
func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB, *tagsystem.SysTag) {
	// 1. 初始化数据库
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移所有相关表
	db.AutoMigrate(
		&assetModel.RawAsset{},
		&assetModel.RawAssetNetwork{},
		&tagsystem.SysTag{},
		&tagsystem.SysEntityTag{},
	)

	// 2. 初始化所有层级依赖
	tagR := tagRepo.NewTagRepository(db)
	tagS := tagService.NewTagService(tagR, db)

	// Repos
	rawRepo := assetRepo.NewRawAssetRepository(db)

	// Services
	rawS := assetService.NewRawAssetService(rawRepo, tagS)

	// Handlers
	rawH := assetHandler.NewRawAssetHandler(rawS)

	// 3. 构建 Router
	r := gin.Default()
	v1 := r.Group("/api/v1")
	assetGroup := v1.Group("/asset")

	// Raw Asset Routes
	rawAssets := assetGroup.Group("/raw-assets")
	{
		rawAssets.POST("", rawH.CreateRawAsset)
		rawAssets.GET("/:id", rawH.GetRawAsset)
		rawAssets.GET("/:id/tags", rawH.GetRawAssetTags)
		rawAssets.POST("/:id/tags", rawH.AddRawAssetTag)
		rawAssets.DELETE("/:id/tags/:tag_id", rawH.RemoveRawAssetTag)
	}

	// Raw Network Routes
	rawNetworks := assetGroup.Group("/raw-networks")
	{
		rawNetworks.POST("", rawH.CreateRawNetwork)
		rawNetworks.GET("/:id", rawH.GetRawNetwork)
		rawNetworks.GET("/:id/tags", rawH.GetRawNetworkTags)
		rawNetworks.POST("/:id/tags", rawH.AddRawNetworkTag)
		rawNetworks.DELETE("/:id/tags/:tag_id", rawH.RemoveRawNetworkTag)
	}

	// 4. 创建测试用 Tag
	testTag := &tagsystem.SysTag{
		Name:        "TestTag_RawAsset",
		Description: "Tag for Raw Asset Test",
		Color:       "#00FF00",
	}
	// 清理旧数据
	db.Where("name = ?", testTag.Name).Delete(&tagsystem.SysTag{})
	tagS.CreateTag(context.Background(), testTag)

	return r, db, testTag
}

// 辅助函数：发送请求
func performRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var jsonBytes []byte
	if body != nil {
		jsonBytes, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRawAssetTagRoutes(t *testing.T) {
	router, db, testTag := setupRouter(t)
	defer func() {
		// 清理 Tag
		db.Delete(testTag)
	}()

	tagID := testTag.ID
	baseAPI := "/api/v1/asset"

	// --- 1. Raw Asset Tagging ---
	t.Run("Raw Asset Tagging", func(t *testing.T) {
		rawAsset := &assetModel.RawAsset{
			SourceType:       "manual",
			SourceName:       "test_source",
			ImportedAt:       time.Now(),
			Payload:          "{}",
			AssetMetadata:    "{}",
			Tags:             "{}",
			ProcessingConfig: "{}",
		}
		db.Create(rawAsset)
		defer db.Delete(rawAsset)

		// Add Tag
		w := performRequest(router, "POST", fmt.Sprintf("%s/raw-assets/%d/tags", baseAPI, rawAsset.ID), map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST RawAsset Tag failed: %d - %s", w.Code, w.Body.String())
		}

		// Get Tags
		w = performRequest(router, "GET", fmt.Sprintf("%s/raw-assets/%d/tags", baseAPI, rawAsset.ID), nil)
		if w.Code != 200 {
			t.Errorf("GET RawAsset Tags failed: %d", w.Code)
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(testTag.Name)) {
			t.Errorf("GET RawAsset Tags response missing tag name")
		}

		// Remove Tag
		w = performRequest(router, "DELETE", fmt.Sprintf("%s/raw-assets/%d/tags/%d", baseAPI, rawAsset.ID, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE RawAsset Tag failed: %d", w.Code)
		}
	})

	// --- 2. Raw Network Tagging ---
	t.Run("Raw Network Tagging", func(t *testing.T) {
		rawNetwork := &assetModel.RawAssetNetwork{
			Network: "192.168.100.0/24",
			Status:  "pending",
		}
		db.Create(rawNetwork)
		defer db.Delete(rawNetwork)

		// Add Tag
		w := performRequest(router, "POST", fmt.Sprintf("%s/raw-networks/%d/tags", baseAPI, rawNetwork.ID), map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST RawNetwork Tag failed: %d - %s", w.Code, w.Body.String())
		}

		// Get Tags
		w = performRequest(router, "GET", fmt.Sprintf("%s/raw-networks/%d/tags", baseAPI, rawNetwork.ID), nil)
		if w.Code != 200 {
			t.Errorf("GET RawNetwork Tags failed: %d", w.Code)
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(testTag.Name)) {
			t.Errorf("GET RawNetwork Tags response missing tag name")
		}

		// Remove Tag
		w = performRequest(router, "DELETE", fmt.Sprintf("%s/raw-networks/%d/tags/%d", baseAPI, rawNetwork.ID, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE RawNetwork Tag failed: %d", w.Code)
		}
	})
}
