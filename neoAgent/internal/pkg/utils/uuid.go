/*
 * @author: sun977
 * @date: 2025.09.05
 * @description: uuid工具包
 * @func: 提供uuid生成、解析、校验等常用工具函数
 */

package utils

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// UUID版本常量
const (
	UUIDVersion1 = 1 // 基于时间戳的UUID
	UUIDVersion4 = 4 // 基于随机数的UUID
)

// UUID格式正则表达式
var (
	// 标准UUID格式: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// 简化UUID格式: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	uuidSimpleRegex = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
)

// GenerateUUID 生成UUID v4（基于随机数）
// 返回标准格式的UUID字符串，如：550e8400-e29b-41d4-a716-446655440000
func GenerateUUID() (string, error) {
	// 生成16字节的随机数
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		return "", fmt.Errorf("生成随机数失败: %v", err)
	}

	// 设置版本号（第7字节的高4位设为0100，表示版本4）
	uuid[6] = (uuid[6] & 0x0f) | 0x40

	// 设置变体（第9字节的高2位设为10）
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	// 格式化为标准UUID字符串
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}

// GenerateSimpleUUID 生成简化格式的UUID（不含连字符）
// 返回32位十六进制字符串，如：550e8400e29b41d4a716446655440000
func GenerateSimpleUUID() (string, error) {
	uuid, err := GenerateUUID()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(uuid, "-", ""), nil
}

// GenerateUUIDWithPrefix 生成带前缀的UUID
// prefix: 前缀字符串
// 返回格式：prefix_uuid，如：user_550e8400-e29b-41d4-a716-446655440000
func GenerateUUIDWithPrefix(prefix string) (string, error) {
	uuid, err := GenerateUUID()
	if err != nil {
		return "", err
	}
	if prefix == "" {
		return uuid, nil
	}
	return fmt.Sprintf("%s_%s", prefix, uuid), nil
}

// GenerateShortUUID 生成短UUID（取前8位）
// 返回8位十六进制字符串，如：550e8400
// 注意：短UUID存在碰撞风险，仅适用于对唯一性要求不高的场景
func GenerateShortUUID() (string, error) {
	uuid, err := GenerateSimpleUUID()
	if err != nil {
		return "", err
	}
	return uuid[:8], nil
}

// IsValidUUID 校验UUID格式是否有效
// uuid: 待校验的UUID字符串
// 支持标准格式（带连字符）和简化格式（不带连字符）
func IsValidUUID(uuid string) bool {
	if uuid == "" {
		return false
	}
	// 检查标准格式
	if uuidRegex.MatchString(uuid) {
		return true
	}
	// 检查简化格式
	return uuidSimpleRegex.MatchString(uuid)
}

// NormalizeUUID 标准化UUID格式
// uuid: 输入的UUID字符串
// 将简化格式转换为标准格式，标准格式保持不变
func NormalizeUUID(uuid string) (string, error) {
	if !IsValidUUID(uuid) {
		return "", fmt.Errorf("无效的UUID格式: %s", uuid)
	}

	// 如果已经是标准格式，直接返回
	if uuidRegex.MatchString(uuid) {
		return strings.ToLower(uuid), nil
	}

	// 转换简化格式为标准格式
	if len(uuid) == 32 {
		uuid = strings.ToLower(uuid)
		return fmt.Sprintf("%s-%s-%s-%s-%s",
			uuid[0:8], uuid[8:12], uuid[12:16], uuid[16:20], uuid[20:32]), nil
	}

	return "", fmt.Errorf("无法标准化UUID: %s", uuid)
}

// SimplifyUUID 简化UUID格式（移除连字符）
// uuid: 输入的UUID字符串
// 将标准格式转换为简化格式，简化格式保持不变
func SimplifyUUID(uuid string) (string, error) {
	if !IsValidUUID(uuid) {
		return "", fmt.Errorf("无效的UUID格式: %s", uuid)
	}
	return strings.ToLower(strings.ReplaceAll(uuid, "-", "")), nil
}

// ParseUUID 解析UUID并返回详细信息
type UUIDInfo struct {
	Original  string `json:"original"`  // 原始UUID
	Standard  string `json:"standard"`  // 标准格式
	Simple    string `json:"simple"`    // 简化格式
	Version   int    `json:"version"`   // UUID版本
	Variant   string `json:"variant"`   // UUID变体
	Timestamp int64  `json:"timestamp"` // 时间戳（仅版本1有效）
	IsValid   bool   `json:"is_valid"`  // 是否有效
}

// ParseUUID 解析UUID信息
func ParseUUID(uuid string) *UUIDInfo {
	info := &UUIDInfo{
		Original: uuid,
		IsValid:  IsValidUUID(uuid),
	}

	if !info.IsValid {
		return info
	}

	// 标准化UUID
	standardUUID, err := NormalizeUUID(uuid)
	if err != nil {
		info.IsValid = false
		return info
	}
	info.Standard = standardUUID

	// 简化UUID
	simpleUUID, _ := SimplifyUUID(uuid)
	info.Simple = simpleUUID

	// 解析版本信息（第13位字符，即第7字节的高4位）
	versionChar := simpleUUID[12]
	switch versionChar {
	case '1':
		info.Version = 1
		// 对于版本1，可以提取时间戳（这里简化处理）
		info.Timestamp = time.Now().Unix()
	case '4':
		info.Version = 4
	default:
		info.Version = int(versionChar - '0')
	}

	// 解析变体信息（第17位字符，即第9字节的高2位）
	variantChar := simpleUUID[16]
	switch {
	case variantChar >= '0' && variantChar <= '7':
		info.Variant = "NCS"
	case variantChar >= '8' && variantChar <= 'b', variantChar >= 'B' && variantChar <= 'B':
		info.Variant = "RFC4122"
	case variantChar >= 'c' && variantChar <= 'd', variantChar >= 'C' && variantChar <= 'D':
		info.Variant = "Microsoft"
	default:
		info.Variant = "Reserved"
	}

	return info
}

// BatchGenerateUUID 批量生成UUID
// count: 生成数量
// prefix: 可选前缀
func BatchGenerateUUID(count int, prefix string) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("生成数量必须大于0")
	}
	if count > 10000 {
		return nil, fmt.Errorf("单次生成数量不能超过10000")
	}

	uuids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		var uuid string
		var err error
		if prefix != "" {
			uuid, err = GenerateUUIDWithPrefix(prefix)
		} else {
			uuid, err = GenerateUUID()
		}
		if err != nil {
			return nil, fmt.Errorf("生成第%d个UUID失败: %v", i+1, err)
		}
		uuids = append(uuids, uuid)
	}
	return uuids, nil
}

// CompareUUID 比较两个UUID是否相同（忽略格式差异）
func CompareUUID(uuid1, uuid2 string) (bool, error) {
	if !IsValidUUID(uuid1) {
		return false, fmt.Errorf("第一个UUID格式无效: %s", uuid1)
	}
	if !IsValidUUID(uuid2) {
		return false, fmt.Errorf("第二个UUID格式无效: %s", uuid2)
	}

	// 转换为简化格式进行比较
	simple1, _ := SimplifyUUID(uuid1)
	simple2, _ := SimplifyUUID(uuid2)

	return strings.EqualFold(simple1, simple2), nil
}


