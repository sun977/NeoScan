// Package api 的 fetch.go：独立的单页抓取实现（go-rod 渲染 + 内联/外链 JS
// 获取）与 Target 输入形态判断，不 import web/crawler，见
// API扫描功能设计.md 第二节"原子扫描器隔离原则"。
package api

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"neoagent/internal/core/lib/browser"
	"neoagent/internal/pkg/logger"

	"github.com/go-rod/rod"
)

// normalizeTarget 把 Target/Ports 换算成一个可以直接发起抓取的起始 URL。
// 逻辑与 WebScanner 的 normalizeURL（scanner/web/web_scanner.go）行为等价
// 但实现简化（"URL 优先、Ports 取第一个端口而非遍历"），物理独立实现，
// 不 import 该函数——见 API扫描功能设计.md 5.1 节。
func normalizeTarget(target string, ports string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}

	port := firstPort(ports)
	host := target
	if port != "" && port != "80" && port != "443" && !strings.Contains(target, ":") {
		host = target + ":" + port
	}

	switch port {
	case "443", "8443", "9443", "10443":
		return "https://" + host
	default:
		return "http://" + host
	}
}

// firstPort 从 "80,443" 这种逗号分隔的端口列表里取第一个非空项。
// ApiScan 的起始页面只需要一个入口 URL，不像 WebScanner 需要对每个端口
// 都独立探测一次——ApiScan 关心的是"从这一个入口爬出多少接口"，不是
// "这台机器有多少个端口在提供 Web 服务"，用第一个端口即可。
func firstPort(ports string) string {
	if ports == "" {
		return ""
	}
	parts := strings.Split(ports, ",")
	return strings.TrimSpace(parts[0])
}

// jsSource 描述一段待提取的 JS/HTML 文本及其来源，供 extractAPICandidates
// 之后统一打标 model.APIInfo.Source。
type jsSource struct {
	Text string
	From string // "inline" 或外链 JS 的绝对 URL
}

// fetchedPage 是单页抓取的完整产出：渲染后的 HTML body + 所有待提取文本
// （HTML 本身 + 内联 JS + 外链 JS），以及本页发现的导航链接（供 BFS 使用）。
type fetchedPage struct {
	URL       string
	Sources   []jsSource
	Links     []string
	Truncated bool // 外链 JS 数量超过 maxJSFiles 被截断
}

// fetchPage 用 go-rod 渲染单页，拿到 HTML/内联JS/外链JS 三种文本来源。
// browserLauncher 由调用方（ApiScanner，在 NewApiScanner 时已初始化）
// 传入独立实例，不与 WebScanner 共享运行时状态。
func fetchPage(ctx context.Context, launcher *browser.BrowserLauncher, pageURL string, maxJSFiles int) (*fetchedPage, error) {
	br, err := launcher.Launch(ctx)
	if err != nil {
		return nil, err
	}

	page, err := launcher.OpenPage(ctx, br, "")
	if err != nil {
		return nil, err
	}
	defer page.Close()

	if err := page.Navigate(pageURL); err != nil {
		return nil, err
	}
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := page.Context(waitCtx).WaitLoad(); err != nil {
		logger.Warnf("[ApiScanner] WaitLoad timeout for %s: %v", pageURL, err)
	}

	body, err := page.HTML()
	if err != nil {
		return nil, err
	}

	result := &fetchedPage{URL: pageURL}
	result.Sources = append(result.Sources, jsSource{Text: body, From: pageURL})
	result.Sources = append(result.Sources, extractInlineScripts(page)...)

	links, jsFileURLs := extractLinksAndScriptSrcs(page, pageURL)
	result.Links = links

	if len(jsFileURLs) > maxJSFiles {
		result.Truncated = true
		jsFileURLs = jsFileURLs[:maxJSFiles]
	}
	for _, jsURL := range jsFileURLs {
		text, err := downloadJSFile(ctx, jsURL)
		if err != nil {
			logger.Warnf("[ApiScanner] Failed to download JS file %s: %v", jsURL, err)
			continue
		}
		result.Sources = append(result.Sources, jsSource{Text: text, From: jsURL})
	}

	return result, nil
}

// downloadJSFile 用纯 net/http 下载外链 JS 文件文本内容——静态 JS 文件本身
// 是文本资源，不需要浏览器渲染，见 API扫描功能设计.md 第八节。
func downloadJSFile(ctx context.Context, jsURL string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, jsURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// extractInlineScripts 读取页面里所有没有 src 属性的 <script> 标签的文本内容。
func extractInlineScripts(page *rod.Page) []jsSource {
	elements, err := page.Elements("script:not([src])")
	if err != nil {
		return nil
	}
	var out []jsSource
	for _, el := range elements {
		text, err := el.Text()
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, jsSource{Text: text, From: "inline"})
	}
	return out
}

// extractLinksAndScriptSrcs 读取页面里所有 <a href> 绝对化后的链接（供 BFS
// 使用）与所有 <script src> 绝对化后的 URL（供后续下载）。用 net/url 的
// ResolveReference 做相对路径换算，逻辑与 web/crawler/extract.go 的 resolve
// 一致，但独立实现，不 import 该函数。
func extractLinksAndScriptSrcs(page *rod.Page, pageURL string) (links []string, scriptSrcs []string) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, nil
	}

	if anchors, err := page.Elements("a[href]"); err == nil {
		for _, a := range anchors {
			href, err := a.Attribute("href")
			if err != nil || href == nil {
				continue
			}
			if abs := resolveHref(base, *href); abs != "" {
				links = append(links, abs)
			}
		}
	}

	if scripts, err := page.Elements("script[src]"); err == nil {
		for _, s := range scripts {
			src, err := s.Attribute("src")
			if err != nil || src == nil {
				continue
			}
			if abs := resolveHref(base, *src); abs != "" {
				scriptSrcs = append(scriptSrcs, abs)
			}
		}
	}
	return links, scriptSrcs
}

// resolveHref 把相对/绝对路径换算成绝对 URL，过滤掉 javascript:/mailto:/
// tel:/纯锚点这类不可爬取的伪链接，逻辑与 web/crawler/extract.go 的 resolve
// 一致但独立实现（原子隔离原则不允许 import）。
func resolveHref(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" ||
		strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "mailto:") ||
		strings.HasPrefix(href, "tel:") ||
		strings.HasPrefix(href, "#") {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}
