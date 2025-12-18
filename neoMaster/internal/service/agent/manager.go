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

	// Agent任务支持管理 (替代能力管理)
	IsValidTaskSupportId(taskID string) bool                              // 判断任务支持ID是否有效
	IsValidTaskSupportByName(taskName string) bool                        // 判断任务支持名称是否有效
	AddAgentTaskSupport(req *agentModel.AgentTaskSupportRequest) error    // 添加Agent任务支持
	RemoveAgentTaskSupport(req *agentModel.AgentTaskSupportRequest) error // 移除Agent任务支持
	GetAgentTaskSupport(agentID string) ([]string, error)                 // 获取Agent任务支持

	// System Bootstrap & Sync
	BootstrapSystemTags(ctx context.Context) error // 初始化Agent管理相关的系统预设标签骨架
	SyncScanTypesToTags(ctx context.Context) error // 同步ScanType到系统标签
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
		TaskSupport:      agent.TaskSupport,
		Feature:          agent.Feature,
		Tags:             nil, // Tags 字段已移除，此处设为nil，后续应通过TagService获取
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

	// 检查TaskSupport是否为空
	if len(req.TaskSupport) == 0 {
		return fmt.Errorf("at least one task support (capability) is required")
	}

	// 检查TaskSupport是否包含有效值 - 委托给Repository层验证
	// 允许 Agent 上传字符串标识 (Key/Name)，如果数据库中不存在，则视为无效或仅记录
	// 这里不再强制校验 ID 是否存在，改为在注册逻辑中尝试匹配 Name -> ID
	// for _, id := range req.TaskSupport {
	// 	if !s.agentRepo.IsValidTaskSupportId(id) {
	// 		// 暂时只记录警告，或者返回错误。
	// 		// return fmt.Errorf("invalid capability/task_support id: %s", id)
	// 	}
	// }

	// 检查Feature的有效性 (长度和数量限制)
	if len(req.Feature) > 50 {
		return fmt.Errorf("too many features (limit 50)")
	}
	for _, f := range req.Feature {
		if len(f) > 64 {
			return fmt.Errorf("feature string too long (limit 64 chars): %s", f)
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
		TaskSupport:   req.TaskSupport, // 新增字段：TaskSupport (对应 ScanType)
		Feature:       req.Feature,     // 新增字段：Feature (备用)
		Remark:        req.Remark,
		Status:        agentModel.AgentStatusOnline,
		GRPCToken:     generateGRPCToken(),
		TokenExpiry:   time.Now().Add(24 * time.Hour), // Token 24小时后过期
		LastHeartbeat: time.Now(),
	}

	if err1 := s.agentRepo.Create(newAgent); err1 != nil {
		logger.LogBusinessError(err1, "", 0, "", "service.agent.manager.RegisterAgent", "", map[string]interface{}{
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
	// 修改逻辑：Agent 上传的是字符串 Key (Name)，需要转换为 TagID
	// 优先尝试作为 Name 查询，如果查不到再尝试作为 ID 查询 (兼容旧逻辑)
	tagIDs, err := s.agentRepo.GetTagIDsByTaskSupportNames(req.TaskSupport)
	if err != nil || len(tagIDs) == 0 {
		// 尝试作为 ID 查询 (兼容性处理)
		tagIDs, err = s.agentRepo.GetTagIDsByTaskSupportIDs(req.TaskSupport)
	}

	if err != nil {
		logger.LogWarn("获取TaskSupport对应的TagID失败", "", 0, "", "RegisterAgent", "GetTagIDsByTaskSupportNames/IDs", map[string]interface{}{
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
// 支持分页和过滤条件：status 状态、keyword 关键字、tags TaskSupport 功能模块
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

	// 页码 页码大小 状态 关键字 标签 任务支持
	agents, total, err := s.agentRepo.GetList(req.Page, req.PageSize, status, keyword, req.Tags, req.TaskSupport)
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
		info := convertToAgentInfo(agent)
		// 填充 Tags 信息 (保持向后兼容)
		// 遵循 "Never break userspace" 原则
		tags, err := s.GetAgentTags(agent.AgentID)
		if err == nil && len(tags) > 0 {
			// 将 SysTag 对象转换回前端习惯的 []string 列表
			tagNames := make([]string, len(tags))
			for i, t := range tags {
				tagNames[i] = t.Name
			}
			info.Tags = tagNames
		} else {
			info.Tags = []string{}
		}
		agentInfos = append(agentInfos, info)
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

	info := convertToAgentInfo(agent)
	// 填充 Tags 信息 (保持向后兼容)
	// 遵循 "Never break userspace" 原则
	tags, err := s.GetAgentTags(agentID)
	if err == nil && len(tags) > 0 {
		tagNames := make([]string, len(tags))
		for i, t := range tags {
			tagNames[i] = t.Name
		}
		info.Tags = tagNames
	} else {
		info.Tags = []string{}
	}

	return info, nil
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

// ============================================================================
// Agent 任务支持管理模块 (TaskSupport) - 新增
// ============================================================================

// AddAgentTaskSupport 为Agent添加任务支持
func (s *agentManagerService) AddAgentTaskSupport(req *agentModel.AgentTaskSupportRequest) error {
	// 1. 输入验证
	if req.AgentID == "" || req.TaskSupport == "" {
		logger.Error("参数错误: AgentID或TaskSupport为空",
			"path", "AddAgentTaskSupport",
			"operation", "add_agent_task_support",
			"option", "validate_input",
			"func_name", "service.agent.manager.AddAgentTaskSupport",
			"agent_id", req.AgentID,
			"task_support", req.TaskSupport,
		)
		return fmt.Errorf("agent_id and task_support cannot be empty")
	}

	// 2. 验证任务支持ID是否有效 (TaskSupport 对应 ScanType)
	// 注意：TaskSupport 存储的是 ScanType 的 ID (数字字符串)
	if !s.IsValidTaskSupportId(req.TaskSupport) {
		logger.Error("无效的任务支持ID",
			"path", "AddAgentTaskSupport",
			"operation", "add_agent_task_support",
			"option", "validate_task_support",
			"func_name", "service.agent.manager.AddAgentTaskSupport",
			"task_support", req.TaskSupport,
		)
		return fmt.Errorf("invalid task_support id: %s", req.TaskSupport)
	}

	// 3. 调用Repository添加
	if err := s.agentRepo.AddTaskSupport(req.AgentID, req.TaskSupport); err != nil {
		logger.Error("添加Agent任务支持失败",
			"path", "AddAgentTaskSupport",
			"operation", "add_agent_task_support",
			"option", "repo.AddTaskSupport",
			"func_name", "service.agent.manager.AddAgentTaskSupport",
			"agent_id", req.AgentID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to add task support: %w", err)
	}

	logger.Info("Agent任务支持添加成功",
		"path", "AddAgentTaskSupport",
		"operation", "add_agent_task_support",
		"func_name", "service.agent.manager.AddAgentTaskSupport",
		"agent_id", req.AgentID,
		"task_support", req.TaskSupport,
	)
	return nil
}

// RemoveAgentTaskSupport 移除Agent任务支持
func (s *agentManagerService) RemoveAgentTaskSupport(req *agentModel.AgentTaskSupportRequest) error {
	// 1. 输入验证
	if req.AgentID == "" || req.TaskSupport == "" {
		return fmt.Errorf("agent_id and task_support cannot be empty")
	}

	// 2. 调用Repository移除
	if err := s.agentRepo.RemoveTaskSupport(req.AgentID, req.TaskSupport); err != nil {
		logger.Error("移除Agent任务支持失败",
			"path", "RemoveAgentTaskSupport",
			"operation", "remove_agent_task_support",
			"func_name", "service.agent.manager.RemoveAgentTaskSupport",
			"agent_id", req.AgentID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to remove task support: %w", err)
	}

	logger.Info("Agent任务支持移除成功",
		"path", "RemoveAgentTaskSupport",
		"operation", "remove_agent_task_support",
		"func_name", "service.agent.manager.RemoveAgentTaskSupport",
		"agent_id", req.AgentID,
		"task_support", req.TaskSupport,
	)
	return nil
}

// GetAgentTaskSupport 获取Agent的所有任务支持
func (s *agentManagerService) GetAgentTaskSupport(agentID string) ([]string, error) {
	// 1. 输入验证
	if agentID == "" {
		return []string{}, fmt.Errorf("agent_id cannot be empty")
	}

	// 2. 调用Repository获取
	taskSupports := s.agentRepo.GetTaskSupport(agentID)

	logger.Info("Agent任务支持列表获取成功",
		"path", "GetAgentTaskSupport",
		"operation", "get_agent_task_support",
		"func_name", "service.agent.manager.GetAgentTaskSupport",
		"agent_id", agentID,
		"count", len(taskSupports),
	)

	return taskSupports, nil
}

// IsValidTaskSupportId 判断任务支持ID是否有效
func (s *agentManagerService) IsValidTaskSupportId(taskID string) bool {
	if taskID == "" {
		return false
	}
	return s.agentRepo.IsValidTaskSupportId(taskID)
}

// IsValidTaskSupportByName 判断任务支持名称是否有效
func (s *agentManagerService) IsValidTaskSupportByName(taskName string) bool {
	if taskName == "" {
		return false
	}
	return s.agentRepo.IsValidTaskSupportByName(taskName)
}

// ==================== System Bootstrap & Sync ====================

// BootstrapSystemTags 初始化系统预设标签骨架
// 按照设计文档构建标签树:
// ROOT (ID: 1)
// ├── System (Category: 'system')
// │   ├── TaskSupport (Category: 'system')  <-- 对应 ScanType
// │   └── Feature (Category: 'system')      <-- 对应通用能力
// └── AgentGroup (Category: 'agent_group')
//
//	└── Default (Category: 'agent_group')
func (s *agentManagerService) BootstrapSystemTags(ctx context.Context) error {
	// Helper function to ensure a tag exists
	ensureTag := func(name, category string, parentID uint64, description string) (*tagSystemModel.SysTag, error) {
		// 1. 尝试按名称和父节点查找
		tag, err := s.tagService.GetTagByNameAndParent(ctx, name, parentID)
		if err != nil {
			// 假设错误不仅仅是 RecordNotFound，也可能是其他DB错误，但为了简化流程，我们假设找不到
			// 更好的做法是 tagService 提供更明确的错误或 IsNotFound 判定
		}
		if tag != nil {
			return tag, nil
		}

		// 2. 创建新标签
		newTag := &tagSystemModel.SysTag{
			Name:        name,
			Description: description,
			Category:    category,
			ParentID:    parentID,
		}
		if err := s.tagService.CreateTag(ctx, newTag); err != nil {
			return nil, err
		}
		// CreateTag 应该回填 ID
		return newTag, nil
	}

	// 1. ROOT
	rootTag, err := ensureTag("ROOT", "system", 0, "Root Tag")
	if err != nil {
		return fmt.Errorf("ensure ROOT failed: %v", err)
	}

	// 2. System
	systemTag, err := ensureTag("System", "system", rootTag.ID, "System Internal Tags")
	if err != nil {
		return fmt.Errorf("ensure System failed: %v", err)
	}

	// 3. TaskSupport
	_, err = ensureTag("TaskSupport", "system", systemTag.ID, "Agent Task Capabilities")
	if err != nil {
		return fmt.Errorf("ensure TaskSupport failed: %v", err)
	}

	// 4. AgentFeature (New) - 为未来自动打标预留
	_, err = ensureTag("AgentFeature", "system", systemTag.ID, "Agent Hardware/Software Features")
	if err != nil {
		return fmt.Errorf("ensure AgentFeature failed: %v", err)
	}

	// 5. Feature
	_, err = ensureTag("Feature", "system", systemTag.ID, "System Features")
	if err != nil {
		return fmt.Errorf("ensure Feature failed: %v", err)
	}

	// 6. AgentGroup
	agentGroupTag, err := ensureTag("AgentGroup", "agent_group", rootTag.ID, "Agent Group Root")
	if err != nil {
		return fmt.Errorf("ensure AgentGroup failed: %v", err)
	}

	// 7. Default Group
	_, err = ensureTag("Default", "agent_group", agentGroupTag.ID, "Default Agent Group")
	if err != nil {
		return fmt.Errorf("ensure Default Group failed: %v", err)
	}

	logger.LogInfo("BootstrapSystemTags success", "", 0, "", "BootstrapSystemTags", "completed", nil)
	return nil
}

// SyncScanTypesToTags 同步ScanType到系统标签
// 确保每个 ScanType 都在 "System/TaskSupport" 下有一个对应的 Tag
func (s *agentManagerService) SyncScanTypesToTags(ctx context.Context) error {
	// 1. 确保骨架存在
	if err := s.BootstrapSystemTags(ctx); err != nil {
		return err
	}

	// 2. 定位 TaskSupport 标签
	// 路径: ROOT -> System -> TaskSupport
	// 为准确起见，我们按层级查找
	rootTag, err := s.tagService.GetTagByNameAndParent(ctx, "ROOT", 0)
	if err != nil || rootTag == nil {
		return fmt.Errorf("tag 'ROOT' not found")
	}
	systemTag, err := s.tagService.GetTagByNameAndParent(ctx, "System", rootTag.ID)
	if err != nil || systemTag == nil {
		return fmt.Errorf("tag 'System' not found")
	}
	taskSupportTag, err := s.tagService.GetTagByNameAndParent(ctx, "TaskSupport", systemTag.ID)
	if err != nil || taskSupportTag == nil {
		return fmt.Errorf("tag 'TaskSupport' not found")
	}

	// 3. 获取所有 ScanTypes
	scanTypes, err := s.agentRepo.GetAllScanTypes()
	if err != nil {
		return fmt.Errorf("failed to get all scan types: %v", err)
	}

	// 4. 遍历处理
	for _, st := range scanTypes {
		needsUpdate := false

		// 4.1 检查是否已关联 Tag
		if st.TagID != 0 {
			tag, err := s.tagService.GetTag(ctx, st.TagID)
			if err == nil && tag != nil {
				// 检查父节点是否正确 (可选，如果想强制移动到 TaskSupport 下)
				if tag.ParentID != taskSupportTag.ID {
					// 修正父节点
					tag.ParentID = taskSupportTag.ID
					_ = s.tagService.UpdateTag(ctx, tag)
				}
				// 检查名称同步
				if tag.Name != st.Name {
					tag.Name = st.Name
					_ = s.tagService.UpdateTag(ctx, tag)
				}
				continue
			}
			st.TagID = 0
			needsUpdate = true
		}

		// 4.2 查找或创建 Tag (在 TaskSupport 下)
		existingTag, err := s.tagService.GetTagByNameAndParent(ctx, st.Name, taskSupportTag.ID)
		if err == nil && existingTag != nil {
			st.TagID = existingTag.ID
			needsUpdate = true
		} else {
			newTag := &tagSystemModel.SysTag{
				Name:        st.Name,
				Description: st.Description,
				Category:    "system", // ScanType 属于 system 分类
				ParentID:    taskSupportTag.ID,
			}
			if err := s.tagService.CreateTag(ctx, newTag); err != nil {
				logger.LogError(err, "", 0, "", "SyncScanTypesToTags", "CreateTag", map[string]interface{}{
					"scan_type": st.Name,
				})
				continue
			}
			st.TagID = newTag.ID
			needsUpdate = true
		}

		// 4.3 更新 ScanType
		if needsUpdate {
			if err := s.agentRepo.UpdateScanType(st); err != nil {
				logger.LogError(err, "", 0, "", "SyncScanTypesToTags", "UpdateScanType", map[string]interface{}{
					"scan_type": st.Name,
					"tag_id":    st.TagID,
				})
			}
		}
	}

	return nil
}
