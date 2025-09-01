package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"neomaster/internal/model"
	"neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// RefreshHandler 令牌刷新接口处理器
type RefreshHandler struct {
	sessionService *auth.SessionService
}

// NewRefreshHandler 创建令牌刷新处理器实例
func NewRefreshHandler(sessionService *auth.SessionService) *RefreshHandler {
	return &RefreshHandler{
		sessionService: sessionService,
	}
}

// RefreshToken 刷新访问令牌接口
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body model.RefreshTokenRequest true "刷新令牌请求"
// @Success 200 {object} model.APIResponse{data=model.LoginResponse} "刷新成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 401 {object} model.APIResponse "刷新令牌无效或已过期"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/auth/refresh [post]
func (h *RefreshHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 解析请求体
	var req model.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// 验证请求参数
	if err := h.validateRefreshRequest(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "validation failed", err)
		return
	}

	// 执行令牌刷新
	resp, err := h.sessionService.RefreshToken(r.Context(), &req)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		h.writeErrorResponse(w, statusCode, "refresh token failed", err)
		return
	}

	// 返回成功响应
	h.writeSuccessResponse(w, http.StatusOK, "refresh token successful", resp)
}

// RefreshTokenFromHeader 从请求头刷新访问令牌接口
// @Summary 从请求头刷新访问令牌
// @Description 从Authorization头中提取刷新令牌并获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.APIResponse{data=model.LoginResponse} "刷新成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 401 {object} model.APIResponse "刷新令牌无效或已过期"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/auth/refresh-header [post]
func (h *RefreshHandler) RefreshTokenFromHeader(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 从请求头中获取刷新令牌
	refreshToken, err := h.extractTokenFromHeader(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "missing or invalid authorization header", err)
		return
	}

	// 构造刷新请求
	req := &model.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	// 执行令牌刷新
	resp, err := h.sessionService.RefreshToken(r.Context(), req)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		h.writeErrorResponse(w, statusCode, "refresh token failed", err)
		return
	}

	// 返回成功响应
	h.writeSuccessResponse(w, http.StatusOK, "refresh token successful", resp)
}

// CheckTokenExpiry 检查令牌过期时间接口
// @Summary 检查令牌过期时间
// @Description 检查访问令牌的过期时间和剩余有效时间
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.APIResponse{data=model.TokenExpiryInfo} "检查成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 401 {object} model.APIResponse "令牌无效或已过期"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/auth/check-expiry [get]
func (h *RefreshHandler) CheckTokenExpiry(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 从请求头中获取访问令牌
	accessToken, err := h.extractTokenFromHeader(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusUnauthorized, "missing or invalid authorization header", err)
		return
	}

	// 获取令牌剩余时间
	remainingTime, err := h.sessionService.GetTokenRemainingTime(accessToken)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		h.writeErrorResponse(w, statusCode, "check token expiry failed", err)
		return
	}

	// 构建过期信息响应
	expiryInfo := map[string]interface{}{
		"remaining_seconds": int64(remainingTime.Seconds()),
		"remaining_time":    remainingTime.String(),
		"is_expiring_soon":  remainingTime.Minutes() < 5, // 5分钟内过期算即将过期
	}

	// 返回成功响应
	h.writeSuccessResponse(w, http.StatusOK, "check token expiry successful", expiryInfo)
}

// validateRefreshRequest 验证刷新令牌请求参数
func (h *RefreshHandler) validateRefreshRequest(req *model.RefreshTokenRequest) error {
	if req.RefreshToken == "" {
		return &model.ValidationError{Field: "refresh_token", Message: "refresh token cannot be empty"}
	}

	if len(req.RefreshToken) < 10 {
		return &model.ValidationError{Field: "refresh_token", Message: "refresh token format is invalid"}
	}

	return nil
}

// extractTokenFromHeader 从请求头中提取令牌
func (h *RefreshHandler) extractTokenFromHeader(r *http.Request) (string, error) {
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
		return "", &model.ValidationError{Field: "authorization", Message: "token cannot be empty"}
	}

	return token, nil
}

// getErrorStatusCode 根据错误类型获取HTTP状态码
func (h *RefreshHandler) getErrorStatusCode(err error) int {
	switch {
	case strings.Contains(err.Error(), "invalid refresh token"):
		return http.StatusUnauthorized
	case strings.Contains(err.Error(), "refresh token expired"):
		return http.StatusUnauthorized
	case strings.Contains(err.Error(), "refresh token revoked"):
		return http.StatusUnauthorized
	case strings.Contains(err.Error(), "invalid token"):
		return http.StatusUnauthorized
	case strings.Contains(err.Error(), "token expired"):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// writeSuccessResponse 写入成功响应
func (h *RefreshHandler) writeSuccessResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
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
func (h *RefreshHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
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

// GinRefreshToken Gin框架适配的刷新访问令牌接口
func (h *RefreshHandler) GinRefreshToken(c *gin.Context) {
	// 解析请求体
	var req model.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Success: false,
			Message: "invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 验证请求参数
	if err := h.validateRefreshRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Success: false,
			Message: "validation failed",
			Error:   err.Error(),
		})
		return
	}

	// 执行令牌刷新
	resp, err := h.sessionService.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Success: false,
			Message: "refresh token failed",
			Error:   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Success: true,
		Message: "refresh token successful",
		Data:    resp,
	})
}

// GinRefreshTokenFromHeader Gin框架适配的从请求头刷新访问令牌接口
func (h *RefreshHandler) GinRefreshTokenFromHeader(c *gin.Context) {
	// 从请求头中获取刷新令牌
	refreshToken, err := h.extractTokenFromGinHeader(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.APIResponse{
			Code:    http.StatusUnauthorized,
			Success: false,
			Message: "missing or invalid authorization header",
			Error:   err.Error(),
		})
		return
	}

	// 构造刷新请求
	req := &model.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	// 执行令牌刷新
	resp, err := h.sessionService.RefreshToken(c.Request.Context(), req)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Success: false,
			Message: "refresh token failed",
			Error:   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Success: true,
		Message: "refresh token successful",
		Data:    resp,
	})
}

// GinCheckTokenExpiry Gin框架适配的检查令牌过期时间接口
func (h *RefreshHandler) GinCheckTokenExpiry(c *gin.Context) {
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

	// 获取令牌剩余时间
	remainingTime, err := h.sessionService.GetTokenRemainingTime(accessToken)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Success: false,
			Message: "check token expiry failed",
			Error:   err.Error(),
		})
		return
	}

	// 构建过期信息响应
	expiryInfo := map[string]interface{}{
		"remaining_seconds": int64(remainingTime.Seconds()),
		"remaining_time":    remainingTime.String(),
		"is_expiring_soon":  remainingTime.Minutes() < 5, // 5分钟内过期算即将过期
	}

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Success: true,
		Message: "check token expiry successful",
		Data:    expiryInfo,
	})
}

// extractTokenFromGinHeader 从Gin请求头中提取令牌
func (h *RefreshHandler) extractTokenFromGinHeader(c *gin.Context) (string, error) {
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
		return "", &model.ValidationError{Field: "authorization", Message: "token cannot be empty"}
	}

	return token, nil
}
