package orchestrator

import (
	"neomaster/internal/model/basemodel"
)

// ScanToolTemplate 扫描工具参数模板表
// 用于存储用户预定义的工具配置，方便快速创建 ScanStage
//
// 使用场景:
// 1. 保存模板: 用户配置好一个复杂的扫描参数(如: nmap -sS -T4 -p1-65535 --min-rate 1000)后，保存为模板
// 2. 使用模板: 用户创建 ScanStage 时，选择工具(如 nmap)，前端加载该工具的所有模板供选择
// 3. 自动填充: 用户选择模板后，系统自动将模板中的 ToolParams 等字段填充到 ScanStage 中
type ScanToolTemplate struct {
	basemodel.BaseModel

	Name        string `json:"name" gorm:"size:100;not null;comment:模板名称"`
	ToolName    string `json:"tool_name" gorm:"size:100;index;not null;comment:所属工具名称(nmap/masscan/nuclei等)"`
	ToolParams  string `json:"tool_params" gorm:"type:text;comment:工具命令行参数模板"`
	Description string `json:"description" gorm:"size:255;comment:模板描述/使用场景说明"`
	Category    string `json:"category" gorm:"size:50;comment:分类标签(quick/full/stealth等)"`
	IsPublic    bool   `json:"is_public" gorm:"default:false;comment:是否公开(系统预置或管理员共享)"`
	CreatedBy   string `json:"created_by" gorm:"size:50;comment:创建人"`
}

// TableName 定义数据库表名
func (ScanToolTemplate) TableName() string {
	return "scan_tool_templates"
}
