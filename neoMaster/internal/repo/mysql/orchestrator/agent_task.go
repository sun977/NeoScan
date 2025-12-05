package orchestrator

import (
	"context"
	"errors"
	"fmt"
	agentModel "neomaster/internal/model/orchestrator"
	"time"

	"gorm.io/gorm"
)

// TaskRepository Agent任务仓库接口
type TaskRepository interface {
	CreateTask(ctx context.Context, task *agentModel.AgentTask) error
	UpdateTaskStatus(ctx context.Context, taskID string, status string) error
	GetTaskByID(ctx context.Context, taskID string) (*agentModel.AgentTask, error)
	GetPendingTasks(ctx context.Context, limit int) ([]*agentModel.AgentTask, error)
	UpdateTaskResult(ctx context.Context, taskID string, result string, errorMsg string, status string) error
	GetLatestTaskByProjectID(ctx context.Context, projectID uint64) (*agentModel.AgentTask, error)
	GetTasksByAgentID(ctx context.Context, agentID string) ([]*agentModel.AgentTask, error)
	ClaimTask(ctx context.Context, taskID string, agentID string) error
	HasRunningTasks(ctx context.Context, projectID uint64) (bool, error)
}

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{
		db: db,
	}
}

// CreateTask 创建任务
func (r *taskRepository) CreateTask(ctx context.Context, task *agentModel.AgentTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// UpdateTaskStatus 更新任务状态
func (r *taskRepository) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	return r.db.WithContext(ctx).Model(&agentModel.AgentTask{}).
		Where("task_id = ?", taskID).
		Update("status", status).Error
}

// GetTaskByID 获取指定任务
func (r *taskRepository) GetTaskByID(ctx context.Context, taskID string) (*agentModel.AgentTask, error) {
	var task agentModel.AgentTask
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

// GetPendingTasks 获取所有待处理的任务
func (r *taskRepository) GetPendingTasks(ctx context.Context, limit int) ([]*agentModel.AgentTask, error) {
	var tasks []*agentModel.AgentTask
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("priority desc, created_at asc").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// UpdateTaskResult 更新任务结果
func (r *taskRepository) UpdateTaskResult(ctx context.Context, taskID string, result string, errorMsg string, status string) error {
	updates := map[string]interface{}{
		"output_result": result,
		"error_msg":     errorMsg,
		"status":        status,
		"finished_at":   gorm.Expr("NOW()"),
	}
	return r.db.WithContext(ctx).Model(&agentModel.AgentTask{}).
		Where("task_id = ?", taskID).
		Updates(updates).Error
}

// GetLatestTaskByProjectID 获取指定项目的最新任务
func (r *taskRepository) GetLatestTaskByProjectID(ctx context.Context, projectID uint64) (*agentModel.AgentTask, error) {
	var task agentModel.AgentTask
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at desc").
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

// GetTasksByAgentID 获取指定Agent的已分配任务
func (r *taskRepository) GetTasksByAgentID(ctx context.Context, agentID string) ([]*agentModel.AgentTask, error) {
	var tasks []*agentModel.AgentTask
	err := r.db.WithContext(ctx).
		Where("agent_id = ? AND status IN ?", agentID, []string{"assigned", "running"}).
		Order("priority desc, created_at asc").
		Find(&tasks).Error
	return tasks, err
}

// // ClaimTask 认领任务
func (r *taskRepository) ClaimTask(ctx context.Context, taskID string, agentID string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&agentModel.AgentTask{}).
		Where("task_id = ? AND status = ?", taskID, "pending").
		Updates(map[string]interface{}{
			"agent_id":    agentID,
			"status":      "assigned",
			"assigned_at": &now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task %s not available for claim", taskID)
	}
	return nil
}

// HasRunningTasks 检查项目是否有正在运行的任务
func (r *taskRepository) HasRunningTasks(ctx context.Context, projectID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&agentModel.AgentTask{}).
		Where("project_id = ? AND status IN ?", projectID, []string{"pending", "assigned", "running"}).
		Count(&count).Error
	return count > 0, err
}
