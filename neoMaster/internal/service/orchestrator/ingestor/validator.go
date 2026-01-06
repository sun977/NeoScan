// ResultValidator 结果校验器接口
// 职责: 验证 StageResult 的格式合法性、签名正确性
package ingestor

import (
	"context"
	orcModel "neomaster/internal/model/orchestrator"
)

// ResultValidator 结果校验器接口
type ResultValidator interface {
	// Validate 校验结果是否合法
	// 检查 TaskID 是否存在、AgentID 是否匹配、签名是否正确等
	Validate(ctx context.Context, result *orcModel.StageResult) error
}
