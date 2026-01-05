package test

import (
	"context"
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

// SetupWorkflowTestEnv 初始化工作流测试环境
func SetupWorkflowTestEnv() (*gin.Engine, *gorm.DB, *router.Router, error) {
	// 1. 初始化日志
	logger.InitLogger(&config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
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
		return nil, nil, nil, fmt.Errorf("failed to connect to mysql: %v", err)
	}

	// 自动迁移 Orchestrator 相关表
	err = db.AutoMigrate(
		&orcModel.Project{},
		&orcModel.Workflow{},
		&orcModel.ScanStage{},
		&orcModel.ScanToolTemplate{},
		&orcModel.AgentTask{},
		&orcModel.ProjectWorkflow{},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to migrate tables: %v", err)
	}

	// 3. Redis 配置
	redisConfig := &config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		Database:     0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
	}

	// 连接 Redis
	redisClient, err := database.NewRedisConnection(redisConfig)
	if err != nil {
		// 如果 Redis 连接失败，尝试继续（某些测试可能不需要 Redis）
		fmt.Printf("Warning: failed to connect to redis: %v\n", err)
	}

	// 4. 构建 Config
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			MySQL: *dbConfig,
			Redis: *redisConfig,
		},
		Security: config.SecurityConfig{
			JWT: config.JWTConfig{
				Secret: "test_secret",
			},
		},
	}

	// 5. 初始化 Router
	gin.SetMode(gin.TestMode)
	appRouter := router.NewRouter(db, redisClient, cfg)
	appRouter.SetupRoutes()

	return appRouter.GetEngine(), db, appRouter, nil
}

// TestWorkflowScheduler 测试工作流调度
func TestWorkflowScheduler(t *testing.T) {
	// 1. 初始化环境
	engine, db, _, err := SetupWorkflowTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	_ = engine // 这里的测试主要关注后台调度，不一定通过 HTTP 触发

	// 清理旧数据
	db.Exec("DELETE FROM agent_tasks")
	db.Exec("DELETE FROM scan_stages")
	db.Exec("DELETE FROM project_workflows")
	db.Exec("DELETE FROM workflows")
	db.Exec("DELETE FROM projects")

	// 1. 创建 Project
	project := orcModel.Project{
		Name:         "Test Project",
		Status:       "running", // 必须是 running 才能被调度
		TargetScope:  "192.168.1.1",
		ExtendedData: "{}", // Required for MySQL JSON column
		NotifyConfig: "{}",
		ExportConfig: "{}",
		Tags:         "[]",
	}
	err = db.Create(&project).Error
	assert.NoError(t, err)

	// 2. 创建 Workflow
	workflow := orcModel.Workflow{
		Name:         "Test Workflow",
		Description:  "For testing scheduler",
		Enabled:      true,
		GlobalVars:   "{}",
		PolicyConfig: "{}",
		Tags:         "[]",
	}
	err = db.Create(&workflow).Error
	assert.NoError(t, err)

	// 关联 Project 和 Workflow
	projectWorkflow := orcModel.ProjectWorkflow{
		ProjectID:  uint64(project.ID),
		WorkflowID: uint64(workflow.ID),
		SortOrder:  1,
	}
	err = db.Create(&projectWorkflow).Error
	assert.NoError(t, err)

	// 3. 创建 ScanStage
	stage := orcModel.ScanStage{
		WorkflowID:          uint64(workflow.ID),
		StageName:           "Test Stage 1",
		ToolName:            "nmap",
		TargetPolicy:        orcModel.TargetPolicy{},
		ExecutionPolicy:     orcModel.ExecutionPolicy{},
		PerformanceSettings: orcModel.PerformanceSettings{},
		OutputConfig:        orcModel.OutputConfig{},
		NotifyConfig:        orcModel.NotifyConfig{},
	}
	err = db.Create(&stage).Error
	assert.NoError(t, err)

	// 4. 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 手动初始化调度器以使用短轮询间隔 (1s)
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
	schedulerService.Start(ctx)

	// 5. 等待调度 (调度循环 1s，我们等待 3s 确保至少运行一次)
	fmt.Println("Waiting for scheduler...")
	time.Sleep(3 * time.Second)

	// 6. 验证任务生成
	var tasks []orcModel.AgentTask
	err = db.Where("project_id = ?", project.ID).Find(&tasks).Error
	assert.NoError(t, err)

	// 验证是否有任务生成
	if assert.NotEmpty(t, tasks, "Scheduler should generate tasks") {
		task := tasks[0]
		assert.Equal(t, uint64(project.ID), task.ProjectID)
		assert.Equal(t, uint64(stage.ID), task.StageID)
		assert.Equal(t, "pending", task.Status)
		assert.Equal(t, "nmap", task.ToolName)
		fmt.Printf("Task generated: ID=%s, Tool=%s, Status=%s\n", task.TaskID, task.ToolName, task.Status)
	}
}
