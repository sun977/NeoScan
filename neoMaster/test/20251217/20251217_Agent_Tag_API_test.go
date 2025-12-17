package test_20251217

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	agentHandler "neomaster/internal/handler/agent"
	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/system"
	"neomaster/internal/model/tag_system"
	agentRepo "neomaster/internal/repo/mysql/agent"
	tagRepo "neomaster/internal/repo/mysql/tag_system"
	agentService "neomaster/internal/service/agent"
	tagService "neomaster/internal/service/tag_system"

	"github.com/gin-gonic/gin"
)

// TestAgentTagAPI 测试 Agent 标签管理的 HTTP 接口
// 模拟前端请求 -> Router -> Handler -> Service -> DB
func TestAgentTagAPI(t *testing.T) {
	// 1. 初始化依赖
	db := setupTestDB(t) // 复用同一包下的 setupTestDB
	ctx := context.Background()

	// Repo
	tagRepoInst := tagRepo.NewTagRepository(db)
	agentRepoInst := agentRepo.NewAgentRepository(db)

	// Service
	tagSvc := tagService.NewTagService(tagRepoInst, db)
	agentSvc := agentService.NewAgentManagerService(agentRepoInst, tagSvc)

	// Handler
	// 注意：NewAgentHandler 需要 Monitor 和 Config 服务，这里传 nil，因为只测试 Tag 功能
	h := agentHandler.NewAgentHandler(agentSvc, nil, nil)

	// 2. 设置 Gin 路由
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 注册路由 (参考 internal/app/master/router/agent_routers.go)
	agentGroup := r.Group("/agent")
	{
		agentGroup.GET("/:id/tags", h.GetAgentTags)
		agentGroup.POST("/:id/tags", h.AddAgentTag)
		agentGroup.PUT("/:id/tags", h.UpdateAgentTags)
		agentGroup.DELETE("/:id/tags", h.RemoveAgentTag)
	}

	// 3. 准备测试数据
	// 清理旧数据
	db.Exec("DELETE FROM sys_tags WHERE name LIKE 'APITestTag%'")
	db.Exec("DELETE FROM agents WHERE agent_id = 'api_test_agent'")
	db.Exec("DELETE FROM sys_entity_tags WHERE entity_type = 'agent' AND entity_id = 'api_test_agent'")

	// 创建 Tag
	tag1 := &tag_system.SysTag{Name: "APITestTag_1", Description: "API Test 1"}
	if err := tagSvc.CreateTag(ctx, tag1); err != nil {
		t.Fatalf("Failed to create tag 1: %v", err)
	}
	tag2 := &tag_system.SysTag{Name: "APITestTag_2", Description: "API Test 2"}
	if err := tagSvc.CreateTag(ctx, tag2); err != nil {
		t.Fatalf("Failed to create tag 2: %v", err)
	}
	// 创建 Child Tag
	tag3 := &tag_system.SysTag{Name: "APITestTag_3", ParentID: tag1.ID}
	if err := tagSvc.CreateTag(ctx, tag3); err != nil {
		t.Fatalf("Failed to create tag 3: %v", err)
	}

	// 创建 Agent
	agent := &agentModel.Agent{
		AgentID:       "api_test_agent",
		Hostname:      "api-test-host",
		IPAddress:     "192.168.1.100",
		Status:        agentModel.AgentStatusOnline,
		TokenExpiry:   time.Now().Add(24 * time.Hour),
		LastHeartbeat: time.Now(),
	}
	if err := db.Create(agent).Error; err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	t.Logf("Setup Complete. AgentID: %s, Tag1ID: %d, Tag2ID: %d, Tag3ID: %d", agent.AgentID, tag1.ID, tag2.ID, tag3.ID)

	// === 测试 Case 1: AddAgentTag (POST) ===
	t.Run("AddAgentTag", func(t *testing.T) {
		payload := map[string]uint64{"tag_id": tag3.ID}
		jsonBody, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/agent/"+agent.AgentID+"/tags", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		t.Logf("Response: %d %s", w.Code, w.Body.String())
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}

		// 验证 DB
		tags, _ := agentSvc.GetAgentTags(agent.AgentID)
		if len(tags) != 1 || tags[0].Name != tag3.Name {
			t.Errorf("DB check failed. Expected [%s], got %v", tag3.Name, tags)
		}
	})

	// === 测试 Case 2: UpdateAgentTags (PUT) ===
	t.Run("UpdateAgentTags", func(t *testing.T) {
		payload := map[string][]uint64{"tag_ids": {tag3.ID}}
		jsonBody, _ := json.Marshal(payload)
		req, _ := http.NewRequest("PUT", "/agent/"+agent.AgentID+"/tags", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		t.Logf("Response: %d %s", w.Code, w.Body.String())
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}

		// 验证 DB
		tags, _ := agentSvc.GetAgentTags(agent.AgentID)
		if len(tags) != 1 || tags[0].Name != tag3.Name {
			t.Errorf("DB check failed. Expected [%s], got %v", tag3.Name, tags)
		}
	})

	// === 测试 Case 3: GetAgentTags (GET) - Should have 1 tag now ===
	t.Run("GetAgentTags", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/agent/"+agent.AgentID+"/tags", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		t.Logf("Response: %d %s", w.Code, w.Body.String())
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}

		// 解析响应，验证是否包含 ID
		var resp system.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		dataMap, ok := resp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Data is not a map")
		}

		tagsInterface, ok := dataMap["tags"]
		if !ok {
			t.Fatalf("Tags field missing in data")
		}

		tagsList, ok := tagsInterface.([]interface{})
		if !ok {
			t.Fatalf("Tags is not a list")
		}

		// 检查第一个 tag 是否包含 id 和 name
		if len(tagsList) > 0 {
			tagMap, ok := tagsList[0].(map[string]interface{})
			if !ok {
				t.Fatalf("Tag item is not a map")
			}
			if _, ok := tagMap["id"]; !ok {
				t.Errorf("Tag ID missing in response")
			}
			if _, ok := tagMap["name"]; !ok {
				t.Errorf("Tag Name missing in response")
			}
			// 验证 FullPathName
			if fullPath, ok := tagMap["full_path_name"].(string); !ok {
				t.Errorf("Tag FullPathName missing in response")
			} else {
				expectedPath := "APITestTag_1/APITestTag_3"
				if fullPath != expectedPath {
					t.Errorf("Expected FullPathName %s, got %s", expectedPath, fullPath)
				}
			}
		} else {
			t.Errorf("Expected at least 1 tag, got 0")
		}
	})

	// === 测试 Case 4: RemoveAgentTag (DELETE) ===
	t.Run("RemoveAgentTag", func(t *testing.T) {
		payload := map[string]uint64{"tag_id": tag3.ID}
		jsonBody, _ := json.Marshal(payload)
		req, _ := http.NewRequest("DELETE", "/agent/"+agent.AgentID+"/tags", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		t.Logf("Response: %d %s", w.Code, w.Body.String())
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}

		// 验证 DB
		tags, _ := agentSvc.GetAgentTags(agent.AgentID)
		if len(tags) != 0 {
			t.Errorf("DB check failed. Expected empty, got %v", tags)
		}
	})

	// === 测试 Case 4: RemoveAgentTag (DELETE) ===
	t.Run("RemoveAgentTag", func(t *testing.T) {
		payload := map[string]uint64{"tag_id": tag2.ID}
		jsonBody, _ := json.Marshal(payload)
		req, _ := http.NewRequest("DELETE", "/agent/"+agent.AgentID+"/tags", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		t.Logf("Response: %d %s", w.Code, w.Body.String())
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}
	})

	// === 测试 Case 5: Add Non-Existent Tag (Not Found) ===
	t.Run("AddNonExistentTag", func(t *testing.T) {
		payload := map[string]uint64{"tag_id": 999999}
		jsonBody, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/agent/"+agent.AgentID+"/tags", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		t.Logf("Response: %d %s", w.Code, w.Body.String())
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found, got %d", w.Code)
		}
	})

	// === 测试 Case 6: Invalid Input (Bad Request) ===
	t.Run("InvalidInput", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/agent/"+agent.AgentID+"/tags", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", w.Code)
		}
	})
}
