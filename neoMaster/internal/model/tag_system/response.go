package tag_system

import (
	"time"

	"neomaster/internal/pkg/matcher"
)

// TagResponse 标签详情响应
type TagResponse struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	ParentID    uint64 `json:"parent_id"`
	Path        string `json:"path"`
	Level       int    `json:"level"`
	Color       string `json:"color"`
	Category    string `json:"category"`
	Description string `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RuleResponse 规则详情响应
type RuleResponse struct {
	ID         uint64            `json:"id"`
	TagID      uint64            `json:"tag_id"`
	Name       string            `json:"name"`
	EntityType string            `json:"entity_type"`
	Priority   int               `json:"priority"`
	RuleJSON   matcher.MatchRule `json:"rule_json"` // 解析后的JSON对象
	IsEnabled  bool              `json:"is_enabled"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// EntityTagResponse 实体关联标签响应
type EntityTagResponse struct {
	TagID   uint64 `json:"tag_id"`
	TagName string `json:"tag_name"`
	Source  string `json:"source"`
	RuleID  uint64 `json:"rule_id"`
}
