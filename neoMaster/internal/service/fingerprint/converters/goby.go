package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	"neomaster/internal/model/asset"
)

// GobyConverter 实现 RuleConverter 接口
type GobyConverter struct{}

// NewGobyConverter 创建 Goby 转换器
func NewGobyConverter() *GobyConverter {
	return &GobyConverter{}
}

// Decode 实现 RuleConverter 接口
func (c *GobyConverter) Decode(data []byte) ([]*asset.AssetFinger, []*asset.AssetCPE, error) {
	var gobyFile GobyRuleFile
	if err := json.Unmarshal(data, &gobyFile); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal goby rules: %w", err)
	}

	var fingers []*asset.AssetFinger
	for _, rule := range gobyFile.Rule {
		finger := ConvertGobyToCMS(&rule)
		// 标记来源为 goby
		finger.Source = "goby"
		fingers = append(fingers, finger)
	}

	// Goby 规则主要是 CMS 指纹，不涉及 CPE
	return fingers, nil, nil
}

// Encode 实现 RuleConverter 接口 (不支持导出为 Goby 格式)
func (c *GobyConverter) Encode(fingers []*asset.AssetFinger, cpes []*asset.AssetCPE) ([]byte, error) {
	return nil, fmt.Errorf("export to goby format not supported")
}

// GobyRule Goby 原生规则结构 (仅用于解析转换)
type GobyRule struct {
	Name     string `json:"name"`
	Level    string `json:"level"`
	SoftHard string `json:"soft_hard"`
	Rule     string `json:"rule"`
	Product  string `json:"product"`
	Company  string `json:"company"`
	Category string `json:"category"`
}

type GobyRuleFile struct {
	Rule []GobyRule `json:"rule"`
}

// ConvertGobyToCMS 将 Goby 规则转换为统一的 AssetFinger 结构
// Goby Rule 示例: header="thinkphp" || body="thinkphp"
func ConvertGobyToCMS(goby *GobyRule) *asset.AssetFinger {
	cmsRule := &asset.AssetFinger{
		Name: goby.Product, // 优先使用 Product 作为名称
	}
	if cmsRule.Name == "" {
		cmsRule.Name = goby.Name
	}

	// 简单解析器: 提取关键特征填入 AssetFinger
	// 注意: AssetFinger 是 AND 逻辑，而 Goby 可能是 OR 逻辑
	// 如果 Goby 规则包含 OR (||)，我们只取第一部分作为主规则。
	// TODO: 对于复杂的 OR 逻辑，未来应考虑拆分为多条规则或支持复杂匹配语法。

	// 处理规则字符串
	ruleStr := goby.Rule

	// 1. 处理 OR (||) - 只取第一部分
	if idx := strings.Index(ruleStr, "||"); idx != -1 {
		ruleStr = strings.TrimSpace(ruleStr[:idx])
	}

	// 2. 处理 AND (&&) - 拆分条件
	conditions := strings.Split(ruleStr, "&&")
	for _, cond := range conditions {
		parseConditionToCMS(strings.TrimSpace(cond), cmsRule)
	}

	return cmsRule
}

func parseConditionToCMS(condition string, cmsRule *asset.AssetFinger) {
	// 支持 = 和 ==
	// 示例: body="test" 或 body=="test"
	idx := strings.Index(condition, "=")
	if idx == -1 {
		// 可能是单值? 暂不支持
		return
	}

	field := strings.TrimSpace(condition[:idx])
	// 处理 ==
	valuePart := condition[idx+1:]
	valuePart = strings.TrimPrefix(valuePart, "=")

	value := strings.Trim(strings.TrimSpace(valuePart), `"'`)

	switch strings.ToLower(field) {
	case "header":
		cmsRule.Header = value
	case "body":
		cmsRule.Body = value
	case "title":
		cmsRule.Title = value
	case "server":
		cmsRule.Server = value
	case "protocol":
		// ignore
	}
}
