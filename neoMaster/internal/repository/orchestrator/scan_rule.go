/*
 * 扫描规则仓库层：扫描规则数据访问
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 单纯数据访问，不应该包含业务逻辑
 * @func:
 * 1.创建扫描规则配置
 * 2.更新扫描规则配置
 * 3.删除扫描规则配置
 * 4.扫描规则状态变更等
 */

//  基础CRUD操作:
//  	CreateScanRule - 创建扫描规则
//  	GetScanRuleByID - 根据ID获取扫描规则
//  	GetScanRuleByName - 根据规则名获取扫描规则
//  	UpdateScanRule - 更新扫描规则信息
//  	DeleteScanRule - 软删除扫描规则
//  高级查询功能:
//  	GetScanRuleList - 分页获取扫描规则列表
//  	GetScanRulesByType - 根据类型获取扫描规则
//  	GetScanRulesBySeverity - 根据严重程度获取扫描规则
//  	GetActiveScanRules - 获取活跃的扫描规则
//  	ScanRuleExists - 检查扫描规则是否存在
//  	GetScanRulesByScope - 根据适用范围获取扫描规则
//  状态管理:
//  	UpdateScanRuleStatus - 更新扫描规则状态
//  	EnableScanRule - 启用扫描规则
//  	DisableScanRule - 禁用扫描规则
//  统计功能:
//  	UpdateScanRuleStats - 更新扫描规则统计信息
//  	IncrementExecutionCount - 增加执行次数
//  	IncrementMatchCount - 增加匹配次数
//  事务支持:
//  	BeginTx - 开始事务
//  	UpdateScanRuleWithTx - 事务更新扫描规则
//  	DeleteScanRuleWithTx - 事务删除扫描规则
//  字段更新:
//  	UpdateScanRuleFields - 使用map更新特定字段

package orchestrator

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// ScanRuleRepository 扫描规则仓库结构体
// 负责处理扫描规则相关的数据访问，不包含业务逻辑
type ScanRuleRepository struct {
	db *gorm.DB // 数据库连接
}

// NewScanRuleRepository 创建扫描规则仓库实例
// 注入数据库连接，专注于数据访问操作
func NewScanRuleRepository(db *gorm.DB) *ScanRuleRepository {
	return &ScanRuleRepository{
		db: db,
	}
}

// CreateScanRule 创建扫描规则
// @param ctx 上下文
// @param rule 扫描规则对象
// @return 错误信息
func (r *ScanRuleRepository) CreateScanRule(ctx context.Context, rule *orchestrator.ScanRule) error {
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Create(rule).Error
	if err != nil {
		// 记录创建失败日志
		logger.LogError(err, "", 0, "", "scan_rule_create", "POST", map[string]interface{}{
			"operation": "create_scan_rule",
			"rule_name": rule.Name,
			"rule_type": rule.Type,
			"severity":  rule.Severity,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// GetScanRuleByID 根据ID获取扫描规则
// @param ctx 上下文
// @param id 扫描规则ID
// @return 扫描规则对象和错误信息
func (r *ScanRuleRepository) GetScanRuleByID(ctx context.Context, id uint) (*orchestrator.ScanRule, error) {
	var rule orchestrator.ScanRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("scan rule not found"), "", 0, "", "scan_rule_get", "GET", map[string]interface{}{
				"operation": "get_scan_rule_by_id",
				"id":        id,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "scan_rule_get", "GET", map[string]interface{}{
			"operation": "get_scan_rule_by_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &rule, nil
}

// GetScanRuleByName 根据规则名获取扫描规则
// @param ctx 上下文
// @param name 规则名称
// @return 扫描规则对象和错误信息
func (r *ScanRuleRepository) GetScanRuleByName(ctx context.Context, name string) (*orchestrator.ScanRule, error) {
	var rule orchestrator.ScanRule
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&rule).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("scan rule not found"), "", 0, "", "scan_rule_get", "GET", map[string]interface{}{
				"operation": "get_scan_rule_by_name",
				"rule_name": name,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "scan_rule_get", "GET", map[string]interface{}{
			"operation": "get_scan_rule_by_name",
			"rule_name": name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &rule, nil
}

// UpdateScanRule 更新扫描规则
// @param ctx 上下文
// @param rule 扫描规则对象
// @return 错误信息
func (r *ScanRuleRepository) UpdateScanRule(ctx context.Context, rule *orchestrator.ScanRule) error {
	rule.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(rule).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(rule.ID), "", "scan_rule_update", "PUT", map[string]interface{}{
			"operation": "update_scan_rule",
			"rule_name": rule.Name,
			"rule_id":   rule.ID,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// DeleteScanRule 软删除扫描规则
// @param ctx 上下文
// @param id 扫描规则ID
// @return 错误信息
func (r *ScanRuleRepository) DeleteScanRule(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Delete(&orchestrator.ScanRule{}, id).Error
	if err != nil {
		// 记录删除失败日志
		logger.LogError(err, "", id, "", "scan_rule_delete", "DELETE", map[string]interface{}{
			"operation": "delete_scan_rule",
			"rule_id":   id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// GetScanRuleList 分页获取扫描规则列表
// @param ctx 上下文
// @param offset 偏移量
// @param limit 限制数量
// @param ruleType 规则类型过滤（可选）
// @param severity 严重程度过滤（可选）
// @param status 状态过滤（可选）
// @return 扫描规则列表、总数和错误信息
func (r *ScanRuleRepository) GetScanRuleList(ctx context.Context, offset, limit int, ruleType *orchestrator.ScanRuleType, severity *orchestrator.ScanRuleSeverity, status *orchestrator.ScanRuleStatus) ([]*orchestrator.ScanRule, int64, error) {
	var rules []*orchestrator.ScanRule
	var total int64

	query := r.db.WithContext(ctx).Model(&orchestrator.ScanRule{})

	// 规则类型过滤
	if ruleType != nil {
		query = query.Where("type = ?", *ruleType)
	}

	// 严重程度过滤
	if severity != nil {
		query = query.Where("severity = ?", *severity)
	}

	// 状态过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "scan_rule_list", "GET", map[string]interface{}{
			"operation": "get_scan_rule_list_count",
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, err
	}

	// 获取分页数据
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&rules).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_rule_list", "GET", map[string]interface{}{
			"operation": "get_scan_rule_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, err
	}

	return rules, total, nil
}

// GetScanRulesByType 根据类型获取扫描规则
// @param ctx 上下文
// @param ruleType 规则类型
// @return 扫描规则列表和错误信息
func (r *ScanRuleRepository) GetScanRulesByType(ctx context.Context, ruleType orchestrator.ScanRuleType) ([]*orchestrator.ScanRule, error) {
	var rules []*orchestrator.ScanRule
	err := r.db.WithContext(ctx).Where("type = ? AND status = ?", ruleType, orchestrator.ScanRuleStatusEnabled).Find(&rules).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_rule_list", "GET", map[string]interface{}{
			"operation": "get_scan_rules_by_type",
			"rule_type": ruleType,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return rules, nil
}

// GetScanRulesBySeverity 根据严重程度获取扫描规则
// @param ctx 上下文
// @param severity 严重程度
// @return 扫描规则列表和错误信息
func (r *ScanRuleRepository) GetScanRulesBySeverity(ctx context.Context, severity orchestrator.ScanRuleSeverity) ([]*orchestrator.ScanRule, error) {
	var rules []*orchestrator.ScanRule
	err := r.db.WithContext(ctx).Where("severity = ? AND status = ?", severity, orchestrator.ScanRuleStatusEnabled).Find(&rules).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_rule_list", "GET", map[string]interface{}{
			"operation": "get_scan_rules_by_severity",
			"severity":  severity,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return rules, nil
}

// GetActiveRules 获取活跃的扫描规则（支持类型过滤）
// @param ctx 上下文
// @param ruleType 规则类型过滤（可选）
// @return 扫描规则列表和错误信息
func (r *ScanRuleRepository) GetActiveRules(ctx context.Context, ruleType *orchestrator.ScanRuleType) ([]*orchestrator.ScanRule, error) {
	var rules []*orchestrator.ScanRule
	query := r.db.WithContext(ctx).Where("status = ?", orchestrator.ScanRuleStatusEnabled)
	
	// 类型过滤
	if ruleType != nil {
		query = query.Where("type = ?", *ruleType)
	}
	
	err := query.Find(&rules).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_rule_list", "GET", map[string]interface{}{
			"operation": "get_active_rules",
			"rule_type": ruleType,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return rules, nil
}

// ScanRuleExists 检查扫描规则是否存在
// @param ctx 上下文
// @param name 规则名称
// @return 是否存在和错误信息
func (r *ScanRuleRepository) ScanRuleExists(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanRule{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_rule_exists", "GET", map[string]interface{}{
			"operation": "scan_rule_exists",
			"rule_name": name,
			"timestamp": logger.NowFormatted(),
		})
		return false, err
	}
	return count > 0, nil
}

// GetScanRulesByScope 根据适用范围获取扫描规则
// @param ctx 上下文
// @param scope 适用范围（JSON字符串匹配）
// @return 扫描规则列表和错误信息
func (r *ScanRuleRepository) GetScanRulesByScope(ctx context.Context, scope string) ([]*orchestrator.ScanRule, error) {
	var rules []*orchestrator.ScanRule
	err := r.db.WithContext(ctx).Where("JSON_CONTAINS(applicable_scope, ?) AND status = ?", scope, orchestrator.ScanRuleStatusEnabled).Find(&rules).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_rule_list", "GET", map[string]interface{}{
			"operation": "get_scan_rules_by_scope",
			"scope":     scope,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return rules, nil
}

// UpdateScanRuleStatus 更新扫描规则状态
// @param ctx 上下文
// @param id 扫描规则ID
// @param status 新状态
// @return 错误信息
func (r *ScanRuleRepository) UpdateScanRuleStatus(ctx context.Context, id uint, status orchestrator.ScanRuleStatus) error {
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanRule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_rule_update", "PUT", map[string]interface{}{
			"operation": "update_scan_rule_status",
			"rule_id":   id,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// EnableScanRule 启用扫描规则
// @param ctx 上下文
// @param id 扫描规则ID
// @return 错误信息
func (r *ScanRuleRepository) EnableScanRule(ctx context.Context, id uint) error {
	return r.UpdateScanRuleStatus(ctx, id, orchestrator.ScanRuleStatusEnabled)
}

// DisableScanRule 禁用扫描规则
// @param ctx 上下文
// @param id 扫描规则ID
// @return 错误信息
func (r *ScanRuleRepository) DisableScanRule(ctx context.Context, id uint) error {
	return r.UpdateScanRuleStatus(ctx, id, orchestrator.ScanRuleStatusDisabled)
}

// UpdateScanRuleStats 更新扫描规则统计信息
// @param ctx 上下文
// @param id 扫描规则ID
// @param executionCount 执行次数
// @param matchCount 匹配次数
// @param lastExecutionTime 最后执行时间
// @return 错误信息
func (r *ScanRuleRepository) UpdateScanRuleStats(ctx context.Context, id uint, executionCount, matchCount int, lastExecutionTime time.Time) error {
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanRule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"execution_count":     executionCount,
		"match_count":         matchCount,
		"last_execution_time": lastExecutionTime,
		"updated_at":          time.Now(),
	}).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_rule_update", "PUT", map[string]interface{}{
			"operation":           "update_scan_rule_stats",
			"rule_id":             id,
			"execution_count":     executionCount,
			"match_count":         matchCount,
			"last_execution_time": lastExecutionTime,
			"timestamp":           logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// IncrementExecutionCount 增加执行次数
// @param ctx 上下文
// @param id 扫描规则ID
// @return 错误信息
func (r *ScanRuleRepository) IncrementExecutionCount(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanRule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"execution_count":     gorm.Expr("execution_count + 1"),
		"last_execution_time": time.Now(),
		"updated_at":          time.Now(),
	}).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_rule_update", "PUT", map[string]interface{}{
			"operation": "increment_execution_count",
			"rule_id":   id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// IncrementMatchCount 增加匹配次数
// @param ctx 上下文
// @param id 扫描规则ID
// @return 错误信息
func (r *ScanRuleRepository) IncrementMatchCount(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanRule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"match_count": gorm.Expr("match_count + 1"),
		"updated_at":  time.Now(),
	}).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_rule_update", "PUT", map[string]interface{}{
			"operation": "increment_match_count",
			"rule_id":   id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// BeginTx 开始事务
// @return 事务对象和错误信息
func (r *ScanRuleRepository) BeginTx() (*gorm.DB, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		logger.LogError(tx.Error, "", 0, "", "scan_rule_transaction", "BEGIN", map[string]interface{}{
			"operation": "begin_transaction",
			"timestamp": logger.NowFormatted(),
		})
		return nil, tx.Error
	}
	return tx, nil
}

// UpdateScanRuleWithTx 使用事务更新扫描规则
// @param ctx 上下文
// @param tx 事务对象
// @param rule 扫描规则对象
// @return 错误信息
func (r *ScanRuleRepository) UpdateScanRuleWithTx(ctx context.Context, tx *gorm.DB, rule *orchestrator.ScanRule) error {
	rule.UpdatedAt = time.Now()
	err := tx.WithContext(ctx).Save(rule).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(rule.ID), "", "scan_rule_update_with_tx", "PUT", map[string]interface{}{
			"operation": "update_scan_rule_with_transaction",
			"rule_name": rule.Name,
			"rule_id":   rule.ID,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// DeleteScanRuleWithTx 使用事务删除扫描规则
// @param ctx 上下文
// @param tx 事务对象
// @param id 扫描规则ID
// @return 错误信息
func (r *ScanRuleRepository) DeleteScanRuleWithTx(ctx context.Context, tx *gorm.DB, id uint) error {
	err := tx.WithContext(ctx).Delete(&orchestrator.ScanRule{}, id).Error
	if err != nil {
		// 记录删除失败日志
		logger.LogError(err, "", id, "", "scan_rule_delete_with_tx", "DELETE", map[string]interface{}{
			"operation": "delete_scan_rule_with_transaction",
			"rule_id":   id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// UpdateScanRuleFields 使用map更新特定字段
// @param ctx 上下文
// @param id 扫描规则ID
// @param fields 要更新的字段
// @return 错误信息
func (r *ScanRuleRepository) UpdateScanRuleFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanRule{}).Where("id = ?", id).Updates(fields).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_rule_update", "PUT", map[string]interface{}{
			"operation": "update_scan_rule_fields",
			"rule_id":   id,
			"fields":    fields,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}
