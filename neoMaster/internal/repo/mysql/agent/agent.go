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
	// Agent基础数据操作
	Create(agentData *agentModel.Agent) error
	GetByID(agentID string) (*agentModel.Agent, error)
	GetByHostname(hostname string) (*agentModel.Agent, error)
	GetByHostnameAndPort(hostname string, port int) (*agentModel.Agent, error) // 根据主机名和端口获取Agent
	Update(agentData *agentModel.Agent) error
	Delete(agentID string) error

	// Agent状态管理
	UpdateStatus(agentID string, status agentModel.AgentStatus) error
	UpdateLastHeartbeat(agentID string) error

	// Agent性能指标管理 - 直接操作agent_metrics表
	CreateMetrics(metrics *agentModel.AgentMetrics) error
	GetLatestMetrics(agentID string) (*agentModel.AgentMetrics, error)
	UpdateAgentMetrics(agentID string, metrics *agentModel.AgentMetrics) error
	GetMetricsList(page, pageSize int, workStatus *agentModel.AgentWorkStatus, scanType *agentModel.AgentScanType, keyword *string) ([]*agentModel.AgentMetrics, int64, error) // 性能指标批量查询（分页 + 过滤）

	// Agent查询操作
	GetList(page, pageSize int, status *agentModel.AgentStatus, keyword *string, tags []string, capabilities []string) ([]*agentModel.Agent, int64, error)
	GetByStatus(status agentModel.AgentStatus) ([]*agentModel.Agent, error)

	// Agent能力管理
	IsValidCapabilityId(capability string) bool                 // 判断能力ID是否有效
	IsValidCapabilityByName(capability string) bool             // 判断能力名称是否有效
	AddCapability(agentID string, capabilityID string) error    // 添加Agent能力
	RemoveCapability(agentID string, capabilityID string) error // 移除Agent能力
	HasCapability(agentID string, capabilityID string) bool     // 判断Agent是否有指定能力
	GetCapabilities(agentID string) []string                    // 获取Agent所有能力ID列表

	// Agent标签管理
	IsValidTagId(tag string) bool                 // 判断标签ID是否有效
	IsValidTagByName(tag string) bool             // 判断标签名称是否有效
	AddTag(agentID string, tagID string) error    // 添加Agent标签
	RemoveTag(agentID string, tagID string) error // 移除Agent标签
	HasTag(agentID string, tagID string) bool     // 判断Agent是否有指定标签
	GetTags(agentID string) []string              // 获取Agent所有标签ID列表
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
		err := fmt.Errorf("Agent not found")
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
		err := fmt.Errorf("Agent not found")
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
		return fmt.Errorf("Agent不存在: %s", agentID)
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
		return fmt.Errorf("Agent不存在: %s", agentID)
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
		return fmt.Errorf("Agent不存在: %s", agentID)
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
		return fmt.Errorf("Agent不存在: %s", agentID)
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
