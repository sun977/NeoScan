/*
 * 扫描规则服务层：扫描规则管理业务逻辑
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 处理扫描规则管理和配置相关的业务逻辑
 * @func:
 * 1.扫描规则配置管理
 * 2.扫描规则匹配和执行
 * 3.扫描规则统计分析
 * 4.扫描规则适用性检查
 */

//  核心业务功能:
//  	CreateScanRule - 创建扫描规则配置
//  	UpdateScanRule - 更新扫描规则配置
//  	GetScanRule - 获取扫描规则详情
//  	ListScanRules - 分页获取扫描规则列表
//  	DeleteScanRule - 删除扫描规则配置
//  状态管理功能:
//  	EnableScanRule - 启用扫描规则
//  	DisableScanRule - 禁用扫描规则
//  	ValidateScanRuleConfig - 验证扫描规则配置
//  规则匹配功能:
//  	MatchScanRules - 匹配适用的扫描规则
//  	EvaluateRuleCondition - 评估规则条件
//  	ExecuteRuleAction - 执行规则动作
//  统计分析功能:
//  	UpdateScanRuleStats - 更新扫描规则统计
//  	GetScanRuleStats - 获取扫描规则统计信息
//  	GetScanRulePerformance - 获取扫描规则性能指标
//  规则管理功能:
//  	GetScanRulesByType - 根据类型获取扫描规则
//  	GetScanRulesBySeverity - 根据严重程度获取扫描规则
//  	GetActiveRules - 获取活跃规则
//  	ImportScanRules - 导入扫描规则
//  	ExportScanRules - 导出扫描规则

package scan_config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"neomaster/internal/model/scan_config"
	"neomaster/internal/pkg/logger"
	scanConfigRepo "neomaster/internal/repository/scan_config"
)

// ScanRuleService 扫描规则服务结构体
// 负责处理扫描规则相关的业务逻辑
type ScanRuleService struct {
	scanRuleRepo *scanConfigRepo.ScanRuleRepository // 扫描规则仓库
}

// NewScanRuleService 创建扫描规则服务实例
// 注入必要的Repository依赖，遵循依赖注入原则
func NewScanRuleService(scanRuleRepo *scanConfigRepo.ScanRuleRepository) *ScanRuleService {
	return &ScanRuleService{
		scanRuleRepo: scanRuleRepo,
	}
}

// CreateScanRule 创建扫描规则配置
// @param ctx 上下文
// @param rule 扫描规则配置对象
// @return 创建的扫描规则配置和错误信息
func (s *ScanRuleService) CreateScanRule(ctx context.Context, rule *scan_config.ScanRule) (*scan_config.ScanRule, error) {
	// 参数验证 - Linus式：消除特殊情况
	if rule == nil {
		logger.LogError(errors.New("scan rule is nil"), "", 0, "", "create_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "create_scan_rule",
			"error":     "nil_rule",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则配置不能为空")
	}

	// 业务验证 - 检查规则名称唯一性
	if err := s.ValidateScanRuleConfig(ctx, rule); err != nil {
		logger.LogError(err, "", 0, "", "create_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "create_scan_rule",
			"error":     "validation_failed",
			"rule_name": rule.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("扫描规则配置验证失败: %w", err)
	}

	// 检查规则名称是否已存在
	exists, err := s.scanRuleRepo.ScanRuleExists(ctx, rule.Name)
	if err != nil {
		logger.LogError(err, "", 0, "", "create_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "create_scan_rule",
			"error":     "check_exists_failed",
			"rule_name": rule.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("检查扫描规则名称是否存在失败: %w", err)
	}

	if exists {
		logger.LogError(errors.New("scan rule name already exists"), "", 0, "", "create_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "create_scan_rule",
			"error":     "name_already_exists",
			"rule_name": rule.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则名称已存在")
	}

	// 设置默认值 - 简化数据结构
	s.setDefaultValues(rule)

	// 创建扫描规则配置
	if err := s.scanRuleRepo.CreateScanRule(ctx, rule); err != nil {
		logger.LogError(err, "", 0, "", "create_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "create_scan_rule",
			"error":     "create_failed",
			"rule_name": rule.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建扫描规则配置失败: %w", err)
	}

	// 更新最后执行时间
	rule.UpdatedAt = time.Now()

	// 记录成功日志
	logger.Info("create_scan_rule success", map[string]interface{}{
		"operation":    "create_scan_rule",
		"rule_name":    rule.Name,
		"rule_id":      rule.ID,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return rule, nil
}

// UpdateScanRule 更新扫描规则配置
// @param ctx 上下文
// @param id 扫描规则配置ID
// @param rule 更新的扫描规则配置对象
// @return 更新后的扫描规则配置和错误信息
func (s *ScanRuleService) UpdateScanRule(ctx context.Context, id uint, rule *scan_config.ScanRule) (*scan_config.ScanRule, error) {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan rule ID"), "", 0, "", "update_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则配置ID不能为0")
	}

	if rule == nil {
		logger.LogError(errors.New("scan rule is nil"), "", id, "", "update_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule",
			"error":     "nil_rule",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则配置不能为空")
	}

	// 检查扫描规则配置是否存在
	existingRule, err := s.scanRuleRepo.GetScanRuleByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "update_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule",
			"error":     "get_existing_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取现有扫描规则配置失败: %w", err)
	}

	if existingRule == nil {
		logger.LogError(errors.New("scan rule not found"), "", id, "", "update_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则配置不存在")
	}

	// 如果名称发生变化，检查新名称是否已存在
	if rule.Name != existingRule.Name {
		exists, err := s.scanRuleRepo.ScanRuleExists(ctx, rule.Name)
		if err != nil {
			logger.LogError(err, "", id, "", "update_scan_rule", "SERVICE", map[string]interface{}{
				"operation": "update_scan_rule",
				"error":     "check_name_exists_failed",
				"id":        id,
				"rule_name": rule.Name,
				"timestamp": logger.NowFormatted(),
			})
			return nil, fmt.Errorf("检查扫描规则名称是否存在失败: %w", err)
		}

		if exists {
			logger.LogError(errors.New("scan rule name already exists"), "", id, "", "update_scan_rule", "SERVICE", map[string]interface{}{
				"operation": "update_scan_rule",
				"error":     "name_already_exists",
				"id":        id,
				"rule_name": rule.Name,
				"timestamp": logger.NowFormatted(),
			})
			return nil, errors.New("扫描规则名称已存在")
		}
	}

	// 业务验证
	if err := s.ValidateScanRuleConfig(ctx, rule); err != nil {
		logger.LogError(err, "", id, "", "update_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule",
			"error":     "validation_failed",
			"id":        id,
			"rule_name": rule.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("扫描规则配置验证失败: %w", err)
	}

	// 保持ID和创建时间不变
	rule.ID = uint64(id)
	rule.CreatedAt = existingRule.CreatedAt

	// 更新扫描规则配置
	if err := s.scanRuleRepo.UpdateScanRule(ctx, rule); err != nil {
		logger.LogError(err, "", id, "", "update_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule",
			"error":     "update_failed",
			"id":        id,
			"rule_name": rule.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("更新扫描规则配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("update_scan_rule success", map[string]interface{}{
		"operation":    "update_scan_rule",
		"rule_name":    rule.Name,
		"rule_id":      id,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return rule, nil
}

// GetScanRule 获取扫描规则配置详情
// @param ctx 上下文
// @param id 扫描规则配置ID
// @return 扫描规则配置对象和错误信息
func (s *ScanRuleService) GetScanRule(ctx context.Context, id uint) (*scan_config.ScanRule, error) {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan rule ID"), "", 0, "", "get_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "get_scan_rule",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则配置ID不能为0")
	}

	// 获取扫描规则配置
	rule, err := s.scanRuleRepo.GetScanRuleByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "get_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "get_scan_rule",
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取扫描规则配置失败: %w", err)
	}

	if rule == nil {
		logger.LogError(errors.New("scan rule not found"), "", id, "", "get_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "get_scan_rule",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则配置不存在")
	}

	return rule, nil
}

// ListScanRules 分页获取扫描规则配置列表
// @param ctx 上下文
// @param offset 偏移量
// @param limit 限制数量
// @param ruleType 规则类型过滤（可选）
// @param status 状态过滤（可选）
// @param severity 严重程度过滤（可选）
// @return 扫描规则配置列表、总数和错误信息
func (s *ScanRuleService) ListScanRules(ctx context.Context, offset, limit int, ruleType *scan_config.ScanRuleType, status *scan_config.ScanRuleStatus, severity *scan_config.ScanRuleSeverity) ([]*scan_config.ScanRule, int64, error) {
	// 参数验证
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认分页大小
	}

	// 获取扫描规则配置列表
	rules, total, err := s.scanRuleRepo.GetScanRuleList(ctx, offset, limit, ruleType, severity, status)
	if err != nil {
		logger.LogError(err, "", 0, "", "list_scan_rules", "SERVICE", map[string]interface{}{
			"operation": "list_scan_rules",
			"error":     "list_failed",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, fmt.Errorf("获取扫描规则配置列表失败: %w", err)
	}

	return rules, total, nil
}

// DeleteScanRule 删除扫描规则配置
// @param ctx 上下文
// @param id 扫描规则配置ID
// @return 错误信息
func (s *ScanRuleService) DeleteScanRule(ctx context.Context, id uint) error {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan rule ID"), "", 0, "", "delete_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_rule",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描规则配置ID不能为0")
	}

	// 检查扫描规则配置是否存在
	rule, err := s.scanRuleRepo.GetScanRuleByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "delete_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_rule",
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取扫描规则配置失败: %w", err)
	}

	if rule == nil {
		logger.LogError(errors.New("scan rule not found"), "", id, "", "delete_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_rule",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描规则配置不存在")
	}

	// 删除扫描规则配置
	if err := s.scanRuleRepo.DeleteScanRule(ctx, id); err != nil {
		logger.LogError(err, "", id, "", "delete_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_rule",
			"error":     "delete_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("删除扫描规则配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("delete_scan_rule success", map[string]interface{}{
		"operation":    "delete_scan_rule",
		"rule_name":    rule.Name,
		"rule_id":      id,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return nil
}

// EnableScanRule 启用扫描规则
// @param ctx 上下文
// @param id 扫描规则配置ID
// @return 错误信息
func (s *ScanRuleService) EnableScanRule(ctx context.Context, id uint) error {
	return s.updateScanRuleStatus(ctx, id, scan_config.ScanRuleStatusEnabled, "enable_scan_rule")
}

// DisableScanRule 禁用扫描规则
// @param ctx 上下文
// @param id 扫描规则配置ID
// @return 错误信息
func (s *ScanRuleService) DisableScanRule(ctx context.Context, id uint) error {
	return s.updateScanRuleStatus(ctx, id, scan_config.ScanRuleStatusDisabled, "disable_scan_rule")
}

// ValidateScanRuleConfig 验证扫描规则配置
// @param ctx 上下文
// @param rule 扫描规则配置对象
// @return 错误信息
func (s *ScanRuleService) ValidateScanRuleConfig(ctx context.Context, rule *scan_config.ScanRule) error {
	// 基础字段验证
	if strings.TrimSpace(rule.Name) == "" {
		return errors.New("扫描规则名称不能为空")
	}

	if len(rule.Name) > 100 {
		return errors.New("扫描规则名称长度不能超过100个字符")
	}

	if len(rule.Description) > 500 {
		return errors.New("扫描规则描述长度不能超过500个字符")
	}

	// 条件配置验证
	if rule.Condition != "" {
		var conditions []map[string]interface{}
		if err := json.Unmarshal([]byte(rule.Condition), &conditions); err != nil {
			return fmt.Errorf("条件配置JSON格式无效: %w", err)
		}
	}

	// 动作配置验证
	if rule.Action != "" {
		var actions []map[string]interface{}
		if err := json.Unmarshal([]byte(rule.Action), &actions); err != nil {
			return fmt.Errorf("动作配置JSON格式无效: %w", err)
		}
	}

	// 参数配置验证
	if rule.Parameters != "" {
		var parameters map[string]interface{}
		if err := json.Unmarshal([]byte(rule.Parameters), &parameters); err != nil {
			return fmt.Errorf("参数配置JSON格式无效: %w", err)
		}
	}

	// 适用工具验证
	if rule.ApplicableTools != "" {
		// 验证适用工具格式
		tools := strings.Split(rule.ApplicableTools, ",")
		for _, tool := range tools {
			if strings.TrimSpace(tool) == "" {
				return fmt.Errorf("适用工具配置格式无效")
			}
		}
	}

	// 执行配置验证
	if rule.TimeoutSeconds <= 0 {
		rule.TimeoutSeconds = 30 // 设置默认超时时间
	}

	// 元数据验证
	if rule.Metadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(rule.Metadata), &metadata); err != nil {
			return fmt.Errorf("元数据JSON格式无效: %w", err)
		}
	}

	// 优先级验证
	if rule.Priority < 0 || rule.Priority > 100 {
		return errors.New("优先级必须在0-100之间")
	}

	return nil
}

// MatchScanRules 匹配适用的扫描规则
// @param ctx 上下文
// @param target 目标对象（资产、项目等）
// @param ruleType 规则类型过滤（可选）
// @return 匹配的扫描规则列表和错误信息
func (s *ScanRuleService) MatchScanRules(ctx context.Context, target map[string]interface{}, ruleType *scan_config.ScanRuleType) ([]*scan_config.ScanRule, error) {
	// 获取活跃的扫描规则
	rules, err := s.scanRuleRepo.GetActiveRules(ctx, ruleType)
	if err != nil {
		logger.LogError(err, "", 0, "", "match_scan_rules", "SERVICE", map[string]interface{}{
			"operation": "match_scan_rules",
			"error":     "get_active_rules_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取活跃扫描规则失败: %w", err)
	}

	var matchedRules []*scan_config.ScanRule

	// 遍历规则进行匹配
	for _, rule := range rules {
		// 检查规则适用性
		if s.isRuleApplicable(rule, target) {
			// 评估规则条件
			if s.evaluateRuleCondition(rule, target) {
				matchedRules = append(matchedRules, rule)
			}
		}
	}

	logger.LogSystemEvent("scan_rule", "match_scan_rules", "匹配扫描规则成功", logrus.InfoLevel, map[string]interface{}{
		"operation":     "match_scan_rules",
		"total_rules":   len(rules),
		"matched_rules": len(matchedRules),
		"timestamp":     logger.NowFormatted(),
	})

	return matchedRules, nil
}

// EvaluateRuleCondition 评估规则条件
// @param ctx 上下文
// @param rule 扫描规则
// @param target 目标对象
// @return 是否满足条件
func (s *ScanRuleService) EvaluateRuleCondition(ctx context.Context, rule *scan_config.ScanRule, target map[string]interface{}) bool {
	return s.evaluateRuleCondition(rule, target)
}

// ExecuteRuleAction 执行规则动作
// @param ctx 上下文
// @param rule 扫描规则
// @param target 目标对象
// @return 执行结果和错误信息
func (s *ScanRuleService) ExecuteRuleAction(ctx context.Context, rule *scan_config.ScanRule, target map[string]interface{}) (map[string]interface{}, error) {
	// 参数验证
	if rule == nil {
		return nil, errors.New("扫描规则不能为空")
	}

	// 解析动作配置
	var actions map[string]interface{}
	if rule.Action != "" {
		if err := json.Unmarshal([]byte(rule.Action), &actions); err != nil {
			return nil, fmt.Errorf("解析动作配置失败: %w", err)
		}
	}

	// TODO: 实现动作执行逻辑
	// 1. 根据动作类型执行相应操作
	// 2. 记录执行结果
	// 3. 更新统计信息

	result := map[string]interface{}{
		"rule_id":     rule.ID,
		"rule_name":   rule.Name,
		"executed_at": time.Now(),
		"status":      "success",
		"actions":     actions,
		"target":      target,
	}

	// 更新执行统计
	if err := s.scanRuleRepo.IncrementExecutionCount(ctx, uint(rule.ID)); err != nil {
		logger.LogError(err, "", uint(rule.ID), "", "execute_rule_action", "SERVICE", map[string]interface{}{
			"operation": "execute_rule_action",
			"error":     "update_stats_failed",
			"rule_id":   rule.ID,
			"timestamp": logger.NowFormatted(),
		})
		// 不返回错误，继续执行
	}

	logger.LogSystemEvent("scan_rule", "execute_rule_action", "执行规则动作成功", logrus.InfoLevel, map[string]interface{}{
		"operation": "execute_rule_action",
		"rule_name": rule.Name,
		"rule_id":   rule.ID,
		"status":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return result, nil
}

// BatchImportScanRules 批量导入扫描规则
// @param ctx 上下文
// @param req 导入请求参数
// @return map[string]interface{} 导入结果
// @return error 错误信息
func (s *ScanRuleService) BatchImportScanRules(ctx context.Context, req *scan_config.ImportScanRulesRequest) (map[string]interface{}, error) {
	logger.Info("开始批量导入扫描规则", map[string]interface{}{
		"path":      "service.scan_config.scan_rule",
		"operation": "batch_import_scan_rules",
		"option":    "scanRuleService.BatchImportScanRules",
		"func_name": "service.scan_config.scan_rule.BatchImportScanRules",
		"format":    req.Format,
		"overwrite": req.Overwrite,
		"validate":  req.Validate,
		"timestamp": logger.NowFormatted(),
	})

	// TODO: 实现批量导入逻辑
	// 1. 解析数据格式（JSON/YAML/XML）
	// 2. 验证规则数据
	// 3. 批量创建规则
	// 4. 返回导入结果统计

	result := map[string]interface{}{
		"total":     0,
		"success":   0,
		"failed":    0,
		"skipped":   0,
		"errors":    []string{},
		"overwrite": req.Overwrite,
	}

	return result, nil
}

// BatchUpdateScanRuleStatus 批量更新扫描规则状态
// @param ctx 上下文
// @param ruleIDs 规则ID列表
// @param status 目标状态
// @return map[string]interface{} 更新结果
// @return error 错误信息
func (s *ScanRuleService) BatchUpdateScanRuleStatus(ctx context.Context, ruleIDs []uint, status scan_config.ScanRuleStatus) (map[string]interface{}, error) {
	logger.Info("开始批量更新扫描规则状态", map[string]interface{}{
		"path":      "service.scan_config.scan_rule",
		"operation": "batch_update_scan_rule_status",
		"option":    "scanRuleService.BatchUpdateScanRuleStatus",
		"func_name": "service.scan_config.scan_rule.BatchUpdateScanRuleStatus",
		"rule_ids":  ruleIDs,
		"status":    status,
		"count":     len(ruleIDs),
		"timestamp": logger.NowFormatted(),
	})

	successCount := 0
	failedCount := 0
	errors := []string{}

	// 批量更新规则状态
	for _, ruleID := range ruleIDs {
		err := s.scanRuleRepo.UpdateScanRuleStatus(ctx, ruleID, status)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("规则ID %d 更新失败: %v", ruleID, err))
			logger.Error("更新扫描规则状态失败", map[string]interface{}{
				"path":      "service.scan_config.scan_rule",
				"operation": "batch_update_scan_rule_status",
				"option":    "scanRuleRepo.UpdateScanRuleStatus",
				"func_name": "service.scan_config.scan_rule.BatchUpdateScanRuleStatus",
				"rule_id":   ruleID,
				"status":    status,
				"error":     err.Error(),
				"timestamp": logger.NowFormatted(),
			})
		} else {
			successCount++
		}
	}

	result := map[string]interface{}{
		"total":   len(ruleIDs),
		"success": successCount,
		"failed":  failedCount,
		"errors":  errors,
		"status":  status,
	}

	logger.Info("批量更新扫描规则状态完成", map[string]interface{}{
		"path":      "service.scan_config.scan_rule",
		"operation": "batch_update_scan_rule_status",
		"option":    "success",
		"func_name": "service.scan_config.scan_rule.BatchUpdateScanRuleStatus",
		"result":    result,
		"timestamp": logger.NowFormatted(),
	})

	return result, nil
}

// UpdateScanRuleStats 更新扫描规则统计
// @param ctx 上下文
// @param id 扫描规则配置ID
// @param matched 是否匹配
// @return 错误信息
func (s *ScanRuleService) UpdateScanRuleStats(ctx context.Context, id uint, matched bool) error {
	// 参数验证
	if id == 0 {
		return errors.New("扫描规则配置ID不能为0")
	}

	// 更新执行次数
	if err := s.scanRuleRepo.IncrementExecutionCount(ctx, id); err != nil {
		logger.LogError(err, "", id, "", "update_scan_rule_stats", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule_stats",
			"error":     "increment_execution_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("更新执行次数失败: %w", err)
	}

	// 如果匹配，更新匹配次数
	if matched {
		if err := s.scanRuleRepo.IncrementMatchCount(ctx, id); err != nil {
			logger.LogError(err, "", id, "", "update_scan_rule_stats", "SERVICE", map[string]interface{}{
				"operation": "update_scan_rule_stats",
				"error":     "increment_match_failed",
				"id":        id,
				"timestamp": logger.NowFormatted(),
			})
			return fmt.Errorf("更新匹配次数失败: %w", err)
		}
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_rule", "update_scan_rule_stats", "更新扫描规则统计成功", logrus.InfoLevel, map[string]interface{}{
		"operation": "update_scan_rule_stats",
		"rule_id":   id,
		"matched":   matched,
		"status":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return nil
}

// GetScanRuleStats 获取扫描规则统计信息
// @param ctx 上下文
// @return 统计信息和错误信息
func (s *ScanRuleService) GetScanRuleStats(ctx context.Context) (map[string]interface{}, error) {
	// TODO: 实现统计逻辑
	// 统计总数、各类型数量、各状态数量、各严重程度数量等

	stats := map[string]interface{}{
		"total_count":    0,
		"active_count":   0,
		"inactive_count": 0,
		"by_type": map[string]int{
			"vulnerability":     0,
			"compliance":        0,
			"security_baseline": 0,
			"custom":            0,
		},
		"by_severity": map[string]int{
			"critical": 0,
			"high":     0,
			"medium":   0,
			"low":      0,
			"info":     0,
		},
	}

	return stats, nil
}

// GetScanRulePerformance 获取扫描规则性能指标
// @param ctx 上下文
// @param id 扫描规则配置ID
// @return 性能指标和错误信息
func (s *ScanRuleService) GetScanRulePerformance(ctx context.Context, id uint) (map[string]interface{}, error) {
	if id == 0 {
		return nil, errors.New("扫描规则ID不能为0")
	}

	// 获取扫描规则配置
	rule, err := s.scanRuleRepo.GetScanRuleByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取扫描规则配置失败: %w", err)
	}

	if rule == nil {
		return nil, errors.New("扫描规则配置不存在")
	}

	// 计算性能指标
	var matchRate float64
	if rule.ExecutionCount > 0 {
		matchRate = float64(rule.MatchCount) / float64(rule.ExecutionCount) * 100
	}

	performance := map[string]interface{}{
		"rule_id":         id,
		"rule_name":       rule.Name,
		"execution_count": rule.ExecutionCount,
		"match_count":     rule.MatchCount,
		"match_rate":      matchRate,
		"last_executed":   rule.UpdatedAt,
		"priority":        rule.Priority,
		"severity":        rule.Severity,
	}

	return performance, nil
}

// GetScanRulesByType 根据类型获取扫描规则
// @param ctx 上下文
// @param ruleType 规则类型
// @return 扫描规则列表和错误信息
func (s *ScanRuleService) GetScanRulesByType(ctx context.Context, ruleType scan_config.ScanRuleType) ([]*scan_config.ScanRule, error) {
	rules, err := s.scanRuleRepo.GetScanRulesByType(ctx, ruleType)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_scan_rules_by_type", "SERVICE", map[string]interface{}{
			"operation": "get_scan_rules_by_type",
			"error":     "get_failed",
			"rule_type": ruleType,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("根据类型获取扫描规则失败: %w", err)
	}

	return rules, nil
}

// GetScanRulesBySeverity 根据严重程度获取扫描规则
// @param ctx 上下文
// @param severity 严重程度
// @return 扫描规则列表和错误信息
func (s *ScanRuleService) GetScanRulesBySeverity(ctx context.Context, severity scan_config.ScanRuleSeverity) ([]*scan_config.ScanRule, error) {
	rules, err := s.scanRuleRepo.GetScanRulesBySeverity(ctx, severity)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_scan_rules_by_severity", "SERVICE", map[string]interface{}{
			"operation": "get_scan_rules_by_severity",
			"error":     "get_failed",
			"severity":  severity,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("根据严重程度获取扫描规则失败: %w", err)
	}

	return rules, nil
}

// GetActiveScanRules 获取活跃扫描规则
// @param ctx 上下文
// @return 活跃扫描规则列表和错误信息
func (s *ScanRuleService) GetActiveScanRules(ctx context.Context) ([]*scan_config.ScanRule, error) {
	return s.GetActiveRules(ctx, nil)
}

// GetScanRuleMetrics 获取扫描规则指标
// @param ctx 上下文
// @param id 扫描规则ID
// @return 指标信息和错误信息
func (s *ScanRuleService) GetScanRuleMetrics(ctx context.Context, id uint) (map[string]interface{}, error) {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan rule ID"), "", 0, "", "get_scan_rule_metrics", "SERVICE", map[string]interface{}{
			"operation": "get_scan_rule_metrics",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则配置ID不能为0")
	}

	// 获取扫描规则详情
	rule, err := s.scanRuleRepo.GetScanRuleByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "get_scan_rule_metrics", "SERVICE", map[string]interface{}{
			"operation": "get_scan_rule_metrics",
			"error":     "get_rule_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取扫描规则失败: %w", err)
	}

	if rule == nil {
		logger.LogError(errors.New("scan rule not found"), "", id, "", "get_scan_rule_metrics", "SERVICE", map[string]interface{}{
			"operation": "get_scan_rule_metrics",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描规则不存在")
	}

	// 构建指标数据
	metrics := map[string]interface{}{
		"rule_id":             rule.ID,
		"rule_name":           rule.Name,
		"execution_count":     rule.ExecutionCount,
		"match_count":         rule.MatchCount,
		"error_count":         rule.ErrorCount,
		"avg_execution_time":  rule.AvgExecutionTime,
		"max_execution_time":  rule.MaxExecutionTime,
		"last_execution_time": rule.UpdatedAt,
		"success_rate":        0.0,
		"match_rate":          0.0,
	}

	// 计算成功率
	if rule.ExecutionCount > 0 {
		successCount := rule.ExecutionCount - rule.ErrorCount
		metrics["success_rate"] = float64(successCount) / float64(rule.ExecutionCount) * 100
		metrics["match_rate"] = float64(rule.MatchCount) / float64(rule.ExecutionCount) * 100
	}

	// 记录成功日志
	logger.LogSystemEvent("scan_rule", "get_scan_rule_metrics", "获取扫描规则指标成功", logrus.InfoLevel, map[string]interface{}{
		"operation": "get_scan_rule_metrics",
		"rule_id":   id,
		"status":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return metrics, nil
}

// UpdateScanRuleStatus 更新扫描规则状态
// @param ctx 上下文
// @param id 扫描规则ID
// @param status 新状态
// @return 错误信息
func (s *ScanRuleService) UpdateScanRuleStatus(ctx context.Context, id uint, status string) error {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan rule ID"), "", 0, "", "update_scan_rule_status", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule_status",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描规则配置ID不能为0")
	}

	if status == "" {
		logger.LogError(errors.New("status is empty"), "", id, "", "update_scan_rule_status", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule_status",
			"error":     "empty_status",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("状态不能为空")
	}

	// 转换状态字符串为枚举类型
	var ruleStatus scan_config.ScanRuleStatus
	switch strings.ToLower(status) {
	case "enabled", "active":
		ruleStatus = scan_config.ScanRuleStatusEnabled
	case "disabled", "inactive":
		ruleStatus = scan_config.ScanRuleStatusDisabled
	case "draft":
		ruleStatus = scan_config.ScanRuleStatusTesting
	default:
		logger.LogError(errors.New("invalid status"), "", id, "", "update_scan_rule_status", "SERVICE", map[string]interface{}{
			"operation": "update_scan_rule_status",
			"error":     "invalid_status",
			"id":        id,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("无效的状态: %s", status)
	}

	return s.updateScanRuleStatus(ctx, id, ruleStatus, "update_scan_rule_status")
}

// GetActiveRules 获取活跃规则
// @param ctx 上下文
// @param ruleType 规则类型过滤（可选）
// @return 活跃扫描规则列表和错误信息
func (s *ScanRuleService) GetActiveRules(ctx context.Context, ruleType *scan_config.ScanRuleType) ([]*scan_config.ScanRule, error) {
	rules, err := s.scanRuleRepo.GetActiveRules(ctx, ruleType)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_active_rules", "SERVICE", map[string]interface{}{
			"operation": "get_active_rules",
			"error":     "get_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取活跃扫描规则失败: %w", err)
	}

	return rules, nil
}

// ImportScanRules 导入扫描规则
// @param ctx 上下文
// @param rules 扫描规则列表
// @return 导入结果和错误信息
func (s *ScanRuleService) ImportScanRules(ctx context.Context, rules []*scan_config.ScanRule) (map[string]interface{}, error) {
	if len(rules) == 0 {
		return nil, errors.New("导入的扫描规则列表不能为空")
	}

	var successCount, failureCount int
	var errors []string

	for _, rule := range rules {
		// 验证规则配置
		if err := s.ValidateScanRuleConfig(ctx, rule); err != nil {
			failureCount++
			errors = append(errors, fmt.Sprintf("规则 %s 验证失败: %v", rule.Name, err))
			continue
		}

		// 检查规则是否已存在
		exists, err := s.scanRuleRepo.ScanRuleExists(ctx, rule.Name)
		if err != nil {
			failureCount++
			errors = append(errors, fmt.Sprintf("检查规则 %s 是否存在失败: %v", rule.Name, err))
			continue
		}

		if exists {
			failureCount++
			errors = append(errors, fmt.Sprintf("规则 %s 已存在", rule.Name))
			continue
		}

		// 设置默认值
		s.setDefaultValues(rule)

		// 创建规则
		if err := s.scanRuleRepo.CreateScanRule(ctx, rule); err != nil {
			failureCount++
			errors = append(errors, fmt.Sprintf("创建规则 %s 失败: %v", rule.Name, err))
			continue
		}

		successCount++
	}

	result := map[string]interface{}{
		"total_count":   len(rules),
		"success_count": successCount,
		"failure_count": failureCount,
		"errors":        errors,
		"imported_at":   time.Now(),
	}

	logger.LogSystemEvent("scan_rule", "import_scan_rules", "导入扫描规则成功", logrus.InfoLevel, map[string]interface{}{
		"operation":     "import_scan_rules",
		"total_count":   len(rules),
		"success_count": successCount,
		"failure_count": failureCount,
		"timestamp":     logger.NowFormatted(),
	})

	return result, nil
}

// ExportScanRules 导出扫描规则
// @param ctx 上下文
// @param ruleType 规则类型过滤（可选）
// @param status 状态过滤（可选）
// @return 扫描规则列表和错误信息
func (s *ScanRuleService) ExportScanRules(ctx context.Context, ruleType *scan_config.ScanRuleType, status *scan_config.ScanRuleStatus) ([]*scan_config.ScanRule, error) {
	// 获取扫描规则列表（不分页，获取所有）
	rules, _, err := s.scanRuleRepo.GetScanRuleList(ctx, 0, 10000, ruleType, nil, status)
	if err != nil {
		logger.LogError(err, "", 0, "", "export_scan_rules", "SERVICE", map[string]interface{}{
			"operation": "export_scan_rules",
			"error":     "get_rules_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取扫描规则列表失败: %w", err)
	}

	logger.LogSystemEvent("scan_rule", "export_scan_rules", "导出扫描规则成功", logrus.InfoLevel, map[string]interface{}{
		"operation":   "export_scan_rules",
		"rules_count": len(rules),
		"timestamp":   logger.NowFormatted(),
	})

	return rules, nil
}

// 私有方法：更新扫描规则状态
func (s *ScanRuleService) updateScanRuleStatus(ctx context.Context, id uint, status scan_config.ScanRuleStatus, operation string) error {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan rule ID"), "", 0, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描规则配置ID不能为0")
	}

	// 检查扫描规则配置是否存在
	rule, err := s.scanRuleRepo.GetScanRuleByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取扫描规则配置失败: %w", err)
	}

	if rule == nil {
		logger.LogError(errors.New("scan rule not found"), "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描规则配置不存在")
	}

	// 更新状态
	if err := s.scanRuleRepo.UpdateScanRuleStatus(ctx, id, status); err != nil {
		logger.LogError(err, "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "update_status_failed",
			"id":        id,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("更新扫描规则状态失败: %w", err)
	}

	// 记录成功日志
	logger.Info(operation+" success", map[string]interface{}{
		"operation":    operation,
		"rule_name":    rule.Name,
		"rule_id":      id,
		"status":       status,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return nil
}

// 私有方法：检查规则适用性
func (s *ScanRuleService) isRuleApplicable(rule *scan_config.ScanRule, target map[string]interface{}) bool {
	// 解析适用工具
	if rule.ApplicableTools == "" {
		return true // 没有限制，适用于所有目标
	}

	// 简单的工具匹配逻辑
	tools := strings.Split(rule.ApplicableTools, ",")
	targetTool, exists := target["tool"].(string)
	if !exists {
		return false // 目标没有指定工具
	}

	for _, tool := range tools {
		if strings.TrimSpace(tool) == targetTool {
			return true
		}
	}

	// TODO: 实现适用性检查逻辑
	// 1. 检查目标类型是否匹配
	// 2. 检查目标属性是否满足条件
	// 3. 检查环境条件是否满足

	return true
}

// 私有方法：评估规则条件
func (s *ScanRuleService) evaluateRuleCondition(rule *scan_config.ScanRule, target map[string]interface{}) bool {
	// 解析条件配置
	if rule.Condition == "" {
		return true // 没有条件，直接通过
	}

	var conditions map[string]interface{}
	if err := json.Unmarshal([]byte(rule.Condition), &conditions); err != nil {
		logger.LogError(err, "", uint(rule.ID), "", "evaluate_rule_condition", "SERVICE", map[string]interface{}{
			"operation": "evaluate_rule_condition",
			"error":     "parse_conditions_failed",
			"rule_id":   rule.ID,
			"timestamp": logger.NowFormatted(),
		})
		return false
	}

	// TODO: 实现条件评估逻辑
	// 1. 解析条件表达式
	// 2. 根据目标属性计算条件值
	// 3. 返回评估结果

	return true
}

// 私有方法：设置默认值
func (s *ScanRuleService) setDefaultValues(rule *scan_config.ScanRule) {
	// 设置默认状态
	if rule.Status == 0 {
		rule.Status = scan_config.ScanRuleStatusDisabled
	}

	if rule.Type == "" {
		rule.Type = scan_config.ScanRuleTypeCustom
	}

	if rule.Severity == "" {
		rule.Severity = scan_config.ScanRuleSeverityMedium
	}

	if rule.Condition == "" {
		rule.Condition = "{}"
	}

	if rule.Action == "" {
		rule.Action = "{}"
	}

	if rule.Parameters == "" {
		rule.Parameters = "{}"
	}

	// 设置默认的适用工具
	if rule.ApplicableTools == "" {
		rule.ApplicableTools = ""
	}

	// 设置默认的超时时间
	if rule.TimeoutSeconds == 0 {
		rule.TimeoutSeconds = 300 // 默认5分钟
	}

	if rule.Metadata == "" {
		rule.Metadata = "{}"
	}

	// 设置默认优先级
	if rule.Priority == 0 {
		rule.Priority = 50 // 默认中等优先级
	}

	// 设置时间戳
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
}

// TestScanRule 测试扫描规则
func (s *ScanRuleService) TestScanRule(ctx context.Context, ruleID uint, target map[string]interface{}) (map[string]interface{}, error) {
	// 获取扫描规则
	rule, err := s.GetScanRule(ctx, ruleID)
	if err != nil {
		logger.LogError(err, "", 0, "", "test_scan_rule", "SERVICE", map[string]interface{}{
			"operation": "test_scan_rule",
			"rule_id":   ruleID,
			"error":     "get_scan_rule_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取扫描规则失败: %w", err)
	}

	// 检查规则状态
	if rule.Status != scan_config.ScanRuleStatusEnabled {
		return map[string]interface{}{
			"matched":    false,
			"reason":     "规则未激活",
			"rule_id":    ruleID,
			"rule_name":  rule.Name,
			"status":     rule.Status,
			"timestamp":  time.Now().Format(time.RFC3339),
		}, nil
	}

	// 检查规则适用性
	applicable := s.isRuleApplicable(rule, target)
	if !applicable {
		return map[string]interface{}{
			"matched":    false,
			"reason":     "规则不适用于当前目标",
			"rule_id":    ruleID,
			"rule_name":  rule.Name,
			"timestamp":  time.Now().Format(time.RFC3339),
		}, nil
	}

	// 评估规则条件
	matched := s.evaluateRuleCondition(rule, target)
	
	result := map[string]interface{}{
		"matched":    matched,
		"rule_id":    ruleID,
		"rule_name":  rule.Name,
		"rule_type":  rule.Type,
		"severity":   rule.Severity,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if matched {
		result["reason"] = "规则条件匹配成功"
		// 执行规则动作（测试模式，不实际执行）
		action, err := s.ExecuteRuleAction(ctx, rule, target)
		if err != nil {
			result["action_error"] = err.Error()
		} else {
			result["action_result"] = action
		}
	} else {
		result["reason"] = "规则条件不匹配"
	}

	logger.LogBusinessOperation("test_scan_rule", 0, "", "", "", "success", "测试扫描规则成功", map[string]interface{}{
		"operation": "test_scan_rule",
		"rule_id":   ruleID,
		"matched":   matched,
		"timestamp": logger.NowFormatted(),
	})

	return result, nil
}
