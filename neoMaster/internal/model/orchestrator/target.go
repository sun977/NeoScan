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
	Type   string     `json:"type"`   // 目标类型: ip, ip_range, domain 大网段切块, 小网段展开
	Value  string     `json:"value"`  // 目标值
	Source string     `json:"source"` // 来源标识
	Meta   TargetMeta `json:"meta"`   // 结构化元数据
}

// TargetMeta 目标元数据
// 对应设计文档: 运行时对象 (Target Object) 中的 Meta 字段
// 负责存储目标的额外信息，用于在扫描过程中传递上下文
type TargetMeta struct {
	// --- 类型特定信息 (按需填充) ---
	Network NetworkDetail `json:"network,omitempty"` // 适用于 ip_range / ip
	Domain  DomainDetail  `json:"domain,omitempty"`  // 适用于 domain
	Ports   []PortDetail  `json:"ports,omitempty"`   // 适用于 ip / domain (开放端口信息)
}

// NetworkDetail 网络/主机详情
type NetworkDetail struct {
	CIDR       string `json:"cidr,omitempty"`        // 网段 (e.g., "192.168.1.0/24")
	Location   string `json:"location,omitempty"`    // 地理位置
	SubnetMask string `json:"subnet_mask,omitempty"` // 子网掩码
	Gateway    string `json:"gateway,omitempty"`     // 网关
}

// DomainDetail 域名详情
type DomainDetail struct {
	RootDomain  string   `json:"root_domain,omitempty"`  // 根域名
	Subdomains  []string `json:"subdomains,omitempty"`   // 关联子域名
	Registrar   string   `json:"registrar,omitempty"`    // 注册商
	NameServers []string `json:"name_servers,omitempty"` // NS记录
	ResolvedIPs []string `json:"resolved_ips,omitempty"` // 解析到的IP列表
	IsWildcard  bool     `json:"is_wildcard,omitempty"`  // 是否泛解析
}

// PortDetail 端口详情
type PortDetail struct {
	Port     int               `json:"port"`
	Protocol string            `json:"protocol"`          // tcp, udp
	State    string            `json:"state"`             // open, closed, filtered
	Service  string            `json:"service,omitempty"` // http, ssh, mysql
	Version  string            `json:"version,omitempty"` // e.g., "8.0.23"
	Product  string            `json:"product,omitempty"` // e.g., "MySQL"
	Banner   string            `json:"banner,omitempty"`  // 原始Banner信息
	Extra    map[string]string `json:"extra,omitempty"`   // 漏洞探测或其他脚本输出
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
