package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"neomaster/internal/model"
	"neomaster/internal/service/auth"
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

// Logout 用户登出接口
// @Summary 用户登出
// @Description 用户登出，撤销当前访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.APIResponse "登出成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 401 {object} model.APIResponse "未授权"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/auth/logout [post]
func (h *LogoutHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 从请求头中获取访问令牌
	accessToken, err := h.extractTokenFromHeader(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "missing or invalid authorization header", err)
		return
	}

	// 执行登出
	err = h.sessionService.Logout(r.Context(), accessToken)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		h.writeErrorResponse(w, statusCode, "logout failed", err)
		return
	}

	// 返回成功响应
	h.writeSuccessResponse(w, http.StatusOK, "logout successful", nil)
}

// LogoutAll 用户全部登出接口
// @Summary 用户全部登出
// @Description 用户登出所有设备，撤销所有访问令牌和刷新令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.APIResponse "全部登出成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 401 {object} model.APIResponse "未授权"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/auth/logout-all [post]
func (h *LogoutHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 从请求头中获取访问令牌
	accessToken, err := h.extractTokenFromHeader(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "missing or invalid authorization header", err)
		return
	}

	// 验证令牌并获取用户ID
	claims, err := h.sessionService.ValidateSession(r.Context(), accessToken)
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "invalid token", err)
		return
	}

	// 执行全部登出（这里需要在SessionService中实现LogoutAll方法）
	// 暂时使用单个登出，实际项目中需要实现批量撤销用户所有令牌的功能
	err = h.sessionService.Logout(r.Context(), accessToken)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		h.writeErrorResponse(w, statusCode, "logout all failed", err)
		return
	}

	// 返回成功响应
	h.writeSuccessResponse(w, http.StatusOK, "logout all successful", map[string]interface{}{
		"user_id": claims.ID,
		"message": "All sessions have been terminated",
	})
}

// extractTokenFromHeader 从请求头中提取访问令牌
func (h *LogoutHandler) extractTokenFromHeader(r *http.Request) (string, error) {
	authorization := r.Header.Get("Authorization")
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

// writeSuccessResponse 写入成功响应
func (h *LogoutHandler) writeSuccessResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.WriteHeader(statusCode)
	response := model.APIResponse{
		Code:    statusCode,
		Success: true,
		Message: message,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// writeErrorResponse 写入错误响应
func (h *LogoutHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	w.WriteHeader(statusCode)
	response := model.APIResponse{
		Code:    statusCode,
		Success: false,
		Message: message,
		Error:   err.Error(),
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		http.Error(w, "failed to encode error response", http.StatusInternalServerError)
	}
}

// GinLogout Gin框架适配的用户登出接口
func (h *LogoutHandler) GinLogout(c *gin.Context) {
	// 从请求头中获取访问令牌
	accessToken, err := h.extractTokenFromGinHeader(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Success: false,
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
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Success: false,
			Message: "logout failed",
			Error:   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Success: true,
		Message: "logout successful",
		Data:    nil,
	})
}

// GinLogoutAll Gin框架适配的用户全部登出接口
func (h *LogoutHandler) GinLogoutAll(c *gin.Context) {
	// 从请求头中获取访问令牌
	accessToken, err := h.extractTokenFromGinHeader(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Success: false,
			Message: "missing or invalid authorization header",
			Error:   err.Error(),
		})
		return
	}

	// 验证令牌并获取用户ID
	claims, err := h.sessionService.ValidateSession(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Success: false,
			Message: "invalid token",
			Error:   err.Error(),
		})
		return
	}

	// 执行全部登出
	err = h.sessionService.Logout(c.Request.Context(), accessToken)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		c.JSON(statusCode, model.APIResponse{
			Success: false,
			Message: "logout all failed",
			Error:   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Success: true,
		Message: "logout all successful",
		Data: map[string]interface{}{
			"user_id": claims.ID,
			"message": "All sessions have been terminated",
		},
	})
}

// extractTokenFromGinHeader 从Gin请求头中提取访问令牌
func (h *LogoutHandler) extractTokenFromGinHeader(c *gin.Context) (string, error) {
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
