package policy

import (
	"context"
	"fmt"
	"neomaster/internal/pkg/logger"

	orcmodel "neomaster/internal/model/orchestrator"
)

// ApiProvider API 来源提供者 (占位符)
// 用于从外部 API 获取扫描目标
type ApiProvider struct{}

func (p *ApiProvider) Name() string { return "api" }

func (p *ApiProvider) Provide(ctx context.Context, config orcmodel.TargetSource, seedTargets []string) ([]Target, error) {
	// TODO: 实现 API 调用逻辑
	// 1. 解析 AuthConfig 获取 API 凭证
	// 2. 发起 HTTP 请求
	// 3. 根据 ParserConfig 解析响应内容
	logger.LogWarn("API source provider is not implemented yet", "", 0, "", "ApiProvider.Provide", "", nil)
	return nil, fmt.Errorf("api provider not implemented")
}

func (p *ApiProvider) HealthCheck(ctx context.Context) error {
	// TODO: 检查 API 可用性
	return nil
}
