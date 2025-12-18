package tag_system

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"neomaster/internal/pkg/logger"
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
	GetTagByName(ctx context.Context, name string) (*tag_system.SysTag, error)                           // 通过名称获取标签
	GetTagByNameAndParent(ctx context.Context, name string, parentID uint64) (*tag_system.SysTag, error) // 通过名称和父ID获取标签
	GetTagsByIDs(ctx context.Context, ids []uint64) ([]tag_system.SysTag, error)                         // 批量获取标签
	UpdateTag(ctx context.Context, tag *tag_system.SysTag) error
	MoveTag(ctx context.Context, id, targetParentID uint64) error // 移动标签 改变标签的层级结构
	DeleteTag(ctx context.Context, id uint64, force bool) error
	ListTags(ctx context.Context, req *tag_system.ListTagsRequest) ([]tag_system.SysTag, int64, error)

	// --- 规则 Rules CRUD ---
	CreateRule(ctx context.Context, rule *tag_system.SysMatchRule) error                                       // 创建匹配规则
	UpdateRule(ctx context.Context, rule *tag_system.SysMatchRule) error                                       // 更新匹配规则
	DeleteRule(ctx context.Context, id uint64) error                                                           // 删除匹配规则
	GetRule(ctx context.Context, id uint64) (*tag_system.SysMatchRule, error)                                  // 根据ID获取匹配规则
	ListRules(ctx context.Context, req *tag_system.ListRulesRequest) ([]tag_system.SysMatchRule, int64, error) // 获取所有匹配规则
	ReloadMatchRules() error                                                                                   // 从数据库加载所有启用规则到内存中，缓存规则，提高性能

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

	// // --- 系统初始化 Bootstrap ---
	// // BootstrapSystemTags 初始化系统预设标签骨架 (Root, System, AgentGroup 等)
	// BootstrapSystemTags(ctx context.Context) error

	// --- 实体标签操作 (Single Entity) ---
	AddEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64, source string, ruleID uint64) error // 给实体添加标签
	RemoveEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64) error                            // 删除实体的标签
	GetEntityTags(ctx context.Context, entityType string, entityID string) ([]tag_system.SysEntityTag, error)               // 获取实体所有标签
	GetEntityIDsByTagIDs(ctx context.Context, entityType string, tagIDs []uint64) ([]string, error)                         // 根据标签ID获取实体ID列表                                                                                               // 重载所有规则到内存缓存
}

type CachedRule struct {
	RuleID uint64
	TagID  uint64
	Rule   matcher.MatchRule
}

type MatchRuleCache struct {
	mu    sync.RWMutex
	rules map[string][]CachedRule // key: entityType
}

func (c *MatchRuleCache) Get(entityType string) []CachedRule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rules[entityType]
}

type tagService struct {
	repo      repo.TagRepository
	db        *gorm.DB // 用于直接插入任务，或者需要事务
	ruleCache *MatchRuleCache
}

func NewTagService(repo repo.TagRepository, db *gorm.DB) TagService {
	s := &tagService{
		repo: repo,
		db:   db,
		ruleCache: &MatchRuleCache{
			rules: make(map[string][]CachedRule),
		},
	}
	// 初始化时加载规则
	// 注意：如果数据库连接失败，这里可能会报错，建议在应用启动时处理错误，或者这里记录日志但不panic
	if err := s.ReloadMatchRules(); err != nil {
		logger.LogError(err, "", 0, "", "service.tag_system.NewTagService", "INIT", map[string]interface{}{
			"action": "reload_match_rules",
		})
	}
	return s
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

func (s *tagService) GetTagByNameAndParent(ctx context.Context, name string, parentID uint64) (*tag_system.SysTag, error) {
	children, err := s.repo.GetTagsByParent(parentID)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		if child.Name == name {
			return &child, nil
		}
	}
	return nil, fmt.Errorf("tag not found: name=%s, parentID=%d", name, parentID)
}

func (s *tagService) GetTagsByIDs(ctx context.Context, ids []uint64) ([]tag_system.SysTag, error) {
	tags, err := s.repo.GetTagsByIDs(ids)
	if err != nil {
		return nil, err
	}
	// 填充全路径名称
	if len(tags) > 0 {
		_ = s.enrichTagsWithFullPath(tags)
	}
	return tags, nil
}

// enrichTagsWithFullPath 批量填充标签的全路径名称
// 核心逻辑: 解析 Path 字段获取所有祖先ID，批量查询名称，构建 "Root/Parent/Child" 字符串
func (s *tagService) enrichTagsWithFullPath(tags []tag_system.SysTag) error {
	ancestorIDs := make(map[uint64]struct{})
	for _, tag := range tags {
		// Parse path: /1/5/ -> ["", "1", "5", ""]
		parts := strings.Split(tag.Path, "/")
		for _, part := range parts {
			if part == "" {
				continue
			}
			id, err := strconv.ParseUint(part, 10, 64)
			if err == nil && id > 0 {
				ancestorIDs[id] = struct{}{}
			}
		}
	}

	if len(ancestorIDs) == 0 {
		// No ancestors (all are roots), just set FullPathName = Name
		for i := range tags {
			tags[i].FullPathName = tags[i].Name
		}
		return nil
	}

	var ids []uint64
	for id := range ancestorIDs {
		ids = append(ids, id)
	}

	// Fetch ancestors from Repo directly to avoid recursion loop
	ancestorTags, err := s.repo.GetTagsByIDs(ids)
	if err != nil {
		return err
	}

	nameMap := make(map[uint64]string)
	for _, t := range ancestorTags {
		nameMap[t.ID] = t.Name
	}

	for i := range tags {
		parts := strings.Split(tags[i].Path, "/")
		var names []string
		for _, part := range parts {
			if part == "" {
				continue
			}
			id, _ := strconv.ParseUint(part, 10, 64)
			if name, ok := nameMap[id]; ok {
				names = append(names, name)
			} else {
				names = append(names, part) // Fallback to ID if name missing
			}
		}
		names = append(names, tags[i].Name)
		tags[i].FullPathName = strings.Join(names, "/")
	}

	return nil
}

func (s *tagService) UpdateTag(ctx context.Context, tag *tag_system.SysTag) error {
	// 注意: Repo层已限制只能更新 Name, Color, Description 等非结构字段
	// 如果需要修改 ParentID，必须使用 MoveTag 方法
	return s.repo.UpdateTag(tag)
}

func (s *tagService) MoveTag(ctx context.Context, id, targetParentID uint64) error {
	return s.repo.MoveTag(id, targetParentID)
}

func (s *tagService) DeleteTag(ctx context.Context, id uint64, force bool) error {
	// 调用 repo 层的删除逻辑，repo 层负责级联删除的检查和执行
	return s.repo.DeleteTag(id, force)
}

func (s *tagService) ListTags(ctx context.Context, req *tag_system.ListTagsRequest) ([]tag_system.SysTag, int64, error) {
	return s.repo.ListTags(req)
}

// --- Rules Implementation ---

// CreateRule 创建匹配规则
func (s *tagService) CreateRule(ctx context.Context, rule *tag_system.SysMatchRule) error {
	// 验证 RuleJSON 格式
	if _, err := matcher.ParseJSON(rule.RuleJSON); err != nil {
		return fmt.Errorf("invalid rule json: %v", err)
	}
	if err := s.repo.CreateRule(rule); err != nil {
		return err
	}

	// 规则变更，自动刷新缓存
	if err := s.ReloadMatchRules(); err != nil {
		// Log error but don't fail the request as DB update succeeded
		logger.LogBusinessError(err, "", 0, "", "service.tag_system.CreateRule", "POST", map[string]interface{}{
			"action":  "reload_match_rules",
			"rule_id": rule.ID,
		})
	}

	return nil
}

func (s *tagService) UpdateRule(ctx context.Context, rule *tag_system.SysMatchRule) error {
	if _, err := matcher.ParseJSON(rule.RuleJSON); err != nil {
		return fmt.Errorf("invalid rule json: %v", err)
	}
	if err := s.repo.UpdateRule(rule); err != nil {
		return err
	}

	// 规则变更，自动刷新缓存
	if err := s.ReloadMatchRules(); err != nil {
		// Log error but don't fail the request as DB update succeeded
		logger.LogBusinessError(err, "", 0, "", "service.tag_system.UpdateRule", "PUT", map[string]interface{}{
			"action":  "reload_match_rules",
			"rule_id": rule.ID,
		})
	}

	return nil
}

func (s *tagService) DeleteRule(ctx context.Context, id uint64) error {
	if err := s.repo.DeleteRule(id); err != nil {
		return err
	}

	// 规则变更，自动刷新缓存
	if err := s.ReloadMatchRules(); err != nil {
		// Log error but don't fail the request as DB update succeeded
		logger.LogBusinessError(err, "", 0, "", "service.tag_system.DeleteRule", "DELETE", map[string]interface{}{
			"action":  "reload_match_rules",
			"rule_id": id,
		})
	}

	return nil
}

func (s *tagService) GetRule(ctx context.Context, id uint64) (*tag_system.SysMatchRule, error) {
	return s.repo.GetRuleByID(id)
}

func (s *tagService) ListRules(ctx context.Context, req *tag_system.ListRulesRequest) ([]tag_system.SysMatchRule, int64, error) {
	return s.repo.ListRules(req)
}

// ReloadMatchRules 从数据库加载所有启用规则到内存中
// 缓存规则，提高性能，避免每次匹配规则都需要查询规则库
func (s *tagService) ReloadMatchRules() error {
	enabled := true
	// Page=0, PageSize=0 means no limit in repo implementation
	req := &tag_system.ListRulesRequest{
		IsEnabled: &enabled,
	}
	allRules, _, err := s.repo.ListRules(req)
	if err != nil {
		return err
	}

	newCache := make(map[string][]CachedRule)
	for _, r := range allRules {
		parsedRule, err := matcher.ParseJSON(r.RuleJSON)
		if err != nil {
			logger.LogError(err, "", 0, "", "service.tag_system.ReloadMatchRules", "INTERNAL", map[string]interface{}{
				"rule_id": r.ID,
				"action":  "parse_rule_json",
			})
			continue
		}
		cr := CachedRule{
			RuleID: r.ID,
			TagID:  r.TagID,
			Rule:   parsedRule,
		}
		newCache[r.EntityType] = append(newCache[r.EntityType], cr)
	}

	s.ruleCache.mu.Lock()
	s.ruleCache.rules = newCache
	s.ruleCache.mu.Unlock()
	return nil
}

// --- Auto Tagging Implementation (Moved to auto_tag.go or here) ---
// 为了保持文件简洁，AutoTag 和 SubmitPropagationTask 可以放在单独文件，或者这里
// 这里先放这里，如果太长再拆分
func (s *tagService) AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error {
	// 1. 获取该实体类型的所有启用规则 (FROM CACHE)
	cachedRules := s.ruleCache.Get(entityType)

	if len(cachedRules) == 0 {
		return nil
	}

	// 2. 遍历规则进行匹配
	var matchedTagIDs []uint64
	var matchedRuleIDs []uint64

	for _, cr := range cachedRules {
		// 执行匹配
		matched, err2 := matcher.Match(attributes, cr.Rule)
		if err2 != nil {
			continue
		}

		if matched {
			// 匹配成功,记录标签ID和规则ID成列表
			matchedTagIDs = append(matchedTagIDs, cr.TagID)
			matchedRuleIDs = append(matchedRuleIDs, cr.RuleID)
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

// AddEntityTag 添加实体标签
func (s *tagService) AddEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64, source string, ruleID uint64) error {
	return s.repo.AddEntityTag(&tag_system.SysEntityTag{
		EntityType: entityType,
		EntityID:   entityID,
		TagID:      tagID,
		Source:     source,
		RuleID:     ruleID,
	})
}

// RemoveEntityTag 移除实体标签
func (s *tagService) RemoveEntityTag(ctx context.Context, entityType string, entityID string, tagID uint64) error {
	return s.repo.RemoveEntityTag(entityType, entityID, tagID)
}

// GetEntityTags 获取实体标签
func (s *tagService) GetEntityTags(ctx context.Context, entityType string, entityID string) ([]tag_system.SysEntityTag, error) {
	return s.repo.GetEntityTags(entityType, entityID)
}

// GetEntityIDsByTagIDs 根据标签ID获取实体ID列表
func (s *tagService) GetEntityIDsByTagIDs(ctx context.Context, entityType string, tagIDs []uint64) ([]string, error) {
	return s.repo.GetEntityIDsByTagIDs(entityType, tagIDs)
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
// func (s *tagService) BootstrapSystemTags(ctx context.Context) error {
// 	// Helper to create tag if not exists
// 	ensureTag := func(name string, parentID uint64, category string) (*tag_system.SysTag, error) {
// 		// 1. Check if exists by name and parent
// 		// Note: repo should ideally provide FindByNameAndParent.
// 		// Since we don't have it, we might need to rely on CreateTag's idempotency or add a check.
// 		// For now, let's assume CreateTag handles duplication gracefully or we check manually.
// 		// But wait, CreateTag uses GORM which might return error on unique constraint.
// 		// Let's list children of parent and find.
// 		children, err := s.repo.GetTagsByParent(parentID)
// 		if err != nil {
// 			return nil, err
// 		}
// 		for _, child := range children {
// 			if child.Name == name {
// 				return &child, nil
// 			}
// 		}

// 		// 2. Create if not found
// 		newTag := &tag_system.SysTag{
// 			Name:     name,
// 			ParentID: parentID,
// 			Category: category,
// 		}
// 		if err := s.CreateTag(ctx, newTag); err != nil {
// 			return nil, err
// 		}
// 		return newTag, nil
// 	}

// 	// 1. Root Node (ID: 1 is usually reserved or we find/create a "ROOT" tag)
// 	// In our system, ParentID=0 is root. But usually we want a visible Root tag?
// 	// Based on design, we have top-level tags like "System", "AgentGroup".
// 	// They have ParentID = 0.
// 	// Wait, standard practice: ParentID=0 are top level.
// 	// Let's ensure top level tags exist.

// 	// 1. System
// 	sysTag, err := ensureTag("System", 0, "system")
// 	if err != nil {
// 		return fmt.Errorf("failed to ensure System tag: %w", err)
// 	}

// 	// 1.1 TaskSupport
// 	_, err = ensureTag("TaskSupport", sysTag.ID, "system")
// 	if err != nil {
// 		return fmt.Errorf("failed to ensure TaskSupport tag: %w", err)
// 	}

// 	// 1.2 Feature
// 	_, err = ensureTag("Feature", sysTag.ID, "system")
// 	if err != nil {
// 		return fmt.Errorf("failed to ensure Feature tag: %w", err)
// 	}

// 	// 2. AgentGroup
// 	groupTag, err := ensureTag("AgentGroup", 0, "agent_group")
// 	if err != nil {
// 		return fmt.Errorf("failed to ensure AgentGroup tag: %w", err)
// 	}

// 	// 2.1 Default Group
// 	_, err = ensureTag("Default", groupTag.ID, "agent_group")
// 	if err != nil {
// 		return fmt.Errorf("failed to ensure Default group tag: %w", err)
// 	}

// 	return nil
// }

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
