/**
 * 服务层:Agent基础管理服务
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent基础管理核心业务逻辑，遵循"好品味"原则 - 只管Agent的生命周期
 * @func: Agent注册、查询、状态更新、删除、Agent分组管理
 */
package agent

import (
	"fmt"
	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	agentRepository "neomaster/internal/repository/agent"
	"time"
)

// AgentManagerService Agent基础管理服务接口
// 只负责Agent的基础CRUD操作，遵循单一职责原则
type AgentManagerService interface {
	// Agent基础管理 - 核心职责
	RegisterAgent(req *agentModel.RegisterAgentRequest) (*agentModel.RegisterAgentResponse, error)
	GetAgentList(req *agentModel.GetAgentListRequest) (*agentModel.GetAgentListResponse, error)
	GetAgentInfo(agentID string) (*agentModel.AgentInfo, error)
	UpdateAgentStatus(agentID string, status agentModel.AgentStatus) error
	DeleteAgent(agentID string) error
}

// agentManagerService Agent基础管理服务实现
type agentManagerService struct {
	agentRepo agentRepository.AgentRepository // Agent数据访问层
}

// NewAgentManagerService 创建Agent基础管理服务实例
// 遵循依赖注入原则，保持代码的可测试性
func NewAgentManagerService(agentRepo agentRepository.AgentRepository) AgentManagerService {
	return &agentManagerService{
		agentRepo: agentRepo,
	}
}

// RegisterAgent Agent注册服务
// 处理Agent注册请求，生成唯一ID和Token
func (s *agentManagerService) RegisterAgent(req *agentModel.RegisterAgentRequest) (*agentModel.RegisterAgentResponse, error) {
	// 参数验证
	if err := validateRegisterRequest(req); err != nil {
		logger.LogError(err, "", 0, "", "service.agent.manager.RegisterAgent", "", map[string]interface{}{
			"operation": "register_agent",
			"option":    "parameter_validation",
			"func_name": "service.agent.manager.RegisterAgent",
			"hostname":  req.Hostname,
		})
		return nil, err
	}

	// 生成Agent唯一ID
	agentID := generateAgentID(req.Hostname)

	// 检查Agent是否已存在
	existingAgent, err := s.agentRepo.GetByHostname(req.Hostname)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.manager.RegisterAgent", "", map[string]interface{}{
			"operation": "register_agent",
			"option":    "agentManagerService.RegisterAgent",
			"func_name": "service.agent.manager.RegisterAgent",
			"hostname":  req.Hostname,
		})
		return nil, fmt.Errorf("检查Agent是否存在失败: %v", err)
	}

	if existingAgent != nil {
		// Agent已存在，返回409冲突错误
		logger.LogError(
			fmt.Errorf("agent with hostname %s already exists", req.Hostname),
			"", 0, "", "service.agent.manager.RegisterAgent", "",
			map[string]interface{}{
				"operation": "register_agent",
				"option":    "duplicate_hostname_check",
				"func_name": "service.agent.manager.RegisterAgent",
				"hostname":  req.Hostname,
				"agent_id":  existingAgent.AgentID,
			},
		)
		return nil, fmt.Errorf("agent with hostname %s already exists", req.Hostname)
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
		logger.LogError(err, "", 0, "", "service.agent.manager.RegisterAgent", "", map[string]interface{}{
			"operation": "register_agent",
			"option":    "agentManagerService.RegisterAgent",
			"func_name": "service.agent.manager.RegisterAgent",
			"agent_id":  agentID,
		})
		return nil, fmt.Errorf("创建新Agent失败: %v", err)
	}

	logger.LogInfo("Agent注册成功", "", 0, "", "service.agent.manager.RegisterAgent", "", map[string]interface{}{
		"operation": "register_agent",
		"option":    "agentManagerService.RegisterAgent",
		"func_name": "service.agent.manager.RegisterAgent",
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
func (s *agentManagerService) GetAgentList(req *agentModel.GetAgentListRequest) (*agentModel.GetAgentListResponse, error) {
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
		logger.LogError(err, "", 0, "", "service.agent.manager.GetAgentList", "", map[string]interface{}{
			"operation": "get_agent_list",
			"option":    "agentManagerService.GetAgentList",
			"func_name": "service.agent.manager.GetAgentList",
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
func (s *agentManagerService) GetAgentInfo(agentID string) (*agentModel.AgentInfo, error) {
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.manager.GetAgentInfo", "", map[string]interface{}{
			"operation": "get_agent_info",
			"option":    "agentManagerService.GetAgentInfo",
			"func_name": "service.agent.manager.GetAgentInfo",
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
func (s *agentManagerService) UpdateAgentStatus(agentID string, status agentModel.AgentStatus) error {
	err := s.agentRepo.UpdateStatus(agentID, status)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.manager.UpdateAgentStatus", "", map[string]interface{}{
			"operation": "update_agent_status",
			"option":    "agentManagerService.UpdateAgentStatus",
			"func_name": "service.agent.manager.UpdateAgentStatus",
			"agent_id":  agentID,
			"status":    string(status),
		})
		return fmt.Errorf("更新Agent状态失败: %v", err)
	}

	logger.LogInfo("Agent状态更新成功", "", 0, "", "service.agent.manager.UpdateAgentStatus", "", map[string]interface{}{
		"operation": "update_agent_status",
		"option":    "agentManagerService.UpdateAgentStatus",
		"func_name": "service.agent.manager.UpdateAgentStatus",
		"agent_id":  agentID,
		"status":    string(status),
	})

	return nil
}

// DeleteAgent 删除Agent服务
func (s *agentManagerService) DeleteAgent(agentID string) error {
	err := s.agentRepo.Delete(agentID)
	if err != nil {
		logger.LogError(err, "", 0, "", "service.agent.manager.DeleteAgent", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "agentManagerService.DeleteAgent",
			"func_name": "service.agent.manager.DeleteAgent",
			"agent_id":  agentID,
		})
		return fmt.Errorf("删除Agent失败: %v", err)
	}

	logger.LogInfo("Agent删除成功", "", 0, "", "service.agent.manager.DeleteAgent", "", map[string]interface{}{
		"operation": "delete_agent",
		"option":    "agentManagerService.DeleteAgent",
		"func_name": "service.agent.manager.DeleteAgent",
		"agent_id":  agentID,
	})

	return nil
}

// generateAgentID 生成Agent唯一ID
// 基于主机名和时间生成唯一标识
func generateAgentID(hostname string) string {
	// 使用UUID生成固定长度的agent_id，避免超过数据库字段限制
	// 格式：agent_uuid，总长度约42字符，远小于数据库的100字符限制
	uuid, err := utils.GenerateUUID()
	if err != nil {
		// 如果UUID生成失败，使用时间戳作为后备方案，但要截断hostname避免过长
		shortHostname := hostname
		if len(hostname) > 20 {
			shortHostname = hostname[:20]
		}
		return fmt.Sprintf("agent_%s_%d", shortHostname, time.Now().Unix())
	}
	return fmt.Sprintf("agent_%s", uuid)
}

// validateRegisterRequest 验证Agent注册请求参数
// 参数: req - Agent注册请求结构体指针
// 返回: error - 验证失败时的错误信息
func validateRegisterRequest(req *agentModel.RegisterAgentRequest) error {
	// 检查hostname长度
	if len(req.Hostname) > 255 {
		return fmt.Errorf("hostname too long")
	}

	// 检查version长度
	if len(req.Version) > 50 {
		return fmt.Errorf("version too long")
	}

	// 检查CPU核心数
	if req.CPUCores < 0 {
		return fmt.Errorf("invalid CPU cores")
	}

	// 检查内存总量
	if req.MemoryTotal < 0 {
		return fmt.Errorf("invalid memory total")
	}

	// 检查磁盘总量
	if req.DiskTotal < 0 {
		return fmt.Errorf("invalid disk total")
	}

	// 检查端口范围
	if req.Port < 1 || req.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	// 检查capabilities是否为空
	if len(req.Capabilities) == 0 {
		return fmt.Errorf("at least one capability is required")
	}

	// 检查capabilities是否包含有效值
	validCapabilities := map[string]bool{
		"port_scan":          true,
		"service_scan":       true,
		"vulnerability_scan": true,
		"vuln_scan":          true, // 添加测试用例中使用的简写形式
		"web_scan":           true,
		"network_scan":       true,
	}

	for _, capability := range req.Capabilities {
		if !validCapabilities[capability] {
			return fmt.Errorf("invalid capability")
		}
	}

	return nil
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
