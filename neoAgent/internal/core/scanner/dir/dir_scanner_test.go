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
		case "/api/":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "api-root-content")
		case "/api/config":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "config-content")
		case "/api/users":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "users-content")
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "not found")
		}
	}))
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":             5,
		"timeout":             3,
		"skip_builtin":        true,
		"recursive":           true,
		"max_recursion_depth": 3,
	})
	// 字典含目录条目 /api/ 和两个文件名条目；
	// /api/config、/api/users 只能通过"命中 /api/ 后字典在其前缀下重放"被发现。
	task.Params["wordlists"] = writeTempWordlist(t, []string{"/api/", "config", "users"})

	results, err := s.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	found := collectHitPaths(t, results[0])
	if !found["/api/"] {
		t.Errorf("expected /api/ to be found, got %v", found)
	}
	// 真递归的核心断言：子路径是"前缀×字典"拼接展开的，不在原始字典里
	if !found["/api/config"] {
		t.Errorf("expected /api/config to be found via recursion, got %v", found)
	}
	if !found["/api/users"] {
		t.Errorf("expected /api/users to be found via recursion, got %v", found)
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

// TestGenerateSubpaths 验证"目录前缀 × 字典"拼接逻辑（真递归核心）。
func TestGenerateSubpaths(t *testing.T) {
	entries := []string{"config.php", "/login", "admin/"}
	got := generateSubpaths("/api/", entries)
	want := []string{"/api/config.php", "/api/login", "/api/admin/"}
	if len(got) != len(want) {
		t.Fatalf("generateSubpaths len = %d, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("generateSubpaths[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestParentDirsOf 验证父级目录枚举（深度递归模式）。
func TestParentDirsOf(t *testing.T) {
	got := parentDirsOf("/a/b/c")
	want := []string{"/a/", "/a/b/", "/a/b/c/"}
	if len(got) != len(want) {
		t.Fatalf("parentDirsOf len = %d, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("parentDirsOf[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestDirDepth 验证目录嵌套层数计算（递归深度判断依据）。
func TestDirDepth(t *testing.T) {
	cases := map[string]int{
		"/":           0,
		"/admin/":     1,
		"/admin/api/": 2,
		"/a/b/c/d/":   4,
	}
	for dir, want := range cases {
		if got := dirDepth(dir); got != want {
			t.Errorf("dirDepth(%q) = %d, want %d", dir, got, want)
		}
	}
}

// TestRecursionCtl_TryMarkExpand 验证同一目录只能被展开一次（原子标记）。
func TestRecursionCtl_TryMarkExpand(t *testing.T) {
	rc := &recursionCtl{expanded: make(map[string]bool)}
	if !rc.tryMarkExpand("/admin/") {
		t.Error("first tryMarkExpand(/admin/) should return true")
	}
	if rc.tryMarkExpand("/admin/") {
		t.Error("second tryMarkExpand(/admin/) should return false (already expanded)")
	}
	if !rc.tryMarkExpand("/api/") {
		t.Error("tryMarkExpand(/api/) should return true (different dir)")
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

// ── 熔断器（circuitBreaker，实施文档 Task 9.1） ──────────────────────────────

// hangServer 模拟"丢包模式"目标：/hang 前缀的路径挂起 2s（超过客户端 1s 超时），
// 其余路径（通配符探测等）快速 404，保证 Precheck/采样不拖慢测试。
func hangServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/hang") {
			time.Sleep(2 * time.Second)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func hangWordlist(t *testing.T, n int) string {
	t.Helper()
	paths := make([]string, n)
	for i := range paths {
		paths[i] = fmt.Sprintf("/hang%03d", i)
	}
	return writeTempWordlist(t, paths)
}

// TestDirScanner_CircuitBreaker 验证连续超时达到阈值后扫描被取消：
// 无熔断时 200 条 × ~4.5s（3 次尝试×1s+退避）/ 3 线程 ≈ 300s；
// 有熔断时 5 个连续超时即触发（约两轮线程，~10s 量级）。
func TestDirScanner_CircuitBreaker(t *testing.T) {
	server := hangServer()
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":         3,
		"timeout":         1,
		"skip_builtin":    true,
		"circuit_breaker": 5,
	})
	task.Params["wordlists"] = hangWordlist(t, 200)

	start := time.Now()
	results, err := s.Run(context.Background(), task)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if elapsed > 60*time.Second {
		t.Fatalf("circuit breaker did not abort scan in time, elapsed %v", elapsed)
	}
	dr := results[0].Result.(*result.DirResult)
	if dr.Stats.ErrorReqs >= 100 {
		t.Errorf("expected scan aborted early by breaker, got %d errors", dr.Stats.ErrorReqs)
	}
}

// TestDirScanner_CircuitBreakerDisabled 验证 circuit_breaker=0 时熔断不生效，
// 全部路径都会走完（保持旧行为，Never break userspace）。
func TestDirScanner_CircuitBreakerDisabled(t *testing.T) {
	server := hangServer()
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":         5,
		"timeout":         1,
		"skip_builtin":    true,
		"circuit_breaker": 0, // 禁用
	})
	task.Params["wordlists"] = hangWordlist(t, 6)

	results, err := s.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	dr := results[0].Result.(*result.DirResult)
	if dr.Stats.ErrorReqs != 6 {
		t.Errorf("expected all 6 paths errored with breaker disabled, got %d", dr.Stats.ErrorReqs)
	}
}

// TestDirScanner_CircuitBreakerResetOnSuccess 验证成功响应清零连续超时计数：
// 超时与成功交替出现时计数无法累积到阈值，扫描不被熔断（偶发抖动不误判）。
func TestDirScanner_CircuitBreakerResetOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/hang") {
			time.Sleep(2 * time.Second)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	s := NewDirScanner()
	task := newTestTask(server.URL, map[string]interface{}{
		"threads":         1, // 单线程保证"超时-成功"严格交替
		"timeout":         1,
		"skip_builtin":    true,
		"circuit_breaker": 3,
	})
	// 交替构造：超时、成功、超时、成功……连续超时永远到不了 3
	var paths []string
	for i := 0; i < 4; i++ {
		paths = append(paths, fmt.Sprintf("/hang%03d", i), fmt.Sprintf("/ok%03d", i))
	}
	task.Params["wordlists"] = writeTempWordlist(t, paths)

	results, err := s.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	dr := results[0].Result.(*result.DirResult)
	// 4 个超时 + 4 个成功全部走完：未被熔断截断
	if dr.Stats.ErrorReqs != 4 {
		t.Errorf("expected 4 timeout errors, got %d", dr.Stats.ErrorReqs)
	}
	if dr.Stats.SuccessfulReqs != 4 {
		t.Errorf("expected 4 successful requests, got %d", dr.Stats.SuccessfulReqs)
	}
}
