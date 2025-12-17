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
 */
package agent

import (
	"time"

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

	// Capability (ScanType) Management
	GetAllScanTypes() ([]*agentModel.ScanType, error)
	UpdateScanType(scanType *agentModel.ScanType) error

	// Agent 性能指标分析支撑 - 只读聚合的基础（返回当前快照集）
	GetAllMetrics() ([]*agentModel.AgentMetrics, error)                                               // 获取所有Agent的最新快照（单表全量）
	GetMetricsSince(since time.Time) ([]*agentModel.AgentMetrics, error)                              // 获取指定时间窗口内的快照（timestamp >= since）
	GetMetricsByAgentIDs(agentIDs []string) ([]*agentModel.AgentMetrics, error)                       // 按AgentID集合过滤获取快照
	GetMetricsByAgentIDsSince(agentIDs []string, since time.Time) ([]*agentModel.AgentMetrics, error) // 按AgentID集合+时间窗口过滤获取快照

	// Agent 能力管理 - 能力是Agent自己属性,需要结合Agent实际情况(Agent需要有自检能力的方法),不同于标签
	IsValidCapabilityId(capability string) bool                 // 判断能力ID是否有效
	IsValidCapabilityByName(capability string) bool             // 判断能力名称是否有效
	GetTagIDsByScanTypeNames(names []string) ([]uint64, error)  // 新增：根据能力名称获取TagID
	GetTagIDsByScanTypeIDs(ids []string) ([]uint64, error)      // 新增：根据能力ID获取TagID
	AddCapability(agentID string, capabilityID string) error    // 添加Agent能力
	RemoveCapability(agentID string, capabilityID string) error // 移除Agent能力
	HasCapability(agentID string, capabilityID string) bool     // 判断Agent是否有指定能力
	GetCapabilities(agentID string) []string                    // 获取Agent所有能力ID列表

	// Agent 标签管理
	IsValidTagId(tag string) bool                          // 判断标签ID是否有效
	IsValidTagByName(tag string) bool                      // 判断标签名称是否有效
	AddTag(agentID string, tagID string) error             // 添加Agent标签
	RemoveTag(agentID string, tagID string) error          // 移除Agent标签
	HasTag(agentID string, tagID string) bool              // 判断Agent是否有指定标签
	GetTags(agentID string) []string                       // 获取Agent所有标签ID列表
	GetAgentIDsByTagIDs(tagIDs []uint64) ([]string, error) // 根据标签ID获取AgentID列表

	// Agent 分组管理
	// (已移除 AgentGroup 相关功能，改用 Tag 系统)

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
