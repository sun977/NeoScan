package asset

import (
	"neomaster/internal/model/basemodel"
)

// AssetETLError ETL 错误记录
// 用于记录在 ETL 过程中解析 (Mapper) 或入库 (Merger) 失败的原始数据
// 充当 "死信队列" 的角色，支持后续手动重试或分析
type AssetETLError struct {
	basemodel.BaseModel
	ProjectID  uint64 `gorm:"index;not null" json:"project_id"`
	TaskID     string `gorm:"index;size:64;not null" json:"task_id"`
	ResultType string `gorm:"size:64;not null" json:"result_type"`
	RawData    string `gorm:"type:text" json:"raw_data"`                 // 原始 StageResult JSON
	ErrorMsg   string `gorm:"type:text" json:"error_msg"`                // 错误堆栈信息
	ErrorStage string `gorm:"size:20" json:"error_stage"`                // 出错阶段: mapper, merger
	RetryCount int    `gorm:"default:0" json:"retry_count"`              // 重试次数
	Status     string `gorm:"size:20;default:'new';index" json:"status"` // new, retrying, resolved, ignored
}

// TableName 指定表名
func (AssetETLError) TableName() string {
	return "asset_etl_errors"
}
