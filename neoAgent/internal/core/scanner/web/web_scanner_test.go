package web

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"neoagent/internal/core/model"
)

// TestWebScanner_Fingerprint 验证 buildWebResult 这条收口链路，指纹识别能力
// 符合预期。
//
// 重构前这个测试通过私有方法 fallbackFetch 抓取原始数据；fallbackFetch 已经
// 随 web扫描模块重构实施文档.md 步骤 2 下沉为 crawler.FetchAndCrawl 内部的
// 私有实现，web 包不再能直接调用它。这个测试真正要验证的是 buildWebResult
// 的指纹匹配能力，与"用什么方式抓取原始数据"无关，改用标准库 http.Get 直接
// 拿 body/headers/status，语义不变、验证目标不变。
func TestWebScanner_Fingerprint(t *testing.T) {
	// 1. Mock Server (Nginx + PHP)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "nginx/1.18.0")
		w.Header().Set("X-Powered-By", "PHP/7.4.3")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, "<html><head><title>Test Page</title></head><body>Hello World</body></html>")
	}))
	defer ts.Close()

	// 2. Initialize Scanner
	scanner := NewWebScanner()

	// 3. Force Init (to load rules from relative path)
	// ensureInit is private, but we are in package web
	scanner.ensureInit()

	// 4. 第一步：直接用标准库抓取原始数据（等价于原 fallbackFetch 的"抓取"职责）
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	body := string(bodyBytes)
	headers := map[string]string{}
	for k, v := range resp.Header {
		headers[k] = v[0]
	}

	// 5. 第二步：buildWebResult 负责指纹匹配 + 组装最终结果
	task := &model.Task{ID: "test-task", Target: "127.0.0.1"}
	result := scanner.buildWebResult(task, time.Now(), "127.0.0.1", 0, pageData{
		URL: ts.URL, StatusCode: resp.StatusCode, Title: "Test Page", Body: body, Headers: headers,
	})

	res, ok := result.Result.(*model.WebResult)
	if !ok {
		t.Fatal("Result is not *model.WebResult")
	}

	// 6. Verify Fingerprints
	t.Logf("TechStack: %v", res.TechStack)

	foundNginx := false
	foundPHP := false

	for _, tech := range res.TechStack {
		if tech == "Nginx" {
			foundNginx = true
		}
		if tech == "PHP" {
			foundPHP = true
		}
	}

	if !foundNginx {
		t.Error("Failed to identify Nginx (Check if rules/fingerprint/web/web_fingerprints.json is loaded)")
	}
	if !foundPHP {
		t.Error("Failed to identify PHP")
	}
	if res.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", res.Title)
	}
}

// TestBuildWebResult_ConsistentAcrossSources 验证 buildWebResult 收口后，
// go-rod 路径（带完整 RichContext）和 fallback 路径（RichContext=nil，
// 由 buildWebResult 内部退化拼一个最小版本）对同一份 Body/Headers 跑指纹匹配，
// 得到的 TechStack 完全一致——这是收口重构最核心的正确性要求：不能因为
// 收口就让某一条路径的指纹识别精度发生静默退化。
func TestBuildWebResult_ConsistentAcrossSources(t *testing.T) {
	scanner := NewWebScanner()
	scanner.ensureInit()

	body := "<html><head><title>Test Page</title></head><body>Hello World</body></html>"
	headers := map[string]string{
		"Server":       "nginx/1.18.0",
		"X-Powered-By": "PHP/7.4.3",
		"Content-Type": "text/html",
	}
	task := &model.Task{ID: "test-task", Target: "127.0.0.1"}

	// fallback 路径：不传 RichContext，buildWebResult 内部会用 Body/Headers/Title
	// 拼一个最小可用的 RichContext。
	fallbackResult := scanner.buildWebResult(task, time.Now(), "127.0.0.1", 0, pageData{
		URL: "http://127.0.0.1/", StatusCode: 200, Title: "Test Page", Body: body, Headers: headers,
	})

	// go-rod 路径：模拟已经提取好的完整 richCtx（真实场景来自 ExtractRichContext）。
	richCtx := map[string]interface{}{
		"body":  body,
		"title": "Test Page",
	}
	rodResult := scanner.buildWebResult(task, time.Now(), "127.0.0.1", 0, pageData{
		URL: "http://127.0.0.1/", StatusCode: 200, Title: "Test Page", Body: body, Headers: headers,
		RichContext: richCtx,
	})

	fallbackWebResult, ok := fallbackResult.Result.(*model.WebResult)
	if !ok {
		t.Fatal("fallback result is not *model.WebResult")
	}
	rodWebResult, ok := rodResult.Result.(*model.WebResult)
	if !ok {
		t.Fatal("go-rod result is not *model.WebResult")
	}

	if len(fallbackWebResult.TechStack) == 0 {
		t.Fatal("expected non-empty TechStack from fallback path")
	}
	if len(fallbackWebResult.TechStack) != len(rodWebResult.TechStack) {
		t.Fatalf("TechStack length mismatch: fallback=%v, go-rod=%v", fallbackWebResult.TechStack, rodWebResult.TechStack)
	}
	for i := range fallbackWebResult.TechStack {
		if fallbackWebResult.TechStack[i] != rodWebResult.TechStack[i] {
			t.Fatalf("TechStack mismatch at index %d: fallback=%s, go-rod=%s", i, fallbackWebResult.TechStack[i], rodWebResult.TechStack[i])
		}
	}
}
