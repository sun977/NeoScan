package api

import (
	"net/url"
	"testing"
)

func TestNormalizeTarget(t *testing.T) {
	cases := []struct {
		name   string
		target string
		ports  string
		want   string
	}{
		{"完整HTTPS_URL忽略Ports", "https://example.com", "80,443", "https://example.com"},
		{"完整HTTP_URL带端口忽略Ports", "http://example.com:8080", "80,443", "http://example.com:8080"},
		{"裸域名标准端口443猜https", "example.com", "443", "https://example.com"},
		{"裸域名标准端口80猜http", "example.com", "80", "http://example.com"},
		{"裸域名非标准端口显式拼接", "example.com", "8080", "http://example.com:8080"},
		{"裸IP多端口取第一个", "192.168.1.1", "443,80", "https://192.168.1.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTarget(tc.target, tc.ports)
			if got != tc.want {
				t.Errorf("normalizeTarget(%q, %q) = %q, want %q", tc.target, tc.ports, got, tc.want)
			}
		})
	}
}

func TestResolveHref(t *testing.T) {
	base, _ := url.Parse("http://example.com/dir/page.html")
	cases := []struct {
		href string
		want string
	}{
		{"foo.html", "http://example.com/dir/foo.html"},
		{"/api/user", "http://example.com/api/user"},
		{"javascript:void(0)", ""},
		{"#section", ""},
		{"mailto:a@b.com", ""},
		{"tel:+1234", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := resolveHref(base, tc.href)
		if got != tc.want {
			t.Errorf("resolveHref(base, %q) = %q, want %q", tc.href, got, tc.want)
		}
	}
}
