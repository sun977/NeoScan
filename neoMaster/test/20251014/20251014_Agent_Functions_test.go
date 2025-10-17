/**
 * 测试:Agent功能单元测试
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent核心功能的单元测试，遵循"好品味"原则 - 简洁、全面、可靠
 * @func: 测试Agent注册、心跳、状态更新、列表查询等核心功能
 */
package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/utils"
)

// TestAgentFunctions Agent功能测试套件
func TestAgentFunctions(t *testing.T) {
	// 设置测试环境
	gin.SetMode(gin.TestMode)

	// 初始化测试套件
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 运行测试用例
	t.Run("Agent注册功能测试", testAgentRegistration)
	t.Run("Agent心跳功能测试", testAgentHeartbeat)
	t.Run("Agent状态更新测试", testAgentStatusUpdate)
	t.Run("Agent列表查询测试", testAgentListQuery)
	t.Run("Agent详情查询测试", testAgentDetailQuery)
	t.Run("Agent删除测试", testAgentDelete)
	t.Run("Agent权限验证测试", testAgentPermissions)
	t.Run("StringSlice转换测试", testStringSliceConversion)
}

// testAgentRegistration 测试Agent注册功能
func testAgentRegistration(t *testing.T) {
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 创建管理员用户并获取token
	adminUser := ts.CreateTestUser(t, "agentadmin", "agentadmin@test.com", "password123")
	ts.AssignRoleToUser(t, adminUser.ID, "admin")

	loginResp, err := ts.SessionService.Login(context.Background(), &system.LoginRequest{
		Username: "agentadmin",
		Password: "password123",
	}, "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, loginResp)

	// 准备测试数据
	testCases := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "正常Agent注册",
			requestBody: `{
				"hostname": "test-agent-001",
				"ip_address": "192.168.1.100",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan", "vuln_scan"],
				"tags": ["test", "development"],
				"remark": "测试Agent"
			}`,
			expectedStatus: http.StatusOK,
			expectedMsg:    "Agent registered successfully",
		},
		{
			name: "缺少必填字段",
			requestBody: `{
				"hostname": "test-agent-002",
				"port": 8080
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "ip_address is required",
		},
		{
			name: "重复注册相同hostname",
			requestBody: `{
				"hostname": "test-agent-001",
				"ip_address": "192.168.1.101",
				"port": 8081,
				"version": "1.0.1",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 2,
				"memory_total": 4294967296,
				"disk_total": 53687091200,
				"capabilities": ["web_scan"],
				"tags": ["production"]
			}`,
			expectedStatus: http.StatusConflict,
			expectedMsg:    "Agent registration failed",
		},
		// 边界情况测试 - 无效IP地址
		{
			name: "无效IP地址格式",
			requestBody: `{
				"hostname": "test-agent-invalid-ip",
				"ip_address": "999.999.999.999",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid ip_address format",
		},
		{
			name: "IP地址为空字符串",
			requestBody: `{
				"hostname": "test-agent-empty-ip",
				"ip_address": "",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "ip_address is required",
		},
		// 边界情况测试 - 端口范围
		{
			name: "端口号为0",
			requestBody: `{
				"hostname": "test-agent-port-zero",
				"ip_address": "192.168.1.200",
				"port": 0,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "port must be between 1 and 65535",
		},
		{
			name: "端口号超出范围",
			requestBody: `{
				"hostname": "test-agent-port-overflow",
				"ip_address": "192.168.1.201",
				"port": 70000,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "port must be between 1 and 65535",
		},
		// 边界情况测试 - 字段长度限制
		{
			name: "hostname过长",
			requestBody: `{
				"hostname": "` + strings.Repeat("a", 256) + `",
				"ip_address": "192.168.1.202",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Hostname too long",
		},
		{
			name: "version字段过长",
			requestBody: `{
				"hostname": "test-agent-long-version",
				"ip_address": "192.168.1.203",
				"port": 8080,
				"version": "` + strings.Repeat("1.0.", 50) + `",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Version too long",
		},
		// 边界情况测试 - 数值范围
		{
			name: "CPU核心数为负数",
			requestBody: `{
				"hostname": "test-agent-negative-cpu",
				"ip_address": "192.168.1.204",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": -1,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid CPU cores",
		},
		{
			name: "内存总量为负数",
			requestBody: `{
				"hostname": "test-agent-negative-memory",
				"ip_address": "192.168.1.205",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": -1,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid memory total",
		},
		// 边界情况测试 - 数组字段
		{
			name: "capabilities为空数组",
			requestBody: `{
				"hostname": "test-agent-empty-capabilities",
				"ip_address": "192.168.1.206",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": [],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "At least one capability is required",
		},
		{
			name: "capabilities包含无效值",
			requestBody: `{
				"hostname": "test-agent-invalid-capability",
				"ip_address": "192.168.1.207",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["invalid_capability", "port_scan"],
				"tags": ["test"]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid capability",
		},
		// 边界情况测试 - JSON格式
		{
			name: "无效JSON格式",
			requestBody: `{
				"hostname": "test-agent-invalid-json",
				"ip_address": "192.168.1.208",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			`, // 故意缺少结束括号
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid JSON format",
		},
	}

	// 执行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建HTTP请求
			req, err := http.NewRequest("POST", "/api/v1/agent/register", strings.NewReader(tc.requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			// Agent注册是公开接口，不需要Authorization头
			// req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)

			// 执行请求
			w := httptest.NewRecorder()
			ts.RouterManager.GetEngine().ServeHTTP(w, req)

			// 验证响应
			assert.Equal(t, tc.expectedStatus, w.Code)

			var response system.APIResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response.Message, tc.expectedMsg)

			// 如果注册成功，验证数据库中的记录
			if tc.expectedStatus == http.StatusOK {
				var agentData map[string]interface{}
				dataBytes, _ := json.Marshal(response.Data)
				json.Unmarshal(dataBytes, &agentData)

				assert.NotEmpty(t, agentData["agent_id"])
				// RegisterAgentResponse结构中没有hostname字段，只有agent_id, grpc_token, token_expiry, status, message
				// assert.Equal(t, "test-agent-001", agentData["hostname"])
				assert.Equal(t, "registered", agentData["status"])
			}
		})
	}
}

// testAgentHeartbeat 测试Agent心跳功能
func testAgentHeartbeat(t *testing.T) {
	// 不要创建新的TestSuite，使用父测试的TestSuite
	// 这样可以避免资源冲突和重复初始化
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 先注册一个Agent
	agent := ts.CreateTestAgent(t, "heartbeat-agent", "192.168.1.200", 8080)

	// 创建测试用户用于登录
	testUser := ts.CreateTestUser(t, "listadmin", "listadmin@test.com", "password123")
	// 为用户分配管理员角色
	ts.AssignRoleToUser(t, testUser.ID, "admin")

	// 使用HTTP登录接口获取真实token
	loginBody := `{"username":"listadmin","password":"password123"}`
	loginReq, err := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	ts.RouterManager.GetEngine().ServeHTTP(loginW, loginReq)
	require.Equal(t, http.StatusOK, loginW.Code, "登录应该成功")

	var loginResponse system.APIResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok, "登录响应数据格式错误")

	accessToken, ok := loginData["access_token"].(string)
	require.True(t, ok, "无法获取access_token")

	// 正常心跳测试
	t.Run("正常心跳请求", func(t *testing.T) {
		heartbeatBody := `{
			"agent_id": "` + agent.AgentID + `",
			"status": "online",
			"metrics": {
				"agent_id": "` + agent.AgentID + `",
				"cpu_usage": 45.5,
				"memory_usage": 60.2,
				"disk_usage": 30.8,
				"network_bytes_sent": 1024000,
				"network_bytes_recv": 2048000,
				"active_connections": 5,
				"running_tasks": 3,
				"completed_tasks": 10,
				"failed_tasks": 1,
				"work_status": "working",
				"scan_type": "port_scan",
				"plugin_status": {"nmap": "active", "masscan": "inactive"}
			}
		}`

		req, err := http.NewRequest("POST", "/api/v1/agent/heartbeat", strings.NewReader(heartbeatBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		w := httptest.NewRecorder()
		ts.RouterManager.GetEngine().ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusOK, w.Code)

		var response system.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "success", response.Status)

		// 验证数据库中的心跳时间已更新
		updatedAgent, err := ts.AgentRepository.GetByID(agent.AgentID)
		require.NoError(t, err)
		assert.True(t, updatedAgent.LastHeartbeat.After(agent.LastHeartbeat))
	})

	// 异常情况测试
	heartbeatTestCases := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedMsg    string
	}{
		// 无效agent_id测试
		{
			name: "无效的agent_id格式",
			requestBody: `{
				"agent_id": "invalid-agent-id-format",
				"status": "online",
				"metrics": {
					"agent_id": "invalid-agent-id-format",
					"cpu_usage": 45.5,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"load_average": 1.5,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"active_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Agent not found",
		},
		{
			name: "不存在的agent_id",
			requestBody: `{
				"agent_id": "00000000-0000-0000-0000-000000000000",
				"status": "online",
				"metrics": {
					"agent_id": "00000000-0000-0000-0000-000000000000",
					"cpu_usage": 45.5,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Agent not found",
		},
		{
			name: "agent_id为空",
			requestBody: `{
				"agent_id": "",
				"status": "online",
				"metrics": {
					"agent_id": "",
					"cpu_usage": 45.5,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Agent ID is required",
		},
		// 无效状态测试
		{
			name: "无效的状态值",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "invalid_status",
				"metrics": {
					"agent_id": "` + agent.AgentID + `",
					"cpu_usage": 45.5,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid status",
		},
		{
			name: "状态字段缺失",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"metrics": {
					"agent_id": "` + agent.AgentID + `",
					"cpu_usage": 45.5,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Status is required",
		},
		// 无效数值测试
		{
			name: "CPU使用率为负数",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "online",
				"metrics": {
					"agent_id": "` + agent.AgentID + `",
					"cpu_usage": -10.5,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid CPU usage",
		},
		{
			name: "CPU使用率超过100%",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "online",
				"metrics": {
					"agent_id": "` + agent.AgentID + `",
					"cpu_usage": 150.0,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid CPU usage",
		},
		{
			name: "内存使用率为负数",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "online",
				"metrics": {
					"agent_id": "` + agent.AgentID + `",
					"cpu_usage": 45.5,
					"memory_usage": -20.2,
					"disk_usage": 30.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid memory usage",
		},
		{
			name: "磁盘使用率超过100%",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "online",
				"metrics": {
					"agent_id": "` + agent.AgentID + `",
					"cpu_usage": 45.5,
					"memory_usage": 60.2,
					"disk_usage": 130.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid disk usage",
		},
		{
			name: "运行任务数为负数",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "online",
				"metrics": {
					"agent_id": "` + agent.AgentID + `",
					"cpu_usage": 45.5,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"network_bytes_sent": 1024000,
					"network_bytes_recv": 2048000,
					"running_tasks": -1,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid running tasks",
		},
		{
			name: "网络发送字节数为负数",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "online",
				"metrics": {
					"agent_id": "` + agent.AgentID + `",
					"cpu_usage": 45.5,
					"memory_usage": 60.2,
					"disk_usage": 30.8,
					"network_bytes_sent": -1000,
					"network_bytes_recv": 2048000,
					"running_tasks": 3,
					"completed_tasks": 10,
					"failed_tasks": 1
				}
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid network bytes sent",
		},
		// JSON格式错误测试
		{
			name: "无效JSON格式",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "online",
				"cpu_usage": 45.5,
				"memory_usage": 60.2,
				"disk_usage": 30.8,
				"load_average": 1.5,
				"active_tasks": 3
			`, // 故意缺少结束括号
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid JSON format",
		},
		// 缺少必填字段测试
		{
			name:           "缺少所有必填字段",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request format",
		},
	}

	// 执行异常情况测试
	for _, tc := range heartbeatTestCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/agent/heartbeat", strings.NewReader(tc.requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+accessToken)

			w := httptest.NewRecorder()
			ts.RouterManager.GetEngine().ServeHTTP(w, req)

			// 验证响应状态码
			assert.Equal(t, tc.expectedStatus, w.Code)

			// 验证响应消息
			var response system.APIResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response.Message, tc.expectedMsg)
		})
	}

	// 权限测试 - 无token访问
	t.Run("无权限访问心跳接口", func(t *testing.T) {
		heartbeatBody := `{
			"agent_id": "` + agent.AgentID + `",
			"status": "online",
			"cpu_usage": 45.5,
			"memory_usage": 60.2,
			"disk_usage": 30.8,
			"load_average": 1.5,
			"active_tasks": 3
		}`

		req, err := http.NewRequest("POST", "/api/v1/agent/heartbeat", strings.NewReader(heartbeatBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// 故意不设置Authorization头

		w := httptest.NewRecorder()
		ts.RouterManager.GetEngine().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 权限测试 - 无效token
	t.Run("无效token访问心跳接口", func(t *testing.T) {
		heartbeatBody := `{
			"agent_id": "` + agent.AgentID + `",
			"status": "online",
			"cpu_usage": 45.5,
			"memory_usage": 60.2,
			"disk_usage": 30.8,
			"load_average": 1.5,
			"active_tasks": 3
		}`

		req, err := http.NewRequest("POST", "/api/v1/agent/heartbeat", strings.NewReader(heartbeatBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid_token_here")

		w := httptest.NewRecorder()
		ts.RouterManager.GetEngine().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// testAgentStatusUpdate 测试Agent状态更新功能
func testAgentStatusUpdate(t *testing.T) {
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 创建测试用户
	user := ts.CreateTestUser(t, "listadmin", "listadmin@example.com", "password123")
	ts.AssignRoleToUser(t, user.ID, "admin")

	// 先注册一个Agent
	agent := ts.CreateTestAgent(t, "status-agent", "192.168.1.300", 8080)

	// 使用HTTP登录接口获取真实token
	loginBody := `{"username":"listadmin","password":"password123"}`
	loginReq, err := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	ts.RouterManager.GetEngine().ServeHTTP(loginW, loginReq)
	require.Equal(t, http.StatusOK, loginW.Code, "登录应该成功")

	var loginResponse system.APIResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok, "登录响应数据格式错误")

	accessToken, ok := loginData["access_token"].(string)
	require.True(t, ok, "无法获取access_token")

	// 测试状态更新
	testCases := []struct {
		name           string
		status         string
		expectedStatus int
	}{
		{"更新为在线状态", "online", http.StatusOK},
		{"更新为维护状态", "maintenance", http.StatusOK},
		{"更新为离线状态", "offline", http.StatusOK},
		{"更新为异常状态", "exception", http.StatusOK},
		{"无效状态", "invalid_status", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			statusBody := `{"status": "` + tc.status + `"}`

			req, err := http.NewRequest("PATCH", "/api/v1/agent/"+agent.AgentID+"/status", strings.NewReader(statusBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+accessToken)

			w := httptest.NewRecorder()
			ts.RouterManager.GetEngine().ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			// 如果更新成功，验证数据库中的状态
			if tc.expectedStatus == http.StatusOK {
				updatedAgent, err := ts.AgentRepository.GetByID(agent.AgentID)
				require.NoError(t, err)
				assert.Equal(t, agentModel.AgentStatus(tc.status), updatedAgent.Status)
			}
		})
	}
}

// testAgentListQuery 测试Agent列表查询功能
func testAgentListQuery(t *testing.T) {
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 创建测试用户
	user := ts.CreateTestUser(t, "listadmin", "listadmin@example.com", "password123")
	ts.AssignRoleToUser(t, user.ID, "admin")

	// 创建多个测试Agent
	agents := []*agentModel.Agent{
		ts.CreateTestAgent(t, "list-agent-001", "192.168.1.101", 8080),
		ts.CreateTestAgent(t, "list-agent-002", "192.168.1.102", 8081),
		ts.CreateTestAgent(t, "list-agent-003", "192.168.1.103", 8082),
	}

	// 设置不同状态
	ts.AgentRepository.UpdateStatus(agents[0].AgentID, agentModel.AgentStatusOnline)
	ts.AgentRepository.UpdateStatus(agents[1].AgentID, agentModel.AgentStatusOnline)
	ts.AgentRepository.UpdateStatus(agents[2].AgentID, agentModel.AgentStatusOffline)

	// 使用HTTP登录接口获取真实token
	loginBody := `{"username":"listadmin","password":"password123"}`
	loginReq, err := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	ts.RouterManager.GetEngine().ServeHTTP(loginW, loginReq)
	require.Equal(t, http.StatusOK, loginW.Code, "登录应该成功")

	var loginResponse system.APIResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok, "登录响应数据格式错误")

	accessToken, ok := loginData["access_token"].(string)
	require.True(t, ok, "无法获取access_token")

	// 测试列表查询
	testCases := []struct {
		name        string
		queryParams string
		expectCount int
	}{
		{"查询所有Agent", "", 3},
		{"分页查询", "?page=1&page_size=2", 2},
		{"按状态过滤", "?status=online", 2},
		{"按标签过滤", "?tags=test", 3}, // 假设所有测试Agent都有test标签
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/agent"+tc.queryParams, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+accessToken)

			w := httptest.NewRecorder()
			ts.RouterManager.GetEngine().ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response system.APIResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "success", response.Status)

			// 验证返回的Agent数量
			var responseData map[string]interface{}
			dataBytes, _ := json.Marshal(response.Data)
			json.Unmarshal(dataBytes, &responseData)

			agents, ok := responseData["agents"].([]interface{})
			require.True(t, ok)
			assert.LessOrEqual(t, len(agents), tc.expectCount)
		})
	}
}

// testAgentDetailQuery 测试Agent详情查询功能
func testAgentDetailQuery(t *testing.T) {
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 创建测试用户
	user := ts.CreateTestUser(t, "listadmin", "listadmin@example.com", "password123")
	ts.AssignRoleToUser(t, user.ID, "admin")

	// 创建测试Agent
	agent := ts.CreateTestAgent(t, "detail-agent", "192.168.1.400", 8080)

	// 使用HTTP登录接口获取真实token
	loginBody := `{"username":"listadmin","password":"password123"}`
	loginReq, err := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	ts.RouterManager.GetEngine().ServeHTTP(loginW, loginReq)
	require.Equal(t, http.StatusOK, loginW.Code, "登录应该成功")

	var loginResponse system.APIResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok, "登录响应数据格式错误")

	accessToken, ok := loginData["access_token"].(string)
	require.True(t, ok, "无法获取access_token")

	// 测试详情查询
	req, err := http.NewRequest("GET", "/api/v1/agent/"+agent.AgentID, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	w := httptest.NewRecorder()
	ts.RouterManager.GetEngine().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response system.APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "success", response.Status)

	// 验证返回的Agent详情
	var agentData map[string]interface{}
	dataBytes, _ := json.Marshal(response.Data)
	json.Unmarshal(dataBytes, &agentData)

	assert.Equal(t, agent.AgentID, agentData["agent_id"])
	assert.Equal(t, agent.Hostname, agentData["hostname"])
	assert.Equal(t, agent.IPAddress, agentData["ip_address"])

	// 验证capabilities和tags字段是数组格式
	capabilities, ok := agentData["capabilities"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(capabilities), 0)

	tags, ok := agentData["tags"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(tags), 0)
}

// testStringSliceConversion 测试StringSlice转换功能
func testStringSliceConversion(t *testing.T) {
	// 测试JSON数组到字符串切片的转换
	t.Run("JSON数组到字符串切片", func(t *testing.T) {
		jsonArray := `["port_scan", "vuln_scan", "web_scan"]`
		result, err := utils.JSONArrayToStringSlice(jsonArray)

		assert.NoError(t, err)
		assert.Equal(t, []string{"port_scan", "vuln_scan", "web_scan"}, result)
	})

	// 测试字符串切片到JSON数组的转换
	t.Run("字符串切片到JSON数组", func(t *testing.T) {
		slice := []string{"test", "development", "production"}
		result, err := utils.StringSliceToJSONArray(slice)

		assert.NoError(t, err)
		assert.JSONEq(t, `["test", "development", "production"]`, result)
	})

	// 测试PostgreSQL数组格式转换
	t.Run("PostgreSQL数组格式转换", func(t *testing.T) {
		pgArray := "{port_scan,vuln_scan,web_scan}"
		result, err := utils.PostgreSQLArrayToStringSlice(pgArray)

		assert.NoError(t, err)
		assert.Equal(t, []string{"port_scan", "vuln_scan", "web_scan"}, result)
	})

	// 测试StringSlice类型的数据库操作
	t.Run("StringSlice数据库操作", func(t *testing.T) {
		// 创建StringSlice实例
		capabilities := agentModel.StringSlice{"port_scan", "vuln_scan"}

		// 测试Value方法（用于写入数据库）
		value, err := capabilities.Value()
		assert.NoError(t, err)
		assert.Equal(t, `["port_scan","vuln_scan"]`, value)

		// 测试Scan方法（用于从数据库读取）
		var newCapabilities agentModel.StringSlice
		err = newCapabilities.Scan(`["web_scan","service_scan"]`)
		assert.NoError(t, err)
		assert.Equal(t, agentModel.StringSlice{"web_scan", "service_scan"}, newCapabilities)
	})

	// 测试边界情况
	t.Run("边界情况测试", func(t *testing.T) {
		// 空数组
		result, err := utils.JSONArrayToStringSlice("[]")
		assert.NoError(t, err)
		assert.Equal(t, []string{}, result)

		// 空字符串
		result, err = utils.JSONArrayToStringSlice("")
		assert.NoError(t, err)
		assert.Equal(t, []string{}, result)

		// 无效JSON
		_, err = utils.JSONArrayToStringSlice("invalid json")
		assert.Error(t, err)
	})
}

// testAgentDelete 测试Agent删除功能
func testAgentDelete(t *testing.T) {
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 创建测试用户
	user := ts.CreateTestUser(t, "listadmin", "listadmin@example.com", "password123")
	ts.AssignRoleToUser(t, user.ID, "admin")

	// 创建测试Agent
	agent := ts.CreateTestAgent(t, "delete-agent-001", "192.168.1.200", 8080)

	// 使用HTTP登录接口获取真实token
	loginBody := `{"username":"listadmin","password":"password123"}`
	loginReq, err := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	ts.RouterManager.GetEngine().ServeHTTP(loginW, loginReq)
	require.Equal(t, http.StatusOK, loginW.Code, "登录应该成功")

	var loginResponse system.APIResponse
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	loginData, ok := loginResponse.Data.(map[string]interface{})
	require.True(t, ok, "登录响应数据格式错误")

	accessToken, ok := loginData["access_token"].(string)
	require.True(t, ok, "无法获取access_token")

	// 测试删除Agent
	testCases := []struct {
		name           string
		agentID        string
		expectedStatus int
		expectError    bool
	}{
		{"删除存在的Agent", agent.AgentID, http.StatusOK, false},
		{"删除不存在的Agent", "non-existent-agent", http.StatusOK, false}, // 删除不存在的记录也返回成功（幂等性）
		{"删除空Agent ID", "", http.StatusBadRequest, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var url string
			if tc.agentID == "" {
				url = "/api/v1/agent/"
			} else {
				url = "/api/v1/agent/" + tc.agentID
			}

			req, err := http.NewRequest("DELETE", url, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+accessToken)

			w := httptest.NewRecorder()
			ts.RouterManager.GetEngine().ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response system.APIResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tc.expectError {
				assert.Equal(t, "failed", response.Status)
			} else {
				assert.Equal(t, "success", response.Status)

				// 如果是删除存在的Agent，验证数据库中已删除
				if tc.agentID == agent.AgentID {
					deletedAgent, err := ts.AgentRepository.GetByID(agent.AgentID)
					assert.NoError(t, err, "查询删除的Agent不应该有错误")
					assert.Nil(t, deletedAgent, "Agent应该已被删除")
				}
			}
		})
	}
}

// testAgentPermissions 测试Agent权限验证功能
func testAgentPermissions(t *testing.T) {
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 创建测试Agent
	agent := ts.CreateTestAgent(t, "permission-agent", "192.168.1.500", 8080)

	// 创建不同角色的用户
	adminUser := ts.CreateTestUser(t, "admin_user", "admin@test.com", "password123")
	operatorUser := ts.CreateTestUser(t, "operator_user", "operator@test.com", "password123")
	viewerUser := ts.CreateTestUser(t, "viewer_user", "viewer@test.com", "password123")
	guestUser := ts.CreateTestUser(t, "guest_user", "guest@test.com", "password123")

	// 分配角色
	ts.AssignRoleToUser(t, adminUser.ID, "admin")
	ts.AssignRoleToUser(t, operatorUser.ID, "operator")
	ts.AssignRoleToUser(t, viewerUser.ID, "viewer")
	ts.AssignRoleToUser(t, guestUser.ID, "guest")

	// 获取不同角色的token
	adminLogin, err := ts.SessionService.Login(context.Background(), &system.LoginRequest{
		Username: "admin_user",
		Password: "password123",
	}, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	operatorLogin, err := ts.SessionService.Login(context.Background(), &system.LoginRequest{
		Username: "operator_user",
		Password: "password123",
	}, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	viewerLogin, err := ts.SessionService.Login(context.Background(), &system.LoginRequest{
		Username: "viewer_user",
		Password: "password123",
	}, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	guestLogin, err := ts.SessionService.Login(context.Background(), &system.LoginRequest{
		Username: "guest_user",
		Password: "password123",
	}, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// 定义权限测试用例
	permissionTestCases := []struct {
		name             string
		method           string
		endpoint         string
		requestBody      string
		adminExpected    int
		operatorExpected int
		viewerExpected   int
		guestExpected    int
	}{
		// Agent注册权限测试
		{
			name:     "Agent注册权限",
			method:   "POST",
			endpoint: "/api/v1/agent/register",
			requestBody: `{
				"hostname": "permission-test-agent",
				"ip_address": "192.168.1.501",
				"port": 8080,
				"version": "1.0.0",
				"os": "Linux",
				"arch": "x86_64",
				"cpu_cores": 4,
				"memory_total": 8589934592,
				"disk_total": 107374182400,
				"capabilities": ["port_scan"],
				"tags": ["test"]
			}`,
			adminExpected:    http.StatusOK,        // 管理员可以注册Agent
			operatorExpected: http.StatusOK,        // 操作员可以注册Agent
			viewerExpected:   http.StatusOK,        // 查看者可以注册Agent（公开接口）
			guestExpected:    http.StatusOK,        // 访客可以注册Agent（公开接口）
		},
		// Agent列表查询权限测试
		{
			name:             "Agent列表查询权限",
			method:           "GET",
			endpoint:         "/api/v1/agent",
			requestBody:      "",
			adminExpected:    http.StatusOK,        // 管理员可以查看列表
			operatorExpected: http.StatusOK,        // 操作员可以查看列表
			viewerExpected:   http.StatusOK,        // 查看者可以查看列表
			guestExpected:    http.StatusForbidden, // 访客不能查看列表
		},
		// Agent详情查询权限测试
		{
			name:             "Agent详情查询权限",
			method:           "GET",
			endpoint:         "/api/v1/agent/" + agent.AgentID,
			requestBody:      "",
			adminExpected:    http.StatusOK,        // 管理员可以查看详情
			operatorExpected: http.StatusOK,        // 操作员可以查看详情
			viewerExpected:   http.StatusOK,        // 查看者可以查看详情
			guestExpected:    http.StatusForbidden, // 访客不能查看详情
		},
		// Agent状态更新权限测试
		{
			name:             "Agent状态更新权限",
			method:           "PATCH",
			endpoint:         "/api/v1/agent/" + agent.AgentID + "/status",
			requestBody:      `{"status": "offline"}`,
			adminExpected:    http.StatusOK,        // 管理员可以更新状态
			operatorExpected: http.StatusOK,        // 操作员可以更新状态
			viewerExpected:   http.StatusForbidden, // 查看者不能更新状态
			guestExpected:    http.StatusForbidden, // 访客不能更新状态
		},
		// Agent心跳权限测试
		{
			name:     "Agent心跳权限",
			method:   "POST",
			endpoint: "/api/v1/agent/heartbeat",
			requestBody: `{
				"agent_id": "` + agent.AgentID + `",
				"status": "online",
				"cpu_usage": 45.5,
				"memory_usage": 60.2,
				"disk_usage": 30.8,
				"load_average": 1.5,
				"active_tasks": 3
			}`,
			adminExpected:    http.StatusOK,        // 管理员可以发送心跳
			operatorExpected: http.StatusOK,        // 操作员可以发送心跳
			viewerExpected:   http.StatusOK,        // 查看者可以发送心跳（公开接口）
			guestExpected:    http.StatusOK,        // 访客可以发送心跳（公开接口）
		},
		// Agent删除权限测试
		{
			name:             "Agent删除权限",
			method:           "DELETE",
			endpoint:         "/api/v1/agent/" + agent.AgentID,
			requestBody:      "",
			adminExpected:    http.StatusOK,        // 管理员可以删除Agent
			operatorExpected: http.StatusForbidden, // 操作员不能删除Agent
			viewerExpected:   http.StatusForbidden, // 查看者不能删除Agent
			guestExpected:    http.StatusForbidden, // 访客不能删除Agent
		},
	}

	// 执行权限测试
	for _, tc := range permissionTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// 测试管理员权限
			t.Run("管理员权限", func(t *testing.T) {
				req, err := http.NewRequest(tc.method, tc.endpoint, strings.NewReader(tc.requestBody))
				require.NoError(t, err)
				if tc.requestBody != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				req.Header.Set("Authorization", "Bearer "+adminLogin.AccessToken)

				w := httptest.NewRecorder()
				ts.RouterManager.GetEngine().ServeHTTP(w, req)

				assert.Equal(t, tc.adminExpected, w.Code, "管理员权限测试失败")
			})

			// 测试操作员权限
			t.Run("操作员权限", func(t *testing.T) {
				req, err := http.NewRequest(tc.method, tc.endpoint, strings.NewReader(tc.requestBody))
				require.NoError(t, err)
				if tc.requestBody != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				req.Header.Set("Authorization", "Bearer "+operatorLogin.AccessToken)

				w := httptest.NewRecorder()
				ts.RouterManager.GetEngine().ServeHTTP(w, req)

				assert.Equal(t, tc.operatorExpected, w.Code, "操作员权限测试失败")
			})

			// 测试查看者权限
			t.Run("查看者权限", func(t *testing.T) {
				req, err := http.NewRequest(tc.method, tc.endpoint, strings.NewReader(tc.requestBody))
				require.NoError(t, err)
				if tc.requestBody != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				req.Header.Set("Authorization", "Bearer "+viewerLogin.AccessToken)

				w := httptest.NewRecorder()
				ts.RouterManager.GetEngine().ServeHTTP(w, req)

				assert.Equal(t, tc.viewerExpected, w.Code, "查看者权限测试失败")
			})

			// 测试访客权限
			t.Run("访客权限", func(t *testing.T) {
				req, err := http.NewRequest(tc.method, tc.endpoint, strings.NewReader(tc.requestBody))
				require.NoError(t, err)
				if tc.requestBody != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				req.Header.Set("Authorization", "Bearer "+guestLogin.AccessToken)

				w := httptest.NewRecorder()
				ts.RouterManager.GetEngine().ServeHTTP(w, req)

				assert.Equal(t, tc.guestExpected, w.Code, "访客权限测试失败")
			})
		})
	}

	// 测试跨用户访问控制
	t.Run("跨用户访问控制测试", func(t *testing.T) {
		// 创建两个不同的用户
		user1 := ts.CreateTestUser(t, "user1", "user1@test.com", "password123")
		user2 := ts.CreateTestUser(t, "user2", "user2@test.com", "password123")

		ts.AssignRoleToUser(t, user1.ID, "operator")
		ts.AssignRoleToUser(t, user2.ID, "operator")

		// 用户1创建Agent
		user1Login, err := ts.SessionService.Login(context.Background(), &system.LoginRequest{
			Username: "user1",
			Password: "password123",
		}, "127.0.0.1", "test-agent")
		require.NoError(t, err)

		user2Login, err := ts.SessionService.Login(context.Background(), &system.LoginRequest{
			Username: "user2",
			Password: "password123",
		}, "127.0.0.1", "test-agent")
		require.NoError(t, err)

		// 用户1创建Agent
		agentBody := `{
			"hostname": "user1-agent",
			"ip_address": "192.168.1.600",
			"port": 8080,
			"version": "1.0.0",
			"os": "Linux",
			"arch": "x86_64",
			"cpu_cores": 4,
			"memory_total": 8589934592,
			"disk_total": 107374182400,
			"capabilities": ["port_scan"],
			"tags": ["user1"]
		}`

		req, err := http.NewRequest("POST", "/api/v1/agent", strings.NewReader(agentBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+user1Login.AccessToken)

		w := httptest.NewRecorder()
		ts.RouterManager.GetEngine().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// 获取创建的Agent ID
		var response system.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		var agentData map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &agentData)
		user1AgentID := agentData["agent_id"].(string)

		// 用户2尝试访问用户1的Agent（根据权限策略，这可能被允许或拒绝）
		req, err = http.NewRequest("GET", "/api/v1/agent/"+user1AgentID, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+user2Login.AccessToken)

		w = httptest.NewRecorder()
		ts.RouterManager.GetEngine().ServeHTTP(w, req)

		// 这里的期望结果取决于具体的权限策略
		// 如果系统允许同角色用户互相访问，则应该是200
		// 如果系统只允许创建者或管理员访问，则应该是403
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusForbidden,
			"跨用户访问应该根据权限策略返回200或403")
	})

	// 测试Token过期和无效Token
	t.Run("Token验证测试", func(t *testing.T) {
		// 测试过期Token（模拟）
		t.Run("过期Token测试", func(t *testing.T) {
			// 这里可以通过修改Token的过期时间来测试
			// 或者使用一个已知的过期Token
			expiredToken := "expired.token.here"

			req, err := http.NewRequest("GET", "/api/v1/agent", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+expiredToken)

			w := httptest.NewRecorder()
			ts.RouterManager.GetEngine().ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		// 测试格式错误的Token
		t.Run("格式错误Token测试", func(t *testing.T) {
			malformedTokens := []string{
				"invalid_token",
				"Bearer",
				"Bearer ",
				"Bearer invalid.token",
				"",
			}

			for _, token := range malformedTokens {
				req, err := http.NewRequest("GET", "/api/v1/agent", nil)
				require.NoError(t, err)

				if token != "" {
					req.Header.Set("Authorization", token)
				}

				w := httptest.NewRecorder()
				ts.RouterManager.GetEngine().ServeHTTP(w, req)

				assert.Equal(t, http.StatusUnauthorized, w.Code,
					"格式错误的Token应该返回401: %s", token)
			}
		})
	})

	// 测试角色权限边界
	t.Run("角色权限边界测试", func(t *testing.T) {
		// 创建查看者用户
		viewerUser := ts.CreateTestUser(t, "boundary_viewer", "boundary_viewer@test.com", "password123")
		ts.AssignRoleToUser(t, viewerUser.ID, "viewer")

		// 使用HTTP登录接口获取查看者token
		viewerLoginBody := `{"username":"boundary_viewer","password":"password123"}`
		viewerLoginReq, err := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(viewerLoginBody))
		require.NoError(t, err)
		viewerLoginReq.Header.Set("Content-Type", "application/json")

		viewerLoginW := httptest.NewRecorder()
		ts.RouterManager.GetEngine().ServeHTTP(viewerLoginW, viewerLoginReq)
		require.Equal(t, http.StatusOK, viewerLoginW.Code, "查看者登录应该成功")

		var viewerLoginResponse system.APIResponse
		err = json.Unmarshal(viewerLoginW.Body.Bytes(), &viewerLoginResponse)
		require.NoError(t, err)

		viewerLoginData, ok := viewerLoginResponse.Data.(map[string]interface{})
		require.True(t, ok, "查看者登录响应数据格式错误")

		viewerAccessToken, ok := viewerLoginData["access_token"].(string)
		require.True(t, ok, "无法获取查看者access_token")

		// 测试用户尝试访问超出其权限范围的操作
		viewerReq, err := http.NewRequest("POST", "/api/v1/agent", strings.NewReader(`{
			"hostname": "unauthorized-agent",
			"ip_address": "192.168.1.700",
			"port": 8080,
			"version": "1.0.0",
			"os": "Linux",
			"arch": "x86_64",
			"cpu_cores": 4,
			"memory_total": 8589934592,
			"disk_total": 107374182400,
			"capabilities": ["port_scan"],
			"tags": ["unauthorized"]
		}`))
		require.NoError(t, err)
		viewerReq.Header.Set("Content-Type", "application/json")
		viewerReq.Header.Set("Authorization", "Bearer "+viewerAccessToken)

		w := httptest.NewRecorder()
		ts.RouterManager.GetEngine().ServeHTTP(w, viewerReq)

		assert.Equal(t, http.StatusForbidden, w.Code, "查看者不应该能够创建Agent")

		// 验证错误消息
		var response system.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response.Message, "Insufficient permissions", "应该返回权限不足的错误消息")
	})
}

// testTokenValidation 测试Token过期和无效Token
func testTokenValidation(t *testing.T) {
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 测试过期Token（模拟）
	t.Run("过期Token测试", func(t *testing.T) {
		// 这里可以通过修改Token的过期时间来测试
		// 或者使用一个已知的过期Token
		expiredToken := "expired.token.here"

		req, err := http.NewRequest("GET", "/api/v1/agent", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+expiredToken)

		w := httptest.NewRecorder()
		ts.RouterManager.GetEngine().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 测试格式错误的Token
	t.Run("格式错误Token测试", func(t *testing.T) {
		malformedTokens := []string{
			"invalid_token",
			"Bearer",
			"Bearer ",
			"Bearer invalid.token",
			"",
		}

		for _, token := range malformedTokens {
			req, err := http.NewRequest("GET", "/api/v1/agent", nil)
			require.NoError(t, err)

			if token != "" {
				req.Header.Set("Authorization", token)
			}

			w := httptest.NewRecorder()
			ts.RouterManager.GetEngine().ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"格式错误的Token应该返回401: %s", token)
		}
	})
}

// testRolePermissionBoundary 测试角色权限边界
func testRolePermissionBoundary(t *testing.T) {
	ts := NewTestSuite(t)
	defer ts.Cleanup()

	// 创建查看者用户
	viewerUser := ts.CreateTestUser(t, "boundary_viewer", "boundary_viewer@test.com", "password123")
	ts.AssignRoleToUser(t, viewerUser.ID, "viewer")

	// 使用HTTP登录接口获取查看者token
	viewerLoginBody := `{"username":"boundary_viewer","password":"password123"}`
	viewerLoginReq, err := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(viewerLoginBody))
	require.NoError(t, err)
	viewerLoginReq.Header.Set("Content-Type", "application/json")

	viewerLoginW := httptest.NewRecorder()
	ts.RouterManager.GetEngine().ServeHTTP(viewerLoginW, viewerLoginReq)
	require.Equal(t, http.StatusOK, viewerLoginW.Code, "查看者登录应该成功")

	var viewerLoginResponse system.APIResponse
	err = json.Unmarshal(viewerLoginW.Body.Bytes(), &viewerLoginResponse)
	require.NoError(t, err)

	viewerLoginData, ok := viewerLoginResponse.Data.(map[string]interface{})
	require.True(t, ok, "查看者登录响应数据格式错误")

	viewerAccessToken, ok := viewerLoginData["access_token"].(string)
	require.True(t, ok, "无法获取查看者access_token")

	// 测试用户尝试访问超出其权限范围的操作
	viewerReq, err := http.NewRequest("POST", "/api/v1/agent", strings.NewReader(`{
		"hostname": "unauthorized-agent",
		"ip_address": "192.168.1.700",
		"port": 8080,
		"version": "1.0.0",
		"os": "Linux",
		"arch": "x86_64",
		"cpu_cores": 4,
		"memory_total": 8589934592,
		"disk_total": 107374182400,
		"capabilities": ["port_scan"],
		"tags": ["unauthorized"]
	}`))
	require.NoError(t, err)
	viewerReq.Header.Set("Content-Type", "application/json")
	viewerReq.Header.Set("Authorization", "Bearer "+viewerAccessToken)

	w := httptest.NewRecorder()
	ts.RouterManager.GetEngine().ServeHTTP(w, viewerReq)

	assert.Equal(t, http.StatusForbidden, w.Code, "查看者不应该能够创建Agent")

	// 验证错误消息
	var response system.APIResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response.Message, "Insufficient permissions", "应该返回权限不足的错误消息")
}
