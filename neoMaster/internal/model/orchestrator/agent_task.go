package orchestrator

import (
	"time"

	"neomaster/internal/model/basemodel"
)

// AgentTask Agent任务实体
// 记录 Master 分发给 Agent 的具体执行任务
// ScanStage(定义) -- AgentTask(执行)[下发给Agent节点] -- ScanResult(结果)[Agent节点返回]
// AgentTask.OutputResult 只返回任务执行结果摘要 (JSON 格式)，详细结果存储在 StageResult 中
// AgentTask 是轻量的控制流
// StageResult 是重量的数据流，存储具体结果数据
type AgentTask struct {
	basemodel.BaseModel

	TaskID     string `json:"task_id" gorm:"uniqueIndex;not null;size:100;comment:任务唯一标识ID"`
	ProjectID  uint64 `json:"project_id" gorm:"index;not null;comment:所属项目ID"`
	WorkflowID uint64 `json:"workflow_id" gorm:"index;not null;comment:所属工作流ID"`
	StageID    uint64 `json:"stage_id" gorm:"index;not null;comment:所属阶段ID"`
	AgentID    string `json:"agent_id" gorm:"index;size:100;comment:执行Agent的ID"`
	Status     string `json:"status" gorm:"size:20;default:'pending';comment:任务状态(pending/assigned/running/completed/failed)"`
	Priority   int    `json:"priority" gorm:"default:0;comment:任务优先级"`
	TaskType   string `json:"task_type" gorm:"size:20;default:'tool';comment:任务类型"`

	// 任务参数
	ToolName     string `json:"tool_name" gorm:"size:100;comment:工具名称"`
	ToolParams   string `json:"tool_params" gorm:"type:text;comment:工具参数"`
	InputTarget  string `json:"input_target" gorm:"type:json;comment:输入目标(JSON)"`
	RequiredTags string `json:"required_tags" gorm:"type:json;comment:执行所需标签(JSON)"`

	// 执行结果
	OutputResult string `json:"output_result" gorm:"type:json;comment:输出结果摘要(JSON)"`
	ErrorMsg     string `json:"error_msg" gorm:"type:text;comment:错误信息"`

	// 时间记录
	AssignedAt *time.Time `json:"assigned_at" gorm:"comment:分配时间"`
	StartedAt  *time.Time `json:"started_at" gorm:"comment:开始执行时间"`
	FinishedAt *time.Time `json:"finished_at" gorm:"comment:完成时间"`
	Timeout    int        `json:"timeout" gorm:"default:3600;comment:超时时间(秒)"`
}

// TableName 定义表名
func (AgentTask) TableName() string {
	return "agent_tasks"
}
