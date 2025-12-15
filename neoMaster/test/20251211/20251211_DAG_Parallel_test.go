package test

import (
	"context"
	"fmt"
	"neomaster/internal/config"
	agentModel "neomaster/internal/model/orchestrator"
	agentRepo "neomaster/internal/repo/mysql/agent"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/core/scheduler"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestDAGParallelExecution 验证 DAG 并行调度能力
// 场景:
// Branch 1: Stage A -> Stage B
// Branch 2: Stage C (Long Running)
// 预期: 当 A 完成时，即使 C 还在运行，B 也应该立即启动。
func TestDAGParallelExecution(t *testing.T) {
	// 1. Setup In-Memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// Migrate schemas
	db.AutoMigrate(
		&agentModel.Project{},
		&agentModel.Workflow{},
		&agentModel.ProjectWorkflow{}, // Add association table
		&agentModel.ScanStage{},
		&agentModel.AgentTask{},
		&agentModel.StageResult{},
	)

	// 2. Initialize Repos & Services
	projectRepo := orcRepo.NewProjectRepository(db)
	workflowRepo := orcRepo.NewWorkflowRepository(db)
	stageRepo := orcRepo.NewScanStageRepository(db)
	taskRepo := orcRepo.NewTaskRepository(db)
	agentRepoInstance := agentRepo.NewAgentRepository(db)

	// Services
	cfg := &config.Config{
		App: config.AppConfig{
			Master: config.MasterConfig{
				Task: config.TaskConfig{
					ChunkSize:  50,
					Timeout:    3600,
					MaxRetries: 3,
				},
			},
		},
	}
	schedulerService := scheduler.NewSchedulerService(
		db,
		cfg,
		projectRepo,
		workflowRepo,
		stageRepo,
		taskRepo,
		agentRepoInstance,
		1*time.Second,
	)

	ctx := context.Background()

	// 3. Prepare Data
	// Project
	project := agentModel.Project{
		Name:        "Parallel DAG Project",
		Status:      "active",
		TargetScope: `["127.0.0.1"]`,
	}
	db.Create(&project)

	// Workflow
	workflow := agentModel.Workflow{
		Name: "Parallel Flow",
	}
	db.Create(&workflow)

	// Association
	db.Create(&agentModel.ProjectWorkflow{
		ProjectID:  uint64(project.ID),
		WorkflowID: uint64(workflow.ID),
	})

	// Stage A (Initial, Fast)
	stageA := agentModel.ScanStage{
		WorkflowID:   workflow.ID,
		StageName:    "Stage A",
		ToolName:     "echo", // Dummy tool
		TargetPolicy: `{"provider": "manual", "manual_targets": ["1.1.1.1"]}`,
	}
	db.Create(&stageA)

	// Stage C (Initial, Slow)
	stageC := agentModel.ScanStage{
		WorkflowID:   workflow.ID,
		StageName:    "Stage C",
		ToolName:     "sleep", // Dummy tool
		TargetPolicy: `{"provider": "manual", "manual_targets": ["2.2.2.2"]}`,
	}
	db.Create(&stageC)

	// Stage B (Depends on A)
	stageB := agentModel.ScanStage{
		WorkflowID:   workflow.ID,
		StageName:    "Stage B",
		ToolName:     "echo",
		TargetPolicy: `{"provider": "manual", "manual_targets": ["3.3.3.3"]}`,
		Predecessors: []uint64{uint64(stageA.ID)},
	}
	db.Create(&stageB)

	// 4. Execution Flow

	// --- Tick 1: Initial Scheduling ---
	fmt.Println(">>> Tick 1: Scheduling Initial Stages (A & C)")
	schedulerService.ProcessProject(ctx, &project)

	// Verify A and C are created
	var tasksA []*agentModel.AgentTask
	db.WithContext(ctx).Where("stage_id = ?", stageA.ID).Find(&tasksA)
	var tasksC []*agentModel.AgentTask
	db.WithContext(ctx).Where("stage_id = ?", stageC.ID).Find(&tasksC)
	assert.Equal(t, 1, len(tasksA), "Stage A should have a task")
	assert.Equal(t, 1, len(tasksC), "Stage C should have a task")

	// --- Simulate Execution ---
	// Set Stage A to 'finished' (Fast)
	taskA := tasksA[0]
	taskA.Status = "finished"
	db.Save(taskA)

	// Set Stage C to 'running' (Slow)
	taskC := tasksC[0]
	taskC.Status = "running"
	db.Save(taskC)

	// --- Tick 2: The Parallel Test ---
	fmt.Println(">>> Tick 2: Checking Parallel Scheduling (Should schedule B while C runs)")
	schedulerService.ProcessProject(ctx, &project)

	// Verify Stage B
	var tasksB []*agentModel.AgentTask
	if err := db.WithContext(ctx).Where("stage_id = ?", stageB.ID).Find(&tasksB).Error; err != nil {
		t.Fatalf("Failed to get tasks for Stage B: %v", err)
	}

	// CRITICAL ASSERTION
	if len(tasksB) == 0 {
		t.Fatal("FAILURE: Stage B was NOT scheduled! The global barrier is likely still active.")
	}
	assert.Equal(t, 1, len(tasksB), "Stage B should be scheduled immediately")
	fmt.Printf("Success! Stage B Task ID: %s\n", tasksB[0].TaskID)

	// --- Tick 3: Finish C and B ---
	taskB := tasksB[0]
	taskB.Status = "finished"
	db.Save(taskB)

	taskC.Status = "finished"
	db.Save(taskC)

	// --- Tick 4: Completion ---
	fmt.Println(">>> Tick 3: Finalizing Project")
	schedulerService.ProcessProject(ctx, &project)

	// Check Project Status
	var updatedProject agentModel.Project
	db.First(&updatedProject, project.ID)
	assert.Equal(t, "finished", updatedProject.Status)
}
