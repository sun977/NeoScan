package tag_system

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

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
	Action     string            `json:"action"`      // add, remove, refresh
	Rule       matcher.MatchRule `json:"rule"`        // 匹配规则
	TagIDs     []uint64          `json:"tag_ids"`     // 关联的标签ID列表
}

type TagService interface {
	// --- Basic CRUD ---
	CreateTag(ctx context.Context, tag *tag_system.SysTag) error
	GetTag(ctx context.Context, id uint64) (*tag_system.SysTag, error)
	UpdateTag(ctx context.Context, tag *tag_system.SysTag) error
	DeleteTag(ctx context.Context, id uint64) error
	ListTags(ctx context.Context, req *tag_system.ListTagsRequest) ([]tag_system.SysTag, int64, error)

	// --- Rules ---
	CreateRule(ctx context.Context, rule *tag_system.SysMatchRule) error
	UpdateRule(ctx context.Context, rule *tag_system.SysMatchRule) error
	DeleteRule(ctx context.Context, id uint64) error
	GetRule(ctx context.Context, id uint64) (*tag_system.SysMatchRule, error)
	ListRules(ctx context.Context, req *tag_system.ListRulesRequest) ([]tag_system.SysMatchRule, int64, error)

	// --- Auto Tagging ---
	AutoTag(ctx context.Context, entityType string, entityID string, attributes map[string]interface{}) error

	// --- Propagation ---
	SubmitPropagationTask(ctx context.Context, ruleID uint64) (string, error)
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
	return s.repo.CreateRule(rule)
}

func (s *tagService) UpdateRule(ctx context.Context, rule *tag_system.SysMatchRule) error {
	if _, err := matcher.ParseJSON(rule.RuleJSON); err != nil {
		return fmt.Errorf("invalid rule json: %v", err)
	}
	return s.repo.UpdateRule(rule)
}

func (s *tagService) DeleteRule(ctx context.Context, id uint64) error {
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
			matchedTagIDs = append(matchedTagIDs, ruleRecord.TagID)
			matchedRuleIDs = append(matchedRuleIDs, ruleRecord.ID)
		}
	}

	// 3. 更新实体标签
	// 策略：AutoTag 是增量还是覆盖？
	// 通常 AutoTag 负责"确保命中规则的标签存在"。
	// 如果之前命中了规则A打了标签X，现在不命中了，是否要移除标签X？
	// 这是一个复杂问题 (State Management)。
	// 简化版本 (MVP): 只增不减，或者由"清洗任务"负责移除。
	// 但如果规则变了，标签不移除会产生脏数据。
	// 改进版本:
	// 找出该实体所有 Source='auto' 的标签 -> 比较 -> 多退少补。

	// Step 3.1: 获取现有 Auto 标签
	existingTags, err := s.repo.GetEntityTags(entityType, entityID)
	if err != nil {
		return err
	}

	existingAutoTagMap := make(map[uint64]uint64) // TagID -> RuleID
	for _, t := range existingTags {
		if t.Source == "auto" {
			existingAutoTagMap[t.TagID] = t.RuleID
		}
	}

	// Step 3.2: 计算差异
	// 需要添加的
	for i, tagID := range matchedTagIDs {
		ruleID := matchedRuleIDs[i]

		// 如果已存在且RuleID一致，跳过
		if currRuleID, exists := existingAutoTagMap[tagID]; exists {
			if currRuleID == ruleID {
				delete(existingAutoTagMap, tagID) // 标记为已处理
				continue
			}
		}

		// 添加/更新
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

func (s *tagService) SubmitPropagationTask(ctx context.Context, ruleID uint64) (string, error) {
	// 1. 获取规则详情
	ruleRecord, err := s.repo.GetRuleByID(ruleID)
	if err != nil {
		return "", err
	}

	matchRule, err := matcher.ParseJSON(ruleRecord.RuleJSON)
	if err != nil {
		return "", fmt.Errorf("invalid rule json: %v", err)
	}

	// 2. 构造任务载荷
	payload := TagPropagationPayload{
		TargetType: ruleRecord.EntityType,
		Action:     "refresh", // 刷新模式
		Rule:       matchRule,
		TagIDs:     []uint64{ruleRecord.TagID},
	}

	payloadBytes, _ := json.Marshal(payload)

	// 3. 创建系统任务 (直接写入 agent_tasks 表)
	// 注意：这里需要与 orchestrator.AgentTask 结构保持一致
	taskID := uuid.New().String()
	task := orchestrator.AgentTask{
		TaskID:       taskID,
		TaskType:     "sys_tag_propagation", // 对应 ToolName ?
		AgentID:      "master",              // 系统任务归属 master
		Status:       "pending",
		Priority:     10, // 高优先级
		ToolName:     ToolNameSysTagPropagation,
		ToolParams:   string(payloadBytes),
		TaskCategory: TaskCategorySystem, // 关键字段
	}

	if err := s.db.Create(&task).Error; err != nil {
		return "", err
	}

	return taskID, nil
}
