/**
 * @author: sun977
 * @date: 2025.09.11
 * @description: 权限管理接口(管理员专用)
 * @func:
 * 	1.创建权限
 * 	2.获取权限列表
 * 	3.获取单个权限
 * 	4.更新权限
 * 	5.删除权限
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

// PermissionHandler 权限管理处理器
type PermissionHandler struct {
	permissionService *auth.PermissionService // 权限服务
}

// NewPermissionHandler 创建权限管理处理器
func NewPermissionHandler(permissionService *auth.PermissionService) *PermissionHandler {
	return &PermissionHandler{permissionService: permissionService}
}

// CreatePermission 创建权限
func (h *PermissionHandler) CreatePermission(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "create_permission", "POST", map[string]interface{}{
			"operation":  "create_permission",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "create_permission", "POST", map[string]interface{}{
			"operation":  "create_permission",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	var req model.CreatePermissionRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logger.LogError(bindErr, "", userID, "", "create_permission", "POST", map[string]interface{}{
			"operation":  "create_permission",
			"user_id":    userID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "请求参数无效: " + bindErr.Error()})
		return
	}

	permission, err := h.permissionService.CreatePermission(c.Request.Context(), &req)
	if err != nil {
		logger.LogError(err, "", userID, "", "create_permission", "POST", map[string]interface{}{
			"operation":  "create_permission",
			"user_id":    userID,
			"name":       req.Name,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "创建权限失败: " + err.Error()})
		return
	}

	logger.LogBusinessOperation("create_permission", userID, "", "", "", "success", "权限创建成功", map[string]interface{}{
		"operation":  "create_permission",
		"user_id":    userID,
		"perm_id":    permission.ID,
		"perm_name":  permission.Name,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	c.JSON(http.StatusCreated, model.APIResponse{Code: http.StatusCreated, Status: "success", Message: "权限创建成功", Data: permission})
}

// GetPermissionList 获取权限列表
func (h *PermissionHandler) GetPermissionList(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "get_permission_list", "GET", map[string]interface{}{
			"operation":  "get_permission_list",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "get_permission_list", "GET", map[string]interface{}{
			"operation":  "get_permission_list",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	page, limit := parsePaginationParams(c)
	offset := (page - 1) * limit

	permissions, total, err := h.permissionService.GetPermissionList(c.Request.Context(), offset, limit)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_permission_list", "GET", map[string]interface{}{
			"operation":  "get_permission_list",
			"user_id":    userID,
			"page":       page,
			"limit":      limit,
			"offset":     offset,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "获取权限列表失败: " + err.Error()})
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := &model.PaginationResponse{Page: page, PageSize: limit, Total: total, TotalPages: totalPages, HasNext: page < totalPages, HasPrevious: page > 1}

	permList := make([]model.Permission, len(permissions))
	for i, p := range permissions {
		permList[i] = *p
	}

	response := model.PermissionListResponse{Permissions: permList, Pagination: pagination}

	logger.LogBusinessOperation("get_permission_list", userID, "", "", "", "success", "获取权限列表成功", map[string]interface{}{
		"operation":    "get_permission_list",
		"user_id":      userID,
		"page":         page,
		"limit":        limit,
		"total":        total,
		"result_count": len(permissions),
		"client_ip":    c.ClientIP(),
		"user_agent":   c.GetHeader("User-Agent"),
		"request_id":   c.GetHeader("X-Request-ID"),
		"timestamp":    logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, model.APIResponse{Code: http.StatusOK, Status: "success", Message: "获取权限列表成功", Data: response})
}

// GetPermissionByID 获取单个权限
func (h *PermissionHandler) GetPermissionByID(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "get_permission_by_id", "GET", map[string]interface{}{
			"operation":  "get_permission_by_id",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "get_permission_by_id", "GET", map[string]interface{}{
			"operation":  "get_permission_by_id",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	permIDStr := c.Param("id")
	permID, err := strconv.ParseUint(permIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", userID, "", "get_permission_by_id", "GET", map[string]interface{}{
			"operation":  "get_permission_by_id",
			"user_id":    userID,
			"perm_id":    permIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "无效的权限ID"})
		return
	}

	permission, serr := h.permissionService.GetPermissionByID(c.Request.Context(), uint(permID))
	if serr != nil {
		logger.LogError(serr, "", userID, "", "get_permission_by_id", "GET", map[string]interface{}{
			"operation":  "get_permission_by_id",
			"user_id":    userID,
			"perm_id":    permID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "获取权限信息失败: " + serr.Error()})
		return
	}

	logger.LogBusinessOperation("get_permission_by_id", userID, "", "", "", "success", "获取权限信息成功", map[string]interface{}{
		"operation":  "get_permission_by_id",
		"user_id":    userID,
		"perm_id":    permID,
		"perm_name":  permission.Name,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, model.APIResponse{Code: http.StatusOK, Status: "success", Message: "获取权限信息成功", Data: permission})
}

// UpdatePermission 更新权限
func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "update_permission", "POST", map[string]interface{}{
			"operation":  "update_permission",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "update_permission", "POST", map[string]interface{}{
			"operation":  "update_permission",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	permIDStr := c.Param("id")
	permID, err := strconv.ParseUint(permIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", userID, "", "update_permission", "POST", map[string]interface{}{
			"operation":  "update_permission",
			"user_id":    userID,
			"perm_id":    permIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "无效的权限ID"})
		return
	}

	var req model.UpdatePermissionRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		logger.LogError(bindErr, "", userID, "", "update_permission", "POST", map[string]interface{}{
			"operation":  "update_permission",
			"user_id":    userID,
			"perm_id":    permID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "请求参数无效: " + bindErr.Error()})
		return
	}

	permission, uerr := h.permissionService.UpdatePermissionByID(c.Request.Context(), uint(permID), &req)
	if uerr != nil {
		logger.LogError(uerr, "", userID, "", "update_permission", "POST", map[string]interface{}{
			"operation":  "update_permission",
			"user_id":    userID,
			"perm_id":    permID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "更新权限失败: " + uerr.Error()})
		return
	}

	logger.LogBusinessOperation("update_permission", userID, "", "", "", "success", "权限更新成功", map[string]interface{}{
		"operation":  "update_permission",
		"user_id":    userID,
		"perm_id":    permID,
		"perm_name":  permission.Name,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, model.APIResponse{Code: http.StatusOK, Status: "success", Message: "权限更新成功", Data: permission})
}

// DeletePermission 删除权限
func (h *PermissionHandler) DeletePermission(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), "", 0, "", "delete_permission", "DELETE", map[string]interface{}{
			"operation":  "delete_permission",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("invalid user_id type in context"), "", 0, "", "delete_permission", "DELETE", map[string]interface{}{
			"operation":  "delete_permission",
			"user_id":    userIDInterface,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	permIDStr := c.Param("id")
	permID, err := strconv.ParseUint(permIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, "", userID, "", "delete_permission", "DELETE", map[string]interface{}{
			"operation":  "delete_permission",
			"user_id":    userID,
			"perm_id":    permIDStr,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "无效的权限ID"})
		return
	}

	if derr := h.permissionService.DeletePermission(c.Request.Context(), uint(permID)); derr != nil {
		logger.LogError(derr, "", userID, "", "delete_permission", "DELETE", map[string]interface{}{
			"operation":  "delete_permission",
			"user_id":    userID,
			"perm_id":    permID,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "删除权限失败: " + derr.Error()})
		return
	}

	logger.LogBusinessOperation("delete_permission", userID, "", "", "", "success", "权限删除成功", map[string]interface{}{
		"operation":  "delete_permission",
		"user_id":    userID,
		"perm_id":    permID,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, model.APIResponse{Code: http.StatusOK, Status: "success", Message: "权限删除成功"})
}
