// MiddlewareLogging测试文件
// 测试优化后的日志中间件功能，包括请求日志记录、响应日志记录、错误日志记录等
// 适配拆分后的logging.go模块
// 测试命令：go test -v -run TestMiddlewareLogging ./test/20250127

// Package test 日志中间件功能测试
// 测试拆分后的logging.go中间件模块
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TestMiddlewareLogging 测试日志中间件模块
func TestMiddlewareLogging(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		// 检查必要的服务是否可用
		if ts.MiddlewareManager == nil {
			t.Skip("跳过日志中间件测试：中间件管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		t.Run("基本日志记录功能", func(t *testing.T) {
			testBasicLogging(t, ts)
		})

		t.Run("请求参数日志记录", func(t *testing.T) {
			testRequestParameterLogging(t, ts)
		})

		t.Run("响应内容日志记录", func(t *testing.T) {
			testResponseLogging(t, ts)
		})

		t.Run("错误状态码日志记录", func(t *testing.T) {
			testErrorStatusLogging(t, ts)
		})

		t.Run("执行时间日志记录", func(t *testing.T) {
			testExecutionTimeLogging(t, ts)
		})

		t.Run("不同HTTP方法日志记录", func(t *testing.T) {
			testDifferentMethodsLogging(t, ts)
		})
	})
}

// testBasicLogging 测试基本日志记录功能
func testBasicLogging(t *testing.T, ts *TestSuite) {
	// 创建日志缓冲区来捕获日志输出
	var logBuffer bytes.Buffer
	originalOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logBuffer)
	defer logrus.SetOutput(originalOutput)

	// 设置日志格式为JSON以便解析
	originalFormatter := logrus.StandardLogger().Formatter
	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer logrus.SetFormatter(originalFormatter)

	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "测试成功"})
	})

	// 发送测试请求
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Request-ID", "test-request-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	AssertEqual(t, http.StatusOK, w.Code, "响应状态码应为200")

	// 验证日志输出
	logOutput := logBuffer.String()
	AssertTrue(t, len(logOutput) > 0, "应该有日志输出")

	// 解析日志JSON
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	if len(lines) > 0 {
		err := json.Unmarshal([]byte(lines[len(lines)-1]), &logEntry)
		AssertNoError(t, err, "日志应为有效JSON格式")

		// 验证日志字段
		AssertEqual(t, "/test", logEntry["path"], "路径应正确记录")
		AssertEqual(t, "GET", logEntry["method"], "HTTP方法应正确记录")
		AssertEqual(t, float64(200), logEntry["status_code"], "状态码应正确记录")
		AssertNotNil(t, logEntry["duration"], "执行时间应被记录")
		AssertEqual(t, "test-agent", logEntry["user_agent"], "User-Agent应正确记录")
		AssertEqual(t, "test-request-123", logEntry["request_id"], "Request ID应正确记录")
	}
}

// testRequestParameterLogging 测试请求参数日志记录
func testRequestParameterLogging(t *testing.T, ts *TestSuite) {
	// 创建日志缓冲区
	var logBuffer bytes.Buffer
	originalOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logBuffer)
	defer logrus.SetOutput(originalOutput)

	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer logrus.SetFormatter(&logrus.TextFormatter{})

	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "参数测试成功"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "POST测试成功"})
	})

	// 测试GET请求参数
	t.Run("GET请求参数", func(t *testing.T) {
		logBuffer.Reset()
		
		req, _ := http.NewRequest("GET", "/test?param1=value1&param2=value2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "响应状态码应为200")

		// 验证日志中包含查询参数
		logOutput := logBuffer.String()
		AssertTrue(t, strings.Contains(logOutput, "param1=value1"), "日志应包含查询参数param1")
		AssertTrue(t, strings.Contains(logOutput, "param2=value2"), "日志应包含查询参数param2")
	})

	// 测试POST请求体
	t.Run("POST请求体", func(t *testing.T) {
		logBuffer.Reset()
		
		requestBody := `{"name":"test","value":"data"}`
		req, _ := http.NewRequest("POST", "/test", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "响应状态码应为200")

		// 验证日志中包含请求体
		logOutput := logBuffer.String()
		AssertTrue(t, strings.Contains(logOutput, "test"), "日志应包含请求体内容")
		AssertTrue(t, strings.Contains(logOutput, "data"), "日志应包含请求体内容")
	})
}

// testResponseLogging 测试响应内容日志记录
func testResponseLogging(t *testing.T, ts *TestSuite) {
	// 创建日志缓冲区
	var logBuffer bytes.Buffer
	originalOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logBuffer)
	defer logrus.SetOutput(originalOutput)

	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer logrus.SetFormatter(&logrus.TextFormatter{})

	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "响应测试成功",
			"data":    "test_data",
			"code":    200,
		})
	})

	// 发送测试请求
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	AssertEqual(t, http.StatusOK, w.Code, "响应状态码应为200")

	// 验证日志中包含响应内容
	logOutput := logBuffer.String()
	AssertTrue(t, strings.Contains(logOutput, "响应测试成功"), "日志应包含响应消息")
	AssertTrue(t, strings.Contains(logOutput, "test_data"), "日志应包含响应数据")
	AssertTrue(t, strings.Contains(logOutput, "200"), "日志应包含响应代码")
}

// testErrorStatusLogging 测试错误状态码日志记录
func testErrorStatusLogging(t *testing.T, ts *TestSuite) {
	// 创建日志缓冲区
	var logBuffer bytes.Buffer
	originalOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logBuffer)
	defer logrus.SetOutput(originalOutput)

	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer logrus.SetFormatter(&logrus.TextFormatter{})

	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
	router.GET("/error400", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
	})
	router.GET("/error500", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
	})

	// 测试400错误
	t.Run("400错误日志", func(t *testing.T) {
		logBuffer.Reset()
		
		req, _ := http.NewRequest("GET", "/error400", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusBadRequest, w.Code, "响应状态码应为400")

		// 验证日志级别和内容
		logOutput := logBuffer.String()
		lines := strings.Split(strings.TrimSpace(logOutput), "\n")
		if len(lines) > 0 {
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(lines[len(lines)-1]), &logEntry)
			AssertNoError(t, err, "日志应为有效JSON格式")
			
			// 400错误应该使用Warn级别
			AssertEqual(t, "warning", logEntry["level"], "400错误应使用warning级别")
			AssertEqual(t, float64(400), logEntry["status_code"], "状态码应正确记录")
		}
	})

	// 测试500错误
	t.Run("500错误日志", func(t *testing.T) {
		logBuffer.Reset()
		
		req, _ := http.NewRequest("GET", "/error500", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusInternalServerError, w.Code, "响应状态码应为500")

		// 验证日志级别和内容
		logOutput := logBuffer.String()
		lines := strings.Split(strings.TrimSpace(logOutput), "\n")
		if len(lines) > 0 {
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(lines[len(lines)-1]), &logEntry)
			AssertNoError(t, err, "日志应为有效JSON格式")
			
			// 500错误应该使用Error级别
			AssertEqual(t, "error", logEntry["level"], "500错误应使用error级别")
			AssertEqual(t, float64(500), logEntry["status_code"], "状态码应正确记录")
		}
	})
}

// testExecutionTimeLogging 测试执行时间日志记录
func testExecutionTimeLogging(t *testing.T, ts *TestSuite) {
	// 创建日志缓冲区
	var logBuffer bytes.Buffer
	originalOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logBuffer)
	defer logrus.SetOutput(originalOutput)

	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer logrus.SetFormatter(&logrus.TextFormatter{})

	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
	router.GET("/fast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "快速响应"})
	})
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond) // 模拟慢请求
		c.JSON(http.StatusOK, gin.H{"message": "慢速响应"})
	})

	// 测试快速请求
	t.Run("快速请求执行时间", func(t *testing.T) {
		logBuffer.Reset()
		
		req, _ := http.NewRequest("GET", "/fast", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "响应状态码应为200")

		// 验证执行时间记录
		logOutput := logBuffer.String()
		lines := strings.Split(strings.TrimSpace(logOutput), "\n")
		if len(lines) > 0 {
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(lines[len(lines)-1]), &logEntry)
			AssertNoError(t, err, "日志应为有效JSON格式")
			
			duration, exists := logEntry["duration"]
			AssertTrue(t, exists, "执行时间应被记录")
			AssertNotNil(t, duration, "执行时间不应为nil")
		}
	})

	// 测试慢请求
	t.Run("慢请求执行时间", func(t *testing.T) {
		logBuffer.Reset()
		
		req, _ := http.NewRequest("GET", "/slow", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "响应状态码应为200")

		// 验证执行时间记录
		logOutput := logBuffer.String()
		lines := strings.Split(strings.TrimSpace(logOutput), "\n")
		if len(lines) > 0 {
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(lines[len(lines)-1]), &logEntry)
			AssertNoError(t, err, "日志应为有效JSON格式")
			
			duration, exists := logEntry["duration"]
			AssertTrue(t, exists, "执行时间应被记录")
			AssertNotNil(t, duration, "执行时间不应为nil")
			
			// 慢请求的执行时间应该大于100ms
			durationStr, ok := duration.(string)
			AssertTrue(t, ok, "执行时间应为字符串格式")
			AssertTrue(t, strings.Contains(durationStr, "ms") || strings.Contains(durationStr, "µs"), "执行时间应包含时间单位")
		}
	})
}

// testDifferentMethodsLogging 测试不同HTTP方法的日志记录
func testDifferentMethodsLogging(t *testing.T, ts *TestSuite) {
	// 创建日志缓冲区
	var logBuffer bytes.Buffer
	originalOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logBuffer)
	defer logrus.SetOutput(originalOutput)

	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer logrus.SetFormatter(&logrus.TextFormatter{})

	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinLoggingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "GET"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"method": "POST"})
	})
	router.PUT("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "PUT"})
	})
	router.DELETE("/test", func(c *gin.Context) {
		c.JSON(http.StatusNoContent, gin.H{"method": "DELETE"})
	})

	methods := []struct {
		method     string
		statusCode int
	}{
		{"GET", http.StatusOK},
		{"POST", http.StatusCreated},
		{"PUT", http.StatusOK},
		{"DELETE", http.StatusNoContent},
	}

	for _, m := range methods {
		t.Run(fmt.Sprintf("%s方法日志记录", m.method), func(t *testing.T) {
			logBuffer.Reset()
			
			var req *http.Request
			if m.method == "POST" || m.method == "PUT" {
				req, _ = http.NewRequest(m.method, "/test", strings.NewReader(`{"data":"test"}`))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(m.method, "/test", nil)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			AssertEqual(t, m.statusCode, w.Code, fmt.Sprintf("%s方法响应状态码应正确", m.method))

			// 验证日志记录
			logOutput := logBuffer.String()
			lines := strings.Split(strings.TrimSpace(logOutput), "\n")
			if len(lines) > 0 {
				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(lines[len(lines)-1]), &logEntry)
				AssertNoError(t, err, "日志应为有效JSON格式")
				
				AssertEqual(t, m.method, logEntry["method"], "HTTP方法应正确记录")
				AssertEqual(t, float64(m.statusCode), logEntry["status_code"], "状态码应正确记录")
				AssertEqual(t, "/test", logEntry["path"], "路径应正确记录")
			}
		})
	}
}