// ResultQueue 结果缓冲队列接口
// 职责: 解耦 Agent 提交与 Master 处理速率，实现削峰填谷
package ingestor

import (
	"context"
	orcModel "neomaster/internal/model/orchestrator"
)

// ResultQueue 结果缓冲队列接口
type ResultQueue interface {
	// Push 推送结果到队列
	Push(ctx context.Context, result *orcModel.StageResult) error
	// Pop 从队列获取结果 (供 Processor 消费)
	Pop(ctx context.Context) (*orcModel.StageResult, error)
	// Len 获取队列长度 (用于监控)
	Len(ctx context.Context) (int64, error)
}
