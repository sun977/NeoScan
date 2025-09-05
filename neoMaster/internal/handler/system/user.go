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
 * @note：有两类函数
 * 	1.用户自己查看自己的信息和操作 --- 从token中获取用户自己的用户ID
 * 	2.管理员查看所有用户的信息和操作 --- 从上下文中获取用户ID（中间件已验证并存储），管理员自己的ID记录操作日志
 */
package system

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model"
	pkgAuth "neomaster/internal/pkg/auth"
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

// extractTokenFromContext 从gin.Context中提取访问令牌
// 使用jwt包的ExtractTokenFromHeader函数，统一令牌提取逻辑
func (h *UserHandler) extractTokenFromContext(c *gin.Context) (string, error) {
	// 从请求头获取Authorization令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header required")
	}

	// 使用jwt包的ExtractTokenFromHeader函数提取令牌
	accessToken := pkgAuth.ExtractTokenFromHeader(authHeader)
	if accessToken == "" {
		return "", errors.New("invalid authorization header format or empty token")
	}

	return accessToken, nil
}

// CreateUser 创建用户（管理员专用） 【已完成】
// 创建新用户，包含完整的参数验证和权限检查
func (h *UserHandler) CreateUser(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		// 记录获取用户ID失败错误日志
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "create_user", "POST", map[string]interface{}{
			"operation":  "create_user",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "user context not found",
		})
		return
	}

	// 类型转换用户ID
	userID, ok := userIDInterface.(uint)
	if !ok {
		// 记录用户ID类型转换失败错误日志
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "create_user", "POST", map[string]interface{}{
			"operation":  "create_user",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "invalid user context",
		})
		return
	}

	// 解析请求体
	var req model.CreateUserRequest
	if parseErr := c.ShouldBindJSON(&req); parseErr != nil {
		// 记录请求参数解析失败错误日志
		logger.LogError(parseErr, "", userID, "", "create_user", "POST", map[string]interface{}{
			"operation":  "create_user",
			"user_id":    userID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "invalid request body",
			Error:   parseErr.Error(),
		})
		return
	}

	// 调用服务层创建用户
	createdUser, err := h.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		// 记录创建用户失败错误日志
		logger.LogError(err, "", userID, "", "create_user", "POST", map[string]interface{}{
			"operation":       "create_user",
			"user_id":         userID,
			"target_username": req.Username,
			"target_email":    req.Email,
			"client_ip":       c.ClientIP(),
			"user_agent":      c.GetHeader("User-Agent"),
			"request_id":      c.GetHeader("X-Request-ID"),
			"timestamp":       logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "failed to create user",
			Error:   err.Error(),
		})
		return
	}

	// 记录创建用户成功业务日志
	logger.LogBusinessOperation("create_user", userID, "", "", "", "success", "创建用户成功", map[string]interface{}{
		"operation":        "create_user",
		"operator_id":      userID,
		"created_user_id":  createdUser.ID,
		"created_username": createdUser.Username,
		"created_email":    createdUser.Email,
		"client_ip":        c.ClientIP(),
		"user_agent":       c.GetHeader("User-Agent"),
		"request_id":       c.GetHeader("X-Request-ID"),
		"timestamp":        logger.NowFormatted(),
	})

	// 构造响应数据，不返回敏感信息
	responseData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":         createdUser.ID,
			"username":   createdUser.Username,
			"email":      createdUser.Email,
			"nickname":   createdUser.Nickname,
			"phone":      createdUser.Phone,
			"status":     createdUser.Status,
			"created_at": createdUser.CreatedAt,
		},
	}

	// 返回创建成功响应
	c.JSON(http.StatusCreated, model.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "user created successfully",
		Data:    responseData,
	})
}

// GetUserByID 获取当前用户信息（管理员专用）
func (h *UserHandler) GetUserByID(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		// 记录获取用户ID失败错误日志
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "get_user_by_id", "GET", map[string]interface{}{
			"operation":  "get_user_by_id",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "user context not found",
		})
		return
	}

	// 类型转换用户ID
	userID, ok := userIDInterface.(uint)
	if !ok {
		// 记录用户ID类型转换失败错误日志
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "get_user_by_id", "GET", map[string]interface{}{
			"operation":  "get_user_by_id",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "invalid user context",
		})
		return
	}

	// 调用服务层获取用户详细信息
	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		// 记录获取用户详细信息失败错误日志
		logger.LogError(err, "", userID, "", "get_user_by_id", "GET", map[string]interface{}{
			"operation":  "get_user_by_id",
			"user_id":    userID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "failed to get user details",
			Error:   err.Error(),
		})
		return
	}

	// 记录获取用户信息成功业务日志
	logger.LogBusinessOperation("get_user_by_id", userID, user.Username, "", "", "success", "获取用户信息成功", map[string]interface{}{
		"operation":   "get_user_by_id",
		"user_id":     userID,
		"username":    user.Username,
		"target_id":   user.ID,
		"target_name": user.Username,
		"client_ip":   c.ClientIP(),
		"user_agent":  c.GetHeader("User-Agent"),
		"request_id":  c.GetHeader("X-Request-ID"),
		"timestamp":   logger.NowFormatted(),
	})

	// 构造响应数据，不返回敏感信息
	responseData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"nickname":   user.Nickname,
			"phone":      user.Phone,
			"status":     user.Status,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	}

	// 返回用户信息
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "user information retrieved successfully",
		Data:    responseData,
	})
}

// GetUserList 获取用户列表（用户管理员专用）【待实现，有问题】
// 支持分页查询，返回用户基本信息列表
func (h *UserHandler) GetUserList(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		// 记录获取用户ID失败错误日志
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "get_user_list", "GET", map[string]interface{}{
			"operation":  "get_user_list",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "user context not found",
		})
		return
	}

	// 类型转换用户ID
	userID, ok := userIDInterface.(uint)
	if !ok {
		// 记录用户ID类型转换失败错误日志
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "get_user_list", "GET", map[string]interface{}{
			"operation":  "get_user_list",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "invalid user context",
		})
		return
	}

	// 解析分页参数
	page := 1
	limit := 10

	// 从查询参数获取页码，默认为1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, parseErr := strconv.Atoi(pageStr); parseErr == nil && p > 0 {
			page = p
		}
	}

	// 从查询参数获取每页数量，默认为10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, parseErr := strconv.Atoi(limitStr); parseErr == nil && l > 0 {
			limit = l
		}
	}

	// 计算偏移量
	offset := (page - 1) * limit

	// 调用service层获取用户列表
	users, total, err := h.userService.GetUserList(c.Request.Context(), offset, limit)
	if err != nil {
		// 记录获取用户列表失败错误日志
		logger.LogError(err, "", userID, "", "get_user_list", "GET", map[string]interface{}{
			"operation":  "get_user_list",
			"user_id":    userID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"page":       page,
			"limit":      limit,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "failed to get user list",
			Error:   err.Error(),
		})
		return
	}

	// 转换用户数据为响应格式
	userInfos := make([]model.UserInfo, 0, len(users))
	for _, user := range users {
		userInfos = append(userInfos, model.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			Nickname:    user.Nickname,
			Phone:       user.Phone,
			Status:      user.Status,
			CreatedAt:   user.CreatedAt,
			LastLoginAt: user.LastLoginAt,
		})
	}

	// 计算总页数
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// 记录获取用户列表成功业务日志
	logger.LogBusinessOperation("get_user_list", userID, "", "", "", "success", "获取用户列表成功", map[string]interface{}{
		"operation":   "get_user_list",
		"user_id":     userID,
		"client_ip":   c.ClientIP(),
		"user_agent":  c.GetHeader("User-Agent"),
		"request_id":  c.GetHeader("X-Request-ID"),
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
		"user_count":  len(userInfos),
		"timestamp":   logger.NowFormatted(),
	})

	// 构造响应数据，符合API文档v2.0规范
	responseData := map[string]interface{}{
		"items": userInfos,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": totalPages,
		},
	}

	// 返回用户列表信息
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "user list retrieved successfully",
		Data:    responseData,
	})
}

// GetUserInfo 获取单个用户信息（当前用户信息） （用户专用）【已完成】
// 从accesstoken获取用户ID并获取用户的全量信息(包含权限和角色信息)
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	// 从请求头提取访问令牌
	accessToken, err := h.extractTokenFromContext(c)
	if err != nil {
		// 记录令牌提取失败错误日志
		logger.LogError(err, "", 0, "", "get_user", "GET", map[string]interface{}{
			"operation":  "get_user",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "failed to extract token",
			Error:   err.Error(),
		})
		return
	}

	// 获取当前用户信息
	userInfo, err := h.userService.GetCurrentUserInfo(c.Request.Context(), accessToken)
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

// GetUserPermission 获取用户权限（用户专用）【已完成】
func (h *UserHandler) GetUserPermission(c *gin.Context) {
	// 从请求头提取访问令牌
	accessToken, err := h.extractTokenFromContext(c)
	if err != nil {
		// 记录令牌提取失败错误日志
		logger.LogError(err, "", 0, "", "get_user_permissions", "GET", map[string]interface{}{
			"operation":  "get_user_permissions",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "failed to extract token",
			Error:   err.Error(),
		})
		return
	}

	// 获取当前用户ID以获取用户权限
	userID, err := h.userService.GetUserIDFromToken(c.Request.Context(), accessToken)
	if err != nil {
		// 记录获取用户ID失败错误日志
		logger.LogError(err, "", 0, "", "get_user_permissions", "GET", map[string]interface{}{
			"operation":  "get_user_permissions",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
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

	// 获取用户权限
	permissions, err := h.userService.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		// 记录获取用户权限失败错误日志
		logger.LogError(err, "", userID, "", "get_user_permissions", "GET", map[string]interface{}{
			"operation":  "get_user_permissions",
			"user_id":    userID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "failed to get user permissions",
			Error:   err.Error(),
		})
		return
	}

	// 记录获取用户权限成功业务日志
	logger.LogBusinessOperation("get_user_permissions", userID, "", "", "", "success", "获取用户权限成功", map[string]interface{}{
		"operation":        "get_user_permissions",
		"user_id":          userID,
		"permission_count": len(permissions),
		"client_ip":        c.ClientIP(),
		"user_agent":       c.GetHeader("User-Agent"),
		"request_id":       c.GetHeader("X-Request-ID"),
		"timestamp":        logger.NowFormatted(),
	})

	// 构造响应数据，符合API文档规范
	responseData := map[string]interface{}{
		"permissions": permissions,
	}

	// 返回用户权限信息
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "user permissions retrieved successfully",
		Data:    responseData,
	})
}

// GetUserRoles 获取用户角色（用户专用）【已完成】
func (h *UserHandler) GetUserRoles(c *gin.Context) {
	// 从请求头提取访问令牌
	accessToken, err := h.extractTokenFromContext(c)
	if err != nil {
		// 记录令牌提取失败错误日志
		logger.LogError(err, "", 0, "", "get_user_roles", "GET", map[string]interface{}{
			"operation":  "get_user_roles",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "failed to extract token",
			Error:   err.Error(),
		})
		return
	}

	// 获取当前用户ID以获取用户角色
	userID, err := h.userService.GetUserIDFromToken(c.Request.Context(), accessToken)
	if err != nil {
		// 记录获取用户ID失败错误日志
		logger.LogError(err, "", 0, "", "get_user_roles", "GET", map[string]interface{}{
			"operation":  "get_user_roles",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
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

	// 获取用户角色
	roles, err := h.userService.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		// 记录获取用户角色失败错误日志
		logger.LogError(err, "", userID, "", "get_user_roles", "GET", map[string]interface{}{
			"operation":  "get_user_roles",
			"user_id":    userID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "failed to get user roles",
			Error:   err.Error(),
		})
		return
	}

	// 记录获取用户角色成功业务日志
	logger.LogBusinessOperation("get_user_roles", userID, "", "", "", "success", "获取用户角色成功", map[string]interface{}{
		"operation":  "get_user_roles",
		"user_id":    userID,
		"role_count": len(roles),
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 构造响应数据，符合API文档规范
	responseData := map[string]interface{}{
		"roles": roles,
	}

	// 返回用户角色信息
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "user roles retrieved successfully",
		Data:    responseData,
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

// ChangePassword 修改用户密码 (用户专用) 【已完成】
func (h *UserHandler) ChangePassword(c *gin.Context) {
	// 从中间件上下文获取用户ID
	// 本身是自己修改自己的密码，所以令牌和中间件封装的上下文中用户ID一致，所以可以这样写，节省了解析令牌时间（中间件已经解析过令牌）
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
