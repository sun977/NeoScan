package parser

// ScanResult 代表通用的扫描结果
// 这是一个中间格式，不依赖于具体的业务 Model
type ScanResult struct {
	ToolName  string                   `json:"tool_name"`
	Target    string                   `json:"target"`
	Timestamp int64                    `json:"timestamp"`
	Data      []map[string]interface{} `json:"data"` // 原始结果列表
	Error     string                   `json:"error,omitempty"`
}

// Parser 接口定义了所有工具解析器必须实现的方法
// 遵循 "Small Interfaces" 原则
type Parser interface {
	// Parse 将工具的原始输出（通常是文本或JSON字符串）解析为结构化数据
	Parse(output string) (*ScanResult, error)
}
