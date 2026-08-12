package api

import "testing"

func TestFilterAPICandidates(t *testing.T) {
	input := []candidate{
		{URL: "/api/user/info", Confidence: "high"},
		{URL: "/api/user/info", Confidence: "high"}, // 重复，应被去重
		{URL: "/static/app.js", Confidence: "low"},  // 静态资源，应被排除
		{URL: "location.href", Confidence: "low"},   // 黑名单命中，应被排除
		{URL: "  /api/order/list  ", Confidence: "low"}, // 前后空白，应被清洗
	}

	got := filterAPICandidates(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 candidates after filter, got %d: %+v", len(got), got)
	}

	urls := make(map[string]bool)
	for _, c := range got {
		urls[c.URL] = true
	}
	if !urls["/api/user/info"] || !urls["/api/order/list"] {
		t.Errorf("unexpected filtered result: %+v", got)
	}
}

func TestFilterAPICandidates_StaticResourceSuffix(t *testing.T) {
	input := []candidate{
		{URL: "/assets/main.css?v=123"},
		{URL: "/assets/logo.png"},
		{URL: "/api/real/endpoint"},
	}
	got := filterAPICandidates(input)
	if len(got) != 1 || got[0].URL != "/api/real/endpoint" {
		t.Fatalf("expected only /api/real/endpoint to survive, got %+v", got)
	}
}
