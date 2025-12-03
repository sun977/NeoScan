package orchestrator

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// WorkflowStats 工作流运行时统计表
// 读写分离优化，独立于配置表
type WorkflowStats struct {
	basemodel.BaseModel

	WorkflowID     uint64     `json:"workflow_id" gorm:"uniqueIndex;not null;comment:工作流ID"`
	TotalExecs     int        `json:"total_execs" gorm:"default:0;comment:总执行次数"`
	SuccessExecs   int        `json:"success_execs" gorm:"default:0;comment:成功次数"`
	FailedExecs    int        `json:"failed_execs" gorm:"default:0;comment:失败次数"`
	AvgDurationMs  int        `json:"avg_duration_ms" gorm:"default:0;comment:平均执行耗时(ms)"`
	LastExecID     string     `json:"last_exec_id" gorm:"size:100;comment:最后一次执行ID"`
	LastExecStatus string     `json:"last_exec_status" gorm:"size:20;comment:最后一次执行状态"`
	LastExecTime   *time.Time `json:"last_exec_time" gorm:"comment:最后一次执行时间"`
}

// TableName 定义数据库表名
func (WorkflowStats) TableName() string {
	return "workflow_stats"
}
