/**
 * Master通信服务
 * @author: sun977
 * @date: 2025.10.21
 * @description: 处理Agent与Master端的通信，包括注册、心跳、任务拉取和结果上报
 */
package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	modelComm "neoagent/internal/model/client"
	httpclient "neoagent/internal/pkg/client"
	"neoagent/internal/pkg/logger"
	"neoagent/internal/pkg/monitor"
)

// MasterService Master通信服务接口
type MasterService interface {
	// Register 向Master注册Agent
	Register(ctx context.Context, req *modelComm.AgentRegisterRequest) error

	// StartHeartbeat 开启心跳上报
	StartHeartbeat(ctx context.Context)

	// StartTaskPoller 开启任务轮询
	StartTaskPoller(ctx context.Context, interval time.Duration) <-chan []modelComm.Task

	// ReportTask 上报任务状态/结果
	ReportTask(ctx context.Context, taskID string, status string, result string, errorMsg string) error

	// GetAgentID 获取Agent ID
	GetAgentID() string
}

// masterService Master通信服务实现
type masterService struct {
	client    httpclient.HTTPClient
	agentID   string
	token     string
	status    string
	mu        sync.RWMutex
	stopChan  chan struct{}
	taskStats struct {
		Running   int
		Completed int
		Failed    int
	}
}

// NewMasterService 创建Master通信服务实例
func NewMasterService(baseURL string) MasterService {
	return &masterService{
		client:   httpclient.NewHTTPClient(baseURL),
		status:   "offline",
		stopChan: make(chan struct{}),
	}
}

// Register 向Master注册Agent
func (s *masterService) Register(ctx context.Context, req *modelComm.AgentRegisterRequest) error {
	logger.LogSystemEvent("MasterService", "Register", "Starting registration...", logger.InfoLevel, nil)

	resp, err := s.client.RegisterAgent(ctx, req)
	if err != nil {
		logger.LogSystemEvent("MasterService", "Register", fmt.Sprintf("Registration failed: %v", err), logger.ErrorLevel, nil)
		return err
	}

	if resp.Code != 200 {
		return fmt.Errorf("registration failed with code %d: %s", resp.Code, resp.Status)
	}

	s.mu.Lock()
	s.agentID = resp.Data.AgentID
	s.token = resp.Data.AuthToken
	s.status = "online"
	s.client.SetAuthToken(s.token)
	s.mu.Unlock()

	logger.LogSystemEvent("MasterService", "Register", fmt.Sprintf("Registered successfully. AgentID: %s", s.agentID), logger.InfoLevel, nil)
	return nil
}

// StartHeartbeat 开启心跳上报
func (s *masterService) StartHeartbeat(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.sendHeartbeat(ctx)
			}
		}
	}()
}

// sendHeartbeat 发送单次心跳
func (s *masterService) sendHeartbeat(ctx context.Context) {
	s.mu.RLock()
	agentID := s.agentID
	status := s.status
	stats := s.taskStats
	s.mu.RUnlock()

	if agentID == "" {
		return
	}

	// Collect real system metrics
	sysMetrics, err := monitor.GetSystemMetrics()
	if err != nil {
		logger.LogSystemEvent("MasterService", "Heartbeat", fmt.Sprintf("Failed to get system metrics: %v", err), logger.WarnLevel, nil)
		// Continue with empty metrics or partial metrics
		if sysMetrics == nil {
			sysMetrics = &monitor.SystemMetrics{}
		}
	}

	metrics := &modelComm.HeartbeatMetrics{
		AgentID:           agentID,
		CPUUsage:          sysMetrics.CPUUsage,
		MemoryUsage:       sysMetrics.MemoryUsage,
		DiskUsage:         sysMetrics.DiskUsage,
		NetworkBytesSent:  sysMetrics.NetworkBytesSent,
		NetworkBytesRecv:  sysMetrics.NetworkBytesRecv,
		ActiveConnections: 0, // Placeholder, difficult to get accurate count cheaply without netstat
		RunningTasks:      stats.Running,
		CompletedTasks:    stats.Completed,
		FailedTasks:       stats.Failed,
		WorkStatus:        s.determineWorkStatus(stats.Running),
		ScanType:          "idle", // TODO: Get from TaskEngine if possible
		PluginStatus:      make(modelComm.PluginStatus),
		Timestamp:         time.Now(),
	}

	req := &modelComm.HeartbeatRequest{
		AgentID: agentID,
		Status:  status,
		Metrics: metrics,
	}

	resp, err := s.client.SendHeartbeat(ctx, req)
	if err != nil {
		logger.LogSystemEvent("MasterService", "Heartbeat", fmt.Sprintf("Failed to send heartbeat: %v", err), logger.ErrorLevel, nil)
		return
	}

	if len(resp.Data.RuleVersions) > 0 {
		logger.LogSystemEvent("MasterService", "Heartbeat", fmt.Sprintf("Received rule versions: %v", resp.Data.RuleVersions), logger.InfoLevel, nil)
	}
}

// StartTaskPoller 开启任务轮询
func (s *masterService) StartTaskPoller(ctx context.Context, interval time.Duration) <-chan []modelComm.Task {
	taskChan := make(chan []modelComm.Task)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(taskChan)

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopChan:
				return
			case <-ticker.C:
				tasks, err := s.fetchTasks(ctx)
				if err != nil {
					logger.LogSystemEvent("MasterService", "TaskPoller", fmt.Sprintf("Failed to fetch tasks: %v", err), logger.ErrorLevel, nil)
					continue
				}
				if len(tasks) > 0 {
					taskChan <- tasks
				}
			}
		}
	}()

	return taskChan
}

// fetchTasks 拉取任务
func (s *masterService) fetchTasks(ctx context.Context) ([]modelComm.Task, error) {
	s.mu.RLock()
	agentID := s.agentID
	s.mu.RUnlock()

	if agentID == "" {
		return nil, fmt.Errorf("agent not registered")
	}

	resp, err := s.client.FetchTasks(ctx, agentID)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("fetch tasks failed with code %d: %s", resp.Code, resp.Status)
	}

	return resp.Data, nil
}

// ReportTask 上报任务状态/结果
func (s *masterService) ReportTask(ctx context.Context, taskID string, status string, result string, errorMsg string) error {
	s.mu.Lock() // Use Lock for updating stats
	agentID := s.agentID

	// Update stats
	switch status {
	case "running":
		s.taskStats.Running++
	case "completed", "success":
		if s.taskStats.Running > 0 {
			s.taskStats.Running--
		}
		s.taskStats.Completed++
	case "failed":
		if s.taskStats.Running > 0 {
			s.taskStats.Running--
		}
		s.taskStats.Failed++
	}
	s.mu.Unlock()

	if agentID == "" {
		return fmt.Errorf("agent not registered")
	}

	report := &modelComm.TaskStatusReport{
		Status:   status,
		Result:   result,
		ErrorMsg: errorMsg,
	}

	resp, err := s.client.ReportTaskStatus(ctx, agentID, taskID, report)
	if err != nil {
		return err
	}

	if resp.Code != 200 {
		return fmt.Errorf("report task status failed with code %d: %s", resp.Code, resp.Status)
	}

	return nil
}

// determineWorkStatus 根据运行任务数确定工作状态
func (s *masterService) determineWorkStatus(runningTasks int) string {
	if runningTasks > 0 {
		return "working"
	}
	return "idle"
}

// GetAgentID 获取Agent ID
func (s *masterService) GetAgentID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agentID
}
