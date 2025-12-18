package test_20251212

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/orchestrator"
	tagSystemModel "neomaster/internal/model/tag_system"
	"neomaster/internal/service/orchestrator/allocator"
)

// MockTagService 用于测试的 Mock TagService
type MockTagService struct {
	agentTags map[string][]string
}

func (m *MockTagService) GetEntityTags(ctx context.Context, entityType string, entityID string) ([]tagSystemModel.SysEntityTag, error) {
	if entityType != "agent" {
		return nil, nil
	}
	tags, ok := m.agentTags[entityID]
	if !ok {
		return nil, nil
	}
	var entityTags []tagSystemModel.SysEntityTag
	tagNameToID := map[string]uint64{
		"dmz":      1,
		"internal": 2,
		"DMZ":      3,
	}
	for _, tagName := range tags {
		if id, ok := tagNameToID[tagName]; ok {
			entityTags = append(entityTags, tagSystemModel.SysEntityTag{
				TagID: id,
			})
		}
	}
	return entityTags, nil
}

func (m *MockTagService) GetTagsByIDs(ctx context.Context, ids []uint64) ([]tagSystemModel.SysTag, error) {
	// 这是一个简化的 Mock，假设我们能反向查找 (在真实 Mock 中需要更复杂的逻辑，或者简单的这里只处理我们知道的 case)
	// 为了简单起见，我们遍历所有 agentTags 找到匹配的 ID
	// 但这里我们没有存储 ID 到 Name 的映射。
	// 让我们改进 Mock: agentTags 存储 Name，GetEntityTags 返回伪造 ID，GetTagsByIDs 返回对应 Name。
	// 假设 ID = index + 1 (这就要求我们必须知道是哪个 Agent 的请求，但 GetTagsByIDs 只接收 IDs)
	// 这在并发或多 Agent 场景下会有问题。
	// 更好的方法：Mock 预存 ID->Name 映射。
	var result []tagSystemModel.SysTag
	// Hack: 在测试中我们只用简单的标签名，我们可以做一个全局映射或者简单的假设
	// 假设 ID 1 = "dmz", ID 2 = "internal", ID 3 = "DMZ"
	idNameMap := map[uint64]string{
		1: "dmz",
		2: "internal",
		3: "DMZ",
	}

	for _, id := range ids {
		if name, ok := idNameMap[id]; ok {
			result = append(result, tagSystemModel.SysTag{
				Name: name,
			})
		}
	}
	return result, nil
}

// 其他接口方法实现为空 (Stub)
func (m *MockTagService) CreateTag(ctx context.Context, tag *tagSystemModel.SysTag) error { return nil }
func (m *MockTagService) GetTag(ctx context.Context, id uint64) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) GetTagByName(ctx context.Context, name string) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) GetTagByNameAndParent(ctx context.Context, name string, parentID uint64) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) UpdateTag(ctx context.Context, tag *tagSystemModel.SysTag) error { return nil }
func (m *MockTagService) DeleteTag(ctx context.Context, id uint64, force bool) error      { return nil }
func (m *MockTagService) ListTags(ctx context.Context, req *tagSystemModel.ListTagsRequest) ([]tagSystemModel.SysTag, int64, error) {
	return nil, 0, nil
}
func (m *MockTagService) CreateRule(ctx context.Context, rule *tagSystemModel.SysMatchRule) error {
	return nil
}
func (m *MockTagService) UpdateRule(ctx context.Context, rule *tagSystemModel.SysMatchRule) error {
	return nil
}
func (m *MockTagService) DeleteRule(ctx context.Context, id uint64) error { return nil }
func (m *MockTagService) GetRule(ctx context.Context, id uint64) (*tagSystemModel.SysMatchRule, error) {
	return nil, nil
}
func (m *MockTagService) ListRules(ctx context.Context, req *tagSystemModel.ListRulesRequest) ([]tagSystemModel.SysMatchRule, int64, error) {
	return nil, 0, nil
}
func (m *MockTagService) ReloadMatchRules() error { return nil }
func (m *MockTagService) AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error {
	return nil
}
func (m *MockTagService) SubmitPropagationTask(ctx context.Context, ruleID uint64, action string) (string, error) {
	return "", nil
}
func (m *MockTagService) SubmitEntityPropagationTask(ctx context.Context, entityType string, entityID uint64, tagIDs []uint64, action string) (string, error) {
	return "", nil
}
func (m *MockTagService) SyncEntityTags(ctx context.Context, entityType string, entityID string, targetTagIDs []uint64, sourceScope string, ruleID uint64) error {
	return nil
}
func (m *MockTagService) AddEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64, source string, ruleID uint64) error {
	return nil
}
func (m *MockTagService) RemoveEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64) error {
	return nil
}
func (m *MockTagService) GetEntityIDsByTagIDs(ctx context.Context, entityType string, tagIDs []uint64) ([]string, error) {
	return nil, nil
}

func TestAllocator_CanExecute(t *testing.T) {
	mockTagService := &MockTagService{
		agentTags: map[string][]string{
			"agent-1": {"dmz"},      // ID 1
			"agent-4": {"internal"}, // ID 2
			"agent-6": {"DMZ"},      // ID 3
		},
	}
	alloc := allocator.NewResourceAllocator(mockTagService)
	ctx := context.Background()

	tests := []struct {
		name        string
		agent       *agentModel.Agent
		task        *orchestrator.AgentTask
		expected    bool
		description string
	}{
		{
			name: "Basic Success",
			agent: &agentModel.Agent{
				AgentID:     "agent-1",
				Status:      agentModel.AgentStatusOnline,
				TaskSupport: agentModel.StringSlice{"nmap"},
			},
			task: &orchestrator.AgentTask{
				ToolName:     "nmap",
				RequiredTags: `["dmz"]`,
			},
			expected:    true,
			description: "Agent online, has tool, has tag",
		},
		{
			name: "Agent Offline",
			agent: &agentModel.Agent{
				AgentID:     "agent-2",
				Status:      agentModel.AgentStatusOffline,
				TaskSupport: agentModel.StringSlice{"nmap"},
			},
			task:        &orchestrator.AgentTask{ToolName: "nmap"},
			expected:    false,
			description: "Agent offline should fail",
		},
		{
			name: "Missing Capability",
			agent: &agentModel.Agent{
				AgentID:     "agent-3",
				Status:      agentModel.AgentStatusOnline,
				TaskSupport: agentModel.StringSlice{"masscan"},
			},
			task:        &orchestrator.AgentTask{ToolName: "nmap"},
			expected:    false,
			description: "Agent missing tool should fail",
		},
		{
			name: "Missing Tag",
			agent: &agentModel.Agent{
				AgentID:     "agent-4",
				Status:      agentModel.AgentStatusOnline,
				TaskSupport: agentModel.StringSlice{"nmap"},
			},
			task: &orchestrator.AgentTask{
				ToolName:     "nmap",
				RequiredTags: `["dmz"]`,
			},
			expected:    false,
			description: "Agent missing required tag should fail",
		},
		{
			name: "Capability Case Insensitive",
			agent: &agentModel.Agent{
				AgentID:     "agent-5",
				Status:      agentModel.AgentStatusOnline,
				TaskSupport: agentModel.StringSlice{"NMAP"},
			},
			task:        &orchestrator.AgentTask{ToolName: "nmap"},
			expected:    true,
			description: "Capability check should be case insensitive",
		},
		{
			name: "Tag Case Sensitive (Current Behavior)",
			agent: &agentModel.Agent{
				AgentID:     "agent-6",
				Status:      agentModel.AgentStatusOnline,
				TaskSupport: agentModel.StringSlice{"nmap"},
			},
			task: &orchestrator.AgentTask{
				ToolName:     "nmap",
				RequiredTags: `["dmz"]`,
			},
			expected:    false,
			description: "Tag matching is currently case sensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := alloc.CanExecute(ctx, tt.agent, tt.task)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}
