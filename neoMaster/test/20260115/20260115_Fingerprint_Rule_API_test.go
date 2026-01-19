package test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"neomaster/internal/config"
	"neomaster/internal/handler/asset"
	assetModel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/utils"
	assetRepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/fingerprint"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupFingerprintRuleEnv 初始化测试环境
func setupFingerprintRuleEnv(t *testing.T) (*gin.Engine, *gorm.DB, string) {
	// 1. Setup DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// AutoMigrate
	if err1 := db.AutoMigrate(&assetModel.AssetFinger{}, &assetModel.AssetCPE{}); err1 != nil {
		t.Fatalf("failed to migrate database: %v", err1)
	}

	// 2. Setup Config & Dirs
	tmpDir, err := os.MkdirTemp("", "neoscan_rule_test")
	if err != nil {
		t.Fatal(err)
	}
	// defer os.RemoveAll(tmpDir) // 测试结束后清理，或者保留以供检查

	cfg := &config.Config{
		App: config.AppConfig{
			Rules: config.RulesConfig{
				RootPath: tmpDir,
				Fingerprint: config.RuleDirConfig{
					Dir: "fingerprint",
				},
				Backup: config.RuleDirConfig{
					Dir: "backups",
				},
			},
		},
		Security: config.SecurityConfig{
			Agent: config.AgentConfig{
				RuleEncryptionKey: "test-secret-key-123456",
			},
		},
	}

	// 3. Setup Components
	fingerRepo := assetRepo.NewAssetFingerRepository(db)
	cpeRepo := assetRepo.NewAssetCPERepository(db)
	ruleRepo := assetRepo.NewRuleRepository(db)
	ruleManager := fingerprint.NewRuleManager(ruleRepo, fingerRepo, cpeRepo, cfg.Security.Agent.RuleEncryptionKey, cfg)
	handler := asset.NewFingerprintRuleHandler(ruleManager)

	// 4. Setup Router
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 模拟 Router 中的分组结构
	api := r.Group("/api/v1/asset/fingerprint/rules")
	{
		api.GET("/version", handler.GetVersion)
		api.GET("/backups", handler.ListBackups)
		api.GET("/export", handler.ExportRules)
		api.POST("/import", handler.ImportRules)
		api.POST("/rollback", handler.RollbackRules)
		api.POST("/publish", handler.PublishRules)
	}

	return r, db, tmpDir
}

// TestFingerprintRuleAPI_ImportExport 测试导入导出流程
func TestFingerprintRuleAPI_ImportExport(t *testing.T) {
	r, db, tmpDir := setupFingerprintRuleEnv(t)
	defer os.RemoveAll(tmpDir)

	// 1. 准备测试数据 (JSON 文件)
	ruleContent := `{
		"version": "1.0",
		"source": "NeoScan Export",
		"fingers": [
			{
				"name": "TestCMS",
				"title": "Test CMS Home",
				"body": "Powered by TestCMS",
				"source": "custom",
				"enabled": true
			}
		],
		"cpes": []
	}`

	// 2. 测试 Import (Custom Upload)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "rules.json")
	part.Write([]byte(ruleContent))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/asset/fingerprint/rules/import?overwrite=true&source=custom", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证 DB 数据
	var count int64
	db.Model(&assetModel.AssetFinger{}).Where("source = ?", "custom").Count(&count)
	assert.Equal(t, int64(1), count)

	var finger assetModel.AssetFinger
	db.First(&finger)
	assert.Equal(t, "TestCMS", finger.Name)
	assert.Equal(t, "custom", finger.Source)

	// 3. 测试 Export
	reqExport := httptest.NewRequest("GET", "/api/v1/asset/fingerprint/rules/export", nil)
	wExport := httptest.NewRecorder()
	r.ServeHTTP(wExport, reqExport)

	assert.Equal(t, http.StatusOK, wExport.Code)
	assert.Contains(t, wExport.Body.String(), "TestCMS")

	// 验证签名头
	signature := wExport.Header().Get("X-Content-Signature")
	assert.NotEmpty(t, signature)
}

// TestFingerprintRuleAPI_Rollback 测试回滚流程 (True Rollback)
func TestFingerprintRuleAPI_Rollback(t *testing.T) {
	r, db, tmpDir := setupFingerprintRuleEnv(t)
	defer os.RemoveAll(tmpDir)

	// 1. 初始化 V1 状态 (1个规则)
	v1Rule := assetModel.AssetFinger{Name: "RuleV1", Title: "V1", Source: "system"}
	db.Create(&v1Rule)

	// 2. 创建 V1 备份
	// 由于 API 层面没有直接创建备份的接口(通常是Import时自动创建)，我们这里模拟手动创建备份文件
	// 或者调用 Export 并保存到 backups 目录

	// 模拟 RuleManager.createBackup 逻辑
	// 实际上我们可以通过 Import 一个空文件或者 dummy 文件来触发备份，但为了精确控制，我们手动写入备份文件
	backupDir := filepath.Join(tmpDir, "backups", "fingerprint")
	utils.MkdirAll(backupDir, 0755)

	// 构造 V1 备份数据
	v1Data := `{
		"version": "1.0",
		"source": "System Backup",
		"fingers": [
			{
				"name": "RuleV1",
				"title": "V1",
				"source": "system",
				"enabled": true
			}
		],
		"cpes": []
	}`
	backupFile := "rules_backup_20260115_100000.json"
	utils.WriteFile(filepath.Join(backupDir, backupFile), []byte(v1Data), 0644)

	// 3. 演进到 V2 状态 (新增一个脏数据 RuleV2)
	v2Rule := assetModel.AssetFinger{Name: "RuleV2_Dirty", Title: "V2_Dirty", Source: "custom"}
	db.Create(&v2Rule)

	// 验证当前状态：有2条规则
	var count int64
	db.Model(&assetModel.AssetFinger{}).Count(&count)
	assert.Equal(t, int64(2), count)

	// 4. 执行回滚到 V1
	rollbackReqBody := map[string]string{
		"filename": backupFile,
	}
	jsonBody, _ := json.Marshal(rollbackReqBody)
	req := httptest.NewRequest("POST", "/api/v1/asset/fingerprint/rules/rollback", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 5. 验证回滚结果 (True Rollback)
	// 期望：RuleV2_Dirty 被删除，只剩下 RuleV1
	db.Model(&assetModel.AssetFinger{}).Count(&count)
	assert.Equal(t, int64(1), count, "Should have exactly 1 rule after rollback")

	var remainingRule assetModel.AssetFinger
	db.First(&remainingRule)
	assert.Equal(t, "RuleV1", remainingRule.Name)
}

// TestFingerprintRuleAPI_Publish 测试发布流程
func TestFingerprintRuleAPI_Publish(t *testing.T) {
	r, db, tmpDir := setupFingerprintRuleEnv(t)
	defer os.RemoveAll(tmpDir)

	// 1. 准备 DB 数据
	db.Create(&assetModel.AssetFinger{Name: "PublishRule", Title: "Publish", Enabled: true})
	db.Create(&assetModel.AssetFinger{Name: "DisabledRule", Title: "Disabled"})
	db.Model(&assetModel.AssetFinger{}).Where("name = ?", "DisabledRule").Update("enabled", false)

	// 2. 调用 Publish
	req := httptest.NewRequest("POST", "/api/v1/asset/fingerprint/rules/publish", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 3. 验证文件生成
	targetFile := filepath.Join(tmpDir, "fingerprint", "neoscan_fingerprint_rules.json")
	assert.FileExists(t, targetFile)

	// 4. 验证文件内容 (应只包含 Enabled=true 的规则)
	content, _ := os.ReadFile(targetFile)
	assert.Contains(t, string(content), "PublishRule")
	assert.NotContains(t, string(content), "DisabledRule")
}
