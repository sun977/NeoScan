/**
 * Nuclei扫描工具执行器实现
 * @author: Sun977
 * @date: 2025.10.11
 * @description: Nuclei扫描工具的具体执行器实现，支持漏洞扫描和模板执行
 * @func: 执行nuclei命令，解析输出结果，监控执行状态
 */
package executor_drop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"neomaster/internal/model/orchestrator_model_drop"
	"neomaster/internal/pkg/logger"
)

// NucleiExecutor Nuclei执行器实现
type NucleiExecutor struct {
	name    string // 执行器名称
	version string // 执行器版本
}

// NewNucleiExecutor 创建新的Nuclei执行器
func NewNucleiExecutor() *NucleiExecutor {
	return &NucleiExecutor{
		name:    "nuclei_executor",
		version: "1.0.0",
	}
}

// GetName 获取执行器名称
func (e *NucleiExecutor) GetName() string {
	return e.name
}

// GetVersion 获取执行器版本
func (e *NucleiExecutor) GetVersion() string {
	return e.version
}

// GetSupportedTools 获取支持的扫描工具列表
func (e *NucleiExecutor) GetSupportedTools() []string {
	return []string{"nuclei"}
}

// IsToolSupported 检查是否支持指定工具
func (e *NucleiExecutor) IsToolSupported(toolName string) bool {
	return strings.ToLower(toolName) == "nuclei"
}

// ValidateConfig 验证工具配置是否正确
func (e *NucleiExecutor) ValidateConfig(tool *orchestrator_model_drop.ScanTool) error {
	if tool == nil {
		return fmt.Errorf("scan tool cannot be nil")
	}

	if !e.IsToolSupported(tool.Name) {
		return fmt.Errorf("tool %s is not supported by nuclei executor", tool.Name)
	}

	// 检查执行路径是否存在
	if tool.ExecutablePath == "" {
		return fmt.Errorf("execute path cannot be empty")
	}

	// 检查nuclei可执行文件是否存在
	if _, err := os.Stat(tool.ExecutablePath); os.IsNotExist(err) {
		return fmt.Errorf("nuclei executable not found at path: %s", tool.ExecutablePath)
	}

	// 验证nuclei版本
	if err := e.validateNucleiVersion(tool.ExecutablePath); err != nil {
		return fmt.Errorf("nuclei validation failed: %w", err)
	}

	return nil
}

// validateNucleiVersion 验证nuclei版本
func (e *NucleiExecutor) validateNucleiVersion(nucleiPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, nucleiPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get nuclei version: %w", err)
	}

	versionOutput := string(output)
	if !strings.Contains(versionOutput, "nuclei") {
		return fmt.Errorf("invalid nuclei executable, version output: %s", versionOutput)
	}

	// 使用 WithFields 记录调试日志
	logger.WithFields(map[string]interface{}{
		"path":      "executor.nuclei.validateNucleiVersion",
		"operation": "validate_nuclei",
		"option":    "version_check",
		"func_name": "executor.nuclei.validateNucleiVersion",
		"version":   strings.TrimSpace(versionOutput),
	}).Debug("Nuclei version validated")

	return nil
}

// Execute 执行扫描任务
func (e *NucleiExecutor) Execute(ctx context.Context, request *ScanRequest) (*ScanResult, error) {
	if request == nil {
		return nil, fmt.Errorf("scan request cannot be nil")
	}

	startTime := time.Now()

	// 使用 WithFields 记录信息日志
	logger.WithFields(map[string]interface{}{
		"path":      "executor.nuclei.Execute",
		"operation": "execute_nuclei",
		"option":    "start_execution",
		"func_name": "executor.nuclei.Execute",
		"task_id":   request.TaskID,
		"target":    request.Target,
	}).Info("Starting nuclei scan")

	// 构建nuclei命令
	args, err := e.buildNucleiArgs(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build nuclei args: %w", err)
	}

	// 创建输出目录
	if request.OutputPath != "" {
		outputDir := filepath.Dir(request.OutputPath)
		if err1 := os.MkdirAll(outputDir, 0755); err1 != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err1)
		}
	}

	// 执行nuclei命令
	cmd := exec.CommandContext(ctx, request.Tool.ExecutablePath, args...)

	// 设置工作目录
	if request.WorkingDir != "" {
		cmd.Dir = request.WorkingDir
	}

	// 设置环境变量
	if len(request.Environment) > 0 {
		env := os.Environ()
		for key, value := range request.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	result := &ScanResult{
		TaskID:    request.TaskID,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
		Output:    string(output),
	}

	// 获取退出码
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	// 检查执行结果
	if err != nil {
		result.Status = ScanTaskStatusFailed
		result.Error = err.Error()

		// 使用 WithFields 记录错误日志
		logger.WithFields(map[string]interface{}{
			"path":      "executor.nuclei.Execute",
			"operation": "execute_nuclei",
			"option":    "execution_failed",
			"func_name": "executor.nuclei.Execute",
			"task_id":   request.TaskID,
		}).WithError(err).Error("Nuclei execution failed")

		return result, nil // 返回结果而不是错误，让调用者处理
	}

	result.Status = ScanTaskStatusCompleted

	// 收集输出文件
	if request.OutputPath != "" {
		outputFiles, err1 := e.collectOutputFiles(request.OutputPath)
		if err1 != nil {
			// 使用 WithFields 记录警告日志
			logger.WithFields(map[string]interface{}{
				"path":      "executor.nuclei.Execute",
				"operation": "execute_nuclei",
				"option":    "collect_output_files_failed",
				"func_name": "executor.nuclei.Execute",
				"task_id":   request.TaskID,
			}).WithError(err1).Warn("Failed to collect output files")
		} else {
			result.OutputFiles = outputFiles
		}
	}

	// 解析扫描结果元数据
	metadata, err := e.parseNucleiOutput(string(output))
	if err != nil {
		// 使用 WithFields 记录警告日志
		logger.WithFields(map[string]interface{}{
			"path":      "executor.nuclei.Execute",
			"operation": "execute_nuclei",
			"option":    "parse_output_failed",
			"func_name": "executor.nuclei.Execute",
			"task_id":   request.TaskID,
		}).WithError(err).Warn("Failed to parse nuclei output")
	} else {
		result.Metadata = metadata
	}

	// 使用 WithFields 记录信息日志
	logger.WithFields(map[string]interface{}{
		"path":      "executor.nuclei.Execute",
		"operation": "execute_nuclei",
		"option":    "execution_completed",
		"func_name": "executor.nuclei.Execute",
		"task_id":   request.TaskID,
		"duration":  duration,
	}).Info("Nuclei scan completed")

	return result, nil
}

// buildNucleiArgs 构建nuclei命令参数
func (e *NucleiExecutor) buildNucleiArgs(request *ScanRequest) ([]string, error) {
	args := []string{}

	// 添加目标
	args = append(args, "-target", request.Target)

	// 处理扫描选项
	if request.Options != nil {
		// 模板路径或标签
		if templates, ok := request.Options["templates"].(string); ok && templates != "" {
			args = append(args, "-t", templates)
		} else if templateTags, ok := request.Options["template_tags"].(string); ok && templateTags != "" {
			args = append(args, "-tags", templateTags)
		} else {
			// 默认使用所有模板
			args = append(args, "-t", ".")
		}

		// 严重程度过滤
		if severity, ok := request.Options["severity"].(string); ok && severity != "" {
			args = append(args, "-severity", severity)
		}

		// 排除模板
		if excludeTemplates, ok := request.Options["exclude_templates"].(string); ok && excludeTemplates != "" {
			args = append(args, "-exclude-templates", excludeTemplates)
		}

		// 排除标签
		if excludeTags, ok := request.Options["exclude_tags"].(string); ok && excludeTags != "" {
			args = append(args, "-exclude-tags", excludeTags)
		}

		// 并发数
		if concurrency, ok := request.Options["concurrency"].(string); ok && concurrency != "" {
			args = append(args, "-c", concurrency)
		} else if concurrencyInt, ok := request.Options["concurrency"].(int); ok {
			args = append(args, "-c", strconv.Itoa(concurrencyInt))
		}

		// 速率限制
		if rateLimit, ok := request.Options["rate_limit"].(string); ok && rateLimit != "" {
			args = append(args, "-rl", rateLimit)
		} else if rateLimitInt, ok := request.Options["rate_limit"].(int); ok {
			args = append(args, "-rl", strconv.Itoa(rateLimitInt))
		}

		// 超时设置
		if timeout, ok := request.Options["timeout"].(string); ok && timeout != "" {
			args = append(args, "-timeout", timeout)
		}

		// 重试次数
		if retries, ok := request.Options["retries"].(string); ok && retries != "" {
			args = append(args, "-retries", retries)
		} else if retriesInt, ok := request.Options["retries"].(int); ok {
			args = append(args, "-retries", strconv.Itoa(retriesInt))
		}

		// 输出格式
		if request.OutputPath != "" {
			// 默认输出JSON格式便于解析
			args = append(args, "-json", "-o", request.OutputPath)

			// 如果指定了其他格式
			if outputFormat, ok := request.Options["output_format"].(string); ok {
				switch outputFormat {
				case "sarif":
					args = append(args, "-sarif", strings.Replace(request.OutputPath, ".json", ".sarif", 1))
				case "markdown":
					args = append(args, "-markdown", strings.Replace(request.OutputPath, ".json", ".md", 1))
				}
			}
		}

		// 详细输出
		if verbose, ok := request.Options["verbose"].(bool); ok && verbose {
			args = append(args, "-v")
		}

		// 调试模式
		if debug, ok := request.Options["debug"].(bool); ok && debug {
			args = append(args, "-debug")
		}

		// 静默模式
		if silent, ok := request.Options["silent"].(bool); ok && silent {
			args = append(args, "-silent")
		}

		// 无颜色输出
		if noColor, ok := request.Options["no_color"].(bool); ok && noColor {
			args = append(args, "-no-color")
		}

		// 更新模板
		if updateTemplates, ok := request.Options["update_templates"].(bool); ok && updateTemplates {
			args = append(args, "-update-templates")
		}

		// 自定义头部
		if headers, ok := request.Options["headers"].([]string); ok && len(headers) > 0 {
			for _, header := range headers {
				args = append(args, "-H", header)
			}
		}

		// 用户代理
		if userAgent, ok := request.Options["user_agent"].(string); ok && userAgent != "" {
			args = append(args, "-user-agent", userAgent)
		}

		// 代理设置
		if proxy, ok := request.Options["proxy"].(string); ok && proxy != "" {
			args = append(args, "-proxy", proxy)
		}

		// 跟随重定向
		if followRedirects, ok := request.Options["follow_redirects"].(bool); ok && followRedirects {
			args = append(args, "-follow-redirects")
		}

		// 其他自定义参数
		if customArgs, ok := request.Options["custom_args"].([]string); ok {
			args = append(args, customArgs...)
		}
	} else {
		// 默认参数
		args = append(args, "-t", ".", "-c", "50")
	}

	return args, nil
}

// collectOutputFiles 收集输出文件
func (e *NucleiExecutor) collectOutputFiles(outputPath string) ([]string, error) {
	var files []string

	// 检查主输出文件
	if _, err := os.Stat(outputPath); err == nil {
		files = append(files, outputPath)
	}

	// 检查其他可能的输出文件
	baseDir := filepath.Dir(outputPath)
	baseName := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))

	possibleFiles := []string{
		filepath.Join(baseDir, baseName+".sarif"),
		filepath.Join(baseDir, baseName+".md"),
		filepath.Join(baseDir, baseName+".json"),
	}

	for _, file := range possibleFiles {
		if _, err := os.Stat(file); err == nil {
			files = append(files, file)
		}
	}

	return files, nil
}

// parseNucleiOutput 解析nuclei输出
func (e *NucleiExecutor) parseNucleiOutput(output string) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})

	lines := strings.Split(output, "\n")
	vulnerabilities := 0
	severityCount := make(map[string]int)

	// 解析基本信息
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析发现的漏洞
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			vulnerabilities++

			// 解析严重程度
			if strings.Contains(line, "[critical]") {
				severityCount["critical"]++
			} else if strings.Contains(line, "[high]") {
				severityCount["high"]++
			} else if strings.Contains(line, "[medium]") {
				severityCount["medium"]++
			} else if strings.Contains(line, "[low]") {
				severityCount["low"]++
			} else if strings.Contains(line, "[info]") {
				severityCount["info"]++
			}
		}

		// 解析模板统计
		if strings.Contains(line, "Templates loaded") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "Templates" && i+2 < len(parts) && parts[i+1] == "loaded:" {
					metadata["templates_loaded"] = parts[i+2]
					break
				}
			}
		}

		// 解析扫描统计
		if strings.Contains(line, "Targets loaded") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "Targets" && i+2 < len(parts) && parts[i+1] == "loaded:" {
					metadata["targets_loaded"] = parts[i+2]
					break
				}
			}
		}
	}

	metadata["vulnerabilities_count"] = vulnerabilities
	metadata["severity_breakdown"] = severityCount

	return metadata, nil
}

// Stop 停止正在执行的扫描任务
func (e *NucleiExecutor) Stop(ctx context.Context, taskID string) error {
	// Nuclei执行器本身不维护任务状态，依赖上下文取消
	// 使用 WithFields 记录信息日志
	logger.WithFields(map[string]interface{}{
		"path":      "executor.nuclei.Stop",
		"operation": "stop_nuclei",
		"option":    "stop_requested",
		"func_name": "executor.nuclei.Stop",
		"task_id":   taskID,
	}).Info("Stop requested for nuclei task")

	return nil
}

// GetStatus 获取扫描任务状态
func (e *NucleiExecutor) GetStatus(ctx context.Context, taskID string) (*ScanStatus, error) {
	// Nuclei执行器本身不维护任务状态，返回错误让管理器使用本地状态
	return nil, fmt.Errorf("nuclei executor does not maintain task status")
}

// Cleanup 清理资源
func (e *NucleiExecutor) Cleanup() error {
	// 使用 WithFields 记录信息日志
	logger.WithFields(map[string]interface{}{
		"path":      "executor.nuclei.Cleanup",
		"operation": "cleanup",
		"option":    "cleanup_completed",
		"func_name": "executor.nuclei.Cleanup",
	}).Info("Nuclei executor cleanup completed")

	return nil
}
