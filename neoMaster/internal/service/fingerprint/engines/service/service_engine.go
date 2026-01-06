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
	CPE      string         `json:"cpe"` // 目标 CPE (可含 $1, $2 占位符)
	Vendor   string         `json:"vendor"`
	Product  string         `json:"product"`
	Part     string         `json:"part"`
}

type ServiceEngine struct {
	repo  assetRepo.AssetCPERepository
	rules []CPERule
	mu    sync.RWMutex
}

func NewServiceEngine(repo assetRepo.AssetCPERepository) *ServiceEngine {
	return &ServiceEngine{
		repo:  repo,
		rules: defaultRules(),
	}
}

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

	for _, rule := range e.rules {
		if rule.Regex == nil {
			continue
		}

		if submatches := rule.Regex.FindStringSubmatch(input.Banner); len(submatches) > 0 {
			// 提取版本号等信息填充 CPE
			cpe := rule.CPE
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
				Product:    rule.Product,
				Vendor:     rule.Vendor,
				Type:       rule.Part, // Map Part to Type
				CPE:        cpe,
				Version:    version,
				Confidence: 90,
				Source:     "service_banner",
			})
		}
	}

	return matches, nil
}

func (e *ServiceEngine) LoadRules(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. 数据库加载
	if path == "db" || e.repo != nil {
		if e.repo != nil {
			dbRules, err := e.repo.FindAll(context.Background())
			if err == nil {
				for _, r := range dbRules {
					re, err := regexp.Compile(r.MatchStr)
					if err != nil {
						continue
					}
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

		byteValue, _ := io.ReadAll(file)
		var ruleFile ServiceRuleFile
		if err := json.Unmarshal(byteValue, &ruleFile); err != nil {
			// Fallback to array for backward compatibility
			var rules []CPERule
			if err2 := json.Unmarshal(byteValue, &rules); err2 != nil {
				// check if it is other engine file
				var genericFile map[string]interface{}
				if err3 := json.Unmarshal(byteValue, &genericFile); err3 == nil {
					if t, ok := genericFile["type"].(string); ok && t != "service" {
						return nil // ignore other type
					}
				}
				return fmt.Errorf("failed to unmarshal service rules (struct or array): %v, %v", err, err2)
			}
			ruleFile.Samples = rules
		} else {
			// Check type if present
			if ruleFile.Type != "" && ruleFile.Type != "service" {
				return nil // ignore other type
			}
		}

		for i := range ruleFile.Samples {
			re, err := regexp.Compile(ruleFile.Samples[i].MatchStr)
			if err != nil {
				continue
			}
			ruleFile.Samples[i].Regex = re
			e.rules = append(e.rules, ruleFile.Samples[i])
		}
	}
	return nil
}

type ServiceRuleFile struct {
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Type    string    `json:"type"`
	Samples []CPERule `json:"samples"`
}

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
