/**
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent 标签管理实现
 * @func: 提供Agent标签的CRUD操作，不包含业务逻辑
 * Agent 标签管理实现（基于 Agent 模型 JSON 字段）
 * - IsValidTagId/ByName 校验标签ID/名称是否存在
 * - Add/RemoveTag 为Agent添加/移除标签
 * - HasTag 检查Agent是否有指定标签
 * - GetTags 获取Agent所有标签
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

// ==================== Agent标签管理方法 ====================
// 标签ID是否合法
func (r *agentRepository) IsValidTagId(tagId string) bool {
	var count int64
	result := r.db.Model(&agentModel.TagType{}).Where("id = ?", tagId).Count(&count)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.IsValidTagId", "gorm", map[string]interface{}{
			"operation": "validate_tag_id",
			"option":    "agentRepository.IsValidTagId",
			"func_name": "repo.agent.IsValidTagId",
			"tagId":     tagId,
		})
		return false
	}
	return count > 0
}

// 标签名称是否合法
func (r *agentRepository) IsValidTagByName(tag string) bool {
	var count int64
	result := r.db.Model(&agentModel.TagType{}).Where("JSON_CONTAINS(tags, ?)", `"`+tag+`"`).Count(&count)
	if result.Error != nil {
		logger.LogError(result.Error, "", 0, "", "repo.agent.IsValidTagByName", "gorm", map[string]interface{}{
			"operation": "validate_tag_name",
			"option":    "agentRepository.IsValidTagByName",
			"func_name": "repo.agent.IsValidTagByName",
			"tag":       tag,
		})
		return false
	}
	return count > 0
}

// 为Agent添加标签
// 参数: agentID - Agent ID, tagID - 标签ID
// 返回: error - 错误信息
func (r *agentRepository) AddTag(agentID string, tagID string) error {
	// 参数校验
	if agentID == "" || tagID == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.AddTag", "", map[string]interface{}{
			"operation": "add_tag",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.AddTag",
			"agent_id":  agentID,
			"tag_id":    tagID,
		})
		return gorm.ErrInvalidData
	}

	// 校验标签有效性
	if !r.IsValidTagId(tagID) {
		logger.LogError(nil, "", 0, "", "repo.agent.AddTag", "gorm", map[string]interface{}{
			"operation": "add_tag",
			"option":    "agentRepository.AddTag",
			"func_name": "repo.agent.AddTag",
			"agentID":   agentID,
			"tagID":     tagID,
			"error":     "invalid_tag_id",
		})
		return fmt.Errorf("invalid tag id: %s", tagID)
	}

	// 1.读取 Agent 并幂等更新 JSON 字段
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddTag", "gorm", map[string]interface{}{
			"operation": "add_tag",
			"option":    "agentRepository.AddTag",
			"func_name": "repo.agent.AddTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return fmt.Errorf("agent does not exist: %s", agentID)
	}

	// 2.检查标签是否已存在
	if agent.HasTag(tagID) {
		logger.LogInfo("Tag already exists for agent, skip", "", 0, "", "repo.agent.AddTag", "gorm", map[string]interface{}{
			"operation": "add_tag",
			"option":    "idempotent.skip",
			"func_name": "repo.agent.AddTag",
			"agentID":   agentID,
			"tagID":     tagID,
			"message":   "tag_already_exists",
		})
		return nil // 已存在不算错误，直接返回成功
	}

	// 3.使用Agent模型方法添加标签
	agent.AddTag(tagID)

	// 4.更新数据库
	if err := r.db.Model(&agent).Select("tags").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.AddTag", "grom", map[string]interface{}{
			"operation": "add_tag",
			"option":    "agentRepository.AddTag",
			"func_name": "repo.agent.AddTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return fmt.Errorf("update agent tags failed: %v", err)
	}

	logger.LogInfo("Tag added to agent successfully", "", 0, "", "repo.agent.AddTag", "gorm", map[string]interface{}{
		"operation": "add_tag",
		"option":    "agentRepository.AddTag",
		"func_name": "repo.agent.AddTag",
		"agentID":   agentID,
		"tagID":     tagID,
		"status":    "success",
	})

	return nil
}

// 移除Agent的某个标签
func (r *agentRepository) RemoveTag(agentID string, tagID string) error {
	// 参数校验
	if agentID == "" || tagID == "" {
		logger.LogError(fmt.Errorf("agentID or tagID empty"), "", 0, "", "repo.mysql.agent.RemoveTag", "gorm", map[string]interface{}{
			"operation": "remove_tag",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.RemoveTag",
			"agent_id":  agentID,
			"tag_id":    tagID,
		})
		return fmt.Errorf("invalid input")
	}

	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.RemoveTag", "gorm", map[string]interface{}{
			"operation": "remove_tag",
			"option":    "agentRepository.RemoveTag",
			"func_name": "repo.agent.RemoveTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return fmt.Errorf("agent does not exist: %s", agentID)
	}

	if !agent.HasTag(tagID) {
		logger.LogInfo("Agent does not have tag, skip remove", "", 0, "", "repo.agent.RemoveTag", "gorm", map[string]interface{}{
			"operation": "remove_tag",
			"option":    "agentRepository.RemoveTag",
			"func_name": "repo.agent.RemoveTag",
			"agentID":   agentID,
			"tagID":     tagID,
			"message":   "tag_not_exists",
		})
		return nil // 不存在不算错误，直接返回成功
	}

	agent.RemoveTag(tagID)

	// 更新数据库中的tags字段
	if err := r.db.Model(&agent).Select("tags").Updates(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.RemoveTag", "gorm", map[string]interface{}{
			"operation": "remove_tag",
			"option":    "agentRepository.RemoveTag",
			"func_name": "repo.agent.RemoveTag",
			"agentID":   agentID,
			"tagID":     tagID,
		})
		return fmt.Errorf("update agent tags failed: %v", err)
	}

	logger.LogInfo("Tag removed from agent successfully", "", 0, "", "repo.agent.RemoveTag", "gorm", map[string]interface{}{
		"operation": "remove_tag",
		"option":    "agentRepository.RemoveTag",
		"func_name": "repo.agent.RemoveTag",
		"agentID":   agentID,
		"tagID":     tagID,
		"status":    "success",
	})

	return nil
}

// 判断Agent是否拥有某个标签
func (r *agentRepository) HasTag(agentID string, tagID string) bool {
	// 1. 输入验证 - 避免无效查询
	if agentID == "" || tagID == "" {
		logger.LogInfo("Invalid input", "", 0, "", "repo.agent.HasTag", "gorm", map[string]interface{}{
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
		logger.LogError(err, "", 0, "", "repo.agent.HasTag", "gorm", map[string]interface{}{
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

// 获取Agent的所有标签ID列表
func (r *agentRepository) GetTags(agentID string) []string {
	// 1. 输入验证：避免空值查询
	if agentID == "" {
		logger.LogError(nil, "", 0, "", "repo.agent.GetTags", "gorm", map[string]interface{}{
			"operation": "get_tags",
			"option":    "agentRepository.GetTags",
			"func_name": "repo.agent.GetTags",
			"agentID":   agentID,
			"error":     "agentID is empty",
		})
		return []string{} // 返回空切片而非nil
	}

	// 2. 检查Agent是否存在并获取完整信息
	var agent agentModel.Agent
	if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.agent.GetTags", "gorm", map[string]interface{}{
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
