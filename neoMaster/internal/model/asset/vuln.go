package asset

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// AssetVuln 漏洞资产表
// 存储发现的漏洞信息，通过多态关联指向 Host/Service/Web
type AssetVuln struct {
	basemodel.BaseModel

	TargetType  string     `json:"target_type" gorm:"size:50;index;not null;comment:目标类型(host/service/web/api)"`
	TargetRefID uint64     `json:"target_ref_id" gorm:"index;not null;comment:指向对应实体的ID"`
	CVE         string     `json:"cve" gorm:"size:50;index;comment:CVE编号"`
	IDAlias     string     `json:"id_alias" gorm:"size:100;comment:漏洞标识(自定义ID或扫描器ID)"`
	Severity    string     `json:"severity" gorm:"size:20;default:'medium';comment:严重程度(low/medium/high/critical)"`
	Confidence  float64    `json:"confidence" gorm:"default:0;comment:置信度(0-1)"`
	Evidence    string     `json:"evidence" gorm:"type:json;comment:原始证据(JSON)"`
	Attributes  string     `json:"attributes" gorm:"type:json;comment:结构化属性(JSON)"`
	FirstSeenAt *time.Time `json:"first_seen_at" gorm:"comment:首次发现时间"`
	LastSeenAt  *time.Time `json:"last_seen_at" gorm:"comment:最后发现时间"`
	Status      string     `json:"status" gorm:"size:20;default:'open';comment:状态(open/confirmed/resolved/ignored)"`
}

// TableName 定义数据库表名
func (AssetVuln) TableName() string {
	return "asset_vulns"
}
