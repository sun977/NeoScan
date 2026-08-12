package api

import (
	"context"
	"testing"

	"neoagent/internal/core/model"
)

// TestApiScanner_Name 验证 Name() 返回值与 RunnerManager 注册键一致。
func TestApiScanner_Name(t *testing.T) {
	scanner := NewApiScanner()
	if scanner.Name() != model.TaskTypeApiScan {
		t.Errorf("expected Name()=%q, got %q", model.TaskTypeApiScan, scanner.Name())
	}
}

// TestApiScanner_Run_DoesNotPanicOnUnreachableTarget 验证不可达目标时
// Run() 不 panic、不返回 error——目标不可达是"扫描结果为空"，不是"扫描器故障"。
// 真实抓取覆盖见 api_scanner_e2e_test.go（步骤 14）。
func TestApiScanner_Run_DoesNotPanicOnUnreachableTarget(t *testing.T) {
	scanner := NewApiScanner()
	task := model.NewTask(model.TaskTypeApiScan, "http://127.0.0.1:1")
	task.PortRange = "1"

	results, err := scanner.Run(context.Background(), task)
	// 不可达目标：crawl 内部 fetchPage 会失败，outcomes 为空，Run 本身
	// 不应该返回 error（这不是"扫描器故障"，是"目标不可达"，两者语义不同）。
	if err != nil {
		t.Fatalf("Run should not return error for unreachable target, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results for unreachable target, got %d", len(results))
	}
}
