/**
 * 服务层:Agent基础管理服务
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent基础管理核心业务逻辑，遵循"好品味"原则 - 只管Agent的生命周期
 * @func: Agent注册、查询、状态更新、删除、分组管理、打标签
 */
package agent

import (
	"fmt"
	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	agentRepository "neomaster/internal/repository/mysql/agent"
	"time"
)

// AgentManagerService Agent基础管理服务接口
// 负责Agent的基础CRUD操作和分组管理，遵循单一职责原则
type AgentManagerService interface {
	// Agent基础管理 - 核心职责
	RegisterAgent(req *agentModel.RegisterAgentRequest) (*agentModel.RegisterAgentResponse, error)
	GetAgentList(req *agentModel.GetAgentListRequest) (*agentModel.GetAgentListResponse, error)
	GetAgentInfo(agentID string) (*agentModel.AgentInfo, error)
	UpdateAgentStatus(agentID string, status agentModel.AgentStatus) error
	DeleteAgent(agentID string) error

	// Agent分组管理
	CreateAgentGroup(req *agentModel.AgentGroupCreateRequest) (*agentModel.AgentGroupResponse, error)
	GetAgentGroups() ([]*agentModel.AgentGroupResponse, error)                      // 获取所有分组
	GetAgentGroup(groupID string) (*agentModel.AgentGroupResponse, error)           // 获取指定分组
	UpdateAgentGroup(groupID string, req *agentModel.AgentGroupCreateRequest) error // 更新分组信息
	DeleteAgentGroup(groupID string) error                                          // 删除分组
	AddAgentToGroup(req *agentModel.AgentGroupMemberRequest) error
	RemoveAgentFromGroup(req *agentModel.AgentGroupMemberRequest) error
	GetGroupMembers(groupID string) ([]*agentModel.AgentInfo, error) // 获取分组成员

	// Agent标签管理
	AddAgentTag(req *agentModel.AgentTagRequest) error
	RemoveAgentTag(req *agentModel.AgentTagRequest) error
	GetAgentTags(agentID string) ([]string, error)
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

// ========== Agent分组管理功能 ==========

// CreateAgentGroup 创建Agent分组服务
func (s *agentManagerService) CreateAgentGroup(req *agentModel.AgentGroupCreateRequest) (*agentModel.AgentGroupResponse, error) {
	// TODO: 实现Agent分组创建
	logger.LogInfo("创建Agent分组", "", 0, "", "service.agent.manager.CreateAgentGroup", "", map[string]interface{}{
		"operation":  "create_agent_group",
		"option":     "agentManagerService.CreateAgentGroup",
		"func_name":  "service.agent.manager.CreateAgentGroup",
		"group_name": req.Name,
	})
	return nil, fmt.Errorf("功能暂未实现")
}

// GetAgentGroups 获取所有Agent分组服务
func (s *agentManagerService) GetAgentGroups() ([]*agentModel.AgentGroupResponse, error) {
	// TODO: 实现获取所有分组
	logger.LogInfo("获取所有Agent分组", "", 0, "", "service.agent.manager.GetAgentGroups", "", map[string]interface{}{
		"operation": "get_agent_groups",
		"option":    "agentManagerService.GetAgentGroups",
		"func_name": "service.agent.manager.GetAgentGroups",
	})
	return nil, fmt.Errorf("功能暂未实现")
}

// GetAgentGroup 获取指定Agent分组服务
func (s *agentManagerService) GetAgentGroup(groupID string) (*agentModel.AgentGroupResponse, error) {
	// TODO: 实现获取指定分组
	logger.LogInfo("获取指定Agent分组", "", 0, "", "service.agent.manager.GetAgentGroup", "", map[string]interface{}{
		"operation": "get_agent_group",
		"option":    "agentManagerService.GetAgentGroup",
		"func_name": "service.agent.manager.GetAgentGroup",
		"group_id":  groupID,
	})
	return nil, fmt.Errorf("功能暂未实现")
}

// UpdateAgentGroup 更新Agent分组服务
func (s *agentManagerService) UpdateAgentGroup(groupID string, req *agentModel.AgentGroupCreateRequest) error {
	// TODO: 实现分组信息更新
	logger.LogInfo("更新Agent分组", "", 0, "", "service.agent.manager.UpdateAgentGroup", "", map[string]interface{}{
		"operation":  "update_agent_group",
		"option":     "agentManagerService.UpdateAgentGroup",
		"func_name":  "service.agent.manager.UpdateAgentGroup",
		"group_id":   groupID,
		"group_name": req.Name,
	})
	return fmt.Errorf("功能暂未实现")
}

// DeleteAgentGroup 删除Agent分组服务
func (s *agentManagerService) DeleteAgentGroup(groupID string) error {
	// TODO: 实现分组删除
	logger.LogInfo("删除Agent分组", "", 0, "", "service.agent.manager.DeleteAgentGroup", "", map[string]interface{}{
		"operation": "delete_agent_group",
		"option":    "agentManagerService.DeleteAgentGroup",
		"func_name": "service.agent.manager.DeleteAgentGroup",
		"group_id":  groupID,
	})
	return fmt.Errorf("功能暂未实现")
}

// AddAgentToGroup 添加Agent到分组服务
func (s *agentManagerService) AddAgentToGroup(req *agentModel.AgentGroupMemberRequest) error {
	// TODO: 实现添加Agent到分组
	logger.LogInfo("添加Agent到分组", "", 0, "", "service.agent.manager.AddAgentToGroup", "", map[string]interface{}{
		"operation": "add_agent_to_group",
		"option":    "agentManagerService.AddAgentToGroup",
		"func_name": "service.agent.manager.AddAgentToGroup",
		"group_id":  req.GroupID,
		"agent_id":  req.AgentID,
	})
	return fmt.Errorf("功能暂未实现")
}

// RemoveAgentFromGroup 从分组移除Agent服务
func (s *agentManagerService) RemoveAgentFromGroup(req *agentModel.AgentGroupMemberRequest) error {
	// TODO: 实现从分组移除Agent
	logger.LogInfo("从分组移除Agent", "", 0, "", "service.agent.manager.RemoveAgentFromGroup", "", map[string]interface{}{
		"operation": "remove_agent_from_group",
		"option":    "agentManagerService.RemoveAgentFromGroup",
		"func_name": "service.agent.manager.RemoveAgentFromGroup",
		"group_id":  req.GroupID,
		"agent_id":  req.AgentID,
	})
	return fmt.Errorf("功能暂未实现")
}

// GetGroupMembers 获取分组成员服务
func (s *agentManagerService) GetGroupMembers(groupID string) ([]*agentModel.AgentInfo, error) {
	// TODO: 实现获取分组成员
	logger.LogInfo("获取分组成员", "", 0, "", "service.agent.manager.GetGroupMembers", "", map[string]interface{}{
		"operation": "get_group_members",
		"option":    "agentManagerService.GetGroupMembers",
		"func_name": "service.agent.manager.GetGroupMembers",
		"group_id":  groupID,
	})
	return nil, fmt.Errorf("功能暂未实现")
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

// ==================== Agent标签管理方法 ====================

// AddAgentTag 为Agent添加标签
func (s *agentManagerService) AddAgentTag(req *agentModel.AgentTagRequest) error {
	// 记录操作日志
	logger.Info("开始为Agent添加标签",
		"path", "AddAgentTag",
		"operation", "add_agent_tag",
		"option", "agentManagerService.AddAgentTag",
		"func_name", "service.agent.manager.AddAgentTag",
		"agent_id", req.AgentID,
		"tag", req.Tag,
	)

	// TODO: 实现Agent标签添加逻辑
	// 1. 验证Agent是否存在
	// 2. 验证标签格式和长度
	// 3. 检查标签是否已存在
	// 4. 将标签添加到数据库
	// 5. 更新Agent的标签列表

	logger.Warn("Agent标签添加功能暂未实现",
		"path", "AddAgentTag",
		"operation", "add_agent_tag",
		"option", "agentManagerService.AddAgentTag",
		"func_name", "service.agent.manager.AddAgentTag",
		"agent_id", req.AgentID,
		"tag", req.Tag,
	)

	return fmt.Errorf("Agent标签添加功能暂未实现")
}

// RemoveAgentTag 移除Agent标签
func (s *agentManagerService) RemoveAgentTag(req *agentModel.AgentTagRequest) error {
	// 记录操作日志
	logger.Info("开始移除Agent标签",
		"path", "RemoveAgentTag",
		"operation", "remove_agent_tag",
		"option", "agentManagerService.RemoveAgentTag",
		"func_name", "service.agent.manager.RemoveAgentTag",
		"agent_id", req.AgentID,
		"tag", req.Tag,
	)

	// TODO: 实现Agent标签移除逻辑
	// 1. 验证Agent是否存在
	// 2. 验证标签是否存在
	// 3. 从数据库中移除标签
	// 4. 更新Agent的标签列表

	logger.Warn("Agent标签移除功能暂未实现",
		"path", "RemoveAgentTag",
		"operation", "remove_agent_tag",
		"option", "agentManagerService.RemoveAgentTag",
		"func_name", "service.agent.manager.RemoveAgentTag",
		"agent_id", req.AgentID,
		"tag", req.Tag,
	)

	return fmt.Errorf("Agent标签移除功能暂未实现")
}

// GetAgentTags 获取Agent的所有标签
func (s *agentManagerService) GetAgentTags(agentID string) ([]string, error) {
	// 记录操作日志
	logger.Info("开始获取Agent标签列表",
		"path", "GetAgentTags",
		"operation", "get_agent_tags",
		"option", "agentManagerService.GetAgentTags",
		"func_name", "service.agent.manager.GetAgentTags",
		"agent_id", agentID,
	)

	// TODO: 实现Agent标签获取逻辑
	// 1. 验证Agent是否存在
	// 2. 从数据库中查询Agent的所有标签
	// 3. 返回标签列表

	logger.Warn("Agent标签获取功能暂未实现",
		"path", "GetAgentTags",
		"operation", "get_agent_tags",
		"option", "agentManagerService.GetAgentTags",
		"func_name", "service.agent.manager.GetAgentTags",
		"agent_id", agentID,
	)

	return nil, fmt.Errorf("Agent标签获取功能暂未实现")
}
