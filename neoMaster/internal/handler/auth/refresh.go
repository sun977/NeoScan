package auth

import (
	"neomaster/internal/model/system"
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

// validateRefreshRequest 验证刷新令牌请求参数
func (h *RefreshHandler) validateRefreshRequest(req *system.RefreshTokenRequest) error {
	if req.RefreshToken == "" {
		return &model.ValidationError{Field: "refresh_token", Message: "refresh token cannot be empty"}
	}

	if len(req.RefreshToken) < 10 {
		return &model.ValidationError{Field: "refresh_token", Message: "refresh token format is invalid"}
	}

	return nil
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

// RefreshToken 刷新访问令牌接口
func (h *RefreshHandler) RefreshToken(c *gin.Context) {
	// 解析请求体
	var req system.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 验证请求参数
	if err := h.validateRefreshRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
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
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "error",
			Message: "refresh token failed",
			Error:   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "refresh token successful",
		Data:    resp,
	})
}

// RefreshTokenFromHeader 从请求头刷新访问令牌接口
func (h *RefreshHandler) RefreshTokenFromHeader(c *gin.Context) {
	// 从请求头中获取刷新令牌
	refreshToken, err := h.extractTokenFromHeader(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, system.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "missing or invalid authorization header",
			Error:   err.Error(),
		})
		return
	}

	// 构造刷新请求
	req := &system.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	// 执行令牌刷新
	resp, err := h.sessionService.RefreshToken(c.Request.Context(), req)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "error",
			Message: "refresh token failed",
			Error:   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "refresh token successful",
		Data:    resp,
	})
}

// CheckTokenExpiry 检查令牌过期时间接口
func (h *RefreshHandler) CheckTokenExpiry(c *gin.Context) {
	// 从请求头中获取访问令牌
	accessToken, err := h.extractTokenFromHeader(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, system.APIResponse{
			Code:    http.StatusUnauthorized,
			Status:  "error",
			Message: "missing or invalid authorization header",
			Error:   err.Error(),
		})
		return
	}

	// 获取令牌剩余时间
	remainingTime, err := h.sessionService.GetTokenRemainingTime(accessToken)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "error",
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
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "check token expiry successful",
		Data:    expiryInfo,
	})
}

// extractTokenFromHeader 从请求头中提取令牌
func (h *RefreshHandler) extractTokenFromHeader(c *gin.Context) (string, error) {
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
