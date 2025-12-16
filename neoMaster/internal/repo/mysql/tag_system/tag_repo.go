package tag_system

import (
	"errors"

	"gorm.io/gorm"

	"neomaster/internal/model/tag_system"
)

// TagRepository 标签系统数据访问接口
type TagRepository interface {
	// 标签定义管理
	CreateTag(tag *tag_system.SysTag) error
	GetTagByID(id uint64) (*tag_system.SysTag, error)
	GetTagsByIDs(ids []uint64) ([]tag_system.SysTag, error)
	UpdateTag(tag *tag_system.SysTag) error
	DeleteTag(id uint64) error
	ListTags(parentID uint64) ([]tag_system.SysTag, error) // 获取子标签

	// 标签规则管理
	CreateRule(rule *tag_system.SysMatchRule) error
	GetRulesByEntityType(entityType string) ([]tag_system.SysMatchRule, error)
	GetRuleByID(id uint64) (*tag_system.SysMatchRule, error)
	UpdateRule(rule *tag_system.SysMatchRule) error
	DeleteRule(id uint64) error

	// 实体关联管理
	AddEntityTag(et *tag_system.SysEntityTag) error
	RemoveEntityTag(entityType, entityID string, tagID uint64) error
	GetEntityTags(entityType, entityID string) ([]tag_system.SysEntityTag, error)
	RemoveAllEntityTags(entityType, entityID string) error // 清除实体的所有标签
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

// --- 标签定义管理 ---

func (r *tagRepository) CreateTag(tag *tag_system.SysTag) error {
	return r.db.Create(tag).Error
}

func (r *tagRepository) GetTagByID(id uint64) (*tag_system.SysTag, error) {
	var tag tag_system.SysTag
	err := r.db.First(&tag, id).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *tagRepository) GetTagsByIDs(ids []uint64) ([]tag_system.SysTag, error) {
	var tags []tag_system.SysTag
	err := r.db.Where("id IN ?", ids).Find(&tags).Error
	return tags, err
}

func (r *tagRepository) UpdateTag(tag *tag_system.SysTag) error {
	return r.db.Save(tag).Error
}

func (r *tagRepository) DeleteTag(id uint64) error {
	// 注意：删除标签时，应该递归删除子标签吗？或者禁止删除非空标签？
	// 这里简单实现为删除单个标签，业务层负责逻辑检查
	return r.db.Delete(&tag_system.SysTag{}, id).Error
}

func (r *tagRepository) ListTags(parentID uint64) ([]tag_system.SysTag, error) {
	var tags []tag_system.SysTag
	err := r.db.Where("parent_id = ?", parentID).Find(&tags).Error
	return tags, err
}

// --- 规则管理 ---

func (r *tagRepository) CreateRule(rule *tag_system.SysMatchRule) error {
	return r.db.Create(rule).Error
}

func (r *tagRepository) GetRulesByEntityType(entityType string) ([]tag_system.SysMatchRule, error) {
	var rules []tag_system.SysMatchRule
	// 获取所有启用的规则，按优先级降序排序
	err := r.db.Where("entity_type = ? AND is_enabled = ?", entityType, true).
		Order("priority desc").
		Find(&rules).Error
	return rules, err
}

func (r *tagRepository) GetRuleByID(id uint64) (*tag_system.SysMatchRule, error) {
	var rule tag_system.SysMatchRule
	err := r.db.First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *tagRepository) UpdateRule(rule *tag_system.SysMatchRule) error {
	return r.db.Save(rule).Error
}

func (r *tagRepository) DeleteRule(id uint64) error {
	return r.db.Delete(&tag_system.SysMatchRule{}, id).Error
}

// --- 实体关联管理 ---

func (r *tagRepository) AddEntityTag(et *tag_system.SysEntityTag) error {
	// 使用 FirstOrCreate 避免重复添加
	var existing tag_system.SysEntityTag
	result := r.db.Where("entity_type = ? AND entity_id = ? AND tag_id = ?", et.EntityType, et.EntityID, et.TagID).
		First(&existing)

	if result.Error == nil {
		// 已存在，不再重复添加，但可能需要更新 Source 或 RuleID
		// 如果原有是 manual，现在是 auto，可能不应该覆盖？或者 auto 覆盖 manual？
		// 暂时策略：如果记录存在，更新 Source 和 RuleID
		existing.Source = et.Source
		existing.RuleID = et.RuleID
		return r.db.Save(&existing).Error
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// 不存在，创建新记录
		return r.db.Create(et).Error
	} else {
		return result.Error
	}
}

func (r *tagRepository) RemoveEntityTag(entityType, entityID string, tagID uint64) error {
	return r.db.Where("entity_type = ? AND entity_id = ? AND tag_id = ?", entityType, entityID, tagID).
		Delete(&tag_system.SysEntityTag{}).Error
}

func (r *tagRepository) GetEntityTags(entityType, entityID string) ([]tag_system.SysEntityTag, error) {
	var tags []tag_system.SysEntityTag
	err := r.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).Find(&tags).Error
	return tags, err
}

func (r *tagRepository) RemoveAllEntityTags(entityType, entityID string) error {
	return r.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Delete(&tag_system.SysEntityTag{}).Error
}
