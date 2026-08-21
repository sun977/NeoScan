package engine

import (
	"regexp"
	"testing"
)

func makeResp(status int, body string, size int64) *Response {
	return &Response{
		StatusCode: status,
		Body:       body,
		Size:       size,
	}
}

func TestFilter_ExcludeStatus(t *testing.T) {
	f := NewFilter(FilterConfig{
		ExcludeStatus: []int{404, 500},
	})

	if f.Match(makeResp(404, "", 0), "/any") {
		t.Error("404 should be excluded")
	}
	if f.Match(makeResp(500, "", 0), "/any") {
		t.Error("500 should be excluded")
	}
	if !f.Match(makeResp(200, "", 0), "/any") {
		t.Error("200 should pass")
	}
}

func TestFilter_IncludeStatus(t *testing.T) {
	f := NewFilter(FilterConfig{
		IncludeStatus: []int{200, 301},
		ExcludeStatus: []int{}, // 清空黑名单避免干扰
	})

	if !f.Match(makeResp(200, "", 0), "/any") {
		t.Error("200 should pass with whitelist")
	}
	if !f.Match(makeResp(301, "", 0), "/any") {
		t.Error("301 should pass with whitelist")
	}
	if f.Match(makeResp(403, "", 0), "/any") {
		t.Error("403 not in whitelist, should be excluded")
	}
}

func TestFilter_DefaultExcludeStatus(t *testing.T) {
	// 不传 ExcludeStatus 时应自动使用默认值
	f := NewFilter(FilterConfig{})
	defaults := DefaultExcludeStatus()
	for _, code := range defaults {
		if f.Match(makeResp(code, "", 0), "/any") {
			t.Errorf("status %d should be excluded by default", code)
		}
	}
}

func TestFilter_ExcludeKeyword(t *testing.T) {
	f := NewFilter(FilterConfig{
		ExcludeStatus:   []int{},
		ExcludeKeywords: []string{"Access Denied", "403 Forbidden"},
	})

	resp := makeResp(200, "Access Denied - You don't have permission", 100)
	if f.Match(resp, "/any") {
		t.Error("response containing keyword should be excluded")
	}

	respOK := makeResp(200, "Welcome to the site", 100)
	if !f.Match(respOK, "/any") {
		t.Error("response without keyword should pass")
	}
}

func TestFilter_ExcludeRegex(t *testing.T) {
	re := regexp.MustCompile(`<title>404 Not Found</title>`)
	f := NewFilter(FilterConfig{
		ExcludeStatus: []int{},
		ExcludeRegex:  []*regexp.Regexp{re},
	})

	resp := makeResp(200, "<html><title>404 Not Found</title></html>", 100)
	if f.Match(resp, "/any") {
		t.Error("response matching regex should be excluded")
	}

	respOK := makeResp(200, "<html><title>Admin Panel</title></html>", 100)
	if !f.Match(respOK, "/any") {
		t.Error("response not matching regex should pass")
	}
}

func TestFilter_ExcludeSize(t *testing.T) {
	f := NewFilter(FilterConfig{
		ExcludeStatus: []int{},
		ExcludeSize:   []int64{0, 1234},
	})

	if f.Match(makeResp(200, "", 0), "/any") {
		t.Error("size 0 should be excluded")
	}
	if f.Match(makeResp(200, "", 1234), "/any") {
		t.Error("size 1234 should be excluded")
	}
	if !f.Match(makeResp(200, "", 999), "/any") {
		t.Error("size 999 should pass")
	}
}

func TestFilter_AllPass(t *testing.T) {
	f := NewFilter(FilterConfig{
		ExcludeStatus: DefaultExcludeStatus(),
	})

	resp := &Response{
		StatusCode:  200,
		Body:        "<html><body>Admin Panel</body></html>",
		Size:        1000,
		ContentType: "text/html",
	}
	if !f.Match(resp, "/admin") {
		t.Error("normal 200 response should pass all filters")
	}
}
