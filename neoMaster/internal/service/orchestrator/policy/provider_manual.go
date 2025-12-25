// ManualProvider 人工输入数据来源
// 实现 TargetProvider 接口，从人工输入字符串解析目标
// 功能：
// 1. 支持逗号、换行、分号分隔的目标列表
// 2. 过滤空行、去重
// 3. 解析配置，将每个目标作为 Target.Value 保存
// 注意：静态数据,且没有对数据进行校验,本质是只字符串分割器
// 不论输入 192.168.0.1/24 还是 192.168.0.1-192.168.0.100 甚至乱码,都会原样传给 Target 对象

package policy

import (
	"context"
	"strings"

	orcmodel "neomaster/internal/model/orchestrator"
)

// ManualProvider 人工输入
type ManualProvider struct{}

func (m *ManualProvider) Name() string { return "manual" }

func (m *ManualProvider) Provide(ctx context.Context, config orcmodel.TargetSource, seedTargets []string) ([]Target, error) {
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
