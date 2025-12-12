package test_20251212

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/matcher"
	"neomaster/internal/service/orchestrator/policy"
)

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Auto migrate
	err = db.AutoMigrate(
		&orchestrator.ScanStage{},
		&orchestrator.AgentTask{},
		&orchestrator.StageResult{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	return db
}

func TestPreviousStageProvider_StageStatus(t *testing.T) {
	db := setupDB(t)
	provider := policy.NewPreviousStageProvider(db)

	projectID := uint64(1)
	workflowID := uint64(100)

	// 1. Setup Data
	// Stage 1: Running
	stage1 := orchestrator.ScanStage{
		WorkflowID: workflowID,
		StageName:  "stage_running",
	}
	db.Create(&stage1)

	task1 := orchestrator.AgentTask{
		TaskID:     "task_1",
		ProjectID:  projectID,
		WorkflowID: workflowID,
		StageID:    uint64(stage1.ID),
		AgentID:    "agent_1",
		Status:     "running",
	}
	db.Create(&task1)

	result1 := orchestrator.StageResult{
		ProjectID:   projectID,
		WorkflowID:  workflowID,
		StageID:     uint64(stage1.ID),
		AgentID:     "agent_1",
		ResultType:  "ip",
		TargetValue: "1.1.1.1",
	}
	db.Create(&result1)

	// Stage 2: Completed
	stage2 := orchestrator.ScanStage{
		WorkflowID: workflowID,
		StageName:  "stage_completed",
	}
	db.Create(&stage2)

	task2 := orchestrator.AgentTask{
		TaskID:     "task_2",
		ProjectID:  projectID,
		WorkflowID: workflowID,
		StageID:    uint64(stage2.ID),
		AgentID:    "agent_2",
		Status:     "completed",
	}
	db.Create(&task2)

	result2 := orchestrator.StageResult{
		ProjectID:   projectID,
		WorkflowID:  workflowID,
		StageID:     uint64(stage2.ID),
		AgentID:     "agent_2",
		ResultType:  "ip",
		TargetValue: "2.2.2.2",
	}
	db.Create(&result2)

	// Stage 3: Current Stage (depends on Stage 1 and Stage 2)
	stage3 := orchestrator.ScanStage{
		WorkflowID:   workflowID,
		StageName:    "stage_current",
		Predecessors: []uint64{uint64(stage1.ID), uint64(stage2.ID)},
	}
	db.Create(&stage3)

	ctx := context.Background()
	ctx = context.WithValue(ctx, policy.CtxKeyProjectID, projectID)
	ctx = context.WithValue(ctx, policy.CtxKeyWorkflowID, workflowID)
	ctx = context.WithValue(ctx, policy.CtxKeyStageID, uint64(stage3.ID))

	filterRules1 := `{"stage_name": "stage_completed", "stage_status": ["completed"]}`
	config1 := policy.TargetSourceConfig{
		FilterRules: []byte(filterRules1),
	}

	targets1, err := provider.Provide(ctx, config1, nil)
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}
	if len(targets1) != 1 || targets1[0].Value != "2.2.2.2" {
		t.Errorf("Expected 1 target (2.2.2.2), got %v", targets1)
	}

	// 3. Test Case 2: Filter "running"
	filterRules2 := `{"stage_name": "stage_running", "stage_status": ["running"]}`
	config2 := policy.TargetSourceConfig{
		FilterRules: []byte(filterRules2),
	}
	targets2, err := provider.Provide(ctx, config2, nil)
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}
	if len(targets2) != 1 || targets2[0].Value != "1.1.1.1" {
		t.Errorf("Expected 1 target (1.1.1.1), got %v", targets2)
	}

	// 4. Test Case 3: Filter "completed" on running stage (should fail)
	filterRules3 := `{"stage_name": "stage_running", "stage_status": ["completed"]}`
	config3 := policy.TargetSourceConfig{
		FilterRules: []byte(filterRules3),
	}
	targets3, err := provider.Provide(ctx, config3, nil)
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}
	if len(targets3) != 0 {
		t.Errorf("Expected 0 targets, got %v", targets3)
	}
}

func TestPreviousStageProvider_ComplexFilter(t *testing.T) {
	db := setupDB(t)
	provider := policy.NewPreviousStageProvider(db)

	projectID := uint64(1)
	workflowID := uint64(100)

	stage1 := orchestrator.ScanStage{
		WorkflowID: workflowID,
		StageName:  "stage_data",
	}
	db.Create(&stage1)

	// Result with complex attributes
	jsonAttr := `
	[
		{"port": 80, "service": "http", "state": "open"},
		{"port": 22, "service": "ssh", "state": "open"},
		{"port": 443, "service": "https", "state": "closed"}
	]
	`
	result1 := orchestrator.StageResult{
		ProjectID:   projectID,
		WorkflowID:  workflowID,
		StageID:     uint64(stage1.ID),
		AgentID:     "agent_1",
		Attributes:  jsonAttr,
		TargetValue: "10.0.0.1",
	}
	db.Create(&result1)

	// Stage 2: Current Stage (depends on Stage 1)
	stage2 := orchestrator.ScanStage{
		WorkflowID:   workflowID,
		StageName:    "stage_current",
		Predecessors: []uint64{uint64(stage1.ID)},
	}
	db.Create(&stage2)

	ctx := context.Background()
	ctx = context.WithValue(ctx, policy.CtxKeyProjectID, projectID)
	ctx = context.WithValue(ctx, policy.CtxKeyWorkflowID, workflowID)
	ctx = context.WithValue(ctx, policy.CtxKeyStageID, uint64(stage2.ID))

	// Filter: state == open AND (port > 100 OR service == http)
	// port 80: state=open(T), port>100(F) OR service=http(T) -> T
	// port 22: state=open(T), port>100(F) OR service=ssh(F) -> F (Wait, 22 is not > 100 and service is ssh)
	// port 443: state=closed(F) -> F

	// Construct complex rule
	rule := matcher.MatchRule{
		And: []matcher.MatchRule{
			{Field: "state", Operator: "equals", Value: "open"},
			{
				Or: []matcher.MatchRule{
					{Field: "port", Operator: "greater_than", Value: 100},
					{Field: "service", Operator: "equals", Value: "http"},
				},
			},
		},
	}
	ruleBytes, _ := json.Marshal(rule)

	filterRules := `{"stage_name": "stage_data"}`
	parserConfig := fmt.Sprintf(`{
		"unwind": {
			"path": "@this",
			"filter": %s
		},
		"generate": {
			"value_template": "{{item.port}}"
		}
	}`, string(ruleBytes))

	config := policy.TargetSourceConfig{
		FilterRules:  []byte(filterRules),
		ParserConfig: []byte(parserConfig),
	}

	targets, err := provider.Provide(ctx, config, nil)
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	// Expecting port 80 only.
	// Wait, port 22: state=open. port>100 (False), service=ssh!=http. So Or is False. And is False.
	// Correct.
	if len(targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets))
		for _, tg := range targets {
			t.Logf("Got target: %v", tg.Value)
		}
	} else {
		if targets[0].Value != "80" {
			t.Errorf("Expected target value 80, got %s", targets[0].Value)
		}
	}
}
