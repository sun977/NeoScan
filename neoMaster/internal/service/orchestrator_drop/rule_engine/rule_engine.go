/*
 * 规则引擎：扫描策略规则的解析和执行
 * @author: Sun977
 * @date: 2025.10.11
 * @description: 提供规则条件解析、匹配算法、动作执行等核心功能
 * @func:
 * 1.规则条件表达式解析
 * 2.规则匹配算法执行
 * 3.规则动作处理
 * 4.规则缓存和优先级管理
 */

//  核心功能:
//  	ParseCondition - 解析规则条件表达式
//  	EvaluateCondition - 评估条件是否匹配
//  	ExecuteAction - 执行规则动作
//  	MatchRules - 批量规则匹配
//  缓存功能:
//  	CacheRule - 缓存规则到内存
//  	InvalidateCache - 失效规则缓存
//  	GetCachedRule - 获取缓存的规则
//  优先级功能:
//  	SortRulesByPriority - 按优先级排序规则
//  	ResolveConflicts - 解决规则冲突

package rule_engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"neomaster/internal/model/orchestrator_drop"
	"neomaster/internal/pkg/logger"
)

// RuleEngine 规则引擎结构体
// 负责规则的解析、匹配、执行和缓存管理
type RuleEngine struct {
	ruleCache    map[uint]*CachedRule // 规则缓存
	cacheMutex   sync.RWMutex         // 缓存读写锁
	cacheTimeout time.Duration        // 缓存超时时间
	evaluator    *Evaluator           // 条件评估器
	executor     *ActionExecutor      // 动作执行器
	rules        map[string]*Rule     // 规则存储
	rulesMux     sync.RWMutex         // 规则读写锁
	metrics      *RuleEngineMetrics   // 引擎指标
}

// CachedRule 缓存的规则结构体
type CachedRule struct {
	Rule            *orchestrator_drop.ScanRule // 规则对象
	ParsedCondition *ParsedCondition            // 解析后的条件
	CachedAt        time.Time                   // 缓存时间
}

// ParsedCondition 解析后的条件结构体
type ParsedCondition struct {
	Operator string             // 操作符 (and, or, not, eq, ne, gt, lt, gte, lte, in, contains, regex)
	Field    string             // 字段名
	Value    interface{}        // 比较值
	Children []*ParsedCondition // 子条件（用于复合条件）
}

// RuleMatchResult 规则匹配结果
type RuleMatchResult struct {
	Rule    *orchestrator_drop.ScanRule // 匹配的规则
	Matched bool                        // 是否匹配
	Score   float64                     // 匹配分数
	Reason  string                      // 匹配原因
	Actions []RuleAction                // 要执行的动作
}

// RuleAction 规则动作
type RuleAction struct {
	Type       string                 // 动作类型
	Parameters map[string]interface{} // 动作参数
}

// NewRuleEngine 创建规则引擎实例
// @param cacheTimeout 缓存超时时间
// @return *RuleEngine 规则引擎实例
func NewRuleEngine(cacheTimeout time.Duration) *RuleEngine {
	return &RuleEngine{
		ruleCache:    make(map[uint]*CachedRule),
		cacheTimeout: cacheTimeout,
		evaluator:    NewEvaluator(nil),
		executor:     NewActionExecutor(nil),
		rules:        make(map[string]*Rule),
		rulesMux:     sync.RWMutex{},
		metrics:      &RuleEngineMetrics{},
	}
}

// AddRule 添加规则
func (re *RuleEngine) AddRule(rule *Rule) error {
	if rule == nil {
		return fmt.Errorf("规则不能为空")
	}

	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}

	// 验证规则
	if err := re.ValidateRule(rule); err != nil {
		return fmt.Errorf("规则验证失败: %v", err)
	}

	re.rulesMux.Lock()
	defer re.rulesMux.Unlock()

	// 检查规则是否已存在
	if _, exists := re.rules[rule.ID]; exists {
		return fmt.Errorf("规则已存在: %s", rule.ID)
	}

	// 设置时间戳
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	re.rules[rule.ID] = rule
	re.metrics.TotalRules++
	if rule.Enabled {
		re.metrics.EnabledRules++
	}

	logger.LogBusinessOperation("add_rule", 0, "", "", "", "success", "规则添加成功", map[string]interface{}{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
		"func_name": "rule_engine.AddRule",
	})

	return nil
}

// UpdateRule 更新规则
func (re *RuleEngine) UpdateRule(rule *Rule) error {
	if rule == nil {
		return fmt.Errorf("规则不能为空")
	}

	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}

	// 验证规则
	if err := re.ValidateRule(rule); err != nil {
		return fmt.Errorf("规则验证失败: %v", err)
	}

	re.rulesMux.Lock()
	defer re.rulesMux.Unlock()

	// 检查规则是否存在
	oldRule, exists := re.rules[rule.ID]
	if !exists {
		return fmt.Errorf("规则不存在: %s", rule.ID)
	}

	// 保留创建时间，更新修改时间
	rule.CreatedAt = oldRule.CreatedAt
	rule.UpdatedAt = time.Now()

	// 更新启用规则计数
	if oldRule.Enabled != rule.Enabled {
		if rule.Enabled {
			re.metrics.EnabledRules++
		} else {
			re.metrics.EnabledRules--
		}
	}

	re.rules[rule.ID] = rule

	logger.LogBusinessOperation("update_rule", 0, "", "", "", "success", "规则更新成功", map[string]interface{}{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
		"func_name": "rule_engine.UpdateRule",
	})

	return nil
}

// RemoveRule 删除规则
func (re *RuleEngine) RemoveRule(ruleID string) error {
	if ruleID == "" {
		return fmt.Errorf("规则ID不能为空")
	}

	re.rulesMux.Lock()
	defer re.rulesMux.Unlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return fmt.Errorf("规则不存在: %s", ruleID)
	}

	delete(re.rules, ruleID)
	re.metrics.TotalRules--
	if rule.Enabled {
		re.metrics.EnabledRules--
	}

	logger.LogBusinessOperation("remove_rule", 0, "", "", "", "success", "规则删除成功", map[string]interface{}{
		"rule_id":   ruleID,
		"rule_name": rule.Name,
		"func_name": "rule_engine.RemoveRule",
	})

	return nil
}

// GetRule 获取规则
func (re *RuleEngine) GetRule(ruleID string) (*Rule, error) {
	re.rulesMux.RLock()
	defer re.rulesMux.RUnlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("规则不存在: %s", ruleID)
	}

	// 返回规则的副本
	ruleCopy := *rule
	return &ruleCopy, nil
}

// ListRules 列出所有规则
func (re *RuleEngine) ListRules() []*Rule {
	re.rulesMux.RLock()
	defer re.rulesMux.RUnlock()

	rules := make([]*Rule, 0, len(re.rules))
	for _, rule := range re.rules {
		ruleCopy := *rule
		rules = append(rules, &ruleCopy)
	}

	return rules
}

// EnableRule 启用规则
func (re *RuleEngine) EnableRule(ruleID string) error {
	re.rulesMux.Lock()
	defer re.rulesMux.Unlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return fmt.Errorf("规则不存在: %s", ruleID)
	}

	if !rule.Enabled {
		rule.Enabled = true
		rule.UpdatedAt = time.Now()
		re.metrics.EnabledRules++

		logger.LogBusinessOperation("enable_rule", 0, "", "", "", "success", "规则已启用", map[string]interface{}{
			"rule_id":   ruleID,
			"func_name": "rule_engine.EnableRule",
		})
	}

	return nil
}

// DisableRule 禁用规则
func (re *RuleEngine) DisableRule(ruleID string) error {
	re.rulesMux.Lock()
	defer re.rulesMux.Unlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return fmt.Errorf("规则不存在: %s", ruleID)
	}

	if rule.Enabled {
		rule.Enabled = false
		rule.UpdatedAt = time.Now()
		re.metrics.EnabledRules--

		logger.LogBusinessOperation("disable_rule", 0, "", "", "", "success", "规则已禁用", map[string]interface{}{
			"rule_id":   ruleID,
			"func_name": "rule_engine.DisableRule",
		})
	}

	return nil
}

// ExecuteRule 执行单个规则
func (re *RuleEngine) ExecuteRule(ruleID string, context *RuleContext) (*RuleResult, error) {
	re.rulesMux.RLock()
	rule, exists := re.rules[ruleID]
	re.rulesMux.RUnlock()

	if !exists {
		return nil, fmt.Errorf("规则不存在: %s", ruleID)
	}

	if !rule.Enabled {
		logger.WithFields(map[string]interface{}{
			"rule_id":   ruleID,
			"func_name": "rule_engine.ExecuteRule",
		}).Info("规则已禁用，跳过执行")
		return &RuleResult{
			RuleID:    ruleID,
			Matched:   false,
			Actions:   []ActionResult{},
			Message:   "规则已禁用",
			Metadata:  make(map[string]interface{}),
			Timestamp: time.Now(),
		}, nil
	}

	// 更新指标
	re.metrics.TotalExecutions++

	// 评估条件
	matched, err := re.evaluator.EvaluateConditions(rule.Conditions, context)
	if err != nil {
		re.metrics.FailedRuns++
		logger.Error("规则条件评估失败", map[string]interface{}{
			"rule_id":   ruleID,
			"error":     err.Error(),
			"func_name": "rule_engine.ExecuteRule",
		})
		return nil, fmt.Errorf("规则条件评估失败: %v", err)
	}

	result := &RuleResult{
		RuleID:    ruleID,
		Matched:   matched,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// 如果条件匹配，执行动作
	if matched {
		logger.Info("规则匹配成功", map[string]interface{}{
			"rule_id":    rule.ID,
			"rule_name":  rule.Name,
			"match_type": "condition",
			"timestamp":  time.Now().Format("2006-01-02 15:04:05"),
		})

		actionResults := re.executor.ExecuteActions(rule.Actions, context)
		result.Actions = actionResults
		result.Message = fmt.Sprintf("规则匹配，执行了 %d 个动作", len(actionResults))

		// 检查是否有动作执行失败
		for _, actionResult := range actionResults {
			if !actionResult.Success {
				re.metrics.FailedRuns++
				result.Message += fmt.Sprintf(", 动作 %s 执行失败: %s", actionResult.Type, actionResult.Error)
			}
		}
	} else {
		logger.Warn("规则条件不匹配", map[string]interface{}{
			"rule_id":   rule.ID,
			"rule_name": rule.Name,
			"condition": rule.Conditions,
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		})
		result.Message = "规则条件不匹配"
	}

	re.metrics.SuccessfulRuns++
	re.metrics.LastExecutionAt = time.Now()

	return result, nil
}

// ExecuteRules 批量执行规则
func (re *RuleEngine) ExecuteRules(context *RuleContext) (*BatchRuleResult, error) {
	startTime := time.Now()

	re.rulesMux.RLock()
	rules := make([]*Rule, 0, len(re.rules))
	for _, rule := range re.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	re.rulesMux.RUnlock()

	// 按优先级排序 - 直接对Rule类型进行排序
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	results := make([]RuleResult, 0, len(rules))
	matched := 0
	failed := 0

	for _, rule := range rules {
		result, err := re.ExecuteRule(rule.ID, context)
		if err != nil {
			failed++
			logger.LogBusinessError(err, "", 0, "", "", "", map[string]interface{}{
				"rule_id":   rule.ID,
				"error":     err.Error(),
				"func_name": "rule_engine.ExecuteRules",
			})
			continue
		}

		results = append(results, *result)
		if result.Matched {
			matched++
		}

		// 检查是否被阻止
		if context.Variables != nil {
			if blocked, exists := context.Variables["blocked"]; exists && blocked.(bool) {
				logger.Info("检测到阻止标志，停止后续规则执行", map[string]interface{}{
					"func_name": "rule_engine.ExecuteRules",
				})
				break
			}
		}
	}

	batchResult := &BatchRuleResult{
		Results:   results,
		Total:     len(rules),
		Matched:   matched,
		Failed:    failed,
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	return batchResult, nil
}

// sortRulesByPriority 按优先级对规则进行排序（删除重复定义）
// 已在第438行定义，此处删除重复定义

// ParseCondition 解析规则条件表达式
// @param ctx 上下文
// @param condition 条件表达式JSON字符串
// @return *ParsedCondition 解析后的条件对象
// @return error 错误信息
func (re *RuleEngine) ParseCondition(ctx context.Context, condition string) (*ParsedCondition, error) {
	if condition == "" {
		return nil, errors.New("条件表达式不能为空")
	}

	// 解析JSON格式的条件表达式
	var conditionMap map[string]interface{}
	if err := json.Unmarshal([]byte(condition), &conditionMap); err != nil {
		logger.Error("解析条件失败", map[string]interface{}{
			"operation": "parse_condition",
			"option":    "json.Unmarshal",
			"error":     err.Error(),
			"condition": condition,
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		})
		return nil, fmt.Errorf("条件表达式JSON解析失败: %w", err)
	}

	// 递归解析条件
	parsed, err := re.parseConditionRecursive(conditionMap)
	if err != nil {
		logger.Error("解析条件递归失败", map[string]interface{}{
			"operation": "parse_condition",
			"option":    "parseConditionRecursive",
			"error":     err.Error(),
			"condition": condition,
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		})
		return nil, fmt.Errorf("条件表达式解析失败: %w", err)
	}

	return parsed, nil
}

// parseConditionRecursive 递归解析条件表达式
// @param conditionMap 条件映射
// @return *ParsedCondition 解析后的条件
// @return error 错误信息
func (re *RuleEngine) parseConditionRecursive(conditionMap map[string]interface{}) (*ParsedCondition, error) {
	// 获取操作符
	operator, ok := conditionMap["operator"].(string)
	if !ok {
		return nil, errors.New("缺少操作符")
	}

	parsed := &ParsedCondition{
		Operator: strings.ToLower(operator),
	}

	// 根据操作符类型解析
	switch parsed.Operator {
	case "and", "or":
		// 逻辑操作符，解析子条件
		conditions, ok := conditionMap["conditions"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("逻辑操作符 %s 缺少子条件", operator)
		}

		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				return nil, errors.New("子条件格式错误")
			}

			child, err := re.parseConditionRecursive(condMap)
			if err != nil {
				return nil, fmt.Errorf("解析子条件失败: %w", err)
			}
			parsed.Children = append(parsed.Children, child)
		}

	case "not":
		// 否定操作符，解析单个子条件
		condition, ok := conditionMap["condition"].(map[string]interface{})
		if !ok {
			return nil, errors.New("NOT操作符缺少子条件")
		}

		child, err := re.parseConditionRecursive(condition)
		if err != nil {
			return nil, fmt.Errorf("解析NOT子条件失败: %w", err)
		}
		parsed.Children = []*ParsedCondition{child}

	case "eq", "ne", "gt", "lt", "gte", "lte", "in", "contains", "regex":
		// 比较操作符，解析字段和值
		field, ok := conditionMap["field"].(string)
		if !ok {
			return nil, fmt.Errorf("比较操作符 %s 缺少字段名", operator)
		}
		parsed.Field = field

		value, exists := conditionMap["value"]
		if !exists {
			return nil, fmt.Errorf("比较操作符 %s 缺少比较值", operator)
		}
		parsed.Value = value

	default:
		return nil, fmt.Errorf("不支持的操作符: %s", operator)
	}

	return parsed, nil
}

// EvaluateCondition 评估条件是否匹配
// @param ctx 上下文
// @param condition 解析后的条件
// @param target 目标数据
// @return bool 是否匹配
// @return error 错误信息
func (re *RuleEngine) EvaluateCondition(ctx context.Context, condition *ParsedCondition, target map[string]interface{}) (bool, error) {
	if condition == nil {
		return false, errors.New("条件不能为空")
	}

	switch condition.Operator {
	case "and":
		// 所有子条件都必须为真
		for _, child := range condition.Children {
			matched, err := re.EvaluateCondition(ctx, child, target)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
		return true, nil

	case "or":
		// 任一子条件为真即可
		for _, child := range condition.Children {
			matched, err := re.EvaluateCondition(ctx, child, target)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil

	case "not":
		// 子条件为假
		if len(condition.Children) != 1 {
			return false, errors.New("NOT操作符只能有一个子条件")
		}
		matched, err := re.EvaluateCondition(ctx, condition.Children[0], target)
		if err != nil {
			return false, err
		}
		return !matched, nil

	case "eq", "ne", "gt", "lt", "gte", "lte", "in", "contains", "regex":
		// 比较操作
		return re.evaluateComparison(condition, target)

	default:
		return false, fmt.Errorf("不支持的操作符: %s", condition.Operator)
	}
}

// evaluateComparison 评估比较操作
// @param condition 条件
// @param target 目标数据
// @return bool 是否匹配
// @return error 错误信息
func (re *RuleEngine) evaluateComparison(condition *ParsedCondition, target map[string]interface{}) (bool, error) {
	// 获取目标字段值
	targetValue, exists := target[condition.Field]
	if !exists {
		return false, nil // 字段不存在，不匹配
	}

	switch condition.Operator {
	case "eq":
		return re.compareValues(targetValue, condition.Value, "eq")
	case "ne":
		result, err := re.compareValues(targetValue, condition.Value, "eq")
		return !result, err
	case "gt":
		return re.compareValues(targetValue, condition.Value, "gt")
	case "lt":
		return re.compareValues(targetValue, condition.Value, "lt")
	case "gte":
		return re.compareValues(targetValue, condition.Value, "gte")
	case "lte":
		return re.compareValues(targetValue, condition.Value, "lte")
	case "in":
		return re.evaluateInCondition(targetValue, condition.Value)
	case "contains":
		return re.evaluateContainsCondition(targetValue, condition.Value)
	case "regex":
		return re.evaluateRegexCondition(targetValue, condition.Value)
	default:
		return false, fmt.Errorf("不支持的比较操作符: %s", condition.Operator)
	}
}

// compareValues 比较两个值
// @param target 目标值
// @param expected 期望值
// @param operator 操作符
// @return bool 比较结果
// @return error 错误信息
func (re *RuleEngine) compareValues(target, expected interface{}, operator string) (bool, error) {
	// 尝试转换为数字进行比较
	if targetNum, targetOk := re.toFloat64(target); targetOk {
		if expectedNum, expectedOk := re.toFloat64(expected); expectedOk {
			switch operator {
			case "eq":
				return targetNum == expectedNum, nil
			case "gt":
				return targetNum > expectedNum, nil
			case "lt":
				return targetNum < expectedNum, nil
			case "gte":
				return targetNum >= expectedNum, nil
			case "lte":
				return targetNum <= expectedNum, nil
			}
		}
	}

	// 字符串比较
	targetStr := fmt.Sprintf("%v", target)
	expectedStr := fmt.Sprintf("%v", expected)

	switch operator {
	case "eq":
		return targetStr == expectedStr, nil
	case "gt":
		return targetStr > expectedStr, nil
	case "lt":
		return targetStr < expectedStr, nil
	case "gte":
		return targetStr >= expectedStr, nil
	case "lte":
		return targetStr <= expectedStr, nil
	default:
		return false, fmt.Errorf("不支持的比较操作符: %s", operator)
	}
}

// toFloat64 尝试将值转换为float64
// @param value 要转换的值
// @return float64 转换后的值
// @return bool 是否转换成功
func (re *RuleEngine) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// evaluateInCondition 评估in条件
// @param target 目标值
// @param expected 期望值（数组）
// @return bool 是否匹配
// @return error 错误信息
func (re *RuleEngine) evaluateInCondition(target, expected interface{}) (bool, error) {
	expectedList, ok := expected.([]interface{})
	if !ok {
		return false, errors.New("in操作符的期望值必须是数组")
	}

	targetStr := fmt.Sprintf("%v", target)
	for _, item := range expectedList {
		itemStr := fmt.Sprintf("%v", item)
		if targetStr == itemStr {
			return true, nil
		}
	}
	return false, nil
}

// evaluateContainsCondition 评估contains条件
// @param target 目标值
// @param expected 期望值
// @return bool 是否匹配
// @return error 错误信息
func (re *RuleEngine) evaluateContainsCondition(target, expected interface{}) (bool, error) {
	targetStr := fmt.Sprintf("%v", target)
	expectedStr := fmt.Sprintf("%v", expected)

	return strings.Contains(targetStr, expectedStr), nil
}

// evaluateRegexCondition 评估正则表达式条件
// @param target 目标值
// @param pattern 正则表达式模式
// @return bool 是否匹配
// @return error 错误信息
func (re *RuleEngine) evaluateRegexCondition(target, pattern interface{}) (bool, error) {
	targetStr := fmt.Sprintf("%v", target)
	patternStr := fmt.Sprintf("%v", pattern)

	regex, err := regexp.Compile(patternStr)
	if err != nil {
		return false, fmt.Errorf("无效的正则表达式: %s", patternStr)
	}

	return regex.MatchString(targetStr), nil
}

// ExecuteAction 执行规则动作
// @param ctx 上下文
// @param action 规则动作
// @param target 目标数据
// @return map[string]interface{} 执行结果
// @return error 错误信息
func (re *RuleEngine) ExecuteAction(ctx context.Context, action *orchestrator_drop.RuleAction, target map[string]interface{}) (map[string]interface{}, error) {
	if action == nil {
		return nil, errors.New("动作不能为空")
	}

	result := make(map[string]interface{})
	result["action_type"] = action.Type
	result["executed_at"] = time.Now()

	// 根据动作类型执行相应操作
	switch action.Type {
	case "log":
		// 记录日志动作
		message := fmt.Sprintf("规则动作执行: %s", action.Parameters["message"])
		logger.Info("规则动作执行", map[string]interface{}{
			"operation": "execute_action",
			"option":    "log_action",
			"message":   message,
			"target":    target,
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		})
		result["message"] = message

	case "alert":
		// 告警动作
		alertLevel := action.Parameters["level"]
		alertMessage := action.Parameters["message"]
		logger.Warn("规则告警动作执行", map[string]interface{}{
			"operation": "execute_action",
			"option":    "alert_action",
			"level":     alertLevel,
			"message":   alertMessage,
			"target":    target,
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		})
		result["alert_level"] = alertLevel
		result["alert_message"] = alertMessage

	case "block":
		// 阻断动作
		blockReason := action.Parameters["reason"]
		logger.Error("规则阻断动作执行", map[string]interface{}{
			"operation":   "execute_action",
			"option":      "block_action",
			"action_type": action.Type,
			"target":      target,
			"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		})
		result["blocked"] = true
		result["block_reason"] = blockReason

	case "modify":
		// 修改动作
		modifications := action.Parameters["modifications"]
		result["modifications"] = modifications
		result["original_target"] = target

	default:
		return nil, fmt.Errorf("不支持的动作类型: %s", action.Type)
	}

	return result, nil
}

// MatchRules 批量规则匹配
// @param ctx 上下文
// @param rules 规则列表
// @param target 目标数据
// @return []*RuleMatchResult 匹配结果列表
// @return error 错误信息
func (re *RuleEngine) MatchRules(ctx context.Context, rules []*orchestrator_drop.ScanRule, target map[string]interface{}) ([]*RuleMatchResult, error) {
	var results []*RuleMatchResult

	// 按优先级排序规则
	sortedRules := re.SortRulesByPriority(rules)

	for _, rule := range sortedRules {
		// 获取或解析规则条件
		cachedRule := re.getCachedRule(uint(rule.ID))
		var parsedCondition *ParsedCondition
		var err error

		if cachedRule != nil && time.Since(cachedRule.CachedAt) < re.cacheTimeout {
			parsedCondition = cachedRule.ParsedCondition
		} else {
			// 解析条件并缓存
			parsedCondition, err = re.ParseCondition(ctx, rule.Condition)
			if err != nil {
				logger.Error("解析规则条件失败", map[string]interface{}{
					"operation": "match_rules",
					"option":    "ParseCondition",
					"rule_id":   rule.ID,
					"rule_name": rule.Name,
					"error":     err.Error(),
					"timestamp": time.Now().Format("2006-01-02 15:04:05"),
				})
				continue
			}

			// 缓存规则
			re.CacheRule(uint(rule.ID), rule, parsedCondition)
		}

		// 评估条件
		matched, err := re.EvaluateCondition(ctx, parsedCondition, target)
		if err != nil {
			logger.Error("评估规则条件失败", map[string]interface{}{
				"operation": "match_rules",
				"option":    "EvaluateCondition",
				"rule_id":   rule.ID,
				"rule_name": rule.Name,
				"error":     err.Error(),
				"timestamp": time.Now().Format("2006-01-02 15:04:05"),
			})
			continue
		}

		// 创建匹配结果
		result := &RuleMatchResult{
			Rule:    rule,
			Matched: matched,
			Score:   re.calculateMatchScore(rule, matched),
			Reason:  re.generateMatchReason(rule, matched),
		}

		// 如果匹配，准备执行动作
		if matched {
			result.Actions = re.prepareRuleActions(rule)
		}

		results = append(results, result)
	}

	return results, nil
}

// calculateMatchScore 计算匹配分数
// @param rule 规则
// @param matched 是否匹配
// @return float64 匹配分数
func (re *RuleEngine) calculateMatchScore(rule *orchestrator_drop.ScanRule, matched bool) float64 {
	if !matched {
		return 0.0
	}

	// 基础分数
	score := 1.0

	// 根据规则严重程度调整分数
	switch rule.Severity {
	case orchestrator_drop.ScanRuleSeverityCritical:
		score *= 1.5
	case orchestrator_drop.ScanRuleSeverityHigh:
		score *= 1.3
	case orchestrator_drop.ScanRuleSeverityMedium:
		score *= 1.1
	case orchestrator_drop.ScanRuleSeverityLow:
		score *= 1.0
	}

	// 根据规则优先级调整分数
	score += float64(rule.Priority) * 0.1

	return score
}

// generateMatchReason 生成匹配原因
// @param rule 规则
// @param matched 是否匹配
// @return string 匹配原因
func (re *RuleEngine) generateMatchReason(rule *orchestrator_drop.ScanRule, matched bool) string {
	if matched {
		return fmt.Sprintf("规则 '%s' 匹配成功，严重程度: %s", rule.Name, rule.Severity.String())
	}
	return fmt.Sprintf("规则 '%s' 不匹配", rule.Name)
}

// prepareRuleActions 准备规则动作
// @param rule 规则
// @return []RuleAction 动作列表
func (re *RuleEngine) prepareRuleActions(rule *orchestrator_drop.ScanRule) []RuleAction {
	var actions []RuleAction

	// 解析Action字段（JSON格式）
	if rule.Action != "" {
		var actionData RuleAction
		if err := json.Unmarshal([]byte(rule.Action), &actionData); err == nil {
			actions = append(actions, actionData)
		}
	}

	return actions
}

// CacheRule 缓存规则到内存
// @param ruleID 规则ID
// @param rule 规则对象
// @param parsedCondition 解析后的条件
func (re *RuleEngine) CacheRule(ruleID uint, rule *orchestrator_drop.ScanRule, parsedCondition *ParsedCondition) {
	re.cacheMutex.Lock()
	defer re.cacheMutex.Unlock()

	re.ruleCache[ruleID] = &CachedRule{
		Rule:            rule,
		ParsedCondition: parsedCondition,
		CachedAt:        time.Now(),
	}
}

// InvalidateCache 失效规则缓存
// @param ruleID 规则ID，如果为0则清空所有缓存
func (re *RuleEngine) InvalidateCache(ruleID uint) {
	re.cacheMutex.Lock()
	defer re.cacheMutex.Unlock()

	if ruleID == 0 {
		// 清空所有缓存
		re.ruleCache = make(map[uint]*CachedRule)
	} else {
		// 删除指定规则缓存
		delete(re.ruleCache, ruleID)
	}
}

// getCachedRule 获取缓存的规则
// @param ruleID 规则ID
// @return *CachedRule 缓存的规则，如果不存在返回nil
func (re *RuleEngine) getCachedRule(ruleID uint) *CachedRule {
	re.cacheMutex.RLock()
	defer re.cacheMutex.RUnlock()

	if cached, exists := re.ruleCache[ruleID]; exists {
		return cached
	}
	return nil
}

// ValidateRule 验证规则
func (re *RuleEngine) ValidateRule(rule *Rule) error {
	if rule == nil {
		return fmt.Errorf("规则不能为空")
	}

	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}

	if rule.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}

	// 验证严重程度
	validSeverities := []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}
	validSeverity := false
	for _, severity := range validSeverities {
		if rule.Severity == severity {
			validSeverity = true
			break
		}
	}
	if !validSeverity {
		return fmt.Errorf("无效的严重程度: %s", rule.Severity)
	}

	// 验证优先级
	if rule.Priority < 1 || rule.Priority > 100 {
		return fmt.Errorf("优先级必须在1-100之间: %d", rule.Priority)
	}

	// 验证条件
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("规则必须包含至少一个条件")
	}

	for i, condition := range rule.Conditions {
		if err := re.evaluator.ValidateCondition(condition); err != nil {
			return fmt.Errorf("条件 %d 验证失败: %v", i, err)
		}
	}

	// 验证动作
	if len(rule.Actions) == 0 {
		return fmt.Errorf("规则必须包含至少一个动作")
	}

	for i, action := range rule.Actions {
		if err := re.ValidateAction(action); err != nil {
			return fmt.Errorf("动作 %d 验证失败: %v", i, err)
		}
	}

	return nil
}

// ValidateAction 验证动作
func (re *RuleEngine) ValidateAction(action Action) error {
	// 验证动作类型
	validTypes := []string{ActionLog, ActionAlert, ActionBlock, ActionModify, ActionExecute}
	validType := false
	for _, actionType := range validTypes {
		if action.Type == actionType {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("无效的动作类型: %s", action.Type)
	}

	// 验证动作参数
	if action.Parameters == nil {
		action.Parameters = make(map[string]interface{})
	}

	// 根据动作类型验证特定参数
	switch action.Type {
	case ActionModify:
		if _, exists := action.Parameters["field"]; !exists {
			return fmt.Errorf("修改动作必须指定field参数")
		}
	case ActionExecute:
		if _, exists := action.Parameters["command"]; !exists {
			return fmt.Errorf("执行动作必须指定command参数")
		}
	}

	return nil
}

// GetMetrics 获取引擎指标
func (re *RuleEngine) GetMetrics() *RuleEngineMetrics {
	re.rulesMux.RLock()
	defer re.rulesMux.RUnlock()

	// 计算总规则数和启用规则数
	totalRules := int64(len(re.rules))
	enabledRules := int64(0)
	activeRules := int64(0)

	for _, rule := range re.rules {
		if rule.Enabled {
			enabledRules++
			// 活跃规则定义为已启用且最近有执行记录的规则
			// 这里简化处理，将所有启用的规则都视为活跃规则
			activeRules++
		}
	}

	// 更新指标
	re.metrics.TotalRules = totalRules
	re.metrics.EnabledRules = enabledRules
	re.metrics.ActiveRules = activeRules

	// 计算平均延迟
	if re.metrics.TotalExecutions > 0 {
		// 这里简化处理，实际应该维护延迟历史
		re.metrics.AverageLatency = time.Millisecond * 10
	}

	// 计算缓存命中率
	re.cacheMutex.RLock()
	cacheSize := len(re.ruleCache)
	re.cacheMutex.RUnlock()

	// 简化缓存命中率计算
	if cacheSize > 0 {
		re.metrics.CacheHitRate = 0.8 // 简化处理
	}

	// 返回指标副本
	metricsCopy := *re.metrics
	return &metricsCopy
}

// ClearCache 清空缓存
func (re *RuleEngine) ClearCache() {
	re.cacheMutex.Lock()
	defer re.cacheMutex.Unlock()

	re.ruleCache = make(map[uint]*CachedRule)

	logger.Info("规则引擎缓存已清空", map[string]interface{}{
		"func_name": "rule_engine.ClearCache",
	})
}

// Stop 停止规则引擎
func (re *RuleEngine) Stop() {
	logger.Info("规则引擎已停止", map[string]interface{}{
		"func_name": "rule_engine.Stop",
	})
}

// sortRulesByPriority 按优先级排序规则
// @param rules 规则列表
// @return []*orchestrator_drop.ScanRule 排序后的规则列表
func (re *RuleEngine) sortRulesByPriority(rules []*orchestrator_drop.ScanRule) []*orchestrator_drop.ScanRule {
	// 创建副本避免修改原始切片
	sortedRules := make([]*orchestrator_drop.ScanRule, len(rules))
	copy(sortedRules, rules)

	// 按优先级降序排序（优先级数值越大越优先）
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority > sortedRules[j].Priority
	})

	return sortedRules
}

// SortRulesByPriority 按优先级排序规则（公开方法）
// @param rules 规则列表
// @return []*orchestrator_drop.ScanRule 排序后的规则列表
func (re *RuleEngine) SortRulesByPriority(rules []*orchestrator_drop.ScanRule) []*orchestrator_drop.ScanRule {
	return re.sortRulesByPriority(rules)
}

// getSeverityWeight 获取严重程度权重
// @param severity 严重程度
// @return int 权重值
func (re *RuleEngine) getSeverityWeight(severity orchestrator_drop.ScanRuleSeverity) int {
	switch severity {
	case orchestrator_drop.ScanRuleSeverityCritical:
		return 4
	case orchestrator_drop.ScanRuleSeverityHigh:
		return 3
	case orchestrator_drop.ScanRuleSeverityMedium:
		return 2
	case orchestrator_drop.ScanRuleSeverityLow:
		return 1
	default:
		return 0
	}
}

// ResolveConflicts 解决规则冲突
// @param results 匹配结果列表
// @return []*RuleMatchResult 解决冲突后的结果列表
func (re *RuleEngine) ResolveConflicts(results []*RuleMatchResult) []*RuleMatchResult {
	if len(results) <= 1 {
		return results
	}

	// 按匹配分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	var resolvedResults []*RuleMatchResult
	conflictGroups := make(map[string][]*RuleMatchResult)

	// 按规则类型分组
	for _, result := range results {
		if result.Matched {
			ruleType := result.Rule.Type.String()
			conflictGroups[ruleType] = append(conflictGroups[ruleType], result)
		}
	}

	// 每个类型只保留分数最高的规则
	for _, group := range conflictGroups {
		if len(group) > 0 {
			resolvedResults = append(resolvedResults, group[0])
		}
	}

	return resolvedResults
}

// GetCacheStats 获取缓存统计信息
// @return map[string]interface{} 缓存统计信息
func (re *RuleEngine) GetCacheStats() map[string]interface{} {
	re.cacheMutex.RLock()
	defer re.cacheMutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_cached_rules"] = len(re.ruleCache)
	stats["cache_timeout"] = re.cacheTimeout.String()

	// 统计过期缓存数量
	expiredCount := 0
	now := time.Now()
	for _, cached := range re.ruleCache {
		if now.Sub(cached.CachedAt) > re.cacheTimeout {
			expiredCount++
		}
	}
	stats["expired_cached_rules"] = expiredCount

	return stats
}

// CleanExpiredCache 清理过期缓存
// @return int 清理的缓存数量
func (re *RuleEngine) CleanExpiredCache() int {
	re.cacheMutex.Lock()
	defer re.cacheMutex.Unlock()

	cleanedCount := 0
	now := time.Now()

	for ruleID, cached := range re.ruleCache {
		if now.Sub(cached.CachedAt) > re.cacheTimeout {
			delete(re.ruleCache, ruleID)
			cleanedCount++
		}
	}

	return cleanedCount
}
