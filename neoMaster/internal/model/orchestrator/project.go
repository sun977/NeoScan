package orchestrator

import (
	"neomaster/internal/model/basemodel"
	"time"

	"gorm.io/gorm"
)

// Project 项目主表
// 定义扫描任务的顶层容器，管理调度策略和通知配置
type Project struct {
	basemodel.BaseModel

	Name           string         `json:"name" gorm:"size:100;uniqueIndex;not null;comment:项目唯一标识名"`
	DisplayName    string         `json:"display_name" gorm:"size:200;comment:显示名称"`
	Description    string         `json:"description" gorm:"type:text;comment:项目描述"`
	Status         string         `json:"status" gorm:"size:20;default:'idle';comment:运行状态(idle/running/paused/finished/error)"`
	Enabled        bool           `json:"enabled" gorm:"default:true;comment:是否启用"`
	ScheduleType   string         `json:"schedule_type" gorm:"size:20;default:'immediate';comment:调度类型(immediate/cron/api/event)"`
	CronExpr       string         `json:"cron_expr" gorm:"size:100;comment:Cron表达式"`
	ExecMode       string         `json:"exec_mode" gorm:"size:20;default:'sequential';comment:工作流执行模式(sequential/parallel)"`
	NotifyConfig   string         `json:"notify_config" gorm:"type:json;comment:通知配置聚合(JSON)"`
	ExportConfig   string         `json:"export_config" gorm:"type:json;comment:结果导出配置(JSON)"`
	ExtendedData   string         `json:"extended_data" gorm:"type:json;comment:扩展数据(JSON)"`
	Tags           string         `json:"tags" gorm:"type:json;comment:标签列表(JSON)"`
	LastExecTime   *time.Time     `json:"last_exec_time" gorm:"comment:最后一次执行开始时间"`
	LastExecID     string         `json:"last_exec_id" gorm:"size:100;comment:最后一次执行的任务ID"`
	CreatedBy      uint64         `json:"created_by" gorm:"comment:创建者UserID"`
	UpdatedBy      uint64         `json:"updated_by" gorm:"comment:更新者UserID"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index;comment:软删除时间"`
}

// TableName 定义数据库表名
func (Project) TableName() string {
	return "projects"
}
