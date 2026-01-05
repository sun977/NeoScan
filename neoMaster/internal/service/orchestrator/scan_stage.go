package orchestrator

import (
	"context"
	"errors"
	"strconv"

	orcmodel "neomaster/internal/model/orchestrator"
	tagmodel "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	orcrepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/tag_system"
)

// ScanStageService 扫描阶段服务
// 负责处理扫描阶段的业务逻辑
type ScanStageService struct {
	repo       *orcrepo.ScanStageRepository
	tagService tag_system.TagService
}

// NewScanStageService 创建 ScanStageService 实例
func NewScanStageService(repo *orcrepo.ScanStageRepository, tagService tag_system.TagService) *ScanStageService {
	return &ScanStageService{
		repo:       repo,
		tagService: tagService,
	}
}

// CreateStage 创建扫描阶段
func (s *ScanStageService) CreateStage(ctx context.Context, stage *orcmodel.ScanStage) error {
	if stage == nil {
		return errors.New("stage data cannot be nil")
	}

	// 验证 Predecessors 不包含自身 (仅 Update 有效，但作为防御性编程放在这里)
	if stage.ID > 0 {
		for _, pid := range stage.Predecessors {
			if pid == stage.ID {
				return errors.New("stage cannot depend on itself")
			}
		}
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

	// 验证 Predecessors 不包含自身
	for _, pid := range stage.Predecessors {
		if pid == stage.ID {
			return errors.New("stage cannot depend on itself")
		}
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

// AddTagToStage 为扫描阶段添加标签
func (s *ScanStageService) AddTagToStage(ctx context.Context, stageID uint64, tagID uint64) error {
	// 1. 检查阶段是否存在
	stage, err := s.repo.GetStageByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return errors.New("stage not found")
	}

	// 2. 调用 TagService 添加标签
	entityIDStr := strconv.FormatUint(stageID, 10)
	err = s.tagService.AddEntityTag(ctx, "scan_stage", entityIDStr, tagID, "manual", 0)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "add_tag_to_stage", "SERVICE", map[string]interface{}{
			"operation": "add_tag_to_stage",
			"stage_id":  stageID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

// RemoveTagFromStage 从扫描阶段移除标签
func (s *ScanStageService) RemoveTagFromStage(ctx context.Context, stageID uint64, tagID uint64) error {
	// 1. 检查阶段是否存在
	stage, err := s.repo.GetStageByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return errors.New("stage not found")
	}

	// 2. 调用 TagService 移除标签
	entityIDStr := strconv.FormatUint(stageID, 10)
	err = s.tagService.RemoveEntityTag(ctx, "scan_stage", entityIDStr, tagID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "remove_tag_from_stage", "SERVICE", map[string]interface{}{
			"operation": "remove_tag_from_stage",
			"stage_id":  stageID,
			"tag_id":    tagID,
		})
		return err
	}
	return nil
}

// GetStageTags 获取扫描阶段的所有标签
func (s *ScanStageService) GetStageTags(ctx context.Context, stageID uint64) ([]tagmodel.SysEntityTag, error) {
	// 1. 检查阶段是否存在
	stage, err := s.repo.GetStageByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, errors.New("stage not found")
	}

	// 2. 调用 TagService 获取标签
	entityIDStr := strconv.FormatUint(stageID, 10)
	tags, err := s.tagService.GetEntityTags(ctx, "scan_stage", entityIDStr)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "get_stage_tags", "SERVICE", map[string]interface{}{
			"operation": "get_stage_tags",
			"stage_id":  stageID,
		})
		return nil, err
	}
	return tags, nil
}
