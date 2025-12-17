/**
 * 模型:项目配置模型
 * @author: Sun977
 * @date: 2025.10.11
 * @description: 扫描项目配置数据模型，遵循"好品味"设计原则 - 简洁、无特殊情况、数据结构优先
 * @func: ProjectConfig 结构体及相关方法
 */
package orchestrator_model_drop

import (
	"neomaster/internal/model/basemodel"
	"strings"
	"time"
)

// ProjectConfigStatus 项目配置状态枚举
// 遵循Linus原则：明确的状态定义，避免魔法数字
type ProjectConfigStatus int

const (
	ProjectConfigStatusInactive ProjectConfigStatus = 0 // 未激活
	ProjectConfigStatusActive   ProjectConfigStatus = 1 // 激活
	ProjectConfigStatusArchived ProjectConfigStatus = 2 // 已归档
)

// String 实现Stringer接口，提供状态的字符串表示
func (s ProjectConfigStatus) String() string {
	switch s {
	case ProjectConfigStatusInactive:
		return "inactive"
	case ProjectConfigStatusActive:
		return "active"
	case ProjectConfigStatusArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// ProjectConfig 项目配置模型
// 设计哲学：数据结构决定算法复杂度，好的数据结构让代码变得简单
type ProjectConfig struct {
	// 继承基础模型 - 复用而非重复
	basemodel.BaseModel

	// 项目基本信息
	Name        string `json:"name" gorm:"not null;size:100;comment:项目名称" validate:"required,min=1,max=100"`
	DisplayName string `json:"display_name" gorm:"size:200;comment:项目显示名称"`
	Description string `json:"description" gorm:"type:text;comment:项目描述"`

	// 项目配置
	TargetScope string `json:"target_scope" gorm:"type:text;comment:扫描目标范围，支持IP段、域名等" validate:"required"`
	ExcludeList string `json:"exclude_list" gorm:"type:text;comment:排除列表，不扫描的目标"`

	// 扫描配置
	ScanFrequency int `json:"scan_frequency" gorm:"default:24;comment:扫描频率(小时)" validate:"min=1,max=168"`
	MaxConcurrent int `json:"max_concurrent" gorm:"default:10;comment:最大并发数" validate:"min=1,max=100"`
	TimeoutSecond int `json:"timeout_second" gorm:"default:300;comment:超时时间(秒)" validate:"min=30,max=3600"`
	Priority      int `json:"priority" gorm:"default:5;comment:优先级(1-10)" validate:"min=1,max=10"`

	// 通知配置
	NotifyOnSuccess bool   `json:"notify_on_success" gorm:"default:false;comment:成功时是否通知"`
	NotifyOnFailure bool   `json:"notify_on_failure" gorm:"default:true;comment:失败时是否通知"`
	NotifyEmails    string `json:"notify_emails" gorm:"type:text;comment:通知邮箱列表，逗号分隔"`

	// 状态管理
	Status    ProjectConfigStatus `json:"status" gorm:"default:0;comment:项目状态:0-未激活,1-激活,2-已归档"`
	IsEnabled bool                `json:"is_enabled" gorm:"default:true;comment:是否启用"`

	// 元数据
	Tags     string `json:"tags" gorm:"type:text;comment:项目标签，逗号分隔"`
	Metadata string `json:"metadata" gorm:"type:json;comment:扩展元数据"`

	// 审计字段
	CreatedBy uint64     `json:"created_by" gorm:"comment:创建者ID"`
	UpdatedBy uint64     `json:"updated_by" gorm:"comment:更新者ID"`
	LastScan  *time.Time `json:"last_scan" gorm:"comment:最后扫描时间"`

	// 关联关系 - 延迟加载，避免N+1问题
	Workflows []WorkflowConfig `json:"workflows,omitempty" gorm:"foreignKey:ProjectID"`
}

// TableName 定义数据库表名
// 遵循项目命名约定
func (ProjectConfig) TableName() string {
	return "project_configs"
}

// IsActive 检查项目是否处于激活状态
// 简单的业务逻辑方法，避免在多处重复判断
func (pc *ProjectConfig) IsActive() bool {
	return pc.Status == ProjectConfigStatusActive && pc.IsEnabled
}

// CanExecuteScan 检查项目是否可以执行扫描
// 业务规则封装，单一职责
func (pc *ProjectConfig) CanExecuteScan() bool {
	return pc.IsActive() && pc.TargetScope != ""
}

// GetNotifyEmailList 获取通知邮箱列表
// 数据转换逻辑封装，避免在业务层重复处理
func (pc *ProjectConfig) GetNotifyEmailList() []string {
	if pc.NotifyEmails == "" {
		return []string{}
	}

	// 简单的字符串分割，避免复杂的正则表达式
	emails := make([]string, 0)
	for _, email := range strings.Split(pc.NotifyEmails, ",") {
		if trimmed := strings.TrimSpace(email); trimmed != "" {
			emails = append(emails, trimmed)
		}
	}
	return emails
}

// GetTagList 获取标签列表
// 统一的标签处理逻辑
func (pc *ProjectConfig) GetTagList() []string {
	if pc.Tags == "" {
		return []string{}
	}

	tags := make([]string, 0)
	for _, tag := range strings.Split(pc.Tags, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}
