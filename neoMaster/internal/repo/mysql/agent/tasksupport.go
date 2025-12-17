/**
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent 任务支持管理实现
 * @func: 提供Agent任务支持的CRUD操作，不包含业务逻辑
 * Agent 任务支持管理实现
 * 包含：
 * - IsValidTaskSupportId: 任务支持ID是否合法（是否在任务支持表定义）
 * - IsValidTaskSupportByName: 任务支持名称是否合法
 * - AddTaskSupport: 为Agent添加任务支持
 * - RemoveTaskSupport: 从Agent移除任务支持
 * - HasTaskSupport: 检查Agent是否有指定任务支持
 * - GetTaskSupports: 获取Agent所有任务支持
 * 重构说明：
 * - 统一使用 logger.LogInfo / logger.LogError
 * - 移除不存在的关系表（AgentCapabilityRel、AgentTagRel）
 * - 任务支持校验对齐 ScanType 表，标签校验对齐 TagType 表
 */
package agent

import (
	"fmt"

	"gorm.io/gorm"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// ============================================================================
// Agent 任务支持管理实现 (TaskSupport)
// ============================================================================

// IsValidTaskSupportId 判断任务支持ID是否有效
func (r *agentRepository) IsValidTaskSupportId(taskID string) bool {
	var count int64
	// 直接查询ScanType表的id字段，验证该ID是否存在且激活
	result := r.db.Model(&agentModel.ScanType{}).Where("id = ? AND is_active = ?", taskID, 1).Count(&count)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.IsValidTaskSupportId", "", map[string]interface{}{
			"operation": "validate_task_support_id",
			"option":    "agentRepository.IsValidTaskSupportId",
			"func_name": "repo.agent.IsValidTaskSupportId",
			"taskID":    taskID,
		})
		return false
	}
	return count > 0
}

// IsValidTaskSupportByName 判断任务支持名称是否有效
func (r *agentRepository) IsValidTaskSupportByName(taskName string) bool {
	var count int64
	result := r.db.Model(&agentModel.ScanType{}).Where("name = ? AND is_active = ?", taskName, 1).Count(&count)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.IsValidTaskSupportByName", "", map[string]interface{}{
			"operation": "validate_task_support_name",
			"option":    "agentRepository.IsValidTaskSupportByName",
			"func_name": "repo.agent.IsValidTaskSupportByName",
			"taskName":  taskName,
		})
		return false
	}
	return count > 0
}

// GetTagIDsByTaskSupportNames 根据任务支持名称列表获取对应的TagID列表
func (r *agentRepository) GetTagIDsByTaskSupportNames(names []string) ([]uint64, error) {
	if len(names) == 0 {
		return []uint64{}, nil
	}
	var tagIDs []uint64
	// 查询ScanType表，获取对应TagID
	// 只查询激活的ScanType
	err := r.db.Model(&agentModel.ScanType{}).
		Where("name IN ? AND is_active = ?", names, true).
		Pluck("tag_id", &tagIDs).Error

	if err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetTagIDsByTaskSupportNames", "gorm", map[string]interface{}{
			"operation": "get_tag_ids_by_task_support_names",
			"option":    "db.Pluck",
			"names":     names,
		})
		return nil, err
	}
	return tagIDs, nil
}

// GetTagIDsByTaskSupportIDs 根据任务支持ID列表获取对应的TagID列表
func (r *agentRepository) GetTagIDsByTaskSupportIDs(ids []string) ([]uint64, error) {
	if len(ids) == 0 {
		return []uint64{}, nil
	}
	var tagIDs []uint64
	err := r.db.Model(&agentModel.ScanType{}).
		Where("id IN ? AND is_active = ?", ids, true).
		Pluck("tag_id", &tagIDs).Error

	if err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetTagIDsByTaskSupportIDs", "gorm", map[string]interface{}{
			"operation": "get_tag_ids_by_task_support_ids",
			"option":    "db.Pluck",
			"ids":       ids,
		})
		return nil, err
	}
	return tagIDs, nil
}

// AddTaskSupport 为Agent添加任务支持
func (r *agentRepository) AddTaskSupport(agentID string, taskID string) error {
	// 参数校验
	if agentID == "" || taskID == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.agent.AddTaskSupport", "", map[string]interface{}{
			"operation": "add_task_support",
			"error":     "invalid_input",
		})
		return gorm.ErrInvalidData
	}

	// 校验ID是否有效
	if !r.IsValidTaskSupportId(taskID) {
		return fmt.Errorf("invalid task_support id: %s", taskID)
	}

	// 读取 Agent
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// 幂等检查
	if agent.HasTaskSupport(taskID) {
		return nil
	}

	// 更新模型
	agent.AddTaskSupport(taskID)

	// 更新数据库 task_support 字段
	if err := r.db.Model(&agent).Select("task_support").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddTaskSupport", "gorm", map[string]interface{}{
			"operation": "add_task_support",
			"agentID":   agentID,
			"taskID":    taskID,
		})
		return fmt.Errorf("update agent task_support failed: %v", err)
	}

	return nil
}

// RemoveTaskSupport 移除Agent任务支持
func (r *agentRepository) RemoveTaskSupport(agentID string, taskID string) error {
	if agentID == "" || taskID == "" {
		return gorm.ErrInvalidData
	}

	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if !agent.HasTaskSupport(taskID) {
		return nil
	}

	agent.RemoveTaskSupport(taskID)

	if err := r.db.Model(&agent).Select("task_support").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.RemoveTaskSupport", "gorm", map[string]interface{}{
			"operation": "remove_task_support",
			"agentID":   agentID,
			"taskID":    taskID,
		})
		return fmt.Errorf("update agent task_support failed: %v", err)
	}

	return nil
}

// HasTaskSupport 判断Agent是否支持指定任务
func (r *agentRepository) HasTaskSupport(agentID string, taskID string) bool {
	if agentID == "" || taskID == "" {
		return false
	}
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		return false
	}
	return agent.HasTaskSupport(taskID)
}

// GetTaskSupport 获取Agent所有任务支持列表
func (r *agentRepository) GetTaskSupport(agentID string) []string {
	if agentID == "" {
		return []string{}
	}
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		return []string{}
	}
	if agent.TaskSupport == nil {
		return []string{}
	}
	return agent.TaskSupport
}

// GetAllScanTypes 获取所有ScanType
func (r *agentRepository) GetAllScanTypes() ([]*agentModel.ScanType, error) {
	var scanTypes []*agentModel.ScanType
	err := r.db.Model(&agentModel.ScanType{}).Find(&scanTypes).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetAllScanTypes", "gorm", map[string]interface{}{
			"operation": "get_all_scan_types",
		})
		return nil, err
	}
	return scanTypes, nil
}

// UpdateScanType 更新ScanType
func (r *agentRepository) UpdateScanType(scanType *agentModel.ScanType) error {
	if scanType == nil {
		return nil
	}
	return r.db.Save(scanType).Error
}
