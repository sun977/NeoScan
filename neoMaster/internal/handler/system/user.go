/**
 * @author: sun977
 * @date: 2025.08.29
 * @description: 用户管理接口(管理员和普通用户都可以使用)
 * @func:
 * 	1.创建用户
 * 	2.获取用户列表
 * 	3.获取单个用户
 * 	4.更新用户
 * 	5.删除用户等
 */
package system

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/auth"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService     *auth.UserService     // 用户服务，用于获取用户信息
	passwordService *auth.PasswordService // 密码服务，用于密码相关操作
}

// NewUserHandler 创建用户管理处理器
func NewUserHandler(userService *auth.UserService, passwordService *auth.PasswordService) *UserHandler {
	return &UserHandler{
		userService:     userService,
		passwordService: passwordService,
	}
}

// CreateUser 创建用户(用户注册里面有调用useService.CreateUser)
func (h *UserHandler) CreateUser(c *gin.Context) {
	// TODO: 实现创建用户逻辑
	c.JSON(http.StatusNotImplemented, model.APIResponse{
		Code:    http.StatusNotImplemented,
		Status:  "error",
		Message: "not implemented",
	})
}

// GetUsers 获取用户列表
func (h *UserHandler) GetUsers(c *gin.Context) {
	// TODO: 实现获取用户列表逻辑
	c.JSON(http.StatusNotImplemented, model.APIResponse{
		Code:    http.StatusNotImplemented,
		Status:  "error",
		Message: "not implemented",
	})
}

// GetUser 获取单个用户信息（当前用户信息）【完成】
func (h *UserHandler) GetUser(c *gin.Context) {
	// 从请求头获取Authorization令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		// 记录缺少授权头错误日志
		logger.LogError(errors.New("authorization header required"), "", 0, "", "get_user", "GET", map[string]interface{}{
			"operation":  "get_user",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "authorization header required",
		})
		return
	}

	// 提取Bearer令牌
	if !strings.HasPrefix(authHeader, "Bearer ") {
		// 记录授权头格式错误日志
		logger.LogError(errors.New("invalid authorization header format"), "", 0, "", "get_user", "GET", map[string]interface{}{
			"operation":          "get_user",
			"client_ip":          c.ClientIP(),
			"user_agent":         c.GetHeader("User-Agent"),
			"request_id":         c.GetHeader("X-Request-ID"),
			"auth_header_prefix": authHeader[:min(len(authHeader), 10)],
			"timestamp":          logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "invalid authorization header format",
		})
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	if accessToken == "" {
		// 记录访问令牌为空错误日志
		logger.LogError(errors.New("access token required"), "", 0, "", "get_user", "GET", map[string]interface{}{
			"operation":  "get_user",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "access token required",
		})
		return
	}

	// 获取当前用户信息
	userInfo, err := h.userService.GetCurrentUser(c.Request.Context(), accessToken)
	if err != nil {
		// 记录获取用户信息失败错误日志
		logger.LogError(err, "", 0, "", "get_user", "GET", map[string]interface{}{
			"operation":  "get_user",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"has_token":  accessToken != "",
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "failed to get user info",
			Error:   err.Error(),
		})
		return
	}

	// 记录获取用户信息成功业务日志
	logger.LogBusinessOperation("get_user", uint(userInfo.ID), userInfo.Username, "", "", "success", "获取用户信息成功", map[string]interface{}{
		"operation":  "get_user",
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回用户信息
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "user info retrieved successfully",
		Data:    userInfo,
	})
}

// GetUserPermission 获取用户权限
func (h *UserHandler) GetUserPermission(c *gin.Context) {
	// TODO: 实现获取用户权限逻辑
	c.JSON(http.StatusNotImplemented, model.APIResponse{
		Code:    http.StatusNotImplemented,
		Status:  "error",
		Message: "not implemented",
	})
}

// GetUserRoles 获取用户角色
func (h *UserHandler) GetUserRoles(c *gin.Context) {
	// TODO: 实现获取用户角色逻辑
	c.JSON(http.StatusNotImplemented, model.APIResponse{
		Code:    http.StatusNotImplemented,
		Status:  "error",
		Message: "not implemented",
	})
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// TODO: 实现更新用户逻辑
	c.JSON(http.StatusNotImplemented, model.APIResponse{
		Code:    http.StatusNotImplemented,
		Status:  "error",
		Message: "not implemented",
	})
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// TODO: 实现删除用户逻辑
	c.JSON(http.StatusNotImplemented, model.APIResponse{
		Code:    http.StatusNotImplemented,
		Status:  "error",
		Message: "not implemented",
	})
}

// ChangePassword 修改用户密码
func (h *UserHandler) ChangePassword(c *gin.Context) {
	// 从JWT令牌中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user ID not found in context"), "", 0, "", "change_password", "PUT", map[string]interface{}{
			"operation": "change_password",
			"error":     "user_id_missing",
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "用户身份验证失败",
		})
		return
	}

	// 类型断言获取用户ID
	userIDUint, ok := userID.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user ID type"), "", 0, "", "change_password", "PUT", map[string]interface{}{
			"operation": "change_password",
			"error":     "invalid_user_id_type",
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "服务器内部错误",
		})
		return
	}

	// 解析请求体
	var req model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogError(err, "", userIDUint, "", "change_password", "PUT", map[string]interface{}{
			"operation": "change_password",
			"error":     "invalid_request_body",
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数格式错误",
		})
		return
	}

	// 参数验证
	if strings.TrimSpace(req.OldPassword) == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "原密码不能为空",
		})
		return
	}

	if strings.TrimSpace(req.NewPassword) == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "新密码不能为空",
		})
		return
	}

	// 调用密码服务修改密码
	err := h.passwordService.ChangePassword(c.Request.Context(), userIDUint, req.OldPassword, req.NewPassword)
	if err != nil {
		// 根据错误类型返回不同的HTTP状态码
		var statusCode int
		var message string

		// 判断错误类型
		errorMsg := err.Error()
		switch {
		case strings.Contains(errorMsg, "原密码错误"):
			statusCode = http.StatusBadRequest
			message = "原密码错误"
		case strings.Contains(errorMsg, "用户不存在"):
			statusCode = http.StatusNotFound
			message = "用户不存在"
		case strings.Contains(errorMsg, "新密码长度至少为8位"):
			statusCode = http.StatusBadRequest
			message = "新密码长度至少为8位"
		default:
			statusCode = http.StatusInternalServerError
			message = "密码修改失败，请稍后重试"
		}

		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Status:  "error",
			Message: message,
		})
		return
	}

	// 记录成功操作日志
	logger.LogBusinessOperation("change_password", userIDUint, "", "", "", "success", "用户密码修改成功", map[string]interface{}{
		"operation": "change_password",
		"timestamp": logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "密码修改成功，请重新登录",
	})
}
