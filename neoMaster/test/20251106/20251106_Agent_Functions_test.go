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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	agentHandler "neomaster/internal/handler/agent"
	agentModel "neomaster/internal/model/agent"
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
	return fmt.Errorf("not implemented in mock")
}

func (m *MockAgentMonitorService) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	return fmt.Errorf("not implemented in mock")
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
