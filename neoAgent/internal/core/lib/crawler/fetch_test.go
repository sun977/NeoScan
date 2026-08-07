package crawler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"neoagent/internal/core/lib/browser"

	"github.com/go-rod/rod"
)

// 本文件是 web扫描模块重构实施文档.md 步骤 3：验证 FetchAndCrawl 迁移后的
// 函数签名和调用方式是否正确传导了各项参数，不重新验证 go-rod/fallback
// 本身的正确性（那部分逻辑零改动，已经被 web_scanner.go 现有测试覆盖过，
// 见该文档第四节）。

// newTestLauncher 构造一个真实可用的 BrowserLauncher，与
// internal/core/scanner/web 现有测试（如 web_scanner_protocol_test.go）
// 依赖同一套系统 Chrome 环境，不额外引入 mock。
func newTestLauncher() *browser.BrowserLauncher {
	return browser.NewLauncher(browser.NewBrowserManager())
}

// TestFetchAndCrawl_OnPageReadyCalledExactlyOnce 验证 go-rod 渲染成功时，
// OnPageReady 回调确实在 WaitLoad 之后、page.Close() 之前被调用了恰好
// 一次，且传入的 *rod.Page 是可用的（能正常执行 Eval，不是已关闭的页面）。
func TestFetchAndCrawl_OnPageReadyCalledExactlyOnce(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>Fetch Test</title></head><body>ok</body></html>`)
	}))
	defer ts.Close()
	port := fmt.Sprintf("%d", ts.Listener.Addr().(*net.TCPAddr).Port)

	var callCount int32
	var pageUsable bool

	home, subPages, err := FetchAndCrawl(context.Background(), "127.0.0.1", port, "http",
		newTestLimiter(), newTestLauncher(), FetchOptions{
			OnPageReady: func(page *rod.Page) {
				atomic.AddInt32(&callCount, 1)
				// 页面此时应仍然可用（尚未 Close），Screenshot 不应 panic/报错。
				if _, errShot := page.Screenshot(true, nil); errShot == nil {
					pageUsable = true
				}
			},
		})
	if err != nil {
		t.Fatalf("FetchAndCrawl failed: %v", err)
	}
	if home.Title != "Fetch Test" {
		t.Errorf("expected title %q, got %q", "Fetch Test", home.Title)
	}
	if subPages != nil {
		t.Errorf("expected nil subPages when CrawlDepth<=0, got %d pages", len(subPages))
	}
	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("expected OnPageReady to be called exactly once, got %d", got)
	}
	if !pageUsable {
		t.Error("expected the *rod.Page passed to OnPageReady to still be usable (not yet closed)")
	}
}

// TestFetchAndCrawl_RodFailsFallbackSucceeds 验证 go-rod 路径拿不到 body 时
// （这里用一个只监听 TLS 的服务、且不给协议提示模拟"猜错协议"的降级场景），
// 触发 fallback 双发选优，最终 home.Body 非空、拿到了正确协议的内容——
// 对应 web_scanner_protocol_test.go 里
// TestProtocolDualFetch_HTTPGuessedButHTTPSOnly_PicksHTTPS 同样的场景，
// 只是这里直接调用 FetchAndCrawl 而不是通过 WebScanner.Run()。
func TestFetchAndCrawl_RodFailsFallbackSucceeds(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>TLS Only</title></head><body>secure</body></html>")
	}))
	defer ts.Close()
	port := fmt.Sprintf("%d", ts.Listener.Addr().(*net.TCPAddr).Port)

	home, _, err := FetchAndCrawl(context.Background(), "127.0.0.1", port, "", // 不传协议提示，触发猜测+双发
		newTestLimiter(), newTestLauncher(), FetchOptions{})
	if err != nil {
		t.Fatalf("FetchAndCrawl failed, expected dual-fetch to pick the working https response: %v", err)
	}
	if home.Body == "" {
		t.Fatal("expected non-empty home.Body after fallback dual-fetch")
	}
	if home.Title != "TLS Only" {
		t.Errorf("expected title %q (fetched via https fallback), got %q", "TLS Only", home.Title)
	}
}

// TestFetchAndCrawl_ProtocolGuessedAnd400TriggersVerification 验证协议是
// "猜"出来的且拿到 400 响应时，会触发双发验证——用只监听 TLS 的服务、不
// 显式指定协议，400（Go TLS 服务端对明文请求的提示）应该被识别为"协议
// 可能猜错了"，最终选中 https 那次成功的响应，而不是原样采信 400。
func TestFetchAndCrawl_ProtocolGuessedAnd400TriggersVerification(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>Should Win</title></head><body>secure</body></html>")
	}))
	defer ts.Close()
	port := fmt.Sprintf("%d", ts.Listener.Addr().(*net.TCPAddr).Port)

	home, _, err := FetchAndCrawl(context.Background(), "127.0.0.1", port, "", // 协议猜测：非标准端口默认猜 http
		newTestLimiter(), newTestLauncher(), FetchOptions{})
	if err != nil {
		t.Fatalf("FetchAndCrawl failed: %v", err)
	}
	if home.StatusCode == http.StatusBadRequest {
		t.Errorf("expected 400 to trigger verification and be replaced by the successful https fetch, got StatusCode=400 Title=%q", home.Title)
	}
	if home.Title != "Should Win" {
		t.Errorf("expected title %q after protocol verification, got %q", "Should Win", home.Title)
	}
}

// TestFetchAndCrawl_ExplicitProtocolNeverVerifies 验证协议是用户显式指定
// 时（不是猜的），即使拿到 400 响应也不触发双发验证，原样采信显式协议下
// 抓到的结果——尊重用户明确指定的协议意图。
func TestFetchAndCrawl_ExplicitProtocolNeverVerifies(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>Should Not Be Reached</title></head><body>secure</body></html>")
	}))
	defer ts.Close()
	port := fmt.Sprintf("%d", ts.Listener.Addr().(*net.TCPAddr).Port)

	home, _, err := FetchAndCrawl(context.Background(), "127.0.0.1", port, "http", // 显式指定 http，即使是错的
		newTestLimiter(), newTestLauncher(), FetchOptions{})
	if err != nil {
		t.Fatalf("FetchAndCrawl failed unexpectedly: %v (a 400 response is still a successful fetch)", err)
	}
	if home.Title == "Should Not Be Reached" {
		t.Fatal("explicit protocol=http should never be silently corrected to https")
	}
	if home.StatusCode != http.StatusBadRequest {
		t.Errorf("expected the raw protocol-mismatch 400 response, got status %d title %q", home.StatusCode, home.Title)
	}
}

// TestFetchAndCrawl_CrawlDepthZero_NoSubPages 验证 CrawlDepth<=0 时不触发
// 任何 BFS 请求：即使首页有链接，subPages 也应该是空/nil。
func TestFetchAndCrawl_CrawlDepthZero_NoSubPages(t *testing.T) {
	var bfsHit int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><a href="/child">child</a></body></html>`)
	})
	mux.HandleFunc("/child", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&bfsHit, 1)
		fmt.Fprint(w, "child page")
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	port := fmt.Sprintf("%d", ts.Listener.Addr().(*net.TCPAddr).Port)

	home, subPages, err := FetchAndCrawl(context.Background(), "127.0.0.1", port, "http",
		newTestLimiter(), newTestLauncher(), FetchOptions{CrawlDepth: 0})
	if err != nil {
		t.Fatalf("FetchAndCrawl failed: %v", err)
	}
	if len(home.SeedLinks) == 0 {
		t.Fatal("expected home page to have seed links for this test to be meaningful")
	}
	if len(subPages) != 0 {
		t.Errorf("expected no subPages when CrawlDepth<=0, got %d", len(subPages))
	}
	if atomic.LoadInt32(&bfsHit) != 0 {
		t.Error("expected /child to never be requested when CrawlDepth<=0")
	}
}

// TestFetchAndCrawl_CrawlDepthPositive_ReturnsSubPages 验证 CrawlDepth>0
// 且首页有种子链接时，subPages 非空——BFS 确实被触发了。
func TestFetchAndCrawl_CrawlDepthPositive_ReturnsSubPages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><a href="/child">child</a></body></html>`)
	})
	mux.HandleFunc("/child", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "child page")
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	port := fmt.Sprintf("%d", ts.Listener.Addr().(*net.TCPAddr).Port)

	home, subPages, err := FetchAndCrawl(context.Background(), "127.0.0.1", port, "http",
		newTestLimiter(), newTestLauncher(), FetchOptions{CrawlDepth: 2})
	if err != nil {
		t.Fatalf("FetchAndCrawl failed: %v", err)
	}
	if len(home.SeedLinks) == 0 {
		t.Fatal("expected home page to have seed links for this test to be meaningful")
	}
	if len(subPages) == 0 {
		t.Fatal("expected non-empty subPages when CrawlDepth>0 and seed links exist")
	}
}

// TestFetchAndCrawl_BothFail_ReturnsError 验证 go-rod 和 fallback 双发都
// 失败时（目标端口拒绝连接），FetchAndCrawl 返回 error 而不是一个空壳
// 成功结果。
func TestFetchAndCrawl_BothFail_ReturnsError(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate a throwaway port: %v", err)
	}
	port := fmt.Sprintf("%d", lis.Addr().(*net.TCPAddr).Port)
	lis.Close() // 立刻关闭，端口变为"拒绝连接"状态

	_, _, err = FetchAndCrawl(context.Background(), "127.0.0.1", port, "http",
		newTestLimiter(), newTestLauncher(), FetchOptions{})
	if err == nil {
		t.Fatal("expected FetchAndCrawl to fail for an unreachable port")
	}
}
