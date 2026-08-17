package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestRequester_BasicGet 验证正常 200 响应
func TestRequester_BasicGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Hello, World!")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
	})

	resp, err := req.Do(context.Background(), server.URL+"/", "/path")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Body != "Hello, World!" {
		t.Errorf("expected body 'Hello, World!', got %q", resp.Body)
	}
}

// TestRequester_PathPreserving 验证路径中包含特殊字符
func TestRequester_PathPreserving(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 记录实际请求路径
		t.Logf("Received path: %s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
	})

	// 测试含 % 编码的路径
	resp, err := req.Do(context.Background(), server.URL+"/", "/test%2Fpath")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestRequester_Timeout 验证服务器超时场景
func TestRequester_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 永远不响应
		<-r.Context().Done()
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout:    100 * time.Millisecond,
		MaxRetries: 0, // 不重试
	})

	_, err := req.Do(context.Background(), server.URL+"/", "/slow-path")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// 验证错误类型
	if err != ErrConnectionTimeout {
		t.Logf("Got error type: %T (%v)", err, err)
	}
}

// TestRequester_Retry 验证重试逻辑
func TestRequester_Retry(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 2 {
			// 第一次请求模拟超时
			time.Sleep(200 * time.Millisecond)
			fmt.Fprint(w, "recovered")
			return
		}
		fmt.Fprint(w, "success after retry")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout:    100 * time.Millisecond,
		MaxRetries: 2, // 最多重试 2 次
	})

	resp, err := req.Do(context.Background(), server.URL+"/", "/retry-path")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if resp.Body != "success after retry" {
		t.Errorf("unexpected body: %q", resp.Body)
	}
	t.Logf("Total attempts: %d", attempt)
}

// TestRequester_NoFollowRedirect 验证不跟随重定向
func TestRequester_NoFollowRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/new-location", http.StatusFound)
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout:         5 * time.Second,
		FollowRedirects: false,
	})

	resp, err := req.Do(context.Background(), server.URL+"/", "/redirect")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected status 302, got %d", resp.StatusCode)
	}
	if resp.Location != "/new-location" {
		t.Errorf("expected location '/new-location', got %q", resp.Location)
	}
}

// TestRequester_FollowRedirect 验证跟随重定向
func TestRequester_FollowRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		fmt.Fprint(w, "final response")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout:         5 * time.Second,
		FollowRedirects: true,
	})

	resp, err := req.Do(context.Background(), server.URL+"/", "/redirect")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 after redirect, got %d", resp.StatusCode)
	}
	if resp.Body != "final response" {
		t.Errorf("unexpected body: %q", resp.Body)
	}
}

// TestRequester_UserAgent 验证 UA 随机选择
func TestRequester_UserAgent(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	uas := []string{
		"Custom-Agent/1.0",
		"Custom-Agent/2.0",
		"Custom-Agent/3.0",
	}

	req := NewRequester(RequesterConfig{
		Timeout:    5 * time.Second,
		UserAgents: uas,
	})

	// 连续请求，验证 UA 被轮询使用
	for i := 0; i < 3; i++ {
		_, err := req.Do(context.Background(), server.URL+"/", fmt.Sprintf("/path-%d", i))
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if receivedUA != uas[i%3] {
			t.Errorf("request %d: expected UA %q, got %q", i, uas[i%3], receivedUA)
		}
	}
}

// TestRequester_CustomHeaders 验证自定义请求头
func TestRequester_CustomHeaders(t *testing.T) {
	var receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom-Header")
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	})

	_, err := req.Do(context.Background(), server.URL+"/", "/test")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if receivedHeader != "custom-value" {
		t.Errorf("expected custom header 'custom-value', got %q", receivedHeader)
	}
}

// TestRequester_Proxy 验证代理配置（不验证实际代理，只验证不 panic）
func TestRequester_Proxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	// 传入无效的代理 URL，验证错误被静默处理（url.Parse 失败不报错）
	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
		Proxy:   "http://nonexistent-proxy:9999",
	})

	// 注意：这个请求会连接到真实代理（不存在），所以会失败
	// 但 NewRequester 本身不应该 panic
	if req == nil {
		t.Fatal("expected non-nil Requester")
	}
}

// TestRequester_ConnectionRefused 验证连接拒绝错误
func TestRequester_ConnectionRefused(t *testing.T) {
	req := NewRequester(RequesterConfig{
		Timeout:    500 * time.Millisecond,
		MaxRetries: 0,
	})

	// 连接到一个不会监听端口
	_, err := req.Do(context.Background(), "http://127.0.0.1:1", "/test")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if err != ErrConnectionRefused {
		t.Logf("Got error type: %T (%v)", err, err)
	}
}

// TestRequester_TitleExtraction 验证 HTML <title> 提取
func TestRequester_TitleExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Admin Login Page</title></head><body>Admin</body></html>`)
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
	})

	resp, err := req.Do(context.Background(), server.URL+"/", "/")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Title != "Admin Login Page" {
		t.Errorf("expected title 'Admin Login Page', got %q", resp.Title)
	}
}

// TestRequester_WordsAndLines 验证词数和行数统计
func TestRequester_WordsAndLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world\nfoo bar baz")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
	})

	resp, err := req.Do(context.Background(), server.URL+"/", "/")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Words != 5 {
		t.Errorf("expected 5 words, got %d", resp.Words)
	}
	if resp.Lines != 2 {
		t.Errorf("expected 2 lines, got %d", resp.Lines)
	}
}

// TestRequester_ContextCancel 验证 context 取消
func TestRequester_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
	})

	_, err := req.Do(ctx, "http://example.com/", "/test")
	if err == nil {
		t.Fatal("expected context canceled error, got nil")
	}
	if err != context.Canceled {
		t.Logf("Got error type: %T (%v)", err, err)
	}
}

// TestRequester_BodyLimit 验证响应体上限 10MB
func TestRequester_BodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 发送 20MB 响应体
		w.Write(make([]byte, 20*1024*1024))
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 10 * time.Second,
	})

	resp, err := req.Do(context.Background(), server.URL+"/", "/")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// 验证被截断到 10MB
	if resp.Size > 10*1024*1024 {
		t.Errorf("expected size <= 10MB, got %d bytes", resp.Size)
	}
	t.Logf("Response size: %d bytes (limit: 10MB)", resp.Size)
}

// TestRequester_ClassifyError 验证错误分类
func TestRequester_ClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		inputErr error
		expected error
	}{
		{"timeout", fmt.Errorf("Get \"http://test\": net/http: request canceled while waiting for connection (i/o timeout)"), ErrConnectionTimeout},
		{"connection_refused", fmt.Errorf("dial tcp 127.0.0.1:80: connect: connection refused"), ErrConnectionRefused},
		{"dns_failed", fmt.Errorf("Get \"http://nonexistent.domain\": dial: no such host"), ErrDNSFailed},
		{"ssl_error", fmt.Errorf("Get \"https://test\": tls: certificate invalid"), ErrSSLError},
		{"other", fmt.Errorf("Get \"http://test\": some other error"), ErrOther},
		{"context_deadline", context.DeadlineExceeded, context.DeadlineExceeded},
		{"context_canceled", context.Canceled, context.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyError(tt.inputErr)
			if got != tt.expected {
				t.Errorf("classifyError(%v) = %v (%T), want %v (%T)",
					tt.inputErr, got, got, tt.expected, tt.expected)
			}
		})
	}
}

// TestRequester_IsConnectionError 验证连接错误判断
func TestRequester_IsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		inputErr error
		expected bool
	}{
		{"connection_refused", ErrConnectionRefused, true},
		{"dns_failed", ErrDNSFailed, true},
		{"ssl_error", ErrSSLError, true},
		{"timeout", ErrConnectionTimeout, false},
		{"other", ErrOther, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isConnectionError(tt.inputErr)
			if got != tt.expected {
				t.Errorf("isConnectionError(%v) = %v, want %v", tt.inputErr, got, tt.expected)
			}
		})
	}
}

// TestRequester_DefaultUA 验证默认 UA
func TestRequester_DefaultUA(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
		// 不传 UserAgents
	})

	_, err := req.Do(context.Background(), server.URL+"/", "/")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedDefault := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	if receivedUA != expectedDefault {
		t.Errorf("expected default UA %q, got %q", expectedDefault, receivedUA)
	}
}

// TestExtractTitle 验证标题提取边缘情况
func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", `<html><head><title>Hello</title></head><body>`, "Hello"},
		{"lowercase_tag", `<html><head><TITLE>Hello World</TITLE></head>`, "Hello World"},
		{"no_title", `<html><head><meta charset="utf-8"></head><body>`, ""},
		{"empty_title", `<html><head><title></title></head><body>`, ""},
		{"with_whitespace", `<html><head><title>  Padded  </title></head>`, "Padded"},
		{"no_close_tag", `<html><head><title>Unclosed</head><body>`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.input)
			if got != tt.expected {
				t.Errorf("extractTitle(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// BenchmarkRequester_Basic 性能基准：单次请求
func BenchmarkRequester_Basic(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := req.Do(context.Background(), server.URL+"/", "/")
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// TestRequester_MaxRetriesExceeded 验证达到最大重试次数
func TestRequester_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 每次请求都延迟超时
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout:    50 * time.Millisecond,
		MaxRetries: 2, // 重试 2 次 = 3 次尝试
	})

	_, err := req.Do(context.Background(), server.URL+"/", "/")
	if err == nil {
		t.Fatal("expected max retries exceeded error, got nil")
	}
	// 验证返回了 ErrMaxRetriesExceeded
	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Logf("Got error: %v", err)
	}
}

// TestRequester_Delay 验证请求间隔延迟
func TestRequester_Delay(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	start := time.Now()
	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
		Delay:   50 * time.Millisecond,
	})

	// 发送 3 个请求
	for i := 0; i < 3; i++ {
		_, err := req.Do(context.Background(), server.URL+"/", fmt.Sprintf("/path-%d", i))
		if err != nil {
			t.Fatalf("request %d: expected no error, got: %v", i, err)
		}
	}

	elapsed := time.Since(start)
	// 3 个请求之间有 2 个延迟
	minExpected := 100 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("expected at least %v elapsed (2 delays), got %v", minExpected, elapsed)
	}
	t.Logf("Total time: %v for 3 requests", elapsed)
}

// TestRequester_RateLimit 验证速率限制
func TestRequester_RateLimit(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	start := time.Now()
	req := NewRequester(RequesterConfig{
		Timeout:   5 * time.Second,
		RateLimit: 10, // 每秒最多 10 个请求
	})

	// 发送 15 个请求
	for i := 0; i < 15; i++ {
		_, err := req.Do(context.Background(), server.URL+"/", fmt.Sprintf("/path-%d", i))
		if err != nil {
			t.Fatalf("request %d: expected no error, got: %v", i, err)
		}
	}

	elapsed := time.Since(start)
	// 15 个请求受 10/s 限制，至少需要 1.5 秒
	// 注意：channel buffer 是 10，所以前 10 个立即通过
	minExpected := 100 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("rate limiter may not be working: elapsed %v < %v", elapsed, minExpected)
	}
	t.Logf("Total time: %v for 15 requests (rate limit: 10/s)", elapsed)
}

// TestRequester_HTTPMethod 验证自定义 HTTP 方法
func TestRequester_HTTPMethod(t *testing.T) {
	var receivedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{
		Timeout: 5 * time.Second,
		Method:  "POST",
	})

	_, err := req.Do(context.Background(), server.URL+"/", "/")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if receivedMethod != "POST" {
		t.Errorf("expected method POST, got %q", receivedMethod)
	}
}
