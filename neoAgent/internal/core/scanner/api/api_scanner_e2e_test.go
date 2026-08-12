package api

// 本文件是 ApiScanner 的端到端验收测试，对应
// docs/API扫描-js提取/API扫描实施文档.md 第十五节的用例 A/B/C/D/E。
//
// 测试通过本地 httptest.NewServer 模拟站点，调用完整的 ApiScanner.Run()
// 链路（go-rod 渲染 + BFS + 提取 + 过滤），验证整条链路的行为符合
// 方案文档第一节的预期效果。
//
// 依赖真实 Chromium 进程：如果当前环境没有可用的 Chromium，所有用例
// 自动 Skip，不影响无浏览器的 CI 环境。

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"neoagent/internal/core/model"

	"github.com/go-rod/rod/lib/launcher"
)

// skipIfNoBrowser 检查 Chromium 是否可用，没有就跳过测试。
func skipIfNoBrowser(t *testing.T) {
	t.Helper()
	if _, ok := launcher.LookPath(); !ok {
		t.Skip("Chromium not found, skipping browser-dependent E2E test")
	}
}

// serverPort 从 httptest.Server 里取端口号（字符串形式）。
func serverPort(ts *httptest.Server) string {
	return fmt.Sprintf("%d", ts.Listener.Addr().(*net.TCPAddr).Port)
}

// runApiScan 用本地 httptest.Server 跑一次完整的 ApiScanner.Run()。
func runApiScan(t *testing.T, ts *httptest.Server, crawlDepth, maxJSFiles int) []*model.TaskResult {
	t.Helper()
	scanner := NewApiScanner()
	task := model.NewTask(model.TaskTypeApiScan, ts.URL)
	task.PortRange = serverPort(ts)
	task.Params["crawl_depth"] = crawlDepth
	task.Params["max_js_files"] = maxJSFiles

	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	return results
}

// 用例 A：单页高置信度接口提取。
// 本地页面包含 fetch('/api/user') 调用，期望结果里至少有一条
// Confidence=high、URL="/api/user" 的 APIInfo。
func TestApiScannerE2E_A_SinglePageHighConfidence(t *testing.T) {
	skipIfNoBrowser(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head></head><body>
<script>
fetch('/api/user');
</script>
</body></html>`)
	}))
	defer ts.Close()

	results := runApiScan(t, ts, 1, 20)
	if len(results) == 0 {
		t.Fatal("expected at least one ApiResult, got 0")
	}

	var found bool
	for _, r := range results {
		ar, ok := r.Result.(*model.ApiResult)
		if !ok {
			continue
		}
		for _, api := range ar.APIs {
			if api.Confidence == "high" && strings.Contains(api.URL, "/api/user") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected at least one high-confidence API /api/user, results: %+v", results)
	}
}

// 用例 B：BFS 深度爬取生效。
// 首页有链接指向 /page2，/page2 页面里有另一个接口调用。
// 期望返回结果包含 2 条 ApiResult（Depth 0 和 Depth 1 各一条）。
func TestApiScannerE2E_B_BFSCrawlSubPage(t *testing.T) {
	skipIfNoBrowser(t)

	mux := http.NewServeMux()
	var baseURL string

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><head></head><body>
<script>fetch('/api/home');</script>
<a href="%s/page2">page2</a>
</body></html>`, baseURL)
	})
	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head></head><body>
<script>fetch('/api/detail');</script>
</body></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	baseURL = ts.URL

	results := runApiScan(t, ts, 2, 20)

	if len(results) < 2 {
		t.Fatalf("expected at least 2 ApiResult (depth 0 + depth 1), got %d", len(results))
	}

	depths := make(map[int]bool)
	for _, r := range results {
		ar, ok := r.Result.(*model.ApiResult)
		if ok {
			depths[ar.Depth] = true
		}
	}
	if !depths[0] || !depths[1] {
		t.Errorf("expected results at both depth=0 and depth=1, got depths: %v", depths)
	}
}

// 用例 C：外链 JS 超限截断。
// 首页引用超过 MaxJSFiles（测试里设为 2）个外链 JS 文件，
// 期望首页 ApiResult.APIsTruncated == true。
func TestApiScannerE2E_C_JSFilesTruncated(t *testing.T) {
	skipIfNoBrowser(t)

	mux := http.NewServeMux()
	// 首页引用 5 个外链 JS，MaxJSFiles=2，应触发截断
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head>
<script src="/js/a.js"></script>
<script src="/js/b.js"></script>
<script src="/js/c.js"></script>
<script src="/js/d.js"></script>
<script src="/js/e.js"></script>
</head><body></body></html>`)
	})
	// 每个 JS 文件返回一个合法接口调用，确认截断后只提取了前 2 个文件的内容
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		n := name
		mux.HandleFunc("/js/"+n+".js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprintf(w, `fetch('/api/%s');`, n)
		})
	}

	ts := httptest.NewServer(mux)
	defer ts.Close()

	results := runApiScan(t, ts, 1, 2) // MaxJSFiles=2

	if len(results) == 0 {
		t.Fatal("expected at least one ApiResult, got 0")
	}

	var truncated bool
	for _, r := range results {
		ar, ok := r.Result.(*model.ApiResult)
		if ok && ar.APIsTruncated {
			truncated = true
		}
	}
	if !truncated {
		t.Error("expected APIsTruncated=true when JS files exceed MaxJSFiles=2, but none of the results had it set")
	}
}

// 用例 D：跨域链接不进入 BFS。
// 首页有一个指向外部域名的链接，期望结果集里只有当前域的 ApiResult，
// 不出现外部域名对应的 ApiResult。
func TestApiScannerE2E_D_CrossDomainLinkExcluded(t *testing.T) {
	skipIfNoBrowser(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head></head><body>
<script>fetch('/api/home');</script>
<a href="https://external.example.com/other">external</a>
</body></html>`)
	}))
	defer ts.Close()

	results := runApiScan(t, ts, 2, 20)

	for _, r := range results {
		ar, ok := r.Result.(*model.ApiResult)
		if !ok {
			continue
		}
		if strings.Contains(ar.URL, "external.example.com") {
			t.Errorf("expected cross-domain URL to be excluded from BFS, but found %s in results", ar.URL)
		}
	}
}

// 用例 E：目标不可达。
// 构造一个必然连接失败的地址，期望 Run() 不返回 error、不 panic，results 为空。
// 注：此用例与 api_scanner_test.go 中的单元测试等价，在这里作为 E2E 链路
// 的完整性验证（通过真实的 ApiScanner 和 BFS 执行路径）。
func TestApiScannerE2E_E_UnreachableTarget(t *testing.T) {
	skipIfNoBrowser(t)

	scanner := NewApiScanner()
	task := model.NewTask(model.TaskTypeApiScan, "http://127.0.0.1:1")
	task.PortRange = "1"
	task.Params["crawl_depth"] = 1
	task.Params["max_js_files"] = 5

	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() should not return error for unreachable target, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results for unreachable target, got %d", len(results))
	}
}
