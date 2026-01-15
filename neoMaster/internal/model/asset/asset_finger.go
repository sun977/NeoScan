package asset

import (
	"neomaster/internal/model/basemodel"
)

// AssetFinger Web指纹表
// 存储 HTTP 特征指纹规则
type AssetFinger struct {
	basemodel.BaseModel

	Name       string `json:"name" gorm:"size:255;default:'';comment:指纹名称"`
	StatusCode string `json:"status_code" gorm:"size:50;default:'';comment:HTTP状态码"`
	URL        string `json:"url" gorm:"size:500;default:'';comment:URL路径"`
	Title      string `json:"title" gorm:"size:255;comment:网页标题"`
	Subtitle   string `json:"subtitle" gorm:"size:255;comment:网页副标题"`
	Footer     string `json:"footer" gorm:"size:255;comment:网页页脚"`
	Header     string `json:"header" gorm:"size:255;default:'';comment:HTTP响应头"`
	Response   string `json:"response" gorm:"size:1000;default:'';comment:HTTP响应内容"`
	Server     string `json:"server" gorm:"size:500;default:'';comment:Server头"`
	XPoweredBy string `json:"x_powered_by" gorm:"size:255;default:'';comment:X-Powered-By头"`
	Body       string `json:"body" gorm:"size:255;comment:HTTP响应体"`
	Match      string `json:"match" gorm:"size:255;comment:匹配模式(如正则)"`
	Enabled    bool   `json:"enabled" gorm:"default:true;comment:是否启用"`
	Source     string `json:"source" gorm:"size:20;default:'system';comment:来源(system/custom)"`
}

// TableName 定义数据库表名
func (AssetFinger) TableName() string {
	return "asset_finger"
}
