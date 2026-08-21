// Package engine 提供 DirScanner 的 HTTP 请求、过滤和通配符检测能力。
// 此包仅供 scanner/dir 包内部使用（原子扫描器隔离原则）。
package engine

import (
	"regexp"
	"strings"
)

// Response 封装 HTTP 响应的关键信息，由 Requester 填充。
type Response struct {
	StatusCode  int
	Body        string
	Size        int64
	ContentType string
	Location    string // 重定向目标（301/302 时）
	Words       int    // 响应体词数（由 Requester 计算）
	Lines       int    // 响应体行数（由 Requester 计算）
	Title       string // HTML <title> 内容（由 Requester 提取）
}

// FilterConfig 过滤器配置。
type FilterConfig struct {
	// Layer 1：状态码过滤
	IncludeStatus []int // 白名单（非空时，不在白名单内的一律排除）
	ExcludeStatus []int // 黑名单（默认 [404, 500, 502, 503]）

	// Layer 1c：响应大小过滤
	ExcludeSize []int64 // 精确大小黑名单（字节）

	// Layer 3：内容过滤
	ExcludeKeywords []string         // Body 含任意关键词时排除
	ExcludeRegex    []*regexp.Regexp // Body 匹配任意正则时排除
}

// DefaultExcludeStatus 返回默认状态码黑名单。
func DefaultExcludeStatus() []int {
	return []int{404, 500, 502, 503}
}

// Filter 执行多层过滤，判断响应是否应当输出为命中结果。
type Filter struct {
	cfg         FilterConfig
	includeSet  map[int]bool
	excludeSet  map[int]bool
	excludeSizeSet map[int64]bool
}

// NewFilter 创建过滤器。
// 若 cfg.ExcludeStatus 为空，自动使用默认黑名单。
func NewFilter(cfg FilterConfig) *Filter {
	if len(cfg.ExcludeStatus) == 0 {
		cfg.ExcludeStatus = DefaultExcludeStatus()
	}

	f := &Filter{
		cfg:            cfg,
		includeSet:     make(map[int]bool, len(cfg.IncludeStatus)),
		excludeSet:     make(map[int]bool, len(cfg.ExcludeStatus)),
		excludeSizeSet: make(map[int64]bool, len(cfg.ExcludeSize)),
	}
	for _, code := range cfg.IncludeStatus {
		f.includeSet[code] = true
	}
	for _, code := range cfg.ExcludeStatus {
		f.excludeSet[code] = true
	}
	for _, size := range cfg.ExcludeSize {
		f.excludeSizeSet[size] = true
	}
	return f
}

// Match 对响应执行三层过滤，返回 true 表示应当作为命中结果输出。
//
// Layer 1a: 白名单过滤（IncludeStatus 非空时启用）
// Layer 1b: 黑名单过滤（ExcludeStatus）
// Layer 1c: 响应大小黑名单过滤（ExcludeSize）
// Layer 3a: 关键词排除（ExcludeKeywords）
// Layer 3b: 正则排除（ExcludeRegex）
func (f *Filter) Match(resp *Response, path string) bool {
	// Layer 1a：白名单（非空时，不在白名单内的排除）
	if len(f.includeSet) > 0 && !f.includeSet[resp.StatusCode] {
		return false
	}

	// Layer 1b：黑名单
	if f.excludeSet[resp.StatusCode] {
		return false
	}

	// Layer 1c：响应大小黑名单
	if len(f.excludeSizeSet) > 0 && f.excludeSizeSet[resp.Size] {
		return false
	}

	// Layer 3a：关键词排除（Body 中包含任意关键词则排除）
	if len(f.cfg.ExcludeKeywords) > 0 {
		for _, kw := range f.cfg.ExcludeKeywords {
			if strings.Contains(resp.Body, kw) {
				return false
			}
		}
	}

	// Layer 3b：正则排除
	if len(f.cfg.ExcludeRegex) > 0 {
		for _, re := range f.cfg.ExcludeRegex {
			if re.MatchString(resp.Body) {
				return false
			}
		}
	}

	return true
}
