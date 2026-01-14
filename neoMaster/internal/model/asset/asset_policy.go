package asset

import (
	"neomaster/internal/model/basemodel"
	"time"
)

// AssetWhitelist 资产白名单表
// 用于定义不需要扫描或需要特殊处理的目标
type AssetWhitelist struct {
	basemodel.BaseModel

	WhitelistName string     `json:"whitelist_name" gorm:"size:100;not null;comment:白名单名称"`
	WhitelistType string     `json:"whitelist_type" gorm:"size:50;comment:白名单类型(global/project/workflow)"`
	TargetType    string     `json:"target_type" gorm:"size:50;not null;comment:目标类型(ip/domain/url/keyword)"`
	TargetValue   string     `json:"target_value" gorm:"type:text;not null;comment:目标值"`
	Description   string     `json:"description" gorm:"size:255;comment:描述信息"`
	ValidFrom     *time.Time `json:"valid_from" gorm:"comment:生效开始时间"`
	ValidTo       *time.Time `json:"valid_to" gorm:"comment:生效结束时间"`
	CreatedBy     string     `json:"created_by" gorm:"size:100;comment:创建人"`
	Scope         string     `json:"scope" gorm:"type:json;comment:作用域配置"`
	Enabled       bool       `json:"enabled" gorm:"default:true;comment:是否启用"`
	Note          string     `json:"note" gorm:"type:text;comment:备注信息"`
}

// TableName 定义数据库表名
func (AssetWhitelist) TableName() string {
	return "asset_whitelists"
}

// AssetSkipPolicy 资产跳过策略表
// 用于定义在特定条件下跳过扫描的规则
type AssetSkipPolicy struct {
	basemodel.BaseModel

	PolicyName     string     `json:"policy_name" gorm:"size:100;not null;comment:策略名称"`
	PolicyType     string     `json:"policy_type" gorm:"size:50;comment:策略类型"`
	Description    string     `json:"description" gorm:"size:255;comment:策略描述"`
	ConditionRules string     `json:"condition_rules" gorm:"type:json;comment:条件规则"`
	ActionConfig   string     `json:"action_config" gorm:"type:json;comment:动作配置"`
	Scope          string     `json:"scope" gorm:"type:json;comment:作用域配置"`
	Priority       int        `json:"priority" gorm:"default:0;comment:优先级"`
	Enabled        bool       `json:"enabled" gorm:"default:true;comment:是否启用"`
	CreatedBy      string     `json:"created_by" gorm:"size:100;comment:创建人"`
	ValidFrom      *time.Time `json:"valid_from" gorm:"comment:生效开始时间"`
	ValidTo        *time.Time `json:"valid_to" gorm:"comment:生效结束时间"`
	Note           string     `json:"note" gorm:"type:text;comment:备注信息"`
}

// TableName 定义数据库表名
func (AssetSkipPolicy) TableName() string {
	return "asset_skip_policies"
}
