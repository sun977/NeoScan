/**
 * 服务层:Agent管理服务
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent管理核心业务逻辑，遵循"好品味"原则 - 简洁而强大
 * @func: Agent注册、状态管理、心跳处理、配置推送等核心功能
 */
package agent

import (
	"fmt"
	"time"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
	agentRepository "neomaster/internal/repository/agent"
)

// AgentService Agent服务接口定义
// 定义Agent管理的核心业务方法
type AgentService interface {
	// Agent基础管理
	RegisterAgent(req *agentModel.RegisterAgentRequest) (*agentModel.RegisterAgentResponse, error)
	GetAgentList(req *agentModel.GetAgentListRequest) (*agentModel.GetAgentListResponse, error)
	GetAgentInfo(agentID string) (*agentModel.AgentInfo, error)
	UpdateAgentStatus(agentID string, status agentModel.AgentStatus) error
	DeleteAgent(agentID string) error

	// Agent心跳和状态监控
	ProcessHeartbeat(req *agentModel.HeartbeatRequest) (*agentModel.HeartbeatResponse, error) // 处理Agent发送过来的心跳，更新状态和性能指标
	GetAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error)                 // 主动获取Agent的性能指标
	// 注意：性能指标更新通过ProcessHeartbeat流程中的CreateMetrics完成，无需单独的UpdateAgentMetrics方法

	// Agent配置管理
	GetAgentConfig(agentID string) (*agentModel.AgentConfigResponse, error)
	UpdateAgentConfig(agentID string, config *agentModel.AgentConfigUpdateRequest) error

	// Agent任务管理
	AssignTask(req *agentModel.AgentTaskAssignRequest) (*agentModel.AgentTaskAssignmentResponse, error)
	GetAgentTasks(agentID string) ([]*agentModel.AgentTaskAssignmentResponse, error)

	// Agent分组管理
	CreateAgentGroup(req *agentModel.AgentGroupCreateRequest) (*agentModel.AgentGroupResponse, error)
	AddAgentToGroup(req *agentModel.AgentGroupMemberRequest) error
	RemoveAgentFromGroup(req *agentModel.AgentGroupMemberRequest) error
}

// agentService Agent服务实现
type agentService struct {
	agentRepo agentRepository.AgentRepository // Agent数据访问层
}

// NewAgentService 创建Agent服务实例
// 遵循依赖注入原则，保持代码的可测试性
func NewAgentService(agentRepo agentRepository.AgentRepository) AgentService {
	return &agentService{
		agentRepo: agentRepo,
	}
}

// RegisterAgent Agent注册服务
// 处理Agent注册请求，生成唯一ID和Token
func (s *agentService) RegisterAgent(req *agentModel.RegisterAgentRequest) (*agentModel.RegisterAgentResponse, error) {
	// 生成Agent唯一ID
	agentID := generateAgentID(req.Hostname, req.IPAddress)

	// 检查Agent是否已存在
	existingAgent, err := s.agentRepo.GetByHostname(req.Hostname)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.RegisterAgent", "", map[string]interface{}{
			"operation": "register_agent",
			"option":    "agentService.RegisterAgent",
			"func_name": "service.agent.RegisterAgent",
			"hostname":  req.Hostname,
		})
		return nil, fmt.Errorf("检查Agent是否存在失败: %v", err)
	}

	if existingAgent != nil {
		// Agent已存在，更新信息
		existingAgent.IPAddress = req.IPAddress
		existingAgent.Port = req.Port
		existingAgent.Version = req.Version
		existingAgent.OS = req.OS
		existingAgent.Arch = req.Arch
		existingAgent.CPUCores = req.CPUCores
		existingAgent.MemoryTotal = req.MemoryTotal
		existingAgent.DiskTotal = req.DiskTotal
		existingAgent.ContainerID = req.ContainerID
		existingAgent.PID = req.PID
		existingAgent.Capabilities = req.Capabilities
		existingAgent.Tags = req.Tags
		existingAgent.Remark = req.Remark
		existingAgent.Status = agentModel.AgentStatusOnline
		existingAgent.LastHeartbeat = time.Now()

		if err := s.agentRepo.Update(existingAgent); err != nil {
			logger.LogError(err, "", 0, "", "service.agent.RegisterAgent", "", map[string]interface{}{
				"operation": "register_agent",
				"option":    "agentService.RegisterAgent",
				"func_name": "service.agent.RegisterAgent",
				"agent_id":  existingAgent.AgentID,
			})
			return nil, fmt.Errorf("更新已存在Agent失败: %v", err)
		}

		return &agentModel.RegisterAgentResponse{
			AgentID:     existingAgent.AgentID,
			GRPCToken:   existingAgent.GRPCToken,
			TokenExpiry: time.Now().Add(24 * time.Hour), // Token 24小时后过期
			Status:      "updated",
			Message:     "Agent信息已更新",
		}, nil
	}

	// 创建新Agent
	newAgent := &agentModel.Agent{
		AgentID:       agentID,
		Hostname:      req.Hostname,
		IPAddress:     req.IPAddress,
		Port:          req.Port,
		Version:       req.Version,
		OS:            req.OS,
		Arch:          req.Arch,
		CPUCores:      req.CPUCores,
		MemoryTotal:   req.MemoryTotal,
		DiskTotal:     req.DiskTotal,
		ContainerID:   req.ContainerID,
		PID:           req.PID,
		Capabilities:  req.Capabilities,
		Tags:          req.Tags,
		Remark:        req.Remark,
		Status:        agentModel.AgentStatusOnline,
		GRPCToken:     generateGRPCToken(),
		TokenExpiry:   time.Now().Add(24 * time.Hour), // Token 24小时后过期
		LastHeartbeat: time.Now(),
	}

	if err := s.agentRepo.Create(newAgent); err != nil {
		logger.LogError(err, "", 0, "", "service.agent.RegisterAgent", "", map[string]interface{}{
			"operation": "register_agent",
			"option":    "agentService.RegisterAgent",
			"func_name": "service.agent.RegisterAgent",
			"agent_id":  agentID,
		})
		return nil, fmt.Errorf("创建新Agent失败: %v", err)
	}

	logger.LogInfo("Agent注册成功", "", 0, "", "service.agent.RegisterAgent", "", map[string]interface{}{
		"operation": "register_agent",
		"option":    "agentService.RegisterAgent",
		"func_name": "service.agent.RegisterAgent",
		"agent_id":  agentID,
		"hostname":  req.Hostname,
	})

	return &agentModel.RegisterAgentResponse{
		AgentID:     agentID,
		GRPCToken:   newAgent.GRPCToken,
		TokenExpiry: newAgent.TokenExpiry,
		Status:      "registered",
		Message:     "Agent注册成功",
	}, nil
}

// GetAgentList 获取Agent列表服务
// 支持分页和过滤条件：status 状态、keyword 关键字、tags 标签、capabilities 功能模块
func (s *agentService) GetAgentList(req *agentModel.GetAgentListRequest) (*agentModel.GetAgentListResponse, error) {
	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	// 构建 status 状态过滤条件
	var status *agentModel.AgentStatus
	if req.Status != "" {
		status = &req.Status
	}

	// 构建 keyword 过滤条件
	var keyword *string
	if req.Keyword != "" {
		keyword = &req.Keyword
	}

	// 页码 页码大小 状态 关键字 标签 能力
	agents, total, err := s.agentRepo.GetList(req.Page, req.PageSize, status, keyword, req.Tags, req.Capabilities)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.GetAgentList", "", map[string]interface{}{
			"operation": "get_agent_list",
			"option":    "agentService.GetAgentList",
			"func_name": "service.agent.GetAgentList",
		})
		return nil, fmt.Errorf("获取Agent列表失败: %v", err)
	}

	// 转换为响应格式
	agentInfos := make([]*agentModel.AgentInfo, 0, len(agents))
	for _, agent := range agents {
		agentInfos = append(agentInfos, convertToAgentInfo(agent))
	}

	return &agentModel.GetAgentListResponse{
		Agents: agentInfos,
		Pagination: &agentModel.PaginationResponse{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}, nil
}

// GetAgentInfo 获取Agent详细信息服务
func (s *agentService) GetAgentInfo(agentID string) (*agentModel.AgentInfo, error) {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.GetAgentInfo", "", map[string]interface{}{
			"operation": "get_agent_info",
			"option":    "agentService.GetAgentInfo",
			"func_name": "service.agent.GetAgentInfo",
			"agent_id":  agentID,
		})
		return nil, fmt.Errorf("获取Agent信息失败: %v", err)
	}

	if agent == nil {
		return nil, fmt.Errorf("agent不存在: %s", agentID)
	}

	return convertToAgentInfo(agent), nil
}

// UpdateAgentStatus 更新Agent状态服务
func (s *agentService) UpdateAgentStatus(agentID string, status agentModel.AgentStatus) error {
	err := s.agentRepo.UpdateStatus(agentID, status)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.UpdateAgentStatus", "", map[string]interface{}{
			"operation": "update_agent_status",
			"option":    "agentService.UpdateAgentStatus",
			"func_name": "service.agent.UpdateAgentStatus",
			"agent_id":  agentID,
			"status":    string(status),
		})
		return fmt.Errorf("更新Agent状态失败: %v", err)
	}

	logger.LogInfo("Agent状态更新成功", "", 0, "", "service.agent.UpdateAgentStatus", "", map[string]interface{}{
		"operation": "update_agent_status",
		"option":    "agentService.UpdateAgentStatus",
		"func_name": "service.agent.UpdateAgentStatus",
		"agent_id":  agentID,
		"status":    string(status),
	})

	return nil
}

// DeleteAgent 删除Agent服务
func (s *agentService) DeleteAgent(agentID string) error {
	err := s.agentRepo.Delete(agentID)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.DeleteAgent", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "agentService.DeleteAgent",
			"func_name": "service.agent.DeleteAgent",
			"agent_id":  agentID,
		})
		return fmt.Errorf("删除Agent失败: %v", err)
	}

	logger.LogInfo("Agent删除成功", "", 0, "", "service.agent.DeleteAgent", "", map[string]interface{}{
		"operation": "delete_agent",
		"option":    "agentService.DeleteAgent",
		"func_name": "service.agent.DeleteAgent",
		"agent_id":  agentID,
	})

	return nil
}

// ProcessHeartbeat 处理Agent心跳请求 - 优化后的版本
// 将心跳状态更新和性能指标存储分离，体现"好品味"的数据处理逻辑
func (s *agentService) ProcessHeartbeat(req *agentModel.HeartbeatRequest) (*agentModel.HeartbeatResponse, error) {
	// 1. 更新Agent心跳状态信息到agents表
	// 只更新last_heartbeat、updated_at、status字段，其他字段在注册时已确定
	err := s.agentRepo.UpdateStatus(req.AgentID, req.Status)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateStatus",
			"func_name": "service.agent.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
		return nil, err
	}

	// 更新最后心跳时间
	err = s.agentRepo.UpdateLastHeartbeat(req.AgentID)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.UpdateLastHeartbeat",
			"func_name": "service.agent.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
		return nil, err
	}

	// 2. 处理性能指标数据到agent_metrics表
	// Agent已经提供了完整的性能指标数据，直接使用即可
	if req.Metrics != nil {
		// 确保AgentID正确设置
		req.Metrics.AgentID = req.AgentID

		// 将性能指标存储到agent_metrics表
		err = s.agentRepo.CreateMetrics(req.Metrics)
		if err != nil {
			logger.LogError(err, "", 0, "", "service.agent.ProcessHeartbeat", "", map[string]interface{}{
				"operation": "process_heartbeat",
				"option":    "agentRepo.CreateMetrics",
				"func_name": "service.agent.ProcessHeartbeat",
				"agent_id":  req.AgentID,
			})
			return nil, err
		}

		logger.LogInfo("Agent性能指标存储成功", "", 0, "", "service.agent.ProcessHeartbeat", "", map[string]interface{}{
			"operation": "process_heartbeat",
			"option":    "agentRepo.CreateMetrics",
			"func_name": "service.agent.ProcessHeartbeat",
			"agent_id":  req.AgentID,
		})
	}

	logger.LogInfo("Agent心跳处理成功", "", 0, "", "service.agent.ProcessHeartbeat", "", map[string]interface{}{
		"operation": "process_heartbeat",
		"option":    "ProcessHeartbeat",
		"func_name": "service.agent.ProcessHeartbeat",
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
func (s *agentService) GetAgentMetrics(agentID string) (*agentModel.AgentMetricsResponse, error) {
	// TODO: 实现从时序数据库获取性能指标
	return nil, fmt.Errorf("功能暂未实现")
}

// GetAgentConfig 获取Agent配置服务
func (s *agentService) GetAgentConfig(agentID string) (*agentModel.AgentConfigResponse, error) {
	// TODO: 实现Agent配置获取
	return nil, fmt.Errorf("功能暂未实现")
}

// UpdateAgentConfig 更新Agent配置服务
func (s *agentService) UpdateAgentConfig(agentID string, config *agentModel.AgentConfigUpdateRequest) error {
	// TODO: 实现Agent配置更新
	return fmt.Errorf("功能暂未实现")
}

// AssignTask 分配任务给Agent服务
func (s *agentService) AssignTask(req *agentModel.AgentTaskAssignRequest) (*agentModel.AgentTaskAssignmentResponse, error) {
	// TODO: 实现任务分配
	return nil, fmt.Errorf("功能暂未实现")
}

// GetAgentTasks 获取Agent任务列表服务
func (s *agentService) GetAgentTasks(agentID string) ([]*agentModel.AgentTaskAssignmentResponse, error) {
	// TODO: 实现获取Agent任务列表
	return nil, fmt.Errorf("功能暂未实现")
}

// CreateAgentGroup 创建Agent分组服务
func (s *agentService) CreateAgentGroup(req *agentModel.AgentGroupCreateRequest) (*agentModel.AgentGroupResponse, error) {
	// TODO: 实现Agent分组创建
	return nil, fmt.Errorf("功能暂未实现")
}

// AddAgentToGroup 添加Agent到分组服务
func (s *agentService) AddAgentToGroup(req *agentModel.AgentGroupMemberRequest) error {
	// TODO: 实现添加Agent到分组
	return fmt.Errorf("功能暂未实现")
}

// RemoveAgentFromGroup 从分组移除Agent服务
func (s *agentService) RemoveAgentFromGroup(req *agentModel.AgentGroupMemberRequest) error {
	// TODO: 实现从分组移除Agent
	return fmt.Errorf("功能暂未实现")
}

// generateAgentID 生成Agent唯一ID
// 基于主机名和IP地址生成唯一标识
func generateAgentID(hostname, ipAddress string) string {
	return fmt.Sprintf("agent_%s_%s_%d", hostname, ipAddress, time.Now().Unix())
}

// generateGRPCToken 生成GRPC通信Token
// 用于Agent与Master之间的安全通信
func generateGRPCToken() string {
	return fmt.Sprintf("token_%d", time.Now().UnixNano())
}

// convertToAgentInfo 将Agent模型转换为AgentInfo响应
func convertToAgentInfo(agent *agentModel.Agent) *agentModel.AgentInfo {
	return &agentModel.AgentInfo{
		ID:               uint(agent.ID), // 转换类型从uint64到uint
		AgentID:          agent.AgentID,
		Hostname:         agent.Hostname,
		IPAddress:        agent.IPAddress,
		Port:             agent.Port,
		Version:          agent.Version,
		Status:           agent.Status,
		OS:               agent.OS,
		Arch:             agent.Arch,
		CPUCores:         agent.CPUCores,
		MemoryTotal:      agent.MemoryTotal,
		DiskTotal:        agent.DiskTotal,
		Capabilities:     agent.Capabilities,
		Tags:             agent.Tags,
		LastHeartbeat:    agent.LastHeartbeat,
		ResultLatestTime: agent.ResultLatestTime,
		Remark:           agent.Remark,
		ContainerID:      agent.ContainerID,
		PID:              agent.PID,
		CreatedAt:        agent.CreatedAt,
		UpdatedAt:        agent.UpdatedAt,
	}
}
