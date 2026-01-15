package asset

import (
	"context"
	"errors"
	"neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// RuleRepository 规则仓库
// 负责处理指纹规则和 CPE 规则的批量导入、导出、回滚等聚合操作
type RuleRepository interface {
	// ImportRules 批量导入规则 (事务支持)
	ImportRules(ctx context.Context, fingers []*asset.AssetFinger, cpes []*asset.AssetCPE) error
	// RestoreRules 恢复规则 (清空并重新插入)
	RestoreRules(ctx context.Context, fingers []*asset.AssetFinger, cpes []*asset.AssetCPE) error
}

type ruleRepository struct {
	db *gorm.DB
}

// NewRuleRepository 创建 RuleRepository 实例
func NewRuleRepository(db *gorm.DB) RuleRepository {
	return &ruleRepository{db: db}
}

// ImportRules 批量导入规则
func (r *ruleRepository) ImportRules(ctx context.Context, fingers []*asset.AssetFinger, cpes []*asset.AssetCPE) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Process Fingers
		for _, f := range fingers {
			if err := r.upsertFinger(ctx, tx, f); err != nil {
				return err
			}
		}

		// 2. Process CPEs
		for _, c := range cpes {
			if err := r.upsertCPE(ctx, tx, c); err != nil {
				return err
			}
		}

		return nil
	})
}

// RestoreRules 恢复规则
func (r *ruleRepository) RestoreRules(ctx context.Context, fingers []*asset.AssetFinger, cpes []*asset.AssetCPE) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Clear Existing Data
		// 使用 Unscoped 硬删除以确保环境干净
		if err := tx.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&asset.AssetFinger{}).Error; err != nil {
			logger.LogError(err, "", 0, "", "restore_rules_clear_fingers", "REPO", nil)
			return err
		}
		if err := tx.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&asset.AssetCPE{}).Error; err != nil {
			logger.LogError(err, "", 0, "", "restore_rules_clear_cpes", "REPO", nil)
			return err
		}

		// 2. Insert Backup Data
		if len(fingers) > 0 {
			if err := tx.CreateInBatches(fingers, 100).Error; err != nil {
				logger.LogError(err, "", 0, "", "restore_rules_insert_fingers", "REPO", nil)
				return err
			}
		}

		if len(cpes) > 0 {
			if err := tx.CreateInBatches(cpes, 100).Error; err != nil {
				logger.LogError(err, "", 0, "", "restore_rules_insert_cpes", "REPO", nil)
				return err
			}
		}

		return nil
	})
}

// upsertFinger 内部 Upsert 逻辑 (使用传入的 tx)
func (r *ruleRepository) upsertFinger(ctx context.Context, tx *gorm.DB, rule *asset.AssetFinger) error {
	var existing asset.AssetFinger
	err := tx.WithContext(ctx).Where("name = ?", rule.Name).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err1 := tx.WithContext(ctx).Create(rule).Error; err1 != nil {
				logger.LogError(err1, "", 0, "", "import_rules_create_finger", "REPO", map[string]interface{}{"name": rule.Name})
				return err1
			}
			return nil
		}
		return err
	}

	// 存在则更新，保持 ID 和 CreatedAt 不变
	rule.ID = existing.ID
	rule.CreatedAt = existing.CreatedAt
	if err := tx.WithContext(ctx).Model(rule).Updates(rule).Error; err != nil {
		logger.LogError(err, "", 0, "", "import_rules_update_finger", "REPO", map[string]interface{}{"name": rule.Name})
		return err
	}
	return nil
}

// upsertCPE 内部 Upsert 逻辑 (使用传入的 tx)
func (r *ruleRepository) upsertCPE(ctx context.Context, tx *gorm.DB, rule *asset.AssetCPE) error {
	var existing asset.AssetCPE
	err := tx.WithContext(ctx).Where("name = ?", rule.Name).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err1 := tx.WithContext(ctx).Create(rule).Error; err1 != nil {
				logger.LogError(err1, "", 0, "", "import_rules_create_cpe", "REPO", map[string]interface{}{"name": rule.Name})
				return err1
			}
			return nil
		}
		return err
	}

	// 存在则更新，保持 ID 和 CreatedAt 不变
	rule.ID = existing.ID
	rule.CreatedAt = existing.CreatedAt
	if err := tx.WithContext(ctx).Model(rule).Updates(rule).Error; err != nil {
		logger.LogError(err, "", 0, "", "import_rules_update_cpe", "REPO", map[string]interface{}{"name": rule.Name})
		return err
	}
	return nil
}
