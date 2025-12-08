package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"neomaster/internal/config"
	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"
	agentRepo "neomaster/internal/repo/mysql/agent"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/core/scheduler"

	"gorm.io/gorm"
)

// SetupTestEnv 初始化测试环境 (Simplified)
func SetupTestEnv() (*gorm.DB, error) {
	// 1. 初始化日志
	logger.InitLogger(&config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	})

	// 2. 数据库配置
	dbConfig := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		Username:        "root",
		Password:        "ROOT",
		Database:        "neoscan_dev",
		Charset:         "utf8mb4",
		ParseTime:       true,
		Loc:             "Local",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
	}

	// 连接 MySQL
	db, err := database.NewMySQLConnection(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(
		&orcModel.Project{},
		&orcModel.Workflow{},
		&orcModel.ScanStage{},
		&orcModel.AgentTask{},
		&orcModel.ProjectWorkflow{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %v", err)
	}

	return db, nil
}

func TestTargetPolicyResolution(t *testing.T) {
	db, err := SetupTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 清理数据
	db.Exec("DELETE FROM agent_tasks")
	db.Exec("DELETE FROM scan_stages")
	db.Exec("DELETE FROM workflows")
	db.Exec("DELETE FROM projects")
	db.Exec("DELETE FROM project_workflows")

	ctx := context.Background()

	// 1. 创建 Workflow
	workflow := &orcModel.Workflow{
		Name:         "Target Policy Test Workflow",
		Description:  "Testing dynamic target resolution",
		Version:      "v1",
		GlobalVars:   "{}",
		PolicyConfig: "{}",
		Tags:         "[]",
		Enabled:      true,
	}
	if err := db.Create(workflow).Error; err != nil {
		t.Fatalf("Create workflow failed: %v", err)
	}

	// 2. 创建 Stage，配置 TargetPolicy
	// 使用 Manual 策略，指定 1.1.1.1 和 2.2.2.2
	targetPolicy := map[string]interface{}{
		"target_sources": []map[string]interface{}{
			{
				"source_type":  "manual",
				"source_value": "1.1.1.1,2.2.2.2",
			},
		},
	}
	policyJSON, _ := json.Marshal(targetPolicy)

	stage := &orcModel.ScanStage{
		WorkflowID:          uint64(workflow.ID),
		StageName:           "Manual Target Stage",
		StageType:           "port_scan",
		StageOrder:          1,
		TargetPolicy:        string(policyJSON),
		Enabled:             true,
		OutputConfig:        "{}",
		NotifyConfig:        "{}",
		ExecutionPolicy:     "{}",
		PerformanceSettings: "{}",
		ToolName:            "nmap",
	}
	if err := db.Create(stage).Error; err != nil {
		t.Fatalf("Create stage failed: %v", err)
	}

	// 3. 创建 Project，故意不设置 Seed Targets (ExtendedData)
	project := &orcModel.Project{
		Name:         "Target Policy Project",
		Status:       "running",
		ExtendedData: "{}", // Empty seed targets
		NotifyConfig: "{}",
		ExportConfig: "{}",
		Tags:         "[]",
	}
	if err := db.Create(project).Error; err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	// 关联 Project 和 Workflow
	pw := &orcModel.ProjectWorkflow{
		ProjectID:  uint64(project.ID),
		WorkflowID: uint64(workflow.ID),
		SortOrder:  1,
	}
	if err := db.Create(pw).Error; err != nil {
		t.Fatalf("Create project_workflow failed: %v", err)
	}

	// 4. 手动初始化 Scheduler (使用 1秒间隔)
	projectRepo := orcRepo.NewProjectRepository(db)
	workflowRepo := orcRepo.NewWorkflowRepository(db)
	stageRepo := orcRepo.NewScanStageRepository(db)
	taskRepo := orcRepo.NewTaskRepository(db)
	agentRepository := agentRepo.NewAgentRepository(db)

	schedulerService := scheduler.NewSchedulerService(
		projectRepo,
		workflowRepo,
		stageRepo,
		taskRepo,
		agentRepository,
		1*time.Second,
	)

	// 启动
	schedulerService.Start(ctx)
	defer schedulerService.Stop()

	// 5. 等待任务生成
	fmt.Println("Waiting for task generation...")
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var tasks []orcModel.AgentTask
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for tasks")
		case <-ticker.C:
			db.Where("project_id = ?", project.ID).Find(&tasks)
			if len(tasks) > 0 {
				goto Check
			}
		}
	}

Check:
	// 6. 验证任务
	if len(tasks) != 1 {
		t.Logf("Expected 1 task (containing 2 targets), got %d tasks", len(tasks))
	}

	task := tasks[0]
	// 检查 InputTarget 字段 (JSON)
	var inputTarget []string
	if err := json.Unmarshal([]byte(task.InputTarget), &inputTarget); err != nil {
		t.Fatalf("Failed to unmarshal InputTarget: %v. Raw: %s", err, task.InputTarget)
	}

	if len(inputTarget) != 2 {
		t.Fatalf("Expected 2 targets in InputTarget, got %d", len(inputTarget))
	}

	targetMap := make(map[string]bool)
	for _, t := range inputTarget {
		targetMap[t] = true
	}

	if !targetMap["1.1.1.1"] || !targetMap["2.2.2.2"] {
		t.Fatalf("Targets mismatch. Got: %v", inputTarget)
	}

	fmt.Println("Test Passed: Target Policy resolved correctly!")
}
