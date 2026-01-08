package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// ScanStageRepository 扫描阶段仓库
// 负责 ScanStage 的数据访问
type ScanStageRepository struct {
	db *gorm.DB
}

// NewScanStageRepository 创建 ScanStageRepository 实例
func NewScanStageRepository(db *gorm.DB) *ScanStageRepository {
	return &ScanStageRepository{db: db}
}

// -----------------------------------------------------------------------------
// ScanStage (扫描阶段) CRUD
// -----------------------------------------------------------------------------

// CreateStage 创建扫描阶段
func (r *ScanStageRepository) CreateStage(ctx context.Context, stage *orcmodel.ScanStage) error {
	if stage == nil {
		return errors.New("stage is nil")
	}
	err := r.db.WithContext(ctx).Create(stage).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "create_stage", "REPO", map[string]interface{}{
			"operation":   "create_stage",
			"workflow_id": stage.WorkflowID,
			"stage_name":  stage.StageName,
		})
		return err
	}
	return nil
}

// GetStagesByWorkflowID 获取工作流的所有阶段
func (r *ScanStageRepository) GetStagesByWorkflowID(ctx context.Context, workflowID uint64) ([]*orcmodel.ScanStage, error) {
	var stages []*orcmodel.ScanStage
	err := r.db.WithContext(ctx).
		Where("workflow_id = ? AND enabled = ?", workflowID, true).
		Order("id ASC").
		Find(&stages).Error

	if err != nil {
		logger.LogError(err, "", 0, "", "get_stages_by_workflow_id", "REPO", map[string]interface{}{
			"operation":   "get_stages_by_workflow_id",
			"workflow_id": workflowID,
		})
		return nil, err
	}
	return stages, nil
}

// GetStageByID 根据ID获取扫描阶段
func (r *ScanStageRepository) GetStageByID(ctx context.Context, id uint64) (*orcmodel.ScanStage, error) {
	var stage orcmodel.ScanStage
	err := r.db.WithContext(ctx).First(&stage, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_stage_by_id", "REPO", map[string]interface{}{
			"operation": "get_stage_by_id",
			"id":        id,
		})
		return nil, err
	}
	return &stage, nil
}

// UpdateStage 更新扫描阶段
func (r *ScanStageRepository) UpdateStage(ctx context.Context, stage *orcmodel.ScanStage) error {
	if stage == nil || stage.ID == 0 {
		return errors.New("invalid stage or id")
	}
	err := r.db.WithContext(ctx).Save(stage).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "update_stage", "REPO", map[string]interface{}{
			"operation": "update_stage",
			"id":        stage.ID,
		})
		return err
	}
	return nil
}

// DeleteStage 删除扫描阶段
func (r *ScanStageRepository) DeleteStage(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&orcmodel.ScanStage{}, id).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "delete_stage", "REPO", map[string]interface{}{
			"operation": "delete_stage",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListStagesByWorkflowID 获取指定工作流的所有阶段
func (r *ScanStageRepository) ListStagesByWorkflowID(ctx context.Context, workflowID uint64) ([]*orcmodel.ScanStage, error) {
	var stages []*orcmodel.ScanStage
	// 按 ID 升序排列
	err := r.db.WithContext(ctx).Where("workflow_id = ?", workflowID).Order("id asc").Find(&stages).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_stages_by_workflow_id", "REPO", map[string]interface{}{
			"operation":   "list_stages_by_workflow_id",
			"workflow_id": workflowID,
		})
		return nil, err
	}
	return stages, nil
}

// ListStagesByWorkflowIDAndStageIDs 获取指定工作流的阶段列表（按阶段 ID 白名单过滤）
// 说明：阶段 ID 列表通常来自标签系统筛选结果；这里保持仓库层只做数据过滤，不掺杂任何业务逻辑。
func (r *ScanStageRepository) ListStagesByWorkflowIDAndStageIDs(ctx context.Context, workflowID uint64, stageIDs []uint64) ([]*orcmodel.ScanStage, error) {
	if len(stageIDs) == 0 {
		return []*orcmodel.ScanStage{}, nil
	}

	var stages []*orcmodel.ScanStage
	err := r.db.WithContext(ctx).
		Where("workflow_id = ? AND id IN ?", workflowID, stageIDs).
		Order("id asc").
		Find(&stages).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "list_stages_by_workflow_id_and_stage_ids", "REPO", map[string]interface{}{
			"operation":   "list_stages_by_workflow_id_and_stage_ids",
			"workflow_id": workflowID,
			"stage_ids":   stageIDs,
		})
		return nil, err
	}
	return stages, nil
}
