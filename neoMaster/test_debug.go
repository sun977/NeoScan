package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"neomaster/internal/handler/auth"
	authPkg "neomaster/internal/pkg/auth"
	"neomaster/internal/repository/mysql"
	authService "neomaster/internal/service/auth"
)

func main() {
	// 创建一个简单的测试路由
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 创建模拟的依赖
	passwordManager := authPkg.NewPasswordManager(nil) // 使用默认配置
	userRepo := &mysql.UserRepository{} // 这里只是为了编译通过
	jwtService := &authService.JWTService{} // 这里只是为了编译通过
	sessionService := authService.NewSessionService(userRepo, passwordManager, jwtService, nil)
	
	// 创建注册处理器
	registerHandler := auth.NewRegisterHandler(sessionService)
	
	// 注册路由
	public := router.Group("/api/v1")
	public.POST("/register", registerHandler.GinRegister)
	
	// 测试数据
	data := map[string]interface{}{
		"username": "testuser",
		"email":    "test@test.com",
		"password": "password123",
	}
	
	body, _ := json.Marshal(data)
	req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer(body))
	// 不设置Content-Type
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	fmt.Printf("Status Code: %d\n", w.Code)
	fmt.Printf("Response Body: %s\n", w.Body.String())
}