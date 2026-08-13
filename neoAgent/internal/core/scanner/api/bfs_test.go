package api

import "testing"

func TestNormalizeKey_QueryParamOrderInsensitive(t *testing.T) {
	a := normalizeKey("http://example.com/x?b=2&a=1")
	b := normalizeKey("http://example.com/x?a=1&b=2")
	if a != b {
		t.Errorf("expected same normalized key regardless of query param order, got %q vs %q", a, b)
	}
}

func TestNormalizeKey_StripsFragment(t *testing.T) {
	got := normalizeKey("http://example.com/x#section")
	if got != "http://example.com/x" {
		t.Errorf("expected fragment stripped, got %q", got)
	}
}

func TestApiCrawler_Enqueue_DedupeAndScope(t *testing.T) {
	// launcher/limiter 传 nil，单元测试只验证 enqueue 的去重和 Scope 逻辑，
	// 不调用 fetchPage，不需要真实基础设施实例。
	c := newAPICrawler(nil, nil, 2, 20, 5)
	c.seedHost = "example.com"
	c.queue = make(chan *crawlItem, 10)

	c.enqueue("http://example.com/a", 1)
	c.enqueue("http://example.com/a", 1) // 重复，应被去重
	c.enqueue("http://other.com/b", 1)   // 跨域，应被 Scope 拒绝

	if len(c.visited) != 1 {
		t.Fatalf("expected 1 visited entry after dedupe+scope filtering, got %d: %+v", len(c.visited), c.visited)
	}
}
