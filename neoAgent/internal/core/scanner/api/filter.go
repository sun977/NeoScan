// Package api 的 filter.go：统一做转义清理、静态资源后缀排除、黑名单
// 关键词过滤、去重。只处理 extractAPICandidates 已经产出的 candidate，
// 不做二次提取，见 API扫描功能设计.md 7.3 节。
package api

import (
	"regexp"
	"strings"
)

// staticResourceSuffix 命中即丢弃：这些是静态资源，不是接口。
var staticResourceSuffix = regexp.MustCompile(`\.(js|css|png|jpe?g|gif|svg|woff2?)(\?.*)?$`)

// blacklistPattern 直接借鉴 URLFinder config.go 的 UrlFiler 规则（已经过
// 实战验证），排除低置信度正则容易误伤的 JS 属性访问模式（.src/.href 等）。
var blacklistPattern = regexp.MustCompile(
	`\.js\?|\.css\?|\.jpeg\?|\.jpg\?|\.png\?|\.gif\?|www\.w3\.org|example\.com|<|>|\{|\}|\[|\]|\||\^|;|/js/|\.src|\.replace|\.url|\.att|\.href|location\.href|javascript:|location:|application/x-www-form-urlencoded|\.createObject|:location|\.path`,
)

// filterAPICandidates 对 extractAPICandidates 的产出做统一清洗，返回值
// 已经是去重后的最终结果，调用方（fetch.go/bfs.go 的编排逻辑）拿到之后
// 直接转换成 []model.APIInfo，不需要再做任何二次处理。
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
