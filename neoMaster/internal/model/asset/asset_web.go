package asset

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// AssetWeb Web资产表
// 存储Web服务、站点、API等信息
type AssetWeb struct {
	basemodel.BaseModel

	HostID     uint64     `json:"host_id" gorm:"index;default:0;comment:主机ID(可选)"`
	Domain     string     `json:"domain" gorm:"size:255;index;comment:域名(可选)"`
	URL        string     `json:"url" gorm:"size:2048;comment:完整的URL地址"` // URL可能很长
	AssetType  string     `json:"asset_type" gorm:"size:50;default:'web';comment:资产类型(web/api/domain)"`
	TechStack  string     `json:"tech_stack" gorm:"type:json;comment:技术栈信息(JSON)"`
	Status     string     `json:"status" gorm:"size:20;default:'active';comment:资产状态"`
	Tags       string     `json:"tags" gorm:"type:json;comment:标签信息(JSON)"`
	BasicInfo  string     `json:"basic_info" gorm:"type:json;comment:基础Web信息(Title/Headers等)(JSON)"`
	ScanLevel  int        `json:"scan_level" gorm:"default:0;comment:扫描级别"`
	LastSeenAt *time.Time `json:"last_seen_at" gorm:"comment:最后发现时间"`
}

// TableName 定义数据库表名
func (AssetWeb) TableName() string {
	return "asset_webs"
}

// AssetWebDetail Web详细信息表
// 存储爬虫深度抓取的结果，与AssetWeb分离以减少主表大小
type AssetWebDetail struct {
	basemodel.BaseModel

	AssetWebID      uint64     `json:"asset_web_id" gorm:"uniqueIndex;not null;comment:关联AssetWeb表"`
	CrawlTime       *time.Time `json:"crawl_time" gorm:"comment:爬取时间"`
	CrawlStatus     string     `json:"crawl_status" gorm:"size:20;comment:爬取状态"`
	ErrorMessage    string     `json:"error_message" gorm:"type:text;comment:错误信息"`
	ContentDetails  string     `json:"content_details" gorm:"type:json;comment:详细内容信息(Body/Links等)(JSON)"`
	LoginIndicators string     `json:"login_indicators" gorm:"type:text;comment:登录相关标识"`
	Cookies         string     `json:"cookies" gorm:"type:text;comment:Cookie信息"`
	Screenshot      string     `json:"screenshot" gorm:"type:longtext;comment:页面截图(Base64或路径)"` // 可能很大，用longtext
}

// TableName 定义数据库表名
func (AssetWebDetail) TableName() string {
	return "asset_web_details"
}
