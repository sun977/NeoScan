// ResultIngestor 结果摄入服务接口
// 职责: 提供 SubmitResult 方法供 API 层调用
package ingestor

import (
	"context"
	orcModel "neomaster/internal/model/orchestrator"
)

// ResultIngestor 结果摄入服务接口
type ResultIngestor interface {
	// SubmitResult 提交扫描结果
	// 1. 校验数据
	// 2. 归档证据
	// 3. 推入队列
	SubmitResult(ctx context.Context, result *orcModel.StageResult) error
}
