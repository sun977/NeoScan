package main

import (
	"fmt"
	"net/url"
	"strings"
)

func main() {
	// 模拟URL参数解析
	testURL := "http://localhost:8123/api/v1/agent?page=1&page_size=10&status=online&tags=2,7"
	u, _ := url.Parse(testURL)
	values := u.Query()
	
	fmt.Printf("测试URL: %s\n", testURL)
	fmt.Printf("Raw tags value: %v\n", values.Get("tags"))
	fmt.Printf("QueryArray result: %v\n", values["tags"])
	
	// 测试逗号分隔的解析
	tagsStr := values.Get("tags")
	if tagsStr != "" {
		tags := strings.Split(tagsStr, ",")
		fmt.Printf("Split result: %v\n", tags)
		fmt.Printf("Split result length: %d\n", len(tags))
		for i, tag := range tags {
			fmt.Printf("  tags[%d] = '%s'\n", i, tag)
		}
	}
	
	// 测试多个tags参数的情况
	fmt.Println("\n--- 测试多个tags参数 ---")
	testURL2 := "http://localhost:8123/api/v1/agent?tags=2&tags=7"
	u2, _ := url.Parse(testURL2)
	values2 := u2.Query()
	
	fmt.Printf("测试URL2: %s\n", testURL2)
	fmt.Printf("Raw tags value: %v\n", values2.Get("tags"))
	fmt.Printf("QueryArray result: %v\n", values2["tags"])
}