// SchedulerService 调度引擎服务接口
//  1. 启动调度引擎
//  2. 停止调度引擎
//
// 调度器引擎主要职责包括：
// 1.定时任务管理：检查并触发配置了Cron表达式的定时扫描项目
// 2.项目状态跟踪：监控运行中项目的进度和状态
// 3.阶段流转控制：确保项目按预定义的工作流顺序执行各个阶段
// 4.任务生成：根据阶段配置和目标生成具体可执行的任务
// 5.策略执行：在任务执行前进行安全和合规性检查
// 调度引擎的工作流程如下：
// 1.启动后按照设定的时间间隔（默认10秒）循环执行调度逻辑
// 2.每次调度首先检查是否有定时项目需要触发执行
// 3.获取所有处于"running"状态的项目进行处理
// 4.对每个项目：
// - 检查是否有正在运行的任务，如有则等待
// - 获取最新任务状态，如果上一个任务失败则暂停项目
// - 查找下一个需要执行的阶段
// - 如果没有下一阶段则标记项目为完成
// - 否则生成新任务并存入数据库供agent执行
package scheduler

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	agentRepo "neomaster/internal/repo/mysql/agent"
	assetRepo "neomaster/internal/repo/mysql/asset"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/policy" // 策略执行器模块

	"github.com/robfig/cron/v3" // 定时任务库
	"gorm.io/gorm"
)

// SchedulerService 调度引擎服务接口
type SchedulerService interface {
	Start(ctx context.Context)
	Stop()
	ProcessProject(ctx context.Context, project *orcModel.Project)
}

type schedulerService struct {
	projectRepo    *orcRepo.ProjectRepository
	workflowRepo   *orcRepo.WorkflowRepository
	stageRepo      *orcRepo.ScanStageRepository
	taskRepo       orcRepo.TaskRepository
	agentRepo      agentRepo.AgentRepository
	taskGenerator  TaskGenerator         // 任务生成器接口
	targetProvider policy.TargetProvider // 目标提供者接口
	policyEnforcer policy.PolicyEnforcer // 策略执行器接口

	stopChan chan struct{} // 停止信号通道
	interval time.Duration // 轮询间隔, 默认10秒
}

// NewSchedulerService 创建调度引擎服务
// 初始化调度引擎服务，设置必要的依赖和参数
func NewSchedulerService(
	db *gorm.DB,
	projectRepo *orcRepo.ProjectRepository,
	workflowRepo *orcRepo.WorkflowRepository,
	stageRepo *orcRepo.ScanStageRepository,
	taskRepo orcRepo.TaskRepository,
	agentRepo agentRepo.AgentRepository,
	interval time.Duration,
) SchedulerService {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	// 初始化策略仓库
	policyRepo := assetRepo.NewAssetPolicyRepository(db)

	return &schedulerService{
		projectRepo:    projectRepo,
		workflowRepo:   workflowRepo,
		stageRepo:      stageRepo,
		taskRepo:       taskRepo,
		agentRepo:      agentRepo,
		taskGenerator:  NewTaskGenerator(),
		targetProvider: policy.NewTargetProvider(db),
		policyEnforcer: policy.NewPolicyEnforcer(projectRepo, policyRepo),
		stopChan:       make(chan struct{}),
		interval:       interval,
	}
}

// Start 启动调度引擎
func (s *schedulerService) Start(ctx context.Context) {
	logger.LogInfo("Starting Scheduler Engine...", "", 0, "", "service.scheduler.Start", "", map[string]interface{}{
		"interval": s.interval.String(),
	})
	go s.loop(ctx) // 启动调度循环 在goroutine中运行主循环
}

// Stop 停止调度引擎
func (s *schedulerService) Stop() {
	close(s.stopChan)
	logger.LogInfo("Scheduler Engine Stopped", "", 0, "", "service.scheduler.Stop", "", nil)
}

// loop 调度循环
// 1. 立即执行一次调度
// 2. 按照配置的轮询间隔执行调度
func (s *schedulerService) loop(ctx context.Context) {
	ticker := time.NewTicker(s.interval) // 使用配置的轮询间隔
	defer ticker.Stop()

	// 立即执行一次调度
	s.schedule(ctx)

	for {
		select {
		case <-ctx.Done(): // 上下文取消信号
			return
		case <-s.stopChan: // 停止信号
			return
		case <-ticker.C: // 轮询信号
			s.schedule(ctx)
		}
	}
}

// schedule 执行单次调度逻辑
// 1. 检查是否有定时任务需要触发
// 2. 获取运行中的项目
// 3. 处理每个项目的扫描阶段
func (s *schedulerService) schedule(ctx context.Context) {
	// 0. 检查定时任务触发
	s.checkScheduledProjects(ctx)

	// 1. 获取运行中的项目
	projects, err := s.projectRepo.GetRunningProjects(ctx)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.schedule", "REPO", map[string]interface{}{
			"msg": "failed to get running projects",
		})
		return
	}

	if len(projects) == 0 {
		return
	}

	for _, project := range projects {
		s.ProcessProject(ctx, project)
	}
}

// checkScheduledProjects 检查是否有定时任务需要触发
// 1. 获取所有已配置定时任务的项目
// 2. 解析 Cron 表达式
// 3. 检查是否到了执行时间
func (s *schedulerService) checkScheduledProjects(ctx context.Context) {
	projects, err := s.projectRepo.GetScheduledProjects(ctx)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.checkScheduledProjects", "REPO", nil)
		return
	}

	if len(projects) == 0 {
		return
	}

	// 使用标准 Cron 解析器 (5位: 分 时 日 月 周)
	// 如果需要支持秒级，可以使用 cron.NewParser(cron.Second | cron.Minute | ...)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	now := time.Now()

	for _, project := range projects {
		if project.CronExpr == "" {
			continue
		}

		schedule, err := parser.Parse(project.CronExpr)
		if err != nil {
			logger.LogError(err, "", 0, "", "service.scheduler.checkScheduledProjects", "INTERNAL", map[string]interface{}{
				"project_id": project.ID,
				"cron_expr":  project.CronExpr,
				"msg":        "invalid cron expression",
			})
			continue
		}

		// 计算上次执行后的下一次执行时间
		// 如果 LastExecTime 为 nil (从未执行过)，则假设上次执行时间为 CreatedAt
		var lastTime time.Time
		if project.LastExecTime != nil {
			lastTime = *project.LastExecTime
		} else {
			lastTime = project.CreatedAt
		}

		// 防御性编程：如果 lastTime 是零值，设为很久以前
		if lastTime.IsZero() {
			lastTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		nextTime := schedule.Next(lastTime)

		// 如果下一次执行时间 <= 当前时间，说明到了执行时间 (或者错过了执行时间)
		// 并且 nextTime 不能是零值 (如果 cron 表达式不再匹配任何时间)
		if !nextTime.IsZero() && (nextTime.Before(now) || nextTime.Equal(now)) {
			logger.LogInfo("Triggering scheduled project", "", 0, "", "service.scheduler.checkScheduledProjects", "", map[string]interface{}{
				"project_id": project.ID,
				"next_time":  nextTime,
				"now":        now,
			})

			// 触发项目执行
			project.Status = "running"
			project.LastExecTime = &now
			if err := s.projectRepo.UpdateProject(ctx, project); err != nil {
				logger.LogError(err, "", 0, "", "service.scheduler.checkScheduledProjects", "REPO", map[string]interface{}{
					"project_id": project.ID,
				})
			}
		}
	}
}

// ProcessProject 处理单个项目的调度逻辑 (Public for testing and manual trigger)
func (s *schedulerService) ProcessProject(ctx context.Context, project *orcModel.Project) {
	loggerFields := map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
	}

	// 1. 检查是否有正在运行的任务 (Barrier: 只有当所有任务都完成/失败后，才进行下一步)
	hasRunning, err := s.taskRepo.HasRunningTasks(ctx, uint64(project.ID))
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.processProject", "REPO", loggerFields)
		return
	}
	if hasRunning {
		return // 等待当前任务全部完成
	}

	// 2. 获取该项目最新的任务状态 (用于判断上一阶段结果)
	lastTask, err := s.taskRepo.GetLatestTaskByProjectID(ctx, uint64(project.ID))
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.processProject", "REPO", loggerFields)
		return
	}

	// 3. 判断状态
	// Case B: 上一个任务失败，暂停项目
	if lastTask != nil && lastTask.Status == "failed" {
		logger.LogInfo("Project paused due to task failure", "", 0, "", "service.scheduler.processProject", "", loggerFields)
		project.Status = "error" // 或者 paused
		s.projectRepo.UpdateProject(ctx, project)
		return
	}

	// Case C: 初始启动 或 上一个任务完成 -> 寻找下一批需要执行的 Stages (DAG)
	nextStages, err := s.findNextStages(ctx, project)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.processProject", "INTERNAL", loggerFields)
		return
	}

	// Case D: 没有下一个 Stage 了 -> 项目完成
	// 注意：DAG 模式下，如果所有分支都走完了，nextStages 为空
	if len(nextStages) == 0 {
		// 双重检查：确认是否真的所有 Stage 都完成了？
		// 暂时简化：如果没有可执行的 Stage，且没有正在运行的任务（前面已检查），则认为项目完成
		logger.LogInfo("Project finished", "", 0, "", "service.scheduler.processProject", "", loggerFields)
		project.Status = "finished"
		project.LastExecTime = nil // Optional: update finish time if needed
		s.projectRepo.UpdateProject(ctx, project)
		return
	}

	// Case E: 生成新任务 (对每个可执行的 Stage)
	for _, nextStage := range nextStages {
		s.generateTasksForStage(ctx, project, nextStage)
	}
}

// generateTasksForStage 为单个 Stage 生成任务
func (s *schedulerService) generateTasksForStage(ctx context.Context, project *orcModel.Project, nextStage *orcModel.ScanStage) {
	loggerFields := map[string]interface{}{
		"project_id":   project.ID,
		"project_name": project.Name,
		"stage_id":     nextStage.ID,
		"stage_name":   nextStage.StageName,
	}

	// 1. 获取种子目标 (Seed Targets) 从 Project.TargetScope 配置
	var seedTargets []string
	if project.TargetScope != "" {
		// 尝试解析为 JSON 数组
		var targets []string
		if json.Unmarshal([]byte(project.TargetScope), &targets) == nil {
			seedTargets = targets
		} else {
			// 如果不是 JSON，尝试按逗号、换行符分隔
			f := func(c rune) bool {
				return c == ',' || c == ';' || c == '\n' || c == '\r' || c == ' '
			}
			fields := strings.FieldsFunc(project.TargetScope, f)
			for _, field := range fields {
				if t := strings.TrimSpace(field); t != "" {
					seedTargets = append(seedTargets, t)
				}
			}
		}
	}

	// 2. 使用 TargetProvider 解析最终目标 (应用 TargetPolicy)
	// [Context Injection]
	ctx = context.WithValue(ctx, policy.CtxKeyProjectID, uint64(project.ID))    // 项目ID 注入上下文
	ctx = context.WithValue(ctx, policy.CtxKeyWorkflowID, nextStage.WorkflowID) // WorkflowID 注入上下文
	ctx = context.WithValue(ctx, policy.CtxKeyStageID, nextStage.ID)            // StageID 注入上下文
	// ctx = context.WithValue(ctx, policy.CtxKeyStageOrder, nextStage.StageOrder) // Deprecated: DAG模式下不再使用 Order

	//  1. 解析种子目标 (Seed Targets)
	//  2. 应用 TargetPolicy 进行转换/过滤
	resolvedTargetObjs, err := s.targetProvider.ResolveTargets(ctx, nextStage.TargetPolicy, seedTargets)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.processProject", "TARGET_RESOLVE", loggerFields)
		return
	}

	// 转换 []Target 为 []string，供 GenerateTasks 使用
	resolvedTargets := make([]string, 0, len(resolvedTargetObjs))
	for _, t := range resolvedTargetObjs {
		resolvedTargets = append(resolvedTargets, t.Value)
	}

	// Fallback if no targets found (Safety net)
	if len(resolvedTargets) == 0 {
		logger.LogWarn("No targets resolved, using fallback", "", 0, "", "service.scheduler.processProject", "", loggerFields)
		resolvedTargets = []string{"127.0.0.1"}
	}

	newTasks, err := s.taskGenerator.GenerateTasks(nextStage, uint64(project.ID), resolvedTargets)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.processProject", "INTERNAL", loggerFields)
		return
	}

	// 保存任务到数据库
	for _, task := range newTasks {
		// 3. 策略检查 (Policy Enforcer)
		if err := s.policyEnforcer.Enforce(ctx, task); err != nil {
			logger.LogWarn("Task blocked by policy", "", 0, "", "service.scheduler.processProject", "", map[string]interface{}{
				"task_id": task.TaskID,
				"error":   err.Error(),
			})
			task.Status = "failed"
			task.ErrorMsg = "Policy violation: " + err.Error()
		}

		if err := s.taskRepo.CreateTask(ctx, task); err != nil {
			logger.LogError(err, "", 0, "", "service.scheduler.processProject", "REPO", loggerFields)
			continue
		}
		logger.LogInfo("Generated new task", "", 0, "", "service.scheduler.processProject", "", map[string]interface{}{
			"task_id":  task.TaskID,
			"stage_id": task.StageID,
			"tool":     task.ToolName,
			"status":   task.Status,
		})
	}
}

// findNextStages 查找下一批需要执行的 Stages (DAG核心逻辑)
// 逻辑：
// 1. 获取 Workflow 下所有 Stages
// 2. 获取该 Project 已完成(或已开始)的所有 Stage IDs
// 3. 遍历所有 Stages:
//   - 如果 Stage 已经执行过，跳过
//   - 检查 Stage 的 Predecessors (依赖)
//   - 如果所有 Predecessors 都已在 "已完成列表" 中，则该 Stage Ready
func (s *schedulerService) findNextStages(ctx context.Context, project *orcModel.Project) ([]*orcModel.ScanStage, error) {
	// 1. 获取 Workflow
	workflows, err := s.workflowRepo.GetWorkflowsByProjectID(ctx, uint64(project.ID))
	if err != nil || len(workflows) == 0 {
		return nil, err
	}
	workflow := workflows[0] // 假设单 Workflow

	// 2. 获取所有 Stages
	stages, err := s.stageRepo.GetStagesByWorkflowID(ctx, uint64(workflow.ID))
	if err != nil || len(stages) == 0 {
		return nil, err
	}

	// 3. 获取已执行的任务，推导已完成的 Stages
	// 注意：这里需要从 TaskRepo 获取所有任务，以判断哪些 Stage 已经跑过了
	// 这是一个相对昂贵的操作，生产环境应优化为专门的 ProjectStageStatus 表
	tasks, err := s.taskRepo.GetTasksByProjectID(ctx, uint64(project.ID))
	if err != nil {
		return nil, err
	}

	// 构建已完成/已开始的 Stage Set
	executedStageIDs := make(map[uint64]bool)
	for _, task := range tasks {
		executedStageIDs[task.StageID] = true
	}

	var nextStages []*orcModel.ScanStage

	// 4. DAG 判定
	for _, stage := range stages {
		// 如果该 Stage 已经执行过(或正在执行)，跳过
		if executedStageIDs[uint64(stage.ID)] {
			continue
		}

		// 解析依赖
		predecessors := stage.Predecessors

		// 检查依赖是否满足
		dependenciesResolved := true
		if len(predecessors) > 0 {
			for _, predID := range predecessors {
				if !executedStageIDs[predID] {
					dependenciesResolved = false
					break
				}
			}
		} else {
			// 没有依赖 = 入口节点 (Initial Stage)
			// dependenciesResolved = true
		}

		if dependenciesResolved {
			nextStages = append(nextStages, stage)
		}
	}

	return nextStages, nil
}

// Deprecated: 原 getFirstStage 和 getNextStage 已废弃，被 findNextStages 取代
