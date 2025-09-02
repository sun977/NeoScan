package auth

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"neomaster/internal/model"
	"neomaster/internal/service/auth"
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

// Register 用户注册接口
// @Summary 用户注册
// @Description 用户通过用户名、邮箱和密码进行注册
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body model.RegisterRequest true "注册请求"
// @Success 201 {object} model.APIResponse{data=model.RegisterResponse} "注册成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 409 {object} model.APIResponse "用户已存在"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/auth/register [post]
func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 解析请求体
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// 验证请求参数
	if err := h.validateRegisterRequest(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "validation failed", err)
		return
	}

	// 调用服务层进行注册
	response, err := h.sessionService.Register(r.Context(), &req)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)
		h.writeErrorResponse(w, statusCode, "registration failed", err)
		return
	}

	// 返回成功响应
	h.writeSuccessResponse(w, http.StatusCreated, "registration successful", response)
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
	default:
		return http.StatusInternalServerError
	}
}

// writeSuccessResponse 写入成功响应
func (h *RegisterHandler) writeSuccessResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	response := model.APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse 写入错误响应
func (h *RegisterHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	response := model.APIResponse{
		Status:  "error",
		Message: message,
		Error:   err.Error(),
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// GinRegister Gin框架的注册处理器
func (h *RegisterHandler) GinRegister(c *gin.Context) {
	// 检查Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
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
			Status:  "error",
			Message: "invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// 验证请求参数
	if err := h.validateRegisterRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
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
			Status:  "error",
			Message: "registration failed",
			Error:   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusCreated, model.APIResponse{
		Status:  "success",
		Message: "registration successful",
		Data:    response,
	})
}