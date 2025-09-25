// RoleHandler测试文件
// 测试了角色处理器功能，包括创建角色、获取角色列表、根据ID获取角色等
// 测试命令：go test -v -run TestRoleHandler ./test

// Package test 角色处理器测试
// 测试角色相关的API接口功能
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

// TestRoleHandler covers admin role endpoints
func TestRoleHandler(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.DB == nil || ts.SessionService == nil {
			t.Skip("跳过角色Handler测试：数据库或会话服务不可用")
			return
		}

		gin.SetMode(gin.TestMode)

		// Build services and handler
		roleRepo := mysqlrepo.NewRoleRepository(ts.DB)
		roleService := authsvc.NewRoleService(roleRepo)
		roleHandler := system.NewRoleHandler(roleService)

		// Prepare admin user and token
		adminUser := ts.CreateTestUser(t, "roleadmin", "roleadmin@test.com", "password123")
		adminRole := ts.CreateTestRole(t, "admin", "管理员角色")
		ts.AssignRoleToUser(t, adminUser.ID, adminRole.ID)

		loginResp, err := ts.SessionService.Login(context.Background(), &model.LoginRequest{Username: "roleadmin", Password: "password123"}, "127.0.0.1", "test-agent")
		AssertNoError(t, err, "管理员登录不应该出错")

		// Router with admin group
		r := gin.New()
		admin := r.Group("/api/v1/admin")
		admin.Use(func(c *gin.Context) {
			c.Set("user_id", adminUser.ID)
			c.Next()
		})
		{
			admin.POST("/roles/create", roleHandler.CreateRole)
			admin.GET("/roles/list", roleHandler.GetRoleList)
			admin.GET("/roles/:id", roleHandler.GetRoleByID)
		}

		// Create role
		createBody, _ := json.Marshal(map[string]any{
			"name":         "auditor",
			"display_name": "审计员",
			"description":  "负责审计相关操作",
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/roles/create", bytes.NewBuffer(createBody))
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		AssertEqual(t, http.StatusCreated, w.Code, "创建角色应该返回201")

		// Parse created role id
		var createResp model.APIResponse
		AssertNoError(t, json.Unmarshal(w.Body.Bytes(), &createResp), "解析创建响应不应出错")
		roleJSON, _ := json.Marshal(createResp.Data)
		var createdRole model.Role
		_ = json.Unmarshal(roleJSON, &createdRole)
		AssertTrue(t, createdRole.ID > 0, "创建的角色ID应大于0")

		// List roles
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, httptest.NewRequest("GET", "/api/v1/admin/roles/list?page=1&limit=10", nil))
		AssertEqual(t, http.StatusOK, w2.Code, "获取角色列表应该成功")

		// Get role by id
		w3 := httptest.NewRecorder()
		r.ServeHTTP(w3, httptest.NewRequest("GET", "/api/v1/admin/roles/"+strconv.FormatUint(uint64(createdRole.ID), 10), nil))
		AssertEqual(t, http.StatusOK, w3.Code, "根据ID获取角色应该成功")
	})
}
