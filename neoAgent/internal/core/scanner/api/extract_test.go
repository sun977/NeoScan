package api

import (
	"regexp"
	"testing"
)

func TestExtractAPICandidates(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		pageHost string
		wantURLs []string
		wantConf string // 断言至少一条命中带有这个 Confidence
	}{
		{
			name:     "fetch高置信度",
			text:     `fetch('/api/user/info')`,
			wantURLs: []string{"/api/user/info"},
			wantConf: "high",
		},
		{
			name:     "axios高置信度带method",
			text:     `axios.post('/api/order/create')`,
			wantURLs: []string{"/api/order/create"},
			wantConf: "high",
		},
		{
			name:     "ajax高置信度",
			text:     `$.ajax({url: '/api/legacy/list', type: 'GET'})`,
			wantURLs: []string{"/api/legacy/list"},
			wantConf: "high",
		},
		{
			name:     "中置信度scope锚定",
			text:     `var base = "https://example.com/api/v2/user";`,
			pageHost: "example.com",
			wantURLs: []string{"https://example.com/api/v2/user"},
			wantConf: "medium",
		},
		{
			name:     "中置信度排除第三方域名",
			text:     `var cdn = "https://cdn.other.com/lib.js";`,
			pageHost: "example.com",
			wantConf: "", // 不应该命中中置信度（域名不匹配）
		},
		{
			name:     "低置信度非固定前缀路径",
			text:     `"url": "/user/getUserInfo"`,
			wantURLs: []string{"/user/getUserInfo"},
			wantConf: "low",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mediumPattern *regexp.Regexp
			if tc.pageHost != "" {
				mediumPattern = regexp.MustCompile(
					`https?://` + regexp.QuoteMeta(tc.pageHost) + `[a-zA-Z0-9_\-/?&=.%]*`,
				)
			}
			got := extractAPICandidates(tc.text, mediumPattern)

			gotURLs := make(map[string]bool)
			gotConfSet := make(map[string]bool)
			for _, c := range got {
				gotURLs[c.URL] = true
				gotConfSet[c.Confidence] = true
			}

			for _, want := range tc.wantURLs {
				if !gotURLs[want] {
					t.Errorf("expected URL %q in candidates, got %+v", want, got)
				}
			}
			if tc.wantConf != "" && !gotConfSet[tc.wantConf] {
				t.Errorf("expected at least one candidate with Confidence=%q, got %+v", tc.wantConf, got)
			}
			if tc.name == "中置信度排除第三方域名" && gotConfSet["medium"] {
				t.Errorf("did not expect medium confidence hit for third-party domain, got %+v", got)
			}
		})
	}
}

func TestExtractHighConfidence_MethodCapture(t *testing.T) {
	got := extractHighConfidence(`axios.delete('/api/user/1')`)
	if len(got) != 1 || got[0].Method != "DELETE" {
		t.Fatalf("expected Method=DELETE, got %+v", got)
	}
}

func TestExtractHighConfidence_MultiMatch(t *testing.T) {
	got := extractHighConfidence(`fetch('/api/a'); axios.post('/api/b'); $.ajax({url: '/api/c'})`)
	if len(got) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(got))
	}
	// Verify all have high confidence and no method (except axios which should have POST)
	for i, tc := range []struct {
		url    string
		method string
	}{
		{"/api/a", ""},
		{"/api/b", "POST"},
		{"/api/c", ""},
	} {
		if got[i].URL != tc.url {
			t.Errorf("[%d] expected URL %q, got %q", i, tc.url, got[i].URL)
		}
		if got[i].Method != tc.method {
			t.Errorf("[%d] expected Method %q, got %q", i, tc.method, got[i].Method)
		}
	}
}

func TestExtractHighConfidence_Empty(t *testing.T) {
	got := extractHighConfidence(`no api calls here`)
	if len(got) != 0 {
		t.Fatalf("expected 0 matches, got %d: %+v", len(got), got)
	}
}

func TestExtractMediumConfidence(t *testing.T) {
	t.Run("pattern nil returns nil", func(t *testing.T) {
		got := extractMediumConfidence(`https://example.com/api/test`, nil)
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("matches same-domain URL", func(t *testing.T) {
		pattern := regexp.MustCompile(`https?://` + regexp.QuoteMeta("example.com") + `[a-zA-Z0-9_\-/?&=.%]*`)
		got := extractMediumConfidence(`var base = "https://example.com/api/v2/data";`, pattern)
		if len(got) != 1 || got[0].URL != "https://example.com/api/v2/data" || got[0].Confidence != "medium" {
			t.Fatalf("unexpected result: %+v", got)
		}
	})

	t.Run("no match for different domain", func(t *testing.T) {
		pattern := regexp.MustCompile(`https?://` + regexp.QuoteMeta("example.com") + `[a-zA-Z0-9_\-/?&=.%]*`)
		got := extractMediumConfidence(`var cdn = "https://cdn.other.com/lib.js";`, pattern)
		if len(got) != 0 {
			t.Fatalf("expected 0 matches, got %d: %+v", len(got), got)
		}
	})
}

func TestExtractLowConfidence(t *testing.T) {
	t.Run("matches valid path", func(t *testing.T) {
		got := extractLowConfidence(`"path": "/api/users/123"`)
		if len(got) != 1 || got[0].URL != "/api/users/123" || got[0].Confidence != "low" {
			t.Fatalf("unexpected result: %+v", got)
		}
	})

	t.Run("no match for invalid path format", func(t *testing.T) {
		// Must have at least two path segments starting with /
		got := extractLowConfidence(`"x": "/single"`)
		if len(got) != 0 {
			t.Fatalf("expected 0 matches, got %d: %+v", len(got), got)
		}
	})
}

func TestExtractAPICandidates_EmptyInput(t *testing.T) {
	got := extractAPICandidates("", nil)
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %+v", got)
	}
}

func TestExtractAPICandidates_MediumPatternNil(t *testing.T) {
	got := extractAPICandidates(`fetch('/api/test')`, nil)
	// mediumPattern nil 时跳过中置信度，但 high + low 仍会命中
	if len(got) != 2 {
		t.Fatalf("expected 2 matches (high + low), got %d", len(got))
	}
	// 确保没有 medium 置信度命中
	for _, c := range got {
		if c.Confidence == "medium" {
			t.Errorf("unexpected medium confidence candidate: %+v", c)
		}
	}
}
