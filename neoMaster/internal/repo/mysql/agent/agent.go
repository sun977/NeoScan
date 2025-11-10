/**
 * Agent仓库层:Agent数据访问
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent数据访问层，专注于数据操作，不包含业务逻辑
 * @func: 单纯数据访问，遵循"好品味"原则 - 数据结构优先，消除特殊情况
 */
package agent

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
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

	// Agent 能力管理
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
	GetAgentsInGroup(groupID string) ([]*agentModel.Agent, int64, error)                                                  // 获取分组中的Agent列表（分页）

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

// Create 创建Agent（纯数据访问）
// 直接将Agent数据插入数据库，不包含业务逻辑验证
func (r *agentRepository) Create(agentData *agentModel.Agent) error {
	// 设置创建时间
	agentData.CreatedAt = time.Now()
	agentData.UpdatedAt = time.Now()

	result := r.db.Create(agentData)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.Create", "", map[string]interface{}{
			"operation": "create_agent",
			"option":    "agentRepository.Create",
			"func_name": "repo.agent.Create",
			"agent_id":  agentData.AgentID,
			"hostname":  agentData.Hostname,
		})
		return result.Error
	}

	logger.LogInfo("Agent记录创建成功", "", 0, "", "repo.agent.Create", "", map[string]interface{}{
		"operation": "create_agent",
		"option":    "agentRepository.Create",
		"func_name": "repo.agent.Create",
		"agent_id":  agentData.AgentID,
		"hostname":  agentData.Hostname,
	})

	return nil
}

// GetByID 根据AgentID获取Agent
// 参数: agentID - Agent的业务ID
// 返回: *agentModel.Agent - Agent数据, error - 错误信息
func (r *agentRepository) GetByID(agentID string) (*agentModel.Agent, error) {
	var agentData agentModel.Agent

	// 实际上是根据 agentID 查询
	result := r.db.Where("agent_id = ?", agentID).First(&agentData)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // 返回nil表示未找到，不是错误
		}
		logger.LogError(result.Error, "", 0, "", "repo.agent.GetByID", "", map[string]interface{}{
			"operation": "get_agent_by_id",
			"option":    "agentRepository.GetByID",
			"func_name": "repo.agent.GetByID",
			"agent_id":  agentID,
		})
		return nil, result.Error
	}

	return &agentData, nil
}

// GetByHostnameAndPort 根据主机名和端口获取Agent
// 参数: hostname - 主机名, port - 端口号
// 返回: *agentModel.Agent - Agent数据, error - 错误信息
// 用于支持一机多Agent部署场景，通过hostname+port唯一标识Agent
func (r *agentRepository) GetByHostnameAndPort(hostname string, port int) (*agentModel.Agent, error) {
	var agentData agentModel.Agent

	result := r.db.Where("hostname = ? AND port = ?", hostname, port).First(&agentData)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // 返回nil表示未找到，不是错误
		}
		logger.LogError(result.Error, "", 0, "", "repo.agent.GetByHostnameAndPort", "", map[string]interface{}{
			"operation": "get_agent_by_hostname_and_port",
			"option":    "agentRepository.GetByHostnameAndPort",
			"func_name": "repo.agent.GetByHostnameAndPort",
			"hostname":  hostname,
			"port":      port,
		})
		return nil, result.Error
	}

	return &agentData, nil
}

// GetByHostname 根据主机名获取Agent
// 参数: hostname - 主机名
// 返回: *agentModel.Agent - Agent数据, error - 错误信息
func (r *agentRepository) GetByHostname(hostname string) (*agentModel.Agent, error) {
	var agentData agentModel.Agent

	result := r.db.Where("hostname = ?", hostname).First(&agentData)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // 返回nil表示未找到，不是错误
		}
		logger.LogError(result.Error, "", 0, "", "repo.agent.GetByHostname", "", map[string]interface{}{
			"operation": "get_agent_by_hostname",
			"option":    "agentRepository.GetByHostname",
			"func_name": "repo.agent.GetByHostname",
			"hostname":  hostname,
		})
		return nil, result.Error
	}

	return &agentData, nil
}

// Update 更新Agent记录
// 参数: agentData - Agent数据指针
// 返回: error - 更新过程中的错误信息
func (r *agentRepository) Update(agentData *agentModel.Agent) error {
	// 设置更新时间
	agentData.UpdatedAt = time.Now()

	result := r.db.Save(agentData)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.Update", "", map[string]interface{}{
			"operation": "update_agent",
			"option":    "agentRepository.Update",
			"func_name": "repo.agent.Update",
			"agent_id":  agentData.AgentID,
			"hostname":  agentData.Hostname,
		})
		return result.Error
	}

	logger.LogInfo("Agent记录更新成功", "", 0, "", "repo.agent.Update", "", map[string]interface{}{
		"operation": "update_agent",
		"option":    "agentRepository.Update",
		"func_name": "repo.agent.Update",
		"agent_id":  agentData.AgentID,
		"hostname":  agentData.Hostname,
	})

	return nil
}

// UpdateStatus 更新Agent状态
// 参数: agentID - Agent的业务ID, status - 新状态
// 返回: error - 更新过程中的错误信息
func (r *agentRepository) UpdateStatus(agentID string, status agentModel.AgentStatus) error {
	result := r.db.Model(&agentModel.Agent{}).
		Where("agent_id = ?", agentID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.UpdateStatus", "", map[string]interface{}{
			"operation": "update_agent_status",
			"option":    "agentRepository.UpdateStatus",
			"func_name": "repo.agent.UpdateStatus",
			"agent_id":  agentID,
			"status":    string(status),
		})
		return result.Error
	}

	// 检查是否真的更新了记录
	if result.RowsAffected == 0 {
		err := fmt.Errorf("agent not found")
		logger.LogError(err, "", 0, "", "repo.agent.UpdateStatus", "", map[string]interface{}{
			"operation": "update_agent_status",
			"option":    "agentRepository.UpdateStatus",
			"func_name": "repo.agent.UpdateStatus",
			"agent_id":  agentID,
			"status":    string(status),
			"message":   "No rows affected, agent not found",
		})
		return err
	}

	logger.LogInfo("Agent状态更新成功", "", 0, "", "repo.agent.UpdateStatus", "", map[string]interface{}{
		"operation": "update_agent_status",
		"option":    "agentRepository.UpdateStatus",
		"func_name": "repo.agent.UpdateStatus",
		"agent_id":  agentID,
		"status":    string(status),
	})

	return nil
}

// UpdateLastHeartbeat 更新Agent最后心跳时间
// 参数: agentID - Agent的业务ID (同时更新 updated_at 和 last_heartbeat 字段)
// 返回: error - 更新过程中的错误信息
func (r *agentRepository) UpdateLastHeartbeat(agentID string) error {
	result := r.db.Model(&agentModel.Agent{}).
		Where("agent_id = ?", agentID).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"updated_at":     time.Now(),
		})

	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.UpdateLastHeartbeat", "", map[string]interface{}{
			"operation": "update_heartbeat",
			"option":    "agentRepository.UpdateLastHeartbeat",
			"func_name": "repo.agent.UpdateLastHeartbeat",
			"agent_id":  agentID,
		})
		return result.Error
	}

	// 检查是否真的更新了记录
	if result.RowsAffected == 0 {
		err := fmt.Errorf("agent not found")
		logger.LogError(err, "", 0, "", "repo.agent.UpdateLastHeartbeat", "", map[string]interface{}{
			"operation": "update_heartbeat",
			"option":    "agentRepository.UpdateLastHeartbeat",
			"func_name": "repo.agent.UpdateLastHeartbeat",
			"agent_id":  agentID,
			"message":   "No rows affected, agent not found",
		})
		return err
	}

	logger.LogInfo("Agent心跳时间更新成功", "", 0, "", "repo.agent.UpdateLastHeartbeat", "", map[string]interface{}{
		"operation": "update_heartbeat",
		"option":    "agentRepository.UpdateLastHeartbeat",
		"func_name": "repo.agent.UpdateLastHeartbeat",
		"agent_id":  agentID,
	})

	return nil
}

// CreateMetrics 创建Agent性能指标记录
// 参数: metrics - 性能指标数据指针
// 返回: error - 创建过程中的错误信息
func (r *agentRepository) CreateMetrics(metrics *agentModel.AgentMetrics) error {
	// 设置时间戳
	metrics.Timestamp = time.Now()

	result := r.db.Create(metrics)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.CreateMetrics", "", map[string]interface{}{
			"operation": "create_agent_metrics",
			"option":    "agentRepository.CreateMetrics",
			"func_name": "repo.agent.CreateMetrics",
			"agent_id":  metrics.AgentID,
		})
		return result.Error
	}

	logger.LogInfo("Agent性能指标创建成功", "", 0, "", "repo.agent.CreateMetrics", "", map[string]interface{}{
		"operation": "create_agent_metrics",
		"option":    "agentRepository.CreateMetrics",
		"func_name": "repo.agent.CreateMetrics",
		"agent_id":  metrics.AgentID,
	})

	return nil
}

// GetLatestMetrics 获取Agent最新性能指标
// 参数: agentID - Agent的业务ID
// 返回: *agentModel.AgentMetrics - 最新的性能指标数据, error - 查询过程中的错误信息
func (r *agentRepository) GetLatestMetrics(agentID string) (*agentModel.AgentMetrics, error) {
	var metrics agentModel.AgentMetrics
	result := r.db.Where("agent_id = ?", agentID).
		Order("timestamp DESC").
		First(&metrics)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // 没有找到记录，返回nil而不是错误
		}
		logger.LogError(result.Error, "", 0, "", "repo.agent.GetLatestMetrics", "", map[string]interface{}{
			"operation": "get_latest_metrics",
			"option":    "agentRepository.GetLatestMetrics",
			"func_name": "repo.agent.GetLatestMetrics",
			"agent_id":  agentID,
		})
		return nil, result.Error
	}

	return &metrics, nil
}

// GetMetricsList 获取所有Agent的最新性能指标（SQL分页）
// 参数: page - 页码（从1开始）, pageSize - 每页大小
// 返回: []*agentModel.AgentMetrics - 当前页的性能指标数据, int64 - 总记录数, error - 错误信息
// 设计说明：
// - agent_metrics 表采用单快照模型（每个Agent最多一条记录，AgentID唯一索引），因此无需对每个Agent做额外聚合或去重。
// - 直接对该表按 timestamp DESC 排序后进行分页，可避免一次性加载全集数据导致的内存压力与N+1查询问题。
func (r *agentRepository) GetMetricsList(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetrics, int64, error) {
	var metricsList []*agentModel.AgentMetrics
	var total int64

	// 防御性处理：页码与页大小需为正数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 构建查询
	query := r.db.Model(&agentModel.AgentMetrics{})

	// 过滤条件：work_status
	if workStatus != nil {
		query = query.Where("work_status = ?", *workStatus)
	}

	// 过滤条件：scan_type
	if scanType != nil {
		query = query.Where("scan_type = ?", *scanType)
	}

	// 过滤条件：agent_id 关键词（模糊查询）
	if keyword != nil && *keyword != "" {
		like := "%" + *keyword + "%"
		query = query.Where("agent_id LIKE ?", like)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetMetricsList", "", map[string]interface{}{
			"operation": "get_metrics_list",
			"option":    "agentRepository.GetMetricsList.count",
			"func_name": "repo.agent.GetMetricsList",
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
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据，按更新时间戳倒序，必要时可增加次级排序保证稳定性
	if err := query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&metricsList).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetMetricsList", "", map[string]interface{}{
			"operation": "get_metrics_list",
			"option":    "agentRepository.GetMetricsList.query",
			"func_name": "repo.agent.GetMetricsList",
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
		return nil, 0, err
	}

	logger.LogInfo("Agent性能指标分页获取成功", "", 0, "", "repo.agent.GetMetricsList", "", map[string]interface{}{
		"operation": "get_metrics_list",
		"option":    "agentRepository.GetMetricsList",
		"func_name": "repo.agent.GetMetricsList",
		"count":     len(metricsList),
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

	return metricsList, total, nil
}

// UpdateAgentMetrics 更新Agent性能指标 - 实现upsert逻辑（存在则更新，不存在则创建）
// 参数: agentID - Agent的业务ID, metrics - 性能指标数据指针
// 返回: error - 更新过程中的错误信息
func (r *agentRepository) UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error {
	// 防御性编程：检查输入参数
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	// 设置AgentID和时间戳
	metrics.AgentID = agentID
	metrics.Timestamp = time.Now()

	// 查找是否已存在该Agent的性能指标记录
	var existingMetrics agentModel.AgentMetrics
	result := r.db.Where("agent_id = ?", agentID).First(&existingMetrics)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// 不存在记录，创建新记录
			createResult := r.db.Create(metrics)
			if createResult.Error != nil {
				logger.LogError(createResult.Error, "", 0, "", "repo.agent.UpdateAgentMetrics", "", map[string]interface{}{
					"operation": "create_agent_metrics",
					"option":    "agentRepository.UpdateAgentMetrics",
					"func_name": "repo.agent.UpdateAgentMetrics",
					"agent_id":  agentID,
				})
				return createResult.Error
			}

			logger.LogInfo("Agent性能指标创建成功", "", 0, "", "repo.agent.UpdateAgentMetrics", "", map[string]interface{}{
				"operation": "create_agent_metrics",
				"option":    "agentRepository.UpdateAgentMetrics",
				"func_name": "repo.agent.UpdateAgentMetrics",
				"agent_id":  agentID,
			})
		} else {
			// 查询出错
			logger.LogError(result.Error, "", 0, "", "repo.agent.UpdateAgentMetrics", "", map[string]interface{}{
				"operation": "query_agent_metrics",
				"option":    "agentRepository.UpdateAgentMetrics",
				"func_name": "repo.agent.UpdateAgentMetrics",
				"agent_id":  agentID,
			})
			return result.Error
		}
	} else {
		// 存在记录，更新现有记录
		// 构建更新字段映射，避免空指针访问
		updateFields := map[string]interface{}{
			"cpu_usage":          metrics.CPUUsage,
			"memory_usage":       metrics.MemoryUsage,
			"disk_usage":         metrics.DiskUsage,
			"network_bytes_sent": metrics.NetworkBytesSent,
			"network_bytes_recv": metrics.NetworkBytesRecv,
			"active_connections": metrics.ActiveConnections,
			"running_tasks":      metrics.RunningTasks,
			"completed_tasks":    metrics.CompletedTasks,
			"failed_tasks":       metrics.FailedTasks,
			"work_status":        metrics.WorkStatus,
			"scan_type":          metrics.ScanType,
			"timestamp":          metrics.Timestamp,
		}

		// 安全处理PluginStatus字段 这是一个 JSON 字段 {"nmap": {"pid": 12345, "status": "running"}, "nuclei": {"status": "idle"}}
		if metrics.PluginStatus != nil {
			updateFields["plugin_status"] = metrics.PluginStatus
		}

		updateResult := r.db.Model(&existingMetrics).Updates(updateFields)

		if updateResult.Error != nil {
			logger.LogError(updateResult.Error, "", 0, "", "repo.agent.UpdateAgentMetrics", "", map[string]interface{}{
				"operation": "update_agent_metrics",
				"option":    "agentRepository.UpdateAgentMetrics",
				"func_name": "repo.agent.UpdateAgentMetrics",
				"agent_id":  agentID,
			})
			return updateResult.Error
		}

		logger.LogInfo("Agent性能指标更新成功", "", 0, "", "repo.agent.UpdateAgentMetrics", "", map[string]interface{}{
			"operation": "update_agent_metrics",
			"option":    "agentRepository.UpdateAgentMetrics",
			"func_name": "repo.agent.UpdateAgentMetrics",
			"agent_id":  agentID,
		})
	}

	return nil
}

// Delete 删除Agent记录
// 参数: agentID - Agent的业务ID
// 返回: error - 删除过程中的错误信息
func (r *agentRepository) Delete(agentID string) error {
	result := r.db.Where("agent_id = ?", agentID).Delete(&agentModel.Agent{})
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.Delete", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "agentRepository.Delete",
			"func_name": "repo.agent.Delete",
			"agent_id":  agentID,
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.LogInfo("Agent记录不存在，无需删除", "", 0, "", "repo.agent.Delete", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "agentRepository.Delete",
			"func_name": "repo.agent.Delete",
			"agent_id":  agentID,
		})
		return nil // 不存在也不算错误
	}

	logger.LogInfo("Agent记录删除成功", "", 0, "", "repo.agent.Delete", "", map[string]interface{}{
		"operation": "delete_agent",
		"option":    "agentRepository.Delete",
		"func_name": "repo.agent.Delete",
		"agent_id":  agentID,
	})

	return nil
}

// GetList 获取Agent列表
// 参数: page - 页码, pageSize - 每页大小, status - 状态过滤, keyword - 关键字过滤, tags - 标签过滤, capabilities - 功能模块过滤
// 返回: []*agentModel.Agent - Agent列表, int64 - 总数量, error - 错误信息
func (r *agentRepository) GetList(page, pageSize int, status *agentModel.AgentStatus, keyword *string, tags []string, capabilities []string) ([]*agentModel.Agent, int64, error) {
	var agents []*agentModel.Agent
	var total int64

	// 构建查询条件
	query := r.db.Model(&agentModel.Agent{})

	// 状态过滤 - 仅支持单个状态值过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 关键字模糊过滤
	if keyword != nil {
		query = query.Where("agent_id LIKE ? OR hostname LIKE ? OR ip_address LIKE ?", "%"+*keyword+"%", "%"+*keyword+"%", "%"+*keyword+"%")
	}

	// 标签过滤 - 使用JSON查询精确匹配标签ID
	// 支持多个标签的AND逻辑：Agent必须包含所有指定的标签ID
	if len(tags) > 0 {
		for _, tag := range tags {
			// 使用JSON_CONTAINS函数精确匹配JSON数组中的字符串值
			// 注意：tag需要用双引号包围，因为JSON数组中存储的是字符串
			query = query.Where("JSON_CONTAINS(tags, ?)", `"`+tag+`"`)
		}
	}

	// Capabilities 过滤 - 使用JSON查询精确匹配功能模块ID
	// 支持多个功能模块的AND逻辑：Agent必须包含所有指定的功能模块ID
	if len(capabilities) > 0 {
		for _, capability := range capabilities {
			// 使用JSON_CONTAINS函数精确匹配JSON数组中的字符串值
			// 注意：capability需要用双引号包围，因为JSON数组中存储的是字符串
			query = query.Where("JSON_CONTAINS(capabilities, ?)", `"`+capability+`"`)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetList", "", map[string]interface{}{
			"operation": "get_agent_list",
			"option":    "agentRepository.GetList",
			"func_name": "repo.agent.GetList",
		})
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据
	if err := query.Offset(offset).Limit(pageSize).Find(&agents).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetList", "", map[string]interface{}{
			"operation": "get_agent_list",
			"option":    "agentRepository.GetList",
			"func_name": "repo.agent.GetList",
		})
		return nil, 0, err
	}

	logger.LogInfo("Agent列表获取成功", "", 0, "", "repo.agent.GetList", "", map[string]interface{}{
		"operation": "get_agent_list",
		"option":    "agentRepository.GetList",
		"func_name": "repo.agent.GetList",
		"count":     len(agents),
		"total":     total,
	})

	return agents, total, nil
}

// GetByStatus 根据状态获取Agent列表
// 参数: status - Agent状态
// 返回: []*agentModel.Agent - Agent列表, error - 错误信息
func (r *agentRepository) GetByStatus(status agentModel.AgentStatus) ([]*agentModel.Agent, error) {
	var agents []*agentModel.Agent

	result := r.db.Where("status = ?", status).Find(&agents)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.GetByStatus", "", map[string]interface{}{
			"operation": "get_agents_by_status",
			"option":    "agentRepository.GetByStatus",
			"func_name": "repo.agent.GetByStatus",
			"status":    string(status),
		})
		return nil, result.Error
	}

	return agents, nil
}

// ==================== Agent能力管理方法 ====================

// IsValidCapabilityId 判断能力ID是否有效 - agent_scan_type
// 参数: capabilityId - 能力ID（字符串形式的数字ID）
// 返回: bool - 是否有效
func (r *agentRepository) IsValidCapabilityId(capabilityId string) bool {
	var count int64
	// 直接查询ScanType表的id字段，验证该ID是否存在且激活
	result := r.db.Model(&agentModel.ScanType{}).Where("id = ? AND is_active = ?", capabilityId, 1).Count(&count)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.IsValidCapabilityId", "", map[string]interface{}{
			"operation":    "validate_capability_id",
			"option":       "agentRepository.IsValidCapabilityId",
			"func_name":    "repo.agent.IsValidCapabilityId",
			"capabilityId": capabilityId,
		})
		return false
	}
	return count > 0
}

// IsValidCapabilityByName 判断能力名称是否有效 - agent_scan_type
// 参数: capability - 能力名称
// 返回: bool - 是否有效
func (r *agentRepository) IsValidCapabilityByName(capability string) bool {
	var count int64
	// 查询ScanType表的name字段，验证该名称是否存在且激活
	result := r.db.Model(&agentModel.ScanType{}).Where("name = ? AND is_active = ?", capability, true).Count(&count)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.IsValidCapabilityByName", "", map[string]interface{}{
			"operation":  "validate_capability_name",
			"option":     "agentRepository.IsValidCapabilityByName",
			"func_name":  "repo.agent.IsValidCapabilityByName",
			"capability": capability,
		})
		return false
	}
	return count > 0
}

// AddCapability 为Agent添加能力
// 参数: agentID - Agent ID, capabilityID - 能力ID
// 返回: error - 错误信息
func (r *agentRepository) AddCapability(agentID string, capabilityID string) error {
	// 1. 验证能力ID是否有效
	if !r.IsValidCapabilityId(capabilityID) {
		logger.LogError(nil, "", 0, "", "repo.agent.AddCapability", "", map[string]interface{}{
			"operation":    "add_capability",
			"option":       "agentRepository.AddCapability",
			"func_name":    "repo.agent.AddCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
			"error":        "invalid_capability_id",
		})
		return fmt.Errorf("无效的能力ID: %s", capabilityID)
	}

	// 2. 检查Agent是否存在并获取完整信息
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddCapability", "", map[string]interface{}{
			"operation":    "add_capability",
			"option":       "agentRepository.AddCapability",
			"func_name":    "repo.agent.AddCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
		})
		return fmt.Errorf("agent不存在: %s", agentID)
	}

	// 3. 检查能力是否已存在（避免重复添加）
	if agent.HasCapability(capabilityID) {
		logger.LogInfo("Agent已具有该能力，跳过添加", "", 0, "", "repo.agent.AddCapability", "", map[string]interface{}{
			"operation":    "add_capability",
			"option":       "agentRepository.AddCapability",
			"func_name":    "repo.agent.AddCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
			"status":       "already_exists",
		})
		return nil // 已存在不算错误，直接返回成功
	}

	// 4. 使用Agent模型方法添加能力
	agent.AddCapability(capabilityID)

	// 5. 更新数据库中的capabilities字段
	if err := r.db.Model(&agent).Select("capabilities").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddCapability", "", map[string]interface{}{
			"operation":    "add_capability",
			"option":       "agentRepository.AddCapability",
			"func_name":    "repo.agent.AddCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
		})
		return fmt.Errorf("更新Agent能力失败: %v", err)
	}

	logger.LogInfo("Agent能力添加成功", "", 0, "", "repo.agent.AddCapability", "", map[string]interface{}{
		"operation":    "add_capability",
		"option":       "agentRepository.AddCapability",
		"func_name":    "repo.agent.AddCapability",
		"agentID":      agentID,
		"capabilityID": capabilityID,
		"status":       "success",
	})

	return nil
}

// RemoveCapability 移除Agent能力
// 参数: agentID - Agent ID, capabilityID - 能力ID
// 返回: error - 错误信息
func (r *agentRepository) RemoveCapability(agentID string, capabilityID string) error {
	// 1. 检查Agent是否存在并获取完整信息
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.RemoveCapability", "", map[string]interface{}{
			"operation":    "remove_capability",
			"option":       "agentRepository.RemoveCapability",
			"func_name":    "repo.agent.RemoveCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
		})
		return fmt.Errorf("agent不存在: %s", agentID)
	}

	// 2. 检查能力是否存在（幂等性处理）
	if !agent.HasCapability(capabilityID) {
		logger.LogInfo("Agent不具有该能力，跳过移除", "", 0, "", "repo.agent.RemoveCapability", "", map[string]interface{}{
			"operation":    "remove_capability",
			"option":       "agentRepository.RemoveCapability",
			"func_name":    "repo.agent.RemoveCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
			"status":       "not_exists",
		})
		return nil // 不存在不算错误，直接返回成功
	}

	// 3. 使用Agent模型方法移除能力
	agent.RemoveCapability(capabilityID)

	// 4. 更新数据库中的capabilities字段
	if err := r.db.Model(&agent).Select("capabilities").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.RemoveCapability", "", map[string]interface{}{
			"operation":    "remove_capability",
			"option":       "agentRepository.RemoveCapability",
			"func_name":    "repo.agent.RemoveCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
		})
		return fmt.Errorf("更新Agent能力失败: %v", err)
	}

	logger.LogInfo("Agent能力移除成功", "", 0, "", "repo.agent.RemoveCapability", "", map[string]interface{}{
		"operation":    "remove_capability",
		"option":       "agentRepository.RemoveCapability",
		"func_name":    "repo.agent.RemoveCapability",
		"agentID":      agentID,
		"capabilityID": capabilityID,
		"status":       "success",
	})

	return nil
}

// HasCapability 判断Agent是否具有指定能力
// 参数: agentID - Agent ID, capabilityID - 能力ID
// 返回: bool - 是否具有能力
func (r *agentRepository) HasCapability(agentID string, capabilityID string) bool {
	// 1. 输入验证 - 避免无效查询
	if agentID == "" || capabilityID == "" {
		logger.LogInfo("输入参数为空，返回false", "", 0, "", "repo.agent.HasCapability", "", map[string]interface{}{
			"operation":    "has_capability",
			"option":       "agentRepository.HasCapability",
			"func_name":    "repo.agent.HasCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
			"status":       "invalid_input",
		})
		return false
	}

	// 2. 查询Agent信息（获取Agent的完整信息，包括capabilities字段）
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.HasCapability", "", map[string]interface{}{
			"operation":    "has_capability",
			"option":       "agentRepository.HasCapability",
			"func_name":    "repo.agent.HasCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
		})
		return false
	}

	// 3. 使用Agent模型方法检查能力（在内存中进行检查）
	return agent.HasCapability(capabilityID)
}

// GetCapabilities 获取Agent的能力列表
// 参数: agentID - Agent ID
// 返回: []string - 能力列表
func (r *agentRepository) GetCapabilities(agentID string) []string {
	// 1. 输入验证 - 避免无效查询
	if agentID == "" {
		logger.LogInfo("Agent ID为空，返回空列表", "", 0, "", "repo.agent.GetCapabilities", "", map[string]interface{}{
			"operation": "get_capabilities",
			"option":    "agentRepository.GetCapabilities",
			"func_name": "repo.agent.GetCapabilities",
			"agentID":   agentID,
			"status":    "invalid_input",
		})
		return []string{} // 返回空切片而不是nil
	}

	// 2. 检查Agent是否存在并获取完整信息
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetCapabilities", "", map[string]interface{}{
			"operation": "get_capabilities",
			"option":    "agentRepository.GetCapabilities",
			"func_name": "repo.agent.GetCapabilities",
			"agentID":   agentID,
		})
		return []string{} // 返回空切片而不是nil
	}

	// 3. 返回Agent的能力列表（确保不返回nil）
	if agent.Capabilities == nil {
		return []string{}
	}
	return agent.Capabilities
}

// ==================== Agent标签管理方法 ====================

// IsValidTagId 判断标签ID是否有效 - agent_tag_type
// 参数: tagId - 标签ID
// 返回: bool - 是否有效
func (r *agentRepository) IsValidTagId(tagId string) bool {
	var count int64
	result := r.db.Model(&agentModel.TagType{}).Where("id = ?", tagId).Count(&count)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.IsValidTagId", "", map[string]interface{}{
			"operation": "validate_tag_id",
			"option":    "agentRepository.IsValidTagId",
			"func_name": "repo.agent.IsValidTagId",
			"tagId":     tagId,
		})
		return false
	}
	return count > 0
}

// IsValidTagByName 判断标签名称是否有效 - agent_tag_type
// 参数: tag - 标签名称
// 返回: bool - 是否有效
func (r *agentRepository) IsValidTagByName(tag string) bool {
	var count int64
	result := r.db.Model(&agentModel.TagType{}).Where("JSON_CONTAINS(tags, ?)", `"`+tag+`"`).Count(&count)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.IsValidTagByName", "", map[string]interface{}{
			"operation": "validate_tag_name",
			"option":    "agentRepository.IsValidTagByName",
			"func_name": "repo.agent.IsValidTagByName",
			"tag":       tag,
		})
		return false
	}
	return count > 0
}

// AddTag 添加标签 - agents
// 参数: agentID - Agent ID, tagID - 标签ID
// 返回: error - 错误信息
func (r *agentRepository) AddTag(agentID string, tagID string) error {
	// 1. 验证标签ID是否有效
	if !r.IsValidTagId(tagID) {
		logger.LogError(nil, "", 0, "", "repo.agent.AddTag", "", map[string]interface{}{
			"operation": "add_tag",
			"option":    "agentRepository.AddTag",
			"func_name": "repo.agent.AddTag",
			"agentID":   agentID,
			"tagID":     tagID,
			"error":     "invalid_tag_id",
		})
		return fmt.Errorf("无效的标签ID: %s", tagID)
	}

	// 2. 检查Agent是否存在并获取完整信息
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddTag", "", map[string]interface{}{
			"operation": "add_tag",
			"option":    "agentRepository.AddTag",
			"func_name": "repo.agent.AddTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return fmt.Errorf("agent不存在: %s", agentID)
	}

	// 3. 检查标签是否已存在（避免重复添加）
	if agent.HasTag(tagID) {
		logger.LogInfo("Agent已具有该标签，跳过添加", "", 0, "", "repo.agent.AddTag", "", map[string]interface{}{
			"operation": "add_tag",
			"option":    "agentRepository.AddTag",
			"func_name": "repo.agent.AddTag",
			"agentID":   agentID,
			"tagID":     tagID,
			"status":    "already_exists",
		})
		return nil // 已存在不算错误，直接返回成功
	}

	// 4. 使用Agent模型方法添加标签
	agent.AddTag(tagID)

	// 5. 更新数据库中的tags字段
	if err := r.db.Model(&agent).Select("tags").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddTag", "", map[string]interface{}{
			"operation": "add_tag",
			"option":    "agentRepository.AddTag",
			"func_name": "repo.agent.AddTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return fmt.Errorf("更新Agent标签失败: %v", err)
	}

	logger.LogInfo("Agent标签添加成功", "", 0, "", "repo.agent.AddTag", "", map[string]interface{}{
		"operation": "add_tag",
		"option":    "agentRepository.AddTag",
		"func_name": "repo.agent.AddTag",
		"agentID":   agentID,
		"tagID":     tagID,
		"status":    "success",
	})

	return nil
}

// RemoveTag 移除标签 - agents
// 参数: agentID - Agent ID, tagID - 标签ID
// 返回: error - 错误信息
func (r *agentRepository) RemoveTag(agentID string, tagID string) error {
	// 1. 验证标签ID不为空
	if tagID == "" {
		logger.LogError(nil, "", 0, "", "repo.agent.RemoveTag", "", map[string]interface{}{
			"operation": "remove_tag",
			"option":    "agentRepository.RemoveTag",
			"func_name": "repo.agent.RemoveTag",
			"agentID":   agentID,
			"tagID":     tagID,
			"error":     "empty_tag_id",
		})
		return fmt.Errorf("标签ID不能为空")
	}

	// 2. 检查Agent是否存在并获取完整信息
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.RemoveTag", "", map[string]interface{}{
			"operation": "remove_tag",
			"option":    "agentRepository.RemoveTag",
			"func_name": "repo.agent.RemoveTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return fmt.Errorf("agent不存在: %s", agentID)
	}

	// 3. 检查标签是否存在（幂等性处理）
	if !agent.HasTag(tagID) {
		logger.LogInfo("Agent未具有该标签，跳过移除", "", 0, "", "repo.agent.RemoveTag", "", map[string]interface{}{
			"operation": "remove_tag",
			"option":    "agentRepository.RemoveTag",
			"func_name": "repo.agent.RemoveTag",
			"agentID":   agentID,
			"tagID":     tagID,
			"status":    "not_exists",
		})
		return nil // 不存在不算错误，直接返回成功
	}

	// 4. 使用Agent模型方法移除标签
	agent.RemoveTag(tagID)

	// 5. 更新数据库中的tags字段
	if err := r.db.Model(&agent).Select("tags").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.RemoveTag", "", map[string]interface{}{
			"operation": "remove_tag",
			"option":    "agentRepository.RemoveTag",
			"func_name": "repo.agent.RemoveTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return fmt.Errorf("更新Agent标签失败: %v", err)
	}

	logger.LogInfo("Agent标签移除成功", "", 0, "", "repo.agent.RemoveTag", "", map[string]interface{}{
		"operation": "remove_tag",
		"option":    "agentRepository.RemoveTag",
		"func_name": "repo.agent.RemoveTag",
		"agentID":   agentID,
		"tagID":     tagID,
		"status":    "success",
	})

	return nil
}

// HasTag 判断Agent是否具有指定标签
// 参数: agentID - Agent ID, tagID - 标签ID
// 返回: bool - 是否具有标签
func (r *agentRepository) HasTag(agentID string, tagID string) bool {
	// 1. 输入验证 - 避免无效查询
	if agentID == "" || tagID == "" {
		logger.LogInfo("输入参数为空，返回false", "", 0, "", "repo.agent.HasTag", "", map[string]interface{}{
			"operation": "has_tag",
			"option":    "agentRepository.HasTag",
			"func_name": "repo.agent.HasTag",
			"agentID":   agentID,
			"tagID":     tagID,
			"status":    "invalid_input",
		})
		return false
	}

	// 2. 查询Agent信息
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.HasTag", "", map[string]interface{}{
			"operation": "has_tag",
			"option":    "agentRepository.HasTag",
			"func_name": "repo.agent.HasTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return false
	}

	// 3. 使用Agent模型方法检查标签
	return agent.HasTag(tagID)
}

// GetTags 获取Agent的标签列表
// 参数: agentID - Agent ID
// 返回: []string - 标签ID列表，始终返回非nil切片
func (r *agentRepository) GetTags(agentID string) []string {
	// 1. 输入验证：避免空值查询
	if agentID == "" {
		logger.LogError(nil, "", 0, "", "repo.agent.GetTags", "", map[string]interface{}{
			"operation": "get_tags",
			"option":    "agentRepository.GetTags",
			"func_name": "repo.agent.GetTags",
			"agentID":   agentID,
			"error":     "agentID不能为空",
		})
		return []string{} // 返回空切片而非nil
	}

	// 2. 检查Agent是否存在并获取完整信息
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetTags", "", map[string]interface{}{
			"operation": "get_tags",
			"option":    "agentRepository.GetTags",
			"func_name": "repo.agent.GetTags",
			"agentID":   agentID,
		})
		return []string{} // 返回空切片而非nil
	}

	// 3. 处理Agent.Tags为nil的情况，确保返回值一致性
	if agent.Tags == nil {
		return []string{}
	}

	// 4. 返回Agent的标签ID列表
	return agent.Tags
}

// ==================== Agent 分组管理方法 ====================

// IsValidGroupId 验证分组ID是否有效
func (r *agentRepository) IsValidGroupId(groupID string) bool {
	// 直接查询数据库中是否存在该分组ID
	var count int64
	if err := r.db.Model(&agentModel.AgentGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.IsValidGroupId", "", map[string]interface{}{
			"operation": "is_valid_group_id",
			"option":    "agentRepository.IsValidGroupId",
			"func_name": "repo.agent.IsValidGroupId",
			"groupID":   groupID,
		})
		return false
	}
	return count > 0
}

// IsValidGroupName 验证分组名称是否可用
func (r *agentRepository) IsValidGroupName(groupName string) bool {
	// 直接查询数据库中是否存在该分组名称 agent_groups
	var count int64
	if err := r.db.Model(&agentModel.AgentGroup{}).Where("name = ?", groupName).Count(&count).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.IsValidGroupName", "", map[string]interface{}{
			"operation": "is_valid_group_name",
			"option":    "agentRepository.IsValidGroupName",
			"func_name": "repo.agent.IsValidGroupName",
			"groupName": groupName,
		})
		return false
	}
	return count > 0
}

// IsAgentInGroup 验证Agent是否在指定分组中
func (r *agentRepository) IsAgentInGroup(agentID string, groupID string) bool {
	// 直接查询数据库中该Agent是否在指定分组中 agent_group_members
	var count int64
	if err := r.db.Model(&agentModel.AgentGroupMember{}).Where("agent_id = ? AND group_id = ?", agentID, groupID).Count(&count).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.IsAgentInGroup", "", map[string]interface{}{
			"operation": "is_agent_in_group",
			"option":    "agentRepository.IsAgentInGroup",
			"func_name": "repo.agent.IsAgentInGroup",
			"agentID":   agentID,
			"groupID":   groupID,
		})
		return false
	}
	return count > 0
}

// GetGroupByGID 根据业务分组ID获取分组详情
// 注意：不强制过滤 status，调用方可以自行决定是否只接受激活分组
func (r *agentRepository) GetGroupByGID(groupID string) (*agentModel.AgentGroup, error) {
	var group agentModel.AgentGroup
	// 直接按业务ID查询分组
	if err := r.db.Where("group_id = ?", groupID).First(&group).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetGroupByGID", "", map[string]interface{}{
			"operation": "get_group_by_gid",
			"option":    "agentRepository.GetGroupByGID.find_group",
			"func_name": "repo.agent.GetGroupByGID",
			"group_id":  groupID,
		})
		return nil, err
	}

	// 成功日志：输出关键字段（is_system/status）便于审计
	logger.LogInfo("got group by gid", "", 0, "", "repo.agent.GetGroupByGID", "", map[string]interface{}{
		"operation":       "get_group_by_gid",
		"option":          "agentRepository.GetGroupByGID.success",
		"func_name":       "repo.agent.GetGroupByGID",
		"group_id":        groupID,
		"group_name":      group.Name,
		"group_status":    group.Status,
		"is_system_group": group.IsSystem,
	})
	return &group, nil
}

// GetGroupList 获取所有分组列表（分页 + 过滤）
// 过滤字段: tags, status(1启用0禁用), keywords(模糊匹配：group_id, name, description)
func (r *agentRepository) GetGroupList(page, pageSize int, tags []string, status int, keywords string) ([]*agentModel.AgentGroup, int64, error) {
	// - agent_groups 表为分组信息的单表存储，Tags 使用 JSON 数组存储（[]string）。
	// - 过滤策略：
	//   1) tags 采用 AND 逻辑：当提供多个标签时，分组必须同时包含这些标签；（tag没有指定专门的标签表，标签是字符串）
	//   2) status 采用 0/1 逻辑：0 表示禁用，1 表示启用。
	//   3) keywords 采用 OR 模糊查询：匹配 group_id、name、description 三个字段。
	// - 分页策略：标准 SQL 分页（Offset + Limit），并以 updated_at DESC, id DESC 进行稳定排序。

	var groups []*agentModel.AgentGroup
	var total int64

	// 防御性处理：页码与页大小需为正数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 构建查询
	query := r.db.Model(&agentModel.AgentGroup{})

	// 标签过滤（AND 逻辑）：JSON_CONTAINS 精确匹配字符串元素
	if len(tags) > 0 {
		for _, tag := range tags {
			// 注意：JSON_CONTAINS 期望第二参数为合法 JSON 文本，这里为字符串需加双引号
			query = query.Where("JSON_CONTAINS(tags, ?)", `"`+tag+`"`)
		}
	}

	// 关键字模糊过滤（OR 逻辑）：匹配 group_id、name、description
	if keywords != "" {
		like := "%" + keywords + "%"
		query = query.Where("group_id LIKE ? OR name LIKE ? OR description LIKE ?", like, like, like)
	}

	// 新增状态过滤：当入参 status 为 0/1 时按参数过滤；
	// 设计原因：支持根据调用方传入的状态过滤，同时保持向后兼容（不传或非法值时仍只展示激活分组）。
	if status == 0 || status == 1 {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetGroupList", "", map[string]interface{}{
			"operation": "get_group_list",
			"option":    "agentRepository.GetGroupList.count",
			"func_name": "repo.agent.GetGroupList",
			"page":      page,
			"page_size": pageSize,
			"tags":      tags,
			"keywords":  keywords,
			"status":    status,
		})
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据，按更新时间倒序，次级以 id 保证稳定性
	if err := query.Order("updated_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&groups).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetGroupList", "", map[string]interface{}{
			"operation": "get_group_list",
			"option":    "agentRepository.GetGroupList.query",
			"func_name": "repo.agent.GetGroupList",
			"page":      page,
			"page_size": pageSize,
			"tags":      tags,
			"keywords":  keywords,
			"status":    status,
		})
		return nil, 0, err
	}

	// 成功日志
	logger.LogInfo("Agent分组列表获取成功", "", 0, "", "repo.agent.GetGroupList", "", map[string]interface{}{
		"operation": "get_group_list",
		"option":    "agentRepository.GetGroupList",
		"func_name": "repo.agent.GetGroupList",
		"count":     len(groups),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"tags":      tags,
		"keywords":  keywords,
		"status":    status,
	})

	return groups, total, nil
}

// CreateGroup 创建Agent分组
func (r *agentRepository) CreateGroup(group *agentModel.AgentGroup) error {
	// 入参校验
	if group == nil {
		err := fmt.Errorf("CreateGroup: group 不能为空")
		logger.LogError(err, "", 0, "", "repo.agent.CreateGroup", "", map[string]interface{}{
			"operation": "create_group",
			"option":    "agentRepository.CreateGroup",
			"func_name": "repo.agent.CreateGroup",
			"group":     nil,
		})
		return err
	}

	// 防御性设置：
	// 1) is_system 不允许通过普通创建接口设为系统分组，统一由系统事务（如默认分组保障）创建。
	// 2) status 只允许 0/1，其他值按默认激活(1)处理。
	// 注意：Status 为非指针类型，GORM的 default 仅在字段未显式赋值时生效，这里进行显式校正。
	if group.IsSystem {
		// 普通创建禁止创建系统分组，强制回退为非系统分组
		group.IsSystem = false
	}
	if group.Status != 0 && group.Status != 1 {
		group.Status = 1
	}

	// 直接插入 agent_groups 表
	if err := r.db.Create(group).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.CreateGroup", "", map[string]interface{}{
			"operation": "create_group",
			"option":    "agentRepository.CreateGroup",
			"func_name": "repo.agent.CreateGroup",
			"group":     group,
		})
		return err
	}

	// 成功日志
	logger.LogInfo("Agent分组创建成功", "", 0, "", "repo.agent.CreateGroup", "", map[string]interface{}{
		"operation": "create_group",
		"option":    "agentRepository.CreateGroup",
		"func_name": "repo.agent.CreateGroup",
		"group":     group,
	})

	return nil
}

// UpdateGroup 更新Agent分组信息
// 采用“定向部分更新”：按业务唯一键 group_id 精确定位并更新安全字段，避免 Save 带来的主键依赖与全字段覆盖风险。
// Save 方法会更新所有字段,包括未被修改的字段(save 如果存在则更新不存在则创建,依赖主键)。Update 方法只更新被修改的字段,不依赖主键，这里使用 Update 方法。
func (r *agentRepository) UpdateGroup(groupID string, group *agentModel.AgentGroup) error {
	// 1. 入参校验（调用方传入 groupID，避免依赖 group.GroupID）
	if groupID == "" {
		err := fmt.Errorf("UpdateGroup: groupID 不能为空")
		logger.LogError(err, "", 0, "", "repo.agent.UpdateGroup", "", map[string]interface{}{
			"operation": "update_group",
			"option":    "agentRepository.UpdateGroup",
			"func_name": "repo.agent.UpdateGroup",
			"group_id":  groupID,
			"group":     group,
		})
		return err
	}
	if group == nil {
		err := fmt.Errorf("UpdateGroup: group 不能为空")
		logger.LogError(err, "", 0, "", "repo.agent.UpdateGroup", "", map[string]interface{}{
			"operation": "update_group",
			"option":    "agentRepository.UpdateGroup",
			"func_name": "repo.agent.UpdateGroup",
			"group_id":  groupID,
			"group":     nil,
		})
		return err
	}

	// 2. 先检查分组是否存在（按 group_id）——不存在直接返回 gorm.ErrRecordNotFound
	var existing agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&existing).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.UpdateGroup", "", map[string]interface{}{
			"operation": "update_group",
			"option":    "agentRepository.UpdateGroup",
			"func_name": "repo.agent.UpdateGroup",
			"group_id":  groupID,
			"group":     group,
		})
		return err
	}

	// 3. 组装允许更新的字段（只更新安全字段）
	updateFields := map[string]interface{}{
		"description": group.Description, // 描述
		"updated_at":  time.Now(),        // 更新时间戳
	}

	// 名称更新：对系统分组进行限制（不可修改名称），普通分组可更新
	if !existing.IsSystem {
		updateFields["name"] = group.Name // 名称（若有唯一约束可能触发重复键错误）
	} else if group.Name != "" && group.Name != existing.Name {
		// 系统分组尝试改名，直接拒绝并返回错误
		err := fmt.Errorf("UpdateGroup: 系统分组禁止修改名称(%s -> %s)", existing.Name, group.Name)
		logger.LogError(err, "", 0, "", "repo.agent.UpdateGroup", "", map[string]interface{}{
			"operation": "update_group",
			"option":    "agentRepository.UpdateGroup.prevent_rename_system",
			"func_name": "repo.agent.UpdateGroup",
			"group_id":  groupID,
			"old_name":  existing.Name,
			"new_name":  group.Name,
		})
		return err
	}

	// Tags 为 JSON 字段，仅当传入非 nil 时更新，避免清空数据库已有 JSON
	if group.Tags != nil {
		updateFields["tags"] = group.Tags
	}

	// 状态更新：仅当传入与当前不一致时更新；系统分组禁止禁用（status 只能为 1）
	if group.Status == 0 || group.Status == 1 {
		if group.Status != existing.Status {
			if existing.IsSystem && group.Status == 0 {
				err := fmt.Errorf("UpdateGroup: 系统分组禁止禁用(status 只能为1)")
				logger.LogError(err, "", 0, "", "repo.agent.UpdateGroup", "", map[string]interface{}{
					"operation": "update_group",
					"option":    "agentRepository.UpdateGroup.prevent_disable_system",
					"func_name": "repo.agent.UpdateGroup",
					"group_id":  groupID,
					"status":    group.Status,
				})
				return err
			}
			updateFields["status"] = group.Status
		}
	}

	// 4. 定向更新并检查受影响行数
	result := r.db.Model(&agentModel.AgentGroup{}).
		Where("group_id = ?", groupID).
		Updates(updateFields)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.UpdateGroup", "", map[string]interface{}{
			"operation":    "update_group",
			"option":       "agentRepository.UpdateGroup",
			"func_name":    "repo.agent.UpdateGroup",
			"group_id":     groupID,
			"group":        group,
			"updateFields": updateFields,
		})
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 极端情况下（并发删除导致行不存在），也按不存在返回
		err := gorm.ErrRecordNotFound
		logger.LogError(err, "", 0, "", "repo.agent.UpdateGroup", "", map[string]interface{}{
			"operation":    "update_group",
			"option":       "agentRepository.UpdateGroup",
			"func_name":    "repo.agent.UpdateGroup",
			"group_id":     groupID,
			"group":        group,
			"updateFields": updateFields,
		})
		return err
	}

	// 5. 成功日志
	logger.LogInfo("Agent分组更新成功", "", 0, "", "repo.agent.UpdateGroup", "", map[string]interface{}{
		"operation":     "update_group",
		"option":        "agentRepository.UpdateGroup",
		"func_name":     "repo.agent.UpdateGroup",
		"group_id":      groupID,
		"group":         group,
		"updateFields":  updateFields,
		"rows_affected": result.RowsAffected,
	})

	return nil
}

// DeleteGroup 删除Agent分组
// 删除后组内的Agent自动移到默认分组
// 理论上不允许删除默认分组,但是如果默认分组被删或者不存在,程序会自动创建一个默认分组,默认分组名称固定为 'default'
func (r *agentRepository) DeleteGroup(groupID string) error {
	// 1. 入参校验
	if groupID == "" {
		err := fmt.Errorf("DeleteGroup: groupID 不能为空")
		logger.LogError(err, "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
			"operation": "delete_group",
			"option":    "agentRepository.DeleteGroup",
			"func_name": "repo.agent.DeleteGroup",
			"group_id":  groupID,
		})
		return err
	}

	// 2. 查询待删除的分组是否存在（按 group_id）
	var target agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&target).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
			"operation": "delete_group",
			"option":    "agentRepository.DeleteGroup.find_target",
			"func_name": "repo.agent.DeleteGroup",
			"group_id":  groupID,
		})
		return err
	}

	// 防御性：不允许删除系统分组 或 默认分组（通过名称或种子ID识别）
	if target.IsSystem || target.Name == "default" || target.GroupID == "default" {
		err := fmt.Errorf("DeleteGroup: 不允许删除默认分组(%s/%s)", target.GroupID, target.Name)
		logger.LogError(err, "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
			"operation":       "delete_group",
			"option":          "agentRepository.DeleteGroup.prevent_default",
			"func_name":       "repo.agent.DeleteGroup",
			"group_id":        groupID,
			"name":            target.Name,
			"target_group_id": target.GroupID,
			"is_system":       target.IsSystem,
		})
		return err
	}

	// 4. 事务处理：确保默认分组存在（不存在则创建），先迁移成员到默认分组，再删除分组，以保证外键约束下的数据一致性
	var defaultGroupIDForLog string
	txErr := r.db.Transaction(func(tx *gorm.DB) error {
		// 4.0 确保默认分组存在
		// 查找策略调整：
		// 1) 优先按 is_system = 1 查找系统分组，确保命中真正的系统默认分组；
		// 2) 若未找到，则回退按 group_id = 'default' 查找；
		// 3) 不再使用 name 作为判断依据，避免名称变更导致识别失败。
		var defaultGroup agentModel.AgentGroup
		findDefaultBySystem := tx.Where("is_system = ?", 1).First(&defaultGroup).Error
		if findDefaultBySystem != nil {
			// 回退：按 group_id 查找
			findDefaultByID := tx.Where("group_id = ?", "default").First(&defaultGroup).Error
			if findDefaultByID != nil {
				// 均未找到则创建默认分组
				newDefault := &agentModel.AgentGroup{
					GroupID:     "default",
					Name:        "default",
					Description: "默认分组(禁止删除)",
					Tags:        []string{"default"},
					IsSystem:    true, // 标记为系统分组，受保护不可删除
					Status:      1,    // 默认激活
				}
				if createErr := tx.Create(newDefault).Error; createErr != nil {
					// 并发场景可能触发唯一约束错误，记录后重查一次
					logger.LogError(createErr, "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
						"operation": "delete_group",
						"option":    "agentRepository.DeleteGroup.create_default",
						"func_name": "repo.agent.DeleteGroup",
						"group_id":  groupID,
					})
					// 重查优先按 is_system，再回退 group_id，避免名称变更造成误判
					if err2 := tx.Where("is_system = ?", 1).First(&defaultGroup).Error; err2 != nil {
						if err3 := tx.Where("group_id = ?", "default").First(&defaultGroup).Error; err3 != nil {
							return createErr
						}
					}
				} else {
					defaultGroup = *newDefault
				}
			}
		}

		// 记录默认分组ID用于最终日志
		defaultGroupIDForLog = defaultGroup.GroupID

		// 4.1 将待删除分组中的所有成员迁移到默认分组
		// 使用 INSERT IGNORE + SELECT 的批量操作，避免重复成员导致的唯一索引冲突
		// 说明：agent_group_members 的唯一键为 (agent_id, group_id)，若某 Agent 已在默认分组则忽略插入
		moveRes := tx.Exec(
			"INSERT IGNORE INTO agent_group_members (agent_id, group_id, joined_at, created_at, updated_at) "+
				"SELECT agent_id, ?, NOW(), NOW(), NOW() FROM agent_group_members WHERE group_id = ?",
			defaultGroup.GroupID, groupID,
		)
		if moveRes.Error != nil {
			logger.LogError(moveRes.Error, "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
				"operation":        "delete_group",
				"option":           "agentRepository.DeleteGroup.move_members",
				"func_name":        "repo.agent.DeleteGroup",
				"group_id":         groupID,
				"default_group_id": defaultGroup.GroupID,
			})
			return moveRes.Error
		}

		// 4.2 删除分组（agent_groups），外键约束 ON DELETE CASCADE 会同步删除原分组的成员关系
		delRes := tx.Where("group_id = ?", groupID).Delete(&agentModel.AgentGroup{})
		if delRes.Error != nil {
			logger.LogError(delRes.Error, "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
				"operation":        "delete_group",
				"option":           "agentRepository.DeleteGroup.delete_group",
				"func_name":        "repo.agent.DeleteGroup",
				"group_id":         groupID,
				"default_group_id": defaultGroup.GroupID,
			})
			return delRes.Error
		}
		if delRes.RowsAffected == 0 {
			// 极端情况下（并发删除）返回未找到
			err := gorm.ErrRecordNotFound
			logger.LogError(err, "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
				"operation":        "delete_group",
				"option":           "agentRepository.DeleteGroup.delete_group.not_found",
				"func_name":        "repo.agent.DeleteGroup",
				"group_id":         groupID,
				"default_group_id": defaultGroup.GroupID,
			})
			return err
		}

		// 4.3 成功日志（事务内只记录迁移数量）
		logger.LogInfo("分组成员迁移完成，准备提交删除事务", "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
			"operation":        "delete_group",
			"option":           "agentRepository.DeleteGroup.tx_move",
			"func_name":        "repo.agent.DeleteGroup",
			"group_id":         groupID,
			"default_group_id": defaultGroup.GroupID,
			"moved_count":      moveRes.RowsAffected,
		})
		return nil
	})
	if txErr != nil {
		return txErr
	}

	// 5. 成功日志（事务已提交）
	logger.LogInfo("Agent分组删除成功，组内成员已迁移至默认分组", "", 0, "", "repo.agent.DeleteGroup", "", map[string]interface{}{
		"operation":        "delete_group",
		"option":           "agentRepository.DeleteGroup",
		"func_name":        "repo.agent.DeleteGroup",
		"group_id":         groupID,
		"default_group_id": defaultGroupIDForLog,
	})

	return nil
}

// SetGroupStatus 设置分组状态 - 允上层使用这个方法改变分组状态(1-启用 0-禁用)
// 优化点：
// 1) 校验入参 status 仅允许 0/1；
// 2) 系统分组（is_system=1）禁止修改状态；
// 3) 若状态未变化则直接返回，避免不必要的写操作；
// 4) 完整日志记录，便于审计与排查。
func (r *agentRepository) SetGroupStatus(groupID string, status int) error {
	// 入参校验：status 仅允许 0 或 1
	if status != 0 && status != 1 {
		err := gorm.ErrInvalidData
		logger.LogError(err, "", 0, "", "repo.agent.SetGroupStatus", "", map[string]interface{}{
			"operation": "set_group_status",
			"option":    "agentRepository.SetGroupStatus.validate",
			"func_name": "repo.agent.SetGroupStatus",
			"group_id":  groupID,
			"status":    status,
			"reason":    "invalid status, only 0/1 allowed",
		})
		return err
	}

	// 查询分组
	var g agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&g).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.SetGroupStatus", "", map[string]interface{}{
			"operation": "set_group_status",
			"option":    "agentRepository.SetGroupStatus.get_group",
			"func_name": "repo.agent.SetGroupStatus",
			"group_id":  groupID,
			"status":    status,
		})
		return err
	}

	// 系统分组保护：禁止修改状态
	if g.IsSystem {
		err := gorm.ErrInvalidData
		logger.LogError(err, "", 0, "", "repo.agent.SetGroupStatus", "", map[string]interface{}{
			"operation": "set_group_status",
			"option":    "agentRepository.SetGroupStatus.protect_system_group",
			"func_name": "repo.agent.SetGroupStatus",
			"group_id":  groupID,
			"status":    status,
			"is_system": g.IsSystem,
			"reason":    "system group cannot change status",
		})
		return err
	}

	// 幂等处理：状态未变化则直接返回
	if int(g.Status) == status {
		logger.LogInfo("分组状态未变化，跳过更新", "", 0, "", "repo.agent.SetGroupStatus", "", map[string]interface{}{
			"operation": "set_group_status",
			"option":    "agentRepository.SetGroupStatus.noop",
			"func_name": "repo.agent.SetGroupStatus",
			"group_id":  groupID,
			"status":    status,
		})
		return nil
	}

	// 执行更新
	result := r.db.Model(&agentModel.AgentGroup{}).Where("group_id = ?", groupID).Update("status", status)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.SetGroupStatus", "", map[string]interface{}{
			"operation":  "set_group_status",
			"option":     "agentRepository.SetGroupStatus.update",
			"func_name":  "repo.agent.SetGroupStatus",
			"group_id":   groupID,
			"old_status": g.Status,
			"new_status": status,
		})
		return result.Error
	}

	logger.LogInfo("分组状态更新成功", "", 0, "", "repo.agent.SetGroupStatus", "", map[string]interface{}{
		"operation":  "set_group_status",
		"option":     "agentRepository.SetGroupStatus",
		"func_name":  "repo.agent.SetGroupStatus",
		"group_id":   groupID,
		"old_status": g.Status,
		"new_status": status,
	})
	return nil
}

// AddAgentToGroup 添加Agent到分组
func (r *agentRepository) AddAgentToGroup(agentID string, groupID string) error {
	// 将 Agent 加入到指定分组
	// 设计约束：
	// 1) 分组必须存在；
	// 2) 分组若为禁用状态（status=0），不允许新增成员，以避免向“冻结”分组添加数据；
	// 3) Agent 必须存在；
	// 4) 若 Agent 已在该分组中，操作幂等化，直接返回成功；
	// 5) 使用唯一键约束(agent_id, group_id)保证并发安全；

	// 1. 校验分组是否存在，并获取分组详情（含 status、is_system 等）
	var group agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&group).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddAgentToGroup", "", map[string]interface{}{
			"operation": "add_agent_to_group",
			"option":    "agentRepository.AddAgentToGroup.find_group",
			"func_name": "repo.agent.AddAgentToGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return err
	}

	// 2. 禁止向禁用分组添加成员（保证数据一致性和业务语义清晰）
	if group.Status == 0 {
		err := gorm.ErrInvalidData
		logger.LogError(err, "", 0, "", "repo.agent.AddAgentToGroup", "", map[string]interface{}{
			"operation":    "add_agent_to_group",
			"option":       "agentRepository.AddAgentToGroup.group_disabled",
			"func_name":    "repo.agent.AddAgentToGroup",
			"agent_id":     agentID,
			"group_id":     groupID,
			"group_status": group.Status,
		})
		return err
	}

	// 3. 校验 Agent 是否存在（满足外键约束前置条件）
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddAgentToGroup", "", map[string]interface{}{
			"operation": "add_agent_to_group",
			"option":    "agentRepository.AddAgentToGroup.find_agent",
			"func_name": "repo.agent.AddAgentToGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return err
	}

	// 4. 幂等校验：若已在分组中，直接返回成功
	if r.IsAgentInGroup(agentID, groupID) {
		logger.LogInfo("agent already in group, skip", "", 0, "", "repo.agent.AddAgentToGroup", "", map[string]interface{}{
			"operation": "add_agent_to_group",
			"option":    "agentRepository.AddAgentToGroup.idempotent_skip",
			"func_name": "repo.agent.AddAgentToGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return nil
	}

	// 5. 创建分组成员关系记录
	member := &agentModel.AgentGroupMember{
		AgentID:  agentID,
		GroupID:  groupID,
		JoinedAt: time.Now(),
	}

	if err := r.db.Create(member).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddAgentToGroup", "", map[string]interface{}{
			"operation": "add_agent_to_group",
			"option":    "agentRepository.AddAgentToGroup.create_member",
			"func_name": "repo.agent.AddAgentToGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return err
	}

	// 6. 成功日志
	logger.LogInfo("agent added to group", "", 0, "", "repo.agent.AddAgentToGroup", "", map[string]interface{}{
		"operation":       "add_agent_to_group",
		"option":          "agentRepository.AddAgentToGroup.success",
		"func_name":       "repo.agent.AddAgentToGroup",
		"agent_id":        agentID,
		"group_id":        groupID,
		"group_status":    group.Status,
		"is_system_group": group.IsSystem,
	})
	return nil
}

// RemoveAgentFromGroup 从分组中移除Agent
func (r *agentRepository) RemoveAgentFromGroup(agentID string, groupID string) error {
	// 从指定分组移除 Agent
	// 设计约束：
	// 1) 操作幂等化：若成员关系不存在，直接返回成功；
	// 2) 遵循外键约束：若分组或 Agent 不存在，成员关系也不存在；
	// 3) 删除使用条件删除，避免误删其他数据；

	// 1. 判断成员关系是否存在
	var count int64
	if err := r.db.Model(&agentModel.AgentGroupMember{}).
		Where("agent_id = ? AND group_id = ?", agentID, groupID).
		Count(&count).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.RemoveAgentFromGroup", "", map[string]interface{}{
			"operation": "remove_agent_from_group",
			"option":    "agentRepository.RemoveAgentFromGroup.count_member",
			"func_name": "repo.agent.RemoveAgentFromGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return err
	}

	if count == 0 {
		// 幂等返回：成员关系不存在
		logger.LogInfo("agent not in group, skip", "", 0, "", "repo.agent.RemoveAgentFromGroup", "", map[string]interface{}{
			"operation": "remove_agent_from_group",
			"option":    "agentRepository.RemoveAgentFromGroup.idempotent_skip",
			"func_name": "repo.agent.RemoveAgentFromGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return nil
	}

	// 2. 执行删除
	delRes := r.db.Where("agent_id = ? AND group_id = ?", agentID, groupID).
		Delete(&agentModel.AgentGroupMember{})
	if delRes.Error != nil {
		logger.LogError(delRes.Error, "", 0, "", "repo.agent.RemoveAgentFromGroup", "", map[string]interface{}{
			"operation": "remove_agent_from_group",
			"option":    "agentRepository.RemoveAgentFromGroup.delete_member",
			"func_name": "repo.agent.RemoveAgentFromGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return delRes.Error
	}

	logger.LogInfo("agent removed from group", "", 0, "", "repo.agent.RemoveAgentFromGroup", "", map[string]interface{}{
		"operation":     "remove_agent_from_group",
		"option":        "agentRepository.RemoveAgentFromGroup.success",
		"func_name":     "repo.agent.RemoveAgentFromGroup",
		"agent_id":      agentID,
		"group_id":      groupID,
		"rows_affected": delRes.RowsAffected,
	})
	return nil
}

// GetAgentsInGroup 获取分组中的Agent列表（分页）
func (r *agentRepository) GetAgentsInGroup(groupID string) ([]*agentModel.Agent, int64, error) {
	// 获取指定分组中的所有 Agent 列表
	// 说明：本函数不做分组 status 过滤（管理员可能需要查看禁用分组的成员）；如需只查询激活分组，请在上层调用处控制。
	// 返回值：agents 列表 + 总成员数 total

	// 1. 校验分组是否存在（避免误查询无效 group_id 导致误解）
	var group agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&group).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetAgentsInGroup", "", map[string]interface{}{
			"operation": "get_agents_in_group",
			"option":    "agentRepository.GetAgentsInGroup.find_group",
			"func_name": "repo.agent.GetAgentsInGroup",
			"group_id":  groupID,
		})
		return nil, 0, err
	}

	// 2. 统计总成员数量（用于前端展示或分页器）
	var total int64
	if err := r.db.Model(&agentModel.AgentGroupMember{}).
		Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetAgentsInGroup", "", map[string]interface{}{
			"operation": "get_agents_in_group",
			"option":    "agentRepository.GetAgentsInGroup.count_members",
			"func_name": "repo.agent.GetAgentsInGroup",
			"group_id":  groupID,
		})
		return nil, 0, err
	}

	// 3. 通过 JOIN 查询获取 Agent 详细信息
	var agents []*agentModel.Agent
	// 为了保证 SQL 可读性，显式指定表名，并按照 Agent 更新时间倒序，便于观察最近活动的 Agent
	agentsTable := agentModel.Agent{}.TableName() // "agents"
	err := r.db.Table(agentsTable).
		Select(agentsTable+".*").
		Joins("JOIN agent_group_members gm ON gm.agent_id = "+agentsTable+".agent_id").
		Where("gm.group_id = ?", groupID).
		Order(agentsTable + ".updated_at DESC").
		Find(&agents).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetAgentsInGroup", "", map[string]interface{}{
			"operation": "get_agents_in_group",
			"option":    "agentRepository.GetAgentsInGroup.query_agents",
			"func_name": "repo.agent.GetAgentsInGroup",
			"group_id":  groupID,
		})
		return nil, 0, err
	}

	// 4. 成功日志
	logger.LogInfo("got agents in group", "", 0, "", "repo.agent.GetAgentsInGroup", "", map[string]interface{}{
		"operation":       "get_agents_in_group",
		"option":          "agentRepository.GetAgentsInGroup.success",
		"func_name":       "repo.agent.GetAgentsInGroup",
		"group_id":        groupID,
		"total":           total,
		"group_status":    group.Status,
		"is_system_group": group.IsSystem,
	})
	return agents, total, nil
}
