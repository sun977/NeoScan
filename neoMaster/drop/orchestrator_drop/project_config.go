/*
 * 项目配置仓库层：项目配置数据访问
 * @author: Sun977
 * @date: 2025.10.11
 * @description: 单纯数据访问，不应该包含业务逻辑
 * @func:
 * 1.创建项目配置
 * 2.更新项目配置
 * 3.删除项目配置
 * 4.项目配置状态变更等
 */

//  基础CRUD操作:
//  	CreateProjectConfig - 创建项目配置
//  	GetProjectConfigByID - 根据ID获取项目配置
//  	GetProjectConfigByName - 根据项目名获取项目配置
//  	UpdateProjectConfig - 更新项目配置信息
//  	DeleteProjectConfig - 软删除项目配置
//  高级查询功能:
//  	GetProjectConfigList - 分页获取项目配置列表
//  	GetProjectConfigWithWorkflows - 获取项目配置及其工作流
//  	ProjectConfigExists - 检查项目配置是否存在
//  	GetActiveProjectConfigs - 获取活跃的项目配置
//  状态管理:
//  	UpdateProjectConfigStatus - 更新项目配置状态
//  	EnableProjectConfig - 启用项目配置
//  	DisableProjectConfig - 禁用项目配置
//  事务支持:
//  	BeginTx - 开始事务
//  	UpdateProjectConfigWithTx - 事务更新项目配置
//  	DeleteProjectConfigWithTx - 事务删除项目配置
//  字段更新:
//  	UpdateProjectConfigFields - 使用map更新特定字段

package orchestrator_drop

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/model/orchestrator_model_drop"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// ProjectConfigRepository 项目配置仓库结构体
// 负责处理项目配置相关的数据访问，不包含业务逻辑
type ProjectConfigRepository struct {
	db *gorm.DB // 数据库连接
}

// NewProjectConfigRepository 创建项目配置仓库实例
// 注入数据库连接，专注于数据访问操作
func NewProjectConfigRepository(db *gorm.DB) *ProjectConfigRepository {
	return &ProjectConfigRepository{
		db: db,
	}
}

// CreateProjectConfig 创建项目配置
// @param ctx 上下文
// @param config 项目配置对象
// @return 错误信息
func (r *ProjectConfigRepository) CreateProjectConfig(ctx context.Context, config *orchestrator_model_drop.ProjectConfig) error {
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Create(config).Error
	if err != nil {
		// 记录创建失败日志
		logger.LogError(err, "", 0, "", "project_config_create", "POST", map[string]interface{}{
			"operation":    "create_project_config",
			"project_name": config.Name,
			"timestamp":    logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// GetProjectConfigByID 根据ID获取项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @return 项目配置对象和错误信息
func (r *ProjectConfigRepository) GetProjectConfigByID(ctx context.Context, id uint) (*orchestrator_model_drop.ProjectConfig, error) {
	var config orchestrator_model_drop.ProjectConfig
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("project config not found"), "", 0, "", "project_config_get", "GET", map[string]interface{}{
				"operation": "get_project_config_by_id",
				"id":        id,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "project_config_get", "GET", map[string]interface{}{
			"operation": "get_project_config_by_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &config, nil
}

// GetProjectConfigByName 根据项目名获取项目配置
// @param ctx 上下文
// @param name 项目名称
// @return 项目配置对象和错误信息
func (r *ProjectConfigRepository) GetProjectConfigByName(ctx context.Context, name string) (*orchestrator_model_drop.ProjectConfig, error) {
	var config orchestrator_model_drop.ProjectConfig
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("project config not found"), "", 0, "", "project_config_get", "GET", map[string]interface{}{
				"operation":    "get_project_config_by_name",
				"project_name": name,
				"timestamp":    logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "project_config_get", "GET", map[string]interface{}{
			"operation":    "get_project_config_by_name",
			"project_name": name,
			"timestamp":    logger.NowFormatted(),
		})
		return nil, err
	}
	return &config, nil
}

// UpdateProjectConfig 更新项目配置
// @param ctx 上下文
// @param config 项目配置对象
// @return 错误信息
func (r *ProjectConfigRepository) UpdateProjectConfig(ctx context.Context, config *orchestrator_model_drop.ProjectConfig) error {
	config.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(config).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(config.ID), "", "project_config_update", "PUT", map[string]interface{}{
			"operation":    "update_project_config",
			"project_name": config.Name,
			"config_id":    config.ID,
			"timestamp":    logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// DeleteProjectConfig 软删除项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @return 错误信息
func (r *ProjectConfigRepository) DeleteProjectConfig(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Delete(&orchestrator_model_drop.ProjectConfig{}, id).Error
	if err != nil {
		// 记录删除失败日志
		logger.LogError(err, "", id, "", "project_config_delete", "DELETE", map[string]interface{}{
			"operation": "delete_project_config",
			"config_id": id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// GetProjectConfigList 分页获取项目配置列表
// @param ctx 上下文
// @param offset 偏移量
// @param limit 限制数量
// @param status 状态过滤（可选）
// @return 项目配置列表、总数和错误信息
func (r *ProjectConfigRepository) GetProjectConfigList(ctx context.Context, offset, limit int, status *orchestrator_model_drop.ProjectConfigStatus) ([]*orchestrator_model_drop.ProjectConfig, int64, error) {
	var configs []*orchestrator_model_drop.ProjectConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&orchestrator_model_drop.ProjectConfig{})

	// 状态过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "project_config_list", "GET", map[string]interface{}{
			"operation": "get_project_config_list_count",
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, err
	}

	// 获取分页数据
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&configs).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "project_config_list", "GET", map[string]interface{}{
			"operation": "get_project_config_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, err
	}

	return configs, total, nil
}

// GetProjectConfigWithWorkflows 获取项目配置及其工作流
// @param ctx 上下文
// @param id 项目配置ID
// @return 项目配置对象和错误信息
func (r *ProjectConfigRepository) GetProjectConfigWithWorkflows(ctx context.Context, id uint) (*orchestrator_model_drop.ProjectConfig, error) {
	var config orchestrator_model_drop.ProjectConfig
	err := r.db.WithContext(ctx).Preload("WorkflowConfigs").First(&config, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.LogError(err, "", id, "", "project_config_get", "GET", map[string]interface{}{
			"operation": "get_project_config_with_workflows",
			"config_id": id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &config, nil
}

// ProjectConfigExists 检查项目配置是否存在
// @param ctx 上下文
// @param name 项目名称
// @return 是否存在和错误信息
func (r *ProjectConfigRepository) ProjectConfigExists(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&orchestrator_model_drop.ProjectConfig{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "project_config_exists", "GET", map[string]interface{}{
			"operation":    "project_config_exists",
			"project_name": name,
			"timestamp":    logger.NowFormatted(),
		})
		return false, err
	}
	return count > 0, nil
}

// GetActiveProjectConfigs 获取活跃的项目配置
// @param ctx 上下文
// @return 项目配置列表和错误信息
func (r *ProjectConfigRepository) GetActiveProjectConfigs(ctx context.Context) ([]*orchestrator_model_drop.ProjectConfig, error) {
	var configs []*orchestrator_model_drop.ProjectConfig
	err := r.db.WithContext(ctx).Where("status = ?", orchestrator_model_drop.ProjectConfigStatusActive).Find(&configs).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "project_config_list", "GET", map[string]interface{}{
			"operation": "get_active_project_configs",
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return configs, nil
}

// UpdateProjectConfigStatus 更新项目配置状态
// @param ctx 上下文
// @param id 项目配置ID
// @param status 新状态
// @return 错误信息
func (r *ProjectConfigRepository) UpdateProjectConfigStatus(ctx context.Context, id uint, status orchestrator_model_drop.ProjectConfigStatus) error {
	err := r.db.WithContext(ctx).Model(&orchestrator_model_drop.ProjectConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		logger.LogError(err, "", id, "", "project_config_update", "PUT", map[string]interface{}{
			"operation": "update_project_config_status",
			"config_id": id,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// EnableProjectConfig 启用项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @return 错误信息
func (r *ProjectConfigRepository) EnableProjectConfig(ctx context.Context, id uint) error {
	return r.UpdateProjectConfigStatus(ctx, id, orchestrator_model_drop.ProjectConfigStatusActive)
}

// DisableProjectConfig 禁用项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @return 错误信息
func (r *ProjectConfigRepository) DisableProjectConfig(ctx context.Context, id uint) error {
	return r.UpdateProjectConfigStatus(ctx, id, orchestrator_model_drop.ProjectConfigStatusInactive)
}

// BeginTx 开始事务
// @return 事务对象和错误信息
func (r *ProjectConfigRepository) BeginTx() (*gorm.DB, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		logger.LogError(tx.Error, "", 0, "", "project_config_transaction", "BEGIN", map[string]interface{}{
			"operation": "begin_transaction",
			"timestamp": logger.NowFormatted(),
		})
		return nil, tx.Error
	}
	return tx, nil
}

// UpdateProjectConfigWithTx 使用事务更新项目配置
// @param ctx 上下文
// @param tx 事务对象
// @param config 项目配置对象
// @return 错误信息
func (r *ProjectConfigRepository) UpdateProjectConfigWithTx(ctx context.Context, tx *gorm.DB, config *orchestrator_model_drop.ProjectConfig) error {
	config.UpdatedAt = time.Now()
	err := tx.WithContext(ctx).Save(config).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(config.ID), "", "project_config_update_with_tx", "PUT", map[string]interface{}{
			"operation":    "update_project_config_with_transaction",
			"project_name": config.Name,
			"config_id":    config.ID,
			"timestamp":    logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// DeleteProjectConfigWithTx 使用事务删除项目配置
// @param ctx 上下文
// @param tx 事务对象
// @param id 项目配置ID
// @return 错误信息
func (r *ProjectConfigRepository) DeleteProjectConfigWithTx(ctx context.Context, tx *gorm.DB, id uint) error {
	err := tx.WithContext(ctx).Delete(&orchestrator_model_drop.ProjectConfig{}, id).Error
	if err != nil {
		// 记录删除失败日志
		logger.LogError(err, "", id, "", "project_config_delete_with_tx", "DELETE", map[string]interface{}{
			"operation": "delete_project_config_with_transaction",
			"config_id": id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// UpdateProjectConfigFields 使用map更新特定字段
// @param ctx 上下文
// @param id 项目配置ID
// @param fields 要更新的字段
// @return 错误信息
func (r *ProjectConfigRepository) UpdateProjectConfigFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	err := r.db.WithContext(ctx).Model(&orchestrator_model_drop.ProjectConfig{}).Where("id = ?", id).Updates(fields).Error
	if err != nil {
		logger.LogError(err, "", id, "", "project_config_update", "PUT", map[string]interface{}{
			"operation": "update_project_config_fields",
			"config_id": id,
			"fields":    fields,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}
