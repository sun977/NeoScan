/**
 * @title: AgentRepository
 * @author: Sun977
 * @date: 2025.10.14
 * @description:仓库基础定义与构造函数：保持接口与结构体集中，便于文件拆分后的引用一致性
 * @func: Agent仓库接口定义
 * - AgentRepository 接口
 * - agentRepository 结构体以及构造函数
 * @note: 具体方法实现拆分在其他文件中：
 * - base.go 基础数据操作
 * - metrics.go 性能指标操作
 * - capability.go 能力操作
 * - tag.go 标签操作
 * - group.go 分组操作
 */
package agent

import (
	"gorm.io/gorm"

	agentModel "neomaster/internal/model/agent"
)

// AgentRepository Agent仓库接口定义 [定义接口层供上层调用，然后底下实现这些接口]
// 定义Agent数据访问的核心方法
type AgentRepository interface {
	// Agent 基础数据操作
	Create(agentData *agentModel.Agent) error
	GetByID(agentID string) (*agentModel.Agent, error)
	GetByHostname(hostname string) (*agentModel.Agent, error)
	GetByHostnameAndPort(hostname string, port int) (*agentModel.Agent, error) // 根据主机名和端口获取Agent
	Update(agentData *agentModel.Agent) error
	Delete(agentID string) error
	// Agent 查询操作
	GetList(page, pageSize int, status *agentModel.AgentStatus, keyword *string, tags []string, capabilities []string) ([]*agentModel.Agent, int64, error)
	GetByStatus(status agentModel.AgentStatus) ([]*agentModel.Agent, error)

	// Agent 状态和心跳管理
	UpdateStatus(agentID string, status agentModel.AgentStatus) error
	UpdateLastHeartbeat(agentID string) error

	// Agent 性能指标管理 - 直接操作agent_metrics表
	CreateMetrics(metrics *agentModel.AgentMetrics) error
	GetLatestMetrics(agentID string) (*agentModel.AgentMetrics, error)
	UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error
	GetMetricsList(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetrics, int64, error) // 性能指标批量查询（分页 + 过滤）

	// Agent 能力管理 - 能力是Agent自己属性,需要结合Agent实际情况(Agent需要有自检能力的方法),不同于标签
	IsValidCapabilityId(capability string) bool                 // 判断能力ID是否有效
	IsValidCapabilityByName(capability string) bool             // 判断能力名称是否有效
	AddCapability(agentID string, capabilityID string) error    // 添加Agent能力
	RemoveCapability(agentID string, capabilityID string) error // 移除Agent能力
	HasCapability(agentID string, capabilityID string) bool     // 判断Agent是否有指定能力
	GetCapabilities(agentID string) []string                    // 获取Agent所有能力ID列表

	// Agent 标签管理
	IsValidTagId(tag string) bool                 // 判断标签ID是否有效
	IsValidTagByName(tag string) bool             // 判断标签名称是否有效
	AddTag(agentID string, tagID string) error    // 添加Agent标签
	RemoveTag(agentID string, tagID string) error // 移除Agent标签
	HasTag(agentID string, tagID string) bool     // 判断Agent是否有指定标签
	GetTags(agentID string) []string              // 获取Agent所有标签ID列表

	// Agent 分组管理
	IsValidGroupId(groupID string) bool                                                                                   // 判断分组ID是否有效
	IsValidGroupName(groupName string) bool                                                                               // 判断分组名称是否有效
	IsAgentInGroup(agentID string, groupID string) bool                                                                   // 判断Agent是否在分组中
	GetGroupByGID(groupID string) (*agentModel.AgentGroup, error)                                                         // 获取分组详情
	GetGroupList(page, pageSize int, tags []string, status int, keywords string) ([]*agentModel.AgentGroup, int64, error) // 获取所有分组列表（分页 + 过滤）
	CreateGroup(group *agentModel.AgentGroup) error                                                                       // 创建Agent分组
	UpdateGroup(groupID string, group *agentModel.AgentGroup) error                                                       // 更新Agent分组
	DeleteGroup(groupID string) error                                                                                     // 删除Agent分组
	SetGroupStatus(groupID string, status int) error                                                                      // 设置分组状态
	AddAgentToGroup(agentID string, groupID string) error                                                                 // 将Agent添加到分组
	RemoveAgentFromGroup(agentID string, groupID string) error                                                            // 从分组中移除Agent
    GetAgentsInGroup(page, pageSize int, groupID string) ([]*agentModel.Agent, int64, error)                              // 获取分组中的Agent列表（分页）
}

// agentRepository Agent仓库实现
type agentRepository struct {
	db *gorm.DB // 数据库连接
}

// NewAgentRepository 创建Agent仓库实例
// 注入数据库连接，专注于数据访问操作
func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &agentRepository{
		db: db,
	}
}
