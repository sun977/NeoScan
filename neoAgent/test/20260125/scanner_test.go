package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"neoagent/internal/core/model"
	"neoagent/internal/core/scanner/alive"
	"neoagent/internal/core/scanner/port_service"
)

// TestIpAliveScanner_LocalNet 测试本地网段的 IP 存活扫描
func TestIpAliveScanner_LocalNet(t *testing.T) {
	// 1. 准备 Scanner
	scanner := alive.NewIpAliveScanner()

	// 2. 准备任务
	// 使用 10.44.96.1/24 网段 (根据用户指示)
	// 建议限制范围以加快测试速度，例如 10.44.96.1-10.44.96.20
	// 或者直接扫整个网段，但要注意并发和超时
	target := "10.44.96.1/24"

	task := &model.Task{
		ID:      "test-alive-001",
		Type:    model.TaskTypeIpAliveScan,
		Target:  target,
		Timeout: 5 * time.Second,
		Params: map[string]interface{}{
			"rate":             500,  // 并发数
			"resolve_hostname": true, // 启用 Hostname 解析
		},
	}

	t.Logf("Starting IpAliveScan on %s...", target)
	start := time.Now()

	// 3. 执行扫描
	ctx := context.Background()
	results, err := scanner.Run(ctx, task)
	if err != nil {
		t.Fatalf("IpAliveScan failed: %v", err)
	}

	duration := time.Since(start)
	t.Logf("Scan completed in %s", duration)

	// 4. 分析结果
	if len(results) == 0 {
		t.Log("No alive hosts found.")
		return
	}

	// 打印详细结果
	// IpAliveScanner 的 Run 返回的是一个 *model.TaskResult
	// 其 Result 字段是 []interface{} (IpAliveResult)
	// 或者在重构后可能是直接的 IpAliveResult slice

	// 检查结果结构
	if len(results) != 1 {
		t.Logf("Unexpected results count: %d", len(results))
	}

	taskResult := results[0]
	t.Logf("Task Status: %s", taskResult.Status)

	if resultList, ok := taskResult.Result.([]interface{}); ok {
		t.Logf("Found %d alive hosts:", len(resultList))
		for i, raw := range resultList {
			// 尝试转换为具体的 IpAliveResult
			// 由于 Result 是 interface{}，我们需要通过 JSON 转换或者类型断言
			// 简单起见，打印 JSON
			jsonData, _ := json.MarshalIndent(raw, "", "  ")
			t.Logf("[%d] %s", i, string(jsonData))

			// 如果能断言到 model.IpAliveResult 更好
			if res, ok := raw.(*model.IpAliveResult); ok {
				t.Logf("  -> IP: %s, RTT: %s, OS: %s", res.IP, res.RTT, res.OS)
			}
		}
	} else {
		t.Logf("Result type mismatch: %T", taskResult.Result)
	}
}

// TestPortServiceScanner_LocalNet 测试端口服务扫描
func TestPortServiceScanner_LocalNet(t *testing.T) {
	// 1. 准备 Scanner
	scanner := port_service.NewPortServiceScanner()

	// 2. 准备任务
	// 针对已知存在多端口的 IP 进行测试
	target := "10.44.96.183"

	// 常见端口 + 一些特定端口 + cool-admin端口
	portRange := "22,80,443,445,3389,8080,8440,8443,8445,8000-8100"

	task := &model.Task{
		ID:        "test-port-002",
		Type:      model.TaskTypePortScan,
		Target:    target,
		PortRange: portRange,
		Timeout:   2 * time.Second, // 单个连接超时
		Params: map[string]interface{}{
			"rate":           200,
			"service_detect": true, // 启用服务识别
		},
	}

	t.Logf("Starting PortServiceScan on %s ports %s...", target, portRange)
	start := time.Now()

	ctx := context.Background()
	results, err := scanner.Run(ctx, task)
	if err != nil {
		t.Fatalf("PortServiceScan failed: %v", err)
	}

	duration := time.Since(start)
	t.Logf("Scan completed in %s", duration)

	if len(results) == 0 {
		t.Log("No open ports found.")
		return
	}

	taskResult := results[0]

	if resultList, ok := taskResult.Result.([]interface{}); ok {
		t.Logf("Found %d open ports:", len(resultList))
		for i, raw := range resultList {
			jsonData, _ := json.MarshalIndent(raw, "", "  ")
			t.Logf("[%d] %s", i, string(jsonData))
		}
	}
}
