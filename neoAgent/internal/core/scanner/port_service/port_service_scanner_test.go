package port_service

import (
	"context"
	"testing"
	"time"

	"neoagent/internal/config"
	"neoagent/internal/core/model"
	"neoagent/internal/pkg/logger"
)

func init() {
	// 初始化 Logger 以便查看输出
	logger.InitLogger(&config.LogConfig{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
}

func TestPortServiceScanner_Compile(t *testing.T) {
	// Simple compilation check
	s := NewPortServiceScanner()
	_ = s.Name()
}

func TestPortServiceScanner_Run_Mock(t *testing.T) {
	// This test just ensures the Run method doesn't panic and handles empty inputs gracefully
	scanner := NewPortServiceScanner()

	task := &model.Task{
		ID:        "test-task",
		Target:    "127.0.0.1",
		PortRange: "80,443",
		Params: map[string]interface{}{
			"service_detect": false,
			"rate":           10,
		},
	}

	// We don't actually expect it to find anything on localhost without a server running,
	// but we want to ensure it runs without error.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	results, err := scanner.Run(ctx, task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Results might be empty if ports are closed
	t.Logf("Scan finished with %d results", len(results))
}

func TestFunctional_LocalScan(t *testing.T) {
	// 这是一个实际的功能测试，会扫描本机
	// 可以通过 go test -v -run TestFunctional_LocalScan 来运行

	scanner := NewPortServiceScanner()
	target := "127.0.0.1" // 本机

	// 测试常用端口
	ports := "135,445,80,22,3389"

	t.Logf("Starting functional scan on %s ports: %s", target, ports)

	// Case 1: Service Detect OFF
	task1 := &model.Task{
		ID:        "func-test-1",
		Target:    target,
		PortRange: ports,
		Params: map[string]interface{}{
			"service_detect": false,
			"rate":           1000,
		},
	}

	ctx := context.Background()
	results1, err := scanner.Run(ctx, task1)
	if err != nil {
		t.Fatalf("Case 1 failed: %v", err)
	}
	t.Logf("[Case 1] Found %d open ports (No Service Detect)", len(results1))
	for _, res := range results1 {
		if psRes, ok := res.Result.(*model.PortServiceResult); ok {
			t.Logf("  Port %d: %s", psRes.Port, psRes.Service)
		}
	}

	// Case 2: Service Detect ON
	task2 := &model.Task{
		ID:        "func-test-2",
		Target:    target,
		PortRange: ports,
		Params: map[string]interface{}{
			"service_detect": true,
			"rate":           1000,
		},
	}

	results2, err := scanner.Run(ctx, task2)
	if err != nil {
		t.Fatalf("Case 2 failed: %v", err)
	}
	t.Logf("[Case 2] Found %d open ports (With Service Detect)", len(results2))
	for _, res := range results2 {
		if psRes, ok := res.Result.(*model.PortServiceResult); ok {
			t.Logf("  Port %d: %s Product: %s Version: %s", psRes.Port, psRes.Service, psRes.Product, psRes.Version)
		}
	}
}
