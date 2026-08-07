package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"neoagent/internal/core/model"
	"neoagent/internal/pkg/edge"
)

// 本文件是 CDN 边缘节点识别功能的步骤 7：端到端验收，对应
// docs/爬虫/Web扫描CDN识别实施文档.md 第八节的用例 A/B/C/D。
//
// 不像单元测试那样只测 edge.Detector 本身，这里跑的是完整的
// WebScanner.Run() 链路，验证 checkCDN 判断、截图/深度爬取跳过、
// EdgeComponents 结果标注这几个环节真正串联起来生效。

// testCDNRulesFixture 是端到端测试专用的 CDN 规则，刻意把 127.0.0.0/8
// （回环网段）注册成一个虚构厂商 "TestCDN"。这样可以让 task.Target
// 设为 "127.0.0.1"（本地 httptest.Server 的监听地址）时，checkCDN
// 判断為"命中 CDN"，同时网络连接依然打到本地 mock server，不依赖
// 任何真实公网 CDN 域名（用例 A 里实施文档允许的替代方案："或直接
// 构造一个 rules/edge/cdn.json 网段内的测试 IP"）。
const testCDNRulesFixture = `[
	{
		"provider": "TestCDN",
		"cidrs": ["127.0.0.0/8"]
	},
	{
		"provider": "Cloudflare",
		"cidrs": ["173.245.48.0/20"]
	}
]`

// newScannerWithTestCDNRules 创建一个 WebScanner，先跑 ensureInit 加载
// 生产指纹规则，再用测试专属的 CDN 规则覆盖 edgeDetector——不依赖也不
// 污染生产的 rules/edge/cdn.json。
func newScannerWithTestCDNRules(t *testing.T) *WebScanner {
	t.Helper()
	scanner := NewWebScanner()
	scanner.ensureInit()

	path := filepath.Join(t.TempDir(), "cdn.json")
	if err := os.WriteFile(path, []byte(testCDNRulesFixture), 0o644); err != nil {
		t.Fatalf("write test cdn rules: %v", err)
	}
	if err := scanner.edgeDetector.Load(path); err != nil {
		t.Fatalf("load test cdn rules: %v", err)
	}
	return scanner
}

// 用例 A：命中 CDN——目标 IP 落在测试规则的 127.0.0.0/8 网段内。
// 预期：EdgeComponents 标注命中 TestCDN；screenshot 为空；不触发深度
// 爬取（即使首页有链接、状态码 200、Content-Type 是 text/html，正常
// 情况下 decideCrawlDepth 会判定要爬，这里断言它没有爬，证明是 CDN
// 判断拦下来的，不是自动决策本来就没触发）。
func TestWebScanner_CDN_HitSkipsScreenshotAndCrawl(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>Edge Node</title></head><body>
			<a href="/a">a</a><a href="/b">b</a>
		</body></html>`)
	}))
	defer ts.Close()
	port := ts.Listener.Addr().(*net.TCPAddr).Port

	scanner := newScannerWithTestCDNRules(t)

	task := &model.Task{
		ID:        "test-cdn-hit",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d", port),
		Params: map[string]interface{}{
			"protocol":   "http",
			"screenshot": true, // 显式要求截图，验证 CDN 命中时依然被跳过
		},
	}

	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result (home page only, no crawled sub-pages), got %d", len(results))
	}

	wr, ok := results[0].Result.(*model.WebResult)
	if !ok {
		t.Fatalf("result is not *model.WebResult: %+v", results[0].Result)
	}

	if !wr.IsEdgeNode() {
		t.Fatal("expected IsEdgeNode() = true when target is inside a known CDN range")
	}
	if len(wr.EdgeComponents) != 1 || wr.EdgeComponents[0].Type != "cdn" || wr.EdgeComponents[0].Provider != "TestCDN" {
		t.Fatalf("expected EdgeComponents = [{cdn TestCDN}], got %+v", wr.EdgeComponents)
	}
	if wr.Screenshot != "" {
		t.Error("expected Screenshot to be skipped (empty) when target is a CDN node")
	}
}

// 用例 B：不命中 CDN——目标 IP 不在任何已知网段内。
// 预期：EdgeComponents 为空、IsEdgeNode() 为 false，行为与 CDN 功能
// 上线前完全一致（截图正常生效、深度爬取正常触发）。
func TestWebScanner_CDN_MissBehavesLikeBeforeCDNFeature(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>Origin Site</title></head><body>
			<a href="/a">a</a>
		</body></html>`)
	}))
	defer ts.Close()
	port := ts.Listener.Addr().(*net.TCPAddr).Port

	scanner := newScannerWithTestCDNRules(t)

	// checkCDN 只接受一个 target string 参数，这里不需要构造完整的 model.Task
	// （之前的写法构造了 ID/PortRange/Params 但从未被读取，是死代码）。
	const target = "10.201.28.126" // 不在测试规则的任何网段内的内网 IP，且不会真的发起连接（协议+目标不可达会快速失败，这里只关心 checkCDN 的判断，不关心整体扫描是否成功）

	isCDN, provider := scanner.checkCDN(target)
	if isCDN {
		t.Fatalf("expected checkCDN(%q) = false, got true (provider=%q)", target, provider)
	}

	// 额外验证：一个真实可达、协议匹配、不在任何 CDN 网段内的目标，扫描
	// 结果里 EdgeComponents 应为空、IsEdgeNode() 为 false——用 127.0.0.1
	// 但加载不含 127.0.0.0/8 的规则集，模拟"目标可达但不是 CDN"这一真实
	// 场景，比只测 checkCDN 本身更贴近端到端验收的意图。
	scanner2 := NewWebScanner()
	scanner2.ensureInit()
	emptyRulesPath := filepath.Join(t.TempDir(), "cdn.json")
	if err := os.WriteFile(emptyRulesPath, []byte(`[{"provider":"Cloudflare","cidrs":["173.245.48.0/20"]}]`), 0o644); err != nil {
		t.Fatalf("write empty-ish test cdn rules: %v", err)
	}
	if err := scanner2.edgeDetector.Load(emptyRulesPath); err != nil {
		t.Fatalf("load test cdn rules: %v", err)
	}

	task2 := &model.Task{
		ID:        "test-cdn-miss-e2e",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d", port),
		Params: map[string]interface{}{
			"protocol":   "http",
			"screenshot": true,
		},
	}
	results, err := scanner2.Run(context.Background(), task2)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	wr, ok := results[0].Result.(*model.WebResult)
	if !ok {
		t.Fatalf("result is not *model.WebResult: %+v", results[0].Result)
	}
	if wr.IsEdgeNode() {
		t.Errorf("expected IsEdgeNode() = false for a non-CDN target, got EdgeComponents = %+v", wr.EdgeComponents)
	}
}

// 用例 C：规则文件缺失——edgeDetector 从未成功 Load 过任何规则（模拟
// ensureInit 里全部候选路径都加载失败的情况）。
// 预期：扫描任务正常完成，不因为规则缺失而报错或崩溃，行为退化为
// "不是 CDN"，等同于用例 B。
func TestWebScanner_CDN_MissingRulesFileDegradesGracefully(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>No Rules Loaded</title></head><body>ok</body></html>`)
	}))
	defer ts.Close()
	port := ts.Listener.Addr().(*net.TCPAddr).Port

	// 刻意不调用 edgeDetector.Load，模拟 ensureInit 里全部候选路径都
	// 加载失败的情况——edgeDetector 内部规则集为空（零值 Detector）。
	scanner := NewWebScanner()
	scanner.ensureInit() // 只加载指纹规则；CDN 规则试探列表大概率会命中生产文件，
	// 所以这里显式创建一个全新的、从未 Load 过 CDN 规则的 Detector 替换掉它，
	// 精确模拟"规则文件缺失"而不是"意外加载到了真实生产规则"。
	scanner.edgeDetector = edge.NewDetector()

	task := &model.Task{
		ID:        "test-cdn-missing-rules",
		Target:    "127.0.0.1",
		PortRange: fmt.Sprintf("%d", port),
		Params: map[string]interface{}{
			"protocol": "http",
		},
	}

	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("expected scan to complete successfully even with no CDN rules loaded, got error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	wr, ok := results[0].Result.(*model.WebResult)
	if !ok {
		t.Fatalf("result is not *model.WebResult: %+v", results[0].Result)
	}
	if wr.IsEdgeNode() {
		t.Errorf("expected IsEdgeNode() = false when no CDN rules are loaded, got EdgeComponents = %+v", wr.EdgeComponents)
	}
	if wr.Title != "No Rules Loaded" {
		t.Errorf("expected scan to still succeed and extract title, got %q", wr.Title)
	}
}

// 用例 D：域名解析失败——target 是一个不存在的域名。
// 预期：checkCDN 内部 net.LookupHost 失败，直接返回 (false, "")，不
// panic、不阻塞，CDN 判断这一步本身不应该是扫描失败的原因。
func TestWebScanner_CDN_DNSResolutionFailureDoesNotBlock(t *testing.T) {
	scanner := newScannerWithTestCDNRules(t)

	// RFC 2606 保留域名，保证不会被真实解析成功。
	isCDN, provider := scanner.checkCDN("this-domain-does-not-exist.invalid")
	if isCDN {
		t.Fatalf("expected checkCDN to return false for unresolvable domain, got true (provider=%q)", provider)
	}
	if provider != "" {
		t.Errorf("expected empty provider, got %q", provider)
	}
}