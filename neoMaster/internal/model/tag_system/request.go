package tag_system

import "neomaster/internal/pkg/matcher"

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name        string `json:"name" validate:"required"` // 标签名称
	ParentID    uint64 `json:"parent_id"`                // 父标签ID
	Color       string `json:"color"`                    // 标签颜色
	Category    string `json:"category"`                 // 业务分类
	Description string `json:"description"`              // 描述
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name        string `json:"name"`        // 标签名称
	Color       string `json:"color"`       // 标签颜色
	Description string `json:"description"` // 描述
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	TagID      uint64            `json:"tag_id" validate:"required"`      // 关联标签ID
	Name       string            `json:"name" validate:"required"`        // 规则名称
	EntityType string            `json:"entity_type" validate:"required"` // 实体类型
	Priority   int               `json:"priority"`                        // 优先级
	RuleJSON   matcher.MatchRule `json:"rule_json" validate:"required"`   // 匹配规则 (JSON对象)
	IsEnabled  bool              `json:"is_enabled"`                      // 是否启用
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name      string             `json:"name"`       // 规则名称
	Priority  int                `json:"priority"`   // 优先级
	RuleJSON  *matcher.MatchRule `json:"rule_json"`  // 匹配规则
	IsEnabled *bool              `json:"is_enabled"` // 是否启用 (指针用于区分是否更新)
}

// AutoTagRequest 自动打标请求 (通常由内部调用，但也可以作为API暴露)
type AutoTagRequest struct {
	EntityType string                 `json:"entity_type" validate:"required"`
	EntityID   string                 `json:"entity_id" validate:"required"`
	Attributes map[string]interface{} `json:"attributes" validate:"required"`
}

// ManualTagRequest 手动打标请求
type ManualTagRequest struct {
	EntityType string   `json:"entity_type" validate:"required"`
	EntityID   string   `json:"entity_id" validate:"required"`
	TagIDs     []uint64 `json:"tag_ids" validate:"required"`
}

// ListTagsRequest 获取标签列表请求
type ListTagsRequest struct {
	ParentID *uint64 `form:"parent_id"` // 父标签ID (可选)
	Keyword  string  `form:"keyword"`   // 搜索关键字 (名称/描述)
	Category string  `form:"category"`  // 业务分类 (可选) (asset, vul, user...)
	Page     int     `form:"page"`      // 页码
	PageSize int     `form:"page_size"` // 每页数量
}

// ListRulesRequest 获取规则列表请求
type ListRulesRequest struct {
	EntityType string `form:"entity_type"` // 实体类型
	TagID      uint64 `form:"tag_id"`      // 标签ID
	Keyword    string `form:"keyword"`     // 搜索关键字 (名称)
	IsEnabled  *bool  `form:"is_enabled"`  // 是否启用
	Page       int    `form:"page"`        // 页码
	PageSize   int    `form:"page_size"`   // 每页数量
}
