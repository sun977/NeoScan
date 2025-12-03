package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
	"time"
)

// StageResultService 阶段结果服务
type StageResultService struct {
	repo *orcrepo.StageResultRepository
}

// NewStageResultService 创建 StageResultService 实例
func NewStageResultService(repo *orcrepo.StageResultRepository) *StageResultService {
	return &StageResultService{repo: repo}
}

// CreateResult 记录扫描结果
func (s *StageResultService) CreateResult(ctx context.Context, result *orcmodel.StageResult) error {
	if result == nil {
		return errors.New("result data cannot be nil")
	}
	if result.ProducedAt.IsZero() {
		result.ProducedAt = time.Now()
	}
	
	err := s.repo.CreateResult(ctx, result)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_stage_result", "SERVICE", map[string]interface{}{
			"operation":   "create_stage_result",
			"workflow_id": result.WorkflowID,
		})
		return err
	}
	return nil
}

// GetResult 获取结果详情
func (s *StageResultService) GetResult(ctx context.Context, id uint64) (*orcmodel.StageResult, error) {
	result, err := s.repo.GetResultByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_stage_result", "SERVICE", map[string]interface{}{
			"operation": "get_stage_result",
			"id":        id,
		})
		return nil, err
	}
	if result == nil {
		return nil, errors.New("result not found")
	}
	return result, nil
}

// ListResults 获取结果列表
func (s *StageResultService) ListResults(ctx context.Context, page, pageSize int, workflowID, stageID uint64, resultType string) ([]*orcmodel.StageResult, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	results, total, err := s.repo.ListResults(ctx, page, pageSize, workflowID, stageID, resultType)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_stage_results", "SERVICE", map[string]interface{}{
			"operation": "list_stage_results",
		})
		return nil, 0, err
	}
	return results, total, nil
}

// CleanupOldResults 清理旧数据
func (s *StageResultService) CleanupOldResults(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		return errors.New("retention days must be positive")
	}
	beforeTime := time.Now().AddDate(0, 0, -retentionDays)
	err := s.repo.DeleteOldResults(ctx, beforeTime)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "cleanup_old_results", "SERVICE", map[string]interface{}{
			"operation":      "cleanup_old_results",
			"retention_days": retentionDays,
		})
		return err
	}
	return nil
}
