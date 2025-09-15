package auth

import (
	"errors"
	"net/http"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// RegisterHandler 注册接口处理器
type RegisterHandler struct {
	userService *auth.UserService
}

// NewRegisterHandler 创建注册处理器实例
func NewRegisterHandler(userService *auth.UserService) *RegisterHandler {
	return &RegisterHandler{
		userService: userService,
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
	// 规范化客户端IP与User-Agent（在全流程统一使用）
	clientIPRaw := c.GetHeader("X-Forwarded-For")
	if clientIPRaw == "" {
		clientIPRaw = c.GetHeader("X-Real-IP")
	}
	if clientIPRaw == "" {
		clientIPRaw = c.ClientIP()
	}
	clientIP := utils.NormalizeIP(clientIPRaw)
	userAgent := c.GetHeader("User-Agent")

	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		// 记录Content-Type缺失错误日志
		logger.LogError(errors.New("missing Content-Type header"), "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation":  "register",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
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
		// 记录请求体解析失败错误日志
		logger.LogError(err, "", 0, "", "user_register", "POST", map[string]interface{}{
			"operation":    "register",
			"client_ip":    c.ClientIP(),
			"user_agent":   c.GetHeader("User-Agent"),
			"request_id":   c.GetHeader("X-Request-ID"),
			"content_type": contentType,
			"timestamp":    logger.NowFormatted(),
		})
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
		// 记录参数验证失败错误日志
		logger.LogError(err, "", 0, req.Username, "user_register", "POST", map[string]interface{}{
			"operation":  "register",
			"email":      req.Email,
			"client_ip":  c.ClientIP(),
			"user_agent": c.GetHeader("User-Agent"),
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "error",
			Message: "validation failed",
			Error:   err.Error(),
		})
		return
	}

	// 调用服务层进行注册
	response, err := h.userService.Register(c.Request.Context(), &req, clientIP)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		// 记录注册失败错误日志
		logger.LogError(err, "", 0, req.Username, "user_register", "POST", map[string]interface{}{
			"operation":   "register",
			"email":       req.Email,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"status_code": statusCode,
			"request_id":  c.GetHeader("X-Request-ID"),
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Status:  "error",
			Message: "registration failed",
			Error:   err.Error(),
		})
		return
	}

	// 给新注册的用户分配角色 普通用户 role_id = 2 【注册服务的响应体里面有 user_id 】
	err = h.userService.AssignRoleToUser(c.Request.Context(), uint(response.User.ID), 2)
	if err != nil {
		// 记录角色分配失败错误日志
		logger.LogError(err, "", 0, req.Username, "user_register_assign_role", "POST", map[string]interface{}{
			"operation":   "register",
			"email":       req.Email,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  c.GetHeader("X-Request-ID"),
			"timestamp":   logger.NowFormatted(),
			"role_id":     2,
			"role_name":   "普通用户",
			"assign_type": "default",
		})
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "error",
			Message: "role assignment failed",
			Error:   err.Error(),
		})
	}

	// 记录注册成功业务日志
	logger.LogBusinessOperation("user_register", uint(response.User.ID), req.Username, "", "", "success", "用户注册成功", map[string]interface{}{
		"operation":  "register",
		"email":      req.Email,
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": c.GetHeader("X-Request-ID"),
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusCreated, model.APIResponse{
		Code:    http.StatusCreated, // 201 Created 表示资源创建成功
		Status:  "success",
		Message: "registration successful",
		Data:    response,
	})
}
