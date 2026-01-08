package test_20260108

import (
	"context"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	orcModel "neomaster/internal/model/orchestrator"
	tagModel "neomaster/internal/model/tag_system"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	tagRepo "neomaster/internal/repo/mysql/tag_system"
	orcService "neomaster/internal/service/orchestrator"
	tagService "neomaster/internal/service/tag_system"
)

func setupScanStageTagFilterDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(
		&orcModel.ScanStage{},
		&tagModel.SysTag{},
		&tagModel.SysEntityTag{},
		&tagModel.SysMatchRule{},
	); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestScanStage_ListStagesByWorkflowIDWithTag(t *testing.T) {
	db := setupScanStageTagFilterDB(t)

	workflowID := uint64(100)
	stageA := &orcModel.ScanStage{WorkflowID: workflowID, StageName: "stage_a"}
	stageB := &orcModel.ScanStage{WorkflowID: workflowID, StageName: "stage_b"}
	if err := db.Create(stageA).Error; err != nil {
		t.Fatalf("failed to create stageA: %v", err)
	}
	if err := db.Create(stageB).Error; err != nil {
		t.Fatalf("failed to create stageB: %v", err)
	}

	tag := &tagModel.SysTag{Name: "for_stage_b", Path: "/", Level: 0}
	if err := db.Create(tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	entityTag := &tagModel.SysEntityTag{
		EntityType: "scan_stage",
		EntityID:   strconv.FormatUint(uint64(stageB.ID), 10),
		TagID:      tag.ID,
		Source:     "manual",
		RuleID:     0,
	}
	if err := db.Create(entityTag).Error; err != nil {
		t.Fatalf("failed to create entityTag: %v", err)
	}

	scanStageRepo := orcRepo.NewScanStageRepository(db)
	tagRepository := tagRepo.NewTagRepository(db)
	tagSvc := tagService.NewTagService(tagRepository, db)
	svc := orcService.NewScanStageService(scanStageRepo, tagSvc)

	ctx := context.Background()

	allStages, err := svc.ListStagesByWorkflowID(ctx, workflowID)
	if err != nil {
		t.Fatalf("ListStagesByWorkflowID failed: %v", err)
	}
	if len(allStages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(allStages))
	}

	filteredStages, err := svc.ListStagesByWorkflowIDWithTag(ctx, workflowID, tag.ID)
	if err != nil {
		t.Fatalf("ListStagesByWorkflowIDWithTag failed: %v", err)
	}
	if len(filteredStages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(filteredStages))
	}
	if filteredStages[0].ID != stageB.ID {
		t.Fatalf("expected stageB (id=%d), got id=%d", stageB.ID, filteredStages[0].ID)
	}
}

func TestScanStage_ListStagesByWorkflowIDWithTag_EmptyResult(t *testing.T) {
	db := setupScanStageTagFilterDB(t)

	workflowID := uint64(200)
	stage := &orcModel.ScanStage{WorkflowID: workflowID, StageName: "stage_only"}
	if err := db.Create(stage).Error; err != nil {
		t.Fatalf("failed to create stage: %v", err)
	}

	tag := &tagModel.SysTag{Name: "unused_tag", Path: "/", Level: 0}
	if err := db.Create(tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	scanStageRepo := orcRepo.NewScanStageRepository(db)
	tagRepository := tagRepo.NewTagRepository(db)
	tagSvc := tagService.NewTagService(tagRepository, db)
	svc := orcService.NewScanStageService(scanStageRepo, tagSvc)

	filteredStages, err := svc.ListStagesByWorkflowIDWithTag(context.Background(), workflowID, tag.ID)
	if err != nil {
		t.Fatalf("ListStagesByWorkflowIDWithTag failed: %v", err)
	}
	if filteredStages == nil {
		t.Fatalf("expected empty slice, got nil")
	}
	if len(filteredStages) != 0 {
		t.Fatalf("expected 0 stages, got %d", len(filteredStages))
	}
}
