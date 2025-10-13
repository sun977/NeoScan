/**
 * 模型:扫描配置请求模型
 * @author: Linus-inspired AI
 * @date: 2025.10.11
 * @description: 扫描配置相关的请求结构体，遵循"Never break userspace"原则
 * @func: 扫描规则、扫描工具、项目配置等请求模型
 */
package orchestrator

// CreateScanRuleRequest 创建扫描规则请求结构
type CreateScanRuleRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=100"`                      // 规则名称，必填
	Description string                 `json:"description" validate:"max=500"`                              // 规则描述，可选
	Type        ScanRuleType           `json:"type" validate:"required"`                                    // 规则类型，必填
	Category    string                 `json:"category" validate:"required,min=1,max=50"`                   // 规则分类，必填
	Severity    string                 `json:"severity" validate:"required,oneof=low medium high critical"` // 严重程度，必填
	Config      map[string]interface{} `json:"config" validate:"required"`                                  // 规则配置，必填
	Conditions  []RuleCondition        `json:"conditions" validate:"required,min=1"`                        // 规则条件，必填
	Actions     []RuleAction           `json:"actions" validate:"required,min=1"`                           // 规则动作，必填
	Tags        []string               `json:"tags"`                                                        // 标签，可选
	IsBuiltIn   bool                   `json:"is_built_in"`                                                 // 是否内置规则，可选
	Priority    int                    `json:"priority" validate:"min=1,max=100"`                           // 优先级，可选
	Status      ScanRuleStatus         `json:"status"`                                                      // 规则状态，可选
}

// UpdateScanRuleRequest 更新扫描规则请求结构
type UpdateScanRuleRequest struct {
	Name        *string                `json:"name" validate:"omitempty,min=1,max=100"`                      // 规则名称，可选
	Description *string                `json:"description" validate:"omitempty,max=500"`                     // 规则描述，可选
	Type        *ScanRuleType          `json:"type"`                                                         // 规则类型，可选
	Category    *string                `json:"category" validate:"omitempty,min=1,max=50"`                   // 规则分类，可选
	Severity    *string                `json:"severity" validate:"omitempty,oneof=low medium high critical"` // 严重程度，可选
	Config      map[string]interface{} `json:"config"`                                                       // 规则配置，可选
	Conditions  []RuleCondition        `json:"conditions"`                                                   // 规则条件，可选
	Actions     []RuleAction           `json:"actions"`                                                      // 规则动作，可选
	Tags        []string               `json:"tags"`                                                         // 标签，可选
	Priority    *int                   `json:"priority" validate:"omitempty,min=1,max=100"`                  // 优先级，可选
	Status      *ScanRuleStatus        `json:"status"`                                                       // 规则状态，可选
}

// ListScanRulesRequest 获取扫描规则列表请求结构
type ListScanRulesRequest struct {
	Page      int             `json:"page" validate:"min=1"`              // 页码，默认1
	PageSize  int             `json:"page_size" validate:"min=1,max=100"` // 每页数量，默认10
	Type      *ScanRuleType   `json:"type"`                               // 规则类型过滤，可选
	Status    *ScanRuleStatus `json:"status"`                             // 状态过滤，可选
	Severity  *string         `json:"severity"`                           // 严重程度过滤，可选
	Category  *string         `json:"category"`                           // 规则分类过滤，可选
	Keyword   *string         `json:"keyword"`                            // 关键词搜索，可选
	IsBuiltIn *bool           `json:"is_built_in"`                        // 是否内置规则过滤，可选
	CreatedBy *uint64         `json:"created_by"`                         // 创建者过滤，可选
}

// MatchScanRulesRequest 匹配扫描规则请求结构
type MatchScanRulesRequest struct {
	TargetType  string                 `json:"target_type" validate:"required"`    // 目标类型，必填
	ScanPhase   string                 `json:"scan_phase" validate:"required"`     // 扫描阶段，必填
	ScanTool    string                 `json:"scan_tool"`                          // 扫描工具，可选
	RuleType    *ScanRuleType          `json:"rule_type"`                          // 规则类型过滤，可选
	TargetData  map[string]interface{} `json:"target_data" validate:"required"`    // 目标数据，必填
	Context     map[string]interface{} `json:"context"`                            // 上下文数据，可选
	MaxRules    int                    `json:"max_rules" validate:"min=1,max=100"` // 最大返回规则数，可选
	OnlyEnabled bool                   `json:"only_enabled"`                       // 只返回启用的规则，可选
}

// ImportScanRulesRequest 导入扫描规则请求结构
type ImportScanRulesRequest struct {
	Data      string `json:"data" validate:"required"`                       // 规则数据，必填
	Format    string `json:"format" validate:"required,oneof=json yaml xml"` // 数据格式，必填
	Overwrite bool   `json:"overwrite"`                                      // 是否覆盖同名规则，可选
	Validate  bool   `json:"validate"`                                       // 是否验证规则，可选
}

// ExportScanRulesRequest 导出扫描规则请求结构
type ExportScanRulesRequest struct {
	RuleType *ScanRuleType   `json:"rule_type"`                                      // 规则类型过滤，可选
	Status   *ScanRuleStatus `json:"status"`                                         // 状态过滤，可选
	Format   string          `json:"format" validate:"required,oneof=json yaml xml"` // 导出格式，必填
}

// TestScanRuleRequest 测试扫描规则请求结构
type TestScanRuleRequest struct {
	RuleID uint                   `json:"rule_id" validate:"required"` // 规则ID，必填
	Target map[string]interface{} `json:"target" validate:"required"`  // 测试目标数据，必填
}

// CreateScanToolRequest 创建扫描工具请求结构
type CreateScanToolRequest struct {
	Name         string                 `json:"name" validate:"required,min=1,max=100"`    // 工具名称，必填
	Version      string                 `json:"version" validate:"required,min=1,max=50"`  // 工具版本，必填
	Description  string                 `json:"description" validate:"max=500"`            // 工具描述，可选
	Category     string                 `json:"category" validate:"required,min=1,max=50"` // 工具分类，必填
	InstallPath  string                 `json:"install_path" validate:"required"`          // 安装路径，必填
	ExecutePath  string                 `json:"execute_path" validate:"required"`          // 执行路径，必填
	Config       map[string]interface{} `json:"config"`                                    // 工具配置，可选
	Dependencies []string               `json:"dependencies"`                              // 依赖项，可选
	Tags         []string               `json:"tags"`                                      // 标签，可选
	IsBuiltIn    bool                   `json:"is_built_in"`                               // 是否内置工具，可选
}

// UpdateScanToolRequest 更新扫描工具请求结构
type UpdateScanToolRequest struct {
	Name         *string                `json:"name" validate:"omitempty,min=1,max=100"`    // 工具名称，可选
	Version      *string                `json:"version" validate:"omitempty,min=1,max=50"`  // 工具版本，可选
	Description  *string                `json:"description" validate:"omitempty,max=500"`   // 工具描述，可选
	Category     *string                `json:"category" validate:"omitempty,min=1,max=50"` // 工具分类，可选
	InstallPath  *string                `json:"install_path"`                               // 安装路径，可选
	ExecutePath  *string                `json:"execute_path"`                               // 执行路径，可选
	Config       map[string]interface{} `json:"config"`                                     // 工具配置，可选
	Dependencies []string               `json:"dependencies"`                               // 依赖项，可选
	Tags         []string               `json:"tags"`                                       // 标签，可选
}

// ListScanToolsRequest 获取扫描工具列表请求结构
type ListScanToolsRequest struct {
	Page      int     `json:"page" validate:"min=1"`              // 页码，默认1
	PageSize  int     `json:"page_size" validate:"min=1,max=100"` // 每页数量，默认10
	Category  *string `json:"category"`                           // 工具分类过滤，可选
	Status    *string `json:"status"`                             // 状态过滤，可选
	Keyword   *string `json:"keyword"`                            // 关键词搜索，可选
	IsBuiltIn *bool   `json:"is_built_in"`                        // 是否内置工具过滤，可选
}

// BatchInstallScanToolsRequest 批量安装扫描工具请求结构
type BatchInstallScanToolsRequest struct {
	ToolIDs []uint `json:"tool_ids" validate:"required,min=1"` // 工具ID列表，必填
	Force   bool   `json:"force"`                              // 是否强制安装，可选
}

// BatchUninstallScanToolsRequest 批量卸载扫描工具请求结构
type BatchUninstallScanToolsRequest struct {
	ToolIDs []uint `json:"tool_ids" validate:"required,min=1"` // 工具ID列表，必填
	Force   bool   `json:"force"`                              // 是否强制卸载，可选
}

// GetSystemScanConfigRequest 获取系统扫描配置请求结构
type GetSystemScanConfigRequest struct {
	ConfigType string `json:"config_type" validate:"required"` // 配置类型，必填
}

// UpdateSystemScanConfigRequest 更新系统扫描配置请求结构
type UpdateSystemScanConfigRequest struct {
	ConfigType string                 `json:"config_type" validate:"required"` // 配置类型，必填
	Config     map[string]interface{} `json:"config" validate:"required"`      // 配置数据，必填
}

// CreateProjectConfigRequest 创建项目配置请求结构
type CreateProjectConfigRequest struct {
	ProjectID   uint64                 `json:"project_id" validate:"required"`         // 项目ID，必填
	Name        string                 `json:"name" validate:"required,min=1,max=100"` // 配置名称，必填
	Description string                 `json:"description" validate:"max=500"`         // 配置描述，可选
	Config      map[string]interface{} `json:"config" validate:"required"`             // 配置数据，必填
	IsDefault   bool                   `json:"is_default"`                             // 是否默认配置，可选
}

// UpdateProjectConfigRequest 更新项目配置请求结构
type UpdateProjectConfigRequest struct {
	Name        *string                `json:"name" validate:"omitempty,min=1,max=100"`  // 配置名称，可选
	Description *string                `json:"description" validate:"omitempty,max=500"` // 配置描述，可选
	Config      map[string]interface{} `json:"config"`                                   // 配置数据，可选
	IsDefault   *bool                  `json:"is_default"`                               // 是否默认配置，可选
}

// ListProjectConfigsRequest 获取项目配置列表请求结构
type ListProjectConfigsRequest struct {
	ProjectID uint64  `json:"project_id" validate:"required"`     // 项目ID，必填
	Page      int     `json:"page" validate:"min=1"`              // 页码，默认1
	PageSize  int     `json:"page_size" validate:"min=1,max=100"` // 每页数量，默认10
	Keyword   *string `json:"keyword"`                            // 关键词搜索，可选
	IsDefault *bool   `json:"is_default"`                         // 是否默认配置过滤，可选
}

// CreateWorkflowRequest 创建工作流请求结构
type CreateWorkflowRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=100"` // 工作流名称，必填
	Description string                 `json:"description" validate:"max=500"`         // 工作流描述，可选
	Config      map[string]interface{} `json:"config" validate:"required"`             // 工作流配置，必填
	Steps       []WorkflowStep         `json:"steps" validate:"required,min=1"`        // 工作流步骤，必填
	Tags        []string               `json:"tags"`                                   // 标签，可选
	IsBuiltIn   bool                   `json:"is_built_in"`                            // 是否内置工作流，可选
}

// UpdateWorkflowRequest 更新工作流请求结构
type UpdateWorkflowRequest struct {
	Name        *string                `json:"name" validate:"omitempty,min=1,max=100"`  // 工作流名称，可选
	Description *string                `json:"description" validate:"omitempty,max=500"` // 工作流描述，可选
	Config      map[string]interface{} `json:"config"`                                   // 工作流配置，可选
	Steps       []WorkflowStep         `json:"steps"`                                    // 工作流步骤，可选
	Tags        []string               `json:"tags"`                                     // 标签，可选
}

// ListWorkflowsRequest 获取工作流列表请求结构
type ListWorkflowsRequest struct {
	Page      int     `json:"page" validate:"min=1"`              // 页码，默认1
	PageSize  int     `json:"page_size" validate:"min=1,max=100"` // 每页数量，默认10
	Status    *string `json:"status"`                             // 状态过滤，可选
	Keyword   *string `json:"keyword"`                            // 关键词搜索，可选
	IsBuiltIn *bool   `json:"is_built_in"`                        // 是否内置工作流过滤，可选
}
