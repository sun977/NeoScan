/**
 * Agent仓库层:Agent数据访问
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent数据访问层，专注于数据操作，不包含业务逻辑
 * @func: 单纯数据访问，遵循"好品味"原则 - 数据结构优先，消除特殊情况
 */
package agent

import (
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
	Update(agentData *agentModel.Agent) error
	Delete(agentID string) error

	// Agent状态管理
	UpdateStatus(agentID string, status agentModel.AgentStatus) error
	UpdateLastHeartbeat(agentID string) error

	// Agent查询操作
	GetList(page, pageSize int, status *agentModel.AgentStatus, keyword *string, tags []string, capabilities []string) ([]*agentModel.Agent, int64, error)
	GetByStatus(status agentModel.AgentStatus) ([]*agentModel.Agent, error)
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
		logger.LogError(result.Error, "", 0, "", "repository.agent.Create", "", map[string]interface{}{
			"operation": "create_agent",
			"option":    "agentRepository.Create",
			"func_name": "repository.agent.Create",
			"agent_id":  agentData.AgentID,
			"hostname":  agentData.Hostname,
		})
		return result.Error
	}

	logger.LogInfo("Agent记录创建成功", "", 0, "", "repository.agent.Create", "", map[string]interface{}{
		"operation": "create_agent",
		"option":    "agentRepository.Create",
		"func_name": "repository.agent.Create",
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
		logger.LogError(result.Error, "", 0, "", "repository.agent.GetByID", "", map[string]interface{}{
			"operation": "get_agent_by_id",
			"option":    "agentRepository.GetByID",
			"func_name": "repository.agent.GetByID",
			"agent_id":  agentID,
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
		logger.LogError(result.Error, "", 0, "", "repository.agent.GetByHostname", "", map[string]interface{}{
			"operation": "get_agent_by_hostname",
			"option":    "agentRepository.GetByHostname",
			"func_name": "repository.agent.GetByHostname",
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
		logger.LogError(result.Error, "", 0, "", "repository.agent.Update", "", map[string]interface{}{
			"operation": "update_agent",
			"option":    "agentRepository.Update",
			"func_name": "repository.agent.Update",
			"agent_id":  agentData.AgentID,
			"hostname":  agentData.Hostname,
		})
		return result.Error
	}

	logger.LogInfo("Agent记录更新成功", "", 0, "", "repository.agent.Update", "", map[string]interface{}{
		"operation": "update_agent",
		"option":    "agentRepository.Update",
		"func_name": "repository.agent.Update",
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
		logger.LogError(result.Error, "", 0, "", "repository.agent.UpdateStatus", "", map[string]interface{}{
			"operation": "update_agent_status",
			"option":    "agentRepository.UpdateStatus",
			"func_name": "repository.agent.UpdateStatus",
			"agent_id":  agentID,
			"status":    string(status),
		})
		return result.Error
	}

	logger.LogInfo("Agent状态更新成功", "", 0, "", "repository.agent.UpdateStatus", "", map[string]interface{}{
		"operation": "update_agent_status",
		"option":    "agentRepository.UpdateStatus",
		"func_name": "repository.agent.UpdateStatus",
		"agent_id":  agentID,
		"status":    string(status),
	})

	return nil
}

// UpdateLastHeartbeat 更新Agent最后心跳时间
// 参数: agentID - Agent的业务ID
// 返回: error - 更新过程中的错误信息
func (r *agentRepository) UpdateLastHeartbeat(agentID string) error {
	result := r.db.Model(&agentModel.Agent{}).
		Where("agent_id = ?", agentID).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"updated_at":     time.Now(),
		})

	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repository.agent.UpdateLastHeartbeat", "", map[string]interface{}{
			"operation": "update_heartbeat",
			"option":    "agentRepository.UpdateLastHeartbeat",
			"func_name": "repository.agent.UpdateLastHeartbeat",
			"agent_id":  agentID,
		})
		return result.Error
	}

	logger.LogInfo("Agent心跳时间更新成功", "", 0, "", "repository.agent.UpdateLastHeartbeat", "", map[string]interface{}{
		"operation": "update_heartbeat",
		"option":    "agentRepository.UpdateLastHeartbeat",
		"func_name": "repository.agent.UpdateLastHeartbeat",
		"agent_id":  agentID,
	})

	return nil
}

// Delete 删除Agent记录
// 参数: agentID - Agent的业务ID
// 返回: error - 删除过程中的错误信息
func (r *agentRepository) Delete(agentID string) error {
	result := r.db.Where("agent_id = ?", agentID).Delete(&agentModel.Agent{})
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repository.agent.Delete", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "agentRepository.Delete",
			"func_name": "repository.agent.Delete",
			"agent_id":  agentID,
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.LogInfo("Agent记录不存在，无需删除", "", 0, "", "repository.agent.Delete", "", map[string]interface{}{
			"operation": "delete_agent",
			"option":    "agentRepository.Delete",
			"func_name": "repository.agent.Delete",
			"agent_id":  agentID,
		})
		return nil // 不存在也不算错误
	}

	logger.LogInfo("Agent记录删除成功", "", 0, "", "repository.agent.Delete", "", map[string]interface{}{
		"operation": "delete_agent",
		"option":    "agentRepository.Delete",
		"func_name": "repository.agent.Delete",
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
		logger.LogError(err, "", 0, "", "repository.agent.GetList", "", map[string]interface{}{
			"operation": "get_agent_list",
			"option":    "agentRepository.GetList",
			"func_name": "repository.agent.GetList",
		})
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据
	if err := query.Offset(offset).Limit(pageSize).Find(&agents).Error; err != nil {
		logger.LogError(err, "", 0, "", "repository.agent.GetList", "", map[string]interface{}{
			"operation": "get_agent_list",
			"option":    "agentRepository.GetList",
			"func_name": "repository.agent.GetList",
		})
		return nil, 0, err
	}

	logger.LogInfo("Agent列表获取成功", "", 0, "", "repository.agent.GetList", "", map[string]interface{}{
		"operation": "get_agent_list",
		"option":    "agentRepository.GetList",
		"func_name": "repository.agent.GetList",
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
		logger.LogError(result.Error, "", 0, "", "repository.agent.GetByStatus", "", map[string]interface{}{
			"operation": "get_agents_by_status",
			"option":    "agentRepository.GetByStatus",
			"func_name": "repository.agent.GetByStatus",
			"status":    string(status),
		})
		return nil, result.Error
	}

	return agents, nil
}
