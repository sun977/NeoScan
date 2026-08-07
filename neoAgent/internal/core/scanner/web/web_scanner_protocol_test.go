package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"neoagent/internal/core/model"
)

// --- 集成测试：Run() 端到端验证协议双发选优的完整行为 ---
//
// flipProtocol / isProtocolGuessed / pickBestFetchOutcome 这三个纯函数已经
// 随 web扫描模块重构实施文档.md 步骤 2 下沉到 crawler.FetchAndCrawl 内部
// （neoagent/internal/core/lib/crawler/fetch.go），对应的白盒单元测试
// 迁移到了 crawler 包的 fetch_test.go（TestFetchAndCrawl_ProtocolGuessedAnd400TriggersVerification /
// TestFetchAndCrawl_ExplicitProtocolNeverVerifies 等），不再需要在这里保留
// 对已删除私有函数的直接调用。本文件保留的是 WebScanner.Run() 端到端黑盒
// 集成测试，验证 runOnePort 改为调用 FetchAndCrawl 之后协议双发这一行为
// 在 WebScanner 这一层的可见结果没有回归。

// TestProtocolDualFetch_HTTPGuessedButHTTPSOnly_PicksHTTPS 起一个只监听 TLS 的
// httptest.NewTLSServer，不显式指定协议（模拟 normalizeURL 对非常规端口默认
// 猜成 http 的场景），断言双发之后选中的是 https 那次成功的响应。
func TestProtocolDualFetch_HTTPGuessedButHTTPSOnly_PicksHTTPS(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>TLS Only</title></head><body>secure</body></html>")
	}))
	defer ts.Close()

	port := ts.Listener.Addr().(*net.TCPAddr).Port

	scanner := NewWebScanner()
	scanner.ensureInit()

	task := &model.Task{
		ID:        "test-protocol-dualfetch",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d", port),
		// 不传 "protocol"，让 normalizeURL 对这个非常规端口默认猜成 http，
		// 触发双发条件（协议是猜出来的）。
	}

	// fallbackFetch 内部对 https 用的是 InsecureSkipVerify: true，测试自带
	// 的自签名证书不会导致额外的证书校验失败，http 一侧会因协议不匹配失败或
	// 拿到不可用响应，https 一侧会成功，双发选优后应该拿到 https 的结果。
	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run failed, expected dual-fetch to pick the working https response: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(results))
	}
	wr, ok := results[0].Result.(*model.WebResult)
	if !ok {
		t.Fatalf("result is not *model.WebResult: %+v", results[0].Result)
	}
	if wr.Title != "TLS Only" {
		t.Errorf("expected title 'TLS Only' (fetched via https), got %q", wr.Title)
	}
}

// TestProtocolDualFetch_ExplicitProtocolHint_NeverDualFetches 显式指定了协议，
// 即使拿到的是协议不匹配的 400 响应，也不应该触发另一协议的双发验证——用一个
// 只监听 TLS 的服务，但显式声明 protocol=http，断言最终结果就是 http 直接
// 请求 TLS 端口拿到的那个 400 响应本身（Go TLS 服务端对明文请求的提示文本），
// 而不是被"偷偷"纠正成 https 抓到的正确页面。
//
// 这里不能用"Run 返回 error"来断言"没有发生双发"——go-rod/fallback 路径下
// 拿到一个 400 响应本身就是一次"成功的抓取"（有 body、有明确状态码），
// Run() 不会也不应该因为状态码是 400 就返回顶层 error。用响应内容本身
// （是不是那段协议不匹配的提示文本）来断言才是符合语义的验证方式。
func TestProtocolDualFetch_ExplicitProtocolHint_NeverDualFetches(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>Should Not Be Reached</title></head><body>secure</body></html>")
	}))
	defer ts.Close()

	port := ts.Listener.Addr().(*net.TCPAddr).Port

	scanner := NewWebScanner()
	scanner.ensureInit()

	task := &model.Task{
		ID:        "test-protocol-explicit",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d", port),
		Params: map[string]interface{}{
			"protocol": "http", // 显式指定，即使是错的，也不应该被自动纠正
		},
	}

	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run failed unexpectedly: %v (a 400 response is still a successful fetch)", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(results))
	}
	wr, ok := results[0].Result.(*model.WebResult)
	if !ok {
		t.Fatalf("result is not *model.WebResult: %+v", results[0].Result)
	}
	if wr.Title == "Should Not Be Reached" {
		t.Fatal("explicit protocol=http should never be silently corrected to https")
	}
	if wr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected the raw protocol-mismatch 400 response, got status %d title %q", wr.StatusCode, wr.Title)
	}
}

// TestProtocolDualFetch_TCPUnreachable_FailsFast 目标端口直接拒绝连接
// （net.Dial 阶段失败），双发场景下两侧都会快速失败，断言总耗时不应该出现
// "长时间等待"的现象——用总耗时作为间接证据，证明双发是并发而不是串行。
func TestProtocolDualFetch_TCPUnreachable_FailsFast(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate a throwaway port: %v", err)
	}
	unreachablePort := lis.Addr().(*net.TCPAddr).Port
	lis.Close() // 立刻关闭，端口变为"拒绝连接"状态

	scanner := NewWebScanner()
	scanner.ensureInit()

	task := &model.Task{
		ID:        "test-protocol-tcp-unreachable",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d", unreachablePort),
	}

	start := time.Now()
	_, err = scanner.Run(context.Background(), task)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected Run to fail for an unreachable port")
	}
	// dial 失败（connection refused）通常是毫秒级返回，即使并发双发两侧
	// 都失败，也不该接近 fallbackFetch 的 15s 超时量级。
	if elapsed > 5*time.Second {
		t.Errorf("expected fast failure for connection-refused case, took %v", elapsed)
	}
}

// TestProtocolDualFetch_BothFail_ReturnsError 用一个立刻返回垃圾数据后关闭
// 连接的 TCP 服务，模拟"连接建立成功，但既不是有效的 HTTP 也不是有效的 TLS"
// 的场景——双发两侧都会失败，断言最终返回 error。
func TestProtocolDualFetch_BothFail_ReturnsError(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate a throwaway port: %v", err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	go func() {
		for {
			conn, acceptErr := lis.Accept()
			if acceptErr != nil {
				return
			}
			conn.Write([]byte("not a valid http or tls response"))
			conn.Close()
		}
	}()
	defer lis.Close()

	scanner := NewWebScanner()
	scanner.ensureInit()

	task := &model.Task{
		ID:        "test-protocol-both-fail",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d", port),
	}

	_, err = scanner.Run(context.Background(), task)
	if err == nil {
		t.Fatal("expected Run to fail when both http and https attempts fail")
	}
}
