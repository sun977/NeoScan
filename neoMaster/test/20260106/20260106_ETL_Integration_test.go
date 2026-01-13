package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"neomaster/internal/config"
	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/database"
	"neomaster/internal/pkg/logger"
	assetRepo "neomaster/internal/repo/mysql/asset"
	"neomaster/internal/service/asset/etl"
	"neomaster/internal/service/orchestrator/ingestor"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// SetupETLTestEnv 初始化 ETL 测试环境
func SetupETLTestEnv() (*gorm.DB, etl.ResultProcessor, ingestor.ResultQueue, error) {
	// 1. 初始化日志
	logger.InitLogger(&config.LogConfig{
		Level:  "info",
		Format: "console",
		Output: "console",
	})

	// 2. 数据库配置 (neoscan_dev)
	dbConfig := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		Username:        "root",
		Password:        "ROOT",
		Database:        "neoscan_dev",
		Charset:         "utf8mb4",
		ParseTime:       true,
		Loc:             "Local",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
	}

	// 连接 MySQL
	db, err := database.NewMySQLConnection(dbConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to mysql: %v", err)
	}

	// 3. 清理数据
	db.Exec("DELETE FROM asset_services")
	db.Exec("DELETE FROM asset_hosts")

	// 4. 初始化组件
	hostRepo := assetRepo.NewAssetHostRepository(db)
	webRepo := assetRepo.NewAssetWebRepository(db)
	vulnRepo := assetRepo.NewAssetVulnRepository(db)
	unifiedRepo := assetRepo.NewAssetUnifiedRepository(db)
	merger := etl.NewAssetMerger(hostRepo, webRepo, vulnRepo, unifiedRepo)

	queue := ingestor.NewMemoryQueue(100)
	// 在测试中暂时不注入 FingerprintService (nil)
	processor := etl.NewResultProcessor(queue, merger, 2) // 2 workers

	return db, processor, queue, nil
}

func TestETLIntegration_PortScan(t *testing.T) {
	db, processor, queue, err := SetupETLTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 启动 Processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	processor.Start(ctx)
	defer processor.Stop()

	// 构造测试数据
	attributes := etl.PortScanAttributes{}
	attributes.Ports = []struct {
		IP          string `json:"ip"`
		Port        int    `json:"port"`
		Proto       string `json:"proto"`
		State       string `json:"state"`
		ServiceHint string `json:"service_hint"`
		Banner      string `json:"banner"`
	}{
		{IP: "192.168.1.100", Port: 80, Proto: "tcp", State: "open", ServiceHint: "http", Banner: "nginx/1.18"},
		{IP: "192.168.1.100", Port: 22, Proto: "tcp", State: "open", ServiceHint: "ssh", Banner: "OpenSSH 7.9"},
	}
	attributes.Summary.OpenCount = 2

	attrJSON, _ := json.Marshal(attributes)

	result := &orcModel.StageResult{
		TaskID:      "task-1001",
		AgentID:     "agent-01",
		TargetValue: "192.168.1.100",
		TargetType:  "ip",
		ResultType:  "fast_port_scan",
		Attributes:  string(attrJSON),
		ProducedAt:  time.Now(),
	}

	// 推送数据
	err = queue.Push(ctx, result)
	assert.NoError(t, err)

	// 等待处理
	time.Sleep(2 * time.Second)

	// 验证 Host
	var host assetModel.AssetHost
	err = db.Where("ip = ?", "192.168.1.100").First(&host).Error
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.100", host.IP)

	// 验证 Services
	var services []assetModel.AssetService
	err = db.Where("host_id = ?", host.ID).Find(&services).Error
	assert.NoError(t, err)
	assert.Len(t, services, 2)

	// 验证具体服务
	serviceMap := make(map[int]assetModel.AssetService)
	for _, s := range services {
		serviceMap[s.Port] = s
	}

	assert.Contains(t, serviceMap, 80)
	assert.Equal(t, "tcp", serviceMap[80].Proto)
	assert.Equal(t, "http", serviceMap[80].Name) // Mapper should map ServiceHint to Name

	assert.Contains(t, serviceMap, 22)
	assert.Equal(t, "tcp", serviceMap[22].Proto)
	assert.Equal(t, "ssh", serviceMap[22].Name)
}

func TestETLIntegration_PortScan_Upsert(t *testing.T) {
	db, processor, queue, err := SetupETLTestEnv()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	processor.Start(ctx)
	defer processor.Stop()

	// 1. 初始扫描
	attrJSON1, _ := json.Marshal(etl.PortScanAttributes{
		Ports: []struct {
			IP          string `json:"ip"`
			Port        int    `json:"port"`
			Proto       string `json:"proto"`
			State       string `json:"state"`
			ServiceHint string `json:"service_hint"`
			Banner      string `json:"banner"`
		}{
			{IP: "10.0.0.1", Port: 80, Proto: "tcp", State: "open", ServiceHint: "http"},
		},
	})
	queue.Push(ctx, &orcModel.StageResult{
		TaskID:      "task-1",
		TargetValue: "10.0.0.1",
		TargetType:  "ip",
		ResultType:  "fast_port_scan",
		Attributes:  string(attrJSON1),
	})
	time.Sleep(1 * time.Second)

	// 2. 记录第一次的时间
	var host1 assetModel.AssetHost
	db.Where("ip = ?", "10.0.0.1").First(&host1)
	firstSeen := host1.LastSeenAt

	// 3. 第二次扫描 (更新)
	attrJSON2, _ := json.Marshal(etl.PortScanAttributes{
		Ports: []struct {
			IP          string `json:"ip"`
			Port        int    `json:"port"`
			Proto       string `json:"proto"`
			State       string `json:"state"`
			ServiceHint string `json:"service_hint"`
			Banner      string `json:"banner"`
		}{
			{IP: "10.0.0.1", Port: 80, Proto: "tcp", State: "open", ServiceHint: "http-alt"}, // Name 变更
		},
	})
	queue.Push(ctx, &orcModel.StageResult{
		TaskID:      "task-2",
		TargetValue: "10.0.0.1",
		TargetType:  "ip",
		ResultType:  "fast_port_scan",
		Attributes:  string(attrJSON2),
	})
	time.Sleep(1 * time.Second)

	// 4. 验证
	var host2 assetModel.AssetHost
	db.Where("ip = ?", "10.0.0.1").First(&host2)

	// Host ID 应该不变
	assert.Equal(t, host1.ID, host2.ID)
	// LastSeenAt 应该更新
	assert.True(t, host2.LastSeenAt.After(*firstSeen))

	var service assetModel.AssetService
	db.Where("host_id = ? AND port = ?", host2.ID, 80).First(&service)
	assert.Equal(t, "http-alt", service.Name)
}
