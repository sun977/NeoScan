package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"neomaster/internal/model"
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

// Login 用户登录接口
// @Summary 用户登录
// @Description 用户通过用户名/邮箱和密码进行登录认证
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "登录请求"
// @Success 200 {object} model.APIResponse{data=model.LoginResponse} "登录成功"
// @Failure 400 {object} model.APIResponse "请求参数错误"
// @Failure 401 {object} model.APIResponse "认证失败"
// @Failure 500 {object} model.APIResponse "服务器内部错误"
// @Router /api/v1/auth/login [post]
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 解析请求体
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// 验证请求参数
	if err := h.validateLoginRequest(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "validation failed", err)
		return
	}

	// 执行登录
	resp, err := h.sessionService.Login(r.Context(), &req)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		h.writeErrorResponse(w, statusCode, "login failed", err)
		return
	}

	// 返回成功响应
	h.writeSuccessResponse(w, http.StatusOK, "login successful", resp)
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

// writeSuccessResponse 写入成功响应
func (h *LoginHandler) writeSuccessResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.WriteHeader(statusCode)
	response := model.APIResponse{
		Code:    statusCode,
		Status:  "success",
		Message: message,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// writeErrorResponse 写入错误响应
func (h *LoginHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	w.WriteHeader(statusCode)
	response := model.APIResponse{
		Code:    statusCode,
		Status:  "error",
		Message: message,
		Error:   err.Error(),
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		http.Error(w, "failed to encode error response", http.StatusInternalServerError)
	}
}

// GetLoginForm 获取登录表单页面（可选，用于Web界面）
// @Summary 获取登录表单
// @Description 返回登录表单页面
// @Tags 认证
// @Produce html
// @Success 200 {string} string "登录表单页面"
// @Router /auth/login [get]
func (h *LoginHandler) GetLoginForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	loginForm := `
<!DOCTYPE html>
<html>
<head>
    <title>NeoScan - 用户登录</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 50px; }
        .login-form { max-width: 400px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; }
        input[type="text"], input[type="password"] { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 3px; }
        button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 3px; cursor: pointer; }
        button:hover { background-color: #0056b3; }
        .error { color: red; margin-top: 10px; }
        .success { color: green; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="login-form">
        <h2>NeoScan 用户登录</h2>
        <form id="loginForm">
            <div class="form-group">
                <label for="username">用户名/邮箱:</label>
                <input type="text" id="username" name="username" required>
            </div>
            <div class="form-group">
                <label for="password">密码:</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit">登录</button>
        </form>
        <div id="message"></div>
    </div>

    <script>
        document.getElementById('loginForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const messageDiv = document.getElementById('message');
            
            try {
                const response = await fetch('/api/v1/auth/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, password })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    messageDiv.innerHTML = '<div class="success">登录成功！正在跳转...</div>';
                    // 存储令牌
                    localStorage.setItem('access_token', result.data.access_token);
                    localStorage.setItem('refresh_token', result.data.refresh_token);
                    // 跳转到主页或仪表板
                    setTimeout(() => {
                        window.location.href = '/dashboard';
                    }, 1000);
                } else {
                    messageDiv.innerHTML = '<div class="error">登录失败: ' + result.message + '</div>';
                }
            } catch (error) {
                messageDiv.innerHTML = '<div class="error">网络错误: ' + error.message + '</div>';
            }
        });
    </script>
</body>
</html>
`

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(loginForm))
}

// Gin框架处理器适配器

// GinLogin Gin登录处理器
func (h *LoginHandler) GinLogin(c *gin.Context) { // c 是 *gin.Context 类型，提供了处理 HTTP 请求的上下文
	// 解析请求体
	var req model.LoginRequest // 创建一个LoginRequest结构体变量
	if err := c.ShouldBindJSON(&req); err != nil {
		// 使用Gin的ShouldBindJSON方法解析并绑定请求体到req结构体中
		// 如果解析失败，返回400 Bad Request错误
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest, // 400
			"status":  "error",
			"message": "invalid request body",
			"error":   err.Error(),
		})
		return // 终止当前处理函数
	}

	// 验证请求参数
	if err := h.validateLoginRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"status":  "error",
			"message": "validation failed",
			"error":   err.Error(),
		})
		return
	}

	// 执行登录
	resp, err := h.sessionService.Login(c.Request.Context(), &req)
	if err != nil {
		// 根据错误类型返回不同的状态码
		statusCode := h.getErrorStatusCode(err)
		c.JSON(statusCode, gin.H{
			"code":    statusCode,
			"status":  "error",
			"message": "login failed",
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "login successful",
		"data":    resp,
	})
}

// GinGetLoginForm Gin获取登录表单处理器
func (h *LoginHandler) GinGetLoginForm(c *gin.Context) {
	loginForm := `
<!DOCTYPE html>
<html>
<head>
    <title>NeoScan - 用户登录</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 50px; }
        .login-form { max-width: 400px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; }
        input[type="text"], input[type="password"] { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 3px; }
        button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 3px; cursor: pointer; }
        button:hover { background-color: #0056b3; }
        .error { color: red; margin-top: 10px; }
        .success { color: green; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="login-form">
        <h2>NeoScan 用户登录</h2>
        <form id="loginForm">
            <div class="form-group">
                <label for="username">用户名/邮箱:</label>
                <input type="text" id="username" name="username" required>
            </div>
            <div class="form-group">
                <label for="password">密码:</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit">登录</button>
        </form>
        <div id="message"></div>
    </div>

    <script>
        document.getElementById('loginForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const messageDiv = document.getElementById('message');
            
            try {
                const response = await fetch('/api/v1/auth/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, password })
                });
                
                const result = await response.json();
                
                if (result.success) {
                    messageDiv.innerHTML = '<div class="success">登录成功！正在跳转...</div>';
                    // 存储令牌
                    localStorage.setItem('access_token', result.data.access_token);
                    localStorage.setItem('refresh_token', result.data.refresh_token);
                    // 跳转到主页或仪表板
                    setTimeout(() => {
                        window.location.href = '/dashboard';
                    }, 1000);
                } else {
                    messageDiv.innerHTML = '<div class="error">登录失败: ' + result.message + '</div>';
                }
            } catch (error) {
                messageDiv.innerHTML = '<div class="error">网络错误: ' + error.message + '</div>';
            }
        });
    </script>
</body>
</html>
`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, loginForm)
}
