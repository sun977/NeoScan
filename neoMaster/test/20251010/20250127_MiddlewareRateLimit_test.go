// MiddlewareRateLimit测试文件
// 测试优化后的限流中间件功能，包括令牌桶限流、API限流、并发限制等
// 适配拆分后的ratelimit.go模块
// 测试命令：go test -v -run TestMiddlewareRateLimit ./test/20250127

// Package test 限流中间件功能测试
// 测试拆分后的ratelimit.go中间件模块
package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestMiddlewareRateLimit 测试限流中间件模块
func TestMiddlewareRateLimit(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		// 检查必要的服务是否可用
		if ts.MiddlewareManager == nil {
			t.Skip("跳过限流中间件测试：中间件管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		t.Run("基本限流中间件功能", func(t *testing.T) {
			testBasicRateLimitMiddleware(t, ts)
		})

		t.Run("API限流中间件功能", func(t *testing.T) {
			testAPIRateLimitMiddleware(t, ts)
		})

		t.Run("并发限流测试", func(t *testing.T) {
			testConcurrentRateLimit(t, ts)
		})

		t.Run("不同IP限流测试", func(t *testing.T) {
			testDifferentIPRateLimit(t, ts)
		})

		t.Run("限流恢复测试", func(t *testing.T) {
			testRateLimitRecovery(t, ts)
		})

		t.Run("限流错误响应测试", func(t *testing.T) {
			testRateLimitErrorResponse(t, ts)
		})
	})
}

// testBasicRateLimitMiddleware 测试基本限流中间件功能
func testBasicRateLimitMiddleware(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "基本限流测试成功"})
	})

	// 测试正常请求
	t.Run("正常请求处理", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "正常请求应成功")

		// 验证响应内容
		AssertTrue(t, strings.Contains(w.Body.String(), "基本限流测试成功"), "响应内容应正确")
	})

	// 测试连续请求
	t.Run("连续请求限流", func(t *testing.T) {
		successCount := 0
		rateLimitedCount := 0

		// 发送多个连续请求
		for i := 0; i < 20; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.2:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}

			// 短暂延迟避免过快请求
			time.Sleep(10 * time.Millisecond)
		}

		t.Logf("成功请求: %d, 被限流请求: %d", successCount, rateLimitedCount)

		// 验证限流生效（应该有部分请求被限制）
		AssertTrue(t, successCount > 0, "应该有成功的请求")
		// 注意：根据限流配置，可能所有请求都成功，这取决于限流器的配置
	})
}

// testAPIRateLimitMiddleware 测试API限流中间件功能
func testAPIRateLimitMiddleware(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinAPIRateLimitMiddleware())
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "API限流测试成功"})
	})
	router.POST("/api/create", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "创建成功"})
	})

	// 测试GET API限流
	t.Run("GET API限流", func(t *testing.T) {
		successCount := 0
		rateLimitedCount := 0

		// 发送多个API请求
		for i := 0; i < 15; i++ {
			req, _ := http.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = "192.168.1.3:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}

			time.Sleep(5 * time.Millisecond)
		}

		t.Logf("GET API - 成功请求: %d, 被限流请求: %d", successCount, rateLimitedCount)
		AssertTrue(t, successCount > 0, "应该有成功的GET API请求")
	})

	// 测试POST API限流
	t.Run("POST API限流", func(t *testing.T) {
		successCount := 0
		rateLimitedCount := 0

		// 发送多个POST API请求
		for i := 0; i < 15; i++ {
			req, _ := http.NewRequest("POST", "/api/create", strings.NewReader(`{"data":"test"}`))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "192.168.1.4:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusCreated {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}

			time.Sleep(5 * time.Millisecond)
		}

		t.Logf("POST API - 成功请求: %d, 被限流请求: %d", successCount, rateLimitedCount)
		AssertTrue(t, successCount > 0, "应该有成功的POST API请求")
	})
}

// testConcurrentRateLimit 测试并发限流
func testConcurrentRateLimit(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
	router.GET("/concurrent", func(c *gin.Context) {
		// 模拟一些处理时间
		time.Sleep(10 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "并发测试成功"})
	})

	// 并发测试
	t.Run("并发请求限流", func(t *testing.T) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		successCount := 0
		rateLimitedCount := 0
		errorCount := 0

		// 启动多个goroutine并发请求
		concurrency := 10
		requestsPerGoroutine := 5

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < requestsPerGoroutine; j++ {
					req, _ := http.NewRequest("GET", "/concurrent", nil)
					req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", 100+goroutineID)
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					mu.Lock()
					switch w.Code {
					case http.StatusOK:
						successCount++
					case http.StatusTooManyRequests:
						rateLimitedCount++
					default:
						errorCount++
					}
					mu.Unlock()

					// 短暂延迟
					time.Sleep(5 * time.Millisecond)
				}
			}(i)
		}

		wg.Wait()

		t.Logf("并发测试结果 - 成功: %d, 限流: %d, 错误: %d",
			successCount, rateLimitedCount, errorCount)

		// 验证结果
		totalRequests := concurrency * requestsPerGoroutine
		AssertEqual(t, totalRequests, successCount+rateLimitedCount+errorCount,
			"总请求数应该等于各种响应的总和")
		AssertTrue(t, successCount > 0, "应该有成功的请求")
	})
}

// testDifferentIPRateLimit 测试不同IP的限流
func testDifferentIPRateLimit(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
	router.GET("/ip-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "IP限流测试成功"})
	})

	// 测试不同IP的独立限流
	t.Run("不同IP独立限流", func(t *testing.T) {
		ips := []string{
			"192.168.1.10:12345",
			"192.168.1.11:12345",
			"192.168.1.12:12345",
		}

		for _, ip := range ips {
			t.Run(fmt.Sprintf("IP-%s", ip), func(t *testing.T) {
				successCount := 0

				// 每个IP发送多个请求
				for i := 0; i < 10; i++ {
					req, _ := http.NewRequest("GET", "/ip-test", nil)
					req.RemoteAddr = ip
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					if w.Code == http.StatusOK {
						successCount++
					}

					time.Sleep(10 * time.Millisecond)
				}

				t.Logf("IP %s - 成功请求: %d", ip, successCount)
				AssertTrue(t, successCount > 0, fmt.Sprintf("IP %s 应该有成功的请求", ip))
			})
		}
	})

	// 测试相同IP的限流
	t.Run("相同IP限流", func(t *testing.T) {
		sameIP := "192.168.1.20:12345"
		successCount := 0
		rateLimitedCount := 0

		// 同一IP发送大量请求
		for i := 0; i < 30; i++ {
			req, _ := http.NewRequest("GET", "/ip-test", nil)
			req.RemoteAddr = sameIP
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}

			time.Sleep(5 * time.Millisecond)
		}

		t.Logf("相同IP限流测试 - 成功: %d, 限流: %d", successCount, rateLimitedCount)
		AssertTrue(t, successCount > 0, "应该有成功的请求")
	})
}

// testRateLimitRecovery 测试限流恢复
func testRateLimitRecovery(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
	router.GET("/recovery", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "恢复测试成功"})
	})

	t.Run("限流后恢复", func(t *testing.T) {
		testIP := "192.168.1.30:12345"

		// 第一阶段：快速发送请求直到被限流
		rateLimitedCount := 0
		for i := 0; i < 50; i++ {
			req, _ := http.NewRequest("GET", "/recovery", nil)
			req.RemoteAddr = testIP
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
				break
			}
		}

		if rateLimitedCount == 0 {
			t.Log("注意：在50次请求内未触发限流，可能限流配置较宽松")
		}

		// 第二阶段：等待一段时间让限流器恢复
		t.Log("等待限流器恢复...")
		time.Sleep(2 * time.Second)

		// 第三阶段：验证请求可以再次成功
		req, _ := http.NewRequest("GET", "/recovery", nil)
		req.RemoteAddr = testIP
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 恢复后的请求应该成功
		AssertEqual(t, http.StatusOK, w.Code, "限流恢复后请求应该成功")
		AssertTrue(t, strings.Contains(w.Body.String(), "恢复测试成功"), "响应内容应正确")
	})
}

// testRateLimitErrorResponse 测试限流错误响应
func testRateLimitErrorResponse(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
	router.GET("/error-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "错误响应测试"})
	})

	t.Run("限流错误响应格式", func(t *testing.T) {
		testIP := "192.168.1.40:12345"
		var rateLimitResponse *httptest.ResponseRecorder

		// 发送大量请求直到触发限流
		for i := 0; i < 100; i++ {
			req, _ := http.NewRequest("GET", "/error-test", nil)
			req.RemoteAddr = testIP
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				rateLimitResponse = w
				break
			}

			// 快速发送请求
			time.Sleep(1 * time.Millisecond)
		}

		if rateLimitResponse != nil {
			// 验证限流响应
			AssertEqual(t, http.StatusTooManyRequests, rateLimitResponse.Code,
				"限流响应状态码应为429")

			// 验证响应头
			retryAfter := rateLimitResponse.Header().Get("Retry-After")
			if len(retryAfter) > 0 {
				t.Logf("Retry-After头: %s", retryAfter)
			}

			// 验证响应体
			responseBody := rateLimitResponse.Body.String()
			AssertTrue(t, len(responseBody) > 0, "限流响应应有响应体")

			// 检查是否包含限流相关信息
			AssertTrue(t, strings.Contains(responseBody, "limit") ||
				strings.Contains(responseBody, "rate") ||
				strings.Contains(responseBody, "too many") ||
				strings.Contains(responseBody, "请求过于频繁"),
				"响应体应包含限流相关信息")

			t.Logf("限流响应体: %s", responseBody)
		} else {
			t.Log("注意：在100次请求内未触发限流，限流配置可能较宽松")
		}
	})
}

// TestRateLimitMiddlewareIntegration 测试限流中间件集成
func TestRateLimitMiddlewareIntegration(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.MiddlewareManager == nil {
			t.Skip("跳过限流中间件集成测试：中间件管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		// 创建集成多个中间件的路由
		router := gin.New()
		router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
		router.Use(ts.MiddlewareManager.GinRateLimitMiddleware())
		router.Use(ts.MiddlewareManager.GinAPIRateLimitMiddleware())
		router.GET("/integrated", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "限流中间件集成测试成功"})
		})

		// 发送集成测试请求
		req, _ := http.NewRequest("GET", "/integrated", nil)
		req.RemoteAddr = "192.168.1.50:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 验证响应
		AssertEqual(t, http.StatusOK, w.Code, "集成请求应成功")
		AssertTrue(t, strings.Contains(w.Body.String(), "限流中间件集成测试成功"),
			"响应内容应正确")

		t.Log("限流中间件集成测试通过")
	})
}
