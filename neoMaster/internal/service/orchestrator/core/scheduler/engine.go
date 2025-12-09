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
	orcRepo "neomaster/internal/repo/mysql/orchestrator"
	"neomaster/internal/service/orchestrator/policy" // 策略执行器模块

	"github.com/robfig/cron/v3" // 定时任务库
	"gorm.io/gorm"
)

// SchedulerService 调度引擎服务接口
type SchedulerService interface {
	Start(ctx context.Context)
	Stop()
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
	return &schedulerService{
		projectRepo:    projectRepo,
		workflowRepo:   workflowRepo,
		stageRepo:      stageRepo,
		taskRepo:       taskRepo,
		agentRepo:      agentRepo,
		taskGenerator:  NewTaskGenerator(),
		targetProvider: policy.NewTargetProvider(db),
		policyEnforcer: policy.NewPolicyEnforcer(projectRepo),
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
		s.processProject(ctx, project)
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

// processProject 处理单个项目的调度
// 1.检查是否有正在运行的任务，如果有则等待
// 2.获取最新的任务以确定上一阶段的结果
// 3.如果上一任务失败，则暂停整个项目
// 4.查找下一阶段，如果没有下一阶段则标记项目为完成
// 5.从项目TargetScope中提取种子目标：
// - 首先尝试解析为JSON数组
// - 如果不是JSON，则按常见分隔符（逗号、分号、换行等）分割
// 6.使用TargetProvider解析最终目标（应用TargetPolicy）
// 7.使用策略执行器对任务进行策略检查
// 8.使用任务生成器基于下一阶段和目标生成新任务
// 9.将新任务保存到数据库
func (s *schedulerService) processProject(ctx context.Context, project *orcModel.Project) {
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

	// Case C: 初始启动 或 上一个任务完成 -> 寻找下一个 Stage
	nextStage, err := s.findNextStage(ctx, project, lastTask)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.processProject", "INTERNAL", loggerFields)
		return
	}

	// Case D: 没有下一个 Stage 了 -> 项目完成
	if nextStage == nil {
		logger.LogInfo("Project finished", "", 0, "", "service.scheduler.processProject", "", loggerFields)
		project.Status = "finished"
		project.LastExecTime = nil // Optional: update finish time if needed
		s.projectRepo.UpdateProject(ctx, project)
		return
	}

	// Case E: 生成新任务
	// 1. 获取种子目标 (Seed Targets) 从 Project.TargetScope 配置
	var seedTargets []string
	if project.TargetScope != "" {
		// 尝试解析为 JSON 数组
		var targets []string
		if json.Unmarshal([]byte(project.TargetScope), &targets) == nil {
			seedTargets = targets
		} else {
			// 如果不是 JSON，尝试按逗号、换行符分隔
			// 支持常见的分隔符：逗号、分号、换行、空格
			// 这种方式兼容 CIDR/Domain 列表的简单文本格式
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
		// 在任务落库前进行最后一道"安检"
		if err := s.policyEnforcer.Enforce(ctx, task); err != nil {
			logger.LogWarn("Task blocked by policy", "", 0, "", "service.scheduler.processProject", "", map[string]interface{}{
				"task_id": task.TaskID,
				"error":   err.Error(),
			})
			// 标记任务为失败或拒绝，并保存以便审计？
			// 这里我们选择直接丢弃不合规的任务，或者将其状态设为 'blocked' 存库
			// 为了审计，最好存库并标记为 failed/blocked
			task.Status = "failed"
			task.ErrorMsg = "Policy violation: " + err.Error()
			// 继续执行存库，以便用户知道任务被拦截了
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

// findNextStage 查找下一个需要执行的 Stage
//  1. 获取项目关联的所有 Workflow (暂时假设一个 Project 只关联一个 Workflow)
//  2. 检查是否有上一个任务
//     Case A: 没有上一个任务 -> 返回第一个 Stage
//     Case B: 有上一个任务 -> 找到上一个 Stage，返回它的下一个 Stage
func (s *schedulerService) findNextStage(ctx context.Context, project *orcModel.Project, lastTask *orcModel.AgentTask) (*orcModel.ScanStage, error) {
	// 获取项目关联的所有 Workflow
	workflows, err := s.workflowRepo.GetWorkflowsByProjectID(ctx, uint64(project.ID))
	if err != nil {
		return nil, err
	}
	if len(workflows) == 0 {
		return nil, nil
	}

	// 暂时假设一个 Project 只关联一个 Workflow (简化逻辑)
	// 实际逻辑可能需要处理多个 Workflow 的顺序
	workflow := workflows[0]

	// 获取 Workflow 下所有 Stages
	stages, err := s.stageRepo.GetStagesByWorkflowID(ctx, uint64(workflow.ID))
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return nil, nil
	}

	// 如果没有上一个任务，说明是刚开始，返回第一个 Stage
	if lastTask == nil {
		return s.getFirstStage(stages), nil
	}

	// 如果有上一个任务，找到上一个 Stage，返回它的下一个 Stage
	return s.getNextStage(stages, lastTask.StageID), nil
}

// getFirstStage 获取第一个 Stage
//  1. 遍历所有 Stage，找到 order 最小的 Stage
func (s *schedulerService) getFirstStage(stages []*orcModel.ScanStage) *orcModel.ScanStage {
	// 寻找 order 最小的 stage
	if len(stages) == 0 {
		return nil
	}
	first := stages[0]
	for _, stage := range stages {
		if stage.StageOrder < first.StageOrder {
			first = stage
		}
	}
	return first
}

// getNextStage 获取下一个 Stage
//  1. 找到当前 stage 的 order
//  2. 遍历所有 Stage，找到 order 比 currentOrder 大且最小的 Stage
func (s *schedulerService) getNextStage(stages []*orcModel.ScanStage, currentStageID uint64) *orcModel.ScanStage {
	// 1. 找到当前 stage 的 order
	var currentOrder int
	found := false
	for _, stage := range stages {
		if uint64(stage.ID) == currentStageID {
			currentOrder = stage.StageOrder
			found = true
			break
		}
	}
	if !found {
		return nil // 异常：当前任务的 StageID 不存在于 Workflow 中
	}

	// 2. 找到 order 比 currentOrder 大且最小的 stage
	var nextStage *orcModel.ScanStage
	for _, stage := range stages {
		if stage.StageOrder > currentOrder {
			if nextStage == nil || stage.StageOrder < nextStage.StageOrder {
				nextStage = stage
			}
		}
	}
	return nextStage
}
