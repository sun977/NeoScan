/**
 * 工具包:数据转换工具
 * @author: sun977
 * @date: 2025.08.29
 * @description: 提供各种数据类型转换、格式转换和结构体转换的工具函数
 * @func: 数据转换相关的工具函数集合
 */
package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ==================== 基础类型转换 ====================

// StringToInt 字符串转整数，支持默认值
// 参数: str - 待转换的字符串, defaultValue - 转换失败时的默认值
// 返回: 转换后的整数值
func StringToInt(str string, defaultValue int) int {
	if str == "" {
		return defaultValue
	}
	if result, err := strconv.Atoi(str); err == nil {
		return result
	}
	return defaultValue
}

// StringToInt64 字符串转64位整数，支持默认值
// 参数: str - 待转换的字符串, defaultValue - 转换失败时的默认值
// 返回: 转换后的64位整数值
func StringToInt64(str string, defaultValue int64) int64 {
	if str == "" {
		return defaultValue
	}
	if result, err := strconv.ParseInt(str, 10, 64); err == nil {
		return result
	}
	return defaultValue
}

// StringToUint 字符串转无符号整数，支持默认值
// 参数: str - 待转换的字符串, defaultValue - 转换失败时的默认值
// 返回: 转换后的无符号整数值
func StringToUint(str string, defaultValue uint) uint {
	if str == "" {
		return defaultValue
	}
	if result, err := strconv.ParseUint(str, 10, 32); err == nil {
		return uint(result)
	}
	return defaultValue
}

// StringToFloat64 字符串转64位浮点数，支持默认值
// 参数: str - 待转换的字符串, defaultValue - 转换失败时的默认值
// 返回: 转换后的64位浮点数值
func StringToFloat64(str string, defaultValue float64) float64 {
	if str == "" {
		return defaultValue
	}
	if result, err := strconv.ParseFloat(str, 64); err == nil {
		return result
	}
	return defaultValue
}

// StringToBool 字符串转布尔值，支持多种格式
// 参数: str - 待转换的字符串, defaultValue - 转换失败时的默认值
// 返回: 转换后的布尔值
// 支持的true值: "true", "1", "yes", "on", "enabled"
// 支持的false值: "false", "0", "no", "off", "disabled"
func StringToBool(str string, defaultValue bool) bool {
	if str == "" {
		return defaultValue
	}

	str = strings.ToLower(strings.TrimSpace(str))
	switch str {
	case "true", "1", "yes", "on", "enabled":
		return true
	case "false", "0", "no", "off", "disabled":
		return false
	default:
		return defaultValue
	}
}

// ParseIntList 解析整数列表字符串，支持逗号分隔和范围
// 参数: input - 逗号分隔的整数字符串或范围 (e.g., "80,443,1000-2000")
// 返回: 整数切片，如果解析失败则忽略该项
func ParseIntList(input string) []int {
	if input == "" {
		return nil
	}
	var result []int
	// 去重 map
	seen := make(map[int]bool)

	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// 处理范围 (e.g. "1000-2000")
		if strings.Contains(p, "-") {
			rangeParts := strings.Split(p, "-")
			if len(rangeParts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err1 == nil && err2 == nil && start <= end {
					for i := start; i <= end; i++ {
						if !seen[i] {
							result = append(result, i)
							seen[i] = true
						}
					}
				}
			}
			continue
		}

		// 处理单个端口
		if val, err := strconv.Atoi(p); err == nil {
			if !seen[val] {
				result = append(result, val)
				seen[val] = true
			}
		}
	}
	return result
}

// IntToString 整数转字符串
// 参数: value - 待转换的整数值
// 返回: 转换后的字符串
func IntToString(value int) string {
	return strconv.Itoa(value)
}

// Int64ToString 64位整数转字符串
// 参数: value - 待转换的64位整数值
// 返回: 转换后的字符串
func Int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

// UintToString 无符号整数转字符串
// 参数: value - 待转换的无符号整数值
// 返回: 转换后的字符串
func UintToString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

// Float64ToString 64位浮点数转字符串
// 参数: value - 待转换的64位浮点数值, precision - 小数位数
// 返回: 转换后的字符串
func Float64ToString(value float64, precision int) string {
	return strconv.FormatFloat(value, 'f', precision, 64)
}

// BoolToString 布尔值转字符串
// 参数: value - 待转换的布尔值
// 返回: 转换后的字符串 ("true" 或 "false")
func BoolToString(value bool) string {
	return strconv.FormatBool(value)
}

// BoolToInt 布尔值转整数
// 参数: value - 待转换的布尔值
// 返回: true返回1，false返回0
func BoolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// IntToBool 整数转布尔值
// 参数: value - 待转换的整数值
// 返回: 非0值返回true，0值返回false
func IntToBool(value int) bool {
	return value != 0
}

// ==================== 时间转换 ====================

// StringToTime 字符串转时间，支持多种格式
// 参数: str - 待转换的时间字符串, layout - 时间格式（可选，默认使用常见格式）
// 返回: 转换后的时间和错误信息
func StringToTime(str string, layout ...string) (time.Time, error) {
	if str == "" {
		return time.Time{}, fmt.Errorf("时间字符串不能为空")
	}

	// 如果指定了格式，使用指定格式
	if len(layout) > 0 && layout[0] != "" {
		return time.Parse(layout[0], str)
	}

	// 尝试常见的时间格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05+08:00",
		"2006-01-02",
		"15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间字符串: %s", str)
}

// TimeToString 时间转字符串
// 参数: t - 待转换的时间, layout - 时间格式（可选，默认使用标准格式）
// 返回: 转换后的时间字符串
func TimeToString(t time.Time, layout ...string) string {
	if t.IsZero() {
		return ""
	}

	// 如果指定了格式，使用指定格式
	if len(layout) > 0 && layout[0] != "" {
		return t.Format(layout[0])
	}

	// 默认使用标准格式
	return t.Format("2006-01-02 15:04:05")
}

// UnixToTime Unix时间戳转时间
// 参数: timestamp - Unix时间戳（秒）
// 返回: 转换后的时间
func UnixToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

// TimeToUnix 时间转Unix时间戳
// 参数: t - 待转换的时间
// 返回: Unix时间戳（秒）
func TimeToUnix(t time.Time) int64 {
	return t.Unix()
}

// MillisToTime 毫秒时间戳转时间
// 参数: millis - 毫秒时间戳
// 返回: 转换后的时间
func MillisToTime(millis int64) time.Time {
	return time.Unix(millis/1000, (millis%1000)*1000000)
}

// TimeToMillis 时间转毫秒时间戳
// 参数: t - 待转换的时间
// 返回: 毫秒时间戳
func TimeToMillis(t time.Time) int64 {
	return t.UnixNano() / 1000000
}

// ==================== JSON转换 ====================

// StructToJSON 结构体转JSON字符串 - json序列化
// 参数: data - 待转换的数据
// 返回: JSON字符串和错误信息
func StructToJSON(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("结构体转JSON失败: %v", err)
	}
	return string(bytes), nil
}

// JSONToStruct JSON字符串转结构体 - json反序列化
// 参数: jsonStr - JSON字符串, target - 目标结构体指针
// 返回: 错误信息
func JSONToStruct(jsonStr string, target interface{}) error {
	if jsonStr == "" {
		return fmt.Errorf("JSON字符串不能为空")
	}

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("JSON转结构体失败: %v", err)
	}
	return nil
}

// StructToMap 结构体转Map
// 参数: data - 待转换的结构体
// 返回: 转换后的Map和错误信息
func StructToMap(data interface{}) (map[string]interface{}, error) {
	// 先转为JSON，再转为Map
	jsonStr, err := StructToJSON(data)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("结构体转Map失败: %v", err)
	}

	return result, nil
}

// MapToStruct Map转结构体
// 参数: data - 待转换的Map, target - 目标结构体指针
// 返回: 错误信息
func MapToStruct(data map[string]interface{}, target interface{}) error {
	// 先转为JSON，再转为结构体
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("Map转JSON失败: %v", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("JSON转结构体失败: %v", err)
	}

	return nil
}

// ==================== 切片转换 ====================

// StringSliceToIntSlice 字符串切片转整数切片
// 参数: strSlice - 字符串切片
// 返回: 整数切片和错误信息
func StringSliceToIntSlice(strSlice []string) ([]int, error) {
	result := make([]int, len(strSlice))
	for i, str := range strSlice {
		if val, err := strconv.Atoi(str); err != nil {
			return nil, fmt.Errorf("转换失败，索引%d的值'%s'不是有效整数: %v", i, str, err)
		} else {
			result[i] = val
		}
	}
	return result, nil
}

// IntSliceToStringSlice 整数切片转字符串切片
// 参数: intSlice - 整数切片
// 返回: 字符串切片
func IntSliceToStringSlice(intSlice []int) []string {
	result := make([]string, len(intSlice))
	for i, val := range intSlice {
		result[i] = strconv.Itoa(val)
	}
	return result
}

// UintSliceToStringSlice 无符号整数切片转字符串切片
// 参数: uintSlice - 无符号整数切片
// 返回: 字符串切片
func UintSliceToStringSlice(uintSlice []uint) []string {
	result := make([]string, len(uintSlice))
	for i, val := range uintSlice {
		result[i] = strconv.FormatUint(uint64(val), 10)
	}
	return result
}

// StringSliceToUintSlice 字符串切片转无符号整数切片
// 参数: strSlice - 字符串切片
// 返回: 无符号整数切片和错误信息
func StringSliceToUintSlice(strSlice []string) ([]uint, error) {
	result := make([]uint, len(strSlice))
	for i, str := range strSlice {
		if val, err := strconv.ParseUint(str, 10, 32); err != nil {
			return nil, fmt.Errorf("转换失败，索引%d的值'%s'不是有效无符号整数: %v", i, str, err)
		} else {
			result[i] = uint(val)
		}
	}
	return result, nil
}

// ==================== 字符串处理转换 ====================

// CamelToSnake 驼峰命名转蛇形命名
// 参数: str - 驼峰命名字符串
// 返回: 蛇形命名字符串
// 例如: "UserName" -> "user_name"
func CamelToSnake(str string) string {
	if str == "" {
		return ""
	}

	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

// SnakeToCamel 蛇形命名转驼峰命名
// 参数: str - 蛇形命名字符串, firstUpper - 首字母是否大写
// 返回: 驼峰命名字符串
// 例如: "user_name" -> "UserName" (firstUpper=true) 或 "userName" (firstUpper=false)
func SnakeToCamel(str string, firstUpper bool) string {
	if str == "" {
		return ""
	}

	parts := strings.Split(str, "_")
	var result strings.Builder

	for i, part := range parts {
		if part == "" {
			continue
		}

		if i == 0 && !firstUpper {
			result.WriteString(strings.ToLower(part))
		} else {
			result.WriteString(strings.Title(strings.ToLower(part)))
		}
	}

	return result.String()
}

// StringToSlice 字符串按分隔符转切片
// 参数: str - 待分割的字符串, separator - 分隔符
// 返回: 字符串切片
func StringToSlice(str, separator string) []string {
	if str == "" {
		return []string{}
	}
	return strings.Split(str, separator)
}

// SliceToString 切片按分隔符转字符串
// 参数: slice - 字符串切片, separator - 分隔符
// 返回: 连接后的字符串
func SliceToString(slice []string, separator string) string {
	return strings.Join(slice, separator)
}

// ==================== 指针转换 ====================

// StringPtr 字符串转指针
// 参数: s - 字符串值
// 返回: 字符串指针
func StringPtr(s string) *string {
	return &s
}

// IntPtr 整数转指针
// 参数: i - 整数值
// 返回: 整数指针
func IntPtr(i int) *int {
	return &i
}

// UintPtr 无符号整数转指针
// 参数: u - 无符号整数值
// 返回: 无符号整数指针
func UintPtr(u uint) *uint {
	return &u
}

// BoolPtr 布尔值转指针
// 参数: b - 布尔值
// 返回: 布尔值指针
func BoolPtr(b bool) *bool {
	return &b
}

// TimePtr 时间转指针
// 参数: t - 时间值
// 返回: 时间指针
func TimePtr(t time.Time) *time.Time {
	return &t
}

// PtrToString 字符串指针转字符串
// 参数: ptr - 字符串指针, defaultValue - 指针为nil时的默认值
// 返回: 字符串值
func PtrToString(ptr *string, defaultValue string) string {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// PtrToInt 整数指针转整数
// 参数: ptr - 整数指针, defaultValue - 指针为nil时的默认值
// 返回: 整数值
func PtrToInt(ptr *int, defaultValue int) int {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// PtrToUint 无符号整数指针转无符号整数
// 参数: ptr - 无符号整数指针, defaultValue - 指针为nil时的默认值
// 返回: 无符号整数值
func PtrToUint(ptr *uint, defaultValue uint) uint {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// PtrToBool 布尔值指针转布尔值
// 参数: ptr - 布尔值指针, defaultValue - 指针为nil时的默认值
// 返回: 布尔值
func PtrToBool(ptr *bool, defaultValue bool) bool {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// PtrToTime 时间指针转时间
// 参数: ptr - 时间指针, defaultValue - 指针为nil时的默认值
// 返回: 时间值
func PtrToTime(ptr *time.Time, defaultValue time.Time) time.Time {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// ==================== 反射转换 ====================

// ConvertType 通用类型转换（使用反射）
// 参数: src - 源数据, dst - 目标数据指针
// 返回: 错误信息
// 注意: 这是一个通用但性能较低的转换方法，建议优先使用具体的转换函数
func ConvertType(src interface{}, dst interface{}) error {
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	// 检查目标是否为指针
	if dstValue.Kind() != reflect.Ptr {
		return fmt.Errorf("目标参数必须是指针类型")
	}

	// 获取目标指针指向的值
	dstElem := dstValue.Elem()
	if !dstElem.CanSet() {
		return fmt.Errorf("目标值不可设置")
	}

	// 类型相同，直接赋值
	if srcValue.Type() == dstElem.Type() {
		dstElem.Set(srcValue)
		return nil
	}

	// 尝试类型转换
	if srcValue.Type().ConvertibleTo(dstElem.Type()) {
		dstElem.Set(srcValue.Convert(dstElem.Type()))
		return nil
	}

	return fmt.Errorf("无法将类型 %v 转换为 %v", srcValue.Type(), dstElem.Type())
}

// DeepCopy 深拷贝（使用JSON序列化/反序列化）
// 参数: src - 源数据, dst - 目标数据指针
// 返回: 错误信息
// 注意: 这种方法简单但性能较低，且要求数据可JSON序列化
func DeepCopy(src interface{}, dst interface{}) error {
	// 序列化源数据
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("序列化源数据失败: %v", err)
	}

	// 反序列化到目标数据
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("反序列化到目标数据失败: %v", err)
	}

	return nil
}

// ==================== 数据验证转换 ====================

// SafeStringToInt 安全的字符串转整数（带验证）
// 参数: str - 待转换的字符串, min - 最小值, max - 最大值
// 返回: 转换后的整数值和错误信息
func SafeStringToInt(str string, min, max int) (int, error) {
	if str == "" {
		return 0, fmt.Errorf("字符串不能为空")
	}

	value, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("字符串'%s'不是有效整数: %v", str, err)
	}

	if value < min || value > max {
		return 0, fmt.Errorf("值%d超出范围[%d, %d]", value, min, max)
	}

	return value, nil
}

// SafeStringToFloat64 安全的字符串转浮点数（带验证）
// 参数: str - 待转换的字符串, min - 最小值, max - 最大值
// 返回: 转换后的浮点数值和错误信息
func SafeStringToFloat64(str string, min, max float64) (float64, error) {
	if str == "" {
		return 0, fmt.Errorf("字符串不能为空")
	}

	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("字符串'%s'不是有效浮点数: %v", str, err)
	}

	if value < min || value > max {
		return 0, fmt.Errorf("值%f超出范围[%f, %f]", value, min, max)
	}

	return value, nil
}

// ==================== JSON数组转换 ====================

// JSONArrayToStringSlice JSON数组字符串转字符串切片
// 参数: jsonStr - JSON数组字符串，如 ["a","b","c"]
// 返回: 字符串切片和错误信息
// 用于处理MySQL JSON字段到Go切片的转换
func JSONArrayToStringSlice(jsonStr string) ([]string, error) {
	if jsonStr == "" || jsonStr == "null" {
		return []string{}, nil
	}

	var result []string
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("JSON数组转字符串切片失败: %v", err)
	}

	return result, nil
}

// StringSliceToJSONArray 字符串切片转JSON数组字符串
// 参数: slice - 字符串切片
// 返回: JSON数组字符串和错误信息
// 用于处理Go切片到MySQL JSON字段的转换
func StringSliceToJSONArray(slice []string) (string, error) {
	if slice == nil {
		return "null", nil
	}

	if len(slice) == 0 {
		return "[]", nil
	}

	bytes, err := json.Marshal(slice)
	if err != nil {
		return "", fmt.Errorf("字符串切片转JSON数组失败: %v", err)
	}

	return string(bytes), nil
}

// PostgreSQLArrayToStringSlice PostgreSQL数组格式转字符串切片
// 参数: pgArray - PostgreSQL数组字符串，如 {a,b,c}
// 返回: 字符串切片和错误信息
// 用于处理PostgreSQL数组格式到Go切片的转换
func PostgreSQLArrayToStringSlice(pgArray string) ([]string, error) {
	if pgArray == "" || pgArray == "{}" {
		return []string{}, nil
	}

	// 移除大括号
	pgArray = strings.Trim(pgArray, "{}")
	if pgArray == "" {
		return []string{}, nil
	}

	// 按逗号分割
	parts := strings.Split(pgArray, ",")
	result := make([]string, len(parts))

	for i, part := range parts {
		// 去除前后空格和引号
		result[i] = strings.Trim(strings.TrimSpace(part), "\"")
	}

	return result, nil
}

// StringSliceToPostgreSQLArray 字符串切片转PostgreSQL数组格式
// 参数: slice - 字符串切片
// 返回: PostgreSQL数组字符串
// 用于处理Go切片到PostgreSQL数组格式的转换
func StringSliceToPostgreSQLArray(slice []string) string {
	if slice == nil || len(slice) == 0 {
		return "{}"
	}

	// 对每个元素进行引号包装（如果需要）
	quoted := make([]string, len(slice))
	for i, s := range slice {
		// 如果字符串包含特殊字符，需要加引号
		if strings.ContainsAny(s, " ,{}\"\\") {
			quoted[i] = fmt.Sprintf("\"%s\"", strings.ReplaceAll(s, "\"", "\\\""))
		} else {
			quoted[i] = s
		}
	}

	return fmt.Sprintf("{%s}", strings.Join(quoted, ","))
}

// ConvertJSONArrayToPostgreSQLArray JSON数组格式转PostgreSQL数组格式
// 参数: jsonArray - JSON数组字符串，如 ["a","b","c"]
// 返回: PostgreSQL数组字符串，如 {a,b,c}，和错误信息
// 用于不同数据库格式之间的转换
func ConvertJSONArrayToPostgreSQLArray(jsonArray string) (string, error) {
	slice, err := JSONArrayToStringSlice(jsonArray)
	if err != nil {
		return "", err
	}
	return StringSliceToPostgreSQLArray(slice), nil
}

// ConvertPostgreSQLArrayToJSONArray PostgreSQL数组格式转JSON数组格式
// 参数: pgArray - PostgreSQL数组字符串，如 {a,b,c}
// 返回: JSON数组字符串，如 ["a","b","c"]，和错误信息
// 用于不同数据库格式之间的转换
func ConvertPostgreSQLArrayToJSONArray(pgArray string) (string, error) {
	slice, err := PostgreSQLArrayToStringSlice(pgArray)
	if err != nil {
		return "", err
	}
	return StringSliceToJSONArray(slice)
}

// ==================== 特殊转换 ====================

// BytesToString 字节数组转字符串
// 参数: data - 字节数组
// 返回: 字符串
func BytesToString(data []byte) string {
	return string(data)
}

// StringToBytes 字符串转字节数组
// 参数: str - 字符串
// 返回: 字节数组
func StringToBytes(str string) []byte {
	return []byte(str)
}

// InterfaceToString 接口转字符串
// 参数: value - 接口值
// 返回: 字符串表示
func InterfaceToString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

// IsZeroValue 检查值是否为零值
// 参数: value - 待检查的值
// 返回: 是否为零值
func IsZeroValue(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	return v.IsZero()
}
