package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"neoagent/internal/core/model"
)

// TestParsePortsForScan 验证端口范围字符串解析 + 去重的行为，覆盖
// Run() 编排层判断"单端口兼容路径 vs 多端口并发路径"依赖的核心函数。
func TestParsePortsForScan(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []int
	}{
		{"empty", "", nil},
		{"single", "80", []int{80}},
		{"list", "80,443", []int{80, 443}},
		{"range", "8000-8002", []int{8000, 8001, 8002}},
		{"dedup", "80,80,443", []int{80, 443}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parsePortsForScan(c.input)
			if len(got) != len(c.want) {
				t.Fatalf("parsePortsForScan(%q) = %v, want %v", c.input, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("parsePortsForScan(%q) = %v, want %v", c.input, got, c.want)
				}
			}
		})
	}
}

// TestWebScanner_MultiPort_ScansEachPortIndependently 起两个各自独立的
// HTTP mock server，把它们的端口用逗号拼成 task.PortRange，断言 Run()
// 对两个端口都发起了探测并各自拿到了正确的首页结果——验证多端口编排层
// 真正做到了并发探测，而不是像 Sprint 0-5 那样把 "port1,port2" 整串
// 当成一个不认识的字符串，落进默认协议猜测分支只测一次。
func TestWebScanner_MultiPort_ScansEachPortIndependently(t *testing.T) {
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>Port One</title></head><body>one</body></html>")
	}))
	defer ts1.Close()
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>Port Two</title></head><body>two</body></html>")
	}))
	defer ts2.Close()

	port1 := ts1.Listener.Addr().(*net.TCPAddr).Port
	port2 := ts2.Listener.Addr().(*net.TCPAddr).Port

	scanner := NewWebScanner()
	scanner.ensureInit()

	task := &model.Task{
		ID:        "test-multiport",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d,%d", port1, port2),
		Params: map[string]interface{}{
			"protocol": "http", // 都是明文 mock server，显式指定协议避免双发引入不确定性
		},
	}

	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (one per port), got %d", len(results))
	}

	gotTitles := make(map[string]bool)
	for _, r := range results {
		wr, ok := r.Result.(*model.WebResult)
		if !ok {
			t.Fatalf("result is not *model.WebResult: %+v", r.Result)
		}
		gotTitles[wr.Title] = true
	}
	if !gotTitles["Port One"] || !gotTitles["Port Two"] {
		t.Fatalf("expected both 'Port One' and 'Port Two' titles, got %v", gotTitles)
	}
}

// TestWebScanner_MultiPort_PartialFailureStillReturnsSuccessfulResults 验证
// 多端口场景下"部分端口失败"不应该掩盖"部分端口成功"——一个不可达端口和一个
// 正常端口放在同一个 task 里，断言最终仍然拿到正常端口的结果且不返回顶层 error。
func TestWebScanner_MultiPort_PartialFailureStillReturnsSuccessfulResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><head><title>Alive</title></head><body>alive</body></html>")
	}))
	defer ts.Close()
	alivePort := ts.Listener.Addr().(*net.TCPAddr).Port

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate throwaway port: %v", err)
	}
	deadPort := lis.Addr().(*net.TCPAddr).Port
	lis.Close()

	scanner := NewWebScanner()
	scanner.ensureInit()

	task := &model.Task{
		ID:        "test-multiport-partial-failure",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d,%d", alivePort, deadPort),
		Params: map[string]interface{}{
			"protocol": "http",
		},
	}

	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("expected partial success to not surface as a top-level error, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result (from the alive port), got %d", len(results))
	}
	wr, ok := results[0].Result.(*model.WebResult)
	if !ok {
		t.Fatalf("result is not *model.WebResult: %+v", results[0].Result)
	}
	if wr.Title != "Alive" {
		t.Errorf("expected title 'Alive', got %q", wr.Title)
	}
}
