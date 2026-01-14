package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	assetModel "neomaster/internal/model/asset"
	assetRepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/asset/etl"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newNeoScanDevDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open mysql neoscan_dev failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying sql db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(2 * time.Minute)

	if err := db.AutoMigrate(
		&assetModel.AssetHost{},
		&assetModel.AssetService{},
		&assetModel.AssetWeb{},
		&assetModel.AssetWebDetail{},
		&assetModel.AssetUnified{},
		&assetModel.AssetVuln{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return db
}

func newTestNamespace(t *testing.T) (ip string, projectID uint64, aliasPrefix string) {
	n := time.Now().UnixNano()
	ip = fmt.Sprintf("172.31.%d.%d", (n/1e6)%250+1, (n/1e3)%250+1)
	projectID = uint64(900000 + (n % 100000))
	aliasPrefix = fmt.Sprintf("TEST-%d", n)
	return
}

func cleanupNamespace(t *testing.T, db *gorm.DB, hostID uint64, ip string, projectID uint64, aliasPrefix string) {
	_ = db.Unscoped().Where("host_id = ?", hostID).Delete(&assetModel.AssetService{}).Error
	_ = db.Unscoped().Where("target_type = ? AND target_ref_id = ? AND id_alias LIKE ?", "host", hostID, aliasPrefix+"%").Delete(&assetModel.AssetVuln{}).Error
	_ = db.Unscoped().Where("target_type = ? AND target_ref_id IN (?) AND id_alias LIKE ?", "service", db.Table("asset_services").Select("id").Where("host_id = ?", hostID), aliasPrefix+"%").Delete(&assetModel.AssetVuln{}).Error
	_ = db.Unscoped().Where("project_id = ? AND ip = ?", projectID, ip).Delete(&assetModel.AssetUnified{}).Error
	_ = db.Unscoped().Where("ip = ?", ip).Delete(&assetModel.AssetHost{}).Error
}

func TestVulnIdempotency_UpsertSameIdentity(t *testing.T) {
	ctx := context.Background()
	db := newNeoScanDevDB(t)
	ip, projectID, aliasPrefix := newTestNamespace(t)

	hostRepo := assetRepo.NewAssetHostRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)

	host := &assetModel.AssetHost{
		IP:             ip,
		OS:             "Linux",
		Tags:           "{}",
		SourceStageIDs: "[]",
	}
	if err := hostRepo.CreateHost(ctx, host); err != nil {
		t.Fatalf("create host failed: %v", err)
	}
	defer cleanupNamespace(t, db, host.ID, ip, projectID, aliasPrefix)

	now := time.Now()
	later := now.Add(5 * time.Minute)

	identityTargetType := "host"
	identityTargetID := host.ID
	identityAlias := aliasPrefix + "-RULE-ETL-001"

	v1 := &assetModel.AssetVuln{
		TargetType:  identityTargetType,
		TargetRefID: identityTargetID,
		CVE:         "CVE-2025-0001",
		IDAlias:     identityAlias,
		Severity:    "high",
		Confidence:  0.8,
		Attributes:  "{\"port\":8080}",
		Evidence:    "{\"raw\":\"payload-1\"}",
		FirstSeenAt: &now,
		LastSeenAt:  &now,
		Status:      "open",
	}
	if err := vulnRepo.UpsertVuln(ctx, v1); err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}

	v2 := &assetModel.AssetVuln{
		TargetType:  identityTargetType,
		TargetRefID: identityTargetID,
		CVE:         "CVE-2025-0001",
		IDAlias:     identityAlias,
		Severity:    "critical",
		Confidence:  0.95,
		Attributes:  "{\"port\":8080}",
		Evidence:    "{\"raw\":\"payload-2\"}",
		FirstSeenAt: &now,
		LastSeenAt:  &later,
		Status:      "confirmed",
	}
	if err := vulnRepo.UpsertVuln(ctx, v2); err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}

	var count int64
	if err := db.Model(&assetModel.AssetVuln{}).
		Where("target_type = ? AND target_ref_id = ? AND id_alias = ?", identityTargetType, identityTargetID, identityAlias).
		Count(&count).Error; err != nil {
		t.Fatalf("count vulns failed: %v", err)
	}
	assert.Equal(t, int64(1), count)

	existing, err := vulnRepo.GetVulnByTargetAndAlias(ctx, identityTargetType, identityTargetID, identityAlias)
	if err != nil {
		t.Fatalf("get vuln by alias failed: %v", err)
	}
	if existing == nil {
		t.Fatalf("vuln not found")
	}
	assert.Equal(t, "critical", existing.Severity)
	assert.Equal(t, "confirmed", existing.Status)
	assert.Equal(t, "CVE-2025-0001", existing.CVE)
	assert.Equal(t, "{\"raw\":\"payload-2\"}", existing.Evidence)
	assert.NotNil(t, existing.LastSeenAt)
}

func TestVulnIdempotency_UpsertDifferentAlias(t *testing.T) {
	ctx := context.Background()
	db := newNeoScanDevDB(t)
	ip, projectID, aliasPrefix := newTestNamespace(t)

	hostRepo := assetRepo.NewAssetHostRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)

	host := &assetModel.AssetHost{
		IP:             ip,
		OS:             "Linux",
		Tags:           "{}",
		SourceStageIDs: "[]",
	}
	if err := hostRepo.CreateHost(ctx, host); err != nil {
		t.Fatalf("create host failed: %v", err)
	}
	defer cleanupNamespace(t, db, host.ID, ip, projectID, aliasPrefix)

	now := time.Now()

	v1 := &assetModel.AssetVuln{
		TargetType:  "host",
		TargetRefID: host.ID,
		CVE:         "CVE-2025-0002",
		IDAlias:     aliasPrefix + "-RULE-ETL-002",
		Severity:    "medium",
		Attributes:  "{}",
		Evidence:    "{}",
		LastSeenAt:  &now,
		Status:      "open",
	}
	if err := vulnRepo.UpsertVuln(ctx, v1); err != nil {
		t.Fatalf("upsert v1 failed: %v", err)
	}

	v2 := &assetModel.AssetVuln{
		TargetType:  "host",
		TargetRefID: host.ID,
		CVE:         "CVE-2025-0003",
		IDAlias:     aliasPrefix + "-RULE-ETL-003",
		Severity:    "high",
		Attributes:  "{}",
		Evidence:    "{}",
		LastSeenAt:  &now,
		Status:      "open",
	}
	if err := vulnRepo.UpsertVuln(ctx, v2); err != nil {
		t.Fatalf("upsert v2 failed: %v", err)
	}

	var count int64
	if err := db.Model(&assetModel.AssetVuln{}).
		Where("target_type = ? AND target_ref_id = ?", "host", host.ID).
		Count(&count).Error; err != nil {
		t.Fatalf("count vulns failed: %v", err)
	}
	assert.Equal(t, int64(2), count)
}

func TestVulnIdempotency_CreateVulnBlockedByUniqueConstraint(t *testing.T) {
	ctx := context.Background()
	db := newNeoScanDevDB(t)
	ip, projectID, aliasPrefix := newTestNamespace(t)

	hostRepo := assetRepo.NewAssetHostRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)

	host := &assetModel.AssetHost{
		IP:             ip,
		OS:             "Linux",
		Tags:           "{}",
		SourceStageIDs: "[]",
	}
	if err := hostRepo.CreateHost(ctx, host); err != nil {
		t.Fatalf("create host failed: %v", err)
	}
	defer cleanupNamespace(t, db, host.ID, ip, projectID, aliasPrefix)

	now := time.Now()

	v1 := &assetModel.AssetVuln{
		TargetType:  "host",
		TargetRefID: host.ID,
		CVE:         "CVE-2025-0005",
		IDAlias:     aliasPrefix + "-RULE-ETL-005",
		Severity:    "high",
		Attributes:  "{}",
		Evidence:    "{}",
		LastSeenAt:  &now,
		Status:      "open",
	}
	err := vulnRepo.CreateVuln(ctx, v1)
	assert.NoError(t, err)

	v2 := &assetModel.AssetVuln{
		TargetType:  "host",
		TargetRefID: host.ID,
		CVE:         "CVE-2025-0005",
		IDAlias:     aliasPrefix + "-RULE-ETL-005",
		Severity:    "critical",
		Attributes:  "{}",
		Evidence:    "{}",
		LastSeenAt:  &now,
		Status:      "confirmed",
	}
	err = vulnRepo.CreateVuln(ctx, v2)
	assert.Error(t, err)
}

func TestVulnIdempotency_ETLMergeTwiceNoDuplicate(t *testing.T) {
	ctx := context.Background()
	db := newNeoScanDevDB(t)
	ip, projectID, aliasPrefix := newTestNamespace(t)

	hostRepo := assetRepo.NewAssetHostRepository(db)
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)

	merger := etl.NewAssetMerger(hostRepo, webRepo, vulnRepo, unifiedRepo)

	now := time.Now()
	v := &assetModel.AssetVuln{
		TargetType:  "service",
		CVE:         "CVE-2025-0004",
		IDAlias:     aliasPrefix + "-RULE-ETL-004",
		Severity:    "critical",
		Confidence:  0.9,
		Attributes:  "{\"port\":8081}",
		Evidence:    "{\"raw\":\"payload-X\"}",
		FirstSeenAt: &now,
		Status:      "open",
	}

	bundle := &etl.AssetBundle{
		ProjectID: projectID,
		Host: &assetModel.AssetHost{
			IP:             ip,
			Tags:           "{}",
			SourceStageIDs: "[]",
		},
		Vulns: []*assetModel.AssetVuln{v},
	}

	if err := merger.Merge(ctx, bundle); err != nil {
		t.Fatalf("first merge failed: %v", err)
	}
	if err := merger.Merge(ctx, bundle); err != nil {
		t.Fatalf("second merge failed: %v", err)
	}

	host, err := hostRepo.GetHostByIP(ctx, ip)
	if err != nil {
		t.Fatalf("get host failed: %v", err)
	}
	if host == nil {
		t.Fatalf("host not found")
	}
	defer cleanupNamespace(t, db, host.ID, ip, projectID, aliasPrefix)

	svc, err := hostRepo.GetServiceByHostIDAndPort(ctx, host.ID, 8081, "tcp")
	if err != nil {
		t.Fatalf("get service failed: %v", err)
	}
	if svc == nil {
		t.Fatalf("service not found")
	}

	var count int64
	if err := db.Model(&assetModel.AssetVuln{}).
		Where("target_type = ? AND target_ref_id = ? AND id_alias = ?", "service", svc.ID, aliasPrefix+"-RULE-ETL-004").
		Count(&count).Error; err != nil {
		t.Fatalf("count vulns failed: %v", err)
	}
	assert.Equal(t, int64(1), count)
}

func TestVulnIdempotency_ConcurrentUpsertSameIdentity(t *testing.T) {
	ctx := context.Background()
	db := newNeoScanDevDB(t)
	ip, projectID, aliasPrefix := newTestNamespace(t)

	hostRepo := assetRepo.NewAssetHostRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)

	host := &assetModel.AssetHost{
		IP:             ip,
		OS:             "Linux",
		Tags:           "{}",
		SourceStageIDs: "[]",
	}
	if err := hostRepo.CreateHost(ctx, host); err != nil {
		t.Fatalf("create host failed: %v", err)
	}
	defer cleanupNamespace(t, db, host.ID, ip, projectID, aliasPrefix)

	identityAlias := aliasPrefix + "-CONCURRENT"

	workers := 32
	iters := 5

	errCh := make(chan error, workers*iters)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				now := time.Now()
				v := &assetModel.AssetVuln{
					TargetType:  "host",
					TargetRefID: host.ID,
					CVE:         "CVE-2025-9999",
					IDAlias:     identityAlias,
					Severity:    "high",
					Confidence:  0.9,
					Attributes:  "{}",
					Evidence:    fmt.Sprintf("{\"worker\":%d,\"iter\":%d}", worker, j),
					LastSeenAt:  &now,
					Status:      "open",
				}
				if err := vulnRepo.UpsertVuln(ctx, v); err != nil {
					errCh <- err
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent upsert failed: %v", err)
		}
	}

	var count int64
	if err := db.Model(&assetModel.AssetVuln{}).
		Where("target_type = ? AND target_ref_id = ? AND id_alias = ?", "host", host.ID, identityAlias).
		Count(&count).Error; err != nil {
		t.Fatalf("count vulns failed: %v", err)
	}
	assert.Equal(t, int64(1), count)
}
