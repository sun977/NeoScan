package rule_engine

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Evaluator 条件评估器
type Evaluator struct {
	logger *logrus.Logger
}

// NewEvaluator 创建新的评估器
func NewEvaluator(logger *logrus.Logger) *Evaluator {
	return &Evaluator{
		logger: logger,
	}
}

// EvaluateCondition 评估单个条件
func (e *Evaluator) EvaluateCondition(condition Condition, context *RuleContext) (bool, error) {
	// 从上下文中获取字段值
	fieldValue, exists := e.getFieldValue(condition.Field, context)
	if !exists {
		e.logger.WithFields(logrus.Fields{
			"field": condition.Field,
			"func_name": "evaluator.EvaluateCondition",
		}).Debug("字段不存在")
		return false, nil
	}

	// 根据操作符进行评估
	switch condition.Operator {
	case OpEqual:
		return e.evaluateEqual(fieldValue, condition.Value)
	case OpNotEqual:
		result, err := e.evaluateEqual(fieldValue, condition.Value)
		return !result, err
	case OpGreater:
		return e.evaluateGreater(fieldValue, condition.Value)
	case OpLess:
		return e.evaluateLess(fieldValue, condition.Value)
	case OpGreaterEqual:
		return e.evaluateGreaterEqual(fieldValue, condition.Value)
	case OpLessEqual:
		return e.evaluateLessEqual(fieldValue, condition.Value)
	case OpIn:
		return e.evaluateIn(fieldValue, condition.Value)
	case OpNotIn:
		result, err := e.evaluateIn(fieldValue, condition.Value)
		return !result, err
	case OpContains:
		return e.evaluateContains(fieldValue, condition.Value)
	case OpNotContains:
		result, err := e.evaluateContains(fieldValue, condition.Value)
		return !result, err
	case OpRegex:
		return e.evaluateRegex(fieldValue, condition.Value)
	case OpNotRegex:
		result, err := e.evaluateRegex(fieldValue, condition.Value)
		return !result, err
	default:
		return false, fmt.Errorf("不支持的操作符: %s", condition.Operator)
	}
}

// EvaluateConditions 评估多个条件 (AND逻辑)
func (e *Evaluator) EvaluateConditions(conditions []Condition, context *RuleContext) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	for _, condition := range conditions {
		result, err := e.EvaluateCondition(condition, context)
		if err != nil {
			e.logger.WithFields(logrus.Fields{
				"condition": condition,
				"error": err.Error(),
				"func_name": "evaluator.EvaluateConditions",
			}).Error("条件评估失败")
			return false, err
		}
		
		// AND逻辑：任何一个条件为false，整体结果为false
		if !result {
			return false, nil
		}
	}

	return true, nil
}

// getFieldValue 从上下文中获取字段值
func (e *Evaluator) getFieldValue(field string, context *RuleContext) (interface{}, bool) {
	// 支持嵌套字段访问，如 "scan.result.ports"
	parts := strings.Split(field, ".")
	current := context.Data

	for i, part := range parts {
		if current == nil {
			return nil, false
		}

		value, exists := current[part]
		if !exists {
			return nil, false
		}

		// 如果是最后一个部分，返回值
		if i == len(parts)-1 {
			return value, true
		}

		// 否则继续向下查找
		if nextMap, ok := value.(map[string]interface{}); ok {
			current = nextMap
		} else {
			return nil, false
		}
	}

	return nil, false
}

// evaluateEqual 评估相等条件
func (e *Evaluator) evaluateEqual(fieldValue, expectedValue interface{}) (bool, error) {
	// 类型转换和比较
	fieldStr := e.convertToString(fieldValue)
	expectedStr := e.convertToString(expectedValue)
	
	return fieldStr == expectedStr, nil
}

// evaluateGreater 评估大于条件
func (e *Evaluator) evaluateGreater(fieldValue, expectedValue interface{}) (bool, error) {
	fieldNum, err := e.convertToNumber(fieldValue)
	if err != nil {
		return false, err
	}
	
	expectedNum, err := e.convertToNumber(expectedValue)
	if err != nil {
		return false, err
	}
	
	return fieldNum > expectedNum, nil
}

// evaluateLess 评估小于条件
func (e *Evaluator) evaluateLess(fieldValue, expectedValue interface{}) (bool, error) {
	fieldNum, err := e.convertToNumber(fieldValue)
	if err != nil {
		return false, err
	}
	
	expectedNum, err := e.convertToNumber(expectedValue)
	if err != nil {
		return false, err
	}
	
	return fieldNum < expectedNum, nil
}

// evaluateGreaterEqual 评估大于等于条件
func (e *Evaluator) evaluateGreaterEqual(fieldValue, expectedValue interface{}) (bool, error) {
	fieldNum, err := e.convertToNumber(fieldValue)
	if err != nil {
		return false, err
	}
	
	expectedNum, err := e.convertToNumber(expectedValue)
	if err != nil {
		return false, err
	}
	
	return fieldNum >= expectedNum, nil
}

// evaluateLessEqual 评估小于等于条件
func (e *Evaluator) evaluateLessEqual(fieldValue, expectedValue interface{}) (bool, error) {
	fieldNum, err := e.convertToNumber(fieldValue)
	if err != nil {
		return false, err
	}
	
	expectedNum, err := e.convertToNumber(expectedValue)
	if err != nil {
		return false, err
	}
	
	return fieldNum <= expectedNum, nil
}

// evaluateIn 评估包含条件
func (e *Evaluator) evaluateIn(fieldValue, expectedValue interface{}) (bool, error) {
	fieldStr := e.convertToString(fieldValue)
	
	// 期望值应该是数组
	expectedArray, ok := expectedValue.([]interface{})
	if !ok {
		// 尝试转换为字符串数组
		if strArray, ok := expectedValue.([]string); ok {
			for _, item := range strArray {
				if fieldStr == item {
					return true, nil
				}
			}
			return false, nil
		}
		return false, fmt.Errorf("期望值必须是数组类型")
	}
	
	for _, item := range expectedArray {
		if fieldStr == e.convertToString(item) {
			return true, nil
		}
	}
	
	return false, nil
}

// evaluateContains 评估包含字符串条件
func (e *Evaluator) evaluateContains(fieldValue, expectedValue interface{}) (bool, error) {
	fieldStr := e.convertToString(fieldValue)
	expectedStr := e.convertToString(expectedValue)
	
	return strings.Contains(fieldStr, expectedStr), nil
}

// evaluateRegex 评估正则表达式条件
func (e *Evaluator) evaluateRegex(fieldValue, expectedValue interface{}) (bool, error) {
	fieldStr := e.convertToString(fieldValue)
	pattern := e.convertToString(expectedValue)
	
	matched, err := regexp.MatchString(pattern, fieldStr)
	if err != nil {
		return false, fmt.Errorf("正则表达式错误: %v", err)
	}
	
	return matched, nil
}

// convertToString 将值转换为字符串
func (e *Evaluator) convertToString(value interface{}) string {
	if value == nil {
		return ""
	}
	
	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// convertToNumber 将值转换为数字
func (e *Evaluator) convertToNumber(value interface{}) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("值为空")
	}
	
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("无法转换为数字: %v (类型: %s)", value, reflect.TypeOf(value))
	}
}

// ValidateCondition 验证条件配置
func (e *Evaluator) ValidateCondition(condition Condition) error {
	// 验证字段名
	if condition.Field == "" {
		return fmt.Errorf("字段名不能为空")
	}
	
	// 验证操作符
	validOperators := []string{
		OpEqual, OpNotEqual, OpGreater, OpLess, OpGreaterEqual, OpLessEqual,
		OpIn, OpNotIn, OpContains, OpNotContains, OpRegex, OpNotRegex,
	}
	
	valid := false
	for _, op := range validOperators {
		if condition.Operator == op {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("无效的操作符: %s", condition.Operator)
	}
	
	// 验证值类型
	if condition.Value == nil {
		return fmt.Errorf("条件值不能为空")
	}
	
	// 特定操作符的值类型验证
	switch condition.Operator {
	case OpIn, OpNotIn:
		// 必须是数组类型
		if reflect.TypeOf(condition.Value).Kind() != reflect.Slice {
			return fmt.Errorf("操作符 %s 的值必须是数组类型", condition.Operator)
		}
	case OpRegex, OpNotRegex:
		// 验证正则表达式
		pattern := e.convertToString(condition.Value)
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("无效的正则表达式: %s, 错误: %v", pattern, err)
		}
	}
	
	return nil
}