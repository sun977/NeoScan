package auth

import (
	"net/http"

	"neomaster/internal/model"
	"neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// RegisterHandler 注册接口处理器
type RegisterHandler struct {
	sessionService *auth.SessionService
}

// NewRegisterHandler 创建注册处理器实例
func NewRegisterHandler(sessionService *auth.SessionService) *RegisterHandler {
	return &RegisterHandler{
		sessionService: sessionService,
	}
}

// validateRegisterRequest 验证注册请求参数
func (h *RegisterHandler) validateRegisterRequest(req *model.RegisterRequest) error {
	if req.Username == "" {
		return model.ErrInvalidUsername
	}
	if req.Email == "" {
		return model.ErrInvalidEmail
	}
	if req.Password == "" {
		return model.ErrInvalidPassword
	}
	return nil
}

// getErrorStatusCode 根据错误类型返回对应的HTTP状态码
func (h *RegisterHandler) getErrorStatusCode(err error) int {
	switch err {
	case model.ErrUserAlreadyExists:
		return http.StatusConflict
	case model.ErrUsernameAlreadyExists:
		return http.StatusConflict
	case model.ErrEmailAlreadyExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// Register 注册处理器
func (h *RegisterHandler) Register(c *gin.Context) {
	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "Content-Type header is required",
			Error:   "missing Content-Type header",
		})
		return
	}

	// 解析请求体
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 验证请求参数
	if err := h.validateRegisterRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "validation failed",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层进行注册
	response, err := h.sessionService.Register(c.Request.Context(), &req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Status:  "error",
			Message: "registration failed",
			Error:   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusCreated, model.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "registration successful",
		Data:    response,
	})
}
