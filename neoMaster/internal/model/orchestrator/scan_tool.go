/**
 * 模型:扫描工具模型
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 第三方扫描工具配置模型，遵循"实用主义"原则 - 解决实际问题，不追求理论完美
 * @func: ScanTool 结构体及相关方法
 */
package orchestrator

import (
	"encoding/json"
	"fmt"
	"strings"

	"neomaster/internal/model"
)

// ScanToolType 扫描工具类型枚举
// 明确的工具分类，避免字符串魔法值
type ScanToolType string

const (
	ScanToolTypePortScan    ScanToolType = "port_scan"    // 端口扫描
	ScanToolTypeVulnScan    ScanToolType = "vuln_scan"    // 漏洞扫描
	ScanToolTypeWebScan     ScanToolType = "web_scan"     // Web扫描
	ScanToolTypeNetworkScan ScanToolType = "network_scan" // 网络扫描
	ScanToolTypeCustom      ScanToolType = "custom"       // 自定义工具
)

// String 实现Stringer接口
func (t ScanToolType) String() string {
	return string(t)
}

// ScanToolStatus 扫描工具状态枚举
type ScanToolStatus int

const (
	ScanToolStatusDisabled ScanToolStatus = 0 // 禁用
	ScanToolStatusEnabled  ScanToolStatus = 1 // 启用
	ScanToolStatusTesting  ScanToolStatus = 2 // 测试中
)

// String 实现Stringer接口
func (s ScanToolStatus) String() string {
	switch s {
	case ScanToolStatusDisabled:
		return "disabled"
	case ScanToolStatusEnabled:
		return "enabled"
	case ScanToolStatusTesting:
		return "testing"
	default:
		return "unknown"
	}
}

// ScanTool 扫描工具模型
// 设计原则：数据结构优先，让工具配置变得简单直观
type ScanTool struct {
	// 继承基础模型
	model.BaseModel

	// 工具基本信息
	Name        string       `json:"name" gorm:"uniqueIndex;not null;size:100;comment:工具名称，唯一" validate:"required,min=1,max=100"`
	DisplayName string       `json:"display_name" gorm:"size:200;comment:工具显示名称"`
	Description string       `json:"description" gorm:"type:text;comment:工具描述"`
	Type        ScanToolType `json:"type" gorm:"size:50;not null;comment:工具类型" validate:"required"`
	Version     string       `json:"version" gorm:"size:50;comment:工具版本"`

	// 执行配置
	ExecutablePath  string `json:"executable_path" gorm:"size:500;comment:可执行文件路径" validate:"required"`
	WorkingDir      string `json:"working_dir" gorm:"size:500;comment:工作目录"`
	CommandTemplate string `json:"command_template" gorm:"type:text;comment:命令模板，支持变量替换" validate:"required"`

	// 参数配置 - 使用JSON存储，灵活且易于扩展
	DefaultParams string `json:"default_params" gorm:"type:json;comment:默认参数配置"`
	ParamSchema   string `json:"param_schema" gorm:"type:json;comment:参数模式定义"`

	// 输出配置
	OutputFormat  string `json:"output_format" gorm:"size:50;default:'json';comment:输出格式:json,xml,text"`
	OutputParser  string `json:"output_parser" gorm:"size:100;comment:输出解析器名称"`
	ResultMapping string `json:"result_mapping" gorm:"type:json;comment:结果字段映射配置"`

	// 执行限制
	MaxExecutionTime int  `json:"max_execution_time" gorm:"default:3600;comment:最大执行时间(秒)" validate:"min=30,max=86400"`
	MaxMemoryMB      int  `json:"max_memory_mb" gorm:"default:1024;comment:最大内存使用(MB)" validate:"min=64,max=8192"`
	RequiresSudo     bool `json:"requires_sudo" gorm:"default:false;comment:是否需要sudo权限"`

	// 状态管理
	Status    ScanToolStatus `json:"status" gorm:"default:0;comment:工具状态:0-禁用,1-启用,2-测试中"`
	IsBuiltIn bool           `json:"is_built_in" gorm:"default:false;comment:是否为内置工具"`

	// 兼容性信息
	SupportedOS  string `json:"supported_os" gorm:"size:200;comment:支持的操作系统，逗号分隔"`
	Dependencies string `json:"dependencies" gorm:"type:text;comment:依赖项说明"`
	InstallGuide string `json:"install_guide" gorm:"type:text;comment:安装指南"`

	// 元数据
	Tags     string `json:"tags" gorm:"type:text;comment:工具标签，逗号分隔"`
	Metadata string `json:"metadata" gorm:"type:json;comment:扩展元数据"`

	// 审计字段
	CreatedBy uint64 `json:"created_by" gorm:"comment:创建者ID"`
	UpdatedBy uint64 `json:"updated_by" gorm:"comment:更新者ID"`

	// 统计信息
	UsageCount   int64 `json:"usage_count" gorm:"default:0;comment:使用次数"`
	SuccessCount int64 `json:"success_count" gorm:"default:0;comment:成功次数"`
	FailureCount int64 `json:"failure_count" gorm:"default:0;comment:失败次数"`
}

// TableName 定义数据库表名
func (ScanTool) TableName() string {
	return "scan_tools"
}

// IsEnabled 检查工具是否启用
func (st *ScanTool) IsEnabled() bool {
	return st.Status == ScanToolStatusEnabled
}

// CanExecute 检查工具是否可以执行
func (st *ScanTool) CanExecute() bool {
	return st.IsEnabled() && st.ExecutablePath != "" && st.CommandTemplate != ""
}

// GetDefaultParamsMap 获取默认参数映射
// 将JSON字符串转换为map，便于参数处理
func (st *ScanTool) GetDefaultParamsMap() (map[string]interface{}, error) {
	if st.DefaultParams == "" {
		return make(map[string]interface{}), nil
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(st.DefaultParams), &params); err != nil {
		return nil, fmt.Errorf("解析默认参数失败: %w", err)
	}
	return params, nil
}

// GetParamSchemaMap 获取参数模式映射
func (st *ScanTool) GetParamSchemaMap() (map[string]interface{}, error) {
	if st.ParamSchema == "" {
		return make(map[string]interface{}), nil
	}

	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(st.ParamSchema), &schema); err != nil {
		return nil, fmt.Errorf("解析参数模式失败: %w", err)
	}
	return schema, nil
}

// GetResultMappingMap 获取结果映射配置
func (st *ScanTool) GetResultMappingMap() (map[string]interface{}, error) {
	if st.ResultMapping == "" {
		return make(map[string]interface{}), nil
	}

	var mapping map[string]interface{}
	if err := json.Unmarshal([]byte(st.ResultMapping), &mapping); err != nil {
		return nil, fmt.Errorf("解析结果映射失败: %w", err)
	}
	return mapping, nil
}

// GetSupportedOSList 获取支持的操作系统列表
func (st *ScanTool) GetSupportedOSList() []string {
	if st.SupportedOS == "" {
		return []string{}
	}

	osList := make([]string, 0)
	for _, os := range strings.Split(st.SupportedOS, ",") {
		if trimmed := strings.TrimSpace(os); trimmed != "" {
			osList = append(osList, trimmed)
		}
	}
	return osList
}

// GetTagList 获取标签列表
func (st *ScanTool) GetTagList() []string {
	if st.Tags == "" {
		return []string{}
	}

	tags := make([]string, 0)
	for _, tag := range strings.Split(st.Tags, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

// BuildCommand 构建执行命令
// 简单的模板替换，避免复杂的模板引擎
func (st *ScanTool) BuildCommand(params map[string]interface{}) string {
	command := st.CommandTemplate

	// 简单的变量替换逻辑
	for key, value := range params {
		placeholder := fmt.Sprintf("{{%s}}", key)
		command = strings.ReplaceAll(command, placeholder, fmt.Sprintf("%v", value))
	}

	return command
}

// UpdateUsageStats 更新使用统计
// 简单的计数器更新，避免复杂的统计逻辑
func (st *ScanTool) UpdateUsageStats(success bool) {
	st.UsageCount++
	if success {
		st.SuccessCount++
	} else {
		st.FailureCount++
	}
}

// GetSuccessRate 获取成功率
func (st *ScanTool) GetSuccessRate() float64 {
	if st.UsageCount == 0 {
		return 0.0
	}
	return float64(st.SuccessCount) / float64(st.UsageCount) * 100
}
