package orchestrator

import (
	"neomaster/internal/model/basemodel"
)

// ScanStage 扫描阶段定义表
// 定义工作流中的具体执行步骤，包含工具配置和策略
// StageType 阶段类型有枚举定义
// ScanStage(定义) -- AgentTask(执行)[下发给Agent节点] -- ScanResult(结果)[Agent节点返回]
type ScanStage struct {
	basemodel.BaseModel

	WorkflowID uint64 `json:"workflow_id" gorm:"index;not null;comment:所属工作流ID"`
	StageName  string `json:"stage_name" gorm:"size:100;comment:阶段名称"`
	StageType  string `json:"stage_type" gorm:"size:50;comment:阶段类型枚举(ipAlive/serviceScan/PocScan等)"`

	// DAG 核心字段
	Predecessors []uint64 `json:"predecessors" gorm:"serializer:json;type:json;comment:前置依赖阶段ID列表(JSON数组),为空表示起始节点"`

	// UI/低代码 专用字段
	UIConfig map[string]interface{} `json:"ui_config" gorm:"serializer:json;type:json;comment:前端UI布局配置(JSON),包含x,y坐标等"`

	ToolName            string `json:"tool_name" gorm:"size:100;comment:使用的扫描工具名称"`
	ToolParams          string `json:"tool_params" gorm:"type:text;comment:扫描工具参数"`
	TargetPolicy        string `json:"target_policy" gorm:"type:json;comment:目标策略配置(JSON)"`        // 包含目标获取方式,白名单策略和跳过策略
	ExecutionPolicy     string `json:"execution_policy" gorm:"type:json;comment:执行策略配置(JSON)"`     // 包含代理配置和优先级,决定任务的属性
	PerformanceSettings string `json:"performance_settings" gorm:"type:json;comment:性能设置配置(JSON)"` // 包含并发数,超时时间等
	OutputConfig        string `json:"output_config" gorm:"type:json;comment:输出配置(JSON)"`          // 包含结果输出方式,是否输出到下一阶段,是否输出到数据库,是否输出到文件
	NotifyConfig        string `json:"notify_config" gorm:"type:json;comment:通知配置(JSON)"`
	Enabled             bool   `json:"enabled" gorm:"default:true;comment:阶段是否启用"`
}

// TableName 定义数据库表名
func (ScanStage) TableName() string {
	return "scan_stages"
}

// TargetPolicy 样例
// {
//   "target_policy": { // 扫描阶段的 target_policy 是一个 JSON 字符串，包含白名单、跳过条件等策略配置
//     "target_sources": [
//       {
//         "source_type": "file", // 来源类型：file/db/view/sql/manual/api/previous_stage【上一个阶段结果】
//         "source_value": "/path/to/targets.txt",
//         "target_type": "ip_range" // 目标类型：ip/ip_range/domain/url
//       }
//     ],
//     "whitelist_enabled": true,
//     "whitelist_sources": [ // 白名单来源/文件/手动输入
//       {
//         "source_type": "file",
//         "source_value": "/path/to/whitelist.txt" // file 对应文件路径, manual 对应手动输入内容["192.168.1.0/24","10.0.0.0/16"]
//       }
//     ],
//     "skip_enabled": true,
//     "skip_conditions": [  // 跳过条件,列表中可添加多个条件，这里写的条件直接执行跳过动作
//       {
//         "condition_field": "device_type",
//         "operator": "equals",
//         "value": "honeypot"
//       },
//       {
//         "condition_field": "os",
//         "operator": "contains",
//         "value": "linux"
//       }
//     ]
//   }
// }

// TargetPolicy 定义目标策略配置
type TargetPolicy struct {
	TargetSources    []TargetSource    `json:"target_sources"`    // 目标来源配置
	WhitelistEnabled bool              `json:"whitelist_enabled"` // 是否启用白名单
	WhitelistSources []WhitelistSource `json:"whitelist_sources"` // 白名单来源配置
	SkipEnabled      bool              `json:"skip_enabled"`      // 是否启用跳过条件
	SkipConditions   []SkipCondition   `json:"skip_conditions"`   // 跳过条件配置
}

// TargetSource 定义目标来源
type TargetSource struct {
	SourceType  string `json:"source_type"`  // 来源类型：file/db/view/sql/manual/api/previous_stage
	SourceValue string `json:"source_value"` // 来源值
	TargetType  string `json:"target_type"`  // 目标类型：ip/ip_range/domain/url
}

// WhitelistSource 定义白名单来源
type WhitelistSource struct {
	SourceType  string `json:"source_type"`  // 来源类型：file/db/manual
	SourceValue string `json:"source_value"` // 来源值
}

// SkipCondition 定义跳过条件
type SkipCondition struct {
	ConditionField string `json:"condition_field"` // 条件字段
	Operator       string `json:"operator"`        // 操作符
	Value          string `json:"value"`           // 值
}

// ExecutionPolicy 样例
// 	{
//   "proxy_config": {                    // 代理配置
//     "enabled": true,
//     "proxy_type": "http",             // http/https/socks4/socks5
//     "address": "proxy.example.com",
//     "port": 8080,
//     "username": "user",
//     "password": "pass"
//   },
//   "priority": 1,                       // 任务优先级（1-10，默认5） 优先级越高，越先被执行
// }

// ExecutionPolicy 执行策略结构
type ExecutionPolicy struct {
	ProxyConfig ProxyConfig `json:"proxy_config"` // 代理配置
	Priority    int         `json:"priority"`     // 任务优先级（1-10，默认5） 优先级越高，越先被执行
}

// ProxyConfig 定义代理配置
type ProxyConfig struct {
	Enabled   bool   `json:"enabled"`    // 是否启用代理
	ProxyType string `json:"proxy_type"` // 代理类型：http/https/socks4/socks5
	Address   string `json:"address"`    // 代理地址
	Port      int    `json:"port"`       // 代理端口
	Username  string `json:"username"`   // 代理用户名
	Password  string `json:"password"`   // 代理密码
}

// PerformanceSettings 样例
// 	{
//   "scan_rate": 50,        // 扫描速率（每秒发包数）
//   "scan_depth": 1,        // 扫描深度（爬虫类工具参数）
//   "concurrency": 50,      // 扫描并发数
//   "process_count": 50,    // 扫描进程数
//   "chunk_size": 50,       // 分块大小（批量处理目标数） --- 编排器-任务生成器-使用
//   "timeout": 180,          // 超时时间（秒） --- 编排器-任务生成器-使用
//   "retry_count": 3         // 重试次数
// }

// PerformanceSettings 性能设置结构
type PerformanceSettings struct {
	ScanRate     int `json:"scan_rate"`     // 扫描速率（每秒发包数）
	ScanDepth    int `json:"scan_depth"`    // 扫描深度（爬虫类工具参数）
	Concurrency  int `json:"concurrency"`   // 扫描并发数
	ProcessCount int `json:"process_count"` // 扫描进程数
	ChunkSize    int `json:"chunk_size"`    // 分块大小（批量处理目标数） --- 编排器-任务生成器-使用
	Timeout      int `json:"timeout"`       // 超时时间（秒） --- 编排器-任务生成器-使用
	RetryCount   int `json:"retry_count"`   // 重试次数
}

// OutputConfig 样例
// {
//   "output_to_next_stage": { // 输出到下一阶段
//     "enabled": true,
//     "output_fields": ["ip", "port", "service"]
//   },
//   "save_to_database": { // 保存到数据库
//     "enabled": true,
//     "save_type": "stage_result", // stage_result/final_asset/extract_fields
//     "table_name": "stage_results",
//     "extract_fields": {
//       "fields": ["target_value", "result_type"],
//       "target_table": "custom_scanned_hosts",
//       "field_mapping": {
//         "target_value": "ip_address"
//       }
//     },
//     "retention_days": 30
//   },
//   "save_to_file": { // 保存到文件
//     "enabled": true,
//     "file_path": "/var/scan/results/",
//     "file_format": "json", // json/xml/csv/html/markdown/text
//     "retention_days": 7
//   }
// }

// OutputConfig 定义输出配置结构
type OutputConfig struct {
	OutputToNextStage OutputToNextStageConfig `json:"output_to_next_stage,omitempty"` // 输出到下一阶段
	SaveToDatabase    SaveToDatabaseConfig    `json:"save_to_database,omitempty"`     // 保存到数据库
	SaveToFile        SaveToFileConfig        `json:"save_to_file,omitempty"`         // 保存到文件
}

// OutputToNextStageConfig 定义输出到下一阶段的配置
type OutputToNextStageConfig struct {
	Enabled      bool     `json:"enabled"`       // 是否启用
	OutputFields []string `json:"output_fields"` // 输出字段列表
}

// SaveToDatabaseConfig 定义保存到数据库的配置
type SaveToDatabaseConfig struct {
	Enabled       bool          `json:"enabled"`                  // 是否启用
	SaveType      string        `json:"save_type"`                // 保存类型: stage_result/final_asset/extract_fields
	TableName     string        `json:"table_name,omitempty"`     // 表名
	ExtractFields ExtractFields `json:"extract_fields,omitempty"` // 提取字段配置
	RetentionDays int           `json:"retention_days,omitempty"` // 保留天数
}

// ExtractFields 定义提取字段配置
type ExtractFields struct {
	Fields       []string          `json:"fields"`        // 字段列表
	TargetTable  string            `json:"target_table"`  // 目标表
	FieldMapping map[string]string `json:"field_mapping"` // 字段映射
}

// SaveToFileConfig 定义保存到文件的配置
type SaveToFileConfig struct {
	Enabled       bool   `json:"enabled"`                  // 是否启用
	FilePath      string `json:"file_path"`                // 文件路径
	FileFormat    string `json:"file_format"`              // 文件格式: json/xml/csv/html/markdown/text
	RetentionDays int    `json:"retention_days,omitempty"` // 保留天数
}

// NotifyConfig 样例
// {
//   "enabled": false,                    // 是否发送通知
//   "notify_methods": ["email"],   // 通知方式：email/sec/wechat/websocket
//   "recipients": ["admin@example.com"], // 通知接收人
//   "message_template": "Stage {stage_name} completed with {result_count} findings"  // 通知模板
// }

// NotifyConfig 定义通知配置结构
type NotifyConfig struct {
	Enabled         bool     `json:"enabled"`          // 是否发送通知
	NotifyMethods   []string `json:"notify_methods"`   // 通知方式：email/sec/wechat/websocket
	Recipients      []string `json:"recipients"`       // 通知接收人
	MessageTemplate string   `json:"message_template"` // 通知模板
}
