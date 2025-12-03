package orchestrator

import (
	"context"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
)

// WorkflowStatsService 工作流统计服务
type WorkflowStatsService struct {
	repo *orcrepo.WorkflowStatsRepository
}

// NewWorkflowStatsService 创建 WorkflowStatsService 实例
func NewWorkflowStatsService(repo *orcrepo.WorkflowStatsRepository) *WorkflowStatsService {
	return &WorkflowStatsService{repo: repo}
}

// GetStats 获取统计信息
func (s *WorkflowStatsService) GetStats(ctx context.Context, workflowID uint64) (*orcmodel.WorkflowStats, error) {
	stats, err := s.repo.GetStatsByWorkflowID(ctx, workflowID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_workflow_stats", "SERVICE", map[string]interface{}{
			"operation":   "get_workflow_stats",
			"workflow_id": workflowID,
		})
		return nil, err
	}
	if stats == nil {
		// 如果不存在，返回一个空的统计对象而不是报错，或者自动初始化
		return &orcmodel.WorkflowStats{
			WorkflowID: workflowID,
		}, nil
	}
	return stats, nil
}

// RecordExecution 记录一次执行结果
func (s *WorkflowStatsService) RecordExecution(ctx context.Context, workflowID uint64, isSuccess bool, durationMs int, execID string, status string) error {
	// 确保统计记录存在
	err := s.repo.UpsertStats(ctx, &orcmodel.WorkflowStats{WorkflowID: workflowID})
	if err != nil {
		return err
	}

	// 更新计数
	err = s.repo.UpdateExecutionStats(ctx, workflowID, isSuccess, durationMs, execID, status)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "record_execution", "SERVICE", map[string]interface{}{
			"operation":   "record_execution",
			"workflow_id": workflowID,
			"status":      status,
		})
		return err
	}
	return nil
}
