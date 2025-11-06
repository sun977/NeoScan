/**
 * 测试: Agent监控接口(`/agent/:id/metrics` 与 `/agent/metrics`)路由功能
 * 作者: Sun977 (由AI助手补充测试)
 * 日期: 2025-11-06
 * 说明:
 *  - 本测试文件仅聚焦于 agent 监控相关两个只读接口：
 *    1) GET /api/v1/agent/:id/metrics         -> 获取指定Agent的性能快照（来自Master端数据库）
 *    2) GET /api/v1/agent/metrics             -> 分页获取所有Agent的性能快照列表（支持状态与关键词过滤）
 *  - 遵循项目分层：Handler → Service → Repository → DB。测试通过 Mock Service 注入到 Handler，避免依赖数据库与中间件。
 *  - 统一使用系统 APIResponse 响应结构进行断言，同时覆盖成功、未找到、内部错误等场景。
 */
package test

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"

    agentHandler "neomaster/internal/handler/agent"
    agentModel "neomaster/internal/model/agent"
    "neomaster/internal/pkg/logger"
    cfgpkg "neomaster/internal/config"
)

// MockAgentMonitorService 模拟 Agent 监控服务
// 说明: 为了测试 Handler 在不同返回场景下的行为，我们记录入参并返回预设的数据或错误
type MockAgentMonitorService struct {
    // 单Agent查询返回
    MetricsByAgent map[string]*agentModel.AgentMetricsResponse
    ErrorByAgent   map[string]error

    // 列表查询返回
    ListResponse []*agentModel.AgentMetricsResponse
    ListTotal    int64
    ListError    error

    // 最近一次列表查询的入参记录，便于断言 Handler 是否正确传递过滤参数
    LastListParams struct {
        Page       int
        PageSize   int
        WorkStatus *agentModel.AgentWorkStatus
        ScanType   *agentModel.AgentScanType
        Keyword    *string
    }

    // Create/Update 行为控制与记录
    CreateErrorByAgent map[string]error
    UpdateErrorByAgent map[string]error
    LastCreated        struct {
        AgentID string
        Metrics agentModel.AgentMetrics
    }
    LastUpdated struct {
        AgentID string
        Metrics agentModel.AgentMetrics
    }
}

// 以下为接口实现（仅测试需要的两个方法有完整逻辑，其他方法作为占位返回默认值）
func (m *MockAgentMonitorService) ProcessHeartbeat(req *agentModel.HeartbeatRequest) (*agentModel.HeartbeatResponse, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *MockAgentMonitorService) GetAgentMetricsFromDB(agentID string) (*agentModel.AgentMetricsResponse, error) {
	if m.ErrorByAgent != nil {
		if err, ok := m.ErrorByAgent[agentID]; ok {
			return nil, err
		}
	}
	if m.MetricsByAgent != nil {
		if resp, ok := m.MetricsByAgent[agentID]; ok {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *MockAgentMonitorService) GetAgentListAllMetricsFromDB(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetricsResponse, int64, error) {
	m.LastListParams.Page = page
	m.LastListParams.PageSize = pageSize
	m.LastListParams.WorkStatus = workStatus
	m.LastListParams.ScanType = scanType
	m.LastListParams.Keyword = keyword

	if m.ListError != nil {
		return nil, 0, m.ListError
	}
	if m.ListResponse == nil {
		return []*agentModel.AgentMetricsResponse{}, 0, nil
	}
	return m.ListResponse, m.ListTotal, nil
}

func (m *MockAgentMonitorService) PullAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *MockAgentMonitorService) PullAgentListAllMetrics() ([]*agentModel.AgentMetricsResponse, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *MockAgentMonitorService) CreateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
    // 记录入参，便于测试断言
    if metrics != nil {
        m.LastCreated.AgentID = agentID
        m.LastCreated.Metrics = *metrics
    }
    if m.CreateErrorByAgent != nil {
        if err, ok := m.CreateErrorByAgent[agentID]; ok {
            return err
        }
    }
    return nil
}

func (m *MockAgentMonitorService) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
    // 记录入参，便于测试断言
    if metrics != nil {
        m.LastUpdated.AgentID = agentID
        m.LastUpdated.Metrics = *metrics
    }
    if m.UpdateErrorByAgent != nil {
        if err, ok := m.UpdateErrorByAgent[agentID]; ok {
            return err
        }
    }
    return nil
}

// newTestEngineWithHandler 初始化仅包含待测路由的 Gin 引擎
// 说明: 为了避免认证等中间件影响单元测试，这里直接注册必要的 GET 路由并绑定到 Handler
func newTestEngineWithHandler(h *agentHandler.AgentHandler) *gin.Engine {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    v1 := r.Group("/api/v1")
    agentGroup := v1.Group("/agent")

    // 注册两个待测路由（与生产环境路径保持一致）
    agentGroup.GET("/:id/metrics", h.GetAgentMetrics)    // 标准路径
    agentGroup.GET("/metrics", h.GetAgentListAllMetrics) // 列表查询
    // 新增：创建/更新指标接口（POST/PUT）
    agentGroup.POST("/:id/metrics", h.CreateAgentMetrics)
    agentGroup.PUT("/:id/metrics", h.UpdateAgentMetrics)

    return r
}

// APIResponseMetrics 用于反序列化 GetAgentMetrics 响应
type APIResponseMetrics struct {
	Code    int                             `json:"code"`
	Status  string                          `json:"status"`
	Message string                          `json:"message"`
	Error   string                          `json:"error"`
	Data    agentModel.AgentMetricsResponse `json:"data"`
}

// APIResponseMetricsList 用于反序列化 GetAgentListAllMetrics 响应
type APIResponseMetricsList struct {
    Code    int                                 `json:"code"`
    Status  string                              `json:"status"`
    Message string                              `json:"message"`
    Error   string                              `json:"error"`
    Data    agentModel.AgentMetricsListResponse `json:"data"`
}

// APIResponseGeneric 用于通用响应断言（Data 为任意结构）
type APIResponseGeneric struct {
    Code    int         `json:"code"`
    Status  string      `json:"status"`
    Message string      `json:"message"`
    Error   string      `json:"error"`
    Data    interface{} `json:"data"`
}

// setupTestLoggerBuffer 初始化logger到JSON+stdout，并将输出重定向到内存缓冲区以便断言
func setupTestLoggerBuffer() *bytes.Buffer {
    buf := &bytes.Buffer{}
    // 初始化日志到debug级别，json格式，stdout输出
    _, _ = logger.InitLogger(&cfgpkg.LogConfig{Level: "debug", Format: "json", Output: "stdout", Caller: false})
    // 重定向输出到缓冲区，避免污染控制台
    if logger.LoggerInstance != nil {
        logger.LoggerInstance.GetLogger().SetOutput(buf)
    }
    return buf
}

// Test_GetAgentMetrics_Success 测试: 获取指定Agent的性能快照成功场景
// 变更原因解释: 验证 Handler 层正确调用 Service 并返回统一响应结构（200 + 数据体）
func Test_GetAgentMetrics_Success(t *testing.T) {
	mock := &MockAgentMonitorService{
		MetricsByAgent: map[string]*agentModel.AgentMetricsResponse{
			"agent-123": {
				AgentID:           "agent-123",
				CPUUsage:          0.35,
				MemoryUsage:       0.60,
				DiskUsage:         0.40,
				NetworkBytesSent:  1024,
				NetworkBytesRecv:  2048,
				ActiveConnections: 10,
				RunningTasks:      2,
				CompletedTasks:    100,
				FailedTasks:       3,
				WorkStatus:        agentModel.AgentWorkStatus("running"),
				ScanType:          "port_scan",
				PluginStatus:      map[string]interface{}{"pluginA": "ok"},
			},
		},
	}

	h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
	r := newTestEngineWithHandler(h)

	req := httptest.NewRequest("GET", "/api/v1/agent/agent-123/metrics", nil)
	req.Header.Set("User-Agent", "unit-test")
	req.Header.Set("X-Request-ID", "req-001")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP状态码应该为200")

	var resp APIResponseMetrics
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "响应JSON应该可反序列化")
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "Agent metrics retrieved successfully", resp.Message)
	assert.Equal(t, "agent-123", resp.Data.AgentID)
	assert.Equal(t, agentModel.AgentWorkStatus("running"), resp.Data.WorkStatus)
	assert.Equal(t, "port_scan", resp.Data.ScanType)
}

// Test_GetAgentMetrics_NotFound 测试: 获取指定Agent性能快照未找到场景（404）
// 变更原因解释: 验证 Handler 的 getErrorStatusCode 对 "not found" 文本映射为 404
func Test_GetAgentMetrics_NotFound(t *testing.T) {
	mock := &MockAgentMonitorService{
		ErrorByAgent: map[string]error{
			"agent-404": fmt.Errorf("not found"),
		},
	}

	h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
	r := newTestEngineWithHandler(h)

	req := httptest.NewRequest("GET", "/api/v1/agent/agent-404/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "HTTP状态码应该为404")

	var resp APIResponseMetrics
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "failed", resp.Status)
}

// Test_GetAgentListAllMetrics_Default 测试: 获取所有Agent性能列表（默认分页）
// 变更原因解释: 验证 Handler 层分页响应结构与 Service 返回一致，并计算总页数
func Test_GetAgentListAllMetrics_Default(t *testing.T) {
	mock := &MockAgentMonitorService{
		ListResponse: []*agentModel.AgentMetricsResponse{
			{AgentID: "agent-1", CPUUsage: 0.1, WorkStatus: agentModel.AgentWorkStatus("running"), ScanType: "port_scan"},
			{AgentID: "agent-2", CPUUsage: 0.2, WorkStatus: agentModel.AgentWorkStatus("idle"), ScanType: "web_scan"},
		},
		ListTotal: 2,
	}

	h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
	r := newTestEngineWithHandler(h)

	req := httptest.NewRequest("GET", "/api/v1/agent/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIResponseMetricsList
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	// 响应数据断言
	assert.NotNil(t, resp.Data.Pagination)
	assert.Equal(t, int64(2), resp.Data.Pagination.Total)
	assert.Equal(t, 2, len(resp.Data.Metrics))
}

// Test_GetAgentListAllMetrics_FiltersAndPagination 测试: 分页+过滤参数正确传递
// 变更原因解释: 验证 Handler 是否正确解析 work_status、scan_type、keyword 并传递给 Service
func Test_GetAgentListAllMetrics_FiltersAndPagination(t *testing.T) {
	mock := &MockAgentMonitorService{
		ListResponse: []*agentModel.AgentMetricsResponse{
			{AgentID: "agent-1"},
			{AgentID: "agent-2"},
		},
		ListTotal: 20,
	}

	h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
	r := newTestEngineWithHandler(h)

	// page=2, page_size=5, work_status=running, scan_type=port_scan, keyword=agent-1
	req := httptest.NewRequest("GET", "/api/v1/agent/metrics?page=2&page_size=5&work_status=running&scan_type=port_scan&keyword=agent-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 断言 Service 收到的过滤参数
	assert.Equal(t, 2, mock.LastListParams.Page)
	assert.Equal(t, 5, mock.LastListParams.PageSize)
	if assert.NotNil(t, mock.LastListParams.WorkStatus) {
		assert.Equal(t, agentModel.AgentWorkStatus("running"), *mock.LastListParams.WorkStatus)
	}
	if assert.NotNil(t, mock.LastListParams.ScanType) {
		assert.Equal(t, agentModel.AgentScanType("port_scan"), *mock.LastListParams.ScanType)
	}
	if assert.NotNil(t, mock.LastListParams.Keyword) {
		assert.Equal(t, "agent-1", *mock.LastListParams.Keyword)
	}

	// 响应结构断言
	var resp APIResponseMetricsList
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.Data.Pagination)
	assert.Equal(t, 2, resp.Data.Pagination.Page)
	assert.Equal(t, 5, resp.Data.Pagination.PageSize)
	assert.Equal(t, int64(20), resp.Data.Pagination.Total)
	// 总页数 = ceil(20 / 5) = 4
	assert.Equal(t, 4, resp.Data.Pagination.TotalPages)
}

// Test_GetAgentListAllMetrics_InternalError 测试: 列表接口内部错误映射为500
// 变更原因解释: 验证 Handler 的 getErrorStatusCode 默认分支返回500，并包含错误信息
func Test_GetAgentListAllMetrics_InternalError(t *testing.T) {
    mock := &MockAgentMonitorService{ListError: fmt.Errorf("db failure")}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    req := httptest.NewRequest("GET", "/api/v1/agent/metrics", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusInternalServerError, w.Code)

    var resp APIResponseMetricsList
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "failed", resp.Status)
    assert.Equal(t, "Failed to get all agents metrics", resp.Message)
}

// ------------------------------
// CreateAgentMetrics（POST）测试用例
// ------------------------------

// Test_CreateAgentMetrics_Success 测试: 创建指标成功（201），并验证AgentID覆盖与响应结构
// 变更原因解释: 验证 Handler 将路径参数覆盖请求体中的 agent_id，且服务层被正确调用
func Test_CreateAgentMetrics_Success(t *testing.T) {
    mock := &MockAgentMonitorService{}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{
        "agent_id":"fake-body-id",
        "cpu_usage": 55.5,
        "memory_usage": 33.3,
        "disk_usage": 12.1,
        "network_bytes_sent": 1024,
        "network_bytes_recv": 2048,
        "active_connections": 3,
        "running_tasks": 1,
        "completed_tasks": 10,
        "failed_tasks": 0,
        "work_status": "working",
        "scan_type": "port_scan"
    }`
    req := httptest.NewRequest("POST", "/api/v1/agent/agent-201/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)
    var resp APIResponseGeneric
    err := json.Unmarshal(w.Body.Bytes(), &resp)
    assert.NoError(t, err)
    assert.Equal(t, "success", resp.Status)
    assert.Equal(t, "Agent metrics created successfully", resp.Message)

    // 响应Data为对象，检查包含 agent_id 字段
    dataMap, ok := resp.Data.(map[string]interface{})
    if !ok {
        // 某些JSON库在反序列化时可能将数字时间戳转换为结构，这里使用再次编码再解码的方式保证为map
        var tmp map[string]interface{}
        _ = json.Unmarshal(w.Body.Bytes(), &tmp)
        dataMap, _ = tmp["data"].(map[string]interface{})
    }
    if assert.NotNil(t, dataMap) {
        assert.Equal(t, "agent-201", fmt.Sprintf("%v", dataMap["agent_id"]))
    }

    // 断言服务层收到的AgentID已被覆盖为路径参数
    assert.Equal(t, "agent-201", mock.LastCreated.AgentID)
    assert.Equal(t, "agent-201", mock.LastCreated.Metrics.AgentID)
}

// Test_CreateAgentMetrics_ScanTypeNotStrict 测试: scan_type 不进行强校验，传入未知值亦可成功
// 变更原因解释: 根据用户需求“不对ScanType进行强校验”，确保Handler不拒绝未知扫描类型
func Test_CreateAgentMetrics_ScanTypeNotStrict(t *testing.T) {
    mock := &MockAgentMonitorService{}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 5, "memory_usage": 5, "disk_usage": 5, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0, "scan_type":"unknown-type"}`
    req := httptest.NewRequest("POST", "/api/v1/agent/agent-scan/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusCreated, w.Code)
    // 服务层应收到原样 scan_type（模型里该字段类型为 string）
    assert.Equal(t, "unknown-type", mock.LastCreated.Metrics.ScanType)
}

// Test_CreateAgentMetrics_InvalidParams 测试: 非法参数触发400（例如CPU/Memory/Disk越界）
// 变更原因解释: 验证 Handler 的防御性校验逻辑生效，不进入Service层
func Test_CreateAgentMetrics_InvalidParams(t *testing.T) {
    mock := &MockAgentMonitorService{}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": -5}`
    req := httptest.NewRequest("POST", "/api/v1/agent/agent-400/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusBadRequest, w.Code)

    var resp APIResponseGeneric
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "failed", resp.Status)
    assert.Contains(t, resp.Message, "Invalid CPU usage")
    // 未调用Service，LastCreated 应为空
    assert.Empty(t, mock.LastCreated.AgentID)
}

// Test_CreateAgentMetrics_NotFound 测试: 服务返回 not found → 404 映射
// 变更原因解释: 验证 getErrorStatusCode 对包含 not found 文本的错误映射为404
func Test_CreateAgentMetrics_NotFound(t *testing.T) {
    mock := &MockAgentMonitorService{CreateErrorByAgent: map[string]error{"agent-404": fmt.Errorf("not found")}}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 1, "memory_usage": 1, "disk_usage": 1, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    req := httptest.NewRequest("POST", "/api/v1/agent/agent-404/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusNotFound, w.Code)
    var resp APIResponseGeneric
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "failed", resp.Status)
}

// Test_CreateAgentMetrics_InternalError 测试: 服务失败 → 500 映射
// 变更原因解释: 验证 getErrorStatusCode 默认分支返回500
func Test_CreateAgentMetrics_InternalError(t *testing.T) {
    mock := &MockAgentMonitorService{CreateErrorByAgent: map[string]error{"agent-500": fmt.Errorf("db failure")}}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 1, "memory_usage": 1, "disk_usage": 1, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    req := httptest.NewRequest("POST", "/api/v1/agent/agent-500/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ------------------------------
// UpdateAgentMetrics（PUT）测试用例
// ------------------------------

// Test_UpdateAgentMetrics_Success 测试: 更新指标成功（200）
// 变更原因解释: 验证 Handler 更新流程与返回结构
func Test_UpdateAgentMetrics_Success(t *testing.T) {
    mock := &MockAgentMonitorService{}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 10, "memory_usage": 20, "disk_usage": 30, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0, "work_status":"idle", "scan_type":"web_scan"}`
    req := httptest.NewRequest("PUT", "/api/v1/agent/agent-200/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    var resp APIResponseGeneric
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "success", resp.Status)
    assert.Equal(t, "Agent metrics updated successfully", resp.Message)
    // 断言服务层收到的参数正确
    assert.Equal(t, "agent-200", mock.LastUpdated.AgentID)
    assert.Equal(t, "agent-200", mock.LastUpdated.Metrics.AgentID)
}

// Test_UpdateAgentMetrics_ScanTypeNotStrict 测试: 更新也允许未知 scan_type（不强校验）
// 变更原因解释: 与创建接口一致，scan_type 作为可选自定义类型
func Test_UpdateAgentMetrics_ScanTypeNotStrict(t *testing.T) {
    mock := &MockAgentMonitorService{}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 10, "memory_usage": 20, "disk_usage": 30, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0, "work_status":"idle", "scan_type":"freeform-type"}`
    req := httptest.NewRequest("PUT", "/api/v1/agent/agent-scan-update/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    // 模型字段为 string，断言直接等于原值
    assert.Equal(t, "freeform-type", mock.LastUpdated.Metrics.ScanType)
}

// Test_UpdateAgentMetrics_InvalidParams 测试: 非法工作状态触发400
// 变更原因解释: 验证 WorkStatus 非法值被拒绝
func Test_UpdateAgentMetrics_InvalidParams(t *testing.T) {
    mock := &MockAgentMonitorService{}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 10, "memory_usage": 20, "disk_usage": 30, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0, "work_status":"unknown"}`
    req := httptest.NewRequest("PUT", "/api/v1/agent/agent-400/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusBadRequest, w.Code)
    var resp APIResponseGeneric
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "failed", resp.Status)
    assert.Contains(t, resp.Message, "Invalid work status")
}

// Test_UpdateAgentMetrics_NotFound 测试: 服务返回 not found → 404
func Test_UpdateAgentMetrics_NotFound(t *testing.T) {
    mock := &MockAgentMonitorService{UpdateErrorByAgent: map[string]error{"agent-404": fmt.Errorf("not found")}}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 1, "memory_usage": 1, "disk_usage": 1, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    req := httptest.NewRequest("PUT", "/api/v1/agent/agent-404/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusNotFound, w.Code)
    var resp APIResponseGeneric
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "failed", resp.Status)
    assert.Equal(t, "Agent metrics not found", resp.Message)
}

// Test_UpdateAgentMetrics_InternalError 测试: 服务失败 → 500
func Test_UpdateAgentMetrics_InternalError(t *testing.T) {
    mock := &MockAgentMonitorService{UpdateErrorByAgent: map[string]error{"agent-500": fmt.Errorf("db failure")}}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 1, "memory_usage": 1, "disk_usage": 1, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    req := httptest.NewRequest("PUT", "/api/v1/agent/agent-500/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Test_UpdateAgentMetrics_ErrorMapping400 测试: 服务返回包含 "invalid" 的错误被映射到400
// 变更原因解释: 覆盖 getErrorStatusCode 对400场景的处理（Create/Update接口）
func Test_UpdateAgentMetrics_ErrorMapping400(t *testing.T) {
    mock := &MockAgentMonitorService{UpdateErrorByAgent: map[string]error{"agent-400": fmt.Errorf("invalid param: memory_usage")}}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 1, "memory_usage": 1, "disk_usage": 1, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    req := httptest.NewRequest("PUT", "/api/v1/agent/agent-400/metrics", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    assert.Equal(t, http.StatusBadRequest, w.Code)
    var resp APIResponseGeneric
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    // Update接口对404有特殊message，这里是400，保持通用失败信息
    assert.Equal(t, "Failed to update agent metrics", resp.Message)
}

// ------------------------------
// 列表接口边界与参数解析测试
// ------------------------------

// Test_GetAgentListAllMetrics_PageBoundary 测试: page/page_size 边界（0或负数回退到默认）
func Test_GetAgentListAllMetrics_PageBoundary(t *testing.T) {
    mock := &MockAgentMonitorService{
        ListResponse: []*agentModel.AgentMetricsResponse{{AgentID: "agent-1"}},
        ListTotal:    1,
    }
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    req := httptest.NewRequest("GET", "/api/v1/agent/metrics?page=0&page_size=0", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    var resp APIResponseMetricsList
    _ = json.Unmarshal(w.Body.Bytes(), &resp)
    // 应回退到默认值 page=1, page_size=10
    assert.Equal(t, 1, resp.Data.Pagination.Page)
    assert.Equal(t, 10, resp.Data.Pagination.PageSize)
    // 服务入参也应为默认值
    assert.Equal(t, 1, mock.LastListParams.Page)
    assert.Equal(t, 10, mock.LastListParams.PageSize)
}

// Test_GetAgentListAllMetrics_InvalidEnums 测试: 非法 work_status/scan_type 作为字符串透传
func Test_GetAgentListAllMetrics_InvalidEnums(t *testing.T) {
    mock := &MockAgentMonitorService{
        ListResponse: []*agentModel.AgentMetricsResponse{{AgentID: "agent-x"}},
        ListTotal:    1,
    }
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    req := httptest.NewRequest("GET", "/api/v1/agent/metrics?work_status=unknown&scan_type=special_scan", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    // 断言服务接收到原始字符串
    if assert.NotNil(t, mock.LastListParams.WorkStatus) {
        assert.Equal(t, agentModel.AgentWorkStatus("unknown"), *mock.LastListParams.WorkStatus)
    }
    if assert.NotNil(t, mock.LastListParams.ScanType) {
        assert.Equal(t, agentModel.AgentScanType("special_scan"), *mock.LastListParams.ScanType)
    }
}

// Test_GetAgentListAllMetrics_KeywordSpecialChars 测试: keyword 空与特殊字符
func Test_GetAgentListAllMetrics_KeywordSpecialChars(t *testing.T) {
    mock := &MockAgentMonitorService{
        ListResponse: []*agentModel.AgentMetricsResponse{{AgentID: "agent-special"}},
        ListTotal:    1,
    }
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    // 1) keyword 为空：不应传递到Service（nil）
    req1 := httptest.NewRequest("GET", "/api/v1/agent/metrics", nil)
    w1 := httptest.NewRecorder()
    r.ServeHTTP(w1, req1)
    assert.Nil(t, mock.LastListParams.Keyword)

    // 2) keyword 包含特殊字符：原样透传
    // 注意：Gin 的 c.Query 会对URL进行解码，因此这里传入的 %20 会被解析成空格
    // 期望在服务层收到的是解码后的字符串 "agent-1 special!@#"
    kw := "agent-1%20special!@#"
    req2 := httptest.NewRequest("GET", "/api/v1/agent/metrics?keyword="+kw, nil)
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if assert.NotNil(t, mock.LastListParams.Keyword) {
        assert.Equal(t, "agent-1 special!@#", *mock.LastListParams.Keyword)
    }
}

// ------------------------------
// 日志字段断言（path、operation、option、func_name）
// ------------------------------

// Test_Logging_CreateAgentMetrics_SuccessFields 测试: 成功日志包含关键字段
func Test_Logging_CreateAgentMetrics_SuccessFields(t *testing.T) {
    buf := setupTestLoggerBuffer()
    mock := &MockAgentMonitorService{}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 1, "memory_usage": 1, "disk_usage": 1, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    path := "/api/v1/agent/agent-log/metrics"
    req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    // 简单等待日志写入
    time.Sleep(10 * time.Millisecond)

    // 日志应包含关键字段
    logs := buf.String()
    assert.Contains(t, logs, "\"operation\":\"create_agent_metrics\"")
    assert.Contains(t, logs, "\"option\":\"success\"")
    assert.Contains(t, logs, "\"func_name\":\"handler.agent.CreateAgentMetrics\"")
    assert.Contains(t, logs, "\"path\":\""+path+"\"")
    assert.Contains(t, logs, "\"method\":\"POST\"")
}

// Test_Logging_CreateAgentMetrics_ErrorFields 测试: 业务错误日志包含关键字段（BusinessLog）
func Test_Logging_CreateAgentMetrics_ErrorFields(t *testing.T) {
    buf := setupTestLoggerBuffer()
    mock := &MockAgentMonitorService{CreateErrorByAgent: map[string]error{"agent-log-404": fmt.Errorf("not found")}}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 1, "memory_usage": 1, "disk_usage": 1, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    path := "/api/v1/agent/agent-log-404/metrics"
    req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    time.Sleep(10 * time.Millisecond)

    logs := buf.String()
    // 业务错误日志使用 type=business
    assert.Contains(t, logs, "\"type\":\"business\"")
    assert.Contains(t, logs, "System error occurred: not found")
    assert.Contains(t, logs, "\"operation\":\"create_agent_metrics\"")
    assert.Contains(t, logs, "\"option\":\"agentMonitorService.CreateAgentMetrics\"")
    assert.Contains(t, logs, "\"func_name\":\"handler.agent.CreateAgentMetrics\"")
    assert.Contains(t, logs, "\"path\":\""+path+"\"")
    assert.Contains(t, logs, "\"method\":\"POST\"")
}

// Test_Logging_UpdateAgentMetrics_SuccessFields 测试: 更新成功日志包含关键字段
func Test_Logging_UpdateAgentMetrics_SuccessFields(t *testing.T) {
    buf := setupTestLoggerBuffer()
    mock := &MockAgentMonitorService{}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 2, "memory_usage": 3, "disk_usage": 4, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    path := "/api/v1/agent/agent-update-log/metrics"
    req := httptest.NewRequest("PUT", path, bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    time.Sleep(10 * time.Millisecond)

    logs := buf.String()
    assert.Contains(t, logs, "\"operation\":\"update_agent_metrics\"")
    assert.Contains(t, logs, "\"option\":\"success\"")
    assert.Contains(t, logs, "\"func_name\":\"handler.agent.UpdateAgentMetrics\"")
    assert.Contains(t, logs, "\"path\":\""+path+"\"")
    assert.Contains(t, logs, "\"method\":\"PUT\"")
}

// Test_Logging_UpdateAgentMetrics_ErrorFields 测试: 更新失败日志包含关键字段（BusinessLog）
func Test_Logging_UpdateAgentMetrics_ErrorFields(t *testing.T) {
    buf := setupTestLoggerBuffer()
    mock := &MockAgentMonitorService{UpdateErrorByAgent: map[string]error{"agent-update-500": fmt.Errorf("db failure")}}
    h := agentHandler.NewAgentHandler(nil, mock, nil, nil)
    r := newTestEngineWithHandler(h)

    body := `{"cpu_usage": 2, "memory_usage": 3, "disk_usage": 4, "network_bytes_sent": 0, "network_bytes_recv": 0, "running_tasks":0, "completed_tasks":0, "failed_tasks":0}`
    path := "/api/v1/agent/agent-update-500/metrics"
    req := httptest.NewRequest("PUT", path, bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    time.Sleep(10 * time.Millisecond)

    logs := buf.String()
    assert.Contains(t, logs, "\"type\":\"business\"")
    assert.Contains(t, logs, "db failure")
    assert.Contains(t, logs, "\"operation\":\"update_agent_metrics\"")
    assert.Contains(t, logs, "\"option\":\"agentMonitorService.UpdateAgentMetrics\"")
    assert.Contains(t, logs, "\"func_name\":\"handler.agent.UpdateAgentMetrics\"")
    assert.Contains(t, logs, "\"path\":\""+path+"\"")
    assert.Contains(t, logs, "\"method\":\"PUT\"")
}
