package tag_system

import (
	"context"
	"testing"

	"neomaster/internal/model/basemodel"
	"neomaster/internal/model/tag_system"
)

// MockTagRepository
type MockTagRepository struct {
	Rules      []tag_system.SysMatchRule
	EntityTags []tag_system.SysEntityTag
}

func (m *MockTagRepository) CreateTag(tag *tag_system.SysTag) error { return nil }
func (m *MockTagRepository) GetTagByID(id uint64) (*tag_system.SysTag, error) {
	return &tag_system.SysTag{BaseModel: basemodel.BaseModel{ID: id}}, nil
}
func (m *MockTagRepository) GetTagByName(name string) (*tag_system.SysTag, error) { return nil, nil }
func (m *MockTagRepository) GetTagsByParent(parentID uint64) ([]tag_system.SysTag, error) {
	return nil, nil
}
func (m *MockTagRepository) GetTagsByIDs(ids []uint64) ([]tag_system.SysTag, error) { return nil, nil }
func (m *MockTagRepository) UpdateTag(tag *tag_system.SysTag) error                 { return nil }
func (m *MockTagRepository) DeleteTag(id uint64) error                              { return nil }
func (m *MockTagRepository) ListTags(req *tag_system.ListTagsRequest) ([]tag_system.SysTag, int64, error) {
	return nil, 0, nil
}

func (m *MockTagRepository) CreateRule(rule *tag_system.SysMatchRule) error { return nil }
func (m *MockTagRepository) GetRulesByEntityType(entityType string) ([]tag_system.SysMatchRule, error) {
	return m.Rules, nil
}
func (m *MockTagRepository) ListRules(req *tag_system.ListRulesRequest) ([]tag_system.SysMatchRule, int64, error) {
	return m.Rules, int64(len(m.Rules)), nil
}
func (m *MockTagRepository) GetRuleByID(id uint64) (*tag_system.SysMatchRule, error) { return nil, nil }
func (m *MockTagRepository) UpdateRule(rule *tag_system.SysMatchRule) error          { return nil }
func (m *MockTagRepository) DeleteRule(id uint64) error                              { return nil }

func (m *MockTagRepository) AddEntityTag(et *tag_system.SysEntityTag) error {
	m.EntityTags = append(m.EntityTags, *et)
	return nil
}
func (m *MockTagRepository) RemoveEntityTag(entityType, entityID string, tagID uint64) error {
	newTags := []tag_system.SysEntityTag{}
	for _, t := range m.EntityTags {
		if !(t.EntityType == entityType && t.EntityID == entityID && t.TagID == tagID) {
			newTags = append(newTags, t)
		}
	}
	m.EntityTags = newTags
	return nil
}
func (m *MockTagRepository) GetEntityTags(entityType, entityID string) ([]tag_system.SysEntityTag, error) {
	var res []tag_system.SysEntityTag
	for _, t := range m.EntityTags {
		if t.EntityType == entityType && t.EntityID == entityID {
			res = append(res, t)
		}
	}
	return res, nil
}
func (m *MockTagRepository) RemoveAllEntityTags(entityType, entityID string) error { return nil }

func TestAutoTag(t *testing.T) {
	// 1. Setup Mock Repo
	mockRepo := &MockTagRepository{
		Rules: []tag_system.SysMatchRule{
			{
				BaseModel:  basemodel.BaseModel{ID: 1},
				TagID:      100,
				EntityType: "host",
				RuleJSON:   `{"field": "os", "operator": "contains", "value": "linux"}`,
				IsEnabled:  true,
			},
			{
				BaseModel:  basemodel.BaseModel{ID: 2},
				TagID:      200,
				EntityType: "host",
				RuleJSON:   `{"field": "open_ports", "operator": "list_contains", "value": 22}`,
				IsEnabled:  true,
			},
		},
		EntityTags: []tag_system.SysEntityTag{}, // Initial empty
	}

	service := NewTagService(mockRepo, nil) // db is nil, don't test SubmitPropagationTask here

	// 2. Test Case 1: Match Rule 1 only
	ctx := context.Background()
	attrs1 := map[string]interface{}{
		"os":         "ubuntu linux",
		"open_ports": []int{80, 443},
	}

	err := service.AutoTag(ctx, "host", "host-1", attrs1)
	if err != nil {
		t.Fatalf("AutoTag failed: %v", err)
	}

	// Verify: Should have Tag 100
	if len(mockRepo.EntityTags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(mockRepo.EntityTags))
	}
	if len(mockRepo.EntityTags) > 0 && mockRepo.EntityTags[0].TagID != 100 {
		t.Errorf("Expected TagID 100, got %d", mockRepo.EntityTags[0].TagID)
	}

	// 3. Test Case 2: Match Rule 1 and 2
	attrs2 := map[string]interface{}{
		"os":         "redhat linux",
		"open_ports": []int{22, 80},
	}
	// Reset repo tags for host-2
	mockRepo.EntityTags = []tag_system.SysEntityTag{}

	err = service.AutoTag(ctx, "host", "host-2", attrs2)
	if err != nil {
		t.Fatalf("AutoTag failed: %v", err)
	}

	if len(mockRepo.EntityTags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(mockRepo.EntityTags))
	}

	// 4. Test Case 3: Diff Logic (Remove old tag)
	// host-2 previously had Tag 100 and 200.
	// Now attributes change, only matches Rule 2.
	// Tag 100 should be removed.

	// Pre-populate mock repo with state from Case 2
	// (Already populated from previous step, but we reset mockRepo.EntityTags manually in step 3 loop)
	// Wait, step 3 populated mockRepo.EntityTags. It has 2 tags.

	attrs3 := map[string]interface{}{
		"os":         "windows server", // No longer matches "linux" (Rule 1)
		"open_ports": []int{22},        // Still matches Rule 2
	}

	err = service.AutoTag(ctx, "host", "host-2", attrs3)
	if err != nil {
		t.Fatalf("AutoTag failed: %v", err)
	}

	// Verify: Should have only Tag 200
	if len(mockRepo.EntityTags) != 1 {
		t.Errorf("Expected 1 tag after update, got %d", len(mockRepo.EntityTags))
	}
	if len(mockRepo.EntityTags) > 0 && mockRepo.EntityTags[0].TagID != 200 {
		t.Errorf("Expected TagID 200, got %d", mockRepo.EntityTags[0].TagID)
	}
}
