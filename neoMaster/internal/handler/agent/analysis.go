/**
 * Agent高级查询与统计控制器
 * 作者: Sun977
 * 日期: 2025-11-07
 * 说明: 与Agent高级查询与统计相关的 Handler 方法占位，未来承载数据分析与统计接口。
 * - GetAgentStatistics
 * - GetAgentLoadBalance
 * - GetAgentPerformanceAnalysis
 * - GetAgentCapacityAnalysis
 */
package agent

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
)

// GetAgentStatistics 获取Agent统计信息
func (h *AgentHandler) GetAgentStatistics(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	windowSeconds := 180
	if v := c.Query("window_seconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			windowSeconds = n
		}
	}

	var tagIDs []uint64
	if v := c.Query("tag_ids"); v != "" {
		ids := strings.Split(v, ",")
		for _, idStr := range ids {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	// resp, err := h.agentMonitorService.GetAgentStatistics(groupID, windowSeconds)
	resp, err := h.agentMonitorService.GetAgentStatistics(windowSeconds, tagIDs)
	if err != nil {
		status := h.getErrorStatusCode(err)
		logger.LogBusinessError(err, XRequestID, currentUserID, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation":      "get_agent_statistics",
			"option":         "service.GetAgentStatistics",
			"func_name":      "handler.agent.GetAgentStatistics",
			"window_seconds": windowSeconds,
			"tag_ids":        tagIDs,
			"user_agent":     userAgent,
		})
		c.JSON(status, system.APIResponse{Code: status, Status: "error", Message: err.Error(), Data: nil})
		return
	}

	logger.LogBusinessOperation("get_agent_statistics", currentUserID, "", clientIP, XRequestID, "success", "获取Agent统计信息成功", map[string]interface{}{
		"func_name":      "handler.agent.GetAgentStatistics",
		"option":         "response.success",
		"path":           pathUrl,
		"method":         "GET",
		"user_agent":     userAgent,
		"window_seconds": windowSeconds,
		"tag_ids":        tagIDs,
	})

	c.JSON(http.StatusOK, system.APIResponse{Code: http.StatusOK, Status: "success", Message: "OK", Data: resp})
}

// GetAgentLoadBalance 获取Agent负载均衡状态
func (h *AgentHandler) GetAgentLoadBalance(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	windowSeconds := 180
	if v := c.Query("window_seconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			windowSeconds = n
		}
	}
	topN := 5
	if v := c.Query("top_n"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			topN = n
		}
	}

	var tagIDs []uint64
	if v := c.Query("tag_ids"); v != "" {
		ids := strings.Split(v, ",")
		for _, idStr := range ids {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	// resp, err := h.agentMonitorService.GetAgentLoadBalance(groupID, windowSeconds, topN)

	resp, err := h.agentMonitorService.GetAgentLoadBalance(windowSeconds, topN, tagIDs)
	if err != nil {
		status := h.getErrorStatusCode(err)
		logger.LogBusinessError(err, XRequestID, currentUserID, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation":      "get_agent_load_balance",
			"option":         "service.GetAgentLoadBalance",
			"func_name":      "handler.agent.GetAgentLoadBalance",
			"window_seconds": windowSeconds,
			"top_n":          topN,
			"tag_ids":        tagIDs,
			"user_agent":     userAgent,
		})
		c.JSON(status, system.APIResponse{Code: status, Status: "error", Message: err.Error(), Data: nil})
		return
	}

	logger.LogBusinessOperation("get_agent_load_balance", currentUserID, "", clientIP, XRequestID, "success", "获取Agent负载均衡成功", map[string]interface{}{
		"func_name":      "handler.agent.GetAgentLoadBalance",
		"option":         "response.success",
		"path":           pathUrl,
		"method":         "GET",
		"user_agent":     userAgent,
		"window_seconds": windowSeconds,
		"top_n":          topN,
		"tag_ids":        tagIDs,
	})

	c.JSON(http.StatusOK, system.APIResponse{Code: http.StatusOK, Status: "success", Message: "OK", Data: resp})
}

// GetAgentPerformanceAnalysis 获取Agent性能分析
func (h *AgentHandler) GetAgentPerformanceAnalysis(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	windowSeconds := 180
	if v := c.Query("window_seconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			windowSeconds = n
		}
	}
	topN := 5
	if v := c.Query("top_n"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			topN = n
		}
	}

	var tagIDs []uint64
	if v := c.Query("tag_ids"); v != "" {
		ids := strings.Split(v, ",")
		for _, idStr := range ids {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	// groupID := c.Query("group_id")
	// resp, err := h.agentMonitorService.GetAgentPerformanceAnalysis(groupID, windowSeconds, topN)

	resp, err := h.agentMonitorService.GetAgentPerformanceAnalysis(windowSeconds, topN, tagIDs)
	if err != nil {
		status := h.getErrorStatusCode(err)
		logger.LogBusinessError(err, XRequestID, currentUserID, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation":      "get_agent_performance_analysis",
			"option":         "service.GetAgentPerformanceAnalysis",
			"func_name":      "handler.agent.GetAgentPerformanceAnalysis",
			"window_seconds": windowSeconds,
			"top_n":          topN,
			"tag_ids":        tagIDs,
			"user_agent":     userAgent,
		})
		c.JSON(status, system.APIResponse{Code: status, Status: "error", Message: err.Error(), Data: nil})
		return
	}

	logger.LogBusinessOperation("get_agent_performance_analysis", currentUserID, "", clientIP, XRequestID, "success", "获取Agent性能分析成功", map[string]interface{}{
		"func_name":      "handler.agent.GetAgentPerformanceAnalysis",
		"option":         "response.success",
		"path":           pathUrl,
		"method":         "GET",
		"user_agent":     userAgent,
		"window_seconds": windowSeconds,
		"top_n":          topN,
		"tag_ids":        tagIDs,
	})

	c.JSON(http.StatusOK, system.APIResponse{Code: http.StatusOK, Status: "success", Message: "OK", Data: resp})
}

// GetAgentCapacityAnalysis 获取Agent容量分析
func (h *AgentHandler) GetAgentCapacityAnalysis(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	currentUserID := utils.GetCurrentUserIDFromGinContext(c)
	userAgent := c.GetHeader("User-Agent")
	XRequestID := c.GetHeader("X-Request-ID")
	pathUrl := c.Request.URL.String()

	windowSeconds := 180
	if v := c.Query("window_seconds"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			windowSeconds = n
		}
	}
	// groupID := c.Query("group_id")
	cpuThr := 80.0
	if v := c.Query("cpu_threshold"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cpuThr = f
		}
	}
	memThr := 80.0
	if v := c.Query("memory_threshold"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			memThr = f
		}
	}
	diskThr := 80.0
	if v := c.Query("disk_threshold"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			diskThr = f
		}
	}

	var tagIDs []uint64
	if v := c.Query("tag_ids"); v != "" {
		ids := strings.Split(v, ",")
		for _, idStr := range ids {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	// resp, err := h.agentMonitorService.GetAgentCapacityAnalysis(groupID, windowSeconds, cpuThr, memThr, diskThr)

	resp, err := h.agentMonitorService.GetAgentCapacityAnalysis(windowSeconds, cpuThr, memThr, diskThr, tagIDs)
	if err != nil {
		status := h.getErrorStatusCode(err)
		logger.LogBusinessError(err, XRequestID, currentUserID, clientIP, pathUrl, "GET", map[string]interface{}{
			"operation":        "get_agent_capacity_analysis",
			"option":           "service.GetAgentCapacityAnalysis",
			"func_name":        "handler.agent.GetAgentCapacityAnalysis",
			"window_seconds":   windowSeconds,
			"cpu_threshold":    cpuThr,
			"memory_threshold": memThr,
			"disk_threshold":   diskThr,
			"tag_ids":          tagIDs,
			"user_agent":       userAgent,
		})
		c.JSON(status, system.APIResponse{Code: status, Status: "error", Message: err.Error(), Data: nil})
		return
	}

	logger.LogBusinessOperation("get_agent_capacity_analysis", currentUserID, "", clientIP, XRequestID, "success", "获取Agent容量分析成功", map[string]interface{}{
		"func_name":        "handler.agent.GetAgentCapacityAnalysis",
		"option":           "response.success",
		"path":             pathUrl,
		"method":           "GET",
		"user_agent":       userAgent,
		"window_seconds":   windowSeconds,
		"cpu_threshold":    cpuThr,
		"memory_threshold": memThr,
		"disk_threshold":   diskThr,
		"tag_ids":          tagIDs,
	})

	c.JSON(http.StatusOK, system.APIResponse{Code: http.StatusOK, Status: "success", Message: "OK", Data: resp})
}
