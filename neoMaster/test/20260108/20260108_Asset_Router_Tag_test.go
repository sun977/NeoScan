package test_20260108

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
		&assetModel.AssetNetwork{},
		&assetModel.AssetHost{},
		&assetModel.AssetService{},
		&assetModel.AssetWeb{},
		&assetModel.AssetWhitelist{},
		&assetModel.AssetSkipPolicy{},
		&assetModel.AssetVuln{},
		&assetModel.AssetVulnPoc{},
		&assetModel.AssetUnified{},
		&tagsystem.SysTag{},
		&tagsystem.SysEntityTag{},
	)

	// 2. 初始化所有层级依赖
	tagR := tagRepo.NewTagRepository(db)
	tagS := tagService.NewTagService(tagR, db)

	// Repos
	rawRepo := assetRepo.NewRawAssetRepository(db)
	hostRepo := assetRepo.NewAssetHostRepository(db)
	networkRepo := assetRepo.NewAssetNetworkRepository(db)
	policyRepo := assetRepo.NewAssetPolicyRepository(db)
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)
	scanRepo := assetRepo.NewAssetScanRepository(db)

	// Services
	rawS := assetService.NewRawAssetService(rawRepo)
	hostS := assetService.NewAssetHostService(hostRepo, tagS)
	networkS := assetService.NewAssetNetworkService(networkRepo, tagS)
	policyS := assetService.NewAssetPolicyService(policyRepo, tagS)
	webS := assetService.NewAssetWebService(webRepo, tagS)
	vulnS := assetService.NewAssetVulnService(vulnRepo, tagS)
	unifiedS := assetService.NewAssetUnifiedService(unifiedRepo, tagS)
	scanS := assetService.NewAssetScanService(scanRepo, networkRepo)

	// Handlers
	_ = assetHandler.NewRawAssetHandler(rawS) // rawH
	hostH := assetHandler.NewAssetHostHandler(hostS)
	networkH := assetHandler.NewAssetNetworkHandler(networkS)
	policyH := assetHandler.NewAssetPolicyHandler(policyS)
	webH := assetHandler.NewAssetWebHandler(webS)
	vulnH := assetHandler.NewAssetVulnHandler(vulnS)
	unifiedH := assetHandler.NewAssetUnifiedHandler(unifiedS)
	_ = assetHandler.NewAssetScanHandler(scanS) // scanH

	// 3. 构建 Router
	// 注意：我们需要模拟 Router 结构体中的 setupAssetRoutes 方法
	// 由于 setupAssetRoutes 是私有方法且挂载在 Router 上，我们这里手动构建类似的路由结构
	r := gin.Default()
	v1 := r.Group("/api/v1")
	assetGroup := v1.Group("/asset")

	// Network Routes
	networks := assetGroup.Group("/networks")
	{
		networks.GET("/:id/tags", networkH.GetNetworkTags)
		networks.POST("/:id/tags", networkH.AddNetworkTag)
		networks.DELETE("/:id/tags/:tag_id", networkH.RemoveNetworkTag)
	}

	// Host Routes
	hosts := assetGroup.Group("/hosts")
	{
		hosts.GET("/:id/tags", hostH.GetHostTags)
		hosts.POST("/:id/tags", hostH.AddHostTag)
		hosts.DELETE("/:id/tags/:tag_id", hostH.RemoveHostTag)

		services := hosts.Group("/:id/services")
		{
			services.GET("/:service_id/tags", hostH.GetServiceTags)
			services.POST("/:service_id/tags", hostH.AddServiceTag)
			services.DELETE("/:service_id/tags/:tag_id", hostH.RemoveServiceTag)
		}
	}

	// Web Routes
	webs := assetGroup.Group("/webs")
	{
		webs.GET("/:id/tags", webH.GetWebTags)
		webs.POST("/:id/tags", webH.AddWebTag)
		webs.DELETE("/:id/tags/:tag_id", webH.RemoveWebTag)
	}

	// Policy Routes
	policies := assetGroup.Group("/policies")
	{
		whitelists := policies.Group("/whitelists")
		{
			whitelists.GET("/:id/tags", policyH.GetWhitelistTags)
			whitelists.POST("/:id/tags", policyH.AddWhitelistTag)
			whitelists.DELETE("/:id/tags/:tag_id", policyH.RemoveWhitelistTag)
		}

		skipPolicies := policies.Group("/skip-policies")
		{
			skipPolicies.GET("/:id/tags", policyH.GetSkipPolicyTags)
			skipPolicies.POST("/:id/tags", policyH.AddSkipPolicyTag)
			skipPolicies.DELETE("/:id/tags/:tag_id", policyH.RemoveSkipPolicyTag)
		}
	}

	// Vuln Routes
	vulns := assetGroup.Group("/vulns")
	{
		vulns.GET("/:id/tags", vulnH.GetVulnTags)
		vulns.POST("/:id/tags", vulnH.AddVulnTag)
		vulns.DELETE("/:id/tags/:tag_id", vulnH.RemoveVulnTag)

		vulns.GET("/pocs/:id/tags", vulnH.GetPocTags)
		vulns.POST("/pocs/:id/tags", vulnH.AddPocTag)
		vulns.DELETE("/pocs/:id/tags/:tag_id", vulnH.RemovePocTag)
	}

	// Unified Routes
	unified := assetGroup.Group("/unified")
	{
		unified.GET("/:id/tags", unifiedH.GetUnifiedAssetTags)
		unified.POST("/:id/tags", unifiedH.AddUnifiedAssetTag)
		unified.DELETE("/:id/tags/:tag_id", unifiedH.RemoveUnifiedAssetTag)
	}

	// 4. 创建测试用 Tag
	testTag := &tagsystem.SysTag{
		Name:        "TestTag_API_Router",
		Description: "Tag for Router Test",
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

func TestAssetTagRoutes(t *testing.T) {
	router, db, testTag := setupRouter(t)
	defer func() {
		// 清理 Tag
		db.Delete(testTag)
	}()

	tagID := testTag.ID
	baseAPI := "/api/v1/asset"

	// --- 1. Network Tagging ---
	t.Run("Network Tagging", func(t *testing.T) {
		network := &assetModel.AssetNetwork{CIDR: "10.0.0.0/24", Status: "active", Tags: "{}"}
		db.Create(network)
		defer db.Delete(network)

		// Add Tag
		w := performRequest(router, "POST", fmt.Sprintf("%s/networks/%d/tags", baseAPI, network.ID), map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST Network Tag failed: %d", w.Code)
		}

		// Get Tags
		w = performRequest(router, "GET", fmt.Sprintf("%s/networks/%d/tags", baseAPI, network.ID), nil)
		if w.Code != 200 {
			t.Errorf("GET Network Tags failed: %d", w.Code)
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(testTag.Name)) {
			t.Errorf("GET Network Tags response missing tag name")
		}

		// Remove Tag
		w = performRequest(router, "DELETE", fmt.Sprintf("%s/networks/%d/tags/%d", baseAPI, network.ID, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE Network Tag failed: %d", w.Code)
		}
	})

	// --- 2. Host Tagging ---
	t.Run("Host Tagging", func(t *testing.T) {
		host := &assetModel.AssetHost{IP: "10.0.0.1", Tags: "{}", SourceStageIDs: "[]"}
		db.Create(host)
		defer db.Delete(host)

		// Add Tag
		w := performRequest(router, "POST", fmt.Sprintf("%s/hosts/%d/tags", baseAPI, host.ID), map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST Host Tag failed: %d", w.Code)
		}

		// Get Tags
		w = performRequest(router, "GET", fmt.Sprintf("%s/hosts/%d/tags", baseAPI, host.ID), nil)
		if w.Code != 200 {
			t.Errorf("GET Host Tags failed: %d", w.Code)
		}

		// Remove Tag
		w = performRequest(router, "DELETE", fmt.Sprintf("%s/hosts/%d/tags/%d", baseAPI, host.ID, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE Host Tag failed: %d", w.Code)
		}
	})

	// --- 3. Host Service Tagging ---
	t.Run("Host Service Tagging", func(t *testing.T) {
		// Need a host first
		host := &assetModel.AssetHost{IP: "10.0.0.2", Tags: "{}", SourceStageIDs: "[]"}
		db.Create(host)
		service := &assetModel.AssetService{HostID: host.ID, Port: 80, Name: "http", Tags: "{}", Fingerprint: "{}"}
		db.Create(service)
		defer func() {
			db.Delete(service)
			db.Delete(host)
		}()

		// Add Tag (URL structure: /hosts/:id/services/:service_id/tags)
		// 注意: Router 定义是 services := hosts.Group("/:id/services") -> /:service_id/tags
		// 这里的 :id 是 host_id，但在 Group 内部其实没用到，真正用到的是 :service_id
		url := fmt.Sprintf("%s/hosts/%d/services/%d/tags", baseAPI, host.ID, service.ID)

		w := performRequest(router, "POST", url, map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST Service Tag failed: %d - %s", w.Code, w.Body.String())
		}

		w = performRequest(router, "GET", url, nil)
		if w.Code != 200 {
			t.Errorf("GET Service Tags failed: %d", w.Code)
		}

		w = performRequest(router, "DELETE", fmt.Sprintf("%s/%d", url, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE Service Tag failed: %d", w.Code)
		}
	})

	// --- 4. Web Tagging ---
	t.Run("Web Tagging", func(t *testing.T) {
		web := &assetModel.AssetWeb{URL: "http://example.com", TechStack: "{}", Tags: "{}", BasicInfo: "{}"}
		db.Create(web)
		defer db.Delete(web)

		w := performRequest(router, "POST", fmt.Sprintf("%s/webs/%d/tags", baseAPI, web.ID), map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST Web Tag failed: %d", w.Code)
		}

		w = performRequest(router, "GET", fmt.Sprintf("%s/webs/%d/tags", baseAPI, web.ID), nil)
		if w.Code != 200 {
			t.Errorf("GET Web Tags failed: %d", w.Code)
		}

		w = performRequest(router, "DELETE", fmt.Sprintf("%s/webs/%d/tags/%d", baseAPI, web.ID, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE Web Tag failed: %d", w.Code)
		}
	})

	// --- 5. Whitelist Tagging ---
	t.Run("Whitelist Tagging", func(t *testing.T) {
		wl := &assetModel.AssetWhitelist{TargetType: "ip", TargetValue: "1.2.3.4", WhitelistName: "Test WL", Tags: "{}", Scope: "{}"}
		db.Create(wl)
		defer db.Delete(wl)

		url := fmt.Sprintf("%s/policies/whitelists/%d/tags", baseAPI, wl.ID)
		w := performRequest(router, "POST", url, map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST Whitelist Tag failed: %d", w.Code)
		}

		w = performRequest(router, "GET", url, nil)
		if w.Code != 200 {
			t.Errorf("GET Whitelist Tags failed: %d", w.Code)
		}

		w = performRequest(router, "DELETE", fmt.Sprintf("%s/%d", url, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE Whitelist Tag failed: %d", w.Code)
		}
	})

	// --- 6. SkipPolicy Tagging ---
	t.Run("SkipPolicy Tagging", func(t *testing.T) {
		sp := &assetModel.AssetSkipPolicy{PolicyName: "Test SP", PolicyType: "host", ConditionRules: "{}", ActionConfig: "{}", Scope: "{}", Tags: "{}"}
		db.Create(sp)
		defer db.Delete(sp)

		url := fmt.Sprintf("%s/policies/skip-policies/%d/tags", baseAPI, sp.ID)
		w := performRequest(router, "POST", url, map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST SkipPolicy Tag failed: %d", w.Code)
		}

		w = performRequest(router, "GET", url, nil)
		if w.Code != 200 {
			t.Errorf("GET SkipPolicy Tags failed: %d", w.Code)
		}

		w = performRequest(router, "DELETE", fmt.Sprintf("%s/%d", url, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE SkipPolicy Tag failed: %d", w.Code)
		}
	})

	// --- 7. Vuln Tagging ---
	t.Run("Vuln Tagging", func(t *testing.T) {
		vuln := &assetModel.AssetVuln{TargetType: "host", TargetRefID: 1, CVE: "CVE-2025-0001", Evidence: "{}", Attributes: "{}"}
		db.Create(vuln)
		defer db.Delete(vuln)

		url := fmt.Sprintf("%s/vulns/%d/tags", baseAPI, vuln.ID)
		w := performRequest(router, "POST", url, map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST Vuln Tag failed: %d", w.Code)
		}

		w = performRequest(router, "GET", url, nil)
		if w.Code != 200 {
			t.Errorf("GET Vuln Tags failed: %d", w.Code)
		}

		w = performRequest(router, "DELETE", fmt.Sprintf("%s/%d", url, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE Vuln Tag failed: %d", w.Code)
		}
	})

	// --- 8. PoC Tagging ---
	t.Run("PoC Tagging", func(t *testing.T) {
		// Need a vuln first
		vuln := &assetModel.AssetVuln{TargetType: "host", TargetRefID: 1, CVE: "CVE-2025-0002", Evidence: "{}", Attributes: "{}"}
		db.Create(vuln)
		poc := &assetModel.AssetVulnPoc{VulnID: vuln.ID, Name: "Test PoC"}
		db.Create(poc)
		defer func() {
			db.Delete(poc)
			db.Delete(vuln)
		}()

		url := fmt.Sprintf("%s/vulns/pocs/%d/tags", baseAPI, poc.ID)
		w := performRequest(router, "POST", url, map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST PoC Tag failed: %d", w.Code)
		}

		w = performRequest(router, "GET", url, nil)
		if w.Code != 200 {
			t.Errorf("GET PoC Tags failed: %d", w.Code)
		}

		w = performRequest(router, "DELETE", fmt.Sprintf("%s/%d", url, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE PoC Tag failed: %d", w.Code)
		}
	})

	// --- 9. Unified Asset Tagging ---
	t.Run("Unified Asset Tagging", func(t *testing.T) {
		ua := &assetModel.AssetUnified{IP: "8.8.8.8", Port: 53, ProjectID: 1, TechStack: "{}"}
		db.Create(ua)
		defer db.Delete(ua)

		url := fmt.Sprintf("%s/unified/%d/tags", baseAPI, ua.ID)
		w := performRequest(router, "POST", url, map[string]uint64{"tag_id": tagID})
		if w.Code != 200 {
			t.Errorf("POST UnifiedAsset Tag failed: %d", w.Code)
		}

		w = performRequest(router, "GET", url, nil)
		if w.Code != 200 {
			t.Errorf("GET UnifiedAsset Tags failed: %d", w.Code)
		}

		w = performRequest(router, "DELETE", fmt.Sprintf("%s/%d", url, tagID), nil)
		if w.Code != 200 {
			t.Errorf("DELETE UnifiedAsset Tag failed: %d", w.Code)
		}
	})
}
