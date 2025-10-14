/*
 * 扫描工具服务层：扫描工具管理业务逻辑
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 处理扫描工具管理和配置相关的业务逻辑
 * @func:
 * 1.扫描工具配置管理
 * 2.扫描工具状态监控
 * 3.扫描工具执行统计
 * 4.扫描工具兼容性检查
 */

//  核心业务功能:
//  	CreateScanTool - 创建扫描工具配置
//  	UpdateScanTool - 更新扫描工具配置
//  	GetScanTool - 获取扫描工具详情
//  	ListScanTools - 分页获取扫描工具列表
//  	DeleteScanTool - 删除扫描工具配置
//  状态管理功能:
//  	EnableScanTool - 启用扫描工具
//  	DisableScanTool - 禁用扫描工具
//  	CheckScanToolHealth - 检查扫描工具健康状态
//  	ValidateScanToolConfig - 验证扫描工具配置
//  执行统计功能:
//  	UpdateScanToolUsage - 更新扫描工具使用统计
//  	GetScanToolStats - 获取扫描工具统计信息
//  	GetScanToolPerformance - 获取扫描工具性能指标
//  工具管理功能:
//  	GetAvailableScanTools - 获取可用扫描工具
//  	GetScanToolsByType - 根据类型获取扫描工具
//  	InstallScanTool - 安装扫描工具
//  	UninstallScanTool - 卸载扫描工具

package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	scanConfigRepo "neomaster/internal/repository/orchestrator"
)

// ScanToolService 扫描工具服务结构体
// 负责处理扫描工具相关的业务逻辑
type ScanToolService struct {
	scanToolRepo *scanConfigRepo.ScanToolRepository // 扫描工具仓库
}

// NewScanToolService 创建扫描工具服务实例
// 注入必要的Repository依赖，遵循依赖注入原则
func NewScanToolService(scanToolRepo *scanConfigRepo.ScanToolRepository) *ScanToolService {
	return &ScanToolService{
		scanToolRepo: scanToolRepo,
	}
}

// CreateScanTool 创建扫描工具配置
// @param ctx 上下文
// @param tool 扫描工具配置对象
// @return 创建的扫描工具配置和错误信息
func (s *ScanToolService) CreateScanTool(ctx context.Context, tool *orchestrator.ScanTool) (*orchestrator.ScanTool, error) {
	// 参数验证 - Linus式：消除特殊情况
	if tool == nil {
		logger.LogError(errors.New("scan tool is nil"), "", 0, "", "create_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "create_scan_tool",
			"error":     "nil_tool",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描工具配置不能为空")
	}

	// 业务验证 - 检查工具名称唯一性
	if err := s.ValidateScanToolConfig(ctx, tool); err != nil {
		logger.LogError(err, "", 0, "", "create_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "create_scan_tool",
			"error":     "validation_failed",
			"tool_name": tool.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("扫描工具配置验证失败: %w", err)
	}

	// 检查工具名称是否已存在
	exists, err := s.scanToolRepo.ScanToolExists(ctx, tool.Name)
	if err != nil {
		logger.LogError(err, "", 0, "", "create_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "create_scan_tool",
			"error":     "check_exists_failed",
			"tool_name": tool.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("检查扫描工具名称是否存在失败: %w", err)
	}

	if exists {
		logger.LogError(errors.New("scan tool name already exists"), "", 0, "", "create_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "create_scan_tool",
			"error":     "name_already_exists",
			"tool_name": tool.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描工具名称已存在")
	}

	// 设置默认值 - 简化数据结构
	s.setDefaultValues(tool)

	// 创建扫描工具配置
	if err := s.scanToolRepo.CreateScanTool(ctx, tool); err != nil {
		logger.LogError(err, "", 0, "", "create_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "create_scan_tool",
			"error":     "create_failed",
			"tool_name": tool.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("创建扫描工具配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("create_scan_tool success", map[string]interface{}{
		"operation": "create_scan_tool",
		"tool_name": tool.Name,
		"tool_id":   tool.ID,
		"result":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return tool, nil
}

// UpdateScanTool 更新扫描工具配置
// @param ctx 上下文
// @param id 扫描工具配置ID
// @param tool 更新的扫描工具配置对象
// @return 更新后的扫描工具配置和错误信息
func (s *ScanToolService) UpdateScanTool(ctx context.Context, id uint, tool *orchestrator.ScanTool) (*orchestrator.ScanTool, error) {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan tool ID"), "", 0, "", "update_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "update_scan_tool",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描工具配置ID不能为0")
	}

	if tool == nil {
		logger.LogError(errors.New("scan tool is nil"), "", id, "", "update_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "update_scan_tool",
			"error":     "nil_tool",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描工具配置不能为空")
	}

	// 检查扫描工具配置是否存在
	existingTool, err := s.scanToolRepo.GetScanToolByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "update_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "update_scan_tool",
			"error":     "get_existing_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取现有扫描工具配置失败: %w", err)
	}

	if existingTool == nil {
		logger.LogError(errors.New("scan tool not found"), "", id, "", "update_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "update_scan_tool",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描工具配置不存在")
	}

	// 如果名称发生变化，检查新名称是否已存在
	if tool.Name != existingTool.Name {
		exists, err := s.scanToolRepo.ScanToolExists(ctx, tool.Name)
		if err != nil {
			logger.LogError(err, "", id, "", "update_scan_tool", "SERVICE", map[string]interface{}{
				"operation": "update_scan_tool",
				"error":     "check_name_exists_failed",
				"id":        id,
				"tool_name": tool.Name,
				"timestamp": logger.NowFormatted(),
			})
			return nil, fmt.Errorf("检查扫描工具名称是否存在失败: %w", err)
		}

		if exists {
			logger.LogError(errors.New("scan tool name already exists"), "", id, "", "update_scan_tool", "SERVICE", map[string]interface{}{
				"operation": "update_scan_tool",
				"error":     "name_already_exists",
				"id":        id,
				"tool_name": tool.Name,
				"timestamp": logger.NowFormatted(),
			})
			return nil, errors.New("扫描工具名称已存在")
		}
	}

	// 业务验证
	if err := s.ValidateScanToolConfig(ctx, tool); err != nil {
		logger.LogError(err, "", id, "", "update_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "update_scan_tool",
			"error":     "validation_failed",
			"id":        id,
			"tool_name": tool.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("扫描工具配置验证失败: %w", err)
	}

	// 保持ID和创建时间不变
	tool.ID = uint64(id)
	tool.CreatedAt = existingTool.CreatedAt

	// 更新扫描工具配置
	if err := s.scanToolRepo.UpdateScanTool(ctx, tool); err != nil {
		logger.LogError(err, "", id, "", "update_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "update_scan_tool",
			"error":     "update_failed",
			"id":        id,
			"tool_name": tool.Name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("更新扫描工具配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("update_scan_tool success", map[string]interface{}{
		"operation": "update_scan_tool",
		"tool_name": tool.Name,
		"tool_id":   id,
		"result":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return tool, nil
}

// GetScanTool 获取扫描工具配置详情
// @param ctx 上下文
// @param id 扫描工具配置ID
// @return 扫描工具配置对象和错误信息
func (s *ScanToolService) GetScanTool(ctx context.Context, id uint) (*orchestrator.ScanTool, error) {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan tool ID"), "", 0, "", "get_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "get_scan_tool",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描工具配置ID不能为0")
	}

	// 获取扫描工具配置
	tool, err := s.scanToolRepo.GetScanToolByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "get_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "get_scan_tool",
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取扫描工具配置失败: %w", err)
	}

	if tool == nil {
		logger.LogError(errors.New("scan tool not found"), "", id, "", "get_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "get_scan_tool",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("扫描工具配置不存在")
	}

	return tool, nil
}

// ListScanTools 分页获取扫描工具配置列表
// @param ctx 上下文
// @param offset 偏移量
// @param limit 限制数量
// @param toolType 工具类型过滤（可选）
// @param status 状态过滤（可选）
// @return 扫描工具配置列表、总数和错误信息
func (s *ScanToolService) ListScanTools(ctx context.Context, offset, limit int, toolType *orchestrator.ScanToolType, status *orchestrator.ScanToolStatus) ([]*orchestrator.ScanTool, int64, error) {
	// 参数验证
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认分页大小
	}

	// 获取扫描工具配置列表
	tools, total, err := s.scanToolRepo.GetScanToolList(ctx, offset, limit, toolType, status)
	if err != nil {
		logger.LogError(err, "", 0, "", "list_scan_tools", "SERVICE", map[string]interface{}{
			"operation": "list_scan_tools",
			"error":     "list_failed",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, fmt.Errorf("获取扫描工具配置列表失败: %w", err)
	}

	return tools, total, nil
}

// DeleteScanTool 删除扫描工具配置
// @param ctx 上下文
// @param id 扫描工具配置ID
// @return 错误信息
func (s *ScanToolService) DeleteScanTool(ctx context.Context, id uint) error {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan tool ID"), "", 0, "", "delete_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_tool",
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描工具配置ID不能为0")
	}

	// 检查扫描工具配置是否存在
	tool, err := s.scanToolRepo.GetScanToolByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "delete_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_tool",
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取扫描工具配置失败: %w", err)
	}

	if tool == nil {
		logger.LogError(errors.New("scan tool not found"), "", id, "", "delete_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_tool",
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描工具配置不存在")
	}

	// 检查工具是否正在使用
	if tool.Status == orchestrator.ScanToolStatusTesting {
		logger.LogError(errors.New("scan tool is testing"), "", id, "", "delete_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_tool",
			"error":     "tool_testing",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描工具正在测试中，无法删除")
	}

	// 删除扫描工具配置
	if err := s.scanToolRepo.DeleteScanTool(ctx, id); err != nil {
		logger.LogError(err, "", id, "", "delete_scan_tool", "SERVICE", map[string]interface{}{
			"operation": "delete_scan_tool",
			"error":     "delete_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("删除扫描工具配置失败: %w", err)
	}

	// 记录成功日志
	logger.Info("delete_scan_tool success", map[string]interface{}{
		"operation": "delete_scan_tool",
		"tool_name": tool.Name,
		"tool_id":   id,
		"result":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return nil
}

// EnableScanTool 启用扫描工具
// @param ctx 上下文
// @param id 扫描工具配置ID
// @return 错误信息
func (s *ScanToolService) EnableScanTool(ctx context.Context, id uint) error {
	return s.updateScanToolStatus(ctx, id, orchestrator.ScanToolStatusEnabled, "enable_scan_tool")
}

// DisableScanTool 禁用扫描工具
// @param ctx 上下文
// @param id 扫描工具配置ID
// @return 错误信息
func (s *ScanToolService) DisableScanTool(ctx context.Context, id uint) error {
	return s.updateScanToolStatus(ctx, id, orchestrator.ScanToolStatusDisabled, "disable_scan_tool")
}

// CheckScanToolHealth 检查扫描工具健康状态
// @param ctx 上下文
// @param id 扫描工具配置ID
// @return 健康状态和错误信息
func (s *ScanToolService) CheckScanToolHealth(ctx context.Context, id uint) (map[string]interface{}, error) {
	// 参数验证
	if id == 0 {
		return nil, errors.New("扫描工具配置ID不能为0")
	}

	// 获取扫描工具配置
	tool, err := s.scanToolRepo.GetScanToolByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", "check_scan_tool_health", "SERVICE", map[string]interface{}{
			"operation": "check_scan_tool_health",
			"error":     "get_tool_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取扫描工具配置失败: %w", err)
	}

	if tool == nil {
		return nil, errors.New("扫描工具配置不存在")
	}

	// TODO: 实现健康检查逻辑
	// 1. 检查工具可执行文件是否存在
	// 2. 检查工具版本是否兼容
	// 3. 检查工具依赖是否满足
	// 4. 执行简单的测试命令

	health := map[string]interface{}{
		"tool_id":     id,
		"tool_name":   tool.Name,
		"status":      "healthy",
		"version":     tool.Version,
		"last_check":  time.Now(),
		"issues":      []string{},
		"suggestions": []string{},
	}

	logger.LogSystemEvent("SERVICE", "check_scan_tool_health", "工具健康检查完成", logger.InfoLevel, map[string]interface{}{
		"operation": "check_scan_tool_health",
		"tool_name": tool.Name,
		"tool_id":   id,
		"health":    "healthy",
		"timestamp": logger.NowFormatted(),
	})

	return health, nil
}

// ValidateScanToolConfig 验证扫描工具配置
// @param ctx 上下文
// @param tool 扫描工具配置对象
// @return 错误信息
func (s *ScanToolService) ValidateScanToolConfig(ctx context.Context, tool *orchestrator.ScanTool) error {
	// 基础字段验证
	if strings.TrimSpace(tool.Name) == "" {
		return errors.New("扫描工具名称不能为空")
	}

	if len(tool.Name) > 100 {
		return errors.New("扫描工具名称长度不能超过100个字符")
	}

	if len(tool.Description) > 500 {
		return errors.New("扫描工具描述长度不能超过500个字符")
	}

	if strings.TrimSpace(tool.ExecutablePath) == "" {
		return errors.New("可执行文件路径不能为空")
	}

	if strings.TrimSpace(tool.Version) == "" {
		return errors.New("版本号不能为空")
	}

	// 默认参数验证
	if tool.DefaultParams != "" {
		var defaultParams map[string]interface{}
		if err := json.Unmarshal([]byte(tool.DefaultParams), &defaultParams); err != nil {
			return fmt.Errorf("默认参数JSON格式无效: %w", err)
		}
	}

	// 参数模式验证
	if tool.ParamSchema != "" {
		var paramSchema map[string]interface{}
		if err := json.Unmarshal([]byte(tool.ParamSchema), &paramSchema); err != nil {
			return fmt.Errorf("参数模式JSON格式无效: %w", err)
		}
	}

	// 结果映射验证
	if tool.ResultMapping != "" {
		var resultMapping map[string]interface{}
		if err := json.Unmarshal([]byte(tool.ResultMapping), &resultMapping); err != nil {
			return fmt.Errorf("结果映射JSON格式无效: %w", err)
		}
	}

	// 依赖项验证 - 简单的字符串验证
	if tool.Dependencies == "" {
		tool.Dependencies = ""
	}

	// 元数据验证
	if tool.Metadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(tool.Metadata), &metadata); err != nil {
			return fmt.Errorf("元数据JSON格式无效: %w", err)
		}
	}

	// 执行限制验证
	if tool.MaxExecutionTime < 0 {
		return errors.New("最大执行时间不能为负数")
	}
	// 验证超时时间
	if tool.MaxExecutionTime < 0 {
		return errors.New("最大执行时间不能为负数")
	}

	if tool.MaxMemoryMB < 0 {
		return errors.New("最大内存限制不能为负数")
	}

	return nil
}

// UpdateScanToolUsage 更新扫描工具使用统计
// @param ctx 上下文
// @param id 扫描工具配置ID
// @param success 是否成功
// @param executionTime 执行时间（秒）
// @return 错误信息
func (s *ScanToolService) UpdateScanToolUsage(ctx context.Context, id uint, success bool, executionTime int) error {
	// 参数验证
	if id == 0 {
		return errors.New("扫描工具配置ID不能为0")
	}

	// 更新使用统计
	if err := s.scanToolRepo.IncrementUsageCount(ctx, id, true); err != nil {
		logger.LogError(err, "", id, "", "update_scan_tool_usage", "SERVICE", map[string]interface{}{
			"operation": "update_scan_tool_usage",
			"error":     "increment_usage_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("更新使用次数失败: %w", err)
	}

	// 更新成功或失败次数
	if success {
		// 更新成功统计 - 使用IncrementUsageCount方法
		if err := s.scanToolRepo.IncrementUsageCount(ctx, id, true); err != nil {
			logger.LogError(err, "", id, "", "update_scan_tool_usage", "SERVICE", map[string]interface{}{
				"operation": "update_scan_tool_usage",
				"error":     "increment_success_failed",
				"id":        id,
				"timestamp": logger.NowFormatted(),
			})
			return fmt.Errorf("更新成功次数失败: %w", err)
		}
	} else {
		// 更新失败统计 - 使用IncrementUsageCount方法
		if err := s.scanToolRepo.IncrementUsageCount(ctx, id, false); err != nil {
			logger.LogError(err, "", id, "", "update_scan_tool_usage", "SERVICE", map[string]interface{}{
				"operation": "update_scan_tool_usage",
				"error":     "increment_failure_failed",
				"id":        id,
				"timestamp": logger.NowFormatted(),
			})
			return fmt.Errorf("更新失败次数失败: %w", err)
		}
	}

	// 记录成功日志
	logger.LogSystemEvent("SERVICE", "update_scan_tool_usage", "更新工具使用统计", logger.InfoLevel, map[string]interface{}{
		"operation":      "update_scan_tool_usage",
		"tool_id":        id,
		"success":        success,
		"execution_time": executionTime,
		"status":         "success",
		"timestamp":      logger.NowFormatted(),
	})

	return nil
}

// GetScanToolStats 获取扫描工具统计信息
// @param ctx 上下文
// @return 统计信息和错误信息
func (s *ScanToolService) GetScanToolStats(ctx context.Context) (map[string]interface{}, error) {
	// TODO: 实现统计逻辑
	// 统计总数、各类型数量、各状态数量等

	stats := map[string]interface{}{
		"total_count":    0,
		"active_count":   0,
		"inactive_count": 0,
		"running_count":  0,
		"by_type": map[string]int{
			"asset_scan":        0,
			"vulnerability":     0,
			"compliance":        0,
			"security_baseline": 0,
			"custom":            0,
		},
	}

	return stats, nil
}

// GetScanToolPerformance 获取扫描工具性能指标
// @param ctx 上下文
// @param id 扫描工具配置ID
// @return 性能指标和错误信息
func (s *ScanToolService) GetScanToolPerformance(ctx context.Context, id uint) (map[string]interface{}, error) {
	if id == 0 {
		return nil, errors.New("扫描工具ID不能为0")
	}

	// 获取扫描工具配置
	tool, err := s.scanToolRepo.GetScanToolByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取扫描工具配置失败: %w", err)
	}

	if tool == nil {
		return nil, errors.New("扫描工具配置不存在")
	}

	// 计算性能指标
	var successRate float64
	if tool.UsageCount > 0 {
		successRate = float64(tool.SuccessCount) / float64(tool.UsageCount) * 100
	}

	var failureRate float64
	if tool.UsageCount > 0 {
		failureRate = float64(tool.FailureCount) / float64(tool.UsageCount) * 100
	}

	performance := map[string]interface{}{
		"tool_id":       id,
		"tool_name":     tool.Name,
		"usage_count":   tool.UsageCount,
		"success_count": tool.SuccessCount,
		"failure_count": tool.FailureCount,
		"success_rate":  successRate,
		"failure_rate":  failureRate,
		"last_used_at":  tool.UpdatedAt,
		"avg_exec_time": 0, // TODO: 计算平均执行时间
	}

	return performance, nil
}

// GetAvailableScanTools 获取可用扫描工具
// @param ctx 上下文
// @return 可用扫描工具列表和错误信息
func (s *ScanToolService) GetAvailableScanTools(ctx context.Context) ([]*orchestrator.ScanTool, error) {
	tools, err := s.scanToolRepo.GetAvailableScanTools(ctx)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_available_scan_tools", "SERVICE", map[string]interface{}{
			"operation": "get_available_scan_tools",
			"error":     "get_failed",
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取可用扫描工具失败: %w", err)
	}

	return tools, nil
}

// GetScanToolsByType 根据类型获取扫描工具
// @param ctx 上下文
// @param toolType 工具类型
// @return 扫描工具列表和错误信息
func (s *ScanToolService) GetScanToolsByType(ctx context.Context, toolType orchestrator.ScanToolType) ([]*orchestrator.ScanTool, error) {
	tools, err := s.scanToolRepo.GetScanToolsByType(ctx, toolType)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_scan_tools_by_type", "SERVICE", map[string]interface{}{
			"operation": "get_scan_tools_by_type",
			"error":     "get_failed",
			"tool_type": toolType,
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("根据类型获取扫描工具失败: %w", err)
	}

	return tools, nil
}

// InstallScanTool 安装扫描工具
// @param ctx 上下文
// @param id 扫描工具配置ID
// @return 错误信息
func (s *ScanToolService) InstallScanTool(ctx context.Context, id uint) error {
	// 参数验证
	if id == 0 {
		return errors.New("扫描工具配置ID不能为0")
	}

	// 获取扫描工具配置
	tool, err := s.scanToolRepo.GetScanToolByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取扫描工具配置失败: %w", err)
	}

	if tool == nil {
		return errors.New("扫描工具配置不存在")
	}

	// TODO: 实现安装逻辑
	// 1. 下载工具包
	// 2. 验证工具包完整性
	// 3. 安装工具
	// 4. 配置环境
	// 5. 验证安装结果

	// 更新状态为已安装
	if err := s.scanToolRepo.UpdateScanToolStatus(ctx, id, orchestrator.ScanToolStatusEnabled); err != nil {
		return fmt.Errorf("更新工具状态失败: %w", err)
	}

	logger.LogSystemEvent("SERVICE", "install_scan_tool", "安装扫描工具", logger.InfoLevel, map[string]interface{}{
		"operation": "install_scan_tool",
		"tool_name": tool.Name,
		"tool_id":   id,
		"status":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return nil
}

// UninstallScanTool 卸载扫描工具
// @param ctx 上下文
// @param id 扫描工具配置ID
// @return 错误信息
func (s *ScanToolService) UninstallScanTool(ctx context.Context, id uint) error {
	// 参数验证
	if id == 0 {
		return errors.New("扫描工具配置ID不能为0")
	}

	// 获取扫描工具配置
	tool, err := s.scanToolRepo.GetScanToolByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取扫描工具配置失败: %w", err)
	}

	if tool == nil {
		return errors.New("扫描工具配置不存在")
	}

	// 检查工具是否正在测试中
	if tool.Status == orchestrator.ScanToolStatusTesting {
		return errors.New("扫描工具正在运行中，无法卸载")
	}

	// TODO: 实现卸载逻辑
	// 1. 停止相关进程
	// 2. 清理工具文件
	// 3. 清理配置文件
	// 4. 清理环境变量

	// 更新状态为未安装
	if err := s.scanToolRepo.UpdateScanToolStatus(ctx, id, orchestrator.ScanToolStatusDisabled); err != nil {
		return fmt.Errorf("更新工具状态失败: %w", err)
	}

	logger.LogSystemEvent("SERVICE", "uninstall_scan_tool", "卸载扫描工具", logger.InfoLevel, map[string]interface{}{
		"operation": "uninstall_scan_tool",
		"tool_name": tool.Name,
		"tool_id":   id,
		"status":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return nil
}

// 私有方法：更新扫描工具状态
func (s *ScanToolService) updateScanToolStatus(ctx context.Context, id uint, status orchestrator.ScanToolStatus, operation string) error {
	// 参数验证
	if id == 0 {
		logger.LogError(errors.New("invalid scan tool ID"), "", 0, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "invalid_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描工具配置ID不能为0")
	}

	// 检查扫描工具配置是否存在
	tool, err := s.scanToolRepo.GetScanToolByID(ctx, id)
	if err != nil {
		logger.LogError(err, "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "get_failed",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("获取扫描工具配置失败: %w", err)
	}

	if tool == nil {
		logger.LogError(errors.New("scan tool not found"), "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "not_found",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return errors.New("扫描工具配置不存在")
	}

	// 更新状态
	if err := s.scanToolRepo.UpdateScanToolStatus(ctx, id, status); err != nil {
		logger.LogError(err, "", id, "", operation, "SERVICE", map[string]interface{}{
			"operation": operation,
			"error":     "update_status_failed",
			"id":        id,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return fmt.Errorf("更新扫描工具状态失败: %w", err)
	}

	// 记录成功日志
	logger.Info(operation+" success", map[string]interface{}{
		"operation": operation,
		"tool_name": tool.Name,
		"tool_id":   id,
		"status":    status,
		"result":    "success",
		"timestamp": logger.NowFormatted(),
	})

	return nil
}

// 私有方法：设置默认值
func (s *ScanToolService) setDefaultValues(tool *orchestrator.ScanTool) {
	if tool.Status == 0 {
		tool.Status = orchestrator.ScanToolStatusDisabled
	}

	if tool.Type == "" {
		tool.Type = orchestrator.ScanToolTypeCustom
	}

	// 设置默认的参数配置
	if tool.DefaultParams == "" {
		tool.DefaultParams = "{}"
	}

	if tool.ParamSchema == "" {
		tool.ParamSchema = "{}"
	}

	if tool.ResultMapping == "" {
		tool.ResultMapping = "{}"
	}

	// 设置默认的依赖项
	if tool.Dependencies == "" {
		tool.Dependencies = ""
	}

	// 设置默认的标签
	if tool.Tags == "" {
		tool.Tags = ""
	}

	// 设置默认执行限制
	if tool.MaxExecutionTime == 0 {
		tool.MaxExecutionTime = 3600 // 默认1小时
	}

	if tool.MaxMemoryMB == 0 {
		tool.MaxMemoryMB = 1024 // 默认1GB内存限制
	}

	// 设置时间戳
	now := time.Now()
	tool.CreatedAt = now
	tool.UpdatedAt = now
}

// BatchInstallScanTools 批量安装扫描工具
// @param ctx 上下文
// @param toolIDs 工具ID列表
// @return 安装结果和错误信息
func (s *ScanToolService) BatchInstallScanTools(ctx context.Context, toolIDs []uint) (map[string]interface{}, error) {
	// 参数验证
	if len(toolIDs) == 0 {
		logger.LogError(errors.New("tool IDs list is empty"), "", 0, "", "batch_install_scan_tools", "SERVICE", map[string]interface{}{
			"operation": "batch_install_scan_tools",
			"error":     "empty_tool_ids",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("工具ID列表不能为空")
	}

	// 记录开始日志
	logger.Info("batch_install_scan_tools start", map[string]interface{}{
		"operation": "batch_install_scan_tools",
		"tool_ids":  toolIDs,
		"count":     len(toolIDs),
		"timestamp": logger.NowFormatted(),
	})

	results := make(map[string]interface{})
	successCount := 0
	failureCount := 0
	details := make([]map[string]interface{}, 0)

	// 逐个安装工具
	for _, toolID := range toolIDs {
		detail := map[string]interface{}{
			"tool_id": toolID,
		}

		// 调用单个工具安装方法
		err := s.InstallScanTool(ctx, toolID)
		if err != nil {
			failureCount++
			detail["status"] = "failed"
			detail["error"] = err.Error()
			logger.LogError(err, "", 0, "", "batch_install_scan_tools", "SERVICE", map[string]interface{}{
				"operation": "batch_install_scan_tools",
				"tool_id":   toolID,
				"error":     err.Error(),
				"timestamp": logger.NowFormatted(),
			})
		} else {
			successCount++
			detail["status"] = "success"
		}

		details = append(details, detail)
	}

	// 构建结果
	results["total"] = len(toolIDs)
	results["success_count"] = successCount
	results["failure_count"] = failureCount
	results["details"] = details

	// 记录完成日志
	logger.Info("batch_install_scan_tools completed", map[string]interface{}{
		"operation":     "batch_install_scan_tools",
		"total":         len(toolIDs),
		"success_count": successCount,
		"failure_count": failureCount,
		"timestamp":     logger.NowFormatted(),
	})

	return results, nil
}

// BatchUninstallScanTools 批量卸载扫描工具
// @param ctx 上下文
// @param toolIDs 工具ID列表
// @return 卸载结果和错误信息
func (s *ScanToolService) BatchUninstallScanTools(ctx context.Context, toolIDs []uint) (map[string]interface{}, error) {
	// 参数验证
	if len(toolIDs) == 0 {
		logger.LogError(errors.New("tool IDs list is empty"), "", 0, "", "batch_uninstall_scan_tools", "SERVICE", map[string]interface{}{
			"operation": "batch_uninstall_scan_tools",
			"error":     "empty_tool_ids",
			"timestamp": logger.NowFormatted(),
		})
		return nil, errors.New("工具ID列表不能为空")
	}

	// 记录开始日志
	logger.Info("batch_uninstall_scan_tools start", map[string]interface{}{
		"operation": "batch_uninstall_scan_tools",
		"tool_ids":  toolIDs,
		"count":     len(toolIDs),
		"timestamp": logger.NowFormatted(),
	})

	results := make(map[string]interface{})
	successCount := 0
	failureCount := 0
	details := make([]map[string]interface{}, 0)

	// 逐个卸载工具
	for _, toolID := range toolIDs {
		detail := map[string]interface{}{
			"tool_id": toolID,
		}

		// 调用单个工具卸载方法
		err := s.UninstallScanTool(ctx, toolID)
		if err != nil {
			failureCount++
			detail["status"] = "failed"
			detail["error"] = err.Error()
			logger.LogError(err, "", 0, "", "batch_uninstall_scan_tools", "SERVICE", map[string]interface{}{
				"operation": "batch_uninstall_scan_tools",
				"tool_id":   toolID,
				"error":     err.Error(),
				"timestamp": logger.NowFormatted(),
			})
		} else {
			successCount++
			detail["status"] = "success"
		}

		details = append(details, detail)
	}

	// 构建结果
	results["total"] = len(toolIDs)
	results["success_count"] = successCount
	results["failure_count"] = failureCount
	results["details"] = details

	// 记录完成日志
	logger.Info("batch_uninstall_scan_tools completed", map[string]interface{}{
		"operation":     "batch_uninstall_scan_tools",
		"total":         len(toolIDs),
		"success_count": successCount,
		"failure_count": failureCount,
		"timestamp":     logger.NowFormatted(),
	})

	return results, nil
}

// GetSystemToolStatus 获取系统工具状态
// @param ctx 上下文
// @return 系统工具状态信息和错误信息
func (s *ScanToolService) GetSystemToolStatus(ctx context.Context) (map[string]interface{}, error) {
	// 记录开始日志
	logger.Info("get_system_tool_status start", map[string]interface{}{
		"operation": "get_system_tool_status",
		"timestamp": logger.NowFormatted(),
	})

	// 获取所有工具
	tools, _, err := s.ListScanTools(ctx, 0, 1000, nil, nil)
	if err != nil {
		logger.LogError(err, "", 0, "", "get_system_tool_status", "SERVICE", map[string]interface{}{
			"operation": "get_system_tool_status",
			"error":     err.Error(),
			"timestamp": logger.NowFormatted(),
		})
		return nil, fmt.Errorf("获取工具列表失败: %v", err)
	}

	// 统计工具状态
	statusStats := make(map[string]int)
	typeStats := make(map[string]int)
	totalTools := len(tools)
	installedCount := 0
	enabledCount := 0

	for _, tool := range tools {
		// 统计状态
		statusKey := tool.Status.String()
		statusStats[statusKey]++

		// 统计类型
		typeKey := tool.Type.String()
		typeStats[typeKey]++

		// 统计安装和启用状态
		if tool.Status == orchestrator.ScanToolStatusEnabled || tool.Status == orchestrator.ScanToolStatusTesting {
			installedCount++
		}
		if tool.Status == orchestrator.ScanToolStatusEnabled {
			enabledCount++
		}
	}

	// 构建系统状态信息
	systemStatus := map[string]interface{}{
		"total_tools":     totalTools,
		"installed_count": installedCount,
		"enabled_count":   enabledCount,
		"disabled_count":  totalTools - enabledCount,
		"status_stats":    statusStats,
		"type_stats":      typeStats,
		"system_health":   "healthy", // 简单的健康状态判断
		"last_updated":    time.Now().Format("2006-01-02 15:04:05"),
	}

	// 简单的健康状态判断
	if enabledCount == 0 {
		systemStatus["system_health"] = "warning"
	} else if float64(enabledCount)/float64(totalTools) < 0.5 {
		systemStatus["system_health"] = "degraded"
	}

	// 记录成功日志
	logger.Info("get_system_tool_status success", map[string]interface{}{
		"operation":   "get_system_tool_status",
		"total_tools": totalTools,
		"enabled":     enabledCount,
		"installed":   installedCount,
		"timestamp":   logger.NowFormatted(),
	})

	return systemStatus, nil
}
