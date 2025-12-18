package tag_system

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"neomaster/internal/model/tag_system"
)

// TagRepository 标签系统数据访问接口
type TagRepository interface {
	// 标签定义管理
	CreateTag(tag *tag_system.SysTag) error
	GetTagByID(id uint64) (*tag_system.SysTag, error)
	GetTagByName(name string) (*tag_system.SysTag, error)   // 获取标签
	GetTagsByIDs(ids []uint64) ([]tag_system.SysTag, error) // 批量获取标签
	GetTagsByParent(parentID uint64) ([]tag_system.SysTag, error)
	UpdateTag(tag *tag_system.SysTag) error
	DeleteTag(id uint64, force bool) error
	ListTags(req *tag_system.ListTagsRequest) ([]tag_system.SysTag, int64, error) // 获取标签列表

	// 标签规则管理
	CreateRule(rule *tag_system.SysMatchRule) error
	GetRulesByEntityType(entityType string) ([]tag_system.SysMatchRule, error)
	ListRules(req *tag_system.ListRulesRequest) ([]tag_system.SysMatchRule, int64, error) // 获取规则列表
	GetRuleByID(id uint64) (*tag_system.SysMatchRule, error)
	UpdateRule(rule *tag_system.SysMatchRule) error
	DeleteRule(id uint64) error

	// 实体关联管理
	AddEntityTag(et *tag_system.SysEntityTag) error
	RemoveEntityTag(entityType, entityID string, tagID uint64) error
	GetEntityTags(entityType, entityID string) ([]tag_system.SysEntityTag, error)
	RemoveAllEntityTags(entityType, entityID string) error                     // 清除实体的所有标签
	GetEntityIDsByTagIDs(entityType string, tagIDs []uint64) ([]string, error) // 根据标签ID获取实体ID列表
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

// GetTagByName 获取标签
func (r *tagRepository) GetTagByName(name string) (*tag_system.SysTag, error) {
	var tag tag_system.SysTag
	err := r.db.Where("name = ?", name).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetTagsByIDs 批量获取标签
func (r *tagRepository) GetTagsByIDs(ids []uint64) ([]tag_system.SysTag, error) {
	var tags []tag_system.SysTag
	err := r.db.Where("id IN ?", ids).Find(&tags).Error
	return tags, err
}

// GetTagsByParent 获取子标签
func (r *tagRepository) GetTagsByParent(parentID uint64) ([]tag_system.SysTag, error) {
	var tags []tag_system.SysTag
	err := r.db.Where("parent_id = ?", parentID).Find(&tags).Error
	return tags, err
}

func (r *tagRepository) UpdateTag(tag *tag_system.SysTag) error {
	return r.db.Save(tag).Error
}

func (r *tagRepository) DeleteTag(id uint64, force bool) error {
	// 1. 检查是否有子标签
	var count int64
	if err := r.db.Model(&tag_system.SysTag{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return err
	}

	// 检查是否有子标签且未使用强制删除(如果选择强制删除，会级联删除所有后代标签)
	if count > 0 && !force {
		return errors.New("cannot delete tag with children, use force=true to cascade delete")
	}

	// 2. 收集所有要删除的ID（包括自标签和后代标签）
	idsToDelete := []uint64{id}
	if count > 0 {
		// 	在SQL中，递归搜索所有后代节点的成本很高。
		//  由于我们拥有“路径”字段（例如 /1/5/10/），因此可以加以利用。
		//  先获取标签本身以查找其路径
		var tag tag_system.SysTag
		if err := r.db.First(&tag, id).Error; err != nil {
			return err
		}

		if tag.Path != "" {
			var descendantIDs []uint64
			// Path like '/root/parent/%'
			// Note: Path format needs to be consistent. Assuming Path stores IDs like "/1/5/".
			if err := r.db.Model(&tag_system.SysTag{}).
				Where("path LIKE ?", tag.Path+"%").
				Pluck("id", &descendantIDs).Error; err != nil {
				return err
			}
			idsToDelete = append(idsToDelete, descendantIDs...)
		}
	}

	// 3. 执行事务
	return r.db.Transaction(func(tx *gorm.DB) error {
		// A. 删除实体标签关联
		if err := tx.Where("tag_id IN ?", idsToDelete).Delete(&tag_system.SysEntityTag{}).Error; err != nil {
			return err
		}

		// B. 删除失效的匹配规则 --- 规则需要匹配的标签不存在了，那么使用这个标签的规则本身也就失效了
		if err := tx.Where("tag_id IN ?", idsToDelete).Delete(&tag_system.SysMatchRule{}).Error; err != nil {
			return err
		}

		// C. 删除标签
		if err := tx.Where("id IN ?", idsToDelete).Delete(&tag_system.SysTag{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *tagRepository) ListTags(req *tag_system.ListTagsRequest) ([]tag_system.SysTag, int64, error) {
	var tags []tag_system.SysTag
	var total int64
	db := r.db.Model(&tag_system.SysTag{})

	if req.ParentID != nil {
		db = db.Where("parent_id = ?", *req.ParentID)
	}
	if req.Category != "" {
		db = db.Where("category = ?", req.Category)
	}
	if req.Keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		db = db.Offset(offset).Limit(req.PageSize)
	}

	err := db.Find(&tags).Error
	return tags, total, err
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

func (r *tagRepository) ListRules(req *tag_system.ListRulesRequest) ([]tag_system.SysMatchRule, int64, error) {
	var rules []tag_system.SysMatchRule
	var total int64
	db := r.db.Model(&tag_system.SysMatchRule{})

	if req.EntityType != "" {
		db = db.Where("entity_type = ?", req.EntityType)
	}
	if req.TagID != 0 {
		db = db.Where("tag_id = ?", req.TagID)
	}
	if req.Keyword != "" {
		db = db.Where("name LIKE ?", "%"+req.Keyword+"%")
	}
	if req.IsEnabled != nil {
		db = db.Where("is_enabled = ?", *req.IsEnabled)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		db = db.Offset(offset).Limit(req.PageSize)
	}

	// Order by priority desc by default
	db = db.Order("priority desc")

	err := db.Find(&rules).Error
	return rules, total, err
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

// GetEntityIDsByTagIDs 根据标签ID获取实体ID列表 (包含子标签)
func (r *tagRepository) GetEntityIDsByTagIDs(entityType string, tagIDs []uint64) ([]string, error) {
	if len(tagIDs) == 0 {
		return []string{}, nil
	}

	// 1. 获取目标标签的 Path，以便查找子标签
	var tags []tag_system.SysTag
	if err := r.db.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return []string{}, nil
	}

	// 2. 收集所有相关的 TagID (自身 + 子标签)
	var allTagIDs []uint64

	// 构造查询子标签的条件: path LIKE 'parent_path%'
	likeConditions := make([]string, 0, len(tags))
	args := make([]interface{}, 0, len(tags))

	for _, tag := range tags {
		// 始终包含自身
		allTagIDs = append(allTagIDs, tag.ID)

		if tag.Path != "" {
			// Path 存储格式如 "/1/5/"，子标签为 "/1/5/10/"
			// 使用 Path + "%" 匹配所有后代
			likeConditions = append(likeConditions, "path LIKE ?")
			args = append(args, tag.Path+"%")
		}
	}

	// 如果有 Path，查询所有子标签
	if len(likeConditions) > 0 {
		var childTagIDs []uint64
		// 拼接 OR 条件: path LIKE 'A%' OR path LIKE 'B%' ...
		sql := strings.Join(likeConditions, " OR ")
		if err := r.db.Model(&tag_system.SysTag{}).Where(sql, args...).Pluck("id", &childTagIDs).Error; err != nil {
			return nil, err
		}
		allTagIDs = append(allTagIDs, childTagIDs...)
	}

	// 3. 查询 SysEntityTag 表，获取对应 EntityID
	var entityIDs []string
	err := r.db.Model(&tag_system.SysEntityTag{}).
		Where("entity_type = ? AND tag_id IN ?", entityType, allTagIDs).
		Distinct("entity_id").
		Pluck("entity_id", &entityIDs).Error

	return entityIDs, err
}
