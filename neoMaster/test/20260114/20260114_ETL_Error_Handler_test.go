package test_20260114

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	assetHandler "neomaster/internal/handler/asset"
	assetModel "neomaster/internal/model/asset"
	orcModel "neomaster/internal/model/orchestrator"
	assetRepo "neomaster/internal/repo/mysql/asset"
	assetService "neomaster/internal/service/asset"
	"neomaster/internal/service/asset/etl"
)

// MockResultProcessor 模拟 Processor
type MockResultProcessor struct {
	ReplayedCount int
}

func (m *MockResultProcessor) Start(ctx context.Context) {}
func (m *MockResultProcessor) Stop()                     {}
func (m *MockResultProcessor) ReplayErrors(ctx context.Context) (int, error) {
	return m.ReplayedCount, nil
}

// setupETLErrorEnv 构建 ETL 错误管理测试环境
func setupETLErrorEnv(t *testing.T) (*gin.Engine, *gorm.DB, *MockResultProcessor) {
	// 1. 初始化数据库
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移
	db.AutoMigrate(&assetModel.AssetETLError{})

	// 2. 初始化组件
	repo := assetRepo.NewETLErrorRepository(db)
	mockProcessor := &MockResultProcessor{ReplayedCount: 5}
	service := assetService.NewAssetETLErrorService(repo, mockProcessor)
	handler := assetHandler.NewETLErrorHandler(service)

	// 3. 构建路由
	r := gin.Default()
	v1 := r.Group("/api/v1")
	etlGroup := v1.Group("/asset/etl/errors")
	{
		etlGroup.GET("", handler.ListErrors)
		etlGroup.GET("/:id", handler.GetError)
		etlGroup.POST("/replay", handler.TriggerReplay)
	}

	return r, db, mockProcessor
}

func TestETLError_ListAndGet(t *testing.T) {
	r, db, _ := setupETLErrorEnv(t)

	// 1. 准备测试数据
	// 清理旧数据
	db.Exec("DELETE FROM asset_etl_errors WHERE task_id LIKE 'test-task-%'")

	// 插入数据
	errRecord := &assetModel.AssetETLError{
		ProjectID:  1,
		TaskID:     "test-task-001",
		ResultType: "port_scan",
		RawData:    `{"ip":"1.1.1.1"}`,
		ErrorMsg:   "connection timeout",
		ErrorStage: "merger",
		Status:     "new",
	}
	db.Create(errRecord)

	// 2. 测试列表查询
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/asset/etl/errors?task_id=test-task-001", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			List  []assetModel.AssetETLError `json:"list"`
			Total int64                      `json:"total"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)

	assert.Equal(t, 200, listResp.Code)
	assert.Equal(t, int64(1), listResp.Data.Total)
	assert.Equal(t, "test-task-001", listResp.Data.List[0].TaskID)

	// 3. 测试详情查询
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/asset/etl/errors/%d", errRecord.ID), nil)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var getResp struct {
		Code int                      `json:"code"`
		Data assetModel.AssetETLError `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &getResp)

	assert.Equal(t, 200, getResp.Code)
	assert.Equal(t, errRecord.ID, getResp.Data.ID)
	assert.Equal(t, "port_scan", getResp.Data.ResultType)

	// 清理
	db.Delete(errRecord)
}

func TestETLError_Replay(t *testing.T) {
	r, _, mockProcessor := setupETLErrorEnv(t)

	// 测试触发重放
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/asset/etl/errors/replay", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ReplayedCount int `json:"replayed_count"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, 200, resp.Code)
	// 验证是否返回了 Mock Processor 的预设值
	assert.Equal(t, mockProcessor.ReplayedCount, resp.Data.ReplayedCount)
}

// 真实场景模拟：使用真实的 Processor 进行重放逻辑验证 (集成测试)
// 注意：这需要真实的 DB 环境和 Processor 逻辑
func TestETLError_Integration_ReplayLogic(t *testing.T) {
	// 1. 初始化
	dsn := "root:ROOT@tcp(127.0.0.1:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 清理
	db.Exec("DELETE FROM asset_etl_errors WHERE task_id = 'test-replay-001'")

	// 2. 插入一条 "new" 状态的错误记录
	// 构造一个合法的 StageResult JSON
	stageResult := orcModel.StageResult{
		TaskID:      "test-replay-001",
		ResultType:  "test_scan",
		TargetValue: "1.1.1.1",
		Attributes:  "{}",
	}
	rawJSON, _ := json.Marshal(stageResult)

	errRecord := &assetModel.AssetETLError{
		ProjectID:  1,
		TaskID:     "test-replay-001",
		ResultType: "test_scan",
		RawData:    string(rawJSON),
		Status:     "new",
	}
	db.Create(errRecord)

	// 3. 初始化真实的 Processor (使用 Mock Queue 以便验证是否被重新投递)
	repo := assetRepo.NewETLErrorRepository(db)

	// Mock Queue
	mockQueue := &MockQueue{}

	// 这里我们需要一个部分真实的 Processor，只测试 ReplayErrors 方法
	// 由于 ReplayErrors 依赖 queue.Push，我们注入 mockQueue
	processor := etl.NewResultProcessor(mockQueue, nil, repo, 1)

	// 4. 执行重放
	count, err := processor.ReplayErrors(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 5. 验证 DB 状态变更
	var updatedRecord assetModel.AssetETLError
	db.First(&updatedRecord, errRecord.ID)
	assert.Equal(t, "retrying", updatedRecord.Status)

	// 6. 验证 Queue 是否收到了消息
	assert.Equal(t, 1, mockQueue.PushCount)

	// 清理
	db.Delete(errRecord)
}

// MockQueue for Integration Test
type MockQueue struct {
	PushCount int
}

func (q *MockQueue) Push(ctx context.Context, result *orcModel.StageResult) error {
	q.PushCount++
	return nil
}
func (q *MockQueue) Pop(ctx context.Context) (*orcModel.StageResult, error) { return nil, nil }
func (q *MockQueue) Len(ctx context.Context) (int64, error)                 { return 0, nil }
func (q *MockQueue) Close() error                                           { return nil }
func (q *MockQueue) Clear(ctx context.Context) error                        { return nil }
