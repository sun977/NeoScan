/**
 * Nmap扫描工具执行器实现
 * @author: Sun977
 * @date: 2025.10.11
 * @description: Nmap扫描工具的具体执行器实现，支持端口扫描、服务识别等
 * @func: 执行nmap命令，解析输出结果，监控执行状态
 */
package executor_drop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"neomaster/internal/model/orchestrator_model_drop"
	"neomaster/internal/pkg/logger"
)

// NmapExecutor Nmap执行器实现
type NmapExecutor struct {
	name    string // 执行器名称
	version string // 执行器版本
}

// NewNmapExecutor 创建新的Nmap执行器
func NewNmapExecutor() *NmapExecutor {
	return &NmapExecutor{
		name:    "nmap_executor",
		version: "1.0.0",
	}
}

// GetName 获取执行器名称
func (e *NmapExecutor) GetName() string {
	return e.name
}

// GetVersion 获取执行器版本
func (e *NmapExecutor) GetVersion() string {
	return e.version
}

// GetSupportedTools 获取支持的扫描工具列表
func (e *NmapExecutor) GetSupportedTools() []string {
	return []string{"nmap"}
}

// IsToolSupported 检查是否支持指定工具
func (e *NmapExecutor) IsToolSupported(toolName string) bool {
	return strings.ToLower(toolName) == "nmap"
}

// ValidateConfig 验证工具配置是否正确
func (e *NmapExecutor) ValidateConfig(tool *orchestrator_model_drop.ScanTool) error {
	if tool == nil {
		return fmt.Errorf("scan tool cannot be nil")
	}

	if !e.IsToolSupported(tool.Name) {
		return fmt.Errorf("tool %s is not supported by nmap executor", tool.Name)
	}

	// 检查执行路径是否存在
	if tool.ExecutablePath == "" {
		return fmt.Errorf("execute path cannot be empty")
	}

	// 检查nmap可执行文件是否存在
	if _, err := os.Stat(tool.ExecutablePath); os.IsNotExist(err) {
		return fmt.Errorf("nmap executable not found at path: %s", tool.ExecutablePath)
	}

	// 验证nmap版本
	if err := e.validateNmapVersion(tool.ExecutablePath); err != nil {
		return fmt.Errorf("nmap validation failed: %w", err)
	}

	return nil
}

// validateNmapVersion 验证nmap版本
func (e *NmapExecutor) validateNmapVersion(nmapPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, nmapPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get nmap version: %w", err)
	}

	versionOutput := string(output)
	if !strings.Contains(versionOutput, "Nmap") {
		return fmt.Errorf("invalid nmap executable, version output: %s", versionOutput)
	}

	logger.WithFields(map[string]interface{}{
		"path":      "executor.nmap.validateNmapVersion",
		"operation": "validate_nmap",
		"option":    "version_check",
		"func_name": "executor.nmap.validateNmapVersion",
		"version":   strings.TrimSpace(versionOutput),
	}).Debug("Nmap version validation")

	return nil
}

// Execute 执行扫描任务
func (e *NmapExecutor) Execute(ctx context.Context, request *ScanRequest) (*ScanResult, error) {
	if request == nil {
		return nil, fmt.Errorf("scan request cannot be nil")
	}

	startTime := time.Now()

	logger.WithFields(map[string]interface{}{
		"path":      "executor.nmap.Execute",
		"operation": "execute_nmap",
		"option":    "start_execution",
		"func_name": "executor.nmap.Execute",
		"task_id":   request.TaskID,
		"target":    request.Target,
	}).Info("Starting nmap scan")

	// 构建nmap命令
	args, err := e.buildNmapArgs(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build nmap args: %w", err)
	}

	// 创建输出目录
	if request.OutputPath != "" {
		outputDir := filepath.Dir(request.OutputPath)
		if err1 := os.MkdirAll(outputDir, 0755); err1 != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err1)
		}
	}

	// 执行nmap命令
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
			"path":      "executor.nmap.Execute",
			"operation": "execute_nmap",
			"option":    "execution_failed",
			"func_name": "executor.nmap.Execute",
			"task_id":   request.TaskID,
		}).WithError(err).Error("Nmap execution failed")

		return result, nil // 返回结果而不是错误，让调用者处理
	}

	result.Status = ScanTaskStatusCompleted

	// 收集输出文件
	if request.OutputPath != "" {
		outputFiles, err1 := e.collectOutputFiles(request.OutputPath)
		if err1 != nil {
			logger.WithFields(map[string]interface{}{
				"path":      "executor.nmap.Execute",
				"operation": "execute_nmap",
				"option":    "collect_output_files_failed",
				"func_name": "executor.nmap.Execute",
				"task_id":   request.TaskID,
			}).WithError(err1).Warn("Failed to collect output files")
		} else {
			result.OutputFiles = outputFiles
		}
	}

	// 解析扫描结果元数据
	metadata, err := e.parseNmapOutput(string(output))
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"path":      "executor.nmap.Execute",
			"operation": "execute_nmap",
			"option":    "parse_output_failed",
			"func_name": "executor.nmap.Execute",
			"task_id":   request.TaskID,
		}).WithError(err).Warn("Failed to parse nmap output")
	} else {
		result.Metadata = metadata
	}

	logger.WithFields(map[string]interface{}{
		"path":      "executor.nmap.Execute",
		"operation": "execute_nmap",
		"option":    "execution_completed",
		"func_name": "executor.nmap.Execute",
		"task_id":   request.TaskID,
		"duration":  duration,
	}).Info("Nmap scan completed")

	return result, nil
}

// buildNmapArgs 构建nmap命令参数
func (e *NmapExecutor) buildNmapArgs(request *ScanRequest) ([]string, error) {
	args := []string{}

	// 添加目标
	args = append(args, request.Target)

	// 处理扫描选项
	if request.Options != nil {
		// 端口范围
		if ports, ok := request.Options["ports"].(string); ok && ports != "" {
			args = append(args, "-p", ports)
		}

		// 扫描类型
		if scanType, ok := request.Options["scan_type"].(string); ok {
			switch scanType {
			case "tcp_syn":
				args = append(args, "-sS")
			case "tcp_connect":
				args = append(args, "-sT")
			case "udp":
				args = append(args, "-sU")
			case "tcp_ack":
				args = append(args, "-sA")
			case "tcp_window":
				args = append(args, "-sW")
			case "tcp_maimon":
				args = append(args, "-sM")
			}
		}

		// 服务版本检测
		if serviceDetection, ok := request.Options["service_detection"].(bool); ok && serviceDetection {
			args = append(args, "-sV")
		}

		// 操作系统检测
		if osDetection, ok := request.Options["os_detection"].(bool); ok && osDetection {
			args = append(args, "-O")
		}

		// 脚本扫描
		if scripts, ok := request.Options["scripts"].(string); ok && scripts != "" {
			args = append(args, "--script", scripts)
		}

		// 扫描速度
		if timing, ok := request.Options["timing"].(string); ok && timing != "" {
			args = append(args, "-T", timing)
		}

		// 详细输出
		if verbose, ok := request.Options["verbose"].(bool); ok && verbose {
			args = append(args, "-v")
		}

		// 输出格式
		if request.OutputPath != "" {
			// 默认输出XML格式便于解析
			args = append(args, "-oX", request.OutputPath)

			// 如果指定了其他格式
			if outputFormat, ok := request.Options["output_format"].(string); ok {
				switch outputFormat {
				case "normal":
					args = append(args, "-oN", strings.Replace(request.OutputPath, ".xml", ".txt", 1))
				case "grepable":
					args = append(args, "-oG", strings.Replace(request.OutputPath, ".xml", ".gnmap", 1))
				case "all":
					args = append(args, "-oA", strings.TrimSuffix(request.OutputPath, filepath.Ext(request.OutputPath)))
				}
			}
		}

		// 其他自定义参数
		if customArgs, ok := request.Options["custom_args"].([]string); ok {
			args = append(args, customArgs...)
		}
	}

	return args, nil
}

// collectOutputFiles 收集输出文件
func (e *NmapExecutor) collectOutputFiles(outputPath string) ([]string, error) {
	var files []string

	// 检查主输出文件
	if _, err := os.Stat(outputPath); err == nil {
		files = append(files, outputPath)
	}

	// 检查其他可能的输出文件
	baseDir := filepath.Dir(outputPath)
	baseName := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))

	possibleFiles := []string{
		filepath.Join(baseDir, baseName+".txt"),
		filepath.Join(baseDir, baseName+".gnmap"),
		filepath.Join(baseDir, baseName+".xml"),
	}

	for _, file := range possibleFiles {
		if _, err := os.Stat(file); err == nil {
			files = append(files, file)
		}
	}

	return files, nil
}

// parseNmapOutput 解析nmap输出
func (e *NmapExecutor) parseNmapOutput(output string) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})

	lines := strings.Split(output, "\n")

	// 解析基本信息
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析扫描统计
		if strings.Contains(line, "Nmap scan report for") {
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				metadata["target_host"] = parts[4]
			}
		}

		// 解析扫描时间
		if strings.Contains(line, "Nmap done:") {
			if strings.Contains(line, "scanned in") {
				parts := strings.Fields(line)
				for i, part := range parts {
					if part == "scanned" && i+2 < len(parts) && parts[i+1] == "in" {
						metadata["scan_duration"] = parts[i+2]
						break
					}
				}
			}
		}

		// 解析端口统计
		if strings.Contains(line, "open ports") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				metadata["open_ports_count"] = parts[0]
			}
		}
	}

	return metadata, nil
}

// Stop 停止正在执行的扫描任务
func (e *NmapExecutor) Stop(ctx context.Context, taskID string) error {
	// Nmap执行器本身不维护任务状态，依赖上下文取消
	logger.WithFields(map[string]interface{}{
		"path":      "executor.nmap.Stop",
		"operation": "stop_nmap",
		"option":    "stop_requested",
		"func_name": "executor.nmap.Stop",
		"task_id":   taskID,
	}).Info("Stop requested for nmap task")

	return nil
}

// GetStatus 获取扫描任务状态
func (e *NmapExecutor) GetStatus(ctx context.Context, taskID string) (*ScanStatus, error) {
	// Nmap执行器本身不维护任务状态，返回错误让管理器使用本地状态
	return nil, fmt.Errorf("nmap executor does not maintain task status")
}

// Cleanup 清理资源
func (e *NmapExecutor) Cleanup() error {
	logger.WithFields(map[string]interface{}{
		"path":      "executor.nmap.Cleanup",
		"operation": "cleanup",
		"option":    "cleanup_completed",
		"func_name": "executor.nmap.Cleanup",
	}).Info("Nmap executor cleanup completed")

	return nil
}
