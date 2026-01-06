// ResultProcessor 结果处理器接口 (Consumer)
// 职责: 启动 Worker 消费队列，驱动 ETL 流程
package etl

import (
	"context"
)

// ResultProcessor 结果处理器接口
type ResultProcessor interface {
	// Start 启动处理循环
	Start(ctx context.Context)
	// Stop 停止处理
	Stop()
}
