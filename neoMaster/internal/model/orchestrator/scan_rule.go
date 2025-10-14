/**
 * 模型:扫描规则模型
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 扫描规则配置模型，遵循"Never break userspace"原则 - 规则变更不能破坏现有扫描
 * @func: ScanRule 结构体及相关方法
 */
package orchestrator

import (
	"encoding/json"
	"fmt"
	"neomaster/internal/model/basemodel"
	"strings"
)

// ScanRuleType 扫描规则类型枚举
type ScanRuleType string

const (
	ScanRuleTypeFilter     ScanRuleType = "filter"     // 过滤规则
	ScanRuleTypeValidation ScanRuleType = "validation" // 验证规则
	ScanRuleTypeTransform  ScanRuleType = "transform"  // 转换规则
	ScanRuleTypeAlert      ScanRuleType = "alert"      // 告警规则
	ScanRuleTypeCustom     ScanRuleType = "custom"     // 自定义规则
)

// String 实现Stringer接口
func (t ScanRuleType) String() string {
	return string(t)
}

// ScanRuleStatus 扫描规则状态枚举
type ScanRuleStatus int

const (
	ScanRuleStatusDisabled ScanRuleStatus = 0 // 禁用
	ScanRuleStatusEnabled  ScanRuleStatus = 1 // 启用
	ScanRuleStatusTesting  ScanRuleStatus = 2 // 测试中
)

// String 实现Stringer接口
func (s ScanRuleStatus) String() string {
	switch s {
	case ScanRuleStatusDisabled:
		return "disabled"
	case ScanRuleStatusEnabled:
		return "enabled"
	case ScanRuleStatusTesting:
		return "testing"
	default:
		return "unknown"
	}
}

// ScanRuleSeverity 规则严重程度枚举
type ScanRuleSeverity string

const (
	ScanRuleSeverityLow      ScanRuleSeverity = "low"      // 低
	ScanRuleSeverityMedium   ScanRuleSeverity = "medium"   // 中
	ScanRuleSeverityHigh     ScanRuleSeverity = "high"     // 高
	ScanRuleSeverityCritical ScanRuleSeverity = "critical" // 严重
)

// String 实现Stringer接口
func (s ScanRuleSeverity) String() string {
	return string(s)
}

// ScanRule 扫描规则模型
// 设计原则：规则应该是原子的、可组合的，避免复杂的嵌套逻辑
type ScanRule struct {
	// 继承基础模型
	basemodel.BaseModel

	// 规则基本信息
	Name        string       `json:"name" gorm:"uniqueIndex;not null;size:100;comment:规则名称，唯一" validate:"required,min=1,max=100"`
	DisplayName string       `json:"display_name" gorm:"size:200;comment:规则显示名称"`
	Description string       `json:"description" gorm:"type:text;comment:规则描述"`
	Type        ScanRuleType `json:"type" gorm:"size:20;not null;comment:规则类型" validate:"required"`
	Category    string       `json:"category" gorm:"size:50;comment:规则分类"`

	// 规则定义 - 使用简单的JSON表达式，避免复杂的DSL
	Condition  string `json:"condition" gorm:"type:text;not null;comment:规则条件表达式" validate:"required"`
	Action     string `json:"action" gorm:"type:text;comment:规则动作定义"`
	Parameters string `json:"parameters" gorm:"type:json;comment:规则参数配置"`

	// 适用范围
	ApplicableTools string `json:"applicable_tools" gorm:"type:text;comment:适用的扫描工具，逗号分隔"`
	TargetTypes     string `json:"target_types" gorm:"type:text;comment:适用的目标类型，逗号分隔"`
	ScanPhases      string `json:"scan_phases" gorm:"type:text;comment:适用的扫描阶段，逗号分隔"`

	// 规则属性
	Severity   ScanRuleSeverity `json:"severity" gorm:"size:20;default:'medium';comment:规则严重程度"`
	Priority   int              `json:"priority" gorm:"default:5;comment:规则优先级(1-10)" validate:"min=1,max=10"`
	Confidence float64          `json:"confidence" gorm:"default:0.8;comment:规则置信度(0-1)" validate:"min=0,max=1"`

	// 执行配置
	MaxExecutions   int  `json:"max_executions" gorm:"default:0;comment:最大执行次数，0表示无限制"`
	TimeoutSeconds  int  `json:"timeout_seconds" gorm:"default:30;comment:执行超时时间(秒)" validate:"min=1,max=300"`
	ContinueOnError bool `json:"continue_on_error" gorm:"default:true;comment:出错时是否继续"`

	// 状态管理
	Status    ScanRuleStatus `json:"status" gorm:"default:0;comment:规则状态:0-禁用,1-启用,2-测试中"`
	IsBuiltIn bool           `json:"is_built_in" gorm:"default:false;comment:是否为内置规则"`
	Version   string         `json:"version" gorm:"size:20;default:'1.0';comment:规则版本"`

	// 元数据
	Tags     string `json:"tags" gorm:"type:text;comment:规则标签，逗号分隔"`
	Metadata string `json:"metadata" gorm:"type:json;comment:扩展元数据"`

	// 审计字段
	CreatedBy uint64 `json:"created_by" gorm:"comment:创建者ID"`
	UpdatedBy uint64 `json:"updated_by" gorm:"comment:更新者ID"`

	// 统计信息
	ExecutionCount int64 `json:"execution_count" gorm:"default:0;comment:执行次数"`
	MatchCount     int64 `json:"match_count" gorm:"default:0;comment:匹配次数"`
	ErrorCount     int64 `json:"error_count" gorm:"default:0;comment:错误次数"`

	// 性能统计
	AvgExecutionTime float64 `json:"avg_execution_time" gorm:"default:0;comment:平均执行时间(毫秒)"`
	MaxExecutionTime float64 `json:"max_execution_time" gorm:"default:0;comment:最大执行时间(毫秒)"`
}

// RuleCondition 规则条件结构
// 简单的条件表达式，支持基本的逻辑运算
type RuleCondition struct {
	Field    string      `json:"field" validate:"required"`    // 字段名
	Operator string      `json:"operator" validate:"required"` // 操作符：eq, ne, gt, lt, gte, lte, in, not_in, contains, regex
	Value    interface{} `json:"value" validate:"required"`    // 比较值
	Logic    string      `json:"logic,omitempty"`              // 逻辑连接符：and, or（用于多条件）
}

// RuleAction 规则动作结构
type RuleAction struct {
	Type       string                 `json:"type" validate:"required"` // 动作类型：block, allow, modify, alert, log
	Parameters map[string]interface{} `json:"parameters,omitempty"`     // 动作参数
	Message    string                 `json:"message,omitempty"`        // 动作消息
}

// TableName 定义数据库表名
func (ScanRule) TableName() string {
	return "scan_rules"
}

// IsEnabled 检查规则是否启用
func (sr *ScanRule) IsEnabled() bool {
	return sr.Status == ScanRuleStatusEnabled
}

// CanExecute 检查规则是否可以执行
func (sr *ScanRule) CanExecute() bool {
	return sr.IsEnabled() && sr.Condition != ""
}

// IsApplicableToTool 检查规则是否适用于指定工具
func (sr *ScanRule) IsApplicableToTool(toolName string) bool {
	if sr.ApplicableTools == "" {
		return true // 空表示适用于所有工具
	}

	tools := sr.GetApplicableToolsList()
	for _, tool := range tools {
		if tool == toolName || tool == "*" {
			return true
		}
	}
	return false
}

// IsApplicableToTargetType 检查规则是否适用于指定目标类型
func (sr *ScanRule) IsApplicableToTargetType(targetType string) bool {
	if sr.TargetTypes == "" {
		return true // 空表示适用于所有目标类型
	}

	types := sr.GetTargetTypesList()
	for _, t := range types {
		if t == targetType || t == "*" {
			return true
		}
	}
	return false
}

// IsApplicableToScanPhase 检查规则是否适用于指定扫描阶段
func (sr *ScanRule) IsApplicableToScanPhase(phase string) bool {
	if sr.ScanPhases == "" {
		return true // 空表示适用于所有阶段
	}

	phases := sr.GetScanPhasesList()
	for _, p := range phases {
		if p == phase || p == "*" {
			return true
		}
	}
	return false
}

// GetParametersMap 获取规则参数映射
func (sr *ScanRule) GetParametersMap() (map[string]interface{}, error) {
	if sr.Parameters == "" {
		return make(map[string]interface{}), nil
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(sr.Parameters), &params); err != nil {
		return nil, fmt.Errorf("解析规则参数失败: %w", err)
	}
	return params, nil
}

// GetConditionStruct 获取规则条件结构
func (sr *ScanRule) GetConditionStruct() (*RuleCondition, error) {
	if sr.Condition == "" {
		return nil, fmt.Errorf("规则条件为空")
	}

	var condition RuleCondition
	if err := json.Unmarshal([]byte(sr.Condition), &condition); err != nil {
		return nil, fmt.Errorf("解析规则条件失败: %w", err)
	}
	return &condition, nil
}

// GetActionStruct 获取规则动作结构
func (sr *ScanRule) GetActionStruct() (*RuleAction, error) {
	if sr.Action == "" {
		return nil, nil // 动作可以为空
	}

	// 首先尝试解析为单个RuleAction对象
	var action RuleAction
	if err := json.Unmarshal([]byte(sr.Action), &action); err == nil {
		return &action, nil
	}

	// 如果失败，尝试解析为RuleAction数组，取第一个元素
	var actions []RuleAction
	if err := json.Unmarshal([]byte(sr.Action), &actions); err != nil {
		return nil, fmt.Errorf("解析规则动作失败: %w", err)
	}

	if len(actions) == 0 {
		return nil, nil // 动作数组为空
	}

	return &actions[0], nil // 返回第一个动作
}

// GetApplicableToolsList 获取适用工具列表
func (sr *ScanRule) GetApplicableToolsList() []string {
	if sr.ApplicableTools == "" {
		return []string{}
	}

	tools := make([]string, 0)
	for _, tool := range strings.Split(sr.ApplicableTools, ",") {
		if trimmed := strings.TrimSpace(tool); trimmed != "" {
			tools = append(tools, trimmed)
		}
	}
	return tools
}

// GetTargetTypesList 获取目标类型列表
func (sr *ScanRule) GetTargetTypesList() []string {
	if sr.TargetTypes == "" {
		return []string{}
	}

	types := make([]string, 0)
	for _, t := range strings.Split(sr.TargetTypes, ",") {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			types = append(types, trimmed)
		}
	}
	return types
}

// GetScanPhasesList 获取扫描阶段列表
func (sr *ScanRule) GetScanPhasesList() []string {
	if sr.ScanPhases == "" {
		return []string{}
	}

	phases := make([]string, 0)
	for _, phase := range strings.Split(sr.ScanPhases, ",") {
		if trimmed := strings.TrimSpace(phase); trimmed != "" {
			phases = append(phases, trimmed)
		}
	}
	return phases
}

// GetTagList 获取标签列表
func (sr *ScanRule) GetTagList() []string {
	if sr.Tags == "" {
		return []string{}
	}

	tags := make([]string, 0)
	for _, tag := range strings.Split(sr.Tags, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

// UpdateExecutionStats 更新执行统计
func (sr *ScanRule) UpdateExecutionStats(matched bool, executionTime float64, hasError bool) {
	sr.ExecutionCount++

	if matched {
		sr.MatchCount++
	}

	if hasError {
		sr.ErrorCount++
	}

	// 更新平均执行时间
	if sr.ExecutionCount == 1 {
		sr.AvgExecutionTime = executionTime
	} else {
		sr.AvgExecutionTime = (sr.AvgExecutionTime*float64(sr.ExecutionCount-1) + executionTime) / float64(sr.ExecutionCount)
	}

	// 更新最大执行时间
	if executionTime > sr.MaxExecutionTime {
		sr.MaxExecutionTime = executionTime
	}
}

// GetMatchRate 获取匹配率
func (sr *ScanRule) GetMatchRate() float64 {
	if sr.ExecutionCount == 0 {
		return 0.0
	}
	return float64(sr.MatchCount) / float64(sr.ExecutionCount) * 100
}

// GetErrorRate 获取错误率
func (sr *ScanRule) GetErrorRate() float64 {
	if sr.ExecutionCount == 0 {
		return 0.0
	}
	return float64(sr.ErrorCount) / float64(sr.ExecutionCount) * 100
}

// ValidateCondition 验证规则条件的合法性
// 简单的条件验证，确保基本语法正确
func (sr *ScanRule) ValidateCondition() error {
	condition, err := sr.GetConditionStruct()
	if err != nil {
		return err
	}

	if condition.Field == "" {
		return fmt.Errorf("规则条件字段不能为空")
	}

	validOperators := []string{"eq", "ne", "gt", "lt", "gte", "lte", "in", "not_in", "contains", "regex"}
	isValidOperator := false
	for _, op := range validOperators {
		if condition.Operator == op {
			isValidOperator = true
			break
		}
	}
	if !isValidOperator {
		return fmt.Errorf("无效的操作符: %s", condition.Operator)
	}

	return nil
}
