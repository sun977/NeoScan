package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	assetRepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/fingerprint"
)

// CPERule CPE 映射规则
// 用于直接将正则匹配结果映射到 CPE
type CPERule struct {
	MatchStr string         `json:"match_str"`
	Regex    *regexp.Regexp `json:"-"`
	CPE      string         `json:"cpe"`     // 目标 CPE (可含 $1, $2 占位符)
	Vendor   string         `json:"vendor"`  // 供应商名称
	Product  string         `json:"product"` // 产品名称
	Part     string         `json:"part"`    // CPE 组件类型 (a=application, o=operating_system, h=hardware)
}

// ServiceEngine 服务指纹识别引擎
type ServiceEngine struct {
	repo  assetRepo.AssetCPERepository // 资产 CPE 仓库
	rules []CPERule                    // CPE 映射规则
	mu    sync.RWMutex
}

// NewServiceEngine 创建服务指纹识别引擎实例
func NewServiceEngine(repo assetRepo.AssetCPERepository) *ServiceEngine {
	return &ServiceEngine{
		repo:  repo,
		rules: defaultRules(),
	}
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
		// 检查规则的正则表达式是否已编译
		if rule.Regex == nil {
			continue
		}

		// 使用正则表达式匹配 Banner 中查找匹配
		if submatches := rule.Regex.FindStringSubmatch(input.Banner); len(submatches) > 0 {
			// 提取版本号等信息填充 CPE
			cpe := rule.CPE
			version := ""

			// 简单的占位符替换 logic ($1, $2...)
			for i, match := range submatches {
				if i == 0 {
					// 跳过完整匹配项
					continue
				}
				// 创建占位符
				placeholder := fmt.Sprintf("$%d", i)
				// 如果 CPE 中包含占位符，则替换
				if strings.Contains(cpe, placeholder) {
					cpe = strings.ReplaceAll(cpe, placeholder, match)
				}
				// 假设第一个捕获组通常是版本
				if i == 1 {
					version = match
				}
			}

			// 如果没有占位符，但捕获到了版本，尝试自动修补 CPE
			if version != "" && strings.HasSuffix(cpe, ":*") {
				// 这是一个非常粗糙的 heuristic，实际应完全依赖规则定义
				// 但为了兼容 defaultRules 里的简单定义，这里不做过多魔法
			}

			// 如果规则里没有完整 CPE，使用 Generate 逻辑 (这里简化处理)
			if cpe == "" {
				part := rule.Part
				if part == "" {
					part = "a" // default
				}
				cpe = fmt.Sprintf("cpe:2.3:%s:%s:%s:%s:*:*:*:*:*:*:*", part, rule.Vendor, rule.Product, version)
			}

			matches = append(matches, fingerprint.Match{
				Product:    rule.Product,     // 产品名称
				Vendor:     rule.Vendor,      // 供应商名称
				Type:       rule.Part,        // Map Part to Type
				CPE:        cpe,              // 完整 CPE 字符串
				Version:    version,          // 版本号
				Confidence: 90,               // 置信度
				Source:     "service_banner", // 标识来源
			})
		}
	}

	// 返回匹配结果
	return matches, nil
}

// 加载规则
func (e *ServiceEngine) LoadRules(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. 数据库加载
	if path == "db" || e.repo != nil {
		if e.repo != nil {
			// 从数据库加载 CPE 规则
			dbRules, err := e.repo.FindAll(context.Background())
			if err == nil {
				// 遍历数据库规则
				for _, r := range dbRules {
					// 编译正则表达式
					re, err := regexp.Compile(r.MatchStr)
					if err != nil {
						continue // 编译失败，跳过
					}
					// 将数据库规则添加到引擎规则列表
					e.rules = append(e.rules, CPERule{
						MatchStr: r.MatchStr,
						Regex:    re,
						CPE:      r.CPE,
						Vendor:   r.Vendor,
						Product:  r.Product,
						Part:     r.Part,
					})
				}
			}
		}
		if path == "db" {
			return nil
		}
	}

	// 2. 文件加载
	if path != "" && path != "db" {
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open service rules file: %w", err)
		}
		defer file.Close()

		// 读取文件内容
		byteValue, _ := io.ReadAll(file)

		// 尝试解析为 Service Rule 文件格式
		var ruleFile ServiceRuleFile
		if err := json.Unmarshal(byteValue, &ruleFile); err != nil {
			// 回退到数组格式（向后兼容）
			var rules []CPERule
			if err2 := json.Unmarshal(byteValue, &rules); err2 != nil {
				// 检查是否为其他引擎的文件
				var genericFile map[string]interface{}
				if err3 := json.Unmarshal(byteValue, &genericFile); err3 == nil {
					if t, ok := genericFile["type"].(string); ok && t != "service" {
						return nil // 忽略其他类型的文件
					}
				}
				return fmt.Errorf("failed to unmarshal service rules (struct or array): %v, %v", err, err2)
			}
			ruleFile.Samples = rules
		} else {
			// 检查类型字段是否为 "service"
			if ruleFile.Type != "" && ruleFile.Type != "service" {
				return nil // 忽略其他类型的文件
			}
		}

		// 遍历规则样本并编译正则表达式
		for i := range ruleFile.Samples {
			re, err := regexp.Compile(ruleFile.Samples[i].MatchStr)
			if err != nil {
				continue // 忽略编译失败的规则
			}
			// 编译成功，添加到引擎规则列表
			ruleFile.Samples[i].Regex = re
			e.rules = append(e.rules, ruleFile.Samples[i])
		}
	}
	return nil
}

// ServiceRuleFile 服务规则文件结构
type ServiceRuleFile struct {
	Name    string    `json:"name"`    // 规则文件名称
	Version string    `json:"version"` // 规则文件版本
	Type    string    `json:"type"`    // 文件类型，必须为 "service"
	Samples []CPERule `json:"samples"` // 服务规则样本列表
}

// defaultRules 默认 cpe 服务规则
func defaultRules() []CPERule {
	return []CPERule{
		{
			MatchStr: `(?i)^SSH-[\d\.]+-OpenSSH_([\w\.]+)`,
			Regex:    regexp.MustCompile(`(?i)^SSH-[\d\.]+-OpenSSH_([\w\.]+)`),
			Vendor:   "openbsd",
			Product:  "openssh",
			Part:     "a",
			CPE:      "cpe:2.3:a:openbsd:openssh:$1:*:*:*:*:*:*:*",
		},
		{
			MatchStr: `(?i)nginx/([\d\.]+)`,
			Regex:    regexp.MustCompile(`(?i)nginx/([\d\.]+)`),
			Vendor:   "f5",
			Product:  "nginx",
			Part:     "a",
			CPE:      "cpe:2.3:a:f5:nginx:$1:*:*:*:*:*:*:*",
		},
		{
			MatchStr: `(?i)Apache/([\d\.]+)`,
			Regex:    regexp.MustCompile(`(?i)Apache/([\d\.]+)`),
			Vendor:   "apache",
			Product:  "http_server",
			Part:     "a",
			CPE:      "cpe:2.3:a:apache:http_server:$1:*:*:*:*:*:*:*",
		},
	}
}
