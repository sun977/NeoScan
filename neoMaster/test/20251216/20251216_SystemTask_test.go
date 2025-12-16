package test_20251216

import (
	"context"
	"fmt"
	"testing"
	"time"

	"neomaster/internal/config"
	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
	agentRepo "neomaster/internal/repo/mysql/agent"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/core/local_agent"
	"neomaster/internal/service/orchestrator/core/scheduler"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// SetupTestEnv 初始化测试环境
func SetupTestEnv() (*gorm.DB, error) {
	// 使用 SQLite 内存数据库 (开启 shared cache 以支持多 goroutine)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移 schema
	err = db.AutoMigrate(
		&orcModel.Project{},
		&orcModel.Workflow{},
		&orcModel.ScanStage{},
		&orcModel.AgentTask{},
		&orcModel.ProjectWorkflow{},
		&assetModel.AssetWhitelist{},
		&assetModel.AssetSkipPolicy{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestSystemTaskCategorization(t *testing.T) {
	db, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	ctx := context.Background()

	// 1. Initialize Repositories
	projectRepo := orcRepo.NewProjectRepository(db)
	workflowRepo := orcRepo.NewWorkflowRepository(db)
	stageRepo := orcRepo.NewScanStageRepository(db)
	taskRepo := orcRepo.NewTaskRepository(db)
	agentRepository := agentRepo.NewAgentRepository(db)

	cfg := &config.Config{}
	schedulerService := scheduler.NewSchedulerService(
		db,
		cfg,
		projectRepo,
		workflowRepo,
		stageRepo,
		taskRepo,
		agentRepository,
		1*time.Second,
	)

	// 2. Create Project & Workflow
	project := &orcModel.Project{
		Name:         "System Task Test Project",
		Status:       "running",
		TargetScope:  "127.0.0.1",
		ExtendedData: "{}",
		NotifyConfig: "{}",
		ExportConfig: "{}",
		Tags:         "[]",
	}
	db.Create(project)

	workflow := &orcModel.Workflow{
		Name:    "System Task Workflow",
		Enabled: true,
	}
	db.Create(workflow)

	db.Create(&orcModel.ProjectWorkflow{
		ProjectID:  uint64(project.ID),
		WorkflowID: uint64(workflow.ID),
	})

	// 3. Create Stages
	// Stage 1: Agent Task (nmap)
	stageAgent := &orcModel.ScanStage{
		WorkflowID:          uint64(workflow.ID),
		StageName:           "Agent Stage",
		ToolName:            "nmap",
		TargetPolicy:        "{}",
		Enabled:             true,
		ExecutionPolicy:     "{}",
		PerformanceSettings: "{}",
		OutputConfig:        "{}",
		NotifyConfig:        "{}",
	}
	db.Create(stageAgent)

	// Stage 2: System Task (sys_tag_propagation)
	stageSystem := &orcModel.ScanStage{
		WorkflowID:          uint64(workflow.ID),
		StageName:           "System Stage",
		ToolName:            "sys_tag_propagation",
		TargetPolicy:        "{}",
		Enabled:             true,
		ExecutionPolicy:     "{}",
		PerformanceSettings: "{}",
		OutputConfig:        "{}",
		NotifyConfig:        "{}",
	}
	db.Create(stageSystem)

	// 4. Trigger Scheduler
	schedulerService.ProcessProject(ctx, project)

	// 5. Verify Tasks
	var tasks []orcModel.AgentTask
	db.Where("project_id = ?", project.ID).Find(&tasks)
	assert.Equal(t, 2, len(tasks), "Should generate 2 tasks")

	var agentTask, systemTask *orcModel.AgentTask
	for i := range tasks {
		switch tasks[i].ToolName {
		case "nmap":
			agentTask = &tasks[i]
		case "sys_tag_propagation":
			systemTask = &tasks[i]
		}
	}

	assert.NotNil(t, agentTask, "Agent task should exist")
	assert.Equal(t, "agent", agentTask.TaskCategory, "Nmap task should be category 'agent'")

	assert.NotNil(t, systemTask, "System task should exist")
	assert.Equal(t, "system", systemTask.TaskCategory, "Sys task should be category 'system'")

	// 6. Test GetPendingTasks filtering
	pendingAgentTasks, err := taskRepo.GetPendingTasks(ctx, "agent", 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pendingAgentTasks))
	assert.Equal(t, agentTask.TaskID, pendingAgentTasks[0].TaskID)

	pendingSystemTasks, err := taskRepo.GetPendingTasks(ctx, "system", 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pendingSystemTasks))
	assert.Equal(t, systemTask.TaskID, pendingSystemTasks[0].TaskID)

	fmt.Println("TestSystemTaskCategorization Passed!")
}

func TestSystemTaskExecution(t *testing.T) {
	db, err := SetupTestEnv()
	assert.NoError(t, err)

	// Add AssetHost migration
	err = db.AutoMigrate(&assetModel.AssetHost{})
	assert.NoError(t, err)

	// 1. Seed Asset
	host := &assetModel.AssetHost{
		IP:   "192.168.1.100",
		Tags: "[]",
	}
	db.Create(host)

	// 2. Create System Task
	taskRepo := orcRepo.NewTaskRepository(db)

	payload := `{"target_type": "host", "action": "add", "tags": ["internal"], "rule": {"field": "ip", "operator": "cidr", "value": "192.168.1.0/24"}}`

	task := &orcModel.AgentTask{
		TaskID:       "sys_task_001",
		ToolName:     "sys_tag_propagation",
		TaskCategory: "system",
		Status:       "pending",
		ToolParams:   payload,
	}
	db.Create(task)

	// 3. Start Worker
	// Note: We reduce interval for testing, but since it's private field, we just rely on immediate process or we can't easily change it without exposing option.
	// But `Start()` runs `processTasks` in a loop.
	// Actually, `NewSystemTaskWorker` sets 5s interval.
	// To speed up test, we might want to manually call `processTasks` if we could, but it's private.
	// However, `Start()` runs in a goroutine.
	// Wait, 5 seconds is too long for unit test.
	// We should probably allow configuring interval or manually trigger.
	// For this test, I will modify NewSystemTaskWorker in the main code to accept options?
	// Or just use reflection? No, too hacky.
	// Pragmatic approach: Just modify the worker to have a shorter interval for testing?
	// Or better: Expose a method to run once, or pass config.
	// For now, let's assume we can wait or modify `worker.go` to have SetInterval.

	worker := local_agent.NewLocalAgent(db, taskRepo)
	// Hack: Use reflection to set interval if possible, or just add SetInterval method.
	// Let's add SetInterval to worker.go first?
	// Or simpler: Just rely on the fact that we can call internal methods in same package test?
	// But this test is in `test_20251216` package (different package).

	// Let's add SetInterval to `SystemTaskWorker` in `worker.go`.
	// It's a useful utility anyway.
	worker.SetInterval(100 * time.Millisecond)

	worker.Start()
	defer worker.Stop()

	// 4. Wait for execution
	time.Sleep(500 * time.Millisecond)

	// 5. Verify
	var updatedHost assetModel.AssetHost
	db.First(&updatedHost, host.ID)

	assert.Contains(t, updatedHost.Tags, "internal")

	// Verify Task Status
	var updatedTask orcModel.AgentTask
	db.Where("task_id = ?", "sys_task_001").First(&updatedTask)
	assert.Equal(t, "completed", updatedTask.Status)
}

func TestSystemTaskCleanup(t *testing.T) {
	db, err := SetupTestEnv()
	assert.NoError(t, err)

	err = db.AutoMigrate(&assetModel.AssetHost{})
	assert.NoError(t, err)

	// 1. Seed Asset
	host := &assetModel.AssetHost{
		IP:   "192.168.1.200",
		Tags: "[\"deprecated\"]",
	}
	db.Create(host)

	// 2. Create Cleanup Task
	taskRepo := orcRepo.NewTaskRepository(db)

	payload := `{"target_type": "host", "rule": {"field": "tags", "operator": "contains", "value": "deprecated"}}`

	task := &orcModel.AgentTask{
		TaskID:       "sys_task_002",
		ToolName:     "sys_asset_cleanup",
		TaskCategory: "system",
		Status:       "pending",
		ToolParams:   payload,
	}
	db.Create(task)

	// 3. Start Worker
	worker := local_agent.NewLocalAgent(db, taskRepo)
	worker.SetInterval(100 * time.Millisecond)
	worker.Start()
	defer worker.Stop()

	// 4. Wait
	time.Sleep(500 * time.Millisecond)

	// 5. Verify
	var count int64
	db.Model(&assetModel.AssetHost{}).Where("ip = ?", "192.168.1.200").Count(&count)
	assert.Equal(t, int64(0), count)

	var updatedTask orcModel.AgentTask
	db.Where("task_id = ?", "sys_task_002").First(&updatedTask)
	assert.Equal(t, "completed", updatedTask.Status)
}
