// ResultProcessor 结果处理器接口 (Consumer)
// 职责: 启动 Worker 消费队列，驱动 ETL 流程
package etl

import (
	"context"
	"sync"
	"time"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/orchestrator/ingestor"
)

// ResultProcessor 结果处理器接口
type ResultProcessor interface {
	// Start 启动处理循环
	Start(ctx context.Context)
	// Stop 停止处理
	Stop()
}

// resultProcessor 默认实现
type resultProcessor struct {
	queue ingestor.ResultQueue
	// merger  Merger // 依赖 Merger 组件 (后续注入)
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	workerNum int
}

// NewResultProcessor 创建结果处理器
func NewResultProcessor(queue ingestor.ResultQueue, workerNum int) ResultProcessor {
	if workerNum <= 0 {
		workerNum = 5 // 默认 5 个 Worker
	}
	return &resultProcessor{
		queue:     queue,
		workerNum: workerNum,
	}
}

// Start 启动处理循环
func (p *resultProcessor) Start(ctx context.Context) {
	p.ctx, p.cancel = context.WithCancel(ctx)

	logger.LogInfo("Starting ETL ResultProcessor", "", 0, "", "etl.processor.Start", "", map[string]interface{}{
		"workers": p.workerNum,
	})

	for i := 0; i < p.workerNum; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop 停止处理
func (p *resultProcessor) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	logger.LogInfo("ETL ResultProcessor stopped", "", 0, "", "etl.processor.Stop", "", nil)
}

// worker 工作协程
func (p *resultProcessor) worker(id int) {
	defer p.wg.Done()
	logger.LogInfo("Worker started", "", 0, "", "etl.processor.worker", "", map[string]interface{}{"worker_id": id})

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			// 1. 从队列获取结果 (阻塞模式)
			result, err := p.queue.Pop(p.ctx)
			if err != nil {
				// 如果是 Context 取消导致的错误，直接退出
				if p.ctx.Err() != nil {
					return
				}
				// 队列为空或其他错误，稍作休眠避免空转 (取决于 Queue 实现，MemoryQueue 是阻塞的)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if result == nil {
				continue
			}

			// 2. 调用 Mapper 进行映射
			bundle, err := MapToAssetBundle(result)
			if err != nil {
				logger.LogError(err, "Failed to map result", 0, "", "etl.processor.worker", "", map[string]interface{}{
					"task_id":     result.TaskID,
					"result_type": result.ResultType,
				})
				// TODO: 记录到死信队列或错误日志表
				continue
			}

			// 3. 调用 Merger 进行入库 (占位)
			// if err := p.merger.Merge(p.ctx, bundle); err != nil { ... }
			_ = bundle // 避免未使用变量报错
			logger.LogInfo("Processed result successfully", "", 0, "", "etl.processor.worker", "", map[string]interface{}{
				"task_id":     result.TaskID,
				"result_type": result.ResultType,
				"has_host":    bundle.Host != nil,
				"services":    len(bundle.Services),
			})
		}
	}
}
