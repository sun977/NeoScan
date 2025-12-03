package orchestrator

import (
	"neomaster/internal/model/basemodel"

	"gorm.io/gorm"
)

// Workflow 工作流定义表
// 定义具体的扫描逻辑流程，可被多个 Project 复用
type Workflow struct {
	basemodel.BaseModel

	Name         string         `json:"name" gorm:"size:100;uniqueIndex;not null;comment:工作流唯一标识名"`
	DisplayName  string         `json:"display_name" gorm:"size:200;comment:显示名称"`
	Version      string         `json:"version" gorm:"size:20;default:'1.0.0';comment:版本号"`
	Description  string         `json:"description" gorm:"type:text;comment:描述"`
	Enabled      bool           `json:"enabled" gorm:"default:true;comment:启用状态"`
	ExecMode     string         `json:"exec_mode" gorm:"size:20;default:'sequential';comment:阶段执行模式(sequential/parallel/dag)"`
	GlobalVars   string         `json:"global_vars" gorm:"type:json;comment:全局变量定义(JSON)"`
	PolicyConfig string         `json:"policy_config" gorm:"type:json;comment:执行策略配置(超时/重试/通知)(JSON)"`
	Tags         string         `json:"tags" gorm:"type:json;comment:标签列表(JSON)"`
	CreatedBy    uint64         `json:"created_by" gorm:"comment:创建者ID"`
	UpdatedBy    uint64         `json:"updated_by" gorm:"comment:更新者ID"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index;comment:软删除时间"`
}

// TableName 定义数据库表名
func (Workflow) TableName() string {
	return "workflows"
}

// ProjectWorkflow 项目与工作流关联表
// 多对多关系
type ProjectWorkflow struct {
	basemodel.BaseModel

	ProjectID  uint64 `json:"project_id" gorm:"primaryKey;autoIncrement:false;comment:项目ID"`
	WorkflowID uint64 `json:"workflow_id" gorm:"primaryKey;autoIncrement:false;comment:工作流ID"`
	SortOrder  int    `json:"sort_order" gorm:"default:0;comment:执行顺序"`
	// CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
}

// TableName 定义数据库表名
func (ProjectWorkflow) TableName() string {
	return "project_workflows"
}
