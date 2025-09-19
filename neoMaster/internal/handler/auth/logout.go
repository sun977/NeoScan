package auth

import (
	"net/http"
	"strings"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// LogoutHandler 登出接口处理器
type LogoutHandler struct {
	sessionService *auth.SessionService
}

// NewLogoutHandler 创建登出处理器实例
func NewLogoutHandler(sessionService *auth.SessionService) *LogoutHandler {
	return &LogoutHandler{
		sessionService: sessionService,
	}
}

// getErrorStatusCode 根据错误类型获取HTTP状态码
func (h *LogoutHandler) getErrorStatusCode(err error) int {
	switch {
	case strings.Contains(err.Error(), "invalid token"):
		return http.StatusUnauthorized
	case strings.Contains(err.Error(), "token expired"):
		return http.StatusUnauthorized
	case strings.Contains(err.Error(), "token revoked"):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// Logout 用户登出接口(AccessToken撤销)
func (h *LogoutHandler) Logout(c *gin.Context) {
	// 从请求头中获取访问令牌
	accessToken, err := h.extractTokenFromHeader(c)
	if err != nil {
		// 记录令牌提取失败错误日志
		logger.LogError(err, "", 0, "", "user_logout", "POST", map[string]interface{}{
			"operation":            "logout",
			"client_ip":            c.ClientIP(),
			"user_agent":           c.GetHeader("User-Agent"),
			"request_id":           c.GetHeader("X-Request-ID"),
			"authorization_header": c.GetHeader("Authorization") != "",
			"timestamp":            logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "missing or invalid authorization header",
			Error:   err.Error(),
		})
		return
	}

	// 执行登出
	err = h.sessionService.Logout(c.Request.Context(), accessToken)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		// 记录登出失败错误日志
		logger.LogError(err, "", 0, "", "user_logout", "POST", map[string]interface{}{
			"operation":   "logout",
			"client_ip":   c.ClientIP(),
			"user_agent":  c.GetHeader("User-Agent"),
			"status_code": statusCode,
			"request_id":  c.GetHeader("X-Request-ID"),
			"has_token":   accessToken != "",
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Status:  "error",
			Message: "logout failed",
			Error:   err.Error(),
		})
		return
	}

	// 记录登出成功业务日志
	logger.LogBusinessOperation("user_logout", 0, "", "", "", "success", "用户登出成功", map[string]interface{}{
		"operation":  "logout",
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "logout successful",
		Data:    nil,
	})
}

// LogoutAll 用户全部登出接口(更新密码版本,所有类型token失效)
func (h *LogoutHandler) LogoutAll(c *gin.Context) {
	// 从请求头中获取访问令牌
	accessToken, err := h.extractTokenFromHeader(c)
	if err != nil {
		// 记录令牌提取失败错误日志
		logger.LogError(err, "", 0, "", "user_logout_all", "POST", map[string]interface{}{
			"operation":            "logout_all",
			"client_ip":            c.ClientIP(),
			"user_agent":           c.GetHeader("User-Agent"),
			"request_id":           c.GetHeader("X-Request-ID"),
			"authorization_header": c.GetHeader("Authorization") != "",
			"timestamp":            logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "missing or invalid authorization header",
			Error:   err.Error(),
		})
		return
	}

	// 验证令牌并获取用户ID
	user, err := h.sessionService.ValidateSession(c.Request.Context(), accessToken)
	if err != nil {
		// 记录令牌验证失败错误日志
		logger.LogError(err, "", 0, "", "user_logout_all", "POST", map[string]interface{}{
			"operation":  "logout_all",
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"has_token":  accessToken != "",
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "invalid token",
			Error:   err.Error(),
		})
		return
	}

	// 执行全部登出
	err = h.sessionService.LogoutAll(c.Request.Context(), accessToken)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		// 记录全部登出失败错误日志
		logger.LogError(err, "", uint(user.ID), "", "user_logout_all", "POST", map[string]interface{}{
			"operation":   "logout_all",
			"client_ip":   c.ClientIP(),
			"user_agent":  c.GetHeader("User-Agent"),
			"status_code": statusCode,
			"request_id":  c.GetHeader("X-Request-ID"),
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Status:  "error",
			Message: "logout all failed",
			Error:   err.Error(),
		})
		return
	}

	// 记录全部登出成功业务日志
	logger.LogBusinessOperation("user_logout_all", uint(user.ID), "", "", "", "success", "用户全部登出成功", map[string]interface{}{
		"operation":  "logout_all",
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "logout all successful",
		Data: map[string]interface{}{
			"user_id": user.ID,
			"message": "All sessions have been terminated",
		},
	})
}

// extractTokenFromHeader 从请求头中提取访问令牌
func (h *LogoutHandler) extractTokenFromHeader(c *gin.Context) (string, error) {
	authorization := c.GetHeader("Authorization")
	if authorization == "" {
		return "", &model.ValidationError{Field: "authorization", Message: "authorization header is required"}
	}

	// 检查Bearer前缀
	if !strings.HasPrefix(authorization, "Bearer ") {
		return "", &model.ValidationError{Field: "authorization", Message: "authorization header must start with 'Bearer '"}
	}

	// 提取令牌
	token := strings.TrimPrefix(authorization, "Bearer ")
	if token == "" {
		return "", &model.ValidationError{Field: "authorization", Message: "access token cannot be empty"}
	}

	return token, nil
}
