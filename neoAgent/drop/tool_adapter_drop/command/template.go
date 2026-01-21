package command

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// TemplateCommandBuilder 基于 Go Template 的通用命令构建器
// 适用于大多数参数结构固定的工具 (如 Masscan, HTTPX, Nuclei)
type TemplateCommandBuilder struct {
	BinaryPath   string             // 二进制文件路径 (可选，如果在 PATH 中)
	TemplateStr  string             // 命令模板字符串
	DefaultParams map[string]interface{} // 默认参数
}

// NewTemplateBuilder 创建一个新的模板构建器
func NewTemplateBuilder(binary string, tmpl string, defaults map[string]interface{}) *TemplateCommandBuilder {
	return &TemplateCommandBuilder{
		BinaryPath:    binary,
		TemplateStr:   tmpl,
		DefaultParams: defaults,
	}
}

// Build 根据目标和配置生成命令
// config 中的参数会覆盖 DefaultParams
// 模板上下文数据 = DefaultParams + config + {"Target": target, "Binary": binary}
func (b *TemplateCommandBuilder) Build(target string, config map[string]interface{}) (string, []string, error) {
	// 1. 准备模板数据
	data := make(map[string]interface{})
	
	// 填入默认参数
	for k, v := range b.DefaultParams {
		data[k] = v
	}
	
	// 填入运行时配置 (覆盖默认值)
	for k, v := range config {
		data[k] = v
	}

	// 填入核心参数
	data["Target"] = target
	
	// 如果未指定 BinaryPath，尝试从 config 中获取，或者默认为空(假设在模板里写死了或者在 PATH)
	binary := b.BinaryPath
	if val, ok := config["binary_path"].(string); ok && val != "" {
		binary = val
	}
	data["Binary"] = binary

	// 2. 解析模板
	// 使用 Option("missingkey=zero") 允许模板中引用不存在的参数(会被置为空值)，防止 panic
	tmpl, err := template.New("cmd").Option("missingkey=zero").Parse(b.TemplateStr)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse command template: %w", err)
	}

	// 3. 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", nil, fmt.Errorf("failed to execute command template: %w", err)
	}

	// 4. 解析结果为 args 切片
	// 简单的按空格分割是不够的，因为参数可能包含带空格的字符串 (e.g. --script-args "a=1 b=2")
	// 这里我们做一个简单的假设：模板生成的字符串就是完整的命令行
	// 如果需要精确的 args 切片用于 exec.Command，我们需要一个 shell token parser
	// 为了简化，我们暂时返回整个命令行字符串作为 args[0] (如果是 shell=true 模式)
	// 或者尝试简单的 split
	
	fullCmd := strings.TrimSpace(buf.String())
	
	// Linux/Unix 下通常直接返回 fullCmd 供 shell 执行
	// 或者手动 split
	args := strings.Fields(fullCmd)
	
	// 如果 Binary 已经包含在 fullCmd 中 (模板里写了 {{.Binary}})，则第一个就是 binary
	// 如果模板只是参数，则 binary 分离
	
	// 策略: 我们约定 TemplateStr 必须包含 {{.Binary}} 或者写死的 binary 名
	// 这样 fullCmd 就是完整的命令行
	
	// 提取 binary (第一个 token)
	if len(args) == 0 {
		return "", nil, fmt.Errorf("generated command is empty")
	}
	
	cmdBinary := args[0]
	cmdArgs := args[1:]

	return cmdBinary, cmdArgs, nil
}
