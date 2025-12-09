// StubProvider 桩实现
// 实现 TargetProvider 接口，用于占位或测试
// 功能：
// 1. 记录警告日志，提示该数据来源暂不支持
// 2. 返回空目标列表和 nil 错误
package policy

import (
	"context"
	"fmt"
	"neomaster/internal/pkg/logger"
)

// StubProvider 桩实现
type StubProvider struct {
	name string
}

func (s *StubProvider) Name() string { return s.name }

func (s *StubProvider) Provide(ctx context.Context, config TargetSourceConfig, seedTargets []string) ([]Target, error) {
	logger.LogWarn(fmt.Sprintf("%s source not supported yet", s.name), "", 0, "", "Provide", "", nil)
	return nil, nil
}

func (s *StubProvider) HealthCheck(ctx context.Context) error {
	return nil
}
