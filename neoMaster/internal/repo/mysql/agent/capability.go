/**
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent 能力管理实现
 * @func: 提供Agent能力的CRUD操作，不包含业务逻辑
 * Agent 能力管理实现（基于 Agent 模型 JSON 字段）
 * 包含：
 * - IsValidCapabilityId: 能力ID是否合法（是否在能力表定义）
 * - IsValidCapabilityByName: 能力名称是否合法
 * - AddCapability: 为Agent添加能力
 * - RemoveCapability: 从Agent移除能力
 * - HasCapability: 检查Agent是否有指定能力
 * - GetCapabilities: 获取Agent所有能力
 * 重构说明：
 * - 统一使用 logger.LogInfo / logger.LogError
 * - 移除不存在的关系表（AgentCapabilityRel、AgentTagRel），改为更新 agents 表的 JSON 字段
 * - 能力校验对齐 ScanType 表，标签校验对齐 TagType 表
 */
package agent

import (
	"fmt"

	"gorm.io/gorm"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// 能力ID是否合法（是否在能力表定义）
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

// 能力名称是否合法
func (r *agentRepository) IsValidCapabilityByName(capability string) bool {
	var count int64
	// 查询ScanType表的name字段，验证该名称是否存在且激活
	result := r.db.Model(&agentModel.ScanType{}).Where("name = ? AND is_active = ?", capability, 1).Count(&count)
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

// GetTagIDsByScanTypeNames 根据ScanType名称列表获取对应的TagID列表
func (r *agentRepository) GetTagIDsByScanTypeNames(names []string) ([]uint64, error) {
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
		logger.LogError(err, "", 0, "", "repo.agent.GetTagIDsByScanTypeNames", "gorm", map[string]interface{}{
			"operation": "get_tag_ids_by_scantype_names",
			"option":    "db.Pluck",
			"names":     names,
		})
		return nil, err
	}
	return tagIDs, nil
}

// GetTagIDsByScanTypeIDs 根据ScanType ID列表获取对应的TagID列表
func (r *agentRepository) GetTagIDsByScanTypeIDs(ids []string) ([]uint64, error) {
	if len(ids) == 0 {
		return []uint64{}, nil
	}
	var tagIDs []uint64
	err := r.db.Model(&agentModel.ScanType{}).
		Where("id IN ? AND is_active = ?", ids, true).
		Pluck("tag_id", &tagIDs).Error

	if err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetTagIDsByScanTypeIDs", "gorm", map[string]interface{}{
			"operation": "get_tag_ids_by_scantype_ids",
			"option":    "db.Pluck",
			"ids":       ids,
		})
		return nil, err
	}
	return tagIDs, nil
}

// 为Agent添加能力
func (r *agentRepository) AddCapability(agentID string, capabilityID string) error {
	// 参数校验
	if agentID == "" || capabilityID == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repository.mysql.agent.AddCapability", "gorm", map[string]interface{}{
			"operation":     "add_capability",
			"option":        "validate.input",
			"func_name":     "repository.mysql.agent.AddCapability",
			"agent_id":      agentID,
			"capability_id": capabilityID,
		})
		return gorm.ErrInvalidData
	}

	// 校验能力是否有效
	if !r.IsValidCapabilityId(capabilityID) {
		logger.LogError(nil, "", 0, "", "repo.agent.AddCapability", "gorm", map[string]interface{}{
			"operation":    "add_capability",
			"option":       "agentRepository.AddCapability",
			"func_name":    "repo.agent.AddCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
			"error":        "invalid_capability_id",
		})
		return fmt.Errorf("invalid capability id: %s", capabilityID)
	}

	// 读取 Agent 并幂等更新 JSON 字段
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddCapability", "gorm", map[string]interface{}{
			"operation":    "add_capability",
			"option":       "agentRepository.AddCapability",
			"func_name":    "repo.agent.AddCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
		})
		return fmt.Errorf("agent not found: %s", agentID)
	}
	// 幂等：已存在则跳过(检查能力是否已经存在-避免重复添加)
	if agent.HasCapability(capabilityID) {
		logger.LogInfo("Agent already has the capability, skip", "", 0, "", "repo.agent.AddCapability", "", map[string]interface{}{
			"operation":    "add_capability",
			"option":       "agentRepository.AddCapability",
			"func_name":    "repo.agent.AddCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
			"status":       "already_exists",
		})
		return nil // 已存在不算错误，直接返回成功
	}

	// 添加能力并更新 JSON 字段 (使用Agent模型的AddCapability方法)
	agent.AddCapability(capabilityID)

	// 更新数据库的capabilities字段
	if err := r.db.Model(&agent).Select("capabilities").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddCapability", "gorm", map[string]interface{}{
			"operation":    "add_capability",
			"option":       "agentRepository.AddCapability",
			"func_name":    "repo.agent.AddCapability",
			"agentID":      agentID,
			"capabilityID": capabilityID,
		})
		return fmt.Errorf("update agent capabilities failed: %v", err)
	}

	logger.LogInfo("Agent capability added successfully", "", 0, "", "repo.agent.AddCapability", "gorm", map[string]interface{}{
		"operation":    "add_capability",
		"option":       "agentRepository.AddCapability",
		"func_name":    "repo.agent.AddCapability",
		"agentID":      agentID,
		"capabilityID": capabilityID,
		"status":       "success",
	})
	return nil
}

// 移除Agent的某项能力
func (r *agentRepository) RemoveCapability(agentID string, capabilityID string) error {
	// 参数校验
	if agentID == "" || capabilityID == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repository.mysql.agent.RemoveCapability", "", map[string]interface{}{
			"operation": "agent", "option": "validate.input", "func_name": "repository.mysql.agent.RemoveCapability", "agent_id": agentID, "capability_id": capabilityID,
		})
		return gorm.ErrInvalidData
	}

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
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// 2. 检查能力是否存在（幂等性处理）
	if !agent.HasCapability(capabilityID) {
		logger.LogInfo("Agent does not have the capability, skip", "", 0, "", "repo.agent.RemoveCapability", "", map[string]interface{}{
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
		return fmt.Errorf("update agent capabilities failed: %v", err)
	}

	logger.LogInfo("Agent capability removed successfully", "", 0, "", "repo.agent.RemoveCapability", "", map[string]interface{}{
		"operation":    "remove_capability",
		"option":       "agentRepository.RemoveCapability",
		"func_name":    "repo.agent.RemoveCapability",
		"agentID":      agentID,
		"capabilityID": capabilityID,
		"status":       "success",
	})

	return nil
}

// 判断Agent是否具备指定能力
func (r *agentRepository) HasCapability(agentID string, capabilityID string) bool {
	// 1. 输入验证 - 避免无效查询
	if agentID == "" || capabilityID == "" {
		logger.LogInfo("Invalid input: agentID or capabilityID is empty", "", 0, "", "repo.agent.HasCapability", "", map[string]interface{}{
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

// 获取Agent的所有能力ID列表
// 参数: agentID - Agent ID
// 返回: []string - 能力列表
func (r *agentRepository) GetCapabilities(agentID string) []string {
	// 1. 输入验证 - 避免无效查询
	if agentID == "" {
		logger.LogInfo("Agent ID is empty, return empty capability list", "", 0, "", "repo.agent.GetCapabilities", "", map[string]interface{}{
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

// ============================================================================
// Agent 任务支持管理实现 (TaskSupport) - 新增
// ============================================================================

// IsValidTaskSupportId 判断任务支持ID是否有效
// 逻辑独立，不复用 IsValidCapabilityId
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
