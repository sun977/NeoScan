package web

import (
	"testing"

	"neoagent/internal/core/model"
)

// TestDecideCrawlDepth 覆盖实施文档 7.2 节列出的六个用例：状态码、Content-Type、
// 种子链接数量三个免费信号的组合判断。
func TestDecideCrawlDepth(t *testing.T) {
	cases := []struct {
		name           string
		statusCode     int
		contentType    string
		seedLinksCount int
		wantDepth      int
	}{
		{"正常站点", 200, "text/html", 10, 2},
		{"404", 404, "text/html", 0, 0},
		{"401但有链接", 401, "text/html", 3, 2},
		{"纯JSON API", 200, "application/json", 0, 0},
		{"200但无链接", 200, "text/html", 0, 0},
		{"500", 500, "text/html", 5, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decideCrawlDepth(c.statusCode, c.contentType, c.seedLinksCount)
			if got != c.wantDepth {
				t.Errorf("decideCrawlDepth(%d, %q, %d) = %d, want %d",
					c.statusCode, c.contentType, c.seedLinksCount, got, c.wantDepth)
			}
		})
	}
}

// TestResolveCrawlDepth_ThreeStates 验证 task.Params["crawl"] 三态分流：
// 显式开启优先级最高，显式关闭次之，未指定才落入 decideCrawlDepth 自动判断。
func TestResolveCrawlDepth_ThreeStates(t *testing.T) {
	s := NewWebScanner()
	headers := map[string]string{"Content-Type": "application/json"} // 自动判断下会返回 0

	t.Run("显式开启覆盖自动判断", func(t *testing.T) {
		task := &model.Task{Params: map[string]interface{}{"crawl": true}}
		depth := s.resolveCrawlDepth(task, 500, headers, nil) // 500 + JSON + 无链接，自动判断必然是 0
		if depth == 0 {
			t.Fatal("expected non-zero depth when crawl=true explicitly set")
		}
	})

	t.Run("显式关闭覆盖自动判断", func(t *testing.T) {
		task := &model.Task{Params: map[string]interface{}{"crawl": false}}
		htmlHeaders := map[string]string{"Content-Type": "text/html"}
		depth := s.resolveCrawlDepth(task, 200, htmlHeaders, []string{"a", "b"}) // 信号全部满足，自动判断会是 2
		if depth != 0 {
			t.Fatalf("expected depth=0 when crawl=false explicitly set, got %d", depth)
		}
	})

	t.Run("未指定走自动判断", func(t *testing.T) {
		task := &model.Task{Params: map[string]interface{}{}}
		htmlHeaders := map[string]string{"Content-Type": "text/html"}
		depth := s.resolveCrawlDepth(task, 200, htmlHeaders, []string{"a"})
		if depth != 2 {
			t.Fatalf("expected depth=2 from auto decision, got %d", depth)
		}
	})

	t.Run("显式开启并指定crawl_depth", func(t *testing.T) {
		task := &model.Task{Params: map[string]interface{}{"crawl": true, "crawl_depth": 3}}
		depth := s.resolveCrawlDepth(task, 200, nil, nil)
		if depth != 3 {
			t.Fatalf("expected depth=3, got %d", depth)
		}
	})
}
