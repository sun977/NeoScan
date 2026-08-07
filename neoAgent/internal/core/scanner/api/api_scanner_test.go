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

// TestApiScanner_Run_SkeletonReturnsNoResult 验证骨架阶段 Run() 不做任何
// 抓取/分析，直接返回空结果且不报错，等待后续实施文档补充真实实现。
func TestApiScanner_Run_SkeletonReturnsNoResult(t *testing.T) {
	scanner := NewApiScanner()
	task := model.NewTask(model.TaskTypeApiScan, "127.0.0.1")

	results, err := scanner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected skeleton Run to return no results, got %d", len(results))
	}
}
