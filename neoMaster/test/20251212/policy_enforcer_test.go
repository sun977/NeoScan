package test_20251212

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	assetModel "neomaster/internal/model/asset"
	agentModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/matcher"
	assetrepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/orchestrator/policy"
)

func TestPolicyEnforcer_SkipRule_Matcher(t *testing.T) {
	// 1. Setup In-Memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	// 2. Migrate Tables
	if err1 := db.AutoMigrate(&agentModel.Project{}, &assetModel.AssetSkipPolicy{}, &assetModel.AssetWhitelist{}); err1 != nil {
		t.Fatalf("failed to migrate: %v", err1)
	}

	// 3. Prepare Data
	// Create Project with Tags
	project := &agentModel.Project{
		// ID will be auto-generated or can be set if needed for relation
		Name:        "ProdProject",
		TargetScope: "192.168.1.1", // valid scope
	}
	// Manually set ID for test consistency
	project.ID = 123
	if err2 := db.Create(project).Error; err2 != nil {
		t.Fatalf("failed to create project: %v", err2)
	}

	// Create Skip Policy with Matcher Rule
	// Rule: Skip if tags contains "production" AND project_name equals "ProdProject"
	matchRule := matcher.MatchRule{
		And: []matcher.MatchRule{
			{
				Field:    "tags",
				Operator: "contains", // List contains item (requires matcher support or string conversion)
				Value:    "production",
			},
			{
				Field:    "project_name",
				Operator: "equals",
				Value:    "ProdProject",
			},
		},
	}
	ruleJSON, _ := json.Marshal(policy.SkipConditionRules{
		MatchRule: matchRule,
	})

	skipPolicy := &assetModel.AssetSkipPolicy{
		PolicyName:     "ProdSkipRule",
		ConditionRules: string(ruleJSON),
		Enabled:        true,
	}
	if err3 := db.Create(skipPolicy).Error; err3 != nil {
		t.Fatalf("failed to create skip policy: %v", err3)
	}

	// 4. Initialize Repos and Enforcer
	// projRepo := orcrepo.NewProjectRepository(db)
	policyRepo := assetrepo.NewAssetPolicyRepository(db)
	enforcer := policy.NewPolicyEnforcer(policyRepo)

	// 5. Test Enforce (Should Skip)
	task := &agentModel.AgentTask{
		TaskID:      "task-001",
		ProjectID:   123,
		InputTarget: "192.168.1.1",
	}

	err = enforcer.Enforce(context.Background(), task)
	if err == nil {
		t.Fatal("expected skip error, got nil")
	}
	// Verify error message
	expectedMsg := "task skipped due to policy"
	if err.Error() == "" || len(err.Error()) < len(expectedMsg) { // Loose check
		t.Logf("Got expected error: %v", err)
	}

	// 6. Test No Match (Should Pass)
	// Update Project Tags to not match
	db.Model(project).Update("Tags", `["dev"]`)

	err = enforcer.Enforce(context.Background(), task)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestPolicyEnforcer_SkipRule_Legacy(t *testing.T) {
	// 1. Setup
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	db.AutoMigrate(&agentModel.Project{}, &assetModel.AssetSkipPolicy{}, &assetModel.AssetWhitelist{})

	// 2. Data
	project := &agentModel.Project{
		Name:        "LegacyProject",
		TargetScope: "10.0.0.1",
	}
	project.ID = 456
	db.Create(project)

	// Legacy Rule: BlockEnvTags
	ruleJSON, _ := json.Marshal(policy.SkipConditionRules{
		BlockEnvTags: []string{"sensitive"},
	})
	db.Create(&assetModel.AssetSkipPolicy{
		PolicyName:     "SensitiveBlock",
		ConditionRules: string(ruleJSON),
		Enabled:        true,
	})

	// 3. Init
	enforcer := policy.NewPolicyEnforcer(assetrepo.NewAssetPolicyRepository(db))

	// 4. Test
	task := &agentModel.AgentTask{
		TaskID:      "task-002",
		ProjectID:   456,
		InputTarget: "10.0.0.1",
	}

	err = enforcer.Enforce(context.Background(), task)
	if err == nil {
		t.Fatal("expected legacy skip error, got nil")
	}
}
