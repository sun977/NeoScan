package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	system "neomaster/internal/handler/system"
	"neomaster/internal/model"
	mysqlrepo "neomaster/internal/repository/mysql"
	authsvc "neomaster/internal/service/auth"
)

// TestPermissionHandler covers admin permission CRUD endpoints
func TestPermissionHandler(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.DB == nil || ts.SessionService == nil {
			t.Skip("跳过权限Handler测试：数据库或会话服务不可用")
			return
		}

		gin.SetMode(gin.TestMode)

		// Build services and handler
		permRepo := mysqlrepo.NewPermissionRepository(ts.DB)
		permService := authsvc.NewPermissionService(permRepo)
		permHandler := system.NewPermissionHandler(permService)

		// Prepare admin user and token
		adminUser := ts.CreateTestUser(t, "permadmin", "permadmin@test.com", "password123")
		adminRole := ts.CreateTestRole(t, "admin", "管理员角色")
		ts.AssignRoleToUser(t, adminUser.ID, adminRole.ID)

		loginResp, err := ts.SessionService.Login(context.Background(), &model.LoginRequest{Username: "permadmin", Password: "password123"}, "127.0.0.1", "test-agent")
		AssertNoError(t, err, "管理员登录不应该出错")

		// Router with admin group
		r := gin.New()
		admin := r.Group("/api/v1/admin")
		admin.Use(func(c *gin.Context) {
			c.Set("user_id", adminUser.ID)
			c.Next()
		})
		{
			admin.POST("/permissions/create", permHandler.CreatePermission)
			admin.GET("/permissions/list", permHandler.GetPermissionList)
			admin.GET("/permissions/:id", permHandler.GetPermissionByID)
			admin.POST("/permissions/:id", permHandler.UpdatePermission)
			admin.DELETE("/permissions/:id", permHandler.DeletePermission)
		}

		// Create permission
		createBody, _ := json.Marshal(map[string]any{
			"name":         "perm.view.logs",
			"display_name": "查看日志",
			"description":  "允许查看系统日志",
			"resource":     "logs",
			"action":       "view",
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/permissions/create", bytes.NewBuffer(createBody))
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		AssertEqual(t, http.StatusCreated, w.Code, "创建权限应该返回201")

		var createResp model.APIResponse
		AssertNoError(t, json.Unmarshal(w.Body.Bytes(), &createResp), "解析创建响应不应该出错")
		AssertEqual(t, "success", createResp.Status, "创建权限响应状态应为success")

		// List permissions
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, httptest.NewRequest("GET", "/api/v1/admin/permissions/list?page=1&limit=10", nil))
		AssertEqual(t, http.StatusOK, w2.Code, "获取权限列表应该成功")

		// Read created permission id from first response
		permJSON, _ := json.Marshal(createResp.Data)
		var created model.Permission
		_ = json.Unmarshal(permJSON, &created)
		AssertTrue(t, created.ID > 0, "创建的权限ID应大于0")

		// Get by id
		w3 := httptest.NewRecorder()
		r.ServeHTTP(w3, httptest.NewRequest("GET", "/api/v1/admin/permissions/"+itoa(created.ID), nil))
		AssertEqual(t, http.StatusOK, w3.Code, "根据ID获取权限应该成功")
	})
}

// helper to convert uint to string without importing strconv in many places
func itoa(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
