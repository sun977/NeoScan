/**
 * Agent仓库层:Agent数据访问
 * @author: Linus-style implementation
 * @date: 2025.10.14
 * @description: Agent数据访问层，专注于数据操作，不包含业务逻辑
 * @func: 单纯数据访问，遵循"好品味"原则 - 数据结构优先，消除特殊情况
 */
package agent

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// AgentRepository Agent仓库接口定义
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
	GetList(page, pageSize int, status *agentModel.AgentStatus, tags []string) ([]*agentModel.Agent, int64, error)
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
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.Create",
			"operation": "create_agent",
			"option":    "agentRepository.Create",
			"func_name": "repository.agent.Create",
			"agent_id":  agentData.AgentID,
			"hostname":  agentData.Hostname,
			"error":     result.Error.Error(),
		}).Error("创建Agent记录失败")
		return result.Error
	}
	
	logger.WithFields(logrus.Fields{
		"path":      "repository.agent.Create",
		"operation": "create_agent",
		"option":    "agentRepository.Create",
		"func_name": "repository.agent.Create",
		"agent_id":  agentData.AgentID,
		"hostname":  agentData.Hostname,
	}).Info("Agent记录创建成功")
	
	return nil
}

// GetByID 根据AgentID获取Agent
// 参数: agentID - Agent的业务ID
// 返回: *agentModel.Agent - Agent数据, error - 错误信息
func (r *agentRepository) GetByID(agentID string) (*agentModel.Agent, error) {
	var agentData agentModel.Agent
	
	result := r.db.Where("agent_id = ?", agentID).First(&agentData)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // 返回nil表示未找到，不是错误
		}
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.GetByID",
			"operation": "get_agent_by_id",
			"option":    "agentRepository.GetByID",
			"func_name": "repository.agent.GetByID",
			"agent_id":  agentID,
			"error":     result.Error.Error(),
		}).Error("根据AgentID获取Agent失败")
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
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.GetByHostname",
			"operation": "get_agent_by_hostname",
			"option":    "agentRepository.GetByHostname",
			"func_name": "repository.agent.GetByHostname",
			"hostname":  hostname,
			"error":     result.Error.Error(),
		}).Error("根据主机名获取Agent失败")
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
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.Update",
			"operation": "update_agent",
			"option":    "agentRepository.Update",
			"func_name": "repository.agent.Update",
			"agent_id":  agentData.AgentID,
			"hostname":  agentData.Hostname,
			"error":     result.Error.Error(),
		}).Error("更新Agent记录失败")
		return result.Error
	}
	
	logger.WithFields(logrus.Fields{
		"path":      "repository.agent.Update",
		"operation": "update_agent",
		"option":    "agentRepository.Update",
		"func_name": "repository.agent.Update",
		"agent_id":  agentData.AgentID,
		"hostname":  agentData.Hostname,
	}).Info("Agent记录更新成功")
	
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
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.UpdateStatus",
			"operation": "update_agent_status",
			"option":    "agentRepository.UpdateStatus",
			"func_name": "repository.agent.UpdateStatus",
			"agent_id":  agentID,
			"status":    string(status),
			"error":     result.Error.Error(),
		}).Error("更新Agent状态失败")
		return result.Error
	}
	
	logger.WithFields(logrus.Fields{
		"path":      "repository.agent.UpdateStatus",
		"operation": "update_agent_status",
		"option":    "agentRepository.UpdateStatus",
		"func_name": "repository.agent.UpdateStatus",
		"agent_id":  agentID,
		"status":    string(status),
	}).Info("Agent状态更新成功")
	
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
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.UpdateLastHeartbeat",
			"operation": "update_heartbeat",
			"option":    "agentRepository.UpdateLastHeartbeat",
			"func_name": "repository.agent.UpdateLastHeartbeat",
			"agent_id":  agentID,
			"error":     result.Error.Error(),
		}).Error("更新Agent心跳时间失败")
		return result.Error
	}

	logger.WithFields(logrus.Fields{
		"path":      "repository.agent.UpdateLastHeartbeat",
		"operation": "update_heartbeat",
		"option":    "agentRepository.UpdateLastHeartbeat",
		"func_name": "repository.agent.UpdateLastHeartbeat",
		"agent_id":  agentID,
	}).Info("Agent心跳时间更新成功")

	return nil
}

// Delete 删除Agent记录
// 参数: agentID - Agent的业务ID
// 返回: error - 删除过程中的错误信息
func (r *agentRepository) Delete(agentID string) error {
	result := r.db.Where("agent_id = ?", agentID).Delete(&agentModel.Agent{})
	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.Delete",
			"operation": "delete_agent",
			"option":    "agentRepository.Delete",
			"func_name": "repository.agent.Delete",
			"agent_id":  agentID,
			"error":     result.Error.Error(),
		}).Error("删除Agent记录失败")
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.Delete",
			"operation": "delete_agent",
			"option":    "agentRepository.Delete",
			"func_name": "repository.agent.Delete",
			"agent_id":  agentID,
		}).Info("Agent记录不存在，无需删除")
		return nil // 不存在也不算错误
	}
	
	logger.WithFields(logrus.Fields{
		"path":      "repository.agent.Delete",
		"operation": "delete_agent",
		"option":    "agentRepository.Delete",
		"func_name": "repository.agent.Delete",
		"agent_id":  agentID,
	}).Info("Agent记录删除成功")
	
	return nil
}

// GetList 获取Agent列表
// 参数: page - 页码, pageSize - 每页大小, status - 状态过滤, tags - 标签过滤
// 返回: []*agentModel.Agent - Agent列表, int64 - 总数量, error - 错误信息
func (r *agentRepository) GetList(page, pageSize int, status *agentModel.AgentStatus, tags []string) ([]*agentModel.Agent, int64, error) {
	var agents []*agentModel.Agent
	var total int64

	// 构建查询条件
	query := r.db.Model(&agentModel.Agent{})
	
	// 状态过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	
	// 标签过滤（简单实现，实际可能需要更复杂的JSON查询）
	if len(tags) > 0 {
		for _, tag := range tags {
			query = query.Where("tags LIKE ?", "%"+tag+"%")
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.GetList",
			"operation": "get_agent_list",
			"option":    "agentRepository.GetList",
			"func_name": "repository.agent.GetList",
			"error":     err.Error(),
		}).Error("获取Agent总数失败")
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取分页数据
	if err := query.Offset(offset).Limit(pageSize).Find(&agents).Error; err != nil {
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.GetList",
			"operation": "get_agent_list",
			"option":    "agentRepository.GetList",
			"func_name": "repository.agent.GetList",
			"error":     err.Error(),
		}).Error("获取Agent列表失败")
		return nil, 0, err
	}

	logger.WithFields(logrus.Fields{
		"path":      "repository.agent.GetList",
		"operation": "get_agent_list",
		"option":    "agentRepository.GetList",
		"func_name": "repository.agent.GetList",
		"count":     len(agents),
		"total":     total,
	}).Info("Agent列表获取成功")

	return agents, total, nil
}

// GetByStatus 根据状态获取Agent列表
// 参数: status - Agent状态
// 返回: []*agentModel.Agent - Agent列表, error - 错误信息
func (r *agentRepository) GetByStatus(status agentModel.AgentStatus) ([]*agentModel.Agent, error) {
	var agents []*agentModel.Agent
	
	result := r.db.Where("status = ?", status).Find(&agents)
	if result.Error != nil {
		logger.WithFields(logrus.Fields{
			"path":      "repository.agent.GetByStatus",
			"operation": "get_agents_by_status",
			"option":    "agentRepository.GetByStatus",
			"func_name": "repository.agent.GetByStatus",
			"status":    string(status),
			"error":     result.Error.Error(),
		}).Error("根据状态获取Agent列表失败")
		return nil, result.Error
	}
	
	return agents, nil
}