package etl

import (
	"context"
	"testing"
	"time"

	assetModel "neomaster/internal/model/asset"
	assetRepo "neomaster/internal/repo/mysql/asset"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	err = db.AutoMigrate(
		&assetModel.AssetHost{},
		&assetModel.AssetService{},
		&assetModel.AssetWeb{},
		&assetModel.AssetWebDetail{},
		&assetModel.AssetUnified{},
		&assetModel.AssetVuln{},
	)
	if err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestAssetMerger_WebAssets_CreateWebAndDetail(t *testing.T) {
	db := newTestDB(t)
	hostRepo := assetRepo.NewAssetHostRepository(db)
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)

	merger := NewAssetMerger(hostRepo, webRepo, vulnRepo, unifiedRepo)

	ctx := context.Background()

	bundle := &AssetBundle{
		ProjectID: 1,
		Host: &assetModel.AssetHost{
			IP:             "10.0.0.1",
			Tags:           "{}",
			SourceStageIDs: "[]",
		},
		WebAssets: []*WebAsset{
			{
				Web: &assetModel.AssetWeb{
					URL:       "http://example.com/login",
					Domain:    "example.com",
					TechStack: "[\"nginx\"]",
					BasicInfo: "{\"title\":\"Login\"}",
					Tags:      "{}",
				},
				Detail: &assetModel.AssetWebDetail{
					ContentDetails: "{\"url\":\"http://example.com/login\"}",
					Screenshot:     "mock",
				},
			},
		},
	}

	err := merger.Merge(ctx, bundle)
	assert.NoError(t, err)

	web, err := webRepo.GetWebByURL(ctx, "http://example.com/login")
	assert.NoError(t, err)
	assert.NotNil(t, web)
	assert.NotZero(t, web.ID)

	detail, err := webRepo.GetDetailByWebID(ctx, web.ID)
	assert.NoError(t, err)
	assert.NotNil(t, detail)
	assert.Equal(t, web.ID, detail.AssetWebID)
	assert.Equal(t, "mock", detail.Screenshot)
}

func TestAssetMerger_Vuln_ServiceTargetCreatesStubService(t *testing.T) {
	db := newTestDB(t)
	hostRepo := assetRepo.NewAssetHostRepository(db)
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)

	merger := NewAssetMerger(hostRepo, webRepo, vulnRepo, unifiedRepo)

	ctx := context.Background()
	now := time.Now()

	bundle := &AssetBundle{
		ProjectID: 1,
		Host: &assetModel.AssetHost{
			IP:             "10.0.0.2",
			Tags:           "{}",
			SourceStageIDs: "[]",
		},
		Vulns: []*assetModel.AssetVuln{
			{
				TargetType:  "service",
				CVE:         "CVE-2021-44228",
				IDAlias:     "SCAN-123",
				Severity:    "critical",
				Confidence:  0.9,
				Attributes:  "{\"port\":8080}",
				Evidence:    "{\"raw\":\"x\"}",
				FirstSeenAt: &now,
				Status:      "open",
			},
		},
	}

	err := merger.Merge(ctx, bundle)
	assert.NoError(t, err)

	host, err := hostRepo.GetHostByIP(ctx, "10.0.0.2")
	assert.NoError(t, err)
	assert.NotNil(t, host)

	svc, err := hostRepo.GetServiceByHostIDAndPort(ctx, host.ID, 8080, "tcp")
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	existing, err := vulnRepo.GetVulnByTargetAndCVE(ctx, "service", svc.ID, "CVE-2021-44228")
	assert.NoError(t, err)
	assert.NotNil(t, existing)
	assert.Equal(t, "service", existing.TargetType)
	assert.Equal(t, svc.ID, existing.TargetRefID)
}
