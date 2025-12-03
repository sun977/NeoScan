package asset

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// RawAsset 原始导入记录表
// 存储从外部源(API/Excel/CMDB)导入的原始数据，作为资产清洗的源头
type RawAsset struct {
	basemodel.BaseModel

	SourceType       string    `json:"source_type" gorm:"size:50;comment:数据来源类型(api/file/manual)"`
	SourceName       string    `json:"source_name" gorm:"size:100;comment:来源名称"`
	ExternalID       string    `json:"external_id" gorm:"size:100;index;comment:外部ID"`
	Payload          string    `json:"payload" gorm:"type:json;comment:原始数据(JSON)"`
	Checksum         string    `json:"checksum" gorm:"size:64;index;comment:校验和"`
	ImportBatchID    string    `json:"import_batch_id" gorm:"size:50;index;comment:导入批次标识"`
	Priority         int       `json:"priority" gorm:"default:0;comment:处理优先级"`
	AssetMetadata    string    `json:"asset_metadata" gorm:"type:json;comment:资产元数据(JSON)"`
	Tags             string    `json:"tags" gorm:"type:json;comment:标签(JSON)"`
	ProcessingConfig string    `json:"processing_config" gorm:"type:json;comment:处理配置(JSON)"`
	ImportedAt       time.Time `json:"imported_at" gorm:"comment:导入时间"`
	NormalizeStatus  string    `json:"normalize_status" gorm:"size:20;default:'pending';comment:规范化状态"`
	NormalizeError   string    `json:"normalize_error" gorm:"type:text;comment:规范化失败原因"`
}

// TableName 定义数据库表名
func (RawAsset) TableName() string {
	return "raw_assets"
}

// RawAssetNetwork 待确认网段表
// 存储从 RawAsset 解析出或直接导入的网段信息，等待进一步确认或拆分
type RawAssetNetwork struct {
	basemodel.BaseModel

	Network          string `json:"network" gorm:"size:50;index;not null;comment:网段"`
	Name             string `json:"name" gorm:"size:100;comment:资产名称"`
	Description      string `json:"description" gorm:"size:255;comment:描述"`
	ExcludeIP        string `json:"exclude_ip" gorm:"type:text;comment:排除的IP"`
	Location         string `json:"location" gorm:"size:100;comment:地理位置"`
	SecurityZone     string `json:"security_zone" gorm:"size:50;comment:安全区域"`
	NetworkType      string `json:"network_type" gorm:"size:20;comment:网络类型"`
	Priority         int    `json:"priority" gorm:"default:0;comment:调度优先级"`
	Tags             string `json:"tags" gorm:"type:json;comment:标签(JSON)"`
	SourceType       string `json:"source_type" gorm:"size:50;comment:数据来源类型"`
	SourceIdentifier string `json:"source_identifier" gorm:"size:100;comment:来源标识"`
	Status           string `json:"status" gorm:"size:20;default:'pending';comment:状态(pending/approved/rejected)"`
	Note             string `json:"note" gorm:"type:text;comment:备注"`
	CreatedBy        string `json:"created_by" gorm:"size:100;comment:创建人"`
}

// TableName 定义数据库表名
func (RawAssetNetwork) TableName() string {
	return "raw_asset_networks"
}
