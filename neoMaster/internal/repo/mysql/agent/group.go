/**
 * @author: Sun977
 * @date: 2025.10.14
 * @description: Agent 分组管理实现
 * @func: 提供Agent分组的CRUD操作，不包含业务逻辑
 * - IsValidGroupId/Name 验证分组ID/名称是否存在
 * - IsAgentInGroup 判断Agent是否在分组中
 * - GetGroupByGID 获取分组详情
 * - GetGroupList 获取分组列表
 * - CreateGroup 创建分组
 * - UpdateGroup 更新分组
 * - DeleteGroup 删除分组
 * - SetGroupStatus 设置分组状态
 * - AddAgentToGroup 添加Agent到分组
 * - RemoveAgentFromGroup 从分组移除Agent
 * - GetAgentsInGroup 获取分组中的Agent列表
 */
package agent

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/pkg/logger"
)

// 默认分组别名集合：用于兼容历史数据
var defaultGroupAliases = []string{"default"}

// isDefaultGroup 根据分组对象判断是否为默认分组（包含别名）
func isDefaultGroup(g *agentModel.AgentGroup) bool {
	if g == nil {
		return false
	}
	if g.IsSystem || g.Name == "default" {
		return true
	}
	for _, id := range defaultGroupAliases {
		if g.GroupID == id {
			return true
		}
	}
	return false
}

// IsValidGroupId 判断分组ID是否有效
func (r *agentRepository) IsValidGroupId(groupID string) bool {
	var count int64
	err := r.db.Model(&agentModel.AgentGroup{}).Where("group_id = ?", groupID).Count(&count).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.IsValidGroupId", "", map[string]interface{}{
			"operation": "is_valid_group_id",
			"option":    "db.Count(agent_groups.by_group_id)",
			"func_name": "repo.mysql.agent.IsValidGroupId",
			"group_id":  groupID,
		})
		return false
	}
	return count > 0
}

// IsValidGroupName 判断分组名称是否有效
func (r *agentRepository) IsValidGroupName(groupName string) bool {
	var count int64
	err := r.db.Model(&agentModel.AgentGroup{}).Where("name = ?", groupName).Count(&count).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.IsValidGroupName", "", map[string]interface{}{
			"operation": "is_valid_group_name",
			"option":    "db.Count(agent_groups.by_name)",
			"func_name": "repo.mysql.agent.IsValidGroupName",
			"name":      groupName,
		})
		return false
	}
	return count > 0
}

// IsAgentInGroup 判断Agent是否在分组中
func (r *agentRepository) IsAgentInGroup(agentID string, groupID string) bool {
	// 直接查询数据库中该Agent是否在指定分组中 agent_group_members
	var count int64
	err := r.db.Model(&agentModel.AgentGroupMember{}).Where("agent_id = ? AND group_id = ?", agentID, groupID).Count(&count).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.IsAgentInGroup", "", map[string]interface{}{
			"operation": "is_agent_in_group",
			"option":    "db.Count(agent_group_members)",
			"func_name": "repo.mysql.agent.IsAgentInGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return false
	}
	return count > 0
}

// GetGroupByGID 获取分组详情
func (r *agentRepository) GetGroupByGID(groupID string) (*agentModel.AgentGroup, error) {
	var group agentModel.AgentGroup
	err := r.db.Where("group_id = ?", groupID).First(&group).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.GetGroupByGID", "", map[string]interface{}{
			"operation": "get_group_by_gid",
			"option":    "db.First(agent_groups)",
			"func_name": "repo.mysql.agent.GetGroupByGID",
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
func (r *agentRepository) GetGroupList(page, pageSize int, tags []string, status int, keywords string) ([]*agentModel.AgentGroup, int64, error) {
	// - agent_groups 表为分组信息的单表存储，Tags 使用 JSON 数组存储（[]string）。
	// - 过滤策略：
	//   1) tags 采用 AND 逻辑：当提供多个标签时，分组必须同时包含这些标签；（tag没有指定专门的标签表，标签是字符串）
	//   2) status 采用 0/1 逻辑：0 表示禁用，1 表示启用。
	//   3) keywords 采用 OR 模糊查询：匹配 group_id、name、description 三个字段。
	// - 分页策略：Offset + Limit，并以 updated_at DESC, id DESC 进行稳定排序。
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
	if status == 1 || status == 0 {
		query = query.Where("status = ?", status)
	}
	if keywords != "" {
		like := "%" + keywords + "%"
		// AgentGroup 字段对齐：使用 name / group_id / description 进行关键词过滤
		query = query.Where("group_id LIKE ? OR name LIKE ? OR description LIKE ?", like, like, like)
	}
	if len(tags) > 0 {
		// 改为 JSON 字段过滤（每个 tag 至少匹配一次）
		for _, t := range tags {
			query = query.Where("JSON_SEARCH(tags, 'one', ?) IS NOT NULL", t)
			// // 注意：JSON_CONTAINS 期望第二参数为合法 JSON 文本，这里为字符串需加双引号
			// query = query.Where("JSON_CONTAINS(tags, ?)", `"`+tag+`"`)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.GetGroupList", "", map[string]interface{}{
			"operation": "get_group_list",
			"option":    "db.Count(agent_groups)",
			"func_name": "repo.mysql.agent.GetGroupList",
			"page":      page,
			"page_size": pageSize,
			"tags":      tags,
			"status":    status,
			"keywords":  keywords,
		})
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize
	// 分页获取数据
	if err := query.Offset(offset).Limit(pageSize).Order("updated_at DESC").Find(&groups).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.GetGroupList", "", map[string]interface{}{
			"operation": "get_group_list",
			"option":    "db.Find(agent_groups)",
			"func_name": "repo.mysql.agent.GetGroupList",
			"page":      page,
			"page_size": pageSize,
			"tags":      tags,
			"status":    status,
			"keywords":  keywords,
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
	// 参数验证
	if group == nil || group.GroupID == "" || group.Name == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.CreateGroup", "", map[string]interface{}{
			"operation": "create_group",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.CreateGroup",
			"message":   "invalid input parameters",
		})
		return gorm.ErrInvalidData
	}

	if group.IsSystem {
		// 普通创建禁止创建系统分组，强制回退为非系统分组
		group.IsSystem = false
	}
	// 如果未设置状态，默认为启用
	if group.Status != 0 && group.Status != 1 {
		group.Status = 1
	}

	// 插入数据库
	if err := r.db.Create(group).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.CreateGroup", "", map[string]interface{}{
			"operation": "create_group",
			"option":    "db.Create(agent_groups)",
			"func_name": "repo.mysql.agent.CreateGroup",
			"group":     group,
		})
		return err
	}
	// 成功日志
	logger.LogInfo("Group created successfully", "", 0, "", "repo.mysql.agent.CreateGroup", "", map[string]interface{}{
		"operation": "create_group",
		"option":    "result.success",
		"func_name": "repo.mysql.agent.CreateGroup",
		"group":     group,
	})
	return nil
}

// UpdateGroup 更新Agent分组
// 采用“定向部分更新”：按业务唯一键 group_id 精确定位并更新安全字段，避免 Save 带来的主键依赖与全字段覆盖风险。
// Save 方法会更新所有字段,包括未被修改的字段(save 如果存在则更新不存在则创建,依赖主键)。Update 方法只更新被修改的字段,不依赖主键，这里使用 Update 方法。
func (r *agentRepository) UpdateGroup(groupID string, group *agentModel.AgentGroup) (*agentModel.AgentGroup, error) {
	if groupID == "" || group == nil {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.UpdateGroup", "", map[string]interface{}{
			"operation": "update_group",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.UpdateGroup",
			"message":   "invalid input parameters",
		})
		return nil, gorm.ErrInvalidData
	}

	// 默认/系统分组保护：不允许更改 group_id/is_system
	var target agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&target).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.UpdateGroup", "", map[string]interface{}{
			"operation": "update_group",
			"option":    "db.First(agent_groups)",
			"func_name": "repo.mysql.agent.UpdateGroup",
			"group_id":  groupID,
		})
		return nil, err
	}
	if isDefaultGroup(&target) {
		// 允许更新 remark/状态等非关键字段
		// 如果尝试改变关键字段，直接报错
		if group.GroupID != "" && group.GroupID != target.GroupID {
			return nil, fmt.Errorf("cannot change group_id of system/default group")
		}
		// if group.Name != "" && group.Name != target.Name {
		// 	return fmt.Errorf("cannot change name of system/default group")
		// }
		if group.IsSystem != target.IsSystem {
			return nil, fmt.Errorf("cannot change is_system of system/default group")
		}
	}

	// 构建“部分更新（PATCH语义）”的更新字段集合：
	// - description/status/updated_at 始终更新（由业务层控制是否传递期望值）
	// - tags 为 JSON 字段，仅当传入非 nil 时更新，避免清空数据库已有 JSON；
	//   这样可以区分“未提供（nil）不更新”与“显式清空（[]string{}）更新为空数组”的不同意图。
	updates := map[string]interface{}{
		"name":        group.Name,
		"description": group.Description,
		"status":      group.Status,
		"updated_at":  time.Now(),
	}
	if group.Tags != nil {
		updates["tags"] = group.Tags
	}

	if err := r.db.Model(&agentModel.AgentGroup{}).Where("group_id = ?", groupID).Updates(updates).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.UpdateGroup", "", map[string]interface{}{
			"operation": "update_group",
			"option":    "db.Updates(agent_groups)",
			"func_name": "repo.mysql.agent.UpdateGroup",
			"group_id":  groupID,
			"group":     group,
			"updates":   updates,
		})
		return nil, err
	}
	// 成功日志
	logger.LogInfo("Group updated successfully", "", 0, "", "repo.mysql.agent.UpdateGroup", "", map[string]interface{}{
		"operation": "update_group",
		"option":    "result.success",
		"func_name": "repo.mysql.agent.UpdateGroup",
		"group_id":  groupID,
		"group":     group,
		"updates":   updates,
	})

	// 返回更新后的分组信息
	result := &agentModel.AgentGroup{
		GroupID:     groupID,
		Name:        group.Name,
		Description: group.Description,
		Status:      group.Status,
	}
	// 补充时间戳：CreatedAt 从原记录继承，UpdatedAt 为当前时间
	result.CreatedAt = target.CreatedAt
	result.UpdatedAt = time.Now()
	if group.Tags != nil {
		result.Tags = group.Tags
	}
	return result, nil
}

// DeleteGroup 删除Agent分组（增强默认分组/系统分组保护与别名兼容）
// 删除后组内的Agent自动移到默认分组
// 理论上不允许删除默认分组,但是如果默认分组被删或者不存在,程序会自动创建一个默认分组,默认分组名称固定为 'default'
func (r *agentRepository) DeleteGroup(groupID string) error {
	// 参数验证
	if groupID == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
			"operation": "delete_group",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.DeleteGroup",
			"message":   "group_id is required",
		})
		return gorm.ErrInvalidData
	}

	// 查询待删除分组
	var target agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&target).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
			"operation": "delete_group",
			"option":    "db.First(agent_groups)",
			"func_name": "repo.mysql.agent.DeleteGroup",
			"group_id":  groupID,
		})
		return err
	}
	// 系统/默认分组保护
	if isDefaultGroup(&target) {
		err := fmt.Errorf("DeleteGroup: 不允许删除默认分组(%s/%s)", target.GroupID, target.Name)
		logger.LogError(err, "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
			"operation":  "delete_group",
			"option":     "protect.system_group",
			"func_name":  "repo.mysql.agent.DeleteGroup",
			"group_id":   groupID,
			"group_name": target.Name,
			"message":    "delete default group is not allowed",
		})
		return err
	}

	// 采用事务：
	// 1) 确保默认分组存在（优先 is_system=true；若不存在回退 group_id 在 defaultGroupAliases 内；仍不存在则创建默认分组）；
	// 2) 将待删除分组内的成员迁移到默认分组（INSERT IGNORE + SELECT，避免唯一索引冲突）；
	// 3) 删除分组记录；
	// 这样可以在不打断操作的前提下保持数据一致性与可用性。

	var defaultGroupIDForLog string
	txErr := r.db.Transaction(func(tx *gorm.DB) error {
		// 1) 确保默认分组存在
		var defaultGroup agentModel.AgentGroup
		findDefaultBySystem := tx.Where("is_system = ?", true).First(&defaultGroup).Error
		if findDefaultBySystem != nil {
			// 回退：按别名集合查找（兼容 default 等别名）
			findDefaultByAlias := tx.Where("group_id IN ?", defaultGroupAliases).First(&defaultGroup).Error
			if findDefaultByAlias != nil {
				// 均未找到则创建默认分组
				newDefault := &agentModel.AgentGroup{
					GroupID:     "default",
					Name:        "default",
					Description: "默认分组(禁止删除)",
					Tags:        []string{"default"},
					IsSystem:    true,
					Status:      1,
				}
				if createErr := tx.Create(newDefault).Error; createErr != nil {
					// 并发场景可能触发唯一约束错误，记录后重查一次
					logger.LogError(createErr, "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
						"operation": "delete_group",
						"option":    "tx.create_default_group",
						"func_name": "repo.mysql.agent.DeleteGroup",
						"group_id":  groupID,
					})
					// 重查优先 is_system=true，再回退别名
					if err2 := tx.Where("is_system = ?", 1).First(&defaultGroup).Error; err2 != nil {
						if err3 := tx.Where("group_id IN ?", defaultGroupAliases).First(&defaultGroup).Error; err3 != nil {
							return createErr
						}
					}
				} else {
					defaultGroup = *newDefault
				}
			}
		}

		defaultGroupIDForLog = defaultGroup.GroupID

		// 2) 迁移成员到默认分组
		moveRes := tx.Exec(
			"INSERT IGNORE INTO agent_group_members (agent_id, group_id, joined_at, created_at, updated_at) "+
				"SELECT agent_id, ?, NOW(), NOW(), NOW() FROM agent_group_members WHERE group_id = ?",
			defaultGroup.GroupID, groupID,
		)
		if moveRes.Error != nil {
			logger.LogError(moveRes.Error, "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
				"operation":        "delete_group",
				"option":           "tx.move_members_to_default",
				"func_name":        "repo.mysql.agent.DeleteGroup",
				"group_id":         groupID,
				"default_group_id": defaultGroup.GroupID,
			})
			return moveRes.Error
		}

		// 3) 删除分组（若并发删除导致未找到，返回 RecordNotFound） ，外键约束 ON DELETE CASCADE 会同步删除原分组的成员关系
		delRes := tx.Where("group_id = ?", groupID).Delete(&agentModel.AgentGroup{})
		if delRes.Error != nil {
			logger.LogError(delRes.Error, "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
				"operation":        "delete_group",
				"option":           "tx.delete_group",
				"func_name":        "repo.mysql.agent.DeleteGroup",
				"group_id":         groupID,
				"default_group_id": defaultGroup.GroupID,
			})
			return delRes.Error
		}
		if delRes.RowsAffected == 0 {
			// 极端情况下（并发删除）返回未找到
			err := gorm.ErrRecordNotFound
			logger.LogError(err, "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
				"operation":        "delete_group",
				"option":           "tx.delete_group.not_found",
				"func_name":        "repo.mysql.agent.DeleteGroup",
				"group_id":         groupID,
				"default_group_id": defaultGroup.GroupID,
			})
			return err
		}

		// 事务内成功日志（记录迁移数量）
		logger.LogInfo("分组成员迁移完成，准备提交删除事务", "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
			"operation":        "delete_group",
			"option":           "tx_move",
			"func_name":        "repo.mysql.agent.DeleteGroup",
			"group_id":         groupID,
			"default_group_id": defaultGroup.GroupID,
			"moved_count":      moveRes.RowsAffected,
		})

		return nil
	})
	if txErr != nil {
		return txErr
	}

	// 成功日志（事务提交后）
	logger.LogInfo("Agent分组删除成功，组内成员已迁移至默认分组", "", 0, "", "repo.mysql.agent.DeleteGroup", "", map[string]interface{}{
		"operation":        "delete_group",
		"option":           "result.success",
		"func_name":        "repo.mysql.agent.DeleteGroup",
		"group_id":         groupID,
		"default_group_id": defaultGroupIDForLog,
	})

	return nil
}

// SetGroupStatus 设置分组状态（增强默认/系统分组保护与别名兼容）
func (r *agentRepository) SetGroupStatus(groupID string, status int) error {
	if groupID == "" || (status != 0 && status != 1) {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.SetGroupStatus", "", map[string]interface{}{
			"operation": "set_group_status",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.SetGroupStatus",
			"group_id":  groupID,
			"status":    status,
			"message":   "Invalid input: group_id is empty or status is invalid (0 or 1)",
		})
		return gorm.ErrInvalidData
	}

	var group agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&group).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.SetGroupStatus", "", map[string]interface{}{
			"operation": "set_group_status",
			"option":    "db.First(agent_groups)",
			"func_name": "repo.mysql.agent.SetGroupStatus",
			"group_id":  groupID,
			"status":    status,
			"message":   "Group not found",
		})
		return err
	}
	if isDefaultGroup(&group) {
		logger.LogWarn("Attempt to disable system/default group is forbidden", "", 0, "", "repo.mysql.agent.SetGroupStatus", "", map[string]interface{}{
			"operation": "set_group_status",
			"option":    "protect.system_group",
			"func_name": "repo.mysql.agent.SetGroupStatus",
			"group_id":  groupID,
			"status":    status,
		})
		return gorm.ErrInvalidData
	}

	// 幂等：若状态没有变化则直接返回
	if group.Status == uint8(status) {
		logger.LogInfo("Group status unchanged, skip", "", 0, "", "repo.mysql.agent.SetGroupStatus", "", map[string]interface{}{
			"operation": "set_group_status",
			"option":    "idempotent.skip",
			"func_name": "repo.mysql.agent.SetGroupStatus",
			"group_id":  groupID,
			"status":    status,
		})
		return nil
	}

	// 执行更新
	if err := r.db.Model(&agentModel.AgentGroup{}).Where("group_id = ?", groupID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.SetGroupStatus", "", map[string]interface{}{
			"operation":  "set_group_status",
			"option":     "db.Updates(agent_groups)",
			"func_name":  "repo.mysql.agent.SetGroupStatus",
			"group_id":   groupID,
			"old_status": group.Status,
			"new_status": status,
		})
		return err
	}
	logger.LogInfo("Group status updated successfully", "", 0, "", "repo.mysql.agent.SetGroupStatus", "", map[string]interface{}{
		"operation":  "set_group_status",
		"option":     "result.success",
		"func_name":  "repo.mysql.agent.SetGroupStatus",
		"group_id":   groupID,
		"old_status": group.Status,
		"new_status": status,
	})
	return nil
}

// AddAgentToGroup 将Agent添加到分组
func (r *agentRepository) AddAgentToGroup(agentID string, groupID string) (*agentModel.AgentGroupMember, error) {
	// 将 Agent 加入到指定分组
	// 1) 分组必须存在；
	// 2) 分组若为禁用状态（status=0），不允许新增成员，以避免向“冻结”分组添加数据；
	// 3) Agent 必须存在；
	// 4) 若 Agent 已在该分组中，操作幂等化，直接返回成功；
	// 5) 使用唯一键约束(agent_id, group_id)保证并发安全
	if agentID == "" || groupID == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.AddAgentToGroup", "", map[string]interface{}{
			"operation": "add_agent_to_group",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.AddAgentToGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return nil, gorm.ErrInvalidData
	}

	// // 校验分组存在性 - 上层已经校验
	// var group agentModel.AgentGroup
	// if err := r.db.Where("group_id = ?", groupID).First(&group).Error; err != nil {
	// 	logger.LogError(err, "", 0, "", "repo.mysql.agent.AddAgentToGroup", "", map[string]interface{}{
	// 		"operation": "add_agent_to_group",
	// 		"option":    "db.First(agent_groups)",
	// 		"func_name": "repo.mysql.agent.AddAgentToGroup",
	// 		"group_id":  groupID,
	// 		"agent_id":  agentID,
	// 	})
	// 	return err
	// }
	// // 禁用分组不可添加成员
	// if group.Status == 0 {
	// 	logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.AddAgentToGroup", "", map[string]interface{}{
	// 		"operation":    "add_agent_to_group",
	// 		"option":       "validate.group_status",
	// 		"func_name":    "repo.mysql.agent.AddAgentToGroup",
	// 		"group_id":     groupID,
	// 		"agent_id":     agentID,
	// 		"group_status": group.Status,
	// 	})
	// 	return gorm.ErrInvalidData
	// }
	// // 校验Agent存在性
	// var agent agentModel.Agent
	// if err := r.db.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
	// 	logger.LogError(err, "", 0, "", "repo.mysql.agent.AddAgentToGroup", "", map[string]interface{}{
	// 		"operation": "add_agent_to_group",
	// 		"option":    "db.First(agents)",
	// 		"func_name": "repo.mysql.agent.AddAgentToGroup",
	// 		"agent_id":  agentID,
	// 		"group_id":  groupID,
	// 	})
	// 	return err
	// }
	// // 幂等：存在关系则忽略(若已在分组中，直接返回成功)
	// if r.IsAgentInGroup(agentID, groupID) {
	// 	logger.LogInfo("agent already in group, skip", "", 0, "", "repo.agent.AddAgentToGroup", "", map[string]interface{}{
	// 		"operation": "add_agent_to_group",
	// 		"option":    "agentRepository.AddAgentToGroup.idempotent_skip",
	// 		"func_name": "repo.agent.AddAgentToGroup",
	// 		"agent_id":  agentID,
	// 		"group_id":  groupID,
	// 	})
	// 	return nil
	// }

	// 创建分组成员关系记录
	member := &agentModel.AgentGroupMember{AgentID: agentID, GroupID: groupID, JoinedAt: time.Now()}
	if err := r.db.Create(member).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.AddAgentToGroup", "", map[string]interface{}{
			"operation": "add_agent_to_group",
			"option":    "db.Create(agent_group_members)",
			"func_name": "repo.mysql.agent.AddAgentToGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return nil, err
	}
	logger.LogInfo("Agent added to group", "", 0, "", "repo.mysql.agent.AddAgentToGroup", "", map[string]interface{}{
		"operation": "add_agent_to_group",
		"option":    "result.success",
		"func_name": "repo.mysql.agent.AddAgentToGroup",
		"agent_id":  agentID,
		"group_id":  groupID,
		// "group_status":    group.Status,
		// "is_system_group": group.IsSystem,
	})

	// 装填分组成员关系数据,返回给调用方
	return member, nil
}

// RemoveAgentFromGroup 从分组中移除Agent
func (r *agentRepository) RemoveAgentFromGroup(agentID string, groupID string) error {
	// 从指定分组移除 Agent
	// 设计约束：
	// 1) 操作幂等化：若成员关系不存在，直接返回成功；
	// 2) 遵循外键约束：若分组或 Agent 不存在，成员关系也不存在；
	// 3) 删除使用条件删除，避免误删其他数据；
	if agentID == "" || groupID == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.RemoveAgentFromGroup", "", map[string]interface{}{
			"operation": "remove_agent_from_group",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.RemoveAgentFromGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return gorm.ErrInvalidData
	}

	// 校验关系是否存在
	var count int64
	if err := r.db.Model(&agentModel.AgentGroupMember{}).Where("agent_id = ? AND group_id = ?", agentID, groupID).Count(&count).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.RemoveAgentFromGroup", "", map[string]interface{}{
			"operation": "remove_agent_from_group",
			"option":    "db.Count(agent_group_members)",
			"func_name": "repo.mysql.agent.RemoveAgentFromGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return err
	}
	if count == 0 {
		// 幂等返回：成员关系不存在
		logger.LogInfo("Relation not exists, skip remove", "", 0, "", "repo.mysql.agent.RemoveAgentFromGroup", "", map[string]interface{}{
			"operation": "remove_agent_from_group",
			"option":    "idempotent.skip",
			"func_name": "repo.mysql.agent.RemoveAgentFromGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return nil
	}

	// 执行删除
	if err := r.db.Where("agent_id = ? AND group_id = ?", agentID, groupID).Delete(&agentModel.AgentGroupMember{}).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.RemoveAgentFromGroup", "", map[string]interface{}{
			"operation": "remove_agent_from_group",
			"option":    "db.Delete(agent_group_members)",
			"func_name": "repo.mysql.agent.RemoveAgentFromGroup",
			"agent_id":  agentID,
			"group_id":  groupID,
		})
		return err
	}
	logger.LogInfo("Agent removed from group", "", 0, "", "repo.mysql.agent.RemoveAgentFromGroup", "", map[string]interface{}{
		"operation": "remove_agent_from_group",
		"option":    "result.success",
		"func_name": "repo.mysql.agent.RemoveAgentFromGroup",
		"agent_id":  agentID,
		"group_id":  groupID,
	})
	return nil
}

// GetAgentsInGroup 获取分组中的Agent列表（分页）
func (r *agentRepository) GetAgentsInGroup(page, pageSize int, groupID string) ([]*agentModel.Agent, int64, error) {
	if groupID == "" {
		logger.LogError(gorm.ErrInvalidData, "", 0, "", "repo.mysql.agent.GetAgentsInGroup", "", map[string]interface{}{
			"operation": "get_agents_in_group",
			"option":    "validate.input",
			"func_name": "repo.mysql.agent.GetAgentsInGroup",
			"group_id":  groupID,
		})
		return nil, 0, gorm.ErrInvalidData
	}

	// 校验分组存在性
	var group agentModel.AgentGroup
	if err := r.db.Where("group_id = ?", groupID).First(&group).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.GetAgentsInGroup", "", map[string]interface{}{
			"operation": "get_agents_in_group",
			"option":    "db.First(agent_groups)",
			"func_name": "repo.mysql.agent.GetAgentsInGroup",
			"group_id":  groupID,
		})
		return nil, 0, err
	}

	var agents []*agentModel.Agent
	var total int64

	// 统计数量(统计成员数量)
	if err := r.db.Model(&agentModel.AgentGroupMember{}).Where("group_id = ?", groupID).Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.GetAgentsInGroup", "", map[string]interface{}{
			"operation": "get_agents_in_group",
			"option":    "db.Count(agent_group_members)",
			"func_name": "repo.mysql.agent.GetAgentsInGroup",
			"group_id":  groupID,
		})
		return nil, 0, err
	}

	// JOIN 查询成员详细信息
	agentsTable := agentModel.Agent{}.TableName() // "agents"
	if err := r.db.Table(agentsTable).
		Select(agentsTable+".*").
		Joins("JOIN agent_group_members gm ON gm.agent_id = "+agentsTable+".agent_id").
		Where("gm.group_id = ?", groupID).
		Order(agentsTable + ".updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&agents).Error; err != nil {
		logger.LogError(err, "", 0, "", "repo.mysql.agent.GetAgentsInGroup", "", map[string]interface{}{
			"operation": "get_agents_in_group",
			"option":    "db.Join(agent_group_members->agents)",
			"func_name": "repo.mysql.agent.GetAgentsInGroup",
			"group_id":  groupID,
		})
		return nil, 0, err
	}

	// 成功日志
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
