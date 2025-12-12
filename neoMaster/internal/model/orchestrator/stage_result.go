package orchestrator

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// StageResult 扫描结果表
// 存储各个阶段产生的原始结果数据，量大，建议分区
// ResultType 结果类型有枚举定义
// ScanStage(定义) -- AgentTask(执行)[下发给Agent节点] -- ScanResult(结果)[Agent节点返回]
type StageResult struct {
	basemodel.BaseModel

	ProjectID        uint64    `json:"project_id" gorm:"index;not null;comment:所属项目ID"` // 所属项目ID 为了性能和管理选择让 项目ID 冗余, 方便查询
	WorkflowID       uint64    `json:"workflow_id" gorm:"index;not null;comment:所属工作流ID"`
	StageID          uint64    `json:"stage_id" gorm:"index;not null;comment:阶段ID"`
	AgentID          string    `json:"agent_id" gorm:"index;size:100;comment:执行扫描的AgentID"`
	ResultType       string    `json:"result_type" gorm:"size:50;comment:结果类型枚举(ipAlive/serviceScan/PocScan等)"`
	TargetType       string    `json:"target_type" gorm:"size:50;comment:目标类型(ip/domain/url)"`
	TargetValue      string    `json:"target_value" gorm:"size:2048;comment:目标值"`
	Attributes       string    `json:"attributes" gorm:"type:json;comment:结构化属性(JSON)"` // 存储扫描结果的详细信息 (JSON 格式)
	Evidence         string    `json:"evidence" gorm:"type:json;comment:原始证据(JSON)"`    // 存储原始扫描证据 (JSON 格式)
	ProducedAt       time.Time `json:"produced_at" gorm:"comment:产生时间"`
	Producer         string    `json:"producer" gorm:"size:100;comment:工具标识与版本"`
	OutputConfigHash string    `json:"output_config_hash" gorm:"size:64;comment:输出配置指纹"`
	OutputActions    string    `json:"output_actions" gorm:"type:json;comment:实际执行的轻量动作摘要(JSON)"`
}

// TableName 定义数据库表名
func (StageResult) TableName() string {
	return "stage_results"
}
