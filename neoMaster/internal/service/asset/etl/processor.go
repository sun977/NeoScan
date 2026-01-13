// ResultProcessor 结果处理器接口 (Consumer)
// 职责: 启动 Worker 消费队列，驱动 ETL 流程
package etl

import (
	"context"
	"fmt"
	"sync"
	"time"

	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	assetRepo "neomaster/internal/repo/mysql/asset"
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
	queue     ingestor.ResultQueue         // 结果队列
	merger    AssetMerger                  // 资产合并器
	errorRepo assetRepo.ETLErrorRepository // 错误仓库
	wg        sync.WaitGroup               // 等待组
	ctx       context.Context              // 上下文
	cancel    context.CancelFunc           // 取消函数
	workerNum int                          // Worker 数量
}

// NewResultProcessor 创建结果处理器
func NewResultProcessor(queue ingestor.ResultQueue, merger AssetMerger, errorRepo assetRepo.ETLErrorRepository, workerNum int) ResultProcessor {
	if workerNum <= 0 {
		workerNum = 5 // 默认 5 个 Worker
	}
	return &resultProcessor{
		queue:     queue,
		merger:    merger,
		errorRepo: errorRepo,
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
			bundles, err := MapToAssetBundles(result)
			if err != nil {
				logger.LogError(err, "", 0, "", "etl.processor.worker", "", map[string]interface{}{
					"msg":         "Failed to map result",
					"task_id":     result.TaskID,
					"result_type": result.ResultType,
				})
				p.logEtlError(p.ctx, result, err, "mapper")
				continue
			}

			if len(bundles) == 0 {
				continue
			}

			// 3. 调用 Merger 进行合并
			for _, bundle := range bundles {
				if err := p.merger.Merge(p.ctx, bundle); err != nil {
					logger.LogError(err, "", 0, "", "etl.processor.worker", "", map[string]interface{}{
						"msg":         "Failed to merge asset bundle",
						"task_id":     result.TaskID,
						"result_type": result.ResultType,
						"host_ip":     bundle.Host.IP,
					})
					p.logEtlError(p.ctx, result, err, "merger")
				}
			}
			logger.LogInfo("Processed result successfully", "", 0, "", "etl.processor.worker", "", map[string]interface{}{
				"task_id":     result.TaskID,
				"result_type": result.ResultType,
				"bundles":     len(bundles),
			})
		}
	}
}

// logEtlError 记录 ETL 错误到数据库
func (p *resultProcessor) logEtlError(ctx context.Context, result *orcModel.StageResult, err error, stage string) {
	if p.errorRepo == nil {
		logger.LogWarn("ETLErrorRepository is nil, cannot log error to DB", "", 0, "", "etl.processor.logEtlError", "", nil)
		return
	}

	etlError := &assetModel.AssetETLError{
		ProjectID:  result.ProjectID,
		TaskID:     result.TaskID,
		ResultType: result.ResultType,
		RawData:    result.Attributes, // 暂存 Attributes, 如果需要完整 JSON 可以序列化整个 result
		ErrorMsg:   fmt.Sprintf("%v", err),
		ErrorStage: stage,
		Status:     "new",
	}

	if dbErr := p.errorRepo.Create(ctx, etlError); dbErr != nil {
		logger.LogError(dbErr, "", 0, "", "etl.processor.logEtlError", "", map[string]interface{}{
			"msg":            "Failed to log ETL error to DB",
			"original_error": err.Error(),
		})
	}
}
