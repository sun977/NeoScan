/**
 * 模型:工作流配置模型
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 扫描工作流配置模型，遵循"简洁执念"原则 - 消除复杂性，让工作流编排变得简单
 * @func: WorkflowConfig 结构体及相关方法
 */
package orchestrator

import (
	"encoding/json"
	"fmt"
	"strings"

	"neomaster/internal/model"
)

// WorkflowStatus 工作流状态枚举
type WorkflowStatus int

const (
	WorkflowStatusDraft    WorkflowStatus = 0 // 草稿
	WorkflowStatusActive   WorkflowStatus = 1 // 激活
	WorkflowStatusInactive WorkflowStatus = 2 // 未激活
	WorkflowStatusArchived WorkflowStatus = 3 // 已归档
)

// String 实现Stringer接口
func (s WorkflowStatus) String() string {
	switch s {
	case WorkflowStatusDraft:
		return "draft"
	case WorkflowStatusActive:
		return "active"
	case WorkflowStatusInactive:
		return "inactive"
	case WorkflowStatusArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// WorkflowTriggerType 工作流触发类型
type WorkflowTriggerType string

const (
	WorkflowTriggerManual    WorkflowTriggerType = "manual"    // 手动触发
	WorkflowTriggerScheduled WorkflowTriggerType = "scheduled" // 定时触发
	WorkflowTriggerEvent     WorkflowTriggerType = "event"     // 事件触发
)

// String 实现Stringer接口
func (t WorkflowTriggerType) String() string {
	return string(t)
}

// WorkflowConfig 工作流配置模型
// 设计哲学：工作流应该是线性的、可预测的，避免复杂的分支和循环
type WorkflowConfig struct {
	// 继承基础模型
	model.BaseModel

	// 工作流基本信息
	Name        string `json:"name" gorm:"not null;size:100;comment:工作流名称" validate:"required,min=1,max=100"`
	DisplayName string `json:"display_name" gorm:"size:200;comment:工作流显示名称"`
	Description string `json:"description" gorm:"type:text;comment:工作流描述"`
	Version     string `json:"version" gorm:"size:20;default:'1.0';comment:工作流版本"`

	// 关联关系
	ProjectID uint64 `json:"project_id" gorm:"not null;index;comment:所属项目ID" validate:"required"`

	// 触发配置
	TriggerType WorkflowTriggerType `json:"trigger_type" gorm:"size:20;not null;comment:触发类型" validate:"required"`
	CronExpr    string              `json:"cron_expr" gorm:"size:100;comment:定时表达式，仅定时触发时使用"`
	EventFilter string              `json:"event_filter" gorm:"type:json;comment:事件过滤条件，仅事件触发时使用"`

	// 工作流定义 - 使用JSON存储步骤定义，简单且灵活
	Steps       string `json:"steps" gorm:"type:json;not null;comment:工作流步骤定义" validate:"required"`
	Variables   string `json:"variables" gorm:"type:json;comment:工作流变量定义"`
	Environment string `json:"environment" gorm:"type:json;comment:环境变量配置"`

	// 执行配置
	MaxRetries      int  `json:"max_retries" gorm:"default:3;comment:最大重试次数" validate:"min=0,max=10"`
	TimeoutMinutes  int  `json:"timeout_minutes" gorm:"default:60;comment:超时时间(分钟)" validate:"min=1,max=1440"`
	ContinueOnError bool `json:"continue_on_error" gorm:"default:false;comment:出错时是否继续执行"`
	ParallelSteps   bool `json:"parallel_steps" gorm:"default:false;comment:是否并行执行步骤"`

	// 通知配置
	NotifyOnStart   bool   `json:"notify_on_start" gorm:"default:false;comment:开始时是否通知"`
	NotifyOnSuccess bool   `json:"notify_on_success" gorm:"default:false;comment:成功时是否通知"`
	NotifyOnFailure bool   `json:"notify_on_failure" gorm:"default:true;comment:失败时是否通知"`
	NotifyChannels  string `json:"notify_channels" gorm:"type:text;comment:通知渠道配置"`

	// 状态管理
	Status    WorkflowStatus `json:"status" gorm:"default:0;comment:工作流状态:0-草稿,1-激活,2-未激活,3-已归档"`
	IsEnabled bool           `json:"is_enabled" gorm:"default:true;comment:是否启用"`

	// 元数据
	Tags     string `json:"tags" gorm:"type:text;comment:工作流标签，逗号分隔"`
	Metadata string `json:"metadata" gorm:"type:json;comment:扩展元数据"`

	// 审计字段
	CreatedBy uint64 `json:"created_by" gorm:"comment:创建者ID"`
	UpdatedBy uint64 `json:"updated_by" gorm:"comment:更新者ID"`

	// 统计信息
	ExecutionCount int64 `json:"execution_count" gorm:"default:0;comment:执行次数"`
	SuccessCount   int64 `json:"success_count" gorm:"default:0;comment:成功次数"`
	FailureCount   int64 `json:"failure_count" gorm:"default:0;comment:失败次数"`

	// 关联关系 - 延迟加载
	Project *ProjectConfig `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

// WorkflowStep 工作流步骤结构
// 简单的步骤定义，避免过度复杂的编排逻辑
type WorkflowStep struct {
	ID              string                 `json:"id" validate:"required"`      // 步骤ID
	Name            string                 `json:"name" validate:"required"`    // 步骤名称
	Type            string                 `json:"type" validate:"required"`    // 步骤类型：scan_tool, script, condition, notification
	ToolID          uint64                 `json:"tool_id,omitempty"`           // 扫描工具ID（当type为scan_tool时）
	Command         string                 `json:"command,omitempty"`           // 执行命令（当type为script时）
	Condition       string                 `json:"condition,omitempty"`         // 条件表达式（当type为condition时）
	Parameters      map[string]interface{} `json:"parameters,omitempty"`        // 步骤参数
	DependsOn       []string               `json:"depends_on,omitempty"`        // 依赖的步骤ID列表
	Timeout         int                    `json:"timeout,omitempty"`           // 步骤超时时间(秒)
	RetryCount      int                    `json:"retry_count,omitempty"`       // 重试次数
	ContinueOnError bool                   `json:"continue_on_error,omitempty"` // 出错时是否继续
}

// TableName 定义数据库表名
func (WorkflowConfig) TableName() string {
	return "workflow_configs"
}

// IsActive 检查工作流是否激活
func (wc *WorkflowConfig) IsActive() bool {
	return wc.Status == WorkflowStatusActive && wc.IsEnabled
}

// CanExecute 检查工作流是否可以执行
func (wc *WorkflowConfig) CanExecute() bool {
	return wc.IsActive() && wc.Steps != ""
}

// IsScheduled 检查是否为定时触发
func (wc *WorkflowConfig) IsScheduled() bool {
	return wc.TriggerType == WorkflowTriggerScheduled && wc.CronExpr != ""
}

// GetSteps 获取工作流步骤列表
func (wc *WorkflowConfig) GetSteps() ([]WorkflowStep, error) {
	if wc.Steps == "" {
		return []WorkflowStep{}, nil
	}

	var steps []WorkflowStep
	if err := json.Unmarshal([]byte(wc.Steps), &steps); err != nil {
		return nil, fmt.Errorf("解析工作流步骤失败: %w", err)
	}
	return steps, nil
}

// SetSteps 设置工作流步骤
func (wc *WorkflowConfig) SetSteps(steps []WorkflowStep) error {
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return fmt.Errorf("序列化工作流步骤失败: %w", err)
	}
	wc.Steps = string(stepsJSON)
	return nil
}

// GetVariables 获取工作流变量
func (wc *WorkflowConfig) GetVariables() (map[string]interface{}, error) {
	if wc.Variables == "" {
		return make(map[string]interface{}), nil
	}

	var variables map[string]interface{}
	if err := json.Unmarshal([]byte(wc.Variables), &variables); err != nil {
		return nil, fmt.Errorf("解析工作流变量失败: %w", err)
	}
	return variables, nil
}

// SetVariables 设置工作流变量
func (wc *WorkflowConfig) SetVariables(variables map[string]interface{}) error {
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return fmt.Errorf("序列化工作流变量失败: %w", err)
	}
	wc.Variables = string(variablesJSON)
	return nil
}

// GetEnvironment 获取环境变量配置
func (wc *WorkflowConfig) GetEnvironment() (map[string]string, error) {
	if wc.Environment == "" {
		return make(map[string]string), nil
	}

	var env map[string]string
	if err := json.Unmarshal([]byte(wc.Environment), &env); err != nil {
		return nil, fmt.Errorf("解析环境变量失败: %w", err)
	}
	return env, nil
}

// GetTagList 获取标签列表
func (wc *WorkflowConfig) GetTagList() []string {
	if wc.Tags == "" {
		return []string{}
	}

	tags := make([]string, 0)
	for _, tag := range strings.Split(wc.Tags, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

// ValidateSteps 验证工作流步骤的合法性
// 简单的验证逻辑，确保步骤定义正确
func (wc *WorkflowConfig) ValidateSteps() error {
	steps, err := wc.GetSteps()
	if err != nil {
		return err
	}

	if len(steps) == 0 {
		return fmt.Errorf("工作流至少需要一个步骤")
	}

	// 检查步骤ID唯一性
	stepIDs := make(map[string]bool)
	for _, step := range steps {
		if step.ID == "" {
			return fmt.Errorf("步骤ID不能为空")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("步骤ID重复: %s", step.ID)
		}
		stepIDs[step.ID] = true

		// 检查依赖关系
		for _, depID := range step.DependsOn {
			if !stepIDs[depID] {
				return fmt.Errorf("步骤 %s 依赖的步骤 %s 不存在", step.ID, depID)
			}
		}
	}

	return nil
}

// UpdateExecutionStats 更新执行统计
func (wc *WorkflowConfig) UpdateExecutionStats(success bool) {
	wc.ExecutionCount++
	if success {
		wc.SuccessCount++
	} else {
		wc.FailureCount++
	}
}

// GetSuccessRate 获取成功率
func (wc *WorkflowConfig) GetSuccessRate() float64 {
	if wc.ExecutionCount == 0 {
		return 0.0
	}
	return float64(wc.SuccessCount) / float64(wc.ExecutionCount) * 100
}
