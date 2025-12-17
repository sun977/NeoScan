package agent_test

import (
	"context"
	"testing"
	"time"

	agentModel "neomaster/internal/model/agent"
	tagSystemModel "neomaster/internal/model/tag_system"
	agentService "neomaster/internal/service/agent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock TagService ---
type MockTagService struct {
	mock.Mock
}

func (m *MockTagService) CreateTag(ctx context.Context, tag *tagSystemModel.SysTag) error {
	args := m.Called(ctx, tag)
	return args.Error(0)
}
func (m *MockTagService) GetTag(ctx context.Context, id uint64) (*tagSystemModel.SysTag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tagSystemModel.SysTag), args.Error(1)
}
func (m *MockTagService) GetTagByName(ctx context.Context, name string) (*tagSystemModel.SysTag, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tagSystemModel.SysTag), args.Error(1)
}
func (m *MockTagService) GetTagsByIDs(ctx context.Context, ids []uint64) ([]tagSystemModel.SysTag, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tagSystemModel.SysTag), args.Error(1)
}
func (m *MockTagService) UpdateTag(ctx context.Context, tag *tagSystemModel.SysTag) error {
	args := m.Called(ctx, tag)
	return args.Error(0)
}
func (m *MockTagService) DeleteTag(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockTagService) ListTags(ctx context.Context, req *tagSystemModel.ListTagsRequest) ([]tagSystemModel.SysTag, int64, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]tagSystemModel.SysTag), args.Get(1).(int64), args.Error(2)
}
func (m *MockTagService) CreateRule(ctx context.Context, rule *tagSystemModel.SysMatchRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}
func (m *MockTagService) UpdateRule(ctx context.Context, rule *tagSystemModel.SysMatchRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}
func (m *MockTagService) DeleteRule(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockTagService) GetRule(ctx context.Context, id uint64) (*tagSystemModel.SysMatchRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tagSystemModel.SysMatchRule), args.Error(1)
}
func (m *MockTagService) ListRules(ctx context.Context, req *tagSystemModel.ListRulesRequest) ([]tagSystemModel.SysMatchRule, int64, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]tagSystemModel.SysMatchRule), args.Get(1).(int64), args.Error(2)
}
func (m *MockTagService) AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error {
	args := m.Called(ctx, entityType, entityID, attributes)
	return args.Error(0)
}
func (m *MockTagService) SubmitPropagationTask(ctx context.Context, ruleID uint64, action string) (string, error) {
	args := m.Called(ctx, ruleID, action)
	return args.String(0), args.Error(1)
}
func (m *MockTagService) SubmitEntityPropagationTask(ctx context.Context, entityType string, entityID uint64, tagIDs []uint64, action string) (string, error) {
	args := m.Called(ctx, entityType, entityID, tagIDs, action)
	return args.String(0), args.Error(1)
}
func (m *MockTagService) SyncEntityTags(ctx context.Context, entityType string, entityID string, targetTagIDs []uint64, sourceScope string, ruleID uint64) error {
	args := m.Called(ctx, entityType, entityID, targetTagIDs, sourceScope, ruleID)
	return args.Error(0)
}
func (m *MockTagService) BootstrapSystemTags(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockTagService) AddEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64, source string, ruleID uint64) error {
	args := m.Called(ctx, entityType, entityID, tagID, source, ruleID)
	return args.Error(0)
}
func (m *MockTagService) RemoveEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64) error {
	args := m.Called(ctx, entityType, entityID, tagID)
	return args.Error(0)
}
func (m *MockTagService) GetEntityTags(ctx context.Context, entityType string, entityID string) ([]tagSystemModel.SysEntityTag, error) {
	args := m.Called(ctx, entityType, entityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tagSystemModel.SysEntityTag), args.Error(1)
}

// --- Mock AgentRepository ---
type MockAgentRepo struct {
	mock.Mock
}

// Implement only used methods with mock logic, others are empty/panic
func (m *MockAgentRepo) Create(agentData *agentModel.Agent) error {
	args := m.Called(agentData)
	return args.Error(0)
}
func (m *MockAgentRepo) GetByHostnameAndPort(hostname string, port int) (*agentModel.Agent, error) {
	args := m.Called(hostname, port)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agentModel.Agent), args.Error(1)
}
func (m *MockAgentRepo) IsValidCapabilityId(capability string) bool {
	args := m.Called(capability)
	return args.Bool(0)
}
func (m *MockAgentRepo) IsValidCapabilityByName(capability string) bool {
	args := m.Called(capability)
	return args.Bool(0)
}
func (m *MockAgentRepo) IsValidTagId(tag string) bool {
	args := m.Called(tag)
	return args.Bool(0)
}
func (m *MockAgentRepo) GetTagIDsByScanTypeNames(names []string) ([]uint64, error) {
	args := m.Called(names)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *MockAgentRepo) GetTagIDsByScanTypeIDs(ids []string) ([]uint64, error) {
	args := m.Called(ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}

// Unused methods stub
func (m *MockAgentRepo) GetByID(agentID string) (*agentModel.Agent, error)        { return nil, nil }
func (m *MockAgentRepo) GetByHostname(hostname string) (*agentModel.Agent, error) { return nil, nil }
func (m *MockAgentRepo) Update(agentData *agentModel.Agent) error                 { return nil }
func (m *MockAgentRepo) Delete(agentID string) error                              { return nil }
func (m *MockAgentRepo) GetList(page, pageSize int, status *agentModel.AgentStatus, keyword *string, tags []string, capabilities []string) ([]*agentModel.Agent, int64, error) {
	return nil, 0, nil
}
func (m *MockAgentRepo) GetByStatus(status agentModel.AgentStatus) ([]*agentModel.Agent, error) {
	return nil, nil
}
func (m *MockAgentRepo) UpdateStatus(agentID string, status agentModel.AgentStatus) error { return nil }
func (m *MockAgentRepo) UpdateLastHeartbeat(agentID string) error                         { return nil }
func (m *MockAgentRepo) CreateMetrics(metrics *agentModel.AgentMetrics) error             { return nil }
func (m *MockAgentRepo) GetLatestMetrics(agentID string) (*agentModel.AgentMetrics, error) {
	return nil, nil
}
func (m *MockAgentRepo) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	return nil
}
func (m *MockAgentRepo) GetMetricsList(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetrics, int64, error) {
	return nil, 0, nil
}
func (m *MockAgentRepo) GetAllMetrics() ([]*agentModel.AgentMetrics, error) { return nil, nil }
func (m *MockAgentRepo) GetMetricsSince(since time.Time) ([]*agentModel.AgentMetrics, error) {
	return nil, nil
}
func (m *MockAgentRepo) GetMetricsByAgentIDs(agentIDs []string) ([]*agentModel.AgentMetrics, error) {
	return nil, nil
}
func (m *MockAgentRepo) GetMetricsByAgentIDsSince(agentIDs []string, since time.Time) ([]*agentModel.AgentMetrics, error) {
	return nil, nil
}
func (m *MockAgentRepo) AddCapability(agentID string, capabilityID string) error    { return nil }
func (m *MockAgentRepo) RemoveCapability(agentID string, capabilityID string) error { return nil }
func (m *MockAgentRepo) HasCapability(agentID string, capabilityID string) bool     { return false }
func (m *MockAgentRepo) GetCapabilities(agentID string) []string                    { return nil }
func (m *MockAgentRepo) IsValidTagByName(tag string) bool                           { return true }
func (m *MockAgentRepo) AddTag(agentID string, tagID string) error                  { return nil }
func (m *MockAgentRepo) RemoveTag(agentID string, tagID string) error               { return nil }
func (m *MockAgentRepo) HasTag(agentID string, tagID string) bool                   { return false }
func (m *MockAgentRepo) GetTags(agentID string) []string                            { return nil }
func (m *MockAgentRepo) GetAgentIDsByTagIDs(tagIDs []uint64) ([]string, error)      { return nil, nil }

func TestRegisterAgent_WithTaskSupport(t *testing.T) {
	mockRepo := new(MockAgentRepo)
	mockTagSvc := new(MockTagService)
	svc := agentService.NewAgentManagerService(mockRepo, mockTagSvc)

	req := &agentModel.RegisterAgentRequest{
		Hostname:     "test-host-ts",
		IPAddress:    "192.168.1.100",
		Port:         8080,
		Version:      "1.0",
		OS:           "linux",
		Arch:         "amd64",
		CPUCores:     4,
		MemoryTotal:  8192,
		DiskTotal:    102400,
		Capabilities: []string{"cap1"}, // 即使 TaskSupport 存在，Validation 仍会检查 Capabilities
		TaskSupport:  []string{"task1", "task2"},
	}

	// 1. Mock Check Existing (None)
	mockRepo.On("GetByHostnameAndPort", "test-host-ts", 8080).Return(nil, nil)

	// 2. Mock Validation Checks
	// validateRegisterRequest checks IsValidCapabilityId for each capability in req.Capabilities
	// 注意：如果 TaskSupport 存在，validateRegisterRequest 只检查 TaskSupport，不再检查 Capabilities
	// 所以 cap1 不会被检查，除非它也在 TaskSupport 中
	// mockRepo.On("IsValidCapabilityId", "cap1").Return(true)

	// RegisterAgent checks IsValidCapabilityId for each TaskSupport (updated from IsValidCapabilityByName)
	mockRepo.On("IsValidCapabilityId", "task1").Return(true)
	mockRepo.On("IsValidCapabilityId", "task2").Return(true)

	// 3. Mock Create
	mockRepo.On("Create", mock.MatchedBy(func(a *agentModel.Agent) bool {
		// Verify TaskSupport is correctly set
		return len(a.TaskSupport) == 2 && a.TaskSupport[0] == "task1" && a.TaskSupport[1] == "task2" &&
			len(a.Feature) == 0 // Feature should be empty as req.Feature is empty
	})).Return(nil)

	// 4. Mock GetTagIDsByScanTypeIDs (updated from GetTagIDsByScanTypeNames)
	mockRepo.On("GetTagIDsByScanTypeIDs", []string{"task1", "task2"}).Return([]uint64{101, 102}, nil)

	// 5. Mock SyncEntityTags
	// Expect call with tagIDs 101, 102
	mockTagSvc.On("SyncEntityTags",
		mock.Anything,
		"agent",
		mock.AnythingOfType("string"),
		[]uint64{101, 102},
		"agent_capability",
		uint64(0),
	).Return(nil)

	// Execute
	resp, err := svc.RegisterAgent(req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "registered", resp.Status)

	mockRepo.AssertExpectations(t)
	mockTagSvc.AssertExpectations(t)
}

func TestRegisterAgent_Compatibility_TaskSupportFromCapabilities(t *testing.T) {
	mockRepo := new(MockAgentRepo)
	mockTagSvc := new(MockTagService)
	svc := agentService.NewAgentManagerService(mockRepo, mockTagSvc)

	req := &agentModel.RegisterAgentRequest{
		Hostname:     "test-host-compat",
		IPAddress:    "192.168.1.101",
		Port:         8081,
		Version:      "1.0",
		OS:           "linux",
		Arch:         "amd64",
		CPUCores:     4,
		MemoryTotal:  8192,
		DiskTotal:    102400,
		Capabilities: []string{"legacy_task1"},
		TaskSupport:  nil, // Empty, should be filled from Capabilities
	}

	// 1. Mock Check Existing
	mockRepo.On("GetByHostnameAndPort", "test-host-compat", 8081).Return(nil, nil)

	// 2. Validation
	// Capabilities is "legacy_task1"
	mockRepo.On("IsValidCapabilityId", "legacy_task1").Return(true)

	// After compatibility fill, TaskSupport is "legacy_task1"
	mockRepo.On("IsValidCapabilityId", "legacy_task1").Return(true)

	// 3. Create
	mockRepo.On("Create", mock.MatchedBy(func(a *agentModel.Agent) bool {
		// TaskSupport should be filled from Capabilities
		return len(a.TaskSupport) == 1 && a.TaskSupport[0] == "legacy_task1"
	})).Return(nil)

	// 4. GetTagIDs
	mockRepo.On("GetTagIDsByScanTypeIDs", []string{"legacy_task1"}).Return([]uint64{201}, nil)

	// 5. SyncEntityTags
	mockTagSvc.On("SyncEntityTags",
		mock.Anything,
		"agent",
		mock.AnythingOfType("string"),
		[]uint64{201},
		"agent_capability",
		uint64(0),
	).Return(nil)

	resp, err := svc.RegisterAgent(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	mockRepo.AssertExpectations(t)
	mockTagSvc.AssertExpectations(t)
}
