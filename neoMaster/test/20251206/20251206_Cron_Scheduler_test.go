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

// SetupCronTestEnv 初始化 Cron 测试环境
func SetupCronTestEnv() (*gin.Engine, *gorm.DB, *router.Router, error) {
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

// TestCronScheduler 测试 Cron 调度
func TestCronScheduler(t *testing.T) {
	// 1. 初始化环境
	engine, db, _, err := SetupCronTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	_ = engine

	// 清理旧数据
	db.Exec("DELETE FROM agent_tasks")
	db.Exec("DELETE FROM scan_stages")
	db.Exec("DELETE FROM project_workflows")
	db.Exec("DELETE FROM workflows")
	db.Exec("DELETE FROM projects")

	// 2. 创建 Project (Cron 类型)
	// 设置 LastExecTime 为 2 分钟前，Cron 为每分钟执行 (* * * * *)
	// 这样 Next 执行时间应该是 1 分钟前，也就是应该立即触发
	lastExecTime := time.Now().Add(-2 * time.Minute)
	project := orcModel.Project{
		Name:         "Cron Test Project",
		Status:       "idle",      // 初始状态 idle
		ScheduleType: "cron",      // 调度类型 cron
		CronExpr:     "* * * * *", // 每分钟
		Enabled:      true,
		TargetScope:  "192.168.1.1", // 设置 TargetScope
		ExtendedData: "{}",          // ExtendedData 为空 JSON
		NotifyConfig: "{}",
		ExportConfig: "{}",
		Tags:         "[]",
		LastExecTime: &lastExecTime,
	}
	err = db.Create(&project).Error
	assert.NoError(t, err)

	// 3. 创建 Workflow (Dummy)
	workflow := orcModel.Workflow{
		Name:         "Cron Workflow",
		Enabled:      true,
		GlobalVars:   "{}",
		PolicyConfig: "{}",
		Tags:         "[]",
	}
	err = db.Create(&workflow).Error
	assert.NoError(t, err)

	// 关联
	projectWorkflow := orcModel.ProjectWorkflow{
		ProjectID:  uint64(project.ID),
		WorkflowID: uint64(workflow.ID),
		SortOrder:  1,
	}
	db.Create(&projectWorkflow)

	// 4. 创建 Stage
	stage := orcModel.ScanStage{
		WorkflowID:          uint64(workflow.ID),
		StageName:           "Cron Stage",
		StageOrder:          1,
		ToolName:            "ping",
		TargetPolicy:        "{}",
		ExecutionPolicy:     "{}",
		PerformanceSettings: "{}",
		OutputConfig:        "{}",
		NotifyConfig:        "{}",
		Enabled:             true,
	}
	db.Create(&stage)

	// 5. 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	projectRepo := orcRepo.NewProjectRepository(db)
	workflowRepo := orcRepo.NewWorkflowRepository(db)
	stageRepo := orcRepo.NewScanStageRepository(db)
	taskRepo := orcRepo.NewTaskRepository(db)
	agentRepository := agentRepo.NewAgentRepository(db)

	schedulerService := scheduler.NewSchedulerService(
		db,
		projectRepo,
		workflowRepo,
		stageRepo,
		taskRepo,
		agentRepository,
		1*time.Second,
	)
	schedulerService.Start(ctx)

	// 6. 等待调度
	fmt.Println("Waiting for cron trigger...")
	// 应该在第一次检查时就触发
	time.Sleep(3 * time.Second)

	// 7. 验证 Project 状态变为 running
	var updatedProject orcModel.Project
	err = db.First(&updatedProject, project.ID).Error
	assert.NoError(t, err)

	fmt.Printf("Project Status: %s, LastExecTime: %v\n", updatedProject.Status, updatedProject.LastExecTime)
	assert.Equal(t, "running", updatedProject.Status, "Project status should be running")
	assert.True(t, updatedProject.LastExecTime.After(lastExecTime), "LastExecTime should be updated")

	// 8. 验证任务生成
	var tasks []orcModel.AgentTask
	err = db.Where("project_id = ?", project.ID).Find(&tasks).Error
	assert.NoError(t, err)
	assert.NotEmpty(t, tasks, "Should have generated tasks")
	fmt.Printf("Generated %d tasks\n", len(tasks))
}
