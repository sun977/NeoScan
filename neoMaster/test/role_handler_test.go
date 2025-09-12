package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	system "neomaster/internal/handler/system"
	"neomaster/internal/model"
	mysqlrepo "neomaster/internal/repository/mysql"
	authsvc "neomaster/internal/service/auth"
)

// TestRoleHandler covers admin role create endpoint
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
	})
}
