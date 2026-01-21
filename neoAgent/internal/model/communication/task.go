/**
 * 任务相关模型
 * @author: sun977
 * @date: 2025.10.21
 * @description: Agent任务执行相关的数据模型，遵循"好品味"原则
 * @func: 定义任务、任务结果和任务指标的数据结构
 */
package communication

import (
	"time"

	"neoagent/internal/core/model"
)

// ==================== 任务相关 ====================

// Task 任务信息
// 遵循Linus原则：任务结构包含完整的生命周期信息
type Task struct {
	ID         string                 `json:"id"`          // 任务ID
	Name       string                 `json:"name"`        // 任务名称
	Type       string                 `json:"type"`        // 任务类型
	Status     string                 `json:"status"`      // 任务状态
	Priority   int                    `json:"priority"`    // 优先级
	Config     map[string]interface{} `json:"config"`      // 任务配置
	Progress   float64                `json:"progress"`    // 执行进度
	StartTime  time.Time              `json:"start_time"`  // 开始时间
	EndTime    time.Time              `json:"end_time"`    // 结束时间
	Timeout    time.Duration          `json:"timeout"`     // 超时时间
	RetryCount int                    `json:"retry_count"` // 重试次数
	CreatedAt  time.Time              `json:"created_at"`  // 创建时间
	UpdatedAt  time.Time              `json:"updated_at"`  // 更新时间
}

// ToCoreTask 将通信层 DTO 转换为核心层 Domain Object
func (t *Task) ToCoreTask() *model.Task {
	coreTask := &model.Task{
		ID:        t.ID,
		Type:      model.TaskType(t.Type),
		Priority:  t.Priority,
		Timeout:   t.Timeout,
		CreatedAt: t.CreatedAt,
		Params:    t.Config,
	}

	// 尝试提取核心字段
	if val, ok := t.Config["target"]; ok {
		if target, ok := val.(string); ok {
			coreTask.Target = target
		}
	}
	if val, ok := t.Config["port_range"]; ok {
		if portRange, ok := val.(string); ok {
			coreTask.PortRange = portRange
		}
	}

	return coreTask
}

// TaskResult 任务结果
// 遵循"好品味"原则：结果包含执行数据、日志和性能指标
type TaskResult struct {
	TaskID    string                 `json:"task_id"`    // 任务ID
	AgentID   string                 `json:"agent_id"`   // Agent ID
	Status    string                 `json:"status"`     // 执行状态
	Result    map[string]interface{} `json:"result"`     // 执行结果
	Error     string                 `json:"error"`      // 错误信息
	Logs      []string               `json:"logs"`       // 执行日志
	Metrics   *TaskMetrics           `json:"metrics"`    // 任务指标
	StartTime time.Time              `json:"start_time"` // 开始时间
	EndTime   time.Time              `json:"end_time"`   // 结束时间
	Duration  time.Duration          `json:"duration"`   // 执行耗时
	Timestamp time.Time              `json:"timestamp"`  // 结果时间戳
}

// TaskMetrics 任务执行指标
// 遵循单一职责原则：专门负责任务执行过程中的资源消耗统计
type TaskMetrics struct {
	CPUTime      time.Duration `json:"cpu_time"`      // CPU时间
	MemoryPeak   int64         `json:"memory_peak"`   // 内存峰值
	DiskRead     int64         `json:"disk_read"`     // 磁盘读取
	DiskWrite    int64         `json:"disk_write"`    // 磁盘写入
	NetworkRead  int64         `json:"network_read"`  // 网络读取
	NetworkWrite int64         `json:"network_write"` // 网络写入
}

// FromCoreResult 将核心层结果转换为通信层 DTO
func FromCoreResult(cr *model.TaskResult, agentID string) *TaskResult {
	// TODO: TaskMetrics 目前在 core 中还没有定义，暂时为空
	return &TaskResult{
		TaskID:    cr.TaskID,
		AgentID:   agentID,
		Status:    string(cr.Status),
		Error:     cr.Error,
		StartTime: cr.StartTime,
		EndTime:   cr.EndTime,
		Duration:  cr.EndTime.Sub(cr.StartTime),
		Timestamp: time.Now(),
		// Result 需要特殊处理，可能需要序列化
		Result: func() map[string]interface{} {
			if m, ok := cr.Data.(map[string]interface{}); ok {
				return m
			}
			return nil // 或者做反射/JSON转换
		}(),
	}
}
