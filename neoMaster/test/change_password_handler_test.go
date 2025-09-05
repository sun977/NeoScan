package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neomaster/internal/handler/system"
	"neomaster/internal/model"
	"neomaster/internal/service/auth"
)

// TestChangePasswordHandler 测试密码修改处理器
func TestChangePasswordHandler(t *testing.T) {
	// 设置测试环境
	ts := SetupTestEnvironment(t)
	defer ts.TeardownTestEnvironment(t)

	// 创建测试用户请求
	createUserReq := &model.CreateUserRequest{
		Username: "testuser_changepassword",
		Email:    "testuser_changepassword@example.com",
		Password: "oldpassword123",
		Nickname: "Test User",
	}

	// 通过 UserService 创建用户（这样密码会被正确哈希）
	testUser, err := ts.UserService.CreateUser(context.Background(), createUserReq)
	assert.NoError(t, err)
	assert.NotZero(t, testUser.ID)

// 创建PasswordService
		passwordService := auth.NewPasswordService(
			ts.UserService,
			ts.SessionService,
			ts.passwordManager,
			24*time.Hour,
		)
	// 创建处理器
	userHandler := system.NewUserHandler(ts.UserService, passwordService)

	// 设置 Gin 路由
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 添加中间件模拟JWT认证
	router.Use(func(c *gin.Context) {
		c.Set("user_id", testUser.ID)
		c.Next()
	})

	router.PUT("/change-password", userHandler.ChangePassword)

	t.Run("成功修改密码", func(t *testing.T) {
		// 准备请求数据
		req := model.ChangePasswordRequest{
			OldPassword: "oldpassword123",
			NewPassword: "newpassword123",
		}

		// 序列化请求数据
		reqBody, err := json.Marshal(req)
		assert.NoError(t, err)

		// 创建HTTP请求
		httpReq, err := http.NewRequest("PUT", "/change-password", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		httpReq.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, httpReq)

		// 验证响应
		assert.Equal(t, http.StatusOK, w.Code)

		// 解析响应
		var response model.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Contains(t, response.Message, "密码修改成功")

		// 验证密码确实被修改了
		updatedUser, err := ts.UserRepo.GetUserByID(context.Background(), testUser.ID)
		assert.NoError(t, err)
		assert.NotEqual(t, testUser.Password, updatedUser.Password) // 密码哈希应该不同
	})

	t.Run("原密码错误", func(t *testing.T) {
		// 准备请求数据
		req := model.ChangePasswordRequest{
			OldPassword: "wrongpassword",
			NewPassword: "newpassword123",
		}

		// 序列化请求数据
		reqBody, err := json.Marshal(req)
		assert.NoError(t, err)

		// 创建HTTP请求
		httpReq, err := http.NewRequest("PUT", "/change-password", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		httpReq.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, httpReq)

		// 验证响应
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 解析响应
		var response model.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "原密码错误")
	})

	t.Run("新密码长度不足", func(t *testing.T) {
		// 准备请求数据
		req := model.ChangePasswordRequest{
			OldPassword: "newpassword123", // 使用之前修改后的密码
			NewPassword: "123",            // 长度不足
		}

		// 序列化请求数据
		reqBody, err := json.Marshal(req)
		assert.NoError(t, err)

		// 创建HTTP请求
		httpReq, err := http.NewRequest("PUT", "/change-password", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		httpReq.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, httpReq)

		// 验证响应
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 解析响应
		var response model.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "新密码长度至少为8位")
	})

	t.Run("参数为空", func(t *testing.T) {
		// 准备请求数据 - 原密码为空
		req := model.ChangePasswordRequest{
			OldPassword: "",
			NewPassword: "newpassword123",
		}

		// 序列化请求数据
		reqBody, err := json.Marshal(req)
		assert.NoError(t, err)

		// 创建HTTP请求
		httpReq, err := http.NewRequest("PUT", "/change-password", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		httpReq.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, httpReq)

		// 验证响应
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 解析响应
		var response model.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "原密码不能为空")
	})

	t.Run("无用户认证信息", func(t *testing.T) {
		// 创建新的路由，不添加用户ID中间件
		router2 := gin.New()
		router2.PUT("/change-password", userHandler.ChangePassword)

		// 准备请求数据
		req := model.ChangePasswordRequest{
			OldPassword: "newpassword123",
			NewPassword: "anotherpassword123",
		}

		// 序列化请求数据
		reqBody, err := json.Marshal(req)
		assert.NoError(t, err)

		// 创建HTTP请求
		httpReq, err := http.NewRequest("PUT", "/change-password", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		httpReq.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 执行请求
		router2.ServeHTTP(w, httpReq)

		// 验证响应
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// 解析响应
		var response model.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response.Status)
		assert.Contains(t, response.Message, "用户身份验证失败")
	})

	// 清理测试数据
	err = ts.UserRepo.DeleteUser(context.Background(), testUser.ID)
	assert.NoError(t, err)
}