package web

import (
	"context"
	"fmt"
	"testing"

	"neoagent/internal/core/lib/crawler"
	"neoagent/internal/core/lib/network/qos"
)

// TestEscalateIfNeeded_ExceedsMaxPages 覆盖实施文档 7.4 节用例 5：待升级页面数
// 超过 defaultMaxEscalationPages 上限时，escalateIfNeeded 应直接跳过，不发起
// 任何浏览器 Launch/渲染。验证方式：构造 11 个 NeedsEscalation=true 的 Page，
// 调用真实的 escalateIfNeeded 后断言每个 Page.Body 都保持原样未被改写——
// 如果真的发起了渲染，成功的话 Body 会被替换成渲染后的内容，超限跳过则不会。
func TestEscalateIfNeeded_ExceedsMaxPages(t *testing.T) {
	scanner := NewWebScanner()

	const total = defaultMaxEscalationPages + 1
	pages := make([]*crawler.Page, 0, total)
	for i := 0; i < total; i++ {
		pages = append(pages, &crawler.Page{
			URL:             fmt.Sprintf("http://example.invalid/page%d", i),
			Body:            "original",
			NeedsEscalation: true,
		})
	}

	cr := crawler.New(crawler.Options{MaxDepth: 1}, qos.NewAdaptiveLimiter(5, 1, 10))
	scanner.escalateIfNeeded(context.Background(), cr, pages)

	for _, p := range pages {
		if p.Body != "original" {
			t.Fatalf("expected page %s body to remain untouched when exceeding max escalation pages, got %q", p.URL, p.Body)
		}
	}
}
