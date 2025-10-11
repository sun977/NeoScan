package rule_engine

import (
	"time"
)

// Rule 扫描规则定义
type Rule struct {
	ID          string            `json:"id"`          // 规则ID
	Name        string            `json:"name"`        // 规则名称
	Description string            `json:"description"` // 规则描述
	Category    string            `json:"category"`    // 规则分类
	Severity    string            `json:"severity"`    // 严重程度: critical, high, medium, low, info
	Priority    int               `json:"priority"`    // 优先级 (1-100, 数字越大优先级越高)
	Enabled     bool              `json:"enabled"`     // 是否启用
	Conditions  []Condition       `json:"conditions"`  // 条件列表
	Actions     []Action          `json:"actions"`     // 动作列表
	Tags        []string          `json:"tags"`        // 标签
	Metadata    map[string]string `json:"metadata"`    // 元数据
	CreatedAt   time.Time         `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time         `json:"updated_at"`  // 更新时间
}

// Condition 规则条件
type Condition struct {
	Field    string      `json:"field"`    // 字段名
	Operator string      `json:"operator"` // 操作符: eq, ne, gt, lt, gte, lte, in, not_in, contains, not_contains, regex, not_regex
	Value    interface{} `json:"value"`    // 期望值
	Type     string      `json:"type"`     // 值类型: string, number, boolean, array
}

// Action 规则动作
type Action struct {
	Type       string                 `json:"type"`       // 动作类型: log, alert, block, modify, execute
	Parameters map[string]interface{} `json:"parameters"` // 动作参数
}

// RuleContext 规则执行上下文
type RuleContext struct {
	Data      map[string]interface{} `json:"data"`      // 输入数据
	Variables map[string]interface{} `json:"variables"` // 变量
	Metadata  map[string]interface{} `json:"metadata"`  // 元数据
}

// RuleResult 规则执行结果
type RuleResult struct {
	RuleID    string                 `json:"rule_id"`    // 规则ID
	Matched   bool                   `json:"matched"`    // 是否匹配
	Actions   []ActionResult         `json:"actions"`    // 执行的动作结果
	Message   string                 `json:"message"`    // 结果消息
	Metadata  map[string]interface{} `json:"metadata"`   // 结果元数据
	Timestamp time.Time              `json:"timestamp"`  // 执行时间
}

// ActionResult 动作执行结果
type ActionResult struct {
	Type      string                 `json:"type"`      // 动作类型
	Success   bool                   `json:"success"`   // 是否成功
	Message   string                 `json:"message"`   // 结果消息
	Data      map[string]interface{} `json:"data"`      // 结果数据
	Error     string                 `json:"error"`     // 错误信息
	Timestamp time.Time              `json:"timestamp"` // 执行时间
}

// BatchRuleResult 批量规则执行结果
type BatchRuleResult struct {
	Results   []RuleResult `json:"results"`   // 规则结果列表
	Total     int          `json:"total"`     // 总规则数
	Matched   int          `json:"matched"`   // 匹配规则数
	Failed    int          `json:"failed"`    // 失败规则数
	Duration  time.Duration `json:"duration"` // 执行耗时
	Timestamp time.Time    `json:"timestamp"` // 执行时间
}

// RuleEngineConfig 规则引擎配置
type RuleEngineConfig struct {
	MaxCacheSize     int           `json:"max_cache_size"`     // 最大缓存大小
	CacheTTL         time.Duration `json:"cache_ttl"`          // 缓存过期时间
	MaxConcurrency   int           `json:"max_concurrency"`    // 最大并发数
	TimeoutDuration  time.Duration `json:"timeout_duration"`   // 超时时间
	EnableMetrics    bool          `json:"enable_metrics"`     // 是否启用指标
	EnableDebugLog   bool          `json:"enable_debug_log"`   // 是否启用调试日志
	RuleLoadPath     string        `json:"rule_load_path"`     // 规则加载路径
	AutoReload       bool          `json:"auto_reload"`        // 是否自动重载
	ReloadInterval   time.Duration `json:"reload_interval"`    // 重载间隔
}

// RuleEngineMetrics 规则引擎指标
type RuleEngineMetrics struct {
	TotalRules       int64         `json:"total_rules"`       // 总规则数
	EnabledRules     int64         `json:"enabled_rules"`     // 启用规则数
	ActiveRules      int64         `json:"active_rules"`      // 活跃规则数（已启用且正在使用的规则）
	TotalExecutions  int64         `json:"total_executions"`  // 总执行次数
	SuccessfulRuns   int64         `json:"successful_runs"`   // 成功执行次数
	FailedRuns       int64         `json:"failed_runs"`       // 失败执行次数
	AverageLatency   time.Duration `json:"average_latency"`   // 平均延迟
	CacheHitRate     float64       `json:"cache_hit_rate"`    // 缓存命中率
	LastExecutionAt  time.Time     `json:"last_execution_at"` // 最后执行时间
}

// OperatorType 操作符类型常量
const (
	OpEqual        = "eq"           // 等于
	OpNotEqual     = "ne"           // 不等于
	OpGreater      = "gt"           // 大于
	OpLess         = "lt"           // 小于
	OpGreaterEqual = "gte"          // 大于等于
	OpLessEqual    = "lte"          // 小于等于
	OpIn           = "in"           // 包含于
	OpNotIn        = "not_in"       // 不包含于
	OpContains     = "contains"     // 包含
	OpNotContains  = "not_contains" // 不包含
	OpRegex        = "regex"        // 正则匹配
	OpNotRegex     = "not_regex"    // 正则不匹配
)

// ActionType 动作类型常量
const (
	ActionLog     = "log"     // 记录日志
	ActionAlert   = "alert"   // 发送告警
	ActionBlock   = "block"   // 阻止操作
	ActionModify  = "modify"  // 修改数据
	ActionExecute = "execute" // 执行命令
)

// SeverityLevel 严重程度常量
const (
	SeverityCritical = "critical" // 严重
	SeverityHigh     = "high"     // 高
	SeverityMedium   = "medium"   // 中
	SeverityLow      = "low"      // 低
	SeverityInfo     = "info"     // 信息
)

// ValueType 值类型常量
const (
	TypeString  = "string"  // 字符串
	TypeNumber  = "number"  // 数字
	TypeBoolean = "boolean" // 布尔值
	TypeArray   = "array"   // 数组
)