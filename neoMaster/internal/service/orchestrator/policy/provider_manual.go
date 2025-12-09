// ManualProvider 人工输入数据来源
// 实现 TargetProvider 接口，从人工输入字符串解析目标
// 功能：
// 1. 支持逗号、换行、分号分隔的目标列表
// 2. 过滤空行、去重
// 3. 解析配置，将每个目标作为 Target.Value 保存

package policy

import (
	"context"
	"strings"
)

// ManualProvider 人工输入
type ManualProvider struct{}

func (m *ManualProvider) Name() string { return "manual" }

func (m *ManualProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	parts := strings.FieldsFunc(config.SourceValue, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	targets := make([]Target, 0, len(parts))
	for _, part := range parts {
		t := strings.TrimSpace(part)
		if t != "" {
			targets = append(targets, Target{
				Type:   config.TargetType, // 使用配置中的类型
				Value:  t,
				Source: "manual",
				Meta:   nil,
			})
		}
	}
	return targets, nil
}

func (m *ManualProvider) HealthCheck(ctx context.Context) error {
	return nil
}
