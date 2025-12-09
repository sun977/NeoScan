package policy

import (
	"context"
	"fmt"
	"neomaster/internal/pkg/logger"
)

// PreviousStageProvider 上一阶段结果提供者 (占位符)
// 用于将上一个扫描阶段的输出作为当前阶段的输入
type PreviousStageProvider struct{}

func (p *PreviousStageProvider) Name() string { return "previous_stage" }

func (p *PreviousStageProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	// TODO: 实现获取上一阶段结果的逻辑
	// 1. 获取当前 Workflow 上下文
	// 2. 查找上一阶段的执行结果
	// 3. 提取目标数据
	logger.LogWarn("PreviousStage provider is not implemented yet", "", 0, "", "PreviousStageProvider.Provide", "", nil)
	return nil, fmt.Errorf("previous_stage provider not implemented")
}

func (p *PreviousStageProvider) HealthCheck(ctx context.Context) error {
	return nil
}
