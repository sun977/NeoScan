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
 * @note：中间件 r.middlewareManager.GinJWTAuthMiddleware() 已经解析了token，并获取了用户信息
 * 解析出来的用户信息保存在 gin.Context 上下文中，可以通过 c.Get("user") 获取
 * 获取的字段有：user_id，username，roles，permissions，claims
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
// 使用jwt包的ExtractTokenFromHeader函数，统一令牌提取逻辑【不需要了,直接获取绕过了令牌中间件,使用gin.Context上下文获取解析后的user_id】
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

// parsePaginationParams 解析分页参数 - 提取公共逻辑，消除重复代码
func parsePaginationParams(c *gin.Context) (page, limit int) {
	page, limit = 1, 10 // 默认值

	// 解析页码参数
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// 解析限制参数，添加合理上限防止滥用
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	return page, limit
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
			"remark":     createdUser.Remark,
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

// GetUserByID 获取用户信息
func (h *UserHandler) GetUserByID(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	// userIDInterface, exists := c.Get("user_id")
	// 从url中获取用户ID
	userIDStr := c.Param("id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "user id is required",
		})
		return
	}

	// 转换用户ID为uint类型
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		// 记录用户ID格式错误日志
		logger.LogError(err, "", 0, "", "get_user_by_id", "GET", map[string]interface{}{
			"user_id_str": userIDStr,
			"error":       "invalid_user_id_format",
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "invalid user id format",
		})
		return
	}

	// 调用服务层获取用户信息
	user, err := h.userService.GetUserByID(c.Request.Context(), uint(userID))
	if err != nil {
		// 记录获取用户详细信息失败错误日志
		logger.LogError(err, "", uint(userID), "", "get_user_by_id", "GET", map[string]interface{}{
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
	logger.LogBusinessOperation("get_user_by_id", uint(userID), user.Username, "", "", "success", "获取用户信息成功", map[string]interface{}{
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

// GetUserList 获取用户列表（用户管理员专用）
// 支持分页查询，返回用户基本信息列表
// GetUserList 获取用户列表 - 重构版本，遵循"好品味"原则
func (h *UserHandler) GetUserList(c *gin.Context) {
	// 解析分页参数，使用简单的默认值处理
	page, limit := parsePaginationParams(c)
	offset := (page - 1) * limit

	// 调用service层获取用户列表 - 核心业务逻辑
	users, total, err := h.userService.GetUserList(c.Request.Context(), offset, limit)
	if err != nil {
		// 简化错误处理 - 只记录必要信息
		logger.LogError(err, "", 0, "", "get_user_list", "GET", map[string]interface{}{
			"page":  page,
			"limit": limit,
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "failed to get user list",
		})
		return
	}

	// 直接构造响应 - 消除不必要的数据转换
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "user list retrieved successfully",
		Data: map[string]interface{}{
			"items": users, // 直接返回service层数据，避免重复转换
			"pagination": map[string]interface{}{
				"page":  page,
				"limit": limit,
				"total": total,
				"pages": (total + int64(limit) - 1) / int64(limit), // 内联计算，避免额外变量
			},
		},
	})
}

// GetUserInfoByID 获取单个用户信息【完成】
func (h *UserHandler) GetUserInfoByID(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	// 从url中获取用户ID
	userIDStr := c.Param("id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "user id is required",
		})
		return
	}

	// 转换用户ID为uint类型
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		// 记录用户ID格式错误日志
		logger.LogError(err, "", 0, "", "get_user_info_by_id", "GET", map[string]interface{}{
			"user_id_str": userIDStr,
			"error":       "invalid_user_id_format",
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "invalid user id format",
		})
		return
	}

	// 调用服务层获取用户信息
	userInfo, err := h.userService.GetUserInfoByID(c.Request.Context(), uint(userID))
	if err != nil {
		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "用户不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: "用户不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "获取用户信息失败",
				Error:   err.Error(),
			})
		}
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取用户信息成功",
		Data:    userInfo,
	})
}

// GetUserInfo 获取单个用户信息（当前用户信息）
// 从accesstoken获取用户ID并获取用户的全量信息(包含权限和角色信息)
func (h *UserHandler) GetUserInfoByAccessToken(c *gin.Context) {
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

// GetUserPermission 获取用户权限
func (h *UserHandler) GetUserPermission(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		// 记录用户ID不存在错误日志
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "get_user_permissions", "GET", map[string]interface{}{
			"operation":  "get_user_permissions",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "user_id not found in context",
		})
		return
	}

	// 类型断言获取用户ID
	userID, ok := userIDInterface.(uint)
	if !ok {
		// 记录用户ID类型转换失败错误日志
		logger.LogError(errors.New("user_id type assertion failed"), "", 0, "", "get_user_permissions", "GET", map[string]interface{}{
			"operation":  "get_user_permissions",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "invalid user_id type",
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

// GetUserRoles 获取用户角色
func (h *UserHandler) GetUserRoles(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		// 记录用户ID不存在错误日志
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "get_user_roles", "GET", map[string]interface{}{
			"operation":  "get_user_roles",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "user_id not found in context",
		})
		return
	}

	// 类型断言获取用户ID
	userID, ok := userIDInterface.(uint)
	if !ok {
		// 记录用户ID类型转换失败错误日志
		logger.LogError(errors.New("user_id type assertion failed"), "", 0, "", "get_user_roles", "GET", map[string]interface{}{
			"operation":  "get_user_roles",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "invalid user_id type",
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

// UpdateUserByID 更新用户信息 - 遵循Handler->Service层级调用原则
func (h *UserHandler) UpdateUserByID(c *gin.Context) {
	// 从URL路径参数获取用户ID
	userIDStr := c.Param("id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "user id is required",
		})
		return
	}

	// 转换用户ID为uint类型
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		// 记录用户ID格式错误日志
		logger.LogError(err, "", 0, "", "update_user_by_id", "POST", map[string]interface{}{
			"user_id_str": userIDStr,
			"error":       "invalid_user_id_format",
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "invalid user id format",
		})
		return
	}

	// 解析请求体中的更新数据
	var req model.UpdateUserRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		// 记录请求参数解析失败日志
		logger.LogError(bindErr, "", uint(userID), "", "update_user_by_id", "PUT", map[string]interface{}{
			"user_id": userID,
			"error":   "request_parse_failed",
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "invalid request format",
		})
		return
	}

	// 调用service层更新用户信息 - 核心业务逻辑
	updatedUser, err := h.userService.UpdateUserByID(c.Request.Context(), uint(userID), &req)
	if err != nil {
		// 根据错误类型返回不同的HTTP状态码
		if err.Error() == "用户不存在" {
			// 用户不存在，返回404
			logger.LogError(err, "", uint(userID), "", "update_user_by_id", "POST", map[string]interface{}{
				"user_id": userID,
				"error":   "user_not_found",
			})
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: "user not found",
			})
			return
		}
		if err.Error() == "邮箱已存在" {
			// 邮箱冲突，返回409
			logger.LogError(err, "", uint(userID), "", "update_user_by_id", "POST", map[string]interface{}{
				"user_id": userID,
				"error":   "email_conflict",
			})
			c.JSON(http.StatusConflict, model.APIResponse{
				Code:    http.StatusConflict,
				Status:  "error",
				Message: "email already exists",
			})
			return
		}
		// 其他错误，返回500
		logger.LogError(err, "", uint(userID), "", "update_user_by_id", "POST", map[string]interface{}{
			"user_id": userID,
			"error":   "update_failed",
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "failed to update user",
		})
		return
	}

	// 构造响应数据，隐藏敏感信息
	userInfo := model.UserInfo{
		ID:          updatedUser.ID,
		Username:    updatedUser.Username,
		Email:       updatedUser.Email,
		Nickname:    updatedUser.Nickname,
		Avatar:      updatedUser.Avatar,
		Phone:       updatedUser.Phone,
		Status:      updatedUser.Status,
		LastLoginAt: updatedUser.LastLoginAt,
		CreatedAt:   updatedUser.CreatedAt,
		Remark:      updatedUser.Remark,
	}

	// 记录更新成功日志
	logger.LogBusinessOperation("update_user_by_id", uint(userID), "", "", "", "success", "用户信息更新成功", map[string]interface{}{
		"user_id":  userID,
		"username": updatedUser.Username,
		"email":    updatedUser.Email,
	})

	// 返回更新成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "user updated successfully",
		Data:    userInfo,
	})
}

// DeleteUser 删除用户
// @Summary 删除用户【硬删除】
// @Description 根据用户ID删除用户，包含级联删除用户角色关联
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} model.APIResponse "删除成功"
// @Failure 400 {object} model.APIResponse "参数错误"
// @Failure 404 {object} model.APIResponse "用户不存在"
// @Failure 403 {object} model.APIResponse "权限不足或业务规则限制"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// 第一层：参数解析和验证 从URL路径参数获取用户ID
	userIDStr := c.Param("id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "用户ID不能为空",
		})
		return
	}

	// 转换用户ID为uint类型
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "用户ID格式无效",
		})
		return
	}

	// 第二层：调用Service层执行删除逻辑
	ctx := c.Request.Context()
	err = h.userService.DeleteUser(ctx, uint(userID))
	if err != nil {
		// 根据错误类型返回不同的HTTP状态码
		switch {
		case strings.Contains(err.Error(), "用户ID不能为0"):
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: err.Error(),
			})
		case strings.Contains(err.Error(), "用户不存在"):
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		case strings.Contains(err.Error(), "用户已被删除") || strings.Contains(err.Error(), "不能删除系统管理员账户"):
			c.JSON(http.StatusForbidden, model.APIResponse{
				Code:    http.StatusForbidden,
				Status:  "error",
				Message: err.Error(),
			})
		default:
			// 数据库错误或其他系统错误
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "删除用户失败，请稍后重试",
			})
		}
		return
	}

	// 第三层：返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "用户删除成功",
		Data:    nil,
	})
}

// ChangePassword 修改用户密码  【已完成】
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

// ActivateUser 激活用户
func (h *UserHandler) ActivateUser(c *gin.Context) {
	// 从URL路径中获取用户ID
	userIDStr := c.Param("id")
	if userIDStr == "" {
		logger.LogError(errors.New("missing user ID"), "", 0, "", "activate_user", "HANDLER", map[string]interface{}{
			"operation": "activate_user",
			"error":     "missing_user_id",
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "用户ID不能为空",
		})
		return
	}

	// 转换用户ID为uint类型
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", 0, "", "activate_user", "HANDLER", map[string]interface{}{
			"operation": "activate_user",
			"error":     "invalid_user_id_format",
			"user_id":   userIDStr,
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "用户ID格式无效",
		})
		return
	}

	// 从上下文获取当前操作用户ID（用于审计日志）
	currentUserIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("unauthorized access"), "", 0, "", "activate_user", "HANDLER", map[string]interface{}{
			"operation":      "activate_user",
			"error":          "unauthorized",
			"target_user_id": userID,
			"timestamp":      logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	// 类型断言检查
	currentUserID, ok := currentUserIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user ID type"), "", 0, "", "activate_user", "HANDLER", map[string]interface{}{
			"operation":      "activate_user",
			"error":          "invalid_user_id_type",
			"target_user_id": userID,
			"user_id_value":  currentUserIDInterface,
			"timestamp":      logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "服务器内部错误",
		})
		return
	}

	// 调用Service层执行激活用户业务逻辑
	err = h.userService.ActivateUser(c.Request.Context(), uint(userID))
	if err != nil {
		// Service层已经记录了详细的错误日志，这里只记录Handler层的处理结果
		logger.LogError(err, "", uint(currentUserID), "", "activate_user", "HANDLER", map[string]interface{}{
			"operation":      "activate_user",
			"error":          "service_call_failed",
			"target_user_id": userID,
			"operator_id":    currentUserID,
			"timestamp":      logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else if strings.Contains(err.Error(), "不能为0") || strings.Contains(err.Error(), "格式") {
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "激活用户失败",
			})
		}
		return
	}

	// 记录成功的Handler操作日志
	logger.LogBusinessOperation("activate_user", uint(currentUserID), "", "", "", "success", "Handler层激活用户成功", map[string]interface{}{
		"operation":      "activate_user",
		"target_user_id": userID,
		"operator_id":    currentUserID,
		"timestamp":      logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "用户激活成功",
		Data: map[string]interface{}{
			"user_id": userID,
			"status":  "activated",
		},
	})
}

// DeactivateUser 禁用用户
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	// 从URL路径中获取用户ID
	userIDStr := c.Param("id")
	if userIDStr == "" {
		logger.LogError(errors.New("missing user ID"), "", 0, "", "deactivate_user", "HANDLER", map[string]interface{}{
			"operation": "deactivate_user",
			"error":     "missing_user_id",
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "用户ID不能为空",
		})
		return
	}

	// 转换用户ID为uint类型
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", 0, "", "deactivate_user", "HANDLER", map[string]interface{}{
			"operation": "deactivate_user",
			"error":     "invalid_user_id_format",
			"user_id":   userIDStr,
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "用户ID格式无效",
		})
		return
	}

	// 从上下文获取当前操作用户ID（用于审计日志）
	currentUserIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("unauthorized access"), "", 0, "", "deactivate_user", "HANDLER", map[string]interface{}{
			"operation":      "deactivate_user",
			"error":          "unauthorized",
			"target_user_id": userID,
			"timestamp":      logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	// 类型断言检查
	currentUserID, ok := currentUserIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user ID type"), "", 0, "", "activate_user", "HANDLER", map[string]interface{}{
			"operation":      "deactivate_user",
			"error":          "invalid_user_id_type",
			"target_user_id": userID,
			"user_id_value":  currentUserIDInterface,
			"timestamp":      logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "服务器内部错误",
		})
		return
	}

	// 调用Service层执行禁用用户业务逻辑
	err = h.userService.DeactivateUser(c.Request.Context(), uint(userID))
	if err != nil {
		// Service层已经记录了详细的错误日志，这里只记录Handler层的处理结果
		logger.LogError(err, "", uint(currentUserID), "", "deactivate_user", "HANDLER", map[string]interface{}{
			"operation":      "deactivate_user",
			"error":          "service_call_failed",
			"target_user_id": userID,
			"operator_id":    currentUserID,
			"timestamp":      logger.NowFormatted(),
		})

		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "不存在") {
			c.JSON(http.StatusNotFound, model.APIResponse{
				Code:    http.StatusNotFound,
				Status:  "error",
				Message: err.Error(),
			})
		} else if strings.Contains(err.Error(), "不能为0") || strings.Contains(err.Error(), "格式") {
			c.JSON(http.StatusBadRequest, model.APIResponse{
				Code:    http.StatusBadRequest,
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, model.APIResponse{
				Code:    http.StatusInternalServerError,
				Status:  "error",
				Message: "禁用用户失败",
			})
		}
		return
	}

	// 记录成功的Handler操作日志
	logger.LogBusinessOperation("deactivate_user", uint(currentUserID), "", "", "", "success", "Handler层禁用用户成功", map[string]interface{}{
		"operation":      "deactivate_user",
		"target_user_id": userID,
		"operator_id":    currentUserID,
		"timestamp":      logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "用户禁用成功",
		Data: map[string]interface{}{
			"user_id": userID,
			"status":  "deactivated",
		},
	})
}

// ResetUserPassword 重置用户密码
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	// 从上下文获取管理员用户ID（中间件已验证并存储）
	adminIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "reset_user_password", "POST", map[string]interface{}{
			"operation":  "reset_user_password",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}

	adminID, ok := adminIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "reset_user_password", "POST", map[string]interface{}{
			"operation":  "reset_user_password",
			"user_id":    adminIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	// 从URL路径中获取目标用户ID
	userIDStr := c.Param("id")
	if userIDStr == "" {
		logger.LogError(errors.New("missing user ID"), "", adminID, "", "reset_user_password", "POST", map[string]interface{}{
			"operation": "reset_user_password",
			"error":     "missing_user_id",
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "缺少用户ID"})
		return
	}

	userID64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", adminID, "", "reset_user_password", "POST", map[string]interface{}{
			"operation":  "reset_user_password",
			"target_id":  userIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "无效的用户ID"})
		return
	}

	// 调用服务层重置密码（服务层固定为 123456）
	if rerr := h.userService.ResetUserPassword(c.Request.Context(), uint(userID64), ""); rerr != nil {
		logger.LogError(rerr, "", adminID, "", "reset_user_password", "POST", map[string]interface{}{
			"operation":  "reset_user_password",
			"target_id":  userID64,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "重置密码失败: " + rerr.Error()})
		return
	}

	// 记录成功业务日志
	logger.LogBusinessOperation("reset_user_password", adminID, "", "", "", "success", "重置用户密码成功", map[string]interface{}{
		"target_id":  userID64,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, model.APIResponse{Code: http.StatusOK, Status: "success", Message: "重置密码成功"})
}
