// Package api 的 filter.go：统一做转义清理、静态资源后缀排除、黑名单
// 关键词过滤、来源过滤（Source filtering）、去重。只处理
// extractAPICandidates 已经产出的 candidate，不做二次提取，
// 见 API扫描功能设计.md 7.3 节。
package api

import (
	"regexp"
	"strings"
)

// staticResourceSuffix 命中即丢弃：这些是静态资源或调试产物，不是接口。
// 相比原版扩充了 .map/.ico/.ttf/.eot/.woff(.2?)/.json 等 CDN 噪声后缀。
var staticResourceSuffix = regexp.MustCompile(
	`\.(js|css|png|jpe?g|gif|svg|woff2?|eot|ttf|ico|map|json)(\?.*)?$`,
)

// pageResourceSuffix 命中即表明这是一个"页面 URL"而非接口——medium 置信度
// 容易把完整的 .html/.htm/.php 等页面链接当成 API，需要额外过滤。
var pageResourceSuffix = regexp.MustCompile(
	`\.(html?|php|asp(x)?|jsp|cfm|do)(\?.*)?$`,
)

// blacklistPattern 直接借鉴 URLFinder config.go 的 UrlFiler 规则（已经过
// 实战验证），排除低置信度正则容易误伤的 JS 属性访问模式（.src/.href 等）。
var blacklistPattern = regexp.MustCompile(
	`\.js\?|\.css\?|\.jpeg\?|\.jpg\?|\.png\?|\.gif\?|www\.w3\.org|example\.com|<|>|\{|\}|\[|\]|\||\^|;|/js/|\.src|\.replace|\.url|\.att|\.href|location\.href|javascript:|location:|application/x-www-form-urlencoded|\.createObject|:location|\.path`,
)

// isJSFileSource 判断一条 candidate 的 Source 是否来自真正的外链 JS 文件，
// 而非页面 HTML 本身。内联脚本的 Source 固定为 "inline"，页面 HTML 的
// Source 是页面 URL（以 http:// 或 https:// 开头且不以 .js 结尾）。
//
// 过滤依据：low 置信度的路径字面量在页面 HTML 中大量存在（路由配置、CSS
// 类名、i18n key、图片路径注释等），这类噪声与"真实 API 路径"几乎无法用
// 纯正则区分；而在真实 JS bundle 中出现的路径字面量命中真实接口的概率
// 远高于 HTML。"只保留 JS 文件来源的 low 候选项"是最低成本的降噪措施。
func isJSFileSource(source string) bool {
	if source == "inline" {
		return true // 内联 <script> 块也属于 JS 代码，保留
	}
	// 外链 JS 文件：以 .js 或 .js?v=xxx 等结尾
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		idx := strings.Index(source, "?")
		base := source
		if idx > 0 {
			base = source[:idx]
		}
		return strings.HasSuffix(base, ".js")
	}
	return false
}

// hasEnoughSegments 判断路径是否至少有 minSegments 个非空路径段。
// 目的是过滤掉 /en/、/zh-CN/、/v1/ 这类过短路径——它们更像语言/版本前缀，
// 几乎不可能是 REST API 端点，但会被 low 置信度正则大量命中。
// 示例：/api/users（2 段）→ 保留；/en/（1 段）→ 丢弃。
func hasEnoughSegments(rawURL string, minSegments int) bool {
	// rawURL 可能是相对路径 /foo/bar 或绝对 URL https://host/foo/bar
	path := rawURL
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		// 取 path 部分
		for _, prefix := range []string{"http://", "https://"} {
			if strings.HasPrefix(rawURL, prefix) {
				rest := rawURL[len(prefix):]
				slash := strings.Index(rest, "/")
				if slash < 0 {
					path = "/"
				} else {
					path = rest[slash:]
				}
				break
			}
		}
	}
	count := 0
	for _, seg := range strings.Split(path, "/") {
		if strings.TrimSpace(seg) != "" {
			count++
		}
	}
	return count >= minSegments
}

// filterAPICandidates 对 extractAPICandidates 的产出做统一清洗，返回值
// 已经是去重后的最终结果，调用方（bfs.go 的编排逻辑）拿到之后直接转换成
// []model.APIInfo，不需要再做任何二次处理。
//
// 过滤顺序（按开销从低到高，短路尽早丢弃）：
//  1. cleanURL（转义清理）
//  2. 空 URL
//  3. staticResourceSuffix（静态资源后缀）
//  4. blacklistPattern（黑名单关键词）
//  5. pageResourceSuffix（页面后缀，仅 medium 置信度）
//  6. Source 来源过滤（仅 low 置信度）
//  7. 路径段数过滤（仅 low 置信度）
//  8. 去重（URL + Method 组合键）
func filterAPICandidates(candidates []candidate) []candidate {
	seen := make(map[string]bool, len(candidates))
	var out []candidate

	for _, c := range candidates {
		c.URL = cleanURL(c.URL)
		if c.URL == "" {
			continue
		}
		if staticResourceSuffix.MatchString(c.URL) {
			continue
		}
		if blacklistPattern.MatchString(c.URL) {
			continue
		}

		// medium 置信度额外过滤：完整页面 URL（.html/.php 等）不是接口
		if c.Confidence == "medium" && pageResourceSuffix.MatchString(c.URL) {
			continue
		}

		// low 置信度额外过滤
		if c.Confidence == "low" {
			// 只保留来自 JS 代码（外链 .js 文件或内联 <script>）的候选项，
			// 来自页面 HTML 本体的 low 命中噪声太高，直接丢弃。
			if !isJSFileSource(c.Source) {
				continue
			}
			// 路径段数过少（如 /en/、/v1/）——几乎不是真实 API 端点，丢弃。
			if !hasEnoughSegments(c.URL, 2) {
				continue
			}
		}

		key := c.URL + "|" + c.Method
		if seen[key] {
			continue
		}
		seen[key] = true

		out = append(out, c)
	}
	return out
}

// cleanURL 做基础的转义清理，对齐 URLFinder jsFilter/urlFilter 的清洗步骤
// （去空白、还原转义斜杠、URL 解码常见的 %3A/%2F）。
func cleanURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, `\/`, "/")
	s = strings.ReplaceAll(s, "%3A", ":")
	s = strings.ReplaceAll(s, "%2F", "/")
	return s
}
