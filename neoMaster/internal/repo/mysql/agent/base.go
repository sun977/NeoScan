/**
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent 基础数据与状态管理实现
 * @func: 提供Agent基础的CRUD操作，不包含业务逻辑
 * 包含：
 * - Create: 创建Agent [实际操作是创建数据库记录]
 * - GetByID: 根据ID获取Agent [读取数据库记录]
 * - GetByHostnameAndPort: 根据主机名和端口获取Agent
 * - GetByHostname: 根据主机名获取Agent
 * - Update: 更新Agent [实际操作是更新数据库记录]
 * - UpdateStatus: 更新Agent状态
 * - UpdateLastHeartbeat: 更新Agent最后心跳时间
 * - Delete: 删除Agent [实际操作是删除数据库记录]
 * - GetList: 获取Agent列表
 * - GetByStatus: 根据状态获取Agent列表
 * 从原来 agent.go 中分离出
 */
package agent

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// Create 创建Agent [实际操作是创建数据库记录]
func (r *agentRepository) Create(agentData *agentModel.Agent) error {
	// 参数校验
	if agentData == nil {
		// 参数非法，记录错误
		logger.LogError(
			gorm.ErrInvalidData,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "create_agent",
				"option":    "repo.agent.Create",
				"func_name": "repo.mysql.agent.Create",
				"reason":    "agentData is nil",
			},
		)
		return gorm.ErrInvalidData
	}
	// 设置时间
	agentData.CreatedAt = time.Now()
	agentData.UpdatedAt = time.Now()

	// 设置默认状态（如果未指定）。AgentStatus 为字符串类型，模型默认值为 offline
	if agentData.Status == "" {
		agentData.Status = agentModel.AgentStatusOffline
	}

	err := r.db.Create(agentData).Error
	if err != nil {
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "create_agent",
				"option":    "repo.agent.Create",
				"func_name": "repo.mysql.agent.Create",
				"agent_id":  agentData.AgentID,
			},
		)
		return err
	}
	logger.LogInfo(
		"Agent created successfully",
		"", 0, "", "repo.mysql.agent", "gorm",
		map[string]interface{}{
			"operation": "create_agent",
			"option":    "repo.agent.Create",
			"func_name": "repo.mysql.agent.Create",
			"agent_id":  agentData.AgentID,
		},
	)
	return nil
}

// GetByID 根据ID获取Agent [读取数据库记录]
// 参数: agentID - Agent的业务ID
// 返回: *agentModel.Agent - Agent数据, error - 错误信息
func (r *agentRepository) GetByID(agentID string) (*agentModel.Agent, error) {
	var agent agentModel.Agent
	err := r.db.Where("agent_id = ?", agentID).First(&agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录信息日志，说明未找到
			logger.LogInfo(
				"Agent not found",
				"", 0, "", "repo.mysql.agent", "gorm",
				map[string]interface{}{
					"operation": "get_agent_by_id",
					"option":    "repo.agent.GetByID",
					"func_name": "repo.mysql.agent.GetByID",
					"agent_id":  agentID,
				},
			)
			return nil, err
		}
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "get_agent_by_id",
				"option":    "repo.agent.GetByID",
				"func_name": "repo.mysql.agent.GetByID",
				"agent_id":  agentID,
			},
		)
		return nil, err
	}
	return &agent, nil
}

// GetByHostnameAndPort 根据主机名和端口获取Agent
// 参数: hostname - 主机名, port - 端口号
// 返回: *agentModel.Agent - Agent数据, error - 错误信息
// 用于支持一机多Agent部署场景，通过hostname+port唯一标识Agent
func (r *agentRepository) GetByHostnameAndPort(hostname string, port int) (*agentModel.Agent, error) {
	var agent agentModel.Agent
	err := r.db.Where("hostname = ? AND port = ?", hostname, port).First(&agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogInfo(
				"Agent not found by hostname and port",
				"", 0, "", "repo.mysql.agent", "gorm",
				map[string]interface{}{
					"operation": "get_agent_by_hostname_and_port",
					"option":    "repo.agent.GetByHostnameAndPort",
					"func_name": "repo.mysql.agent.GetByHostnameAndPort",
					"hostname":  hostname,
					"port":      port,
				},
			)
			// return nil, err
			return nil, nil // 返回nil表示未找到，不是错误
		}
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "get_agent_by_hostname_and_port",
				"option":    "repo.agent.GetByHostnameAndPort",
				"func_name": "repo.mysql.agent.GetByHostnameAndPort",
				"hostname":  hostname,
				"port":      port,
			},
		)
		return nil, err
	}
	return &agent, nil
}

// GetByHostname 根据主机名获取Agent
func (r *agentRepository) GetByHostname(hostname string) (*agentModel.Agent, error) {
	var agent agentModel.Agent
	err := r.db.Where("hostname = ?", hostname).First(&agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogInfo(
				"Agent not found by hostname",
				"", 0, "", "repo.mysql.agent", "gorm",
				map[string]interface{}{
					"operation": "get_agent_by_hostname",
					"option":    "repo.agent.GetByHostname",
					"func_name": "repo.mysql.agent.GetByHostname",
					"hostname":  hostname,
				},
			)
			return nil, err
		}
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "get_agent_by_hostname",
				"option":    "repo.agent.GetByHostname",
				"func_name": "repo.mysql.agent.GetByHostname",
				"hostname":  hostname,
			},
		)
		return nil, err
	}
	return &agent, nil
}

// Update 更新Agent数据 [更新数据库记录]
// 参数: agentData - Agent数据指针
// 返回: error - 更新过程中的错误信息
func (r *agentRepository) Update(agentData *agentModel.Agent) error {
	// 参数校验
	if agentData == nil || agentData.AgentID == "" {
		logger.LogError(
			gorm.ErrInvalidData,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "update_agent",
				"option":    "repo.agent.Update",
				"func_name": "repo.mysql.agent.Update",
				"reason":    "agentData is invalid",
			},
		)
		return gorm.ErrInvalidData
	}

	// 让 GORM 自动维护 UpdatedAt
	// 使用 Updates 方法更新指定字段,前端提供什么字段就更新什么字段
	err := r.db.Model(&agentModel.Agent{}).Where("agent_id = ?", agentData.AgentID).Updates(agentData).Error
	if err != nil {
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "update_agent",
				"option":    "repo.agent.Update",
				"func_name": "repo.mysql.agent.Update",
				"agent_id":  agentData.AgentID,
			},
		)
		return err
	}
	logger.LogInfo(
		"Agent updated successfully",
		"", 0, "", "repo.mysql.agent", "gorm",
		map[string]interface{}{
			"operation": "agent_updated",
			"option":    "repo.agent.Update",
			"func_name": "repo.mysql.agent.Update",
			"agent_id":  agentData.AgentID,
		},
	)
	return nil
}

// UpdateStatus 更新Agent状态
func (r *agentRepository) UpdateStatus(agentID string, status agentModel.AgentStatus) error {
	// 参数校验
	if agentID == "" {
		logger.LogError(
			gorm.ErrInvalidData,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "update_agent_status",
				"option":    "repo.agent.UpdateStatus",
				"func_name": "repo.mysql.agent.UpdateStatus",
				"reason":    "agentID is empty",
			},
		)
		return gorm.ErrInvalidData
	}

	err := r.db.Model(&agentModel.Agent{}).Where("agent_id = ?", agentID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "update_agent_status",
				"option":    "repo.agent.UpdateStatus",
				"func_name": "repo.mysql.agent.UpdateStatus",
				"agent_id":  agentID,
				"status":    status,
			},
		)
		return err
	}
	logger.LogInfo(
		"Agent status updated successfully",
		"", 0, "", "repo.mysql.agent", "gorm",
		map[string]interface{}{
			"operation": "update_agent_status",
			"option":    "repo.agent.UpdateStatus",
			"func_name": "repo.mysql.agent.UpdateStatus",
			"agent_id":  agentID,
			"status":    status,
		},
	)
	return nil
}

// UpdateLastHeartbeat 更新Agent的最后心跳时间
// 参数: agentID - Agent的业务ID (同时更新 updated_at 和 last_heartbeat 字段)
// 返回: error - 更新过程中的错误信息
func (r *agentRepository) UpdateLastHeartbeat(agentID string) error {
	// 参数校验
	if agentID == "" {
		logger.LogError(
			gorm.ErrInvalidData,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "update_agent_last_heartbeat",
				"option":    "repo.agent.UpdateLastHeartbeat",
				"func_name": "repo.mysql.agent.UpdateLastHeartbeat",
				"reason":    "agentID is empty",
			},
		)
		return gorm.ErrInvalidData
	}

	result := r.db.Model(&agentModel.Agent{}).
		Where("agent_id = ?", agentID).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"updated_at":     time.Now(),
		})

	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.UpdateLastHeartbeat", "", map[string]interface{}{
			"operation": "update_agent_last_heartbeat",
			"option":    "repo.agent.UpdateLastHeartbeat",
			"func_name": "repo.mysql.agent.UpdateLastHeartbeat",
			"agent_id":  agentID,
		})
		return result.Error
	}

	// 检查是否真的更新了记录
	if result.RowsAffected == 0 {
		err := fmt.Errorf("agent not found")
		logger.LogError(err, "", 0, "", "repo.agent.UpdateLastHeartbeat", "", map[string]interface{}{
			"operation": "update_agent_last_heartbeat",
			"option":    "repo.agent.UpdateLastHeartbeat",
			"func_name": "repo.mysql.agent.UpdateLastHeartbeat",
			"agent_id":  agentID,
			"message":   "No rows affected, agent not found",
		})
		return err
	}

	logger.LogInfo(
		"Agent last heartbeat updated successfully",
		"", 0, "", "repo.mysql.agent", "gorm",
		map[string]interface{}{
			"operation": "update_agent_last_heartbeat",
			"option":    "repo.agent.UpdateLastHeartbeat",
			"func_name": "repo.mysql.agent.UpdateLastHeartbeat",
			"agent_id":  agentID,
		},
	)
	return nil
}

// Delete 删除Agent [删除数据库记录]
func (r *agentRepository) Delete(agentID string) error {
	// 参数校验
	if agentID == "" {
		logger.LogError(
			gorm.ErrInvalidData,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "delete_agent",
				"option":    "repo.agent.Delete",
				"func_name": "repo.mysql.agent.Delete",
				"reason":    "agentID is empty",
			},
		)
		return gorm.ErrInvalidData
	}

	result := r.db.Where("agent_id = ?", agentID).Delete(&agentModel.Agent{})
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.Delete", "gorm", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "repo.agent.Delete",
			"func_name": "repo.mysql.agent.Delete",
			"agent_id":  agentID,
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.LogInfo("Agent record not found, no need to delete", "", 0, "", "repo.agent.Delete", "gorm", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "repo.agent.Delete",
			"func_name": "repo.mysql.agent.Delete",
			"agent_id":  agentID,
		})
		return nil // 不存在也不算错误
	}

	logger.LogInfo("Agent record deleted successfully", "", 0, "", "repo.agent.Delete", "gorm", map[string]interface{}{
		"operation": "delete_agent",
		"option":    "repo.agent.Delete",
		"func_name": "repo.mysql.agent.Delete",
		"agent_id":  agentID,
	})
	return nil
}

// GetList 获取Agent列表（支持分页、按状态、关键词、标签、任务支持过滤）
// 参数: page - 页码, pageSize - 每页大小, status - 状态过滤, keyword - 关键字过滤, tags - 标签过滤, taskSupport - 任务支持过滤
// 返回: []*agentModel.Agent - Agent列表, int64 - 总数量, error - 错误信息
func (r *agentRepository) GetList(page, pageSize int, status *agentModel.AgentStatus, keyword *string, features []string, taskSupport []string) ([]*agentModel.Agent, int64, error) {
	var agents []*agentModel.Agent
	var total int64

	// 构建查询
	query := r.db.Model(&agentModel.Agent{})

	// 状态过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	// 关键词过滤（AgentID、主机名、IP、备注等）
	if keyword != nil && *keyword != "" {
		like := fmt.Sprintf("%%%s%%", *keyword)
		// 修正字段名为 ip_address，以匹配模型定义
		query = query.Where("agent_id LIKE ? OR hostname LIKE ? OR ip_address LIKE ? OR remark LIKE ?", like, like, like, like)
	}
	// 特性/标签过滤 (Feature) - 替代原 Tags
	if len(features) > 0 {
		// 通过 JSON_CONTAINS 在 JSON 数组字段上逐项过滤
		for _, feature := range features {
			query = query.Where("JSON_CONTAINS(feature, JSON_QUOTE(?))", feature)
		}
	}
	// 任务支持过滤 (TaskSupport) - 替代原 Capabilities
	if len(taskSupport) > 0 {
		for _, task := range taskSupport {
			query = query.Where("JSON_CONTAINS(task_support, JSON_QUOTE(?))", task)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "get_agent_list",
				"option":    "repo.agent.GetList",
				"func_name": "repo.mysql.agent.GetList",
			},
		)
		return nil, 0, err
	}

	// 分页查询
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("updated_at DESC").Find(&agents).Error; err != nil {
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "agent",
				"option":    "repo.agent.GetList",
				"func_name": "repo.mysql.agent.GetList",
			},
		)
		return nil, 0, err
	}
	logger.LogInfo("Agent list fetched successfully", "", 0, "", "repo.agent.GetList", "gorm", map[string]interface{}{
		"operation": "get_agent_list",
		"option":    "repo.agent.GetList",
		"func_name": "repo.mysql.agent.GetList",
		"count":     len(agents),
		"total":     total,
	})

	return agents, total, nil
}

// GetByStatus 按状态获取所有Agent
// 参数: status - 状态过滤
// 返回: []*agentModel.Agent - Agent列表, error - 错误信息
func (r *agentRepository) GetByStatus(status agentModel.AgentStatus) ([]*agentModel.Agent, error) {
	var agents []*agentModel.Agent
	// 默认按更新时间降序排序
	err := r.db.Where("status = ?", status).Order("updated_at DESC").Find(&agents).Error
	if err != nil {
		logger.LogError(
			err,
			"", 0, "", "repo.mysql.agent", "gorm",
			map[string]interface{}{
				"operation": "get_agent_list",
				"option":    "repo.agent.GetByStatus",
				"func_name": "repo.mysql.agent.GetByStatus",
				"status":    status,
			},
		)
		return nil, err
	}
	return agents, nil
}
