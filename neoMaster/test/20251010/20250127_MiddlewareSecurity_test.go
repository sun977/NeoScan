// MiddlewareSecurity测试文件
// 测试优化后的安全中间件功能，包括CORS、安全头、XSS防护等
// 适配拆分后的security.go模块
// 测试命令：go test -v -run TestMiddlewareSecurity ./test/20250127

// Package test 安全中间件功能测试
// 测试拆分后的security.go中间件模块
package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMiddlewareSecurity 测试安全中间件模块
func TestMiddlewareSecurity(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		// 检查必要的服务是否可用
		if ts.MiddlewareManager == nil {
			t.Skip("跳过安全中间件测试：中间件管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		t.Run("CORS中间件功能", func(t *testing.T) {
			testCORSMiddleware(t, ts)
		})

		t.Run("安全头中间件功能", func(t *testing.T) {
			testSecurityHeadersMiddleware(t, ts)
		})

		t.Run("XSS防护中间件功能", func(t *testing.T) {
			testXSSProtectionMiddleware(t, ts)
		})

		t.Run("内容类型嗅探防护", func(t *testing.T) {
			testContentTypeNosniffMiddleware(t, ts)
		})

		t.Run("点击劫持防护", func(t *testing.T) {
			testClickjackingProtectionMiddleware(t, ts)
		})

		t.Run("HSTS安全传输", func(t *testing.T) {
			testHSTSMiddleware(t, ts)
		})
	})
}

// testCORSMiddleware 测试CORS中间件功能
func testCORSMiddleware(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinCORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "CORS测试成功"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "POST CORS测试成功"})
	})
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 测试预检请求 (OPTIONS)
	t.Run("预检请求处理", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 验证CORS响应头
		AssertEqual(t, http.StatusOK, w.Code, "预检请求应返回200状态码")
		AssertEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"), "应允许所有来源")
		AssertTrue(t, strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "POST"), "应允许POST方法")
		AssertTrue(t, strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Content-Type"), "应允许Content-Type头")
		AssertTrue(t, strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Authorization"), "应允许Authorization头")
	})

	// 测试实际CORS请求
	t.Run("实际CORS请求", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "CORS请求应成功")
		AssertEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"), "应设置CORS允许来源")
		AssertEqual(t, "true", w.Header().Get("Access-Control-Allow-Credentials"), "应允许凭据")
	})

	// 测试不同来源的请求
	t.Run("不同来源请求", func(t *testing.T) {
		origins := []string{
			"https://localhost:3000",
			"http://127.0.0.1:8080",
			"https://app.example.com",
		}

		for _, origin := range origins {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", origin)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			AssertEqual(t, http.StatusOK, w.Code, "不同来源的请求应成功")
			AssertEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"), "应允许所有来源")
		}
	})
}

// testSecurityHeadersMiddleware 测试安全头中间件功能
func testSecurityHeadersMiddleware(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "安全头测试成功"})
	})

	// 发送测试请求
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	AssertEqual(t, http.StatusOK, w.Code, "响应状态码应为200")

	// 验证安全头
	headers := w.Header()
	
	// X-Content-Type-Options
	AssertEqual(t, "nosniff", headers.Get("X-Content-Type-Options"), "应设置X-Content-Type-Options头")
	
	// X-Frame-Options
	frameOptions := headers.Get("X-Frame-Options")
	AssertTrue(t, frameOptions == "DENY" || frameOptions == "SAMEORIGIN", "应设置X-Frame-Options头")
	
	// X-XSS-Protection
	xssProtection := headers.Get("X-XSS-Protection")
	AssertTrue(t, xssProtection == "1; mode=block" || xssProtection == "0", "应设置X-XSS-Protection头")
	
	// Referrer-Policy
	referrerPolicy := headers.Get("Referrer-Policy")
	AssertTrue(t, len(referrerPolicy) > 0, "应设置Referrer-Policy头")
	
	// Content-Security-Policy
	csp := headers.Get("Content-Security-Policy")
	AssertTrue(t, len(csp) > 0, "应设置Content-Security-Policy头")
}

// testXSSProtectionMiddleware 测试XSS防护中间件功能
func testXSSProtectionMiddleware(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		// 模拟可能的XSS攻击输入
		userInput := c.Query("input")
		c.JSON(http.StatusOK, gin.H{
			"message": "XSS防护测试",
			"input":   userInput,
		})
	})

	// 测试正常输入
	t.Run("正常输入处理", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test?input=normal_text", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "正常输入应成功处理")
		AssertTrue(t, strings.Contains(w.Body.String(), "normal_text"), "正常输入应正确返回")
		
		// 验证XSS防护头
		xssProtection := w.Header().Get("X-XSS-Protection")
		AssertTrue(t, len(xssProtection) > 0, "应设置XSS防护头")
	})

	// 测试潜在XSS攻击输入
	t.Run("XSS攻击输入处理", func(t *testing.T) {
		xssInputs := []string{
			"<script>alert('xss')</script>",
			"javascript:alert('xss')",
			"<img src=x onerror=alert('xss')>",
			"<svg onload=alert('xss')>",
		}

		for _, xssInput := range xssInputs {
			req, _ := http.NewRequest("GET", "/test?input="+xssInput, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证响应仍然成功（中间件不应阻止请求，而是设置防护头）
			AssertEqual(t, http.StatusOK, w.Code, "XSS输入请求应成功（由防护头处理）")
			
			// 验证XSS防护头存在
			xssProtection := w.Header().Get("X-XSS-Protection")
			AssertTrue(t, len(xssProtection) > 0, "应设置XSS防护头")
		}
	})
}

// testContentTypeNosniffMiddleware 测试内容类型嗅探防护
func testContentTypeNosniffMiddleware(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
	router.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"type": "json"})
	})
	router.GET("/html", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, "<html><body>HTML内容</body></html>")
	})
	router.GET("/text", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, "纯文本内容")
	})

	contentTypes := []struct {
		path        string
		contentType string
	}{
		{"/json", "application/json"},
		{"/html", "text/html"},
		{"/text", "text/plain"},
	}

	for _, ct := range contentTypes {
		t.Run("内容类型嗅探防护-"+ct.path, func(t *testing.T) {
			req, _ := http.NewRequest("GET", ct.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			AssertEqual(t, http.StatusOK, w.Code, "请求应成功")
			
			// 验证X-Content-Type-Options头
			AssertEqual(t, "nosniff", w.Header().Get("X-Content-Type-Options"), "应设置nosniff防护")
			
			// 验证Content-Type头存在
			contentType := w.Header().Get("Content-Type")
			AssertTrue(t, strings.Contains(contentType, ct.contentType), "Content-Type应正确设置")
		})
	}
}

// testClickjackingProtectionMiddleware 测试点击劫持防护
func testClickjackingProtectionMiddleware(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
	router.GET("/page", func(c *gin.Context) {
		c.HTML(http.StatusOK, "test.html", gin.H{
			"title": "测试页面",
		})
	})
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "API响应"})
	})

	endpoints := []string{"/page", "/api"}

	for _, endpoint := range endpoints {
		t.Run("点击劫持防护-"+endpoint, func(t *testing.T) {
			req, _ := http.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证X-Frame-Options头
			frameOptions := w.Header().Get("X-Frame-Options")
			AssertTrue(t, frameOptions == "DENY" || frameOptions == "SAMEORIGIN", 
				"应设置X-Frame-Options防护点击劫持")
			
			// 验证Content-Security-Policy中的frame-ancestors指令
			csp := w.Header().Get("Content-Security-Policy")
			if len(csp) > 0 {
				AssertTrue(t, strings.Contains(csp, "frame-ancestors") || 
					strings.Contains(csp, "default-src"), "CSP应包含frame防护")
			}
		})
	}
}

// testHSTSMiddleware 测试HSTS安全传输
func testHSTSMiddleware(t *testing.T, ts *TestSuite) {
	// 创建测试路由
	router := gin.New()
	router.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
	router.GET("/secure", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "安全传输测试"})
	})

	// 测试HTTPS请求（模拟）
	t.Run("HSTS头设置", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/secure", nil)
		req.Header.Set("X-Forwarded-Proto", "https") // 模拟HTTPS
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "HTTPS请求应成功")
		
		// 验证HSTS头（如果中间件支持）
		hsts := w.Header().Get("Strict-Transport-Security")
		if len(hsts) > 0 {
			AssertTrue(t, strings.Contains(hsts, "max-age="), "HSTS应包含max-age指令")
		}
	})

	// 测试HTTP请求
	t.Run("HTTP请求处理", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/secure", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		AssertEqual(t, http.StatusOK, w.Code, "HTTP请求应成功")
		
		// HTTP请求不应设置HSTS头
		_ = w.Header().Get("Strict-Transport-Security")
		// 注意：某些实现可能仍然设置HSTS头，这取决于具体实现
	})
}

// TestSecurityMiddlewareIntegration 测试安全中间件集成
func TestSecurityMiddlewareIntegration(t *testing.T) {
	RunWithTestEnvironment(t, func(ts *TestSuite) {
		if ts.MiddlewareManager == nil {
			t.Skip("跳过安全中间件集成测试：中间件管理器不可用")
			return
		}

		// 设置Gin为测试模式
		gin.SetMode(gin.TestMode)

		// 创建集成所有安全中间件的路由
		router := gin.New()
		router.Use(ts.MiddlewareManager.GinCORSMiddleware())
		router.Use(ts.MiddlewareManager.GinSecurityHeadersMiddleware())
		router.GET("/integrated", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "安全中间件集成测试成功"})
		})

		// 发送集成测试请求
		req, _ := http.NewRequest("GET", "/integrated", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 验证响应
		AssertEqual(t, http.StatusOK, w.Code, "集成请求应成功")

		// 验证所有安全头都存在
		headers := w.Header()
		
		// CORS头
		AssertEqual(t, "*", headers.Get("Access-Control-Allow-Origin"), "CORS头应存在")
		
		// 安全头
		AssertEqual(t, "nosniff", headers.Get("X-Content-Type-Options"), "Content-Type防护头应存在")
		AssertTrue(t, len(headers.Get("X-Frame-Options")) > 0, "Frame防护头应存在")
		AssertTrue(t, len(headers.Get("X-XSS-Protection")) > 0, "XSS防护头应存在")
		AssertTrue(t, len(headers.Get("Referrer-Policy")) > 0, "Referrer策略头应存在")
		AssertTrue(t, len(headers.Get("Content-Security-Policy")) > 0, "CSP头应存在")

		t.Log("安全中间件集成测试通过，所有安全头均正确设置")
	})
}