package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"neoagent/internal/config"
	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/port_service"
	"neoagent/internal/pkg/logger"
)

func main() {
	// 初始化 Logger
	logger.InitLogger(&config.LogConfig{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	fmt.Println("=== Starting PortServiceScanner Functional Test ===")

	// 1. 初始化 Scanner
	scanner := port_service.NewPortServiceScanner()
	fmt.Printf("Scanner Name: %s\n", scanner.Name())

	// 获取测试目标
	// 默认测试本机和 Linux 测试机
	targets := []string{"127.0.0.1", "10.44.96.183"}
	if len(os.Args) > 1 {
		targets = os.Args[1:]
	}

	for _, target := range targets {
		fmt.Printf("\n--- Testing Target: %s ---\n", target)
		runTestsForTarget(scanner, target)
	}

	fmt.Println("\n=== Test Completed ===")
}

func runTestsForTarget(scanner *port_service.PortServiceScanner, target string) {
	// Case 1: 基础端口扫描 (ServiceDetect=false)
	fmt.Println("\n[Case 1] Basic Port Scan (No Service Detect)")
	task1 := &model.Task{
		ID:        "test-task-1",
		Target:    target,
		PortRange: "22,80,443,135,445,3389,8080", // 混合 Linux/Windows 常用端口
		Params: map[string]interface{}{
			"service_detect": false,
			"rate":           1000,
		},
	}
	runTask(scanner, task1)

	// Case 2: 服务识别扫描 (ServiceDetect=true)
	fmt.Println("\n[Case 2] Service Discovery (With Service Detect)")
	task2 := &model.Task{
		ID:        "test-task-2",
		Target:    target,
		PortRange: "22,80,443,135,445,3389,8080",
		Params: map[string]interface{}{
			"service_detect": true,
			"rate":           1000,
		},
	}
	runTask(scanner, task2)
}

func runTask(scanner *port_service.PortServiceScanner, task *model.Task) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	results, err := scanner.Run(ctx, task)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("Error running task: %v\n", err)
		return
	}

	fmt.Printf("Scan completed in %v. Found %d open ports.\n", duration, len(results))

	// 打印结果详情
	for _, res := range results {
		// 转换 Result 为 PortServiceResult
		if psRes, ok := res.Result.(*model.PortServiceResult); ok {
			output, _ := json.MarshalIndent(psRes, "", "  ")
			fmt.Printf("Port %d: %s\n", psRes.Port, string(output))
		} else {
			fmt.Printf("Result format error: %v\n", res.Result)
		}
	}
}
