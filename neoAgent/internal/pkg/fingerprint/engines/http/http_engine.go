package http

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"neoagent/internal/pkg/fingerprint"
	"neoagent/internal/pkg/fingerprint/model"
	"neoagent/internal/pkg/matcher"
)

// CompiledRule 包含原始规则和编译后的 MatchRule
type CompiledRule struct {
	Original model.FingerRule
	Matcher  matcher.MatchRule
}

// HTTPEngine HTTP 指纹识别引擎
type HTTPEngine struct {
	rules []CompiledRule
	mu    sync.RWMutex
}

// NewHTTPEngine 创建 HTTP 指纹识别引擎
func NewHTTPEngine(rules []model.FingerRule) *HTTPEngine {
	e := &HTTPEngine{
		rules: make([]CompiledRule, 0, len(rules)),
	}
	e.Reload(rules)
	return e
}

// Reload 重新加载规则
func (e *HTTPEngine) Reload(rules []model.FingerRule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	compiled := make([]CompiledRule, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		compiled = append(compiled, compileRule(rule))
	}
	e.rules = compiled
}

// Type 返回引擎类型
func (e *HTTPEngine) Type() string {
	return "http"
}

// Match 执行指纹匹配
func (e *HTTPEngine) Match(input *fingerprint.Input) ([]fingerprint.Match, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var matches []fingerprint.Match
	data := convertInputToMap(input)

	for _, rule := range e.rules {
		matched, err := matcher.Match(data, rule.Matcher)
		if err != nil {
			continue
		}

		if matched {
			matches = append(matches, fingerprint.Match{
				Product:    rule.Original.Name,
				Vendor:     guessVendor(rule.Original.Name),
				Type:       "app",
				CPE:        generateCPE(rule.Original.Name),
				Confidence: 95,
				Source:     "http_engine",
			})
		}
	}

	return matches, nil
}

// compileRule 将扁平的 FingerRule 转换为 MatchRule
func compileRule(rule model.FingerRule) CompiledRule {
	var conditions []matcher.MatchRule

	// 1. Status Code
	if rule.StatusCode != "" {
		if strings.Contains(rule.StatusCode, ",") {
			// List check
			codes := strings.Split(rule.StatusCode, ",")
			for i := range codes {
				codes[i] = strings.TrimSpace(codes[i])
			}
			conditions = append(conditions, matcher.MatchRule{
				Field:    "status_code",
				Operator: "in",
				Value:    codes,
			})
		} else {
			conditions = append(conditions, matcher.MatchRule{
				Field:    "status_code",
				Operator: "equals",
				Value:    strings.TrimSpace(rule.StatusCode),
			})
		}
	}

	// 2. Body
	if rule.Body != "" {
		conditions = append(conditions, matcher.MatchRule{
			Field:    "body",
			Operator: "contains",
			Value:    rule.Body,
		})
	}

	// 3. Header (Specific header check)
	if rule.Header != "" {
		// 假设 header 字段格式为 "Key: Value"
		if strings.Contains(rule.Header, ":") {
			parts := strings.SplitN(rule.Header, ":", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			conditions = append(conditions, matcher.MatchRule{
				Field:      "headers." + key,
				Operator:   "contains",
				Value:      val,
				IgnoreCase: true,
			})
		} else {
			// 否则在 all_headers 中查找
			conditions = append(conditions, matcher.MatchRule{
				Field:      "all_headers",
				Operator:   "contains",
				Value:      rule.Header,
				IgnoreCase: true,
			})
		}
	}

	// 4. Match (JSON rule or Regex)
	if rule.Match != "" {
		var complexRule matcher.MatchRule
		if strings.HasPrefix(strings.TrimSpace(rule.Match), "{") {
			if err := json.Unmarshal([]byte(rule.Match), &complexRule); err == nil {
				conditions = append(conditions, complexRule)
			}
		} else {
			// Treat as regex on all_response
			if re, err := regexp.Compile(rule.Match); err == nil {
				conditions = append(conditions, matcher.MatchRule{
					Field:    "all_response",
					Operator: "regex",
					Value:    re,
				})
			}
		}
	}

	// 5. Title
	if rule.Title != "" {
		conditions = append(conditions, matcher.MatchRule{
			Field:      "title",
			Operator:   "contains",
			Value:      rule.Title,
			IgnoreCase: true,
		})
	}

	if len(conditions) == 0 {
		// Empty rule matches nothing
		return CompiledRule{
			Original: rule,
			Matcher: matcher.MatchRule{
				Field:    "always_false",
				Operator: "equals",
				Value:    "unexpected_value",
			},
		}
	}

	return CompiledRule{
		Original: rule,
		Matcher: matcher.MatchRule{
			And: conditions,
		},
	}
}

func convertInputToMap(input *fingerprint.Input) map[string]interface{} {
	data := make(map[string]interface{})
	data["body"] = input.Body
	data["title"] = extractTitle(input.Body)
	data["headers"] = input.Headers

	// Status Code handling
	if input.StatusCode != 0 {
		data["status_code"] = fmt.Sprintf("%d", input.StatusCode)
	}

	var allHeadersBuilder strings.Builder
	for k, v := range input.Headers {
		allHeadersBuilder.WriteString(k)
		allHeadersBuilder.WriteString(": ")
		allHeadersBuilder.WriteString(v)
		allHeadersBuilder.WriteString("\n")
	}
	allHeadersStr := allHeadersBuilder.String()
	data["all_headers"] = allHeadersStr
	data["all_response"] = allHeadersStr + "\n" + input.Body

	return data
}

func extractTitle(body string) string {
	low := strings.ToLower(body)
	start := strings.Index(low, "<title>")
	if start == -1 {
		return ""
	}
	start += 7
	end := strings.Index(low[start:], "</title>")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(body[start : start+end])
}

func guessVendor(product string) string {
	return strings.ToLower(strings.ReplaceAll(product, " ", "_"))
}

func generateCPE(product string) string {
	p := guessVendor(product)
	return fmt.Sprintf("cpe:2.3:a:%s:%s:*:*:*:*:*:*:*:*", p, p)
}
