// PermissionHandler测试文件
// 测试了权限处理器功能，包括创建权限、获取权限列表、根据ID获取权限等
// 测试命令：go test -v -run TestPermissionHandler ./test

// Package test 权限处理器测试
// 测试权限相关的API接口功能
package test

import (
	"bytes"
	"context"
	"encoding/json"
	system2 "neomaster/internal/model/system"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	system "neomaster/internal/handler/system"
	mysqlrepo "neomaster/internal/repository/mysql"
	authsvc "neomaster/internal/service/auth"
)

// TestPermissionHandler covers admin permission endpoints
func TestPermissionHandler(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.DB == nil || ts.SessionService == nil {
			t.Skip("跳过权限Handler测试：数据库或会话服务不可用")
			return
		}

		gin.SetMode(gin.TestMode)

		// Build services and handler
		permissionRepo := mysqlrepo.NewPermissionRepository(ts.DB)
		permissionService := authsvc.NewPermissionService(permissionRepo)
		permissionHandler := system.NewPermissionHandler(permissionService)

		// Prepare admin user and token
		adminUser := ts.CreateTestUser(t, "permissionadmin", "permissionadmin@test.com", "password123")
		adminRole := ts.CreateTestRole(t, "admin", "管理员角色")
		ts.AssignRoleToUser(t, adminUser.ID, adminRole.ID)

		loginResp, err := ts.SessionService.Login(context.Background(), &system2.LoginRequest{Username: "permissionadmin", Password: "password123"}, "127.0.0.1", "test-agent")
		AssertNoError(t, err, "管理员登录不应该出错")

		// Router with admin group
		r := gin.New()
		admin := r.Group("/api/v1/admin")
		admin.Use(func(c *gin.Context) {
			c.Set("user_id", adminUser.ID)
			c.Next()
		})
		{
			admin.POST("/permissions/create", permissionHandler.CreatePermission)
			admin.GET("/permissions/list", permissionHandler.GetPermissionList)
			admin.GET("/permissions/:id", permissionHandler.GetPermissionByID)
		}

		// Create permission
		createBody, _ := json.Marshal(map[string]any{
			"name":         "view_audit_logs",
			"display_name": "查看审计日志",
			"description":  "允许查看系统审计日志",
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/permissions/create", bytes.NewBuffer(createBody))
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		AssertEqual(t, http.StatusCreated, w.Code, "创建权限应该返回201")

		// Parse created permission id
		var createResp system2.APIResponse
		AssertNoError(t, json.Unmarshal(w.Body.Bytes(), &createResp), "解析创建响应不应出错")
		permissionJSON, _ := json.Marshal(createResp.Data)
		var createdPermission system2.Permission
		_ = json.Unmarshal(permissionJSON, &createdPermission)
		AssertTrue(t, createdPermission.ID > 0, "创建的权限ID应大于0")

		// List permissions
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, httptest.NewRequest("GET", "/api/v1/admin/permissions/list?page=1&limit=10", nil))
		AssertEqual(t, http.StatusOK, w2.Code, "获取权限列表应该成功")

		// Get permission by id
		w3 := httptest.NewRecorder()
		r.ServeHTTP(w3, httptest.NewRequest("GET", "/api/v1/admin/permissions/"+strconv.FormatUint(uint64(createdPermission.ID), 10), nil))
		AssertEqual(t, http.StatusOK, w3.Code, "根据ID获取权限应该成功")
	})
}