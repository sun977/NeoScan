package asset

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// AssetHost 主机资产表
// 存储发现的存活主机信息
type AssetHost struct {
	basemodel.BaseModel

	IP             string     `json:"ip" gorm:"column:ip;size:50;uniqueIndex;not null;comment:IP地址"`
	Hostname       string     `json:"hostname" gorm:"size:200;comment:主机名"`
	OS             string     `json:"os" gorm:"size:100;comment:操作系统"`
	Tags           string     `json:"tags" gorm:"type:json;comment:标签(JSON)"`
	LastSeenAt     *time.Time `json:"last_seen_at" gorm:"comment:最后发现时间"`
	SourceStageIDs string     `json:"source_stage_ids" gorm:"type:json;comment:来源阶段ID列表(JSON)"`
}

// TableName 定义数据库表名
func (AssetHost) TableName() string {
	return "asset_hosts"
}

// AssetService 服务资产表
// 存储主机上开放的端口和服务信息
type AssetService struct {
	basemodel.BaseModel

	HostID      uint64     `json:"host_id" gorm:"index;not null;comment:主机ID"`
	Port        int        `json:"port" gorm:"not null;comment:端口号"`
	Proto       string     `json:"proto" gorm:"size:10;default:'tcp';comment:协议(tcp/udp)"`
	Name        string     `json:"name" gorm:"size:100;comment:服务名称"`
	Product     string     `json:"product" gorm:"size:100;comment:产品名称"` // 新增
	Version     string     `json:"version" gorm:"size:100;comment:服务版本"`
	CPE         string     `json:"cpe" gorm:"size:255;comment:CPE标识"`
	Banner      string     `json:"banner" gorm:"size:2048;comment:服务横幅信息"` // 新增
	Fingerprint string     `json:"fingerprint" gorm:"type:json;comment:指纹信息(JSON)"`
	AssetType   string     `json:"asset_type" gorm:"size:50;default:'service';comment:资产类型(service/database/container)"`
	Tags        string     `json:"tags" gorm:"type:json;comment:标签(JSON)"`
	LastSeenAt  *time.Time `json:"last_seen_at" gorm:"comment:最后发现时间"`
}

// TableName 定义数据库表名
func (AssetService) TableName() string {
	return "asset_services"
}
