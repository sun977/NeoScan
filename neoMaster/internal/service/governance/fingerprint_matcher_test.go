package governance

import (
	"context"
	"testing"

	"neomaster/internal/model/asset"
	tagModel "neomaster/internal/model/tag_system"
	repo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/fingerprint"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// --- Mocks ---

type MockFingerprintService struct {
	mock.Mock
}

func (m *MockFingerprintService) Identify(ctx context.Context, input *fingerprint.Input) (*fingerprint.Result, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*fingerprint.Result), args.Error(1)
}

func (m *MockFingerprintService) LoadRules(dir string) error {
	return nil
}

func (m *MockFingerprintService) GetStats() map[string]int {
	return nil
}

type MockTagService struct {
	mock.Mock
}

// Implement only necessary methods for interface compliance
func (m *MockTagService) CreateTag(ctx context.Context, tag *tagModel.SysTag) error { return nil }
func (m *MockTagService) GetTag(ctx context.Context, id uint64) (*tagModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) GetTagByName(ctx context.Context, name string) (*tagModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) GetTagByNameAndParent(ctx context.Context, name string, parentID uint64) (*tagModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) GetTagsByIDs(ctx context.Context, ids []uint64) ([]tagModel.SysTag, error) {
	return nil, nil
}
func (m *MockTagService) UpdateTag(ctx context.Context, tag *tagModel.SysTag) error    { return nil }
func (m *MockTagService) MoveTag(ctx context.Context, id, targetParentID uint64) error { return nil }
func (m *MockTagService) DeleteTag(ctx context.Context, id uint64, force bool) error   { return nil }
func (m *MockTagService) ListTags(ctx context.Context, req *tagModel.ListTagsRequest) ([]tagModel.SysTag, int64, error) {
	return nil, 0, nil
}
func (m *MockTagService) CreateRule(ctx context.Context, rule *tagModel.SysMatchRule) error {
	return nil
}
func (m *MockTagService) UpdateRule(ctx context.Context, rule *tagModel.SysMatchRule) error {
	return nil
}
func (m *MockTagService) DeleteRule(ctx context.Context, id uint64) error { return nil }
func (m *MockTagService) GetRule(ctx context.Context, id uint64) (*tagModel.SysMatchRule, error) {
	return nil, nil
}
func (m *MockTagService) ListRules(ctx context.Context, req *tagModel.ListRulesRequest) ([]tagModel.SysMatchRule, int64, error) {
	return nil, 0, nil
}
func (m *MockTagService) ReloadMatchRules() error { return nil }
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
func (m *MockTagService) GetEntityTags(ctx context.Context, entityType string, entityID string) ([]tagModel.SysEntityTag, error) {
	return nil, nil
}
func (m *MockTagService) GetEntityIDsByTagIDs(ctx context.Context, entityType string, tagIDs []uint64) ([]string, error) {
	return nil, nil
}

func (m *MockTagService) AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error {
	args := m.Called(ctx, entityType, entityID, attributes)
	return args.Error(0)
}

// --- Tests ---

func TestFingerprintMatcher_ProcessBatch(t *testing.T) {
	// 1. Setup DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&asset.AssetHost{}, &asset.AssetService{})
	assert.NoError(t, err)

	assetRepo := repo.NewAssetHostRepository(db)

	// 2. Setup Data
	host := &asset.AssetHost{IP: "192.168.1.1"}
	err = assetRepo.CreateHost(context.Background(), host)
	assert.NoError(t, err)

	svc1 := &asset.AssetService{
		HostID: host.ID,
		Port:   80,
		Proto:  "tcp",
		Banner: "Apache/2.4.41",
	}
	err = assetRepo.CreateService(context.Background(), svc1)
	assert.NoError(t, err)

	// svc2 has product, should be skipped
	svc2 := &asset.AssetService{
		HostID:  host.ID,
		Port:    22,
		Proto:   "tcp",
		Banner:  "SSH",
		Product: "OpenSSH",
	}
	err = assetRepo.CreateService(context.Background(), svc2)
	assert.NoError(t, err)

	// 3. Setup Mocks
	mockFp := new(MockFingerprintService)
	mockTag := new(MockTagService)

	// Expect Identify call for svc1
	match := fingerprint.Match{
		Product: "Apache HTTP Server",
		Version: "2.4.41",
		Vendor:  "Apache",
		CPE:     "cpe:/a:apache:http_server:2.4.41",
	}
	result := &fingerprint.Result{
		Best: &match,
	}
	mockFp.On("Identify", mock.Anything, mock.MatchedBy(func(input *fingerprint.Input) bool {
		return input.Banner == "Apache/2.4.41" && input.Target == "192.168.1.1"
	})).Return(result, nil)

	// Expect AutoTag call
	mockTag.On("AutoTag", mock.Anything, "service", mock.AnythingOfType("string"), mock.Anything).Return(nil)

	// 4. Run Matcher
	matcher := NewFingerprintMatcher(assetRepo, mockFp, mockTag)
	count, err := matcher.ProcessBatch(context.Background(), 10, true)

	// 5. Assertions
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify svc1 updated
	updatedSvc1, _ := assetRepo.GetServiceByID(context.Background(), svc1.ID)
	assert.Equal(t, "Apache HTTP Server", updatedSvc1.Product)
	assert.Equal(t, "2.4.41", updatedSvc1.Version)
	assert.Equal(t, "cpe:/a:apache:http_server:2.4.41", updatedSvc1.CPE)

	// Verify svc2 untouched
	updatedSvc2, _ := assetRepo.GetServiceByID(context.Background(), svc2.ID)
	assert.Equal(t, "OpenSSH", updatedSvc2.Product)

	mockFp.AssertExpectations(t)
	mockTag.AssertExpectations(t)
}

func TestFingerprintMatcher_ProcessBatch_NoMatch(t *testing.T) {
	// 1. Setup DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&asset.AssetHost{}, &asset.AssetService{})
	assert.NoError(t, err)

	assetRepo := repo.NewAssetHostRepository(db)

	host := &asset.AssetHost{IP: "10.0.0.1"}
	assetRepo.CreateHost(context.Background(), host)

	svc := &asset.AssetService{
		HostID: host.ID,
		Port:   8080,
		Banner: "UnknownBanner",
	}
	assetRepo.CreateService(context.Background(), svc)

	// 2. Mocks
	mockFp := new(MockFingerprintService)
	mockTag := new(MockTagService) // Not called

	// Return nil match
	mockFp.On("Identify", mock.Anything, mock.Anything).Return(nil, nil)

	// 3. Run
	matcher := NewFingerprintMatcher(assetRepo, mockFp, mockTag)
	count, err := matcher.ProcessBatch(context.Background(), 10, false)

	// 4. Assertions
	assert.NoError(t, err)
	assert.Equal(t, 0, count) // processedCount increments only on match update?
	// Wait, code says processedCount++ inside `if result != nil && result.Best != nil`.
	// If not match, it goes to else block and updates to "unknown".
	// But processedCount is NOT incremented in else block in my implementation?
	// Let's check implementation.

	// Implementation:
	// if result != nil ... { ... processedCount++ } else { svc.Product="unknown"; update... }
	// So processedCount will be 0.

	// Verify svc updated to "unknown"
	updatedSvc, _ := assetRepo.GetServiceByID(context.Background(), svc.ID)
	assert.Equal(t, "unknown", updatedSvc.Product)
}
