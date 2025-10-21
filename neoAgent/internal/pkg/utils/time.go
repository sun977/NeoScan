/*
 * @author: sun977
 * @date: 2025.09.05
 * @description: 时间工具包
 * @func: 提供时间格式化、解析、计算等常用工具函数
 */

package utils

import (
	"fmt"
	"time"
)

// 常用时间格式常量
const (
	// DateTimeFormat 标准日期时间格式 "2006-01-02 15:04:05"
	DateTimeFormat = "2006-01-02 15:04:05"
	// DateTimeMilliFormat 带毫秒的日期时间格式 "2006-01-02 15:04:05.000"
	DateTimeMilliFormat = "2006-01-02 15:04:05.000"
	// DateFormat 日期格式 "2006-01-02"
	DateFormat = "2006-01-02"
	// TimeFormat 时间格式 "15:04:05"
	TimeFormat = "15:04:05"
	// DateTimeCompactFormat 紧凑日期时间格式 "20060102150405"
	DateTimeCompactFormat = "20060102150405"
	// ISO8601Format ISO8601标准格式 "2006-01-02T15:04:05Z07:00"
	ISO8601Format = time.RFC3339
	// TimestampFormat Unix时间戳格式（秒）
	TimestampFormat = "1136239445"
)

// FormatDateTime 格式化时间为标准日期时间字符串
// 参数: t - 要格式化的时间
// 返回: 格式化后的字符串 "2006-01-02 15:04:05"
func FormatDateTime(t time.Time) string {
	return t.Format(DateTimeFormat)
}

// FormatDate 格式化时间为日期字符串
// 参数: t - 要格式化的时间
// 返回: 格式化后的字符串 "2006-01-02"
func FormatDate(t time.Time) string {
	return t.Format(DateFormat)
}

// FormatTime 格式化时间为时间字符串
// 参数: t - 要格式化的时间
// 返回: 格式化后的字符串 "15:04:05"
func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// FormatCustom 使用自定义格式格式化时间
// 参数: t - 要格式化的时间, layout - 自定义格式 "2006年01月02日 15:04:05"
// 返回: 格式化后的字符串
func FormatCustom(t time.Time, layout string) string {
	return t.Format(layout)
}

// ParseDateTime 解析日期时间字符串（智能识别格式）
// 参数: dateTimeStr - 日期时间字符串 "2006-01-02 15:04:05" 或 "2006-01-02 15:04:05.000"
// 返回: 解析后的时间对象和错误信息
func ParseDateTime(dateTimeStr string) (time.Time, error) {
	// 首先尝试标准格式（不带毫秒）
	if t, err := time.Parse(DateTimeFormat, dateTimeStr); err == nil {
		return t, nil
	}
	// 然后尝试带毫秒的格式
	if t, err := time.Parse(DateTimeMilliFormat, dateTimeStr); err == nil {
		return t, nil
	}
	// 最后尝试 ISO8601 格式
	return time.Parse(time.RFC3339, dateTimeStr)
}

// ParseDateTimeMilli 解析带毫秒的日期时间字符串
// 参数: dateTimeStr - 日期时间字符串 "2006-01-02 15:04:05.000"
// 返回: 解析后的时间对象和错误信息
func ParseDateTimeMilli(dateTimeStr string) (time.Time, error) {
	return time.Parse(DateTimeMilliFormat, dateTimeStr)
}

// ParseDate 解析日期字符串
// 参数: dateStr - 日期字符串 "2006-01-02"
// 返回: 解析后的时间对象和错误信息
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse(DateFormat, dateStr)
}

// ParseCustom 使用自定义格式解析时间字符串
// 参数: timeStr - 时间字符串, layout - 自定义格式
// 返回: 解析后的时间对象和错误信息
func ParseCustom(timeStr, layout string) (time.Time, error) {
	return time.Parse(layout, timeStr)
}

// GetCurrentDateTime 获取当前日期时间字符串
// 返回: 当前时间的标准格式字符串 "2006-01-02 15:04:05"
func GetCurrentDateTime() string {
	return time.Now().Format(DateTimeFormat)
}

// GetCurrentDate 获取当前日期字符串
// 返回: 当前日期的格式字符串 "2006-01-02"
func GetCurrentDate() string {
	return time.Now().Format(DateFormat)
}

// GetCurrentTime 获取当前时间字符串
// 返回: 当前时间的格式字符串 "15:04:05"
func GetCurrentTime() string {
	return time.Now().Format(TimeFormat)
}

// GetCurrentTimestamp 获取当前Unix时间戳（秒）
// 返回: 当前时间的Unix时间戳
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// GetCurrentTimestampMilli 获取当前Unix时间戳（毫秒）
// 返回: 当前时间的Unix时间戳（毫秒）
func GetCurrentTimestampMilli() int64 {
	return time.Now().UnixMilli()
}

// TimestampToTime 将Unix时间戳转换为时间对象
// 参数: timestamp - Unix时间戳（秒）
// 返回: 时间对象
func TimestampToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

// TimestampMilliToTime 将Unix时间戳（毫秒）转换为时间对象
// 参数: timestampMilli - Unix时间戳（毫秒）
// 返回: 时间对象
func TimestampMilliToTime(timestampMilli int64) time.Time {
	return time.UnixMilli(timestampMilli)
}

// TimeToTimestamp 将时间对象转换为Unix时间戳（秒）
// 参数: t - 时间对象
// 返回: Unix时间戳（秒）
func TimeToTimestamp(t time.Time) int64 {
	return t.Unix()
}

// TimeToTimestampMilli 将时间对象转换为Unix时间戳（毫秒）
// 参数: t - 时间对象
// 返回: Unix时间戳（毫秒）
func TimeToTimestampMilli(t time.Time) int64 {
	return t.UnixMilli()
}

// AddDays 给时间添加指定天数
// 参数: t - 基准时间, days - 要添加的天数（可以为负数）
// 返回: 计算后的时间
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddHours 给时间添加指定小时数
// 参数: t - 基准时间, hours - 要添加的小时数（可以为负数）
// 返回: 计算后的时间
func AddHours(t time.Time, hours int) time.Time {
	return t.Add(time.Duration(hours) * time.Hour)
}

// AddMinutes 给时间添加指定分钟数
// 参数: t - 基准时间, minutes - 要添加的分钟数（可以为负数）
// 返回: 计算后的时间
func AddMinutes(t time.Time, minutes int) time.Time {
	return t.Add(time.Duration(minutes) * time.Minute)
}

// AddSeconds 给时间添加指定秒数
// 参数: t - 基准时间, seconds - 要添加的秒数（可以为负数）
// 返回: 计算后的时间
func AddSeconds(t time.Time, seconds int) time.Time {
	return t.Add(time.Duration(seconds) * time.Second)
}

// DiffDays 计算两个时间之间的天数差
// 参数: t1, t2 - 要比较的两个时间
// 返回: 天数差（t1 - t2）
func DiffDays(t1, t2 time.Time) int {
	duration := t1.Sub(t2)
	return int(duration.Hours() / 24)
}

// DiffHours 计算两个时间之间的小时数差
// 参数: t1, t2 - 要比较的两个时间
// 返回: 小时数差（t1 - t2）
func DiffHours(t1, t2 time.Time) int {
	duration := t1.Sub(t2)
	return int(duration.Hours())
}

// DiffMinutes 计算两个时间之间的分钟数差
// 参数: t1, t2 - 要比较的两个时间
// 返回: 分钟数差（t1 - t2）
func DiffMinutes(t1, t2 time.Time) int {
	duration := t1.Sub(t2)
	return int(duration.Minutes())
}

// DiffSeconds 计算两个时间之间的秒数差
// 参数: t1, t2 - 要比较的两个时间
// 返回: 秒数差（t1 - t2）
func DiffSeconds(t1, t2 time.Time) int {
	duration := t1.Sub(t2)
	return int(duration.Seconds())
}

// IsToday 判断给定时间是否为今天
// 参数: t - 要判断的时间
// 返回: 是否为今天
func IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.YearDay() == now.YearDay()
}

// IsYesterday 判断给定时间是否为昨天
// 参数: t - 要判断的时间
// 返回: 是否为昨天
func IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() && t.YearDay() == yesterday.YearDay()
}

// IsTomorrow 判断给定时间是否为明天
// 参数: t - 要判断的时间
// 返回: 是否为明天
func IsTomorrow(t time.Time) bool {
	tomorrow := time.Now().AddDate(0, 0, 1)
	return t.Year() == tomorrow.Year() && t.YearDay() == tomorrow.YearDay()
}

// GetStartOfDay 获取指定日期的开始时间（00:00:00）
// 参数: t - 指定日期
// 返回: 该日期的开始时间
func GetStartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// GetEndOfDay 获取指定日期的结束时间（23:59:59）
// 参数: t - 指定日期
// 返回: 该日期的结束时间
func GetEndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// GetStartOfWeek 获取指定日期所在周的开始时间（周一00:00:00）
// 参数: t - 指定日期
// 返回: 该周的开始时间
func GetStartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 将周日从0调整为7
	}
	days := weekday - 1 // 计算到周一的天数差
	startOfWeek := t.AddDate(0, 0, -days)
	return GetStartOfDay(startOfWeek)
}

// GetEndOfWeek 获取指定日期所在周的结束时间（周日23:59:59）
// 参数: t - 指定日期
// 返回: 该周的结束时间
func GetEndOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 将周日从0调整为7
	}
	days := 7 - weekday // 计算到周日的天数差
	endOfWeek := t.AddDate(0, 0, days)
	return GetEndOfDay(endOfWeek)
}

// GetStartOfMonth 获取指定日期所在月的开始时间（1号00:00:00）
// 参数: t - 指定日期
// 返回: 该月的开始时间
func GetStartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// GetEndOfMonth 获取指定日期所在月的结束时间（月末23:59:59）
// 参数: t - 指定日期
// 返回: 该月的结束时间
func GetEndOfMonth(t time.Time) time.Time {
	startOfNextMonth := GetStartOfMonth(t).AddDate(0, 1, 0)
	return startOfNextMonth.Add(-time.Nanosecond)
}

// GetAge 根据生日计算年龄
// 参数: birthday - 生日时间
// 返回: 年龄
func GetAge(birthday time.Time) int {
	now := time.Now()
	age := now.Year() - birthday.Year()

	// 如果今年的生日还没到，年龄减1
	if now.Month() < birthday.Month() || (now.Month() == birthday.Month() && now.Day() < birthday.Day()) {
		age--
	}

	return age
}

// FormatDuration 格式化时间间隔为可读字符串
// 参数: d - 时间间隔
// 返回: 格式化后的字符串，如 "2天3小时4分钟5秒"
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var result string
	if days > 0 {
		result += fmt.Sprintf("%d天", days)
	}
	if hours > 0 {
		result += fmt.Sprintf("%d小时", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%d分钟", minutes)
	}
	if seconds > 0 || result == "" {
		result += fmt.Sprintf("%d秒", seconds)
	}

	return result
}

// IsLeapYear 判断指定年份是否为闰年
// 参数: year - 年份
// 返回: 是否为闰年
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// GetDaysInMonth 获取指定年月的天数
// 参数: year - 年份, month - 月份
// 返回: 该月的天数
func GetDaysInMonth(year int, month time.Month) int {
	// 获取下个月的第一天，然后减去一天，就是本月的最后一天
	firstDayOfNextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstDayOfNextMonth.AddDate(0, 0, -1)
	return lastDayOfMonth.Day()
}

// TimeZoneOffset 获取时区偏移量（相对于UTC的小时数）
// 参数: t - 时间对象
// 返回: 时区偏移量（小时）
func TimeZoneOffset(t time.Time) int {
	_, offset := t.Zone()
	return offset / 3600 // 转换为小时
}

// ConvertTimeZone 转换时区
// 参数: t - 原时间, targetLocation - 目标时区
// 返回: 转换后的时间和错误信息
func ConvertTimeZone(t time.Time, targetLocation string) (time.Time, error) {
	loc, err := time.LoadLocation(targetLocation)
	if err != nil {
		return time.Time{}, fmt.Errorf("加载时区失败: %w", err)
	}
	return t.In(loc), nil
}
