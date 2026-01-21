package masscan

import (
	"fmt"
	"strings"

	"neoagent/internal/executor/core"
)

// MasscanAdapter 实现 Masscan 的命令构建和结果解析
type MasscanAdapter struct {
}

// Ensure interface compliance
var _ core.CommandBuilder = (*MasscanAdapter)(nil)
var _ core.Parser = (*MasscanAdapter)(nil)

// Build 构建 Masscan 命令
func (a *MasscanAdapter) Build(target string, config map[string]interface{}) (string, []string, error) {
	// TODO: 实现参数拼接逻辑
	// 示例: masscan -p 80,443 192.168.1.1 --rate 1000 -oJ -
	args := []string{}

	// 处理端口
	if ports, ok := config["ports"].(string); ok {
		args = append(args, "-p", ports)
	}

	// 处理速率
	if rate, ok := config["rate"].(int); ok {
		args = append(args, "--rate", fmt.Sprintf("%d", rate))
	}

	// 强制输出为 JSON
	args = append(args, "-oJ", "-")

	// 目标
	args = append(args, target)

	return "masscan", args, nil
}

// Parse 解析 Masscan JSON 输出
func (a *MasscanAdapter) Parse(output string) (*core.ToolScanResult, error) {
	// Masscan 的 JSON 输出是一个对象列表，或者单个对象
	// 实际上 masscan -oJ 输出的是每个主机一个 JSON 对象，如果是多个主机，可能不是合法的 JSON 数组
	// 这里假设输出已经被处理成合法的 JSON 数组或单个对象

	// 这里的解析逻辑应该从原 tool_adapter/parser/masscan_json.go 迁移过来
	// 为了演示，这里写一个简单的占位逻辑

	result := &core.ToolScanResult{
		ToolName: "masscan",
		Status:   "success",
	}

	// 解析逻辑...
	// 如果 output 为空或非法，返回错误
	if strings.TrimSpace(output) == "" {
		return result, nil
	}

	// ...

	return result, nil
}
