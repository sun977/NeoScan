package asset

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"neomaster/internal/model/system"
	"neomaster/internal/pkg/logger"
	"neomaster/internal/pkg/utils"
	assetRepo "neomaster/internal/repo/mysql/asset"
	assetService "neomaster/internal/service/asset"
)

// ETLErrorHandler ETL 错误管理控制器
type ETLErrorHandler struct {
	service assetService.AssetETLErrorService
}

// NewETLErrorHandler 创建控制器实例
func NewETLErrorHandler(service assetService.AssetETLErrorService) *ETLErrorHandler {
	return &ETLErrorHandler{service: service}
}

// ListErrors 获取错误列表
// GET /api/v1/asset/etl/errors
func (h *ETLErrorHandler) ListErrors(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.service == nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "service not initialized",
		})
		return
	}

	// 1. 解析参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	filter := assetRepo.ETLErrorFilter{
		Page:       page,
		PageSize:   pageSize,
		TaskID:     c.Query("task_id"),
		ResultType: c.Query("result_type"),
		Status:     c.Query("status"),
		ErrorStage: c.Query("error_stage"),
		StartTime:  c.Query("start_time"),
		EndTime:    c.Query("end_time"),
	}

	// 2. 调用 Service
	list, total, err := h.service.ListErrors(c.Request.Context(), filter)
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "GET", map[string]interface{}{
			"operation": "list_etl_errors",
			"filter":    filter,
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "failed to list etl errors",
			Error:   err.Error(),
		})
		return
	}

	// 3. 返回结果
	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "success",
		Data: gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// TriggerReplay 触发重放
// POST /api/v1/asset/etl/errors/replay
func (h *ETLErrorHandler) TriggerReplay(c *gin.Context) {
	clientIP := utils.GetClientIP(c)
	requestID := c.GetHeader("X-Request-ID")
	urlPath := c.Request.URL.String()

	if h.service == nil {
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "service not initialized",
		})
		return
	}

	count, err := h.service.TriggerReplay(c.Request.Context())
	if err != nil {
		logger.LogBusinessError(err, requestID, 0, clientIP, urlPath, "POST", map[string]interface{}{
			"operation": "replay_etl_errors",
		})
		c.JSON(http.StatusInternalServerError, system.APIResponse{
			Code:    http.StatusInternalServerError,
			Status:  "failed",
			Message: "failed to replay etl errors",
			Error:   err.Error(),
		})
		return
	}

	logger.LogBusinessOperation("replay_etl_errors", 0, "", clientIP, requestID, "success", "replay etl errors", map[string]interface{}{
		"count": count,
	})

	c.JSON(http.StatusOK, system.APIResponse{
		Code:    http.StatusOK,
		Status:  "success",
		Message: "replay triggered successfully",
		Data: gin.H{
			"replayed_count": count,
		},
	})
}
