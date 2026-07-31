package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"neoagent/internal/core/lib/network/qos"
)

func newTestLimiter() *qos.AdaptiveLimiter {
	return qos.NewAdaptiveLimiter(10, 1, 20)
}

func newTestOptions() Options {
	return Options{
		MaxDepth:    2,
		MaxPages:    200,
		Concurrency: 5,
		Timeout:     5 * time.Second,
		// AllowCrossHost 不设置，零值 false，即默认同源限制生效
	}
}

// TestCrawl_BasicBFS 构造 3 层链接站点（首页种子 -> 2 个二级页 -> 每个二级页各 2 个三级页），
// MaxDepth=2，断言最终 len(pages) 精确等于二级+三级页面总数，且没有重复。
func TestCrawl_BasicBFS(t *testing.T) {
	mux := http.NewServeMux()
	var baseURL string

	mux.HandleFunc("/level2-1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><a href="%s/level3-1-1">l</a><a href="%s/level3-1-2">l</a></body></html>`, baseURL, baseURL)
	})
	mux.HandleFunc("/level2-2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><a href="%s/level3-2-1">l</a><a href="%s/level3-2-2">l</a></body></html>`, baseURL, baseURL)
	})
	for _, p := range []string{"/level3-1-1", "/level3-1-2", "/level3-2-1", "/level3-2-2"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `<html><body>leaf</body></html>`)
		})
	}

	ts := httptest.NewServer(mux)
	defer ts.Close()
	baseURL = ts.URL

	seedLinks := []string{ts.URL + "/level2-1", ts.URL + "/level2-2"}

	cr := New(newTestOptions(), newTestLimiter())
	pages := cr.Crawl(context.Background(), ts.URL, seedLinks)

	if len(pages) != 6 { // 2 个二级页 + 4 个三级页
		t.Fatalf("expected 6 pages, got %d", len(pages))
	}

	seen := make(map[string]struct{})
	for _, p := range pages {
		if _, dup := seen[p.URL]; dup {
			t.Fatalf("duplicate page found: %s", p.URL)
		}
		seen[p.URL] = struct{}{}
	}
}

// TestCrawl_MaxDepthRespected 同一个 3 层站点，MaxDepth=1，断言只抓到二级页面，三级页面不出现在结果里。
func TestCrawl_MaxDepthRespected(t *testing.T) {
	mux := http.NewServeMux()
	var baseURL string

	mux.HandleFunc("/level2-1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><a href="%s/level3-1-1">l</a></body></html>`, baseURL)
	})
	mux.HandleFunc("/level2-2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><a href="%s/level3-2-1">l</a></body></html>`, baseURL)
	})
	mux.HandleFunc("/level3-1-1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html><body>leaf</body></html>`)
	})
	mux.HandleFunc("/level3-2-1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html><body>leaf</body></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	baseURL = ts.URL

	seedLinks := []string{ts.URL + "/level2-1", ts.URL + "/level2-2"}

	opts := newTestOptions()
	opts.MaxDepth = 1
	cr := New(opts, newTestLimiter())
	pages := cr.Crawl(context.Background(), ts.URL, seedLinks)

	if len(pages) != 2 {
		t.Fatalf("expected 2 pages (depth=1 only), got %d", len(pages))
	}
	for _, p := range pages {
		if p.Depth != 1 {
			t.Fatalf("expected all pages at depth 1, got depth %d for %s", p.Depth, p.URL)
		}
	}
}

// TestCrawl_Deduplication 构造两个互相链接指向同一个 URL（不同 ?a=1&b=2 和 ?b=2&a=1 顺序）的页面，
// 断言只被访问一次。
func TestCrawl_Deduplication(t *testing.T) {
	var hitCount int32
	mux := http.NewServeMux()
	mux.HandleFunc("/target", func(w http.ResponseWriter, r *http.Request) {
		hitCount++
		fmt.Fprintln(w, `<html><body>target</body></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 同一目标 URL 的两种不同查询参数顺序
	seedLinks := []string{
		ts.URL + "/target?a=1&b=2",
		ts.URL + "/target?b=2&a=1",
	}

	cr := New(newTestOptions(), newTestLimiter())
	pages := cr.Crawl(context.Background(), ts.URL, seedLinks)

	if len(pages) != 1 {
		t.Fatalf("expected 1 page after dedup, got %d", len(pages))
	}
}

// TestCrawl_SameHostOnlyByDefault 种子链接里混入一个外部域名的链接，断言在不做任何显式配置
// （零值 Options，只设置 MaxDepth，和生产环境 web_scanner.go 的真实调用方式完全一致）的情况下，
// 外部链接依然不出现在结果里——同源限制必须是零值安全的默认行为，不能依赖调用方记得显式开启。
func TestCrawl_SameHostOnlyByDefault(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/internal", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html><body>internal</body></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	seedLinks := []string{
		ts.URL + "/internal",
		"http://external.example.com/page",
	}

	// 刻意模拟生产环境 web_scanner.go 里的真实写法：crawler.Options{MaxDepth: depth}，
	// 不显式设置任何跨域相关字段。
	cr := New(Options{MaxDepth: 2}, newTestLimiter())
	pages := cr.Crawl(context.Background(), ts.URL, seedLinks)

	if len(pages) != 1 {
		t.Fatalf("expected 1 page (external filtered), got %d", len(pages))
	}
	if pages[0].URL != ts.URL+"/internal" {
		t.Fatalf("unexpected page URL: %s", pages[0].URL)
	}
}

// TestCrawl_AllowCrossHost 显式开启 AllowCrossHost 后，外部域名链接应当被正常爬取。
func TestCrawl_AllowCrossHost(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/internal", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html><body>internal</body></html>`)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	externalMux := http.NewServeMux()
	externalMux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html><body>external</body></html>`)
	})
	externalTS := httptest.NewServer(externalMux)
	defer externalTS.Close()

	seedLinks := []string{
		ts.URL + "/internal",
		externalTS.URL + "/page",
	}

	opts := newTestOptions()
	opts.AllowCrossHost = true
	cr := New(opts, newTestLimiter())
	pages := cr.Crawl(context.Background(), ts.URL, seedLinks)

	if len(pages) != 2 {
		t.Fatalf("expected 2 pages (internal + external), got %d", len(pages))
	}
}

// TestCrawl_MaxPagesLimit 构造一个链接数超过 MaxPages（设小一点，比如 5）的站点，
// 断言最终页面数不超过 MaxPages。
func TestCrawl_MaxPagesLimit(t *testing.T) {
	mux := http.NewServeMux()
	for i := 0; i < 20; i++ {
		mux.HandleFunc(fmt.Sprintf("/page%d", i), func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `<html><body>leaf</body></html>`)
		})
	}

	ts := httptest.NewServer(mux)
	defer ts.Close()

	var seedLinks []string
	for i := 0; i < 20; i++ {
		seedLinks = append(seedLinks, fmt.Sprintf("%s/page%d", ts.URL, i))
	}

	opts := newTestOptions()
	opts.MaxPages = 5
	cr := New(opts, newTestLimiter())
	pages := cr.Crawl(context.Background(), ts.URL, seedLinks)

	if len(pages) > 5 {
		t.Fatalf("expected at most 5 pages (MaxPages limit), got %d", len(pages))
	}
}

// TestCrawl_ConcurrencyNoDeadlock 种子链接数量大于 Concurrency（种子 20 个、并发 5），
// 跑 3 次，用 -race 断言不死锁、不 data race、每次运行结果数量一致。
func TestCrawl_ConcurrencyNoDeadlock(t *testing.T) {
	mux := http.NewServeMux()
	for i := 0; i < 20; i++ {
		mux.HandleFunc(fmt.Sprintf("/p%d", i), func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `<html><body>leaf</body></html>`)
		})
	}

	ts := httptest.NewServer(mux)
	defer ts.Close()

	var seedLinks []string
	for i := 0; i < 20; i++ {
		seedLinks = append(seedLinks, fmt.Sprintf("%s/p%d", ts.URL, i))
	}

	opts := newTestOptions()
	opts.Concurrency = 5
	opts.MaxDepth = 1

	var counts []int
	for i := 0; i < 3; i++ {
		done := make(chan []*Page, 1)
		go func() {
			cr := New(opts, newTestLimiter())
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			done <- cr.Crawl(ctx, ts.URL, seedLinks)
		}()

		select {
		case pages := <-done:
			counts = append(counts, len(pages))
		case <-time.After(20 * time.Second):
			t.Fatal("Crawl deadlocked: did not return within timeout")
		}
	}

	sort.Ints(counts)
	for _, c := range counts {
		if c != 20 {
			t.Fatalf("expected 20 pages every run, got counts=%v", counts)
		}
	}
}
