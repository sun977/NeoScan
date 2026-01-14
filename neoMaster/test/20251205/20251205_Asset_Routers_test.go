package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"neomaster/internal/app/master/router"
	"neomaster/internal/config"
	assetmodel "neomaster/internal/model/asset"
	systemmodel "neomaster/internal/model/system"
	tagsystem "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/auth"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// SetupTestEnv 鍒濆鍖栨祴璇曠幆澧?
func SetupTestEnv() (*gin.Engine, *gorm.DB, string, error) {
	// 1. 鍒濆鍖栨棩蹇?
	logger.InitLogger(&config.LogConfig{
		Level:  "error", // 娴嬭瘯鏃跺噺灏戞棩蹇楄緭鍑?
		Format: "json",
		Output: "console",
	})

	// 2. 鏁版嵁搴撻厤缃?(neoscan_dev)
	dbConfig := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		Username:        "root",
		Password:        "ROOT",
		Database:        "neoscan_dev",
		Charset:         "utf8mb4",
		ParseTime:       true,
		Loc:             "Local",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
	}

	// 杩炴帴 MySQL
	db, err := database.NewMySQLConnection(dbConfig)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to connect to mysql: %v", err)
	}

	// 3. Redis 閰嶇疆
	redisConfig := &config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		Database:     0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
	}

	// 杩炴帴 Redis
	redisClient, err := database.NewRedisConnection(redisConfig)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to connect to redis: %v", err)
	}

	// 4. 鏋勫缓瀹屾暣 Config 瀵硅薄 (Router 闇€瑕?
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			MySQL: *dbConfig,
			Redis: *redisConfig,
		},
		Security: config.SecurityConfig{
			JWT: config.JWTConfig{
				Secret:            "neoscan_jwt_secret_key_at_least_32_characters_long",
				Issuer:            "neoscan",
				AccessTokenExpire: 24 * time.Hour,
			},
			Auth: config.AuthConfig{
				SkipPaths:         []string{"/health"},
				WhitelistIPs:      []string{"9.9.9.9"},
				EnableIPWhitelist: true,
			},
			RateLimit: config.RateLimitConfig{
				Enabled: false, // 娴嬭瘯鏃朵笉闄愭祦
			},
			Logging: config.LoggingConfig{
				EnableRequestLog: false, // 鍑忓皯娴嬭瘯鏃ュ織
			},
		},
	}

	if err1 := ensureTestUserActive(db, 2); err1 != nil {
		return nil, nil, "", err1
	}

	// 5. 鍒濆鍖?Router
	gin.SetMode(gin.TestMode)
	appRouter := router.NewRouter(db, redisClient, cfg)
	// 娉ㄥ唽璺敱
	appRouter.SetupRoutes()

	// 6. 鐢熸垚娴嬭瘯 Token
	jwtManager := auth.NewJWTManager(cfg.Security.JWT.Secret, cfg.Security.JWT.AccessTokenExpire, 7*24*time.Hour)
	token, err := jwtManager.GenerateAccessToken(1, "testuser", "test@example.com", 0, []string{"admin"})
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate token: %v", err)
	}

	return appRouter.GetEngine(), db, token, nil
}

func ensureTestUserActive(db *gorm.DB, userID uint) error {
	var user systemmodel.User
	err := db.First(&user, userID).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to query users table: %w", err)
	}

	user = systemmodel.User{
		ID:       userID,
		Username: fmt.Sprintf("sysuser_test_%d", userID),
		Email:    fmt.Sprintf("sysuser_test_%d@neoscan.local", userID),
		Password: "test_password_placeholder",
		Status:   systemmodel.UserStatusEnabled,
	}

	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}
	return nil
}

func performJSONRequest(engine *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var jsonBytes []byte
	if body != nil {
		jsonBytes, _ = json.Marshal(body)
	}

	req, _ := http.NewRequest(method, path, bytes.NewBuffer(jsonBytes))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func performJSONRequestToServer(t *testing.T, client *http.Client, baseURL, method, path string, body interface{}, headers map[string]string) (int, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		assert.NoError(t, err)
		bodyReader = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	assert.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("X-Forwarded-For", "9.9.9.9")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	return resp.StatusCode, respBody
}

func extractCreatedIDFromBytes(t *testing.T, body []byte) uint64 {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing data field: %s", string(body))
	}
	idFloat, ok := data["id"].(float64)
	if !ok {
		t.Fatalf("missing id field: %s", string(body))
	}
	return uint64(idFloat)
}

func responseContainsIDFromBytes(t *testing.T, body []byte, id uint64) bool {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return false
	}
	items, ok := data["data"].([]interface{})
	if !ok {
		return false
	}
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		idFloat, ok := m["id"].(float64)
		if !ok {
			continue
		}
		if uint64(idFloat) == id {
			return true
		}
	}
	return false
}

func extractCreatedID(t *testing.T, w *httptest.ResponseRecorder) uint64 {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing data field: %s", w.Body.String())
	}
	idFloat, ok := data["id"].(float64)
	if !ok {
		t.Fatalf("missing id field: %s", w.Body.String())
	}
	return uint64(idFloat)
}

func responseContainsID(t *testing.T, w *httptest.ResponseRecorder, id uint64) bool {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return false
	}
	items, ok := data["data"].([]interface{})
	if !ok {
		return false
	}
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		idFloat, ok := m["id"].(float64)
		if !ok {
			continue
		}
		if uint64(idFloat) == id {
			return true
		}
	}
	return false
}

func TestAssetFingerRuleRoutes(t *testing.T) {
	engine, db, _, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.AutoMigrate(&assetmodel.AssetFinger{}, &tagsystem.SysTag{}, &tagsystem.SysEntityTag{})

	now := time.Now().UnixNano()
	ruleName := fmt.Sprintf("test_finger_rule_%d", now)
	updatedName := fmt.Sprintf("test_finger_rule_updated_%d", now)
	tagName := fmt.Sprintf("test_tag_fingers_%d", now)

	tag := &tagsystem.SysTag{
		Name:     tagName,
		ParentID: 0,
		Path:     "/",
		Level:    0,
		Category: "asset",
	}
	if err := db.Create(tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}
	defer db.Delete(tag)

	var ruleID uint64

	t.Run("CreateFingerRule", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":        ruleName,
			"status_code": "200",
			"url":         "/",
			"match":       "contains",
			"body":        "neoscan_test",
		}
		w := performJSONRequest(engine, "POST", "/api/v1/asset/fingers", payload, nil)
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		ruleID = extractCreatedID(t, w)
	})

	if ruleID == 0 {
		t.Fatalf("failed to create finger rule")
	}
	defer db.Exec("DELETE FROM asset_finger WHERE id = ?", ruleID)
	defer db.Exec("DELETE FROM sys_entity_tags WHERE entity_type = ? AND entity_id = ?", "fingers_cms", strconv.FormatUint(ruleID, 10))

	t.Run("GetFingerRule", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/asset/fingers/%d", ruleID)
		w := performJSONRequest(engine, "GET", url, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("UpdateFingerRule", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/asset/fingers/%d", ruleID)
		payload := map[string]interface{}{
			"name": updatedName,
		}
		w := performJSONRequest(engine, "PUT", url, payload, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("ListFingerRules", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/asset/fingers?page=1&page_size=10&name=%s", updatedName)
		w := performJSONRequest(engine, "GET", url, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.True(t, responseContainsID(t, w, ruleID), "list response missing created rule")
	})

	t.Run("TagFingerRule", func(t *testing.T) {
		addURL := fmt.Sprintf("/api/v1/asset/fingers/%d/tags", ruleID)
		w := performJSONRequest(engine, "POST", addURL, map[string]uint64{"tag_id": tag.ID}, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		getURL := fmt.Sprintf("/api/v1/asset/fingers/%d/tags", ruleID)
		w = performJSONRequest(engine, "GET", getURL, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.True(t, bytes.Contains(w.Body.Bytes(), []byte(tagName)), "get tags response missing tag")

		listByTagURL := fmt.Sprintf("/api/v1/asset/fingers?page=1&page_size=10&tag_id=%d", tag.ID)
		w = performJSONRequest(engine, "GET", listByTagURL, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.True(t, responseContainsID(t, w, ruleID), "tag filter list missing created rule")

		delURL := fmt.Sprintf("/api/v1/asset/fingers/%d/tags/%d", ruleID, tag.ID)
		w = performJSONRequest(engine, "DELETE", delURL, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("DeleteFingerRule", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/asset/fingers/%d", ruleID)
		w := performJSONRequest(engine, "DELETE", url, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		w = performJSONRequest(engine, "GET", url, nil, nil)
		assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	})
}

func TestAssetCPERuleRoutes(t *testing.T) {
	engine, db, _, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.AutoMigrate(&assetmodel.AssetCPE{}, &tagsystem.SysTag{}, &tagsystem.SysEntityTag{})

	now := time.Now().UnixNano()
	ruleName := fmt.Sprintf("test_cpe_rule_%d", now)
	updatedMatch := fmt.Sprintf("(?i)NeoScanTest/%d", now)
	tagName := fmt.Sprintf("test_tag_cpes_%d", now)

	tag := &tagsystem.SysTag{
		Name:     tagName,
		ParentID: 0,
		Path:     "/",
		Level:    0,
		Category: "asset",
	}
	if err := db.Create(tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}
	defer db.Delete(tag)

	var ruleID uint64

	t.Run("CreateCPERule", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":      ruleName,
			"probe":     "GenericLines",
			"match_str": "(?i)NeoScanTest/([\\d\\.]+)",
			"vendor":    "neoscan",
			"product":   "test_service",
			"part":      "a",
			"cpe":       "cpe:2.3:a:neoscan:test_service:$1:*:*:*:*:*:*:*",
		}
		w := performJSONRequest(engine, "POST", "/api/v1/asset/cpes", payload, nil)
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		ruleID = extractCreatedID(t, w)
	})

	if ruleID == 0 {
		t.Fatalf("failed to create cpe rule")
	}
	defer db.Exec("DELETE FROM asset_cpe WHERE id = ?", ruleID)
	defer db.Exec("DELETE FROM sys_entity_tags WHERE entity_type = ? AND entity_id = ?", "fingers_cpe", strconv.FormatUint(ruleID, 10))

	t.Run("GetCPERule", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/asset/cpes/%d", ruleID)
		w := performJSONRequest(engine, "GET", url, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("UpdateCPERule", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/asset/cpes/%d", ruleID)
		payload := map[string]interface{}{
			"name":      ruleName,
			"match_str": updatedMatch,
		}
		w := performJSONRequest(engine, "PUT", url, payload, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("ListCPERules", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/asset/cpes?page=1&page_size=10&name=%s", ruleName)
		w := performJSONRequest(engine, "GET", url, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.True(t, responseContainsID(t, w, ruleID), "list response missing created rule")
	})

	t.Run("TagCPERule", func(t *testing.T) {
		addURL := fmt.Sprintf("/api/v1/asset/cpes/%d/tags", ruleID)
		w := performJSONRequest(engine, "POST", addURL, map[string]uint64{"tag_id": tag.ID}, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		getURL := fmt.Sprintf("/api/v1/asset/cpes/%d/tags", ruleID)
		w = performJSONRequest(engine, "GET", getURL, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.True(t, bytes.Contains(w.Body.Bytes(), []byte(tagName)), "get tags response missing tag")

		listByTagURL := fmt.Sprintf("/api/v1/asset/cpes?page=1&page_size=10&tag_id=%d", tag.ID)
		w = performJSONRequest(engine, "GET", listByTagURL, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.True(t, responseContainsID(t, w, ruleID), "tag filter list missing created rule")

		delURL := fmt.Sprintf("/api/v1/asset/cpes/%d/tags/%d", ruleID, tag.ID)
		w = performJSONRequest(engine, "DELETE", delURL, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("DeleteCPERule", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/asset/cpes/%d", ruleID)
		w := performJSONRequest(engine, "DELETE", url, nil, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		w = performJSONRequest(engine, "GET", url, nil, nil)
		assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	})
}

func TestAssetFingerRuleRoutes_SmokeServer(t *testing.T) {
	engine, db, _, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.AutoMigrate(&assetmodel.AssetFinger{}, &tagsystem.SysTag{}, &tagsystem.SysEntityTag{})

	ts := httptest.NewServer(engine)
	defer ts.Close()
	client := ts.Client()

	now := time.Now().UnixNano()
	ruleName := fmt.Sprintf("smoke_finger_rule_%d", now)
	tagName := fmt.Sprintf("smoke_tag_fingers_%d", now)

	tag := &tagsystem.SysTag{
		Name:     tagName,
		ParentID: 0,
		Path:     "/",
		Level:    0,
		Category: "asset",
	}
	if err := db.Create(tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}
	defer db.Delete(tag)

	createPayload := map[string]interface{}{
		"name":        ruleName,
		"status_code": "200",
		"url":         "/",
		"match":       "contains",
		"body":        "neoscan_smoke_test",
	}
	status, respBody := performJSONRequestToServer(t, client, ts.URL, "POST", "/api/v1/asset/fingers", createPayload, nil)
	assert.Equal(t, http.StatusCreated, status, string(respBody))

	ruleID := extractCreatedIDFromBytes(t, respBody)
	if ruleID == 0 {
		t.Fatalf("failed to create finger rule")
	}
	defer db.Exec("DELETE FROM asset_finger WHERE id = ?", ruleID)
	defer db.Exec("DELETE FROM sys_entity_tags WHERE entity_type = ? AND entity_id = ?", "fingers_cms", strconv.FormatUint(ruleID, 10))

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "GET", fmt.Sprintf("/api/v1/asset/fingers/%d", ruleID), nil, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "GET", fmt.Sprintf("/api/v1/asset/fingers?page=1&page_size=10&name=%s", ruleName), nil, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))
	assert.True(t, responseContainsIDFromBytes(t, respBody, ruleID), "list response missing created rule")

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "POST", fmt.Sprintf("/api/v1/asset/fingers/%d/tags", ruleID), map[string]uint64{"tag_id": tag.ID}, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "GET", fmt.Sprintf("/api/v1/asset/fingers/%d/tags", ruleID), nil, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))
	assert.True(t, bytes.Contains(respBody, []byte(tagName)), "get tags response missing tag")

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "DELETE", fmt.Sprintf("/api/v1/asset/fingers/%d", ruleID), nil, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))
}

func TestAssetCPERuleRoutes_SmokeServer(t *testing.T) {
	engine, db, _, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.AutoMigrate(&assetmodel.AssetCPE{}, &tagsystem.SysTag{}, &tagsystem.SysEntityTag{})

	ts := httptest.NewServer(engine)
	defer ts.Close()
	client := ts.Client()

	now := time.Now().UnixNano()
	ruleName := fmt.Sprintf("smoke_cpe_rule_%d", now)
	tagName := fmt.Sprintf("smoke_tag_cpes_%d", now)

	tag := &tagsystem.SysTag{
		Name:     tagName,
		ParentID: 0,
		Path:     "/",
		Level:    0,
		Category: "asset",
	}
	if err := db.Create(tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}
	defer db.Delete(tag)

	createPayload := map[string]interface{}{
		"name":      ruleName,
		"probe":     "GenericLines",
		"match_str": "(?i)NeoScanSmokeTest/([\\d\\.]+)",
		"vendor":    "neoscan",
		"product":   "smoke_service",
		"part":      "a",
		"cpe":       "cpe:2.3:a:neoscan:smoke_service:$1:*:*:*:*:*:*:*",
	}
	status, respBody := performJSONRequestToServer(t, client, ts.URL, "POST", "/api/v1/asset/cpes", createPayload, nil)
	assert.Equal(t, http.StatusCreated, status, string(respBody))

	ruleID := extractCreatedIDFromBytes(t, respBody)
	if ruleID == 0 {
		t.Fatalf("failed to create cpe rule")
	}
	defer db.Exec("DELETE FROM asset_cpe WHERE id = ?", ruleID)
	defer db.Exec("DELETE FROM sys_entity_tags WHERE entity_type = ? AND entity_id = ?", "fingers_cpe", strconv.FormatUint(ruleID, 10))

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "GET", fmt.Sprintf("/api/v1/asset/cpes/%d", ruleID), nil, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "GET", fmt.Sprintf("/api/v1/asset/cpes?page=1&page_size=10&name=%s", ruleName), nil, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))
	assert.True(t, responseContainsIDFromBytes(t, respBody, ruleID), "list response missing created rule")

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "POST", fmt.Sprintf("/api/v1/asset/cpes/%d/tags", ruleID), map[string]uint64{"tag_id": tag.ID}, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "GET", fmt.Sprintf("/api/v1/asset/cpes/%d/tags", ruleID), nil, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))
	assert.True(t, bytes.Contains(respBody, []byte(tagName)), "get tags response missing tag")

	status, respBody = performJSONRequestToServer(t, client, ts.URL, "DELETE", fmt.Sprintf("/api/v1/asset/cpes/%d", ruleID), nil, nil)
	assert.Equal(t, http.StatusOK, status, string(respBody))
}

// TestRawAssetRoutes 娴嬭瘯鍘熷璧勪骇鐩稿叧璺敱
func TestRawAssetRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 鑷姩杩佺Щ琛ㄧ粨鏋?
	db.AutoMigrate(&assetmodel.RawAsset{})

	// 娓呯悊娴嬭瘯鏁版嵁
	defer func() {
		db.Exec("DELETE FROM raw_assets WHERE source_type = ?", "test_source")
	}()

	var rawAssetID uint64

	// 1. 娴嬭瘯鍒涘缓鍘熷璧勪骇 (CreateRawAsset)
	t.Run("CreateRawAsset", func(t *testing.T) {
		rawAsset := map[string]interface{}{
			"source_type":       "test_source",
			"source_name":       "test_import",
			"payload":           `{"ip": "192.168.1.1", "type": "server"}`,
			"priority":          1,
			"asset_metadata":    "{}",
			"processing_config": "{}",
			"imported_at":       time.Now(),
		}
		body, _ := json.Marshal(rawAsset)

		req, _ := http.NewRequest("POST", "/api/v1/asset/raw-assets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		if data, ok := response["data"].(map[string]interface{}); ok {
			rawAssetID = uint64(data["id"].(float64))
		}
	})

	// 2. 娴嬭瘯鑾峰彇鍘熷璧勪骇璇︽儏 (GetRawAsset)
	if rawAssetID > 0 {
		t.Run("GetRawAsset", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/raw-assets/%d", rawAssetID)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 3. 测试更新原始资产状态 (UpdateRawAssetStatus)
		t.Run("UpdateRawAssetStatus", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/raw-assets/%d/status", rawAssetID)
			payload := map[string]interface{}{
				"status": "processed",
			}
			body, _ := json.Marshal(payload)
			req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	// 4. 娴嬭瘯鑾峰彇鍘熷璧勪骇鍒楄〃 (ListRawAssets)
	t.Run("ListRawAssets", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/raw-assets?page=1&page_size=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestRawNetworkRoutes 娴嬭瘯寰呭鐞嗙綉娈电浉鍏宠矾鐢?
func TestRawNetworkRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.AutoMigrate(&assetmodel.RawAssetNetwork{})
	defer db.Exec("DELETE FROM raw_asset_networks WHERE network = ?", "10.10.10.0/24")

	var networkID uint64

	// 1. CreateRawNetwork
	t.Run("CreateRawNetwork", func(t *testing.T) {
		network := map[string]interface{}{
			"network": "10.10.10.0/24",
			"name":    "Test Raw Network",
		}
		body, _ := json.Marshal(network)
		req, _ := http.NewRequest("POST", "/api/v1/asset/raw-networks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if data, ok := response["data"].(map[string]interface{}); ok {
			networkID = uint64(data["id"].(float64))
		}
	})

	if networkID > 0 {
		// 2. GetRawNetwork
		t.Run("GetRawNetwork", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/raw-networks/%d", networkID)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 3. ApproveRawNetwork
		t.Run("ApproveRawNetwork", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/raw-networks/%d/approve", networkID)
			req, _ := http.NewRequest("POST", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			// 杩欓噷鐨勭姸鎬佺爜鍙栧喅浜庡叿浣撶殑瀹炵幇锛岄€氬父鏄?00 OK
			assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated)
		})

		// 4. RejectRawNetwork (Create a new one to reject)
		t.Run("RejectRawNetwork", func(t *testing.T) {
			// Create another one first
			net := assetmodel.RawAssetNetwork{Network: "10.10.20.0/24", Status: "pending", Tags: "{}"}
			db.Create(&net)
			defer db.Delete(&net)

			url := fmt.Sprintf("/api/v1/asset/raw-networks/%d/reject", net.ID)
			req, _ := http.NewRequest("POST", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	// 5. ListRawNetworks
	t.Run("ListRawNetworks", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/raw-networks", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestHostAssetRoutes 娴嬭瘯涓绘満璧勪骇鐩稿叧璺敱
func TestHostAssetRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.Migrator().DropTable(&assetmodel.AssetHost{})
	db.AutoMigrate(&assetmodel.AssetHost{})
	defer db.Exec("DELETE FROM asset_hosts WHERE ip = ?", "10.0.0.1")

	var hostID uint64

	// 1. CreateHost
	t.Run("CreateHost", func(t *testing.T) {
		host := map[string]interface{}{
			"ip":               "10.0.0.1",
			"os_type":          "Linux",
			"hostname":         "test-host",
			"status":           "active",
			"tags":             "{}",
			"source_stage_ids": "[]",
		}
		body, _ := json.Marshal(host)
		req, _ := http.NewRequest("POST", "/api/v1/asset/hosts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.True(t, w.Code == http.StatusCreated || w.Code == http.StatusOK)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if data, ok := response["data"].(map[string]interface{}); ok {
			hostID = uint64(data["id"].(float64))
		}
	})

	if hostID > 0 {
		// 2. GetHost
		t.Run("GetHost", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/hosts/%d", hostID)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 3. UpdateHost
		t.Run("UpdateHost", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/hosts/%d", hostID)
			host := map[string]interface{}{
				"hostname": "updated-host",
			}
			body, _ := json.Marshal(host)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 4. ListServicesByHost
		t.Run("ListServicesByHost", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/hosts/%d/services", hostID)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 5. DeleteHost
		t.Run("DeleteHost", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/hosts/%d", hostID)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	// 6. ListHosts
	t.Run("ListHosts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/hosts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestNetworkRoutes 娴嬭瘯缃戞鐩稿叧璺敱
func TestNetworkRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.Migrator().DropTable(&assetmodel.AssetNetwork{})
	db.AutoMigrate(&assetmodel.AssetNetwork{})
	defer db.Exec("DELETE FROM asset_networks WHERE cidr = ?", "192.168.100.0/24")

	var networkID uint64

	// 1. CreateNetwork
	t.Run("CreateNetwork", func(t *testing.T) {
		network := map[string]interface{}{
			"cidr": "192.168.100.0/24",
			"name": "Test Network",
		}
		body, _ := json.Marshal(network)
		req, _ := http.NewRequest("POST", "/api/v1/asset/networks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if data, ok := response["data"].(map[string]interface{}); ok {
			networkID = uint64(data["id"].(float64))
		}
	})

	if networkID > 0 {
		// 2. GetNetwork
		t.Run("GetNetwork", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/networks/%d", networkID)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 3. UpdateNetwork
		t.Run("UpdateNetwork", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/networks/%d", networkID)
			network := map[string]interface{}{
				"name": "Updated Network",
			}
			body, _ := json.Marshal(network)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 4. UpdateScanStatus
		t.Run("UpdateScanStatus", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/networks/%d/scan-status", networkID)
			payload := map[string]interface{}{
				"status": "scanning",
			}
			body, _ := json.Marshal(payload)
			req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 5. DeleteNetwork
		t.Run("DeleteNetwork", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/networks/%d", networkID)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	// 6. ListNetworks
	t.Run("ListNetworks", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/networks", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestPolicyRoutes 娴嬭瘯绛栫暐绠＄悊鐩稿叧璺敱
func TestPolicyRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.AutoMigrate(&assetmodel.AssetWhitelist{}, &assetmodel.AssetSkipPolicy{})
	defer func() {
		db.Exec("DELETE FROM asset_whitelists WHERE whitelist_name = ?", "Test Whitelist")
		db.Exec("DELETE FROM asset_skip_policies WHERE policy_name = ?", "Test Skip Policy")
	}()

	var whitelistID uint64
	var skipPolicyID uint64

	// --- Whitelist Tests ---
	t.Run("CreateWhitelist", func(t *testing.T) {
		whitelist := map[string]interface{}{
			"whitelist_name": "Test Whitelist",
			"target_type":    "ip",
			"target_value":   "127.0.0.1",
			"enabled":        true,
			"tags":           "{}",
			"scope":          "{}",
		}
		body, _ := json.Marshal(whitelist)
		req, _ := http.NewRequest("POST", "/api/v1/asset/policies/whitelists", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if data, ok := response["data"].(map[string]interface{}); ok {
			whitelistID = uint64(data["id"].(float64))
		}
	})

	if whitelistID > 0 {
		t.Run("GetWhitelist", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/policies/whitelists/%d", whitelistID)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("UpdateWhitelist", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/policies/whitelists/%d", whitelistID)
			whitelist := map[string]interface{}{
				"description": "Updated description",
			}
			body, _ := json.Marshal(whitelist)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("DeleteWhitelist", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/policies/whitelists/%d", whitelistID)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	t.Run("ListWhitelists", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/policies/whitelists", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// --- Skip Policy Tests ---
	t.Run("CreateSkipPolicy", func(t *testing.T) {
		policy := map[string]interface{}{
			"policy_name":     "Test Skip Policy",
			"policy_type":     "temp",
			"enabled":         true,
			"condition_rules": "{}",
			"action_config":   "{}",
			"scope":           "{}",
			"tags":            "{}",
		}
		body, _ := json.Marshal(policy)
		req, _ := http.NewRequest("POST", "/api/v1/asset/policies/skip-policies", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if data, ok := response["data"].(map[string]interface{}); ok {
			skipPolicyID = uint64(data["id"].(float64))
		}
	})

	if skipPolicyID > 0 {
		t.Run("GetSkipPolicy", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/policies/skip-policies/%d", skipPolicyID)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("UpdateSkipPolicy", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/policies/skip-policies/%d", skipPolicyID)
			policy := map[string]interface{}{
				"description": "Updated description",
			}
			body, _ := json.Marshal(policy)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("DeleteSkipPolicy", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/policies/skip-policies/%d", skipPolicyID)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	t.Run("ListSkipPolicies", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/policies/skip-policies", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestWebAssetRoutes 娴嬭瘯Web璧勪骇鐩稿叧璺敱
func TestWebAssetRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	db.Migrator().DropTable(&assetmodel.AssetWeb{}, &assetmodel.AssetWebDetail{})
	db.AutoMigrate(&assetmodel.AssetWeb{}, &assetmodel.AssetWebDetail{})
	defer func() {
		db.Exec("DELETE FROM asset_web_details WHERE asset_web_id IN (SELECT id FROM asset_webs WHERE url = ?)", "http://test.com")
		db.Exec("DELETE FROM asset_webs WHERE url = ?", "http://test.com")
	}()

	var webID uint64

	// 1. CreateWeb
	t.Run("CreateWeb", func(t *testing.T) {
		web := map[string]interface{}{
			"url":        "http://test.com",
			"asset_type": "web",
			"status":     "active",
			"tags":       "{}",
			"tech_stack": "{}",
			"basic_info": "{}",
		}
		body, _ := json.Marshal(web)
		req, _ := http.NewRequest("POST", "/api/v1/asset/webs", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			webID = uint64(data["id"].(float64))
		}
	})

	if webID > 0 {
		// 2. GetWeb
		t.Run("GetWeb", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/webs/%d", webID)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 3. UpdateWeb
		t.Run("UpdateWeb", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/webs/%d", webID)
			web := map[string]interface{}{
				"domain": "updated.test.com",
			}
			body, _ := json.Marshal(web)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 4. Web Detail Operations
		t.Run("UpdateWebDetail", func(t *testing.T) {
			detail := map[string]interface{}{
				"asset_web_id":    webID,
				"crawl_status":    "completed",
				"screenshot":      "base64...",
				"content_details": "{}",
			}
			body, _ := json.Marshal(detail)
			url := "/api/v1/asset/webs/" + strconv.FormatUint(webID, 10) + "/detail"
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("GetWebDetail", func(t *testing.T) {
			url := "/api/v1/asset/webs/" + strconv.FormatUint(webID, 10) + "/detail"
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 5. DeleteWeb
		t.Run("DeleteWeb", func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/asset/webs/%d", webID)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	// 6. ListWebs
	t.Run("ListWebs", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/webs", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestVulnAssetRoutes 娴嬭瘯婕忔礊璧勪骇鐩稿叧璺敱
func TestVulnAssetRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 纭繚琛ㄧ粨鏋勬渶鏂?
	db.Migrator().DropTable(&assetmodel.AssetVuln{}, &assetmodel.AssetVulnPoc{}, &assetmodel.AssetHost{})
	db.AutoMigrate(&assetmodel.AssetVuln{}, &assetmodel.AssetVulnPoc{}, &assetmodel.AssetHost{})
	defer func() {
		db.Exec("DELETE FROM asset_vuln_pocs")
		db.Exec("DELETE FROM asset_vulns")
		db.Exec("DELETE FROM asset_hosts")
	}()

	// 准备前置数据：主机资产
	host := assetmodel.AssetHost{
		IP:             "192.168.1.200",
		OS:             "Linux",
		Tags:           "{}",
		SourceStageIDs: "[]",
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("Failed to create prerequisite host: %v", err)
	}

	var vulnID uint64
	var pocID uint64

	t.Run("CreateVuln", func(t *testing.T) {
		vuln := map[string]interface{}{
			"target_type":   "host",
			"target_ref_id": host.ID,
			"cve":           "CVE-2023-1234",
			"severity":      "high",
			"status":        "open",
			"evidence":      "{}",
			"attributes":    "{}",
		}
		body, _ := json.Marshal(vuln)

		req, _ := http.NewRequest("POST", "/api/v1/asset/vulns", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			vulnID = uint64(data["id"].(float64))
		}
	})

	t.Run("ListVulns", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/vulns", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	if vulnID > 0 {
		t.Run("GetVuln", func(t *testing.T) {
			url := "/api/v1/asset/vulns/" + strconv.FormatUint(vulnID, 10)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("UpdateVuln", func(t *testing.T) {
			update := map[string]interface{}{
				"severity": "critical",
				"status":   "confirmed",
			}
			body, _ := json.Marshal(update)

			url := "/api/v1/asset/vulns/" + strconv.FormatUint(vulnID, 10)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("CreatePoc", func(t *testing.T) {
			poc := map[string]interface{}{
				"vuln_id":     vulnID,
				"poc_type":    "payload",
				"name":        "Test PoC",
				"content":     "test payload",
				"description": "Test Description",
			}
			body, _ := json.Marshal(poc)

			req, _ := http.NewRequest("POST", "/api/v1/asset/vulns/pocs", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			if data, ok := resp["data"].(map[string]interface{}); ok {
				pocID = uint64(data["id"].(float64))
			}
		})

		t.Run("ListPocsByVuln", func(t *testing.T) {
			url := "/api/v1/asset/vulns/" + strconv.FormatUint(vulnID, 10) + "/pocs"
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	if pocID > 0 {
		t.Run("GetPoc", func(t *testing.T) {
			url := "/api/v1/asset/vulns/pocs/" + strconv.FormatUint(pocID, 10)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("UpdatePoc", func(t *testing.T) {
			update := map[string]interface{}{
				"name": "Updated Test PoC",
			}
			body, _ := json.Marshal(update)

			url := "/api/v1/asset/vulns/pocs/" + strconv.FormatUint(pocID, 10)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("DeletePoc", func(t *testing.T) {
			url := "/api/v1/asset/vulns/pocs/" + strconv.FormatUint(pocID, 10)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}

	if vulnID > 0 {
		t.Run("DeleteVuln", func(t *testing.T) {
			url := "/api/v1/asset/vulns/" + strconv.FormatUint(vulnID, 10)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestUnifiedAssetRoutes 娴嬭瘯缁熶竴璧勪骇瑙嗗浘鐩稿叧璺敱
func TestUnifiedAssetRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 纭繚琛ㄧ粨鏋勬渶鏂?
	db.Migrator().DropTable(&assetmodel.AssetUnified{})
	db.AutoMigrate(&assetmodel.AssetUnified{})
	defer func() {
		db.Exec("DELETE FROM asset_unified")
	}()

	var unifiedID uint64

	t.Run("CreateUnifiedAsset", func(t *testing.T) {
		unified := map[string]interface{}{
			"project_id": 1,
			"ip":         "10.0.0.1",
			"port":       80,
			"protocol":   "tcp",
			"service":    "http",
			"tech_stack": "{}",
		}
		body, _ := json.Marshal(unified)

		req, _ := http.NewRequest("POST", "/api/v1/asset/unified", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			unifiedID = uint64(data["id"].(float64))
		}
	})

	t.Run("UpsertUnifiedAsset", func(t *testing.T) {
		// Upsert 搴旇鏇存柊宸插瓨鍦ㄧ殑璁板綍锛堝熀浜嶪P鍜孭ort锛屽鏋滀笟鍔￠€昏緫濡傛璁捐锛夋垨鍒涘缓鏂拌褰?
		// 杩欓噷鍋囪 Upsert 閫昏緫鏄熀浜?IP+Port 鐨?
		unified := map[string]interface{}{
			"project_id": 1,
			"ip":         "10.0.0.1",
			"port":       80,
			"service":    "http-alt",
			"tech_stack": "{}",
		}
		body, _ := json.Marshal(unified)

		req, _ := http.NewRequest("POST", "/api/v1/asset/unified/upsert", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ListUnifiedAssets", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/unified", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	if unifiedID > 0 {
		t.Run("GetUnifiedAsset", func(t *testing.T) {
			url := "/api/v1/asset/unified/" + strconv.FormatUint(unifiedID, 10)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("UpdateUnifiedAsset", func(t *testing.T) {
			update := map[string]interface{}{
				"service": "http-updated",
			}
			body, _ := json.Marshal(update)

			url := "/api/v1/asset/unified/" + strconv.FormatUint(unifiedID, 10)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("DeleteUnifiedAsset", func(t *testing.T) {
			url := "/api/v1/asset/unified/" + strconv.FormatUint(unifiedID, 10)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestScanAssetRoutes 娴嬭瘯璧勪骇鎵弿璁板綍鐩稿叧璺敱
func TestScanAssetRoutes(t *testing.T) {
	engine, db, token, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 纭繚琛ㄧ粨鏋勬渶鏂?
	db.Migrator().DropTable(&assetmodel.AssetNetworkScan{}, &assetmodel.AssetNetwork{})
	db.AutoMigrate(&assetmodel.AssetNetworkScan{}, &assetmodel.AssetNetwork{})
	defer func() {
		db.Exec("DELETE FROM asset_network_scans")
		db.Exec("DELETE FROM asset_networks")
	}()

	// 鍑嗗鍓嶇疆鏁版嵁锛氱綉娈?
	// 准备前置数据：网段
	network := assetmodel.AssetNetwork{
		Network: "192.168.20.0/24",
		CIDR:    "192.168.20.0/24",
		Status:  "active",
		Tags:    "{}",
	}
	if err := db.Create(&network).Error; err != nil {
		t.Fatalf("Failed to create prerequisite network: %v", err)
	}

	var scanID uint64

	t.Run("CreateScan", func(t *testing.T) {
		scan := map[string]interface{}{
			"network_id":  network.ID,
			"agent_id":    1,
			"scan_tool":   "nmap",
			"scan_status": "pending",
			"scan_config": "{}",
		}
		body, _ := json.Marshal(scan)

		req, _ := http.NewRequest("POST", "/api/v1/asset/scans", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			scanID = uint64(data["id"].(float64))
		}
	})

	t.Run("ListScans", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/asset/scans", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	if scanID > 0 {
		t.Run("GetScan", func(t *testing.T) {
			url := "/api/v1/asset/scans/" + strconv.FormatUint(scanID, 10)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("UpdateScan", func(t *testing.T) {
			update := map[string]interface{}{
				"scan_status": "running",
			}
			body, _ := json.Marshal(update)

			url := "/api/v1/asset/scans/" + strconv.FormatUint(scanID, 10)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("GetLatestScanByNetworkID", func(t *testing.T) {
			url := "/api/v1/asset/scans/latest/" + strconv.FormatUint(network.ID, 10)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("DeleteScan", func(t *testing.T) {
			url := "/api/v1/asset/scans/" + strconv.FormatUint(scanID, 10)
			req, _ := http.NewRequest("DELETE", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
