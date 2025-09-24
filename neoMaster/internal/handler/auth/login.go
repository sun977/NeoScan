package auth

import (
	"net/http"
	"strings"

	"neomaster/internal/model"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// LoginHandler 登录接口处理器
type LoginHandler struct {
	sessionService *auth.SessionService
}

// NewLoginHandler 创建登录处理器实例
func NewLoginHandler(sessionService *auth.SessionService) *LoginHandler {
	return &LoginHandler{
		sessionService: sessionService,
	}
}

// validateLoginRequest 验证登录请求参数
func (h *LoginHandler) validateLoginRequest(req *model.LoginRequest) error {
	if req.Username == "" {
		return &model.ValidationError{Field: "username", Message: "username cannot be empty"}
	}

	if req.Password == "" {
		return &model.ValidationError{Field: "password", Message: "password cannot be empty"}
	}

	if len(req.Username) < 3 {
		return &model.ValidationError{Field: "username", Message: "username must be at least 3 characters"}
	}

	if len(req.Password) < 6 {
		return &model.ValidationError{Field: "password", Message: "password must be at least 6 characters"}
	}

	return nil
}

// getErrorStatusCode 根据错误类型获取HTTP状态码
func (h *LoginHandler) getErrorStatusCode(err error) int {
	errorMsg := err.Error()
	switch {
	case strings.Contains(errorMsg, "invalid username or password"):
		return http.StatusUnauthorized
	case strings.Contains(errorMsg, "user account is inactive"):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// HTTP处理器方法

// Login 登录处理器
func (h *LoginHandler) Login(c *gin.Context) { // c 是 *gin.Context 类型，提供了处理 HTTP 请求的上下文
	// 规范化客户端IP与User-Agent（在全流程统一使用）
	// 一步到位获取客户端IP，从上下文中
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")

	// 解析请求体
	var req model.LoginRequest // 创建一个LoginRequest结构体变量
	if err := c.ShouldBindJSON(&req); err != nil {
		// 使用Gin的ShouldBindJSON方法解析并绑定请求体到req结构体中
		// 如果解析失败，返回400 Bad Request错误
		// 记录错误日志
		logger.LogError(err, XRequestID, 0, clientIP, "/api/v1/auth/login", "POST", map[string]interface{}{
			"operation":  "login",
			"option":     "ShouldBindJSON",
			"func_name":  "handler.auth.login.Login",
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest, // 400
			Status:  "failed",
			Message: "invalid request body",
			Error:   err.Error(),
		})
		return // 终止当前处理函数
	}

	// 验证请求参数
	if err := h.validateLoginRequest(&req); err != nil {
		// 记录参数验证失败日志
		logger.LogError(err, XRequestID, 0, clientIP, "/api/v1/auth/login", "POST", map[string]interface{}{
			"operation":  "login",
			"option":     "validateLoginRequest",
			"func_name":  "handler.auth.login.Login",
			"username":   req.Username,
			"client_ip":  clientIP,
			"user_agent": userAgent,
			"request_id": XRequestID,
			"timestamp":  logger.NowFormatted(),
		})
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    http.StatusBadRequest, // 400
			Status:  "failed",
			Message: "validation failed",
			Error:   err.Error(),
		})
		return
	}

	// 执行登录
	resp, err := h.sessionService.Login(c.Request.Context(), &req, clientIP, userAgent)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		// 记录登录失败的错误日志
		logger.LogError(err, XRequestID, 0, clientIP, "/api/v1/auth/login", "POST", map[string]interface{}{
			"operation":   "login",
			"option":      "sessionService.Login",
			"func_name":   "handler.auth.login.Login",
			"username":    req.Username,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"status_code": statusCode,
			"request_id":  XRequestID,
			"timestamp":   logger.NowFormatted(),
		})
		c.JSON(statusCode, model.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "login failed",
			Error:   err.Error(),
		})
		return
	}

	// 记录登录成功的业务日志
	logger.LogBusinessOperation("user_login", uint(resp.User.ID), req.Username, clientIP, XRequestID, "success", "user login success", map[string]interface{}{
		"operation":  "user_login",
		"option":     "user_login:success",
		"func_name":  "handler.auth.login.Login",
		"user_id":    resp.User.ID,
		"username":   req.Username,
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"request_id": XRequestID,
		"timestamp":  logger.NowFormatted(),
	})

	// 返回成功响应
	c.JSON(http.StatusOK, model.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "login successful",
		Data:    resp,
	})
}

// GetLoginForm 获取登录表单处理器
// func (h *LoginHandler) GetLoginForm(c *gin.Context) {
// 	loginForm := `
// <!DOCTYPE html>
// <html>
// <head>
//     <title>NeoScan - 用户登录</title>
//     <meta charset="utf-8">
//     <style>
//         body { font-family: Arial, sans-serif; margin: 50px; }
//         .login-form { max-width: 400px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
//         .form-group { margin-bottom: 15px; }
//         label { display: block; margin-bottom: 5px; }
//         input[type="text"], input[type="password"] { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 3px; }
//         button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 3px; cursor: pointer; }
//         button:hover { background-color: #0056b3; }
//         .error { color: red; margin-top: 10px; }
//         .success { color: green; margin-top: 10px; }
//     </style>
// </head>
// <body>
//     <div class="login-form">
//         <h2>NeoScan 用户登录</h2>
//         <form id="loginForm">
//             <div class="form-group">
//                 <label for="username">用户名/邮箱:</label>
//                 <input type="text" id="username" name="username" required>
//             </div>
//             <div class="form-group">
//                 <label for="password">密码:</label>
//                 <input type="password" id="password" name="password" required>
//             </div>
//             <button type="submit">登录</button>
//         </form>
//         <div id="message"></div>
//     </div>

//     <script>
//         document.getElementById('loginForm').addEventListener('submit', async function(e) {
//             e.preventDefault();

//             const username = document.getElementById('username').value;
//             const password = document.getElementById('password').value;
//             const messageDiv = document.getElementById('message');

//             try {
//                 const response = await fetch('/api/v1/auth/login', {
//                     method: 'POST',
//                     headers: {
//                         'Content-Type': 'application/json',
//                     },
//                     body: JSON.stringify({ username, password })
//                 });

//                 const result = await response.json();

//                 if (result.success) {
//                     messageDiv.innerHTML = '<div class="success">登录成功！正在跳转...</div>';
//                     // 存储令牌
//                     localStorage.setItem('access_token', result.data.access_token);
//                     localStorage.setItem('refresh_token', result.data.refresh_token);
//                     // 跳转到主页或仪表板
//                     setTimeout(() => {
//                         window.location.href = '/dashboard';
//                     }, 1000);
//                 } else {
//                     messageDiv.innerHTML = '<div class="error">登录失败: ' + result.message + '</div>';
//                 }
//             } catch (error) {
//                 messageDiv.innerHTML = '<div class="error">网络错误: ' + error.message + '</div>';
//             }
//         });
//     </script>
// </body>
// </html>
// `

// 	c.Header("Content-Type", "text/html; charset=utf-8")
// 	c.String(http.StatusOK, loginForm)
// }
