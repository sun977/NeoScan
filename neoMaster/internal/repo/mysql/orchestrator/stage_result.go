package orchestrator

import (
	"context"
	"errors"
	"gorm.io/gorm"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	"time"
)

// StageResultRepository 阶段结果仓库
type StageResultRepository struct {
	db *gorm.DB
}

// NewStageResultRepository 创建 StageResultRepository 实例
func NewStageResultRepository(db *gorm.DB) *StageResultRepository {
	return &StageResultRepository{db: db}
}

// CreateResult 创建扫描结果
func (r *StageResultRepository) CreateResult(ctx context.Context, result *orcmodel.StageResult) error {
	if result == nil {
		return errors.New("result is nil")
	}
	err := r.db.WithContext(ctx).Create(result).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_stage_result", "REPO", map[string]interface{}{
			"operation":   "create_stage_result",
			"workflow_id": result.WorkflowID,
			"stage_id":    result.StageID,
		})
		return err
	}
	return nil
}

// GetResultByID 根据ID获取结果
func (r *StageResultRepository) GetResultByID(ctx context.Context, id uint64) (*orcmodel.StageResult, error) {
	var result orcmodel.StageResult
	err := r.db.WithContext(ctx).First(&result, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_stage_result_by_id", "REPO", map[string]interface{}{
			"operation": "get_stage_result_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &result, nil
}

// ListResults 获取结果列表 (筛选)
func (r *StageResultRepository) ListResults(ctx context.Context, page, pageSize int, workflowID, stageID uint64, resultType string) ([]*orcmodel.StageResult, int64, error) {
	var results []*orcmodel.StageResult
	var total int64

	query := r.db.WithContext(ctx).Model(&orcmodel.StageResult{})

	if workflowID != 0 {
		query = query.Where("workflow_id = ?", workflowID)
	}
	if stageID != 0 {
		query = query.Where("stage_id = ?", stageID)
	}
	if resultType != "" {
		query = query.Where("result_type = ?", resultType)
	}

	err := query.Count(&total).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_stage_results_count", "REPO", map[string]interface{}{
			"operation": "list_stage_results_count",
		})
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("id desc").Find(&results).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_stage_results_find", "REPO", map[string]interface{}{
			"operation": "list_stage_results_find",
		})
		return nil, 0, err
	}

	return results, total, nil
}

// DeleteOldResults 删除旧结果 (清理任务)
func (r *StageResultRepository) DeleteOldResults(ctx context.Context, beforeTime time.Time) error {
	err := r.db.WithContext(ctx).Where("produced_at < ?", beforeTime).Delete(&orcmodel.StageResult{}).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_old_stage_results", "REPO", map[string]interface{}{
			"operation":   "delete_old_stage_results",
			"before_time": beforeTime,
		})
		return err
	}
	return nil
}
