package auth

import (
	"errors"
	"neomaster/internal/model/system"
	"net/http"

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
func (h *RegisterHandler) validateRegisterRequest(req *system.RegisterRequest) error {
	if req.Username == "" {
		return system.ErrInvalidUsername
	}
	if req.Email == "" {
		return system.ErrInvalidEmail
	}
	if req.Password == "" {
		return system.ErrInvalidPassword
	}
	return nil
}

// getErrorStatusCode 根据错误类型返回对应的HTTP状态码
func (h *RegisterHandler) getErrorStatusCode(err error) int {
	switch err {
	case system.ErrUserAlreadyExists:
		return http.StatusConflict
	case system.ErrUsernameAlreadyExists:
		return http.StatusConflict
	case system.ErrEmailAlreadyExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// Register 注册处理器
func (h *RegisterHandler) Register(c *gin.Context) {
	// 规范化客户端IP与User-Agent（在全流程统一使用）
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	urlPath := c.Request.URL.String()

	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")
	XRequestID := c.GetHeader("X-Request-ID")
	if contentType == "" {
		// 记录Content-Type缺失错误日志
		logger.LogBusinessError(errors.New("missing Content-Type header"), XRequestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "register",
			"option":     "contentTypeCheck",
			"func_name":  "handler.auth.register.Register",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Content-Type header is required",
			Error:   "missing Content-Type header",
		})
		return
	}

	// 解析请求体
	var req system.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 记录请求体解析失败错误日志
		logger.LogBusinessError(err, XRequestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":    "register",
			"option":       "ShouldBindJSON",
			"func_name":    "handler.auth.register.Register",
			"client_ip":    clientIP,
			"user_agent":   userAgent,
			"request_id":   XRequestID,
			"content_type": contentType,
			"timestamp":    logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 验证请求参数
	if err := h.validateRegisterRequest(&req); err != nil {
		// 记录参数验证失败错误日志
		logger.LogBusinessError(err, XRequestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":  "register",
			"option":     "validateRegisterRequest",
			"func_name":  "handler.auth.register.Register",
			"username":   req.Username,
			"email":      req.Email,
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
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
		logger.LogBusinessError(err, XRequestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":   "register",
			"option":      "userService.Register",
			"func_name":   "handler.auth.register.Register",
			"username":    req.Username,
			"email":       req.Email,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"status_code": statusCode,
			"request_id":  XRequestID,
			// "Error":       err.Error(),
			"timestamp": logger.NowFormatted(),
		})
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "registration failed",
			Error:   err.Error(),
		})
		return
	}

	// 给新注册的用户分配角色 普通用户 role_id = 2 【注册服务的响应体里面有 user_id 】
	err = h.userService.AssignRoleToUser(c.Request.Context(), uint(response.User.ID), 2)
	if err != nil {
		// 记录角色分配失败错误日志
		logger.LogBusinessError(err, XRequestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation":   "register",
			"option":      "userService.AssignRoleToUser",
			"func_name":   "handler.auth.register.Register",
			"username":    req.Username,
			"email":       req.Email,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"request_id":  XRequestID,
			"timestamp":   logger.NowFormatted(),
			"role_id":     2,
			"role_name":   "user",
			"assign_type": "default",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "role assignment failed",
			Error:   err.Error(),
		})
	}

	// 记录注册成功业务日志
	logger.LogBusinessOperation("user_register", uint(response.User.ID), req.Username, clientIP, XRequestID, "success", "user register success", map[string]interface{}{
		"operation":  "register",
		"option":     "user_register:success",
		"func_name":  "handler.auth.register.Register",
		"user_id":    response.User.ID,
		"username":   req.Username,
		"email":      req.Email,
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": XRequestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated, // 201 Created 表示资源创建成功
		Status:  "success",
		Message: "registration successful",
		Data:    response,
	})
}
