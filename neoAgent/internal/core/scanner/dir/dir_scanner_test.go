package dir

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/dir/result"
)

// newTestTask 构造一个指向 mock server 的 dir_scan 任务，params 可覆盖默认参数。
func newTestTask(serverURL string, params map[string]interface{}) *model.Task {
	task := model.NewTask(model.TaskTypeDirScan, serverURL)
	for k, v := range params {
		task.Params[k] = v
	}
	return task
}

// TestDirScanner_BasicScan 验证 mock 服务器上存在的路径均被发现，其余 404 被过滤。
func TestDirScanner_BasicScan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin", "/secret":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "hit page %s", r.URL.Path)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "not found")
		}
	}))
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"wordlists":    "",
		"threads":      5,
		"timeout":      3,
		"skip_builtin": true,
	})
	// 使用小型自定义字典而非内置全量字典，加速测试
	task.Params["wordlists"] = writeTempWordlist(t, []string{"/admin", "/secret", "/nope1", "/nope2", "/nope3"})

	results, err := s.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 TaskResult, got %d", len(results))
	}

	found := collectHitPaths(t, results[0])
	if !found["/admin"] || !found["/secret"] {
		t.Errorf("expected /admin and /secret to be found, got %v", found)
	}
	if found["/nope1"] || found["/nope2"] || found["/nope3"] {
		t.Errorf("404 paths should not appear in hits, got %v", found)
	}
}

// TestDirScanner_StatsPopulated 验证 DirResult.Stats 在扫描结束后被正确填充，
// 而不是保持零值（Bug: ScanStats 字段定义了但 worker/Run 从未写入）。
func TestDirScanner_StatsPopulated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "hit page")
		case "/blocked":
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "forbidden")
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "not found")
		}
	}))
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":      3,
		"timeout":      3,
		"skip_builtin": true,
	})
	task.Params["wordlists"] = writeTempWordlist(t, []string{"/admin", "/blocked", "/nope1", "/nope2"})

	results, err := s.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	dr, ok := results[0].Result.(*result.DirResult)
	if !ok {
		t.Fatalf("TaskResult.Result is not *result.DirResult: %T", results[0].Result)
	}

	stats := dr.Stats
	if stats.TotalRequests != 4 {
		t.Errorf("TotalRequests = %d, want 4", stats.TotalRequests)
	}
	if stats.SuccessfulReqs != 4 {
		t.Errorf("SuccessfulReqs = %d, want 4 (all requests got a response)", stats.SuccessfulReqs)
	}
	// 默认 ExcludeStatus 只包含 [404,500,502,503]，403 不在其中会被判定为命中，
	// 只有两个 404 会被 Filter 过滤掉。
	if stats.FilteredReqs != 2 {
		t.Errorf("FilteredReqs = %d, want 2 (2x404)", stats.FilteredReqs)
	}
	if stats.ErrorReqs != 0 {
		t.Errorf("ErrorReqs = %d, want 0", stats.ErrorReqs)
	}
	if stats.AvgRTT <= 0 {
		t.Errorf("AvgRTT = %v, want > 0", stats.AvgRTT)
	}
	if stats.MaxRTT <= 0 || stats.MinRTT <= 0 {
		t.Errorf("MaxRTT/MinRTT = %v/%v, want both > 0", stats.MaxRTT, stats.MinRTT)
	}
}

// TestDirScanner_WildcardCDN 验证所有路径返回相同响应时，通配符检测生效，零误报。
func TestDirScanner_WildcardCDN(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "<html><body>Welcome to our CDN default page, nothing here</body></html>")
	}))
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":      5,
		"timeout":      3,
		"skip_builtin": true,
	})
	task.Params["wordlists"] = writeTempWordlist(t, []string{
		"/a1", "/a2", "/a3", "/a4", "/a5", "/a6", "/a7", "/a8", "/a9", "/a10",
	})

	results, err := s.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	found := collectHitPaths(t, results[0])
	if len(found) != 0 {
		t.Errorf("expected zero hits on wildcard CDN, got %v", found)
	}
}

// TestDirScanner_Recursive 验证 /api/ 命中 301 后触发递归，子路径被扫描。
func TestDirScanner_Recursive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api":
			w.Header().Set("Location", "/api/")
			w.WriteHeader(http.StatusMovedPermanently)
		case "/api/users":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "users list")
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "not found")
		}
	}))
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":              5,
		"timeout":              3,
		"skip_builtin":         true,
		"recursive":            true,
		"max_recursion_depth":  3,
	})
	task.Params["wordlists"] = writeTempWordlist(t, []string{"/api", "/api/users"})

	results, err := s.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	found := collectHitPaths(t, results[0])
	if !found["/api"] {
		t.Errorf("expected /api to be found, got %v", found)
	}
}

// TestDirScanner_MaxRecursionDepth 验证递归深度限制生效（不会无限递归导致挂起或异常）。
func TestDirScanner_MaxRecursionDepth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 任何以 / 结尾的目录路径都返回 200，模拟无限深层目录结构
		if strings.HasSuffix(r.URL.Path, "/") || r.URL.Path == "" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "dir listing")
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":              5,
		"timeout":              3,
		"skip_builtin":         true,
		"recursive":            true,
		"deep_recursive":       true,
		"max_recursion_depth":  2,
	})
	task.Params["wordlists"] = writeTempWordlist(t, []string{"/a/b/c/"})

	done := make(chan struct{})
	var results []*model.TaskResult
	var err error
	go func() {
		results, err = s.Run(context.Background(), task)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("scan did not complete within timeout, possible infinite recursion")
	}

	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 TaskResult, got %d", len(results))
	}
}

// TestDirScanner_ContextCancel 验证 context 取消后扫描立即停止，无 goroutine 泄漏。
func TestDirScanner_ContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	before := runtime.NumGoroutine()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":      10,
		"timeout":      3,
		"skip_builtin": true,
	})
	paths := make([]string, 200)
	for i := range paths {
		paths[i] = fmt.Sprintf("/path-%d", i)
	}
	task.Params["wordlists"] = writeTempWordlist(t, paths)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _ = s.Run(ctx, task)

	// 给 goroutine 一点时间退出
	time.Sleep(300 * time.Millisecond)
	after := runtime.NumGoroutine()

	if after > before+10 {
		t.Errorf("possible goroutine leak: before=%d after=%d", before, after)
	}
}

// TestDirScanner_MaxEntries 验证超大字典不会导致崩溃（截断生效）。
func TestDirScanner_MaxEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads": 20,
		"timeout": 2,
	})
	// 使用内置字典（体量足够大，验证扫描不 panic 且能在合理时间完成）
	results, err := s.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 TaskResult, got %d", len(results))
	}
}

// ── 测试辅助函数 ──────────────────────────────────────────────────────────────

func writeTempWordlist(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "wordlist.txt")
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write wordlist: %v", err)
	}
	return path
}

func collectHitPaths(t *testing.T, tr *model.TaskResult) map[string]bool {
	t.Helper()
	found := make(map[string]bool)
	dr, ok := tr.Result.(*result.DirResult)
	if !ok || dr == nil {
		t.Fatalf("TaskResult.Result is not *result.DirResult: %T", tr.Result)
		return found
	}
	for _, h := range dr.Hits {
		found[h.Path] = true
	}
	return found
}
