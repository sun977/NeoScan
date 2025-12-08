package scheduler

import (
	"encoding/json"
	"fmt"

	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/utils"
)

// TaskGenerator 任务生成器接口
type TaskGenerator interface {
	GenerateTasks(stage *orcModel.ScanStage, projectID uint64, targets []string) ([]*orcModel.AgentTask, error)
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
//   - targets: 扫描目标列表，每个元素为一个目标（IP、域名等）
//
// 返回值:
//   - []*orcModel.AgentTask: 生成的任务列表
//   - error: 若生成任务过程中发生错误，则返回错误信息
func (g *taskGenerator) GenerateTasks(stage *orcModel.ScanStage, projectID uint64, targets []string) ([]*orcModel.AgentTask, error) {
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
	//   "priority": 1,                       // 任务优先级（1-10，默认5） --- 编排器-任务生成器-使用
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

	var tasks []*orcModel.AgentTask

	for i := 0; i < len(targets); i += chunkSize {
		end := i + chunkSize
		if end > len(targets) {
			end = len(targets)
		}
		chunk := targets[i:end]

		targetsJSON, err := json.Marshal(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal targets: %v", err)
		}

		taskID, err := utils.GenerateUUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate task ID: %v", err)
		}

		task := &orcModel.AgentTask{
			TaskID:       taskID,
			ProjectID:    projectID,
			WorkflowID:   stage.WorkflowID,
			StageID:      uint64(stage.ID),
			Status:       "pending",
			Priority:     priority,
			TaskType:     "tool", // Explicitly set default
			ToolName:     stage.ToolName,
			ToolParams:   stage.ToolParams,
			InputTarget:  string(targetsJSON),
			RequiredTags: "[]", // Default empty JSON array
			OutputResult: "{}", // Default empty JSON object
			Timeout:      timeout,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
