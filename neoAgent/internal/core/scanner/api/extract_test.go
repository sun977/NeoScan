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
