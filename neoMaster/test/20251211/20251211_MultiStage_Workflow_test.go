package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"neomaster/internal/app/master/router"
	"neomaster/internal/config"
	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"
	agentRepo "neomaster/internal/repo/mysql/agent"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/core/scheduler"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// SetupMultiStageEnv 初始化多阶段测试环境
func SetupMultiStageEnv() (*gin.Engine, *gorm.DB, error) {
	// 1. 初始化日志
	logger.InitLogger(&config.LogConfig{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	// 2. 数据库配置 (neoscan_dev)
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
		return nil, nil, fmt.Errorf("failed to connect to mysql: %v", err)
	}

	// 自动迁移相关表
	err = db.AutoMigrate(
		&orcModel.Project{},
		&orcModel.Workflow{},
		&orcModel.ScanStage{},
		&orcModel.ScanToolTemplate{},
		&orcModel.AgentTask{},
		&orcModel.ProjectWorkflow{},
		&orcModel.StageResult{}, // 关键：需要 StageResult 表
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to migrate tables: %v", err)
	}

	// 3. 构建 Config (简化版)
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			MySQL: *dbConfig,
		},
		Security: config.SecurityConfig{
			JWT: config.JWTConfig{Secret: "test_secret"},
		},
	}

	// 4. 初始化 Router
	gin.SetMode(gin.TestMode)
	appRouter := router.NewRouter(db, nil, cfg) // Redis 可选

	return appRouter.GetEngine(), db, nil
}

// TestMultiStageWorkflow 测试多阶段工作流及数据流转
func TestMultiStageWorkflow(t *testing.T) {
	// 1. 初始化环境
	_, db, err := SetupMultiStageEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 清理旧数据
	db.Exec("DELETE FROM asset_whitelists")
	db.Exec("DELETE FROM asset_skip_policies")
	db.Exec("DELETE FROM stage_results")
	db.Exec("DELETE FROM agent_tasks")
	db.Exec("DELETE FROM scan_stages")
	db.Exec("DELETE FROM project_workflows")
	db.Exec("DELETE FROM workflows")
	db.Exec("DELETE FROM projects")

	// ==========================================
	// 准备数据
	// ==========================================

	// 1. 创建 Project
	project := orcModel.Project{
		Name:         "MultiStage Project",
		Status:       "running",
		TargetScope:  "192.168.1.1", // 初始种子目标
		ExtendedData: "{}",
		NotifyConfig: "{}",
		ExportConfig: "{}",
		Tags:         "[]",
	}
	err = db.Create(&project).Error
	assert.NoError(t, err)
	assert.NotZero(t, project.ID, "Project ID should not be zero")
	fmt.Printf("Project ID: %d\n", project.ID)

	// Debug: Verify project in DB
	var dbProject orcModel.Project
	if err1 := db.First(&dbProject, project.ID).Error; err1 != nil {
		fmt.Printf("Debug: Failed to fetch project: %v\n", err1)
	} else {
		fmt.Printf("Debug: Project in DB - ID=%d, Status=%s, DeletedAt=%v\n", dbProject.ID, dbProject.Status, dbProject.DeletedAt)
	}

	// Debug: Manual GetRunningProjects check
	var runningProjects []orcModel.Project
	db.Where("status = ?", "running").Find(&runningProjects)
	fmt.Printf("Debug: Manual GetRunningProjects count: %d\n", len(runningProjects))
	if len(runningProjects) == 0 {
		var allProjects []orcModel.Project
		db.Unscoped().Find(&allProjects)
		fmt.Printf("Debug: All Projects in DB (Unscoped): %+v\n", allProjects)
		t.Fatal("Debug: No running projects found immediately after creation!")
	}

	// 2. 创建 Workflow
	workflow := orcModel.Workflow{
		Name:         "MultiStage Workflow",
		Enabled:      true,
		GlobalVars:   "{}",
		PolicyConfig: "{}",
		Tags:         "[]",
	}
	err = db.Create(&workflow).Error
	assert.NoError(t, err)
	assert.NotZero(t, workflow.ID, "Workflow ID should not be zero")
	fmt.Printf("Workflow ID: %d\n", workflow.ID)

	// 关联 Project 和 Workflow
	projectWorkflow := orcModel.ProjectWorkflow{
		ProjectID:  uint64(project.ID),
		WorkflowID: uint64(workflow.ID),
		SortOrder:  1,
	}
	err = db.Create(&projectWorkflow).Error
	assert.NoError(t, err)

	// 3. 创建 Stage 1: PortScan (Order 1)
	stage1 := orcModel.ScanStage{
		WorkflowID:          uint64(workflow.ID),
		StageName:           "PortScan",
		ToolName:            "nmap",
		TargetPolicy:        "{}", // 默认使用 Project 种子
		ExecutionPolicy:     "{}",
		OutputConfig:        "{}",
		PerformanceSettings: "{}",
		NotifyConfig:        "{}",
	}
	err = db.Create(&stage1).Error
	assert.NoError(t, err)
	assert.NotZero(t, stage1.ID, "Stage 1 ID should not be zero")
	fmt.Printf("Stage 1 ID: %d\n", stage1.ID)

	// 4. 创建 Stage 2: ServiceScan (Order 2)
	// 配置 PreviousStageProvider 从 Stage 1 获取端口
	parserConfig := map[string]interface{}{
		"unwind": map[string]string{
			"path": "ports",
		},
		"generate": map[string]string{
			"type":           "ip_port",
			"value_template": "{{target_value}}:{{item.port}}",
		},
	}
	parserConfigJSON, _ := json.Marshal(parserConfig)

	filterRules := map[string]interface{}{
		"stage_name": "PortScan",
	}
	filterRulesJSON, _ := json.Marshal(filterRules)

	targetPolicy := map[string]interface{}{
		"target_sources": []map[string]interface{}{
			{
				"source_type":   "previous_stage",
				"target_type":   "ip_port",
				"parser_config": json.RawMessage(parserConfigJSON),
				"filter_rules":  json.RawMessage(filterRulesJSON),
			},
		},
	}
	targetPolicyJSON, _ := json.Marshal(targetPolicy)

	stage2 := orcModel.ScanStage{
		WorkflowID:          uint64(workflow.ID),
		StageName:           "ServiceScan",
		Predecessors:        []uint64{uint64(stage1.ID)},
		ToolName:            "nuclei",
		TargetPolicy:        string(targetPolicyJSON),
		ExecutionPolicy:     "{}",
		OutputConfig:        "{}",
		PerformanceSettings: "{}",
		NotifyConfig:        "{}",
	}
	err = db.Create(&stage2).Error
	assert.NoError(t, err)
	assert.NotZero(t, stage2.ID, "Stage 2 ID should not be zero")
	fmt.Printf("Stage 2 ID: %d\n", stage2.ID)

	// ==========================================
	// 启动调度器
	// ==========================================

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	projectRepo := orcRepo.NewProjectRepository(db)
	workflowRepo := orcRepo.NewWorkflowRepository(db)
	stageRepo := orcRepo.NewScanStageRepository(db)
	taskRepo := orcRepo.NewTaskRepository(db)
	agentRepository := agentRepo.NewAgentRepository(db)

	schedulerService := scheduler.NewSchedulerService(
		db,
		&config.Config{},
		projectRepo,
		workflowRepo,
		stageRepo,
		taskRepo,
		agentRepository,
		1*time.Second,
	)
	// Manual trigger for deterministic testing
	// schedulerService.Start(ctx)

	// ==========================================
	// 阶段 1 测试
	// ==========================================
	// 触发调度器生成 Stage 1 任务
	schedulerService.ProcessProject(ctx, &project)

	fmt.Println("Checking Stage 1 scheduling...")
	// time.Sleep(2 * time.Second) // No sleep needed for manual trigger

	// 验证 Stage 1 任务是否生成
	var task1 orcModel.AgentTask
	err = db.Where("project_id = ? AND stage_id = ?", project.ID, stage1.ID).First(&task1).Error
	assert.NoError(t, err, "Stage 1 task should be created")
	assert.Equal(t, "nmap", task1.ToolName)
	assert.Equal(t, "pending", task1.Status)

	// ==========================================
	// 模拟 Stage 1 完成并产生结果
	// ==========================================
	fmt.Println("Simulating Stage 1 completion...")

	// 2. 插入 StageResult
	// 模拟 nmap 发现了 80 和 443 端口
	attributes := `{"ports": [{"port": 80, "protocol": "tcp"}, {"port": 443, "protocol": "tcp"}]}`
	result1 := orcModel.StageResult{
		ProjectID:     uint64(project.ID),
		WorkflowID:    uint64(workflow.ID),
		StageID:       uint64(stage1.ID),
		AgentID:       "0",
		TargetType:    "ip",
		TargetValue:   "192.168.1.1",
		ResultType:    "port_scan",
		Attributes:    attributes,
		Evidence:      "{}", // 必须是有效的 JSON
		OutputActions: "{}", // 必须是有效的 JSON
		ProducedAt:    time.Now(),
	}
	fmt.Printf("Debug: Inserting Result with ProjectID=%d, WorkflowID=%d, StageID=%d\n", result1.ProjectID, result1.WorkflowID, result1.StageID)
	err = db.Create(&result1).Error
	if err != nil {
		fmt.Printf("Debug: Insert Error: %v\n", err)
	}
	assert.NoError(t, err)
	fmt.Printf("Debug: Inserted Result ID: %d\n", result1.ID)

	// Debug: Verify query exactly as Provider does
	var debugResults []orcModel.StageResult
	err = db.Where("project_id = ? AND workflow_id = ? AND stage_id = ?", project.ID, workflow.ID, stage1.ID).Find(&debugResults).Error
	assert.NoError(t, err)
	assert.NotEmpty(t, debugResults)

	// Debug: Check inserted StageResult
	var count int64
	db.Model(&orcModel.StageResult{}).Where("project_id = ?", project.ID).Count(&count)
	assert.Equal(t, int64(1), count)

	// 1. 更新任务状态为 finished (这会触发 Scheduler 处理下一阶段)
	// 必须在插入 Result 之后，否则 Scheduler 可能会在 Result 插入前就查询
	task1.Status = "finished"
	err = db.Save(&task1).Error
	assert.NoError(t, err)
	fmt.Printf("Debug: Updated Task 1 Status to %s\n", task1.Status)

	// Manually trigger scheduler for Stage 2
	schedulerService.ProcessProject(ctx, &project)

	// ==========================================
	// 阶段 2 测试
	// ==========================================
	// 5. Verify Stage 2 Task Created
	fmt.Println("Checking Stage 2 task creation...")

	var task2 orcModel.AgentTask
	err = db.Where("project_id = ? AND stage_id = ?", project.ID, stage2.ID).First(&task2).Error
	assert.NoError(t, err, "Stage 2 task should be created")

	found := true // Mimic previous logic variable

	if !found {
		t.Fatal("Stage 2 task was not created after timeout")
	}

	assert.Equal(t, "nuclei", task2.ToolName)
	if task2.Status == "failed" {
		t.Errorf("Stage 2 task failed with error: %s", task2.ErrorMsg)
		// Print Target Policy for debugging
		var stage2DB orcModel.ScanStage
		db.First(&stage2DB, stage2.ID)
		fmt.Printf("Debug: Stage 2 Target Policy: %s\n", stage2DB.TargetPolicy)
	}
	assert.Equal(t, "pending", task2.Status)

	// 验证任务目标是否正确生成
	// 预期目标: 192.168.1.1:80, 192.168.1.1:443
	fmt.Printf("Stage 2 Targets: %s\n", task2.InputTarget)

	// Task.InputTarget 是 []string 的 JSON
	var targets []string
	err = json.Unmarshal([]byte(task2.InputTarget), &targets)
	assert.NoError(t, err)

	assert.Contains(t, targets, "192.168.1.1:80")
	assert.Contains(t, targets, "192.168.1.1:443")
	assert.Len(t, targets, 2)

	fmt.Println("Multi-stage workflow test passed!")
}
