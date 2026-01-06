// EvidenceArchiver 原始证据归档器接口
// 职责: 将 Agent 上报的原始 JSON/XML/截图 存入对象存储，作为审计依据
package ingestor

import (
	"context"
)

// EvidenceArchiver 原始证据归档器接口
type EvidenceArchiver interface {
	// Archive 归档证据数据
	// key: 存储路径/Key (如: task_id/result_id.json)
	// data: 原始数据
	Archive(ctx context.Context, key string, data []byte) error
}
