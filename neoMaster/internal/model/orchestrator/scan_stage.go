package orchestrator

import (
	"neomaster/internal/model/basemodel"
)

// ScanStage 扫描阶段定义表
// 定义工作流中的具体执行步骤，包含工具配置和策略
// StageType 阶段类型有枚举定义
// ScanStage(定义) -- AgentTask(执行)[下发给Agent节点] -- ScanResult(结果)[Agent节点返回]
type ScanStage struct {
	basemodel.BaseModel

	WorkflowID uint64 `json:"workflow_id" gorm:"index;not null;comment:所属工作流ID"`
	StageName  string `json:"stage_name" gorm:"size:100;comment:阶段名称"`
	StageType  string `json:"stage_type" gorm:"size:50;comment:阶段类型枚举(ipAlive/serviceScan/PocScan等)"`

	// DAG 核心字段
	Predecessors []uint64 `json:"predecessors" gorm:"serializer:json;type:json;comment:前置依赖阶段ID列表(JSON数组),为空表示起始节点"`

	// UI/低代码 专用字段
	UIConfig map[string]interface{} `json:"ui_config" gorm:"serializer:json;type:json;comment:前端UI布局配置(JSON),包含x,y坐标等"`

	ToolName            string `json:"tool_name" gorm:"size:100;comment:使用的扫描工具名称"`
	ToolParams          string `json:"tool_params" gorm:"type:text;comment:扫描工具参数"`
	TargetPolicy        string `json:"target_policy" gorm:"type:json;comment:目标策略配置(JSON)"`        // 包含目标获取方式,白名单策略和跳过策略
	ExecutionPolicy     string `json:"execution_policy" gorm:"type:json;comment:执行策略配置(JSON)"`     // 包含代理配置和优先级,决定任务的属性
	PerformanceSettings string `json:"performance_settings" gorm:"type:json;comment:性能设置配置(JSON)"` // 包含并发数,超时时间等
	OutputConfig        string `json:"output_config" gorm:"type:json;comment:输出配置(JSON)"`          // 包含结果输出方式,是否输出到下一阶段,是否输出到数据库,是否输出到文件
	NotifyConfig        string `json:"notify_config" gorm:"type:json;comment:通知配置(JSON)"`
	Enabled             bool   `json:"enabled" gorm:"default:true;comment:阶段是否启用"`
}

// TableName 定义数据库表名
func (ScanStage) TableName() string {
	return "scan_stages"
}

// TargetPolicy 目标策略结构

// ExecutionPolicy 执行策略结构

// PerformanceSettings 性能设置结构

// OutputConfig 输出配置结构

// NotifyConfig 通知配置结构
