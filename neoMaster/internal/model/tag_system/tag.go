// 标签系统模型
// 定义标签系统的数据库模型
package tag_system

import (
	"neomaster/internal/model/basemodel"
)

// SysTag 标签定义表 (树状结构)
type SysTag struct {
	basemodel.BaseModel
	Name        string `json:"name" gorm:"size:100;not null;index:idx_parent_name,unique"`
	ParentID    uint64 `json:"parent_id" gorm:"default:0;index:idx_parent_name,unique"` // 0表示根节点
	Path        string `json:"path" gorm:"size:700;index"`                              // 物理路径 (Materialized Path) e.g. "/1/5/10/"
	Level       int    `json:"level" gorm:"default:0"`                                  // 层级深度
	Color       string `json:"color" gorm:"size:7"`                                     // 标签颜色 (HEX)
	Category    string `json:"category" gorm:"size:50;index"`                           // 业务分类 (asset, vul, user...) - 仅作UI分组，逻辑上通用
	Description string `json:"description" gorm:"size:255"`
}

func (SysTag) TableName() string {
	return "sys_tags"
}

// SysMatchRule 自动打标规则表
type SysMatchRule struct {
	basemodel.BaseModel
	TagID      uint64 `json:"tag_id" gorm:"index;not null"`
	Name       string `json:"name" gorm:"size:100"`
	EntityType string `json:"entity_type" gorm:"size:50;index;not null"` // host, web, user...
	Priority   int    `json:"priority" gorm:"default:0"`                 // 优先级 (越大越先匹配)
	RuleJSON   string `json:"rule_json" gorm:"type:text;not null"`       // JSON格式的匹配规则 (matcher.MatchRule)
	IsEnabled  bool   `json:"is_enabled" gorm:"default:true;index"`
}

func (SysMatchRule) TableName() string {
	return "sys_match_rules"
}

// SysEntityTag 实体-标签关联表 (Many-to-Many)
// 注意：这里使用 ID 作为主键，而不是联合主键，方便 GORM 管理
type SysEntityTag struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	EntityType string `json:"entity_type" gorm:"size:50;index:idx_entity;not null"`
	EntityID   string `json:"entity_id" gorm:"size:100;index:idx_entity;not null"` // 统一使用字符串ID
	TagID      uint64 `json:"tag_id" gorm:"index;not null"`
	Source     string `json:"source" gorm:"size:50;default:'manual'"` // manual, auto, api
	RuleID     uint64 `json:"rule_id" gorm:"default:0"`               // 如果是 auto，记录命中的规则ID
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

func (SysEntityTag) TableName() string {
	return "sys_entity_tags"
}
