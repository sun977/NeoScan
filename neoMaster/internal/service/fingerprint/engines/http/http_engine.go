package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"neomaster/internal/model/asset"
	assetRepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/fingerprint"
	"neomaster/internal/service/fingerprint/converters"
)

// CustomRuleFile 本地 CMS 规则文件结构
type CustomRuleFile struct {
	Name    string       `json:"name"`
	Version string       `json:"version"`
	Type    string       `json:"type"`
	Samples []CustomRule `json:"samples"`
}

type CustomRule struct {
	Name string             `json:"name"`
	Rule *asset.AssetFinger `json:"rule"`
}

// HTTPEngine 统一 HTTP 指纹识别引擎
// 负责处理所有基于 HTTP 特征的指纹识别 (CMS, Web Framework, Middleware 等)
// 数据源支持: 数据库 (asset_finger), 本地文件 (Goby JSON, Custom JSON)
// 所有规则最终统一转换为 asset.AssetFinger 结构进行匹配
type HTTPEngine struct {
	repo  assetRepo.AssetFingerRepository
	rules []asset.AssetFinger
	mu    sync.RWMutex
}

func NewHTTPEngine(repo assetRepo.AssetFingerRepository) *HTTPEngine {
	return &HTTPEngine{
		repo:  repo,
		rules: make([]asset.AssetFinger, 0),
	}
}

func (e *HTTPEngine) Type() string {
	return "http"
}

// Match 执行指纹匹配
func (e *HTTPEngine) Match(input *fingerprint.Input) ([]fingerprint.Match, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var matches []fingerprint.Match

	for _, rule := range e.rules {
		if matchRule(input, &rule) {
			matches = append(matches, fingerprint.Match{
				Product:    rule.Name,
				Vendor:     guessVendor(rule.Name),
				Type:       "app", // Web 资产统一视为 app
				CPE:        generateCPE(rule.Name),
				Confidence: 95, // 统一置信度
				Source:     "http_engine",
			})
		}
	}

	return matches, nil
}

// LoadRules 加载规则
// path:
//   - "db": 从数据库加载
//   - *.json: 根据内容自动判断是 Goby 还是 Custom 格式，并统一转换为 AssetFinger
func (e *HTTPEngine) LoadRules(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. 数据库加载
	if path == "db" || e.repo != nil {
		if e.repo != nil {
			dbRules, err := e.repo.FindAll(context.Background())
			if err == nil {
				for _, r := range dbRules {
					e.rules = append(e.rules, *r)
				}
			}
		}
		if path == "db" {
			return nil
		}
	}

	// 2. 文件加载
	if path != "" && path != "db" {
		return e.loadFromFile(path)
	}

	return nil
}

func (e *HTTPEngine) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open rule file: %w", err)
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	// 尝试解析为 Custom Rule
	var customFile CustomRuleFile
	if err := json.Unmarshal(byteValue, &customFile); err == nil {
		if customFile.Type == "http" || len(customFile.Samples) > 0 {
			for _, sample := range customFile.Samples {
				if sample.Rule != nil {
					sample.Rule.Name = sample.Name
					e.rules = append(e.rules, *sample.Rule)
				}
			}
			return nil
		}
	}

	// 尝试解析为 Goby Rule
	var gobyFile converters.GobyRuleFile
	if err := json.Unmarshal(byteValue, &gobyFile); err == nil && len(gobyFile.Rule) > 0 {
		for _, rule := range gobyFile.Rule {
			cmsRule := converters.ConvertGobyToCMS(&rule)
			if cmsRule != nil {
				e.rules = append(e.rules, *cmsRule)
			}
		}
		return nil
	}

	// 如果解析失败但不是因为格式错误（例如空文件或不匹配的类型），我们应该忽略还是报错？
	// 这里我们返回一个错误，表明该文件不是预期的格式，让上层决定是否忽略。
	// 但考虑到 loadFromDir 会遍历所有文件，这里报错会导致整个加载过程中断。
	// 最好是：如果无法识别，返回特定的错误，或者在 loadFromDir 中处理。

	// 检查是否是其他引擎的文件
	var genericFile map[string]interface{}
	if err := json.Unmarshal(byteValue, &genericFile); err == nil {
		if t, ok := genericFile["type"].(string); ok && t != "http" {
			// 不是 http 类型的规则文件，忽略
			return nil
		}
	}

	return fmt.Errorf("unknown rule file format: %s", path)
}

// matchRule 统一匹配逻辑 (AssetFinger 结构)
func matchRule(input *fingerprint.Input, rule *asset.AssetFinger) bool {
	// 如果规则没有任何匹配条件，返回 false
	if rule.Title == "" && rule.Header == "" && rule.Server == "" &&
		rule.XPoweredBy == "" && rule.Body == "" && rule.Response == "" &&
		rule.Footer == "" && rule.Subtitle == "" {
		return false
	}

	// Title
	if rule.Title != "" {
		if !strings.Contains(strings.ToLower(extractTitle(input.Body)), strings.ToLower(rule.Title)) {
			return false
		}
	}

	// Header / Server / X-Powered-By
	if rule.Header != "" || rule.Server != "" || rule.XPoweredBy != "" {
		allHeaders := ""
		for k, v := range input.Headers {
			allHeaders += k + ": " + v + "\n"
		}
		allHeaders = strings.ToLower(allHeaders)

		if rule.Header != "" && !strings.Contains(allHeaders, strings.ToLower(rule.Header)) {
			return false
		}
		if rule.Server != "" {
			if !strings.Contains(allHeaders, "server: "+strings.ToLower(rule.Server)) &&
				!strings.Contains(allHeaders, strings.ToLower(rule.Server)) { // 宽容匹配
				return false
			}
		}
		if rule.XPoweredBy != "" {
			if !strings.Contains(allHeaders, "x-powered-by: "+strings.ToLower(rule.XPoweredBy)) &&
				!strings.Contains(allHeaders, strings.ToLower(rule.XPoweredBy)) {
				return false
			}
		}
	}

	// Body / Response / Footer / Subtitle (简化为 Body 包含)
	bodyLower := strings.ToLower(input.Body)
	if rule.Body != "" && !strings.Contains(bodyLower, strings.ToLower(rule.Body)) {
		return false
	}
	if rule.Response != "" && !strings.Contains(bodyLower, strings.ToLower(rule.Response)) {
		return false
	}
	if rule.Footer != "" && !strings.Contains(bodyLower, strings.ToLower(rule.Footer)) {
		return false
	}
	if rule.Subtitle != "" && !strings.Contains(bodyLower, strings.ToLower(rule.Subtitle)) {
		return false
	}

	return true
}

func extractTitle(body string) string {
	low := strings.ToLower(body)
	start := strings.Index(low, "<title>")
	if start == -1 {
		return ""
	}
	end := strings.Index(low[start:], "</title>")
	if end == -1 {
		return ""
	}
	return body[start+7 : start+end]
}

func guessVendor(product string) string {
	return strings.ToLower(product)
}

func generateCPE(product string) string {
	p := strings.ToLower(product)
	p = strings.ReplaceAll(p, " ", "_")
	return fmt.Sprintf("cpe:2.3:a:%s:%s:*:*:*:*:*:*:*:*", p, p)
}
