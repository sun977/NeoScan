/*
 * @author: sun977
 * @date: 2025.09.05
 * @description: 时间工具包测试文件
 * @func: 测试时间工具包的各种功能
 */

package main

import (
	"fmt"
	"time"

	"neomaster/internal/pkg/utils"
)

func main() {
	fmt.Println("=== 时间工具包功能测试 ===")

	// 测试当前时间获取
	fmt.Println("\n1. 当前时间获取:")
	fmt.Printf("当前日期时间: %s\n", utils.GetCurrentDateTime())
	fmt.Printf("当前日期: %s\n", utils.GetCurrentDate())
	fmt.Printf("当前时间: %s\n", utils.GetCurrentTime())
	fmt.Printf("当前时间戳(秒): %d\n", utils.GetCurrentTimestamp())
	fmt.Printf("当前时间戳(毫秒): %d\n", utils.GetCurrentTimestampMilli())

	// 测试时间格式化
	fmt.Println("\n2. 时间格式化:")
	now := time.Now()
	fmt.Printf("标准格式: %s\n", utils.FormatDateTime(now))
	fmt.Printf("日期格式: %s\n", utils.FormatDate(now))
	fmt.Printf("时间格式: %s\n", utils.FormatTime(now))
	fmt.Printf("自定义格式: %s\n", utils.FormatCustom(now, "2006年01月02日 15:04:05"))

	// 测试时间解析
	fmt.Println("\n3. 时间解析:")
	dateTimeStr := "2025-09-05 10:30:45"
	parsedTime, err := utils.ParseDateTime(dateTimeStr)
	if err != nil {
		fmt.Printf("标准格式解析失败: %v\n", err)
	} else {
		fmt.Printf("标准格式解析成功: %s -> %s\n", dateTimeStr, parsedTime.Format(time.RFC3339))
	}

	// 测试带毫秒的时间解析
	dateTimeMilliStr := "2025-09-05 10:30:45.123"
	parsedTimeMilli, err := utils.ParseDateTime(dateTimeMilliStr)
	if err != nil {
		fmt.Printf("毫秒格式解析失败: %v\n", err)
	} else {
		fmt.Printf("毫秒格式解析成功: %s -> %s\n", dateTimeMilliStr, parsedTimeMilli.Format(time.RFC3339Nano))
	}

	// 测试专用毫秒解析函数
	parsedTimeMilli2, err := utils.ParseDateTimeMilli(dateTimeMilliStr)
	if err != nil {
		fmt.Printf("专用毫秒解析失败: %v\n", err)
	} else {
		fmt.Printf("专用毫秒解析成功: %s -> %s\n", dateTimeMilliStr, parsedTimeMilli2.Format(time.RFC3339Nano))
	}

	// 测试时间戳转换
	fmt.Println("\n4. 时间戳转换:")
	timestamp := int64(1725508245) // 2025-09-05 10:30:45 的时间戳
	convertedTime := utils.TimestampToTime(timestamp)
	fmt.Printf("时间戳 %d -> %s\n", timestamp, utils.FormatDateTime(convertedTime))
	backToTimestamp := utils.TimeToTimestamp(convertedTime)
	fmt.Printf("时间 %s -> 时间戳 %d\n", utils.FormatDateTime(convertedTime), backToTimestamp)

	// 测试时间计算
	fmt.Println("\n5. 时间计算:")
	baseTime := time.Date(2025, 9, 5, 10, 30, 45, 0, time.Local)
	fmt.Printf("基准时间: %s\n", utils.FormatDateTime(baseTime))
	fmt.Printf("加3天: %s\n", utils.FormatDateTime(utils.AddDays(baseTime, 3)))
	fmt.Printf("加5小时: %s\n", utils.FormatDateTime(utils.AddHours(baseTime, 5)))
	fmt.Printf("加30分钟: %s\n", utils.FormatDateTime(utils.AddMinutes(baseTime, 30)))
	fmt.Printf("减2天: %s\n", utils.FormatDateTime(utils.AddDays(baseTime, -2)))

	// 测试时间差计算
	fmt.Println("\n6. 时间差计算:")
	time1 := time.Date(2025, 9, 10, 15, 30, 0, 0, time.Local)
	time2 := time.Date(2025, 9, 5, 10, 30, 0, 0, time.Local)
	fmt.Printf("时间1: %s\n", utils.FormatDateTime(time1))
	fmt.Printf("时间2: %s\n", utils.FormatDateTime(time2))
	fmt.Printf("相差天数: %d天\n", utils.DiffDays(time1, time2))
	fmt.Printf("相差小时: %d小时\n", utils.DiffHours(time1, time2))
	fmt.Printf("相差分钟: %d分钟\n", utils.DiffMinutes(time1, time2))

	// 测试日期判断
	fmt.Println("\n7. 日期判断:")
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	fmt.Printf("今天是否为今天: %t\n", utils.IsToday(today))
	fmt.Printf("昨天是否为昨天: %t\n", utils.IsYesterday(yesterday))
	fmt.Printf("明天是否为明天: %t\n", utils.IsTomorrow(tomorrow))

	// 测试时间范围获取
	fmt.Println("\n8. 时间范围获取:")
	testDate := time.Date(2025, 9, 5, 15, 30, 45, 0, time.Local)
	fmt.Printf("测试日期: %s\n", utils.FormatDateTime(testDate))
	fmt.Printf("当天开始: %s\n", utils.FormatDateTime(utils.GetStartOfDay(testDate)))
	fmt.Printf("当天结束: %s\n", utils.FormatDateTime(utils.GetEndOfDay(testDate)))
	fmt.Printf("本周开始: %s\n", utils.FormatDateTime(utils.GetStartOfWeek(testDate)))
	fmt.Printf("本周结束: %s\n", utils.FormatDateTime(utils.GetEndOfWeek(testDate)))
	fmt.Printf("本月开始: %s\n", utils.FormatDateTime(utils.GetStartOfMonth(testDate)))
	fmt.Printf("本月结束: %s\n", utils.FormatDateTime(utils.GetEndOfMonth(testDate)))

	// 测试年龄计算
	fmt.Println("\n9. 年龄计算:")
	birthday := time.Date(1990, 5, 15, 0, 0, 0, 0, time.Local)
	age := utils.GetAge(birthday)
	fmt.Printf("生日: %s\n", utils.FormatDate(birthday))
	fmt.Printf("年龄: %d岁\n", age)

	// 测试时间间隔格式化
	fmt.Println("\n10. 时间间隔格式化:")
	duration1 := 2*24*time.Hour + 3*time.Hour + 45*time.Minute + 30*time.Second
	duration2 := 5*time.Hour + 20*time.Minute
	duration3 := 45 * time.Second
	fmt.Printf("间隔1: %s\n", utils.FormatDuration(duration1))
	fmt.Printf("间隔2: %s\n", utils.FormatDuration(duration2))
	fmt.Printf("间隔3: %s\n", utils.FormatDuration(duration3))

	// 测试闰年判断
	fmt.Println("\n11. 闰年判断:")
	years := []int{2020, 2021, 2024, 2025, 2000, 1900}
	for _, year := range years {
		fmt.Printf("%d年是否为闰年: %t\n", year, utils.IsLeapYear(year))
	}

	// 测试月份天数
	fmt.Println("\n12. 月份天数:")
	months := []time.Month{time.January, time.February, time.April, time.December}
	for _, month := range months {
		days2024 := utils.GetDaysInMonth(2024, month) // 闰年
		days2025 := utils.GetDaysInMonth(2025, month) // 平年
		fmt.Printf("%s: 2024年%d天, 2025年%d天\n", month.String(), days2024, days2025)
	}

	// 测试时区相关
	fmt.Println("\n13. 时区相关:")
	localTime := time.Now()
	offset := utils.TimeZoneOffset(localTime)
	fmt.Printf("当前时区偏移: UTC%+d\n", offset)

	// 尝试转换时区
	utcTime, err := utils.ConvertTimeZone(localTime, "UTC")
	if err != nil {
		fmt.Printf("时区转换失败: %v\n", err)
	} else {
		fmt.Printf("本地时间: %s\n", utils.FormatDateTime(localTime))
		fmt.Printf("UTC时间: %s\n", utils.FormatDateTime(utcTime))
	}

	fmt.Println("\n=== 测试完成 ===")
}
