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

	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/utils"
	"neomaster/internal/service/orchestrator/policy"
)

// TaskGenerator 任务生成器接口
type TaskGenerator interface {
	// GenerateTasks 修改: 接收 []policy.Target 而不是 []string
	GenerateTasks(stage *orcModel.ScanStage, projectID uint64, targets []policy.Target) ([]*orcModel.AgentTask, error)
}

type taskGenerator struct{}

func NewTaskGenerator() TaskGenerator {
	return &taskGenerator{}
}

// GenerateTasks 根据 Stage 和目标生成任务
// Stage 阶段是 Agent 执行的最小单位。
// Stage -> Task -> Agent 执行 -> 结果返回 master -> 下一个 Stage -> Agent 执行 -> 结果返回 master -> 结果聚合落库
// 职责: 根据 ScanStage 配置和目标列表生成 AgentTask 任务
// 参数:
//   - stage: 扫描阶段配置，包含任务生成规则
//   - projectID: 项目ID，用于关联任务
//   - targets: 扫描目标列表 (完整对象，包含 Meta)
//
// 返回值:
//   - []*orcModel.AgentTask: 生成的任务列表
//   - error: 若生成任务过程中发生错误，则返回错误信息
func (g *taskGenerator) GenerateTasks(stage *orcModel.ScanStage, projectID uint64, targets []policy.Target) ([]*orcModel.AgentTask, error) {
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
	chunkSize := 50
	timeout := 3600
	if stage.PerformanceSettings != "" {
		var perf map[string]interface{}
		if err := json.Unmarshal([]byte(stage.PerformanceSettings), &perf); err == nil {
			if cs, ok := perf["chunk_size"].(float64); ok && cs > 0 {
				chunkSize = int(cs)
			}
			if to, ok := perf["timeout"].(float64); ok && to > 0 {
				timeout = int(to)
			}
		}
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
	if stage.ExecutionPolicy != "" {
		var exec map[string]interface{}
		if err := json.Unmarshal([]byte(stage.ExecutionPolicy), &exec); err == nil {
			if p, ok := exec["priority"].(float64); ok {
				priority = int(p)
			}
		}
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

		// 生成任务ID
		taskID, err := utils.GenerateUUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate task ID: %v", err)
		}

		// 2.创建任务对象并添加到任务列表中
		task := &orcModel.AgentTask{
			TaskID:       taskID,              // 任务ID
			ProjectID:    projectID,           // 项目ID
			WorkflowID:   stage.WorkflowID,    // 工作流ID
			StageID:      uint64(stage.ID),    // 阶段ID
			Status:       "pending",           // 任务状态 (pending, running, completed, failed)
			Priority:     priority,            // 任务优先级
			TaskType:     "tool",              // Explicitly set default
			ToolName:     stage.ToolName,      // 工具名称
			ToolParams:   stage.ToolParams,    // 工具参数
			InputTarget:  string(targetsJSON), // 当前批次的目标列表（JSON 字符串，包含 Meta）
			RequiredTags: "[]",                // Default empty JSON array 执行所需标签(JSON)
			OutputResult: "{}",                // Default empty JSON object 输出结果摘要(JSON)
			Timeout:      timeout,             // 任务超时时间（秒）--- stage.PerformanceSettings["timeout"]
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
