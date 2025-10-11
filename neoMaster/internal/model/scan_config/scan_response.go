/**
 * 模型:扫描配置响应模型
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 扫描配置相关的响应结构体，遵循"Never break userspace"原则
 * @func: 扫描规则、扫描工具、项目配置等响应模型
 */
package scan_config

import (
	"time"
)

// ScanRuleResponse 扫描规则响应结构
type ScanRuleResponse struct {
	ID          uint                   `json:"id"`                    // 规则ID
	Name        string                 `json:"name"`                  // 规则名称
	Description string                 `json:"description"`           // 规则描述
	Type        ScanRuleType           `json:"type"`                  // 规则类型
	Category    string                 `json:"category"`              // 规则分类
	Severity    string                 `json:"severity"`              // 严重程度
	Config      map[string]interface{} `json:"config"`                // 规则配置
	Conditions  []RuleCondition        `json:"conditions"`            // 规则条件
	Actions     []RuleAction           `json:"actions"`               // 规则动作
	Tags        []string               `json:"tags"`                  // 标签
	IsBuiltIn   bool                   `json:"is_built_in"`           // 是否内置规则
	Priority    int                    `json:"priority"`              // 优先级
	Status      ScanRuleStatus         `json:"status"`                // 规则状态
	CreatedBy   uint64                 `json:"created_by"`            // 创建者ID
	UpdatedBy   uint64                 `json:"updated_by"`            // 更新者ID
	CreatedAt   time.Time              `json:"created_at"`            // 创建时间
	UpdatedAt   time.Time              `json:"updated_at"`            // 更新时间
}

// ScanRuleListResponse 扫描规则列表响应结构
type ScanRuleListResponse struct {
	Rules []ScanRuleResponse `json:"rules"`      // 规则列表
	Total int64              `json:"total"`      // 总数
	Page  int                `json:"page"`       // 当前页码
	Size  int                `json:"size"`       // 每页数量
}

// MatchScanRulesResponse 匹配扫描规则响应结构
type MatchScanRulesResponse struct {
	Rules       []ScanRuleResponse `json:"rules"`        // 匹配的规则列表
	Total       int                `json:"total"`        // 匹配的规则总数
	MatchedAt   time.Time          `json:"matched_at"`   // 匹配时间
	TargetType  string             `json:"target_type"`  // 目标类型
	ScanPhase   string             `json:"scan_phase"`   // 扫描阶段
	ProcessTime int64              `json:"process_time"` // 处理时间(毫秒)
}

// TestScanRuleResponse 测试扫描规则响应结构
type TestScanRuleResponse struct {
	RuleID      uint                   `json:"rule_id"`      // 规则ID
	RuleName    string                 `json:"rule_name"`    // 规则名称
	Matched     bool                   `json:"matched"`      // 是否匹配
	Result      map[string]interface{} `json:"result"`       // 测试结果
	Message     string                 `json:"message"`      // 结果消息
	ProcessTime int64                  `json:"process_time"` // 处理时间(毫秒)
	TestedAt    time.Time              `json:"tested_at"`    // 测试时间
}

// ImportScanRulesResponse 导入扫描规则响应结构
type ImportScanRulesResponse struct {
	Success     []string  `json:"success"`      // 成功导入的规则名称
	Failed      []string  `json:"failed"`       // 导入失败的规则名称
	Skipped     []string  `json:"skipped"`      // 跳过的规则名称
	Total       int       `json:"total"`        // 总规则数
	SuccessNum  int       `json:"success_num"`  // 成功数量
	FailedNum   int       `json:"failed_num"`   // 失败数量
	SkippedNum  int       `json:"skipped_num"`  // 跳过数量
	ImportedAt  time.Time `json:"imported_at"`  // 导入时间
	ProcessTime int64     `json:"process_time"` // 处理时间(毫秒)
}

// ExportScanRulesResponse 导出扫描规则响应结构
type ExportScanRulesResponse struct {
	Data        string    `json:"data"`         // 导出的规则数据
	Format      string    `json:"format"`       // 数据格式
	RuleCount   int       `json:"rule_count"`   // 规则数量
	ExportedAt  time.Time `json:"exported_at"`  // 导出时间
	ProcessTime int64     `json:"process_time"` // 处理时间(毫秒)
}

// ScanToolResponse 扫描工具响应结构
type ScanToolResponse struct {
	ID           uint                   `json:"id"`            // 工具ID
	Name         string                 `json:"name"`          // 工具名称
	Version      string                 `json:"version"`       // 工具版本
	Description  string                 `json:"description"`   // 工具描述
	Category     string                 `json:"category"`      // 工具分类
	InstallPath  string                 `json:"install_path"`  // 安装路径
	ExecutePath  string                 `json:"execute_path"`  // 执行路径
	Config       map[string]interface{} `json:"config"`        // 工具配置
	Dependencies []string               `json:"dependencies"`  // 依赖项
	Tags         []string               `json:"tags"`          // 标签
	IsBuiltIn    bool                   `json:"is_built_in"`   // 是否内置工具
	Status       string                 `json:"status"`        // 工具状态
	CreatedBy    uint64                 `json:"created_by"`    // 创建者ID
	UpdatedBy    uint64                 `json:"updated_by"`    // 更新者ID
	CreatedAt    time.Time              `json:"created_at"`    // 创建时间
	UpdatedAt    time.Time              `json:"updated_at"`    // 更新时间
}

// ScanToolListResponse 扫描工具列表响应结构
type ScanToolListResponse struct {
	Tools []ScanToolResponse `json:"tools"` // 工具列表
	Total int64              `json:"total"` // 总数
	Page  int                `json:"page"`  // 当前页码
	Size  int                `json:"size"`  // 每页数量
}

// BatchInstallScanToolsResponse 批量安装扫描工具响应结构
type BatchInstallScanToolsResponse struct {
	Success     []uint    `json:"success"`      // 成功安装的工具ID
	Failed      []uint    `json:"failed"`       // 安装失败的工具ID
	Skipped     []uint    `json:"skipped"`      // 跳过的工具ID
	Total       int       `json:"total"`        // 总工具数
	SuccessNum  int       `json:"success_num"`  // 成功数量
	FailedNum   int       `json:"failed_num"`   // 失败数量
	SkippedNum  int       `json:"skipped_num"`  // 跳过数量
	InstalledAt time.Time `json:"installed_at"` // 安装时间
	ProcessTime int64     `json:"process_time"` // 处理时间(毫秒)
}

// BatchUninstallScanToolsResponse 批量卸载扫描工具响应结构
type BatchUninstallScanToolsResponse struct {
	Success       []uint    `json:"success"`        // 成功卸载的工具ID
	Failed        []uint    `json:"failed"`         // 卸载失败的工具ID
	Skipped       []uint    `json:"skipped"`        // 跳过的工具ID
	Total         int       `json:"total"`          // 总工具数
	SuccessNum    int       `json:"success_num"`    // 成功数量
	FailedNum     int       `json:"failed_num"`     // 失败数量
	SkippedNum    int       `json:"skipped_num"`    // 跳过数量
	UninstalledAt time.Time `json:"uninstalled_at"` // 卸载时间
	ProcessTime   int64     `json:"process_time"`   // 处理时间(毫秒)
}

// SystemScanConfigResponse 系统扫描配置响应结构
type SystemScanConfigResponse struct {
	ConfigType  string                 `json:"config_type"`  // 配置类型
	Config      map[string]interface{} `json:"config"`       // 配置数据
	UpdatedBy   uint64                 `json:"updated_by"`   // 更新者ID
	UpdatedAt   time.Time              `json:"updated_at"`   // 更新时间
	Version     string                 `json:"version"`      // 配置版本
	Description string                 `json:"description"`  // 配置描述
}

// ProjectConfigResponse 项目配置响应结构
type ProjectConfigResponse struct {
	ID          uint                   `json:"id"`           // 配置ID
	ProjectID   uint64                 `json:"project_id"`   // 项目ID
	Name        string                 `json:"name"`         // 配置名称
	Description string                 `json:"description"`  // 配置描述
	Config      map[string]interface{} `json:"config"`       // 配置数据
	IsDefault   bool                   `json:"is_default"`   // 是否默认配置
	CreatedBy   uint64                 `json:"created_by"`   // 创建者ID
	UpdatedBy   uint64                 `json:"updated_by"`   // 更新者ID
	CreatedAt   time.Time              `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time              `json:"updated_at"`   // 更新时间
}

// ProjectConfigListResponse 项目配置列表响应结构
type ProjectConfigListResponse struct {
	Configs []ProjectConfigResponse `json:"configs"` // 配置列表
	Total   int64                   `json:"total"`   // 总数
	Page    int                     `json:"page"`    // 当前页码
	Size    int                     `json:"size"`    // 每页数量
}

// WorkflowResponse 工作流响应结构
type WorkflowResponse struct {
	ID          uint                   `json:"id"`           // 工作流ID
	Name        string                 `json:"name"`         // 工作流名称
	Description string                 `json:"description"`  // 工作流描述
	Config      map[string]interface{} `json:"config"`       // 工作流配置
	Steps       []WorkflowStep         `json:"steps"`        // 工作流步骤
	Tags        []string               `json:"tags"`         // 标签
	IsBuiltIn   bool                   `json:"is_built_in"`  // 是否内置工作流
	Status      string                 `json:"status"`       // 工作流状态
	CreatedBy   uint64                 `json:"created_by"`   // 创建者ID
	UpdatedBy   uint64                 `json:"updated_by"`   // 更新者ID
	CreatedAt   time.Time              `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time              `json:"updated_at"`   // 更新时间
}

// WorkflowListResponse 工作流列表响应结构
type WorkflowListResponse struct {
	Workflows []WorkflowResponse `json:"workflows"` // 工作流列表
	Total     int64              `json:"total"`     // 总数
	Page      int                `json:"page"`      // 当前页码
	Size      int                `json:"size"`      // 每页数量
}

// ScanRuleStatsResponse 扫描规则统计响应结构
type ScanRuleStatsResponse struct {
	RuleID       uint      `json:"rule_id"`        // 规则ID
	RuleName     string    `json:"rule_name"`      // 规则名称
	MatchCount   int64     `json:"match_count"`    // 匹配次数
	SuccessCount int64     `json:"success_count"`  // 成功次数
	FailureCount int64     `json:"failure_count"`  // 失败次数
	AvgTime      float64   `json:"avg_time"`       // 平均处理时间(毫秒)
	LastUsed     time.Time `json:"last_used"`      // 最后使用时间
	CreatedAt    time.Time `json:"created_at"`     // 创建时间
	UpdatedAt    time.Time `json:"updated_at"`     // 更新时间
}

// ScanRulePerformanceResponse 扫描规则性能响应结构
type ScanRulePerformanceResponse struct {
	RuleID      uint    `json:"rule_id"`       // 规则ID
	RuleName    string  `json:"rule_name"`     // 规则名称
	MinTime     int64   `json:"min_time"`      // 最小处理时间(毫秒)
	MaxTime     int64   `json:"max_time"`      // 最大处理时间(毫秒)
	AvgTime     float64 `json:"avg_time"`      // 平均处理时间(毫秒)
	TotalTime   int64   `json:"total_time"`    // 总处理时间(毫秒)
	CallCount   int64   `json:"call_count"`    // 调用次数
	SuccessRate float64 `json:"success_rate"`  // 成功率
	UpdatedAt   time.Time `json:"updated_at"`  // 更新时间
}