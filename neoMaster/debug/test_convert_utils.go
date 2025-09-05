/**
 * 测试文件:数据转换工具测试
 * @author: sun977
 * @date: 2025.08.29
 * @description: 测试convert.go中的各种数据转换功能
 * @func: 转换工具函数的测试用例
 */
package main

import (
	"fmt"
	"log"
	"neomaster/internal/pkg/utils"
	"time"
)

// TestUser 测试用户结构体
type TestUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Active   bool   `json:"active"`
}

func main() {
	fmt.Println("=== 数据转换工具测试 ===")
	fmt.Println()

	// 测试基础类型转换
	testBasicConversions()
	fmt.Println()

	// 测试时间转换
	testTimeConversions()
	fmt.Println()

	// 测试JSON转换
	testJSONConversions()
	fmt.Println()

	// 测试切片转换
	testSliceConversions()
	fmt.Println()

	// 测试字符串处理转换
	testStringConversions()
	fmt.Println()

	// 测试指针转换
	testPointerConversions()
	fmt.Println()

	// 测试安全转换
	testSafeConversions()
	fmt.Println()

	// 测试特殊转换
	testSpecialConversions()
	fmt.Println()

	fmt.Println("=== 所有转换工具测试完成 ===")
}

// testBasicConversions 测试基础类型转换
func testBasicConversions() {
	fmt.Println("--- 基础类型转换测试 ---")

	// 字符串转整数
	result1 := utils.StringToInt("123", 0)
	fmt.Printf("StringToInt('123', 0) = %d\n", result1)

	result2 := utils.StringToInt("invalid", 999)
	fmt.Printf("StringToInt('invalid', 999) = %d\n", result2)

	// 字符串转布尔值
	result3 := utils.StringToBool("true", false)
	fmt.Printf("StringToBool('true', false) = %t\n", result3)

	result4 := utils.StringToBool("yes", false)
	fmt.Printf("StringToBool('yes', false) = %t\n", result4)

	result5 := utils.StringToBool("0", true)
	fmt.Printf("StringToBool('0', true) = %t\n", result5)

	// 整数转字符串
	result6 := utils.IntToString(456)
	fmt.Printf("IntToString(456) = '%s'\n", result6)

	// 布尔值转整数
	result7 := utils.BoolToInt(true)
	fmt.Printf("BoolToInt(true) = %d\n", result7)

	result8 := utils.BoolToInt(false)
	fmt.Printf("BoolToInt(false) = %d\n", result8)

	// 整数转布尔值
	result9 := utils.IntToBool(1)
	fmt.Printf("IntToBool(1) = %t\n", result9)

	result10 := utils.IntToBool(0)
	fmt.Printf("IntToBool(0) = %t\n", result10)
}

// testTimeConversions 测试时间转换
func testTimeConversions() {
	fmt.Println("--- 时间转换测试 ---")

	// 字符串转时间
	if t1, err := utils.StringToTime("2023-12-25 15:30:45"); err == nil {
		fmt.Printf("StringToTime('2023-12-25 15:30:45') = %v\n", t1)
	} else {
		fmt.Printf("StringToTime error: %v\n", err)
	}

	// 时间转字符串
	now := time.Now()
	result1 := utils.TimeToString(now)
	fmt.Printf("TimeToString(now) = '%s'\n", result1)

	// 自定义格式
	result2 := utils.TimeToString(now, "2006/01/02")
	fmt.Printf("TimeToString(now, '2006/01/02') = '%s'\n", result2)

	// Unix时间戳转换
	timestamp := int64(1703505045) // 2023-12-25 15:30:45
	t2 := utils.UnixToTime(timestamp)
	fmt.Printf("UnixToTime(%d) = %v\n", timestamp, t2)

	unixTime := utils.TimeToUnix(t2)
	fmt.Printf("TimeToUnix(t2) = %d\n", unixTime)

	// 毫秒时间戳转换
	millis := int64(1703505045000)
	t3 := utils.MillisToTime(millis)
	fmt.Printf("MillisToTime(%d) = %v\n", millis, t3)

	millisResult := utils.TimeToMillis(t3)
	fmt.Printf("TimeToMillis(t3) = %d\n", millisResult)
}

// testJSONConversions 测试JSON转换
func testJSONConversions() {
	fmt.Println("--- JSON转换测试 ---")

	// 结构体转JSON
	user := TestUser{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
		Active:   true,
	}

	if jsonStr, err := utils.StructToJSON(user); err == nil {
		fmt.Printf("StructToJSON(user) = %s\n", jsonStr)

		// JSON转结构体
		var newUser TestUser
		if err := utils.JSONToStruct(jsonStr, &newUser); err == nil {
			fmt.Printf("JSONToStruct result: %+v\n", newUser)
		} else {
			fmt.Printf("JSONToStruct error: %v\n", err)
		}
	} else {
		fmt.Printf("StructToJSON error: %v\n", err)
	}

	// 结构体转Map
	if userMap, err := utils.StructToMap(user); err == nil {
		fmt.Printf("StructToMap(user) = %+v\n", userMap)

		// Map转结构体
		var mapUser TestUser
		if err := utils.MapToStruct(userMap, &mapUser); err == nil {
			fmt.Printf("MapToStruct result: %+v\n", mapUser)
		} else {
			fmt.Printf("MapToStruct error: %v\n", err)
		}
	} else {
		fmt.Printf("StructToMap error: %v\n", err)
	}
}

// testSliceConversions 测试切片转换
func testSliceConversions() {
	fmt.Println("--- 切片转换测试 ---")

	// 字符串切片转整数切片
	strSlice := []string{"1", "2", "3", "4", "5"}
	if intSlice, err := utils.StringSliceToIntSlice(strSlice); err == nil {
		fmt.Printf("StringSliceToIntSlice(%v) = %v\n", strSlice, intSlice)

		// 整数切片转字符串切片
		newStrSlice := utils.IntSliceToStringSlice(intSlice)
		fmt.Printf("IntSliceToStringSlice(%v) = %v\n", intSlice, newStrSlice)
	} else {
		fmt.Printf("StringSliceToIntSlice error: %v\n", err)
	}

	// 无符号整数切片转换
	uintSlice := []uint{10, 20, 30}
	uintStrSlice := utils.UintSliceToStringSlice(uintSlice)
	fmt.Printf("UintSliceToStringSlice(%v) = %v\n", uintSlice, uintStrSlice)

	if newUintSlice, err := utils.StringSliceToUintSlice(uintStrSlice); err == nil {
		fmt.Printf("StringSliceToUintSlice(%v) = %v\n", uintStrSlice, newUintSlice)
	} else {
		fmt.Printf("StringSliceToUintSlice error: %v\n", err)
	}
}

// testStringConversions 测试字符串处理转换
func testStringConversions() {
	fmt.Println("--- 字符串处理转换测试 ---")

	// 驼峰转蛇形
	camelStr := "UserNameEmail"
	snakeStr := utils.CamelToSnake(camelStr)
	fmt.Printf("CamelToSnake('%s') = '%s'\n", camelStr, snakeStr)

	// 蛇形转驼峰
	snakeStr2 := "user_name_email"
	camelStr2 := utils.SnakeToCamel(snakeStr2, true)
	fmt.Printf("SnakeToCamel('%s', true) = '%s'\n", snakeStr2, camelStr2)

	camelStr3 := utils.SnakeToCamel(snakeStr2, false)
	fmt.Printf("SnakeToCamel('%s', false) = '%s'\n", snakeStr2, camelStr3)

	// 字符串分割和连接
	str := "apple,banana,orange"
	slice := utils.StringToSlice(str, ",")
	fmt.Printf("StringToSlice('%s', ',') = %v\n", str, slice)

	newStr := utils.SliceToString(slice, "|")
	fmt.Printf("SliceToString(%v, '|') = '%s'\n", slice, newStr)
}

// testPointerConversions 测试指针转换
func testPointerConversions() {
	fmt.Println("--- 指针转换测试 ---")

	// 值转指针
	strPtr := utils.StringPtr("hello")
	fmt.Printf("StringPtr('hello') = %p, value = '%s'\n", strPtr, *strPtr)

	intPtr := utils.IntPtr(42)
	fmt.Printf("IntPtr(42) = %p, value = %d\n", intPtr, *intPtr)

	boolPtr := utils.BoolPtr(true)
	fmt.Printf("BoolPtr(true) = %p, value = %t\n", boolPtr, *boolPtr)

	// 指针转值
	strValue := utils.PtrToString(strPtr, "default")
	fmt.Printf("PtrToString(strPtr, 'default') = '%s'\n", strValue)

	intValue := utils.PtrToInt(intPtr, 0)
	fmt.Printf("PtrToInt(intPtr, 0) = %d\n", intValue)

	boolValue := utils.PtrToBool(boolPtr, false)
	fmt.Printf("PtrToBool(boolPtr, false) = %t\n", boolValue)

	// nil指针测试
	var nilStrPtr *string
	nilStrValue := utils.PtrToString(nilStrPtr, "nil_default")
	fmt.Printf("PtrToString(nil, 'nil_default') = '%s'\n", nilStrValue)
}

// testSafeConversions 测试安全转换
func testSafeConversions() {
	fmt.Println("--- 安全转换测试 ---")

	// 安全字符串转整数
	if value, err := utils.SafeStringToInt("25", 1, 100); err == nil {
		fmt.Printf("SafeStringToInt('25', 1, 100) = %d\n", value)
	} else {
		fmt.Printf("SafeStringToInt error: %v\n", err)
	}

	// 超出范围测试
	if _, err := utils.SafeStringToInt("150", 1, 100); err != nil {
		fmt.Printf("SafeStringToInt('150', 1, 100) error: %v\n", err)
	}

	// 安全字符串转浮点数
	if value, err := utils.SafeStringToFloat64("3.14", 0.0, 10.0); err == nil {
		fmt.Printf("SafeStringToFloat64('3.14', 0.0, 10.0) = %f\n", value)
	} else {
		fmt.Printf("SafeStringToFloat64 error: %v\n", err)
	}

	// 无效输入测试
	if _, err := utils.SafeStringToFloat64("invalid", 0.0, 10.0); err != nil {
		fmt.Printf("SafeStringToFloat64('invalid', 0.0, 10.0) error: %v\n", err)
	}
}

// testSpecialConversions 测试特殊转换
func testSpecialConversions() {
	fmt.Println("--- 特殊转换测试 ---")

	// 字节数组和字符串转换
	str := "Hello, 世界!"
	bytes := utils.StringToBytes(str)
	fmt.Printf("StringToBytes('%s') = %v\n", str, bytes)

	newStr := utils.BytesToString(bytes)
	fmt.Printf("BytesToString(%v) = '%s'\n", bytes, newStr)

	// 接口转字符串
	var values []interface{} = []interface{}{123, "hello", true, 3.14, nil}
	for i, value := range values {
		strValue := utils.InterfaceToString(value)
		fmt.Printf("InterfaceToString(values[%d]) = '%s'\n", i, strValue)
	}

	// 零值检查
	var zeroValues []interface{} = []interface{}{0, "", false, nil, time.Time{}}
	for i, value := range zeroValues {
		isZero := utils.IsZeroValue(value)
		fmt.Printf("IsZeroValue(zeroValues[%d]) = %t\n", i, isZero)
	}

	// 深拷贝测试
	originalUser := TestUser{
		ID:       100,
		Username: "original",
		Email:    "original@test.com",
		Age:      30,
		Active:   true,
	}

	var copiedUser TestUser
	if err := utils.DeepCopy(originalUser, &copiedUser); err == nil {
		fmt.Printf("DeepCopy original: %+v\n", originalUser)
		fmt.Printf("DeepCopy result: %+v\n", copiedUser)
		
		// 修改原始数据，验证深拷贝
		originalUser.Username = "modified"
		fmt.Printf("After modification - original: %+v\n", originalUser)
		fmt.Printf("After modification - copied: %+v\n", copiedUser)
	} else {
		log.Printf("DeepCopy error: %v\n", err)
	}
}