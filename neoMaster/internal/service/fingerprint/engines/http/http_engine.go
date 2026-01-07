package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/matcher"                    // 匹配器
	assetRepo "neomaster/internal/repo/mysql/asset"     // 资产仓库
	"neomaster/internal/service/fingerprint"            // 指纹识别服务
	"neomaster/internal/service/fingerprint/converters" // 指纹识别转换器
)

// CustomRuleFile 本地 CMS 规则文件结构
type CustomRuleFile struct {
	Name    string       `json:"name"`    // 规则名称
	Version string       `json:"version"` // 规则版本
	Type    string       `json:"type"`    // 规则类型
	Samples []CustomRule `json:"samples"` // 规则样本规则
}

type CustomRule struct {
	Name string             `json:"name"` // 规则样本名称
	Rule *asset.AssetFinger `json:"rule"` // 指向资产指纹 AssetFinger 结构体的指针
}

// CompiledRule 包含原始规则和编译后的 MatchRule
type CompiledRule struct {
	Original asset.AssetFinger // 原始规则
	Matcher  matcher.MatchRule // 匹配规则
}

// HTTPEngine 统一 HTTP 指纹识别引擎
// 负责处理所有基于 HTTP 特征的指纹识别 (CMS, Web Framework, Middleware 等)
// 数据源支持: 数据库 (asset_finger), 本地文件 (Goby JSON, Custom JSON)
// 所有规则最终统一转换为 asset.AssetFinger 结构进行匹配
type HTTPEngine struct {
	repo  assetRepo.AssetFingerRepository // 资产指纹仓库
	rules []CompiledRule                  // 编译后的规则
	mu    sync.RWMutex                    // 读写锁
}

// NewHTTPEngine 创建新的 HTTP 指纹识别引擎
func NewHTTPEngine(repo assetRepo.AssetFingerRepository) *HTTPEngine {
	return &HTTPEngine{
		repo:  repo,
		rules: make([]CompiledRule, 0), // 初始化空规则列表
	}
}

// Type 获取指纹识别引擎类型
func (e *HTTPEngine) Type() string {
	return "http"
}

// Match 执行指纹匹配
func (e *HTTPEngine) Match(input *fingerprint.Input) ([]fingerprint.Match, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 初始化匹配结果列表
	var matches []fingerprint.Match

	// 准备匹配数据 输入数据转化成匹配器可用的数据结构
	data := convertInputToMap(input)

	// 遍历所有编译后的规则进行匹配
	for _, rule := range e.rules {
		// 使用匹配器评估数据是否符合规则
		matched, err := matcher.Match(data, rule.Matcher)
		if err != nil {
			// 记录错误但继续匹配其他规则?
			continue
		}

		// 匹配成功，创建匹配结果，添加到结果列表
		if matched {
			matches = append(matches, fingerprint.Match{
				Product:    rule.Original.Name,              // 产品名称
				Vendor:     guessVendor(rule.Original.Name), // 供应商名称
				Type:       "app",                           // Web 资产统一视为 app
				CPE:        generateCPE(rule.Original.Name), // 生成 CPE 2.3 字符串  【? 为什么要生成】
				Confidence: 95,                              // 统一置信度
				Source:     "http_engine",                   // 来源标识
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
			// 从数据库加载web资产指纹规则
			dbRules, err := e.repo.FindAll(context.Background())
			if err == nil {
				// 将数据库规则转换为 CompiledRule 编译并添加到引擎规则列表
				for _, r := range dbRules {
					e.rules = append(e.rules, compileRule(*r))
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

// loadFromFile 从文件加载规则
func (e *HTTPEngine) loadFromFile(path string) error {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open rule file: %w", err)
	}
	defer file.Close()

	// 读取文件内容
	byteValue, _ := io.ReadAll(file)

	// 尝试解析为 Custom Rule
	var customFile CustomRuleFile
	if err := json.Unmarshal(byteValue, &customFile); err == nil {
		if customFile.Type == "http" || len(customFile.Samples) > 0 {
			// 遍历自定义规则样本
			for _, sample := range customFile.Samples {
				if sample.Rule != nil {
					sample.Rule.Name = sample.Name
					// 编译规则并添加到引擎规则列表
					e.rules = append(e.rules, compileRule(*sample.Rule))
				}
			}
			return nil
		}
	}

	// 尝试解析为 Goby Rule
	var gobyFile converters.GobyRuleFile
	if err := json.Unmarshal(byteValue, &gobyFile); err == nil && len(gobyFile.Rule) > 0 {
		// 遍历 Goby 规则样本
		for _, rule := range gobyFile.Rule {
			// 转换 Goby 规则为 CMS 规则
			cmsRule := converters.ConvertGobyToCMS(&rule)
			if cmsRule != nil {
				// 编译规则并添加到引擎规则列表
				e.rules = append(e.rules, compileRule(*cmsRule))
			}
		}
		return nil
	}

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

// compileRule 将 AssetFinger 转换为 CompiledRule
func compileRule(rule asset.AssetFinger) CompiledRule {
	var conditions []matcher.MatchRule

	// 1. 处理标准字段 (隐式 AND)

	// Title
	if rule.Title != "" {
		conditions = append(conditions, matcher.MatchRule{
			Field:      "title",
			Operator:   "contains",
			Value:      rule.Title,
			IgnoreCase: true,
		})
	}

	// Header
	if rule.Header != "" {
		conditions = append(conditions, matcher.MatchRule{
			Field:      "all_headers",
			Operator:   "contains",
			Value:      rule.Header,
			IgnoreCase: true,
		})
	}

	// Server (支持 server 字段或 header 中的 Server)
	if rule.Server != "" {
		conditions = append(conditions, matcher.MatchRule{
			Or: []matcher.MatchRule{
				{Field: "server", Operator: "contains", Value: rule.Server, IgnoreCase: true},
				{Field: "all_headers", Operator: "contains", Value: rule.Server, IgnoreCase: true},
			},
		})
	}

	// X-Powered-By
	if rule.XPoweredBy != "" {
		conditions = append(conditions, matcher.MatchRule{
			Or: []matcher.MatchRule{
				{Field: "x_powered_by", Operator: "contains", Value: rule.XPoweredBy, IgnoreCase: true},
				{Field: "all_headers", Operator: "contains", Value: rule.XPoweredBy, IgnoreCase: true},
			},
		})
	}

	// Body / Response / Footer / Subtitle (统一查 Body)
	for _, val := range []string{rule.Body, rule.Response, rule.Footer, rule.Subtitle} {
		if val != "" {
			conditions = append(conditions, matcher.MatchRule{
				Field:      "body",
				Operator:   "contains",
				Value:      val,
				IgnoreCase: true,
			})
		}
	}

	// Status Code
	if rule.StatusCode != "" {
		conditions = append(conditions, matcher.MatchRule{
			Field:    "status_code",
			Operator: "equals",
			Value:    rule.StatusCode,
		})
	}

	// 2. 处理 Match 字段 (高级规则或正则)
	if rule.Match != "" {
		// 尝试解析为 JSON MatchRule
		var complexRule matcher.MatchRule
		if strings.HasPrefix(strings.TrimSpace(rule.Match), "{") {
			if err := json.Unmarshal([]byte(rule.Match), &complexRule); err == nil {
				conditions = append(conditions, complexRule)
			} else {
				// 解析失败，回退为正则
				if re, err := regexp.Compile(rule.Match); err == nil {
					conditions = append(conditions, matcher.MatchRule{
						Field:    "all_response",
						Operator: "regex",
						Value:    re, // 使用预编译的正则
					})
				}
			}
		} else {
			// 默认为正则
			if re, err := regexp.Compile(rule.Match); err == nil {
				conditions = append(conditions, matcher.MatchRule{
					Field:    "all_response",
					Operator: "regex",
					Value:    re, // 使用预编译的正则
				})
			}
		}
	}

	// 如果没有生成任何规则，返回一个"空"规则 (实际上 matchRule 会返回 false)
	// 但 matcher.Match 如果 And 为空会返回 true，所以我们需要处理这种情况
	if len(conditions) == 0 {
		// 创建一个永远为 false 的规则
		return CompiledRule{
			Original: rule,
			Matcher: matcher.MatchRule{
				Field:    "always_false",
				Operator: "equals",
				Value:    "unexpected_value",
			},
		}
	}

	// 返回编译后的规则，所有条件通过 AND 连接
	return CompiledRule{
		Original: rule,
		Matcher: matcher.MatchRule{
			And: conditions,
		},
	}
}

// convertInputToMap 将输入转换为 matcher 可用的 map
func convertInputToMap(input *fingerprint.Input) map[string]interface{} {
	data := make(map[string]interface{})

	// 基础字段
	data["body"] = input.Body                // 原始 Body 响应体
	data["title"] = extractTitle(input.Body) // 从 Body 中提取标题

	// Headers
	data["headers"] = input.Headers // 支持 headers.Key 访问

	// 构建 all_headers 字符串
	// 将所有 Header 拼接为一个字符串 【?】
	var allHeadersBuilder strings.Builder
	for k, v := range input.Headers {
		allHeadersBuilder.WriteString(k)
		allHeadersBuilder.WriteString(": ")
		allHeadersBuilder.WriteString(v)
		allHeadersBuilder.WriteString("\n")
	}
	allHeadersStr := allHeadersBuilder.String()
	data["all_headers"] = allHeadersStr

	// 特殊 Header 提取 (方便快速访问)
	if val, ok := input.Headers["Server"]; ok {
		data["server"] = val
	} else if val, ok := input.Headers["server"]; ok {
		data["server"] = val
	}

	if val, ok := input.Headers["X-Powered-By"]; ok {
		data["x_powered_by"] = val
	} else if val, ok := input.Headers["x-powered-by"]; ok {
		data["x_powered_by"] = val
	}

	// All Response (Headers + Body)
	data["all_response"] = allHeadersStr + "\n" + input.Body

	return data
}

// extractTitle 从 HTML 中提取标题
func extractTitle(body string) string {
	low := strings.ToLower(body)
	// 查找 <title> 标签
	start := strings.Index(low, "<title>")
	if start == -1 {
		return "" // 没有找到标题
	}
	// 查找 </title> 标签
	end := strings.Index(low[start:], "</title>")
	if end == -1 {
		return "" // 没有找到结束标签
	}
	// 返回原始字符串中的标题内容
	return body[start+7 : start+end]
}

// guessVendor 猜测供应商名称（小写）
func guessVendor(product string) string {
	return strings.ToLower(product)
}

// generateCPE 生成 CPE 模板
func generateCPE(product string) string {
	p := strings.ToLower(product)
	p = strings.ReplaceAll(p, " ", "_")
	return fmt.Sprintf("cpe:2.3:a:%s:%s:*:*:*:*:*:*:*:*", p, p)
}
