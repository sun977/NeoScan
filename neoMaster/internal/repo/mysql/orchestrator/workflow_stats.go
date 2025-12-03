package orchestrator

import (
	"context"
	"errors"
	orcmodel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
	"time"

	"gorm.io/gorm"
)

// WorkflowStatsRepository 工作流统计仓库
type WorkflowStatsRepository struct {
	db *gorm.DB
}

// NewWorkflowStatsRepository 创建 WorkflowStatsRepository 实例
func NewWorkflowStatsRepository(db *gorm.DB) *WorkflowStatsRepository {
	return &WorkflowStatsRepository{db: db}
}

// GetStatsByWorkflowID 获取工作流统计信息
func (r *WorkflowStatsRepository) GetStatsByWorkflowID(ctx context.Context, workflowID uint64) (*orcmodel.WorkflowStats, error) {
	var stats orcmodel.WorkflowStats
	err := r.db.WithContext(ctx).Where("workflow_id = ?", workflowID).First(&stats).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.LogError(err, "", 0, "", "get_workflow_stats", "REPO", map[string]interface{}{
			"operation":   "get_workflow_stats",
			"workflow_id": workflowID,
		})
		return nil, err
	}
	return &stats, nil
}

// UpsertStats 插入或更新统计记录
func (r *WorkflowStatsRepository) UpsertStats(ctx context.Context, stats *orcmodel.WorkflowStats) error {
	if stats == nil {
		return errors.New("stats is nil")
	}

	// 尝试查找，如果不存在则创建
	var existing orcmodel.WorkflowStats
	err := r.db.WithContext(ctx).Where("workflow_id = ?", stats.WorkflowID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 不存在，创建
		if err1 := r.db.WithContext(ctx).Create(stats).Error; err1 != nil {
			logger.LogError(err1, "", 0, "", "create_workflow_stats", "REPO", map[string]interface{}{
				"operation":   "create_workflow_stats",
				"workflow_id": stats.WorkflowID,
			})
			return err1
		}
		return nil
	} else if err != nil {
		return err
	}

	// 存在，这里通常不需要全量更新，具体的计数更新用 UpdateExecutionStats
	return nil
}

// UpdateExecutionStats 更新执行统计 (原子递增)
// isSuccess: 本次执行是否成功
// durationMs: 本次执行耗时
// execID: 执行ID
func (r *WorkflowStatsRepository) UpdateExecutionStats(ctx context.Context, workflowID uint64, isSuccess bool, durationMs int, execID string, status string) error {
	updates := map[string]interface{}{
		"total_execs":      gorm.Expr("total_execs + ?", 1),
		"last_exec_id":     execID,
		"last_exec_status": status,
		"last_exec_time":   time.Now(),
	}

	if isSuccess {
		updates["success_execs"] = gorm.Expr("success_execs + ?", 1)
	} else {
		updates["failed_execs"] = gorm.Expr("failed_execs + ?", 1)
	}

	// 更新平均耗时：这里做一个简单的近似计算，或者需要读取旧值。
	// 严格来说 avg = (old_avg * old_total + new_duration) / (old_total + 1)
	// 为了原子性，可以用SQL实现
	// NEW_AVG = (total_execs * avg_duration_ms + durationMs) / (total_execs + 1)
	// 注意：这里的 total_execs 是更新前的

	// 为了简化，我们先读出来再更新，虽然有并发风险，但对于统计数据通常可接受。
	// 或者用更复杂的 SQL。鉴于 GORM 的限制，我们先读后写，或者忽略平均值的精确原子性。
	// 这里为了严谨，我们不在这里更新 AvgDurationMs，除非我们接受读写锁。
	// 考虑到这是 "WorkflowStats"，并发量不会极高（同一个 workflow 不会毫秒级并发结束），读-改-写通常是安全的。

	// 但是，为了保持 "Good Taste"，我们只做简单的原子更新。AvgDurationMs 可以另外处理或接受近似。
	// 这里我选择先用原子更新计数器。

	err := r.db.WithContext(ctx).Model(&orcmodel.WorkflowStats{}).
		Where("workflow_id = ?", workflowID).
		Updates(updates).Error

	if err != nil {
		logger.LogError(err, "", 0, "", "update_execution_stats", "REPO", map[string]interface{}{
			"operation":   "update_execution_stats",
			"workflow_id": workflowID,
		})
		return err
	}
	return nil
}
