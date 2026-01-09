/**
 * Agent处理器基础文件：聚合结构体与通用校验/帮助方法
 * 作者: Sun977
 * 日期: 2025-11-07
 * 说明: 为了避免 agent 控制器文件过长，将 AgentHandler 的结构体、构造函数以及通用的校验方法、错误状态码映射方法放在此文件中。
 * 重构动机:
 * 1) 按照 internal/app/master/router/agent_routers.go 中的功能分类进行拆分，提升可读性与可维护性;
 * 2) 保持层级调用关系: Handler → Service → Repository → Database，不改变业务逻辑;
 * 3) 校验与错误码映射为通用能力，供各分类文件内的 Handler 方法复用。
 */
package agent

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	agentModel "neomaster/internal/model/agent"
	agentService "neomaster/internal/service/agent"
)

// AgentHandler Agent处理器
// 说明: 聚合与 Agent 相关的服务对象，供各 Handler 方法使用。
type AgentHandler struct {
	agentManagerService agentService.AgentManagerService // Agent管理服务（包含分组功能）
	agentMonitorService agentService.AgentMonitorService // Agent监控服务
	agentConfigService  agentService.AgentConfigService  // Agent配置服务
	agentUpdateService  agentService.AgentUpdateService  // Agent规则更新服务(Agent自己pull)
}

// NewAgentHandler 创建Agent处理器实例
// 参数:
// - agentManagerService: 管理服务
// - agentMonitorService: 监控服务
// - agentConfigService: 配置服务
// 返回:
// - *AgentHandler: 处理器实例
func NewAgentHandler(
	agentManagerService agentService.AgentManagerService,
	agentMonitorService agentService.AgentMonitorService,
	agentConfigService agentService.AgentConfigService,
	agentUpdateService agentService.AgentUpdateService,
) *AgentHandler {
	return &AgentHandler{
		agentManagerService: agentManagerService,
		agentMonitorService: agentMonitorService,
		agentConfigService:  agentConfigService,
		agentUpdateService:  agentUpdateService,
	}
}

// validateRegisterRequest 验证Agent注册请求参数
// 说明: 保持原有业务校验逻辑不变，提供通用的入参校验能力。
func (h *AgentHandler) validateRegisterRequest(req *agentModel.RegisterAgentRequest) error {
	if req.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	// 验证hostname长度
	if len(req.Hostname) > 255 {
		return fmt.Errorf("hostname too long")
	}
	if req.IPAddress == "" {
		return fmt.Errorf("ip_address is required")
	}
	// 验证IP地址格式
	if net.ParseIP(req.IPAddress) == nil {
		return fmt.Errorf("invalid ip_address format")
	}
	if req.Port <= 0 || req.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if req.Version == "" {
		return fmt.Errorf("version is required")
	}
	// 验证version长度
	if len(req.Version) > 50 {
		return fmt.Errorf("version too long")
	}
	if req.OS == "" {
		return fmt.Errorf("os is required")
	}
	if req.Arch == "" {
		return fmt.Errorf("arch is required")
	}
	// 验证CPU核心数不能为负数或零
	if req.CPUCores <= 0 {
		return fmt.Errorf("invalid CPU cores")
	}
	// 验证memory_total不能为负数
	if req.MemoryTotal < 0 {
		return fmt.Errorf("invalid memory total")
	}
	// 验证disk_total不能为负数
	if req.DiskTotal < 0 {
		return fmt.Errorf("invalid disk total")
	}
	// 验证TaskSupport不能为空
	if len(req.TaskSupport) == 0 {
		return fmt.Errorf("at least one task support is required")
	}
	// 验证TaskSupport包含有效值(根据ID验证) - 委托Service层处理业务逻辑
	for _, taskID := range req.TaskSupport {
		if !h.agentManagerService.IsValidTaskSupportId(taskID) {
			return fmt.Errorf("invalid task support ID: %s", taskID)
		}
	}
	return nil
}

// validateHeartbeatRequest 验证Agent心跳请求参数
// 说明: 保持原有逻辑，用于校验心跳请求中的必填字段与性能指标合理性。
func (h *AgentHandler) validateHeartbeatRequest(req *agentModel.HeartbeatRequest) error {
	if req.AgentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if req.Status == "" {
		return fmt.Errorf("status is required")
	}
	// 验证状态值是否有效
	validStatuses := []agentModel.AgentStatus{
		agentModel.AgentStatusOnline,
		agentModel.AgentStatusOffline,
		agentModel.AgentStatusException,
		agentModel.AgentStatusMaintenance,
	}
	isValidStatus := false
	for _, validStatus := range validStatuses {
		if req.Status == validStatus {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		return fmt.Errorf("invalid status")
	}

	// 验证性能指标数据（如果提供）
	if req.Metrics != nil {
		if req.Metrics.CPUUsage < 0 || req.Metrics.CPUUsage > 100 {
			return fmt.Errorf("invalid CPU usage")
		}
		if req.Metrics.MemoryUsage < 0 || req.Metrics.MemoryUsage > 100 {
			return fmt.Errorf("invalid memory usage")
		}
		if req.Metrics.DiskUsage < 0 || req.Metrics.DiskUsage > 100 {
			return fmt.Errorf("invalid disk usage")
		}
		if req.Metrics.NetworkBytesSent < 0 {
			return fmt.Errorf("invalid network bytes sent")
		}
		if req.Metrics.NetworkBytesRecv < 0 {
			return fmt.Errorf("invalid network bytes received")
		}
		if req.Metrics.RunningTasks < 0 {
			return fmt.Errorf("invalid running tasks")
		}
		if req.Metrics.CompletedTasks < 0 {
			return fmt.Errorf("invalid completed tasks")
		}
		if req.Metrics.FailedTasks < 0 {
			return fmt.Errorf("invalid failed tasks")
		}
	}
	return nil
}

// getErrorStatusCode 根据错误类型返回HTTP状态码
// 说明: 统一的错误到HTTP状态码映射策略，供各分类文件内复用，避免重复实现。
func (h *AgentHandler) getErrorStatusCode(err error) int {
	// 根据错误类型返回相应的HTTP状态码
	if err == nil {
		return http.StatusOK
	}

	// 可以根据具体的错误类型进行更精确的状态码映射
	errMsg := err.Error()
	// 统一处理“未找到”类错误
	if errMsg == "Agent not found" || errMsg == "agent not found" ||
		strings.Contains(errMsg, "不存在") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "未找到") {
		return http.StatusNotFound
	}

	// 参数缺失或不合法
	if strings.Contains(errMsg, "不能为空") || strings.Contains(errMsg, "is required") || strings.Contains(errMsg, "invalid") {
		return http.StatusBadRequest
	}

	// 默认返回内部服务器错误
	return http.StatusInternalServerError
}
