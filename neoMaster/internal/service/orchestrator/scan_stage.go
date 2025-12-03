package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
)

// ScanStageService 扫描阶段服务
// 负责处理扫描阶段的业务逻辑
type ScanStageService struct {
	repo *orcrepo.ScanStageRepository
}

// NewScanStageService 创建 ScanStageService 实例
func NewScanStageService(repo *orcrepo.ScanStageRepository) *ScanStageService {
	return &ScanStageService{repo: repo}
}

// CreateStage 创建扫描阶段
func (s *ScanStageService) CreateStage(ctx context.Context, stage *orcmodel.ScanStage) error {
	if stage == nil {
		return errors.New("stage data cannot be nil")
	}

	err := s.repo.CreateStage(ctx, stage)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "create_stage", "SERVICE", map[string]interface{}{
			"operation":   "create_stage",
			"workflow_id": stage.WorkflowID,
		})
		return err
	}
	return nil
}

// GetStage 获取阶段详情
func (s *ScanStageService) GetStage(ctx context.Context, id uint64) (*orcmodel.ScanStage, error) {
	stage, err := s.repo.GetStageByID(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_stage", "SERVICE", map[string]interface{}{
			"operation": "get_stage",
			"id":        id,
		})
		return nil, err
	}
	if stage == nil {
		return nil, errors.New("stage not found")
	}
	return stage, nil
}

// UpdateStage 更新扫描阶段
func (s *ScanStageService) UpdateStage(ctx context.Context, stage *orcmodel.ScanStage) error {
	if stage == nil {
		return errors.New("stage data cannot be nil")
	}
	existing, err := s.repo.GetStageByID(ctx, stage.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("stage not found")
	}

	err = s.repo.UpdateStage(ctx, stage)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "update_stage", "SERVICE", map[string]interface{}{
			"operation": "update_stage",
			"id":        stage.ID,
		})
		return err
	}
	return nil
}

// DeleteStage 删除扫描阶段
func (s *ScanStageService) DeleteStage(ctx context.Context, id uint64) error {
	existing, err := s.repo.GetStageByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("stage not found")
	}

	err = s.repo.DeleteStage(ctx, id)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_stage", "SERVICE", map[string]interface{}{
			"operation": "delete_stage",
			"id":        id,
		})
		return err
	}
	return nil
}

// ListStagesByWorkflowID 获取工作流的所有阶段
func (s *ScanStageService) ListStagesByWorkflowID(ctx context.Context, workflowID uint64) ([]*orcmodel.ScanStage, error) {
	stages, err := s.repo.ListStagesByWorkflowID(ctx, workflowID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "list_stages_by_workflow_id", "SERVICE", map[string]interface{}{
			"operation":   "list_stages_by_workflow_id",
			"workflow_id": workflowID,
		})
		return nil, err
	}
	return stages, nil
}
