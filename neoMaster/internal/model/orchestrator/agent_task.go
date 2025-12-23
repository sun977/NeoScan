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

	TaskID       string `json:"task_id" gorm:"uniqueIndex;not null;size:100;comment:任务唯一标识ID"`
	ProjectID    uint64 `json:"project_id" gorm:"index;not null;comment:所属项目ID"`
	WorkflowID   uint64 `json:"workflow_id" gorm:"index;not null;comment:所属工作流ID"`
	StageID      uint64 `json:"stage_id" gorm:"index;not null;comment:所属阶段ID"`
	AgentID      string `json:"agent_id" gorm:"index;size:100;comment:执行Agent的ID"`
	Status       string `json:"status" gorm:"size:20;default:'pending';comment:任务状态(pending/assigned/running/completed/failed)"`
	Priority     int    `json:"priority" gorm:"default:0;comment:任务优先级"`
	TaskType     string `json:"task_type" gorm:"size:20;default:'tool';comment:任务类型"`
	TaskCategory string `json:"task_category" gorm:"size:20;default:'agent';comment:任务分类(agent/system)"` // agent: 普通任务(通过Agent执行); system: 系统任务(localAgent)

	// 任务参数
	ToolName       string `json:"tool_name" gorm:"size:100;comment:工具名称"`
	ToolParams     string `json:"tool_params" gorm:"type:text;comment:工具参数"`
	InputTarget    string `json:"input_target" gorm:"type:json;comment:输入目标(JSON)"`
	RequiredTags   string `json:"required_tags" gorm:"type:json;comment:执行所需标签(JSON)"`
	PolicySnapshot string `json:"policy_snapshot" gorm:"type:json;comment:策略快照(JSON) - 包含TargetScope和ScanStage的策略配置"` // 任务执行时的策略配置快照

	// 执行结果
	OutputResult string `json:"output_result" gorm:"type:json;comment:输出结果摘要(JSON)"` // 任务执行结果摘要 (JSON 格式) 详细结果存储在 StageResult 中
	ErrorMsg     string `json:"error_msg" gorm:"type:text;comment:错误信息"`

	// 时间记录
	AssignedAt *time.Time `json:"assigned_at" gorm:"comment:分配时间"`
	StartedAt  *time.Time `json:"started_at" gorm:"comment:开始执行时间"`
	FinishedAt *time.Time `json:"finished_at" gorm:"comment:完成时间"`
	Timeout    int        `json:"timeout" gorm:"default:3600;comment:超时时间(秒)"`

	// 重试机制
	RetryCount int `json:"retry_count" gorm:"default:0;comment:已重试次数"`
	MaxRetries int `json:"max_retries" gorm:"default:3;comment:最大重试次数"`
}

// TableName 定义表名
func (AgentTask) TableName() string {
	return "agent_tasks"
}

// task.PolicySnapshot 样例:
// {
//   "target_scope": ["192.168.1.0/24", "10.0.0.0/16"], // 项目 TargetScope 可以为空，为空时表示不限制范围
//   "target_policy": { // 扫描阶段的 target_policy 是一个 JSON 字符串，包含白名单、跳过条件等策略配置
//     "target_sources": [
//       {
//         "source_type": "file", // 来源类型：file/db/view/sql/manual/api/previous_stage【上一个阶段结果】
//         "source_value": "/path/to/targets.txt",
//         "target_type": "ip_range" // 目标类型：ip/ip_range/domain/url
//       }
//     ],
//     "whitelist_enabled": true,
//     "whitelist_sources": [ // 白名单来源/数据库/文件/手动输入
//       {
//         "source_type": "file",
//         "source_value": "/path/to/whitelist.txt" // file 对应文件路径, db 对应默认全局白名单表(相当于不设置局部白名单), manual 对应手动输入内容["192.168.1.0/24","10.0.0.0/16"]
//       }
//     ],
//     "skip_enabled": true,
//     "skip_conditions": [  // 跳过条件,列表中可添加多个条件，这里写的条件直接执行跳过动作
//       {
//         "condition_field": "device_type",
//         "operator": "equals",
//         "value": "honeypot"
//       },
//       {
//         "condition_field": "os",
//         "operator": "contains",
//         "value": "linux"
//       }
//     ]
//   }
// }

// PolicySnapshot 策略快照结构体
// 对应 Task.PolicySnapshot 字段的 JSON 结构
// PolicySnapshot 定义策略快照的结构
type PolicySnapshot struct {
	TargetScope  []string     `json:"target_scope"`  // 项目目标范围
	TargetPolicy TargetPolicy `json:"target_policy"` // 目标策略配置（注意：是对象而不是数组）
}

// TargetPolicy 定义目标策略配置
type TargetPolicy struct {
	TargetSources    []TargetSource    `json:"target_sources"`    // 目标来源配置
	WhitelistEnabled bool              `json:"whitelist_enabled"` // 是否启用白名单
	WhitelistSources []WhitelistSource `json:"whitelist_sources"` // 白名单来源配置
	SkipEnabled      bool              `json:"skip_enabled"`      // 是否启用跳过条件
	SkipConditions   []SkipCondition   `json:"skip_conditions"`   // 跳过条件配置
}

// TargetSource 定义目标来源
type TargetSource struct {
	SourceType  string `json:"source_type"`  // 来源类型：file/db/view/sql/manual/api/previous_stage
	SourceValue string `json:"source_value"` // 来源值
	TargetType  string `json:"target_type"`  // 目标类型：ip/ip_range/domain/url
}

// WhitelistSource 定义白名单来源
type WhitelistSource struct {
	SourceType  string `json:"source_type"`  // 来源类型：file/db/manual
	SourceValue string `json:"source_value"` // 来源值
}

// SkipCondition 定义跳过条件
type SkipCondition struct {
	ConditionField string `json:"condition_field"` // 条件字段
	Operator       string `json:"operator"`        // 操作符
	Value          string `json:"value"`           // 值
}
