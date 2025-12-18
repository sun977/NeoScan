package test_20251219

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

// --- Minimal Mock TagService ---
type MockTagService struct {
	mock.Mock
}

// Implement only necessary methods
func (m *MockTagService) CreateTag(ctx context.Context, tag *tagSystemModel.SysTag) error { return nil }
func (m *MockTagService) GetTag(ctx context.Context, id uint64) (*tagSystemModel.SysTag, error) { return nil, nil }
func (m *MockTagService) GetTagByName(ctx context.Context, name string) (*tagSystemModel.SysTag, error) { return nil, nil }
func (m *MockTagService) GetTagsByIDs(ctx context.Context, ids []uint64) ([]tagSystemModel.SysTag, error) { return nil, nil }
func (m *MockTagService) UpdateTag(ctx context.Context, tag *tagSystemModel.SysTag) error { return nil }
func (m *MockTagService) MoveTag(ctx context.Context, id, targetParentID uint64) error { return nil }
func (m *MockTagService) DeleteTag(ctx context.Context, id uint64, force bool) error { return nil }
func (m *MockTagService) ListTags(ctx context.Context, req *tagSystemModel.ListTagsRequest) ([]tagSystemModel.SysTag, int64, error) { return nil, 0, nil }
func (m *MockTagService) CreateRule(ctx context.Context, rule *tagSystemModel.SysMatchRule) error { return nil }
func (m *MockTagService) UpdateRule(ctx context.Context, rule *tagSystemModel.SysMatchRule) error { return nil }
func (m *MockTagService) DeleteRule(ctx context.Context, id uint64) error { return nil }
func (m *MockTagService) GetRule(ctx context.Context, id uint64) (*tagSystemModel.SysMatchRule, error) { return nil, nil }
func (m *MockTagService) ListRules(ctx context.Context, req *tagSystemModel.ListRulesRequest) ([]tagSystemModel.SysMatchRule, int64, error) { return nil, 0, nil }
func (m *MockTagService) AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error { return nil }
func (m *MockTagService) SubmitPropagationTask(ctx context.Context, ruleID uint64, action string) (string, error) { return "", nil }
func (m *MockTagService) SubmitEntityPropagationTask(ctx context.Context, entityType string, entityID uint64, tagIDs []uint64, action string) (string, error) { return "", nil }
func (m *MockTagService) SyncEntityTags(ctx context.Context, entityType string, entityID string, targetTagIDs []uint64, sourceScope string, ruleID uint64) error {
	args := m.Called(ctx, entityType, entityID, targetTagIDs, sourceScope, ruleID)
	return args.Error(0)
}
func (m *MockTagService) BootstrapSystemTags(ctx context.Context) error { return nil }
func (m *MockTagService) AddEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64, source string, ruleID uint64) error { return nil }
func (m *MockTagService) RemoveEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64) error { return nil }
func (m *MockTagService) GetEntityTags(ctx context.Context, entityType string, entityID string) ([]tagSystemModel.SysEntityTag, error) { return nil, nil }
func (m *MockTagService) ReloadMatchRules() error { return nil }
func (m *MockTagService) GetEntityIDsByTagIDs(ctx context.Context, entityType string, tagIDs []uint64) ([]string, error) { return nil, nil }
func (m *MockTagService) GetTagByNameAndParent(ctx context.Context, name string, parentID uint64) (*tagSystemModel.SysTag, error) { return nil, nil }
func (m *MockTagService) SyncScanTypesToTags(ctx context.Context) error { return nil }


// --- Minimal Mock AgentRepository ---
type MockAgentRepo struct {
	mock.Mock
}

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
func (m *MockAgentRepo) GetTagIDsByTaskSupportNames(names []string) ([]uint64, error) {
	args := m.Called(names)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}
func (m *MockAgentRepo) GetTagIDsByTaskSupportIDs(ids []string) ([]uint64, error) {
	args := m.Called(ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint64), args.Error(1)
}
// Stub other methods
func (m *MockAgentRepo) GetByID(agentID string) (*agentModel.Agent, error) { return nil, nil }
func (m *MockAgentRepo) GetByHostname(hostname string) (*agentModel.Agent, error) { return nil, nil }
func (m *MockAgentRepo) Update(agentData *agentModel.Agent) error { return nil }
func (m *MockAgentRepo) Delete(agentID string) error { return nil }
func (m *MockAgentRepo) GetList(page, pageSize int, status *agentModel.AgentStatus, keyword *string, tags []string, capabilities []string) ([]*agentModel.Agent, int64, error) { return nil, 0, nil }
func (m *MockAgentRepo) GetByStatus(status agentModel.AgentStatus) ([]*agentModel.Agent, error) { return nil, nil }
func (m *MockAgentRepo) UpdateStatus(agentID string, status agentModel.AgentStatus) error { return nil }
func (m *MockAgentRepo) UpdateLastHeartbeat(agentID string) error { return nil }
func (m *MockAgentRepo) CreateMetrics(metrics *agentModel.AgentMetrics) error { return nil }
func (m *MockAgentRepo) GetLatestMetrics(agentID string) (*agentModel.AgentMetrics, error) { return nil, nil }
func (m *MockAgentRepo) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error { return nil }
func (m *MockAgentRepo) GetMetricsList(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetrics, int64, error) { return nil, 0, nil }
func (m *MockAgentRepo) GetAllMetrics() ([]*agentModel.AgentMetrics, error) { return nil, nil }
func (m *MockAgentRepo) GetMetricsSince(since time.Time) ([]*agentModel.AgentMetrics, error) { return nil, nil }
func (m *MockAgentRepo) GetMetricsByAgentIDs(agentIDs []string) ([]*agentModel.AgentMetrics, error) { return nil, nil }
func (m *MockAgentRepo) GetMetricsByAgentIDsSince(agentIDs []string, since time.Time) ([]*agentModel.AgentMetrics, error) { return nil, nil }
func (m *MockAgentRepo) AddCapability(agentID string, capabilityID string) error { return nil }
func (m *MockAgentRepo) RemoveCapability(agentID string, capabilityID string) error { return nil }
func (m *MockAgentRepo) HasCapability(agentID string, capabilityID string) bool { return false }
func (m *MockAgentRepo) GetCapabilities(agentID string) []string { return nil }
func (m *MockAgentRepo) IsValidTagByName(tag string) bool { return true }
func (m *MockAgentRepo) AddTag(agentID string, tagID string) error { return nil }
func (m *MockAgentRepo) RemoveTag(agentID string, tagID string) error { return nil }
func (m *MockAgentRepo) HasTag(agentID string, tagID string) bool { return false }
func (m *MockAgentRepo) GetTags(agentID string) []string { return nil }
func (m *MockAgentRepo) GetAgentIDsByTagIDs(tagIDs []uint64) ([]string, error) { return nil, nil }
func (m *MockAgentRepo) AddTaskSupport(agentID string, taskID string) error { return nil }
func (m *MockAgentRepo) GetAllScanTypes() ([]*agentModel.ScanType, error) { return nil, nil }
func (m *MockAgentRepo) UpdateScanType(scanType *agentModel.ScanType) error { return nil }
func (m *MockAgentRepo) GetTaskSupport(agentID string) []string { return nil }
func (m *MockAgentRepo) HasTaskSupport(agentID string, taskID string) bool { return false }
func (m *MockAgentRepo) RemoveTaskSupport(agentID string, taskID string) error { return nil }
func (m *MockAgentRepo) IsValidTaskSupportId(taskID string) bool { return true }
func (m *MockAgentRepo) IsValidTaskSupportByName(taskName string) bool { return true }
func (m *MockAgentRepo) IsValidTagId(tag string) bool { return true }


func TestRegisterAgent_WithInvalidTaskSupport(t *testing.T) {
	mockRepo := new(MockAgentRepo)
	mockTagSvc := new(MockTagService)
	svc := agentService.NewAgentManagerService(mockRepo, mockTagSvc)

	req := &agentModel.RegisterAgentRequest{
		Hostname:    "test-host-invalid",
		IPAddress:   "192.168.1.101",
		Port:        8080,
		Version:     "1.0",
		OS:          "linux",
		Arch:        "amd64",
		CPUCores:    4,
		MemoryTotal: 8192,
		DiskTotal:   102400,
		TaskSupport: []string{"fake_task"},
	}

	// 1. Mock Check Existing (None)
	mockRepo.On("GetByHostnameAndPort", "test-host-invalid", 8080).Return(nil, nil)

	// 2. Mock Create
	mockRepo.On("Create", mock.Anything).Return(nil)

	// 3. Mock GetTagIDsByTaskSupportNames -> empty (not found)
	mockRepo.On("GetTagIDsByTaskSupportNames", []string{"fake_task"}).Return([]uint64{}, nil)
	
	// 4. Mock GetTagIDsByTaskSupportIDs -> empty (not found either)
	mockRepo.On("GetTagIDsByTaskSupportIDs", []string{"fake_task"}).Return([]uint64{}, nil)

	// Execute
	resp, err := svc.RegisterAgent(req)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "无效的TaskSupport")

	mockRepo.AssertExpectations(t)
}

func TestRegisterAgent_WithValidTaskSupport(t *testing.T) {
	mockRepo := new(MockAgentRepo)
	mockTagSvc := new(MockTagService)
	svc := agentService.NewAgentManagerService(mockRepo, mockTagSvc)

	req := &agentModel.RegisterAgentRequest{
		Hostname:    "test-host-valid",
		IPAddress:   "192.168.1.102",
		Port:        8080,
		Version:     "1.0",
		OS:          "linux",
		Arch:        "amd64",
		CPUCores:    4,
		MemoryTotal: 8192,
		DiskTotal:   102400,
		TaskSupport: []string{"valid_task"},
	}

	mockRepo.On("GetByHostnameAndPort", "test-host-valid", 8080).Return(nil, nil)
	mockRepo.On("Create", mock.Anything).Return(nil)
	
	// Found by Name
	mockRepo.On("GetTagIDsByTaskSupportNames", []string{"valid_task"}).Return([]uint64{123}, nil)
	
	// Sync Tags
	mockTagSvc.On("SyncEntityTags", mock.Anything, "agent", mock.Anything, []uint64{123}, "agent_capability", uint64(0)).Return(nil)

	resp, err := svc.RegisterAgent(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "registered", resp.Status)
}
