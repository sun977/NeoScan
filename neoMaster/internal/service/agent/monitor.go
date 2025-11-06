/**
 * 服务层:Agent监控服务
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent监控核心业务逻辑，遵循"好品味"原则 - 专注监控职责
 * @func: Agent心跳处理、性能指标监控、健康状态检查、负载监控
 */
package agent

import (
	"fmt"
	agentRepository "neomaster/internal/repo/mysql/agent"
	"time"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// AgentMonitorService Agent监控服务接口
// 专门负责Agent的监控相关功能，遵循单一职责原则
type AgentMonitorService interface {
	// Agent心跳和状态监控
	ProcessHeartbeat(req *agentModel.HeartbeatRequest) (*agentModel.HeartbeatResponse, error)                                                                                                        // 处理Agent发送过来的心跳，更新状态和指标
	GetAgentMetricsFromDB(agentID string) (*agentModel.AgentMetricsResponse, error)                                                                                                                  // 从数据库获取Agent最新的性能指标
	GetAgentListAllMetricsFromDB(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetricsResponse, int64, error) // 从数据库分页获取Agent的最新性能指标（支持状态与关键词过滤）
	PullAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error)                                                                                                                       // 从Agent端拉取最新的性能指标
	PullAgentListAllMetrics() ([]*agentModel.AgentMetricsResponse, error)                                                                                                                            // 从Agent端拉取所有Agent的最新性能指标
	CreateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error                                                                                                                       // 创建Agent性能指标
	UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error                                                                                                                       // 更新Agent性能指标
}

// agentMonitorService Agent监控服务实现
type agentMonitorService struct {
	agentRepo agentRepository.AgentRepository // Agent数据访问层
}

// NewAgentMonitorService 创建Agent监控服务实例
// 遵循依赖注入原则，保持代码的可测试性
func NewAgentMonitorService(agentRepo agentRepository.AgentRepository) AgentMonitorService {
	return &agentMonitorService{
		agentRepo: agentRepo,
	}
}

// ProcessHeartbeat 处理Agent心跳请求 - 优化后的版本
// 将心跳状态更新和性能指标存储分离，体现"好品味"的数据处理逻辑
func (s *agentMonitorService) ProcessHeartbeat(req *agentModel.HeartbeatRequest) (*agentModel.HeartbeatResponse, error) {
	// 1. 更新Agent心跳状态信息到agents表
	// 只更新last_heartbeat、updated_at、status字段，其他字段在注册时已确定
	// 更新 status 字段 - agents 表
	err := s.agentRepo.UpdateStatus(req.AgentID, req.Status)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateStatus",
			"func_name": "service.agent.monitor.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
		return nil, err
	}

	// 更新最后心跳时间 - agents 表 (同时更新 updated_at 和 last_heartbeat 字段)
	err = s.agentRepo.UpdateLastHeartbeat(req.AgentID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateLastHeartbeat",
			"func_name": "service.agent.monitor.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
		return nil, err
	}

	// 2. 处理性能指标数据到agent_metrics表
	// Agent已经提供了完整的性能指标数据，直接使用即可
	if req.Metrics != nil {
		// 确保AgentID正确设置
		req.Metrics.AgentID = req.AgentID

		// 更新性能指标到agent_metrics表（使用upsert逻辑：存在则更新，不存在则创建）
		err = s.agentRepo.UpdateAgentMetrics(req.AgentID, req.Metrics)
		if err != nil {
			logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
				"operation": "process_heartbeat",
				"option":    "agentRepo.UpdateAgentMetrics",
				"func_name": "service.agent.monitor.ProcessHeartbeat",
				"agent_id":  req.AgentID,
			})
			return nil, err
		}

		logger.LogInfo("Agent性能指标更新成功", "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateAgentMetrics",
			"func_name": "service.agent.monitor.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
	}

	logger.LogInfo("Agent心跳处理成功", "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
		"operation": "process_heartbeat",
		"option":    "ProcessHeartbeat",
		"func_name": "service.agent.monitor.ProcessHeartbeat",
		"agent_id":  req.AgentID,
		"status":    string(req.Status),
	})

	// 构造响应
	response := &agentModel.HeartbeatResponse{
		AgentID:   req.AgentID,
		Status:    "success",
		Message:   "Heartbeat processed successfully",
		Timestamp: time.Now(),
	}

	return response, nil
}

// GetAgentMetricsFromDB 获取指定Agent性能指标服务 - 从数据库表 agent_metrics 查询
func (s *agentMonitorService) GetAgentMetricsFromDB(agentID string) (*agentModel.AgentMetricsResponse, error) {
	// 输入校验：agentID不能为空
	if agentID == "" {
		err := fmt.Errorf("agentID 不能为空")
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentMetricsFromDB", "", map[string]interface{}{
			"operation": "get_agent_metrics_from_db",
			"option":    "validate.agentID",
			"func_name": "service.agent.monitor.GetAgentMetricsFromDB",
			"agent_id":  agentID,
		})
		return nil, err
	}

	// 通过仓储层获取该Agent的最新性能指标（遵循分层：Service -> Repository）
	metrics, err := s.agentRepo.GetLatestMetrics(agentID)
	if err != nil {
		// 数据查询错误
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentMetricsFromDB", "", map[string]interface{}{
			"operation": "get_agent_metrics_from_db",
			"option":    "agentRepo.GetLatestMetrics",
			"func_name": "service.agent.monitor.GetAgentMetricsFromDB",
			"agent_id":  agentID,
		})
		return nil, fmt.Errorf("查询Agent性能指标失败: %v", err)
	}

	// 未找到记录（根据当前单快照模型，表示尚未有该Agent的指标被写入）
	if metrics == nil {
		err := fmt.Errorf("未找到Agent的性能指标记录")
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentMetricsFromDB", "", map[string]interface{}{
			"operation": "get_agent_metrics_from_db",
			"option":    "agentRepo.GetLatestMetrics.not_found",
			"func_name": "service.agent.monitor.GetAgentMetricsFromDB",
			"agent_id":  agentID,
		})
		return nil, err
	}

	// 组装响应结构（与AgentMetricsResponse字段一一对应）
	resp := &agentModel.AgentMetricsResponse{
		AgentID:           metrics.AgentID,
		CPUUsage:          metrics.CPUUsage,
		MemoryUsage:       metrics.MemoryUsage,
		DiskUsage:         metrics.DiskUsage,
		NetworkBytesSent:  metrics.NetworkBytesSent,
		NetworkBytesRecv:  metrics.NetworkBytesRecv,
		ActiveConnections: metrics.ActiveConnections,
		RunningTasks:      metrics.RunningTasks,
		CompletedTasks:    metrics.CompletedTasks,
		FailedTasks:       metrics.FailedTasks,
		WorkStatus:        metrics.WorkStatus,
		ScanType:          metrics.ScanType,
		// PluginStatusJSON 是 map[string]interface{} 的自定义类型，直接类型转换即可
		PluginStatus: map[string]interface{}(metrics.PluginStatus),
		Timestamp:    metrics.Timestamp,
	}

	// 成功日志记录
	logger.LogInfo("获取Agent性能指标成功", "", 0, "", "service.agent.monitor.GetAgentMetricsFromDB", "", map[string]interface{}{
		"operation": "get_agent_metrics_from_db",
		"option":    "compose.AgentMetricsResponse",
		"func_name": "service.agent.monitor.GetAgentMetricsFromDB",
		"agent_id":  agentID,
	})

	return resp, nil
}

// GetAgentListAllMetricsFromDB 获取所有Agent性能指标服务 - 从数据表 agent_metrics 批量查询
func (s *agentMonitorService) GetAgentListAllMetricsFromDB(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetricsResponse, int64, error) {
	// 输入校验与默认值处理（防御性编程）
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 直接通过仓储层对 agent_metrics 表进行分页查询（单快照模型，避免N+1与全量加载）
	metricsList, total, err := s.agentRepo.GetMetricsList(page, pageSize, workStatus, scanType, keyword)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.GetAgentListAllMetricsFromDB", "", map[string]interface{}{
			"operation": "get_agent_list_all_metrics_from_db",
			"option":    "agentRepo.GetMetricsList",
			"func_name": "service.agent.monitor.GetAgentListAllMetricsFromDB",
			"page":      page,
			"page_size": pageSize,
			"work_status": func() interface{} {
				if workStatus != nil {
					return *workStatus
				}
				return nil
			}(),
			"scan_type": func() interface{} {
				if scanType != nil {
					return *scanType
				}
				return nil
			}(),
			"keyword": func() interface{} {
				if keyword != nil {
					return *keyword
				}
				return nil
			}(),
		})
		return nil, 0, fmt.Errorf("查询Agent性能指标分页失败: %v", err)
	}

	// 组装响应结构
	results := make([]*agentModel.AgentMetricsResponse, 0, len(metricsList))
	for _, metrics := range metricsList {
		if metrics == nil {
			continue
		}
		resp := &agentModel.AgentMetricsResponse{
			AgentID:           metrics.AgentID,
			CPUUsage:          metrics.CPUUsage,
			MemoryUsage:       metrics.MemoryUsage,
			DiskUsage:         metrics.DiskUsage,
			NetworkBytesSent:  metrics.NetworkBytesSent,
			NetworkBytesRecv:  metrics.NetworkBytesRecv,
			ActiveConnections: metrics.ActiveConnections,
			RunningTasks:      metrics.RunningTasks,
			CompletedTasks:    metrics.CompletedTasks,
			FailedTasks:       metrics.FailedTasks,
			WorkStatus:        metrics.WorkStatus,
			ScanType:          metrics.ScanType,
			PluginStatus:      map[string]interface{}(metrics.PluginStatus),
			Timestamp:         metrics.Timestamp,
		}
		results = append(results, resp)
	}

	// 成功日志记录
	logger.LogInfo("获取所有Agent性能指标成功（分页）", "", 0, "", "service.agent.monitor.GetAgentListAllMetricsFromDB", "", map[string]interface{}{
		"operation": "get_agent_list_all_metrics_from_db",
		"option":    "agentRepo.GetMetricsList",
		"func_name": "service.agent.monitor.GetAgentListAllMetricsFromDB",
		"count":     len(results),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"work_status": func() interface{} {
			if workStatus != nil {
				return *workStatus
			}
			return nil
		}(),
		"scan_type": func() interface{} {
			if scanType != nil {
				return *scanType
			}
			return nil
		}(),
		"keyword": func() interface{} {
			if keyword != nil {
				return *keyword
			}
			return nil
		}(),
	})

	return results, total, nil
}

// PullAgentMetrics 从 agent 端拉取性能指标数据服务 - 从 agent 端调用 /metrics 接口获取数据 (拉取后同步更新数据表 agent_metrics 中数据)
func (s *agentMonitorService) PullAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error) {
	// TODO: 实现从 agent 端调用 /metrics 接口获取数据
	logger.LogInfo("从 agent 端拉取性能指标数据", "", 0, "", "service.agent.monitor.pullAgentMetrics", "", map[string]interface{}{
		"operation": "pull_agent_metrics",
		"option":    "agentMonitorService.PullAgentMetrics",
		"func_name": "service.agent.monitor.PullAgentMetrics",
		"agent_id":  agentID,
	})
	return nil, fmt.Errorf("功能暂未实现")
}

// PullAgentListAllMetrics 从所有 agent 端拉取性能指标数据服务 - 从所有 agent 端调用 /metrics 接口获取数据 (拉取后同步更新数据表 agent_metrics 中数据)
func (s *agentMonitorService) PullAgentListAllMetrics() ([]*agentModel.AgentMetricsResponse, error) {
	// TODO: 实现从所有 agent 端调用 /metrics 接口获取数据
	logger.LogInfo("从所有 agent 端拉取性能指标数据", "", 0, "", "service.agent.monitor.pullAgentListAllMetrics", "", map[string]interface{}{
		"operation": "pull_agent_list_all_metrics",
		"option":    "agentMonitorService.PullAgentListAllMetrics",
		"func_name": "service.agent.monitor.PullAgentListAllMetrics",
	})
	return nil, fmt.Errorf("功能暂未实现")
}

// CreateAgentMetrics 创建Agent性能指标服务 (保留服务,用于agent上报性能指标数据或master创建性能指标数据)
func (s *agentMonitorService) CreateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	// TODO: 实现创建Agent性能指标到数据表 agent_metrics 中
	if err := s.agentRepo.CreateMetrics(metrics); err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.CreateAgentMetrics", "", map[string]interface{}{
			"operation": "create_agent_metrics",
			"option":    "agentMonitorService.CreateAgentMetrics",
			"func_name": "service.agent.monitor.CreateAgentMetrics",
			"agent_id":  agentID,
		})
		return fmt.Errorf("创建Agent性能指标失败: %v", err)
	}

	logger.LogInfo("Agent性能指标创建成功", "", 0, "", "service.agent.monitor.CreateAgentMetrics", "", map[string]interface{}{
		"operation": "create_agent_metrics",
		"option":    "agentMonitorService.CreateAgentMetrics",
		"func_name": "service.agent.monitor.CreateAgentMetrics",
		"agent_id":  agentID,
	})

	return nil
}

// UpdateAgentMetrics 更新Agent性能指标服务 (保留服务,用于master恢复性能指标数据)
func (s *agentMonitorService) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	if err := s.agentRepo.UpdateAgentMetrics(agentID, metrics); err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.monitor.UpdateAgentMetrics", "", map[string]interface{}{
			"operation": "update_agent_metrics",
			"option":    "agentMonitorService.UpdateAgentMetrics",
			"func_name": "service.agent.monitor.UpdateAgentMetrics",
			"agent_id":  agentID,
		})
		return fmt.Errorf("更新Agent性能指标失败: %v", err)
	}

	logger.LogInfo("Agent性能指标更新成功", "", 0, "", "service.agent.monitor.UpdateAgentMetrics", "", map[string]interface{}{
		"operation": "update_agent_metrics",
		"option":    "agentMonitorService.UpdateAgentMetrics",
		"func_name": "service.agent.monitor.UpdateAgentMetrics",
		"agent_id":  agentID,
	})

	return nil
}
