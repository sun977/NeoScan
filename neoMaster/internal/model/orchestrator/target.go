package orchestrator

import (
	"encoding/json"

	"neomaster/internal/pkg/matcher"
)

// Target 标准目标对象
// 对应设计文档: 运行时对象 (Target Object)
// 负责统一扫描目标结构，在不同实体之间流转
// 流转：ScanStage.TargetPolicy.TargetSource.SourceValue 列表中是 Target 结构体对象 ->
// AgentTask.InputTarget 列表中 也是 Target 结构体对象
// AgentTask.PolicySnapshot.TargetPolicy.TargetSource.SourceValue 列表中也是 Target 结构体对象
// TargetProvider 就是把不同的来源的目标，统一成 Target 结构体对象
type Target struct {
	Type   string            `json:"type"`   // 目标类型: ip, domain, url
	Value  string            `json:"value"`  // 目标值
	Source string            `json:"source"` // 来源标识
	Meta   map[string]string `json:"meta"`   // 元数据
}

// TargetPolicy 定义目标策略配置
// 统一 ScanStage.target_policy 和 TargetProvider 中的配置结构
type TargetPolicy struct {
	TargetSources    []TargetSource    `json:"target_sources"`    // 目标来源配置
	WhitelistEnabled bool              `json:"whitelist_enabled"` // 是否启用白名单
	WhitelistSources []RuleSource      `json:"whitelist_sources"` // 白名单来源配置 (统一使用 TargetSource)  --- 负责从不同来源加载白名单
	WhitelistRule    matcher.MatchRule `json:"whitelist_rule"`    // 白名单规则 (逻辑匹配，来自 TargetProvider)  --- 负责根据白名单规则判断是否跳过
	SkipEnabled      bool              `json:"skip_enabled"`      // 是否启用跳过条件
	SkipSources      []RuleSource      `json:"skip_sources"`      // 跳过条件配置 (来自 ScanStage) --- 负责从不同来源加载跳过规则
	SkipRule         matcher.MatchRule `json:"skip_rule"`         // 跳过规则 (逻辑匹配，来自 TargetProvider) --- 负责根据跳过规则判断是否跳过
}

// TargetSource 定义目标来源
// 融合了 ScanStage.TargetSource 和 TargetProvider.TargetSourceConfig 的所有字段
type TargetSource struct {
	SourceType   string          `json:"source_type"`             // 来源类型：file/db/view/sql/manual/api/previous_stage
	SourceValue  string          `json:"source_value,omitempty"`  // 来源值
	TargetType   string          `json:"target_type"`             // 目标类型：ip/ip_range/domain/url
	QueryMode    string          `json:"query_mode,omitempty"`    // table, view, sql (仅用于数据库)
	CustomSQL    string          `json:"custom_sql,omitempty"`    // custom_sql 当 query_mode 为 sql 时使用
	FilterRules  json.RawMessage `json:"filter_rules,omitempty"`  // 过滤规则【暂时json，后续考虑使用结构体】
	AuthConfig   json.RawMessage `json:"auth_config,omitempty"`   // 认证配置【暂时json，后续考虑使用结构体】
	ParserConfig json.RawMessage `json:"parser_config,omitempty"` // 解析配置【暂时json，后续考虑使用结构体】
}

// RuleSource 定义规则来源(白名单规则和跳过规则)
type RuleSource struct {
	SourceType  string `json:"source_type"`  // 来源类型：file/db/manual
	SourceValue string `json:"source_value"` // 来源值
}

// // SkipCondition 定义跳过条件
// // 来自 ScanStage 的定义
// type SkipCondition struct {
// 	ConditionField string `json:"condition_field"` // 条件字段
// 	Operator       string `json:"operator"`        // 操作符
// 	Value          string `json:"value"`           // 值
// }
