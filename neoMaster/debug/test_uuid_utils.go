/*
 * @author: sun977
 * @date: 2025.09.05
 * @description: UUID工具包测试文件
 * @func: 测试UUID生成、解析、校验等功能
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"neomaster/internal/pkg/utils"
)

func main() {
	fmt.Println("=== UUID工具包功能测试 ===")

	// 1. 测试基本UUID生成
	fmt.Println("\n1. 基本UUID生成测试:")
	uuid1, err := utils.GenerateUUID()
	if err != nil {
		log.Printf("生成UUID失败: %v", err)
	} else {
		fmt.Printf("标准UUID: %s\n", uuid1)
	}

	// 2. 测试简化UUID生成
	fmt.Println("\n2. 简化UUID生成测试:")
	simpleUUID, err := utils.GenerateSimpleUUID()
	if err != nil {
		log.Printf("生成简化UUID失败: %v", err)
	} else {
		fmt.Printf("简化UUID: %s\n", simpleUUID)
	}

	// 3. 测试带前缀UUID生成
	fmt.Println("\n3. 带前缀UUID生成测试:")
	prefixUUID, err := utils.GenerateUUIDWithPrefix("user")
	if err != nil {
		log.Printf("生成带前缀UUID失败: %v", err)
	} else {
		fmt.Printf("带前缀UUID: %s\n", prefixUUID)
	}

	// 4. 测试短UUID生成
	fmt.Println("\n4. 短UUID生成测试:")
	shortUUID, err := utils.GenerateShortUUID()
	if err != nil {
		log.Printf("生成短UUID失败: %v", err)
	} else {
		fmt.Printf("短UUID: %s\n", shortUUID)
	}

	// 5. 测试UUID格式校验
	fmt.Println("\n5. UUID格式校验测试:")
	testUUIDs := []string{
		uuid1,
		simpleUUID,
		"550e8400-e29b-41d4-a716-446655440000", // 有效标准格式
		"550e8400e29b41d4a716446655440000",   // 有效简化格式
		"invalid-uuid",                        // 无效格式
		"",                                     // 空字符串
	}

	for _, testUUID := range testUUIDs {
		isValid := utils.IsValidUUID(testUUID)
		fmt.Printf("UUID: %-40s 有效性: %t\n", testUUID, isValid)
	}

	// 6. 测试UUID格式转换
	fmt.Println("\n6. UUID格式转换测试:")
	if utils.IsValidUUID(uuid1) {
		// 标准化
		normalized, err := utils.NormalizeUUID(simpleUUID)
		if err != nil {
			log.Printf("标准化UUID失败: %v", err)
		} else {
			fmt.Printf("简化格式: %s -> 标准格式: %s\n", simpleUUID, normalized)
		}

		// 简化
		simplified, err := utils.SimplifyUUID(uuid1)
		if err != nil {
			log.Printf("简化UUID失败: %v", err)
		} else {
			fmt.Printf("标准格式: %s -> 简化格式: %s\n", uuid1, simplified)
		}
	}

	// 7. 测试UUID解析
	fmt.Println("\n7. UUID解析测试:")
	uuidInfo := utils.ParseUUID(uuid1)
	uuidInfoJSON, _ := json.MarshalIndent(uuidInfo, "", "  ")
	fmt.Printf("UUID解析结果:\n%s\n", string(uuidInfoJSON))

	// 8. 测试批量生成UUID
	fmt.Println("\n8. 批量生成UUID测试:")
	batchUUIDs, err := utils.BatchGenerateUUID(3, "batch")
	if err != nil {
		log.Printf("批量生成UUID失败: %v", err)
	} else {
		fmt.Printf("批量生成的UUID:\n")
		for i, batchUUID := range batchUUIDs {
			fmt.Printf("  %d: %s\n", i+1, batchUUID)
		}
	}

	// 9. 测试UUID比较
	fmt.Println("\n9. UUID比较测试:")
	uuid2, _ := utils.GenerateUUID()
	simpleUUID2, _ := utils.SimplifyUUID(uuid1) // 同一个UUID的简化格式

	// 比较相同UUID的不同格式
	same1, err1 := utils.CompareUUID(uuid1, simpleUUID2)
	if err1 != nil {
		log.Printf("比较UUID失败: %v", err1)
	} else {
		fmt.Printf("UUID1: %s\n", uuid1)
		fmt.Printf("UUID1简化: %s\n", simpleUUID2)
		fmt.Printf("是否相同: %t\n", same1)
	}

	// 比较不同UUID
	same2, err2 := utils.CompareUUID(uuid1, uuid2)
	if err2 != nil {
		log.Printf("比较UUID失败: %v", err2)
	} else {
		fmt.Printf("\nUUID1: %s\n", uuid1)
		fmt.Printf("UUID2: %s\n", uuid2)
		fmt.Printf("是否相同: %t\n", same2)
	}

	fmt.Println("\n=== 测试完成 ===")
}