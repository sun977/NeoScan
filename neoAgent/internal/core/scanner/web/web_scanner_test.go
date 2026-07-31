package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"neoagent/internal/core/model"
)

// TestWebScanner_Fingerprint 验证 fallbackFetch + buildWebResult 这条重构后的
// 收口链路，指纹识别能力和重构前完全一致（零回归）。
//
// 重构前这个测试直接调用私有方法 fallbackScan，一次调用拿到组装好的
// []*model.TaskResult；重构后 fallbackScan 已经拆分成"只负责抓取"的
// fallbackFetch 和"只负责组装"的 buildWebResult 两个函数，所以测试也要
// 对应拆成两步调用，中间的 pageData 就是两者之间传递数据的桥梁。
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

	// 4. 第一步：fallbackFetch 只负责抓取原始数据
	ctx := context.Background()
	body, headers, statusCode, title, _, err := scanner.fallbackFetch(ctx, ts.URL)
	if err != nil {
		t.Fatalf("fallbackFetch failed: %v", err)
	}

	// 5. 第二步：buildWebResult 负责指纹匹配 + 组装最终结果
	task := &model.Task{ID: "test-task", Target: "127.0.0.1"}
	result := scanner.buildWebResult(task, time.Now(), "127.0.0.1", 0, pageData{
		URL: ts.URL, StatusCode: statusCode, Title: title, Body: body, Headers: headers,
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

// TestFallbackFetch_ReturnsSeedLinks 验证 fallbackFetch 除了抓取原始数据之外，
// 还能顺手提取出页面里的链接，作为 Sprint 5 挂上 crawler 后的 BFS 种子。
func TestFallbackFetch_ReturnsSeedLinks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html><body>
			<a href="/a">a</a>
			<a href="/b">b</a>
			<a href="/c">c</a>
		</body></html>`)
	}))
	defer ts.Close()

	scanner := NewWebScanner()
	ctx := context.Background()
	_, _, _, _, links, err := scanner.fallbackFetch(ctx, ts.URL)
	if err != nil {
		t.Fatalf("fallbackFetch failed: %v", err)
	}

	if len(links) != 3 {
		t.Fatalf("expected 3 seed links, got %d: %v", len(links), links)
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
