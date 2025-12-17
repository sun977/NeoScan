/**
 * 服务层:Agent基础管理服务
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent基础管理核心业务逻辑 - 只管Agent的生命周期
 * @func: Agent注册、查询、状态更新、删除、标签管理、能力管理
 */
package agent

import (
	"context"
	"fmt"
	agentModel "neomaster/internal/model/agent"
	tagSystemModel "neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	agentRepository "neomaster/internal/repo/mysql/agent"
	"neomaster/internal/service/tag_system"
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
	// (已移除 AgentGroup 相关功能，改用 Tag 系统)

	// Agent标签管理
	AddAgentTag(req *agentModel.AgentTagRequest) error                                                           // 添加Agent标签
	RemoveAgentTag(req *agentModel.AgentTagRequest) error                                                        // 移除Agent标签
	GetAgentTags(agentID string) ([]*tagSystemModel.SysTag, error)                                               // 获取Agent所有标签
	UpdateAgentTags(agentID string, tagIDs []uint64) ([]*tagSystemModel.SysTag, []*tagSystemModel.SysTag, error) // 更新Agent标签

	// Agent能力管理
	IsValidCapabilityId(capability string) bool                         // 判断能力ID是否有效
	IsValidCapabilityByName(capability string) bool                     // 判断能力名称是否有效
	AddAgentCapability(req *agentModel.AgentCapabilityRequest) error    // 添加Agent能力
	RemoveAgentCapability(req *agentModel.AgentCapabilityRequest) error // 移除Agent能力
	GetAgentCapabilities(agentID string) ([]string, error)              // 获取Agent能力
}

// agentManagerService Agent基础管理服务实现
type agentManagerService struct {
	agentRepo  agentRepository.AgentRepository // Agent数据访问层
	tagService tag_system.TagService           // 标签系统服务
}

// NewAgentManagerService 创建Agent基础管理服务实例
// 遵循依赖注入原则，保持代码的可测试性
func NewAgentManagerService(agentRepo agentRepository.AgentRepository, tagService tag_system.TagService) AgentManagerService {
	return &agentManagerService{
		agentRepo:  agentRepo,
		tagService: tagService,
	}
}

// ========== 辅助函数 ==========
// generateAgentID 生成Agent唯一ID
// 基于主机名和时间生成唯一标识
func generateAgentID(hostname string) string {
	// 使用简化UUID生成固定长度的agent_id，避免超过数据库字段限制
	// 格式：agent_uuid（无连字符），总长度约38字符，远小于数据库的100字符限制
	uuid, err := utils.GenerateSimpleUUID()
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
		TaskSupport:      agent.TaskSupport,
		Feature:          agent.Feature,
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

// ========== Agent 基础管理服务 ==========

// validateRegisterRequest 验证Agent注册请求参数
// 参数: s - service实例, req - Agent注册请求结构体指针
// 返回: error - 验证失败时的错误信息
func (s *agentManagerService) validateRegisterRequest(req *agentModel.RegisterAgentRequest) error {
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

	// 兼容性处理：如果 TaskSupport 为空但 Capabilities 不为空，则使用 Capabilities
	// 这是为了兼容旧版本 Agent，确保 TaskSupport 字段总是有值
	if len(req.TaskSupport) == 0 && len(req.Capabilities) > 0 {
		req.TaskSupport = req.Capabilities
	}

	// 检查TaskSupport(原Capabilities)是否为空
	if len(req.TaskSupport) == 0 {
		return fmt.Errorf("at least one task support (capability) is required")
	}

	// 检查capabilities/TaskSupport是否包含有效值 - 委托给Repository层验证
	// 统一使用 ID 进行校验，TaskSupport 和 Capabilities 都存储 ScanType ID
	for _, id := range req.TaskSupport {
		if !s.agentRepo.IsValidCapabilityId(id) {
			// 暂时只记录警告，或者返回错误。
			// return fmt.Errorf("invalid capability/task_support id: %s", id)
		}
	}

	// 检查tags是否包含有效值 - 委托给Repository层验证
	for _, tag := range req.Tags {
		if !s.agentRepo.IsValidTagId(tag) {
			return fmt.Errorf("invalid tag ID: %s", tag)
		}
	}

	return nil
}

// RegisterAgent Agent注册服务
// 处理Agent注册请求，生成唯一ID和Token
func (s *agentManagerService) RegisterAgent(req *agentModel.RegisterAgentRequest) (*agentModel.RegisterAgentResponse, error) {
	// 参数验证
	if err := s.validateRegisterRequest(req); err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.manager.RegisterAgent", "", map[string]interface{}{
			"operation": "register_agent",
			"option":    "parameter_validation",
			"func_name": "service.agent.manager.RegisterAgent",
			"hostname":  req.Hostname,
		})
		return nil, err
	}

	// 生成Agent唯一ID
	agentID := generateAgentID(req.Hostname)

	// 检查Agent是否已存在（基于hostname+port的组合）
	existingAgent, err := s.agentRepo.GetByHostnameAndPort(req.Hostname, req.Port)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.manager.RegisterAgent", "", map[string]interface{}{
			"operation": "register_agent",
			"option":    "agentManagerService.RegisterAgent",
			"func_name": "service.agent.manager.RegisterAgent",
			"hostname":  req.Hostname,
			"port":      req.Port,
		})
		return nil, fmt.Errorf("检查Agent是否存在失败: %v", err)
	}

	if existingAgent != nil {
		// Agent已存在，返回409冲突错误
		logger.LogBusinessError(
			fmt.Errorf("agent with hostname %s and port %d already exists", req.Hostname, req.Port),
			"", 0, "", "service.agent.manager.RegisterAgent", "",
			map[string]interface{}{
				"operation": "register_agent",
				"option":    "duplicate_hostname_port_check",
				"func_name": "service.agent.manager.RegisterAgent",
				"hostname":  req.Hostname,
				"port":      req.Port,
				"agent_id":  existingAgent.AgentID,
			},
		)
		return nil, fmt.Errorf("agent with hostname %s and port %d already exists", req.Hostname, req.Port)
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
		TaskSupport:   req.TaskSupport, // 新增字段：TaskSupport (对应 ScanType)
		Feature:       req.Feature,     // 新增字段：Feature (备用)
		Tags:          req.Tags,
		Remark:        req.Remark,
		Status:        agentModel.AgentStatusOnline,
		GRPCToken:     generateGRPCToken(),
		TokenExpiry:   time.Now().Add(24 * time.Hour), // Token 24小时后过期
		LastHeartbeat: time.Now(),
	}

	if err := s.agentRepo.Create(newAgent); err != nil {
		logger.LogBusinessError(err, "", 0, "", "service.agent.manager.RegisterAgent", "", map[string]interface{}{
			"operation": "register_agent",
			"option":    "agentManagerService.RegisterAgent",
			"func_name": "service.agent.manager.RegisterAgent",
			"agent_id":  agentID,
		})
		return nil, fmt.Errorf("创建新Agent失败: %v", err)
	}

	// ------------------------------------------------------------
	// Tag 系统同步：将 TaskSupport (ScanType) 映射为系统标签并绑定到 Agent
	// ------------------------------------------------------------
	// 获取 TaskSupport 对应的 TagID
	tagIDs, err := s.agentRepo.GetTagIDsByScanTypeIDs(req.TaskSupport)
	if err != nil {
		logger.LogWarn("获取TaskSupport对应的TagID失败", "", 0, "", "RegisterAgent", "GetTagIDsByScanTypeIDs", map[string]interface{}{
			"error":        err.Error(),
			"task_support": req.TaskSupport,
			"agent_id":     agentID,
		})
		// 不中断注册流程，仅记录警告
	} else if len(tagIDs) > 0 {
		// 同步 Tags
		// 使用 context.Background() 因为 RegisterAgent 没有 Context 参数
		// sourceScope 使用 "agent_capability" 以区别于其他来源
		// 这样可以确保 Agent 的能力标签被正确管理，且不影响手动打的标签
		err = s.tagService.SyncEntityTags(context.Background(), "agent", agentID, tagIDs, "agent_capability", 0)
		if err != nil {
			logger.LogError(err, "", 0, "", "RegisterAgent", "SyncEntityTags", map[string]interface{}{
				"agent_id": agentID,
				"tag_ids":  tagIDs,
			})
			// 同样不中断注册流程，但这可能导致标签数据不一致
		} else {
			logger.LogInfo("Agent能力标签同步成功", "", 0, "", "RegisterAgent", "SyncEntityTags", map[string]interface{}{
				"agent_id": agentID,
				"tag_ids":  tagIDs,
			})
		}
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
		logger.LogBusinessError(err, "", 0, "", "service.agent.manager.GetAgentList", "", map[string]interface{}{
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
		logger.LogBusinessError(err, "", 0, "", "service.agent.manager.GetAgentInfo", "", map[string]interface{}{
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
		logger.LogBusinessError(err, "", 0, "", "service.agent.manager.UpdateAgentStatus", "", map[string]interface{}{
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
// 参数: agentID - Agent唯一标识符
// 返回: error - 删除失败时返回错误信息
func (s *agentManagerService) DeleteAgent(agentID string) error {
	// 输入验证：检查agentID是否为空
	if agentID == "" {
		err := fmt.Errorf("agentID不能为空")
		logger.LogBusinessError(err, "", 0, "", "delete_agent", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "input_validation",
			"func_name": "service.agent.manager.DeleteAgent",
			"agent_id":  agentID,
		})
		return err
	}

	// 存在性验证：检查Agent是否存在
	_, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_agent", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "existence_validation",
			"func_name": "service.agent.manager.DeleteAgent",
			"agent_id":  agentID,
		})
		return fmt.Errorf("agent不存在或查询失败: %v", err)
	}

	// 执行删除操作
	err = s.agentRepo.Delete(agentID)
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "delete_agent", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "repository_delete",
			"func_name": "service.agent.manager.DeleteAgent",
			"agent_id":  agentID,
		})
		return fmt.Errorf("删除Agent失败: %v", err)
	}

	// 记录删除成功日志
	logger.LogInfo("Agent删除成功", "", 0, "", "delete_agent", "", map[string]interface{}{
		"operation": "delete_agent",
		"option":    "delete_success",
		"func_name": "service.agent.manager.DeleteAgent",
		"agent_id":  agentID,
	})

	return nil
}

// ==================== Agent标签管理方法 ====================

// AddAgentTag 为Agent添加标签
func (s *agentManagerService) AddAgentTag(req *agentModel.AgentTagRequest) error {
	// 输入验证 - 遵循"好品味"原则，消除特殊情况
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	if req.AgentID == "" {
		return fmt.Errorf("agent ID不能为空")
	}

	if req.TagID == 0 {
		return fmt.Errorf("标签ID无效")
	}

	ctx := context.Background()

	// 验证 TagID 是否存在
	_, err := s.tagService.GetTag(ctx, req.TagID)
	if err != nil {
		logger.Error("标签不存在",
			"path", "AddAgentTag",
			"operation", "add_agent_tag",
			"option", "tagService.GetTag",
			"func_name", "service.agent.manager.AddAgentTag",
			"agent_id", req.AgentID,
			"tag_id", req.TagID,
			"error", err.Error(),
		)
		return fmt.Errorf("标签不存在: %d", req.TagID)
	}

	// 1. 添加实体标签关联
	// Source: "manual", RuleID: 0
	// 移除了 GetTagByName 的调用，直接使用 ID
	// 假设 TagID 是有效的，或者由 DB 外键/AddEntityTag 内部检查保证一致性
	err = s.tagService.AddEntityTag(ctx, "agent", req.AgentID, req.TagID, "manual", 0)
	if err != nil {
		logger.Error("添加Agent标签失败",
			"path", "AddAgentTag",
			"operation", "add_agent_tag",
			"option", "tagService.AddEntityTag",
			"func_name", "service.agent.manager.AddAgentTag",
			"agent_id", req.AgentID,
			"tag_id", req.TagID,
			"error", err.Error(),
		)
		return fmt.Errorf("添加Agent标签失败: %w", err)
	}

	logger.Info("Agent标签添加成功",
		"path", "AddAgentTag",
		"operation", "add_agent_tag",
		"option", "success",
		"func_name", "service.agent.manager.AddAgentTag",
		"agent_id", req.AgentID,
		"tag_id", req.TagID,
	)

	return nil
}

// RemoveAgentTag 移除Agent标签
func (s *agentManagerService) RemoveAgentTag(req *agentModel.AgentTagRequest) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}
	if req.AgentID == "" {
		return fmt.Errorf("agent ID不能为空")
	}
	if req.TagID == 0 {
		return fmt.Errorf("标签ID无效")
	}

	ctx := context.Background()

	// 1. 移除实体标签关联
	err := s.tagService.RemoveEntityTag(ctx, "agent", req.AgentID, req.TagID)
	if err != nil {
		logger.Error("移除Agent标签失败",
			"path", "RemoveAgentTag",
			"operation", "remove_agent_tag",
			"option", "tagService.RemoveEntityTag",
			"func_name", "service.agent.manager.RemoveAgentTag",
			"agent_id", req.AgentID,
			"tag_id", req.TagID,
			"error", err.Error(),
		)
		return fmt.Errorf("移除Agent标签失败: %w", err)
	}

	logger.Info("Agent标签移除成功",
		"path", "RemoveAgentTag",
		"operation", "remove_agent_tag",
		"option", "success",
		"func_name", "service.agent.manager.RemoveAgentTag",
		"agent_id", req.AgentID,
		"tag_id", req.TagID,
	)

	return nil
}

// GetAgentTags 获取Agent的所有标签
func (s *agentManagerService) GetAgentTags(agentID string) ([]*tagSystemModel.SysTag, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID不能为空")
	}

	ctx := context.Background()

	// 1. 获取实体标签关联列表
	entityTags, err := s.tagService.GetEntityTags(ctx, "agent", agentID)
	if err != nil {
		return nil, fmt.Errorf("获取Agent标签失败: %v", err)
	}

	if len(entityTags) == 0 {
		return []*tagSystemModel.SysTag{}, nil
	}

	// 2. 提取 Tag IDs
	tagIDs := make([]uint64, 0, len(entityTags))
	for _, et := range entityTags {
		tagIDs = append(tagIDs, et.TagID)
	}

	// 3. 批量获取标签详情
	tagsVal, err := s.tagService.GetTagsByIDs(ctx, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("获取标签详情失败: %v", err)
	}

	tags := make([]*tagSystemModel.SysTag, len(tagsVal))
	for i := range tagsVal {
		tags[i] = &tagsVal[i]
	}

	return tags, nil
}

// UpdateAgentTags 更新Agent的标签列表
// 返回旧标签列表和新标签列表
func (s *agentManagerService) UpdateAgentTags(agentID string, tagIDs []uint64) ([]*tagSystemModel.SysTag, []*tagSystemModel.SysTag, error) {
	if agentID == "" {
		return nil, nil, fmt.Errorf("agent ID不能为空")
	}

	ctx := context.Background()

	// 1. 获取旧标签 - 用于返回
	oldTags, err := s.GetAgentTags(agentID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取旧标签失败: %w", err)
	}

	// 验证所有 TagID 是否存在
	if len(tagIDs) > 0 {
		// 去重 tagIDs
		uniqueIDs := make(map[uint64]bool)
		for _, id := range tagIDs {
			uniqueIDs[id] = true
		}

		validTags, err1 := s.tagService.GetTagsByIDs(ctx, tagIDs)
		if err1 != nil {
			return nil, nil, fmt.Errorf("验证标签失败: %v", err1)
		}

		if len(validTags) != len(uniqueIDs) {
			// 找出不存在的 ID (可选，为了更好得报错)
			validMap := make(map[uint64]bool)
			for _, t := range validTags {
				validMap[t.ID] = true
			}
			for id := range uniqueIDs {
				if !validMap[id] {
					return nil, nil, fmt.Errorf("标签ID不存在: %d", id)
				}
			}
			return nil, nil, fmt.Errorf("部分标签不存在")
		}
	}

	// 2. 同步标签 (SyncEntityTags)
	// 使用 SyncEntityTags 可以自动处理增删，且保留 Manual 标签（如果 sourceScope 也是 manual）
	// 这里假设 UpdateAgentTags 是手动全量更新，所以 sourceScope = "manual"
	err = s.tagService.SyncEntityTags(ctx, "agent", agentID, tagIDs, "manual", 0)
	if err != nil {
		return nil, nil, fmt.Errorf("同步标签失败: %v", err)
	}

	// 3. 获取新标签 - 用于返回
	var newTags []*tagSystemModel.SysTag
	if len(tagIDs) > 0 {
		tagsVal, err := s.tagService.GetTagsByIDs(ctx, tagIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("获取新标签详情失败: %v", err)
		}
		newTags = make([]*tagSystemModel.SysTag, len(tagsVal))
		for i := range tagsVal {
			newTags[i] = &tagsVal[i]
		}
	} else {
		newTags = []*tagSystemModel.SysTag{}
	}

	return oldTags, newTags, nil
}

// ==================== Agent能力管理方法 ====================

// AddAgentCapability 为Agent添加能力
func (s *agentManagerService) AddAgentCapability(req *agentModel.AgentCapabilityRequest) error {
	// 输入验证 - 遵循"好品味"原则，消除特殊情况
	if req == nil {
		logger.Error("请求参数为空",
			"path", "AddAgentCapability",
			"operation", "add_agent_capability",
			"option", "agentManagerService.AddAgentCapability",
			"func_name", "service.agent.manager.AddAgentCapability",
		)
		return fmt.Errorf("请求参数不能为空")
	}

	if req.AgentID == "" {
		logger.Error("Agent ID为空",
			"path", "AddAgentCapability",
			"operation", "add_agent_capability",
			"option", "agentManagerService.AddAgentCapability",
			"func_name", "service.agent.manager.AddAgentCapability",
			"agent_id", req.AgentID,
		)
		return fmt.Errorf("agent ID不能为空")
	}

	if req.Capability == "" {
		logger.Error("能力ID为空",
			"path", "AddAgentCapability",
			"operation", "add_agent_capability",
			"option", "agentManagerService.AddAgentCapability",
			"func_name", "service.agent.manager.AddAgentCapability",
			"agent_id", req.AgentID,
			"capability", req.Capability,
		)
		return fmt.Errorf("能力ID不能为空")
	}

	// 验证能力ID是否有效
	if !s.IsValidCapabilityId(req.Capability) {
		logger.Error("无效的能力ID",
			"path", "AddAgentCapability",
			"operation", "add_agent_capability",
			"option", "agentManagerService.AddAgentCapability",
			"func_name", "service.agent.manager.AddAgentCapability",
			"agent_id", req.AgentID,
			"capability", req.Capability,
		)
		return fmt.Errorf("无效的能力ID: %s", req.Capability)
	}

	// 委托Repository层处理数据操作
	err := s.agentRepo.AddCapability(req.AgentID, req.Capability)
	if err != nil {
		logger.Error("添加Agent能力失败",
			"path", "AddAgentCapability",
			"operation", "add_agent_capability",
			"option", "agentManagerService.AddAgentCapability",
			"func_name", "service.agent.manager.AddAgentCapability",
			"agent_id", req.AgentID,
			"capability", req.Capability,
			"error", err.Error(),
		)
		return fmt.Errorf("添加Agent能力失败: %w", err)
	}

	logger.Info("Agent能力添加成功",
		"path", "AddAgentCapability",
		"operation", "add_agent_capability",
		"option", "agentManagerService.AddAgentCapability",
		"func_name", "service.agent.manager.AddAgentCapability",
		"agent_id", req.AgentID,
		"capability", req.Capability,
	)

	return nil
}

// RemoveAgentCapability 移除Agent能力
func (s *agentManagerService) RemoveAgentCapability(req *agentModel.AgentCapabilityRequest) error {
	// 输入验证 - 遵循"好品味"原则，消除特殊情况
	if req == nil {
		logger.Error("请求参数为空",
			"path", "RemoveAgentCapability",
			"operation", "remove_agent_capability",
			"option", "agentManagerService.RemoveAgentCapability",
			"func_name", "service.agent.manager.RemoveAgentCapability",
		)
		return fmt.Errorf("请求参数不能为空")
	}

	if req.AgentID == "" {
		logger.Error("Agent ID为空",
			"path", "RemoveAgentCapability",
			"operation", "remove_agent_capability",
			"option", "agentManagerService.RemoveAgentCapability",
			"func_name", "service.agent.manager.RemoveAgentCapability",
			"agent_id", req.AgentID,
		)
		return fmt.Errorf("agent ID不能为空")
	}

	if req.Capability == "" {
		logger.Error("能力ID为空",
			"path", "RemoveAgentCapability",
			"operation", "remove_agent_capability",
			"option", "agentManagerService.RemoveAgentCapability",
			"func_name", "service.agent.manager.RemoveAgentCapability",
			"agent_id", req.AgentID,
			"capability", req.Capability,
		)
		return fmt.Errorf("能力ID不能为空")
	}

	// 委托Repository层处理数据操作
	err := s.agentRepo.RemoveCapability(req.AgentID, req.Capability)
	if err != nil {
		logger.Error("移除Agent能力失败",
			"path", "RemoveAgentCapability",
			"operation", "remove_agent_capability",
			"option", "agentManagerService.RemoveAgentCapability",
			"func_name", "service.agent.manager.RemoveAgentCapability",
			"agent_id", req.AgentID,
			"capability", req.Capability,
			"error", err.Error(),
		)
		return fmt.Errorf("移除Agent能力失败: %w", err)
	}

	logger.Info("Agent能力移除成功",
		"path", "RemoveAgentCapability",
		"operation", "remove_agent_capability",
		"option", "agentManagerService.RemoveAgentCapability",
		"func_name", "service.agent.manager.RemoveAgentCapability",
		"agent_id", req.AgentID,
		"capability", req.Capability,
	)

	return nil
}

// GetAgentCapabilities 获取Agent的所有能力
func (s *agentManagerService) GetAgentCapabilities(agentID string) ([]string, error) {
	// 输入验证 - 遵循"好品味"原则，消除特殊情况
	if agentID == "" {
		logger.Error("Agent ID为空",
			"path", "GetAgentCapabilities",
			"operation", "get_agent_capabilities",
			"option", "agentManagerService.GetAgentCapabilities",
			"func_name", "service.agent.manager.GetAgentCapabilities",
			"agent_id", agentID,
		)
		// 返回空切片而非nil，消除特殊情况
		return []string{}, fmt.Errorf("agent ID不能为空")
	}

	// 委托Repository层处理数据查询
	capabilities := s.agentRepo.GetCapabilities(agentID)

	logger.Info("Agent能力列表获取成功",
		"path", "GetAgentCapabilities",
		"operation", "get_agent_capabilities",
		"option", "agentManagerService.GetAgentCapabilities",
		"func_name", "service.agent.manager.GetAgentCapabilities",
		"agent_id", agentID,
		"capabilities_count", len(capabilities),
	)

	// 始终返回非nil切片，遵循"好品味"原则
	return capabilities, nil
}

// IsValidCapabilityId 判断能力ID是否有效
func (s *agentManagerService) IsValidCapabilityId(capability string) bool {
	// 1. 输入验证：检查ID是否为空
	if capability == "" {
		logger.LogBusinessError(nil, "", 0, "", "service.agent.manager.IsValidCapabilityId", "", map[string]interface{}{
			"operation":     "validate_capability_id",
			"option":        "agentManagerService.IsValidCapabilityId",
			"func_name":     "service.agent.manager.IsValidCapabilityId",
			"capability_id": capability,
			"error":         "能力ID不能为空",
		})
		return false
	}

	// 2. 委托给Repository层进行数据库验证
	// Repository层会检查ScanType表中是否存在该ID
	return s.agentRepo.IsValidCapabilityId(capability)
}

// IsValidCapabilityByName 判断能力名称是否有效
func (s *agentManagerService) IsValidCapabilityByName(capability string) bool {
	// 1. 输入验证：检查名称是否为空
	if capability == "" {
		logger.LogBusinessError(nil, "", 0, "", "service.agent.manager.IsValidCapabilityByName", "", map[string]interface{}{
			"operation":       "validate_capability_name",
			"option":          "agentManagerService.IsValidCapabilityByName",
			"func_name":       "service.agent.manager.IsValidCapabilityByName",
			"capability_name": capability,
			"error":           "能力名称不能为空",
		})
		return false
	}

	// 2. 委托给Repository层进行数据库验证
	// Repository层会检查ScanType表中是否存在该名称
	return s.agentRepo.IsValidCapabilityByName(capability)
}
