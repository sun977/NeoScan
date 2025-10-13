/*
 * 扫描工具仓库层：扫描工具数据访问
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 单纯数据访问，不应该包含业务逻辑
 * @func:
 * 1.创建扫描工具配置
 * 2.更新扫描工具配置
 * 3.删除扫描工具配置
 * 4.扫描工具状态变更等
 */

//  基础CRUD操作:
//  	CreateScanTool - 创建扫描工具
//  	GetScanToolByID - 根据ID获取扫描工具
//  	GetScanToolByName - 根据工具名获取扫描工具
//  	UpdateScanTool - 更新扫描工具信息
//  	DeleteScanTool - 软删除扫描工具
//  高级查询功能:
//  	GetScanToolList - 分页获取扫描工具列表
//  	GetScanToolsByType - 根据类型获取扫描工具
//  	GetAvailableScanTools - 获取可用的扫描工具
//  	ScanToolExists - 检查扫描工具是否存在
//  状态管理:
//  	UpdateScanToolStatus - 更新扫描工具状态
//  	EnableScanTool - 启用扫描工具
//  	DisableScanTool - 禁用扫描工具
//  统计功能:
//  	UpdateScanToolStats - 更新扫描工具统计信息
//  	IncrementUsageCount - 增加使用次数
//  事务支持:
//  	BeginTx - 开始事务
//  	UpdateScanToolWithTx - 事务更新扫描工具
//  	DeleteScanToolWithTx - 事务删除扫描工具
//  字段更新:
//  	UpdateScanToolFields - 使用map更新特定字段

package orchestrator

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"

	"gorm.io/gorm"
)

// ScanToolRepository 扫描工具仓库结构体
// 负责处理扫描工具相关的数据访问，不包含业务逻辑
type ScanToolRepository struct {
	db *gorm.DB // 数据库连接
}

// NewScanToolRepository 创建扫描工具仓库实例
// 注入数据库连接，专注于数据访问操作
func NewScanToolRepository(db *gorm.DB) *ScanToolRepository {
	return &ScanToolRepository{
		db: db,
	}
}

// CreateScanTool 创建扫描工具
// @param ctx 上下文
// @param tool 扫描工具对象
// @return 错误信息
func (r *ScanToolRepository) CreateScanTool(ctx context.Context, tool *orchestrator.ScanTool) error {
	tool.CreatedAt = time.Now()
	tool.UpdatedAt = time.Now()

	err := r.db.WithContext(ctx).Create(tool).Error
	if err != nil {
		// 记录创建失败日志
		logger.LogError(err, "", 0, "", "scan_tool_create", "POST", map[string]interface{}{
			"operation": "create_scan_tool",
			"tool_name": tool.Name,
			"tool_type": tool.Type,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// GetScanToolByID 根据ID获取扫描工具
// @param ctx 上下文
// @param id 扫描工具ID
// @return 扫描工具对象和错误信息
func (r *ScanToolRepository) GetScanToolByID(ctx context.Context, id uint) (*orchestrator.ScanTool, error) {
	var tool orchestrator.ScanTool
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tool).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("scan tool not found"), "", 0, "", "scan_tool_get", "GET", map[string]interface{}{
				"operation": "get_scan_tool_by_id",
				"id":        id,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "scan_tool_get", "GET", map[string]interface{}{
			"operation": "get_scan_tool_by_id",
			"id":        id,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &tool, nil
}

// GetScanToolByName 根据工具名获取扫描工具
// @param ctx 上下文
// @param name 工具名称
// @return 扫描工具对象和错误信息
func (r *ScanToolRepository) GetScanToolByName(ctx context.Context, name string) (*orchestrator.ScanTool, error) {
	var tool orchestrator.ScanTool
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tool).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 记录查询失败日志
			logger.LogError(fmt.Errorf("scan tool not found"), "", 0, "", "scan_tool_get", "GET", map[string]interface{}{
				"operation": "get_scan_tool_by_name",
				"tool_name": name,
				"timestamp": logger.NowFormatted(),
			})
			return nil, nil // 返回 nil 而不是错误，让业务层处理
		}
		// 记录数据库错误日志
		logger.LogError(err, "", 0, "", "scan_tool_get", "GET", map[string]interface{}{
			"operation": "get_scan_tool_by_name",
			"tool_name": name,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return &tool, nil
}

// UpdateScanTool 更新扫描工具
// @param ctx 上下文
// @param tool 扫描工具对象
// @return 错误信息
func (r *ScanToolRepository) UpdateScanTool(ctx context.Context, tool *orchestrator.ScanTool) error {
	tool.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(tool).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(tool.ID), "", "scan_tool_update", "PUT", map[string]interface{}{
			"operation": "update_scan_tool",
			"tool_name": tool.Name,
			"tool_id":   tool.ID,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// DeleteScanTool 软删除扫描工具
// @param ctx 上下文
// @param id 扫描工具ID
// @return 错误信息
func (r *ScanToolRepository) DeleteScanTool(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Delete(&orchestrator.ScanTool{}, id).Error
	if err != nil {
		// 记录删除失败日志
		logger.LogError(err, "", id, "", "scan_tool_delete", "DELETE", map[string]interface{}{
			"operation": "delete_scan_tool",
			"tool_id":   id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// GetScanToolList 分页获取扫描工具列表
// @param ctx 上下文
// @param offset 偏移量
// @param limit 限制数量
// @param toolType 工具类型过滤（可选）
// @param status 状态过滤（可选）
// @return 扫描工具列表、总数和错误信息
func (r *ScanToolRepository) GetScanToolList(ctx context.Context, offset, limit int, toolType *orchestrator.ScanToolType, status *orchestrator.ScanToolStatus) ([]*orchestrator.ScanTool, int64, error) {
	var tools []*orchestrator.ScanTool
	var total int64

	query := r.db.WithContext(ctx).Model(&orchestrator.ScanTool{})

	// 工具类型过滤
	if toolType != nil {
		query = query.Where("type = ?", *toolType)
	}

	// 状态过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.LogError(err, "", 0, "", "scan_tool_list", "GET", map[string]interface{}{
			"operation": "get_scan_tool_list_count",
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, err
	}

	// 获取分页数据
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&tools).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_tool_list", "GET", map[string]interface{}{
			"operation": "get_scan_tool_list",
			"offset":    offset,
			"limit":     limit,
			"timestamp": logger.NowFormatted(),
		})
		return nil, 0, err
	}

	return tools, total, nil
}

// GetScanToolsByType 根据类型获取扫描工具
// @param ctx 上下文
// @param toolType 工具类型
// @return 扫描工具列表和错误信息
func (r *ScanToolRepository) GetScanToolsByType(ctx context.Context, toolType orchestrator.ScanToolType) ([]*orchestrator.ScanTool, error) {
	var tools []*orchestrator.ScanTool
	err := r.db.WithContext(ctx).Where("type = ? AND status = ?", toolType, orchestrator.ScanToolStatusEnabled).Find(&tools).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_tool_list", "GET", map[string]interface{}{
			"operation": "get_scan_tools_by_type",
			"tool_type": toolType,
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return tools, nil
}

// GetAvailableScanTools 获取可用的扫描工具
// @param ctx 上下文
// @return 扫描工具列表和错误信息
func (r *ScanToolRepository) GetAvailableScanTools(ctx context.Context) ([]*orchestrator.ScanTool, error) {
	var tools []*orchestrator.ScanTool
	err := r.db.WithContext(ctx).Where("status = ?", orchestrator.ScanToolStatusEnabled).Find(&tools).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_tool_list", "GET", map[string]interface{}{
			"operation": "get_available_scan_tools",
			"timestamp": logger.NowFormatted(),
		})
		return nil, err
	}
	return tools, nil
}

// ScanToolExists 检查扫描工具是否存在
// @param ctx 上下文
// @param name 工具名称
// @return 是否存在和错误信息
func (r *ScanToolRepository) ScanToolExists(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanTool{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		logger.LogError(err, "", 0, "", "scan_tool_exists", "GET", map[string]interface{}{
			"operation": "scan_tool_exists",
			"tool_name": name,
			"timestamp": logger.NowFormatted(),
		})
		return false, err
	}
	return count > 0, nil
}

// UpdateScanToolStatus 更新扫描工具状态
// @param ctx 上下文
// @param id 扫描工具ID
// @param status 新状态
// @return 错误信息
func (r *ScanToolRepository) UpdateScanToolStatus(ctx context.Context, id uint, status orchestrator.ScanToolStatus) error {
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanTool{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_tool_update", "PUT", map[string]interface{}{
			"operation": "update_scan_tool_status",
			"tool_id":   id,
			"status":    status,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// EnableScanTool 启用扫描工具
// @param ctx 上下文
// @param id 扫描工具ID
// @return 错误信息
func (r *ScanToolRepository) EnableScanTool(ctx context.Context, id uint) error {
	return r.UpdateScanToolStatus(ctx, id, orchestrator.ScanToolStatusEnabled)
}

// DisableScanTool 禁用扫描工具
// @param ctx 上下文
// @param id 扫描工具ID
// @return 错误信息
func (r *ScanToolRepository) DisableScanTool(ctx context.Context, id uint) error {
	return r.UpdateScanToolStatus(ctx, id, orchestrator.ScanToolStatusDisabled)
}

// UpdateScanToolStats 更新扫描工具统计信息
// @param ctx 上下文
// @param id 扫描工具ID
// @param usageCount 使用次数
// @param successCount 成功次数
// @param failureCount 失败次数
// @return 错误信息
func (r *ScanToolRepository) UpdateScanToolStats(ctx context.Context, id uint, usageCount, successCount, failureCount int) error {
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanTool{}).Where("id = ?", id).Updates(map[string]interface{}{
		"usage_count":   usageCount,
		"success_count": successCount,
		"failure_count": failureCount,
		"updated_at":    time.Now(),
	}).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_tool_update", "PUT", map[string]interface{}{
			"operation":     "update_scan_tool_stats",
			"tool_id":       id,
			"usage_count":   usageCount,
			"success_count": successCount,
			"failure_count": failureCount,
			"timestamp":     logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// IncrementUsageCount 增加使用次数
// @param ctx 上下文
// @param id 扫描工具ID
// @param success 是否成功
// @return 错误信息
func (r *ScanToolRepository) IncrementUsageCount(ctx context.Context, id uint, success bool) error {
	updates := map[string]interface{}{
		"usage_count": gorm.Expr("usage_count + 1"),
		"updated_at":  time.Now(),
	}

	if success {
		updates["success_count"] = gorm.Expr("success_count + 1")
	} else {
		updates["failure_count"] = gorm.Expr("failure_count + 1")
	}

	err := r.db.WithContext(ctx).Model(&orchestrator.ScanTool{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_tool_update", "PUT", map[string]interface{}{
			"operation": "increment_usage_count",
			"tool_id":   id,
			"success":   success,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// BeginTx 开始事务
// @return 事务对象和错误信息
func (r *ScanToolRepository) BeginTx() (*gorm.DB, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		logger.LogError(tx.Error, "", 0, "", "scan_tool_transaction", "BEGIN", map[string]interface{}{
			"operation": "begin_transaction",
			"timestamp": logger.NowFormatted(),
		})
		return nil, tx.Error
	}
	return tx, nil
}

// UpdateScanToolWithTx 使用事务更新扫描工具
// @param ctx 上下文
// @param tx 事务对象
// @param tool 扫描工具对象
// @return 错误信息
func (r *ScanToolRepository) UpdateScanToolWithTx(ctx context.Context, tx *gorm.DB, tool *orchestrator.ScanTool) error {
	tool.UpdatedAt = time.Now()
	err := tx.WithContext(ctx).Save(tool).Error
	if err != nil {
		// 记录更新失败日志
		logger.LogError(err, "", uint(tool.ID), "", "scan_tool_update_with_tx", "PUT", map[string]interface{}{
			"operation": "update_scan_tool_with_transaction",
			"tool_name": tool.Name,
			"tool_id":   tool.ID,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// DeleteScanToolWithTx 使用事务删除扫描工具
// @param ctx 上下文
// @param tx 事务对象
// @param id 扫描工具ID
// @return 错误信息
func (r *ScanToolRepository) DeleteScanToolWithTx(ctx context.Context, tx *gorm.DB, id uint) error {
	err := tx.WithContext(ctx).Delete(&orchestrator.ScanTool{}, id).Error
	if err != nil {
		// 记录删除失败日志
		logger.LogError(err, "", id, "", "scan_tool_delete_with_tx", "DELETE", map[string]interface{}{
			"operation": "delete_scan_tool_with_transaction",
			"tool_id":   id,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}

// UpdateScanToolFields 使用map更新特定字段
// @param ctx 上下文
// @param id 扫描工具ID
// @param fields 要更新的字段
// @return 错误信息
func (r *ScanToolRepository) UpdateScanToolFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	err := r.db.WithContext(ctx).Model(&orchestrator.ScanTool{}).Where("id = ?", id).Updates(fields).Error
	if err != nil {
		logger.LogError(err, "", id, "", "scan_tool_update", "PUT", map[string]interface{}{
			"operation": "update_scan_tool_fields",
			"tool_id":   id,
			"fields":    fields,
			"timestamp": logger.NowFormatted(),
		})
		return err
	}
	return nil
}
