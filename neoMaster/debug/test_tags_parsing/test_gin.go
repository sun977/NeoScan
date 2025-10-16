package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 模拟GetAgentList的参数解析逻辑
	router.GET("/api/v1/agent", func(c *gin.Context) {
		// 标签过滤参数处理 - 支持逗号分隔的标签值
		var tags []string
		tagsArray := c.QueryArray("tags")
		if len(tagsArray) > 1 {
			// 处理多个tags参数: tags=2&tags=7
			tags = tagsArray
		} else if len(tagsArray) == 1 && strings.Contains(tagsArray[0], ",") {
			// 处理逗号分隔的标签值: tags=2,7
			tags = strings.Split(tagsArray[0], ",")
			// 去除空白字符
			for i, tag := range tags {
				tags[i] = strings.TrimSpace(tag)
			}
		} else if len(tagsArray) == 1 {
			// 单个标签: tags=2
			tags = tagsArray
		}

		c.JSON(http.StatusOK, gin.H{
			"tags_array":  tagsArray,
			"parsed_tags": tags,
			"tags_count":  len(tags),
		})
	})

	// 测试逗号分隔格式
	fmt.Println("=== 测试逗号分隔格式: tags=2,7 ===")
	req1, _ := http.NewRequest("GET", "/api/v1/agent?page=1&page_size=10&status=online&tags=2,7", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	fmt.Printf("Response: %s\n", w1.Body.String())

	// 测试多参数格式
	fmt.Println("\n=== 测试多参数格式: tags=2&tags=7 ===")
	req2, _ := http.NewRequest("GET", "/api/v1/agent?page=1&page_size=10&status=online&tags=2&tags=7", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	fmt.Printf("Response: %s\n", w2.Body.String())

	// 测试空格处理
	fmt.Println("\n=== 测试空格处理: tags=2, 7, 8 ===")
	req3, _ := http.NewRequest("GET", "/api/v1/agent?tags=2,%207,%208", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	fmt.Printf("Response: %s\n", w3.Body.String())
}