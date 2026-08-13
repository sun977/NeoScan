// Package api 的 extract.go：只负责"跑正则、收集命中"，不做任何排除判断。
// 过滤逻辑在 filter.go，两者物理分离，见 API扫描功能设计.md 7.3 节。
package api

import (
	"regexp"
	"strings"
)

// candidate 是提取阶段的中间产物，尚未经过 filterAPICandidates 清洗。
// 字段含义与 model.APIInfo 一致，之所以不直接用 model.APIInfo，是保留
// "提取阶段" 和 "最终产出" 之间一层缓冲——如果未来 APIInfo 需要新增仅供
// 展示、不参与过滤判断的字段，两者可以自由分叉，不需要牵动 extract.go。
type candidate struct {
	URL        string
	Method     string
	Source     string // 来源：inline 或外链 JS 的绝对 URL，由编排层逐条赋值
	Confidence string
}

// 高置信度：匹配明确的 HTTP 调用语句结构，自带 Method 信息。
var highConfidencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`fetch\(['"]([^'"]+)['"]`),
	regexp.MustCompile(`axios\.(get|post|put|delete|patch)\(['"]([^'"]+)['"]`),
	regexp.MustCompile(`\.ajax\(\{[^}]*url:\s*['"]([^'"]+)['"]`),
}

// 低置信度兜底：通用路径字面量，不限定固定前缀（方案文档 7.2 节 v1.1 调整）。
var lowConfidencePattern = regexp.MustCompile(`['"](/[a-zA-Z0-9_\-]+/[a-zA-Z0-9_\-/{}]+)['"]`)

// extractAPICandidates 对一段文本（HTML/内联JS/外链JS）跑三层正则，返回
// 全部命中结果，不做任何排除/清洗——那是 filterAPICandidates 的职责。
//
// mediumPattern 是调用方（apiCrawler）按 seedHost 预编译好的中置信度正则，
// 在整个爬取任务中只编译一次，避免每次调用都重新编译。传 nil 时跳过中置信度提取。
func extractAPICandidates(text string, mediumPattern *regexp.Regexp) []candidate {
	var out []candidate

	out = append(out, extractHighConfidence(text)...)
	out = append(out, extractMediumConfidence(text, mediumPattern)...)
	out = append(out, extractLowConfidence(text)...)

	return out
}

// extractHighConfidence 命中结构化调用语句。fetch/ajax 的正则只有 1 个
// 捕获组（URL 本身），axios 的正则有 2 个捕获组（Method + URL），用
// "最后一个捕获组永远是 URL"的规则统一取值，Method 只有 axios 分支才有
// 意义地填充。
func extractHighConfidence(text string) []candidate {
	var out []candidate
	for _, re := range highConfidencePatterns {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			switch len(m) {
			case 2:
				// fetch / .ajax：m[1] 是 URL，没有 Method 信息。
				out = append(out, candidate{URL: m[1], Confidence: "high"})
			case 3:
				// axios.<method>(...)：m[1] 是 method，m[2] 是 URL。
				out = append(out, candidate{URL: m[2], Method: strings.ToUpper(m[1]), Confidence: "high"})
			}
		}
	}
	return out
}

// extractMediumConfidence 使用调用方预编译好的正则匹配明确指向当前站点域名
// 的完整 URL，见 API扫描功能设计.md 7.2 节中置信度规则说明。
// pattern 由 apiCrawler 在 crawl() 开始时按 seedHost 编译一次，整个爬取
// 任务复用，不在此处编译。pattern 为 nil 时直接返回 nil，跳过中置信度提取。
func extractMediumConfidence(text string, pattern *regexp.Regexp) []candidate {
	if pattern == nil {
		return nil
	}
	var out []candidate
	for _, m := range pattern.FindAllString(text, -1) {
		out = append(out, candidate{URL: m, Confidence: "medium"})
	}
	return out
}

// extractLowConfidence 通用路径字面量兜底，误报率最高的一档，产出必须
// 带 Confidence: "low" 标记，交给 filterAPICandidates 的黑名单重点清洗。
func extractLowConfidence(text string) []candidate {
	var out []candidate
	for _, m := range lowConfidencePattern.FindAllStringSubmatch(text, -1) {
		if len(m) == 2 {
			out = append(out, candidate{URL: m[1], Confidence: "low"})
		}
	}
	return out
}
