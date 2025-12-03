package rule_engine

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"neomaster/internal/pkg/logger"
)

// ActionExecutor 动作执行器
type ActionExecutor struct {
}

// NewActionExecutor 创建新的动作执行器
func NewActionExecutor(logger interface{}) *ActionExecutor {
	return &ActionExecutor{}
}

// ExecuteAction 执行单个动作
func (ae *ActionExecutor) ExecuteAction(action Action, context *RuleContext) ActionResult {
	result := ActionResult{
		Type:      action.Type,
		Success:   false,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	logger.WithFields(map[string]interface{}{
		"action_type": action.Type,
		"parameters":  action.Parameters,
		"func_name":   "executor.ExecuteAction",
	}).Debug("开始执行动作")

	switch action.Type {
	case ActionLog:
		result = ae.executeLogAction(action, context)
	case ActionAlert:
		result = ae.executeAlertAction(action, context)
	case ActionBlock:
		result = ae.executeBlockAction(action, context)
	case ActionModify:
		result = ae.executeModifyAction(action, context)
	case ActionExecute:
		result = ae.executeCommandAction(action, context)
	default:
		result.Error = fmt.Sprintf("不支持的动作类型: %s", action.Type)
		logger.WithFields(map[string]interface{}{
			"action_type": action.Type,
			"error":       result.Error,
			"func_name":   "executor.ExecuteAction",
		}).Error("动作执行失败")
	}

	return result
}

// ExecuteActions 执行多个动作
func (ae *ActionExecutor) ExecuteActions(actions []Action, context *RuleContext) []ActionResult {
	results := make([]ActionResult, 0, len(actions))

	for _, action := range actions {
		result := ae.ExecuteAction(action, context)
		results = append(results, result)

		// 如果是阻止动作且执行成功，停止后续动作执行
		if action.Type == ActionBlock && result.Success {
			logger.WithFields(map[string]interface{}{
				"func_name": "executor.ExecuteActions",
			}).Info("阻止动作执行成功，停止后续动作")
			break
		}
	}

	return results
}

// executeLogAction 执行日志动作
func (ae *ActionExecutor) executeLogAction(action Action, context *RuleContext) ActionResult {
	result := ActionResult{
		Type:      ActionLog,
		Success:   true,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	// 获取日志级别，默认为info
	level := ae.getStringParameter(action.Parameters, "level", "info")
	message := ae.getStringParameter(action.Parameters, "message", "规则匹配")
	
	// 替换消息中的变量
	message = ae.replaceVariables(message, context)

	// 记录日志
	logFields := map[string]interface{}{
		"rule_action": "log",
		"context":     context.Data,
		"func_name":   "executor.executeLogAction",
	}

	switch strings.ToLower(level) {
	case "debug":
		logger.WithFields(logFields).Debug(message)
	case "info":
		logger.WithFields(logFields).Info(message)
	case "warn", "warning":
		logger.WithFields(logFields).Warn(message)
	case "error":
		logger.WithFields(logFields).Error(message)
	default:
		logger.WithFields(logFields).Info(message)
	}

	result.Message = fmt.Sprintf("日志记录成功: %s", message)
	result.Data["level"] = level
	result.Data["message"] = message

	return result
}

// executeAlertAction 执行告警动作
func (ae *ActionExecutor) executeAlertAction(action Action, context *RuleContext) ActionResult {
	result := ActionResult{
		Type:      ActionAlert,
		Success:   true,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	// 获取告警参数
	title := ae.getStringParameter(action.Parameters, "title", "安全告警")
	message := ae.getStringParameter(action.Parameters, "message", "检测到安全威胁")
	severity := ae.getStringParameter(action.Parameters, "severity", "medium")
	
	// 替换消息中的变量
	title = ae.replaceVariables(title, context)
	message = ae.replaceVariables(message, context)

	// 记录告警日志
	logger.WithFields(map[string]interface{}{
		"alert_title":    title,
		"alert_message":  message,
		"alert_severity": severity,
		"context":        context.Data,
		"func_name":      "executor.executeAlertAction",
	}).Warn("安全告警触发")

	result.Message = fmt.Sprintf("告警发送成功: %s", title)
	result.Data["title"] = title
	result.Data["message"] = message
	result.Data["severity"] = severity

	// TODO: 这里可以集成实际的告警系统，如邮件、短信、钉钉等
	
	return result
}

// executeBlockAction 执行阻止动作
func (ae *ActionExecutor) executeBlockAction(action Action, context *RuleContext) ActionResult {
	result := ActionResult{
		Type:      ActionBlock,
		Success:   true,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	// 获取阻止参数
	reason := ae.getStringParameter(action.Parameters, "reason", "安全策略阻止")
	blockType := ae.getStringParameter(action.Parameters, "block_type", "request")
	
	// 替换消息中的变量
	reason = ae.replaceVariables(reason, context)

	// 记录阻止日志
	logger.WithFields(map[string]interface{}{
		"block_reason": reason,
		"block_type":   blockType,
		"context":      context.Data,
		"func_name":    "executor.executeBlockAction",
	}).Warn("操作被阻止")

	result.Message = fmt.Sprintf("操作已阻止: %s", reason)
	result.Data["reason"] = reason
	result.Data["block_type"] = blockType

	// 在上下文中标记为已阻止
	if context.Variables == nil {
		context.Variables = make(map[string]interface{})
	}
	context.Variables["blocked"] = true
	context.Variables["block_reason"] = reason

	return result
}

// executeModifyAction 执行修改动作
func (ae *ActionExecutor) executeModifyAction(action Action, context *RuleContext) ActionResult {
	result := ActionResult{
		Type:      ActionModify,
		Success:   false,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	// 获取修改参数
	field := ae.getStringParameter(action.Parameters, "field", "")
	value := action.Parameters["value"]
	operation := ae.getStringParameter(action.Parameters, "operation", "set")

	if field == "" {
		result.Error = "修改字段不能为空"
		return result
	}

	// 执行修改操作
	switch operation {
	case "set":
		ae.setFieldValue(field, value, context)
		result.Success = true
		result.Message = fmt.Sprintf("字段 %s 设置为 %v", field, value)
	case "append":
		if ae.appendFieldValue(field, value, context) {
			result.Success = true
			result.Message = fmt.Sprintf("字段 %s 追加值 %v", field, value)
		} else {
			result.Error = fmt.Sprintf("无法追加到字段 %s", field)
		}
	case "remove":
		if ae.removeFieldValue(field, context) {
			result.Success = true
			result.Message = fmt.Sprintf("字段 %s 已删除", field)
		} else {
			result.Error = fmt.Sprintf("无法删除字段 %s", field)
		}
	default:
		result.Error = fmt.Sprintf("不支持的修改操作: %s", operation)
	}

	result.Data["field"] = field
	result.Data["value"] = value
	result.Data["operation"] = operation

	return result
}

// executeCommandAction 执行命令动作
func (ae *ActionExecutor) executeCommandAction(action Action, context *RuleContext) ActionResult {
	result := ActionResult{
		Type:      ActionExecute,
		Success:   false,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	// 获取命令参数
	command := ae.getStringParameter(action.Parameters, "command", "")
	args := ae.getStringArrayParameter(action.Parameters, "args", []string{})
	timeout := ae.getIntParameter(action.Parameters, "timeout", 30)

	if command == "" {
		result.Error = "命令不能为空"
		return result
	}

	// 替换命令和参数中的变量
	command = ae.replaceVariables(command, context)
	for i, arg := range args {
		args[i] = ae.replaceVariables(arg, context)
	}

	// 执行命令
	cmd := exec.Command(command, args...)
	
	// 设置超时
	if timeout > 0 {
		go func() {
			time.Sleep(time.Duration(timeout) * time.Second)
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()
	}

	// 执行并获取输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Sprintf("命令执行失败: %v", err)
		logger.WithFields(map[string]interface{}{
			"command":   command,
			"args":      args,
			"error":     err.Error(),
			"func_name": "executor.executeCommandAction",
		}).Error("命令执行失败")
	} else {
		result.Success = true
		result.Message = "命令执行成功"
	}

	result.Data["command"] = command
	result.Data["args"] = args
	result.Data["output"] = string(output)
	result.Data["exit_code"] = cmd.ProcessState.ExitCode()

	return result
}

// 辅助方法

// getStringParameter 获取字符串参数
func (ae *ActionExecutor) getStringParameter(params map[string]interface{}, key, defaultValue string) string {
	if value, exists := params[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// getStringArrayParameter 获取字符串数组参数
func (ae *ActionExecutor) getStringArrayParameter(params map[string]interface{}, key string, defaultValue []string) []string {
	if value, exists := params[key]; exists {
		if arr, ok := value.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		if arr, ok := value.([]string); ok {
			return arr
		}
	}
	return defaultValue
}

// getIntParameter 获取整数参数
func (ae *ActionExecutor) getIntParameter(params map[string]interface{}, key string, defaultValue int) int {
	if value, exists := params[key]; exists {
		if num, ok := value.(int); ok {
			return num
		}
		if num, ok := value.(float64); ok {
			return int(num)
		}
	}
	return defaultValue
}

// replaceVariables 替换消息中的变量
func (ae *ActionExecutor) replaceVariables(message string, context *RuleContext) string {
	result := message

	// 替换上下文数据变量 ${data.field}
	for key, value := range context.Data {
		placeholder := fmt.Sprintf("${data.%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	// 替换上下文变量 ${var.field}
	if context.Variables != nil {
		for key, value := range context.Variables {
			placeholder := fmt.Sprintf("${var.%s}", key)
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
		}
	}

	// 替换时间变量
	now := time.Now()
	result = strings.ReplaceAll(result, "${time.now}", now.Format(time.RFC3339))
	result = strings.ReplaceAll(result, "${time.date}", now.Format("2006-01-02"))
	result = strings.ReplaceAll(result, "${time.time}", now.Format("15:04:05"))

	return result
}

// setFieldValue 设置字段值
func (ae *ActionExecutor) setFieldValue(field string, value interface{}, context *RuleContext) {
	parts := strings.Split(field, ".")
	current := context.Data

	// 导航到目标字段的父级
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// 创建新的map
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			}
		} else {
			// 创建新的map
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}

	// 设置最终字段值
	current[parts[len(parts)-1]] = value
}

// appendFieldValue 追加字段值
func (ae *ActionExecutor) appendFieldValue(field string, value interface{}, context *RuleContext) bool {
	// 获取当前值
	currentValue, exists := ae.getFieldValue(field, context)
	if !exists {
		// 如果字段不存在，创建新数组
		ae.setFieldValue(field, []interface{}{value}, context)
		return true
	}

	// 如果当前值是数组，追加新值
	if arr, ok := currentValue.([]interface{}); ok {
		newArr := append(arr, value)
		ae.setFieldValue(field, newArr, context)
		return true
	}

	return false
}

// removeFieldValue 删除字段值
func (ae *ActionExecutor) removeFieldValue(field string, context *RuleContext) bool {
	parts := strings.Split(field, ".")
	current := context.Data

	// 导航到目标字段的父级
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return false
			}
		} else {
			return false
		}
	}

	// 删除最终字段
	delete(current, parts[len(parts)-1])
	return true
}

// getFieldValue 获取字段值（复用evaluator中的逻辑）
func (ae *ActionExecutor) getFieldValue(field string, context *RuleContext) (interface{}, bool) {
	parts := strings.Split(field, ".")
	current := context.Data

	for i, part := range parts {
		if current == nil {
			return nil, false
		}

		value, exists := current[part]
		if !exists {
			return nil, false
		}

		if i == len(parts)-1 {
			return value, true
		}

		if nextMap, ok := value.(map[string]interface{}); ok {
			current = nextMap
		} else {
			return nil, false
		}
	}

	return nil, false
}