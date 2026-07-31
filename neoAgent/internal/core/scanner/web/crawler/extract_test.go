package crawler

import (
	"sort"
	"testing"
)

// TestExtractLinksAndForms_BasicLinks 验证 resolve 的过滤 + 换算逻辑：
// 5 个 <a href> 里，只有"相对路径"和"绝对路径"这两个是真正可爬取的页面地址，
// 其余三个（#fragment、javascript:、mailto:）都应该被过滤掉。
func TestExtractLinksAndForms_BasicLinks(t *testing.T) {
	body := `<html><body>
		<a href="relative.html">relative</a>
		<a href="http://other.example.com/abs">absolute</a>
		<a href="#section2">fragment</a>
		<a href="javascript:void(0)">js</a>
		<a href="mailto:a@b.com">mail</a>
	</body></html>`

	baseURL := "http://a.example.com/dir/page.html"
	links, _, _ := ExtractLinksAndForms(baseURL, body)

	if len(links) != 2 {
		t.Fatalf("expected 2 valid links, got %d: %v", len(links), links)
	}

	sort.Strings(links)
	want := []string{"http://a.example.com/dir/relative.html", "http://other.example.com/abs"}
	sort.Strings(want)
	for i := range want {
		if links[i] != want[i] {
			t.Fatalf("link[%d] = %s, want %s", i, links[i], want[i])
		}
	}
}

// TestExtractLinksAndForms_Forms 验证表单提取：
//  1. 有 method 属性的表单，方法名应被规范化为大写。
//  2. 没有 method 属性的表单，按 HTML 规范默认识别为 GET。
//  3. input/select/textarea 三种带 name 的控件都要被收进 Fields。
func TestExtractLinksAndForms_Forms(t *testing.T) {
	body := `<html><body>
		<form action="/login" method="post">
			<input name="username" type="text">
			<input name="password" type="password">
			<select name="role"><option value="1">a</option></select>
			<textarea name="remark"></textarea>
			<input type="submit" value="Login">
		</form>
		<form action="/search">
			<input name="q" type="text">
		</form>
	</body></html>`

	_, forms, _ := ExtractLinksAndForms("http://a.example.com/", body)

	if len(forms) != 2 {
		t.Fatalf("expected 2 forms, got %d", len(forms))
	}

	loginForm := forms[0]
	if loginForm.Action != "/login" {
		t.Fatalf("expected action=/login, got %s", loginForm.Action)
	}
	if loginForm.Method != "POST" {
		t.Fatalf("expected method=POST (normalized to uppercase), got %s", loginForm.Method)
	}
	wantFields := []string{"username", "password", "role", "remark"}
	if len(loginForm.Fields) != len(wantFields) {
		t.Fatalf("expected %d fields, got %d: %v", len(wantFields), len(loginForm.Fields), loginForm.Fields)
	}

	searchForm := forms[1]
	if searchForm.Method != "GET" {
		t.Fatalf("expected default method=GET when method attr absent, got %s", searchForm.Method)
	}
}

// TestExtractLinksAndForms_Params 验证：baseURL 自身携带的 Query 参数名
// 能被正确抽取出来（不关心参数值，也不关心返回顺序）。
func TestExtractLinksAndForms_Params(t *testing.T) {
	body := `<html><body>no links here</body></html>`
	baseURL := "http://a.example.com/search?id=1&type=admin"

	_, _, params := ExtractLinksAndForms(baseURL, body)

	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d: %v", len(params), params)
	}
	sort.Strings(params)
	want := []string{"id", "type"}
	for i := range want {
		if params[i] != want[i] {
			t.Fatalf("param[%d] = %s, want %s", i, params[i], want[i])
		}
	}
}

// TestResolve_EdgeCases 单独针对 resolve 函数的边界情况做补充验证，
// 覆盖 extract_test.go 主测试没有直接触达的空字符串和纯空白字符串场景。
func TestResolve_EdgeCases(t *testing.T) {
	links, _, _ := ExtractLinksAndForms("http://a.example.com/", `
		<a href="">empty</a>
		<a href="   ">blank</a>
		<a href="tel:10086">tel</a>
	`)
	if len(links) != 0 {
		t.Fatalf("expected 0 links from empty/blank/tel hrefs, got %d: %v", len(links), links)
	}
}
