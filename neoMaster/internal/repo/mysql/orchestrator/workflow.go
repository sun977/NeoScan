/*
 * 工作流配置仓库层：工作流配置数据访问
 * @author: Sun977
 * @date: 2025.10.11
 * @description: 单纯数据访问，不应该包含业务逻辑
 * @func:
 * 1.创建工作流配置
 * 2.更新工作流配置
 * 3.删除工作流配置
 * 4.工作流状态变更等
 */

//  基础CRUD操作:
//  	CreateWorkflowConfig - 创建工作流配置
//  	GetWorkflowConfigByID - 根据ID获取工作流配置
//  	GetWorkflowConfigByName - 根据工作流名获取工作流配置
//  	UpdateWorkflowConfig - 更新工作流配置信息
//  	DeleteWorkflowConfig - 软删除工作流配置
//  高级查询功能:
//  	GetWorkflowConfigList - 分页获取工作流配置列表
//  	GetWorkflowConfigsByProject - 根据项目获取工作流配置
//  	GetActiveWorkflowConfigs - 获取活跃的工作流配置
//  	WorkflowConfigExists - 检查工作流配置是否存在
//  	GetWorkflowConfigsByTrigger - 根据触发类型获取工作流配置
//  状态管理:
//  	UpdateWorkflowConfigStatus - 更新工作流配置状态
//  	EnableWorkflowConfig - 启用工作流配置
//  	DisableWorkflowConfig - 禁用工作流配置
//  统计功能:
//  	UpdateWorkflowConfigStats - 更新工作流配置统计信息
//  	IncrementExecutionCount - 增加执行次数
//  事务支持:
//  	BeginTx - 开始事务
//  	UpdateWorkflowConfigWithTx - 事务更新工作流配置
//  	DeleteWorkflowConfigWithTx - 事务删除工作流配置
//  字段更新:
//  	UpdateWorkflowConfigFields - 使用map更新特定字段

package orchestrator

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// WorkflowConfigRepository 工作流配置仓库结构体
// 负责处理工作流配置相关的数据访问，不包含业务逻辑
type WorkflowConfigRepository struct {
	db *gorm.DB // 数据库连接
}

// NewWorkflowConfigRepository 创建工作流配置仓库实例
// 注入数据库连接，专注于数据访问操作
func NewWorkflowConfigRepository(db *gorm.DB) *WorkflowConfigRepository {
	return &WorkflowConfigRepository{
		db: db,
	}
}

// CreateWorkflowConfig 创建工作流配置
// @param ctx 上下文
// @param config 工作流配置对象
// @return 错误信息
func (r *WorkflowConfigRepository) CreateWorkflowConfig(ctx context.Context, config *orchestrator.WorkflowConfig) error {
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Create(config).Error
	if err != nil {
		// 记录创建失败日志
		logger.LogBusinessError(err, "", 0, "", "workflow_config_create", "POST", map[string]interface{}{
			"operation":     "create_workflow_config",
			"workflow_name": config.Name,
			"project_id":    config.ProjectID,
			"timestamp":     logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// GetWorkflowConfigByID 根据ID获取工作流配置
// @param ctx 上下文
// @param id 工作流配置ID
// @return 工作流配置对象和错误信息
func (r *WorkflowConfigRepository) GetWorkflowConfigByID(ctx context.Context, id uint) (*orchestrator.WorkflowConfig, error) {
	var config orchestrator.WorkflowConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogBusinessError(fmt.Errorf("workflow config not found"), "", 0, "", "workflow_config_get", "GET", map[string]interface{}{
				"operation": "get_workflow_config_by_id",
				"id":        id,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogBusinessError(err, "", 0, "", "workflow_config_get", "GET", map[string]interface{}{
			"operation": "get_workflow_config_by_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &config, nil
}

// GetWorkflowConfigByName 根据工作流名获取工作流配置
// @param ctx 上下文
// @param name 工作流名称
// @param projectID 项目ID（可选，用于区分不同项目的同名工作流）
// @return 工作流配置对象和错误信息
func (r *WorkflowConfigRepository) GetWorkflowConfigByName(ctx context.Context, name string, projectID *uint) (*orchestrator.WorkflowConfig, error) {
	var config orchestrator.WorkflowConfig
	query := r.db.WithContext(ctx).Where("name = ?", name)

	if projectID != nil {
		query = query.Where("project_config_id = ?", *projectID)
	}

	err := query.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogBusinessError(fmt.Errorf("workflow config not found"), "", 0, "", "workflow_config_get", "GET", map[string]interface{}{
				"operation":     "get_workflow_config_by_name",
				"workflow_name": name,
				"project_id":    projectID,
				"timestamp":     logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogBusinessError(err, "", 0, "", "workflow_config_get", "GET", map[string]interface{}{
			"operation":     "get_workflow_config_by_name",
			"workflow_name": name,
			"project_id":    projectID,
			"timestamp":     logger.NowFormatted(),
		})
		return nil, err
	}
	return &config, nil
}

// UpdateWorkflowConfig 更新工作流配置
// @param ctx 上下文
// @param config 工作流配置对象
// @return 错误信息
func (r *WorkflowConfigRepository) UpdateWorkflowConfig(ctx context.Context, config *orchestrator.WorkflowConfig) error {
	config.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(config).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogBusinessError(err, "", uint(config.ID), "", "workflow_config_update", "PUT", map[string]interface{}{
			"operation":     "update_workflow_config",
			"workflow_name": config.Name,
			"config_id":     config.ID,
			"timestamp":     logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// DeleteWorkflowConfig 软删除工作流配置
// @param ctx 上下文
// @param id 工作流配置ID
// @return 错误信息
func (r *WorkflowConfigRepository) DeleteWorkflowConfig(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Delete(&orchestrator.WorkflowConfig{}, id).Error
	if err != nil {
		// 记录删除失败日志
		logger.LogBusinessError(err, "", id, "", "workflow_config_delete", "DELETE", map[string]interface{}{
			"operation": "delete_workflow_config",
			"config_id": id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// GetWorkflowConfigList 分页获取工作流配置列表
// @param ctx 上下文
// @param offset 偏移量
// @param limit 限制数量
// @param projectID 项目ID过滤（可选）
// @param status 状态过滤（可选）
// @return 工作流配置列表、总数和错误信息
func (r *WorkflowConfigRepository) GetWorkflowConfigList(ctx context.Context, offset, limit int, projectID *uint, status *orchestrator.WorkflowStatus) ([]*orchestrator.WorkflowConfig, int64, error) {
	var configs []*orchestrator.WorkflowConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&orchestrator.WorkflowConfig{})

	// 项目ID过滤
	if projectID != nil {
		query = query.Where("project_config_id = ?", *projectID)
	}

	// 状态过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogBusinessError(err, "", 0, "", "workflow_config_list", "GET", map[string]interface{}{
			"operation": "get_workflow_config_list_count",
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, err
	}

	// 获取分页数据
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&configs).Error
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "workflow_config_list", "GET", map[string]interface{}{
			"operation": "get_workflow_config_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, err
	}

	return configs, total, nil
}

// GetWorkflowConfigsByProject 根据项目获取工作流配置
// @param ctx 上下文
// @param projectID 项目ID
// @return 工作流配置列表和错误信息
func (r *WorkflowConfigRepository) GetWorkflowConfigsByProject(ctx context.Context, projectID uint) ([]*orchestrator.WorkflowConfig, error) {
	var configs []*orchestrator.WorkflowConfig
	err := r.db.WithContext(ctx).Where("project_config_id = ? AND status = ?", projectID, orchestrator.WorkflowStatusActive).Find(&configs).Error
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "workflow_config_list", "GET", map[string]interface{}{
			"operation":  "get_workflow_configs_by_project",
			"project_id": projectID,
			"timestamp":  logger.NowFormatted(),
		})
		return nil, err
	}
	return configs, nil
}

// GetActiveWorkflowConfigs 获取活跃的工作流配置
// @param ctx 上下文
// @return 工作流配置列表和错误信息
func (r *WorkflowConfigRepository) GetActiveWorkflowConfigs(ctx context.Context) ([]*orchestrator.WorkflowConfig, error) {
	var configs []*orchestrator.WorkflowConfig
	err := r.db.WithContext(ctx).Where("status = ?", orchestrator.WorkflowStatusActive).Find(&configs).Error
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "workflow_config_list", "GET", map[string]interface{}{
			"operation": "get_active_workflow_configs",
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return configs, nil
}

// WorkflowConfigExists 检查工作流配置是否存在
// @param ctx 上下文
// @param name 工作流名称
// @param projectID 项目ID（可选）
// @return 是否存在和错误信息
func (r *WorkflowConfigRepository) WorkflowConfigExists(ctx context.Context, name string, projectID *uint) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&orchestrator.WorkflowConfig{}).Where("name = ?", name)

	if projectID != nil {
		query = query.Where("project_config_id = ?", *projectID)
	}

	err := query.Count(&count).Error
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "workflow_config_exists", "GET", map[string]interface{}{
			"operation":     "workflow_config_exists",
			"workflow_name": name,
			"project_id":    projectID,
			"timestamp":     logger.NowFormatted(),
		})
		return false, err
	}
	return count > 0, nil
}

// GetWorkflowConfigsByTrigger 根据触发类型获取工作流配置
// @param ctx 上下文
// @param triggerType 触发类型
// @return 工作流配置列表和错误信息
func (r *WorkflowConfigRepository) GetWorkflowConfigsByTrigger(ctx context.Context, triggerType orchestrator.WorkflowTriggerType) ([]*orchestrator.WorkflowConfig, error) {
	var configs []*orchestrator.WorkflowConfig
	err := r.db.WithContext(ctx).Where("trigger_type = ? AND status = ?", triggerType, orchestrator.WorkflowStatusActive).Find(&configs).Error
	if err != nil {
		logger.LogBusinessError(err, "", 0, "", "workflow_config_list", "GET", map[string]interface{}{
			"operation":    "get_workflow_configs_by_trigger",
			"trigger_type": triggerType,
			"timestamp":    logger.NowFormatted(),
		})
		return nil, err
	}
	return configs, nil
}

// UpdateWorkflowConfigStatus 更新工作流配置状态
// @param ctx 上下文
// @param id 工作流配置ID
// @param status 新状态
// @return 错误信息
func (r *WorkflowConfigRepository) UpdateWorkflowConfigStatus(ctx context.Context, id uint, status orchestrator.WorkflowStatus) error {
	err := r.db.WithContext(ctx).Model(&orchestrator.WorkflowConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		logger.LogBusinessError(err, "", id, "", "workflow_config_update", "PUT", map[string]interface{}{
			"operation": "update_workflow_config_status",
			"config_id": id,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// EnableWorkflowConfig 启用工作流配置
// @param ctx 上下文
// @param id 工作流配置ID
// @return 错误信息
func (r *WorkflowConfigRepository) EnableWorkflowConfig(ctx context.Context, id uint) error {
	return r.UpdateWorkflowConfigStatus(ctx, id, orchestrator.WorkflowStatusActive)
}

// DisableWorkflowConfig 禁用工作流配置
// @param ctx 上下文
// @param id 工作流配置ID
// @return 错误信息
func (r *WorkflowConfigRepository) DisableWorkflowConfig(ctx context.Context, id uint) error {
	return r.UpdateWorkflowConfigStatus(ctx, id, orchestrator.WorkflowStatusInactive)
}

// UpdateWorkflowConfigStats 更新工作流配置统计信息
// @param ctx 上下文
// @param id 工作流配置ID
// @param executionCount 执行次数
// @param successCount 成功次数
// @param failureCount 失败次数
// @param avgExecutionTime 平均执行时间
// @return 错误信息
func (r *WorkflowConfigRepository) UpdateWorkflowConfigStats(ctx context.Context, id uint, executionCount, successCount, failureCount int, avgExecutionTime float64) error {
	err := r.db.WithContext(ctx).Model(&orchestrator.WorkflowConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"execution_count":     executionCount,
		"success_count":       successCount,
		"failure_count":       failureCount,
		"avg_execution_time":  avgExecutionTime,
		"last_execution_time": time.Now(),
		"updated_at":          time.Now(),
	}).Error
	if err != nil {
		logger.LogBusinessError(err, "", id, "", "workflow_config_update", "PUT", map[string]interface{}{
			"operation":          "update_workflow_config_stats",
			"config_id":          id,
			"execution_count":    executionCount,
			"success_count":      successCount,
			"failure_count":      failureCount,
			"avg_execution_time": avgExecutionTime,
			"timestamp":          logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// IncrementExecutionCount 增加执行次数
// @param ctx 上下文
// @param id 工作流配置ID
// @param success 是否成功
// @param executionTime 执行时间（秒）
// @return 错误信息
func (r *WorkflowConfigRepository) IncrementExecutionCount(ctx context.Context, id uint, success bool, executionTime float64) error {
	updates := map[string]interface{}{
		"execution_count":     gorm.Expr("execution_count + 1"),
		"last_execution_time": time.Now(),
		"updated_at":          time.Now(),
	}

	if success {
		updates["success_count"] = gorm.Expr("success_count + 1")
	} else {
		updates["failure_count"] = gorm.Expr("failure_count + 1")
	}

	// 更新平均执行时间（简化计算，实际应该考虑历史数据）
	if executionTime > 0 {
		updates["avg_execution_time"] = executionTime
	}

	err := r.db.WithContext(ctx).Model(&orchestrator.WorkflowConfig{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		logger.LogBusinessError(err, "", id, "", "workflow_config_update", "PUT", map[string]interface{}{
			"operation":      "increment_execution_count",
			"config_id":      id,
			"success":        success,
			"execution_time": executionTime,
			"timestamp":      logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// BeginTx 开始事务
// @return 事务对象和错误信息
func (r *WorkflowConfigRepository) BeginTx() (*gorm.DB, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		logger.LogBusinessError(tx.Error, "", 0, "", "workflow_config_transaction", "BEGIN", map[string]interface{}{
			"operation": "begin_transaction",
			"timestamp": logger.NowFormatted(),
		})
		return nil, tx.Error
	}
	return tx, nil
}

// UpdateWorkflowConfigWithTx 使用事务更新工作流配置
// @param ctx 上下文
// @param tx 事务对象
// @param config 工作流配置对象
// @return 错误信息
func (r *WorkflowConfigRepository) UpdateWorkflowConfigWithTx(ctx context.Context, tx *gorm.DB, config *orchestrator.WorkflowConfig) error {
	config.UpdatedAt = time.Now()
	err := tx.WithContext(ctx).Save(config).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogBusinessError(err, "", uint(config.ID), "", "workflow_config_update_with_tx", "PUT", map[string]interface{}{
			"operation":     "update_workflow_config_with_transaction",
			"workflow_name": config.Name,
			"config_id":     config.ID,
			"timestamp":     logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// DeleteWorkflowConfigWithTx 使用事务删除工作流配置
// @param ctx 上下文
// @param tx 事务对象
// @param id 工作流配置ID
// @return 错误信息
func (r *WorkflowConfigRepository) DeleteWorkflowConfigWithTx(ctx context.Context, tx *gorm.DB, id uint) error {
	err := tx.WithContext(ctx).Delete(&orchestrator.WorkflowConfig{}, id).Error
	if err != nil {
		// 记录删除失败日志
		logger.LogBusinessError(err, "", id, "", "workflow_config_delete_with_tx", "DELETE", map[string]interface{}{
			"operation": "delete_workflow_config_with_transaction",
			"config_id": id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// UpdateWorkflowConfigFields 使用map更新特定字段
// @param ctx 上下文
// @param id 工作流配置ID
// @param fields 要更新的字段
// @return 错误信息
func (r *WorkflowConfigRepository) UpdateWorkflowConfigFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	err := r.db.WithContext(ctx).Model(&orchestrator.WorkflowConfig{}).Where("id = ?", id).Updates(fields).Error
	if err != nil {
		logger.LogBusinessError(err, "", id, "", "workflow_config_update", "PUT", map[string]interface{}{
			"operation": "update_workflow_config_fields",
			"config_id": id,
			"fields":    fields,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}
