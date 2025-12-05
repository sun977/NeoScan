package asset

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// AssetUnified 统一资产视图表 (Materialized View)
// 这是一个扁平化的宽表，用于聚合 Host, Service, Web 的信息
// 主要用途：
// 1. 提供给前端进行快速列表查询和搜索
// 2. 提供给外部系统进行数据集成
// 3. 作为搜索引擎(如 ES)的数据源
//
// 注意：这是一个 Read-Model (读模型)，数据应由 Worker 异步从实体表同步而来
type AssetUnified struct {
	basemodel.BaseModel

	// --- 核心索引 ---
	ProjectID uint64 `json:"project_id" gorm:"index;not null;comment:所属项目ID"` // 用于项目数据隔离，场景：用户打开“项目A的资产列表”
	IP        string `json:"ip" gorm:"column:ip;size:50;index;not null;comment:IP地址"`
	Port      int    `json:"port" gorm:"index;not null;comment:端口号"`

	// --- Host 层信息 ---
	HostName   string `json:"host_name" gorm:"size:200;comment:主机名"`
	OS         string `json:"os" gorm:"size:100;comment:操作系统"`
	DeviceType string `json:"device_type" gorm:"size:50;comment:设备类型"`
	MacAddress string `json:"mac_address" gorm:"size:50;comment:MAC地址"`
	Location   string `json:"location" gorm:"size:100;comment:地理位置"` // 预留

	// --- Service 层信息 ---
	Protocol string `json:"protocol" gorm:"size:20;comment:协议(tcp/udp)"`
	Service  string `json:"service" gorm:"size:100;comment:服务名称(http, ssh, mysql)"`
	Product  string `json:"product" gorm:"size:100;comment:产品名称(OpenSSH, Apache)"`
	Version  string `json:"version" gorm:"size:100;comment:版本号"`
	Banner   string `json:"banner" gorm:"size:2048;comment:服务横幅信息"` // 扩大长度以容纳长 Banner

	// --- Web 层信息 (如果该端口运行 Web 服务) ---
	URL         string `json:"url" gorm:"size:2048;comment:Web服务入口URL"`
	Title       string `json:"title" gorm:"size:255;comment:网页标题"`
	StatusCode  int    `json:"status_code" gorm:"comment:HTTP状态码"`
	Component   string `json:"component" gorm:"size:255;index;comment:关键组件/CMS识别结果(WordPress, Struts2)"` // 专门用于 CMS 搜索
	TechStack   string `json:"tech_stack" gorm:"type:json;comment:详细技术栈(JSON: server, lang, framework)"`
	Fingerprint string `json:"fingerprint" gorm:"type:text;comment:关键指纹特征(JSON/Text)"`
	IsWeb       bool   `json:"is_web" gorm:"index;default:false;comment:是否为Web资产"`

	// --- 元数据 ---
	Source   string     `json:"source" gorm:"size:50;comment:数据来源(nmap/masscan/manual)"`
	SyncTime *time.Time `json:"sync_time" gorm:"comment:上次同步时间"`
}

// TableName 定义数据库表名
func (AssetUnified) TableName() string {
	return "asset_unified"
}
