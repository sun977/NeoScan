package tag_system

import (
	"context"
	"encoding/json"
	"fmt"

	"neomaster/internal/pkg/utils"

	"gorm.io/gorm"

	assetModel "neomaster/internal/model/asset"
	"neomaster/internal/model/orchestrator"
	"neomaster/internal/model/tag_system"
	"neomaster/internal/pkg/matcher"
	repo "neomaster/internal/repo/mysql/tag_system"
)

const (
	TaskCategorySystem        = "system"              // 系统级任务
	ToolNameSysTagPropagation = "sys_tag_propagation" // 标签传播任务, 用于自动标签传播
)

// TagPropagationPayload 定义标签传播任务的参数载荷
type TagPropagationPayload struct {
	TargetType string            `json:"target_type"` // host, web, network
	Action     string            `json:"action"`      // add, remove
	Rule       matcher.MatchRule `json:"rule"`        // 匹配规则
	RuleID     uint64            `json:"rule_id"`     // 规则ID (用于关联表)
	Tags       []string          `json:"tags"`        // 标签名称列表 (LocalAgent 使用)
	TagIDs     []uint64          `json:"tag_ids"`     // 关联的标签ID列表 (备用)
}

type TagService interface {
	// --- 标签 CRUD ---
	CreateTag(ctx context.Context, tag *tag_system.SysTag) error
	GetTag(ctx context.Context, id uint64) (*tag_system.SysTag, error)
	GetTagByName(ctx context.Context, name string) (*tag_system.SysTag, error) // 通过名称获取标签
	UpdateTag(ctx context.Context, tag *tag_system.SysTag) error
	DeleteTag(ctx context.Context, id uint64) error
	ListTags(ctx context.Context, req *tag_system.ListTagsRequest) ([]tag_system.SysTag, int64, error)

	// --- 规则 Rules CRUD ---
	CreateRule(ctx context.Context, rule *tag_system.SysMatchRule) error
	UpdateRule(ctx context.Context, rule *tag_system.SysMatchRule) error
	DeleteRule(ctx context.Context, id uint64) error
	GetRule(ctx context.Context, id uint64) (*tag_system.SysMatchRule, error)
	ListRules(ctx context.Context, req *tag_system.ListRulesRequest) ([]tag_system.SysMatchRule, int64, error)

	// --- Auto Tagging ---
	AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error // 添加标签

	// --- 标签扩散 Propagation ---
	SubmitPropagationTask(ctx context.Context, ruleID uint64, action string) (string, error)                                             // 提交标签传播任务
	SubmitEntityPropagationTask(ctx context.Context, entityType string, entityID uint64, tagIDs []uint64, action string) (string, error) // 提交标签扩散任务

	// --- 状态调和 (State Reconciliation) ---
	// SyncEntityTags 全量同步实体的标签。
	// 策略：
	// 1. 找出该实体所有 Source = scope 的标签。
	// 2. 与传入的 targetTagIDs 进行对比。
	// 3. 新增 targetTagIDs 中有但数据库中没有的。
	// 4. 删除数据库中有但 targetTagIDs 中没有的。
	// 5. 忽略 Source != scope 的标签 (保持不变)。
	SyncEntityTags(ctx context.Context, entityType string, entityID string, targetTagIDs []uint64, sourceScope string, ruleID uint64) error

	// --- 系统初始化 Bootstrap ---
	// BootstrapSystemTags 初始化系统预设标签骨架 (Root, System, AgentGroup 等)
	BootstrapSystemTags(ctx context.Context) error
}

type tagService struct {
	repo repo.TagRepository
	db   *gorm.DB // 用于直接插入任务，或者需要事务
}

func NewTagService(repo repo.TagRepository, db *gorm.DB) TagService {
	return &tagService{
		repo: repo,
		db:   db,
	}
}

// --- Basic CRUD Implementation ---

func (s *tagService) CreateTag(ctx context.Context, tag *tag_system.SysTag) error {
	// 简单的路径计算逻辑 (实际项目中可能需要锁或事务来保证路径一致性)
	if tag.ParentID != 0 {
		parent, err := s.repo.GetTagByID(tag.ParentID)
		if err != nil {
			return err
		}
		tag.Path = fmt.Sprintf("%s%d/", parent.Path, tag.ParentID)
		tag.Level = parent.Level + 1
	} else {
		tag.Path = "/"
		tag.Level = 0
	}
	return s.repo.CreateTag(tag)
}

func (s *tagService) GetTag(ctx context.Context, id uint64) (*tag_system.SysTag, error) {
	return s.repo.GetTagByID(id)
}

func (s *tagService) GetTagByName(ctx context.Context, name string) (*tag_system.SysTag, error) {
	return s.repo.GetTagByName(name)
}

func (s *tagService) UpdateTag(ctx context.Context, tag *tag_system.SysTag) error {
	return s.repo.UpdateTag(tag)
}

func (s *tagService) DeleteTag(ctx context.Context, id uint64) error {
	// TODO: 检查是否有子标签或已关联实体
	return s.repo.DeleteTag(id)
}

func (s *tagService) ListTags(ctx context.Context, req *tag_system.ListTagsRequest) ([]tag_system.SysTag, int64, error) {
	return s.repo.ListTags(req)
}

// --- Rules Implementation ---

func (s *tagService) CreateRule(ctx context.Context, rule *tag_system.SysMatchRule) error {
	// 验证 RuleJSON 格式
	if _, err := matcher.ParseJSON(rule.RuleJSON); err != nil {
		return fmt.Errorf("invalid rule json: %v", err)
	}
	if err := s.repo.CreateRule(rule); err != nil {
		return err
	}

	// 触发标签传播任务 (Backfill) - 已移除，改为手动触发
	// if rule.IsEnabled {
	// 	if taskID, err := s.SubmitPropagationTask(ctx, rule.ID, "add"); err != nil {
	// 		logger.LogBusinessError(err, "", 0, "", "service.tag_system.CreateRule", "POST", map[string]interface{}{
	// 			"rule_id": rule.ID,
	// 			"action":  "add",
	// 		})
	// 	} else {
	// 		logger.LogBusinessOperation("tag_propagation_submitted", 0, "", "", "", "success", "Tag propagation task submitted", map[string]interface{}{
	// 			"rule_id": rule.ID,
	// 			"task_id": taskID,
	// 			"action":  "add",
	// 		})
	// 	}
	// }

	return nil
}

func (s *tagService) UpdateRule(ctx context.Context, rule *tag_system.SysMatchRule) error {
	if _, err := matcher.ParseJSON(rule.RuleJSON); err != nil {
		return fmt.Errorf("invalid rule json: %v", err)
	}
	if err := s.repo.UpdateRule(rule); err != nil {
		return err
	}

	// 触发标签传播任务 (Backfill) - 已移除，改为手动触发
	// if rule.IsEnabled {
	// 	if taskID, err := s.SubmitPropagationTask(ctx, rule.ID, "add"); err != nil {
	// 		logger.LogBusinessError(err, "", 0, "", "service.tag_system.UpdateRule", "PUT", map[string]interface{}{
	// 			"rule_id": rule.ID,
	// 			"action":  "add",
	// 		})
	// 	} else {
	// 		logger.LogBusinessOperation("tag_propagation_submitted", 0, "", "", "", "success", "Tag propagation task submitted", map[string]interface{}{
	// 			"rule_id": rule.ID,
	// 			"task_id": taskID,
	// 			"action":  "add",
	// 		})
	// 	}
	// }
	return nil
}

func (s *tagService) DeleteRule(ctx context.Context, id uint64) error {
	// 1. 获取规则详情 (为了触发移除任务)
	// 忽略错误，因为如果规则不存在，删除操作也会失败或者无所谓
	// if rule, err := s.repo.GetRuleByID(id); err == nil {
	// 	// 触发移除任务 - 已移除，改为手动触发
	// 	if taskID, err := s.SubmitPropagationTask(ctx, rule.ID, "remove"); err != nil {
	// 		logger.LogBusinessError(err, "", 0, "", "service.tag_system.DeleteRule", "DELETE", map[string]interface{}{
	// 			"rule_id": rule.ID,
	// 			"action":  "remove",
	// 		})
	// 	} else {
	// 		logger.LogBusinessOperation("tag_propagation_submitted", 0, "", "", "", "success", "Tag propagation task submitted", map[string]interface{}{
	// 			"rule_id": rule.ID,
	// 			"task_id": taskID,
	// 			"action":  "remove",
	// 		})
	// 	}
	// }

	return s.repo.DeleteRule(id)
}

func (s *tagService) GetRule(ctx context.Context, id uint64) (*tag_system.SysMatchRule, error) {
	return s.repo.GetRuleByID(id)
}

func (s *tagService) ListRules(ctx context.Context, req *tag_system.ListRulesRequest) ([]tag_system.SysMatchRule, int64, error) {
	return s.repo.ListRules(req)
}

// --- Auto Tagging Implementation (Moved to auto_tag.go or here) ---
// 为了保持文件简洁，AutoTag 和 SubmitPropagationTask 可以放在单独文件，或者这里
// 这里先放这里，如果太长再拆分
func (s *tagService) AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error {
	// 1. 获取该实体类型的所有启用规则
	rules, err := s.repo.GetRulesByEntityType(entityType)
	if err != nil {
		return err
	}

	if len(rules) == 0 {
		return nil
	}

	// 2. 遍历规则进行匹配
	var matchedTagIDs []uint64
	var matchedRuleIDs []uint64

	for _, ruleRecord := range rules {
		// 解析规则
		rule, err1 := matcher.ParseJSON(ruleRecord.RuleJSON)
		if err1 != nil {
			// 记录错误但继续处理其他规则
			continue
		}

		// 执行匹配
		matched, err2 := matcher.Match(attributes, rule)
		if err2 != nil {
			continue
		}

		if matched {
			// 匹配成功,记录标签ID和规则ID成列表
			matchedTagIDs = append(matchedTagIDs, ruleRecord.TagID)
			matchedRuleIDs = append(matchedRuleIDs, ruleRecord.ID)
		}
	}

	// 3. 更新实体标签 (State Reconciliation)
	// 使用通用的 SyncEntityTags 方法
	// 注意：AutoTag 的 scope 是 "auto"，并且如果是单个规则触发，这里其实有点特殊。
	// AutoTag 现在的逻辑是基于“所有规则”的一次性全量计算。
	// 所以我们可以直接把 matchedTagIDs 传给 SyncEntityTags。
	// 但是 SyncEntityTags 只能接受一个 ruleID，而 AutoTag 不同的标签可能来自不同的 ruleID。
	// 这是一个设计冲突。
	//
	// 修正：AutoTag 场景比较特殊，它是一次计算出多个 Tag 对应 多个 Rule。
	// 而 SyncEntityTags 假设的是 targetTagIDs 都属于同一个 source 和 rule (或者 rule=0)。
	//
	// 方案：
	// 既然 AutoTag 逻辑已经很完善（支持多 Rule），我们保留 AutoTag 的独立逻辑，
	// 或者是让 SyncEntityTags 支持 Map[TagID]RuleID？
	// 考虑到复杂性，我们先实现一个基础版的 SyncEntityTags 用于 Agent Report，
	// AutoTag 保持现状（因为它是最复杂的场景）。

	// ... Wait, actually we can refactor SyncEntityTags to be more generic.
	// But for now, to satisfy the immediate requirement (Agent Report),
	// let's implement SyncEntityTags separately and keep AutoTag as is for now.
	// We can refactor AutoTag later to use a more advanced version of SyncEntityTags.

	// 保持 AutoTag 原有逻辑不变
	// ... (原代码)

	// Step 3.1: 从数据库获取现有 Auto 标签
	existingTags, err := s.repo.GetEntityTags(entityType, entityID)
	if err != nil {
		return err
	}

	existingAutoTagMap := make(map[uint64]uint64) // TagID -> RuleID
	existingSourceMap := make(map[uint64]string)  // TagID -> Source

	for _, t := range existingTags {
		existingSourceMap[t.TagID] = t.Source
		// 只处理 source='auto' 标签,其他来源的标签不处理,比如手动添加的标签
		if t.Source == "auto" {
			existingAutoTagMap[t.TagID] = t.RuleID
		}
	}

	// Step 3.2: 计算差异
	// 需要添加的
	for i, tagID := range matchedTagIDs {
		ruleID := matchedRuleIDs[i]

		// 检查是否存在非 auto 来源的同名标签 (例如 manual)
		// 如果存在，则跳过覆盖，保留原有的 manual 状态
		if source, exists := existingSourceMap[tagID]; exists && source != "auto" {
			continue
		}

		// 如果已存在且RuleID一致，跳过 (已存在标签,且规则ID一致,则无需重复添加)
		if currRuleID, exists := existingAutoTagMap[tagID]; exists {
			if currRuleID == ruleID {
				delete(existingAutoTagMap, tagID) // 标记为已处理
				continue
			}
		}

		// 添加/更新 (不存在或RuleID不一致,更新或添加标签)
		err := s.repo.AddEntityTag(&tag_system.SysEntityTag{
			EntityType: entityType,
			EntityID:   entityID,
			TagID:      tagID,
			Source:     "auto",
			RuleID:     ruleID,
		})
		if err != nil {
			return err
		}

		// 从待删除列表中移除 (如果之前存在)
		delete(existingAutoTagMap, tagID)
	}

	// Step 3.3: 移除不再命中的标签 (剩下的 existingAutoTagMap)
	for tagID := range existingAutoTagMap {
		err := s.repo.RemoveEntityTag(entityType, entityID, tagID)
		if err != nil {
			return err
		}
	}

	return nil
}

// SyncEntityTags 全量同步实体的标签 (用于 Agent Report 等场景)
func (s *tagService) SyncEntityTags(ctx context.Context, entityType string, entityID string, targetTagIDs []uint64, sourceScope string, ruleID uint64) error {
	// 1. 获取现有标签
	existingTags, err := s.repo.GetEntityTags(entityType, entityID)
	if err != nil {
		return err
	}

	// 2. 建立索引
	// existingInScope: 在当前 sourceScope 下已存在的 TagID
	existingInScope := make(map[uint64]bool)
	// existingOthers: 其他 Source 的 TagID (用于防止覆盖 Manual 标签)
	existingOthers := make(map[uint64]string)

	for _, t := range existingTags {
		if t.Source == sourceScope {
			existingInScope[t.TagID] = true
		} else {
			existingOthers[t.TagID] = t.Source
		}
	}

	// 3. 计算差异
	targetSet := make(map[uint64]bool)

	// A. 处理新增 (Add)
	for _, tagID := range targetTagIDs {
		targetSet[tagID] = true

		// 如果该标签已被其他 Source 占用 (e.g. Manual), 则跳过
		// 优先权：Manual > Auto/AgentReport
		if _, exists := existingOthers[tagID]; exists {
			continue
		}

		// 如果已存在于当前 Scope，标记为保留，不做操作
		if _, exists := existingInScope[tagID]; exists {
			// Remove from existingInScope map to mark as "kept"
			delete(existingInScope, tagID)
			continue
		}

		// 执行添加
		err := s.repo.AddEntityTag(&tag_system.SysEntityTag{
			EntityType: entityType,
			EntityID:   entityID,
			TagID:      tagID,
			Source:     sourceScope,
			RuleID:     ruleID,
		})
		if err != nil {
			return err
		}
	}

	// B. 处理删除 (Remove)
	// 此时 existingInScope 中剩下的就是 "数据库中有，但 Target 中没有" 的标签
	for tagID := range existingInScope {
		err := s.repo.RemoveEntityTag(entityType, entityID, tagID)
		if err != nil {
			return err
		}
	}

	return nil
}

// BootstrapSystemTags 初始化系统预设标签骨架
// 结构:
// ROOT (ID: 1)
// ├── System (Category: 'system')
// │   ├── TaskSupport (Category: 'system')
// │   └── Feature (Category: 'system')
// └── AgentGroup (Category: 'agent_group')
//
//	└── Default (Category: 'agent_group')
func (s *tagService) BootstrapSystemTags(ctx context.Context) error {
	// Helper to create tag if not exists
	ensureTag := func(name string, parentID uint64, category string) (*tag_system.SysTag, error) {
		// 1. Check if exists by name and parent
		// Note: repo should ideally provide FindByNameAndParent.
		// Since we don't have it, we might need to rely on CreateTag's idempotency or add a check.
		// For now, let's assume CreateTag handles duplication gracefully or we check manually.
		// But wait, CreateTag uses GORM which might return error on unique constraint.
		// Let's list children of parent and find.
		children, err := s.repo.GetTagsByParent(parentID)
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			if child.Name == name {
				return &child, nil
			}
		}

		// 2. Create if not found
		newTag := &tag_system.SysTag{
			Name:     name,
			ParentID: parentID,
			Category: category,
		}
		if err := s.CreateTag(ctx, newTag); err != nil {
			return nil, err
		}
		return newTag, nil
	}

	// 1. Root Node (ID: 1 is usually reserved or we find/create a "ROOT" tag)
	// In our system, ParentID=0 is root. But usually we want a visible Root tag?
	// Based on design, we have top-level tags like "System", "AgentGroup".
	// They have ParentID = 0.
	// Wait, standard practice: ParentID=0 are top level.
	// Let's ensure top level tags exist.

	// 1. System
	sysTag, err := ensureTag("System", 0, "system")
	if err != nil {
		return fmt.Errorf("failed to ensure System tag: %w", err)
	}

	// 1.1 TaskSupport
	_, err = ensureTag("TaskSupport", sysTag.ID, "system")
	if err != nil {
		return fmt.Errorf("failed to ensure TaskSupport tag: %w", err)
	}

	// 1.2 Feature
	_, err = ensureTag("Feature", sysTag.ID, "system")
	if err != nil {
		return fmt.Errorf("failed to ensure Feature tag: %w", err)
	}

	// 2. AgentGroup
	groupTag, err := ensureTag("AgentGroup", 0, "agent_group")
	if err != nil {
		return fmt.Errorf("failed to ensure AgentGroup tag: %w", err)
	}

	// 2.1 Default Group
	_, err = ensureTag("Default", groupTag.ID, "agent_group")
	if err != nil {
		return fmt.Errorf("failed to ensure Default group tag: %w", err)
	}

	return nil
}

// SubmitPropagationTask 提交标签传播任务
func (s *tagService) SubmitPropagationTask(ctx context.Context, ruleID uint64, action string) (string, error) {
	// 1. 获取规则详情
	ruleRecord, err := s.repo.GetRuleByID(ruleID)
	if err != nil {
		return "", err
	}

	matchRule, err := matcher.ParseJSON(ruleRecord.RuleJSON)
	if err != nil {
		return "", fmt.Errorf("invalid rule json: %v", err)
	}

	// 2. 获取标签详情以获取名称
	tagRecord, err := s.repo.GetTagByID(ruleRecord.TagID)
	if err != nil {
		return "", fmt.Errorf("failed to get tag info: %v", err)
	}

	// 3. 构造任务载荷
	payload := TagPropagationPayload{
		TargetType: ruleRecord.EntityType,
		Action:     action,
		Rule:       matchRule,
		RuleID:     ruleRecord.ID,
		Tags:       []string{tagRecord.Name},
		TagIDs:     []uint64{ruleRecord.TagID},
	}

	payloadBytes, _ := json.Marshal(payload)

	// 4. 创建系统任务 (直接写入 agent_tasks 表)
	// 注意：这里需要与 orchestrator.AgentTask 结构保持一致
	taskID, _ := utils.GenerateUUID()
	// if err != nil {
	// 	return "", fmt.Errorf("failed to generate task ID: %v", err)
	// }
	task := orchestrator.AgentTask{
		TaskID:       taskID,
		TaskType:     "sys_tag_propagation", // 对应 ToolName ?
		AgentID:      "master",              // 系统任务归属 master
		Status:       "pending",
		Priority:     10, // 高优先级
		ToolName:     ToolNameSysTagPropagation,
		ToolParams:   string(payloadBytes),
		TaskCategory: TaskCategorySystem, // 关键字段
		InputTarget:  "{}",               // JSON 字段必须有默认值
		RequiredTags: "[]",               // JSON 字段必须有默认值
		OutputResult: "{}",               // JSON 字段必须有默认值
	}

	if err := s.db.Create(&task).Error; err != nil {
		return "", err
	}

	return taskID, nil
}

// SubmitEntityPropagationTask 提交实体标签传播任务
func (s *tagService) SubmitEntityPropagationTask(ctx context.Context, entityType string, entityID uint64, tagIDs []uint64, action string) (string, error) {
	if entityType != "network" {
		return "", fmt.Errorf("currently only network propagation is supported")
	}

	// 1. 获取网络实体以获取 CIDR (AssetNetwork 有 network 和 cidr 字段) network 不唯一啊
	var network assetModel.AssetNetwork
	if err := s.db.WithContext(ctx).First(&network, entityID).Error; err != nil {
		return "", fmt.Errorf("failed to find network: %v", err)
	}
	// SELECT * FROM `asset_networks` WHERE `asset_networks`.`id` = 5 ORDER BY `asset_networks`.`id` LIMIT 1

	// 2. 获取标签以获取名称
	var tags []tag_system.SysTag
	if err := s.db.WithContext(ctx).Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
		return "", fmt.Errorf("failed to find tags: %v", err)
	}
	var tagNames []string
	for _, t := range tags {
		tagNames = append(tagNames, t.Name)
	}

	if len(tagNames) == 0 {
		return "", fmt.Errorf("no valid tags found")
	}

	// 3. 创建虚拟规则 --- 用来匹配 IP 是否在 CIDR
	// Target: Host (Propagate from Network to Host)
	// Rule: IP in CIDR
	rule := matcher.MatchRule{
		Field:    "ip",
		Operator: "cidr",
		Value:    network.CIDR,
	}

	// 4. 构造任务载荷
	payload := TagPropagationPayload{
		TargetType: "host", // Hardcoded for Network->Host propagation
		Action:     action,
		Rule:       rule,
		RuleID:     0, // No Rule ID for manual propagation
		Tags:       tagNames,
		TagIDs:     tagIDs,
	}

	payloadBytes, _ := json.Marshal(payload)

	// 5. 创建系统任务 (直接写入 agent_tasks 表)
	taskID, _ := utils.GenerateUUID()
	// if err2 != nil {
	// 	return "", fmt.Errorf("failed to generate task ID: %v", err2)
	// }
	task := orchestrator.AgentTask{
		TaskID:       taskID,
		TaskType:     "sys_tag_propagation",
		AgentID:      "master",
		Status:       "pending",
		Priority:     10,
		ToolName:     ToolNameSysTagPropagation,
		ToolParams:   string(payloadBytes),
		TaskCategory: TaskCategorySystem,
		InputTarget:  "{}", // JSON 字段必须有默认值
		RequiredTags: "[]", // JSON 字段必须有默认值
		OutputResult: "{}", // JSON 字段必须有默认值
	}

	if err := s.db.Create(&task).Error; err != nil {
		return "", err
	}

	return taskID, nil
}
