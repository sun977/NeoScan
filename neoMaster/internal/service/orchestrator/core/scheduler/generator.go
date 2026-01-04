// GenerateTasks 根据 Stage 和目标生成任务
// 任务生成器工作流程：
// 1.接收扫描阶段信息、项目ID和目标列表作为输入
// 2.解析阶段的性能设置，获取任务分块大小和超时设置
// 3.解析阶段的执行策略，获取任务优先级
// 4.将目标列表按照分块大小分割成多个批次
// 5.为每个批次创建一个任务，包含：
// - 唯一的任务ID
// - 项目、工作流和阶段的关联信息
// - 工具名称和参数
// - 当前批次的目标列表
// - 优先级和超时设置
// 6.返回所有生成的任务列表
package scheduler

import (
	"encoding/json"
	"fmt"

	"neomaster/internal/config"
	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/service/orchestrator/policy"
)

// TaskGenerator 任务生成器接口
type TaskGenerator interface {
	// GenerateTasks 修改: 接收 []policy.Target 而不是 []string
	// 增加 projectTargetScope 参数，用于注入 PolicySnapshot
	GenerateTasks(stage *orcModel.ScanStage, projectID uint64, targets []policy.Target, projectTargetScope string) ([]*orcModel.AgentTask, error)
}

type taskGenerator struct {
	cfg *config.Config
}

func NewTaskGenerator(cfg *config.Config) TaskGenerator {
	return &taskGenerator{cfg: cfg}
}

// GenerateTasks 根据 Stage 和目标生成任务
// Stage 阶段是 Agent 执行的最小单位。
// Stage -> Task -> Agent 执行 -> 结果返回 master -> 下一个 Stage -> Agent 执行 -> 结果返回 master -> 结果聚合落库
// 职责: 根据 ScanStage 配置和目标列表生成 AgentTask 任务
// 参数:
//   - stage: 扫描阶段配置，包含任务生成规则
//   - projectID: 项目ID，用于关联任务
//   - targets: 扫描目标列表 (完整对象，包含 Meta)
//   - projectTargetScope: 项目定义的目标范围 (Project.TargetScope)
//
// 返回值:
//   - []*orcModel.AgentTask: 生成的任务列表
//   - error: 若生成任务过程中发生错误，则返回错误信息
func (g *taskGenerator) GenerateTasks(stage *orcModel.ScanStage, projectID uint64, targets []policy.Target, projectTargetScope string) ([]*orcModel.AgentTask, error) {
	// 如果没有目标，则返回 nil
	if len(targets) == 0 {
		return nil, nil
	}

	// 解析 PerformanceSettings 参数
	// 	{
	//   "scan_rate": 50,        // 扫描速率（每秒发包数）
	//   "scan_depth": 1,        // 扫描深度（爬虫类工具参数）
	//   "concurrency": 50,      // 扫描并发数
	//   "process_count": 50,    // 扫描进程数
	//   "chunk_size": 50,       // 分块大小（批量处理目标数） --- 编排器-任务生成器-使用
	//   "timeout": 180,          // 超时时间（秒） --- 编排器-任务生成器-使用
	//   "retry_count": 3         // 重试次数
	// }
	// 其他参数后续添加

	// 使用全局配置作为默认值
	chunkSize := g.cfg.App.Master.Task.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 50 // 硬编码兜底
	}

	timeout := g.cfg.App.Master.Task.Timeout
	if timeout <= 0 {
		timeout = 3600 // 硬编码兜底
	}

	maxRetries := g.cfg.App.Master.Task.MaxRetries
	// 注意：maxRetries 为 0 可能表示不重试，也可能表示未配置。
	// 但由于我们有配置文件默认值 3，且 int 零值为 0。
	// 这里我们假设配置文件一定会被正确加载。如果真的配了 0，那就是不重试。
	// 为了安全起见，如果配置文件里也没配（比如旧配置），我们可以给个默认值。
	// 但这里我们信任 Config 对象的值（通常由 viper 设置默认值或 yaml 读取）。

	// PerformanceSettings 已经是结构体
	perf := stage.PerformanceSettings
	if perf.ChunkSize > 0 {
		chunkSize = perf.ChunkSize
	}
	if perf.Timeout > 0 {
		timeout = perf.Timeout
	}
	if perf.RetryCount >= 0 {
		maxRetries = perf.RetryCount
	}

	// 解析 ExecutionPolicy 参数
	// 	{
	//   "proxy_config": {                    // 代理配置
	//     "enabled": true,
	//     "proxy_type": "http",             // http/https/socks4/socks5
	//     "address": "proxy.example.com",
	//     "port": 8080,
	//     "username": "user",
	//     "password": "pass"
	//   },
	//   "priority": 1,                       // 任务优先级（1-10，默认5） 优先级越高，越先被执行
	// }
	priority := 0
	// ExecutionPolicy 已经是结构体
	if stage.ExecutionPolicy.Priority > 0 {
		priority = stage.ExecutionPolicy.Priority
	}

	// 初始化任务列表并分批处理目标
	var tasks []*orcModel.AgentTask
	//// 1.分批处理目标
	for i := 0; i < len(targets); i += chunkSize {
		end := i + chunkSize
		if end > len(targets) {
			end = len(targets)
		}
		chunk := targets[i:end]

		// 序列化目标列表为 JSON 字符串
		// 注意：现在序列化的是 []policy.Target 结构体，包含 Meta 信息
		// Agent 端需要对应更新解析逻辑，能够处理这种富结构
		targetsJSON, err := json.Marshal(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal targets: %v", err)
		}

		// 2.创建任务
		// 生成唯一任务ID (UUID)
		taskID, err := utils.GenerateUUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate task ID: %v", err)
		}

		task := &orcModel.AgentTask{
			TaskID:      taskID,
			ProjectID:   projectID,
			WorkflowID:  stage.WorkflowID,
			StageID:     uint64(stage.ID),
			ToolName:    stage.ToolName,
			ToolParams:  stage.ToolParams,
			InputTarget: string(targetsJSON),
			Status:      "pending", // 初始状态
			Priority:    priority,
			Timeout:     timeout,
			RetryCount:  0,          // 当前重试次数
			MaxRetries:  maxRetries, // 最大重试次数
			PolicySnapshot: orcModel.PolicySnapshot{
				TargetScope:  []string{projectTargetScope}, // 简化处理，暂时只支持单个 Scope，后续扩展为列表
				TargetPolicy: stage.TargetPolicy,
			},
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
