package scheduler

import (
	"context"
	"encoding/json"
	"time"

	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	agentRepo "neomaster/internal/repo/mysql/agent"
	orcRepo "neomaster/internal/repo/mysql/orchestrator"

	"github.com/robfig/cron/v3"
)

// SchedulerService 调度引擎服务接口
type SchedulerService interface {
	Start(ctx context.Context)
	Stop()
}

type schedulerService struct {
	projectRepo   *orcRepo.ProjectRepository
	workflowRepo  *orcRepo.WorkflowRepository
	stageRepo     *orcRepo.ScanStageRepository
	taskRepo      orcRepo.TaskRepository
	agentRepo     agentRepo.AgentRepository
	taskGenerator TaskGenerator

	stopChan chan struct{}
	interval time.Duration
}

// NewSchedulerService 创建调度引擎服务
func NewSchedulerService(
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
		projectRepo:   projectRepo,
		workflowRepo:  workflowRepo,
		stageRepo:     stageRepo,
		taskRepo:      taskRepo,
		agentRepo:     agentRepo,
		taskGenerator: NewTaskGenerator(),
		stopChan:      make(chan struct{}),
		interval:      interval,
	}
}

// Start 启动调度引擎
func (s *schedulerService) Start(ctx context.Context) {
	logger.LogInfo("Starting Scheduler Engine...", "", 0, "", "service.scheduler.Start", "", map[string]interface{}{
		"interval": s.interval.String(),
	})
	go s.loop(ctx)
}

// Stop 停止调度引擎
func (s *schedulerService) Stop() {
	close(s.stopChan)
	logger.LogInfo("Scheduler Engine Stopped", "", 0, "", "service.scheduler.Stop", "", nil)
}

// loop 调度循环
func (s *schedulerService) loop(ctx context.Context) {
	ticker := time.NewTicker(s.interval) // 使用配置的轮询间隔
	defer ticker.Stop()

	// 立即执行一次调度
	s.schedule(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.schedule(ctx)
		}
	}
}

// schedule 执行单次调度逻辑
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
	// 解析目标 (Target)
	// TODO: 这里暂时简化，假设 target 就在 project.ExtendedData 或者需要从哪里获取
	// 实际上，Target 应该是 Project 的一部分，这里我们假设 Project 有 Target 字段或者从 Input 获取
	// 暂时 mock 一个 target，后续需要完善 Project 模型支持 Target 定义
	var targets []string
	// 尝试从 Project 的 ExtendedData 中解析 targets，如果没有则用默认的
	if project.ExtendedData != "" {
		var data map[string]interface{}
		if json.Unmarshal([]byte(project.ExtendedData), &data) == nil {
			if t, ok := data["targets"].([]interface{}); ok {
				for _, v := range t {
					if s, ok := v.(string); ok {
						targets = append(targets, s)
					}
				}
			}
		}
	}
	if len(targets) == 0 {
		targets = []string{"127.0.0.1"} // Fallback
	}

	newTasks, err := s.taskGenerator.GenerateTasks(nextStage, uint64(project.ID), targets)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.scheduler.processProject", "INTERNAL", loggerFields)
		return
	}

	// 保存任务到数据库
	for _, task := range newTasks {
		if err := s.taskRepo.CreateTask(ctx, task); err != nil {
			logger.LogError(err, "", 0, "", "service.scheduler.processProject", "REPO", loggerFields)
			continue
		}
		logger.LogInfo("Generated new task", "", 0, "", "service.scheduler.processProject", "", map[string]interface{}{
			"task_id":  task.TaskID,
			"stage_id": task.StageID,
			"tool":     task.ToolName,
		})
	}
}

// findNextStage 查找下一个需要执行的 Stage
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
