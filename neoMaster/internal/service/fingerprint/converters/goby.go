package converters

import (
	"strings"

	"neomaster/internal/model/asset"
)

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
	// 如果 Goby 规则包含 OR (||)，我们需要拆分为多个 AssetFinger 规则 (但这里为了简化，只处理最简单的 AND 情况或取第一个条件)
	// 更好的做法是：AssetFinger 的 Match 字段存储原始规则字符串，Match 逻辑中支持简单表达式。
	// 但根据需求 "指纹结构要支持两种"，我们尽量映射到 AssetFinger 的字段中。

	// 这里采用一种折中方案：解析 Goby 规则，如果是简单的 k=v，则映射字段；
	// 如果复杂，则暂不支持或仅提取部分。

	// 示例: header="thinkphp" || body="thinkphp"
	// 这种 OR 逻辑在 AssetFinger 单条记录无法表达（AssetFinger 字段间是 AND）。
	// 如果必须支持，需要将一条 Goby 规则拆分为多条 AssetFinger 规则。
	// 这里为了演示，我们只解析不含 || 的简单规则，或者只取 || 的第一部分。

	parts := strings.Split(goby.Rule, "||")
	// 针对每个 OR 部分，其实应该生成一个新的 CMS 规则，但函数签名限制只返回一个。
	// TODO: 架构优化 - 应该返回 []*asset.AssetFinger
	// 暂时只取第一部分
	firstPart := strings.TrimSpace(parts[0])

	subParts := strings.Split(firstPart, "&&")
	for _, part := range subParts {
		parseConditionToCMS(strings.TrimSpace(part), cmsRule)
	}

	return cmsRule
}

func parseConditionToCMS(condition string, cmsRule *asset.AssetFinger) {
	idx := strings.Index(condition, "=")
	if idx == -1 {
		return
	}
	field := strings.TrimSpace(condition[:idx])
	value := strings.Trim(strings.TrimSpace(condition[idx+1:]), `"'`)

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
