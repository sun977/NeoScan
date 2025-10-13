package system

import (
	"errors"
	"neomaster/internal/model/system"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/service/auth"
)

// SessionHandler 会话管理处理器
type SessionHandler struct {
	sessionService *auth.SessionService
}

// NewSessionHandler 创建会话管理处理器
func NewSessionHandler(sessionService *auth.SessionService) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

// ListActiveSessions 列出指定用户的活跃会话
// GET /api/v1/admin/sessions/list?user_id=123
func (h *SessionHandler) ListActiveSessions(c *gin.Context) {
	// 规范化参数变量
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	userIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), XRequestID, 0, clientIP, "list_active_sessions", "GET", map[string]interface{}{
			"operation":  "list_active_sessions",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, system.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("user_id type assertion failed"), XRequestID, userID, clientIP, "list_active_sessions", "GET", map[string]interface{}{
			"operation":  "list_active_sessions",
			"user_id":    userID,
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	// 管理入口：通过查询参数指定要查看的用户ID Query 参数
	queryUserIDStr := c.Query("userId")
	if queryUserIDStr == "" {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "缺少 user_id 参数"})
		return
	}
	queryUserID64, err := strconv.ParseUint(queryUserIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "无效的 user_id 参数"})
		return
	}

	sessions, serr := h.sessionService.GetUserSessions(c.Request.Context(), uint(queryUserID64))
	if serr != nil {
		logger.LogError(serr, XRequestID, userID, clientIP, "list_active_sessions", "GET", map[string]interface{}{
			"operation":   "list_active_sessions",
			"target_user": queryUserID64,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "获取活跃会话失败: " + serr.Error()})
		return
	}

	c.JSON(http.StatusOK, system.APIResponse{Code: http.StatusOK, Status: "success", Message: "获取活跃会话成功", Data: sessions})
}

// RevokeSession 撤销某个用户当前会话
// POST /api/v1/admin/sessions/:userId/revoke
func (h *SessionHandler) RevokeSession(c *gin.Context) {
	// 规范化参数变量
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	adminIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), XRequestID, 0, clientIP, "revoke_session", "POST", map[string]interface{}{
			"operation":  "revoke_session",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, system.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}
	adminID, ok := adminIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("user_id type assertion failed"), XRequestID, adminID, clientIP, "revoke_session", "POST", map[string]interface{}{
			"operation":  "revoke_session",
			"admin_id":   adminID,
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	// 	Param 路径参数
	userIDStr := c.Param("userId")
	userID64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, XRequestID, adminID, clientIP, "revoke_session", "POST", map[string]interface{}{
			"operation":   "revoke_session",
			"admin_id":    adminID,
			"target_user": userID64,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "无效的用户ID"})
		return
	}

	if derr := h.sessionService.DeleteUserSession(c.Request.Context(), uint(userID64)); derr != nil {
		logger.LogError(derr, XRequestID, adminID, clientIP, "revoke_session", "POST", map[string]interface{}{
			"operation":   "revoke_session",
			"admin_id":    adminID,
			"target_user": userID64,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "撤销会话失败: " + derr.Error()})
		return
	}

	logger.LogBusinessOperation("revoke_session", adminID, "", clientIP, XRequestID, "success", "撤销用户会话成功", map[string]interface{}{
		"target_user": userID64,
		"admin_id":    adminID,
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"request_id":  XRequestID,
		"timestamp":   logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, system.APIResponse{Code: http.StatusOK, Status: "success", Message: "撤销会话成功"})
}

// RevokeAllUserSessions 撤销某个用户的所有会话
// POST /api/v1/admin/sessions/user/:userId/revoke-all
func (h *SessionHandler) RevokeAllUserSessions(c *gin.Context) {
	// 规范化参数变量
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	adminIDInterface, exists := c.Get("user_id")
	if !exists {
		logger.LogError(errors.New("user_id not found in context"), XRequestID, 0, clientIP, "revoke_all_user_sessions", "POST", map[string]interface{}{
			"operation":  "revoke_all_user_sessions",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, system.APIResponse{Code: http.StatusUnauthorized, Status: "error", Message: "未授权访问"})
		return
	}
	adminID, ok := adminIDInterface.(uint)
	if !ok {
		logger.LogError(errors.New("user_id type assertion failed"), XRequestID, adminID, clientIP, "revoke_all_user_sessions", "POST", map[string]interface{}{
			"operation":  "revoke_all_user_sessions",
			"admin_id":   adminID,
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "内部服务器错误"})
		return
	}

	userIDStr := c.Param("userId")
	userID64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		logger.LogError(err, XRequestID, adminID, clientIP, "revoke_all_user_sessions", "POST", map[string]interface{}{
			"operation":   "revoke_all_user_sessions",
			"admin_id":    adminID,
			"target_user": userID64,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "error", Message: "无效的用户ID"})
		return
	}

	if derr := h.sessionService.DeleteAllUserSessions(c.Request.Context(), uint(userID64)); derr != nil {
		logger.LogError(derr, XRequestID, adminID, clientIP, "revoke_all_user_sessions", "POST", map[string]interface{}{
			"operation":   "revoke_all_user_sessions",
			"target_user": userID64,
			"admin_id":    adminID,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{Code: http.StatusInternalServerError, Status: "error", Message: "撤销用户所有会话失败: " + derr.Error()})
		return
	}

	logger.LogBusinessOperation("revoke_all_user_sessions", adminID, "", clientIP, XRequestID, "success", "撤销用户所有会话成功", map[string]interface{}{
		"target_user": userID64,
		"admin_id":    adminID,
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"request_id":  XRequestID,
		"timestamp":   logger.NowFormatted(),
	})

	c.JSON(http.StatusOK, system.APIResponse{Code: http.StatusOK, Status: "success", Message: "撤销用户所有会话成功"})
}
