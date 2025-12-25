// ProjectTargetProvider 项目种子目标数据来源
// 实现 TargetProvider 接口，从项目种子目标解析目标
// 功能：
// 1. 从项目配置中提取种子目标
// 2. 将种子目标作为 Target.Value 保存
// 3. 支持自定义目标类型

package policy

import (
	"context"

	orcmodel "neomaster/internal/model/orchestrator"
)

// ProjectTargetProvider 项目种子目标
type ProjectTargetProvider struct{}

func (p *ProjectTargetProvider) Name() string { return "project_target" }

func (p *ProjectTargetProvider) Provide(ctx context.Context, config orcmodel.TargetSource, seedTargets []string) ([]Target, error) {
	targets := make([]Target, 0, len(seedTargets))
	for _, t := range seedTargets {
		targets = append(targets, Target{
			Type:   config.TargetType, // 假设种子目标类型与配置一致，或者需要自动检测
			Value:  t,
			Source: "project_target",
			Meta:   nil,
		})
	}
	return targets, nil
}

func (p *ProjectTargetProvider) HealthCheck(ctx context.Context) error {
	return nil
}
