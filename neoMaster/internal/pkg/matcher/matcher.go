package matcher

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// MatchRule 定义匹配规则树
// 既可以是条件节点(Leaf)，也可以是逻辑节点(Branch)
type MatchRule struct {
	// --- 逻辑节点 (Branch) ---
	And []MatchRule `json:"and,omitempty"`
	Or  []MatchRule `json:"or,omitempty"`

	// --- 条件节点 (Leaf) ---
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`
}

// Match 评估数据是否符合规则
func Match(data interface{}, rule MatchRule) (bool, error) {
	// 1. 处理逻辑节点 (Branch)
	// 优先处理 And
	if len(rule.And) > 0 {
		for _, subRule := range rule.And {
			matched, err := Match(data, subRule)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil // And 只要有一个不匹配，整体就不匹配
			}
		}
		return true, nil // 所有都匹配
	}

	// 处理 Or
	if len(rule.Or) > 0 {
		for _, subRule := range rule.Or {
			matched, err := Match(data, subRule)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil // Or 只要有一个匹配，整体就匹配
			}
		}
		return false, nil // 所有都不匹配
	}

	// 2. 处理条件节点 (Leaf)
	// 如果既没有 And 也没有 Or，则视为条件节点
	if rule.Field == "" && rule.Operator == "" {
		// 空规则，默认匹配？或者报错？
		// 这里暂定为空规则匹配 (true)，类似于空过滤器不过滤任何东西
		return true, nil
	}

	// 获取字段值
	fieldValue, exists := getFieldValue(data, rule.Field)

	// 特殊处理 exists 和 is_null/is_not_null 操作符，它们不一定需要字段值存在
	switch rule.Operator {
	case "exists":
		return exists, nil
	case "is_null":
		return !exists || fieldValue == nil, nil
	case "is_not_null":
		return exists && fieldValue != nil, nil
	}

	// 如果字段不存在，且不是上述操作符，默认不匹配
	if !exists {
		return false, nil
	}

	// 执行具体匹配逻辑
	return evaluateCondition(fieldValue, rule.Operator, rule.Value)
}

// ParseJSON 解析 JSON 规则字符串
func ParseJSON(jsonStr string) (MatchRule, error) {
	var rule MatchRule
	err := json.Unmarshal([]byte(jsonStr), &rule)
	return rule, err
}

// getFieldValue 获取嵌套字段值 (支持 "meta.os" 这种点号语法)
func getFieldValue(data interface{}, fieldPath string) (interface{}, bool) {
	parts := strings.Split(fieldPath, ".")
	current := data

	for _, part := range parts {
		if current == nil {
			return nil, false
		}

		// 处理 map
		val := reflect.ValueOf(current)
		if val.Kind() == reflect.Map {
			// key 必须是 string
			keyVal := val.MapIndex(reflect.ValueOf(part))
			if !keyVal.IsValid() {
				return nil, false
			}
			current = keyVal.Interface()
			continue
		}

		// 处理 struct (暂不支持 struct tag 查找，简单起见仅支持导出字段名匹配)
		// 如果需要支持 json tag，需要更复杂的反射逻辑
		if val.Kind() == reflect.Struct {
			fieldVal := val.FieldByName(part)
			if !fieldVal.IsValid() {
				return nil, false
			}
			current = fieldVal.Interface()
			continue
		}

		// 无法继续深入
		return nil, false
	}

	return current, true
}

// evaluateCondition 评估单个条件
func evaluateCondition(actual interface{}, operator string, expected interface{}) (bool, error) {
	switch operator {
	case "equals":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected), nil
	case "not_equals":
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected), nil
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected)), nil
	case "not_contains":
		return !strings.Contains(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected)), nil
	case "starts_with":
		return strings.HasPrefix(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected)), nil
	case "ends_with":
		return strings.HasSuffix(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected)), nil
	case "regex":
		pattern, ok := expected.(string)
		if !ok {
			return false, fmt.Errorf("regex pattern must be string")
		}
		match, err := regexp.MatchString(pattern, fmt.Sprintf("%v", actual))
		return match, err
	case "like":
		// 简单的 SQL like 实现: % -> .*, _ -> .
		pattern, ok := expected.(string)
		if !ok {
			return false, fmt.Errorf("like pattern must be string")
		}
		regexPattern := "^" + strings.ReplaceAll(strings.ReplaceAll(regexp.QuoteMeta(pattern), "%", ".*"), "_", ".") + "$"
		match, err := regexp.MatchString(regexPattern, fmt.Sprintf("%v", actual))
		return match, err
	case "in", "not_in":
		// expected 应该是一个 slice
		expectedVal := reflect.ValueOf(expected)
		if expectedVal.Kind() != reflect.Slice && expectedVal.Kind() != reflect.Array {
			return false, fmt.Errorf("in/not_in expected value must be a list")
		}
		found := false
		actualStr := fmt.Sprintf("%v", actual)
		for i := 0; i < expectedVal.Len(); i++ {
			if fmt.Sprintf("%v", expectedVal.Index(i).Interface()) == actualStr {
				found = true
				break
			}
		}
		if operator == "in" {
			return found, nil
		}
		return !found, nil

	// 数值比较
	case "greater_than", "less_than", "greater_than_or_equal", "less_than_or_equal":
		return compareNumbers(actual, operator, expected)

	case "cidr":
		ipStr, ok := actual.(string)
		if !ok {
			return false, nil // 不是 IP 字符串，不匹配
		}
		cidrStr, ok := expected.(string)
		if !ok {
			return false, fmt.Errorf("cidr expected value must be string")
		}
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return false, err
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return false, nil // 无效 IP
		}
		return ipNet.Contains(ip), nil

	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// compareNumbers 数值比较辅助函数
func compareNumbers(actual interface{}, op string, expected interface{}) (bool, error) {
	v1, err := toFloat64(actual)
	if err != nil {
		return false, nil // 转换失败视为不匹配 (Fail Safe)
	}
	v2, err := toFloat64(expected)
	if err != nil {
		return false, fmt.Errorf("expected value is not a number: %v", expected)
	}

	switch op {
	case "greater_than":
		return v1 > v2, nil
	case "less_than":
		return v1 < v2, nil
	case "greater_than_or_equal":
		return v1 >= v2, nil
	case "less_than_or_equal":
		return v1 <= v2, nil
	}
	return false, nil
}

func toFloat64(v interface{}) (float64, error) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return val.Float(), nil
	case reflect.String:
		// 尝试解析字符串数字
		f, err := strconv.ParseFloat(val.String(), 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse string to number: %v", v)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("not a number: type=%T value=%v", v, v)
	}
}
