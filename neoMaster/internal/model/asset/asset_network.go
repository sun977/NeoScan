package asset

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// AssetNetwork 正式网段资产表
// 存储经过清洗和拆分后的网段信息，是扫描任务调度的基本单位
type AssetNetwork struct {
	basemodel.BaseModel

	Network     string     `json:"network" gorm:"size:50;index;not null;comment:原始网段"`
	CIDR        string     `json:"cidr" gorm:"column:cidr;size:50;index;not null;comment:拆分后的网段(CIDR格式)"`
	SplitFromID uint64     `json:"split_from_id" gorm:"index;default:0;comment:拆分来源ID(指向RawAssetNetwork)"`
	SplitOrder  int        `json:"split_order" gorm:"default:0;comment:拆分顺序"`
	Round       int        `json:"round" gorm:"default:0;comment:扫描轮次"`
	NetworkType string     `json:"network_type" gorm:"size:20;default:'internal';comment:网络类型(internal/external/dmz)"`
	Priority    int        `json:"priority" gorm:"default:0;comment:调度优先级(0-100)"`
	Tags        string     `json:"tags" gorm:"type:json;comment:标签(JSON)"`
	SourceRef   string     `json:"source_ref" gorm:"size:100;comment:来源引用"`
	Status      string     `json:"status" gorm:"size:20;default:'active';comment:调度状态(active/paused/disabled)"`
	ScanStatus  string     `json:"scan_status" gorm:"size:20;default:'idle';comment:扫描状态(idle/scanning/finished)"`
	LastScanAt  *time.Time `json:"last_scan_at" gorm:"comment:最后扫描时间"`
	NextScanAt  *time.Time `json:"next_scan_at" gorm:"comment:下次扫描时间"`
	Note        string     `json:"note" gorm:"size:255;comment:备注"`
	CreatedBy   string     `json:"created_by" gorm:"size:50;comment:创建人"`
}

// TableName 定义数据库表名
func (AssetNetwork) TableName() string {
	return "asset_networks"
}
