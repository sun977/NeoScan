package service

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"neoagent/internal/pkg/fingerprint"
	"neoagent/internal/pkg/fingerprint/model"
)

// compiledCPERule 包含原始规则和编译后的正则
type compiledCPERule struct {
	Original model.CPERule
	Regex    *regexp.Regexp
}

// ServiceEngine 服务指纹识别引擎
type ServiceEngine struct {
	rules []compiledCPERule
	mu    sync.RWMutex
}

// NewServiceEngine 创建服务指纹识别引擎实例
func NewServiceEngine(rules []model.CPERule) *ServiceEngine {
	e := &ServiceEngine{
		rules: make([]compiledCPERule, 0, len(rules)),
	}
	e.Reload(rules)
	return e
}

// Reload 重新加载规则
func (e *ServiceEngine) Reload(rules []model.CPERule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	compiled := make([]compiledCPERule, 0, len(rules))
	for _, rule := range rules {
		if rule.MatchStr == "" {
			continue
		}
		re, err := regexp.Compile(rule.MatchStr)
		if err != nil {
			// 忽略错误的正则
			continue
		}
		compiled = append(compiled, compiledCPERule{
			Original: rule,
			Regex:    re,
		})
	}
	e.rules = compiled
}

// Type 返回引擎类型
func (e *ServiceEngine) Type() string {
	return "service"
}

func (e *ServiceEngine) Match(input *fingerprint.Input) ([]fingerprint.Match, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var matches []fingerprint.Match

	// 仅当有 Banner 时匹配
	if input.Banner == "" {
		return nil, nil
	}

	// 遍历规则匹配
	for _, rule := range e.rules {
		// 使用正则表达式匹配 Banner
		if submatches := rule.Regex.FindStringSubmatch(input.Banner); len(submatches) > 0 {
			// 提取版本号等信息填充 CPE
			cpe := rule.Original.CPE
			version := ""

			// 简单的占位符替换 logic ($1, $2...)
			for i, match := range submatches {
				if i == 0 {
					continue
				}
				placeholder := fmt.Sprintf("$%d", i)
				if strings.Contains(cpe, placeholder) {
					cpe = strings.ReplaceAll(cpe, placeholder, match)
				}
				// 假设第一个捕获组通常是版本
				if i == 1 {
					version = match
				}
			}

			// 如果规则里没有完整 CPE，使用 Generate 逻辑
			if cpe == "" {
				part := rule.Original.Part
				if part == "" {
					part = "a" // default
				}
				cpe = fmt.Sprintf("cpe:2.3:%s:%s:%s:%s:*:*:*:*:*:*:*", part, rule.Original.Vendor, rule.Original.Product, version)
			}

			matches = append(matches, fingerprint.Match{
				Product:    rule.Original.Product,
				Vendor:     rule.Original.Vendor,
				Type:       rule.Original.Part,
				CPE:        cpe,
				Version:    version,
				Confidence: 90,
				Source:     "service_banner",
			})
		}
	}

	return matches, nil
}
