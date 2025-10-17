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
	agentRepository "neomaster/internal/repository/mysql/agent"
	"time"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// AgentMonitorService Agent监控服务接口
// 专门负责Agent的监控相关功能，遵循单一职责原则
type AgentMonitorService interface {
	// Agent心跳和状态监控
	ProcessHeartbeat(req *agentModel.HeartbeatRequest) (*agentModel.HeartbeatResponse, error) // 处理Agent发送过来的心跳，更新状态和指标
	GetAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error)                 // 获取Agent最新的性能指标
	UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error                // 更新Agent性能指标
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
	err := s.agentRepo.UpdateStatus(req.AgentID, req.Status)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateStatus",
			"func_name": "service.agent.monitor.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
		return nil, err
	}

	// 更新最后心跳时间
	err = s.agentRepo.UpdateLastHeartbeat(req.AgentID)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
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
			logger.LogError(err, "", 0, "", "service.agent.monitor.ProcessHeartbeat", "", map[string]interface{}{
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

// GetAgentMetrics 获取Agent性能指标服务
func (s *agentMonitorService) GetAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error) {
	// TODO: 实现从时序数据库获取性能指标
	logger.LogInfo("获取Agent性能指标", "", 0, "", "service.agent.monitor.GetAgentMetrics", "", map[string]interface{}{
		"operation": "get_agent_metrics",
		"option":    "agentMonitorService.GetAgentMetrics",
		"func_name": "service.agent.monitor.GetAgentMetrics",
		"agent_id":  agentID,
	})
	return nil, fmt.Errorf("功能暂未实现")
}

// UpdateAgentMetrics 更新Agent性能指标服务
func (s *agentMonitorService) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	if err := s.agentRepo.UpdateAgentMetrics(agentID, metrics); err != nil {
		logger.LogError(err, "", 0, "", "service.agent.monitor.UpdateAgentMetrics", "", map[string]interface{}{
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
