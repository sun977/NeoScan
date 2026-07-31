package crawler

import (
	"fmt"
	"strings"
	"testing"
)

// TestDetectEscalation_JSRedirect 覆盖实施文档 7.4 节用例 1：小于 1KB 且包含
// location.href 赋值的页面应判定为需要升级，reason 为 "js_redirect"。
func TestDetectEscalation_JSRedirect(t *testing.T) {
	body := `<html><script>location.href='/real'</script></html>`
	needs, reason := detectEscalation(body)
	if !needs {
		t.Fatal("expected needs=true for JS redirect page")
	}
	if reason != "js_redirect" {
		t.Fatalf("expected reason=js_redirect, got %q", reason)
	}
}

// TestDetectEscalation_SPAShell 覆盖用例 2：有 <div id="root"></div> 挂载点、
// 且除脚本外几乎没有可见文本的页面应判定为 SPA 空壳。
func TestDetectEscalation_SPAShell(t *testing.T) {
	// 用一段超过 1KB 的脚本占位，确保不会被 isJSRedirect 的长度阈值误判，
	// 同时验证 SPA 空壳检测本身不依赖页面长度。
	padding := strings.Repeat("/* padding */\n", 100)
	body := fmt.Sprintf(`<html><body><div id="root"></div><script>%s</script></body></html>`, padding)

	needs, reason := detectEscalation(body)
	if !needs {
		t.Fatal("expected needs=true for SPA shell page")
	}
	if reason != "spa_shell" {
		t.Fatalf("expected reason=spa_shell, got %q", reason)
	}
}

// TestDetectEscalation_NormalPage 覆盖用例 3：正常业务页面有充足的可见文本，
// 不应被判定为需要升级。
func TestDetectEscalation_NormalPage(t *testing.T) {
	body := `<html><body><h1>Welcome</h1><p>` +
		strings.Repeat("This is a normal business page with real content. ", 10) +
		`</p></body></html>`

	needs, reason := detectEscalation(body)
	if needs {
		t.Fatalf("expected needs=false for normal page, got reason=%q", reason)
	}
}

// TestDetectEscalation_NoFalsePositiveOnNormalReactApp 覆盖用例 4（SPA 空壳检测
// 最容易踩的坑）：<div id="root"> 挂载点存在，但服务端已经把内容渲染进去了
// （模拟 SSR），此时不应该被误判为空壳。
func TestDetectEscalation_NoFalsePositiveOnNormalReactApp(t *testing.T) {
	ssrContent := strings.Repeat("Server-side rendered product listing item. ", 10)
	body := fmt.Sprintf(`<html><body><div id="root"><div class="app">%s</div></div></body></html>`, ssrContent)

	needs, reason := detectEscalation(body)
	if needs {
		t.Fatalf("expected needs=false for SSR React app, got reason=%q", reason)
	}
}
