package test_20251212

import (
	"context"
	"testing"
	"time"

	orcModel "neomaster/internal/model/orchestrator"
	agentRepo "neomaster/internal/repo/mysql/agent"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/core/scheduler"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// SetupDAGEnv 初始化 DAG 测试环境 (使用 SQLite 内存数据库)
func SetupDAGEnv(t *testing.T) (*gorm.DB, scheduler.SchedulerService) {
	// 1. 初始化 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to sqlite: %v", err)
	}

	// 2. 自动迁移
	err = db.AutoMigrate(
		&orcModel.Project{},
		&orcModel.Workflow{},
		&orcModel.ScanStage{},
		&orcModel.AgentTask{},
		&orcModel.ProjectWorkflow{},
		&orcModel.StageResult{},
	)
	if err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	// 3. 初始化 Repos
	projectRepo := orcRepo.NewProjectRepository(db)
	workflowRepo := orcRepo.NewWorkflowRepository(db)
	stageRepo := orcRepo.NewScanStageRepository(db)
	taskRepo := orcRepo.NewTaskRepository(db)
	agentRepository := agentRepo.NewAgentRepository(db)

	// Mock TargetProvider (Simple implementation or Mock)
	// 这里我们需要一个简单的 TargetProvider，否则 GenerateTasks 会失败
	// targetProvider := policy.NewTargetProvider(db)

	// 4. 初始化 Scheduler
	svc := scheduler.NewSchedulerService(
		db,
		projectRepo,
		workflowRepo,
		stageRepo,
		taskRepo,
		agentRepository,
		10*time.Second,
	)

	return db, svc
}

// TestDAGScheduling 测试 DAG 调度逻辑
func TestDAGScheduling(t *testing.T) {
	db, svc := SetupDAGEnv(t)
	ctx := context.Background()

	// 1. 创建 Workflow
	workflow := &orcModel.Workflow{
		Name:     "dag_workflow",
		ExecMode: "dag",
	}
	db.Create(workflow)

	// 2. 创建 Stages (Diamond Shape: A -> B, C -> D)
	// Stage A
	stageA := &orcModel.ScanStage{
		WorkflowID: uint64(workflow.ID),
		StageName:  "StageA",
		StageType:  "test",
	}
	db.Create(stageA)

	// Stage B (Depends on A)
	stageB := &orcModel.ScanStage{
		WorkflowID:   uint64(workflow.ID),
		StageName:    "StageB",
		Predecessors: []uint64{uint64(stageA.ID)},
	}
	db.Create(stageB)

	// Stage C (Depends on A)
	stageC := &orcModel.ScanStage{
		WorkflowID:   uint64(workflow.ID),
		StageName:    "StageC",
		Predecessors: []uint64{uint64(stageA.ID)},
	}
	db.Create(stageC)

	// Stage D (Depends on B and C)
	stageD := &orcModel.ScanStage{
		WorkflowID:   uint64(workflow.ID),
		StageName:    "StageD",
		Predecessors: []uint64{uint64(stageB.ID), uint64(stageC.ID)},
	}
	db.Create(stageD)

	// 3. 创建 Project
	project := &orcModel.Project{
		Name:        "dag_project",
		Status:      "running",
		TargetScope: "127.0.0.1", // Seed target
	}
	db.Create(project)

	// Link Project to Workflow
	db.Create(&orcModel.ProjectWorkflow{
		ProjectID:  uint64(project.ID),
		WorkflowID: uint64(workflow.ID),
	})

	// -------------------------------------------------------
	// Round 1: Should schedule Stage A
	// -------------------------------------------------------
	svc.ProcessProject(ctx, project)

	var tasks []orcModel.AgentTask
	db.Where("project_id = ?", project.ID).Find(&tasks)
	assert.Equal(t, 1, len(tasks), "Should have 1 task (Stage A)")
	if len(tasks) > 0 {
		assert.Equal(t, uint64(stageA.ID), tasks[0].StageID)
		// Mark A as finished
		tasks[0].Status = "finished"
		db.Save(&tasks[0])
	}

	// -------------------------------------------------------
	// Round 2: Should schedule Stage B and C
	// -------------------------------------------------------
	svc.ProcessProject(ctx, project)

	db.Where("project_id = ?", project.ID).Find(&tasks)
	// Should have 3 tasks total (A, B, C)
	assert.Equal(t, 3, len(tasks))

	// Verify B and C are present
	hasB := false
	hasC := false
	for _, task := range tasks {
		if task.StageID == uint64(stageB.ID) {
			hasB = true
		}
		if task.StageID == uint64(stageC.ID) {
			hasC = true
		}
	}
	assert.True(t, hasB, "Stage B should be scheduled")
	assert.True(t, hasC, "Stage C should be scheduled")

	// Mark B as finished, C as running
	db.Model(&orcModel.AgentTask{}).Where("stage_id = ?", stageB.ID).Update("status", "finished")

	// -------------------------------------------------------
	// Round 3: Should NOT schedule D yet (C is not finished)
	// -------------------------------------------------------
	svc.ProcessProject(ctx, project)

	db.Where("project_id = ?", project.ID).Find(&tasks)
	assert.Equal(t, 3, len(tasks), "Should still have 3 tasks")

	// Mark C as finished
	db.Model(&orcModel.AgentTask{}).Where("stage_id = ?", stageC.ID).Update("status", "finished")

	// -------------------------------------------------------
	// Round 4: Should schedule D
	// -------------------------------------------------------
	svc.ProcessProject(ctx, project)

	db.Where("project_id = ?", project.ID).Find(&tasks)
	assert.Equal(t, 4, len(tasks), "Should have 4 tasks")
	if len(tasks) == 4 {
		// The last one should be D (or find by ID)
		foundD := false
		for _, task := range tasks {
			if task.StageID == uint64(stageD.ID) {
				foundD = true
			}
		}
		assert.True(t, foundD, "Stage D should be scheduled")
	}

	// Mark D as finished
	db.Model(&orcModel.AgentTask{}).Where("stage_id = ?", stageD.ID).Update("status", "finished")

	// -------------------------------------------------------
	// Round 5: Project should be finished
	// -------------------------------------------------------
	svc.ProcessProject(ctx, project)

	// Reload Project
	var updatedProject orcModel.Project
	db.First(&updatedProject, project.ID)
	assert.Equal(t, "finished", updatedProject.Status)
}
