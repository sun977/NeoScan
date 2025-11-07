/**
 * Agent性能指标管理控制器
 * 作者: Sun977
 * 日期: 2025-11-07
 * 说明: 将与性能指标管理相关的 Handler 方法集中于此，包括：
 * - GetAgentMetrics（获取单个Agent性能快照）
 * - GetAgentListAllMetrics（分页获取所有Agent性能快照）
 * - CreateAgentMetrics（创建/上报性能快照）
 * - UpdateAgentMetrics（更新性能快照）
 * 重构策略: 保持原有业务逻辑和返回格式不变，统一成功日志使用 LogBusinessOperation。
 */
package agent

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	agentModel "neomaster/internal/model/agent"
	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

// GetAgentMetrics 获取指定Agent的最新性能快照（来自Master端数据库 agent_metrics 表）
func (h *AgentHandler) GetAgentMetrics(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID并校验
	agentID := c.Param("id")
	if agentID == "" {
		// 按项目日志规范记录业务错误
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":  "get_agent_metrics",
				"option":     "paramValidation",
				"func_name":  "handler.agent.GetAgentMetrics",
				"user_agent": userAgent,
			},
		)
		// 返回统一错误响应
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 调用服务层，从数据库获取该Agent的最新性能快照
	metrics, err := h.agentMonitorService.GetAgentMetricsFromDB(agentID)
	if err != nil {
		// 映射错误为HTTP状态码
		statusCode := h.getErrorStatusCode(err)

		// 记录业务错误日志，包含关键字段
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":   "get_agent_metrics",
				"option":      "agentMonitorService.GetAgentMetricsFromDB",
				"func_name":   "handler.agent.GetAgentMetrics",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)

		// 针对404返回更清晰的消息，其它情况统一失败描述
		// staticcheck QF1003: 使用带标签的 switch 简化状态码分支并提升可读性
		var message string
		switch statusCode {
		case http.StatusNotFound:
			message = "Agent metrics not found"
		case http.StatusBadRequest:
			message = "Invalid request"
		default:
			message = "Failed to get agent metrics"
		}

		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: message,
			Error:   err.Error(),
		})
		return
	}

	// 成功业务日志：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"get_agent_metrics",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"获取Agent性能快照成功",
		map[string]interface{}{
			"func_name":  "handler.agent.GetAgentMetrics",
			"option":     "success",
			"path":       pathUrl,
			"method":     "GET",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	// 返回统一成功响应
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent metrics retrieved successfully",
		Data:    metrics,
	})
}

// GetAgentListAllMetrics 获取所有Agent的最新性能快照列表（只读，来自Master端数据库 agent_metrics 表）
// 路由：GET /api/v1/agent/metrics
func (h *AgentHandler) GetAgentListAllMetrics(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 解析分页参数（保持与 GetAgentList 中的解析风格一致）
	// page 和 page_size 为可选参数，未提供时使用合理默认值
	page := 1
	pageSize := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	// 解析过滤参数：work_status、scan_type、keyword（agent_id关键词）
	var workStatusPtr *agentModel.AgentWorkStatus
	if wsStr := c.Query("work_status"); wsStr != "" {
		ws := agentModel.AgentWorkStatus(wsStr)
		workStatusPtr = &ws
	}

	var scanTypePtr *agentModel.AgentScanType
	if stStr := c.Query("scan_type"); stStr != "" {
		st := agentModel.AgentScanType(stStr)
		scanTypePtr = &st
	}

	var keywordPtr *string
	if kw := c.Query("keyword"); kw != "" {
		keywordPtr = &kw
	}

	// 调用服务层，分页获取所有Agent的最新性能快照（仓储层SQL分页 + 过滤条件）
	list, total, err := h.agentMonitorService.GetAgentListAllMetricsFromDB(page, pageSize, workStatusPtr, scanTypePtr, keywordPtr)
	if err != nil {
		statusCode := h.getErrorStatusCode(err)

		// 记录业务错误日志
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"GET",
			map[string]interface{}{
				"operation":   "get_agent_list_all_metrics",
				"option":      "agentMonitorService.GetAgentListAllMetricsFromDB",
				"func_name":   "handler.agent.GetAgentListAllMetrics",
				"user_agent":  userAgent,
				"status_code": statusCode,
				"page":        page,
				"page_size":   pageSize,
				"work_status": func() interface{} {
					if workStatusPtr != nil {
						return *workStatusPtr
					}
					return nil
				}(),
				"scan_type": func() interface{} {
					if scanTypePtr != nil {
						return *scanTypePtr
					}
					return nil
				}(),
				"keyword": func() interface{} {
					if keywordPtr != nil {
						return *keywordPtr
					}
					return nil
				}(),
			},
		)

		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to get all agents metrics",
			Error:   err.Error(),
		})
		return
	}

	// Service 已进行分页查询，这里直接使用返回的当前页数据
	pageItems := list

	// 计算总页数（注意类型转换，total 为 int64）
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	// 构造分页响应结构
	resp := &agentModel.AgentMetricsListResponse{
		Metrics: pageItems,
		Pagination: &agentModel.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	// 成功业务日志（补充分页信息）：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"get_agent_list_all_metrics",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"获取所有Agent性能快照列表成功",
		map[string]interface{}{
			"func_name":  "handler.agent.GetAgentListAllMetrics",
			"option":     "success",
			"path":       pathUrl,
			"method":     "GET",
			"user_agent": userAgent,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"work_status": func() interface{} {
				if workStatusPtr != nil {
					return *workStatusPtr
				}
				return nil
			}(),
			"scan_type": func() interface{} {
				if scanTypePtr != nil {
					return *scanTypePtr
				}
				return nil
			}(),
			"keyword": func() interface{} {
				if keywordPtr != nil {
					return *keywordPtr
				}
				return nil
			}(),
		},
	)

	// 返回统一成功响应（包含分页）
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "All agents metrics retrieved successfully",
		Data:    resp,
	})
}

// CreateAgentMetrics 创建/上报Agent性能指标快照（Master端数据库插入）
// 路由：POST /api/v1/agent/:id/metrics
// 设计：
// - 严格遵守分层：Handler → Service → Repository → DB；不直接操作数据库
// - 统一日志与错误返回；校验基础数值边界，防止脏数据入库
func (h *AgentHandler) CreateAgentMetrics(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID并校验
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "create_agent_metrics",
				"option":     "paramValidation",
				"func_name":  "handler.agent.CreateAgentMetrics",
				"user_agent": userAgent,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 解析请求体为 AgentMetrics
	var metrics agentModel.AgentMetrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":  "create_agent_metrics",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.CreateAgentMetrics",
				"user_agent": userAgent,
				"agent_id":   agentID,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid metrics request format",
			Error:   err.Error(),
		})
		return
	}

	// 统一以路径中的 agentID 为准，避免用户伪造
	metrics.AgentID = agentID

	// 基础字段校验（防御性编程）
	if metrics.CPUUsage < 0 || metrics.CPUUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid CPU usage", Error: "invalid CPU usage"})
		return
	}
	if metrics.MemoryUsage < 0 || metrics.MemoryUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid memory usage", Error: "invalid memory usage"})
		return
	}
	if metrics.DiskUsage < 0 || metrics.DiskUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid disk usage", Error: "invalid disk usage"})
		return
	}
	if metrics.NetworkBytesSent < 0 || metrics.NetworkBytesRecv < 0 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid network bytes", Error: "invalid network bytes"})
		return
	}
	if metrics.RunningTasks < 0 || metrics.CompletedTasks < 0 || metrics.FailedTasks < 0 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid task counters", Error: "invalid task counters"})
		return
	}
	// WorkStatus 枚举校验（允许空值按默认）
	if metrics.WorkStatus != "" {
		valid := metrics.WorkStatus == agentModel.AgentWorkStatusIdle ||
			metrics.WorkStatus == agentModel.AgentWorkStatusWorking ||
			metrics.WorkStatus == agentModel.AgentWorkStatusException
		if !valid {
			c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid work status", Error: fmt.Sprintf("invalid work_status: %s", metrics.WorkStatus)})
			return
		}
	}
	// ScanType 可选校验：若提供，需为非空字符串（业务允许自定义类型，暂不强校验）
	// 时间戳：若未提供则自动补齐当前时间
	if metrics.Timestamp.IsZero() {
		metrics.UpdateTimestamp()
	}

	// 调用服务层创建指标
	if err := h.agentMonitorService.CreateAgentMetrics(agentID, &metrics); err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"POST",
			map[string]interface{}{
				"operation":   "create_agent_metrics",
				"option":      "agentMonitorService.CreateAgentMetrics",
				"func_name":   "handler.agent.CreateAgentMetrics",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: "Failed to create agent metrics",
			Error:   err.Error(),
		})
		return
	}

	// 成功业务日志与响应：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"create_agent_metrics",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"创建Agent性能指标成功",
		map[string]interface{}{
			"func_name":  "handler.agent.CreateAgentMetrics",
			"option":     "success",
			"path":       pathUrl,
			"method":     "POST",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	c.JSON(http.StatusCreated, system.APIResponse{
		Code:    http.StatusCreated,
		Status:  "success",
		Message: "Agent metrics created successfully",
		Data: map[string]interface{}{
			"agent_id":  agentID,
			"timestamp": metrics.Timestamp,
		},
	})
}

// UpdateAgentMetrics 更新Agent性能指标快照（Master端数据库更新）
// 路由：PUT /api/v1/agent/:id/metrics
func (h *AgentHandler) UpdateAgentMetrics(c *gin.Context) {
	// 规范化客户端信息
	clientIP := utils.GetClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	// 获取Agent ID并校验
	agentID := c.Param("id")
	if agentID == "" {
		logger.LogBusinessError(
			fmt.Errorf("agent ID is required"),
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":  "update_agent_metrics",
				"option":     "paramValidation",
				"func_name":  "handler.agent.UpdateAgentMetrics",
				"user_agent": userAgent,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Agent ID is required",
			Error:   "missing agent ID parameter",
		})
		return
	}

	// 解析请求体为 AgentMetrics
	var metrics agentModel.AgentMetrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":  "update_agent_metrics",
				"option":     "ShouldBindJSON",
				"func_name":  "handler.agent.UpdateAgentMetrics",
				"user_agent": userAgent,
				"agent_id":   agentID,
			},
		)
		c.JSON(http.StatusBadRequest, system.APIResponse{
			Code:    http.StatusBadRequest,
			Status:  "failed",
			Message: "Invalid metrics request format",
			Error:   err.Error(),
		})
		return
	}

	// 统一以路径中的 agentID 为准
	metrics.AgentID = agentID

	// 基础字段校验（与创建一致）
	if metrics.CPUUsage < 0 || metrics.CPUUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid CPU usage", Error: "invalid CPU usage"})
		return
	}
	if metrics.MemoryUsage < 0 || metrics.MemoryUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid memory usage", Error: "invalid memory usage"})
		return
	}
	if metrics.DiskUsage < 0 || metrics.DiskUsage > 100 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid disk usage", Error: "invalid disk usage"})
		return
	}
	if metrics.NetworkBytesSent < 0 || metrics.NetworkBytesRecv < 0 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid network bytes", Error: "invalid network bytes"})
		return
	}
	if metrics.RunningTasks < 0 || metrics.CompletedTasks < 0 || metrics.FailedTasks < 0 {
		c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid task counters", Error: "invalid task counters"})
		return
	}
	if metrics.WorkStatus != "" {
		valid := metrics.WorkStatus == agentModel.AgentWorkStatusIdle ||
			metrics.WorkStatus == agentModel.AgentWorkStatusWorking ||
			metrics.WorkStatus == agentModel.AgentWorkStatusException
		if !valid {
			c.JSON(http.StatusBadRequest, system.APIResponse{Code: http.StatusBadRequest, Status: "failed", Message: "Invalid work status", Error: fmt.Sprintf("invalid work_status: %s", metrics.WorkStatus)})
			return
		}
	}
	if metrics.Timestamp.IsZero() {
		metrics.UpdateTimestamp()
	}

	// 调用服务层更新指标
	if err := h.agentMonitorService.UpdateAgentMetrics(agentID, &metrics); err != nil {
		statusCode := h.getErrorStatusCode(err)
		logger.LogBusinessError(
			err,
			XRequestID,
			0,
			clientIP,
			pathUrl,
			"PUT",
			map[string]interface{}{
				"operation":   "update_agent_metrics",
				"option":      "agentMonitorService.UpdateAgentMetrics",
				"func_name":   "handler.agent.UpdateAgentMetrics",
				"user_agent":  userAgent,
				"agent_id":    agentID,
				"status_code": statusCode,
			},
		)
		// 针对404返回更清晰的消息，其它情况统一失败描述
		// staticcheck QF1003: 使用带标签的switch更清晰地表达状态码分支
		var message string
		switch statusCode {
		case http.StatusNotFound:
			message = "Agent metrics not found"
		default:
			message = "Failed to update agent metrics"
		}
		c.JSON(statusCode, system.APIResponse{
			Code:    statusCode,
			Status:  "failed",
			Message: message,
			Error:   err.Error(),
		})
		return
	}

	// 成功业务日志与响应：统一使用 LogBusinessOperation
	logger.LogBusinessOperation(
		"update_agent_metrics",
		0,
		"",
		clientIP,
		XRequestID,
		"success",
		"更新Agent性能指标成功",
		map[string]interface{}{
			"func_name":  "handler.agent.UpdateAgentMetrics",
			"option":     "success",
			"path":       pathUrl,
			"method":     "PUT",
			"user_agent": userAgent,
			"agent_id":   agentID,
		},
	)

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "Agent metrics updated successfully",
		Data: map[string]interface{}{
			"agent_id":  agentID,
			"timestamp": metrics.Timestamp,
		},
	})
}
