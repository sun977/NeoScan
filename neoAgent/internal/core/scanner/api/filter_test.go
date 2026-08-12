package api

import "testing"

func TestFilterAPICandidates(t *testing.T) {
	input := []candidate{
		{URL: "/api/user/info", Confidence: "high", Source: "https://example.com/app.js"},
		{URL: "/api/user/info", Confidence: "high", Source: "https://example.com/app.js"}, // 重复，应被去重
		{URL: "/static/app.js", Confidence: "low", Source: "inline"},                      // 静态资源后缀，应被排除
		{URL: "location.href", Confidence: "low", Source: "inline"},                       // 黑名单命中，应被排除
		// low 置信度 + 有效 JS 来源 + 两段路径 → 应保留
		{URL: "  /api/order/list  ", Confidence: "low", Source: "https://example.com/chunk.js"},
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

// TestFilterAPICandidates_LowConfidenceSourceFilter 验证 low 置信度的 Source
// 来源过滤规则：只有来自外链 JS 文件或内联 <script> 的 low 候选项才能通过。
func TestFilterAPICandidates_LowConfidenceSourceFilter(t *testing.T) {
	cases := []struct {
		name    string
		c       candidate
		survive bool
	}{
		{
			name:    "low+内联script→保留",
			c:       candidate{URL: "/api/v1/users", Confidence: "low", Source: "inline"},
			survive: true,
		},
		{
			name:    "low+外链JS文件→保留",
			c:       candidate{URL: "/api/v1/users", Confidence: "low", Source: "https://cdn.example.com/app.chunk.js"},
			survive: true,
		},
		{
			name:    "low+页面HTML来源→丢弃",
			c:       candidate{URL: "/api/v1/users", Confidence: "low", Source: "https://example.com/page"},
			survive: false,
		},
		{
			name:    "low+空来源→丢弃",
			c:       candidate{URL: "/api/v1/users", Confidence: "low", Source: ""},
			survive: false,
		},
		{
			name:    "high+页面HTML来源→保留（high不受Source过滤限制）",
			c:       candidate{URL: "/api/v1/users", Confidence: "high", Source: "https://example.com/page"},
			survive: true,
		},
	}

	for _, tc := range cases {
		got := filterAPICandidates([]candidate{tc.c})
		survived := len(got) > 0
		if survived != tc.survive {
			t.Errorf("[%s] want survive=%v, got survive=%v (result: %+v)", tc.name, tc.survive, survived, got)
		}
	}
}

// TestFilterAPICandidates_LowConfidenceSegments 验证 low 置信度路径段数过滤。
func TestFilterAPICandidates_LowConfidenceSegments(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		survive bool
	}{
		{name: "单段路径/en/→丢弃", url: "/en/", survive: false},
		{name: "单段路径/v1/→丢弃", url: "/v1/", survive: false},
		{name: "两段路径/api/users→保留", url: "/api/users", survive: true},
		{name: "三段路径/api/v1/orders→保留", url: "/api/v1/orders", survive: true},
	}

	for _, tc := range cases {
		c := candidate{URL: tc.url, Confidence: "low", Source: "inline"}
		got := filterAPICandidates([]candidate{c})
		survived := len(got) > 0
		if survived != tc.survive {
			t.Errorf("[%s] want survive=%v, got survive=%v", tc.name, tc.survive, survived)
		}
	}
}

// TestFilterAPICandidates_MediumConfidencePageSuffix 验证 medium 置信度
// 对页面后缀 URL 的过滤。
func TestFilterAPICandidates_MediumConfidencePageSuffix(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		survive bool
	}{
		{name: "medium+.html→丢弃", url: "https://example.com/about.html", survive: false},
		{name: "medium+.php→丢弃", url: "https://example.com/index.php", survive: false},
		{name: "medium+纯路径→保留", url: "https://testsite.local/api/user/info", survive: true},
	}

	for _, tc := range cases {
		c := candidate{URL: tc.url, Confidence: "medium"}
		got := filterAPICandidates([]candidate{c})
		survived := len(got) > 0
		if survived != tc.survive {
			t.Errorf("[%s] want survive=%v, got survive=%v", tc.name, tc.survive, survived)
		}
	}
}
