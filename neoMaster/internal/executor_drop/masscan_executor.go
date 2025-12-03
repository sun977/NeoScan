/**
 * Masscan扫描工具执行器实现
 * @author: Sun977
 * @date: 2025.10.11
 * @description: Masscan扫描工具的具体执行器实现，支持高速端口扫描
 * @func: 执行masscan命令，解析输出结果，监控执行状态
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

	"neomaster/internal/model/orchestrator_drop"
	"neomaster/internal/pkg/logger"
)

// MasscanExecutor Masscan执行器实现
type MasscanExecutor struct {
	name    string // 执行器名称
	version string // 执行器版本
}

// NewMasscanExecutor 创建新的Masscan执行器
func NewMasscanExecutor() *MasscanExecutor {
	return &MasscanExecutor{
		name:    "masscan_executor",
		version: "1.0.0",
	}
}

// GetName 获取执行器名称
func (e *MasscanExecutor) GetName() string {
	return e.name
}

// GetVersion 获取执行器版本
func (e *MasscanExecutor) GetVersion() string {
	return e.version
}

// GetSupportedTools 获取支持的扫描工具列表
func (e *MasscanExecutor) GetSupportedTools() []string {
	return []string{"masscan"}
}

// IsToolSupported 检查是否支持指定工具
func (e *MasscanExecutor) IsToolSupported(toolName string) bool {
	return strings.ToLower(toolName) == "masscan"
}

// ValidateConfig 验证工具配置是否正确
func (e *MasscanExecutor) ValidateConfig(tool *orchestrator_drop.ScanTool) error {
	if tool == nil {
		return fmt.Errorf("scan tool cannot be nil")
	}

	if !e.IsToolSupported(tool.Name) {
		return fmt.Errorf("tool %s is not supported by masscan executor", tool.Name)
	}

	// 检查执行路径是否存在
	if tool.ExecutablePath == "" {
		return fmt.Errorf("execute path cannot be empty")
	}

	// 检查masscan可执行文件是否存在
	if _, err := os.Stat(tool.ExecutablePath); os.IsNotExist(err) {
		return fmt.Errorf("masscan executable not found at path: %s", tool.ExecutablePath)
	}

	// 验证masscan版本
	if err := e.validateMasscanVersion(tool.ExecutablePath); err != nil {
		return fmt.Errorf("masscan validation failed: %w", err)
	}

	return nil
}

// validateMasscanVersion 验证masscan版本
func (e *MasscanExecutor) validateMasscanVersion(masscanPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, masscanPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get masscan version: %w", err)
	}

	versionOutput := string(output)
	if !strings.Contains(versionOutput, "Masscan") {
		return fmt.Errorf("invalid masscan executable, version output: %s", versionOutput)
	}

	logger.WithFields(map[string]interface{}{
		"path":      "executor.masscan.validateMasscanVersion",
		"operation": "validate_masscan",
		"option":    "version_check",
		"func_name": "executor.masscan.validateMasscanVersion",
		"version":   strings.TrimSpace(versionOutput),
	}).Debug("Masscan version validation")

	return nil
}

// Execute 执行扫描任务
func (e *MasscanExecutor) Execute(ctx context.Context, request *ScanRequest) (*ScanResult, error) {
	if request == nil {
		return nil, fmt.Errorf("scan request cannot be nil")
	}

	startTime := time.Now()

	logger.WithFields(map[string]interface{}{
		"path":      "executor.masscan.Execute",
		"operation": "execute_masscan",
		"option":    "start_execution",
		"func_name": "executor.masscan.Execute",
		"task_id":   request.TaskID,
		"target":    request.Target,
	}).Info("Starting masscan scan")

	// 构建masscan命令
	args, err := e.buildMasscanArgs(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build masscan args: %w", err)
	}

	// 创建输出目录
	if request.OutputPath != "" {
		outputDir := filepath.Dir(request.OutputPath)
		if err1 := os.MkdirAll(outputDir, 0755); err1 != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err1)
		}
	}

	// 执行masscan命令
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

		logger.WithFields(map[string]interface{}{
			"path":      "executor.masscan.Execute",
			"operation": "execute_masscan",
			"option":    "execution_failed",
			"func_name": "executor.masscan.Execute",
			"task_id":   request.TaskID,
		}).WithError(err).Error("Masscan execution failed")

		return result, nil // 返回结果而不是错误，让调用者处理
	}

	result.Status = ScanTaskStatusCompleted

	// 收集输出文件
	if request.OutputPath != "" {
		outputFiles, err1 := e.collectOutputFiles(request.OutputPath)
		if err1 != nil {
			logger.WithFields(map[string]interface{}{
				"path":      "executor.masscan.Execute",
				"operation": "execute_masscan",
				"option":    "collect_output_files_failed",
				"func_name": "executor.masscan.Execute",
				"task_id":   request.TaskID,
			}).WithError(err1).Warn("Failed to collect output files")
		} else {
			result.OutputFiles = outputFiles
		}
	}

	// 解析扫描结果元数据
	metadata, err := e.parseMasscanOutput(string(output))
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"path":      "executor.masscan.Execute",
			"operation": "execute_masscan",
			"option":    "parse_output_failed",
			"func_name": "executor.masscan.Execute",
			"task_id":   request.TaskID,
		}).WithError(err).Warn("Failed to parse masscan output")
	} else {
		result.Metadata = metadata
	}

	logger.WithFields(map[string]interface{}{
		"path":      "executor.masscan.Execute",
		"operation": "execute_masscan",
		"option":    "execution_completed",
		"func_name": "executor.masscan.Execute",
		"task_id":   request.TaskID,
		"duration":  duration,
	}).Info("Masscan scan completed")

	return result, nil
}

// buildMasscanArgs 构建masscan命令参数
func (e *MasscanExecutor) buildMasscanArgs(request *ScanRequest) ([]string, error) {
	args := []string{}

	// 处理扫描选项
	if request.Options != nil {
		// 端口范围 (必需参数)
		if ports, ok := request.Options["ports"].(string); ok && ports != "" {
			args = append(args, "-p", ports)
		} else {
			// 默认扫描常用端口
			args = append(args, "-p", "1-1000")
		}

		// 扫描速率
		if rate, ok := request.Options["rate"].(string); ok && rate != "" {
			args = append(args, "--rate", rate)
		} else if rateInt, ok := request.Options["rate"].(int); ok {
			args = append(args, "--rate", strconv.Itoa(rateInt))
		} else {
			// 默认扫描速率
			args = append(args, "--rate", "1000")
		}

		// 输出格式
		if request.OutputPath != "" {
			// 默认输出XML格式便于解析
			args = append(args, "-oX", request.OutputPath)

			// 如果指定了其他格式
			if outputFormat, ok := request.Options["output_format"].(string); ok {
				switch outputFormat {
				case "list":
					args = append(args, "-oL", strings.Replace(request.OutputPath, ".xml", ".list", 1))
				case "grepable":
					args = append(args, "-oG", strings.Replace(request.OutputPath, ".xml", ".gnmap", 1))
				case "json":
					args = append(args, "-oJ", strings.Replace(request.OutputPath, ".xml", ".json", 1))
				}
			}
		}

		// 源IP地址
		if sourceIP, ok := request.Options["source_ip"].(string); ok && sourceIP != "" {
			args = append(args, "--source-ip", sourceIP)
		}

		// 源端口
		if sourcePort, ok := request.Options["source_port"].(string); ok && sourcePort != "" {
			args = append(args, "--source-port", sourcePort)
		}

		// 网络接口
		if iface, ok := request.Options["interface"].(string); ok && iface != "" {
			args = append(args, "--interface", iface)
		}

		// 路由器MAC地址
		if routerMac, ok := request.Options["router_mac"].(string); ok && routerMac != "" {
			args = append(args, "--router-mac", routerMac)
		}

		// 排除目标
		if exclude, ok := request.Options["exclude"].(string); ok && exclude != "" {
			args = append(args, "--exclude", exclude)
		}

		// 排除文件
		if excludeFile, ok := request.Options["exclude_file"].(string); ok && excludeFile != "" {
			args = append(args, "--excludefile", excludeFile)
		}

		// 包含文件
		if includeFile, ok := request.Options["include_file"].(string); ok && includeFile != "" {
			args = append(args, "--includefile", includeFile)
		}

		// 连接超时
		if connectionTimeout, ok := request.Options["connection_timeout"].(string); ok && connectionTimeout != "" {
			args = append(args, "--connection-timeout", connectionTimeout)
		}

		// 最大重传次数
		if maxRetries, ok := request.Options["max_retries"].(string); ok && maxRetries != "" {
			args = append(args, "--max-retries", maxRetries)
		}

		// 等待时间
		if wait, ok := request.Options["wait"].(string); ok && wait != "" {
			args = append(args, "--wait", wait)
		}

		// 其他自定义参数
		if customArgs, ok := request.Options["custom_args"].([]string); ok {
			args = append(args, customArgs...)
		}
	} else {
		// 默认参数
		args = append(args, "-p", "1-1000", "--rate", "1000")
	}

	// 添加目标 (必须在最后)
	args = append(args, request.Target)

	return args, nil
}

// collectOutputFiles 收集输出文件
func (e *MasscanExecutor) collectOutputFiles(outputPath string) ([]string, error) {
	var files []string

	// 检查主输出文件
	if _, err := os.Stat(outputPath); err == nil {
		files = append(files, outputPath)
	}

	// 检查其他可能的输出文件
	baseDir := filepath.Dir(outputPath)
	baseName := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))

	possibleFiles := []string{
		filepath.Join(baseDir, baseName+".list"),
		filepath.Join(baseDir, baseName+".gnmap"),
		filepath.Join(baseDir, baseName+".json"),
		filepath.Join(baseDir, baseName+".xml"),
	}

	for _, file := range possibleFiles {
		if _, err := os.Stat(file); err == nil {
			files = append(files, file)
		}
	}

	return files, nil
}

// parseMasscanOutput 解析masscan输出
func (e *MasscanExecutor) parseMasscanOutput(output string) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})

	lines := strings.Split(output, "\n")
	openPorts := 0

	// 解析基本信息
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析发现的开放端口
		if strings.Contains(line, "Discovered open port") {
			openPorts++
		}

		// 解析扫描统计
		if strings.Contains(line, "rate:") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "rate:") {
					metadata["scan_rate"] = strings.TrimPrefix(part, "rate:")
					break
				}
			}
		}

		// 解析扫描时间
		if strings.Contains(line, "Scanning") && strings.Contains(line, "hosts") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "Scanning" && i+1 < len(parts) {
					metadata["scanned_hosts"] = parts[i+1]
					break
				}
			}
		}
	}

	metadata["open_ports_count"] = openPorts

	return metadata, nil
}

// Stop 停止正在执行的扫描任务
func (e *MasscanExecutor) Stop(ctx context.Context, taskID string) error {
	// Masscan执行器本身不维护任务状态，依赖上下文取消
	logger.WithFields(map[string]interface{}{
		"path":      "executor.masscan.Stop",
		"operation": "stop_masscan",
		"option":    "stop_requested",
		"func_name": "executor.masscan.Stop",
		"task_id":   taskID,
	}).Info("Stop requested for masscan task")

	return nil
}

// GetStatus 获取扫描任务状态
func (e *MasscanExecutor) GetStatus(ctx context.Context, taskID string) (*ScanStatus, error) {
	// Masscan执行器本身不维护任务状态，返回错误让管理器使用本地状态
	return nil, fmt.Errorf("masscan executor does not maintain task status")
}

// Cleanup 清理资源
func (e *MasscanExecutor) Cleanup() error {
	logger.WithFields(map[string]interface{}{
		"path":      "executor.masscan.Cleanup",
		"operation": "cleanup",
		"option":    "cleanup_completed",
		"func_name": "executor.masscan.Cleanup",
	}).Info("Masscan executor cleanup completed")

	return nil
}
