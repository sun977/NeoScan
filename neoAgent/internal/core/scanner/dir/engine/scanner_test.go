package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNormalizeDynamic 验证 UUID/hex/时间戳/Base64 被正确替换。
func TestNormalizeDynamic(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "uuid",
			input: "id=550e8400-e29b-41d4-a716-446655440000 done",
			want:  "id=[UUID] done",
		},
		{
			name:  "hex",
			input: "hash=0123456789abcdef0123456789abcdef end",
			want:  "hash=[HEX] end",
		},
		{
			name:  "timestamp",
			input: "ts=1699999999999 ok",
			want:  "ts=[TS] ok",
		},
		{
			name:  "base64",
			input: "token=QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVphYmNkZWZnaGlqa2xtbg== end",
			want:  "token=[B64] end",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := normalizeDynamic(c.input)
			if got != c.want {
				t.Errorf("normalizeDynamic(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

// TestExtractStaticPatterns 验证两个相同结构但动态内容不同的响应能提取到共同词。
func TestExtractStaticPatterns(t *testing.T) {
	body1 := "Error 404 request-id=550e8400-e29b-41d4-a716-446655440000 not found"
	body2 := "Error 404 request-id=660e8400-e29b-41d4-a716-446655440001 not found"

	common, baseCount := extractStaticPatterns(body1, body2)

	if baseCount == 0 {
		t.Fatal("expected non-zero base word count")
	}

	wantWords := []string{"Error", "404", "not", "found"}
	for _, w := range wantWords {
		found := false
		for _, c := range common {
			if c == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected common word %q in %v", w, common)
		}
	}

	// request-id 各不相同，归一化后都变为 [UUID]，因此也应作为共同词出现，
	// 但原始 "request-id=550e8400..." 不应完整出现（已被替换）。
	for _, c := range common {
		if c == "request-id=550e8400-e29b-41d4-a716-446655440000" {
			t.Error("dynamic UUID content should not appear as a static pattern")
		}
	}
}

// TestWildcardScanner_CDN 验证 mock 服务器对任意路径返回同一 404 页面时，Check 最终返回 false。
func TestWildcardScanner_CDN(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "<html><body>Page not found: %s</body></html>", r.URL.Path)
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{Timeout: 5 * time.Second})
	scanner := NewWildcardScanner(req, server.URL)

	ctx := context.Background()
	var lastMatch bool
	var lastReason string
	for i := 0; i < 20; i++ {
		path := fmt.Sprintf("/random-path-%d", i)
		resp, err := req.Do(ctx, server.URL, path)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		lastMatch, lastReason = scanner.Check(ctx, PathPrefixOf(path), resp)
	}

	if lastMatch {
		t.Errorf("expected wildcard CDN response to be rejected, got match=true reason=%s", lastReason)
	}
}

// TestWildcardScanner_Unique 验证独特内容能被判定为命中。
func TestWildcardScanner_Unique(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "<html><body>Admin Dashboard Login Panel Secret Content Unique</body></html>")
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "<html><body>404 page not found here</body></html>")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{Timeout: 5 * time.Second})
	scanner := NewWildcardScanner(req, server.URL)

	ctx := context.Background()

	// 先用若干随机路径预热采样池（都是 404），确保 /admin/ 前缀的采样池已建立。
	for i := 0; i < 6; i++ {
		path := fmt.Sprintf("/nope-%d", i)
		resp, err := req.Do(ctx, server.URL, path)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		scanner.Check(ctx, PathPrefixOf(path), resp)
	}

	resp, err := req.Do(ctx, server.URL, "/admin")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	match, reason := scanner.Check(ctx, PathPrefixOf("/admin"), resp)
	if !match {
		t.Errorf("expected /admin to be unique match, got match=false reason=%s", reason)
	}
}

// TestWildcardScanner_NoRequesterFallsBackToUnique 验证 requester 为 nil
// （无法发起探测采样）时的兜底行为。
//
// 对齐 dirsearch 的两阶段模型后，采样与判定不再交织在每次 Check 调用里：
// 采样在第一次 Check 时通过 sync.Once 同步完成（Check 内部阻塞直到采样
// 结束），不存在"前几次调用处于 sampling 中间态"这种运行时状态。
// 若采样因缺少 requester 而彻底失败，Check 应保守地判定为 unique（放行），
// 而不是把所有响应都当作 sampling 静默丢弃——通配符引擎自身故障不应该
// 导致真实命中被吞掉，见 Check 的实现注释。
func TestWildcardScanner_NoRequesterFallsBackToUnique(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	scanner := NewWildcardScanner(nil, server.URL)
	ctx := context.Background()

	req := NewRequester(RequesterConfig{Timeout: 5 * time.Second})
	for i := 0; i < defaultSampleThreshold+2; i++ {
		path := fmt.Sprintf("/sample-%d", i)
		resp, err := req.Do(ctx, server.URL, path)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		match, reason := scanner.Check(ctx, "/", resp)
		if !match || reason != "unique" {
			t.Errorf("iteration %d: expected (true, unique) when sampling is unavailable, got (%v, %s)", i, match, reason)
		}
	}
}

// TestWildcardScanner_SamplesOnlyOncePerPrefix 验证同一 pathPrefix 的采样
// 只会被执行一次（sync.Once 语义），第一次 Check 调用内部阻塞完成全部
// 采样探测，后续调用直接复用已固化的 staticPatterns，不再重复发起探测。
func TestWildcardScanner_SamplesOnlyOncePerPrefix(t *testing.T) {
	var probeCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/real-") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "real content for %s", r.URL.Path)
			return
		}
		atomic.AddInt32(&probeCount, 1)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found, padding padding padding padding padding")
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{Timeout: 5 * time.Second})
	scanner := NewWildcardScanner(req, server.URL)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		path := fmt.Sprintf("/real-%d", i)
		resp, err := req.Do(ctx, server.URL, path)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		match, reason := scanner.Check(ctx, "/", resp)
		if !match {
			t.Errorf("iteration %d: expected real content to be unique, got match=false reason=%s", i, reason)
		}
	}

	if got := atomic.LoadInt32(&probeCount); got != int32(defaultSampleThreshold) {
		t.Errorf("expected exactly %d probe requests (one-time sampling), got %d", defaultSampleThreshold, got)
	}
}

// TestWildcardScanner_Concurrent 验证并发 Check 不发生 data race。
func TestWildcardScanner_Concurrent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "not found: %s", r.URL.Path)
	}))
	defer server.Close()

	req := NewRequester(RequesterConfig{Timeout: 5 * time.Second})
	scanner := NewWildcardScanner(req, server.URL)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			path := fmt.Sprintf("/concurrent-%d", idx)
			resp, err := req.Do(ctx, server.URL, path)
			if err != nil {
				return
			}
			scanner.Check(ctx, PathPrefixOf(path), resp)
		}(i)
	}
	wg.Wait()
}
