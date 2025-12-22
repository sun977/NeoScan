package test_20251222

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

// --- Mocks ---

type MockTagService struct {
	mock.Mock
}

func (m *MockTagService) SyncEntityTags(ctx context.Context, entityType string, entityID string, targetTagIDs []uint64, sourceScope string, ruleID uint64) error {
	args := m.Called(ctx, entityType, entityID, targetTagIDs, sourceScope, ruleID)
	return args.Error(0)
}

// Stub others
func (m *MockTagService) CreateTag(ctx context.Context, tag *tagSystemModel.SysTag) error { return nil }
func (m *MockTagService) GetTag(ctx context.Context, id uint64) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) GetTagByName(ctx context.Context, name string) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) GetTagsByIDs(ctx context.Context, ids []uint64) ([]tagSystemModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) UpdateTag(ctx context.Context, tag *tagSystemModel.SysTag) error { return nil }
func (m *MockTagService) MoveTag(ctx context.Context, id, targetParentID uint64) error    { return nil }
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
func (m *MockTagService) AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error {
	return nil
}
func (m *MockTagService) SubmitPropagationTask(ctx context.Context, ruleID uint64, action string) (string, error) {
	return "", nil
}
func (m *MockTagService) SubmitEntityPropagationTask(ctx context.Context, entityType string, entityID uint64, tagIDs []uint64, action string) (string, error) {
	return "", nil
}
func (m *MockTagService) BootstrapSystemTags(ctx context.Context) error { return nil }
func (m *MockTagService) AddEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64, source string, ruleID uint64) error {
	return nil
}
func (m *MockTagService) RemoveEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64) error {
	return nil
}
func (m *MockTagService) GetEntityTags(ctx context.Context, entityType string, entityID string) ([]tagSystemModel.SysEntityTag, error) {
	return nil, nil
}
func (m *MockTagService) ReloadMatchRules() error { return nil }
func (m *MockTagService) GetEntityIDsByTagIDs(ctx context.Context, entityType string, tagIDs []uint64) ([]string, error) {
	return nil, nil
}
func (m *MockTagService) GetTagByNameAndParent(ctx context.Context, name string, parentID uint64) (*tagSystemModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) SyncScanTypesToTags(ctx context.Context) error { return nil }

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
func (m *MockAgentRepo) Update(agentData *agentModel.Agent) error {
	args := m.Called(agentData)
	return args.Error(0)
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

// Stub others
func (m *MockAgentRepo) GetByID(agentID string) (*agentModel.Agent, error)        { return nil, nil }
func (m *MockAgentRepo) GetByHostname(hostname string) (*agentModel.Agent, error) { return nil, nil }
func (m *MockAgentRepo) Delete(agentID string) error                              { return nil }
func (m *MockAgentRepo) GetList(page, pageSize int, status *agentModel.AgentStatus, keyword *string, tags []string, taskSupport []string) ([]*agentModel.Agent, int64, error) {
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

// Stub missing methods
func (m *MockAgentRepo) IsValidTaskSupportId(taskID string) bool               { return true }
func (m *MockAgentRepo) IsValidTaskSupportByName(taskName string) bool         { return true }
func (m *MockAgentRepo) AddTaskSupport(agentID string, taskID string) error    { return nil }
func (m *MockAgentRepo) RemoveTaskSupport(agentID string, taskID string) error { return nil }
func (m *MockAgentRepo) HasTaskSupport(agentID string, taskID string) bool     { return false }
func (m *MockAgentRepo) GetTaskSupport(agentID string) []string                { return nil }

func (m *MockAgentRepo) GetAllScanTypes() ([]*agentModel.ScanType, error)    { return nil, nil }
func (m *MockAgentRepo) GetScanType(id string) (*agentModel.ScanType, error) { return nil, nil }
func (m *MockAgentRepo) CreateScanType(scanType *agentModel.ScanType) error  { return nil }
func (m *MockAgentRepo) UpdateScanType(scanType *agentModel.ScanType) error  { return nil }
func (m *MockAgentRepo) DeleteScanType(id string) error                      { return nil }
func (m *MockAgentRepo) ActivateScanType(id string) error                    { return nil }
func (m *MockAgentRepo) DeactivateScanType(id string) error                  { return nil }

// --- Tests ---

func TestRegisterAgent_UpdateMode(t *testing.T) {
	// Setup
	mockRepo := new(MockAgentRepo)
	mockTagSvc := new(MockTagService)
	svc := agentService.NewAgentManagerService(mockRepo, mockTagSvc)

	existingAgent := &agentModel.Agent{
		AgentID:     "agent_test_123",
		Hostname:    "test-host",
		Port:        8080,
		Token:       "token_abc_123",
		TokenExpiry: time.Now().Add(time.Hour),
		TaskSupport: []string{"scan_a"},
	}

	// Case 1: Update Mode Success (Valid ID & Token)
	t.Run("UpdateMode_Success", func(t *testing.T) {
		req := &agentModel.RegisterAgentRequest{
			Hostname:    "test-host",
			Port:        8080,
			IPAddress:   "127.0.0.1",
			Version:     "1.1.0", // New version
			OS:          "linux",
			Arch:        "amd64",
			TaskSupport: []string{"scan_b"}, // Changed capability
			AgentID:     "agent_test_123",
			Token:       "token_abc_123",
		}

		// Mock Expectations
		mockRepo.On("GetByHostnameAndPort", "test-host", 8080).Return(existingAgent, nil).Once()
		mockRepo.On("Update", mock.MatchedBy(func(a *agentModel.Agent) bool {
			return a.AgentID == "agent_test_123" && a.Version == "1.1.0" && a.TaskSupport[0] == "scan_b"
		})).Return(nil).Once()

		// Tag Sync Mocks
		mockRepo.On("GetTagIDsByTaskSupportNames", []string{"scan_b"}).Return([]uint64{20}, nil).Once()
		mockTagSvc.On("SyncEntityTags", mock.Anything, "agent", "agent_test_123", []uint64{20}, "agent_register_update", uint64(0)).Return(nil).Once()

		resp, err := svc.RegisterAgent(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "agent_test_123", resp.AgentID)
		assert.Equal(t, "token_abc_123", resp.Token) // Should return existing token
	})

	// Case 2: Conflict (Missing Token)
	t.Run("Conflict_NoToken", func(t *testing.T) {
		req := &agentModel.RegisterAgentRequest{
			Hostname:    "test-host",
			Port:        8080,
			IPAddress:   "127.0.0.1",
			Version:     "1.0.0",
			OS:          "linux",
			Arch:        "amd64",
			TaskSupport: []string{"scan_a"},
			// No AgentID/Token
		}

		mockRepo.On("GetByHostnameAndPort", "test-host", 8080).Return(existingAgent, nil).Once()

		resp, err := svc.RegisterAgent(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "already exists")
	})

	// Case 3: Conflict (Wrong Token)
	t.Run("Conflict_WrongToken", func(t *testing.T) {
		req := &agentModel.RegisterAgentRequest{
			Hostname:    "test-host",
			Port:        8080,
			IPAddress:   "127.0.0.1",
			Version:     "1.0.0",
			OS:          "linux",
			Arch:        "amd64",
			TaskSupport: []string{"scan_a"},
			AgentID:     "agent_test_123",
			Token:       "wrong_token",
		}

		mockRepo.On("GetByHostnameAndPort", "test-host", 8080).Return(existingAgent, nil).Once()

		resp, err := svc.RegisterAgent(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "already exists")
	})
}
