package asset

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// AssetNetworkScan 网段扫描记录表
// 记录每次针对网段的扫描任务执行情况，用于审计和历史回溯
type AssetNetworkScan struct {
	basemodel.BaseModel

	NetworkID     uint64    `json:"network_id" gorm:"index;not null;comment:网段ID"`
	AgentID       uint64    `json:"agent_id" gorm:"index;not null;comment:执行Agent ID"`
	ScanStatus    string    `json:"scan_status" gorm:"size:20;default:'pending';comment:扫描状态(pending/running/finished/failed)"`
	Round         int       `json:"round" gorm:"default:1;comment:扫描轮次"`
	ScanTool      string    `json:"scan_tool" gorm:"size:50;comment:扫描工具(nmap/masscan)"`
	ScanConfig    string    `json:"scan_config" gorm:"type:json;comment:扫描配置快照(JSON)"`
	ResultCount   int       `json:"result_count" gorm:"default:0;comment:结果数量"`
	Duration      int       `json:"duration" gorm:"default:0;comment:扫描耗时(秒)"`
	ErrorMessage  string    `json:"error_message" gorm:"type:text;comment:错误信息"`
	StartedAt     *time.Time `json:"started_at" gorm:"comment:开始时间"`
	FinishedAt    *time.Time `json:"finished_at" gorm:"comment:完成时间"`
	AssignedAt    *time.Time `json:"assigned_at" gorm:"comment:分配时间"`
	ScanResultRef string    `json:"scan_result_ref" gorm:"size:255;comment:结果引用(StageResult ID或文件路径)"`
	Note          string    `json:"note" gorm:"size:255;comment:备注"`
	RetryCount    int       `json:"retry_count" gorm:"default:0;comment:重试次数"`
}

// TableName 定义数据库表名
func (AssetNetworkScan) TableName() string {
	return "asset_network_scans"
}
