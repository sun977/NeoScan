// ResultProcessor 结果处理器接口 (Consumer)
// 职责: 启动 Worker 消费队列，驱动 ETL 流程
package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"neomaster/internal/pkg/logger"
	"neomaster/internal/service/fingerprint"
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
	queue     ingestor.ResultQueue // 结果队列
	merger    AssetMerger          // 资产合并器
	fpService fingerprint.Service  // 指纹识别服务
	wg        sync.WaitGroup       // 等待组
	ctx       context.Context      // 上下文
	cancel    context.CancelFunc   // 取消函数
	workerNum int                  // Worker 数量
}

// NewResultProcessor 创建结果处理器
func NewResultProcessor(queue ingestor.ResultQueue, merger AssetMerger, fpService fingerprint.Service, workerNum int) ResultProcessor {
	if workerNum <= 0 {
		workerNum = 5 // 默认 5 个 Worker
	}
	return &resultProcessor{
		queue:     queue,
		merger:    merger,
		fpService: fpService,
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

			// 2.5 调用指纹识别
			if p.fpService != nil {
				// A. 服务指纹识别 (Service Fingerprint)
				// 策略: 优先使用 Agent 返回的 CPE；仅当 Agent 未识别出 CPE 时，Master 尝试基于 Banner 进行补充识别
				if len(bundle.Services) > 0 {
					for _, svc := range bundle.Services {
						// 简单的 heuristic: 如果没有 CPE，尝试识别
						if svc.CPE == "" {
							input := &fingerprint.Input{
								Target:   bundle.Host.IP, // 假设 Host 总是存在
								Port:     svc.Port,
								Protocol: svc.Proto,
								Banner:   svc.Version, // 假设 Version 字段暂时存放 Banner (取决于 Mapper 实现)
							}
							// 如果 version 为空，使用 name
							if input.Banner == "" {
								input.Banner = svc.Name
							}

							fpResult, err := p.fpService.Identify(p.ctx, input)
							if err == nil && fpResult != nil && fpResult.Best != nil {
								svc.CPE = fpResult.Best.CPE
								svc.Version = fpResult.Best.Version

								// 更新指纹 JSON 信息 (Product, Vendor, Type)
								fpMap := make(map[string]interface{})
								if svc.Fingerprint != "" && svc.Fingerprint != "{}" {
									_ = json.Unmarshal([]byte(svc.Fingerprint), &fpMap)
								}
								fpMap["product"] = fpResult.Best.Product
								fpMap["vendor"] = fpResult.Best.Vendor
								fpMap["type"] = fpResult.Best.Type
								fpJSON, _ := json.Marshal(fpMap)
								svc.Fingerprint = string(fpJSON)

								logger.LogInfo("Service Fingerprint identified (Master)", "", 0, "", "etl.processor.worker", "", map[string]interface{}{
									"port": svc.Port,
									"cpe":  svc.CPE,
								})
							}
						}
					}
				}

				// B. Web 指纹识别 (Web Fingerprint)
				// 策略: Master 端拥有更丰富的 CMS 规则库，尝试对 Web 资产进行深度识别并合并结果
				if len(bundle.Webs) > 0 {
					for _, web := range bundle.Webs {
						// 构造 Web 输入 (需要从 BasicInfo 或 WebDetails 中提取)
						// 注意: 这里假设 BasicInfo 包含 Headers 等信息
						input := &fingerprint.Input{
							Target: web.URL,
							// Body, Headers 需要从 web 对象中解析
						}

						// 尝试解析 BasicInfo
						if web.BasicInfo != "" {
							var info map[string]interface{}
							if err := json.Unmarshal([]byte(web.BasicInfo), &info); err == nil {
								if headers, ok := info["headers"].(map[string]interface{}); ok {
									input.Headers = make(map[string]string)
									for k, v := range headers {
										input.Headers[k] = fmt.Sprintf("%v", v)
									}
								}
								if body, ok := info["body"].(string); ok {
									input.Body = body
								}
							}
						}

						// 只有当有足够信息时才识别
						if len(input.Headers) > 0 || input.Body != "" {
							fpResult, err := p.fpService.Identify(p.ctx, input)
							if err == nil && fpResult != nil && len(fpResult.Matches) > 0 {
								// 合并 TechStack
								currentStack := make(map[string]interface{})
								if web.TechStack != "" {
									_ = json.Unmarshal([]byte(web.TechStack), &currentStack)
								}

								for _, match := range fpResult.Matches {
									// 将识别到的组件添加到 TechStack
									currentStack[match.Product] = map[string]interface{}{
										"version": match.Version,
										"type":    match.Type,
										"source":  "master_fingerprint",
									}
								}

								newStack, _ := json.Marshal(currentStack)
								web.TechStack = string(newStack)

								logger.LogInfo("Web Fingerprint identified (Master)", "", 0, "", "etl.processor.worker", "", map[string]interface{}{
									"url":   web.URL,
									"count": len(fpResult.Matches),
								})
							}
						}
					}
				}
			}

			// 3. 调用 Merger 进行入库
			if err := p.merger.Merge(p.ctx, bundle); err != nil {
				logger.LogError(err, "Failed to merge asset bundle", 0, "", "etl.processor.worker", "", map[string]interface{}{
					"task_id": result.TaskID,
				})
				// TODO: 错误处理策略 (重试/丢弃)
			}
			logger.LogInfo("Processed result successfully", "", 0, "", "etl.processor.worker", "", map[string]interface{}{
				"task_id":     result.TaskID,
				"result_type": result.ResultType,
				"has_host":    bundle.Host != nil,
				"services":    len(bundle.Services),
			})
		}
	}
}
