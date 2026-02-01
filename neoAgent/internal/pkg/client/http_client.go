/**
 * HTTP通信客户端
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent端与Master端的HTTP通信客户端，遵循Master-Agent交互协议v1.0
 */
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"neoagent/internal/model/client"
	"net/http"
	"time"
)

// HTTPClient HTTP客户端接口
type HTTPClient interface {
	// SetAuthToken 设置认证令牌
	SetAuthToken(token string)

	// RegisterAgent 注册Agent
	RegisterAgent(ctx context.Context, req *client.AgentRegisterRequest) (*client.AgentRegisterResponse, error)

	// SendHeartbeat 发送心跳
	SendHeartbeat(ctx context.Context, req *client.HeartbeatRequest) (*client.HeartbeatResponse, error)

	// FetchTasks 拉取任务
	FetchTasks(ctx context.Context, agentID string) (*client.FetchTasksResponse, error)

	// ReportTaskStatus 上报任务状态/结果
	ReportTaskStatus(ctx context.Context, agentID, taskID string, report *client.TaskStatusReport) (*client.TaskStatusResponse, error)
}

// httpClient HTTP客户端实现
type httpClient struct {
	client     *http.Client
	baseURL    string
	authToken  string
	userAgent  string
	maxRetries int
	retryDelay time.Duration
}

// NewHTTPClient 创建HTTP客户端实例
func NewHTTPClient(baseURL string) HTTPClient {
	return &httpClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		userAgent:  "NeoAgent/1.0",
		maxRetries: 3,
		retryDelay: 5 * time.Second,
	}
}

// SetAuthToken 设置认证令牌
func (c *httpClient) SetAuthToken(token string) {
	c.authToken = token
}

// RegisterAgent 注册Agent
func (c *httpClient) RegisterAgent(ctx context.Context, req *client.AgentRegisterRequest) (*client.AgentRegisterResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/api/v1/agent", req)
	if err != nil {
		return nil, fmt.Errorf("register agent request: %w", err)
	}
	defer resp.Body.Close()

	var result client.AgentRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode register response: %w", err)
	}
	return &result, nil
}

// SendHeartbeat 发送心跳
func (c *httpClient) SendHeartbeat(ctx context.Context, req *client.HeartbeatRequest) (*client.HeartbeatResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/api/v1/agent/heartbeat", req)
	if err != nil {
		return nil, fmt.Errorf("send heartbeat request: %w", err)
	}
	defer resp.Body.Close()

	var result client.HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode heartbeat response: %w", err)
	}
	return &result, nil
}

// FetchTasks 拉取任务
func (c *httpClient) FetchTasks(ctx context.Context, agentID string) (*client.FetchTasksResponse, error) {
	url := fmt.Sprintf("/api/v1/orchestrator/agent/%s/tasks", agentID)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tasks request: %w", err)
	}
	defer resp.Body.Close()

	var result client.FetchTasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode fetch tasks response: %w", err)
	}
	return &result, nil
}

// ReportTaskStatus 上报任务状态/结果
func (c *httpClient) ReportTaskStatus(ctx context.Context, agentID, taskID string, report *client.TaskStatusReport) (*client.TaskStatusResponse, error) {
	url := fmt.Sprintf("/api/v1/orchestrator/agent/%s/tasks/%s/status", agentID, taskID)
	resp, err := c.doRequest(ctx, "POST", url, report)
	if err != nil {
		return nil, fmt.Errorf("report task status request: %w", err)
	}
	defer resp.Body.Close()

	var result client.TaskStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode report task status response: %w", err)
	}
	return &result, nil
}

// doRequest 执行HTTP请求
func (c *httpClient) doRequest(ctx context.Context, method, url string, data interface{}) (*http.Response, error) {
	fullURL := c.baseURL + url

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshal request data: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	var resp *http.Response
	for i := 0; i <= c.maxRetries; i++ {
		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break
		}
		if i < c.maxRetries {
			time.Sleep(c.retryDelay)
		}
	}

	if err != nil {
		return nil, err
	}

	// Check for non-200 status codes if needed, but for now we let the caller decode the JSON response
	// which usually contains a code field. However, if the server returns 404/500 with non-JSON body, decoding will fail.
	// Ideally we should check StatusCode here.
	if resp.StatusCode >= 400 {
		// Try to read body for error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("http request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
