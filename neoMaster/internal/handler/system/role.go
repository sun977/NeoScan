/**
 * @author: sun977
 * @date: 2025.09.11
 * @description: 角色管理接口(管理员专用)
 * @func:
 * 	1.创建角色
 * 	2.获取角色列表
 * 	3.获取单个角色
 * 	4.更新角色
 * 	5.删除角色
 * 	6.角色状态管理
 * @note：中间件 r.middlewareManager.GinJWTAuthMiddleware() 已经解析了token，并获取了用户信息
 * 解析出来的用户信息保存在 gin.Context 上下文中，可以通过 c.Get("user") 获取
 * 获取的字段有：user_id，username，roles，permissions，claims
 */
package system

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/auth"
)

// RoleHandler 角色管理处理器
type RoleHandler struct {
	roleService *auth.RoleService // 角色服务，用于角色相关操作
}

// NewRoleHandler 创建角色管理处理器
func NewRoleHandler(roleService *auth.RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

// CreateRole 创建角色（管理员专用）
// 创建新角色，包含完整的参数验证和权限检查
func (h *RoleHandler) CreateRole(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		// 记录获取用户ID失败错误日志
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "create_role", "POST", map[string]interface{}{
			"operation":  "create_role",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "create_role", "POST", map[string]interface{}{
			"operation":  "create_role",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "内部服务器错误",
		})
		return
	}

	// 解析请求体
	var req model.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogError(err, "", userID, "", "create_role", "POST", map[string]interface{}{
			"operation":  "create_role",
			"user_id":    userID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数无效: " + err.Error(),
		})
		return
	}

	// 调用服务层创建角色
	role, err := h.roleService.CreateRole(c.Request.Context(), &req)
	if err != nil {
		logger.LogError(err, "", userID, "", "create_role", "POST", map[string]interface{}{
			"operation":  "create_role",
			"user_id":    userID,
			"role_name":  req.Name,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "创建角色失败: " + err.Error(),
		})
		return
	}

	// 记录成功创建角色的业务日志
	logger.LogBusinessOperation("create_role", userID, "", "", "", "success", "角色创建成功", map[string]interface{}{
		"operation":  "create_role",
		"user_id":    userID,
		"role_id":    role.ID,
		"role_name":  role.Name,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusCreated, model.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "角色创建成功",
		Data:    role,
	})
}

// GetRoleList 获取角色列表（管理员专用）
// 支持分页查询，包含完整的参数验证和权限检查
func (h *RoleHandler) GetRoleList(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "get_role_list", "GET", map[string]interface{}{
			"operation":  "get_role_list",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "get_role_list", "GET", map[string]interface{}{
			"operation":  "get_role_list",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "内部服务器错误",
		})
		return
	}

	// 解析分页参数
	page, limit := parsePaginationParams(c)
	offset := (page - 1) * limit

	// 调用服务层获取角色列表
	roles, total, err := h.roleService.GetRoleList(c.Request.Context(), offset, limit)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_role_list", "GET", map[string]interface{}{
			"operation":  "get_role_list",
			"user_id":    userID,
			"page":       page,
			"limit":      limit,
			"offset":     offset,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取角色列表失败: " + err.Error(),
		})
		return
	}

	// 构造分页响应
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := &model.PaginationResponse{
		Page:        page,
		PageSize:    limit,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	// 转换 []*model.Role 为 []model.Role
	roleList := make([]model.Role, len(roles))
	for i, role := range roles {
		roleList[i] = *role
	}

	response := model.RoleListResponse{
		Roles:      roleList,
		Pagination: pagination,
	}

	// 记录成功获取角色列表的业务日志
	logger.LogBusinessOperation("get_role_list", userID, "", "", "", "success", "获取角色列表成功", map[string]interface{}{
		"operation":    "get_role_list",
		"user_id":      userID,
		"page":         page,
		"limit":        limit,
		"total":        total,
		"result_count": len(roles),
		"client_ip":    c.ClientIP(),
		"user_agent":   c.GetHeader("User-Agent"),
		"request_id":   c.GetHeader("X-Request-ID"),
		"timestamp":    logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取角色列表成功",
		Data:    response,
	})
}

// GetRoleByID 根据ID获取角色（管理员专用）
// 获取指定角色的详细信息，包含完整的参数验证和权限检查
func (h *RoleHandler) GetRoleByID(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "get_role_by_id", "GET", map[string]interface{}{
			"operation":  "get_role_by_id",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "get_role_by_id", "GET", map[string]interface{}{
			"operation":  "get_role_by_id",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "内部服务器错误",
		})
		return
	}

	// 解析角色ID参数
	roleIDStr := c.Param("id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_role_by_id", "GET", map[string]interface{}{
			"operation":  "get_role_by_id",
			"user_id":    userID,
			"role_id":    roleIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的角色ID",
		})
		return
	}

	// 调用服务层获取角色信息
	role, err := h.roleService.GetRoleByID(c.Request.Context(), uint(roleID))
	if err != nil {
		logger.LogError(err, "", userID, "", "get_role_by_id", "GET", map[string]interface{}{
			"operation":  "get_role_by_id",
			"user_id":    userID,
			"role_id":    roleID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "获取角色信息失败: " + err.Error(),
		})
		return
	}

	// 记录成功获取角色信息的业务日志
	logger.LogBusinessOperation("get_role_by_id", userID, "", "", "", "success", "获取角色信息成功", map[string]interface{}{
		"operation":  "get_role_by_id",
		"user_id":    userID,
		"role_id":    roleID,
		"role_name":  role.Name,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "获取角色信息成功",
		Data:    role,
	})
}

// UpdateRole 更新角色（管理员专用）
// 更新指定角色的信息，包含完整的参数验证和权限检查
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "update_role", "PUT", map[string]interface{}{
			"operation":  "update_role",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "update_role", "PUT", map[string]interface{}{
			"operation":  "update_role",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "内部服务器错误",
		})
		return
	}

	// 解析角色ID参数
	roleIDStr := c.Param("id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", userID, "", "update_role", "PUT", map[string]interface{}{
			"operation":  "update_role",
			"user_id":    userID,
			"role_id":    roleIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的角色ID",
		})
		return
	}

	// 解析请求体
	var req model.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogError(err, "", userID, "", "update_role", "PUT", map[string]interface{}{
			"operation":  "update_role",
			"user_id":    userID,
			"role_id":    roleID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "请求参数无效: " + err.Error(),
		})
		return
	}

	// 调用服务层更新角色
	role, err := h.roleService.UpdateRoleByID(c.Request.Context(), uint(roleID), &req)
	if err != nil {
		logger.LogError(err, "", userID, "", "update_role", "PUT", map[string]interface{}{
			"operation":  "update_role",
			"user_id":    userID,
			"role_id":    roleID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "更新角色失败: " + err.Error(),
		})
		return
	}

	// 记录成功更新角色的业务日志
	logger.LogBusinessOperation("update_role", userID, "", "", "", "success", "角色更新成功", map[string]interface{}{
		"operation":  "update_role",
		"user_id":    userID,
		"role_id":    roleID,
		"role_name":  role.Name,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "角色更新成功",
		Data:    role,
	})
}

// DeleteRole 删除角色（管理员专用）
// 软删除指定角色，包含完整的参数验证和权限检查
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "delete_role", "DELETE", map[string]interface{}{
			"operation":  "delete_role",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "delete_role", "DELETE", map[string]interface{}{
			"operation":  "delete_role",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "内部服务器错误",
		})
		return
	}

	// 解析角色ID参数
	roleIDStr := c.Param("id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", userID, "", "delete_role", "DELETE", map[string]interface{}{
			"operation":  "delete_role",
			"user_id":    userID,
			"role_id":    roleIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的角色ID",
		})
		return
	}

	// 调用服务层删除角色
	err = h.roleService.DeleteRole(c.Request.Context(), uint(roleID))
	if err != nil {
		logger.LogError(err, "", userID, "", "delete_role", "DELETE", map[string]interface{}{
			"operation":  "delete_role",
			"user_id":    userID,
			"role_id":    roleID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "删除角色失败: " + err.Error(),
		})
		return
	}

	// 记录成功删除角色的业务日志
	logger.LogBusinessOperation("delete_role", userID, "", "", "", "success", "角色删除成功", map[string]interface{}{
		"operation":  "delete_role",
		"user_id":    userID,
		"role_id":    roleID,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "角色删除成功",
	})
}

// ActivateRole 激活角色（管理员专用）
// 将指定角色的状态设置为启用状态
func (h *RoleHandler) ActivateRole(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "activate_role", "POST", map[string]interface{}{
			"operation":  "activate_role",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "activate_role", "POST", map[string]interface{}{
			"operation":  "activate_role",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "内部服务器错误",
		})
		return
	}

	// 解析角色ID参数
	roleIDStr := c.Param("id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", userID, "", "activate_role", "POST", map[string]interface{}{
			"operation":  "activate_role",
			"user_id":    userID,
			"role_id":    roleIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的角色ID",
		})
		return
	}

	// 调用服务层激活角色
	err = h.roleService.ActivateRole(c.Request.Context(), uint(roleID))
	if err != nil {
		logger.LogError(err, "", userID, "", "activate_role", "POST", map[string]interface{}{
			"operation":  "activate_role",
			"user_id":    userID,
			"role_id":    roleID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "激活角色失败: " + err.Error(),
		})
		return
	}

	// 记录成功激活角色的业务日志
	logger.LogBusinessOperation("activate_role", userID, "", "", "", "success", "角色激活成功", map[string]interface{}{
		"operation":  "activate_role",
		"user_id":    userID,
		"role_id":    roleID,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "角色激活成功",
	})
}

// DeactivateRole 禁用角色（管理员专用）
// 将指定角色的状态设置为禁用状态
func (h *RoleHandler) DeactivateRole(c *gin.Context) {
	// 从上下文获取用户ID（中间件已验证并存储）
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "deactivate_role", "POST", map[string]interface{}{
			"operation":  "deactivate_role",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "未授权访问",
		})
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "deactivate_role", "POST", map[string]interface{}{
			"operation":  "deactivate_role",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "内部服务器错误",
		})
		return
	}

	// 解析角色ID参数
	roleIDStr := c.Param("id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", userID, "", "deactivate_role", "POST", map[string]interface{}{
			"operation":  "deactivate_role",
			"user_id":    userID,
			"role_id":    roleIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "无效的角色ID",
		})
		return
	}

	// 调用服务层禁用角色
	err = h.roleService.DeactivateRole(c.Request.Context(), uint(roleID))
	if err != nil {
		logger.LogError(err, "", userID, "", "deactivate_role", "POST", map[string]interface{}{
			"operation":  "deactivate_role",
			"user_id":    userID,
			"role_id":    roleID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "禁用角色失败: " + err.Error(),
		})
		return
	}

	// 记录成功禁用角色的业务日志
	logger.LogBusinessOperation("deactivate_role", userID, "", "", "", "success", "角色禁用成功", map[string]interface{}{
		"operation":  "deactivate_role",
		"user_id":    userID,
		"role_id":    roleID,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "角色禁用成功",
	})
}
