// ResultIngestor 结果摄入服务接口
// 职责: 提供 SubmitResult 方法供 API 层调用
package ingestor

import (
	"context"
	"fmt"
	"time"

	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
)

// ResultIngestor 结果摄入服务接口
type ResultIngestor interface {
	// SubmitResult 提交扫描结果
	// 1. 校验数据
	// 2. 归档证据
	// 3. 推入队列
	SubmitResult(ctx context.Context, result *orcModel.StageResult) error
}

type resultIngestor struct {
	queue     ResultQueue
	validator ResultValidator
	archiver  EvidenceArchiver
}

// NewResultIngestor 创建结果摄入服务
func NewResultIngestor(queue ResultQueue, validator ResultValidator, archiver EvidenceArchiver) ResultIngestor {
	return &resultIngestor{
		queue:     queue,
		validator: validator,
		archiver:  archiver,
	}
}

// SubmitResult 提交扫描结果
func (s *resultIngestor) SubmitResult(ctx context.Context, result *orcModel.StageResult) error {
	loggerFields := map[string]interface{}{
		"task_id":  result.TaskID, // 假设 StageResult 有 TaskID 字段，如果没有需要确认模型
		"agent_id": result.AgentID,
	}

	// 1. 校验数据
	if err := s.validator.Validate(ctx, result); err != nil {
		logger.LogWarn("Result validation failed", "", 0, "", "ingestor.SubmitResult", "", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("validation failed: %w", err)
	}

	// 2. 归档证据 (异步或同步)
	// Evidence 字段通常包含大体积的原始数据
	if result.Evidence != "" {
		// 生成归档 Key: task_id/result_type/timestamp.json
		key := fmt.Sprintf("%s/%s/%d.json", result.TaskID, result.ResultType, time.Now().UnixNano())
		// 尝试归档，如果归档失败，记录日志但不阻断流程 (或者根据策略阻断)
		// 这里假设 Evidence 是 JSON 字符串，转为 byte
		if err := s.archiver.Archive(ctx, key, []byte(result.Evidence)); err != nil {
			logger.LogError(err, "Failed to archive evidence", 0, "", "ingestor.SubmitResult", "ARCHIVER", loggerFields)
			// return fmt.Errorf("archive failed: %w", err) // 可选：是否强一致性
		} else {
			// 归档成功后，可以选择清空 result.Evidence 以减轻队列和后续处理的压力
			// result.Evidence = key // 或者替换为存储路径
		}
	}

	// 3. 推入队列
	if err := s.queue.Push(ctx, result); err != nil {
		if err == ErrQueueFull {
			logger.LogWarn("Result queue full, dropping result", "", 0, "", "ingestor.SubmitResult", "", loggerFields)
			// TODO: 可以考虑降级策略，如写入本地文件或重试
			return fmt.Errorf("system busy, please retry later")
		}
		logger.LogError(err, "Failed to push result to queue", 0, "", "ingestor.SubmitResult", "QUEUE", loggerFields)
		return fmt.Errorf("internal error")
	}

	logger.LogInfo("Result ingested successfully", "", 0, "", "ingestor.SubmitResult", "", loggerFields)
	return nil
}
