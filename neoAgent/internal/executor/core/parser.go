package core

// Parser 接口定义了所有工具解析器必须实现的方法
// 遵循 "Small Interfaces" 原则
type Parser interface {
	// Parse 将工具的原始输出（通常是文本或JSON字符串）解析为结构化数据
	Parse(output string) (*ToolScanResult, error)
}
