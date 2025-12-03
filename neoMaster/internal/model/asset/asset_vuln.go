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
	Status      string     `json:"status" gorm:"size:20;default:'open';comment:状态(open/confirmed/resolved/ignored/false_positive)"` // 目标的漏洞状态

	// 验证流程字段 (Workflow Support)
	VerifyStatus string     `json:"verify_status" gorm:"size:20;default:'not_verified';comment:验证过程状态(not_verified/queued/verifying/completed)"`
	VerifiedBy   string     `json:"verified_by" gorm:"size:100;comment:验证来源(manual/poc:{id}/scanner)"`
	VerifiedAt   *time.Time `json:"verified_at" gorm:"comment:验证完成时间"`
}

// TableName 定义数据库表名
func (AssetVuln) TableName() string {
	return "asset_vulns"
}

// AssetVulnPoc 漏洞验证/利用代码表
// 存储用于验证漏洞存在的具体载荷、脚本或步骤
// 这是一个独立的实体，因为：
// 1. 一个漏洞可能有多种验证方式 (HTTP Payload, Python Script, Nuclei Template)
// 2. PoC 是"方法"，漏洞是"状态"，两者生命周期不同
// 3. PoC 代码可能很大，不适合直接存入主表
// 执行器需要做：
// 1. 根据 PocType 选择合适的解释器执行 PoC 代码
// 2. 处理执行结果，更新 AssetVulnPoc 表的 Status 和 VerifiedMsg 字段
type AssetVulnPoc struct {
	basemodel.BaseModel

	VulnID      uint64     `json:"vuln_id" gorm:"index;not null;comment:关联漏洞ID"`
	PocType     string     `json:"poc_type" gorm:"size:50;comment:PoC类型(payload/script/yaml/command)"` // 标记poc执行使用哪种解释器，python/nuclei/shell等
	Name        string     `json:"name" gorm:"size:100;comment:PoC名称"`
	VerifyURL   string     `json:"verify_url" gorm:"size:2048;comment:验证目标URL(可选)"`      // 例如 http://example.com/product/view?id=100
	Content     string     `json:"content" gorm:"type:longtext;comment:PoC内容(代码/载荷/路径)"` // 具体的poc代码或路径或payload
	Description string     `json:"description" gorm:"type:text;comment:PoC详细描述"`
	Source      string     `json:"source" gorm:"size:100;comment:来源(scanner/manual/exploit-db)"`
	IsValid     bool       `json:"is_valid" gorm:"default:true;comment:PoC本身是否有效(经过测试可用的枪)"`                                        // 标注Poc本身可用状态，可用于poc有效性检测
	Status      string     `json:"status" gorm:"size:20;default:'available';comment:验证状态(available:待验证/verified:验证成功/failed:验证失败)"` // 目标的验证状态
	VerifiedAt  *time.Time `json:"verified_at" gorm:"comment:验证成功时间"`
	VerifiedMsg string     `json:"verified_msg" gorm:"size:255;comment:验证结果信息"` // 验证结果信息
	Author      string     `json:"author" gorm:"size:50;comment:作者"`
	Note        string     `json:"note" gorm:"size:255;comment:备注"`
}

// TableName 定义数据库表名
func (AssetVulnPoc) TableName() string {
	return "asset_vuln_pocs"
}
