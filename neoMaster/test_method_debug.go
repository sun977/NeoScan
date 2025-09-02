package main

import (
	"fmt"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func main() {
	// 创建一个简单的测试路由
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 注册一个简单的POST路由
	router.POST("/api/v1/register", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	
	// 测试DELETE方法（不支持的方法）
	req := httptest.NewRequest("DELETE", "/api/v1/register", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	fmt.Printf("Status Code: %d\n", w.Code)
	fmt.Printf("Response Body: %s\n", w.Body.String())
}