/*
 * 项目配置服务层：项目配置业务逻辑
 * @author: Sun977
 * @date: 2025.10.11
 * @description: 处理项目配置相关的业务逻辑，不直接操作数据库
 * @func:
 * 1.项目配置创建和验证
 * 2.项目配置更新和状态管理
 * 3.项目配置查询和列表
 * 4.项目配置删除和恢复
 * 5.配置热重载和同步
 */

//  核心业务功能:
//  	CreateProjectConfig - 创建项目配置（含验证）
//  	UpdateProjectConfig - 更新项目配置
//  	GetProjectConfig - 获取项目配置详情
//  	ListProjectConfigs - 分页获取项目配置列表
//  	DeleteProjectConfig - 删除项目配置
//  配置管理功能:
//  	ValidateProjectConfig - 验证项目配置
//  	EnableProjectConfig - 启用项目配置
//  	DisableProjectConfig - 禁用项目配置
//  	ReloadProjectConfig - 重新加载项目配置
//  	SyncProjectConfig - 同步项目配置到Agent
//  高级查询功能:
//  	GetActiveProjectConfigs - 获取活跃的项目配置
//  	GetProjectConfigsByUser - 根据用户获取项目配置
//  	SearchProjectConfigs - 搜索项目配置
//  	GetProjectConfigWithWorkflows - 获取项目配置及其工作流
//  统计分析功能:
//  	GetProjectConfigStats - 获取项目配置统计信息
//  	GetProjectConfigUsage - 获取项目配置使用情况

package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	orchestratorRepo "neomaster/internal/repo/mysql/orchestrator"
	"strings"
	"time"

	"neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
)

// ProjectConfigService 项目配置服务结构体
// 负责处理项目配置相关的业务逻辑，协调各个Repository
type ProjectConfigService struct {
	projectRepo  *orchestratorRepo.ProjectConfigRepository  // 项目配置仓库
	workflowRepo *orchestratorRepo.WorkflowConfigRepository // 工作流配置仓库
	scanToolRepo *orchestratorRepo.ScanToolRepository       // 扫描工具仓库
}

// NewProjectConfigService 创建项目配置服务实例
// 注入必要的Repository依赖，遵循依赖注入原则
func NewProjectConfigService(
	projectRepo *orchestratorRepo.ProjectConfigRepository,
	workflowRepo *orchestratorRepo.WorkflowConfigRepository,
	scanToolRepo *orchestratorRepo.ScanToolRepository,
) *ProjectConfigService {
	return &ProjectConfigService{
		projectRepo:  projectRepo,
		workflowRepo: workflowRepo,
		scanToolRepo: scanToolRepo,
	}
}

// CreateProjectConfig 创建项目配置
// @param ctx 上下文
// @param config 项目配置对象
// @return 创建的项目配置和错误信息
func (s *ProjectConfigService) CreateProjectConfig(ctx context.Context, config *orchestrator.ProjectConfig) (*orchestrator.ProjectConfig, error) {
	// 参数验证 - Linus式：消除特殊情况
	if config == nil {
		logger.LogError(errors.New("project config is nil"), "", 0, "", "create_project_config", "SERVICE", map[string]interface{}{
			"operation": "create_project_config",
			"error":     "nil_config",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("项目配置不能为空")
	}

	// 业务验证 - 检查项目名称唯一性
	if err := s.ValidateProjectConfig(ctx, config); err != nil {
		logger.LogError(err, "", 0, "", "create_project_config", "SERVICE", map[string]interface{}{
			"operation":    "create_project_config",
			"error":        "validation_failed",
			"project_name": config.Name,
			"timestamp":    logger.NowFormatted(),
		})
		return nil, fmt.Errorf("项目配置验证失败: %w", err)
	}

	// 检查项目名称是否已存在
	exists, err := s.projectRepo.ProjectConfigExists(ctx, config.Name)
	if err != nil {
		logger.LogError(err, "", 0, "", "create_project_config", "SERVICE", map[string]interface{}{
			"operation":    "create_project_config",
			"error":        "check_exists_failed",
			"project_name": config.Name,
			"timestamp":    logger.NowFormatted(),
		})
		return nil, fmt.Errorf("检查项目名称是否存在失败: %w", err)
	}

	if exists {
		logger.LogError(errors.New("project name already exists"), "", 0, "", "create_project_config", "SERVICE", map[string]interface{}{
			"operation":    "create_project_config",
			"error":        "name_already_exists",
			"project_name": config.Name,
			"timestamp":    logger.NowFormatted(),
		})
		return nil, errors.New("项目名称已存在")
	}

	// 设置默认值 - 简化数据结构
	s.setDefaultValues(config)

	// 创建项目配置
	if err := s.projectRepo.CreateProjectConfig(ctx, config); err != nil {
		logger.LogError(err, "", 0, "", "create_project_config", "SERVICE", map[string]interface{}{
			"operation":    "create_project_config",
			"error":        "create_failed",
			"project_name": config.Name,
			"timestamp":    logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建项目配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("create_project_config success", map[string]interface{}{
		"operation":    "create_project_config",
		"project_name": config.Name,
		"project_id":   config.ID,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return config, nil
}

// UpdateProjectConfig 更新项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @param config 更新的项目配置对象
// @return 更新后的项目配置和错误信息
func (s *ProjectConfigService) UpdateProjectConfig(ctx context.Context, id uint, config *orchestrator.ProjectConfig) (*orchestrator.ProjectConfig, error) {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid project config ID"), "", 0, "", "update_project_config", "SERVICE", map[string]interface{}{
			"operation": "update_project_config",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("项目配置ID不能为0")
	}

	if config == nil {
		logger.LogError(errors.New("project config is nil"), "", id, "", "update_project_config", "SERVICE", map[string]interface{}{
			"operation": "update_project_config",
			"error":     "nil_config",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("项目配置不能为空")
	}

	// 检查项目配置是否存在
	existingConfig, err := s.projectRepo.GetProjectConfigByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "update_project_config", "SERVICE", map[string]interface{}{
			"operation": "update_project_config",
			"error":     "get_existing_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取现有项目配置失败: %w", err)
	}

	if existingConfig == nil {
		logger.LogError(errors.New("project config not found"), "", id, "", "update_project_config", "SERVICE", map[string]interface{}{
			"operation": "update_project_config",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("项目配置不存在")
	}

	// 如果名称发生变化，检查新名称是否已存在
	if config.Name != existingConfig.Name {
		exists, err := s.projectRepo.ProjectConfigExists(ctx, config.Name)
		if err != nil {
			logger.LogError(err, "", id, "", "update_project_config", "SERVICE", map[string]interface{}{
				"operation":    "update_project_config",
				"error":        "check_name_exists_failed",
				"id":           id,
				"project_name": config.Name,
				"timestamp":    logger.NowFormatted(),
			})
			return nil, fmt.Errorf("检查项目名称是否存在失败: %w", err)
		}

		if exists {
			logger.LogError(errors.New("project name already exists"), "", id, "", "update_project_config", "SERVICE", map[string]interface{}{
				"operation":    "update_project_config",
				"error":        "name_already_exists",
				"id":           id,
				"project_name": config.Name,
				"timestamp":    logger.NowFormatted(),
			})
			return nil, errors.New("项目名称已存在")
		}
	}

	// 业务验证
	if err := s.ValidateProjectConfig(ctx, config); err != nil {
		logger.LogError(err, "", id, "", "update_project_config", "SERVICE", map[string]interface{}{
			"operation":    "update_project_config",
			"error":        "validation_failed",
			"id":           id,
			"project_name": config.Name,
			"timestamp":    logger.NowFormatted(),
		})
		return nil, fmt.Errorf("项目配置验证失败: %w", err)
	}

	// 保持ID和创建时间不变
	config.ID = uint64(id)
	config.CreatedAt = existingConfig.CreatedAt

	// 更新项目配置
	if err := s.projectRepo.UpdateProjectConfig(ctx, config); err != nil {
		logger.LogError(err, "", id, "", "update_project_config", "SERVICE", map[string]interface{}{
			"operation":    "update_project_config",
			"error":        "update_failed",
			"id":           id,
			"project_name": config.Name,
			"timestamp":    logger.NowFormatted(),
		})
		return nil, fmt.Errorf("更新项目配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("update_project_config success", map[string]interface{}{
		"operation":    "update_project_config",
		"project_name": config.Name,
		"project_id":   id,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return config, nil
}

// GetProjectConfig 获取项目配置详情
// @param ctx 上下文
// @param id 项目配置ID
// @return 项目配置对象和错误信息
func (s *ProjectConfigService) GetProjectConfig(ctx context.Context, id uint) (*orchestrator.ProjectConfig, error) {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid project config ID"), "", 0, "", "get_project_config", "SERVICE", map[string]interface{}{
			"operation": "get_project_config",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("项目配置ID不能为0")
	}

	// 获取项目配置
	config, err := s.projectRepo.GetProjectConfigByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "get_project_config", "SERVICE", map[string]interface{}{
			"operation": "get_project_config",
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取项目配置失败: %w", err)
	}

	if config == nil {
		logger.LogError(errors.New("project config not found"), "", id, "", "get_project_config", "SERVICE", map[string]interface{}{
			"operation": "get_project_config",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("项目配置不存在")
	}

	return config, nil
}

// ListProjectConfigs 分页获取项目配置列表
// @param ctx 上下文
// @param offset 偏移量
// @param limit 限制数量
// @param status 状态过滤（可选）
// @param userID 用户ID过滤（可选）
// @return 项目配置列表、总数和错误信息
func (s *ProjectConfigService) ListProjectConfigs(ctx context.Context, offset, limit int, status *orchestrator.ProjectConfigStatus, userID *uint) ([]*orchestrator.ProjectConfig, int64, error) {
	// 参数验证
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认分页大小
	}

	// 获取项目配置列表
	configs, total, err := s.projectRepo.GetProjectConfigList(ctx, offset, limit, status)
	if err != nil {
		logger.LogError(err, "", 0, "", "list_project_configs", "SERVICE", map[string]interface{}{
			"operation": "list_project_configs",
			"error":     "list_failed",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, fmt.Errorf("获取项目配置列表失败: %w", err)
	}

	return configs, total, nil
}

// DeleteProjectConfig 删除项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @return 错误信息
func (s *ProjectConfigService) DeleteProjectConfig(ctx context.Context, id uint) error {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid project config ID"), "", 0, "", "delete_project_config", "SERVICE", map[string]interface{}{
			"operation": "delete_project_config",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("项目配置ID不能为0")
	}

	// 检查项目配置是否存在
	config, err := s.projectRepo.GetProjectConfigByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "delete_project_config", "SERVICE", map[string]interface{}{
			"operation": "delete_project_config",
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取项目配置失败: %w", err)
	}

	if config == nil {
		logger.LogError(errors.New("project config not found"), "", id, "", "delete_project_config", "SERVICE", map[string]interface{}{
			"operation": "delete_project_config",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("项目配置不存在")
	}

	// 检查是否有关联的工作流配置
	workflows, err := s.workflowRepo.GetWorkflowConfigsByProject(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "delete_project_config", "SERVICE", map[string]interface{}{
			"operation": "delete_project_config",
			"error":     "check_workflows_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("检查关联工作流失败: %w", err)
	}

	if len(workflows) > 0 {
		logger.LogError(errors.New("project has associated workflows"), "", id, "", "delete_project_config", "SERVICE", map[string]interface{}{
			"operation":      "delete_project_config",
			"error":          "has_workflows",
			"id":             id,
			"workflow_count": len(workflows),
			"timestamp":      logger.NowFormatted(),
		})
		return errors.New("项目配置存在关联的工作流，无法删除")
	}

	// 删除项目配置
	if err := s.projectRepo.DeleteProjectConfig(ctx, id); err != nil {
		logger.LogError(err, "", id, "", "delete_project_config", "SERVICE", map[string]interface{}{
			"operation": "delete_project_config",
			"error":     "delete_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("删除项目配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("delete_project_config success", map[string]interface{}{
		"operation":    "delete_project_config",
		"project_name": config.Name,
		"project_id":   id,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return nil
}

// ValidateProjectConfig 验证项目配置
// @param ctx 上下文
// @param config 项目配置对象
// @return 错误信息
func (s *ProjectConfigService) ValidateProjectConfig(ctx context.Context, config *orchestrator.ProjectConfig) error {
	// 基础字段验证
	if strings.TrimSpace(config.Name) == "" {
		return errors.New("项目名称不能为空")
	}

	if len(config.Name) > 100 {
		return errors.New("项目名称长度不能超过100个字符")
	}

	if len(config.Description) > 500 {
		return errors.New("项目描述长度不能超过500个字符")
	}

	// 元数据验证
	if config.Metadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(config.Metadata), &metadata); err != nil {
			return fmt.Errorf("元数据JSON格式无效: %w", err)
		}
	}

	return nil
}

// EnableProjectConfig 启用项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @return 错误信息
func (s *ProjectConfigService) EnableProjectConfig(ctx context.Context, id uint) error {
	return s.updateProjectConfigStatus(ctx, id, orchestrator.ProjectConfigStatusActive, "enable_project_config")
}

// DisableProjectConfig 禁用项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @return 错误信息
func (s *ProjectConfigService) DisableProjectConfig(ctx context.Context, id uint) error {
	return s.updateProjectConfigStatus(ctx, id, orchestrator.ProjectConfigStatusInactive, "disable_project_config")
}

// ReloadProjectConfig 重新加载项目配置
// @param ctx 上下文
// @param id 项目配置ID
// @return 错误信息
func (s *ProjectConfigService) ReloadProjectConfig(ctx context.Context, id uint) error {
	// 获取项目配置
	config, err := s.GetProjectConfig(ctx, id)
	if err != nil {
		return fmt.Errorf("获取项目配置失败: %w", err)
	}

	// 重新验证配置
	if err := s.ValidateProjectConfig(ctx, config); err != nil {
		logger.LogError(err, "", id, "", "reload_project_config", "SERVICE", map[string]interface{}{
			"operation": "reload_project_config",
			"error":     "validation_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("重新加载项目配置验证失败: %w", err)
	}

	// 更新最后修改时间
	if err := s.projectRepo.UpdateProjectConfigFields(ctx, id, map[string]interface{}{
		"updated_at": time.Now(),
	}); err != nil {
		logger.LogError(err, "", id, "", "reload_project_config", "SERVICE", map[string]interface{}{
			"operation": "reload_project_config",
			"error":     "update_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("更新项目配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("reload_project_config success", map[string]interface{}{
		"operation":    "reload_project_config",
		"project_name": config.Name,
		"project_id":   id,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return nil
}

// SyncProjectConfig 同步项目配置到Agent
// @param ctx 上下文
// @param id 项目配置ID
// @return 错误信息
func (s *ProjectConfigService) SyncProjectConfig(ctx context.Context, id uint) error {
	// 获取项目配置
	config, err := s.GetProjectConfig(ctx, id)
	if err != nil {
		return fmt.Errorf("获取项目配置失败: %w", err)
	}

	// TODO: 实现同步到Agent的逻辑
	// 这里应该调用Agent管理服务，将配置推送到相关的Agent节点

	// 记录成功日志
	logger.Info("sync_project_config success", map[string]interface{}{
		"operation":    "sync_project_config",
		"project_name": config.Name,
		"project_id":   id,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return nil
}

// GetActiveProjectConfigs 获取活跃的项目配置
// @param ctx 上下文
// @return 项目配置列表和错误信息
func (s *ProjectConfigService) GetActiveProjectConfigs(ctx context.Context) ([]*orchestrator.ProjectConfig, error) {
	configs, err := s.projectRepo.GetActiveProjectConfigs(ctx)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_active_project_configs", "SERVICE", map[string]interface{}{
			"operation": "get_active_project_configs",
			"error":     "get_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取活跃项目配置失败: %w", err)
	}

	return configs, nil
}

// GetProjectConfigsByUser 根据用户获取项目配置
// @param ctx 上下文
// @param userID 用户ID
// @return 项目配置列表和错误信息
func (s *ProjectConfigService) GetProjectConfigsByUser(ctx context.Context, userID uint) ([]*orchestrator.ProjectConfig, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为0")
	}

	// 使用现有的ListProjectConfigs方法，传入userID过滤
	configs, _, err := s.ListProjectConfigs(ctx, 0, 100, nil, &userID)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_project_configs_by_user", "SERVICE", map[string]interface{}{
			"operation": "get_project_configs_by_user",
			"error":     "get_failed",
			"user_id":   userID,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("根据用户获取项目配置失败: %w", err)
	}

	return configs, nil
}

// SearchProjectConfigs 搜索项目配置
// @param ctx 上下文
// @param keyword 搜索关键词
// @param offset 偏移量
// @param limit 限制数量
// @return 项目配置列表、总数和错误信息
func (s *ProjectConfigService) SearchProjectConfigs(ctx context.Context, keyword string, offset, limit int) ([]*orchestrator.ProjectConfig, int64, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, 0, errors.New("搜索关键词不能为空")
	}

	// 参数验证
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// TODO: 实现搜索逻辑
	// 这里应该调用Repository的搜索方法，支持按名称、描述等字段搜索

	return nil, 0, errors.New("搜索功能暂未实现")
}

// GetProjectConfigWithWorkflows 获取项目配置及其工作流
// @param ctx 上下文
// @param id 项目配置ID
// @return 项目配置对象和错误信息
func (s *ProjectConfigService) GetProjectConfigWithWorkflows(ctx context.Context, id uint) (*orchestrator.ProjectConfig, error) {
	config, err := s.projectRepo.GetProjectConfigWithWorkflows(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "get_project_config_with_workflows", "SERVICE", map[string]interface{}{
			"operation": "get_project_config_with_workflows",
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取项目配置及工作流失败: %w", err)
	}

	if config == nil {
		return nil, errors.New("项目配置不存在")
	}

	return config, nil
}

// GetProjectConfigStats 获取项目配置统计信息
// @param ctx 上下文
// @return 统计信息和错误信息
func (s *ProjectConfigService) GetProjectConfigStats(ctx context.Context) (map[string]interface{}, error) {
	// TODO: 实现统计逻辑
	// 统计总数、活跃数、各状态数量等

	stats := map[string]interface{}{
		"total_count":    0,
		"active_count":   0,
		"inactive_count": 0,
		"draft_count":    0,
	}

	return stats, nil
}

// GetProjectConfigUsage 获取项目配置使用情况
// @param ctx 上下文
// @param id 项目配置ID
// @return 使用情况和错误信息
func (s *ProjectConfigService) GetProjectConfigUsage(ctx context.Context, id uint) (map[string]interface{}, error) {
	// TODO: 实现使用情况统计
	// 统计执行次数、成功次数、失败次数等

	usage := map[string]interface{}{
		"execution_count": 0,
		"success_count":   0,
		"failure_count":   0,
		"last_execution":  nil,
	}

	return usage, nil
}

// 私有方法：更新项目配置状态
func (s *ProjectConfigService) updateProjectConfigStatus(ctx context.Context, id uint, status orchestrator.ProjectConfigStatus, operation string) error {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid project config ID"), "", 0, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("项目配置ID不能为0")
	}

	// 检查项目配置是否存在
	config, err := s.projectRepo.GetProjectConfigByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取项目配置失败: %w", err)
	}

	if config == nil {
		logger.LogError(errors.New("project config not found"), "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("项目配置不存在")
	}

	// 更新状态
	if err := s.projectRepo.UpdateProjectConfigStatus(ctx, id, status); err != nil {
		logger.LogError(err, "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "update_status_failed",
			"id":        id,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("更新项目配置状态失败: %w", err)
	}

	// 记录成功日志
	logger.Info(operation+" success", map[string]interface{}{
		"operation":    operation,
		"project_name": config.Name,
		"project_id":   id,
		"status":       status,
		"result":       "success",
		"timestamp":    logger.NowFormatted(),
	})

	return nil
}

// 私有方法：设置默认值
func (s *ProjectConfigService) setDefaultValues(config *orchestrator.ProjectConfig) {
	// 设置默认状态
	if config.Status == 0 {
		config.Status = orchestrator.ProjectConfigStatusInactive
	}

	if config.Metadata == "" {
		config.Metadata = "{}"
	}

	// 设置时间戳
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
}

// GetSystemScanConfig 获取系统扫描配置
// @param ctx 上下文
// @return 系统配置数据和错误信息
func (s *ProjectConfigService) GetSystemScanConfig(ctx context.Context) (map[string]interface{}, error) {
	// 记录开始日志
	logger.Info("get_system_scan_config start", map[string]interface{}{
		"operation": "get_system_scan_config",
		"timestamp": logger.NowFormatted(),
	})

	// 这里可以从配置文件或数据库获取系统配置
	// 暂时返回模拟数据
	systemConfig := map[string]interface{}{
		"scan_timeout":   3600,
		"max_concurrent": 10,
		"default_tools":  []string{"nmap", "masscan"},
		"output_format":  "json",
		"enable_logging": true,
		"log_level":      "info",
		"scan_frequency": "daily",
		"notification":   true,
		"auto_update":    false,
		"security_level": "medium",
	}

	// 记录成功日志
	logger.Info("get_system_scan_config success", map[string]interface{}{
		"operation": "get_system_scan_config",
		"result":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return systemConfig, nil
}

// UpdateSystemScanConfig 更新系统扫描配置
// @param ctx 上下文
// @param configData 配置数据
// @return 错误信息
func (s *ProjectConfigService) UpdateSystemScanConfig(ctx context.Context, configData map[string]interface{}) error {
	// 参数验证
	if len(configData) == 0 {
		logger.LogError(errors.New("config data is empty"), "", 0, "", "update_system_scan_config", "SERVICE", map[string]interface{}{
			"operation": "update_system_scan_config",
			"error":     "empty_config_data",
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("配置数据不能为空")
	}

	// 记录开始日志
	logger.Info("update_system_scan_config start", map[string]interface{}{
		"operation": "update_system_scan_config",
		"config":    configData,
		"timestamp": logger.NowFormatted(),
	})

	// 验证配置数据格式
	validKeys := map[string]bool{
		"scan_timeout":   true,
		"max_concurrent": true,
		"default_tools":  true,
		"output_format":  true,
		"enable_logging": true,
		"log_level":      true,
		"scan_frequency": true,
		"notification":   true,
		"auto_update":    true,
		"security_level": true,
	}

	for key := range configData {
		if !validKeys[key] {
			logger.LogError(fmt.Errorf("invalid config key: %s", key), "", 0, "", "update_system_scan_config", "SERVICE", map[string]interface{}{
				"operation":   "update_system_scan_config",
				"error":       "invalid_config_key",
				"invalid_key": key,
				"timestamp":   logger.NowFormatted(),
			})
			return fmt.Errorf("无效的配置项: %s", key)
		}
	}

	// 这里可以将配置保存到数据库或配置文件
	// 暂时只记录日志表示更新成功

	// 记录成功日志
	logger.Info("update_system_scan_config success", map[string]interface{}{
		"operation": "update_system_scan_config",
		"config":    configData,
		"result":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return nil
}
