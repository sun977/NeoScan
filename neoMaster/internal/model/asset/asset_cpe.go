package asset

import (
	"neomaster/internal/model/basemodel"
)

// AssetCPE CPE指纹表
// 存储 Nmap 探针和正则映射规则
type AssetCPE struct {
	basemodel.BaseModel

	Name     string `json:"name" gorm:"size:255;default:'';comment:指纹名称"`
	Probe    string `json:"probe" gorm:"size:255;default:'';comment:Nmap Probe 名称 (e.g. NULL, GenericLines)"`
	MatchStr string `json:"match_str" gorm:"size:500;not null;comment:正则表达式"`                    // 正则表达式
	Vendor   string `json:"vendor" gorm:"size:255;default:'';comment:Vendor"`                    // 厂商名称
	Product  string `json:"product" gorm:"size:255;default:'';comment:Product"`                  // 产品名称
	Version  string `json:"version" gorm:"size:255;default:'';comment:Version (或 Regex 占位符 $1)"` // 版本号
	Update   string `json:"update" gorm:"size:255;default:'';comment:Update"`                    // 更新版本
	Edition  string `json:"edition" gorm:"size:255;default:'';comment:Edition"`                  // 发布版本
	Language string `json:"language" gorm:"size:255;default:'';comment:Language"`                // 语言版本
	Part     string `json:"part" gorm:"size:1;default:'a';comment:Part (a/o/h)"`                 // CPE类型 (a: Application, o: OS, h: Hardware)
	CPE      string `json:"cpe" gorm:"size:500;default:'';comment:完整 CPE 模板"`
}

// TableName 定义数据库表名
func (AssetCPE) TableName() string {
	return "asset_cpe"
}
